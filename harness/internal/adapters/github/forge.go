// Package github drives the merge requests of a GitHub repository through the gh CLI.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// Timing bounds the waits on the forge.
type Timing struct {
	Interval time.Duration // between two polls
	Grace    time.Duration // how long a commit may stay without any pipeline before it is said to have none
	Timeout  time.Duration // longest wait for pipelines to end
}

// Forge runs gh in the directory of a clone.
type Forge struct {
	dir    string
	timing Timing
}

// NewForge builds the adapter.
func NewForge(dir string, timing Timing) *Forge { return &Forge{dir: dir, timing: timing} }

// gh returns the trimmed standard output of a gh call; its error carries what gh wrote.
func (f *Forge) gh(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), f.timing.Timeout+time.Minute)
	defer cancel()
	// The subcommands are fixed by this package; references come from the command line of the user.
	cmd := exec.CommandContext(ctx, "gh", args...) //nolint:gosec // G204
	cmd.Dir = f.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return strings.TrimSpace(stdout.String()), fmt.Errorf("gh %s: %s: %w", args[0], firstLine(msg), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

const prFields = "number,url,headRefName,baseRefName,headRefOid,state,isDraft"

type pr struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	HeadRefOid  string `json:"headRefOid"`
	State       string `json:"state"`
	IsDraft     bool   `json:"isDraft"`
}

func (p pr) domain() domain.PullRequest {
	return domain.PullRequest{
		Number: p.Number, URL: p.URL, Head: p.HeadRefName, Base: p.BaseRefName,
		SHA: p.HeadRefOid, Open: p.State == "OPEN", Draft: p.IsDraft,
	}
}

// DefaultBranch implements app.Forge.
func (f *Forge) DefaultBranch() (string, error) {
	return f.gh("repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
}

// Get implements app.Forge.
func (f *Forge) Get(ref string) (domain.PullRequest, error) {
	out, err := f.gh("pr", "view", ref, "--json", prFields)
	if err != nil {
		return domain.PullRequest{}, err
	}
	var p pr
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		return domain.PullRequest{}, fmt.Errorf("decode gh pr view: %w", err)
	}
	return p.domain(), nil
}

// ByHead implements app.Forge.
func (f *Forge) ByHead(branch string) (domain.PullRequest, bool, error) {
	out, err := f.gh("pr", "list", "--head", branch, "--state", "open", "--json", prFields)
	if err != nil {
		return domain.PullRequest{}, false, err
	}
	var prs []pr
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return domain.PullRequest{}, false, fmt.Errorf("decode gh pr list: %w", err)
	}
	if len(prs) == 0 {
		return domain.PullRequest{}, false, nil
	}
	return prs[0].domain(), true, nil
}

// Retarget implements app.Forge.
func (f *Forge) Retarget(number int, base string) error {
	_, err := f.gh("pr", "edit", strconv.Itoa(number), "--base", base)
	return err
}

// Checks implements app.Forge.
func (f *Forge) Checks(number int) (domain.Checks, error) {
	_, err := f.gh("pr", "checks", strconv.Itoa(number), "--watch", "--interval", strconv.Itoa(max(1, int(f.timing.Interval.Seconds()))))
	switch {
	case err == nil:
		return domain.ChecksPass, nil
	case strings.Contains(err.Error(), "no checks reported"):
		return domain.ChecksNone, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return domain.ChecksFail, nil
	}
	return domain.ChecksFail, err
}

// Merge implements app.Forge. After a change of base the forge computes
// mergeability in the background: wait for its answer before merging.
func (f *Forge) Merge(number int, sha string) error {
	n := strconv.Itoa(number)
	deadline := time.Now().Add(f.timing.Timeout)
	for {
		state, err := f.gh("pr", "view", n, "--json", "mergeable", "--jq", ".mergeable")
		if err != nil {
			return err
		}
		if state == "CONFLICTING" {
			return errors.New("conflicts with its base")
		}
		if state == "MERGEABLE" {
			break
		}
		if time.Now().After(deadline) {
			return errors.New("the forge never said whether it can be merged")
		}
		time.Sleep(f.timing.Interval)
	}
	if _, err := f.gh("pr", "merge", n, "--merge", "--match-head-commit", sha); err != nil {
		return err
	}
	state, err := f.gh("pr", "view", n, "--json", "state", "--jq", ".state")
	if err != nil {
		return err
	}
	if state != "MERGED" {
		return fmt.Errorf("still %s after the merge", strings.ToLower(state))
	}
	return nil
}

type checkRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// CommitChecks implements app.Forge.
func (f *Forge) CommitChecks(sha string) (domain.Checks, error) {
	start := time.Now()
	for {
		out, err := f.gh("api", "repos/{owner}/{repo}/commits/"+sha+"/check-runs", "--jq", ".check_runs")
		if err != nil {
			return domain.ChecksFail, err
		}
		var runs []checkRun
		if err := json.Unmarshal([]byte(out), &runs); err != nil {
			return domain.ChecksFail, fmt.Errorf("decode check runs: %w", err)
		}
		elapsed := time.Since(start)
		if len(runs) == 0 {
			if elapsed >= f.timing.Grace {
				return domain.ChecksNone, nil
			}
		} else if verdict, done := conclude(runs); done {
			return verdict, nil
		}
		if elapsed >= f.timing.Timeout {
			return domain.ChecksFail, errors.New("pipelines still running at the timeout")
		}
		time.Sleep(f.timing.Interval)
	}
}

// conclude gives the verdict of the runs once they have all ended.
func conclude(runs []checkRun) (domain.Checks, bool) {
	verdict := domain.ChecksPass
	for _, r := range runs {
		if r.Status != "completed" {
			return domain.ChecksFail, false
		}
		switch r.Conclusion {
		case "success", "skipped", "neutral":
		default:
			verdict = domain.ChecksFail
		}
	}
	return verdict, true
}

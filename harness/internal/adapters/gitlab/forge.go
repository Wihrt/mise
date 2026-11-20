// Package gitlab drives the merge requests of a GitLab project through the glab CLI.
package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// Forge runs glab in the directory of a clone.
type Forge struct {
	dir    string
	timing domain.Timing
}

// NewForge builds the adapter.
func NewForge(dir string, timing domain.Timing) *Forge { return &Forge{dir: dir, timing: timing} }

// api returns the trimmed standard output of a `glab api` call on the project of the
// clone; its error carries what glab wrote.
func (f *Forge) api(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), f.timing.Timeout+time.Minute)
	defer cancel()
	// The arguments are fixed by this package; references come from the command line of the user.
	cmd := exec.CommandContext(ctx, "glab", append([]string{"api"}, args...)...) //nolint:gosec // G204
	cmd.Dir = f.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("glab api: %s: %w", firstLine(msg), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// get decodes the answer of a GET call.
func (f *Forge) get(into any, path string) error {
	out, err := f.api(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(out), into); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

type pipeline struct {
	SHA    string `json:"sha"`
	Status string `json:"status"`
	Source string `json:"source"`
}

type mr struct {
	IID                 int       `json:"iid"`
	WebURL              string    `json:"web_url"`
	SourceBranch        string    `json:"source_branch"`
	TargetBranch        string    `json:"target_branch"`
	SHA                 string    `json:"sha"`
	State               string    `json:"state"`
	Draft               bool      `json:"draft"`
	HasConflicts        bool      `json:"has_conflicts"`
	DetailedMergeStatus string    `json:"detailed_merge_status"`
	HeadPipeline        *pipeline `json:"head_pipeline"`
}

func (m mr) domain() domain.PullRequest {
	return domain.PullRequest{
		Number: m.IID, URL: m.WebURL, Head: m.SourceBranch, Base: m.TargetBranch,
		SHA: m.SHA, State: m.State, Open: m.State == "opened", Draft: m.Draft,
		Conflicts: m.HasConflicts, MergeStatus: m.DetailedMergeStatus,
	}
}

func path(number int) string { return "projects/:id/merge_requests/" + strconv.Itoa(number) }

func (f *Forge) read(number int) (mr, error) {
	var m mr
	err := f.get(&m, path(number))
	return m, err
}

// transient are the statuses GitLab goes through while it computes mergeability.
var transient = map[string]bool{"checking": true, "unchecked": true, "preparing": true, "ci_still_running": true}

// settled reads a request until GitLab has computed its mergeability.
func (f *Forge) settled(number int) (mr, error) {
	deadline := time.Now().Add(f.timing.Timeout)
	for {
		m, err := f.read(number)
		if err != nil || !transient[m.DetailedMergeStatus] {
			return m, err
		}
		if time.Now().After(deadline) {
			return m, fmt.Errorf("the forge never said whether it can be merged, still %s", m.DetailedMergeStatus)
		}
		time.Sleep(f.timing.Interval)
	}
}

// Get implements app.Forge: a number, !number, the web URL of a request of the
// project, or the source branch of an open request.
func (f *Forge) Get(ref string) (domain.PullRequest, error) {
	if _, tail, isURL := strings.Cut(ref, "/-/merge_requests/"); isURL {
		n, err := strconv.Atoi(strings.TrimSuffix(tail, "/"))
		if err != nil {
			return domain.PullRequest{}, fmt.Errorf("%s is not a merge request URL", ref)
		}
		m, err := f.settled(n)
		if err != nil {
			return domain.PullRequest{}, err
		}
		if m.WebURL != strings.TrimSuffix(ref, "/") {
			return domain.PullRequest{}, fmt.Errorf("%s is not a merge request of this project", ref)
		}
		return m.domain(), nil
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(ref, "!")); err == nil {
		m, err := f.settled(n)
		return m.domain(), err
	}
	pr, found, err := f.ByHead(ref)
	if err == nil && !found {
		err = fmt.Errorf("no open merge request comes from %s", ref)
	}
	return pr, err
}

// ByHead implements app.Forge.
func (f *Forge) ByHead(branch string) (domain.PullRequest, bool, error) {
	var mrs []mr
	if err := f.get(&mrs, "projects/:id/merge_requests?state=opened&source_branch="+url.QueryEscape(branch)); err != nil {
		return domain.PullRequest{}, false, err
	}
	if len(mrs) == 0 {
		return domain.PullRequest{}, false, nil
	}
	// The list has no head pipeline and its merge status is not reliable: read the request.
	m, err := f.settled(mrs[0].IID)
	return m.domain(), err == nil, err
}

// WasHead implements app.Forge: a request of this project, in any state.
func (f *Forge) WasHead(branch string) (domain.PullRequest, bool, error) {
	out, err := f.api("--paginate", "projects/:id/merge_requests?source_branch="+url.QueryEscape(branch))
	if err != nil {
		return domain.PullRequest{}, false, err
	}
	// glab prints one JSON array per page.
	for dec := json.NewDecoder(strings.NewReader(out)); dec.More(); {
		var page []mr
		if err := dec.Decode(&page); err != nil {
			return domain.PullRequest{}, false, fmt.Errorf("decode merge requests of %s: %w", branch, err)
		}
		if len(page) > 0 {
			return page[0].domain(), true, nil
		}
	}
	return domain.PullRequest{}, false, nil
}

// Retarget implements app.Forge.
func (f *Forge) Retarget(number int, base string) error {
	_, err := f.api("-X", "PUT", "-f", "target_branch="+base, path(number))
	return err
}

// Checks implements app.Forge: the head pipeline of the request, once it has ended.
func (f *Forge) Checks(number int) (domain.Checks, error) {
	start := time.Now()
	for {
		m, err := f.read(number)
		if err != nil {
			return domain.ChecksFail, err
		}
		elapsed := time.Since(start)
		if p := m.HeadPipeline; p == nil {
			if elapsed >= f.timing.Grace {
				return domain.ChecksNone, nil
			}
		} else if p.SHA != m.SHA {
			return domain.ChecksFail, nil
		} else if verdict, done := conclude([]pipeline{*p}); done {
			if verdict == domain.ChecksPass {
				return f.onMergeRequestPipeline(number, *p)
			}
			return verdict, nil
		}
		if elapsed >= f.timing.Timeout {
			return domain.ChecksFail, errors.New("pipelines still running at the timeout")
		}
		time.Sleep(f.timing.Interval)
	}
}

// onMergeRequestPipeline demands that, in a project that runs merge request pipelines,
// the green pipeline is one of them.
func (f *Forge) onMergeRequestPipeline(number int, head pipeline) (domain.Checks, error) {
	var all []pipeline
	if err := f.get(&all, path(number)+"/pipelines"); err != nil {
		return domain.ChecksFail, err
	}
	for _, p := range all {
		if p.Source == "merge_request_event" && head.Source != p.Source {
			return domain.ChecksFail, nil
		}
	}
	return domain.ChecksPass, nil
}

// Merge implements app.Forge. After a change of target the forge computes
// mergeability in the background: wait for its answer before merging.
func (f *Forge) Merge(number int, sha string) error {
	m, err := f.settled(number)
	if err != nil {
		return err
	}
	if m.DetailedMergeStatus != "mergeable" {
		return fmt.Errorf("not mergeable: %s", m.DetailedMergeStatus)
	}
	if _, err := f.api("-X", "PUT", "-f", "sha="+sha, "-f", "should_remove_source_branch=true", path(number)+"/merge"); err != nil {
		return err
	}
	m, err = f.read(number)
	if err != nil {
		return err
	}
	if m.State != "merged" {
		return fmt.Errorf("still %s after the merge", m.State)
	}
	return nil
}

// CommitChecks implements app.Forge: the pipelines of the commit, once they have ended.
func (f *Forge) CommitChecks(sha string) (domain.Checks, error) {
	start := time.Now()
	for {
		var runs []pipeline
		if err := f.get(&runs, "projects/:id/pipelines?sha="+url.QueryEscape(sha)); err != nil {
			return domain.ChecksFail, err
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

// conclude gives the verdict of the pipelines once they have all ended.
func conclude(runs []pipeline) (domain.Checks, bool) {
	verdict := domain.ChecksPass
	for _, p := range runs {
		switch p.Status {
		case "success":
		case "failed", "canceled", "skipped":
			verdict = domain.ChecksFail
		default:
			return domain.ChecksFail, false
		}
	}
	return verdict, true
}

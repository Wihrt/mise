package tdd

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	acceptanceBasenames = map[string]bool{"TESTS.md": true, "mise.toml": true, ".gitignore": true, "conftest.py": true}
	acceptanceDirs      = map[string]bool{"test": true, "tests": true, "acceptance": true, "testdata": true}
	hunkOld             = regexp.MustCompile(`^@@ -(\d+)`)
	hunkNew             = regexp.MustCompile(`^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)
	miseRun             = regexp.MustCompile(`mise run` + space + `+([` + word + `:.\-]+)`)
)

// WiringFiles are the commit-hook and CI files that wire a task.
var WiringFiles = []string{".pre-commit-config.yaml", ".gitlab-ci.yml", ".github/workflows/*"}

// IsTestPath reports a path an acceptance commit may touch.
func IsTestPath(path string) bool {
	parts := strings.Split(path, "/")
	base := parts[len(parts)-1]
	if acceptanceBasenames[base] || strings.Contains(base, "_test.") ||
		(strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") && len(base) >= len("test_.py")) ||
		strings.HasSuffix(base, ".tftest.hcl") {
		return true
	}
	for _, d := range parts[:len(parts)-1] {
		if acceptanceDirs[d] {
			return true
		}
	}
	return false
}

// IsWiringFile reports a commit-hook or CI file.
func IsWiringFile(path string) bool {
	for _, w := range WiringFiles {
		if Fnmatch(path, w) {
			return true
		}
	}
	return false
}

// MiseRunTasks lists the tasks a file runs with `mise run`.
func MiseRunTasks(text string) []string {
	var tasks []string
	for _, m := range miseRun.FindAllStringSubmatch(text, -1) {
		tasks = append(tasks, m[1])
	}
	return tasks
}

// MiseNonTools keeps the lines of a mise.toml outside its [tools] table.
func MiseNonTools(text string) []string {
	var keep []string
	in := false
	for _, line := range Lines(text) {
		if strings.HasPrefix(line, "[") {
			in = line != "[tools]"
		}
		if in {
			keep = append(keep, line)
		}
	}
	return keep
}

// tableOf is the task table in force after a line of mise.toml ("" outside a task).
func tableOf(line, table string) string {
	if !strings.HasPrefix(line, "[") {
		return table
	}
	rest, ok := strings.CutPrefix(strings.TrimRightFunc(line, isSpace), "[tasks.")
	if !ok || !strings.HasSuffix(rest, "]") {
		return ""
	}
	inner := rest[:len(rest)-1]
	if len(inner) >= 3 && inner[0] == '"' && inner[len(inner)-1] == '"' {
		return inner[1 : len(inner)-1]
	}
	return inner
}

// MisePatchProblem tells why a patch changes the judged task in mise.toml, ""
// when it does not. old holds the lines of the mise.toml the patch applies to.
func MisePatchProblem(old []string, patch, task string) string {
	table, inFile := "", false
	for _, line := range Lines(patch) {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			inFile = strings.HasSuffix(line, " b/mise.toml")
		case !inFile:
		case hunkOld.MatchString(line):
			n, _ := strconv.Atoi(hunkOld.FindStringSubmatch(line)[1])
			for i := 0; i < n-1 && i < len(old); i++ {
				table = tableOf(old[i], table)
			}
		case line != "" && strings.ContainsRune(" +-", rune(line[0])):
			table = tableOf(line[1:], table)
			if line[0] != ' ' && table == task && task != "" {
				return "touches test task " + table
			}
		}
	}
	return ""
}

// Kills records, for one criterion, the criteria each of its mutants made fail.
type Kills struct {
	ID        string
	PerMutant []map[string]bool
}

// Redundant lists the pairs (A, B) where every mutant of A also fails a test of
// the other criterion B.
func Redundant(kills []Kills) [][2]string {
	var res [][2]string
	for _, k := range kills {
		var common []string
		for id := range k.PerMutant[0] {
			in := id != k.ID
			for _, m := range k.PerMutant[1:] {
				in = in && m[id]
			}
			if in {
				common = append(common, id)
			}
		}
		sort.Strings(common)
		for _, b := range common {
			res = append(res, [2]string{k.ID, b})
		}
	}
	return res
}

// Block is a run of lines a commit adds to a production file.
type Block struct {
	Path       string
	Start, End int
}

// AddedBlocks reads a `git diff -U0` and lists the blocks added to production files.
func AddedBlocks(diff string) []Block {
	var blocks []Block
	path := ""
	for _, line := range Lines(diff) {
		if strings.HasPrefix(line, "+++ ") {
			path = ""
			if len(line) > 6 {
				path = line[6:]
			}
			continue
		}
		m := hunkNew.FindStringSubmatch(line)
		if path == "" || IsTestPath(path) || m == nil {
			continue
		}
		start, _ := strconv.Atoi(m[1])
		count := 1
		if m[2] != "" {
			count, _ = strconv.Atoi(m[2])
		}
		if count > 0 {
			blocks = append(blocks, Block{Path: path, Start: start, End: start + count - 1})
		}
	}
	return blocks
}

package coverage

import (
	"path"
	"regexp"
	"strings"
)

// exclusionKey matches a line that starts one of the keys of coverage.py
// (coverage/config.py) that remove lines or branches from the report.
var exclusionKey = regexp.MustCompile(`^\s*(exclude_lines|exclude_also|partial_branches_always|partial_branches|partial_also)\s*[=:]`)

// reportSections gives, by base name, the sections coverage.py reads for its
// report settings.
var reportSections = map[string][]string{
	".coveragerc":    {"report", "coverage:report"},
	"setup.cfg":      {"coverage:report"},
	"tox.ini":        {"coverage:report"},
	"pyproject.toml": {"tool.coverage.report"},
}

// IsCoverageConfig tells whether coverage.py reads the file by its name.
func IsCoverageConfig(file string) bool {
	_, ok := reportSections[path.Base(file)]
	return ok
}

// Exclusions lists the lines of a coverage.py configuration file that start a
// line-exclusion key in a section the file honours.
func Exclusions(file, text string) []Finding {
	var out []Finding
	section := ""
	for i, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			section = strings.Trim(t, "[] ")
			continue
		}
		if !exclusionKey.MatchString(l) {
			continue
		}
		for _, s := range reportSections[path.Base(file)] {
			if s == section {
				out = append(out, Finding{Kind: "PRAGMA", File: file, Start: i + 1, End: i + 1})
			}
		}
	}
	return out
}

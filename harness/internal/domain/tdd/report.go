package tdd

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// Report states.
const (
	ReportOK         = "ok"
	ReportMissing    = "missing"
	ReportEmpty      = "empty"
	ReportUnreadable = "unreadable"
)

// Test case statuses.
const (
	Pass    = "pass"
	Fail    = "fail"
	Skipped = "skipped"
)

// Case is the result of one reported test.
type Case struct {
	Name   string
	Status string
}

// Report is what a test task wrote about its run.
type Report struct {
	State string
	Cases []Case
	Diag  string
}

func withCases(cases []Case, diag string) Report {
	if len(cases) == 0 {
		return Report{State: ReportEmpty, Diag: diag}
	}
	return Report{State: ReportOK, Cases: cases, Diag: diag}
}

// ParseJUnit reads a JUnit XML report.
func ParseJUnit(data string) Report {
	if data == "" {
		return Report{State: ReportEmpty}
	}
	dec := xml.NewDecoder(strings.NewReader(data))
	dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	var cases []Case
	depth, caseDepth, root := 0, -1, false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Report{State: ReportUnreadable, Diag: err.Error()}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			root = true
			switch {
			case t.Name.Local == "testcase" && t.Name.Space == "":
				name := ""
				for _, a := range t.Attr {
					if a.Name.Local == "name" && a.Name.Space == "" {
						name = a.Value
					}
				}
				cases = append(cases, Case{Name: name, Status: Pass})
				caseDepth = depth
			case caseDepth > 0 && depth == caseDepth+1 && t.Name.Space == "":
				last := &cases[len(cases)-1]
				switch t.Name.Local {
				case "failure", "error":
					last.Status = Fail
				case "skipped":
					if last.Status == Pass {
						last.Status = Skipped
					}
				}
			}
		case xml.EndElement:
			if depth == caseDepth {
				caseDepth = -1
			}
			depth--
		}
	}
	if !root || depth != 0 {
		return Report{State: ReportUnreadable, Diag: "no root element"}
	}
	return withCases(cases, "")
}

// ParseTofuJSON reads the output of `tofu test -json-into`.
func ParseTofuJSON(data string) Report {
	if data == "" {
		return Report{State: ReportEmpty}
	}
	var cases []Case
	var diag []string
	bad := false
	for _, raw := range Lines(data) {
		var ev struct {
			Type    string `json:"type"`
			TestRun struct {
				Run    string `json:"run"`
				Status string `json:"status"`
			} `json:"test_run"`
			Diagnostic struct {
				Summary string `json:"summary"`
				Detail  string `json:"detail"`
			} `json:"diagnostic"`
		}
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			bad = true
			continue
		}
		switch ev.Type {
		case "test_run":
			status := Fail
			switch ev.TestRun.Status {
			case "pass":
				status = Pass
			case "skip":
				status = Skipped
			}
			cases = append(cases, Case{Name: ev.TestRun.Run, Status: status})
		case "diagnostic":
			diag = append(diag, ev.Diagnostic.Summary+": "+ev.Diagnostic.Detail)
		}
	}
	if bad && len(cases) == 0 {
		return Report{State: ReportUnreadable}
	}
	return withCases(cases, strings.Join(diag, "\n"))
}

// ResultsByID groups the reported cases by the normalized ID their name starts with.
func ResultsByID(r Report) map[string][]Case {
	res := map[string][]Case{}
	for _, c := range r.Cases {
		if id := Candidate(c.Name); id != "" {
			res[id] = append(res[id], c)
		}
	}
	return res
}

// FirstNot returns the first case whose status is not want.
func FirstNot(cases []Case, want string) (Case, bool) {
	for _, c := range cases {
		if c.Status != want {
			return c, true
		}
	}
	return Case{}, false
}

// FirstFailed returns the first failed case.
func FirstFailed(cases []Case) (Case, bool) {
	for _, c := range cases {
		if c.Status == Fail {
			return c, true
		}
	}
	return Case{}, false
}

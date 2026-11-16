package coverage

import (
	"reflect"
	"testing"
)

func TestGoProfileNestedModules(t *testing.T) {
	modules := map[string]string{"ex.test/app": ".", "ex.test/app/tools": "tools"}
	profile := "mode: set\n" +
		"ex.test/app/tools/gen/gen.go:3.2,4.5 1 1\n" +
		"ex.test/app/main.go:3.2,4.5 1 0\n"
	m, err := ParseGoProfile(profile, modules)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 || m["main.go"] == nil || m["tools/gen/gen.go"] == nil {
		t.Errorf("files %v, want main.go and tools/gen/gen.go", m)
	}
}

func TestMergeAndFindings(t *testing.T) {
	modules := map[string]string{"ex.test/app": "."}
	a, _ := ParseGoProfile("mode: set\nex.test/app/a.go:2.1,3.2 1 1\nex.test/app/a.go:5.1,6.2 1 0\nex.test/app/a.go:8.1,8.1 0 0\n", modules)
	b, _ := ParseGoProfile("mode: count\nex.test/app/a.go:2.1,3.2 1 0\nex.test/app/a.go:5.1,6.2 1 0\n", modules)
	got := Merge(a, b).Findings()
	want := []Finding{{Kind: "UNCOVERED", File: "a.go", Start: 5, End: 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findings %+v, want %+v", got, want)
	}
}

func TestLCOVFindings(t *testing.T) {
	a, err := ParseLCOV("SF:a.py\nDA:1,1\nDA:2,0\nDA:4,0\nDA:5,1\nDA:6,1\nDA:7,0\nBRDA:5,0,jump to line 6,1\nBRDA:5,0,jump to line 8,-\nBRDA:5,1,jump to line 9,0\nBRDA:7,0,jump to line 8,0\nend_of_record\nSF:empty.py\nend_of_record\n")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := ParseLCOV("SF:a.py\nDA:2,1\nBRDA:5,1,jump to line 9,1\nend_of_record\n")
	got := Merge(a, b).Findings()
	want := []Finding{
		{Kind: "UNCOVERED", File: "a.py", Start: 4, End: 4},
		{Kind: "UNCOVERED BRANCH", File: "a.py", Start: 5, End: 5},
		{Kind: "UNCOVERED", File: "a.py", Start: 7, End: 7},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findings %+v, want %+v", got, want)
	}
	if len(a) != 2 {
		t.Errorf("files %v, want a.py and empty.py", a)
	}
}

func TestPragmas(t *testing.T) {
	got := Pragmas("a.py", "x = 1\ny = 2  # pragma: no cover\nfor i in z:  # pragma: no branch\n    pass\n")
	want := []Finding{{Kind: "PRAGMA", File: "a.py", Start: 2, End: 2}, {Kind: "PRAGMA", File: "a.py", Start: 3, End: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pragmas %+v, want %+v", got, want)
	}
}

func TestParseRejectsUnknownFormat(t *testing.T) {
	if _, err := Parse("hello", nil); err == nil {
		t.Error("want an error")
	}
}

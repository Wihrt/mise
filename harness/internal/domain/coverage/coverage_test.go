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
	if got, want := m.Files(), []string{"main.go", "tools/gen/gen.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files %v, want %v", got, want)
	}
}

func TestMergeAndFindings(t *testing.T) {
	modules := map[string]string{"ex.test/app": "."}
	a, _ := ParseGoProfile("mode: set\nex.test/app/a.go:2.1,3.2 1 1\nex.test/app/a.go:5.1,6.2 1 0\nex.test/app/a.go:8.1,8.1 0 0\n", modules)
	b, _ := ParseGoProfile("mode: count\nex.test/app/a.go:2.1,3.2 1 0\nex.test/app/a.go:5.1,6.2 1 0\n", modules)
	got := Merge(a, b).Findings(func(string) string { return "1\n2\n3\n4\n5\n  six\n" })
	want := []Finding{{Kind: "UNCOVERED", File: "a.go", Start: 5, End: 6, At: Pos{5, 1}, Text: "5"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findings %+v, want %+v", got, want)
	}
}

func TestParseRejectsUnknownFormat(t *testing.T) {
	if _, err := Parse("hello", nil); err == nil {
		t.Error("want an error")
	}
}

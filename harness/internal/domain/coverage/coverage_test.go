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

func TestParseRejectsUnknownFormat(t *testing.T) {
	if _, err := Parse("hello", nil); err == nil {
		t.Error("want an error")
	}
}

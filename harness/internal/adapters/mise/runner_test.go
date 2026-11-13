package mise

import (
	"slices"
	"testing"
)

func TestGoEnvAddsCount1(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"unset", []string{"A=1"}, "GOFLAGS=-count=1"},
		{"other flag", []string{"GOFLAGS=-mod=mod"}, "GOFLAGS=-mod=mod -count=1"},
		{"already there", []string{"GOFLAGS=-count=1 -mod=mod"}, "GOFLAGS=-count=1 -mod=mod"},
	}
	for _, tt := range tests {
		got := goEnv(tt.in)
		if got[len(got)-1] != tt.want {
			t.Errorf("%s: %v, want last %q", tt.name, got, tt.want)
		}
		if tt.name == "unset" && !slices.Contains(got, "A=1") {
			t.Errorf("unset: other variables are kept, got %v", got)
		}
	}
}

func TestDepends(t *testing.T) {
	got := TaskFile{}.Depends("[tools]\ngo = \"1\"\n[tasks.test]\ndepends = [\"test:*\"]\n[tasks.\"test:render\"]\nrun = \"x\"\n[tasks.check]\ndepends = \"lint\"\n")
	if !slices.Equal(got["test"], []string{"test:*"}) || !slices.Equal(got["check"], []string{"lint"}) {
		t.Errorf("Depends = %v", got)
	}
	if _, ok := got["test:render"]; !ok || len(got) != 3 {
		t.Errorf("every task is listed: %v", got)
	}
	if got := (TaskFile{}).Depends("not = [toml"); len(got) != 0 {
		t.Errorf("unreadable file: %v", got)
	}
}

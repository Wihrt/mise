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

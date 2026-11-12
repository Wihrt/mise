package domain

import (
	"slices"
	"testing"
)

func words(w ...string) Invocation { return Invocation{Words: w} }

func TestUnwrap(t *testing.T) {
	tests := []struct {
		name string
		in   Invocation
		want []string
	}{
		{"plain", words("git", "push"), []string{"git", "push"}},
		{"assignment", words("NO_PROXY=x", "kubectl", "get"), []string{"kubectl", "get"}},
		{"wrappers", words("sudo", "env", "A=b", "helm", "upgrade"), []string{"helm", "upgrade"}},
		{"empty", words(), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Unwrap().Words; !slices.Equal(got, tt.want) {
				t.Errorf("Unwrap() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOperands(t *testing.T) {
	tests := []struct {
		name       string
		in         Invocation
		valueFlags []string
		want       []string
	}{
		{"flag with its value", words("kubectl", "--context", "prod", "-n", "app", "delete", "pod"), []string{"--context", "-n"}, []string{"delete", "pod"}},
		{"flag carrying its value", words("kubectl", "--context=prod", "get", "pods"), []string{"--context"}, []string{"get", "pods"}},
		{"unknown flag takes no value", words("git", "push", "-u", "origin", "feat"), []string{"-c", "-C"}, []string{"push", "origin", "feat"}},
		{"value flag at the end", words("git", "-C"), []string{"-C"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Operands(tt.valueFlags...); !slices.Equal(got, tt.want) {
				t.Errorf("Operands() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFlagValue(t *testing.T) {
	tests := []struct {
		name string
		in   Invocation
		want string
	}{
		{"separate word", words("helm", "--kube-context", "kind-dev", "uninstall"), "kind-dev"},
		{"joined", words("kubectl", "--context=prod", "apply"), "prod"},
		{"absent", words("kubectl", "apply"), ""},
		{"flag without value", words("kubectl", "--context"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.FlagValue("--context", "--kube-context"); got != tt.want {
				t.Errorf("FlagValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

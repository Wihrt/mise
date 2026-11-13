package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestParseGateDefaults(t *testing.T) {
	for _, argv := range [][]string{
		{"red", "--slug", "s", "--test", "abc", "--task", "t"},
		{"green", "--slug", "s", "--test", "abc", "--task", "t", "--feat", "def"},
	} {
		o, err := parseGate(argv)
		if err != nil || o.Timeout != 540 || o.Jobs != 1 || o.Repo != "." {
			t.Errorf("%v: %+v, %v", argv, o, err)
		}
	}
}

func TestParseGateLists(t *testing.T) {
	o, err := parseGate([]string{"characterize", "--slug", "s", "--test", "x", "--task", "t", "--spec", "s.md",
		"--ac", "A-001", "--mutants", "m", "--ac-changed", "A-001,B-002", "--replaces=old,,older", "--jobs", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(o.ACChanged, []string{"A-001", "B-002"}) || !slices.Equal(o.Replaces, []string{"old", "older"}) || o.Proofs != "m" || o.Jobs != 4 {
		t.Errorf("options = %+v", o)
	}
	o, err = parseGate([]string{"replay", "--base", "a", "--head", "b", "--proofs", "p", "--task", "t", "--accept-lost", "x", "--accept-lost", "y"})
	if err != nil || !slices.Equal(o.AcceptLost, []string{"x", "y"}) {
		t.Errorf("replay: %+v, %v", o, err)
	}
	o, err = parseGate([]string{"green", "--slug", "s", "--test", "a", "--feat", "b", "--task", "t", "--proofs", "p",
		"--no-neutral", "--no-minimal", "--since", "c", "--unneeded-ok", "f:1-2", "--unneeded-ok", "g:3-3"})
	if err != nil || !o.NoNeutral || !o.NoMinimal || o.Since != "c" || !slices.Equal(o.UnneededOK, []string{"f:1-2", "g:3-3"}) {
		t.Errorf("green: %+v, %v", o, err)
	}
}

func TestParseGateUsageErrors(t *testing.T) {
	bad := [][]string{
		{},
		{"nope"},
		{"green", "--slug", "x"},
		{"red", "--slug", "s", "--test", "a", "--task", "t", "--ac", "A-001"},
		{"red", "--slug", "s", "--test", "a", "--task", "t", "--spec", "s.md"},
		{"red", "--slug", "s", "--test", "a", "--task", "t", "--feat", "b"},
		{"red", "--slug", "s", "--test", "a", "--task", "t", "extra"},
		{"red", "--slug", "s", "--test", "a", "--task", "t", "--format", "tap"},
		{"red", "--slug", "s", "--test", "a", "--task", "t", "--timeout", "soon"},
		{"characterize", "--slug", "s", "--test", "a", "--task", "t", "--spec", "s.md", "--ac", "A-001"},
		{"characterize", "--slug", "s", "--test", "a", "--task", "t", "--spec", "s.md", "--ac", "A-001", "--proofs", "p", "--mutants", "m"},
		{"replay", "--base", "a", "--head", "b", "--task", "t"},
	}
	for _, argv := range bad {
		var stdout, stderr bytes.Buffer
		if code := runGate(argv, &stdout, &stderr); code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "harness gate:") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", argv, code, stdout.String(), stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := runGate([]string{"green", "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "usage:") {
		t.Errorf("--help: exit %d, stdout %q", code, stdout.String())
	}
}

func TestRunLocalUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"local", "nope"}, strings.NewReader(""), &stdout, &stderr, noEnv); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

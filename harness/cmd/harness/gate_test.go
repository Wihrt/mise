package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestParseGateCoverage(t *testing.T) {
	o, err := parseGate([]string{"coverage", "--task", "t", "--coverage", "a", "--coverage", "b", "--baseline"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Task != "t" || o.Repo != "." || o.Timeout != 540 || !o.Baseline || !slices.Equal(o.Coverage, []string{"a", "b"}) {
		t.Errorf("options = %+v", o)
	}
}

func TestParseGateUsageErrors(t *testing.T) {
	bad := [][]string{
		{},
		{"red", "--task", "t"},
		{"coverage", "--task", "t"},
		{"coverage", "--coverage", "a"},
		{"coverage", "--task", "t", "--coverage", "a", "extra"},
		{"coverage", "--task", "t", "--coverage", "a", "--bogus"},
	}
	for _, argv := range bad {
		var stdout, stderr bytes.Buffer
		if code := runGate(argv, &stdout, &stderr); code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "harness gate:") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", argv, code, stdout.String(), stderr.String())
		}
	}
}

func TestRunGateHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runGate([]string{"coverage", "-h"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "harness gate coverage") {
		t.Errorf("exit %d, stdout %q", code, stdout.String())
	}
}

func TestRunLocalUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"local", "nope"}, strings.NewReader(""), &stdout, &stderr, noEnv); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

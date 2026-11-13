package local

import (
	"bytes"
	"errors"
	"testing"
)

type remote struct {
	url string
	err error
}

func (r remote) OriginURL(string) (string, error) { return r.url, r.err }

type files map[string]string

func (f files) Read(path string) (string, error) {
	if text, ok := f[path]; ok {
		return text, nil
	}
	return "", errors.New("no such file")
}

func TestHost(t *testing.T) {
	tests := map[string]string{ //nolint:gosec // G101: remote URLs, no credential
		"https://forge.example.test/groupe/depot.git":   "forge.example.test",
		"git@github.com:groupe/depot.git":               "github.com",
		"ssh://git@gitlab.com:2222/groupe/depot.git":    "gitlab.com",
		"https://jeton@forge.example.test/g/d":          "forge.example.test",
		"git+ssh://git@forge.example.test/groupe/depot": "forge.example.test",
	}
	for url, want := range tests {
		if got := Host(url); got != want {
			t.Errorf("Host(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestForge(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		forges   string
		wantCode int
		wantOut  string
	}{
		{"declared host", "https://forge.example.test/g/d.git", "forge.example.test glab\n", 0, "glab\n"},
		{"public host", "git@github.com:g/d.git", "", 0, "gh\n"},
		{"local file wins", "git@github.com:g/d.git", "github.com glab\n", 0, "glab\n"},
		{"unknown host", "https://ailleurs.test/g/d.git", "forge.example.test glab\n", 1, "hôte ailleurs.test inconnu\n"},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		l := New("/l", remote{url: tt.url + "\n"}, files{"/l/forges": tt.forges}, &out)
		code, err := l.Forge("/repo")
		if err != nil || code != tt.wantCode || out.String() != tt.wantOut {
			t.Errorf("%s: code %d, out %q, err %v", tt.name, code, out.String(), err)
		}
	}
	var out bytes.Buffer
	if code, err := New("/l", remote{err: errors.New("no origin")}, files{}, &out).Forge("/repo"); code != 1 || err == nil {
		t.Errorf("no origin: code %d, err %v", code, err)
	}
}

func TestCasReels(t *testing.T) {
	var out bytes.Buffer
	l := New("/l", remote{}, files{"/l/cas-reels.md": "- PRV-001 : un cas\n- FRM-002 : autre\n- voir PRV-001 aussi\n"}, &out)
	if code := l.CasReels("PRV-001"); code != 0 || out.String() != "- PRV-001 : un cas\n- voir PRV-001 aussi\n" {
		t.Errorf("code %d, out %q", code, out.String())
	}
	out.Reset()
	if code := New("/l", remote{}, files{}, &out).CasReels("PRV-001"); code != 0 || out.Len() != 0 {
		t.Errorf("no file: code %d, out %q", code, out.String())
	}
}

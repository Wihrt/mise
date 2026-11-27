package local

import (
	"bytes"
	"errors"
	"strings"
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

// Exists is true for a file, and for a folder holding one.
func (f files) Exists(path string) bool {
	for name := range f {
		if name == path || strings.HasPrefix(name, path+"/") {
			return true
		}
	}
	return false
}

func TestStack(t *testing.T) {
	const monorepo = "monorepo_root = true\n[monorepo]\nconfig_roots = [\"b\", \"a\"]\n"
	tests := map[string]struct {
		files files
		code  int
		want  string
		err   string
	}{
		"simple repo":                {files: files{"/r/go.mod": "", "/r/internal/x.go": ""}, want: ". stack=stack-go-app image=no\n"},
		"mise.toml of a simple repo": {files: files{"/r/mise.toml": "[tools]\n", "/r/pyproject.toml": "", "/r/src/x.py": ""}, want: ". stack=stack-py-app image=no\n"},
		"monorepo in the list order": {
			files: files{"/r/.config/mise.toml": monorepo, "/r/a/go.mod": "", "/r/a/cmd/m.go": "", "/r/b/x": ""},
			want:  "b stack=none\na stack=stack-go-app image=no\n",
		},
		"missing folder":  {files: files{"/r/mise.toml": monorepo, "/r/a/x": ""}, code: 2, err: "b is not a folder"},
		"no config_roots": {files: files{"/r/mise.toml": "monorepo_root = true\n"}, code: 2, err: "config_roots"},
		"bad toml":        {files: files{"/r/mise.toml": "monorepo_root = ["}, code: 2, err: "mise.toml"},
	}
	for name, tc := range tests {
		var out bytes.Buffer
		code, err := New("/l", remote{}, tc.files, &out).Stack("/r")
		if code != tc.code || out.String() != tc.want || (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: code %d, out %q, err %v; want code %d, out %q, err %q", name, code, out.String(), err, tc.code, tc.want, tc.err)
		}
	}
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

func TestTool(t *testing.T) {
	forges := files{"/l/forges": "forge.example.test glab\n"}
	tests := []struct{ url, want string }{
		{"https://forge.example.test/g/d.git", "glab"},
		{"git@forge.example.test:g/d.git", "glab"},
		{"https://github.com/o/r.git", "gh"},
	}
	for _, tt := range tests {
		got, host, err := New("/l", remote{url: tt.url}, forges, &bytes.Buffer{}).Tool("r")
		if err != nil || got != tt.want || host != Host(tt.url) {
			t.Errorf("Tool(%q) = %q, %q, %v; want %q", tt.url, got, host, err, tt.want)
		}
	}
	_, _, err := New("/l", remote{url: "https://inconnu.example.test/g/d.git"}, forges, &bytes.Buffer{}).Tool("r")
	if err == nil || err.Error() != "hôte inconnu.example.test absent de /l/forges" {
		t.Errorf("unknown host: err = %v", err)
	}
}

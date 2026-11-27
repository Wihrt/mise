// Package local reads the machine-local, never versioned files of the harness:
// `forges` (one `<host> <tool>` line per forge) and `cas-reels.md`.
package local

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Remote gives the URL of the origin of a repository.
type Remote interface {
	OriginURL(repo string) (string, error)
}

// Files reads a file of the local directory and tells whether a path exists.
type Files interface {
	Read(path string) (string, error)
	Exists(path string) bool
}

var (
	scheme = regexp.MustCompile(`^[a-z+]+://`)
	user   = regexp.MustCompile(`^[^@/]*@`)
	public = map[string]string{"github.com": "gh", "gitlab.com": "glab"}
)

// Local answers from the local directory.
type Local struct {
	dir    string
	remote Remote
	files  Files
	stdout io.Writer
}

// New builds the use case over the local directory dir.
func New(dir string, remote Remote, files Files, stdout io.Writer) *Local {
	return &Local{dir: dir, remote: remote, files: files, stdout: stdout}
}

// Host is the host of a git remote URL.
func Host(url string) string {
	host := user.ReplaceAllString(scheme.ReplaceAllString(url, ""), "")
	if i := strings.IndexAny(host, ":/"); i >= 0 {
		host = host[:i]
	}
	return host
}

// Tool is the forge CLI (gh, glab) for the host of the repository's origin, and
// that host. A named host that is neither in forges nor public is an error; an
// origin without host (a local path) gives an empty tool.
func (l *Local) Tool(repo string) (tool, host string, err error) {
	tool, host, err = l.lookup(repo)
	if err == nil && tool == "" && host != "" {
		err = fmt.Errorf("hôte %s absent de %s/forges", host, l.dir)
	}
	return tool, host, err
}

func (l *Local) lookup(repo string) (tool, host string, err error) {
	url, err := l.remote.OriginURL(repo)
	if err != nil {
		return "", "", err
	}
	tools := map[string]string{}
	for h, t := range public {
		tools[h] = t
	}
	if forges, err := l.files.Read(l.dir + "/forges"); err == nil {
		for _, line := range strings.Split(forges, "\n") {
			if f := strings.Fields(line); len(f) == 2 {
				tools[f[0]] = f[1]
			}
		}
	}
	host = Host(strings.TrimSpace(url))
	return tools[host], host, nil
}

// Forge prints the forge CLI (gh, glab) for the host of the repository's origin.
func (l *Local) Forge(repo string) (int, error) {
	tool, host, err := l.lookup(repo)
	if err != nil {
		return 1, err
	}
	if tool == "" {
		_, _ = fmt.Fprintf(l.stdout, "hôte %s inconnu\n", host)
		return 1, nil
	}
	_, _ = fmt.Fprintln(l.stdout, tool)
	return 0, nil
}

// CasReels prints the lines of cas-reels.md that name id.
func (l *Local) CasReels(id string) int {
	text, err := l.files.Read(l.dir + "/cas-reels.md")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		if strings.Contains(line, id) {
			_, _ = fmt.Fprintln(l.stdout, line)
		}
	}
	return 0
}

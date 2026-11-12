// Package gate holds the use cases of the red and green gates: red, green,
// characterize and replay. It reaches git, the test task and the file system
// through the ports below.
package gate

import "errors"

// Git runs git in a directory and returns its exit code and standard output.
type Git interface {
	Run(dir string, args ...string) (code int, stdout string)
}

// Files is the file system the gate reads and writes.
type Files interface {
	Read(path string) (string, error)
	Write(path, content string) error
	IsFile(path string) bool
	Remove(path string)
	MkdirAll(path string) error
	// Patches lists the *.patch files of a directory, sorted.
	Patches(dir string) []string
	// Resolve makes a path absolute, symbolic links followed.
	Resolve(path string) string
	// TempDir creates a directory and returns it with the function that deletes it.
	TempDir(prefix string) (dir string, cleanup func(), err error)
}

// TaskResult is the outcome of one `mise run`.
type TaskResult struct {
	Code     int
	Output   string
	TimedOut bool
}

// ErrNoMise reports that mise is not installed.
var ErrNoMise = errors.New("mise not found in PATH")

// Tasks runs a mise task in a directory, standard output and error merged.
type Tasks interface {
	Run(dir, task string, timeoutSeconds int) (TaskResult, error)
}

// TaskFile reads a mise.toml.
type TaskFile interface {
	// Depends maps each task of the file to its direct `depends`.
	Depends(toml string) map[string][]string
}

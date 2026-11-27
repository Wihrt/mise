package gate

import (
	"errors"
)

// The fakes below stand for git, the file system and mise: the use cases run
// without any process or file.

type fakeFiles map[string]string

func (f fakeFiles) Read(path string) (string, error) {
	if text, ok := f[path]; ok {
		return text, nil
	}
	return "", errors.New("no such file")
}
func (f fakeFiles) Write(path, content string) error {
	f[path] = content
	return nil
}
func (f fakeFiles) IsFile(path string) bool    { _, ok := f[path]; return ok }
func (f fakeFiles) Remove(path string)         { delete(f, path) }
func (f fakeFiles) Resolve(path string) string { return path }

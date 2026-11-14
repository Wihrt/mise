// Package architecture_test holds the architecture criteria ARC-001 to ARC-003.
// It reads the import graph of the module with `go list`; it imports nothing of it.
package architecture_test

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// allowed lists the third-party import paths the domain may depend on
// (a path and everything under it). Adding one is a criterion change of ARC-001.
var allowed = []string{}

const (
	domainDir   = "/internal/domain"
	appDir      = "/internal/app"
	adaptersDir = "/internal/adapters"
	rootDir     = "/cmd"
)

type pkg struct {
	ImportPath string
	Standard   bool
	Imports    []string
}

// graph is the module path and every package the module builds with, by import path.
func graph(t *testing.T) (string, map[string]pkg) {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		if filepath.Dir(root) == root {
			t.Fatal("go.mod not found")
		}
		root = filepath.Dir(root)
	}
	module := strings.TrimSpace(golist(t, root, "list", "-m"))
	pkgs := map[string]pkg{}
	dec := json.NewDecoder(strings.NewReader(golist(t, root, "list", "-deps", "-json=ImportPath,Standard,Imports", "./...")))
	for {
		var p pkg
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		pkgs[p.ImportPath] = p
	}
	return module, pkgs
}

func golist(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go %v: %v", args, err)
	}
	return string(out)
}

// under reports that path is dir or a package below it.
func under(path, dir string) bool { return path == dir || strings.HasPrefix(path, dir+"/") }

// reach lists every package from reaches through imports, with the chain that
// leads to it. It does not look inside a package for which stop is true.
func reach(pkgs map[string]pkg, from string, stop func(pkg) bool) map[string]string {
	chain := map[string]string{from: from}
	queue := []string{from}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current != from && stop(pkgs[current]) {
			continue
		}
		for _, next := range pkgs[current].Imports {
			if _, seen := chain[next]; !seen {
				chain[next] = chain[current] + " -> " + next
				queue = append(queue, next)
			}
		}
	}
	delete(chain, from)
	return chain
}

func TestARC001_DomainDependsOnlyOnItself(t *testing.T) {
	module, pkgs := graph(t)
	permitted := func(p pkg) bool {
		if p.Standard || under(p.ImportPath, module+domainDir) {
			return true
		}
		for _, a := range allowed {
			if under(p.ImportPath, a) {
				return true
			}
		}
		return false
	}
	// A standard or allowed package is a leaf: what it imports is its own business.
	leaf := func(p pkg) bool { return p.Standard || !under(p.ImportPath, module) }
	found := false
	for path := range pkgs {
		if !under(path, module+domainDir) {
			continue
		}
		found = true
		for dep, chain := range reach(pkgs, path, leaf) {
			if !permitted(pkgs[dep]) {
				t.Errorf("the domain depends on %s: %s", dep, chain)
			}
		}
	}
	if !found {
		t.Errorf("no package under %s%s", module, domainDir)
	}
}

func TestARC002_UseCasesKnowNoTechnology(t *testing.T) {
	module, pkgs := graph(t)
	// What the domain imports is judged by ARC-001, not here.
	stop := func(p pkg) bool { return !under(p.ImportPath, module) || under(p.ImportPath, module+domainDir) }
	found := false
	for path := range pkgs {
		if !under(path, module+appDir) {
			continue
		}
		found = true
		for dep, chain := range reach(pkgs, path, stop) {
			if under(dep, module+adaptersDir) || under(dep, module+rootDir) {
				t.Errorf("a use case depends on %s: %s", dep, chain)
			}
		}
	}
	if !found {
		t.Errorf("no package under %s%s", module, appDir)
	}
}

func TestARC003_OnlyTheCompositionRootAssembles(t *testing.T) {
	module, pkgs := graph(t)
	// adapter is the adapter a package belongs to, "" outside the adapters.
	adapter := func(path string) string {
		rest, ok := strings.CutPrefix(path, module+adaptersDir+"/")
		if !ok {
			return ""
		}
		name, _, _ := strings.Cut(rest, "/")
		return name
	}
	found := false
	for path, p := range pkgs {
		found = found || adapter(path) != ""
		// The domain and the use cases are judged by ARC-001 and ARC-002; the root may assemble.
		if !under(path, module) || under(path, module+domainDir) || under(path, module+appDir) || under(path, module+rootDir) {
			continue
		}
		for _, imported := range p.Imports {
			if a := adapter(imported); a != "" && a != adapter(path) {
				t.Errorf("%s imports the adapter %s: only %s%s may", path, imported, module, rootDir)
			}
		}
	}
	if !found {
		t.Errorf("no package under %s%s", module, adaptersDir)
	}
}

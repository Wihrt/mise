package gitlab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// glab puts a stand-in glab on the PATH: it logs its arguments, and answers each
// call with the file named after the last argument (slashes and ? turned into _).
func glab(t *testing.T, answers map[string]string) (*Forge, string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range answers {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	log := filepath.Join(dir, "log")
	script := "#!/bin/sh\necho \"$*\" >>" + log + "\nlast=\"\"; for a; do last=$a; done\n" +
		"n=$(printf %s \"$last\" | tr '/?' '__')\ncat " + dir + "/\"$n\" 2>/dev/null || exit 1\n" +
		"[ -f " + dir + "/\"$n.next\" ] && mv " + dir + "/\"$n.next\" " + dir + "/\"$n\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "glab"), []byte(script), 0o700); err != nil { //nolint:gosec // test stand-in
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return NewForge(dir, domain.Timing{Interval: time.Millisecond, Timeout: time.Second}), log
}

const mr12 = `{"iid":12,"web_url":"https://h.test/g/p/-/merge_requests/12","source_branch":"s","target_branch":"main","sha":"abc","state":"opened","draft":true,"has_conflicts":true,"detailed_merge_status":"conflict"}`

func TestGetAcceptsNumberBangAndUrlOfTheProject(t *testing.T) {
	f, _ := glab(t, map[string]string{"projects_:id_merge_requests_12": mr12})
	for _, ref := range []string{"12", "!12", "https://h.test/g/p/-/merge_requests/12"} {
		pr, err := f.Get(ref)
		if err != nil {
			t.Fatalf("%s: %v", ref, err)
		}
		want := domain.PullRequest{Number: 12, URL: "https://h.test/g/p/-/merge_requests/12", Head: "s", Base: "main", SHA: "abc", Open: true, Draft: true, Conflicts: true, MergeStatus: "conflict"}
		if pr != want {
			t.Errorf("%s: got %+v", ref, pr)
		}
	}
}

func TestGetRefusesTheUrlOfAnotherProject(t *testing.T) {
	f, _ := glab(t, map[string]string{"projects_:id_merge_requests_12": mr12})
	if _, err := f.Get("https://h.test/other/proj/-/merge_requests/12"); err == nil {
		t.Fatal("want an error")
	}
}

func TestMergeWaitsOutATransientStatus(t *testing.T) {
	path := "projects_:id_merge_requests_12"
	f, log := glab(t, map[string]string{
		path:            `{"state":"opened","detailed_merge_status":"ci_still_running"}`,
		path + ".next":  `{"state":"merged","detailed_merge_status":"mergeable"}`,
		path + "_merge": `{}`,
	})
	if err := f.Merge(12, "abc"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(log) //nolint:gosec // G304: path made by the test
	if !strings.Contains(string(b), "-X PUT -f sha=abc -f should_remove_source_branch=true projects/:id/merge_requests/12/merge") {
		t.Errorf("no frozen merge call:\n%s", b)
	}
}

func TestMergeNamesTheTimeout(t *testing.T) {
	f, _ := glab(t, map[string]string{"projects_:id_merge_requests_12": `{"state":"opened","detailed_merge_status":"checking"}`})
	f.timing.Timeout = 20 * time.Millisecond
	err := f.Merge(12, "abc")
	if err == nil || !strings.Contains(err.Error(), "checking") {
		t.Fatalf("got %v", err)
	}
}

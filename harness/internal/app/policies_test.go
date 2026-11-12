package app

import (
	"strings"
	"testing"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// fieldsParser reads a command as one invocation per ";", cut on blanks.
type fieldsParser struct{}

func (fieldsParser) Parse(command string) []domain.Invocation {
	var invs []domain.Invocation
	for _, seg := range strings.Split(command, ";") {
		invs = append(invs, domain.Invocation{Words: strings.Fields(seg)})
	}
	return invs
}

// fakeRepo is a repository whose every directory exists.
type fakeRepo struct {
	main     bool
	ignored  bool
	def, cur string
}

func (f fakeRepo) Exists(string) bool          { return true }
func (f fakeRepo) InMainCheckout(string) bool  { return f.main }
func (f fakeRepo) Ignored(string, string) bool { return f.ignored }
func (f fakeRepo) Default(string) string       { return f.def }
func (f fakeRepo) Current(string) string       { return f.cur }

type fakeKube string

func (f fakeKube) Current() string { return string(f) }

type fakeAgents map[string]string

func (f fakeAgents) Model(agentType string) string { return f[agentType] }

func bash(command string) domain.Request {
	return domain.Request{Tool: "Bash", Command: command, Cwd: "/repo"}
}

func check(t *testing.T, p Policy, req domain.Request, want domain.Verdict) {
	t.Helper()
	if got := p.Evaluate(req); got.Verdict != want {
		t.Errorf("Evaluate(%+v) = %v (%s), want %v", req, got.Verdict, got.Reason, want)
	}
}

func TestNeverMerge(t *testing.T) {
	feature := fakeRepo{def: "main", cur: "feat"}
	onMain := fakeRepo{def: "main", cur: "main"}
	tests := []struct {
		name    string
		repo    fakeRepo
		command string
		want    domain.Verdict
	}{
		{"forge merge", feature, "gh -R o/r pr merge 12", domain.Deny},
		{"forge merge help", feature, "glab mr merge --help", domain.Allow},
		{"forge view", feature, "gh pr view 12", domain.Allow},
		{"push a branch", feature, "git push -u origin feat", domain.Allow},
		{"bare push on the default branch", onMain, "git push", domain.Deny},
		{"refspec to the default branch", feature, "git push origin HEAD:refs/heads/main", domain.Deny},
		{"forced refspec to the default branch", feature, "git push origin +main", domain.Deny},
		{"force without lease", feature, "git push -uf origin feat", domain.Deny},
		{"force with lease", feature, "git push --force-with-lease origin feat", domain.Allow},
		{"merge on the default branch", onMain, "git merge feat", domain.Deny},
		{"merge on a branch", feature, "git merge origin/main", domain.Allow},
		{"second command of a line", feature, "git status; gh pr merge", domain.Deny},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, NewNeverMerge(fieldsParser{}, tt.repo), bash(tt.command), tt.want)
		})
	}
}

func TestRequireWorktree(t *testing.T) {
	edit := domain.Request{Tool: "Edit", Path: "/repo/a.go", Cwd: "/repo"}
	tests := []struct {
		name string
		repo fakeRepo
		req  domain.Request
		want domain.Verdict
	}{
		{"edit in the main checkout", fakeRepo{main: true}, edit, domain.Deny},
		{"edit of an ignored file", fakeRepo{main: true, ignored: true}, edit, domain.Allow},
		{"edit in a worktree", fakeRepo{}, edit, domain.Allow},
		{"commit in the main checkout", fakeRepo{main: true}, bash("git -c user.name=x commit -m y"), domain.Deny},
		{"commit in a worktree", fakeRepo{}, bash("git commit -m y"), domain.Allow},
		{"status in the main checkout", fakeRepo{main: true}, bash("git status"), domain.Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, NewRequireWorktree(fieldsParser{}, tt.repo), tt.req, tt.want)
		})
	}
}

func TestSubagentModel(t *testing.T) {
	agents := fakeAgents{"implementer": "sonnet", "planner": "opus"}
	agent := func(model, kind, description string) domain.Request {
		return domain.Request{Tool: "Agent", Model: model, SubagentType: kind, Description: description}
	}
	tests := []struct {
		name         string
		defaultModel string
		req          domain.Request
		want         domain.Verdict
	}{
		{"sonnet", "", agent("sonnet", "", ""), domain.Allow},
		{"opus", "", agent("claude-opus-5-5", "", "task review"), domain.Deny},
		{"opus for the final review", "", agent("opus", "", "Revue finale de la pile"), domain.Allow},
		{"fork", "", agent("sonnet", "fork", ""), domain.Deny},
		{"no model, agent file says sonnet", "", agent("", "implementer", ""), domain.Allow},
		{"no model, agent file says opus, no default", "", agent("", "planner", ""), domain.Deny},
		{"no model, default sonnet", "claude-sonnet-5-5", agent("", "", ""), domain.Allow},
		{"no model, default opus", "opus", agent("", "", ""), domain.Deny},
		{"another tool", "", domain.Request{Tool: "Bash", Model: "opus"}, domain.Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, NewSubagentModel(agents, tt.defaultModel), tt.req, tt.want)
		})
	}
}

func TestAgentGuard(t *testing.T) {
	write := func(path string) domain.Request {
		return domain.Request{Tool: "Write", Path: path, Cwd: "/repo"}
	}
	tests := []struct {
		name string
		role string
		req  domain.Request
		want domain.Verdict
	}{
		{"acceptance-writer writes a test", "acceptance-writer", write("pkg/calc_test.go"), domain.Allow},
		{"acceptance-writer writes under tests/", "acceptance-writer", write("/repo/tests/api/x.bru"), domain.Allow},
		{"acceptance-writer writes production code", "acceptance-writer", write("pkg/calc.go"), domain.Deny},
		{"implementer writes production code", "implementer", write("pkg/calc.go"), domain.Allow},
		{"implementer pushes", "implementer", bash("git -C /repo push"), domain.Deny},
		{"implementer commits", "implementer", bash("git commit -m x"), domain.Allow},
		{"mutant-writer commits", "mutant-writer", bash("git commit -m x"), domain.Deny},
		{"unknown role pushes", "reviewer", bash("git push"), domain.Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, NewAgentGuard(fieldsParser{}, tt.role), tt.req, tt.want)
		})
	}
}

func TestClusterGuard(t *testing.T) {
	tests := []struct {
		name    string
		context string
		command string
		want    domain.Verdict
	}{
		{"kubectl write", "prod", "kubectl -n app delete pod x", domain.Ask},
		{"kubectl read", "prod", "kubectl auth can-i create jobs", domain.Allow},
		{"kubectl dry run", "prod", "kubectl apply --dry-run=server -f a.yaml", domain.Allow},
		{"kubectl write on kind", "kind-dev", "kubectl apply -f a.yaml", domain.Allow},
		{"explicit context wins over the current one", "kind-dev", "kubectl --context=prod apply -f a.yaml", domain.Ask},
		{"rollout status", "prod", "kubectl rollout status deploy/x", domain.Allow},
		{"rollout restart", "prod", "kubectl rollout restart deploy/x", domain.Ask},
		{"helm template", "prod", "helm template app ./chart", domain.Allow},
		{"flux export", "prod", "flux create source git x --export", domain.Allow},
		{"argocd sync on kind", "kind-dev", "argocd app sync blog", domain.Ask},
		{"tofu plan", "prod", "tofu plan -out=plan", domain.Allow},
		{"tofu state rm", "prod", "tofu state rm a.b", domain.Ask},
		{"wrapped write", "prod", "NO_PROXY=x sudo helm upgrade app ./chart", domain.Ask},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, NewClusterGuard(fieldsParser{}, fakeKube(tt.context)), bash(tt.command), tt.want)
		})
	}
}

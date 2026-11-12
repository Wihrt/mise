package app

import (
	"slices"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// KubeContext gives the current context of the kubeconfig; "" when unknown.
type KubeContext interface {
	Current() string
}

// ClusterGuard hands every write on a cluster or on real infrastructure to the
// user. Reads, help, dry runs and writes on a kind cluster pass.
type ClusterGuard struct {
	parser CommandParser
	kube   KubeContext
}

// NewClusterGuard builds the policy.
func NewClusterGuard(parser CommandParser, kube KubeContext) *ClusterGuard {
	return &ClusterGuard{parser: parser, kube: kube}
}

var (
	// kubeFlags are the global options that take the next word as their value.
	kubeFlags = []string{"--context", "--kube-context", "-n", "--namespace", "--kubeconfig",
		"--as", "--cluster", "--user", "-s", "--server", "--token"}
	kubectlWrites = []string{"apply", "create", "patch", "replace", "delete", "edit", "label",
		"annotate", "scale", "set", "drain", "cordon", "uncordon", "taint"}
	rolloutWrites = []string{"restart", "undo", "pause", "resume"}
	helmWrites    = []string{"install", "upgrade", "uninstall", "delete", "rollback"}
	fluxWrites    = []string{"reconcile", "suspend", "resume", "delete", "create", "bootstrap", "install", "uninstall"}
	argoWrites    = []string{"sync", "delete", "set", "unset", "rollback", "create", "patch", "terminate-op"}
	tofuWrites    = []string{"apply", "destroy", "import", "taint", "untaint", "force-unlock"}
	stateWrites   = []string{"rm", "mv", "push"}
)

const needsGo = " A write needs your go for this exact command."

// Evaluate implements Policy.
func (p *ClusterGuard) Evaluate(req domain.Request) domain.Decision {
	if req.Tool != "Bash" {
		return domain.Allowed()
	}
	for _, inv := range p.parser.Parse(req.Command) {
		if d := p.judge(inv.Unwrap()); d.Verdict != domain.Allow {
			return d
		}
	}
	return domain.Allowed()
}

func (p *ClusterGuard) judge(inv domain.Invocation) domain.Decision {
	ops := inv.Operands(kubeFlags...)
	if len(ops) == 0 || inv.WantsHelp() {
		return domain.Allowed()
	}
	verb, next := ops[0], ""
	if len(ops) > 1 {
		next = ops[1]
	}
	dry := inv.HasArg("--dry-run", "--dry-run=client", "--dry-run=server")
	switch tool := inv.Name(); tool {
	case "kubectl":
		write := slices.Contains(kubectlWrites, verb) ||
			(verb == "rollout" && slices.Contains(rolloutWrites, next))
		if write && !dry && !p.local(inv) {
			return p.clusterWrite(inv, "kubectl "+verb)
		}
	case "helm":
		if slices.Contains(helmWrites, verb) && !dry && !p.local(inv) {
			return p.clusterWrite(inv, "helm "+verb)
		}
	case "flux":
		if slices.Contains(fluxWrites, verb) && !inv.HasArg("--export") && !p.local(inv) {
			return p.clusterWrite(inv, "flux "+verb)
		}
	case "argocd":
		if verb == "app" && slices.Contains(argoWrites, next) && !dry {
			return domain.Asked("Cluster write: argocd app " + next + "." + needsGo)
		}
	case "tofu", "terraform":
		if slices.Contains(tofuWrites, verb) {
			return domain.Asked("Infrastructure write: " + tool + " " + verb + "." + needsGo)
		}
		if verb == "state" && slices.Contains(stateWrites, next) {
			return domain.Asked("Infrastructure write: " + tool + " state " + next + "." + needsGo)
		}
	}
	return domain.Allowed()
}

// context is the kube context an invocation targets: its own flag, else the current one.
func (p *ClusterGuard) context(inv domain.Invocation) string {
	if c := inv.FlagValue("--context", "--kube-context"); c != "" {
		return c
	}
	return p.kube.Current()
}

// local reports an invocation aimed at a kind cluster.
func (p *ClusterGuard) local(inv domain.Invocation) bool {
	return strings.HasPrefix(p.context(inv), "kind-")
}

func (p *ClusterGuard) clusterWrite(inv domain.Invocation, what string) domain.Decision {
	return domain.Asked("Cluster write: " + what + " on context '" + p.context(inv) + "'." + needsGo)
}

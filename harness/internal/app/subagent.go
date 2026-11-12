package app

import (
	"regexp"
	"strings"

	"github.com/Wihrt/mise/harness/internal/domain"
)

// AgentModels gives the model an agent definition asks for; "" when it names none.
type AgentModels interface {
	Model(agentType string) string
}

// SubagentModel keeps sub-agents at Sonnet or below. A fork inherits the session
// model, so it is refused; only the final whole-branch review may run above Sonnet.
type SubagentModel struct {
	agents       AgentModels
	defaultModel string // CLAUDE_CODE_SUBAGENT_MODEL
}

// NewSubagentModel builds the policy; defaultModel is the model of an agent that names none.
func NewSubagentModel(agents AgentModels, defaultModel string) *SubagentModel {
	return &SubagentModel{agents: agents, defaultModel: defaultModel}
}

var finalReview = regexp.MustCompile(`(?i)final (whole-branch )?review|revue finale`)

// aboveSonnet reports a model more capable than Sonnet.
func aboveSonnet(model string) bool {
	return model == "opus" || model == "fable" ||
		strings.HasPrefix(model, "claude-opus-") || strings.HasPrefix(model, "claude-fable-")
}

// Evaluate implements Policy.
func (p *SubagentModel) Evaluate(req domain.Request) domain.Decision {
	if req.Tool != "Agent" && req.Tool != "Task" {
		return domain.Allowed()
	}
	if req.SubagentType == "fork" {
		return domain.Denied(`Forks inherit the session model and are blocked. Start a fresh agent with model "sonnet" or "haiku".`)
	}
	if req.Model == "" {
		if m := p.agentModel(req.SubagentType); m == "sonnet" || m == "haiku" {
			return domain.Allowed()
		}
		if p.defaultModel == "" || aboveSonnet(p.defaultModel) {
			return domain.Denied(`No model given and CLAUDE_CODE_SUBAGENT_MODEL is unset or above Sonnet. Pass model "sonnet" or "haiku".`)
		}
		return domain.Allowed()
	}
	if aboveSonnet(req.Model) && !finalReview.MatchString(req.Description) {
		return domain.Denied("Sub-agents run on Sonnet or Haiku, not " + req.Model +
			`. Only the final whole-branch review may use Opus (description must say "final review" or "revue finale").`)
	}
	return domain.Allowed()
}

func (p *SubagentModel) agentModel(agentType string) string {
	if agentType == "" {
		return ""
	}
	return p.agents.Model(agentType)
}

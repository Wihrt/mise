#!/usr/bin/env bash
# PreToolUse hook: keep sub-agents at Sonnet or below. A fork always inherits
# the session model, and an explicit model overrides CLAUDE_CODE_SUBAGENT_MODEL,
# so both are checked here. Only the final whole-branch review may run on Opus.
set -u

input=$(cat)
tool=$(jq -r '.tool_name // empty' <<<"$input")

deny() {
  jq -n --arg reason "$1" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: $reason
    }
  }'
  exit 0
}

case "$tool" in
  Agent|Task)
    type=$(jq -r '.tool_input.subagent_type // empty' <<<"$input")
    model=$(jq -r '.tool_input.model // empty' <<<"$input")
    description=$(jq -r '.tool_input.description // empty' <<<"$input")
    [ "$type" = "fork" ] && deny "Forks inherit the session model and are blocked. Start a fresh agent with model \"sonnet\" or \"haiku\"."
    if [ -z "$model" ]; then
      # No explicit model: the agent file's model, else CLAUDE_CODE_SUBAGENT_MODEL, decides.
      if [ -n "$type" ] && grep -Eq '^model: *(sonnet|haiku)' "$HOME/.claude/agents/$type.md" 2>/dev/null; then
        exit 0
      fi
      case "${CLAUDE_CODE_SUBAGENT_MODEL:-}" in
        ""|opus|fable|claude-opus-*|claude-fable-*)
          deny "No model given and CLAUDE_CODE_SUBAGENT_MODEL is unset or above Sonnet. Pass model \"sonnet\" or \"haiku\"."
          ;;
      esac
    fi
    case "$model" in
      opus|fable|claude-opus-*|claude-fable-*)
        grep -Eiq 'final (whole-branch )?review|revue finale' <<<"$description" && exit 0
        deny "Sub-agents run on Sonnet or Haiku, not $model. Only the final whole-branch review may use Opus (description must say \"final review\" or \"revue finale\")."
        ;;
    esac
    ;;
esac
exit 0

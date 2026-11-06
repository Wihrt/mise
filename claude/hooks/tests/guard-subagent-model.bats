#!/usr/bin/env bats
# Tests for guard-subagent-model.sh.
HOOK=$BATS_TEST_DIRNAME/../guard-subagent-model.sh

agent() { # tool [model] [subagent_type]
  jq -nc --arg t "$1" --arg m "${2:-}" --arg s "${3:-}" \
    '{tool_name:$t,tool_input:({prompt:"x"} + (if $m!="" then {model:$m} else {} end) + (if $s!="" then {subagent_type:$s} else {} end))}'
}
decision() {
  if jq -e '.hookSpecificOutput.permissionDecision == "deny"' >/dev/null 2>&1 <<<"$("$HOOK" <<<"$1")"; then echo deny; else echo allow; fi
}

@test "denies opus, fable and their full ids" {
  for m in opus fable claude-opus-5-5 claude-fable-5-1; do
    [ "$(decision "$(agent Agent "$m")")" = deny ]
  done
}
@test "allows sonnet and haiku" {
  for m in sonnet haiku; do [ "$(decision "$(agent Agent "$m")")" = allow ]; done
}
@test "denies fork" { [ "$(decision "$(agent Agent sonnet fork)")" = deny ]; }
@test "denies Task with opus" { [ "$(decision "$(agent Task opus)")" = deny ]; }
@test "allows opus for a final review" {
  j=$(jq -nc '{tool_name:"Agent",tool_input:{model:"opus",description:"Final review of branch"}}')
  [ "$(decision "$j")" = allow ]
}
@test "no model, env sonnet: allow" {
  CLAUDE_CODE_SUBAGENT_MODEL=claude-sonnet-5-5 run decision "$(agent Agent)"; [ "$output" = allow ]
}
@test "no model, env opus: deny" {
  CLAUDE_CODE_SUBAGENT_MODEL=claude-opus-5-5 run decision "$(agent Agent)"; [ "$output" = deny ]
}
@test "no model, env unset: deny" {
  unset CLAUDE_CODE_SUBAGENT_MODEL; [ "$(decision "$(agent Agent)")" = deny ]
}
@test "no model, agent file pinned to sonnet: allow" {
  unset CLAUDE_CODE_SUBAGENT_MODEL; [ "$(decision "$(agent Agent "" implementer)")" = allow ]
}
@test "no model, built-in agent, env unset: deny" {
  unset CLAUDE_CODE_SUBAGENT_MODEL; [ "$(decision "$(agent Agent "" general-purpose)")" = deny ]
}
@test "ignores other tools" {
  [ "$(decision '{"tool_name":"Bash","tool_input":{"command":"ls"}}')" = allow ]
}
@test "no model, agent file pinned to opus, env unset: deny" {
  mkdir -p "$BATS_TEST_TMPDIR/.claude/agents"
  printf -- '---\nmodel: opus\n---\n' >"$BATS_TEST_TMPDIR/.claude/agents/bigagent.md"
  unset CLAUDE_CODE_SUBAGENT_MODEL
  [ "$(HOME=$BATS_TEST_TMPDIR decision "$(agent Agent "" bigagent)")" = deny ]
}

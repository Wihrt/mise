#!/usr/bin/env bats
# Acceptance tests of the search-hint hook. Criteria: claude/TESTS.md, section recherche-semble.

hook() { "${HARNESS:?set HARNESS to the harness binary}" hook search-hint; }
bash_call() { jq -nc --arg k "$1" '{tool_name:"Bash",cwd:"/r",tool_input:{command:$k}}'; }
grep_call() { jq -nc --arg p "$1" '{tool_name:"Grep",cwd:"/r",tool_input:{pattern:$p}}'; }

@test "RCH-001 a search that reads like a question passes with a reminder of semble" {
  for call in "$(grep_call 'where are tokens refreshed')" "$(bash_call 'grep -rn "how is the stack merged" harness')" "$(bash_call 'rtk proxy rg "auth|login|session" src')"; do
    out=$(hook <<<"$call")
    [ "$(jq -r '.hookSpecificOutput.hookEventName' <<<"$out")" = PreToolUse ]
    jq -e '.hookSpecificOutput.additionalContext | contains("semble search")' <<<"$out"
    jq -e '.hookSpecificOutput | has("permissionDecision") | not' <<<"$out"
  done
}

@test "RCH-002 an exact search passes in silence" {
  for call in "$(grep_call 'TF_PLUGIN_CACHE_DIR')" "$(bash_call 'grep -rn "func NewMergeStack" harness')" \
    "$(bash_call 'rg -A 3 "permissionDecision.*ask" claude')" "$(bash_call 'git log --oneline -5')"; do
    run hook <<<"$call"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
  done
}

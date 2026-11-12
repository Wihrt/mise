#!/usr/bin/env bats
# Acceptance test of the never-merge rule for `harness merge-stack`.
# Criterion: claude/TESTS.md, section fusion-de-pile.

harness() { "${HARNESS:?set HARNESS to the harness binary}" "$@"; }

decision() {
  jq -nc --arg k "$1" --arg c "$BATS_TEST_TMPDIR" '{tool_name:"Bash",cwd:$c,tool_input:{command:$k}}' | harness hook never-merge |
    jq -r '.hookSpecificOutput.permissionDecision // "allow"' | grep . || echo allow
}

@test "FUS-009 merging a stack goes through the user's go" {
  [ "$(decision 'harness merge-stack 3')" = ask ]
  [ "$(decision '~/.local/bin/harness merge-stack --validate "mise run check" c')" = ask ]
  [ "$(decision 'harness merge-stack --dry-run 3')" = allow ]
  [ "$(decision 'gh pr merge 3 --merge')" = deny ]
  [ "$(decision 'glab mr merge 3')" = deny ]
}

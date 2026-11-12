#!/usr/bin/env bats
# Tests for the push-to-default-branch rule of never-merge.
hook() { "${HARNESS:?set HARNESS to the harness binary}" hook never-merge "$@"; }

setup_file() {
  R=$BATS_FILE_TMPDIR/r
  git init -q -b main "$R"
  git -C "$R" -c user.email=a@b -c user.name=a commit -q --allow-empty -m i
  git -C "$R" worktree add -q "$BATS_FILE_TMPDIR/wt" -b feat
}
cmd() { jq -nc --arg k "$1" --arg c "$2" '{tool_name:"Bash",cwd:$c,tool_input:{command:$k}}'; }
decision() {
  if jq -e '.hookSpecificOutput.permissionDecision == "deny"' >/dev/null 2>&1 <<<"$(hook <<<"$1")"; then echo deny; else echo allow; fi
}
check() { [ "$(decision "$(cmd "$2" "$BATS_FILE_TMPDIR/$3")")" = "$1" ]; }

@test "push on main" { check deny 'git push' r; }
@test "push origin on main" { check deny 'git push origin' r; }
@test "push origin main from feature" { check deny 'git push origin main' wt; }
@test "push HEAD:main from feature" { check deny 'git push origin HEAD:main' wt; }
@test "push -u origin feat" { check allow 'git push -u origin feat' wt; }
@test "push on feature" { check allow 'git push' wt; }
@test "git -C main push" { check deny "git -C $BATS_FILE_TMPDIR/r push" wt; }
@test "push feat:feat from main" { check allow 'git push origin feat:feat' r; }
@test "force flag together with --force-with-lease: allowed" { check allow 'git push -f --force-with-lease origin feat' wt; }
@test "force flag without lease: denied" { check deny 'git push -f origin feat' wt; }

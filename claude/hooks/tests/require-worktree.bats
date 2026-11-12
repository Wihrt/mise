#!/usr/bin/env bats
# Tests for require-worktree.
hook() { "${HARNESS:?set HARNESS to the harness binary}" hook require-worktree "$@"; }

load helpers

setup_file() {
  W=$BATS_FILE_TMPDIR
  git init -q -b main "$W/main"
  git -C "$W/main" -c user.email=a@b -c user.name=a commit -q --allow-empty -m init
  git -C "$W/main" worktree add -q "$W/wt" -b t
  echo ignored.txt >"$W/main/.gitignore"
  mkdir -p "$W/plain" "$W/empty"; git -C "$W/empty" init -q
}
edit() { jq -nc --arg t "$1" --arg c "$2" '{tool_name:$t,cwd:$c,tool_input:{file_path:($c+"/f")}}'; }
cmd() { jq -nc --arg k "$1" --arg c "$2" '{tool_name:"Bash",cwd:$c,tool_input:{command:$k}}'; }
decision() {
  if jq -e '.hookSpecificOutput.permissionDecision == "deny"' >/dev/null 2>&1 <<<"$(hook <<<"$1")"; then echo deny; else echo allow; fi
}

@test "edit tools: denied in main checkout" {
  for t in Edit Write NotebookEdit; do [ "$(decision "$(edit $t "$BATS_FILE_TMPDIR/main")")" = deny ]; done
}
@test "edit tools: allowed in linked worktree, outside git, empty repo" {
  for t in Edit Write NotebookEdit; do
    for d in wt plain empty; do [ "$(decision "$(edit $t "$BATS_FILE_TMPDIR/$d")")" = allow ]; done
  done
}
@test "commit: denied in main checkout" { [ "$(decision "$(cmd 'git commit -m x' "$BATS_FILE_TMPDIR/main")")" = deny ]; }
@test "commit: allowed in linked worktree" { [ "$(decision "$(cmd 'git commit -m x' "$BATS_FILE_TMPDIR/wt")")" = allow ]; }
@test "git -C into main checkout from elsewhere: denied" {
  [ "$(decision "$(cmd "git -C $BATS_FILE_TMPDIR/main commit -m x" "$BATS_FILE_TMPDIR/plain")")" = deny ]
}
@test "git -C into worktree from main cwd: allowed" {
  [ "$(decision "$(cmd "git -C $BATS_FILE_TMPDIR/wt commit -m x" "$BATS_FILE_TMPDIR/main")")" = allow ]
}
@test "non-commit command in main checkout: allowed" { [ "$(decision "$(cmd ls "$BATS_FILE_TMPDIR/main")")" = allow ]; }
@test "Read is ignored" { [ "$(decision "$(edit Read "$BATS_FILE_TMPDIR/main")")" = allow ]; }
@test "edit of a git-ignored file in the main checkout: allowed; tracked-style file: denied" {
  W=$BATS_FILE_TMPDIR
  j() { jq -nc --arg c "$W/main" --arg p "$1" '{tool_name:"Edit",cwd:$c,tool_input:{file_path:$p}}'; }
  [ "$(decide "$HARNESS" hook require-worktree <<<"$(j "$W/main/ignored.txt")")" = allow ]
  [ "$(decide "$HARNESS" hook require-worktree <<<"$(j "$W/main/other.txt")")" = deny ]
}

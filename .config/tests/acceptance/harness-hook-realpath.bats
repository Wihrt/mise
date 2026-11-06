#!/usr/bin/env bats
# HAR-002: edits through links to the main checkout are refused by require-worktree.sh

setup() {
  unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE
  ROOT="$(cd "$BATS_TEST_DIRNAME" && git rev-parse --show-toplevel)"
  HOOK="$ROOT/claude/hooks/require-worktree.sh"
  T="$(cd "$BATS_TEST_TMPDIR" && pwd -P)"
  MAIN="$T/main"; WT="$T/wt"; OUT="$T/outside"
  mkdir -p "$MAIN/sub" "$OUT"
  git -C "$MAIN" init -q -b main
  git -C "$MAIN" config user.email t@example.com
  git -C "$MAIN" config user.name t
  echo a > "$MAIN/sub/tracked.txt"
  git -C "$MAIN" add -A
  git -C "$MAIN" commit -q -m init
  git -C "$MAIN" worktree add -q -b feat "$WT"
  ln -s "$MAIN/sub/tracked.txt" "$OUT/filelink"
  ln -s "$MAIN/sub" "$OUT/dirlink"
  ln -s "$WT/sub/tracked.txt" "$OUT/wtlink"
  echo plain > "$OUT/plain.txt"
}

decide_for() { # tool path
  local json out
  json=$(jq -nc --arg t "$1" --arg p "$2" --arg c "$OUT" \
    '{tool_name:$t,tool_input:{file_path:$p},cwd:$c}')
  out=$(printf '%s' "$json" | "$HOOK") || { echo error; return; }
  if jq -e '.hookSpecificOutput.permissionDecision == "deny"' >/dev/null 2>&1 <<<"$out"; then echo deny; else echo allow; fi
}

@test "HAR-002 Edit through a file link to the main checkout is denied" {
  [ "$(decide_for Edit "$OUT/filelink")" = deny ]
}
@test "HAR-002 Write through a file link to the main checkout is denied" {
  [ "$(decide_for Write "$OUT/filelink")" = deny ]
}
@test "HAR-002 Edit through a directory link to the main checkout is denied" {
  [ "$(decide_for Edit "$OUT/dirlink/tracked.txt")" = deny ]
}
@test "HAR-002 Write through a directory link to the main checkout is denied" {
  [ "$(decide_for Write "$OUT/dirlink/tracked.txt")" = deny ]
}
@test "HAR-002 Edit through a link to the linked worktree is allowed" {
  [ "$(decide_for Edit "$OUT/wtlink")" = allow ]
}
@test "HAR-002 Write through a link to the linked worktree is allowed" {
  [ "$(decide_for Write "$OUT/wtlink")" = allow ]
}
@test "HAR-002 Edit of a plain file outside any repo is allowed" {
  [ "$(decide_for Edit "$OUT/plain.txt")" = allow ]
}
@test "HAR-002 Write of a plain file outside any repo is allowed" {
  [ "$(decide_for Write "$OUT/plain.txt")" = allow ]
}

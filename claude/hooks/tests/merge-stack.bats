#!/usr/bin/env bats
# Acceptance tests of `harness merge-stack`, against a throwaway repository and
# a stand-in forge (fake-forge/gh). Criteria: claude/TESTS.md, section fusion-de-pile.

harness() { "${HARNESS:?set HARNESS to the harness binary}" "$@"; }
g() { git -c user.email=a@b -c user.name=a "$@"; }

# A stack a <- b <- c of three open, green merge requests on main.
setup() {
  export GH_STATE=$BATS_TEST_TMPDIR/forge
  export PATH="$BATS_TEST_DIRNAME/fake-forge:$PATH"
  export HARNESS_POLL_INTERVAL=10ms HARNESS_CHECKS_GRACE=0s HARNESS_CHECKS_TIMEOUT=5s
  mkdir -p "$GH_STATE"
  git init -q --bare -b main "$GH_STATE/origin.git"
  W=$BATS_TEST_TMPDIR/work
  git clone -q "$GH_STATE/origin.git" "$W" 2>/dev/null
  cd "$W"
  g commit -q --allow-empty -m init
  g push -q origin main
  local base=main n=0 prs='[]'
  for b in a b c; do
    n=$((n + 1))
    git switch -q -c "$b"
    echo "$b" >"$b.txt"
    git add "$b.txt"
    g commit -q -m "feat: $b"
    g push -q origin "$b"
    prs=$(jq --argjson n "$n" --arg h "$b" --arg base "$base" --arg sha "$(git rev-parse HEAD)" \
      '. + [{number:$n, url:("https://forge/pr/"+($n|tostring)), headRefName:$h, baseRefName:$base, headRefOid:$sha, state:"OPEN", isDraft:false, mergeable:"MERGEABLE", checks:"pass", refuse:false}]' <<<"$prs")
    base=$b
  done
  echo "$prs" >"$GH_STATE/prs.json"
  echo pass >"$GH_STATE/main-checks"
  TOP_TREE=$(git rev-parse 'c^{tree}')
  MAIN_BEFORE=$(git rev-parse origin/main)
}

set_pr() { jq --argjson n "$1" "map(if .number == \$n then $2 else . end)" "$GH_STATE/prs.json" >"$GH_STATE/p" && mv "$GH_STATE/p" "$GH_STATE/prs.json"; }
state_of() { jq -r --argjson n "$1" '.[] | select(.number == $n) | .state' "$GH_STATE/prs.json"; }
origin_main() { git --git-dir "$GH_STATE/origin.git" rev-parse "main$1"; }
last_line() { [ "${lines[-1]}" = "$1" ] || { echo "last line: ${lines[-1]}"; return 1; }; }
has_line() { grep -qF -- "$1" <<<"$output" || { echo "missing: $1"; echo "$output"; return 1; }; }

@test "FUS-001 a stack is merged bottom first and main holds the code of its top" {
  run harness merge-stack 3
  [ "$status" -eq 0 ]
  has_line "STACK 3: #1 a -> #2 b -> #3 c"
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "MERGED MERGED MERGED" ]
  [ "$(origin_main '^{tree}')" = "$TOP_TREE" ]
  has_line "TREE IDENTICAL main is the code of #3"
  has_line "CHECKS PASS main"
  last_line "PASS"
}

@test "FUS-002 a single merge request is merged alone" {
  run harness merge-stack a
  [ "$status" -eq 0 ]
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "MERGED OPEN OPEN" ]
  [ "$(origin_main '^{tree}')" = "$(git rev-parse 'a^{tree}')" ]
  last_line "PASS"
}

@test "FUS-003 nothing is merged when a merge request of the stack has a red pipeline" {
  set_pr 2 '.checks = "fail"'
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "CHECKS FAIL #2"
  last_line "FAIL nothing merged"
  [ "$(origin_main)" = "$MAIN_BEFORE" ]
  [ "$(state_of 1)" = "OPEN" ]
}

@test "FUS-003 nothing is merged when the default branch has moved under the stack" {
  git switch -q main
  g commit -q --allow-empty -m "chore: elsewhere"
  g push -q origin main
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "OUTDATED #1"
  last_line "FAIL nothing merged"
  [ "$(state_of 1)" = "OPEN" ]
}

@test "FUS-004 the merge stops at the request that resists and says what is left" {
  set_pr 2 '.refuse = true'
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "STOPPED #2"
  has_line "MERGED: #1"
  has_line "REMAINING: #2 #3"
  last_line "FAIL stack partly merged"
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "MERGED OPEN OPEN" ]
}

@test "FUS-005 a default branch that is not the code of the top is reported" {
  touch "$GH_STATE/drift"
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "TREE DIFFERS main is not the code of #3: drift.txt"
  last_line "FAIL stack merged, main differs from the top of the stack"
}

@test "FUS-006 a red default branch after the merge is reported" {
  echo fail >"$GH_STATE/main-checks"
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "TREE IDENTICAL"
  has_line "CHECKS FAIL main"
  last_line "FAIL stack merged, main is red"
}

@test "FUS-007 without a pipeline the validation command decides" {
  echo none >"$GH_STATE/main-checks"
  worktrees=$(git worktree list)
  run harness merge-stack --validate 'test -f c.txt' 3
  [ "$status" -eq 0 ]
  has_line "VALIDATION PASS test -f c.txt"
  last_line "PASS"
  [ "$(git worktree list)" = "$worktrees" ]
}

@test "FUS-007 a failing validation command is reported" {
  echo none >"$GH_STATE/main-checks"
  run harness merge-stack --validate 'test -f absent.txt' 3
  [ "$status" -eq 1 ]
  has_line "VALIDATION FAIL test -f absent.txt"
  last_line "FAIL stack merged, main is red"
}

@test "FUS-007 without a pipeline nor a validation command the merge is not verified" {
  echo none >"$GH_STATE/main-checks"
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "CHECKS NONE"
  last_line "FAIL stack merged, main not verified"
}

@test "FUS-008 a dry run checks the stack and merges nothing" {
  run harness merge-stack --dry-run 3
  [ "$status" -eq 0 ]
  has_line "READY"
  last_line "PASS nothing merged (dry run)"
  [ "$(origin_main)" = "$MAIN_BEFORE" ]
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "OPEN OPEN OPEN" ]
}

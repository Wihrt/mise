#!/usr/bin/env bats
# Acceptance tests of `harness merge-stack`: the target of a stack, on both forges, against a
# throwaway repository and the stand-ins fake-forge/gh and fake-forge/glab (origin on
# forge.example.test for GitLab). Criteria: claude/TESTS.md, section cible-de-pile.

harness() { "${HARNESS:?set HARNESS to the harness binary}" "$@"; }
g() { git -c user.email=a@b -c user.name=a "$@"; }

setup() {
  export GH_STATE=$BATS_TEST_TMPDIR/gh-forge GL_STATE=$BATS_TEST_TMPDIR/gl-forge
  export PATH="$BATS_TEST_DIRNAME/fake-forge:$PATH"
  export GIT_SSH_COMMAND="$BATS_TEST_DIRNAME/fake-forge/ssh-wrapper/ssh"
  export CLAUDE_LOCAL_DIR=$BATS_TEST_TMPDIR/local
  export HARNESS_POLL_INTERVAL=10ms HARNESS_CHECKS_GRACE=0s HARNESS_CHECKS_TIMEOUT=5s
  mkdir -p "$GH_STATE" "$GL_STATE" "$CLAUDE_LOCAL_DIR"
  printf 'forge.example.test glab\n' >"$CLAUDE_LOCAL_DIR/forges"
  : >"$GL_STATE/calls.jsonl"
}

# forge_init <gh|glab>: a bare origin with main, cloned in $W.
forge_init() {
  FORGE=$1
  if [ "$FORGE" = gh ]; then STATE=$GH_STATE; else STATE=$GL_STATE; fi
  git init -q --bare -b main "$STATE/origin.git"
  W=$BATS_TEST_TMPDIR/work
  git clone -q "$STATE/origin.git" "$W" 2>/dev/null
  cd "$W"
  g commit -q --allow-empty -m init
  g push -q origin main
  echo pass >"$STATE/main-checks"
}

# branch_from <name> <base>: a branch with one commit on top of <base>, pushed.
branch_from() {
  git switch -q -c "$1" "origin/$2"
  echo "$1" >"$1.txt"
  git add "$1.txt"
  g commit -q -m "feat: $1"
  g push -q origin "$1"
}

# mr_json <number> <source> <target> <state> <sha>: one open-or-closed request, in the shape of the forge.
mr_json() {
  if [ "$FORGE" = gh ]; then
    jq -n --argjson n "$1" --arg h "$2" --arg base "$3" --arg st "$4" --arg sha "$5" \
      '{number:$n, url:("https://forge/pr/"+($n|tostring)), headRefName:$h, baseRefName:$base, headRefOid:$sha, state:($st|ascii_upcase), isDraft:false, mergeable:"MERGEABLE", checks:"pass", refuse:false}'
  else
    jq -n --argjson n "$1" --arg h "$2" --arg base "$3" --arg st "$4" --arg sha "$5" \
      '{iid:$n, web_url:("https://forge.example.test/grp/proj/-/merge_requests/"+($n|tostring)), source_branch:$h, target_branch:$base, sha:$sha, state:(if $st == "open" then "opened" else $st end), draft:false, has_conflicts:false, detailed_merge_status:"mergeable", head_pipeline:{id:$n, sha:$sha, status:"success", source:"push"}, refuse:false}'
  fi
}

# publish <mr_json>...: the requests of the forge, and the origin the harness reads.
publish() {
  if [ "$FORGE" = gh ]; then
    jq -s . <<<"$*" >"$STATE/prs.json"
  else
    jq -s . <<<"$*" >"$STATE/mrs.json"
    git remote set-url origin git@forge.example.test:grp/proj.git
  fi
}

state_of() {
  if [ "$FORGE" = gh ]; then
    jq -r --argjson n "$1" '.[] | select(.number == $n) | .state' "$STATE/prs.json"
  else
    jq -r --argjson n "$1" '.[] | select(.iid == $n) | .state' "$STATE/mrs.json"
  fi
}
origin_ref() { git --git-dir "$STATE/origin.git" rev-parse "$1"; }
last_line() { [ "${lines[-1]}" = "$1" ] || { echo "last line: ${lines[-1]}"; return 1; }; }
has_line() { grep -qF -- "$1" <<<"$output" || { echo "missing: $1"; echo "$output"; return 1; }; }

# FGL-011: !1 of fix into dev, dev being the head of no request.
stack_on_dev() {
  forge_init "$1"
  branch_from dev main
  branch_from fix dev
  FIX_SHA=$(git rev-parse HEAD)
  MAIN_BEFORE=$(origin_ref refs/heads/main)
  publish "$(mr_json 1 fix dev open "$FIX_SHA")"
}

check_stack_on_dev() {
  run harness merge-stack 1
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
  [ "$(state_of 1)" = "$MERGED" ] || { echo "$output"; return 1; }
  git --git-dir "$STATE/origin.git" merge-base --is-ancestor "$FIX_SHA" refs/heads/dev
  [ "$(origin_ref refs/heads/main)" = "$MAIN_BEFORE" ]
  has_line "TREE IDENTICAL dev"
  has_line "CHECKS PASS dev"
  last_line "PASS"
}

@test "FGL-011 a GitLab stack aimed at dev stops on dev" {
  MERGED=merged
  stack_on_dev glab
  check_stack_on_dev
}

@test "FGL-011 a GitHub stack aimed at dev stops on dev" {
  MERGED=MERGED
  stack_on_dev gh
  check_stack_on_dev
}

# FGL-012: !1 (a into main) closed without merge, branch a still there, !2 (b into a) open.
stack_on_closed_head() {
  forge_init "$1"
  branch_from a main
  A_SHA=$(git rev-parse HEAD)
  branch_from b a
  B_SHA=$(git rev-parse HEAD)
  MAIN_BEFORE=$(origin_ref refs/heads/main)
  publish "$(mr_json 1 a main closed "$A_SHA")" "$(mr_json 2 b a open "$B_SHA")"
}

check_refused_target() {
  run harness merge-stack 2
  [ "$status" -eq 1 ] || { echo "$output"; return 1; }
  [ "$(state_of 1) $(state_of 2)" = "$CLOSED $OPEN" ] || { echo "$output"; return 1; }
  [ "$(origin_ref refs/heads/main)" = "$MAIN_BEFORE" ]
  [ "$(origin_ref refs/heads/a)" = "$A_SHA" ]
  [[ "${lines[-1]}" == FAIL* ]] || { echo "last line: ${lines[-1]}"; return 1; }
  [[ "${lines[-1]}" == *"#1"* ]] || { echo "last line: ${lines[-1]}"; return 1; }
  [[ "${lines[-1]}" =~ (^|[^[:alnum:]_/.-])a([^[:alnum:]_/.-]|$) ]] || { echo "last line: ${lines[-1]}"; return 1; }
}

@test "FGL-012 a GitLab target that was the source branch of a closed request is refused" {
  CLOSED=closed OPEN=opened
  stack_on_closed_head glab
  check_refused_target
}

@test "FGL-012 a GitHub target that was the source branch of a closed request is refused" {
  CLOSED=CLOSED OPEN=OPEN
  stack_on_closed_head gh
  check_refused_target
}

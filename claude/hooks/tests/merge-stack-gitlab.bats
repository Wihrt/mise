#!/usr/bin/env bats
# Acceptance tests of `harness merge-stack` on GitLab, against a throwaway repository whose
# origin is on forge.example.test and a stand-in forge (fake-forge/glab). Criteria: claude/TESTS.md,
# section fusion-gitlab.

harness() { "${HARNESS:?set HARNESS to the harness binary}" "$@"; }
g() { git -c user.email=a@b -c user.name=a "$@"; }

# A stack !1 a <- !2 b <- !3 c of three open, green, mergeable merge requests on main.
setup() {
  export GL_STATE=$BATS_TEST_TMPDIR/forge
  export PATH="$BATS_TEST_DIRNAME/fake-forge:$PATH"
  export GIT_SSH_COMMAND="$BATS_TEST_DIRNAME/fake-forge/ssh-wrapper/ssh"
  export CLAUDE_LOCAL_DIR=$BATS_TEST_TMPDIR/local
  export HARNESS_POLL_INTERVAL=10ms HARNESS_CHECKS_GRACE=0s HARNESS_CHECKS_TIMEOUT=5s
  mkdir -p "$GL_STATE" "$CLAUDE_LOCAL_DIR"
  printf 'forge.example.test glab\n' >"$CLAUDE_LOCAL_DIR/forges"
  : >"$GL_STATE/calls.jsonl"
  git init -q --bare -b main "$GL_STATE/origin.git"
  W=$BATS_TEST_TMPDIR/work
  git clone -q "$GL_STATE/origin.git" "$W" 2>/dev/null
  cd "$W"
  g commit -q --allow-empty -m init
  g push -q origin main
  local base=main n=0 mrs='[]'
  for b in a b c; do
    n=$((n + 1))
    git switch -q -c "$b"
    echo "$b" >"$b.txt"
    git add "$b.txt"
    g commit -q -m "feat: $b"
    g push -q origin "$b"
    mrs=$(jq --argjson n "$n" --arg h "$b" --arg base "$base" --arg sha "$(git rev-parse HEAD)" \
      '. + [{iid:$n, web_url:("https://forge.example.test/grp/proj/-/merge_requests/"+($n|tostring)), source_branch:$h, target_branch:$base, sha:$sha, state:"opened", draft:false, has_conflicts:false, detailed_merge_status:"mergeable", head_pipeline:{id:$n, sha:$sha, status:"success", source:"push"}, refuse:false}]' <<<"$mrs")
    base=$b
  done
  echo "$mrs" >"$GL_STATE/mrs.json"
  echo pass >"$GL_STATE/main-checks"
  TOP_TREE=$(git rev-parse 'c^{tree}')
  MAIN_BEFORE=$(git rev-parse origin/main)
  git remote set-url origin git@forge.example.test:grp/proj.git
}

set_mr() { jq --argjson n "$1" "map(if .iid == \$n then $2 else . end)" "$GL_STATE/mrs.json" >"$GL_STATE/p" && mv "$GL_STATE/p" "$GL_STATE/mrs.json"; }
state_of() { jq -r --argjson n "$1" '.[] | select(.iid == $n) | .state' "$GL_STATE/mrs.json"; }
origin_main() { git --git-dir "$GL_STATE/origin.git" rev-parse "main$1"; }
last_line() { [ "${lines[-1]}" = "$1" ] || { echo "last line: ${lines[-1]}"; return 1; }; }
has_line() { grep -qF -- "$1" <<<"$output" || { echo "missing: $1"; echo "$output"; return 1; }; }
line_of() { grep -nF -- "$1" <<<"$output" | head -1 | cut -d: -f1; }
# Calls received by the stand-in that are not reads.
writes() { jq -c 'select(.method != "GET")' "$GL_STATE/calls.jsonl"; }
merge_calls() { jq -c 'select(.method == "PUT" and (.path | test("merge_requests/[0-9]+/merge$")))' "$GL_STATE/calls.jsonl"; }

@test "FGL-001 a GitLab stack is merged bottom first and main holds the code of its top" {
  run harness merge-stack 3
  [ "$status" -eq 0 ]
  l1=$(line_of "MERGED #1") l2=$(line_of "MERGED #2") l3=$(line_of "MERGED #3")
  [ -n "$l1" ] && [ -n "$l2" ] && [ -n "$l3" ] && [ "$l1" -lt "$l2" ] && [ "$l2" -lt "$l3" ] || { echo "$output"; return 1; }
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "merged merged merged" ]
  [ "$(origin_main '^{tree}')" = "$TOP_TREE" ]
  last_line "PASS"
}

@test "FGL-002 a single GitLab merge request is merged without the ones above it" {
  run harness merge-stack 1
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "merged opened opened" ]
  last_line "PASS"
}

@test "FGL-003 nothing is merged when a GitLab merge request has a failed pipeline" {
  set_mr 2 '.head_pipeline.status = "failed"'
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "opened opened opened" ]
  has_line "CHECKS FAIL #2"
  last_line "FAIL nothing merged"
}

@test "FGL-003 nothing is merged when the pipeline of a GitLab merge request is on another sha" {
  set_mr 2 '.head_pipeline.sha = "0000000000000000000000000000000000000000"'
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "opened opened opened" ]
  has_line "CHECKS FAIL #2"
  last_line "FAIL nothing merged"
}

@test "FGL-003 nothing is merged when a GitLab merge request has conflicts" {
  set_mr 2 '.has_conflicts = true'
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "opened opened opened" ]
  has_line "CONFLICTS #2"
  last_line "FAIL nothing merged"
}

@test "FGL-003 nothing is merged when GitLab does not say mergeable" {
  set_mr 2 '.detailed_merge_status = "not_approved"'
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "opened opened opened" ]
  has_line "NOT MERGEABLE #2: not_approved"
  last_line "FAIL nothing merged"
}

@test "FGL-003 nothing is merged when main has moved under the GitLab stack" {
  git switch -q main
  g commit -q --allow-empty -m "chore: elsewhere"
  g push -q origin main
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "opened opened opened" ]
  has_line "OUTDATED #1"
  last_line "FAIL nothing merged"
}

@test "FGL-004 a GitLab merge is frozen on the verified sha and removes the source branch" {
  run harness merge-stack 3
  [ "$(merge_calls | wc -l)" -eq 3 ] || { echo "$output"; cat "$GL_STATE/calls.jsonl"; return 1; }
  for n in 1 2 3; do
    sha=$(jq -r --argjson n "$n" '.[] | select(.iid == $n) | .sha' "$GL_STATE/mrs.json")
    [ "$(merge_calls | jq -r --arg p "merge_requests/$n/merge" 'select(.path | endswith($p)) | .fields.sha')" = "$sha" ] \
      || { echo "merge of !$n not frozen on $sha"; merge_calls; return 1; }
  done
  [ "$(merge_calls | jq -r '.fields.should_remove_source_branch' | sort -u)" = "true" ] || { merge_calls; return 1; }
}

@test "FGL-005 the GitLab merge stops at the request that resists and says what is left" {
  set_mr 2 '.refuse = true'
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "STOPPED #2"
  has_line "MERGED: #1"
  has_line "REMAINING: #2 #3"
  last_line "FAIL stack partly merged"
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "merged opened opened" ]
}

@test "FGL-006 after a GitLab merge a main that is not the code of the top is reported" {
  touch "$GL_STATE/drift"
  run harness merge-stack 3
  has_line "TREE DIFFERS"
  last_line "FAIL stack merged, main differs from the top of the stack"
}

@test "FGL-007 after a GitLab merge a red main is reported" {
  echo fail >"$GL_STATE/main-checks"
  run harness merge-stack 3
  [ "$status" -eq 1 ]
  has_line "CHECKS FAIL main"
  last_line "FAIL stack merged, main is red"
}

@test "FGL-008 without a GitLab pipeline a validation command that succeeds decides" {
  echo none >"$GL_STATE/main-checks"
  run harness merge-stack --validate 'test -f c.txt' 3
  has_line "VALIDATION PASS"
  last_line "PASS"
}

@test "FGL-008 without a GitLab pipeline a validation command that fails decides" {
  echo none >"$GL_STATE/main-checks"
  run harness merge-stack --validate 'test -f absent.txt' 3
  has_line "VALIDATION FAIL"
  last_line "FAIL stack merged, main is red"
}

@test "FGL-008 without a GitLab pipeline nor a validation command the merge is not verified" {
  echo none >"$GL_STATE/main-checks"
  run harness merge-stack 3
  has_line "CHECKS NONE"
  last_line "FAIL stack merged, main not verified"
}

@test "FGL-009 a GitLab dry run checks the stack and merges nothing" {
  run harness merge-stack --dry-run 3
  has_line "READY"
  [ "$(state_of 1) $(state_of 2) $(state_of 3)" = "opened opened opened" ]
  # The stand-in has seen the stack (so that the run really talked to it), and no write.
  [ -s "$GL_STATE/calls.jsonl" ] || { echo "$output"; return 1; }
  [ -z "$(writes)" ] || { echo "writes:"; writes; return 1; }
  last_line "PASS nothing merged (dry run)"
}

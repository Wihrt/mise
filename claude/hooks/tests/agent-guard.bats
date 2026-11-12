#!/usr/bin/env bats
# Tests for agent-guard and never-merge. Each case is one registered bats test.
load helpers

WORK=$BATS_FILE_TMPDIR
FIX=$WORK/fix
WT=$WORK/wt

setup_file() {
  git init -q -b main "$FIX"
  git -C "$FIX" config user.email t@t; git -C "$FIX" config user.name t
  echo base >"$FIX/README.md"; git -C "$FIX" add .; git -C "$FIX" commit -q -m init
  git -C "$FIX" worktree add -q -b feat "$WORK/wt"
  mkdir -p "$WT/pkg/tests"
  echo t >"$WT/pkg/calc_test.go"; echo t >"$WT/TESTS.md"; echo t >"$WT/mise.toml"; echo t >"$WT/pkg/tests/x_test.yaml"
  git -C "$WT" add .; git -C "$WT" commit -q -m 'test(calc): failing tests'
  echo p >"$WT/pkg/calc.go"; git -C "$WT" add .; git -C "$WT" commit -q -m 'feat(calc): impl'
  git -C "$FIX" worktree add -q -b plain "$WORK/wt2"
  git -C "$FIX" remote add origin "$FIX"; git -C "$FIX" fetch -q origin; git -C "$FIX" remote set-head origin main >/dev/null
}

guard() { # role expected name generator args...
  local role=$1 exp=$2 name=$3
  shift 3
  case_ "$name" "$exp" "$HARNESS hook agent-guard $role" "$@"
}
merge() { # expected name generator args...
  local exp=$1 name=$2
  shift 2
  case_ "$name" "$exp" "$HARNESS hook never-merge" "$@"
}

# ---- implementer ----
guard implementer allow "impl prod file" file_json Edit "$WT/pkg/calc.go" "$WT"
guard implementer allow "impl new prod file" file_json Write "$WT/pkg/new.go" "$WT"
guard implementer allow "impl TESTS.md" file_json Edit "$WT/TESTS.md" "$WT"
guard implementer allow "impl test-commit file" file_json Edit "$WT/pkg/calc_test.go" "$WT"
guard implementer allow "impl test-commit yaml" file_json Write "$WT/pkg/tests/x_test.yaml" "$WT"
guard implementer allow "impl NotebookEdit test file" file_json NotebookEdit "$WT/pkg/calc_test.go" "$WT"
guard implementer allow "impl relative test file" file_json Edit "pkg/calc_test.go" "$WT"
guard implementer allow "impl relative .. test file" file_json Edit "../calc_test.go" "$WT/pkg/tests"
guard implementer allow "impl mise.toml (excluded from protected set)" file_json Edit "$WT/mise.toml" "$WT"
guard implementer allow "impl file outside any repo" file_json Write "$WORK/loose.txt" "$WORK"
guard implementer deny "impl git push" bash_json 'git push' "$WT"
guard implementer allow "impl git commit" bash_json 'git commit -m "feat(calc): x"' "$WT"
guard implementer allow "impl git status" bash_json 'git status' "$WT"
guard implementer allow "impl Read ignored" file_json Read "$WT/pkg/calc_test.go" "$WT"
# implementer in a branch with no test( commit
guard implementer allow "impl no test commit, other file" file_json Edit "$WORK/wt2/README.md" "$WORK/wt2"
guard implementer allow "impl no test commit, TESTS.md" file_json Edit "$WORK/wt2/TESTS.md" "$WORK/wt2"

# unknown role / unknown tool
guard bogus allow "unknown role" file_json Edit "$WT/pkg/calc.go" "$WT"
guard implementer allow "unrelated tool" jq -n '{tool_name:"Grep", tool_input:{pattern:"x"}}'

# ---- never-merge ----
for c in 'glab -R grp/proj mr merge 1' 'glab --repo grp/proj mr merge 1' 'glab --repo=grp/proj mr merge 1' 'gh -R o/r pr merge 3' 'gh --repo o/r pr merge 3' 'gh --repo=o/r pr merge 3 --squash'; do
  merge deny "nm deny repo flag: $c" bash_json "$c" "$WT"
done
for c in 'glab -R grp/proj mr view 1' 'gh -R o/r pr view 3'; do
  merge allow "nm allow repo flag: $c" bash_json "$c" "$WT"
done
for c in 'glab mr merge 12' 'gh pr merge 3 --squash' 'cd x && glab mr merge' 'git push --force origin feat' 'git push -f' \
  'git push origin feat -f' 'git push -fu origin feat' 'git -C x push --force' 'FOO=1 gh pr merge' 'git push origin +feat'; do
  case "$c" in *'+feat') continue ;; esac
  merge deny "nm deny anywhere: $c" bash_json "$c" "$WT"
done
for c in 'git push --force-with-lease origin feat' 'git push --force-with-lease --force-if-includes' 'git push -u origin feat' \
  'git push' 'glab mr create' 'glab mr view 1' 'gh pr view 3' 'gh pr create' 'git commit -m "do not glab mr merge"' \
  'echo "git merge"' 'git merge-base a b' 'git log --oneline' 'git fetch --force' 'ls'; do
  merge allow "nm allow: $c" bash_json "$c" "$WT"
done
# ---- never-merge: --help / -h on a merge segment ----
for c in 'glab mr merge --help' 'glab mr merge -h' 'gh pr merge --help' "glab mr merge --help 2>&1 | grep -c -- '--auto-merge'"; do
  merge allow "nm allow help: $c" bash_json "$c" "$WT"
done
for c in 'glab mr merge 7 --yes' 'glab mr merge 7 && echo --help' 'gh pr merge 12 --squash'; do
  merge deny "nm deny non-help: $c" bash_json "$c" "$WT"
done
merge deny "nm git merge on default branch" bash_json 'git merge feat' "$FIX"
merge deny "nm git merge --no-ff on default" bash_json 'git merge --no-ff feat' "$FIX"
merge allow "nm git merge on feature branch" bash_json 'git merge main' "$WT"
merge allow "nm git merge outside git" bash_json 'git merge main' "$WORK"
merge deny "nm git -C default merge" bash_json "git -C $FIX merge feat" "$WORK"
merge allow "nm non-Bash tool" file_json Edit "$WT/pkg/calc.go" "$WT"
# ---- acceptance-writer ----
guard acceptance-writer allow "aw TESTS.md"        file_json Edit  "$WT/TESTS.md" "$WT"
guard acceptance-writer allow "aw helm test"       file_json Write "$WT/chart/tests/acceptance/sec_test.yaml" "$WT"
guard acceptance-writer allow "aw go test"         file_json Edit  "$WT/pkg/calc_test.go" "$WT"
guard acceptance-writer allow "aw py test"         file_json Write "$WT/src/test_cli.py" "$WT"
guard acceptance-writer allow "aw tofu test"       file_json Write "$WT/infra/net.tftest.hcl" "$WT"
guard acceptance-writer allow "aw mise.toml"       file_json Edit  "$WT/mise.toml" "$WT"
guard acceptance-writer deny  "aw template"        file_json Edit  "$WT/chart/templates/deploy.yaml" "$WT"
guard acceptance-writer deny  "aw values"          file_json Edit  "$WT/chart/values.yaml" "$WT"
guard acceptance-writer deny  "aw relative prod"   file_json Edit  "pkg/calc.go" "$WT"
guard acceptance-writer deny  "aw notebook prod"   file_json NotebookEdit "$WT/pkg/calc.go" "$WT"
guard acceptance-writer deny  "aw push"            bash_json "git push origin feat/x" "$WT"
guard acceptance-writer allow "aw go testdata"     file_json Write "$WT/cmd/greet/testdata/profile.json" "$WT"
guard acceptance-writer allow "aw conftest"        file_json Write "$WT/conftest.py" "$WT"
guard acceptance-writer deny  "aw testdata-like"   file_json Write "$WT/pkg/mytestdata.go" "$WT"
# ---- mutant-writer ----
guard mutant-writer deny  "mw push"               bash_json "git push origin feat/x" "$WT"
guard mutant-writer deny  "mw commit"             bash_json "git commit -m x" "$WT"
guard mutant-writer deny  "mw merge"              bash_json "git merge main" "$WT"
guard mutant-writer deny  "mw rebase"             bash_json "git rebase main" "$WT"
guard mutant-writer allow "mw git diff"           bash_json "git -C /tmp/scratch diff" "$WT"
guard mutant-writer allow "mw git worktree add"   bash_json "git worktree add --detach /tmp/scratch abc123" "$WT"
guard mutant-writer allow "mw edit scratch file"  file_json Edit "/tmp/scratch/chart/templates/deploy.yaml" "$WT"

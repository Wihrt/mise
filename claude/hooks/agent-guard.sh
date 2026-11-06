#!/usr/bin/env bash
# PreToolUse hook wired in the frontmatter of the implementer and acceptance-writer agents.
# Usage: agent-guard.sh implementer|acceptance-writer|mutant-writer
# Reads the hook JSON on stdin; neither role ever pushes. The implementer may edit
# anything; the acceptance-writer may only edit test files, TESTS.md, mise.toml and
# .gitignore (relative paths are resolved against the call's cwd).
set -u

role=${1:-}
input=$(cat)
tool=$(jq -r '.tool_name // empty' <<<"$input")
cwd=$(jq -r '.cwd // empty' <<<"$input")
[ -n "$cwd" ] || cwd=$PWD

deny() {
  jq -n --arg reason "$1" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: $reason
    }
  }'
  exit 0
}

GIT='(^|[^[:alnum:]_-])git( +(-c|-C) +[^ ]+)*'

case "$role" in
  implementer|acceptance-writer|mutant-writer) ;;
  *) exit 0 ;;
esac

case "$tool" in
  Edit|Write|NotebookEdit)
    [ "$role" = acceptance-writer ] || exit 0
    path=$(jq -r '.tool_input.file_path // .tool_input.notebook_path // empty' <<<"$input")
    [ -n "$path" ] || exit 0
    case "$path" in /*) ;; *) path=$cwd/$path ;; esac
    path=$(realpath -m -- "$path" 2>/dev/null) || deny "acceptance-writer: cannot resolve $path."
    # Only the part below the call's cwd counts: a parent directory named test/ proves nothing.
    root=$(realpath -m -- "$cwd" 2>/dev/null) || root=$cwd
    case "$path" in
      "$root"/*) ;;
      *) deny "acceptance-writer may only edit files under $root." ;;
    esac
    base=${path##*/}
    case "$base" in
      TESTS.md|mise.toml|.gitignore|*_test.*|test_*.py|conftest.py|*.tftest.hcl) exit 0 ;;
    esac
    rel=${path#"$root"/}
    case "/${rel%/*}/" in
      */test/*|*/tests/*|*/acceptance/*|*/testdata/*) exit 0 ;;
    esac
    deny "acceptance-writer may only edit test files, TESTS.md, mise.toml, .gitignore."
    ;;
  Bash)
    command=$(jq -r '.tool_input.command // empty' <<<"$input")
    [ -n "$command" ] || exit 0
    grep -Eq "$GIT +push( |\$)" <<<"$command" &&
      deny "$role must not push. The controller pushes."
    if [ "$role" = mutant-writer ]; then
      for verb in commit merge rebase; do
        grep -Eq "$GIT +$verb( |\$)" <<<"$command" && deny "mutant-writer must not $verb."
      done
    fi
    ;;
esac
exit 0

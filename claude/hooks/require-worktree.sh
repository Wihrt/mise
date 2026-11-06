#!/usr/bin/env bash
# PreToolUse hook: refuse edits and commits in the main checkout of a git
# repository, so that all work happens in a linked worktree.
set -u

input=$(cat)
tool=$(jq -r '.tool_name // empty' <<<"$input")

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

# Succeeds when directory $1 is in the main checkout of a git repository that
# already has commits. Fails outside git, in a linked worktree, or in an empty repo.
in_main_checkout() {
  local dir=$1 git_dir common_dir
  [ "$(git -C "$dir" rev-parse --is-inside-work-tree 2>/dev/null)" = "true" ] || return 1
  git -C "$dir" rev-parse --verify --quiet HEAD >/dev/null 2>&1 || return 1
  git_dir=$(git -C "$dir" rev-parse --path-format=absolute --git-dir 2>/dev/null) || return 1
  common_dir=$(git -C "$dir" rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || return 1
  [ "$git_dir" = "$common_dir" ]
}

case "$tool" in
  Edit|Write|NotebookEdit)
    path=$(jq -r '.tool_input.file_path // .tool_input.notebook_path // empty' <<<"$input")
    [ -n "$path" ] || exit 0
    dir=$(dirname "$path")
    while [ ! -d "$dir" ] && [ "$dir" != "/" ]; do
      dir=$(dirname "$dir")
    done
    in_main_checkout "$dir" || exit 0
    git -C "$dir" check-ignore -q "$path" 2>/dev/null && exit 0
    deny "Edits in the main checkout are blocked: $path. Create or enter a git worktree first, then edit the file there."
    ;;
  Bash)
    command=$(jq -r '.tool_input.command // empty' <<<"$input")
    grep -Eq '(^|[^[:alnum:]_-])git( +(-c|-C) +[^ ]+)* +commit( |$)' <<<"$command" || exit 0
    dir=$(jq -r '.cwd // empty' <<<"$input")
    c=$(sed -nE 's/.*git( +-c +[^ ]+)* +-C +([^ ]+).*/\2/p' <<<"$command" | head -1)
    if [ -n "$c" ]; then
      case "$c" in /*) dir=$c ;; *) dir="$dir/$c" ;; esac
    fi
    [ -d "$dir" ] || exit 0
    in_main_checkout "$dir" || exit 0
    deny "Commits from the main checkout are blocked: $dir. Create or enter a git worktree first, then commit there."
    ;;
esac
exit 0

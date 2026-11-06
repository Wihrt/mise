#!/usr/bin/env bash
# PreToolUse hook (Bash): never merge, never force-push without a lease.
# Denies `glab mr merge` and `gh pr merge` anywhere, `git merge` on the default
# branch, and `git push --force`/`-f` unless --force-with-lease is present.
set -u

input=$(cat)
[ "$(jq -r '.tool_name // empty' <<<"$input")" = "Bash" ] || exit 0
command=$(jq -r '.tool_input.command // empty' <<<"$input")
cwd=$(jq -r '.cwd // empty' <<<"$input")
[ -n "$command" ] || exit 0
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

# Default branch of the repo at $1: origin/HEAD, else main or master.
default_branch() {
  local ref
  ref=$(git -C "$1" symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null) && {
    echo "${ref#origin/}"
    return
  }
  for b in main master; do
    git -C "$1" show-ref --verify --quiet "refs/heads/$b" && { echo "$b"; return; }
  done
}

# Succeeds when segment $1 asks for help (exact word --help or -h): nothing is merged.
is_help() { grep -Eq '(^| )(--help|-h)( |$)' <<<"$1"; }

PREFIX='^[[:space:]]*[({]*[[:space:]]*([A-Za-z_][A-Za-z0-9_]*=[^ ]* +)*((command|env|sudo|exec) +)*'

while IFS= read -r seg; do
  [ -n "$seg" ] || continue
  if grep -Eq "${PREFIX}glab( +(-R|--repo)[ =][^ ]+| +-[^ ]+)* +mr +merge( |\$)" <<<"$seg" && ! is_help "$seg"; then
    deny "Merging is never done from here: glab mr merge is blocked. Stop at a green pipeline and hand over the MR link."
  fi
  if grep -Eq "${PREFIX}gh( +(-R|--repo)[ =][^ ]+| +-[^ ]+)* +pr +merge( |\$)" <<<"$seg" && ! is_help "$seg"; then
    deny "Merging is never done from here: gh pr merge is blocked. Stop at a green pipeline and hand over the PR link."
  fi
  if grep -Eq "${PREFIX}git( +(-c|-C) +[^ ]+)* +push( |\$)" <<<"$seg"; then
    if grep -Eq '(^| )(--force|-[[:alpha:]]*f[[:alpha:]]*)( |$)' <<<"$seg" &&
      ! grep -Eq -- '--force-with-lease' <<<"$seg"; then
      deny "Force pushes are blocked. Use git push --force-with-lease instead."
    fi
    dir=$cwd
    c=$(sed -nE 's/.*git( +-c +[^ ]+)* +-C +([^ ]+).*/\2/p' <<<"$seg")
    if [ -n "$c" ]; then
      case "$c" in /*) dir=$c ;; *) dir="$cwd/$c" ;; esac
    fi
    if [ -d "$dir" ]; then
      def=$(default_branch "$dir")
      cur=$(git -C "$dir" symbolic-ref --short HEAD 2>/dev/null)
      if [ -n "$def" ]; then
        # positional arguments after "push": [remote [refspec...]]
        args=$(sed -E 's/.*[ ]push( |$)//' <<<"$seg" | tr ' ' '\n' | grep -v '^-' | grep -v '^$')
        npos=$(grep -c . <<<"$args")
        if grep -Eq "^(\+?(refs/heads/)?$def|[^:]*:(refs/heads/)?$def)\$" <<<"$args" ||
          { [ "$cur" = "$def" ] && [ "$npos" -le 1 ]; }; then
          deny "Pushing to the default branch ($def) is blocked. Push a branch and go through a merge request."
        fi
      fi
    fi
  fi
  if grep -Eq "${PREFIX}git( +(-c|-C) +[^ ]+)* +merge( |\$)" <<<"$seg"; then
    dir=$cwd
    c=$(sed -nE 's/.*git( +-c +[^ ]+)* +-C +([^ ]+).*/\2/p' <<<"$seg")
    if [ -n "$c" ]; then
      case "$c" in /*) dir=$c ;; *) dir="$cwd/$c" ;; esac
    fi
    [ -d "$dir" ] || continue
    cur=$(git -C "$dir" symbolic-ref --short HEAD 2>/dev/null) || continue
    def=$(default_branch "$dir")
    if [ -n "$def" ] && [ "$cur" = "$def" ]; then
      deny "git merge on the default branch ($def) is blocked. Work on a branch and go through a merge request."
    fi
  fi
done < <(sed -E 's/(&&|\|\||;|\|)/\n/g' <<<"$command")
exit 0

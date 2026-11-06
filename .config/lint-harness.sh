#!/usr/bin/env bash
# Hygiene check of the versioned Claude Code harness (claude/, rtk/). Run from the repo root.
set -euo pipefail

dirs=()
for d in claude rtk; do [ -d "$d" ] && dirs+=("$d"); done
[ "${#dirs[@]}" -gt 0 ] || exit 0

fail=0
report() { printf 'lint:harness: %s: %s\n' "$1" "$2"; fail=1; }

while IFS= read -r f; do
  report "$f" "backup file"
done < <(find "${dirs[@]}" -name '*.bak*' -not -path '*/.git/*' | sort)

while IFS= read -r f; do
  report "$f" "contains /home/arnaud"
done < <(grep -rIl --exclude-dir=.git -e '/home/arnaud' "${dirs[@]}" 2>/dev/null |
  grep -v '^claude/skills/[^/]*/references/' | sort || true)

denylist=.config/harness-denylist.local
if [ -f "$denylist" ]; then
  patterns=$(mktemp)
  trap 'rm -f "$patterns"' EXIT
  grep -v '^[[:space:]]*$' "$denylist" >"$patterns" || true
  if [ -s "$patterns" ]; then
    rc=0
    matches=$(grep -rIl --exclude-dir=.git -E -f "$patterns" "${dirs[@]}" 2>/dev/null | sort) || rc=$?
    if [ "$rc" -ge 2 ] || { [ "$rc" -eq 1 ] && [ -n "$matches" ]; }; then
      report "$denylist" "invalid pattern or grep error"
    else
      while IFS= read -r f; do
        [ -n "$f" ] && report "$f" "matches a pattern of $denylist"
      done <<<"$matches"
    fi
  fi
else
  echo "lint:harness: warning: $denylist absent, pattern check skipped"
fi

exit "$fail"

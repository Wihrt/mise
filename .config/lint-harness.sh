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
  report "$f" "contains a personal path"
done < <(grep -ral --exclude-dir=.git -e '/home/arnaud' "${dirs[@]}" 2>/dev/null |
  grep -v '^claude/skills/[^/]*/references/' | sort || true)

denylist=.config/harness-denylist.local
if [ -f "$denylist" ]; then
  patterns=$(mktemp)
  trap 'rm -f "$patterns"' EXIT
  grep -v '^[[:space:]]*$' "$denylist" >"$patterns" || true
  if [ -s "$patterns" ]; then
    rc=0
    matches=$(grep -ral --exclude-dir=.git -E -f "$patterns" "${dirs[@]}" 2>/dev/null | sort) || rc=$?
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

# Every skill has a [dotfiles] entry and every harness entry has an existing source.
# config.toml is read as TOML (python3 >= 3.11, pinned in .config/mise.toml), not grepped.
python3 -I -c 'import tomllib' 2>/dev/null || {
  echo 'lint:harness: python3 with tomllib (>= 3.11) is required'
  exit 1
}
python3 -I - config.toml <<'PY' || fail=1
import os, sys, tomllib

with open(sys.argv[1], "rb") as fh:
    entries = tomllib.load(fh).get("dotfiles", {})
# An entry is either a bare source string or a table with a "source" key.
def source_of(v):
    return v if isinstance(v, str) else v.get("source") if isinstance(v, dict) else None

sources = {source_of(v) for v in entries.values()} - {None}
bad = False
if os.path.isdir("claude/skills"):
    for name in sorted(os.listdir("claude/skills")):
        # A skill is a directory holding a SKILL.md.
        if os.path.isfile(f"claude/skills/{name}/SKILL.md") and f"claude/skills/{name}" not in sources:
            print(f"lint:harness: claude/skills/{name}: no [dotfiles] entry")
            bad = True
for key, v in entries.items():
    src = source_of(v)
    if src and src.split("/")[0] in ("claude", "rtk") and not os.path.exists(src):
        print(f"lint:harness: {key}: source {src} does not exist")
        bad = True
sys.exit(1 if bad else 0)
PY

exit "$fail"

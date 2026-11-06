#!/usr/bin/env bats
# Acceptance tests for HAR-001, HAR-003 (mise dot apply on a temporary HOME)
# and HAR-004 (mise run lint:harness on a temporary copy of the repo).

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME" && git rev-parse --show-toplevel)"
  REAL_DATA="${MISE_DATA_DIR:-$HOME/.local/share/mise}"
  REAL_CACHE="${MISE_CACHE_DIR:-$HOME/.cache/mise}"
  cfg="$BATS_TEST_TMPDIR/cfg"
  home="$BATS_TEST_TMPDIR/home"
  mkdir -p "$cfg" "$home"
  (cd "$ROOT" && git ls-files -z --cached | while IFS= read -r -d '' f; do
    [ -e "$f" ] && printf '%s\0' "$f"
  done | xargs -0 cp --parents -t "$cfg")
  cfg="$(cd "$cfg" && pwd -P)"
  export MISE_YES=1
}

apply() {
  HOME="$home" MISE_CONFIG_DIR="$cfg" MISE_STATE_DIR="$BATS_TEST_TMPDIR/state" \
    MISE_DATA_DIR="$REAL_DATA" MISE_CACHE_DIR="$REAL_CACHE" \
    mise -C "$home" dot apply --yes
}

resolves_to() { # <target under home> <expected path under cfg>
  [ -e "$home/$1" ] || { echo "missing: $1"; return 1; }
  [ "$(realpath "$home/$1")" = "$(realpath "$cfg/$2")" ] || { echo "$1 -> $(realpath "$home/$1"), expected $cfg/$2"; return 1; }
}

@test "HAR-001 every harness target is a link resolving into the repo" {
  run apply
  [ "$status" -eq 0 ]
  for f in CLAUDE.md RTK.md settings.json agents hooks; do
    run resolves_to ".claude/$f" "claude/$f"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
  done
  n=0
  for d in "$cfg"/claude/skills/*/; do
    name="$(basename "$d")"
    run resolves_to ".claude/skills/$name" "claude/skills/$name"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    n=$((n + 1))
  done
  [ "$n" -gt 0 ]
  for f in config.toml filters.toml pipeline.toml pipelines; do
    run resolves_to ".config/rtk/$f" "rtk/$f"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
  done
  run jq . "$home/.claude/settings.json"
  [ "$status" -eq 0 ]
}

@test "HAR-003 third-party content in ~/.claude and ~/.config/rtk is left intact" {
  mkdir -p "$home/.claude/skills/synced/x" "$home/.config/rtk/llm-proxy" "$BATS_TEST_TMPDIR/other"
  echo "synced content" > "$home/.claude/skills/synced/x/SKILL.md"
  echo "other content" > "$BATS_TEST_TMPDIR/other/f"
  ln -s "$BATS_TEST_TMPDIR/other" "$home/.claude/skills/curl-md"
  echo "db content" > "$home/.config/rtk/llm-proxy/rtk-proxy.db"
  run apply
  [ "$status" -eq 0 ]
  [ "$(cat "$home/.claude/skills/synced/x/SKILL.md")" = "synced content" ]
  [ -L "$home/.claude/skills/curl-md" ]
  [ "$(readlink "$home/.claude/skills/curl-md")" = "$BATS_TEST_TMPDIR/other" ]
  [ "$(cat "$home/.config/rtk/llm-proxy/rtk-proxy.db")" = "db content" ]
}

lint() {
  cd "$cfg"
  MISE_STATE_DIR="$BATS_TEST_TMPDIR/state" MISE_TRUSTED_CONFIG_PATHS="$cfg" \
    MISE_DATA_DIR="$REAL_DATA" MISE_CACHE_DIR="$REAL_CACHE" run mise run lint:harness
}

@test "HAR-004 a skill directory without a [dotfiles] entry fails and is named" {
  mkdir -p "$cfg/claude/skills/zz-new"
  echo "# zz-new" > "$cfg/claude/skills/zz-new/SKILL.md"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"zz-new"* ]]
}

@test "HAR-004 a skill directory with a dot in its name without entry fails and is named" {
  mkdir -p "$cfg/claude/skills/a.b"
  echo "# a.b" > "$cfg/claude/skills/a.b/SKILL.md"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"a.b"* ]]
}

@test "HAR-004 a [dotfiles] entry whose source is missing fails and names the source" {
  printf '"~/.claude/zz-ghost" = "claude/zz-ghost-source"\n' >> "$cfg/config.toml"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"zz-ghost-source"* ]]
}

@test "HAR-004 the unchanged repository passes" {
  lint
  echo "$output"
  [ "$status" -eq 0 ]
}

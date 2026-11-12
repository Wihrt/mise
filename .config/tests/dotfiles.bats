#!/usr/bin/env bats
# Acceptance tests of the harness deployment: `mise dot apply` on a throwaway
# HOME, with this repository as the global mise config, and `lint:harness`.
# Criteria: docs/superpowers/specs/2026-11-06-harness-in-mise-design.md.

REPO=$(cd "$BATS_TEST_DIRNAME/../.." && pwd)

setup() {
  export HOME=$BATS_TEST_TMPDIR/home
  export MISE_CONFIG_DIR=$REPO
  export MISE_DATA_DIR=$BATS_TEST_TMPDIR/data MISE_CACHE_DIR=$BATS_TEST_TMPDIR/cache MISE_STATE_DIR=$BATS_TEST_TMPDIR/state
  mkdir -p "$HOME"
  mise trust -q "$REPO/config.toml"
}

# deployed <target under HOME> <source under the repository>
deployed() { [ "$(realpath "$HOME/$1")" = "$(realpath "$REPO/$2")" ] || { echo "$1 is not $2"; return 1; }; }

@test "HAR-001 on an empty home the harness is laid down as links to the repository" {
  run mise dot apply --yes
  [ "$status" -eq 0 ]
  for f in CLAUDE.md RTK.md settings.json agents; do deployed ".claude/$f" "claude/$f"; done
  for dir in "$REPO"/claude/skills/*/; do
    skill=$(basename "$dir")
    deployed ".claude/skills/$skill" "claude/skills/$skill"
  done
  for f in config.toml filters.toml pipeline.toml pipelines; do deployed ".config/rtk/$f" "rtk/$f"; done
  jq -e . "$HOME/.claude/settings.json" >/dev/null
}

@test "HAR-003 third-party content of ~/.claude and ~/.config/rtk is left alone" {
  mkdir -p "$HOME/.claude/skills/synced/x" "$HOME/.config/rtk/llm-proxy" "$HOME/elsewhere"
  echo synced >"$HOME/.claude/skills/synced/x/SKILL.md"
  ln -s "$HOME/elsewhere" "$HOME/.claude/skills/curl-md"
  echo db >"$HOME/.config/rtk/llm-proxy/rtk-proxy.db"
  run mise dot apply --yes
  [ "$status" -eq 0 ]
  [ "$(cat "$HOME/.claude/skills/synced/x/SKILL.md")" = synced ]
  [ "$(readlink "$HOME/.claude/skills/curl-md")" = "$HOME/elsewhere" ]
  [ "$(cat "$HOME/.config/rtk/llm-proxy/rtk-proxy.db")" = db ]
}

@test "HAR-004 a skill without a deployment entry fails the check" {
  run "$REPO/.config/lint-harness"
  [ "$status" -eq 0 ]

  copy=$BATS_TEST_TMPDIR/copy
  mkdir -p "$copy/claude/skills/new-skill" "$copy/rtk"
  cp "$REPO/config.toml" "$copy/config.toml"
  run "$REPO/.config/lint-harness" "$copy"
  [ "$status" -ne 0 ]
  [[ $output == *"skill without a [dotfiles] entry: new-skill"* ]]
  [[ $output == *"source does not exist: claude/CLAUDE.md"* ]]
}

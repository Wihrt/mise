#!/usr/bin/env bats
# Acceptance tests for HAR-006: `mise run lint:harness` black-box, on a temporary copy of the repo.

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME" && git rev-parse --show-toplevel)"
  FIX="$BATS_TEST_TMPDIR/repo"
  mkdir -p "$FIX"
  (cd "$ROOT" && git ls-files -z --cached | while IFS= read -r -d '' f; do
    [ -e "$f" ] && printf '%s\0' "$f"
  done | xargs -0 cp --parents -t "$FIX")
  rm -f "$FIX/.config/harness-denylist.local"
  mkdir -p "$FIX/claude" "$FIX/rtk"
  export MISE_YES=1
  (cd "$FIX" && mise trust --quiet .config/mise.toml >/dev/null 2>&1 || true)
}

lint() {
  cd "$FIX"
  run mise run lint:harness
}

@test "HAR-006 clean tree without local denylist succeeds and warns on exactly one line" {
  lint
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c 'harness-denylist.local')" -eq 1 ]
}

@test "HAR-006 clean tree with a local denylist succeeds" {
  printf 'zz-forbidden-host\\.example\n' > "$FIX/.config/harness-denylist.local"
  lint
  [ "$status" -eq 0 ]
}

@test "HAR-006 a .bak file under claude/ fails and is named" {
  mkdir -p "$FIX/claude/agents"
  echo x > "$FIX/claude/agents/notes.bak.1"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"claude/agents/notes.bak.1"* ]]
}

@test "HAR-006 a .bak file under rtk/ fails and is named" {
  echo x > "$FIX/rtk/config.toml.bak"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"rtk/config.toml.bak"* ]]
}

@test "HAR-006 /home/arnaud in a claude/ file fails and the file is named" {
  mkdir -p "$FIX/claude/agents"
  echo 'path: /home/arnaud/secret' > "$FIX/claude/agents/leaky.md"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"claude/agents/leaky.md"* ]]
}

@test "HAR-006 /home/arnaud in an rtk/ file fails and the file is named" {
  echo 'dir = "/home/arnaud/x"' > "$FIX/rtk/leaky.toml"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"rtk/leaky.toml"* ]]
}

@test "HAR-006 /home/arnaud in a skill references/ file is allowed" {
  mkdir -p "$FIX/claude/skills/zz-fixture/references"
  echo 'see /home/arnaud/docs' > "$FIX/claude/skills/zz-fixture/references/ok.md"
  lint
  [ "$status" -eq 0 ]
}

@test "HAR-006 a denylisted pattern fails, names the file and hides the pattern" {
  printf 'zz-forbidden-host\\.example\n' > "$FIX/.config/harness-denylist.local"
  mkdir -p "$FIX/claude/agents"
  echo 'connect to zz-forbidden-host.example now' > "$FIX/claude/agents/internal.md"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"claude/agents/internal.md"* ]]
  [[ "$output" != *"zz-forbidden-host"* ]]
}

@test "HAR-006 without local denylist the pattern check is skipped and one line signals it" {
  mkdir -p "$FIX/claude/agents"
  echo 'connect to zz-forbidden-host.example now' > "$FIX/claude/agents/internal.md"
  lint
  [ "$status" -eq 0 ]
}

@test "HAR-006 blank lines in the local denylist are ignored" {
  printf '\nzz-forbidden-host\\.example\n\n' > "$FIX/.config/harness-denylist.local"
  lint
  [ "$status" -eq 0 ]
}

@test "HAR-006 an invalid pattern in the local denylist fails and names the denylist file" {
  printf 'zz-bad(\n' > "$FIX/.config/harness-denylist.local"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"harness-denylist.local"* ]]
}

@test "HAR-006 a denylisted pattern under rtk/ fails, names the file and hides the pattern" {
  printf 'zz-forbidden-host\\.example\n' > "$FIX/.config/harness-denylist.local"
  echo 'host = "zz-forbidden-host.example"' > "$FIX/rtk/internal.toml"
  lint
  [ "$status" -ne 0 ]
  [[ "$output" == *"rtk/internal.toml"* ]]
  [[ "$output" != *"zz-forbidden-host"* ]]
}

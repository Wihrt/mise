#!/usr/bin/env bats
# Acceptance tests for HAR-005 (mise run claude-plugins with a fake claude).

PLUGIN="superpowers@anthropic-plugin-directory"

bats_require_minimum_version 1.5.0

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME" && git rev-parse --show-toplevel)"
  REAL_DATA="${MISE_DATA_DIR:-$HOME/.local/share/mise}"
  REAL_CACHE="${MISE_CACHE_DIR:-$HOME/.cache/mise}"
  cfg="$BATS_TEST_TMPDIR/cfg"
  home="$BATS_TEST_TMPDIR/home"
  fake="$BATS_TEST_TMPDIR/fakebin"
  nobin="$BATS_TEST_TMPDIR/nobin"
  calls="$BATS_TEST_TMPDIR/calls.log"
  mkdir -p "$cfg" "$home" "$fake" "$nobin"
  : > "$calls"
  (cd "$ROOT" && git ls-files -z --cached | while IFS= read -r -d '' f; do
    [ -e "$f" ] && printf '%s\0' "$f"
  done | xargs -0 cp --parents -t "$cfg")
  cfg="$(cd "$cfg" && pwd -P)"
  # the sandbox must not install the real claude tool: it would shadow the fake
  if [ -f "$cfg/config.toml" ]; then
    sed -i '/^[[:space:]]*claude[[:space:]]*=/d' "$cfg/config.toml"
  fi
  # PATH without mise shims/installs (which could provide a real claude)
  cleanpath="$(printf '%s' "$PATH" | tr ':' '\n' \
    | grep -v -e '/mise/shims' -e '/mise/installs/' | paste -sd: -)"
  export MISE_YES=1
}

# fake claude: logs argv (one line per call), `plugin list` prints $FAKE_LIST
make_fake() {
  cat > "$fake/claude" <<'SH'
#!/bin/sh
echo "$*" >> "$FAKE_CALLS"
if [ "$1 $2" = "plugin list" ]; then
  [ -n "$FAKE_FAIL" ] && { exit 1; }
  printf '%s\n' "$FAKE_LIST"
fi
exit 0
SH
  chmod +x "$fake/claude"
}

# lightweight PATH: symlinks to everything in /usr/bin and mise, except claude
make_nobin() {
  find /usr/bin /bin -maxdepth 1 \( -type f -o -type l \) ! -name claude \
    -exec ln -sf -t "$nobin" {} + 2>/dev/null || true
  ln -sf "$(command -v mise)" "$nobin/mise"
}

run_task() {
  local path="$1" list="$2"
  HOME="$home" MISE_CONFIG_DIR="$cfg" MISE_STATE_DIR="$BATS_TEST_TMPDIR/state" \
    MISE_TRUSTED_CONFIG_PATHS="$cfg" \
    MISE_DATA_DIR="$REAL_DATA" MISE_CACHE_DIR="$REAL_CACHE" \
    PATH="$path" FAKE_CALLS="$calls" FAKE_LIST="$list" FAKE_FAIL="${FAKE_FAIL:-}" \
    mise -C "$home" run claude-plugins
}

install_calls() { grep -c "^plugin install $PLUGIN\$" "$calls" || true; }

@test "HAR-005 plugin absent from claude plugin list: installed exactly once" {
  make_fake
  [ "$(PATH="$fake:$cleanpath" command -v claude)" = "$fake/claude" ]
  run run_task "$fake:$cleanpath" "other@somewhere"
  [ "$status" -eq 0 ]
  [ "$(install_calls)" -eq 1 ]
}

@test "HAR-005 plugin already listed: never installed" {
  make_fake
  [ "$(PATH="$fake:$cleanpath" command -v claude)" = "$fake/claude" ]
  run run_task "$fake:$cleanpath" "$PLUGIN"
  [ "$status" -eq 0 ]
  [ "$(install_calls)" -eq 0 ]
  ! grep -q "^plugin install" "$calls"
}

@test "HAR-005 no claude on PATH: task succeeds without calling it" {
  make_fake   # present only in $fake, which is NOT on PATH
  make_nobin
  [ ! -e "$nobin/claude" ]
  ! PATH="$nobin" command -v claude
  run run_task "$nobin" ""
  [ "$status" -eq 0 ]
  [ ! -s "$calls" ]
}

@test "HAR-005 claude plugin list fails: task succeeds, no install, warning on stderr" {
  make_fake
  [ "$(PATH="$fake:$cleanpath" command -v claude)" = "$fake/claude" ]
  # reference run: plugin list succeeds, stderr holds only mise's own output
  run --separate-stderr run_task "$fake:$cleanpath" "$PLUGIN"
  [ "$status" -eq 0 ]
  printf '%s\n' "$stderr" > "$BATS_TEST_TMPDIR/stderr.ok"
  : > "$calls"
  # failing run: the fake prints nothing on stderr itself
  FAKE_FAIL=1 run --separate-stderr run_task "$fake:$cleanpath" ""
  [ "$status" -eq 0 ]
  [ "$(install_calls)" -eq 0 ]
  ! grep -q "^plugin install" "$calls"
  # at least one stderr line that the reference run does not have: the task's warning
  printf '%s\n' "$stderr" > "$BATS_TEST_TMPDIR/stderr.fail"
  extra="$(grep -v -x -F -f "$BATS_TEST_TMPDIR/stderr.ok" "$BATS_TEST_TMPDIR/stderr.fail" | grep -c . || true)"
  [ "$extra" -ge 1 ]
}

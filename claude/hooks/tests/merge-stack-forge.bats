#!/usr/bin/env bats
# Acceptance tests of `harness merge-stack` forge selection. Criteria: specs under claude/docs/specs/ (no spec named for forge-hote).

harness() { "${HARNESS:?set HARNESS to the harness binary}" "$@"; }
g() { git -c user.email=a@b -c user.name=a "$@"; }

setup() {
  export CALLS=$BATS_TEST_TMPDIR/calls
  : >"$CALLS"
  mkdir -p "$BATS_TEST_TMPDIR/bin" "$BATS_TEST_TMPDIR/local"
  # Stand-ins that record any call, whatever the arguments.
  for tool in gh glab; do
    printf '#!/usr/bin/env bash\necho "%s $*" >>"%s"\nexit 1\n' "$tool" "$CALLS" >"$BATS_TEST_TMPDIR/bin/$tool"
    chmod +x "$BATS_TEST_TMPDIR/bin/$tool"
  done
  export PATH="$BATS_TEST_TMPDIR/bin:$PATH"
  export CLAUDE_LOCAL_DIR=$BATS_TEST_TMPDIR/local
  printf 'forge.example.test glab\n' >"$CLAUDE_LOCAL_DIR/forges"
  export HARNESS_POLL_INTERVAL=10ms HARNESS_CHECKS_GRACE=0s HARNESS_CHECKS_TIMEOUT=5s
  W=$BATS_TEST_TMPDIR/work
  git init -q -b main "$W"
  cd "$W"
  g commit -q --allow-empty -m init
  git remote add origin git@inconnu.example.test:grp/proj.git
}

@test "FGL-010 an unknown forge host is refused before any forge tool is called" {
  run harness merge-stack 1
  [ "$status" -eq 1 ]
  last=$(grep -v '^[[:space:]]*$' <<<"$output" | tail -1)
  [[ "$last" == *FAIL* ]] || { echo "last line: $last"; return 1; }
  [[ "$last" == *inconnu.example.test* ]] || { echo "last line: $last"; return 1; }
  [[ "$last" == *forges* ]] || { echo "last line: $last"; return 1; }
  [ ! -s "$CALLS" ] || { echo "called:"; cat "$CALLS"; return 1; }
}

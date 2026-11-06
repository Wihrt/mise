#!/usr/bin/env bats
# HAR-007..HAR-012: fixes for review remarks, each tested at its public boundary.

bats_require_minimum_version 1.5.0

setup() {
  unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE
  ROOT="$(cd "$BATS_TEST_DIRNAME" && git rev-parse --show-toplevel)"
  T="$(cd "$BATS_TEST_TMPDIR" && pwd -P)"
  export MISE_YES=1
  REAL_DATA="${MISE_DATA_DIR:-$HOME/.local/share/mise}"
  REAL_CACHE="${MISE_CACHE_DIR:-$HOME/.cache/mise}"
}

# Prints "deny" when the hook stdout carries a deny decision, "allow" otherwise.
decision() {
  [ -n "$1" ] || { echo allow; return; }
  jq -r '.hookSpecificOutput.permissionDecision // "allow"' <<<"$1" 2>/dev/null
}

# Runs mise in $1 with an isolated state dir, trusting only that directory.
mise_in() {
  local dir=$1; shift
  cd "$dir"
  MISE_STATE_DIR="$T/state" MISE_TRUSTED_CONFIG_PATHS="$dir" \
    MISE_DATA_DIR="$REAL_DATA" MISE_CACHE_DIR="$REAL_CACHE" run "$@"
}

# --- HAR-007 ---------------------------------------------------------------

guard_edit() {
  local payload
  payload=$(jq -n --arg cwd "$T/repo7" --arg p "$1" \
    '{tool_name: "Edit", cwd: $cwd, tool_input: {file_path: $p, old_string: "a", new_string: "b"}}')
  out=$(cd "$T/repo7" && "$ROOT/claude/hooks/agent-guard.sh" acceptance-writer <<<"$payload" 2>/dev/null)
}

setup_repo7() {
  mkdir -p "$T/repo7/pkg/tests"
  git -C "$T/repo7" init -q -b main
  echo 'package pkg' > "$T/repo7/pkg/calc.go"
  echo 'package tests' > "$T/repo7/pkg/tests/calc_test.go"
  ln -s ../calc.go "$T/repo7/pkg/tests/link.go"
}

@test "HAR-007 acceptance-writer editing pkg/tests/../calc.go is refused" {
  setup_repo7
  guard_edit "pkg/tests/../calc.go"
  [ "$(decision "$out")" = deny ]
}

@test "HAR-007 acceptance-writer editing a link under tests/ to a non-test file is refused" {
  setup_repo7
  guard_edit "pkg/tests/link.go"
  [ "$(decision "$out")" = deny ]
}

@test "HAR-007 acceptance-writer editing an ordinary file under tests/ is accepted" {
  setup_repo7
  guard_edit "pkg/tests/calc_test.go"
  [ "$(decision "$out")" = allow ]
}

@test "HAR-007 acceptance-writer editing an absolute tests/ path outside the cwd is refused" {
  setup_repo7
  mkdir -p "$T/other/tests"
  guard_edit "$T/other/tests/a_test.go"
  [ "$(decision "$out")" = deny ]
}

@test "HAR-007 acceptance-writer editing an absolute TESTS.md outside the cwd is refused" {
  setup_repo7
  mkdir -p "$T/other"
  guard_edit "$T/other/TESTS.md"
  [ "$(decision "$out")" = deny ]
}

# --- HAR-008 ---------------------------------------------------------------

merge_guard() {
  local payload
  payload=$(jq -n --arg cwd "$T/repo8" --arg c "$1" \
    '{tool_name: "Bash", cwd: $cwd, tool_input: {command: $c}}')
  out=$(cd "$T/repo8" && "$ROOT/claude/hooks/never-merge.sh" <<<"$payload" 2>/dev/null)
}

setup_repo8() {
  mkdir -p "$T/repo8"
  git -C "$T/repo8" init -q -b main
}

@test "HAR-008 a forced short-flag publish with a lease is refused" {
  setup_repo8
  merge_guard "git pu""sh -f --force-with-lease origin feat"
  [ "$(decision "$out")" = deny ]
}

@test "HAR-008 a forced long-flag publish with a lease is refused" {
  setup_repo8
  merge_guard "git pu""sh --force --force-with-lease origin feat"
  [ "$(decision "$out")" = deny ]
}

@test "HAR-008 a publish with only a lease is accepted" {
  setup_repo8
  merge_guard "git pu""sh --force-with-lease origin feat"
  [ "$(decision "$out")" = allow ]
}

# --- HAR-009 ---------------------------------------------------------------

# Runs, in repo $1, the command that the pre-commit hook `lint-secrets` declares.
run_secrets_hook() {
  local entry
  entry=$(awk '/- id: lint-secrets/{f=1;next} f&&/entry:/{sub(/^[^:]*entry: */,"");print;exit}' \
    "$ROOT/.pre-commit-config.yaml")
  [ -n "$entry" ]
  cd "$1"
  MISE_STATE_DIR="$T/state" MISE_TRUSTED_CONFIG_PATHS="$1" \
    MISE_DATA_DIR="$REAL_DATA" MISE_CACHE_DIR="$REAL_CACHE" run bash -c "$entry"
}

@test "HAR-009 the lint-secrets hook task fails on a staged token and passes without it" {
  local R="$T/sec"
  mkdir -p "$R/.config"
  git -C "$R" init -q -b main
  git -C "$R" config user.email t@example.com
  git -C "$R" config user.name t
  cp "$ROOT/.config/mise.toml" "$R/.config/mise.toml"
  cp "$ROOT/.gitleaks.toml" "$R/.gitleaks.toml"
  echo 'hello' > "$R/readme.txt"
  git -C "$R" add -A
  git -C "$R" commit -q -m init

  run_secrets_hook "$R"
  [ "$status" -eq 0 ]

  # Fake GitHub token built at runtime so this file itself is not flagged.
  printf 'token = "%s%s"\n' "ghp_" "aB3dE5gH7jK9mN1pQ3rS5tU7vW9xY1zA3bC5" > "$R/leak.txt"
  git -C "$R" add leak.txt
  run_secrets_hook "$R"
  [ "$status" -ne 0 ]
}

# --- HAR-010 ---------------------------------------------------------------

lint_copy() {
  FIX="$T/repo"
  mkdir -p "$FIX"
  (cd "$ROOT" && git ls-files -z --cached | while IFS= read -r -d '' f; do
    [ -e "$f" ] && printf '%s\0' "$f"
  done | xargs -0 cp --parents -t "$FIX")
  rm -f "$FIX/.config/harness-denylist.local"
  mkdir -p "$FIX/claude" "$FIX/rtk"
}

lint_run() {
  mise_in "$FIX" mise run lint:harness
}

@test "HAR-010 a binary file with the personal path fails, is named, and the path is not printed" {
  lint_copy
  printf 'bin\0 /home/%s\n' arnaud > "$FIX/claude/blob.bin"
  lint_run
  [ "$status" -ne 0 ]
  [[ "$output" == *"claude/blob.bin"* ]]
  [[ "$output" != *"/home/arnaud"* ]]
}

@test "HAR-010 a binary file with a local denylist pattern fails, is named, and the pattern is not printed" {
  lint_copy
  printf 'zz-forbidden-host\\.example\n' > "$FIX/.config/harness-denylist.local"
  printf 'bin\0 zz-forbidden-host.example\n' > "$FIX/claude/blob2.bin"
  lint_run
  [ "$status" -ne 0 ]
  [[ "$output" == *"claude/blob2.bin"* ]]
  [[ "$output" != *"zz-forbidden-host"* ]]
}

# --- HAR-011 ---------------------------------------------------------------

conftest_run() {
  cd "$ROOT"
  MISE_DATA_DIR="$REAL_DATA" MISE_CACHE_DIR="$REAL_CACHE" \
    run mise exec conftest -- conftest test --policy claude/skills/k8s-restricted/policy "$1"
}

@test "HAR-011 a bare Pod with hostNetwork is refused" {
  cat > "$T/pod.yaml" <<'Y'
apiVersion: v1
kind: Pod
metadata: {name: p}
spec:
  hostNetwork: true
  containers:
    - name: c
      image: registry.example/app:1
Y
  conftest_run "$T/pod.yaml"
  [ "$status" -ne 0 ]
  [[ "$output" == *"FAIL"* ]]
}

@test "HAR-011 a Deployment with runAsNonRoot true and runAsUser 0 is refused" {
  cat > "$T/dep.yaml" <<'Y'
apiVersion: apps/v1
kind: Deployment
metadata: {name: d}
spec:
  selector: {matchLabels: {a: b}}
  template:
    metadata: {labels: {a: b}}
    spec:
      containers:
        - name: c
          image: registry.example/app:1
          securityContext:
            runAsNonRoot: true
            runAsUser: 0
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities: {drop: [ALL]}
            seccompProfile: {type: RuntimeDefault}
Y
  conftest_run "$T/dep.yaml"
  [ "$status" -ne 0 ]
  [[ "$output" == *"FAIL"* ]]
}

@test "HAR-011 a bare Pod conforming to the restricted profile passes" {
  cat > "$T/ok.yaml" <<'Y'
apiVersion: v1
kind: Pod
metadata: {name: p}
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 10001
    seccompProfile: {type: RuntimeDefault}
  containers:
    - name: c
      image: registry.example/app:1
      securityContext:
        runAsNonRoot: true
        runAsUser: 10001
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities: {drop: [ALL]}
        seccompProfile: {type: RuntimeDefault}
Y
  conftest_run "$T/ok.yaml"
  [ "$status" -eq 0 ]
}

# --- HAR-012 ---------------------------------------------------------------

@test "HAR-012 the rules report names the world and cluster entities, not none" {
  cat > "$T/cnp.yaml" <<'Y'
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata: {name: x}
spec:
  endpointSelector: {}
  egress:
    - toEntities: [world]
  ingress:
    - fromEntities: [cluster]
Y
  cd "$ROOT"
  # uv supplies pyyaml, which the script needs; the script itself runs as documented.
  MISE_DATA_DIR="$REAL_DATA" MISE_CACHE_DIR="$REAL_CACHE" \
    run mise exec uv -- uv run --no-project --with pyyaml==6.0.3 python3 -I \
      claude/skills/helm-render-diff-check/scripts/policy-rules.py "$T/cnp.yaml"
  [ "$status" -eq 0 ]
  [[ "$output" == *"world"* ]]
  [[ "$output" == *"cluster"* ]]
  [[ "$output" != *"none"* ]]
}

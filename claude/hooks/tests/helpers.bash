# Shared helpers for the hook tests. Pure bash JSON builders: bats re-sources the
# test file for every test, so the load-time cost has to stay low.

_j() { local s=${1//\\/\\\\}; s=${s//\"/\\\"}; printf '%s' "$s"; }
file_json() { printf '{"tool_name":"%s","cwd":"%s","tool_input":{"file_path":"%s"}}' "$(_j "$1")" "$(_j "$3")" "$(_j "$2")"; }
bash_json() { printf '{"tool_name":"Bash","cwd":"%s","tool_input":{"command":"%s"}}' "$(_j "$2")" "$(_j "$1")"; }

# decide <hook...> < json  -> allow | deny | error
decide() {
  local out
  out=$("$@") || { echo error; return; }
  if jq -e '.hookSpecificOutput.permissionDecision == "deny"' >/dev/null 2>&1 <<<"$out"; then echo deny; else echo allow; fi
}

# case_ <name> <allow|deny> <hook command> <generator> [args...]: registers one bats
# test. The JSON is built lazily, inside the test, by running the generator
# (file_json / bash_json) so that loading the file forks nothing.
_CASE_N=0
declare -a _CASE_EXP _CASE_HOOK _CASE_GEN
case_() {
  _CASE_N=$((_CASE_N + 1))
  _CASE_EXP[$_CASE_N]=$2
  _CASE_HOOK[$_CASE_N]=$3
  printf -v "_CASE_GEN[$_CASE_N]" '%q ' "${@:4}"
  eval "case_fn_$_CASE_N() { run_case $_CASE_N; }"
  bats_test_function --description "$1" -- "case_fn_$_CASE_N"
}
run_case() {
  local json got
  eval "json=\$(${_CASE_GEN[$1]})"
  got=$(decide ${_CASE_HOOK[$1]} <<<"$json")
  [ "$got" = "${_CASE_EXP[$1]}" ] || { echo "expected ${_CASE_EXP[$1]}, got $got"; return 1; }
}

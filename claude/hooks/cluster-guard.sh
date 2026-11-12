#!/usr/bin/env bash
# PreToolUse hook (Bash): a write on a cluster or on real infrastructure needs the
# user's go for that exact command. Asks (permissionDecision "ask") before kubectl,
# helm and flux writes, `argocd app` writes, and tofu/terraform apply, destroy,
# import, taint and state changes. Reads, help, dry runs and `--export` pass, and
# so does a kubectl, helm or flux write on a kind cluster (context `kind-*`).
set -u

input=$(cat)
[ "$(jq -r '.tool_name // empty' <<<"$input")" = "Bash" ] || exit 0
command=$(jq -r '.tool_input.command // empty' <<<"$input")
[ -n "$command" ] || exit 0

ask() {
  jq -n --arg reason "$1" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "ask",
      permissionDecisionReason: $reason
    }
  }'
  exit 0
}

PREFIX='^[[:space:]]*[({]*[[:space:]]*([A-Za-z_][A-Za-z0-9_]*=[^ ]* +)*((command|env|sudo|exec) +)*'
# Global flags that take their value as the next word.
VALUE_FLAGS='--context|--kube-context|-n|--namespace|--kubeconfig|--as|--cluster|--user|-s|--server|--token'

# Index, in the words w, of the first word after the tool that is not a flag.
first_word() {
  local i=${1:-1} t
  while [ "$i" -lt "${#w[@]}" ]; do
    t=${w[$i]}
    case "$t" in
      -*=*) i=$((i + 1)) ;;
      -*) if grep -Eqx -- "$VALUE_FLAGS" <<<"$t"; then i=$((i + 2)); else i=$((i + 1)); fi ;;
      *) echo "$i"; return ;;
    esac
  done
}

# Kube context of the segment: --context / --kube-context, else the kubeconfig's current one.
context() {
  local c file
  c=$(sed -nE 's/.*--(kube-)?context[= ]+([^ ]+).*/\2/p' <<<"$seg")
  if [ -z "$c" ]; then
    file=${KUBECONFIG:-$HOME/.kube/config}
    file=${file%%:*}
    c=$(sed -nE 's/^current-context:[[:space:]]*"?([^"]*)"?[[:space:]]*$/\1/p' "$file" 2>/dev/null | head -1)
  fi
  echo "$c"
}

is_local() { case "$(context)" in kind-*) return 0 ;; *) return 1 ;; esac; }
is_help() { grep -Eq '(^| )(--help|-h)( |$)' <<<"$seg"; }
is_dry() { grep -Eq -- '(^| )--dry-run(=client|=server)?( |$)' <<<"$seg"; }

while IFS= read -r seg; do
  [ -n "$seg" ] || continue
  grep -Eq "${PREFIX}(kubectl|helm|flux|argocd|tofu|terraform)( |\$)" <<<"$seg" || continue
  read -ra w <<<"$(sed -E "s/${PREFIX}//" <<<"$seg")"
  tool=${w[0]}
  i=$(first_word)
  [ -n "$i" ] || continue
  verb=${w[$i]}
  next=${w[$((i + 1))]:-}
  is_help && continue
  case "$tool" in
    kubectl)
      case "$verb" in
        apply|create|patch|replace|delete|edit|label|annotate|scale|set|drain|cordon|uncordon|taint) ;;
        rollout) case "$next" in restart|undo|pause|resume) ;; *) continue ;; esac ;;
        *) continue ;;
      esac
      is_dry && continue
      is_local && continue
      ask "Cluster write: kubectl $verb on context '$(context)'. A write needs your go for this exact command."
      ;;
    helm)
      case "$verb" in install|upgrade|uninstall|delete|rollback) ;; *) continue ;; esac
      is_dry && continue
      is_local && continue
      ask "Cluster write: helm $verb on context '$(context)'. A write needs your go for this exact command."
      ;;
    flux)
      case "$verb" in reconcile|suspend|resume|delete|create|bootstrap|install|uninstall) ;; *) continue ;; esac
      grep -Eq -- '(^| )--export( |$)' <<<"$seg" && continue
      is_local && continue
      ask "Cluster write: flux $verb on context '$(context)'. A write needs your go for this exact command."
      ;;
    argocd)
      [ "$verb" = app ] || continue
      case "$next" in sync|delete|set|unset|rollback|create|patch|terminate-op) ;; *) continue ;; esac
      is_dry && continue
      ask "Cluster write: argocd app $next. A write needs your go for this exact command."
      ;;
    tofu|terraform)
      case "$verb" in
        apply|destroy|import|taint|untaint|force-unlock) ;;
        state) case "$next" in rm|mv|push) ;; *) continue ;; esac ;;
        *) continue ;;
      esac
      ask "Infrastructure write: $tool $verb${next:+ $next}. A write needs your go for this exact command."
      ;;
  esac
done < <(sed -E 's/(&&|\|\||;|\|)/\n/g' <<<"$command")
exit 0

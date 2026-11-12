#!/usr/bin/env bats
# Tests for cluster-guard.sh: writes on a cluster or on real infrastructure ask the user.
HOOK=$BATS_TEST_DIRNAME/../cluster-guard.sh

setup_file() {
  printf 'apiVersion: v1\ncurrent-context: prod-eu\n' >"$BATS_FILE_TMPDIR/prod"
  printf 'apiVersion: v1\ncurrent-context: kind-dev\n' >"$BATS_FILE_TMPDIR/kind"
}
cmd() { jq -nc --arg k "$1" '{tool_name:"Bash",cwd:"/tmp",tool_input:{command:$k}}'; }
# decision <kubeconfig name> <command> -> ask | allow
decision() {
  local out
  out=$(KUBECONFIG=$BATS_FILE_TMPDIR/$1 "$HOOK" <<<"$(cmd "$2")")
  if jq -e '.hookSpecificOutput.permissionDecision == "ask"' >/dev/null 2>&1 <<<"$out"; then echo ask; else echo allow; fi
}
check() { [ "$(decision "$2" "$3")" = "$1" ]; }

@test "kubectl apply on a real context" { check ask prod 'kubectl apply -f app.yaml'; }
@test "kubectl delete with flags before the verb" { check ask prod 'kubectl --context prod-eu -n app delete pod x'; }
@test "kubectl get" { check allow prod 'kubectl --context prod-eu -n app get pods'; }
@test "kubectl auth can-i create is a read" { check allow prod 'kubectl auth can-i create jobs -n app'; }
@test "kubectl apply --dry-run=server" { check allow prod 'kubectl apply --dry-run=server -f app.yaml'; }
@test "kubectl apply --dry-run=none is a write" { check ask prod 'kubectl apply --dry-run=none -f app.yaml'; }
@test "kubectl diff" { check allow prod 'kubectl diff -f app.yaml'; }
@test "kubectl rollout status" { check allow prod 'kubectl rollout status deploy/x'; }
@test "kubectl rollout restart" { check ask prod 'kubectl rollout restart deploy/x'; }
@test "kubectl apply after a pipe and an env prefix" { check ask prod 'helm template c | NO_PROXY=x kubectl apply -f -'; }
@test "kubectl apply on the current kind context" { check allow kind 'kubectl apply -f app.yaml'; }
@test "kubectl apply on an explicit kind context" { check allow prod 'kubectl --context kind-dev apply -f app.yaml'; }
@test "kubectl apply on an explicit real context from a kind default" { check ask kind 'kubectl --context=prod-eu apply -f app.yaml'; }
@test "kubectl apply --help" { check allow prod 'kubectl apply --help'; }
@test "a commit message naming kubectl delete" { check allow prod 'git commit -m "doc: kubectl delete is guarded"'; }
@test "helm upgrade" { check ask prod 'helm upgrade --install app ./chart -n app'; }
@test "helm upgrade --dry-run" { check allow prod 'helm upgrade --install app ./chart --dry-run'; }
@test "helm template" { check allow prod 'helm template app ./chart'; }
@test "helm uninstall on kind through --kube-context" { check allow prod 'helm --kube-context kind-dev uninstall app'; }
@test "flux reconcile" { check ask prod 'flux reconcile kustomization apps'; }
@test "flux get" { check allow prod 'flux get kustomizations -A'; }
@test "flux create --export" { check allow prod 'flux create source git x --url=u --export'; }
@test "argocd app sync" { check ask prod 'argocd app sync blog'; }
@test "argocd app sync on kind still asks" { check ask kind 'argocd app sync blog'; }
@test "argocd app diff" { check allow prod 'argocd app diff blog'; }
@test "tofu apply" { check ask prod 'tofu apply plan'; }
@test "tofu -chdir apply" { check ask prod 'tofu -chdir=stacks/dns destroy'; }
@test "tofu plan" { check allow prod 'tofu plan -out=plan'; }
@test "tofu state list" { check allow prod 'tofu state list'; }
@test "tofu state rm" { check ask prod 'tofu state rm aws_s3_bucket.x'; }
@test "terraform import" { check ask prod 'terraform import aws_s3_bucket.x name'; }

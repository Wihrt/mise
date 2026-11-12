---
name: k8s-restricted
description: Use when writing or reviewing Kubernetes manifests, Kustomize overlays or Helm chart output that must pass the restricted Pod Security Standard, or when adding the k8s lint and test tasks to a repo.
category: kubernetes
---

# Kubernetes manifests, PSS restricted

Complements the public `lukasniessen/kubernetes-skill@kubernetes-skill` (generic workload, NetworkPolicy, PDB and kubeconform guidance; install it, do not restate it). This page adds the checks it lacks: helm-unittest, `values.schema.json`, a policy check on the securityContext fields, and the task names.

## Pod and container (every workload, init containers and sidecars included)
- Pod `securityContext`: `runAsNonRoot: true`, numeric `runAsUser`, `runAsGroup`, `fsGroup` (never a name like `nonroot`), `seccompProfile: {type: RuntimeDefault}`.
- Each container `securityContext`: `allowPrivilegeEscalation: false`, `capabilities: {drop: [ALL]}` (add back only `NET_BIND_SERVICE`), `readOnlyRootFilesystem: true`. Every path the app writes (`/tmp`, caches) is an `emptyDir` mount.
- Forbidden: `privileged`, `hostNetwork`, `hostPID`, `hostIPC`, `hostPath` volumes, `hostPort`.
- Resources: `requests` and `limits` on every container. A ResourceQuota sums the declared values, not the usage; see "Resources".
- Probes: `readinessProbe` always, `livenessProbe` and `startupProbe` where the app needs them. A pod without a readiness probe counts as ready.
- Labels: `app.kubernetes.io/name` and `app.kubernetes.io/instance` on every object, and in selectors. These selectors are immutable on a Deployment.

## Namespace and network
- Namespace labels `pod-security.kubernetes.io/enforce`, `/audit` and `/warn` all `restricted`.
- A default-deny `NetworkPolicy` (`podSelector: {}`, both policyTypes), then one explicit allow per flow (DNS, ingress controller, each upstream). No allow rule just to silence a log.
- A `PodDisruptionBudget` when `replicas` > 1.

## Resources
- A ResourceQuota counts the `requests.*` and `limits.*` declared in pod specs; a pod declaring nothing counts 0. A refusal is `FailedCreate` with `exceeded quota`, not a crash loop.
- Size each declared value to the measured floor plus margin for a rolling update (one extra pod starts first). Before editing, read the quota (`kubectl -n <ns> describe resourcequota`) and the declarations of every workload in the namespace; fix them in one pass. A PVC does not relieve an ephemeral-storage quota. Removing a declaration admits the pod at 0 and drops node protection: name that trade-off first.

## Kustomize and Helm
- Kustomize: shared `base/`, one `overlays/<env>/` per environment; overlays patch, never copy the base.
- Helm: `values.schema.json` (generated, see helm-chart-authoring) with `additionalProperties: false`; helm-unittest suites in `tests/*_test.yaml` assert the restricted defaults on each workload kind. Chart organisation rules: helm-chart-authoring.

## Verification (names follow repo-tooling)
Tools in `mise.toml` `[tools]`, exact versions (verified: conftest 0.71.1, kubeconform 0.8.0, kustomize 5.8.3, helm 4.3.1). helm-unittest is a Helm plugin, not in the mise registry; install it inside the repo: `[env] HELM_PLUGINS = "{{config_root}}/.helm/plugins"` in `mise.toml`, `.helm/` in `.gitignore`, and `setup` and `test:helm` run `helm plugin list | grep -q unittest || helm plugin install https://github.com/helm-unittest/helm-unittest --version v1.2.1 --verify=false` (verified with helm 4.3.1: installs 1.2.1 under the given `HELM_PLUGINS`). CI gets it through `mise run test:helm`.
- `lint:k8s`, per overlay and per chart render: `helm lint --strict <chart>`; `kustomize build overlays/<env> | kubeconform -strict -summary -`; the same render `| conftest test --policy policy -`.
- `test:helm`: `helm unittest -t JUnit -o .tdd/junit.xml <chart>` (flags `--output-type`/`--output-file`, verified with unittest 1.2.1: it creates `.tdd/`; the JUnit file is what `harness gate` reads; `.tdd/` is gitignored). Test names carry the ID first: `it: "[ING-001] <what>"`. A failing suite prints ` FAIL  <suite>` then `- [ING-001] ...` on the next lines (verified on a `helm create` chart, unittest 1.2.1). (unverified: `helm lint --strict` and suites on a real production chart.)
- kubeconform validates the schema only, so it passes a Deployment lacking `allowPrivilegeEscalation: false`. The policy check is conftest with `policy/pss-restricted.rego`, copied from `~/.claude/skills/k8s-restricted/policy/pss-restricted.rego` into the repo (one `deny` per rule above: privilege escalation, read-only root, drop ALL, non-root, numeric uid, seccomp, host*, hostPath, hostPort). Pod-level values count for runAsNonRoot, runAsUser and seccompProfile.
- Proof run (kustomize 5.8.3, conftest 0.71.1): without the field, `conftest test` exits 1 with `container web: allowPrivilegeEscalation must be false` while `kubeconform -strict` reports Valid: 1; with the field set, 10 tests pass.
- Do not run `kubectl apply` to check; see readonly-cluster-and-secret-inspection.

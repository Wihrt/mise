---
name: gitops-flux-argocd
description: Use when changing what a GitOps controller deploys (Argo CD Application, ApplicationSet or app-of-apps values; Flux Kustomization, HelmRelease or source), reading a sync or reconcile status, or reviewing a change to the branch a cluster follows.
category: kubernetes
---

Git is the only write path to the cluster: the merge is the deployment. Render and diff before the merge request, and read the controller's status against the commit you expect.

## Both controllers
- The branch the controller follows is production. Never repair the cluster by hand (`kubectl apply|edit|patch`, `helm upgrade`): the controller reverts it, or the cluster drifts from Git. The hook `cluster-guard.sh` asks the user before any such write.
- Find the owner of an object before changing it: Argo CD sets `argocd.argoproj.io/tracking-id` (or the `app.kubernetes.io/instance` label); Flux sets `kustomize.toolkit.fluxcd.io/name` or `helm.toolkit.fluxcd.io/name`. Change the source that owner reads, not the object.
- Render what the controller renders, with the same chart version, the same values files in the same order and the same namespace: `helm template <release> <chart> --version <v> -n <ns> -f <values>...`, then `kubeconform`. Diff the render before and after the change (`helm-render-diff-check`).
- "Synced" or "Ready" is a claim about one revision. Compare the revision the controller applied with the commit expected, and quote both, before saying a change is deployed.
- Order: CRDs before the resources that use them, operators before their instances. Argo CD: `argocd.argoproj.io/sync-wave`. Flux: `dependsOn` with `wait: true` or `healthChecks`.
- Pruning: removing a manifest deletes the object (Argo CD `syncPolicy.automated.prune`, Flux `prune: true`). Before removing or renaming one that owns data (PVC, database cluster, namespace), state what will be deleted and get the user's go. Protect such objects: `argocd.argoproj.io/sync-options: Prune=false`, or `kustomize.toolkit.fluxcd.io/prune: disabled`.
- A rename is a delete plus a create: a renamed release, Application or namespace recreates its workloads and loses its PVCs.
- State set outside Git (auto-sync switched off in the UI, a parameter override, a suspended Kustomization) is invisible in the repo. Read the live object before a risky change, and say when it differs from the render.
- No secret value in Git: an ExternalSecret reference, or SOPS with Flux decryption.

## Argo CD
- Two statuses: sync (`Synced`, `OutOfSync`) and health (`Healthy`, `Progressing`, `Degraded`). `Synced` and `Degraded` together means the manifests applied and the workload fails: read the pod events.
- Diff before merging: `argocd app diff <app> --revision <sha>` (a read). `argocd app sync` is a write.
- Revision check: `.status.sync.revision` and `.status.operationState.syncResult.revision` against `git rev-parse` of the branch; with several sources, compare the lists entry by entry.
- A field another controller owns (HPA replicas, a webhook `caBundle`, defaulted fields under server-side apply) is declared in `ignoreDifferences`; do not disable `selfHeal` for it.
- ApplicationSet: a generator change can delete Applications and their resources. Render the generated list before and after.
- App of apps: the child Application's name comes from the values key; changing the key deletes one Application and creates another.

## Flux
- Chain: source (`GitRepository`, `OCIRepository`, `HelmRepository`) → `Kustomization` or `HelmRelease`. Read each link: `flux get all -A`, then `flux events --for Kustomization/<name> -n <ns>`; `Ready=False` carries the reason.
- Revision check: `.status.lastAppliedRevision` of the Kustomization, or `.status.history[0]` of the HelmRelease, against the commit.
- A HelmRelease `Stalled` with `RetriesExceeded` does not recover alone: fix the cause in Git, then it needs a reconcile, which is a write (`flux reconcile helmrelease <name> --force`, user's go).
- `flux suspend` leaves the object drifting and is not in Git: say it in the report, and resume.
- `postBuild.substitute` replaces every `${var}` in the manifests: escape a literal one as `$${var}`.
- `dependsOn` orders objects of the same kind only; `timeout` shorter than the real rollout marks a healthy release as failed.

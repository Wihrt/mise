---
name: renovate-mr-review
description: Use when reviewing, fixing or deciding on a Renovate merge request, or on any version bump of a chart, image, operator, CLI tool, CI action or OpenTofu provider.
category: verification
---

A version bump is reviewed like code: read what changed between the two versions, render the effect on this repo, and give a verdict with evidence. Never merge; a green pipeline is necessary, not sufficient.

## Read
- Name what moves: package, datasource, from and to, update type (digest, patch, minor, major), and every package of a grouped MR.
- Read the release notes of every version between the two, not only the target: breaking changes, removed or renamed values, changed defaults, required migration steps. The MR body is a summary; for a minor or a major, open the project's upgrade guide.
- Coupled versions move together or not at all: an operator and its CRDs, a chart and the CLI pinned to its `appVersion` in `mise.toml`, a client and its server. Say which pin the MR leaves behind.

## By kind
- Helm chart: diff the default values (`helm show values <chart> --version <old>` against `<new>`), then render this repo's releases before and after and diff them (`helm-render-diff-check`). Expected noise: `helm.sh/chart`, image tags, checksums. Findings: a new or removed object, a changed selector or another immutable field (Deployment selector, StatefulSet `volumeClaimTemplates`, Job template: the upgrade fails or recreates), a value this repo sets that the chart no longer reads.
- CRDs: Helm never upgrades the CRDs of a chart's `crds/` directory. Check how this repo delivers them, whether an API version is removed, and whether the operator supports skipping minors.
- Image: keep the digest pin; check the registry and repository did not change; check the tag exists for the architectures in use (`skopeo inspect --raw`). A major of a data store (PostgreSQL, a message broker) is a data migration, never a plain bump: verdict "hold", with the procedure (`cnpg-operations` for CloudNativePG).
- Kubernetes compatibility: the chart's `kubeVersion` and the APIs it renders against the cluster's version.
- CI action: pinned by commit sha; check the sha is the tag's (`gh api repos/<owner>/<repo>/git/ref/tags/<tag>`).
- OpenTofu provider: read the upgrade guide for a major; `tofu init -upgrade`, then `tofu plan` shows no change, or only the ones the notes announce; the lock file is updated in the same MR for every platform in use (`tofu providers lock -platform=...`).
- CLI tool in `mise.toml`: `mise run check` green, and the notes read for changed defaults or flags the tasks use.

## Verdict
- One comment on the MR: `safe`, `needs change` or `hold`, then the evidence (notes read with links, the render diff, the commands run), then what was not verified. The live upgrade is never verified by the pipeline: say so.
- `needs change`: push the fix as a commit on the Renovate branch (values adapted, a pin moved with it), not a second MR.
- `hold`: write the manual steps and their order in the MR. When one package blocks a group, ask the user whether to split the group.
- Do not propose automerge, a schedule or a new grouping rule unless the user asks.

---
name: dockerfile-restricted
description: Use when writing or reviewing a Dockerfile for an image that runs on Kubernetes under the restricted Pod Security Standard (non-root, read-only root filesystem).
---

Complements the public skill `docker/skills@docker-build-strategies` (multi-stage, layer cache, BuildKit). Read it for build mechanics; this page only adds the restricted-cluster rules and corrects one of its examples. The checks below are run by `mise run lint:docker`.

## Rules
- **Numeric final `USER`**: `USER 10001:10001` in the last stage. `USER nonroot:nonroot` (the public skill's example) cannot be verified by kubelet's `runAsNonRoot`, which needs a numeric UID. hadolint only flags this at `info` (DL3066), so `lint:docker` also greps the last `USER` for `^USER [0-9]+(:[0-9]+)?$`. The chart must set the same UID in `runAsUser`.
- **Pinned base**: every `FROM` is `image:tag@sha256:<64 hex>`; keep the tag for readability. Renovate updates tag and digest (`"extends": ["docker:pinDigests"]`, `dockerfile` manager). `lint:docker` rejects a `FROM` without a digest.
- **Minimal runtime**: final stage is distroless or `scratch`. No shell, curl, package manager or package caches there; compilers stay in the builder (Go: `CGO_ENABLED=0`, `-trimpath`, `-ldflags="-s -w"`).
- **Nothing that needs privilege**: no root, no added capabilities, no setuid/setgid binaries, no ports below 1024 (`EXPOSE 8080`).
- **Read-only root filesystem**: the image must run with `readOnlyRootFilesystem: true`. Every path the app writes (tmp, cache, pid) is a mount point the chart backs with an `emptyDir`; document those paths in the Dockerfile.
- **PSS restricted**: the image must start under `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`, `seccompProfile.type: RuntimeDefault`. Rules for the chart itself are in `k8s-restricted`.
- **No `HEALTHCHECK`**: Kubernetes ignores it; use the chart's liveness/readiness probes.
- **`.dockerignore`**: exclude at least `.git`, `.env`, key files, local build output.
- **Build secrets**: `RUN --mount=type=secret,id=<name> ...` and `docker buildx build --secret id=<name>,src=<file>`. Never `ARG`/`ENV`/`COPY` a secret.
- **OCI labels**: `org.opencontainers.image.source`, `.revision`, `.version`, `.licenses` (via `LABEL`, values passed by `--build-arg` in CI).

## Checks (all via mise tools, pinned in `mise.toml`)
- `lint:docker`: `hadolint Dockerfile` (reads `.hadolint.yaml`, or `-c <file>`; set `failure-threshold: warning`), then the digest and numeric-USER greps. Suppress a rule only with a `# hadolint ignore=DLxxxx` line and a reason.
- CI, after the build: `trivy image --severity HIGH,CRITICAL --exit-code 1 --ignore-unfixed <image>` (the gate); list exceptions in `.trivyignore` with a reason.
- SBOM: `docker buildx build --sbom=true ...` (attestation) or `syft scan <image> -o spdx-json=sbom.spdx.json`; publish it with the image.

---
name: twelve-factor
description: Use when writing or reviewing a Go or Python application that ships as a container image, or when it reads configuration, logs, handles shutdown or keeps state.
---

The twelve factors, in the terms of a Go or Python service deployed on Kubernetes. What can be observed on the running container is an acceptance criterion of the application baseline (`FAC-001` to `FAC-005`, `stacked-tdd-mr/templates/socle-application.md`), tested on the image started by the repo's `compose.yaml` (`compose-dev`). The rest is a written rule, checked at review.

Status: the `FAC-…` tests were not run when this page was written (2026-11-14, no Docker on the workstation): **unverified** until a first application carries them.

| # | Factor | What the code does | Held by |
|---|---|---|---|
| I | Codebase | One repo, one application, one image. Several deployments differ by configuration only. | written rule |
| II | Dependencies | Declared and locked in the repo (`go.mod`/`go.sum`, `pyproject.toml`/`uv.lock`), tools in `mise.toml`. Nothing comes from the workstation. | `repo-tooling` |
| III | Config | Read from environment variables, once, in the composition root (`cmd/<app>/main.go`, `composition.py`), then passed down as values. A missing mandatory variable stops the process and names the variable. No configuration file baked in the image, no branch on the environment name. On Kubernetes: `env` / `envFrom` from a ConfigMap and a Secret. | `FAC-001`, `FAC-002` |
| IV | Backing services | Database, cache, broker and object store are reached through a URL from the environment. Swapping a local one for a managed one changes configuration only. | written rule, `compose-dev` |
| V | Build, release, run | The image is built once per commit and tagged immutably; every environment runs that image. Nothing is compiled or downloaded at startup. | written rule, `docker-build-strategies` |
| VI | Processes | Stateless. Nothing written on the local disk is expected later: sessions, uploads and caches go to a backing service. The root filesystem is read-only, the user non-root. On Kubernetes: `readOnlyRootFilesystem: true`, `runAsNonRoot: true` (`k8s-restricted`). | `FAC-005` |
| VII | Port binding | The process serves on the port given by `PORT`; no web server in front inside the container. | `FAC-001` |
| VIII | Concurrency | Scale by replicas. One kind of process per container (web, worker); no background daemon forked by the application, no process manager. | written rule |
| IX | Disposability | Fast start. On SIGTERM: stop accepting work, finish what is in flight, exit 0 before the grace period ends. `ENTRYPOINT` in exec form, so that the signal reaches the process. Go: `signal.NotifyContext` then `http.Server.Shutdown`. Python: let the server (uvicorn, gunicorn) handle the signal; a worker loop handles it itself. | `FAC-004` |
| X | Dev/prod parity | `compose.yaml` builds the production `Dockerfile` and uses the same kind of backing service as production (PostgreSQL, not SQLite). | `compose-dev` |
| XI | Logs | One event per line on standard output, structured (Go: `log/slog` with the JSON handler; Python: `logging` to `sys.stdout`). No log file, no rotation, no shipping from the application: the platform collects. | `FAC-003` |
| XII | Admin processes | Migrations and one-off tasks are a command of the same image, run with the same configuration (Kubernetes Job or init container; locally `docker compose run --rm app <command>`). | written rule |

## Reading the table
- A `FAC-…` criterion without object in a repo (a worker with no port has no `FAC-001`) is set aside by a ledgered `Ruling:`, not silently skipped.
- A written rule is not a lesser rule: the reviewer of the stack checks it and cites the code.
- Configuration is where hexagonal and twelve-factor meet: only the composition root reads the environment. A use case or the domain reading `os.Getenv` / `os.environ` is a finding.

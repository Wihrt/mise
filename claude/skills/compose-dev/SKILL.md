---
name: compose-dev
description: Use when adding or reviewing the compose.yaml of a Go or Python application, its watch loop, or the tests that start the product through Compose.
---

One `compose.yaml` at the repo root starts the application the way production runs it, reloads it while you edit, and is what the container tests start. For Dockerfile mechanics see `docker-build-strategies` and `dockerfile-restricted`; this page covers Compose only. Not for infrastructure-as-code repos (charts, OpenTofu, Ansible): they have their own test tools.

Status: written 2026-11-14 on a workstation without Docker. Every command and every file below is **unverified** until run once with Docker (Compose 2.22 or later for `develop.watch`); then replace this line with the versions seen. The templates are `repo-tooling/templates/compose.go.yaml` and `compose.py.yaml`.

## The application service
- `build:` points at the production `Dockerfile`, same target as the published image. No second Dockerfile for development (twelve-factor X: dev and prod run the same image).
- `read_only: true`, `cap_drop: [ALL]`, `security_opt: ["no-new-privileges:true"]`; the user comes from the image (non-root, see `dockerfile-restricted`). A directory the process must write is a `tmpfs:` entry, named in the review.
- Settings only through `environment:` (twelve-factor III). Values are development values; no secret of a real environment in the file.
- Port: `"${APP_PORT:-8080}:8080"`. A developer gets 8080; a test run sets its own (see "Started by the tests").
- No `healthcheck:` that needs a shell or curl in a distroless image. Either the binary has a health subcommand, or the tests poll the health route.

## Backing services
- One service per backing service (database, cache, broker), image pinned by tag and digest (Renovate updates it), reached through a URL in the application's `environment:` (twelve-factor IV).
- Each has a `healthcheck:` (PostgreSQL: `pg_isready -U <user> -d <db>`), and the application waits for it: `depends_on: {db: {condition: service_healthy}}`.
- Data in a named volume only when a developer needs it to survive `down`; the tests always start empty (`down -v`).

## Watch loop
`mise run dev` is `docker compose up --build --watch`.

| Language | Source change | Dependency change |
|---|---|---|
| Go | `action: rebuild` on `./cmd`, `./internal`, `go.mod`, `go.sum` (the binary is compiled into the image) | same rule |
| Python | `action: sync+restart`, `path: ./src`, `target: /app/src` | `action: rebuild` on `pyproject.toml`, `uv.lock` |

- `ignore:` test directories and documentation, so that editing a test rebuilds nothing.
- Unverified point to settle first: `sync` copies files into the container; with `read_only: true` it may be refused. If it is, Python uses `action: rebuild` on `./src` too, and `read_only` stays. Do not drop `read_only` to get a faster reload.

## Started by the tests
The container tests (`test:container`, `test:api`, `test:ui`) start the same file, through the common helper of the test directory:

```bash
project="<repo>-$$"                      # one Compose project per run
trap 'docker compose -p "$project" down -v --remove-orphans' EXIT
APP_PORT=0 docker compose -p "$project" up -d --build --wait
port=$(docker compose -p "$project" port app 8080 | sed 's/.*://')
```

- A project name and a host port of its own per run: two runs at once do not collide, so `harness gate --jobs` stays usable. `APP_PORT=0` asks Docker for a free host port; `docker compose port` reads it.
- `up --wait` returns when every service is running or healthy, and fails when one is not: no fixed sleep.
- `down -v` in a trap: a failed run leaves no container, network or volume behind.

## Checks
- `lint:compose`: `docker compose config --quiet` (the file parses and its variables resolve).
- The watch section is not tested by a criterion (a test that reads `develop.watch` in the file would copy the configuration): start `mise run dev` once in the `chore(tooling)` commit, edit a response, see it change, and say so in the MR.

## Related
Generic Compose patterns (overrides, networks, profiles): the public skill `docker-compose-patterns`, named by `docker-build-strategies`; it is not vendored here (not checked on 2026-11-14).

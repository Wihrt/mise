---
name: stack-go-app
description: Use when repo-stack names it - the stack required for a Go application.
---

What a Go application must respect. Each line names the skill that details it: read that skill before the corresponding work. This page repeats none of them.

## Always
1. Hexagonal layout and SOLID: `hexagonal-go`.
2. Repo tooling (mise tasks, hooks, CI, Renovate): `repo-tooling`.
3. Architecture baseline: criteria `ARC-001` to `ARC-003` in `TESTS.md` before the first feature, proven by mutants. Criteria: `stacked-tdd-mr/templates/socle-application.md`; test: `hexagonal-go/templates/arch_test.go`.
4. Acceptance tests at the boundary, one mutant per clause: `stacked-tdd-mr`. An HTTP API is tested with Bruno, a web front with Playwright (catalogue of `agents/acceptance-writer.md`).
5. The smallest code that passes the tests; nothing for a hypothetical case.

## With a Dockerfile at the root (`image=yes`)
6. Twelve factors: `twelve-factor`; baseline criteria `FAC-001` to `FAC-005`, same template as line 3.
7. `compose.yaml` with a watch loop, tasks `dev`, `lint:compose`, `test:container`: `compose-dev`, and `repo-tooling` (section "Go or Python application").
8. Image: `docker-build-strategies`, `dockerfile-restricted`.

## Coverage (`image=yes`)
`harness gate coverage --task test` proves every block of production code is run by the boundary tests. The product writes the measure; `go test` is never instrumented. In Go the measure is by blocks of statements. Verified 2026-11-17, Go 1.26.9, Docker 29.1.3, Compose 2.40.3, on a trial service and tool:
- Dockerfile: `ARG GO_COVER_FLAGS=""` and `go build -trimpath ${GO_COVER_FLAGS} -o /out/server ./cmd/server`; the override passes `-cover -coverpkg=./...`, sets `GOCOVERDIR: /cover` and mounts `./.tdd/cover-raw/service:/cover` (`compose-dev`, `repo-tooling/templates/compose.go.yaml`).
- Tool: the `test` task builds it with `go build -cover -coverpkg=./... -o bin/tool ./cmd/tool`; the test runs it with `cmd.Env = append(os.Environ(), "GOCOVERDIR="+os.Getenv("TOOL_COVERDIR"))`, the task passing `TOOL_COVERDIR="$PWD/.tdd/cover-raw/tool"`. A relative `GOCOVERDIR` is resolved from the test package directory: stderr `output directory ... inaccessible`, exit 0, no data. `bin/` is in the trial `.gitignore`.
- Never `-cover` on `go test`: it replaces `GOCOVERDIR` with its own temporary directory, the binary writes there and nothing is kept. `go test -count=1` (a cached run does not start the product again).
- After the tests and `docker compose stop`: `go tool covdata textfmt -i=.tdd/cover-raw/service,.tdd/cover-raw/tool -o .tdd/cover/go.txt`, the report `gate coverage` reads.
- A map of sub-commands hides the one never called (one block, run); use a `switch` with a `case` per sub-command. A default value never exercised (`if port == "" { port = "8080" }`) is reported: drop it or test it.
- The boundary tests wait for the product themselves (a poll of a route before the first call; the image has no shell, so no healthcheck). Not verified: the trial task waited with `curl`, which is not a declared tool.
- Not verified: distroless (the trial image is `FROM scratch`), macOS, several replicas, a module path different from the repo, a service and a tool where one hides the other's uncovered block (the profile merges both).

## Order in a repo that has none of this
`chore(tooling)`, then the baseline tasks, then the features (`stacked-tdd-mr`, Setup).

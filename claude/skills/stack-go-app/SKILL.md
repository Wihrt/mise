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
`harness gate coverage --task test` proves every block of production code is run by the boundary tests. The product writes the measure. In Go the measure is by blocks of statements. Verified 2026-11-18, Go 1.26.9, gotestsum 1.13.0 (in `[tools]`), Docker 29.1.3, Compose 2.40.3, on a trial service and tool:
- The unit suite is not instrumented and does not count: `test:go` is `gotestsum --junitfile .tdd/junit.xml -- -count=1 ./...`, a function only a unit test calls is `UNCOVERED`. Every file of the boundary test package (helper included) starts with `//go:build container`: `test:go` skips the package silently and starts no container. An untagged file would make the package exist for `./...` and run the tests with no product (not shown).
- Boundary tests are `test:container`; its test helper starts the product (`compose-dev`, "Coverage of the product"). The task runs `rm -rf .tdd/cover-raw`, `mkdir -p .tdd/cover-raw/service .tdd/cover-raw/tool bin`, `chmod 0777 .tdd/cover-raw/service`, `go build -cover -coverpkg=./... -o bin/tool ./cmd/tool`, then `TOOL_BIN="$PWD/bin/tool" TOOL_COVERDIR="$PWD/.tdd/cover-raw/tool" gotestsum --junitfile .tdd/container.xml -- -tags container -count=1 ./test/...`. With the parent and child `TestMain`, `.tdd/container.xml` still lists every boundary test (the child receives the parent's `-test.v=test2json` argument and its stdout is passed through). Nothing deletes the two reports between runs.
- Dockerfile: `ARG GO_COVER_FLAGS=""` and `go build -trimpath ${GO_COVER_FLAGS} -o /out/server ./cmd/server`; the override passes `-cover -coverpkg=./...`, sets `GOCOVERDIR: /cover` and mounts `./.tdd/cover-raw/service:/cover` (`repo-tooling/templates/compose.go.yaml`).
- The test runs the tool with `cmd.Env = append(os.Environ(), "GOCOVERDIR="+os.Getenv("TOOL_COVERDIR"))`. A `GOCOVERDIR` must be absolute: a relative one is resolved from the process directory (stderr `output directory ... inaccessible`, exit 0, no data; seen with the binary run directly). The repo adds `bin/` to its `.gitignore` (the template has only `.tdd/`).
- The `run` of `[tasks.test]` converts, after every `test:*` task: `mkdir -p .tdd/cover` then `go tool covdata textfmt -i=.tdd/cover-raw/service,.tdd/cover-raw/tool -o .tdd/cover/go.txt`.
- Every branch is reported like any block: the `log.Fatal` on a listen error (`UNCOVERED cmd/server/main.go:24-25`), the usage branch and the unknown-sub-command branch of the tool. Each needs a boundary test (the trial runs the instrumented image alone: `docker compose run --rm --no-deps -e PORT=not-a-port app`, expects exit 1) or goes. A `switch` with a `case` per sub-command shows a never-called one.
- The tests wait for the product themselves: the parent `TestMain` polls `/hello` (up to 30 s); the image has no shell, so no healthcheck (cold start shown).
- Seen on 2026-11-17 only (notes not kept), not re-verified: `go test -cover` replaces `GOCOVERDIR` with its own temporary directory (so never `-cover` on `go test`); a default value never exercised (`if port == "" { port = "8080" }`) is reported (drop it or test it); a map of sub-commands hides a never-called one (so use the `switch`).
- Not verified: distroless (the trial image is `FROM scratch`), macOS, several replicas, a service and a tool where one hides the other's uncovered block (the profile merges both), a `go test` killed by SIGKILL, several `test:*` suites each starting an instrumented product on one measure directory.

## Order in a repo that has none of this
`chore(tooling)`, then the baseline tasks, then the features (`stacked-tdd-mr`, Setup).

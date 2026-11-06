---
name: hexagonal-go
description: Use when structuring, adding to or reviewing a Go service with ports and adapters, dependency injection or interfaces, or when domain code starts importing a driver or os/exec.
---

Go hexagonal layout, kept small. For code style (naming, errors, concurrency, struct init) follow the Uber Go style guide instead of restating it here.

## Layout
- `cmd/<app>/main.go` is the composition root: it builds adapters, injects them into services, starts the process. No logic elsewhere may construct an adapter.
- `internal/domain`: entities, value objects, domain errors. Imports the standard library only (no `os/exec`, no driver, no framework).
- `internal/app`: use cases; they call the outside world through interfaces they declare.
- `internal/adapters/<tech>` (`postgres`, `http`, `kafka`): implement those interfaces; the only code that knows the technology.
- No central `port` or `ports` package, no `I` prefix, no `Port` suffix. Name the interface for what it does (`Reader`, `OrderSaver`).

## Interfaces
- Declare the interface in the package that consumes it, with 1 to 3 methods. Wider means split by use case.
- No interface until there is a second implementation or a test seam needs one. A type with one implementation and no seam stays a concrete type.
- Accept interfaces, return structs. Constructors are `NewX(deps...) *X`.
- Dependency injection by hand through constructors. No wire, dig or fx unless the PR states why the graph needs it.

## Errors
- Wrap at every boundary with `fmt.Errorf("save order %s: %w", id, err)`; never discard the cause.
- Domain errors are sentinels (`var ErrNotFound = errors.New(...)`) or typed errors; callers test with `errors.Is` / `errors.As`, never by string. Adapters translate driver errors (`sql.ErrNoRows`) into domain errors.

## Tests
- Table-driven, `t.Run` per case. Domain and app tests use hand-written fakes of the consumer-side interfaces (tiny structs, no mock generator).
- Adapter tests are integration tests against the real dependency with testcontainers, behind a build tag or `-short` skip, in the adapter package.
- Test names carry the TESTS.md ID: `Test<PREFIX><DIGITS>_<Behavior>`, e.g. `TestCALC001_Add` for `CALC-001`. Subtests may add detail but the top-level name keeps the ID.

## Security and verification
Commands live in `mise.toml` (see `repo-tooling`); `lint:go` and `test:go` run them. `test:go` writes the JUnit report that `tdd-gate` reads: `gotestsum --junitfile .tdd/junit.xml -- ./...` with `gotestsum = "1.13.0"` in `[tools]` (registry `aqua:gotestyourself/gotestsum`; verified on the e2e fixture: a compile error gives a report holding only a `TestMain` failure, and the output carries `undefined: <symbol>`). `.tdd/` is gitignored.

| Layer | Control | Where |
|---|---|---|
| Vulnerabilities | `govulncheck ./...`, mise tool `"go:golang.org/x/vuln/cmd/govulncheck"` (not in the registry) | pre-push, CI |
| Static analysis | golangci-lint with `gosec`, `errcheck`, `staticcheck`, `errorlint`, `bodyclose`, `noctx`, `sqlclosecheck`, `forbidigo` | pre-commit, CI |
| Dependencies | `go mod verify`, `go mod tidy -diff`, `GOFLAGS=-mod=readonly`, `GOSUMDB` left on, `toolchain` line pinned in `go.mod` | CI |
| Tests | `go test -race` (needs `CGO_ENABLED=1` and a C compiler); native fuzzing (`go test -fuzz`) on external inputs, corpus in `testdata/fuzz` committed | pre-push, CI |
| Build | `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"`, image `scratch` or distroless | Dockerfile |
| Supply chain | `osv-scanner`, SBOM with `syft` | CI |

## Enforcing the layers
golangci-lint v2 config (`version: "2"`); forbidigo keeps process execution out of the domain and applies only there:

```yaml
version: "2"
linters:
  enable: [gosec, errcheck, staticcheck, errorlint, bodyclose, noctx, sqlclosecheck, forbidigo]
  settings:
    forbidigo:
      analyze-types: true
      forbid:
        - pattern: ^exec\..*$
          msg: domain must not run processes; put it behind an adapter
  exclusions:
    rules:
      - path-except: internal/domain/
        linters: [forbidigo]
```

- The pattern matches the package name, so an aliased import (`import x "os/exec"`) escapes it; review aliases in `internal/domain`.
- Add the same kind of pattern for driver packages (`^sql\.`, `^http\.`) as the domain grows.
- Verified by the fixture: `exec.Command` in `internal/domain` fails `golangci-lint run`; removing it passes; the same call in `cmd/` is not flagged by forbidigo.

When a principle conflicts with a simple function, keep the function and say so in the review.

## Acceptance tests
- Written by the `acceptance-writer` from spec criteria (catalogue: `~/.claude/agents/acceptance-writer.md`). CLI: a test package that imports nothing of the module, builds the binary in `TestMain` and runs it as a subprocess; moving code into `internal/` keeps it green (dry run 2026-10-28). Go caches such tests across source changes: `tdd-gate` runs with `GOFLAGS=-count=1`; do the same in CI.

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

## SOLID, in Go terms
- SRP: one package, one reason to change; a use case does one thing.
- OCP: add behaviour with a new adapter or by wrapping an interface (decorator), not by editing callers.
- LSP: an adapter honours the documented contract of the interface it implements (errors returned, ordering), not only its method set.
- ISP: interfaces of 1 to 3 methods, declared by the consumer; never one wide `Repository`.
- DIP: use cases depend on interfaces they declare and receive through constructors, never on an adapter.

## Errors
- Wrap at every boundary with `fmt.Errorf("save order %s: %w", id, err)`; never discard the cause.
- Domain errors are sentinels (`var ErrNotFound = errors.New(...)`) or typed errors; callers test with `errors.Is` / `errors.As`, never by string. Adapters translate driver errors (`sql.ErrNoRows`) into domain errors.

## Tests
- Table-driven, `t.Run` per case. Domain and app tests use hand-written fakes of the consumer-side interfaces (tiny structs, no mock generator).
- Adapter tests are integration tests against the real dependency with testcontainers, behind a build tag or `-short` skip, in the adapter package.
- Test names carry the TESTS.md ID: `Test<PREFIX><DIGITS>_<Behavior>`, e.g. `TestCALC001_Add` for `CALC-001`. Subtests may add detail but the top-level name keeps the ID.

## Security and verification
Commands live in `mise.toml` (see `repo-tooling`); `lint:go` and `test:go` run them. `test:go` writes the JUnit report that `harness gate` reads: `gotestsum --junitfile .tdd/junit.xml -- -count=1 ./...` (`-count=1`: no cached result, see "Architecture test") with `gotestsum = "1.13.0"` in `[tools]` (registry `aqua:gotestyourself/gotestsum`; verified on the e2e fixture: a compile error gives a report holding only a `TestMain` failure, and the output carries `undefined: <symbol>`). `.tdd/` is gitignored.

| Layer | Control | Where |
|---|---|---|
| Vulnerabilities | `govulncheck ./...`, mise tool `"go:golang.org/x/vuln/cmd/govulncheck"` (not in the registry) | pre-push, CI |
| Static analysis | golangci-lint with `gosec`, `errcheck`, `staticcheck`, `errorlint`, `bodyclose`, `noctx`, `sqlclosecheck`, `forbidigo` | pre-commit, CI |
| Dependencies | `go mod verify`, `go mod tidy -diff`, `GOFLAGS=-mod=readonly`, `GOSUMDB` left on, `toolchain` line pinned in `go.mod` | CI |
| Tests | `go test -race` (needs `CGO_ENABLED=1` and a C compiler); native fuzzing (`go test -fuzz`) on external inputs, corpus in `testdata/fuzz` committed | pre-push, CI |
| Build | `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"`, image `scratch` or distroless | Dockerfile |
| Supply chain | `osv-scanner`, SBOM with `syft` | CI |

## Enforcing the layers
The architecture test is the guarantee; forbidigo is fast feedback at commit.

### Architecture test
`tests/architecture/arch_test.go` (template: `templates/arch_test.go`) carries the criteria `ARC-001` to `ARC-003` of the application baseline (`stacked-tdd-mr/templates/socle-application.md`). It reads the import graph with `go list -deps -json` from the module root, imports nothing of the module, and runs in `test:go`: `TestARC001_DomainDependsOnlyOnItself`, `TestARC002_UseCasesKnowNoTechnology`, `TestARC003_OnlyTheCompositionRootAssembles`.

- Two lists in the test file name the third-party packages a layer may import: `allowedDomain` (ARC-001) and `allowedApp` (ARC-002). Both rules are whitelists: a use case may import the use cases, the domain, the standard library and `allowedApp`, nothing else of the module (an `internal/infra` package fails ARC-002). It is an acceptance file: widening a list is a criterion change (`--ac-changed ARC-001` or `ARC-002`), never an implementer's edit.
- Verified 2026-11-14 (go 1.27.4, trial module): a domain import of an adapter fails ARC-001, also under an alias and through an intermediate package; a third-party import outside `allowedDomain` fails ARC-001; a use case importing an adapter (directly or through an intermediate package), a third-party package outside `allowedApp` or a module package outside the layers fails ARC-002; an adapter importing another fails ARC-003; a new domain package, a new adapter wired in `cmd/` and a third-party package added to `allowedApp` keep the three green. The template passes the golangci-lint configuration below (2.15.0: `0 issues`).
- The three tests do not overlap (a direct domain import of an adapter fails ARC-001 only), so the gate's redundancy proof keeps all three.
- Go caches this test across source changes, since it reads files outside its package: run it with `-count=1` (verified: `ok (cached)` after a forbidden import was added). `harness gate` sets `GOFLAGS=-count=1`; the `test:go` command above carries the flag for local runs and CI.
- A domain import of an adapter that itself imports the domain is an import cycle: the compiler refuses it and the three tests fail on `go list`. A mutant for ARC-001 targets an adapter that does not import the domain, or a third-party package.
- A package sitting in `internal/adapters` itself is an adapter like the others: only `cmd/` may import it, and it may import no other adapter (verified).
- Custom build tags: `go list` reads the files of the default build only. List the module's tags in `buildTags` of the test file; the graph is then read once without them and once with them all (verified: a file under `//go:build extra` importing another adapter fails ARC-003 once `extra` is listed, and is not seen before).
- ARC-001 and ARC-002 allow the whole standard library, `os/exec` and `database/sql` included: keeping those out of the domain and the use cases is forbidigo's job, below.

### forbidigo
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

- The pattern matches the package name, so an aliased import (`import x "os/exec"`) escapes it; review aliases in `internal/domain`. The architecture test has no such limit, but it only judges packages outside the standard library.
- Add the same kind of pattern for driver packages (`^sql\.`, `^http\.`) as the domain grows.
- Verified by the fixture: `exec.Command` in `internal/domain` fails `golangci-lint run`; removing it passes; the same call in `cmd/` is not flagged by forbidigo.

When a principle conflicts with a simple function, keep the function and say so in the review.

An application that ships as a container image (a `Dockerfile` at the root) also follows `twelve-factor` and `compose-dev`.

## Acceptance tests
- Written by the `acceptance-writer` from spec criteria (catalogue: `~/.claude/agents/acceptance-writer.md`). CLI: a test package that imports nothing of the module, builds the binary in `TestMain` and runs it as a subprocess; moving code into `internal/` keeps it green (dry run 2026-10-28). Go caches such tests across source changes: `harness gate` runs with `GOFLAGS=-count=1`; do the same in CI.

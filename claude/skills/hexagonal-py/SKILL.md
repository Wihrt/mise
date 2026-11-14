---
name: hexagonal-py
description: Use when structuring, adding to or reviewing a Python service with hexagonal layers, ports and adapters, dependency injection or SOLID, or when domain code starts importing a framework.
---

Rules are checked by the mise task `lint:py` (ruff, ty); tests by `test:py` (`uv run pytest --junitxml=.tdd/junit.xml`, the JUnit report `harness gate` reads; `.tdd/` is gitignored; verified with pytest 9.1.1: it creates `.tdd/`, and an import error at collection writes one `error` testcase named after the module and exits 2). Names below are the shape, not a framework.

## Layout
```
src/<pkg>/domain/       # pure: dataclasses, value objects, pure functions
src/<pkg>/app/          # use cases; declare the ports they consume
src/<pkg>/adapters/     # DB, HTTP clients, SDKs: implement ports
src/<pkg>/entrypoints/  # HTTP/CLI/worker: parse input, call a use case, map errors
src/<pkg>/composition.py  # the ONLY place that builds adapters and wires use cases
```
Dependencies point inward: entrypoints/adapters -> app -> domain. `domain` imports nothing from the others and no framework/ORM/SDK (`sqlalchemy`, `fastapi`, `boto3`, `requests`...).

## Ports and DI
- A port is a `typing.Protocol` declared in the module of its **consumer** (`app/`), 1-3 methods. Use `abc.ABC` only to share behaviour, never to declare a port.
- Manual DI: pass dependencies as constructor or function arguments. No container, no module-level singleton, no `get_x()` global; `composition.py` builds the graph once, entrypoints receive it.
- Use-case shape: `class RegisterUser: def __init__(self, users: UserStore, clock: Clock): ...`.

## SOLID, in Python terms
- SRP: small modules, one reason to change; a use case does one thing.
- OCP: add behaviour by composing/wrapping a port (decorator, new adapter), not by editing callers.
- LSP: an adapter honours the Protocol's documented contract (errors, ordering), not only its signatures.
- ISP: narrow Protocols per consumer; never one fat `Repository`.
- DIP: use cases depend on injected Protocols, never on a concrete adapter.

## Errors
Domain raises domain exceptions. Adapters translate library errors (`sqlalchemy.exc.*`, `httpx.*`) to domain/app exceptions with `raise ... from err`. Entrypoints map those to status codes/exit codes; no framework exception leaks inward.

## Tests
- Fakes: in-memory classes implementing the Protocol (`FakeUserStore` with a dict). Never `Mock`/`MagicMock` on your own ports.
- One contract suite per port (parametrized fixture over `[FakeX, RealX]`), run against every adapter, so fake and real cannot drift.
- Use cases are tested with fakes, no I/O; adapters with the contract suite on the real backend.
- Test names carrying a TESTS.md ID are `test_<PREFIX><DIGITS>_<what>`, dash removed: `CALC-001` -> `test_CALC001_add`. This matches the `harness gate` normalization; `test_CALC_001_add` is also accepted.

## Tooling (verified: ruff 0.17.1, ty 0.0.90)
`lint:py` runs `ruff check`, `ruff format --check` and `ty check src`. ty is 0.0.x beta: pin the exact version in `mise.toml`.

Root `ruff.toml` selects `TID` (plus `E,F,I`). Ban frameworks in the domain with a per-directory `src/<pkg>/domain/ruff.toml`:
```toml
extend = "../../../ruff.toml"   # path relative to this file; REQUIRED
[lint.flake8-tidy-imports.banned-api]
"sqlalchemy".msg = "domain must not import frameworks"
"fastapi".msg = "domain must not import frameworks"
```
- Without `extend`, the nested file **replaces** the root config: `TID` is not selected and the ban silently passes (verified: `All checks passed!`).
- A banned module also bans its submodules (`from sqlalchemy.orm import X` fails with `TID251`).
- Direct imports only: ruff does not follow transitive imports, so `domain` -> `helpers` -> `sqlalchemy` passes ruff; the architecture test below catches it. Keep `domain` free of imports from `adapters`/`app` too (ban `<pkg>.adapters`, `<pkg>.app` the same way).

## Architecture test
`tests/architecture/test_architecture.py` (template: `templates/test_architecture.py`) carries the criteria `ARC-001` to `ARC-003` of the application baseline (`stacked-tdd-mr/templates/socle-application.md`). It reads the import graph with `grimp` (`grimp` in `[dependency-groups] dev`), imports nothing of the package, and runs in `test:py`: `test_ARC001_domain_depends_only_on_itself`, `test_ARC002_use_cases_know_no_technology`, `test_ARC003_only_the_composition_root_assembles`.

- The third-party packages the domain may import are the `ALLOWED` set of the test file. It is an acceptance file: widening the set is a criterion change (`--ac-changed ARC-001`), never an implementer's edit.
- Verified 2026-11-14 (grimp 3.20, pytest 9.2.0, trial package): a domain import of an adapter fails ARC-001, also under an alias, inside a function, as a relative import and through an intermediate module; a third-party import (the library need not be installed) and an import of `app` fail ARC-001; a use case importing an adapter or `composition` fails ARC-002; an adapter importing another, or an entrypoint importing an adapter, fails ARC-003; a new domain module, a standard-library import and a new adapter wired in `composition.py` keep the three green.
- The three tests do not overlap (a direct domain import of an adapter fails ARC-001 only), so the gate's redundancy proof keeps all three.
- Limit (verified): a dynamic import (`importlib.import_module("<pkg>.adapters.x")`) is not seen. Do not use one to reach across layers.

An application that ships as a container image (a `Dockerfile` at the root) also follows `twelve-factor` and `compose-dev`.

## Anti-patterns
- A Protocol for every class; a port with one implementation and no test need.
- Getters/setters and anemic classes; use plain dataclass attributes.
- One repository per table; model ports on what the use case needs.
- `Port`, `IRepo`, `AbstractX` naming: name by role (`UserStore`, `Clock`, `Mailer`).
- Importing adapters from `app`/`domain`; building adapters inside use cases; DI containers; `unittest.mock.patch` of own modules.

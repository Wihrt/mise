---
name: stack-py-app
description: Use when repo-stack names it - the stack required for a Python application.
---

What a Python application must respect. Each line names the skill that details it: read that skill before the corresponding work. This page repeats none of them.

## Always
1. Hexagonal layout and SOLID: `hexagonal-py`.
2. Repo tooling (mise tasks, hooks, CI, Renovate): `repo-tooling`.
3. Architecture baseline: criteria `ARC-001` to `ARC-003` in `TESTS.md` before the first feature, proven by mutants. Criteria: `stacked-tdd-mr/templates/socle-application.md`; test: `hexagonal-py/templates/test_architecture.py`.
4. Acceptance tests at the boundary, one mutant per clause: `stacked-tdd-mr`. An HTTP API is tested with Bruno, a web front with Playwright (catalogue of `agents/acceptance-writer.md`).
5. The smallest code that passes the tests; nothing for a hypothetical case.

## With a Dockerfile at the root (`image=yes`)
6. Twelve factors: `twelve-factor`; baseline criteria `FAC-001` to `FAC-005`, same template as line 3.
7. `compose.yaml` with a watch loop, tasks `dev`, `lint:compose`, `test:container`: `compose-dev`, and `repo-tooling` (section "Go or Python application").
8. Image: `docker-build-strategies`, `dockerfile-restricted`.

## Coverage (`image=yes`)
`harness gate coverage --task test` proves every line and branch of production code is run by the boundary tests. The product writes the measure; `pytest` is never instrumented. In Python the measure is by lines and branches. Verified 2026-11-17, Python 3.13.18, coverage.py 7.14.0, uv 0.13.9, Docker 29.1.3, Compose 2.40.3, on a trial FastAPI service (uvicorn, one process) and tool:
- `pyproject.toml`: `[dependency-groups] cov = ["coverage==7.14.0"]`, included in `dev`. Dockerfile: `ARG COVERAGE=""`, then `RUN if [ -n "$COVERAGE" ]; then uv sync --frozen --no-dev --group cov --no-install-project; fi` and `COPY .coveragerc ./.coveragerc`. The override passes `COVERAGE: "1"`, `COVERAGE_PROCESS_START: /app/.coveragerc`, `COVERAGE_FILE: /cover/.coverage` and mounts `./.tdd/cover/raw:/cover` (`compose-dev`, `repo-tooling/templates/compose.py.yaml`). coverage.py 7.14.0 ships the startup hook (`a1_coverage.pth`); write none.
- `.coveragerc`: `[run] branch = True`, `parallel = True`, `relative_files = True`, `sigterm = True`, `source = src`. Without `relative_files`, container paths (`/app/src/...`) and host paths differ and `coverage lcov` fails (`No source for code`). With `init: true` (in the trial `compose.yaml`) and no `sigterm`, the stop writes no file. Without `source`, `tests/` files enter the report when pytest is measured.
- The tool: the test fixture runs `[sys.executable, "-m", "<package>", ...]` with `COVERAGE_PROCESS_START` and `COVERAGE_FILE` (same `.tdd/cover/raw`) in the subprocess environment only. Exporting `COVERAGE_PROCESS_START` to pytest measures pytest too.
- The `test` task ends, after `docker compose stop -t 20`, with `COVERAGE_FILE="$PWD/.tdd/cover/.coverage" uv run coverage combine .tdd/cover/raw` then `... uv run coverage lcov -o .tdd/cover/py.lcov`; without `COVERAGE_FILE` at `combine`: `No data to report.`
- A file of `src/` never imported appears as `UNCOVERED <file>:1-N`, not as unmeasured: delete it or test it. A branch never taken shows as `UNCOVERED BRANCH`.
- With `init: true` the stop exits 143 (0 without), instrumented or not; a test requiring exit 0 at the stop: not verified.
- The boundary tests wait for the service themselves (they poll `/openapi.json` before the first call).
- Not verified: a multi-process server (gunicorn, uvicorn workers), distroless, macOS, coverage.py before 7.14.0, a red test in the task (the task keeps pytest's status and returns it after the conversion).

## Order in a repo that has none of this
`chore(tooling)`, then the baseline tasks, then the features (`stacked-tdd-mr`, Setup).

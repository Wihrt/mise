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
6. Unit tests in `tests/unit` are required: `mutation:py` (pytest-gremlins, `repo-tooling` section Mutation) measures them on the files changed.

## With a Dockerfile at the root (`image=yes`)
6. Twelve factors: `twelve-factor`; baseline criteria `FAC-001` to `FAC-005`, same template as line 3.
7. `compose.yaml` with a watch loop, tasks `dev`, `lint:compose`, `test:container`: `compose-dev`, and `repo-tooling` (section "Go or Python application").
8. Image: `docker-build-strategies`, `dockerfile-restricted`.

## Coverage (`image=yes`)
`harness gate coverage --task test` proves every line and branch of production code is run by the boundary tests. The product writes the measure. In Python the measure is by lines and branches. Verified 2026-11-18, Python 3.13.18, coverage.py 7.14.0, uv 0.13.9, Docker 29.1.3, Compose 2.40.3, on a trial FastAPI service (uvicorn, one process) and tool:
- The unit suite is not instrumented and does not count: `test:py` is `uv run pytest -q tests/unit --junitxml=.tdd/junit.xml`, a function only a unit test calls is `UNCOVERED`. The boundary tests live in `tests/container/`: `test:container` runs `rm -rf .tdd/cover`, `mkdir -p .tdd/cover/raw`, `chmod 0777 .tdd/cover/raw`, then `uv run pytest -q tests/container --junitxml=.tdd/container.xml`; its session fixture starts the product (`compose-dev`, "Coverage of the product"). No `testpaths` (a bare `pytest` would run both); each report lists only its own tests and the `rm -rf` leaves them alone.
- `pyproject.toml`: `cov = ["coverage==7.14.0"]` and `dev = [{ include-group = "cov" }, { include-group = "test" }]` under `[dependency-groups]` (`uv run` installs `dev`). `mise.toml` `[env] UV_PYTHON_PREFERENCE = "only-system"`: on the trial uv picked its own Python 3.14.7 over the pinned 3.13.18 (`uv lock` printed `Using CPython 3.14.7`).
- Dockerfile: `ARG COVERAGE=""` in the builder stage, the one with `RUN if [ -n "$COVERAGE" ]; then uv sync --frozen --no-dev --group cov --no-install-project; fi` (after the plain `uv sync`), and `COPY .coveragerc ./.coveragerc` in the final stage. `ENV PYTHONPATH=/app/src` is needed with `--no-install-project` (else `ModuleNotFoundError`). The override passes `COVERAGE: "1"`, `COVERAGE_PROCESS_START: /app/.coveragerc`, `COVERAGE_FILE: /cover/.coverage` and mounts `./.tdd/cover/raw:/cover` (`repo-tooling/templates/compose.py.yaml`). coverage.py 7.14.0 ships the startup hook (`a1_coverage.pth`); write none. The repo adds `__pycache__/` to its `.gitignore` (`.venv/` and `.pytest_cache/` ignore themselves).
- `.coveragerc`: `[run] branch = True`, `parallel = True`, `relative_files = True`, `sigterm = True`, `source = src`. Without `relative_files`, `coverage combine` prints `No source for code` (container and host paths differ) and no report is written. With `init: true` and no `sigterm`, the stop exits 143 and writes no file. The coverage configuration (`.coveragerc`, `[tool.coverage.*]` in `pyproject.toml`) carries no exclusion (`exclude_lines`, `exclude_also`, `partial_branches`, `omit`): the gate refuses the five line-exclusion keys (`exclude_lines`, `exclude_also`, `partial_branches`, `partial_branches_always`, `partial_also`) in the report section coverage.py reads (`[report]` or `[coverage:report]` in `.coveragerc`, `[coverage:report]` in `setup.cfg` and `tox.ini`, `[tool.coverage.report]` in `pyproject.toml`) (`PRAGMA`), an `omit` or `include` makes the file `NOT MEASURED`, and a configuration file under another name (`COVERAGE_RCFILE`, `--rcfile`, `.coveragerc.toml`) is not read, so the rule still stands for it; an exception is written in `COVERAGE.md` only.
- The tool: the fixture runs `[sys.executable, "-m", "<package>", ...]` with `COVERAGE_PROCESS_START=<root>/.coveragerc` and `COVERAGE_FILE=<root>/.tdd/cover/raw/.coverage` in the subprocess environment only; pytest itself gets neither.
- No `if __name__ == "__main__":` guard in production code: it gives an `UNCOVERED BRANCH` no test can clear. The trial starts the service with `CMD ["python", "-c", "from essai.serve import main; main()"]` (`serve.py` reads `PORT` and calls `uvicorn.run`; an exec-form `CMD` does not expand `$PORT`) and the tool through a `__main__.py` that is `raise SystemExit(main())`.
- The `run` of `[tasks.test]` converts, after every `test:*` task: `COVERAGE_FILE="$PWD/.tdd/cover/.coverage" uv run coverage combine .tdd/cover/raw` then the same prefix with `coverage lcov -o .tdd/cover/py.lcov`.
- A file of `src/` never imported appears as `UNCOVERED <file>:1-N`: delete it or test it. A branch never taken shows as `UNCOVERED BRANCH`. A red test makes the task fail: `TASK FAILED test`, then `FAIL`. With `init: true` the instrumented service exits 143 at `docker compose stop`: a test requiring exit 0 there is not verified.
- The fixture waits for the service itself (polls `/openapi.json`, FastAPI's route; another service polls one of its routes).
- Seen on 2026-11-17 only (notes not kept), not re-verified: without `source`, `tests/` files enter the report when pytest is measured; exporting `COVERAGE_PROCESS_START` to pytest measures pytest too (so give it to the subprocess only); `No data to report.` without `COVERAGE_FILE` at `combine`; the stop exits 0 without `init`.
- Not verified: a multi-process server (gunicorn, uvicorn workers), distroless, macOS, coverage.py before 7.14.0, a pytest killed by SIGKILL, several `test:*` suites each starting an instrumented product on one measure directory; `uv run` in the two parallel `test:*` tasks reinstalled the package while the other ran: seen, no failure.

## Order in a repo that has none of this
`chore(tooling)`, then the baseline tasks, then the features (`stacked-tdd-mr`, Setup).

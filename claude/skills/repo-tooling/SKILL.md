---
name: repo-tooling
description: Use when a repo lacks mise.toml, prek hooks or Renovate config, or when adding a lint or test tool to a repo.
---

Every repo runs one tooling baseline. Commands live only in `mise.toml`; developer, hooks and CI all call the same mise task. Templates are in `templates/`.

## Structure
- Every dependency of a repo is declared in the repo, whatever it is: tools in `[tools]` pinned to exact versions; libraries in the project's manifest and its lock file (`pyproject.toml` and `uv.lock`, `go.mod`, `package.json` and its lock file, `Chart.yaml` dependencies, `required_providers` and `.terraform.lock.hcl`); anything else a task needs (a helm plugin, a Playwright browser) installed by a task of the repo. The global mise config of a workstation (`~/.config/mise`) serves daily work only: no task, hook or test may need a tool or an environment variable that comes only from it. A shared cache set there (`TF_PLUGIN_CACHE_DIR`) only saves downloads; the repo works without it. Proof: `mise run check` green in CI, from a fresh clone.
- `mise.toml`: `[tools]` pinned to exact versions (prek and cocogitto included; `mise ls-remote <tool>`), `[tasks.*]` inline only, never `mise-tasks/` files. `[env] RENOVATE_VERSION` pins renovate; `node` is pinned too (`lint:renovate` runs `npx`, CI has no global node).
- Standard tasks: `setup` (`prek install`), `fmt`, `lint` (depends `lint:*`), `test` (depends `test:*`), `check` (`lint`, `test`, and `mutation` once a Mutation fragment is merged), `lint:secrets`, `lint:typos`, `lint:editorconfig`, `lint:renovate`, and `hook:commit-msg`. `hook:*` tasks are called only by hooks, never by lint/check. With no `test:<ns>` task, `mise run test` fails with `task not found: test:*` (verified): the `chore(tooling)` MR defines the first `test:<ns>`; a repo with no tests yet: stop and tell the user. The test task also writes a per-test report under `.tdd/` (`.tdd/junit.xml`, or `.tdd/tofu.json` for OpenTofu; commands in the stack skills) that `~/.local/bin/harness gate` reads; In an application repo with a Dockerfile the boundary tests are the task `test:container` (the mise task prepares the measure directories and runs the suite (Go build tag `container`, Python `tests/container/`), report `.tdd/container.xml`; the test helper starts the product, `compose-dev`), the unit suite `test:go` / `test:py` is not instrumented (report `.tdd/junit.xml`), and `[tasks.test]` is `depends = ["test:*"]` plus a `run` that converts and writes `.tdd/cover/go.txt` (Go) or `.tdd/cover/py.lcov` (Python) (`stack-go-app`, `stack-py-app`). The `test:*` tasks run in parallel; the `run` executes after every one finished; when one fails, `mise run test` exits 1, the `run` does not execute and the gate prints `TASK FAILED test` then `FAIL` (verified 2026-11-18, mise 2026.11.9). `test` is the task `harness gate coverage --task test` receives. `.tdd/` goes in `.gitignore` (`templates/gitignore`, merged into the repo's `.gitignore` without removing lines), else `lint:editorconfig` flags the report (verified). Stack tasks are namespaced: `lint:docker`; `lint:k8s`, `test:helm`; `lint:tofu`, `test:tofu`; `lint:go`, `test:go`; `lint:py`, `test:py`.
- `.pre-commit-config.yaml`: `default_stages: [pre-commit]` (else every local hook also runs at commit-msg and pre-push; verified with `prek run --hook-stage commit-msg`), `repo: builtin` hooks (no `rev`), then `repo: local` hooks, each `entry: mise run <task>`, `language: system`. No command logic in YAML. `default_install_hook_types` lists pre-commit, pre-push, commit-msg so `prek install` installs all three.
- Also: `.editorconfig`, `cog.toml`, `renovate.json`, `_typos.toml` (inline typos ignores do not work in 1.51.1). `lint:secrets` is `gitleaks git` (history only; `dir` scans ignored `.venv/`, `node_modules/`).

## Stages
- pre-commit: fast hygiene and lint tasks. pre-push: `mise run test`. CI: `mise run check`, `cog check <range>`, plus `trivy image` and `glab ci lint` where relevant. Local hooks are advisory (`--no-verify`, no `mise install`); CI is the authority.
- Never use `--no-verify`. A clone without hooks: run `mise run setup`, then commit.
- `hook:commit-msg`: prek appends the message file path to the hook `entry`, so the task is `usage = 'arg "<file>"'` with `run = 'cog verify --file "$usage_file"'`, and the hook has `pass_filenames: true`, `stages: [commit-msg]`. Verified on prek 0.5.7, mise 2026.10.14.
- `lint:renovate` hook uses `files: ^(\.github/|\.gitlab/)?renovate\.json5?$` and `pass_filenames: false`. The first run takes ~30 s (npx). The mise `npm:renovate` backend is refused (trustPolicy), so use `npx`.

## By stack (add the tool to `[tools]`, a `lint:<x>` or `test:<x>` task, a local hook)
| Stack | Tools |
|---|---|
| shell | shellcheck, shfmt |
| CI | actionlint, zizmor (GitHub); `glab ci lint` in CI (GitLab) |
| Docker | hadolint (+ `trivy image` in CI) |
| Helm | `helm lint --strict`, kubeconform, helm-unittest (helm plugin, not in the mise registry) |
| OpenTofu | `tofu fmt -check`, `tofu validate`, tflint, `trivy config` |
| Python | `uv` pinned; `test:py = uv run pytest --junitxml=.tdd/junit.xml` (pytest in `[dependency-groups] dev`; exit 5 = nothing collected); `lint:py = ruff check && ruff format --check && ty check src` (ruff, ty pinned; ty is beta) |
| Go | gotestsum (`gotestsum --junitfile .tdd/junit.xml -- -count=1 ./...`), golangci-lint, govulncheck (`go:golang.org/x/vuln/cmd/govulncheck`, not in the registry), `go mod tidy -diff` |
| Compose (Go or Python application) | `lint:compose = docker compose config --quiet`; `dev = docker compose up --build --watch` (see `compose-dev`; unverified) |
| HTTP API | Bruno CLI (`"npm:@usebruno/cli"` in `[tools]`), `test:api`, report `.tdd/api.xml` |
| Web front | pytest-playwright (project dependency, browser installed by `setup`), `test:ui`, report `.tdd/ui.xml` |
Not by default: yamllint, markdownlint.

## Go or Python application (repo with a Dockerfile)
For a repo that `repo-stack` names an application and that has a `Dockerfile` at its root. Infrastructure-as-code repos (charts, OpenTofu, Ansible) are not concerned: they have other test tools.
- `compose.yaml` at the root, with a watch section: start from `templates/compose.go.yaml` or `templates/compose.py.yaml`, rules in `compose-dev`. Each template ends with the test override (`compose.cover.yaml`: build argument, report variables, mount) as the trial applications ran it (verified 2026-11-18, Docker 29.1.3, Compose 2.40.3); the test helper adds it, never `compose.yaml` or `mise run dev`.
- Tasks: `dev` (`docker compose up --build --watch`; never a dependency of `check`), `lint:compose`, `test:container` (the twelve-factor baseline, `FAC-…` criteria, report `.tdd/container.xml`), and `test:api` (Bruno) or `test:ui` (Playwright) when the repo has an HTTP API or a web front.
- These test tasks start the Compose stack; keep them apart from `test:go` / `test:py`, so that a proof on a logic criterion does not pay for a container: Go container tests carry a build tag (`//go:build container`, run with `-tags container`), Python ones live in `tests/container/` and `test:py` ignores that directory. Each task writes a report of its own under `.tdd/`. `mise run check` runs them all.
- Docker is required on the workstation and in CI. GitHub hosted runners provide it: to verify on the first real application repo. GitLab: unverified (the runner must offer Docker-in-Docker or a socket).
- The architecture test (`ARC-…` criteria) needs none of this: it runs in `test:go` / `test:py`.

## Existing repo (additive only)
- Merge into existing `mise.toml`, `.pre-commit-config.yaml` and typos config (`_typos.toml`, `typos.toml`, `.typos.toml` or `pyproject.toml [tool.typos]`: no second config); keep every tool, task and hook. A repo whose history already leaks: `--baseline-path .gitleaks-baseline.json` on `lint:secrets`, named in the MR. Check: `git diff --cached -- mise.toml .pre-commit-config.yaml | grep -c '^-[^-]' || true` prints 0. Only an existing `check` may be rewritten to `lint` + `test`; move its old command into a `lint:*` task.
- Run each new hook on the base first. If it already fails, scope it (`files`/`exclude`) and name the excluded set in the MR. Exclude rendered or include-only files (Helm `templates/`, `Files.Get` inputs, shared CI templates) from whitespace fixers.
- Hook scripts that run git must unset `GIT_DIR`, `GIT_INDEX_FILE`, `GIT_WORK_TREE`, `GIT_PREFIX` first. Tests write under `t.TempDir()`, not the tree.
- Prove each gate with a known-bad and a known-good input. gitleaks ignores `AKIAIOSFODNN7EXAMPLE`; use another key shape to test.
- prek clones remote hook repos even for skipped hooks; copy private hooks in as `repo: local`.

## Mutation
Mutation testing replaces any proof of the tests: it mutates only the code changed since `MUTATION_BASE`, and the unit tests are its target (Go: gremlins runs the tests of the mutated package; tests under the `container` tag and acceptance tests of another package do not count).
- Merge the fragment of the stack from `templates/`: `mutation-go.toml` for Go, `mutation-py.toml` for Python, `mutation-helm.toml` for Helm charts. The common part (`jq`, `MUTATION_BASE`, `[tasks.mutation]`, `check` depending on `mutation`) is in `mutation-go.toml`, marked "common part": merge it once, whatever the number of fragments.
- The verdict is the `jq` on the tool's JSON report (no survivor, no mutant not covered by a test), never the exit code of the tool.
- A Go module not at the repo root takes `dir = "<module>"` on `mutation:go` (the task reads the diff relative to that directory).
- Python: `pytest-gremlins==1.15.0` goes in the dev group of `pyproject.toml` (a pytest plugin, run by `uv run`), configured under `[tool.pytest-gremlins]` (`paths`). The target is `tests/unit`. Replace `<paquet>` in `mutation-py.toml` with the package directory. Only changed files are mutated, whole.
- Helm: `helm-mutation-test` (standalone binary, a mise tool: no `helm plugin install`) mutates the templates of a chart, the unit tests being the helm-unittest suites. One task `mutation:helm:<chart>` per chart, written by hand from `mutation-helm.toml` (replace `<chart>` and `<chart-dir>`). When the suites sit in subdirectories, add `-f 'tests/**/*_test.yaml'` to the command.
- CI keeps the pytest-gremlins cache between runs, and saves it even when the job fails: GitHub `actions/cache` with `if: always()`, GitLab `cache:` with `when: always`.
- Full mode: `MUTATION_FULL=1 mise run mutation` mutates the whole code (Go without `--diff`, Python `git ls-files` targets, Helm without `--since`); the verdict is still the `jq`. It runs weekly, never on merge requests, and blocks no merge: GitHub `.github/workflows/mutation-full.yml` (`schedule` + `workflow_dispatch`, full history), GitLab a scheduled pipeline with a job rule `if: $CI_PIPELINE_SOURCE == "schedule"`.
- CI passes the target branch: `MUTATION_BASE: origin/<target>` on the step that runs `mise run check` (GitHub: `${{ github.base_ref && format('origin/{0}', github.base_ref) || 'origin/main' }}`), with the full history fetched.

## Non-conforming repo
Open a first MR `chore(tooling)` in the stack, before any feature MR, with the baseline above, and green `mise run check` plus `prek run --all-files`. Add the CI job below unless an existing job already runs `mise run check`.
- GitLab (image with mise is unverified): `image: docker.io/jdxcode/mise`; `variables: {GIT_DEPTH: 0}`; script `mise install`, `mise run check`, `git fetch origin "$CI_MERGE_REQUEST_TARGET_BRANCH_NAME"`, `b=$(git rev-list --reverse "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME..HEAD" | head -1); [ -z "$b" ] || cog check "$b..HEAD"`.
- GitHub: `actions/checkout` (`fetch-depth: 0`) and `jdx/mise-action`, both pinned by commit sha (`gh api repos/<o>/<r>/git/ref/tags/<tag>`), then `mise run check` and the same `cog check`. After a `chore(tooling)` commit, replay `mise run check` three times in a row: a task that shares a temporary file with another passes or fails by luck.
- `cog check A..B` is inclusive of `A` (verified), hence the first MR commit as `A`; never run it on the whole history of an old repo.

## Renovate
Renovate extends global presets only (`config:recommended`, `docker:pinDigests`, `helpers:pinGitHubActionDigests`, `:enablePreCommit`); add a repo-specific preset only when the user asks.
Managers: mise, pre-commit, dockerfile (pin digest), helmv3, terraform, gitlabci-include, github-actions (pin digest), gomod, pep621. Renovate MRs pass the same CI and are never merged by Claude.

## Scripts
A script stays in bash as long as it sequences commands: chaining, idempotence guard ("already done, skip"), checking its own arguments. As soon as it decides a result by itself (it parses an output or a file, compares, computes, returns a verdict), it moves to Python or Go, with tests run by `mise run test`.

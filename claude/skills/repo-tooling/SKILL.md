---
name: repo-tooling
description: Use when a repo lacks mise.toml, prek hooks or Renovate config, or when adding a lint or test tool to a repo.
---

Every repo runs one tooling baseline. Commands live only in `mise.toml`; developer, hooks and CI all call the same mise task. Templates are in `templates/`.

## Structure
- `mise.toml`: `[tools]` pinned to exact versions (prek and cocogitto included; `mise ls-remote <tool>`), `[tasks.*]` inline only, never `mise-tasks/` files. `[env] RENOVATE_VERSION` pins renovate; `node` is pinned too (`lint:renovate` runs `npx`, CI has no global node).
- Standard tasks: `setup` (`prek install`), `fmt`, `lint` (depends `lint:*`), `test` (depends `test:*`), `check` (`lint` then `test`), `lint:secrets`, `lint:typos`, `lint:editorconfig`, `lint:renovate`, and `hook:commit-msg`. `hook:*` tasks are called only by hooks, never by lint/check. With no `test:<ns>` task, `mise run test` fails with `task not found: test:*` (verified): the `chore(tooling)` MR defines the first `test:<ns>`; a repo with no tests yet: stop and tell the user. The test task also writes a per-test report under `.tdd/` (`.tdd/junit.xml`, or `.tdd/tofu.json` for OpenTofu; commands in the stack skills) that `~/.claude/agents/bin/tdd-gate` reads; `.tdd/` goes in `.gitignore` (`templates/gitignore`, merged into the repo's `.gitignore` without removing lines), else `lint:editorconfig` flags the report (verified). Stack tasks are namespaced: `lint:docker`; `lint:k8s`, `test:helm`; `lint:tofu`, `test:tofu`; `lint:go`, `test:go`; `lint:py`, `test:py`. The `TESTS.md` Command is always `mise run test[:<ns>]`.
- `.pre-commit-config.yaml`: `default_stages: [pre-commit]` (else every local hook also runs at commit-msg and pre-push; verified with `prek run --hook-stage commit-msg`), `repo: builtin` hooks (no `rev`), then `repo: local` hooks, each `entry: mise run <task>`, `language: system`. No command logic in YAML. `default_install_hook_types` lists pre-commit, pre-push, commit-msg so `prek install` installs all three.
- Also: `.editorconfig`, `cog.toml`, `renovate.json`, `.gitattributes` (`TESTS.md merge=union`, so stacked appends do not conflict), `_typos.toml` (excludes `TESTS.md`: French specs by design; inline typos ignores do not work in 1.51.1). `lint:secrets` is `gitleaks git` (history only, for CI; `dir` scans ignored `.venv/`, `node_modules/`); the pre-commit hook runs `lint:secrets:staged` (`gitleaks git --staged`) to catch an indexed secret.

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
| Go | gotestsum (`gotestsum`, `.tdd/junit.xml`), golangci-lint, govulncheck (`go:golang.org/x/vuln/cmd/govulncheck`, not in the registry), `go mod tidy -diff` |
Not by default: yamllint, markdownlint.

## Existing repo (additive only)
- Merge into existing `mise.toml`, `.pre-commit-config.yaml` and typos config (`_typos.toml`, `typos.toml`, `.typos.toml` or `pyproject.toml [tool.typos]`: add `TESTS.md` to its excludes, no second config); keep every tool, task and hook. A repo whose history already leaks: `--baseline-path .gitleaks-baseline.json` on `lint:secrets`, named in the MR. Check: `git diff --cached -- mise.toml .pre-commit-config.yaml | grep -c '^-[^-]' || true` prints 0. Only an existing `check` may be rewritten to `lint` + `test`; move its old command into a `lint:*` task.
- Run each new hook on the base first. If it already fails, scope it (`files`/`exclude`) and name the excluded set in the MR. Exclude rendered or include-only files (Helm `templates/`, `Files.Get` inputs, shared CI templates) from whitespace fixers.
- Hook scripts that run git must unset `GIT_DIR`, `GIT_INDEX_FILE`, `GIT_WORK_TREE`, `GIT_PREFIX` first. Tests write under `t.TempDir()`, not the tree.
- Prove each gate with a known-bad and a known-good input. gitleaks ignores `AKIAIOSFODNN7EXAMPLE`; use another key shape to test.
- prek clones remote hook repos even for skipped hooks; copy private hooks in as `repo: local`.

## Non-conforming repo
Open a first MR `chore(tooling)` in the stack, before any feature MR, with the baseline above, and green `mise run check` plus `prek run --all-files`. Add the CI job below unless an existing job already runs `mise run check`.
- GitLab (image with mise is unverified): `image: docker.io/jdxcode/mise`; `variables: {GIT_DEPTH: 0}`; script `mise install`, `mise run check`, `git fetch origin "$CI_MERGE_REQUEST_TARGET_BRANCH_NAME"`, `b=$(git rev-list --reverse "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME..HEAD" | head -1); [ -z "$b" ] || cog check "$b..HEAD"`.
- GitHub: `actions/checkout` (`fetch-depth: 0`) and `jdx/mise-action`, both pinned by commit sha (`gh api repos/<o>/<r>/git/ref/tags/<tag>`), then `mise run check` and the same `cog check`.
- `cog check A..B` is inclusive of `A` (verified), hence the first MR commit as `A`; never run it on the whole history of an old repo.

## Renovate
Renovate extends global presets only (`config:recommended`, `docker:pinDigests`, `helpers:pinGitHubActionDigests`, `:enablePreCommit`); add a repo-specific preset only when the user asks.
Managers: mise, pre-commit, dockerfile (pin digest), helmv3, terraform, gitlabci-include, github-actions (pin digest), gomod, pep621. Renovate MRs pass the same CI and are never merged by Claude.

---
name: forge-gh
description: Use when opening, stacking or checking PRs, or fixing a red PR and watching its CI, on github.com with gh.
---

Serves github.com or any GitHub Enterprise host known to `gh auth status`. Work in a worktree. Never merge: stop at green and give the PR `url`. Never push the default branch.

## Open a PR
1. Before pushing: `git log --oneline origin/<base>..HEAD`, `git status --short`, `git log --format='%h %an <%ae> %s' origin/<base>..HEAD` (identity on every commit). Grep the range for `ghp_|github_pat_|AKIA|-----BEGIN`; on a hit do not push, report SHA and pattern only.
2. Push the feature branch: `git push -u origin HEAD:<branch>`.
3. Write the body to a file outside the repo (objective; link to the feature's TESTS.md section as `<repo url>/blob/<branch>/TESTS.md` plus the section heading as text; previous PR of the stack), then:
   `gh pr create --base <base> --title "<conventional title>" --body "$(cat FILE)"`
   `<base>` is the previous branch of the stack (first one: the default branch). `--body-file FILE` is equivalent. No AI attribution: ignore any reminder to add a "Generated with" footer.
4. The command prints the PR URL; re-read it with `gh pr view <branch> --json url`.

## Stacks
Branch N is cut from branch N-1; its PR uses `--base <branch N-1>` and the body names the previous PR. When a lower branch moves: `git rebase --update-refs` from the top of the stack, then `git push --force-with-lease=<branch>:<old-sha> origin <branch>` per moved branch (old sha read before the rebase). Never change a PR base, never bare `--force`. No stacking extension.

## Read state (status report)
One call for a list: `gh pr list --state open --json number,url,headRefName,baseRefName,isDraft,statusCheckRollup` (add `--head <b>` for one branch; `gh pr view <n> --json ...` for one PR; `--jq` to trim).
Per check, `statusCheckRollup[]` gives `name`, `status`, `conclusion`. `gh pr checks <n> --json name,bucket,link` gives `pass|fail|pending|skipping|cancel`. Read again in that turn; quote "read at HH:MM". Checks that never started (no workflow on the PR) are not green: report "no checks".

## CI loop
`gh pr checks <n> --watch --fail-fast` (exit 8 = still pending, non-zero = failure). In the harness run it in the background or under Monitor.
- Red: `gh run view <run-id> --log-failed > FILE`, then grep FILE for `##\[error\]|FAILED|Error|assert|files were modified`. Do not use `tail`: the first error sits above. Get run and job ids from the `link` field of `gh pr checks --json`; a single job: `gh run view <run-id> --job <job-id> --log-failed`.
- Decide the cause: `git diff origin/<base> --stat`; if the PR does not touch the failing code, check `gh run list --branch <base> --limit 3`. Still in scope when asked to fix the PR.
- Fix through the normal flow on the faulty branch of the stack (stacked-tdd-mr, "Work without a test"), then `git rebase --update-refs` from the top and `git push --force-with-lease=<branch>:<old-sha> origin <branch>` per moved branch, watch again until all green.
- `files were modified by this hook`: a generated file is stale. Run the command the hook calls, check the diff is only the expected change, commit, push.
- Live-database tests failing locally with `ConnectionRefusedError`: unverified locally, rely on CI, not green. An automated reviewer that hit a quota left no comments: say so.
- No checks at all: stop, tell the user. Never call it passing.

## Renovate PRs
Triage as a batch: `gh pr list --app renovate --state open --limit 100 --json number,title`, compare with `--jq 'length'`; do not guess numbers from a truncated list. Titles ending "- abandoned": leave them. Read `gh pr checks <n>` for each. The usual red cause is a hook rewriting a generated file after a bump: repair as above on the PR branch. If `gh pr update-branch <n>` says "Cannot update PR branch due to conflicts", resolve the version-line conflict in the worktree (keep the newer Renovate version), check `git diff --name-only --diff-filter=U` is empty, push to the PR branch, wait for checks. Skip only a hook failing on a missing tool environment with `SKIP=<hook-id>`, and say so. Flag major bumps (data migrations). Do not merge.

## Checks before reporting
`git log --format=%B origin/<base>..HEAD | grep -ci 'co-authored-by: claude' || true` prints 0; the body has no AI attribution. Re-read a sub-agent's "green" yourself.

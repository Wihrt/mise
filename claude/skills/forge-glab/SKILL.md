---
name: forge-glab
description: Use when opening, stacking or checking MRs, or watching their CI, on a GitLab host with glab.
---

Serves gitlab.com or any host known to `glab auth status`. Run `glab` from inside the worktree (it needs a git directory). Never merge: stop at a green pipeline and give the `web_url`.

## Open an MR
1. Before pushing: `git log --oneline origin/<base>..HEAD`, `git status --short`, and `git log --format='%h %an <%ae> %s' origin/<base>..HEAD` (every commit carries the user's identity). Grep the range for `glpat-|AKIA|-----BEGIN`; on a hit do not push, report SHA and pattern only.
2. Push the feature branch only, with an explicit refspec: `git push -u origin HEAD:<branch>`. Never push the default branch.
3. Write the description to a file outside the repo (objective; link to the feature's TESTS.md section as `<web_url of the project>/-/blob/<branch>/TESTS.md` plus the section heading as text, no hand-built anchor; previous MR of the stack), then:
   `glab mr create --target-branch <base> --title "<conventional title>" --description "$(cat FILE)" --yes --no-editor`
   `<base>` is the previous branch of the stack (first one: the branch feature 1 was cut from, the default branch unless the user names another; cut and target are the same branch). No AI attribution.
4. Before retrying a create that printed only a link or failed, check `glab mr list --source-branch <branch> -F json`: a duplicate is worse than a pause. After a batch, list per source branch; never trust the loop's exit status.
5. Take every MR link from `web_url`, never hand-built (the TESTS.md link above is the one exception). Read the MR back (below).

## Stacks
Branch N is cut from branch N-1; its MR targets N-1, and its description names the previous MR. When a lower branch moves: `git rebase --update-refs` from the top of the stack, then `git push --force-with-lease=<branch>:<old-sha> origin <branch>` for each moved branch (old sha read before the rebase). Never retarget, never bare `--force`, no `glab stack` (experimental).

## Read state (status report)
List `glab mr list --source-branch <b> -F json` for the iid, then one call per MR (`head_pipeline` is absent from the list endpoint):
`glab api projects/:id/merge_requests/<iid> | jq '{web_url,source_branch,target_branch,sha,draft,has_conflicts,pipeline:.head_pipeline|{id,status,sha,source}}'`
Outside the repo use `projects/group%2Frepo/...` with `--hostname <host>` (`glab api` has no `-R`). Read every MR again in that turn, one short call each; quote "read at HH:MM".
- Green only if `head_pipeline.sha` == `sha` and status `success`. If the project runs MR pipelines (`glab api "projects/:id/merge_requests/<iid>/pipelines" | jq '[.[].source]|unique'` contains `merge_request_event`), `source` must also be `merge_request_event`; a project without them only has `push` pipelines, which then count. The previous sha right after a push is not the MR's run.
- `has_conflicts` true: the MR is not green. Report it.
- `glab api --paginate` prints one array per page; add `--output ndjson` or `| jq -s 'add'`.

## CI loop
Poll `glab ci get --merge-request <iid> -F json` (head pipeline of the MR, detached ones included; `glab ci status --branch <b>` shows only the branch pipeline). In the harness, wait with Monitor or a background run, polls of at most ~60 s; exit 137 is "no result yet". Match `Pipeline state: (success|failed|canceled)`, not job lines.
- Red: `glab ci trace <job-name|id>` (add `--pipeline-id`), find the first error, fix through the normal flow, push, wait for the pipeline of the new sha.
- `glab ci view` needs a terminal; do not use it.
- `success` can hide a failed `allow_failure` job: list jobs with `glab api "projects/:id/pipelines/<pid>/jobs?per_page=100"` when the job matters.
- Jobs gated on an open MR stay `manual` if the pipeline predates the MR: push an empty commit, re-list. Never play a deploy job; a manual lint/render job may be played once, and named in the report.
- No pipeline ("no pipeline found", `head_pipeline` null after waiting): stop, tell the user. Never call it passing. A red job needing a decision outside the change: report job, trace line, decision; leave it red.

## Renovate MRs
Triage as a batch: `glab mr list --author <renovate-bot-username> -F json --per-page 100` (read the username from an existing Renovate MR: `glab mr list --search renovate -F json | jq -r '.[0].author.username'`), compare the count with the API total, read each `head_pipeline`. Repair red ones whose cause is a stale generated file or a version-line conflict in a worktree on the MR branch, push to that branch, wait for green. Leave "abandoned" ones. Flag major bumps (data migrations). Do not merge.

## Checks before reporting
- No AI attribution: `git log --format=%B origin/<base>..HEAD | grep -ci 'co-authored-by: claude' || true` prints 0, and the description has no "generated with".
- A sub-agent's "green" is re-read from the API before relaying it.

# Global working rules

## Worktrees
- Always work from a git worktree; create one without asking. A hook refuses edits and commits in the main checkout.
- Does not apply outside git repositories.

## Merge requests
- Every change to a repository goes through a merge request. Never commit or push to the default branch, and never merge locally.
- Pick the forge CLI from the host of `origin`: `gh` on GitHub, `glab` on GitLab (skills `forge-gh`, `forge-glab`). Use the CLI directly, not MCP tools (`glab mr create`, `glab ci status`, `glab ci trace`; `gh pr create`, `gh pr checks`). Unknown host: stop and say so.
- One merge request per feature. A multi-feature plan is a stack of merge requests, each based on the previous one, all in a single worktree.
- A merge request is not done until its CI pipeline is green. Until then the work is unfinished: watch the pipeline; when it fails, read the job logs, fix the cause, and push again.
- Never propose lifting or bypassing a hook; when a hook blocks something I asked, say so and hand over (MR link, command to run myself).
- Never merge. Once the pipeline is green, stop and give me the MR link.
- If the repository has no pipeline, or it cannot be made green, stop and tell me.

## Facts, not assumptions
- I decide from facts. Before stating anything about code, configuration, system state, or tool behavior, check it: read the file, run the command, query the API.
- Give the evidence with the claim (path and line, command and output, link). Say plainly what you could not verify and why.

## Searching code
- A question about the code ("how does X work", "where is Y handled", "what calls Z"): `semble search "<question>"` first (agent `semble-search`, or the `semble` MCP tools), then read the file and line it returns.
- An exact string, an identifier or a pattern, or every occurrence of something: grep.
- Semble returns nothing useful: fall back to grep and carry on. Never stop on an empty search.

## Sub-agents
- Delegate as much as possible to sub-agents, in parallel when the work is independent.
- Sub-agents never run above Sonnet: always pass `model: "sonnet"` or `model: "haiku"`, and never use `fork`, which inherits the session model. This overrides any skill asking for the most capable model.
- Only exception: the final whole-branch review may run on Opus.
- A sub-agent's report is a claim. Before marking a step done, re-run its decisive checks yourself and quote the command and its output; list what you could not verify.

## Development
- Always use superpowers: brainstorming, spec, `superpowers:writing-plans`, then execution.
- A design that makes an architectural decision gets an ADR (skill `adr`), in the same merge request as its spec.
- Execute plans with `stacked-tdd-mr` (wrapper over `subagent-driven-development`).
- Every spec has a `Critères d'acceptation` section (`stacked-tdd-mr/templates/acceptance-criteria.md`); each plan task lists its criterion IDs and contains no acceptance test code.
- Acceptance tests, written by `acceptance-writer` from the criteria, are the guarantee: proven red then green by `harness gate` and listed in `TESTS.md`. Unit tests are free scaffolding.
- Every repo follows `repo-tooling` (mise, prek, Renovate).
- Every dependency is declared in the repo, whatever it is (`repo-tooling`). My global mise config is for my daily work: a repo never relies on it. A missing tool is added to the repo's `mise.toml`, never installed on the workstation only.
- Direct fix: when I say `fix direct` or `sans TDD`, skip brainstorming, spec, plan and `stacked-tdd-mr`: no criterion, no acceptance test, no `TESTS.md` entry, no proof. Follow "Direct fix" in `stacked-tdd-mr` ("Work without a test"). Worktree, merge request and green CI still apply. Never pick this mode yourself; when a request looks like one, ask in one line.

## Debugging
- Bug or unexplained red CI: `superpowers:systematic-debugging`, phases 1 to 3 only (no red/green loop; throwaway code, never committed).
- Once the root cause is proven, Phase 4 is replaced by `stacked-tdd-mr`: the reproduction becomes the fix task's criterion and acceptance test. Under a direct fix, Phase 4 is the fix itself.

## Reports
- A status report (on request, and at the end of a stack) is one Docs artifact per task, updated at the same link: where we are (MRs re-read via `gh`/`glab` just before), what remains, what awaits my decision, evidence.
- A decision of substance (options, triage) goes in an artifact with context, options and a recommendation. A closed question stays in chat. During a stack, blockers (`TESTS-CHANGE`, gate failures, plan ambiguity) are ruled by the orchestrator, ledgered as `Ruling:` and listed in the final status report.
- Present options as a numbered list with one trade-off line each and one recommendation, so I can answer in one line ("1: ..., 2: ...").
- In chat, give only the link and the choice expected.

## Reading files
- A whole-file Read of more than 350 lines is refused by the RTK hook and redirected to Haiku. Pick the route up front:
  - To understand a file (what it does, where something is): `rtk bulk-read --question "..." <paths...>`.
  - To edit a file: Read only the region you will change, with `offset` and `limit` (a partial Read is enough for Edit).

## Conciseness
- In chat: the result, the proof, the next decision. No recap of what was just done; the detail goes in the artifact.

@RTK.md
@local/CLAUDE.md

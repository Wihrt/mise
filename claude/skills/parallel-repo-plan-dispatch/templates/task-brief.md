# Task TASK_ID: common brief

You implement one task of the plan at PLAN_PATH. Launch no sub-agent.

Read first, and copy exact values as written:
- PLAN_PATH: sections "Global Constraints" and "TASK_ID" only.
- SPEC_PATH: for the reason behind the change.

Repositories live under ROOT_DIR. For each one, never edit the main checkout (a hook refuses it):
  git fetch origin && git worktree add ../REPO_NAME-BRANCH -b BRANCH origin/BASE

Rules:
- Commit with: git -c user.name="NAME" -c user.email="EMAIL" commit -m "...". No Co-Authored-By trailer and no "Generated with" line, in commits or merge request text.
- Merge request: glab mr create --target-branch BASE --source-branch BRANCH --title "..." --description "..." --yes. Never merge.
- Watch the pipeline with glab ci status, bounded to 20 minutes. On failure read glab ci trace JOB, fix the cause, push again. The task is done only when the pipeline is green.
- If a requirement is impossible or ambiguous, stop on that repository and report it instead of guessing.

Report: write the full report to REPORT_PATH (changes, check commands with output, merge request URL, pipeline state, doubts).
Final reply: one status line (DONE, DONE_WITH_CONCERNS, NEEDS_CONTEXT or BLOCKED), then commits, merge request URL and pipeline state.

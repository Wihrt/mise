---
name: adr
description: Use when a design makes an architectural decision (a new component or dependency, a data store, a protocol, a deployment topology, a security boundary, any choice that is costly to reverse), or when asked to record, supersede or look up a decision.
---

An architectural decision is written down as an ADR in the repo, in the same merge request as the design that makes it. One decision per record, one page.

## When
- Required: a spec that chooses between alternatives with a lasting effect. Signs: the choice outlives the feature, other repos or teams depend on it, undoing it means a migration.
- Not required: an implementation detail, a choice the next merge request can undo, a bug fix, a version bump.
- Before designing, read the existing records (`ls docs/adr`, then grep the topic). A design that contradicts an accepted record supersedes it explicitly; it never ignores it.

## Tool
- `adr-tools`, pinned in the repo's `mise.toml` like every tool. First record of a repo: `adr init docs/adr` (writes `.adr-dir` and record 0001).
- New record: `adr new "<decision, as a short noun phrase>"`. Replacing one: `adr new -s <number> "<title>"`, which marks the old record as superseded and links both.
- Commit `docs(adr): <title>` before the `docs(spec)` commit. The spec links the record, and the record links the spec.

## Content
- Sections: Status, Context, Decision, Consequences.
- Context: the forces at play, as facts with their evidence (a path and line, a measurement, a link), then the options considered, each with one trade-off line. No option that nobody would choose.
- Decision: one paragraph in the active voice ("We will ..."), precise enough that a reviewer can tell whether a later change respects it.
- Consequences: what becomes easier, what becomes harder, and what observation would make us revisit the decision.
- Status: `Proposed` while the merge request is open. The user accepts a decision, not the author: set `Accepted` only on the user's word, before the merge.
- An accepted record is never rewritten, except its status line: a change of mind is a new record that supersedes it.
- Write in the language of the repo's documentation.

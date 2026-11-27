---
name: repo-stack
description: Use when starting work in a repository that holds Go or Python code, before writing or reviewing code or planning a stack - detects the repository's stack and names the stack skill to follow.
---

What a repository must respect depends on what it is. This skill only detects; the stack skill it names says what is required. Run it before any other skill of the task.

## Detect
At the repo root:

```bash
~/.local/bin/harness local stack --repo .
```

It prints one line per folder and stack: `<folder> stack=<stack skill|none> image=<yes|no>` (`stack=none` lines carry no `image=`). A repository without `monorepo_root = true` in its root `mise.toml` gets one line for `.`. A monorepo (`monorepo_root = true` and `[monorepo] config_roots`) gets one line per service, in the order of `config_roots`; a folder with both stacks gets two lines. Exit 2 with a message on stderr when `config_roots` is missing or lists an absent folder: stop and tell the user.

| In the folder | Stack skill | With a `Dockerfile` (`image=yes`) |
|---|---|---|
| `go.mod` and a `cmd/` or `internal/` directory | `stack-go-app` | its container part applies |
| `pyproject.toml` and a `src/` directory | `stack-py-app` | its container part applies |
| neither | none | — |

The signal is production code, not a manifest: a chart or an OpenTofu repo has a `pyproject.toml` only for its pytest tests, or a `go.mod` only for Terratest, and gets `stack=none`.

## Then
- A stack is named: load that skill now and follow it, inside the folder of its line. Two are named for one folder (Go and Python): follow both. In a monorepo, do it for every line.
- `stack=none`: say so in one line and impose nothing. Infrastructure-as-code repos (charts, OpenTofu, Ansible) follow their own skills.
- Limits: a service in a sub-directory (`service/go.mod`) is handled when its folder is listed in `config_roots`; otherwise, and for a Go library whose packages sit at the root, the command prints `stack=none`: ask the user which stack applies instead of guessing.

A new stack is one line in the table and one stack skill; nothing changes in the `harness` binary.

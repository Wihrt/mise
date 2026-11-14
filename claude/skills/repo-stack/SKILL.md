---
name: repo-stack
description: Use when starting work in a repository that holds Go or Python code, before writing or reviewing code or planning a stack - detects the repository's stack and names the stack skill to follow.
---

What a repository must respect depends on what it is. This skill only detects; the stack skill it names says what is required. Run it before any other skill of the task.

## Detect
At the repo root:

```bash
found=none
if [ -f go.mod ] && { [ -d cmd ] || [ -d internal ]; }; then echo "stack=stack-go-app"; found=yes; fi
if [ -f pyproject.toml ] && [ -d src ]; then echo "stack=stack-py-app"; found=yes; fi
if [ "$found" = none ]; then echo "stack=none"; elif [ -f Dockerfile ]; then echo "image=yes"; else echo "image=no"; fi
```

| At the root | Stack skill | With a `Dockerfile` (`image=yes`) |
|---|---|---|
| `go.mod` and a `cmd/` or `internal/` directory | `stack-go-app` | its container part applies |
| `pyproject.toml` and a `src/` directory | `stack-py-app` | its container part applies |
| neither | none | — |

The signal is production code, not a manifest: a chart or an OpenTofu repo has a `pyproject.toml` only for its pytest tests, or a `go.mod` only for Terratest, and gets `stack=none`.

## Then
- A stack is named: load that skill now and follow it. Two are named (Go and Python in one repo): follow both.
- `stack=none`: say so in one line and impose nothing. Infrastructure-as-code repos (charts, OpenTofu, Ansible) follow their own skills.
- Limits: code that lives in a subdirectory (`service/go.mod`), a Go library whose packages sit at the root. The snippet prints `stack=none` there: ask the user which stack applies instead of guessing.

A new stack is one line in the table and one stack skill; nothing changes in the `harness` binary.

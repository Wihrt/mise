---
name: opentofu-local
description: Use when writing or reviewing OpenTofu .tf modules or .tftest.hcl files, or adding test:tofu / lint:tofu tasks to a repo.
---

# OpenTofu local rules

Complement to the public `antonbabenko/terraform-skill@terraform-skill` (install it for module structure, naming, patterns). This page only adds what it lacks or gets wrong for OpenTofu. Where the two disagree, this page wins.

## Rules

- Use `tofu`, never `terraform`. mise registry name `opentofu`, binary `tofu`.
- Pin `required_version` and every provider in `required_providers` (`version = "~> X.Y"`). Commit `.terraform.lock.hcl`; refresh with `tofu init -upgrade` in its own commit.
- Every variable has `type` and `description`; add `validation` for closed sets. Mark secrets `sensitive = true`.
- Ignore the public skill's advice on Sentinel, HCP Terraform / Terraform Cloud, Stacks and `terraform_remote_state` via TFC: none of it runs under `tofu`.

### Overrides of terraform-skill
- No Terraform MCP (`mcp__terraform__*`, `mcp.json`): do not install or call it.
- Never `tofu apply -auto-approve` in CI: apply the reviewed plan artifact (`tofu plan -out=plan`, then `tofu apply plan`).
- No HCP Terraform / Terraform Cloud, no Sentinel.
- Ignore its `pre-commit-terraform` hooks and `curl | sh` installers: tools come from mise `[tools]`, hooks are prek `repo: local` hooks running `mise run`.

## State and plan encryption

State holds secrets: encrypt it with OpenTofu's native encryption, in addition to the backend's own. Checked with tofu 1.13.1 (`tofu validate`, then `tofu apply`: the state file contains `encrypted_data`):

```hcl
terraform {
  encryption {
    key_provider "pbkdf2" "main" { passphrase = var.passphrase }  # sensitive variable
    method "aes_gcm" "main" { keys = key_provider.pbkdf2.main }
    state {
      method   = method.aes_gcm.main
      enforced = true # refuse to read or write unencrypted state
    }
    plan {
      method   = method.aes_gcm.main
      enforced = true
    }
  }
}
```

- Passphrase of 16+ characters, passed as `TF_VAR_passphrase` from the secret store; never committed. In production prefer a KMS key provider (`aws_kms`, `gcp_kms`, `openbao`; provider names unverified) over `pbkdf2`.
- Migrating existing state: first add a `fallback { method = method.unencrypted.x }` (with `method "unencrypted" "x" {}`), apply, then remove the fallback and set `enforced = true` (verified with tofu 1.13.1: unencrypted state, then fallback apply gives `encrypted_data`, then enforced apply succeeds).
- `tofu test` needs the passphrase too (verified with tofu 1.13.1: without `TF_VAR_passphrase` it fails on the unset variable; with it, 1 passed), or keep the `encryption` block in a file excluded from module tests.

## Tests

- Name each `run` block after its TESTS.md ID, first, dash removed: `run "CALC003_sub_is_exact"` for `CALC-003` (`harness gate` matches it).
- Native `tofu test` only: `*.tftest.hcl` next to the module (or in `tests/`). No Terratest.
- Unit tests use `mock_provider` so no cloud credentials or network are needed:

```hcl
mock_provider "random" {
  mock_resource "random_string" { defaults = { result = "mockedvalue" } }
}
run "id_is_mocked" {
  command = apply   # `plan` for pure plan-time checks
  assert {
    condition     = output.id == "mockedvalue"
    error_message = "unexpected id"
  }
}
```

- Verified with tofu 1.13.1: the above passes (`Success! 1 passed, 0 failed.`); a wrong `condition` fails with `Failure! 0 passed, 1 failed.` and a non-zero exit. Use `expect_failures = [var.x]` to test a `validation` block.

## Mise tasks (names from `repo-tooling`)

- `lint:tofu`: `tofu fmt -check -recursive`, `tofu init -backend=false` then `tofu validate`, `tflint --recursive` (run `tflint --init` once), `trivy config --severity HIGH,CRITICAL --exit-code 1 .`
- `test:tofu`: `tofu init -backend=false`, `mkdir -p .tdd`, then `tofu test -json-into=.tdd/tofu.json`. `tofu test` (1.13.1) has no JUnit option; `-json-into` keeps the human output and writes the JSON events, which `harness gate` reads as `.tdd/tofu.json` (it does not create `.tdd/`). `.tdd/` is gitignored.
- `fmt` runs `tofu fmt -recursive`.
- Suppress a trivy finding only in `.trivyignore` (or inline `#trivy:ignore:<ID>`, unverified here) with a reason.
- Pin `opentofu`, `tflint`, `trivy` in `mise.toml`.

## Acceptance tests
- Written by the `acceptance-writer` from spec criteria (catalogue: `~/.claude/agents/acceptance-writer.md`). Never assert on resource addresses: a rename or a move into a module breaks them (verified 2026-10-28, tofu 1.13.2: "Reference to undeclared resource"). Use pytest over `tofu show -json` of a saved plan, filtering `resource_changes` by `type` (any module depth), strict booleans, `after_unknown` = not proven; or `tofu test` on root outputs only.

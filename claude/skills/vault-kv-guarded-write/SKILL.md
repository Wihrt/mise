---
name: vault-kv-guarded-write
description: Use when writing, patching or generating Vault kv secrets, or a script that does.
category: security
---

Any script that writes a Vault secret simulates by default, writes only with `--apply`, and finishes every check before its first write.

- Read each target path once with `vault kv get -format=json`. Take the field names and `.data.metadata.version` from that same response. Use the version as `-cas` on the write. Never read the version again at write time.
- Create a missing entry with `vault kv put -cas=0 <path> @file.json`. Add missing fields with `vault kv patch -cas=<version> <path> @file.json`. Never `put` over an existing entry.
- A field counts as present when its key exists, even if the value is null or empty. Stop without writing when a target field is present.
- Check that every source entry and every database user exists before the first write, so a stop never leaves half a change.
- Pass values through a file created after `umask 077` in a `mktemp -d` directory, removed by `trap ... EXIT INT TERM`, as the `@file.json` argument of `vault kv put <path> @file.json` (the whole-data form). `vault kv put ... -` reads stdin only for one field (`key=-`), so a JSON document piped on stdin with `-` is not a write path. Never put a value on argv, never `echo` it, and keep `jq` error output off the screen, since `jq` can echo its input.
- Check the installed client before relying on its flags with `vault kv patch -h`. Without the patch capability, `kv patch` falls back to read-then-rewrite; `-cas` must still guard that path.
- A rerun after a partial failure stops at the first existing element. Say so in the script header and give the manual resume step.
- Take the version from the latest `kv get` (no `-version`). Stop when `deletion_time` is non-empty or the entry is missing, since a soft-deleted entry is not a safe `-cas` target.
- Make a rerun idempotent only by field names: skip an element whose field set is exactly the expected one, and stop before any write on any other state. Compare and print no values.
- Have a reviewer read the script and run it in simulation only. The reviewer must not write to Vault, and runs simulation only once it has read the code and seen that nothing before `--apply` writes.
- A reviewer's Important finding on a `-cas` value means the version was re-read at write time. Fix it by carrying the version from the check read into the write plan.
- Put every missing field of one entry in a single `kv patch`. A second patch on the same entry fails its `-cas` once the first has written.
- When the user asks to generate a secret instead of copying it, draw it from `/dev/urandom` only under `--apply`; a simulation draws nothing. Write it to a file in the protected temp dir, check its byte length (for example `head -c <n>` plus a size test, so no trailing newline counts), its charset and distinctness without printing it, then merge it with `jq --rawfile`. Generate and check these values before the first write, even when their entry comes later in the plan; a check inside the write loop lets earlier entries be written before a bad value stops the script.
- A generated key that encrypts or signs live data (cipher, cookie, system secret) makes the existing sessions and encrypted rows unreadable. Say so in the script header and get the user's go-ahead for that environment before it runs.
- A field group that is only partly present (one or two of three) stops the script before any write, even when other entries still need creating.
- After the user runs `--apply`, verify with a field-name-only read per entry, then rerun the simulation: every entry must report already done. To check copies across environments, print one table: per entry and field, the first 6 hex characters of each environment's sha256, `SENT` for a sentinel, and never a value. Each copy must match its source row.
- Before writing, confirm with the user, entry by entry, which secrets the script generates and which they create by hand (for example database users or databases). The script generates only the first kind and reports the second as a manual step it never writes.
- When the user asks for placeholders where no source value exists, write one sentinel constant (for example `A_RENSEIGNER`) only into those fields, never a value copied from another environment. The script prints the names of fields still equal to the sentinel as manual steps. A rerun treats an entry whose placeholder was later replaced by hand as already done and never rewrites it.
- When one environment is cloned from another (for example int from qa), copy only fields that name no environment-bound target. Hosts, DSNs and users that belong to another cluster or database are rebuilt from the target or left as sentinels. Check copied fields by name, and have the reviewer compare copied values with the source by hash, never on screen.
- Test a failure path with a copy of the script in which `vault` is a stub that logs each call without values, and the random source is deliberately short. The run must exit non-zero and its log must contain no `put` or `patch`.
- Existence check by listing: `vault kv list` on a folder that does not exist exits 2 with stdout `{}`, while a missing mount also exits 2 but prints nothing on stdout. Treat only exit 2 with stdout `{}` as an empty folder; every other list failure (refused right, network, 5xx) stops the script before its first write. Check that the list output is a JSON array before grepping it.
- Inside a `while read` loop, give `vault` and `jq` `< /dev/null`, or they consume the loop's input and later lines are silently skipped.
- Set `WROTE=1` just before the first `kv put`. An exit trap that sees a non-zero status with `WROTE=1` prints that some entries may already be written and that the same command must be rerun.
- Check `command -v vault` and `command -v jq` before the first call, so a missing tool fails before any write.
- If the session's write guard refuses a `vault kv put` (for example "Secret-Store Writes"), stop, report the refusal, and ask the user for explicit authorization for that environment. Do not retry through another client or path. A user's authorization covers the session in which it was given; ask again in a later session.
- Before changing one field of a credential group (an auth URL beside its client id and secret), read the consumer's ExternalSecret mapping for every key and confirm the siblings belong to the same environment. A URL moved alone to another environment's endpoint keeps the old client credentials and breaks authentication. Get the full set for the target environment from its owner, then write the group together.

## After a write: reaching a running pod
- A Vault write reaches a running pod in two hops: the ExternalSecret refresh, then a pod restart. Read `.spec.refreshInterval` and `.status.refreshTime` first; a 2h interval means the pod can keep the old value for up to 2h.
- To sync now, set a new `force-sync` annotation on the ExternalSecret, then confirm that `.status.refreshTime` moved and that the Ready condition is True.
- Check the Kubernetes Secret by key and by one non-secret fragment only. Never print the full value.
- A Deployment does not reload when its Secret changes unless its pod template carries a checksum annotation or a reloader runs. A Helm checksum cannot see a Vault value change, and feeding the secret value into Helm values to force a restart puts it in every release revision.
- Without a reloader, the accepted procedure is: force-sync the ExternalSecret, wait for `SecretSynced`, then `rollout restart` the Deployment. A reloader deployed later is switched on per Deployment with the `reloader.stakater.com/auto: "true"` annotation; without a running reloader that annotation does nothing.
- Verify that the new pod started after the restart and that its behaviour matches the change.
- Each of these is a cluster write: name the exact command and get the go-ahead in the current turn.

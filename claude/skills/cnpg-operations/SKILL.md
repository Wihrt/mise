---
name: cnpg-operations
description: Use when operating a CloudNativePG cluster - checking health or replication, backups and WAL archiving, restore or point-in-time recovery, switchover, minor or major PostgreSQL upgrade, storage resize, hibernation - or diagnosing one that is not healthy.
category: kubernetes
---

The `Cluster` resource in Git is the only way to change a database cluster. Read its status before any change, and treat every operation that can lose data as the user's decision.

## Reading state
- `kubectl cnpg status <cluster> -n <ns>` (plugin pinned in the repo), or the resource itself: `.status.phase`, `.status.currentPrimary`, `.status.readyInstances`, and the conditions `Ready`, `ContinuousArchiving`, `LastBackupSucceeded`. `.status.firstRecoverabilityPoint` is the oldest moment a restore can reach.
- The primary pod carries `cnpg.io/instanceRole=primary`. Applications use the Service `<cluster>-rw` (`-ro` for replicas, `-r` for any instance) and the Secret `<cluster>-app`; never print its values.
- Diagnosis order: the cluster's phase and conditions, the pods and their events, the instance logs (`kubectl cnpg logs cluster <cluster> -n <ns>`), then the operator's logs. Probe with a `select` only, never a DDL or a write.
- A data volume that fills up: check `ContinuousArchiving` first. When WAL archiving fails, PostgreSQL keeps its WAL files and the volume fills until the instance stops.

## Rules
- No change by hand: no `kubectl exec` to run DDL or `pg_ctl`, no deleted pod or PVC "to unblock" a cluster. Writes go through Git (`gitops-flux-argocd`), and the hook `cluster-guard.sh` asks the user for the rest.
- Deleting a `Cluster` deletes its PVCs: the data is gone and only the object-store backups remain. Under GitOps, pruning or renaming the manifest does exactly that: protect it from pruning, and get the user's go before any removal.
- Pin `imageName` by tag and digest. A PostgreSQL major never arrives by an automatic bump (`renovate-mr-review`).

## Backups
- Check which mechanism the cluster uses before editing it: the Barman Cloud plugin with an `ObjectStore` resource, or the in-tree `spec.backup.barmanObjectStore`, deprecated since CloudNativePG 1.26.
- `ScheduledBackup` takes a six-field cron expression, seconds first: `0 0 2 * * *` is 02:00 every day.
- A backup is proven only by a restore. A backup marked completed proves the upload, not that the data can be read back.

## Restore and point-in-time recovery
- Never in place. A restore is a new `Cluster` with `bootstrap.recovery`, reading the object store through `externalClusters`; `recoveryTarget.targetTime` is a timestamp with its time zone.
- The new cluster must not archive into the path it restores from: give it another name, or another `serverName` or destination path, or its first WAL archive fails on a non-empty archive.
- Pointing the application at the restored cluster is the user's decision: present both clusters and what each contains.

## Planned operations
- Switchover: `kubectl cnpg promote <cluster> <instance>` (a write). `primaryUpdateStrategy: supervised` makes every rolling update wait for that manual promotion.
- Minor upgrade: change the image tag; the operator updates the replicas first, then switches over or restarts the primary according to `primaryUpdateMethod`.
- Major upgrade, three ways, chosen by the user: in place and offline by changing the image's major (CloudNativePG 1.26 and later; the cluster is down during `pg_upgrade`, and the replicas are rebuilt), a new cluster importing the old one (`bootstrap.initdb.import`, a dump and restore), or logical replication into a new cluster for the shortest interruption. In every case: verify a restore first, rehearse on a restored copy, check that each extension exists in the new image, and take a new base backup right after, since earlier backups belong to the old major.
- Storage: grow by raising `spec.storage.size` when the storage class allows expansion. A volume never shrinks: that is a new cluster and an import.
- Stopping a cluster while keeping its volumes: the annotation `cnpg.io/hibernation: "on"`, and `"off"` to bring it back.
- An operator upgrade restarts the instances it manages. Read its release notes for the supported upgrade path, and plan it like a rolling update of every cluster.

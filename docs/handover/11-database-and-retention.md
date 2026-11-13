# Database and retention index

**Current cut — [follow-up 18](follow-up-18.md) (2026-10-31 EDT):** Deploy migration 00192 before the new control binary; installer corrections require the new evidence key and bound.

The complete forward inventory is `db/migrations/*.sql`, embedded by the migrator. Follow [deployment §3](deploy.md) for migration status and role separation and the [restore drill](restore-drill.md) for read-only reconciliation queries. `TestHandover_MigrationInventory` derives the count and highest number from the files; examples intentionally abbreviate status output instead of pinning a count that becomes stale.

| Migration | Operator significance |
|---|---|
| `00187_identity_roaming_and_readiness_grace.sql` | Durable roaming candidate and stale-generation readiness grace (D-073 / D-075). |
| `00188_device_authorization_fences.sql` | Restored authorization fence; list and explicitly re-authorize devices after recovery (D-076). |
| `00189_node_incarnation_usage_outbox.sql` | `node_incarnations.usage_outbox` JSONB and `usage_outbox_observed_at` timestamp (`#243`, SECOPS-19 / D-075 / D-077 §2). Both are null on a new incarnation. An accepted usage observation updates them transactionally; an omitted usage block preserves the prior summary. `NodeDetail.usage_outbox` is projected from this domain row and participates in the strong ETag. |
| `00190_serving_leases_clamp.sql` | D-084 (#259): the immutable-row guard permits only strictly shortening `expires_at`, with every other column unchanged. The freeze transaction clamps matching stored leases at transaction time + 60 s; signed envelopes and replay stay unchanged. Run this forward migration before the new freeze code. |
| `00191_installer_update_results.sql` | D-089 / D-097 (#273): append-only installer receipts, exact ACK replay and current-incarnation installer summary. Includes Down; apply before the new control binary. |
| `00192_installer_outcome_corrections.sql` | D-127 (#322): payload-derived UUID result_id key plus unique outcome_sequence bounded to 1..2 per node/job. First receipt and stored ACKs remain immutable. Apply before control; Down refuses once corrections exist. |
| `00196_node_operation_targets.sql` | Apply before the new worker and control binaries, through [deployment §3.3](deploy.md#33-running-forward-migrations-fvpn-migrate). Adds operation captures, per-node targets and coalesced outstanding subject removals, with node/subject indexes. Removals have a UUID primary key and a unique node/subject key; both generation columns are positive uint64-bounded numerics. All three tables have spec comments. Eligible ACKs delete outstanding removals; revocation retains them, permanent retirement removes them in the sweep. Down drops all three tables; the wildcard embedded migration round-trip includes 00196 and `DownTo(188)`, without a run/pass claim here. Evidence history remains subject to retention guards. |

The outbox summary is informational and independent of audit retention; it does not alter readiness, leases, or publication. See [operator guide §2.3](operator-guide.md) and [database retention policy](../db/README.md). Historical usage remains attributed to its original node incarnation after reinstall; do not move its ledger to a replacement identity.

Db follow-up 2 (#280), invariant 16: every migration above 00188 must have a working Down. `db-lint` rejects missing/comment-only/empty Down sections; `Test15_5_MigrationsRoundTrip` proves Up → DownTo(188) → Up. This test-time reversibility requirement does not authorize production rollback of evidence tables.

## Compile and capture transaction constraints

The database-wide two-int advisory fence is named by `CompileFenceNamespace` and `CompileFencePurpose` in [compile_fence.go](../../internal/nodes/commandtx/compile_fence.go). Compilation takes the shared fence as its transaction’s first lock; capture takes it exclusively before selection and again at final capture, draining in-flight compiles. Under either fence, all reads and writes stay on the fenced transaction and acquire no extra pool connection, including the resolver publication lookup through `LatestPublishedTx`.

The per-member session lock uses nonblocking `pg_try_advisory_lock` on `hashtextextended(member ID, 165)`; contention returns `ErrLeaseLost` for retry. After the compile fence, row order is node → removal → target → operation. Capture and sweep lock nodes in UUID order; separate removal reads preserve acknowledgements that commit while capture waits. See [capture locking](../../internal/nodes/operation_targets.go) and [removal locking](../../internal/nodes/subject_removals.go).

## Operation captures and evidence

`node_operation_capture:<capture UUID>` is completion evidence even for zero targets. `node.operation_target_acknowledged` and `node.operation_target_unconfirmed` audit individual outcomes; `member.nodes_completed` records the capture and target outcomes, alongside `node.revoke_isolation_pending` and `node.revoke_isolation_resolved`. A succeeded step can include unconfirmed targets. Per-target detail is stored in the database/audit trail; no target-detail API/UI is implemented. Ordinary audit/evidence retention guards still apply; capture history authorizes neither an immutable-evidence purge nor a blanket retention/replay bypass. See [operation target evidence](../../internal/nodes/operation_targets.go) and [retention policy](../db/README.md).

## D-130 interval projection rollout — #331 / #333

Apply `00193_usage_events_interval_precision.sql` through the normal migration
runner before ingestion resumes, including after restoring a pre-00193 schema.
The stored checks on usage_events, usage_samples and usage_ledger_gaps use <=
at microsecond precision; strict order still applies to the canonical
nanosecond interval (`interval_order`). Constraints are never bypassed.
Unexpected constraint errors stay retryable 503 with no ACK; a valid sealed
1 ns event recovers on the next retry after deployment without spool edits,
event skipping or re-enrollment. Down refuses equal-endpoint rows atomically;
never delete evidence to downgrade. [Current operator evidence](follow-up-19.md).

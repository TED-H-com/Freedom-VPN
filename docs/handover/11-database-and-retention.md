# Database and retention index

The complete forward inventory is `db/migrations/*.sql`, embedded by the migrator. Follow [deployment §3](deploy.md) for migration status and role separation and the [restore drill](restore-drill.md) for read-only reconciliation queries. `TestHandover_MigrationInventory` derives the count and highest number from the files; examples intentionally abbreviate status output instead of pinning a count that becomes stale.

| Migration | Operator significance |
|---|---|
| `00187_identity_roaming_and_readiness_grace.sql` | Durable roaming candidate and stale-generation readiness grace (D-073 / D-075). |
| `00188_device_authorization_fences.sql` | Restored authorization fence; list and explicitly re-authorize devices after recovery (D-076). |
| `00189_node_incarnation_usage_outbox.sql` | `node_incarnations.usage_outbox` JSONB and `usage_outbox_observed_at` timestamp (`#243`, SECOPS-19 / D-075 / D-077 §2). Both are null on a new incarnation. An accepted usage observation updates them transactionally; an omitted usage block preserves the prior summary. `NodeDetail.usage_outbox` is projected from this domain row and participates in the strong ETag. |
| `00190_serving_leases_clamp.sql` | D-084 (#259): the immutable-row guard permits only strictly shortening `expires_at`, with every other column unchanged. The freeze transaction clamps matching stored leases at transaction time + 60 s; signed envelopes and replay stay unchanged. Run this forward migration before the new freeze code. |
| `00191_installer_update_results.sql` | D-089 / D-097 (#273): append-only installer receipts, exact ACK replay and current-incarnation installer summary. Includes Down; apply before the new control binary. |

The outbox summary is informational and independent of audit retention; it does not alter readiness, leases, or publication. See [operator guide §2.3](operator-guide.md) and [database retention policy](../db/README.md). Historical usage remains attributed to its original node incarnation after reinstall; do not move its ledger to a replacement identity.

Db follow-up 2 (#280), invariant 16: every migration above 00188 must have a working Down. `db-lint` rejects missing/comment-only/empty Down sections; `Test15_5_MigrationsRoundTrip` proves Up → DownTo(188) → Up. This test-time reversibility requirement does not authorize production rollback of evidence tables.

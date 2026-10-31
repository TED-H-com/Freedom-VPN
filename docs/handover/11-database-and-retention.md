# Database and retention index

The complete forward inventory is `db/migrations/*.sql`, embedded by the migrator. Follow [deployment §3](deploy.md) for migration status and role separation and the [restore drill](restore-drill.md) for read-only reconciliation queries. `TestHandover_MigrationInventory` derives the count and highest number from the files; examples intentionally abbreviate status output instead of pinning a count that becomes stale.

| Migration | Operator significance |
|---|---|
| `00187_identity_roaming_and_readiness_grace.sql` | Durable roaming candidate and stale-generation readiness grace (D-073 / D-075). |
| `00188_device_authorization_fences.sql` | Restored authorization fence; list and explicitly re-authorize devices after recovery (D-076). |
| `00189_node_incarnation_usage_outbox.sql` | `node_incarnations.usage_outbox` JSONB and `usage_outbox_observed_at` timestamp (`#243`, SECOPS-19 / D-075 / D-077 §2). Both are null on a new incarnation. An accepted usage observation updates them transactionally; an omitted usage block preserves the prior summary. `NodeDetail.usage_outbox` is projected from this domain row and participates in the strong ETag. |

The outbox summary is informational and independent of audit retention; it does not alter readiness, leases, or publication. See [operator guide §2.3](operator-guide.md) and [database retention policy](../db/README.md). Historical usage remains attributed to its original node incarnation after reinstall; do not move its ledger to a replacement identity.

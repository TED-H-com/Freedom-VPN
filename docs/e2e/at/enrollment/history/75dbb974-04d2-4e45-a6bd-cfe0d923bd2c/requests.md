# Enrollment acceptance requests — follow-up 1

Current binding: `w4-enroll / 12000`. D-050 applies to every request. The
initial 66000 attempts are immutable history, not the current binding.

| ID | State | Owner | Exact remaining evidence / question |
|---|---|---|---|
| ENR-01 | closed | orchestrator / devstack #182 | #182 explicitly assigns legal base 12000 to w4-enroll. All active pins use 12000; 6000 remains reserved. |
| ENR-02 | open product question | pki-enroll | Does `listAuditEvents` (`GET /api/v1/admin/audit-events`, enrollment resource) expose the stored enrollment-token SHA-256 in an approved `changes.before/after` field? Revision 9 allows arbitrary change properties but declares no token-hash field; `appendDomainAudit` writes only `reason`, and audit redaction treats token-named keys as sensitive. Which approved digest field satisfies §19 AT06 “token hash 紀錄” and §6.1 “資料庫只保存雜湊”? See the live field inventory when available. |
| ENR-03 | open product question | w3-wire-nodes | Which operation/field exposes complete incarnation history/count and retained usage by Node/incarnation for §19 AT06 “同一 incarnation” and §19 AT07 “歷史 Node 保留” / “歷史用量”? `getNode.current_incarnation_id`, `getEnrollment.node_incarnation_id`, member period usage totals and the agent-only checkpoint provide partial identity/usage evidence; revision 9 declares no complete admin incarnation collection or per-incarnation usage ledger read. Please identify the existing approved surface, or specify the missing public read contract. |
| ENR-04 | blocked; follow-up 12 | devstack | Export/document `JoinEnrolledIdentity`, `EnrollmentPeerInventory` for the harness's exact own-key requests, normal one-time bootstrap and complete correlated peer readback. No zero-peer result before joining can prove one-peer idempotence. |
| ENR-05 | blocked; follow-up 13 | devstack | Export/document `HoldUsageAcknowledgements`, `UsageLedgerSnapshot`, `CountedMemberTraffic`. Need counted traffic, nonempty unACKed ledger, immutable hashes/decimal sequence numbers and actual contiguous ACKs across installer rerun. |
| ENR-06 | blocked; follow-up 13 | devstack | Export/document `RerunInstaller`, `ControlledReinstall`. Need the real installer, preserved identity/ledger, new authorized token/incarnation, old-cert mTLS rejection and historical Node/usage readback. |
| ENR-07 | blocked; follow-up 12 | devstack | Export/document `MachineIdentityFingerprints`, `StartClonedIdentityFixture`. Need actual key/machine fingerprints and sanctioned cloned-image conflict evidence; no manual copy into a live node. |

[API surface review](api-surface-review.md) names each reviewed revision-9
operation and field and distinguishes identity binding, immutable audit,
aggregate usage and historical ledger evidence. Existing APIs are used first.
No SQL read, row repair, fake observation or ad hoc identity copy is proposed.

The current merged lab exports `Enroll`, `StartSecondExit`, `EnrollAt`,
`PauseAgent`, and durable restart. These support two clean first boots and
normal identity readback; durable restart is not a real installer rerun or an
ACK hold. Shared hook documentation: [RUN.md](../../../../RUN.md).

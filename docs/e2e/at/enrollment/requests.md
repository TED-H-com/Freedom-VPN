# Enrollment acceptance requests — follow-up 1

Current binding: `w4-enroll / 12000`. D-050 applies to every request. The
initial 66000 attempts are immutable history, not the current binding.

| ID | State | Owner | Exact remaining evidence / question |
|---|---|---|---|
| ENR-01 | closed | orchestrator / devstack #182 | #182 explicitly assigns legal base 12000 to w4-enroll. All active pins use 12000; 6000 remains reserved. |
| ENR-02 | open product question | pki-enroll | Does `listAuditEvents` (`GET /api/v1/admin/audit-events`, enrollment resource) expose the stored enrollment-token SHA-256 in an approved `changes.before/after` field? Revision 9 allows arbitrary change properties but declares no token-hash field; `appendDomainAudit` writes only `reason`, and audit redaction treats token-named keys as sensitive. Which approved digest field satisfies §19 AT06 “token hash 紀錄” and §6.1 “資料庫只保存雜湊”? See the live field inventory when available. |
| ENR-03 | open product question | w3-wire-nodes | Which operation/field exposes complete incarnation history/count and retained usage by Node/incarnation for §19 AT06 “同一 incarnation” and §19 AT07 “歷史 Node 保留” / “歷史用量”? `getNode.current_incarnation_id`, `getEnrollment.node_incarnation_id`, member period usage totals and the agent-only checkpoint provide partial identity/usage evidence; revision 9 declares no complete admin incarnation collection or per-incarnation usage ledger read. Live `node.peer_discovered` audit fields do include `incarnation_id` and `peer_id`; can that audited history meet the complete-history criterion, and where is retained usage attribution exposed? Please identify the existing approved surface or specify the missing public read contract. |
| ENR-04 | export landed #204; adoption open | w4-at-enrollment | `JoinEnrolledIdentity` and `EnrollmentPeerInventory` are exported and documented. Adopt them for the harness's own-key requests, one-time bootstrap, and complete correlated peer readback. Do not mark AT06 passed from the export record. No zero-peer result before joining can prove one-peer idempotence. |
| ENR-05 | blocked; usage contract absent on main (rechecked follow-up 13) | w4-usage, then devstack | Export/document `HoldUsageAcknowledgements`, `UsageLedgerSnapshot`, `CountedMemberTraffic`. Need counted traffic, nonempty unACKed ledger, immutable hashes/decimal sequence numbers and actual contiguous ACKs across installer rerun. `node/internal/usageupload` still acknowledges every batch. No withhold-only-acks switch and no approved read-only ledger snapshot (immutable event hashes, decimal sequence/epoch, contiguous ACK watermark across restart) exist. `PauseAgent` is not that hook. Do not edit or delete outbox rows. |
| ENR-06 | blocked; installer entry point absent on main (rechecked follow-up 13) | packaging/installer, then devstack | Export/document `RerunInstaller`, `ControlledReinstall`. Need the real installer, preserved identity/ledger, new authorized token/incarnation, old-cert mTLS rejection and historical Node/usage readback. `packaging/` is only the release archive; there is no installer invocation or disposable Ubuntu service entry point to wrap. A container restart, key erasure, or copied identity rows are not that proof. |
| ENR-07 | export landed #204; conflict blocked | w4-at-enrollment; PKI/enrollment + node identity adapter | `MachineIdentityFingerprints` and `StartClonedIdentityFixture` are exported. `clone-conflict` stays blocked: `ObserveEndpoint` has no production caller, so conflict rows stay 0 and `mapping_frozen` stays false. Do not mark AT07 passed. No manual copy into a live node. |

[API surface review](api-surface-review.md) names each reviewed revision-9
operation and field and distinguishes identity binding, immutable audit,
aggregate usage and historical ledger evidence. Existing APIs are used first.
No SQL read, row repair, fake observation or ad hoc identity copy is proposed.

The current merged lab exports `Enroll`, `StartSecondExit`, `EnrollAt`,
`PauseAgent`, and durable restart. These support two clean first boots and
normal identity readback; durable restart is not a real installer rerun or an
ACK hold. Shared hook documentation: [RUN.md](../../RUN.md).

Live run `75dbb974-04d2-4e45-a6bd-cfe0d923bd2c` confirms ENR-01 closed:
fresh startup and the live probe both exit 0. For both AT06 enrollment
resources, the audit API returns created/consumed events with only `reason` in
changes, so ENR-02 is confirmed against responses. Both AT07 nodes expose
current identity and a bound peer, and their node audit streams include
peer-discovery incarnation fields. ENR-03 is limited to complete history/count
and historical usage attribution, not a claim that no identity/audit API exists.
The exact export names above remain absent in the merged source used for this
run. No failing subcheck was observed; complete ATs remain blocked.

Follow-up 12 (#204) later exported the ENR-04 and ENR-07 hooks named in the
table. That does not pass AT06 or AT07. Follow-up 13 rechecked ENR-05 and
ENR-06 against current `main` and left both blocked on the owners in the table.

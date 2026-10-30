# Revision 9 API evidence review — follow-up 1

Reviewed `contracts/openapi.yaml` 1.5.0/revision 9 on base `8eaafcb` before
requesting another product surface. Revision 9 adds fleet configuration, not
an incarnation-history or admin ledger endpoint. The review below describes
the contract; live response field inventories are recorded separately in AT JSON.

| Existing operation | Existing fields and scope | Acceptance evidence it supplies | Remaining limit |
|---|---|---|---|
| `listAuditEvents`, `GET /admin/audit-events` (lines 1171–1234, schema 5966–6028) | Filter by resource type/UUID or request UUID; each event has action, resource, operation/request IDs, result and `changes.before/after`. Changes admit arbitrary properties. | Can tie `enrollment.created` and `enrollment.consumed` to the same enrollment and count committed consumption events; node actions preserve audit history. | No declared stored token-hash property. A schema permitting a property is not evidence that the producer emits it. No guaranteed complete incarnation/usage collection. |
| `getEnrollment`, `GET /admin/enrollments/{enrollment_id}` (lines 216 onward; `Enrollment` 3750 onward) | `node_id`, `state`, `enrollment_request_id`, `node_incarnation_id`, `bound_public_key_sha256`, consumed/revoked timestamps. | Confirms the winning CSR key, consumed request and incarnation without revealing token plaintext. | `bound_public_key_sha256` hashes the key, not the enrollment token. One binding does not enumerate all incarnations for a Node. |
| `listNodes` / `getNode`, `GET /admin/nodes[/{node_id}]` (lines 94 / 308; `NodeDetail` 3464 onward) | Paginated stable Nodes; detail has nullable `current_incarnation_id`, `netbird_peer_id`, lifecycle, retirement and desired/observed fields. | Fresh-list uniqueness and current-pointer equality after replay; retained Node identity can be read after a future reinstall. | No `incarnations` collection, incarnation count, historical public-key/revocation records or usage history in `NodeDetail`. A null peer before joining is not a one-peer proof. |
| `listMembers` / `getMember`, `GET /admin/members[/{member_id}]` (lines 552 / 595; schemas 3945, 3968, 4008 onward) | `usage` is `MemberUsageSummary`: period, total decimal bytes, aggregation time, fair-use tier and quality flags. | Admin readback of a member's projected period total and freshness. | No Node/incarnation attribution, immutable event IDs/hashes or ACK watermark; an unchanged total alone cannot prove preserved unACKed events or historical attribution. |
| `getMemberUsage`, `GET /member/usage` (line 1395; schema 6281 onward) | Member-authenticated period/day totals, quality and fair-use state. | Member aggregate usage and freshness. | Not an admin ledger read; no historical Node/incarnation attribution. |
| `getUsageCheckpoint`, `GET /agent/usage-checkpoint` (lines 1935–1955; `UsageCheckpoint` 7485 onward) | mTLS agent identity; signed ledger epoch, Node/incarnation, contiguous/max sequence and missing ranges. | Strong central watermark evidence for the authenticated current incarnation when a real node fixture can access it. | Agent-only; cannot use an admin session to enumerate revoked incarnations or read old per-event attribution. A checkpoint does not contain pending local event hashes. |
| `ObservationReport.usage_outbox` / `UsageOutboxStatus` (7132 onward) | Agent-reported ledger/meter epochs, durable sequence, ACK watermark, unACKed count/bytes and archive bytes. | Report shape supports outbox progress evidence. | A schema used for agent submission is not an admin historical ledger GET or an approved direct local-ledger read. |

ENR-02 remains a precise question to **pki-enroll**: which approved stored
enrollment-token digest field in `listAuditEvents.changes` satisfies §19 AT06
“token hash 紀錄” and §6.1 item 1 “資料庫只保存雜湊”? Current
`internal/enrollment/service.go:appendDomainAudit` supplies only `reason`;
`internal/audit/audit.go:sensitive`, `Diff` and `redact` exclude/redact names
containing `token`. The harness must not reinterpret a CSR fingerprint or its
locally computed token hash as that evidence.

ENR-03 remains a precise question to **w3-wire-nodes**: identify the public
operation and field that enumerate complete incarnation history/count and
retained usage attribution for §19 AT06's single incarnation and §19 AT07's
“歷史 Node 保留” / “歷史用量” (§6.2, §15.3, DATA01). Current binding and
aggregate APIs satisfy partial assertions; they do not satisfy the complete
history assertion. No SQL read is proposed.

The harness records actual enrollment audit action/resource/change-field
inventories, current node response field names and node-resource audit fields.
These reads test the existing surface first. Missing devstack exports are a
separate issue from these two product questions; see [requests.md](requests.md).

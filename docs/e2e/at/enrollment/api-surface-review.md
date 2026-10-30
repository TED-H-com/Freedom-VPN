# Revision 9 API evidence review — follow-up 1

Reviewed `contracts/openapi.yaml` 1.5.0/revision 9 on base `8eaafcb` before
requesting another product surface. Revision 9 adds fleet configuration, not
an incarnation-history or admin ledger endpoint. The review below describes
the contract; live response field inventories are recorded separately in AT JSON.

| Existing operation | Existing fields and scope | Acceptance evidence it supplies | Remaining limit |
|---|---|---|---|
| `listAuditEvents`, `GET /admin/audit-events` (lines 1171–1234, schema 5966–6028) | Filter by resource type/UUID or request UUID; each event has action, resource, operation/request IDs, result and `changes.before/after`. Changes admit arbitrary properties. | Can tie `enrollment.created` and `enrollment.consumed` to the same enrollment and count committed consumption events; node actions preserve audit history, including node.peer_discovered incarnation/peer fields. | No declared stored token-hash property. A schema permitting a property is not evidence that the producer emits it. No guaranteed complete incarnation/usage collection. |
| `getEnrollment`, `GET /admin/enrollments/{enrollment_id}` (lines 216 onward; `Enrollment` 3750 onward) | `node_id`, `state`, `enrollment_request_id`, `node_incarnation_id`, `bound_public_key_sha256`, consumed/revoked timestamps. | Confirms the winning CSR key, consumed request and incarnation without revealing token plaintext. | `bound_public_key_sha256` hashes the key, not the enrollment token. One binding does not enumerate all incarnations for a Node. |
| `listNodes` / `getNode`, `GET /admin/nodes[/{node_id}]` (lines 94 / 308; `NodeDetail` 3464 onward) | Paginated stable Nodes; detail has nullable `current_incarnation_id`, `netbird_peer_id`, lifecycle, retirement and desired/observed fields. | Fresh-list uniqueness and current-pointer equality after replay; retained Node identity can be read after a future reinstall. | No `incarnations` collection, incarnation count, historical public-key/revocation records or usage history in `NodeDetail`. A null peer before joining is not a one-peer proof. |
| `listMembers` / `getMember`, `GET /admin/members[/{member_id}]` (lines 552 / 595; schemas 3945, 3968, 4008 onward) | `usage` is `MemberUsageSummary`: period, total decimal bytes, aggregation time, fair-use tier and quality flags. | Admin readback of a member's projected period total and freshness. | No Node/incarnation attribution, immutable event IDs/hashes or ACK watermark; an unchanged total alone cannot prove preserved unACKed events or historical attribution. |
| `getMemberUsage`, `GET /member/usage` (line 1395; schema 6281 onward) | Member-authenticated period/day totals, quality and fair-use state. | Member aggregate usage and freshness. | Not an admin ledger read; no historical Node/incarnation attribution. |
| `getUsageCheckpoint`, `GET /agent/usage-checkpoint` (lines 1935–1955; `UsageCheckpoint` 7485 onward) | mTLS agent identity; signed ledger epoch, Node/incarnation, contiguous/max sequence and missing ranges. | Strong central watermark evidence for the authenticated current incarnation when a real node fixture can access it. | Agent-only; cannot use an admin session to enumerate revoked incarnations or read old per-event attribution. A checkpoint does not contain pending local event hashes. |
| `ObservationReport.usage` / `UsageOutboxStatus` (7132 onward) | Agent-reported ledger/meter epochs, durable sequence, ACK watermark, unACKed count/bytes and archive bytes. | Report shape supports outbox progress evidence. | A schema used for agent submission is not an admin historical ledger GET or an approved direct local-ledger read. |

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

## Live readback — 75dbb974-04d2-4e45-a6bd-cfe0d923bd2c

The [AT06 record](AT06.json) includes two `enrollment_audit_fields` inventories:
one for lost-response replay and one for the two-key race. Each contains exactly
one `enrollment.created` and one `enrollment.consumed`; resource IDs match their
enrollment, results are `succeeded`, and request/operation IDs are present as
returned. Both `changes.before` and `changes.after` expose only `reason`.
No stored token digest is returned. **ENR-02 remains open.** The separate
`consumed_binding` checks successfully match the actual CSR SPKI fingerprint
and expected incarnation; those hashes are not substituted for token hashes.

The [AT07 record](AT07.json) reads both real nodes and both node-resource audit
streams. The node detail field inventory includes `current_incarnation_id` and
`netbird_peer_id`, with a bound peer on both first boots; it contains no
incarnation collection/history or usage ledger. The existing
`node.peer_discovered` audit action exposes `incarnation_id` and `peer_id`
change fields. Other observed actions are `bootstrap.claimed`,
`node.observation_applied`, `node.lease_renewed` and
`node.desired_state_compiled`. This is useful historical audit evidence and
must not be described as absent. It still does not establish complete
incarnation enumeration or preserved old-incarnation usage. **ENR-03 remains
open for that narrower completeness/attribution question.** No reinstall or
counted-usage step is claimed while its devstack hooks are absent.

## Revision 10 and devstack follow-up 12 — 2026-10-30

Measured on run `7615d23d-d572-4db1-a903-9610dd3bcef3`, source `efba7fb`,
contract revision 10 (API 1.6.0). Run `9bb66c4e-e429-450b-8671-521d33926ede`
measured the same two product surfaces on `746651e` before this branch
contained devstack follow-up 12. Both runs agree.

| Surface | What revision 10 / PR #204 now returns | Acceptance evidence recorded |
|---|---|---|
| `Enrollment.token_sha256` on `createEnrollment`, `getEnrollment`, `revokeEnrollment` | Required read-only `^[0-9a-f]{64}$`, SHA-256 of the exact plaintext token bytes. | Harness digest equal on create, pending get, consumed get, revoke response and revoked get. Plaintext absent from those admin bodies. |
| `listAuditEvents` `enrollment.created` `changes.after.token_sha256` | The created event lists the digest. `enrollment.consumed` and `enrollment.revoked` do not carry `token_sha256` or plaintext. Property names ending in `_sha256` stay visible; `token` stays redacted. | `audit_token_hash` passed with `digest_only_on_created` true for the replay, race, revoked and reinstall enrollments. |
| `listNodeIncarnations`, `GET /api/v1/admin/nodes/{node_id}/incarnations` | Newest first (`created_at DESC, id DESC`). Fields used: `current`, `state`, `retire_reason`, `usage.upload_bytes`, `usage.download_bytes`, `usage.through`. Default limit 50, max 100. 404 only when the Node row is absent. | Fresh Nodes, including both AT07 slots and the race winner, have count 1 and the id matches `getNode.current_incarnation_id` and `getEnrollment.node_incarnation_id`. After revoke (202) and a second bound enroll, count is 2, previous reason moves from `revocation` to `reinstall`, usage stays `"0"`/`"0"` with `through` null, `limit=100` returns 2, and a `limit=1` cursor round-trip has no duplicates. A pending enrollment's Node returns `items: []`. An unknown id returns 404. An active incarnation rejects another bound create with 409 `NODE_INCARNATION_ACTIVE`. |
| `JoinEnrolledIdentity`, `EnrollmentPeerInventory` (devstack #204) | Own-key enroll inside a waiting slot, then NetBird setup-key/group/peer readback correlated to the `enrollment.consumed` operation id. | Second fresh stack. Race 201 + 409 `ENROLLMENT_KEY_MISMATCH`. Replay enroll 200, bootstrap 200, `joined` true. Inventory `complete`, peers 1, setup keys 1, uses 1, correlation groups 0. Second read unchanged. |
| `MachineIdentityFingerprints`, `StartClonedIdentityFixture` (devstack #204) | Hashes only: OS machine-id, agent SPKI, NetBird X25519 public key. Clone copies durable identity before `node-clone` boots and `Observe` reads conflict/freeze rows. | `node` and `node2` hashes differ on all three fields and share image `sha256:5d950b704594632e67993c167b8b4d889850dd0bd13800bb81639b445c983e27`. Clone and source hashes match, both stages `enrolled`. `endpoint_rows` 0, `conflicts` 0, `mapping_frozen` false. |

ENR-02 and ENR-03 are closed by the rows above. ENR-04 and the fingerprint
half of ENR-07 are closed. ENR-08 remains: the clone export reads the conflict
and freeze columns, and those columns stayed empty. ENR-05 and ENR-06 remain
because `HoldUsageAcknowledgements`, `UsageLedgerSnapshot`, `CountedMemberTraffic`,
`RerunInstaller` and `ControlledReinstall` are not in `docs/e2e/RUN.md` on this
source. `ForwardingBlockReason.awaiting_member_grants` (D-067) is present in
revision 10 and is outside this harness.

AT07 live nodes each have one incarnation and a usage checkpoint with accepted
bytes `"0"` and `through` set (`2026-10-30T20:51:11.681114Z` and
`2026-10-30T20:51:24.065428Z`). That is the documented checkpoint-with-no-samples
shape. Counted historical usage across an installer rerun is still unmeasured.

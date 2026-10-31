## Follow-up 16 decisions register (2026-10-31 EDT)

Verified against main `982c209` through #303. Follow-up 15 (#292) reconciled
#273–#290, including revision 14; its exact register and r16 dispositions are
preserved in the handover testdata history/follow-up-15 snapshot. Follow-up 14
stopped at #272. Historical acceptance tables retain their original cutoff.
The [current acceptance snapshot](../ops/README.md#follow-up-16-acceptance-snapshot-2026-10-31-edt)
and `TestHandover_Followup16Claims` pin the new merged boundary. Handover ran no
new live acceptance. D-114 and D-119 below record the orchestrator's decisions;
they do not assert a newly merged updater or scheduled plan-change feature.

| Decision | Operator meaning and delivery boundary |
|---|---|
| D-098 | Claims read byte-identical owner archives or run-ID-named snapshots, never current lane evidence; members follow-up 7, enrollment follow-up 6 and secops follow-up 11 now have final archives. Live document digests remain separate. |
| D-099 | Do not close SECOPS-21 from the compiler tests alone. Merged #277; cache reuse is bounded by the ordinary proposal and restore-observation fence. SECOPS-21 remains open after the #301 live run: first delivery, local acceptance and the matching applied ACK still need capture. |
| D-100 | Merged #276: changed topology expectations refresh only through a clean verified Apply of a newer published generation; no out-of-band override. |
| D-101 | Merged #275: fixtures obtain real central serving leases and report the issued ID/expiry unchanged. Portal #293 displays the publication-gate cause; polling timeout leaves the operation running. |
| D-102 | Only the owning lane writes its requests; devstack answers in `docs/e2e/RESPONSES.md`. Fixture delivery is not consuming-case acceptance. |
| D-103 | Security-ops #301 ran two independent live runs under its existing `fvpn-w4-secops-*` namespace; full AT29/AT31 remain blocked. |
| D-104 | Netlab #290 ships the 1 ms deadline quantum and 100 ms guarded-install bound. Agent-core follow-ups 24/25 consumption and isolation follow-up 8 acceptance remain pending; AT13 stays failed. |
| D-105 | Central acceptance merged #296: current mTLS presenter, retired origin on the same node, presenter enrolled strictly after retirement, immutable event identity/hash, one origin per batch, ACK per original stream; the retired certificate is refused at mTLS. Node retention/upload merged #299 but is inactive until D-118; live ENR-05 and usage AT follow-up 5 remain pending. |
| D-106 | Revision 14 defines optional nullable `ObservationReport.tc_evidence`; netlab #295 supplies `ServingEvidence.TCReadbackEvidence()` with actual read completion, candidate generation, full verifier result, bounded detached rows and contract-matching digest. Agent-core follow-up 25 wiring remains pending; absent evidence is not a clean read. |
| D-107 | Merged #287, displayed by #293: `Operation.steps[].message` supplies the gate cause next to `error_code`; D-074 still holds publication running. |
| D-108 | D-096 closes only the measured native IPv4 NetBird-owned path; installer transit, IPv6 and uninterrupted containment are not promoted by this mechanism proof. |
| D-109 | Pin tokens only in handover-owned documents, D-098 archives, roadmap rows / PR numbers / contract versions, or Go test names on main; another lane's living prose is not a pin target. |
| D-110 | Merged #300: `internal/installerjob` derives the deterministic UUIDv5 job ID from the claim namespace, manifest digest and authenticated incarnation; central recomputes it and rejects a new mismatch with `409 STATE_CONFLICT`, `job_id_mismatch`, bounding accepted rows to one per (incarnation, release). Accepted legacy receipts retain replay/conflict semantics; `nodes.resource_version` changes only for visible pointer/summary changes. Main's updater still mints the legacy whole-record ID until updater follow-up 15 merges. |
| D-111 | Merged #300: a null central baseline can be seeded from an eligible installer-reported release as `current_release_id = last_good_release_id = that release`; failure evidence remains stored without promoting the candidate or inventing health. Rollback restores both pointers, generation fences remain, and migration 00191 has a Down/Up round-trip proof. |
| D-112 | Merged #296: the retirement window uses central `sampling_interval_bound_seconds`, default 10 s, with node sampling ≤ that bound; #299 pins the production meter's 10 s default, while activation and live hand-off remain pending under D-118. |
| D-113 | The one-file test-fixture ownership extension let fleet-update bind the nodes keyed-replay fixture to D-110 so the merge kept main green; #300 also records the second r16 replay fixture corrected after integration with #291, with production replay unchanged. |
| D-114 | Pending updater follow-up 15: legacy unaccepted outbox receipts must be re-derived once or retired on 409, while accepted legacy receipts keep their stored replay semantics. |
| D-115 | Merged #296: middleware delegates body identity checks to the usage service only for `submitUsageBatch` with a schema-valid origin; mTLS, schema/body-size validation, idempotency, no-origin recursive checks and other operations remain enforced. |
| D-116 | Merged #302: keep enforcement strict; the installer marker lives in hookless `inet fvpn_install`, rerun/upgrade migrates the exact legacy empty marker, and `inet fvpn` retains exactly four rendered chains. Netlab follow-up 17 scope tests and devstack follow-up 18 serving-fixture rerun remain pending. |
| D-117 | Merged #297: there are no open-ended grants; `valid_until` is NOT NULL with `isfinite` since migration 00001, and far-future finite grants use the real withdrawal route. |
| D-118 | #299 is merged inactive: agent-core follow-up 25 owns the activation hook in `node/agent/usage.go`; retained-stream package tests are not a live reinstall hand-off. |
| D-119 | No scheduled plan-change contract exists: a finite grant's deadline means expiry; a plan change at a boundary is a billing-issued replacement grant. |

Current contract is revision 14: OpenAPI `1.10.0`, `50` operations (#289).
Wire-members follow-up 11 (#297) registers `revokeMemberGrant` at
`POST /api/v1/admin/members/{member_id}/grants/{grant_id}/revoke` with admin
session, CSRF, required `Idempotency-Key` and reason; `202` returns the durable
`member.grant_revoke` operation. Worker `member.grant_revoked` joins #291's
fence-reconciliation batch. Compilation or a 202 receipt is not observed cutoff.
The revision 14 prose about open-ended grants and historical sampling intervals
is superseded by D-117 and D-112; the contract-owner wording update is pending.
See the [usage acceptance note](../usage/2026-10-31-d105-retired-stream-handoff.md),
[installer result note](../updates/2026-10-31-installer-results-d110-d111.md),
and [portal's 15-entry D-093 audit](../portal/DEV.md): D093-06 / D093-10 are
retired, D093-12 stays open because the harness still writes
`release_verifications.state=verified` by SQL; the fixture-transport decision is pending.

## Pending at the follow-up 16 cutoff

| Work | Remaining acceptance boundary |
|---|---|
| Agent-core follow-up 24 | r16 M-1 failed freeze persistence, ISO-F6-01 recovery latch, meter checkpoint and D-104 consumer; pending, no merge claimed. |
| Agent-core follow-up 25; isolation follow-up 8 | Recovery and D-106 observation wiring, D-118 hand-off activation, members F8-01 same-generation restart; AT13 stays failed until assembled re-assertion. |
| Updater follow-up 15 | D-110 producer switch and D-114 legacy outbox handling; full journaled installer upgrade remains red with the missing checkpoint. |
| Netlab follow-up 17; devstack follow-up 18 | D-116 verifier-scope tests and installer serving rerun; #302 baseline verification alone does not prove serving. |
| Devstack follow-up 18; wire-nodes follow-up 32 | SECOPS-21 first-delivered envelope, matching local acceptance and applied ACK capture. |
| Usage AT follow-up 5; enrollment rerun | Live ENR-05 original-stream bytes/ACK assertions after activation; the five measured AT07 blockers are not retroactively passed. |
| Devstack follow-up 18; netlab follow-up 18 | Members F8-02 incomplete dual-stack MTU-1280 transfer; AT09 remains failed. |

[r16 report](../reviews/r16-rev13-d089-d099-review.md) stays byte-identical;
[current dispositions](../reviews/README.md#r16-dispositions-at-the-follow-up-16-cutoff)
close H-1 with #291 and M-2/L-1 with #300, while M-1 stays pending.

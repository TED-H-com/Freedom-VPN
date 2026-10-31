## Follow-up 20 decisions register (2026-10-31 EDT)

Verified against main `96ec0630` through #344. The [operator evidence](follow-up-20.md)
and [acceptance snapshot](../ops/README.md#follow-up-20-acceptance-snapshot-2026-10-31-edt)
pin all seven deliveries #338–#344. The [follow-up 19 register](../../tests/handover/testdata/history/follow-up-20/register19.md)
retains its #337 cutoff, as [follow-up 18](follow-up-18.md) retains its own.
`TestHandover_Followup20Claims` checks current dispositions separately.
D-121, D-122, D-124 and D-133 remain process / test-acceptance decisions.

| Decision | Operator meaning and delivery boundary |
|---|---|
| D-098 | Claims read byte-identical owner archives or run-ID-named snapshots, never current lane evidence; members follow-up 7, enrollment follow-up 6 and secops follow-up 11 now have final archives. Live document digests remain separate. |
| D-099 | Do not close SECOPS-21 from the compiler tests alone. Merged #277; SECOPS-21 closed by security-ops 13 (#316) using the first delivered restore offer 19 > retained 15, matching local digest and applied 202 receipt. Harness-proved on the assembled stack, not a live-fleet guarantee. |
| D-100 | Merged #276: changed topology expectations refresh only through a clean verified Apply of a newer published generation; no out-of-band override. |
| D-101 | Merged #275: fixtures obtain real central serving leases and report the issued ID/expiry unchanged. Portal #293 displays the publication-gate cause; polling timeout leaves the operation running. |
| D-102 | Only the owning lane writes its requests; devstack answers in `docs/e2e/RESPONSES.md`. Fixture delivery is not consuming-case acceptance. |
| D-103 | Security-ops #301 ran two independent live runs under its existing `fvpn-w4-secops-*` namespace; full AT29/AT31 remain blocked. |
| D-104 | Netlab #290 ships the 1 ms quantum and 100 ms installation bound; agent-core #312 consumes absolute deadlines. D-120 leaves the installation bound unchanged. Isolation #334 fails AT13 on ISO-F6-03 early expiry; exact timing is now delivered. Three fresh renewal losses remain counted; the 17 historical losses cannot be assigned new timestamps. |
| D-105 | Central acceptance #296 and active #312 hand-off retain immutable retired-origin identity/hash and independent ACKs; #320 fixes lifetime ownership. Usage #324 passes the controlled-installer hand-off. Enrollment #333 passes all five composed AT07 rows with both D-130 halves; retired 2,724 bytes / ACK 3 stay separate from current 908 bytes / ACK 2. |
| D-106 | Revision 14 defines optional nullable `ObservationReport.tc_evidence`; #295 supplies `ServingEvidence.TCReadbackEvidence()`. #305 transports bounded revision 13 details; #312 retains the authoritative Apply/live/blocked snapshot with actual read time, candidate generation, full result and original digest. Null before any retained read; historical rejection rows survive recovery. |
| D-107 | Merged #287, displayed by #293: `Operation.steps[].message` supplies the gate cause next to `error_code`; D-074 still holds publication running. |
| D-108 | D-096 closes only the measured native IPv4 NetBird-owned path; installer transit, IPv6 and uninterrupted containment are not promoted by this mechanism proof. |
| D-109 | Pin tokens only in handover-owned documents, D-098 archives, roadmap rows / PR numbers / contract versions, or Go test names on main; another lane's living prose is not a pin target. |
| D-110 | Merged #300: deterministic catalog-bound job UUID, adopted by updater #307; mismatched new IDs return 409 STATE_CONFLICT / job_id_mismatch. D-127 (#322) now bounds one node/job to the first receipt plus one eligible correction, preserving both stored ACKs. Legacy replay and visible-change-only resource versions remain. |
| D-111 | Merged #300: a null central baseline can be seeded from an eligible installer-reported release as `current_release_id = last_good_release_id = that release`; failure evidence remains stored without promoting the candidate or inventing health. Rollback restores both pointers, generation fences remain, and migration 00191 has a Down/Up round-trip proof. |
| D-112 | Merged #296: central sampling_interval_bound_seconds defaults to 10 s, with node sampling ≤ that bound; activation merged #312 under D-118, locking fixed #320, live usage hand-off passes #324. Raise the central bound before raising the node interval. |
| D-113 | The one-file test-fixture ownership extension let fleet-update bind the nodes keyed-replay fixture to D-110 so the merge kept main green; #300 also records the second r16 replay fixture corrected after integration with #291, with production replay unchanged. |
| D-114 | Merged #307: legacy unaccepted outbox receipts get one job_id_mismatch 409 and then a replacement; the mismatch is terminal and not resent. Accepted legacy receipts keep stored replay semantics. |
| D-115 | Merged #296: middleware delegates body identity checks to the usage service only for `submitUsageBatch` with a schema-valid origin; mTLS, schema/body-size validation, idempotency, no-origin recursive checks and other operations remain enforced. |
| D-116 | Merged #302: keep enforcement strict; marker in hookless `inet fvpn_install`, exact legacy empty marker migrated, four rendered chains in `inet fvpn`. #304 proves verifier scope and serving with the marker before enrollment; #313 proves the real installer rerun and counted serving. |
| D-117 | Merged #297: there are no open-ended grants; `valid_until` is NOT NULL with `isfinite` since migration 00001, and far-future finite grants use the real withdrawal route. |
| D-118 | Active in agent-core follow-up 25 (#312): activation hook in node/agent/usage.go sets RetirePreviousAt. #320 fixes adoption under the node lock and lifetime usage.db.lock; replacement waits for predecessor close. #324 passes live retired-stream hand-off; #332 retains 8/8 origin events and #333 closes the five enrollment rows. |
| D-119 | No scheduled plan-change contract exists: a finite grant's deadline means expiry; a plan change at a boundary is a billing-issued replacement grant. |
| D-120 | AT10 test-acceptance definition, not a product guarantee: loss must fall inside/boundary of exactly one verified outer guard bracket; both bracket upper bound and render → installed ≤ 100 ms; successful successor, zero TCP timeouts, ≤ 5 losses per 100 reloads, complete checked captures. #304 remains failed under the old rule; #308 passes under D-120. D-104's 100 ms installation bound unchanged. |
| D-121 | Orchestration process only: one-follow-up ownership extension for updater 15 to align packaging/installer/handoff_test.go TestD089_RealUpdaterConsumesWriterRecord with D-110 failed-replay semantics, unchanged seal and cardinality (#307). Test-only; no operator guarantee. |
| D-122 | Orchestration process only: node/enforce/review_r16_quarantine_test.go accepted as a test-only ownership exception for agent-core 24 (#305). No enforcement production change. No operator guarantee. |
| D-123 | v1.0 overlay is IPv6Block by design: no IPv6 forwarding through the node, no end-to-end IPv6 MTU / PMTU acceptance row. IPv6 overlay support is separately scheduled work, never deletion of the guard rules. Native IPv6 underlay failures remain applicable. |
| D-124 | Orchestration process only: agent-core 25's resume edited internal/nodes/publication_evidence_test.go fixture-only under a one-resume ownership extension (#312); the real Decide now needs a valid desired expiry. Usage #324 also refreshes its deadline fixture. Devstack #332 found no Decide call in its brief's named file on that base; a process fact only. No operator guarantee. |
| D-125 | Product mechanism (#320 / #323): every checkpoint names the updater latch identity it finalized against. MeterCheckpoint.BarrierID (barrier_id, omitempty) echoes the exact UUID spelling; each new identity repeats verified withdrawal and durable finalization, same-identity restart pins bytes, resume removes evidence. Installer #326 passes 13/13; devstack #332 serving rerun passes 14/0/0; updates #337 extension passes 23/23. Regular AT23–AT27 re-measurement remains pending with updates 11; #343 closes the shared fixture side. |
| D-126 | Observation reasons are cumulative. Per-check code precedence is safety failure > provisional / awaiting > ok; each stage error carries its own cause. #321 proves central value preservation and stored-ACK replay with 202 Accepted. #325 confirms deliberate single composition, no reporter race; #328 closes F9-01 by accepting the cumulative report. A safety failure never authorizes serving. |
| D-127 | #322 permits one eligible installer correction on the same incarnation: succeeded after an applied D-111 pre-health seed, or rolled_back after succeeded. Current/last-good move under the unchanged generation fence; first receipt and both ACKs stay immutable. Every other distinct outcome, including a third, is ACK_CONFLICT. Deploy 00192 before control; outcome_sequence is 1..2 and Down refuses once corrections exist. #337 captures real success/replays and a TEST ONLY rollback correction; regular physical AT26 rollback remains pending with updates 11. |
| D-128 | D-126 stored unchanged means complete JSON-value equality: byte-identical JCS output and same SHA-256. report_payload is JSONB; literal whitespace / key order is not retained, and no schema change is planned. |
| D-129 | AT09 same-generation restart qualifies on desired == applied across the restart under D-126 at whatever generation the unchanged sequence yields. 30 is not a contract value. #328's 29 → 29 row qualifies under this test-acceptance definition; full AT09 remains blocked. No product change. |
| D-130 | #330 mints UTC whole-microsecond intervals ≥ 1 µs via eventInterval at all four former 1 ns sites. #331 validates strict nanosecond canonical interval_order and stores a microsecond projection. Migration 00193 changes all three stored checks to <=, never bypasses them; Down refuses equal-endpoint rows. Unexpected storage constraint failures remain retryable 503, never definitive event rejection. Already sealed valid 1 ns events recover on the next retry after deployment without operator data edits. #333 closes ENR-07; #338 closes newly minted-event measurement: planBaseline sequence 1, zero records, 1 µs, ACK 1 before traffic; counted event 2 is 1,066 + 1,513 = 2,579 bytes. |
| D-131 | #335's TEST ONLY WithTestOnlyGrantEligibilityReaderFactory on httpwire.Open substitutes the lease-issuance reader once per renewal transaction. Default leases.DBEntitlementReader; a nil reader fails closed. No production config / env / HTTP surface. Desired-state compilation retains its production reader, keeping bindings/generation unchanged. #343 composes the TEST ONLY FVPN_E2E_GRANT_SEAM=1 control and exports the seam (7/0/0 twice); central-seam blocker closed, assembled AT10 pending isolation 10. |
| D-132 | Product tests pin AT evidence only at immutable docs/e2e/at/<lane>/history/<follow-up>/ paths, never living roots. Two internal/fleetupdate/d062_integration_test.go reads broke the chain; #337's dated one-file exception pins the follow-up 9 archive. Fleet-update 13 (#340) closes the lane follow-up with byte-identical owned testdata, explicit empty-result failure and a package-wide mutable-reference guard. Evidence hygiene does not promote acceptance. |
| D-133 | A failed live timing gate is attributed before a delivery is accepted. The lane runs back-to-back A/B on the same host: unchanged main vs the branch, in order A, B, A, B, on a CPU-quiet host defined by runnable queue and idle, never by load average. Retain full p50 / p90 / p99 / max over completed guards, list failed operations separately, and record host state per leg. If the branch is worse, the lane fixes its own regression; “outside my ownership” is accepted only with that A/B and routes to the owner with both distributions. If both pass under equal quiet conditions, attribute the failure to host contention and proceed with the distributions retained. A product change never merges on an ownership claim alone. Decided 12:4x EDT, applied 14:0x EDT on #344; process / test-acceptance decision, no product value changed. See the [retained application and measured limits](follow-up-20.md#agent-core-28--344-historical-writer-still-unattributed). |

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
`release_verifications.state=verified` by SQL; wire-nodes #321 recommends option (ii), an explicit `nodes.release_fetcher = fixture`
selector compiled only into a dev-tag worker. Design advice, not implemented;
the SQL shortcut stays labelled.

## Findings register at the follow-up 20 cutoff

| Finding | Status at #344 | Evidence / next owner |
|---|---|---|
| ENR-07 | CLOSED | D-130 #330/#331 plus enrollment #333: AT06 662/0/0, AT07 48/0/0, fresh gap ACK 1 then sequence 2 ACK 2, 908 bytes; planBaseline attribution corrected. |
| Members F9-01 | CLOSED | #325 / #328 under D-126 / D-128; deliberate cumulative report accepted. Full AT09 stays blocked. |
| Members F10-04 | CLOSED under D-129 | #328 desired/applied 29 → 29 qualifies; 30 is not a contract value. Members 11 complete AT09 row pending. |
| Devstack 18 serving-node failure / candidate seed HASH_MISMATCH | CLOSED by combined-source rerun | #326 privileged 13/13; #332 serving 14/0/0 with candidate seed exit 0 and no tree repair. Historical extractor diagnosis retained; no open pre-#307 binary caveat. |
| Members F10-01 pin path | CLOSED | Netlab #336 moves the AT09 link/pin and three isolation pins to byte-identical archives, digests unchanged. |
| ISO-F6-03 | OPEN — real product defect | Historical writer still unattributed. #344 places all 955–961 operations inside process run 0b191fdd; Boot was not the initial writer and watchdog decision evidence was never retained. Isolation 10 re-measures on the new baseline/heartbeat and watchdog evidence contract; restart adoption does not close #334. |
| D-087 central seam | CLOSED | #343 TEST ONLY composed seam, two fixed-generation runs 7/0/0, 3,980 / 4,112 ms. |
| D-087 assembled AT10 | OPEN | Isolation 10 pending at #344; the central prerequisite pass does not promote assembled AT10. |
| Updates F10-01 shared side | CLOSED | #343 regular release-root artifacts + state/meter.checkpoint, switching 11/0/0. Updates 11 regular AT23–AT27 measurement and consumer assertions pending. |
| Updates F10-02 | CLOSED | #341 validated generated reporting origin, server CA, mTLS identity and exact-origin allowlist; privileged 13/13. Distinct artifact-origin question remains an owner decision for updates 11, not a defect. |
| Members F10-02 | OPEN | #343 247 every-read-success counter samples with same-boot brackets; historical unreadable sample not reproduced. Members 11 re-measurement pending. |
| Members F10-03 | OPEN | #343 36 complete per-query traces; historical responder unrecoverable. Members 11 / dns-publish 9 re-measurement pending. |
| F8-02 / native MTU 1280 | OPEN — unattributed | #343 instrumented failure 10,010 / 64,064 bytes, broader 15/0/1; no global MTU fix. Netlab 21; D-123 retains IPv6Block and native underlay failures. |
| F18-01 | OPEN — classification confirmation | Devstack 21 confirms stopped-producer 1 / 1 / 2 as the intended D-125 negative; historical 13/0/1 remains failed. |
| D-133 measured limits | OPEN — not defects of record | Two pre-D expiry drops (first ingress about D−63.3 ms; unchanged D-104 passes) and historical outer ceiling 52.596 ms (B1 57.92 ms). Netlab 21 enforcement-owner assessment with both distributions; no tolerance requested. |
| Dependabot #22 | OPEN — dependency hygiene | Medium minimatch < 11.0.1, GHSA-g3hv-75wg-m249, apps/portal/package-lock.json. Portal owns reachability assessment; not a product defect until reported. [Frozen alert](../../tests/handover/testdata/history/follow-up-20/dependabot-22.json). |

AT11 real platform routes, AT13 full continuous fault matrix including physical
arm64, AT29 / AT31 resources, OS reboot, power-loss and real platform clients
remain open. The agent-core README restart request is closed by #344; agent-core 28 is on main. Updates 11, isolation 10, members 11 and dns-publish 9 have no results at this cutoff. Exact producer timing is delivered; the three new renewal losses
remain counted without D-120, and the 17 historical losses remain unclassified.
See [the full current evidence](follow-up-20.md). No earlier AT23–AT27 pass is
carried forward.

[r17](../reviews/r17-rev14-d104-d117-review.md) and
[r16](../reviews/r16-rev13-d089-d099-review.md) remain byte-identical archives.
[r17 M-1](../reviews/README.md#r17-disposition-at-the-follow-up-18-cutoff) is
fixed by #322 under D-127; #337 supplies the scoped live extension and labeled
HTTP correction, while regular physical rollback remains pending with updates 11.

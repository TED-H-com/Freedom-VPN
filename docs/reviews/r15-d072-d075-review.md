# Adversarial review archive — `r15-d072-d075-review`

- **Branch:** `r15-d072-d075-review`
- **Commit:** `dfafae0f985a5209ccf380afb9c615987f2bd775`
- **Date:** `2026-10-30T22:16:48-04:00`
- **Reviewing model:** `grok/grok-4.7` (verified from the commit's `Agent:` trailer)
- **Source report:** `.agent/REVIEW.md`, blob `013967acfe49d8829830254a115407e0c60382cf`, SHA-256 `3929d4ca5ec00703e1d2dd9ae6306f9acf5aa3b445f288be31d4ee50acffbbbb`.
- **Status snapshot:** `origin/main` at `c6e3eca`, 2026-10-31 EDT. Findings below describe the reviewed revision; closing PRs describe the subsequent owner fixes.

## Scope

The review covers the r14 fixes now on `main` and the decisions taken with them. D-072 cloned-identity detection is wire-nodes follow-up 21 (#211) and follow-up 22 (#217). D-073, the durable roaming candidate, and D-075, the grace column, are follow-up 24 (#228). D-074, a gated publish that waits, is follow-up 23 (#224). r14 H-1 and M-2 landed in #217, M-1 in agent follow-up 18 (#216), M-3 in netlab follow-up 10 (#220), M-4 / L-1 / L-2 in control follow-up 20 (#215). Fair-Use recompile and the ACK reader are wire-policies-usage follow-up 4 (#222).

## Findings and disposition

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | **H-1** (wire-nodes): The renewal that freezes an incarnation still mints a 24 h member-grant lease. The next observation marks the node failed and does not shorten that lease. | closed | wire-nodes follow-up 26 (#254), follow-up 27 (#259, migration 00190) |
| Medium | **M-1** (wire-nodes): A lease renewal and an observation of the same node take `nodes` and `fleet_config` in opposite orders and deadlock (`40P01`). | closed | wire-nodes follow-up 26 (#254) |
| Medium | **M-2** (wire-nodes): After the member's only grant is revoked, `otherNodePendingIso` still matches that grant's plan revision, so a quiet-enough peer in the old group holds `member_device.revoke`. | closed | wire-nodes follow-up 26 (#254) |
| Medium | **M-3** (wire-nodes): `safety_status=failed` with the generation already applied is gated as `readiness_unconfirmed`. The publish stays running until `worker_attempts`. | closed | wire-nodes follow-up 26 (#254) |
| Medium | **M-4** (wire-nodes): A frozen exit's latest observation row stays `safety_status=passed`, so `fair_use.applying` clears while the node row is `failed`. | closed | wire-nodes follow-up 26 (#254); reader side #263 |
| Medium | **M-5** (wire-policies-usage): Unpublishing the only carrying exit makes the ACK reader fall through to `checks.applied_policy_refs`. Production does not store that there, so `applying` stays true. | closed | wire-policies-usage follow-up 6 (#263) |
| Medium | **M-6** (netlab): A `tc clsact` mirred redirect on the member interface forwards a non-admitted packet while `fvpn-netlab verify` still returns 0. | closed | netlab follow-up 12 (#256), D-090 follow-up 13 (#262); agent consumer #272 |
| Low | **L-1** (control): `anonymousGroupReserve = 1` lets one IPv6 `/48` hold every anonymous slot but one. | closed | control follow-up 22 (#252); browser calibration #233 |

**Owner fixes adopted — 2026-10-31 EDT:** all eight findings have owner fixes and committed proofs on main `c6e3eca`. H-1's stored lease clamp is complete in #259; a server clamp alone does not establish a 60-second packet-stop guarantee. M-6's library quarantine is proved in #262; agent-core follow-up 22 (#272) now consumes quarantine, with the between-step privileged-daemon and uplink-filter limitations in the operator guide. r16 delivered on base `5575470`; its [byte-identical report](r16-rev13-d089-d099-review.md) and [current dispositions](README.md#r16-dispositions-at-the-follow-up-16-cutoff) record H-1 fixed by #291, M-2/L-1 fixed by #300 and M-1 pending, alongside #277’s HTTP cache/replay fixes. #245’s generation floor, #270’s D-095 hold and #277’s D-099 HTTP fix do not close live SECOPS-21; #286’s ACKs are simulated.

## Adopted proof files on main

| Finding | Adopted proof files and PR |
|---|---|
| H-1 | `internal/nodes/httpwire/review_r15_identity_integration_test.go` (#254); `internal/nodes/httpwire/review_r15_clamp_integration_test.go` (#259), including stored expiry, signed/replayed byte identity and forbidden mutations. |
| M-1 | `internal/nodes/httpwire/review_r15_q2_lock_integration_test.go` (#254): real keyed and unkeyed HTTP lease paths, fleet-before-node locks. |
| M-2 | `internal/nodes/review_r15_q3_fence_integration_test.go` (#254): revoked plan revisions and freshness boundary. |
| M-3 | `internal/nodes/httpwire/review_r15_q4_publish_integration_test.go` (#254): current-generation failed/expired safety fails on attempt 1. |
| M-4 | `internal/nodes/review_r15_observation_integration_test.go` (#254); `apps/worker/review_r15_q9_fup_integration_test.go` (#263) checks the current frozen exit remains pending. |
| M-5 | `apps/worker/review_r15_q9_fup_integration_test.go`, `internal/fup/review_r15_q9_ratchet_test.go` (#263): empty carrying set clears applying; generation ratchet retained. |
| M-6 | `node/enforce/review_r15_q8_verify_test.go`, `tests/netlab/review_r15_q8_bypass_test.go` (#256); `node/enforce/quarantine_test.go` and the extended bypass proof (#262), with `docs/poc/G2-D090-evidence.txt`. |
| L-1 | `internal/httpapi/review_r15_q1_xff_test.go`, `internal/httpapi/review_r15_q6_ingress_test.go`, `internal/httpapi/middleware_browser_rate_integration_test.go` (#252); `internal/httpapi/middleware_browser_rate_test.go` (#233/#252) for browser calibration. |

Additional adopted #254 invariants: `internal/nodes/review_r15_q2_grace_integration_test.go` and `internal/pki/review_r15_csr_test.go`. Provenance was checked with `git log -1 --format=%B` and each commit's changed test files: #254 `ebb62f7d`, #259 `847e6c51`, #263 `e0fbe7c2`, #256 `3916d00`, #262 `e0c4a6a9`, #252 `b2012686`. The following original review names describe the pre-fix reproductions; adopted assertions now require the repaired behavior.

## Original review-branch proof identities

- **Proof:** `TestReviewR15_FrozenIncarnationStillReceivesAGrantLease` in `r15-d072-d075-review:internal/nodes/httpwire/review_r15_identity_integration_test.go`.

- **Proof:** `TestReviewR15_LeaseAndObservationLockOrder` in `r15-d072-d075-review:internal/nodes/httpwire/review_r15_q2_lock_integration_test.go`. The proof is two raw connections issuing those statements, with `deadlock_timeout = 200ms`, after the lease session is observed in `pg_stat_activity.wait_event_type = Lock`.

- **Proof:** `TestReviewR15_RevokedPlanRevisionStillHoldsTheFence` in `r15-d072-d075-review:internal/nodes/review_r15_q3_fence_integration_test.go`.

- **Proof:** `TestReviewR15_SafetyFailedPublishWaitsAsReadinessUnconfirmed` in `r15-d072-d075-review:internal/nodes/httpwire/review_r15_q4_publish_integration_test.go`. The dispatcher was given `attempts=2` so the proof does not sleep through the production backoff. The cause and the final `error` transition are what the test asserts.

- **Proof:** `TestReviewR15_FrozenCarryingExitKeepsFairUseApplying` in `r15-d072-d075-review:apps/worker/review_r15_q9_fup_integration_test.go`.

- **Proof:** `TestReviewR15_UnpublishDropsTheCarryingExitAndApplyingSticks` in `r15-d072-d075-review:apps/worker/review_r15_q9_fup_integration_test.go`.

- **Proof:** `TestReviewR15_TcMirredAndPolicyRoutingBypass/tc_clsact_mirred` in `r15-d072-d075-review:tests/netlab/review_r15_q8_bypass_test.go`.

- **Proof:** `TestReviewR15_AnonymousReserveLeavesOneSlot` in `r15-d072-d075-review:internal/httpapi/review_r15_q6_ingress_test.go`.

## What held and limits

The report's positive checks covered authenticated endpoint ownership, rightmost-untrusted proxy selection, durable roaming and bounded endpoint rows, the stale-generation grace column, expiry of the 90 s revoke fence, provisional `AWAITING_MEMBER_GRANTS` waiting, node-URN ingress, and once-only deletion of reassigned addresses. Fair-Use duplicate events did not cause repeated generation bumps. The lab policy-routing subtest was contained; the tc redirect subtest bypassed admission (M-6). Same-prefix clones remain indistinguishable, and legitimate dual-uplink interleaving can trigger the clone detector. The reviewer did not claim a hardware/platform run or a second-process restart proof for the roaming candidate.

The high finding is specifically the detecting renewal: freezing the incarnation still allowed a member-bearing lease at the fleet's 24-hour cap. A later failed observation did not shorten that already minted lease. Do not treat a failed central status alone as immediate data-plane cutoff; follow the revoke/re-enroll recovery workflow and verify withdrawal.

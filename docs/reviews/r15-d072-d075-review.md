# Adversarial review archive — `r15-d072-d075-review`

- **Branch:** `r15-d072-d075-review`
- **Commit:** `dfafae0f985a5209ccf380afb9c615987f2bd775`
- **Date:** `2026-10-30T22:16:48-04:00`
- **Reviewing model:** `grok/grok-4.7` (verified from the commit's `Agent:` trailer)
- **Source report:** `.agent/REVIEW.md`, blob `013967acfe49d8829830254a115407e0c60382cf`, SHA-256 `3929d4ca5ec00703e1d2dd9ae6306f9acf5aa3b445f288be31d4ee50acffbbbb`.
- **Status snapshot:** `origin/main` at `d9e28f8`, 2026-10-30 EDT. Findings below describe the reviewed revision; closing PRs describe the subsequent owner fixes.

## Scope

The review covers the r14 fixes now on `main` and the decisions taken with them. D-072 cloned-identity detection is wire-nodes follow-up 21 (#211) and follow-up 22 (#217). D-073, the durable roaming candidate, and D-075, the grace column, are follow-up 24 (#228). D-074, a gated publish that waits, is follow-up 23 (#224). r14 H-1 and M-2 landed in #217, M-1 in agent follow-up 18 (#216), M-3 in netlab follow-up 10 (#220), M-4 / L-1 / L-2 in control follow-up 20 (#215). Fair-Use recompile and the ACK reader are wire-policies-usage follow-up 4 (#222).

## Findings and disposition

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | **H-1** (wire-nodes): The renewal that freezes an incarnation still mints a 24 h member-grant lease. The next observation marks the node failed and does not shorten that lease. | open | — routing in progress |
| Medium | **M-1** (wire-nodes): A lease renewal and an observation of the same node take `nodes` and `fleet_config` in opposite orders and deadlock (`40P01`). | open | — routing in progress |
| Medium | **M-2** (wire-nodes): After the member's only grant is revoked, `otherNodePendingIso` still matches that grant's plan revision, so a quiet-enough peer in the old group holds `member_device.revoke`. | open | — routing in progress |
| Medium | **M-3** (wire-nodes): `safety_status=failed` with the generation already applied is gated as `readiness_unconfirmed`. The publish stays running until `worker_attempts`. | open | — routing in progress |
| Medium | **M-4** (wire-nodes): A frozen exit's latest observation row stays `safety_status=passed`, so `fair_use.applying` clears while the node row is `failed`. | open | — routing in progress |
| Medium | **M-5** (wire-policies-usage): Unpublishing the only carrying exit makes the ACK reader fall through to `checks.applied_policy_refs`. Production does not store that there, so `applying` stays true. | open | — routing in progress |
| Medium | **M-6** (netlab): A `tc clsact` mirred redirect on the member interface forwards a non-admitted packet while `fvpn-netlab verify` still returns 0. | open | — routing in progress |
| Low | **L-1** (control): `anonymousGroupReserve = 1` lets one IPv6 `/48` hold every anonymous slot but one. | open | — routing in progress |

**Routing in progress — 2026-10-30 EDT:** none of these eight r15 fixes, or their review proof files, is adopted on `main` at `d9e28f8`. H-1 and M-1–M-4 route to wire-nodes; M-5 to wire-policies-usage; M-6 to netlab; L-1 to control. `#245` fixes SECOPS-21 generation recovery, not r15 M-5. `#247` is the D-081 prose erratum, not an r15 closure.

## Review-branch proofs (not yet adopted on main)

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

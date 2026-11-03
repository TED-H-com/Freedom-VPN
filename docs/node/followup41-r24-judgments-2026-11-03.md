# Follow-up 41: r24 judgments — 2026-11-03

D-141 judgments before runtime edits. Starting production tip `b17c8c17`;
review `ba5d3004`. Both proof files were imported byte-for-byte and committed
alone. No API adaptation. Original output is retained in the ignored
`.agent/f41/before-r24.log`.

| Finding or sub-trace | Verdict | Reachable | Introduced by (37, 38, 39, 40 or pre-existing) | Planned fix |
| --- | --- | --- | --- | --- |
| H-1 | Confirmed | Normal successful 304 polls, unchanged lease | Pre-existing; timeout-only retain in `60270cd6` (28), earlier normal path always applied | Compare the entire enforcement model and pending work under the node lock; verify live state on skipped ticks |
| H-3 device deadline collapse | Confirmed; correct the stated mechanism | Two active devices of one member with different ValidUntil, either order | Pre-existing; member aggregation in `ba1e3750` (25), binding window also predates 37 | Keep member limits separate; existing renderer caps each binding |
| H-3 retained model after withdrawal/reboot | Observation true; C1 design warning, not new production authorization defect | Retained model exists for containment/accounting audits | Pre-existing | Keep it; skip requires actual serving, current boot lease and fresh verified evidence, never just a non-nil pointer |
| H-3 Apply's value-copy clock | Observation true; C1 design warning, not this runtime defect | Apply resamples its local model | Pre-existing | Preserve absolute anchors for equality and verification; do not add C1 eligibility |
| H-2 fair-share change | Expected behavior, not an agent defect | Independent member removal changes remaining guaranteed shares | Pre-existing | Keep original passing proof unchanged; C1 owner must account for it |

Before-fix results:

```text
H-1 FAIL: applies 2 -> 5 over three unchanged ten-second ticks (want 2)
H-3 FAIL: one-hour device timeout 5m0s; sibling timeout 5m0s
H-3 retained-model subtest PASS
H-2 fair-share proof PASS
Held Q1 monotonic anchor/same-boot adoption PASS
Held Q3 stale watchdog/held lock PASS
```

H-3's review sentence that memberTimeout ignores ValidUntil is incorrect:
`node/enforce/deadline.go` caps the wall deadline by that binding before
intersecting it with DeadlineMono. The defect is the earlier agent collapse
of every binding window into one member DeadlineMono; the first binding also
supplies the two wall limits. No enforce change or new binding field is needed.

The cadence is the default ten-second poll/heartbeat, or 8,640 steady ticks
per day. Netlab 30's reported outer-guard median 28.5 ms / maximum 94.7 ms are
prior evidence, not measurements in this follow-up. After the fix, renewal
and actual model changes still Apply; the nominal renewal intervals in r24
Q3 are 23h58m for a 24h lease, 13m for 15m, and 30s for 60s, before failures,
backoff, grants and other changes. No new timing claim is made.

## Requests to orchestrator — 2026-11-03

Netlab 32 owns H-4, M-2, tc deletion ordering and C1's design revision.
Retained audit models and Apply's local render clock are not C1 eligibility
receipts. Keep those distinctions in that design. Schedule the owed quiet
D-133 A,B,A,B for 37–41 later; no reservation or timing run is part of 41.

## Supplemental H-1 verifier dependency — 2026-11-03, before fix

Code inspection found an additional pre-existing liveness dependency masked
by unconditional Apply: `verifyAllowedSet` rejects tainted elements, and
`verifyAllowedLifetimes` rejects excessive lifetimes of elements present, but
neither requires every currently authorized address to be present. A deleted
allow element is therefore not detected by the existing live verifier. This
is reachable through live drift. `TestISO01_F41_SkippedTicksStillDetectLiveDrift`
uses the existing rendered executor with real SystemPlane verification; its
deleted-element, deleted-chain and foreign-tc cases are queued behind host
admission. The intended owned-path fix is a read-only completeness check in
SystemPlane using the same nft snapshot, with the existing D-104 early-expiry
allowance. No changes to node/enforce are proposed.

## Earlier deadline-test semantics — 2026-11-03, before adaptation

`TestD104_DeadlineMinimum/binding` and
`TestD104_MemberDeadlinesAreIndependentAndBindingOrderInvariant` explicitly
expect the member DeadlineMono to equal the shortest binding deadline. Those
assertions encode H-3 and conflict with this follow-up's per-device rule.
Keep their per-device minimum, member pointer independence, both orders and
earliest watchdog wake coverage; move the binding-limit assertions to the
renderer ElementTimeouts and check member-only limits separately. Their
before-adaptation run is queued, without claiming a test result while admission
is unavailable. The original r24 files remain unchanged.

## Requests to orchestrator — 2026-11-03, comment correction

The node/enforce owner should update MemberLease.DeadlineMono's comment
(`model.go`): it currently says to include every binding cap in the shared
member deadline. The existing implementation already intersects each binding's
ValidUntil independently; the comment should describe member-level caps and
that separate binding intersection. No enforcement runtime change is needed
for H-3.

## Before-fix receipts and H-1 fixture discrepancy — 2026-11-03

After admission became free, the deleted-element case failed with
`drift not withdrawn: true <nil>` using real SystemPlane verification.
The two legacy D-104 assertions failed exactly as judged above. Initial
supplemental development runs also exposed diagnostic-only and emulated-chain
fixture panics; those runs are retained as failures, not product verdicts.

The original H-1 file remains unchanged. A complete-model comparator reduces
its three extra Applies to one, correctly: fakecontrol.signDesired re-signs
on every poll, so these responses are HTTP 200, not the claimed 304. The
setup issues two leases at one frozen manual instant; fakecontrol advances
the second issuance by 1ms, while the initial desired expiry remains at the
older instant. The first later poll extends that desired expiry and therefore
the effective member deadline by 1ms. The task expressly requires applying
policy-expiry changes, so ignoring this difference to satisfy the zero-Apply
assertion would be wrong. A setup-only correction to produce actual 304s was
asked of the orchestrator because the prompt restricts original-proof
adaptations to API changes. Independent proofs use explicit stable windows
and retain all assertions. No production workaround discards an expiry change.

## Broad-gate fixture judgments — 2026-11-03, before adaptation

`broad-agent.log` records the first broad run (agent FAIL; command/watchdog
packages PASS). Aside from the separately pending original H-1 fixture, these
failures depend on unconditional Apply and do not demonstrate a skipped live
verification in production:

- R15 Apply/foreign-tc and the F6 retained-failure fixtures use a fake live
  verifier that always succeeds; their foreign-tc fault exists only in Apply.
  Trigger their intended Apply with a small valid local host-reserve QoS
  change. Keep failure, quarantine, historical evidence and ACK assertions.
- Follow-up 31 cancellation-on-Apply and follow-up 40's r23 repair/nudge test
  inject their fault inside Apply. Give them the same explicit model change;
  also make the passing cancellation case prove its callback actually ran.
- Follow-up 35 diagnostic-counter persistence intentionally checks Apply's raw
  snapshot writes, not sampler persistence. Trigger real Apply before each
  check. Keep the no-usage/no-epoch-change assertions.
- Follow-up 28's captured producer replay swaps a verified fake plane for an
  empty emulated kernel. Its initial installation must start non-serving;
  an empty new executor cannot inherit the old fake's serving claim. Preserve
  the original lease, replay clock, producer and later repair assertions.
- R15 reassigned-address test reads `last.Revoke` after a tick, even when no
  Apply occurred: that is the previous call's consumed action. Check the
  call-count delta first; if an Apply happened its revoke list must be empty.
  Pending revocation accounting must also be empty. Add an explicit later
  model change so the original no-repeat-delete assertion still runs on Apply.

No R16–R19 or original r20–r24 assertion is weakened by these adaptations.

## Genuine 304 control and fixture follow-through — 2026-11-03

The independent `TestAT10_F41_Actual304RetainsFullModel` checks an actual
`ErrNotModified` response on every tick, preserves desired pointer and policy
expiry, and requires zero additional Applies plus a serving heartbeat.
On an unchanged archive of `b17c8c17` it fails on the first tick with
`applies 2 -> 3`; on the fixed implementation all three ticks pass.
The frozen pre-fix output is `.agent/f41/before-actual304.log`.

The explicit-Apply fault-fixture adaptations pass as a group. A subsequent
integration run has only the original H-1 fixture failure: three ticks now
Apply once for its real 1ms expiry change. Node lint passes. Original H-1's
assertions and setup have not been changed pending the orchestrator's reply;
the independent actual-304 proof avoids mistaking that fixture conflict for
a requirement to ignore a policy change.

## Implementation arithmetic review — 2026-11-03, before correction

Reviewing 41's new watchdog boundary calculation found a development defect:
converting a distant binding ValidUntil to an absolute monotonic deadline
before intersecting the member window can overflow a duration. A 300-year
binding interval saturates `time.Time.Sub`; adding the monotonic start wraps
negative even though the member limit is ordinary. This was introduced by
41's first H-3 implementation, not present on the starting tip, and is a
fail-closed availability error. The isolated proof on `0d920a28` returns
`-2562046h47m16.854775809s` instead of `1h59m59.999s` for PolicyExpires.
Cap the binding duration by the already-admissible member window before
adding its monotonic anchor. The pure calculation proof is added before the
correction; no timing measurement is involved.

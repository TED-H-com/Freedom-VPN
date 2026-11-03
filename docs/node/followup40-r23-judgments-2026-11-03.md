# Follow-up 40: r23 judgments — 2026-11-03

D-141, before product edits. Product `99f93112` plus main `61c7bf59`
merged as `533fa0e3`; unmodified proofs from `26ffc36c` committed as
`dcdc3d96`. All six finding functions fail; all six held functions pass.
Original output: `.agent/f40/before-r23.log`. No reservation or performance
measurement is part of this follow-up.

| Finding or sub-trace | Verdict | Reachable | Introduced by | Fix selected |
| --- | --- | --- | --- | --- |
| H-1a | Confirmed extra mapping gap; reject zero-gap/reset-byte expectation | Yes: reset, fetched rebinding, crash after Baseline | 39's epoch equality; older uncertain restart loss pre-existing | Accept intent across proven uninterrupted same-boot reset epochs |
| H-1b | Confirmed provenance limitation; original absent-counter trace is a guard | A verified renderer creates idle counters; ordinary Sample binds even zero counters. The exact absent-counter fixture does not represent that production state | 39's equality and adoption gate | Preserve guard; test real repair/nudge, idle zero counters and restart |
| H-1c | Confirmed transient read loses durable attribution | Yes, nft listing fails transiently before fallback Baseline | 39 gates intent loading on marker success | Load durable intent independently; contradictory live marker still rejects it |
| H-2 | Confirmed closed epoch confused with unpublished barrier | Yes, replacement/removal, checkpoint read or atomic publication failure | Dual-purpose flag pre-existing; false admission error added in 39 | Separate epoch preparation from barrier completion |
| H-3a | Confirmed stale decision after failed settlement | Yes, ErrHold, counter read or storage error | 39's pre-Apply hand-off return | Recompute on usage/pending/serving changes, including early returns |
| H-3b | Confirmed unverified serving claim | Yes, failed Baseline during pre-Apply hand-off | 39's new call site; underlying hand-off ordering pre-existing | Clear serving/expiry at withdrawal attempt, retain pending settlement and retry withdrawal |
| H-3b′ | Confirmed fail-open early return | Yes, checkpoint removal can fail after the earlier invalidation check or during standalone hand-off | Underlying ordering pre-existing; new pre-Apply exposure in 39 | Withdraw before returning on checkpoint failure; never clear the failed-persistence gate via a successful sample |
| M-1 | Confirmed missed settlement latch | Yes, first Baseline fails, second succeeds | Pre-existing; 38 introduced pending state but did not cover this return | Latch before withdrawal; keep the second fail-closed attempt |
| L-1 | Correct ordering clarification; quoted sentence not present in delivered file | Orphan checkpoint removal occurs before adoption decision | 39 record/reviewer description | Append precise ordering to 39 records; retain original text |

## Proof judgment and causal limits

H-1a's real reset from 4096 to 1024 correctly creates a
`counter_discontinuity` gap and starts the new baseline at 1024. Only the
following 4096-byte delta is recoverable, giving 8192 attributed bytes in all,
not 9216. Convert the proof to require that total and exactly that one gap.
Its additional restart mapping gap is the confirmed defect.

H-1b's original source omits devB even after a verified two-device Apply.
`render_nft.go` creates both named counters, including idle zero counters;
`usageCounters` records their binding metadata on a reset sample regardless
of whether they have bytes. Keep the absent-counter proof explicitly labeled
as a guard. Add a production-shaped zero-counter repair/nudge trace. Epoch
switching is event driven, not periodic: actual decreases/missing counters,
boot change, or the one-shot ruleset-repair nudge. Normal healthy sampling,
month boundaries and renewals do not switch epochs. There is no measured
production incident frequency; do not call every Step an epoch change.

Do not accept arbitrary earlier same-boot intent. A final sample followed by
PrepareSampling has ended its interval, even if a later intent write fails.
Use the ledger's append-only epoch history to prove that every intervening
transition is a same-boot counter discontinuity, with no final sample or new
baseline. This adds a read at restart only; it does not add a serving sample
or fsync. Loading intent after a failed marker read is attribution evidence,
not permission to adopt: full live verification remains required.

H-2 usually self-heals on the next successful hand-off: 39's failed intent
write calls rejectServing/withdraw, which finally sets finalDone. The failed
Step stays off until another admitted pass (default cadence ten seconds plus
work; no finite bound under repeated I/O errors). HEALTH_CHECK_FAILED at
stage apply is false attribution: Apply was never called. Main's same race
can instead resume without preparation and baseline away new bytes.

H-3b's persistent Baseline failure cannot prove that the kernel stopped. The
agent must report forwarding unverified (Serving=false, BaselineOK=false),
retry withdrawal, and never announce successful admission. Existing kernel
expiry/watchdog remain the backstop when the kernel executor itself fails.
H-3b′ is avoidable: checkpoint I/O must not bypass an otherwise working
Baseline. M-1's bytes remain in named counters: an unchanged identity can
still sample them; a later change runs the 39 hand-off, but a crash/retry
must not depend on those contingencies. Preserve the withdrawal retry.

L-1's quoted M-2 sentence and functions `loadUsageIntent`,
`prepareAdoptionUsage`, `resumeAdoptionUsage` are absent at `99f93112`.
Actual calls are latch (before adoption), ensureUsage with ExistingEpochOnly,
then setMeterBindings/activateUsage after the final gate. The clarification
is warranted; the quoted text is not evidence of a sentence in this file.

## Main comparison

Archive `61c7bf59` is under `.agent/f40/main`; no product edits. Whole-file
compilation fails on absent 38/39 APIs and r20/r21 helpers (receipt
`before-main-full.log`). Extracted unmodified H-2 passes its serving assertion
on main, which has no pre-Apply intent gate; this does not prove the epoch was
prepared. Extracted H-3b fails its final assertion but shows BaselineOK=true
and SafetyStatus=passed: main never invokes that injected pre-Apply hand-off,
so this is not the same defect. Receipt `before-main-portable.log`.
The M-1 portable check removes only the unavailable retainedPending predicate
and tests the same open epoch after two withdrawal attempts. Receipt
`before-main-m1.log`. H-1's new intent-specific claims cannot compile on main.

Requirements: MTR01–MTR04 (§10.2, §10.5–§10.9), ISO01/ISO02 (§16.3),
UPD04/AT13/AT16 mechanisms. Physical arm64 and field incident rates remain
unmeasured. Functional gates and further evidence will be appended.

## Implemented boundary and focused proofs

`SameCountingInterval` reads the existing append-only epoch rows, ordered by
SQLite rowid as ClosedCheckpoint already does. It accepts only an uninterrupted
chain of same-boot counter-discontinuity closes/opens ending at the open head.
It cannot cross a final sample, missing origin, new baseline or reboot. No
schema migration or additional durable write is required. A contradictory
successful live-generation read still defeats the intent; a transient failure
does not erase it. All adoption verification gates remain in place.

`epochClosed` records ledger closure independently of `finalDone`, which still
means barrier completion. Every interrupted publication therefore prepares
before a serving Apply. A withdrawal attempt sets retainedPending and clears
serving/expiry before fallible work. Baseline precedes checkpoint removal;
the original second Baseline attempt in withdraw remains, and the pre-Apply
return also retries a failed Baseline. A pending interval is a reporting and
admission gate even before a sample can set usagePersist. The deferred decision
tracks usage and serving changes, including the early settlement return.
Blocked Step returns publish their observation as well as their heartbeat.
A failed pre-Apply intent write is reported at usage, not as a failed Apply.

Focused converted r23 proofs pass, with additional matrix cases for barrier
removal/replacement, canceled ClosedCheckpoint read after a committed close,
atomic checkpoint write failure, held/read/persist sample failures, failed
Baseline, invalidation racing the pre-Apply check, and first-failed/second-good
Baseline. Each runs with and without an intervening restart and verifies the
post-recovery restart's bytes. The real renderer is called with the preserved
counter State for both Baseline and Apply across remove/re-add; it must neither
delete nor recreate the existing counter. Totals remain 8192 old + 1024 new.

H-1b also has a repair-Apply trace with a pending kernelSource notification
injected after intent publication; enforce's actual switchEpoch consumes it.
The original missing-idle-counter fixture remains a labeled guard. The separate
idle-zero fixture uses the real renderer's counter existence behavior. A failed
later intent write after a settled boundary cannot reuse the earlier intent.
The original M-1 portable main comparison fails on the same open/unsettled
state, confirming the pre-existing classification.

Cost: healthy serving Step adds only boolean comparisons; no extra source
read, sample, filesystem sync or SQL write. Restart after an epoch switch adds
one read-only epoch-history query. Failed hand-offs retain the existing retry
and the pre-Apply failure path adds at most one Baseline retry inside the same
local deadline. Blocked returns may enqueue/deliver a changed observation;
normal observation deduplication still applies. No performance claim is made.

### H-3 recovery reporting refinement

The first broad node gate caught the existing R18 direct canceled-enforce guard:
removing its cancellation diagnostic was too broad. Retain the original
cancellation-specific rejectServing path; ordinary intent I/O errors remain
usage failures. No R18 assertion is changed.

A new before-fail (`.agent/f40/healing-before.log`) demonstrates a second
reachable H-3a return: a reported usage persistence failure triggers withdrawal,
but its successful final sample clears usagePersist before the non-serving
receipt. It produces Reason="-" and no observation reasons. Preserve the usage
failure that caused this pass's withdrawal until the next Step's health refresh;
do not report that forwarding resumed merely because final persistence healed.
This is the same reporting invariant, not an additional admission authority.

### Supplemental frozen-product controls

An archive of the merged pre-fix product `dcdc3d96` ran the supplemental
fixtures without product edits. H-1c loses the new device's 2048 bytes into a
gap. H-1b with a real idle zero counter **passes on 39**, while the absent-counter
fixture fails. Thus no H-1b availability loss is demonstrated for a healthy
verified production renderer; that exact finding is a retained guard, not a
claimed production outage. The epoch-continuity limitation itself remains
confirmed by H-1a and the repair-switch trace.

The checkpoint-invalidation race (H-3b′) fails before edits with serving=true,
kernel generation 2 and usagePersist=true, with no retainedPending latch.
It is a reachable fail-open return, now covered by both restart variants.
Receipts: `.agent/f40/before-supplemental-2.log` and `before-h3-prime.log`.
The read-error variant already reports a meter-path failure in 39; the omitted
usage flag specifically matters for held and persistence-failed samples.

Main's subsequent `dff6bdb0` change contains design documents only. It was merged
before freezing the functional lab source; all earlier merges remain intact.
Node lint/unit/integration and amd64/arm64 builds pass with unchanged runtime
sources across that documentation-only merge. Whole-module and downstream
checks also run on the merged tree. No r20–r22 assertion was changed in 40.

Portable held tests on main `61c7bf59`, using only copied fixture helpers:
AQ2 fails (new member gets zero rather than 2048 bytes); AQ3 passes its
initialization-invalidation withdrawal assertion; AQ4 fails on the first
initialization attempt with usageOnce/store/lock retained. Receipt:
`.agent/f40/main-held-portable.log`. AQ1/AQ5/AQ6 depend on 39-only intent,
heartbeat or cleanup APIs and are not executable unchanged on main.
The later main `090bb97b` handover update was also merged before the final
lab freeze. Its handover/evidence changes are covered by the final module
and evidence gates; it changes no node runtime path.

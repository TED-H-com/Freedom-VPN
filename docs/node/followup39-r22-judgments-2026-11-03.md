# Follow-up 39: r22 judgments (2026-11-03)

Product before fixes: `31862b3a`, merged with `origin/main` as `ac5337cb`.
The unmodified proof file from `ef78b9b7` was run before product changes:
all twelve finding cases failed; all six cases in the three held-question
functions passed. Receipt: `.agent/f39/before-r22.log` (agent package 4.322 s).
The original proofs are committed first. Main comparison receipts and converted
proofs will be added after host admission; an attempted whole-file comparison
cannot compile because main lacks follow-up 38's APIs. No product shim is used.

Requirements: MTR01–MTR04, §10.2, §10.5–§10.9; ISO01/ISO02, §16.3;
UPD04 and AT13/AT16 mechanisms. No real-device acceptance is claimed.

| Finding or sub-trace | Verdict | Reachable | Introduced by | Fix to implement |
| --- | --- | --- | --- | --- |
| H-1 A | Confirmed wrong-member accounting | Yes, two serving Applies can precede a sample commit | Pre-existing serving path | Withdraw and commit outgoing attribution before an identity change; persist epoch-scoped provenance before admission |
| H-1 B | Confirmed unsettled hand-off bypass | Yes, small backward wall jump, counter-read or persistence failure | Pre-existing | Pending settlement gate for every failed hand-off, including restart |
| H-1 B paused-sampling sub-claim | Rejected as stated; retain a regression guard | Successful Apply unconditionally calls resumeSampling; skipped PrepareSampling does not skip ResumeSampling | No demonstrated defect | Exercise non-ErrHold recovery and periodic sampling explicitly |
| H-2.1 live retained generation | Confirmed false gap | Yes, desired saved before Apply | 38's candidate intersection; compare main's older generic restart loss separately | Atomic live marker excludes newer unapplied desired |
| H-2.2 unpublished desired | Confirmed false gap | Yes, withdrawal before final sample | 38's candidate intersection; older restart loss pre-existed | Exclude documents that could never admit members |
| H-2.3 added/removed device | Confirmed false gap for unique identity; older reuse needs qualification | Yes; removed counters remain but have no counting rule | 38's intersection; older restart loss pre-existed | Union unique candidate identities; conflict or unknown older provenance becomes a gap |
| H-2.4 closed prior epoch | Confirmed, with corrected crash ordering | Yes: Apply succeeds, retained write fails, rejection installs Baseline, crash before final sample | 38's unscoped intersection; stale-model mechanism existed in 37 | Persist intent scoped to boot and usage epoch before Apply |
| H-2.5 first Apply | Confirmed; original fixture deletes metadata artificially | Real first Apply can count before retained write | 38's nil-model early return; older missing provenance pre-existed | Pre-Apply accounting intent and first-generation legacy recovery |
| M-1 | Confirmed initialization latch | Yes, transient source or store failure | Pre-existing early latch; 38's pending gate makes the wedge persistent | Publish initialized state only after all initialization succeeds; release failed resources |
| M-2 ordering | Confirmed irreversible work before final gate | Yes, TERM or drift after meter initialization | Pre-existing | Prepare adoption meter without ledger mutation or upload; commit only after final gates |
| M-2 original false-gap trace | Rejected as causal proof; retain only as labeled guard if needed | No: it final-samples with nil bindings before Boot, already creating its asserted gap; it also closes while leaving forwarding live | Fixture defect | Replace with reachable open-epoch late-abort proof and assert no pre-decision writes/uploads |
| M-3 A | Confirmed stale heartbeat after Baseline | Yes, ledger block after successful Baseline | 38's bounded worker returns before the old late heartbeat; late-write ordering pre-existed | Write withdrawal before ledger work |
| M-3 B | Confirmed stale heartbeat on timeout | Yes, blocking runtime/baseline call | 38's bounded worker; failure publication gaps pre-existed | Publish conservative startup state before blocking work, and on abort |
| M-3 DatapathValue | Production-unreachable guard | LoadConfig.validate rejects it before Boot; tests can mutate Config | Pre-existing raw-error guard | Wrap ErrBoot and publish conservative heartbeat |
| M-3 C | Confirmed false clear receipt and stale heartbeat | Yes, nft baseline clear fails | Pre-existing | Keep issuance uncleared, retry clear, publish failure after probe cleanup |

## Reachability and proof corrections

The default Step cadence is min(heartbeat, poll), both ten seconds. A slow
Step can leave no sleep before the next one. resumeSampling only queues a
nonblocking wake; the window lasts until the source read and SQLite commit
finish, with no hard upper bound when either blocks. A held wall sample can
span another Step. syncLease before evaluation can obtain the new generation
in the same pass; renewAppliedLease can also re-enter enforce in one pass.
Adding a device then transferring its membership or binding revision in the
next published generation is an ordinary configuration sequence. The original
proof's two 59-minute fake advances explain lease renewal but exaggerate the
normal scheduling window; converted tests will force the real refresh trigger.

ErrHold is specifically a wall-clock rollback no larger than the sample
interval, not an outbox-row-cap error. This tree has no MaxUnackedRows setting.
Outbox age/bytes and disk protections are separate gates. Source I/O and SQLite
failures are independently reachable. FinalSample pauses on all errors; every
successful enforce path nevertheless resumes sampling. The misattribution
bypass is real even though r22's permanent-pause explanation is not.

The nft generation marker and counting rules are rendered into the same atomic
transaction (`node/enforce/render_nft.go`). A retained live generation excludes
a later desired generation, absent an out-of-band rollback (which must fail
closed). A generation omitting a device contains no usage-counting rules for
it, but preserves the named counter object. Removed-then-readded counters can
therefore contain older bytes. A mere absence from one candidate is insufficient
proof of a fresh counter; the committed head and transition provenance matter.

A failed retained_applied write does **not** allow serving to survive:
enforce calls rejectServing, which calls withdraw/handoff. It can install
Baseline and then crash before final settlement. H-2.4 remains reachable by
that order, not by a later successful serving pass overtaking an unfinished
SetMeta. H-2.3's removed-device increment must occur before its removal, and
H-2.5 needs an actual first-Apply crash rather than a metadata deletion.

M-2's manual FinalSample uses r20ServedPendingDelta's returned source, whose
binding callback is nil until r20Successor sets it. Its pre-Boot close already
loses 4096 bytes to a mapping gap. Regular Sample opens a replacement epoch on
reset; it does not leave a closed epoch while serving. These are D-141 fixture
errors, not evidence of production misattribution. The pre-decision constructor
and uploader still violate the required adoption ordering and will be fixed.

Probe source/destination elements have kernel timeouts. A failed clear leaves
the narrow exception usable only until the installed TimeoutCeilSeconds expires,
no later than the durable issuance deadline (floor to whole seconds), and at
most ProbeMaxSeconds (300 seconds) from issuance. Rules/objects may remain after
expiry, but expired elements no longer authorize packets. Marking Cleared on a
failed clear rejects another install for that nonce; the in-process
probeChallenge guard also prevents repeating it. A dedicated clear retry must
not depend on reinstalling or extending the nonce.

The shipped service is `deploy/node/systemd/fvpn-agent.service`, with
TimeoutStopSec=15s, pinned by `packaging/installer/followup9_test.go`. There is
no main.go defaultStopTimeout and no twenty-second shipped stop setting.
Boot's ten seconds plus Close's three seconds must continue to fit within 13s.

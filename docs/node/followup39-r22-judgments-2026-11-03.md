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

## Main comparison receipts — 2026-11-03

An archive of `8a775315` was tested with fixture-only adapters (no product
changes). Unsupported 38-only test functions were removed from that archive;
the in-memory plane was given its read-only RestartGeneration helper. The
original H-1 A restart assertion passes on main because main does not settle
the retained delta at Boot. Sampling **before** that restart exposes exactly
the same wrong charge: revision 3 gets 4096 bytes, revision 2 gets none.
H-1 B also charges 4096 old bytes to revision 3. The portable M-1 retry leaves
usageOnce=true with meter=nil after recovery. M-3 C records Cleared=true and
BaselineOK=true after the clear failed. These are pre-existing mechanisms.

All five H-2 original Boot assertions fail on main too: the new delta has not
yet been recorded there, rather than being finalized as 38's conflict gap.
They do not establish a no-loss result on main. The M-2 original fixture's
pre-Boot false gap also reproduces there and remains rejected as causal
proof. The portable main receipts are `.agent/f39/before-main-r22.log`
(1.170 s) and `.agent/f39/before-main-extra2.log` (0.589 s). Build failures and
all admission refusals are retained, separately from test results.

## Implemented boundary and crash ordering

A device identity is `(device, member, binding_revision)`. A change in the
identity set, including removal, takes Baseline before FinalSample. The old
snapshot remains installed in the meter until that sample commits. A failure
keeps the pending gate set; neither desired refresh nor Apply can overtake it.
The next epoch is opened from a real unbound counter read before admission.
The agent then durably writes `usage_apply_intent` with boot, epoch, generation
and identities. Only then may Apply run. An intent write failure withdraws.
Same-generation/same-epoch renewals reuse the existing intent after reading
the current head; they do not repeat the intent write.

There is no transaction spanning two databases and the kernel. Safety follows
from ordering: before the new epoch, the old interval must be settled; before
the intent, the kernel is still blocked; before admission, the intent is durable.
A crash after intent but before Apply has zero new delta. A crash after Apply
but before retained_applied still has exact epoch-scoped identities. Repeated
crashes before settlement keep that intent. A counter reset can change the
meter epoch independently; a mismatching intent cannot label that new epoch.
Legacy recovery uses the checkpoint and the live-generation-bounded candidates,
recording a gap for unknowable older ownership instead of assigning it to a
new member. A missing device in a generation supplies no counting rules, but
its named counter can survive removal. The remove/re-add regression proves
8192 old bytes remain on revision 1 and only 1024 new bytes reach revision 2.

The added binding-change cost is one verified Baseline, a final counter read
and durable sample, a fresh baseline read/epoch commit, and the pre-Apply intent
write. It is bounded by the existing Step local budget; failure defers serving
to a later pass. Ordinary same-identity lease renewal adds a ledger-head read,
with no extra withdrawal and no redundant intent fsync. The reserved lab
comparison measures the resulting whole serving path; command guard timings
alone do not measure the preceding accounting boundary.

D-141 conversions: H-2 added/removed traffic now advances the removed device
before removal and crashes inside real enforce/Apply. The closed-epoch case
injects an actual retained_applied SQL write failure and crashes at the ensuing
Baseline, before final settlement. The first-Apply case starts with no retained
model and crashes in its real first Apply; it no longer deletes metadata after
an unrelated serving history. r21's direct plane-Apply fixtures are explicitly
legacy upgrade guards and remove the new intent they otherwise bypass; their
unchanged-device and gap assertions are retained. r20's stopped-Boot heartbeat
now expects observed generation zero rather than the old acknowledged two.

Follow-up 39 tests also exercise add/rebind beside an unchanged device with
both absent and nil-bound heads, with and without restart; ErrHold and source
failure across hand-off, pending Step, restart, recovery, a running periodic
sampler, and another restart; late adoption refusal with no premature usage
mutation; failed probe clear and a successful retry without nonce extension;
and the unreachable static-datapath validation guard. The actual 13-second
blocking matrix is retained.

## Follow-up 40 corrections — 2026-11-03

This section corrects the delivered 39 record without replacing its text.
See [r23 judgments](followup40-r23-judgments-2026-11-03.md) for D-141 verdicts,
fixture limits, main comparisons and the implemented mechanisms.

- H-1: the statement that an epoch-mismatching intent cannot label the new
  interval was too broad. Regular Sample can switch epochs while the same
  generation serves, and enforce's repair nudge can do so after publishing
  intent. Those reset successors retain the same counting identities. A
  settled final-sample boundary does not. Follow-up 40 proves that distinction
  from the ledger's epoch history and loads intent even after a transient
  live-marker read failure. The original single-candidate promise therefore
  needed a fix. No real reset is reclassified as gap-free: r23 H-1a's reset
  itself remains a counter_discontinuity gap.
- H-1b: r23's absent-idle-counter proof is retained as a guard, not claimed as
  a healthy production renderer state. The renderer creates zero counters;
  reset sampling records their metadata even before their first packet.
  Follow-up 40 additionally covers zero counters and the actual enforce
  switchEpoch path after repair. No measured production switch frequency is
  available; switching is incident driven, not a periodic sampling action.
- H-2: finalDone did not reliably mean the ledger was closed. It was cleared
  before barrier read-back/publication, so an interrupted publication could
  skip preparation and generate a false Apply health-check failure. Epoch
  closure now has its own flag; barrier identity and completion stay separate.
- H-3/M-1: “a failure keeps the pending gate set” did not cover Baseline or
  checkpoint-invalidation failures before the latch. Nor did the deferred
  decision watch all usage/serving changes. Follow-up 40 latches before those
  operations, withdraws on invalidation failure, recomputes reports, and
  retains the second Baseline attempt. A failed kernel withdrawal is reported
  as unverified, never as verified serving. Normal verified withdrawal and
  persistence-failure traces show both reports agreeing with generation zero.
- L-1 ordering clarification: latch removes orphan meter.checkpoint evidence
  when no barrier is active, both before adoption and under the node lock.
  It is not deferred until adoption's final verification. The delayed work is
  setting live meter bindings and activateUsage (including uploader/restore).
  ExistingEpochOnly prepares the meter without ledger mutation. The sentence
  quoted by r23 and its prepareAdoptionUsage/resumeAdoptionUsage names do not
  occur in the delivered file/source; this clarification records actual calls.

The real renderer is `node/enforce/render_nft.go`. Its forward-chain marker
is `fvpn_generation=<G> fvpn_metering=<V>`; ParseState obtains Generation via
forwardGeneration. Neither serving nor Baseline rendering deletes named
counters or the table. TestRender_Golden and the renderer's deletion guard are
the real tests. The removed/re-added counter argument must use final settlement
and the next unbound baseline, not counter deletion. Follow-up 40 verifies this
with real renderer calls supplied the preserved counter State and restart.
The shipped stop-budget test is in packaging/installer/followup9_test.go.

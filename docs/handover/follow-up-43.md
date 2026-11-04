# Follow-up 43 — #446–#456; recovery fixes merged, acceptance still bounded

Verified against `main` **04d31ea5cd10cb90527e0b4efc10b42522dd9569**, the
final roadmap commit through **#456**, on **2026-11-04 EST**. Immediately after
follow-up 42's cutoff **#445 / 5a942aa1**, the range contains exactly
**#446, #447, #448, #449, #450, #451, #452, #453, #454, #455 and #456**:
**eleven merges and eleven final roadmap rows**. No unbriefed PR merged in the
range. Local main and the public PR listing ended at #456 when checked at start.

**Running at briefing, status only:** devstack **34** (F30-01 acceptance on
both halves, remove the nightly `lease-expiry` quarantine entry only if that
run passes it; also the `publisherTrace32` comment); isolation **22** (make
Isolation F21-01's member-expiry schedule reach its window and run once on
agent-core 49); secops **17** (construct a fresh short lease for SECOPS-F16-01
and run AT29 once); agent-core **50** (diagnose members 21's AT02 interval,
then at most one early renewal per closed recovery under D-157). No delivered
or merged PR for these follow-ups was present at start. Any later delivery is
**after the cutoff** and does not enter this cut.

**D-098 / D-109 / D-132:** `TestHandover_Followup43Claims` pins final roadmap
rows, verbatim PR excerpts, frozen #449 handover inputs, frozen producer
records, review r29, and Go test declarations on main. Every lane-document
quotation comes from a committed frozen copy. No claim reads a living members,
isolation, secops or usage evidence root. Follow-up 42's two document reads
now use its already-complete frozen copies; its other rewritten inputs use
#449 snapshots. All 36 earlier claims functions retain their assertions.

## Handover 42 (#449)

[PR #449](https://github.com/TED-H-com/Freedom-VPN/pull/449) records the
previous **#439–#445 cut, up to 5a942aa1**. Its actual test and manifests pin
**15 subtests**, **seven final roadmap rows and PR excerpts**, **ten frozen
#440 inputs**, **59 immutable records**, **19 producer prose copies** and
**31 main Go test declarations**. Its **35 earlier claims functions** stay
exact after reversing six input redirects. The register has **57 decisions
and 113 findings rows**: seven updated, eight added, 98 byte-identical. These
are #449's counts, not this cut's counts.

The session account is attributed to #449's body: **codex gpt-6-astra**,
launched **21:06 EST** on **2026-11-03**, delivered **21:22**, **5 commits and
58 owned files**, **+12,036/−305**. Scope: docs, claims, fixtures, snapshots,
review archives and digest values; no reservation or Docker.

## F30-01: updater 24 (#446) and agent-core 49 (#454), D-155

[PR #446](https://github.com/TED-H-com/Freedom-VPN/pull/446) and its
[frozen dated note](../../tests/handover/testdata/history/follow-up-43/docs_updates_node-followup24-2026-11-03.md.txt)
record the updater half of review r29 H-1 and L-1. **Exit 3 (`ExitBusy`)**
means only `UPDATE_BUSY` with detail exactly `lock`, a live same-boot holder;
wrapped errors count. Stale-holder details remain **1**; `ErrMembersBlocked`
remains **2** and wins over busy. All **six lock-taking entry points** check
`update.lock` before journal access: `Run`, `Recover`, `ApplyPending`, `Seed`,
`Flush`, `Retain`. Five open/migrate only under the lock; `Retain` never opens
one. Read-only status remains lock-free. The complete exit contract is in the
[operator guide](operator-guide.md#f30-01).

The imported updater r29 file is byte-identical: **two TODO proofs fail before
and pass after; two Held proofs pass throughout**. Import commit **03e24246**
preceded fixes. This was offline work, not an acceptance run.

[PR #454](https://github.com/TED-H-com/Freedom-VPN/pull/454) and its
[frozen dated note](../../tests/handover/testdata/history/follow-up-43/docs_node_followup49-2026-11-03.md.txt)
record the agent half, r29 H-1 and M-1. The candidate defers boot recovery
while its parent holds the live lock, or on exit 3, and retries at the top of
later Steps. Exits **1 and 2** and missing config remain terminal. Recovery
is **one tracked child**, owned by the engine's run context: a **250 ms** first
wait, later Steps poll without waiting, at most one dispatch per second.
Stopping cancels the process group and joins the child. Recovery cannot
self-deadlock Boot or Step while awaiting that agent's checkpoint or health
attestation. Real updater recoveries from **blocked, switched and checking**
crashed stages keep the agent running.

The agent's import **facfd361** is also byte-identical; its two TODO proofs
fail before and pass after, its two Held proofs pass throughout. D-053 health
attestation, D-064 probe exceptions, lease renewal and watchdog publication
remain active while recovery is deferred. See the [operator behaviour](operator-guide.md#f30-01)
for central's `applying` / `recovery_required` and possible zero-grant lease.

**F30-01 stays open.** Both code halves are merged under D-155, but closure
requires an acceptance run passing **`lease-expiry` with the nightly quarantine
entry removed**. Devstack 34 is running for exactly that at cutoff. Agent-core
49's ordinary lab has no scenario starting a candidate under its parent-held
update lock, so its lab pass cannot close this finding.

## Agent-core 48 (#447), F48-01 and D-156

[PR #447](https://github.com/TED-H-com/Freedom-VPN/pull/447) and its
[frozen dated note](../../tests/handover/testdata/history/follow-up-43/docs_node_followup48-2026-11-03.md.txt)
record five fixes:

- **r28 H-1 / Isolation F20-01:** the probe inherited the Step's exhausted
  control budget. Local work, the tunnel probe and usage-store lock now have
  their own context; Boot's R21 `adoption_probe` budget remains protected.
- **r27 H-1:** a missing member interface at restart no longer falsely latches
  `TC_FOREIGN_FILTER`; absence needs a successful independent link inventory.
- **r27 M-1:** the grant-less deadline schedules Step; the verified deny-all
  recovery model is retained while waiting, without a redundant re-baseline.
- **r27 L-1:** one containment attempt per local pass, reusing its verified
  baseline until another rules mutation.
- **R23 boot heartbeat:** remove its independent **100 ms** child deadline;
  retain Boot's deadline and the expiry check before publication.

The four imported TODO proofs fail before and pass after; **six Held/WhatHeld
proofs pass**. R23's deterministic budget proof also fails before and passes
after. D-150's frozen-A code reading is **not determined**, for the record
only; D-153 remains closed without a comparison.

The one lab run records **12 pass, 1 fail, 1 optional skip**. D087/D104's
last-positive lower bound failed: forwarding stopped **2.86 s before D**,
against a **101 ms** bound. The lane assigned no cause. The orchestrator's
capture reading names **F48-01**: a watchdog pass on the successor's startup
heartbeat withdrew the verified retained rules before adoption in a same-boot
restart. **Agent-core 48 found it; it did not cause it.**

Agent-core 49 **#454 fixes F48-01 under D-156**: a pending-adoption receipt
keeps the predecessor's **original heartbeat age and kernel deadline**, with
**no serving claim**. A repeated startup cannot refresh it. A stalled
successor remains bounded by the old staleness limit or the kernel deadline;
a refused adoption takes the baseline. r22 M-3's protection is kept. Its
ordinary lab passed **13/13 on one attempt**. The
[agent/watchdog release rule](operator-guide.md#d156-release-rule) applies.
Live coverage of r28 H-1's fix is isolation 21's qualified short expiry below.

## Shared node writes: secops 15, enrollment 10, updates 15 and usage 9

All devstack 32-audited AT-owned sites are answered on main. The
[frozen original audit](../../tests/handover/testdata/history/follow-up-43/docs_e2e_runs_2026-11-03_followup32_audit.md.txt)
is reconciled with isolation #442, members #445 and these four PRs. Each uses
`Stack.WriteNode`, an **exact read-back predicate**, **embedded receipts**
and an **AST bypass audit**. Exact 412 `RESOURCE_VERSION_CONFLICT` handling,
fresh ETag, same key/body, three writes / ten seconds, and separate operation
and observed-state checks remain. First-attempt success does not exercise the
conflict path.

| PR / frozen dated evidence | Adoption and result |
|---|---|
| [Secops 15 #448](https://github.com/TED-H-com/Freedom-VPN/pull/448) / [note](../../tests/handover/testdata/history/follow-up-43/docs_e2e_at_secops_2026-11-03-follow-up-15.md.txt) | Six sites: four capacity predicates, exact publication and revocation. Capacity/publication ran live, each first-attempt 202; other sites have offline audit/compile coverage. One live AT29 transport run 35116d25. |
| [Enrollment 10 #450](https://github.com/TED-H-com/Freedom-VPN/pull/450) / [note](../../tests/handover/testdata/history/follow-up-43/docs_e2e_at_enrollment_2026-11-03-follow-up-10-node-writes.md.txt) | Three audited sites, five commands: AT06 incarnation revoke, AT07 freeze/installer capacity and publication, and shared clone/reinstall revoke. Exact predicates and receipts. Offline only; no live claim. |
| [Updates 15 #451](https://github.com/TED-H-com/Freedom-VPN/pull/451) / [note](../../tests/handover/testdata/history/follow-up-43/docs_e2e_at_updates_follow-up-15-2026-11-03.md.txt) | Installer fixture retirement DELETE: exact lifecycle_state=retired, fixture_retirement_node_write receipt; 202 only, observed retirement gate unchanged. Offline only; no live claim. |
| [Usage 9 #453](https://github.com/TED-H-com/Freedom-VPN/pull/453) / [note](../../tests/handover/testdata/history/follow-up-43/docs_e2e_at_usage_2026-11-03-node-write.md.txt) | nodeMutation's 16 call sites; exact capacity, publication, revoked and retired predicates. PATCH 200 or 202; publication PUT, revoke and DELETE 202 only. Review fix f2fb3678 restores 202 for seven checks. Offline only; no live claim. |

Secops 15's **manual grant withdrawal uses `revokeMemberGrant`**, through the
public API. The operation completes and both original transports first go
quiet **2.48 s** after the request; both stay quiet with the agent killed and
watchdog running. A new grant's applied ACK brings fresh replies. Original
finite standing is restored through public APIs. All four runnable AT29
sub-cases pass; AT29 then stays blocked **only on `guest_clock_rollback`**.
That run did not assert whole-control-outage continuity. Its status is
superseded by secops 16 below. The AT-owned node-write register row **closes
for adoption**, without promoting the last three lanes' offline work to live
acceptance.

## Isolation 21 (#452)

[PR #452](https://github.com/TED-H-com/Freedom-VPN/pull/452) and its
[frozen dated record](../../tests/handover/testdata/history/follow-up-43/docs_e2e_at_isolation_followup21-2026-11-03.md.txt)
archive **165** follow-up 20 root/support/reload-stream files byte-identically
under D-098 before the new run. **r28 M-1** now judges member expiry only under
a serving node: heartbeat and forward-chain reads before and after the five
denial probes. Run 20's original evidence **replays as fail**. Its historical
pass stays recorded and no longer counts as expiry evidence for a serving node.

One assembled run **7599576e** records **26 pass, 3 blocked, 0 fail**:
AT10 **9/0/0**, AT11 **2/1/0**, AT12 **10/0/0**, AT13 **5/2/0**.
**F20-01 is answered by #447**: short whole-node expiry passes unchanged
**D-140** and qualifies under **D-142**, with 165 positives and 88 drops,
last positive **D−99.54 ms**, first drop **D−96.36 ms**, zero interleavings.
Run 20's failure stays failed.

**Isolation F21-01 is new and open:** the member-expiry window was not reached.
None of 152 renewal samples exceeded the unchanged **25 s** extension bound;
maximum **23.06 s**. No denial probe or serving read ran. This is a coverage
gap, not a product defect; r28 M-1 has no live coverage yet. Isolation 22 is
running for it at cutoff. This is distinct from **Updater F21-01**, fixed by
#417. **15 node writes**, all first-attempt 202, with every receipt retained,
answer follow-up 20's missing-write-trace request.

## Members 21 (#455)

[PR #455](https://github.com/TED-H-com/Freedom-VPN/pull/455) and its
[frozen dated record](../../tests/handover/testdata/history/follow-up-43/docs_e2e_at_members_followup21-2026-11-03.md.txt)
archive run **50117c68**'s **81** root records byte-identically under D-098.
The assembled run **5715939e** on agent-core 48 records **2 pass, 8 blocked,
0 fail; 705 passed, 8 blocked, 0 failed subchecks**. AT02 and AT03 pass;
case-level resource blockers stay separate from measured subchecks.

**AT09:** index **27 passes**, with no `TC_FOREIGN_FILTER` in four post-restart
reports, so r27 H-1's fix holds live. Index **25 is blocked / not exercised**:
no grant-less lease followed restart. Index **26 passes**. This run does not
close F48-01; its retained `baseline_unreadable` event is distinct from the
startup `baseline_unverified` signature, and overwritten logs cannot exclude
an earlier event.

**AT02:** accounting **190** and sibling recovery **191** pass under
**D-135/D-136**. All **1,237 losses among 6,165 sends** are in measured brackets.
Closed-recovery Apply **558** is followed by **4.968 s of deny-all and 241
losses** until Apply **564**. Renewals are **5.0006 s** apart. No baseline
write occurs between them: r27 M-1's re-baseline is gone, but the wait for the
next grant-less renewal slot remains. Passing accounting does not prove
zero-loss availability. This new item goes to agent-core under **D-157**;
agent-core 50 is running, with no merged fix. See the single
[known-behaviour statement](operator-guide.md#at02-closed-recovery).

**Newly blocked, routed to devstack:** **F21-AT08**, the first selected-route
transfer (five member packets, none at node/origin, counters unchanged; later
success does not repair it); and **F21-D146**, native IPv6 receive-only 1280
indexes **49 and 51**, on **`serving_read_ok=false`**. Bytes, MSS, reverse
datagrams and no-PTB pass, but the serving-query cause was not retained.
Neither has a proven cause. **30 node writes**, all first-attempt successes,
with receipts; no conflict-path live claim.

Verification fix **af4b5719** restored the dated follow-up 20
`F20-r27-predicates.md` bytes pinned by handover 42 after an archive-link edit
broke that test on the merged tree. This handover moves that read and
`isolation/followup20-2026-11-03.md` to their frozen follow-up 42 copies,
without changing assertions. **Every historical run stays failed.**

## Secops 16 (#456)

[PR #456](https://github.com/TED-H-com/Freedom-VPN/pull/456) and its
[frozen dated record](../../tests/handover/testdata/history/follow-up-43/docs_e2e_at_secops_2026-11-03-follow-up-16.md.txt)
add **`control_lost_agent_alive`**, AT29's r28 H-1 continuity evaluator.
Synthetic proofs accept continuity to D then quiet; reject withdrawal at
loss + 5 s, replies after D + 3 s, replaced agent, early heartbeat withdrawal,
missing evidence, short windows and stopped transport.

Its one live run **1bfd5cff failed at setup (SECOPS-F16-01)**. The fleet cap
was lowered to **60 s**, but the node retained its lease issued under the
**24-hour cap**. Issued leases keep their cap; no fresh short lease arrived
within the **90 s** wait. **No outage was run and no continuity result exists.**
The failed row remains failed. The orchestrator's reading is **case setup,
not a product defect**. Secops 17 is running to construct a fresh short lease
and run once. Isolation #452 supplies separate live coverage of r28 H-1.

**AT29's current status is failed.** The four original runnable sub-cases
pass; `guest_clock_rollback` stays blocked. **AT31's reproduced bounded relay
row passes; AT31 stays blocked.** No operator resource or platform gap closes.

## Decisions, review archive and findings

The [decision table](README.md#follow-up-43-decisions-register-2026-11-04-est)
adds three provisional orchestrator rulings, owner-overturnable, from **57 to
60**. Main had no decision after D-154; every earlier row stays byte-identical.

- **D-155**, **21:0x EST, 2026-11-03**: r29 Option B, exact live-busy exit 3,
  deferred/retried tracked recovery, terminal 1/2. Reject A's busy exit 0
  (invariants 2/6), C's narrower lock, D1's post-seal restart (D-053), and D2's
  completion marker (invariant 2). Both code halves merged; acceptance pending.
- **D-156**, **21:3x EST, 2026-11-03**: preserve verified retained rules until
  same-boot successor adoption/refusal, bounded by original staleness and D;
  keep r22 M-3. Implemented by #454.
- **D-157**, **00:2x EST, 2026-11-04**: at most one early renewal per closed
  recovery, when the agent observes that central holds every input its grant
  decision reads. D-067 stays unchanged otherwise, including a grant-less
  answer to that early request. If readiness moves in an unobservable job,
  stop and route; no faster polling. Agent-core 50 is running, nothing merged.

[Review r29 and dispositions](../reviews/README.md#r29-dispositions-at-the-follow-up-43-cutoff-2026-11-04-est)
archive the design diagnosis byte-for-byte from **5b1ca726**, base **dd052c60**,
by **agy/gemini-4-argon-high**, committed by that reviewer with the matching
Agent trailer. **180 lines**, SHA-256
`5c54bfb22cc9f407f779f8f2f401530ffe5238711b2adc10401d69abee20b7af`;
**0 Critical, 1 High, 1 Medium, 2 Low**. It is now a D-098 input. H-1: #446
and #454 under D-155; M-1: #454; L-1: #446; L-2: devstack 34 running. The
owner question is answered by provisional D-155. Historical raw wording is
not replaced with current dispositions.

The [findings register](README.md#findings-register-at-the-follow-up-43-cutoff)
has **124 rows: 11 updated, 11 added, 102 unchanged**. AT29 had no standalone
row in #449, so its current failed status is added alongside the ten requested
new findings. The `publisherTrace32` row and every unrelated row are unchanged;
devstack 34 owns that nonblocking comment at cutoff. Historical failures stay
failed. Nothing else in this cut changes operator/member behaviour; no date is
promised for unmerged work. No new handover live run or physical-arm64 claim.

### Requests to orchestrator

**2026-11-04 EST:**

1. **Devstack 34:** retain F30-01 and r29 L-2 as open until the acceptance run
   on #446/#454 passes `lease-expiry` and removes the nightly quarantine entry.
   Also answer the unchanged `publisherTrace32` comment. Running at cutoff.
2. **Isolation 22:** reach the unchanged member-expiry window, then one
   assembled run on agent-core 49 for Isolation F21-01 / r28 M-1. Keep the
   original c52bea6c and 7599576e outcomes. Running at cutoff.
3. **Secops 17:** establish a fresh short lease after the cap change and run
   live AT29 once; preserve failed 1bfd5cff. Running at cutoff.
4. **Agent-core 50:** diagnose central readiness after closed recovery before
   implementing D-157's at-most-once observable early renewal; stop and route
   if the necessary transition cannot be observed. Running at cutoff.
5. **Devstack / members owners:** investigate F21-AT08 and F21-D146 without a
   presumed cause. Members' three archived G2 links need the outside-owned
   `docs/e2e/at/poc/G2-followup34-2026-11-03.md` alias already routed by #455.
   Handover does not rewrite pinned lane evidence or add that alias.

These carry existing routed work. No outside-owned pin needs changing for this
handover, and no item is stopped. Earlier operator/resource requests remain.

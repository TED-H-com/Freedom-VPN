# Follow-up 27 — isolation boundary failure, serving-node gaps and r19

Fixed `main` cutoff **37db5405ee70730fe6a709d983f95f2460136c34**, through
**#385 (members 15)**, checked on 2026-11-01 EST. This cut is **#383–#385**.
Nothing was merged after the cutoff at task start. **Devstack 26, netlab 24
and agent-core 34 were running**; their results do not enter this cut.
Handover ran no new live acceptance. **Merged after the cutoff: #386.**

**D-098 / D-109 / D-132:** [three exact roadmap rows](../../tests/handover/testdata/followup27-main.json)
and [14 immutable snapshots](../../tests/handover/testdata/followup27-snapshots.json)
pin `TestHandover_Followup27Claims`: **19 subtests / 182 owner archive file checks**.
Snapshots retain source commit, full-source digest, byte offset, length and
excerpt digest. PR hand-off excerpts are immutable; they are not copied living
evidence. Run **56d20530** and run **e80bb592** root files, **gzip files included**,
are living until **isolation 15** and **members 16** archive them respectively:
**cite by roadmap row / PR only**. The claims test never reads those roots.
Owned-document secops digests check integrity, not acceptance.

## Previous cut — handover 26, #384

[Follow-up 26](follow-up-26.md) covered **#374–#382 at 9dde5eeb**, plus the
**5f4b455c** row correction. `TestHandover_Followup26Claims` retains **21
subtests / 9 exact roadmap rows / 17 immutable snapshots / 1,075 archive and
run file checks**. Its former register and entrypoints are now frozen under
`tests/handover/testdata/history/follow-up-27/`. Every earlier claims test
stays true; its historical assertions are not reinterpreted. A later roadmap
edit changed #381’s row; its exact cutoff wording is retained in the
[immutable row snapshot](../../tests/handover/testdata/history/follow-up-27/roadmap381.md).

## Isolation 14 — #383

D-098 archive of **38074149**, **112 root files**, under
`docs/e2e/at/isolation/history/follow-up-13/`. Run **56d20530** on **#382's tree**:

| Case | Result at this cutoff |
|---|---|
| AT10 | **PASS 9 / 0 / 0**, **ISO-F6-01 6 / 6**. |
| AT12 | **PASS 10 / 0 / 0**. |
| AT11 | **BLOCKED 2 / 1 / 0**, operator rows. |
| AT13 | **FAIL 5 / 1 / 1**, new **isolation F14-01**. |

**Isolation F14-01 is distinct from members F14-01.** The unchanged whole-node
D-104 predicate needs a positive leaving at or after **D−101 ms**. The last
positive left at **D−162.220 ms**. **Packet 161**, scheduled at **D−101 ms**,
reached ingress at **D−97.638 ms** and was dropped. Closure therefore lies in
**(D−162.2, D−97.6] ms**: **undecided and failed**, not proof of an on-time
closure and not a retrospective pass. The predicate's regression on main is
`TestAT13_D104StrictBoundaryAndMissingEvidence`; archive coverage is
`TestD098_Followup13ArchivePins`.

One **bootstrap hold (zero-member lease)** joined exactly by report, lease and
incident and ended **`generation_matched`**. **No populated hold was produced**.
The **alarm, escalation and active-stop retention are not exercised**; the
bootstrap join does not certify them. `TestAT29_M3CentralHoldEvidence` retains
that boundary. A populated stale-lease fixture remains assigned to devstack,
then isolation and members consume it. All current-run claims above are
pinned to **#383's roadmap row / PR**, not living files.

## Members 15 — #385, cutoff

D-098 archive of **258ba504**, **70 root files**, under
`docs/e2e/at/members/history/follow-up-14/`. Run **e80bb592**: **2 pass / 6
blocked / 2 fail**, **644 / 6 / 2 subchecks**. AT03 and AT08 pass.

**AT02 fails**: **93 successful Applies, 0 failed; 65 baselines paired to
recovery**. Of **4,131 attempts**, **277 have no egress**:

- **249 never reached node ingress**, sequences **1108–1356**: one **5.0975 s**
  gap between observed ingresses, **zero capture drops**, **cause unproven**.
- **28 reached ingress without egress**, **12 inside guards / 16 outside**.

Every loss counts. Missing ingress does not become a zero-loss interval or
proof of a baseline cause. No finite outage ceiling or waiver is introduced.

**AT01's F11-01 first selection fails after proven serving readiness at
generation 5**: **five member SYNs, zero node or origin packets**. **Three
D-077 re-grants pass**. **Members F14-01 stays failed historically**. The first
selection, AT02's 249-sequence ingress gap and members F14-01 are **three
samples of serving readiness proven, member packets absent at node ingress**.
This signature is **distinct from devstack 25's closed-node timeouts**; a
shared cause is unproven. The [#385 routing](../../tests/handover/testdata/history/follow-up-27/members15-serving-routing.md)
assigns the serving-node investigation to **devstack 27 (collected)**. Devstack
26's closed-node investigation and F14-01 work were running at this cutoff.

**AT09:** **682d4434's routed-after-relay order is live-qualified**. The new
**receive-only dual MTU-1280 row fails: 12,012 bytes in 12,011 ms**. It is a
**D-137 limitation, not upgraded or waived**. Historical F8-02 stays failed.

Large evidence files are **lossless gzip with compressed and uncompressed
SHA-256**. The [#385 resume hand-off](../../tests/handover/testdata/history/follow-up-27/members15-index-resume.md)
confirms that the old plain **AT02.md is a 2,551-byte navigation index**, not a
capture: it holds run e80bb592, **FAIL, 142 / 0 / 1 subchecks**, all **277 losses**,
both original capture links, both digest and byte-count pairs, and `gzip -dc`
instructions. Its own bytes are **excluded from the capture pins**. Go and
Python evidence readers **reject the document index**; explicit `AT02.md.gz`
verifies the original pins, while absent logical `AT02.json` falls back to gzip.
`TestEvidenceGzipLosslessAndPins` covers index/capture coexistence and rejection.
The operator guide's existing link remains useful and unchanged. No replacement
target is requested. These claims use **#385's roadmap row / PR only**, including
the resume; neither the index nor the gzip roots becomes a handover claim pin.

## r19 archive and routed findings

The grok review of **agent-core 33's option (b) design**, delivered **2026-11-01
07:47 EST**, is archived byte-for-byte from
`origin/r19-intersection-design-review:.agent/REVIEW.md` at **6048e099** as
[r19-intersection-design-review.md](../reviews/r19-intersection-design-review.md).
It is **41,050 bytes**, SHA-256
**14caf0193b256d6f090e8d53ff77e0d5a5c9dab4d025794a6812e15e83228913**.
Only the report is archived; proof files remain on the review branch. Review
proofs and What held are not assembled acceptance results.

| Finding | Disposition at #385 |
|---|---|
| r19 H-1 | **Routed; in progress — netlab 24**: an established download from a destination newly added to `denied_v4` keeps flowing. |
| r19 M-2 | **Routed; in progress — agent-core 34**: the report stamps the live desired document's refs on an ACK of the applied generation; **production reachability unknown**. |
| r19 L-2 | **Routed; in progress — agent-core 34**: the conntrack attempt is persisted after the delete. |
| r19 H-2, M-1, M-3–M-8, L-1, L-3, L-4 | **Design-only**, recorded as D-141 reconsideration conditions; no implementation or closure claimed. |

## Decisions at #385

**D-140 — orchestrator, provisional, 2026-11-01 07:42 EST.** Isolation's
whole-node D-104 predicate in `tests/e2e/at/isolation/deadline_trace.go` stays
**byte-identical**. Only the **boundary probe schedule** may change: dense
probes timed by measured **member-send → node-ingress latency around D−101 ms**,
with **the schedule and latencies recorded**. A closure interval that still
straddles D−101 ms stays failed. **Run 56d20530 stays failed; no retry waiver**.
Isolation F14-01 is routed under D-140 to **isolation 15**. The
[#383 decision hand-off](../../tests/handover/testdata/history/follow-up-27/isolation14-routing.md)
retains the unchanged positive/drop predicate and populated-hold request.

**D-141 — orchestrator, provisional, 2026-11-01 07:5x EST.** This is the one
D-139 representation decision, taken on r19: **option (b) intersection serving
is not implemented on this evidence; option (c) stays in production**.
**D-139's design track is closed by D-141**. Agent-core 33's
[design note](../node/followup33-intersection-serving-design-2026-11-01.md) and
test-only reference stay on main as history, not an implementation approval.
The [merged #385 hand-off](../../tests/handover/testdata/history/follow-up-27/members15-D141.md)
also records the outcome. Decision timestamps and routing are the orchestrator's
follow-up 27 instructions; the grounds are checked against r19 and merged history.

Grounds: **0 / 80 merged restrictive changes qualify**; the conservative
profile's **promotion baseline is the same order as today's baseline-to-Apply,
173.8 / 195.6 ms p50 / p99**. The representation lacks **a signed intersection
witness, a v2 heartbeat, a partial-service cause, baseline-on-422 and an nft-only
revocation transaction**. Reconsideration needs **a desired-state intersection
witness**, **a measured qualifying share above zero whose promotion costs less
than (c)**, and **r19's design-only findings closed**. The reference's seven
tests do not supply these missing production contracts.

**D-135, D-136, D-137 and D-138 stay provisional and unchanged.** Given D-141,
**AT02's sibling outage on a restrictive generation change rests on the owner's
D-135 / D-136 confirmation or a requirement change**. Every historical failure
stays failed. D-140 and D-141 change **no operator procedure**.

The [findings register](README.md#findings-register-at-the-follow-up-27-cutoff)
retains every other follow-up 26 finding, including the TEST ONLY harness
stop-path gap, updates' historical unobserved exit, members F13-01's unproven
origin, the unexercised populated hold/alarm/escalation, and owner resources.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation**, now carrying
AT02's sibling outage given D-141; every loss counts and there is no finite
outage ceiling.

2026-11-01 — **Owner decision 2: D-137 hardening option**, including the new
failed receive-only dual MTU-1280 row; no upgrade or waiver.

2026-11-01 — **Owner decision 3: D-138 overturn option**.

2026-11-01 — **Owner decision 4: D-140 overturn option**, boundary schedule
only, unchanged predicate and historical failure.

2026-11-01 — **Owner decision 5: D-141 overturn option**, with its witness,
qualifying-share, promotion-cost and design-finding prerequisites.

2026-11-01 — **Owner decision 6: F11-01 repair direction**, still waiting on
**devstack 26**. The new serving-node samples do not establish that cause.

Already assigned, not new requests (2026-11-01): r19 H-1 → netlab 24;
r19 M-2 / L-2 → agent-core 34, all **routed; in progress**. Isolation F14-01 →
isolation 15 under D-140. Serving-node samples → devstack 27; members 16
records member send timing and characterizes the D-137 failure. Devstack's
populated stale-lease fixture then feeds isolation/member assertions. The
TEST ONLY harness stop-path gap and updates wait-receipt adoption remain with
their existing owners. These outside-owned items stop with their owners;
no outside-owned pin needed changing. **Six owner-decision requests; no new
operator guidance, evidence edits or production values.**

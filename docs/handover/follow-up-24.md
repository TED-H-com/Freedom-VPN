# Follow-up 24 — corrected baseline recovery, routed MTU and graceful stop

Fixed `main` cutoff **5bd751026073444087b32ac2a7115f8566104f8b**, through
**#369**, checked for the 2026-11-01 EST handover. This cut includes every PR
**#363–#369**; #363 belongs here even though it merged after the previous
#362 cutoff. Nothing was merged after the cutoff at task start. Grok review
r18 (agent-core 31's design note and shutdown change) and devstack follow-up 24
were running, not merged; no results from either enter this cut.
Handover ran no new live acceptance.

**D-098 / D-109 / D-132:** [seven exact roadmap rows](../../tests/handover/testdata/followup24-main.json)
and [21 exact immutable snapshots](../../tests/handover/testdata/followup24-snapshots.json)
pin the cut. `TestHandover_Followup24Claims` has **21 subtests**.
Snapshots retain dated notes, PR hand-off excerpts, run-scoped
analysis and the previous owned register/entrypoints, with source commit,
full-source digest, byte offset and excerpt digest. Members 13's F13 roots
and usage 7's root records remain living evidence; their results are pinned
by roadmap / PR, never by reading or copying those roots. Archive/run integrity
covers **357 files**: netlab 21 **132**, usage 6 **19**, members 12 **52**, and
devstack 23 **154**. The old claim functions retain their original assertions.
Current owned-document digests in secops scope-review are separate live
integrity checks, not frozen acceptance evidence.

## Security-ops 14 — #363

Agent-core 30 findings 3–4 are fixed in owned test code. The ten PITR events
and the rate-class event take their period from their own interval start via
`usagePeriodForStart`. Offline `TestMTR03_UsagePeriodOverlapsInterval` in
`tests/e2e/at/secops/usage_period_test.go` proves 44 one-second intervals at a
month start, its first seconds, mid-month and the **2026→2027 year boundary**
under ingestion's strict overlap predicate. `restore_test.go` checks replay
bytes and quality **per retained period**, preserving the combined 3,000 bytes.
Product ingestion unchanged; no live suite ran; recorded rows unchanged:
SECOPS-21 closed, AT30 pass, full AT29 / AT31 blocked on owner resources.
See the [#363 hand-off excerpt](../../tests/handover/testdata/history/follow-up-24/secops14-handoff.md).

## Handover 23 — #364

The [previous cut](follow-up-23.md) covers **#357–#362 at bc011b6f**.
`TestHandover_Followup23Claims`: **16 subtests / 6 rows / 18 snapshots /
421 file checks**. It records the month-boundary incident at 2026-11-01
00:00 UTC and the fixed-clock correction: never pair a calendar literal with
the wall clock. It registered D-136 and D-137 as provisional; D-135 stayed
provisional. Its historical #363 exclusion remains true for that cutoff.
This follow-up freezes its live document inputs; a whole-function hash proves
that only those input redirects changed, preserving every earlier assertion.

## Netlab 22 — #365 and D-138

D-098 archives follow-up 21's **132 files** under
`docs/poc/G2-followup21-evidence/archive/`. The opt-in **TEST ONLY**
`WithCommandLifecycleForTest` in `node/enforce/command_lifecycle.go` observes
command lifecycle; `WithApplyCaptureForTest` in `node/enforce/apply_capture.go`
retains synthetic render input/output separately. Both are off unless a test
sets them. Disabled call sites cost one context lookup; no command sampling
or render capture is performed. Go proofs include
`TestD133_CommandLifecycleOutcomes`,
`TestD133_CommandLifecycleDisabledAndDetached` and
`TestD133_AT02_CommandLifecycleJoinsApplyAndRollback`.

The D-133 A,B,A,B comparison spans **two 90-minute windows**. The live quartet
passes. The host never allowed a **qualified second B agent leg**; excluded
B2 attempts remain excluded. The lane's original blocked verdict is retained
in the [dated note](../poc/G2-followup22-2026-10-31.md) and evidence under
`docs/poc/G2-followup22-evidence/` and
`docs/poc/G2-followup22-resume-evidence/`. The orchestrator accepted the
observation under **D-138**, documented separately below, without rewriting
that history. Requests **1–2 → agent-core 31** (TEST ONLY caller and restart
investigation); request 3, the missing qualified B2 leg, is closed by D-138.

The D-135 enforcement assessment confirms fail-closed generation-0 baseline,
no previous-generation restoration and no infallible cleanup. A successful
atomic baseline install is distinct from merely returning an Apply error.
Fresh injected-fault recovery brackets are **312.900556 / 323.540915 /
319.004140 ms**, including a deliberate 200 ms hold: **185 admitted losses,
167 outside guards**, and **zero denied-source egress**. These are measured
fault/hold/retry windows, not an outage ceiling or a historical-overrun cause.

## Usage AT 7 — #366

D-098 archives run **71b1a160**, **19 files**, under
`docs/e2e/at/usage/history/follow-up-6/71b1a160/`. AT15 derives each live
record's period from its own interval start. AT17 takes the daily request and
half-open month bounds from the **summary's returned period**. Usage AT20's
held period comes from the **immutable snapshot**, reused for comparison
queries; this is the ENR-05 usage extension, not §19's DNS AT20.
`TestAT15AT17AT20_LivePeriodsAtUTCBoundaries` proves month starts, first seconds,
offsets crossing a month, mid-month and the year boundary offline, with negative
summary/held-period tests. Product period validation is unchanged.

November run **b8a6e9e1: AT14–AT20 pass**. Summary/daily totals agree at
**31,103 bytes**, held traffic is **41,417 bytes**, fresh **2,580 bytes** is
attributed once. These are November live results, not a live midnight experiment;
boundary instants are the offline proof. **0 requests.**

## Devstack 23 — #367

The D-137 **routed 1280 hop** is exported by `PrepareNativeRouter1280` and
`NativeRouter1280.Select` / `StartCapture` in `tests/e2e/native_underlay.go`.
A separate router's member-facing link is 1280; sender links stay 1500 and peer
routes initially have no configured MTU. **IPv6 passes by PTB with learned
PMTU 1280** (one matched PTB, all 84 fragments arrive). **IPv4 passes by DF=0
fragmentation in transit** (34 oversized outer packets become 68 fragments),
with no frag-needed or learned IPv4 PMTU claimed. Both bounded transfers deliver
**64,064 bytes in 1 ms**. The receive-only no-PTB rows remain limitations;
historical F8-02 rows stay failed. Members adoption is pending.

**F11-01 traced to the pinned NetBird 0.82.0 client's loopback proxy**:
retained-relay `RedirectAs` injects the IPv6 initiation to `::1`; the matching
`::1`-source handshake response reaches the member underlay and no type-4 data
follows during the failed request. The packet/log/source trace identifies the
path; kernel drop and handshake acceptance are not directly instrumented.
Devstack 24 is assigned to locate the drop; no fix or result is imported here.

A **positive IDNA** A-label fixture passes over native UDP and TCP.
**F12-04 not reproduced**: the fresh query succeeds with joined delivery
segments, leaving the historical SERVFAIL unattributed. Members must instrument
its original query next. The retained singular tcpdump-statistics parser failure
is supplemented by a same-capture readback, never rewritten as a historical pass.
Native run **a26d9b0f: 14 / 0 / 3**; DNS run **9bb2098e: 11 / 0 / 1**.
Evidence: `docs/e2e/runs/2026-11-01/followup23/`, pinned by
`docs/e2e/followup23-captures.sha256`. **3 requests:** members' hop adoption,
original F12-04 query instrumentation, and the NetBird proxy trace.

## Members 13 — #368

D-098 archives **b98b78ee**, **52 files**, under
`docs/e2e/at/members/history/follow-up-12/`. Assembled run **4182dd3d**:
**2 pass (AT03, AT08) / 6 blocked / 2 fail**, **616 / 7 / 3 subchecks**.
Its product baseline precedes #367's routed hop and #369's caller/stop changes.

- **AT02 fails** the provisional D-135 / D-136 baseline-transition check.
  All **61 baseline calls** pair to the next successful Apply; only three
  overlap the sibling capture, with sibling coverage unavailable for the other
  58. **21 sibling losses counted** (13 inside guards, eight outside); the
  first bracket is **219.395 ms with nine losses**. Repeated overlapping
  brackets are not summed as unique losses. **95 Applies succeeded, none
  failed**, so **F11-03 stays blocked**; measured recovery does not pass the
  no-loss row.
- **AT04 fails on new F13-01:** two persistent-withdrawal counter reads are
  rejected by the parser (`nft_parse` / `parse`) with **command exit 0**.
  Raw failed bytes and parser errors were not retained in this run; **cause
  unproven**. Devstack 24 is assigned to retain both; a parser fix is conditional
  on evidence of a defect, not presumed. Both failed rows stay failed.
- **F12-02's streamed reader live-qualified:** 22 readiness and 12 monitor
  checks pass, reaching 2,455,378 bytes. The historical eleven readiness
  failures and failed monitor stay failed. **F12-03's quiet window passes**;
  its historical row stays failed and the old cause remains unproven.
- **AT09 under provisional D-137:** known-PMTU 1280 passes, **64,064 bytes in
  1 ms**; receive-only rows are limitation rows, never PASS, even when smaller
  reverse packets allow the fresh transfers. Reactive PTB was absent from this
  tested baseline; #367 supplies the hop for later adoption.
- **F11-01's first selection succeeds with one SYN**: a second data point,
  no cause or repair inferred. It does not erase #367's five-SYN reproduction.

**10 requests** in the #368 hand-off. The immutable
[routing excerpt](../../tests/handover/testdata/history/follow-up-24/members13-routing.md)
records #367's delivered hop/IDNA, members' next query instrumentation and
devstack 24's assigned F11-01 / F13-01 work. Those are assignments, not results.

## Agent-core 31 — #369

[Dated design and measurement note](../node/followup31-generation-change-availability-2026-11-01.md).
**D-136 confirmed with one correction:** the **verified deny-all candidate is
used only when an Apply error is retained**. The ordinary stale-lease transition
installs the **node-wide baseline and ACKs, renews, then Applies, verifies and
heartbeats**. A generation mismatch alone does not enter candidate recovery;
not every generation change necessarily exhibits the window. **No finite
outage ceiling**; D-136's fail-closed reading and D-135's bracket rule remain.
`TestF31_StaleGenerationBaselinePrecedesRenewalWithoutCandidateApply` proves the
ordinary branch. Independent egress and later verification/path/heartbeat
remain distinct from Apply-end and kernel guard removal.

**80 desired-first generation changes**, two separate 40-transition populations,
retain full p50 / p90 / p99 / max distributions and every sibling loss. Baseline
start → Apply end in milliseconds:

| Population | p50 | p90 | p99 | max | Sibling losses (inside / outside guards) |
|---|---:|---:|---:|---:|---|
| First, not quiet-qualified | 177.901491 | 192.802074 | 201.859702 | 201.859702 | 512 (76 / 436) |
| Final, quiet-qualified | 173.766526 | 187.813686 | 195.601257 | 195.601257 | 501 (66 / 435) |

All **1,013 losses** remain recorded. The note retains every other endpoint's
full distribution. These populations are not A/B, not pooled into a bound, and
claim no availability ceiling or uninterrupted-sibling pass.

Unchanged main's `TestLab04_RestartKeepsVerifiedLease` **failed 5 of 30** with
node/member leases about **3,599 s from expiry**, not expired. Deterministic
regressions prove that caller cancellation after SIGTERM manufactured
Apply/member-path failures and could poison local storage reads after the live
read. `Engine.Step` now stops at the safe boundary, then finishes **bounded
local work, at most 10 s**, while **control I/O stays cancellable**. Real Apply/tc
failures still fail closed. The lab restart separately requires a **recorded
graceful exit**; a timeout fails, with no forced kill or new grant. The corrected
30-repeat population passes 30/30; the subsequent full suite records a
40.839979 ms graceful stop. Historical failed rows stay failed, and these fresh
proofs do not identify the exact cancellation point of each old A1/R-A2 failure.

Pins: `TestF31_AT13_CancellationAfterLiveReadKeepsRestartAuthorization`,
`TestF31_AT13_CancellationDuringLocalWorkKeepsRestartAuthorization`,
`TestF31_AT13_CancellationDoesNotMaskRealApplyFailure`, and
`TestLab04_RestartKeepsVerifiedLease`. The TEST ONLY lifecycle caller requires
both **TestMode and TestTimingPath**, proved by
`TestF31_CommandLifecycleRequiresBothTimingGates`.

The design note recommends **intersection serving for r18 review only**;
generation-serving behaviour is unchanged. Revocation-first commit and honest
provenance/ACK semantics remain design questions. No live timing threshold
failed, so **no D-133 run** was triggered. The shutdown repair is functional
regression evidence, not a performance attribution or new outage guarantee.

## Provisional decisions and findings

**D-138 (03:0x EST, 2026-11-01, orchestrator — provisional)** refines how a
D-133 comparison is judged; **D-133 itself is unchanged**. Netlab 22's opt-in
TEST ONLY observation is accepted without the qualified second B agent leg:
disabled, one context lookup per call site; no B-only failure in either window;
the only selected suite failures were unchanged main's TestLab04 (the mechanisms
and fix are explained by #369, without retrospective per-run causality).
Install-command failures: **4 on A and 4 on B**. B legs interleave with A across
windows. The **two-leg envelope rule is withdrawn as underpowered**. A D-133
comparison **pools every qualified leg per arm across windows**; a shift is
attributable **only when it repeats in direction in each window and the code
can explain it**. Excluded legs stay excluded and historical failures stay
failed. The owner may overturn this provisional acceptance.

**D-136 correction (04:3x EST, 2026-11-01, from agent-core 31, checked against
code)** is the retained-error condition and ordinary baseline→ACK→renewal→Apply→
verification→heartbeat sequence above, with **no finite outage ceiling**.
D-135's bracket rule remains: baseline → next successful Apply, every loss
counted, recovery required, never waived. **D-135, D-136, D-137 and D-138 stay
provisional.** D-137 now has #367's routed-hop scope evidence: IPv6 PTB / learned
1280 PMTU and IPv4 DF=0 transit fragmentation, not complete real-platform
qualification or a global MTU/MSS change.

The [findings register](README.md#findings-register-at-the-follow-up-24-cutoff)
adds **F13-01** and the unchanged-main **TestLab04 cause/fix**. F11-01 is traced
by #367, with #368's one-SYN success and devstack 24 locating the drop next.
**F12-01** must read **“policy_missing baseline during generation 11→13;
stale-lease path consistent with code; no failed Apply”**; historical
**cause_status=unproven** stays unchanged because the desired input/branch
observation is missing. F12-02 is live-qualified; F12-03's quiet window passes;
F12-04 is not reproduced; F8-02's routed hop is delivered, members adoption
pending. Every historical failure stays failed.

Unchanged: install-overrun cause **unproven**, **F9-INFLIGHT**, **F18-01 intended
negative**, **ISO-F6-03 writer unattributed**, **Dependabot #22 open for owner
disposition**, and production **blocked_response_ttl owner value**. No new
production value, exception, tolerance or full §19 acceptance is inferred.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: confirm D-135 + D-136 together**, including
the corrected retained-error condition, all baseline brackets and no finite
outage ceiling. Stronger availability remains separate design work.

2026-11-01 — **Owner decision 2: D-137 hardening option**: confirm the supported
routed/known-PMTU scope and choose whether lower tunnel MTU or an MSS clamp is
wanted. No global clamp is introduced by this cut.

2026-11-01 — **Owner decision 3: D-138 overturn option**: confirm or overturn
provisional observation acceptance and the pooled qualified-leg assessment;
no missing leg or historical failure is converted into a pass.

Already assigned, not new requests (2026-11-01): **devstack 24** retains F13-01
raw bytes/parser error in `tests/e2e/internal/counterread/read.go` and locates
F11-01's drop; `node/enforce/parse_nft.go` changes only if evidence shows a parser
defect. **Members next follow-up** adopts the #367 router export and instruments
its original F12-04 query in `tests/e2e/at/members/` using `StartNativeDNSCapture`.
**r18** reviews the #369 note and shutdown change; intersection serving has no
implementation authorization here. These outside-owned items stop with their
owners; the handover delivery is complete. Existing real-platform/QUIC, arm64,
AT11/AT13 and AT29/AT31 resource gaps remain. No outside-owned pin needed changing.

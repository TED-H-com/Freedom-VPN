# F27-01 — one TCP reload diagnostic, 2026-11-03

**No TCP failure was reproduced. Historical follow-up 27 B1 remains FAIL and
unexplained.** The one predeclared window completed 100 reloads / 101 verified
Applies with 118 successful TCP reads and no UDP loss. D-120 passed unchanged.
There was no retry, second diagnostic, A/B, production fix or predicate change.

The new observations show repeated request-data loss at node forwarding and
successful recovery, including retransmissions that themselves encounter later
guards. They establish a candidate mechanism, not the missing history of B1.
[PR #405's orchestrator disposition](https://github.com/TED-H-com/Freedom-VPN/pull/405)
and the roadmap's historical statuses are untouched.

The [declaration](declaration.md) was committed in `ef58364c`; the observed
harness was committed in `b398b5a0`, on task-start main
`8a775315f232c9142d8a884b5be5d7d7b88addb0`. Production enforcement and agent
sources are unchanged. [The claim](window/claim.json) pins the declaration,
binary, source and single-attempt test before execution.

## What the packets show

| Observation | Result |
|---|---|
| Captured endpoint emissions | 375 TCP segments, including handshake and capture tail |
| Captured origin / node / responder observations | 375 / 739 / 364; zero capture drops; parsed counts match |
| TCP failures | 0 / 118 recorded reads; longest read 827.017314 ms |
| Retransmissions | 11 request-data retransmissions across 7 reads |
| Missing forwarding stages | 11 request-data emissions: 7 originals and 4 retries; node ingress present, node egress/responder ingress absent |
| Conntrack around every swap | 200 nonempty snapshots: same ID, ESTABLISHED; two initial generation-1000 snapshots empty before connection creation |
| Distinct segments with a node observation inside an outer guard bracket | 21 |
| Segments whose observed traversal straddles a guard command boundary | 3: segments 122, 156 and 255 |
| Retransmitted SYN / SYN-ACK / reply / FIN / pure ACK | 0 |
| Missing/ambiguous correlation | 0 unmatched observations; no duplicate-stage/identity ambiguity |
| Maximum outer guard bound | 51.504545 ms; this is not a kernel commit timestamp |
| Maximum Apply wall/monotonic elapsed disagreement | 368 ns |

The 21-overlap count means a node timestamp falls in install-start through
release-end, with the retained 2 µs margin. The three straddling segments cross
an install/release *command boundary* between their first and last observations;
two never egressed the node. Neither count asserts forwarding through a known
active guard. Every segment's flags, sequence, ACK, length, timestamps and
install/release/verify classifications are in the
[complete packet analysis](tcp-analysis.json.gz). The
[complete retransmission ledger](retransmissions.md) classifies all 11 retries,
and the [summary](tcp-summary.json) retains every socket RTO sample and missing
stage. The independent [D-120 result](window/reload/TestAT10_AT13_ReloadRevokeBaseline/at10-analysis.json)
uses only its original inputs.

Application cycle 40 is the strongest example. Its original data segment 122
and retries 123/124 are seen at origin egress and node member ingress, but never
at node egress or responder ingress. Those ingresses overlap generation 1031's
release and generations 1032/1033's installs. Retry 125 traverses all four
request stages outside those brackets; the reply reaches the origin. The
retry delays from the original emission are 210.037466, 417.157518 and
826.029367 ms; the read completes after 827.017314 ms. Socket state moves from
RTO 201 ms to RTO 402 ms/backoff 1, then back to 201 ms on recovery. Conntrack
stays ESTABLISHED with the same ID before and after every affected swap.

The unchanged guard rules in `node/enforce/deadline_install.go` drop both member
forwarding directions. Apply atomically installs that guard with the candidate,
checks the installation budget, deletes the guard, then verifies. Missing node
egress during these brackets, repeated on later retries with stable conntrack,
is consistent with the guard suppressing data and retransmissions. It is not a
packet-level nft verdict trace: exact drop-rule attribution and transient state
between conntrack snapshots are not directly observed.

The first retry's unchanged RTO and later doubled RTO are consistent with a
loss probe followed by retransmission backoff. Upstream Linux v7.0 implements
[loss-probe scheduling](https://raw.githubusercontent.com/torvalds/linux/v7.0/net/ipv4/tcp_output.c)
and [retransmission backoff](https://raw.githubusercontent.com/torvalds/linux/v7.0/net/ipv4/tcp_timer.c).
**Counterfactual only:** if the 826 ms retry also disappeared and this backoff
continued to 804 ms, the next retry would be around 1.63 s, beyond the read's
1.5 s deadline. That fourth loss did not occur here. No kernel timer-event trace
or exact distribution-source attestation identifies every timer, and this
calculation cannot explain B1 retroactively.

B1's sequence 43 was on the original persistent connected socket: `stream.py`
calls connect once before the loop and never reconnects. A new SYN exchange is
therefore not the explanation for that recorded read timeout. The original UDP
captures cannot distinguish B1's request loss, reply loss, accumulated RTO or
late-read behavior. This run supplies no failing read with which to make that
distinction. No conntrack reset or socket-reuse artifact was demonstrated.

## Admission and measurement limits

The [quiet-host receipt](window/quiet.json) retains **15 preflights**, **14
rejections**, and **1292.420680 seconds (21.54 minutes)** of waiting. The accepted
sample had r = 0, 0, 1, 1, 0, 0 and idle = 90, 99, 93, 94, 98, 99 percent.
The since-boot row was ignored. Both holds printed `free`, no actual other-lane
suite/test existed at either sample boundary, and the
[immediate prestart check](window/immediate.json) was clear. Shell text containing
search strings was not counted as a suite. The 90-minute deadline was not used
as a waiver.

**The whole interval cannot be called uncontended.** The recorded TCP read
window was 07:55:10.109906–07:55:36.432425 UTC. The
[post-run check](window/after.json) found a new agent unit-test process;
[its process start time](window/post-process-timing.json) is 07:55:35 UTC,
at one-second resolution. It overlaps the final roughly 1.4 seconds, including
cycle 116's recovery. Prestart admission passed, but another lane could start
later; no exclusive reservation existed. Keep that limitation with the result.
No retry or quietness waiver was applied.

Read-only observation also changes experiment cadence. The 202 before/after
snapshots took **5.780168 seconds** in aggregate outside Apply. The initial empty
socket read took 452.61 ms before the probe started; subsequent per-snapshot
conntrack-plus-socket reads were at most 27.76 ms. They do not lengthen the guard
internally, but can change which emissions encounter a reload. This is a
measurement of the unchanged production path under added observation, not a
source-matched performance comparison or a production D-133 cost claim.

Physical arm64 and other kernels/networks remain unmeasured; arm64 cross-build
is a separate gate. Capture tails are retained, but only the original sidecar
boundaries define D-120 reads. The [method](method.md) describes parsing,
clock conversion, snapshot placement, privacy and reproducibility.

## Proposed disposition — no implementation

Keep F27-01 open for the missing failure mechanism. Repeated install/release
loss plus TCP backoff is now an evidence-backed candidate owned by **netlab's
render/install path**, but no 1.5 s failure was captured. A clean run cannot
close B1 or turn its failed leg into PASS. There is no evidence assigning this
case to agent behavior, a conntrack reset, or a harness artifact.

No production fix is selected from this dataset. Any future guard optimization
must retain the late-commit/crash fail-closed property and receive a predeclared
D-133 comparison; removing the guard or admitting established traffic through
it is not a supported fix. No cost or effectiveness for an unimplemented fix
has been measured. Agent scheduling changes, if later warranted, belong to
agent-core and need a separate hand-off.

Keep D-120 clause (iv), its 1.5 s timeout, and every historical status unchanged.
No predicate relaxation is proposed. A future diagnostic can add kernel timer
identity or exact drop-verdict observation, and explicit framed response
attribution if a late read occurs; neither is grounds to alter the current
probe or verdict. A further live window needs a new declaration and host
coordination covering the entire interval, not an unreported retry here.

D-098: follow-up 25/26/27 have 163/165/446 payload files matching complete
original manifests and gzip originals at the [initial check](d098-initial.json).
The [completion check](d098-final.json) repeats the same complete manifest checks. New evidence is confined to this
directory; no prior file or historical result was rewritten.

## Required full-lab gate

The separate full-lab gate is **FAIL**, not a replacement diagnostic window.
AT10 passed D-120; D-104 passed with 26 positives / 7 drops and last positive
D−100.681117 ms. The unchanged AT12 deny-phase usage assertion failed at
`tests/netlab/scenario_at12_test.go:77`: 16,243 → 16,340 upload bytes, a 97-byte
delta. All 17 deny-phase cases met their expected outcomes, target captures were
empty, and denied service accept logs did not grow before that assertion.
The retained [failure record](at12-failure.json) and [full gate output](gate2.txt)
identify the failure without assigning those 97 bytes to a particular flow.
No source change, retry, packet-level cause claim or waiver was made. TCP
observation was disabled for this gate. Container/network/scratch teardown
completed, and [the owned image was removed](gate2-remove-image.json).

This failure prevents an all-green delivery. It requires a separate AT12
measurement/fixture investigation; it does not alter the diagnostic's D-120
result or historical B1. Required consumer and remaining gate receipts are
retained alongside this evidence.

The final offline Python suite passed **42 tests**. A diagnostic-only regression
now treats a read with missing size as failed instead of inferring success,
and retains its packet/guard context. Successful reads also require a captured
handshake and unique return-byte coverage at the origin; empty/partial captures
and duplicate retransmitted bytes cannot manufacture that coverage. This
changed no live observer or D-120
code. The [analysis replay](analysis-replay.json) is byte-identical to the
original result on the retained valid inputs; [the final unit receipt](final-python-coverage.json)
retains the final result.

Final cleanup verification waited **2762.116 seconds
(46.04 minutes)** behind the members host hold, with
46 retained blocked admission checks at 60-second intervals. Once free,
the [inventory](cleanup.json) confirmed zero owned containers, images, networks
and volumes; task binaries/wrappers/scratch were removed. The
[completion record](completion.json) distinguishes this wait from diagnostic
admission. D-098 completion again passed all 774 historical payload files and
three original manifests, including gzip originals.

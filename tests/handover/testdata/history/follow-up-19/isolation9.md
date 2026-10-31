# Isolation follow-up 9 — 2026-10-31

One assembled run: **`2be2797b-7a28-4e9f-a1fe-5fffa4c1b047`**, live source
`e496d826c04054bd718186de078b8f972353fb94`, product baseline `fb5e0025`
(includes netlab 19, agent-core 27 and devstack 19). Source dirty: **false**;
the product tree is unchanged. **14:54:34–15:10:27 UTC, 953.07 seconds**.
The gate returned **2 (lab=0, stack=1)**. This run is retained, including the
new AT13 failure; no second run substitutes for it.

| Case | Result: pass / blocked / fail | Deciding evidence / next owner |
|---|---|---|
| AT10 / ISO01, ISO02 | **blocked: 8 / 1 / 0** | [AT10](AT10.json). D-087 blocked on wire-nodes 33's central grant-eligibility seam, then devstack 20's hook. |
| AT11 / ISO01, ISO03 | **blocked: 2 / 1 / 0** | [AT11](AT11.json). Native IPv6/member-platform matrix: devstack/operators. |
| AT12 / ISO01, ISO04, DNS03 | **pass: 10 / 0 / 0** | [AT12](AT12.json). Counted peer, management, DNS, spoof and group-withdrawal controls. |
| AT13 / ISO02, ENT02, SEC01 | **fail: 5 / 1 / 1** | [AT13](AT13.json). Whole-node expiry has early drops followed by later positives; full fault/platform matrix separately blocked. |
| ISO-F6-01 | **pass: six assertions** | [Recovery verdict](recovery-verdict.json), [raw sequence](foreign-redirect.json), [durable Apply state](recovery-applied-state.json). |
| ISO-F6-02 / D-106 | **pass: 557 / 557 digests** | [Wire read index](tc-evidence-index.json); all also match retained producer verifier digests. |
| ISO-F6-03 renewal | **3 losses / 1,396 probes**, all three ingress reads inside install brackets | [Exact correlation](continuity-analysis.json). Finding is guard duration; no tolerance change and no loss-free claim. |
| ISO-F6-03 expiry / D-104 | **fail** | [Packet verdict](deadline-verdict.json), [failure index](expiry-failure-correlation.json). Agent-core serving/heartbeat/watchdog interaction; baseline writer unproved. |

Spec: §19 AT10–AT13, §8.2–8.3, §16.3. Real arm64 and the complete platform
matrix remain 待實測. Helper cross-builds cover linux/amd64 and linux/arm64.
Mechanism results remain separate from assembled acceptance.

## D-098 first; historical evidence unchanged

Follow-up 8 run `7779d83c` is archived **byte-identically, 80/80 SHA-256 pins**,
including root files, support records and reload-stream inputs, in
[history/follow-up-8](history/follow-up-8/archive.json). Its deadline, recovery
and continuity pins exactly match netlab 20's requested digests. Follow-up 6
`bf671f3a` (**44/44**) and follow-up 7 `041e9e48` (**52/52**) are untouched.
Handover already reads archives/immutable fixtures. Its living AT10 path-string
rejection test does not read that file. **No handover change or pin move was
needed; the one-file exception was unused.**

The historical **17 losses among 1,396 probes** lack exact producer timestamps.
They remain in the pinned [follow-up 8 continuity record](history/follow-up-8/continuity-analysis.json)
and are not retrospectively assigned the new run's timing.

## Exact producer timing and renewal

[Original source JSONL](producer-operations.jsonl) retains **3,289 snapshots**
across three process runs: 810 Applies, 2,436 standalone verifications and 43
renewal transactions. **41 failed operations** remain present. The successful
quarantine recovery has both verifier records (process `0b191fdd`, sequence 1031).
There are 3,240 verifier invocations in total. Sequences are contiguous per run;
all nanosecond fields remain decimal strings. The [full analysis](producer-timing-analysis.json)
validates order, flags, clocks and matching D-106 digests.

| Producer bracket, entire source trace | Samples | Median ms | Worst ms |
|---|---:|---:|---:|
| Apply | 810 | 78.578258 | 156.946129 |
| Final render → installed-by | 803 | 16.115437 | 30.980245 |
| Install command | 803 | 15.966763 | 30.823930 |
| Release command | 803 | 14.709242 | 30.966682 |
| Outer guard | 803 | 30.901338 | 52.596424 |
| Complete verifier | 3,240 | 23.331450 | 74.176367 |
| nft ruleset read | 3,240 | 2.293239 | 8.888025 |
| Millisecond precision text read | 3,207 | 1.471820 | 19.565508 |
| Aggregate tc read | 3,240 | 16.379201 | 66.671903 |

Partial/failed calls remain in the source trace; only completed install/release
commands establish an outer guard. Total Apply duration is not D-104's
render-to-install budget. The inner definitely-guarded interval has median
0.006019 ms and maximum 0.026280 ms; kernel commits occur somewhere inside
the surrounding command brackets.

The independent **80 ms dual-phase diagnostic** captured two real signed
renewals (producer sequences 780 and 913), **70 Applies / 70 complete guards**,
**1,396 probes / 1,393 echoes / three losses (99.7851% echo availability)**.
Every probe has ingress; all three losses lack uplink egress. AF_PACKET drops=0.
All three send→completion windows and captured ingress reads overlap the exact
install command, and all three overlap the outer guard:

| Lost packet | Apply sequence in process `0b191fdd` | Install ms | Release ms | Outer guard ms |
|---|---:|---:|---:|---:|
| 54 | 660 | 12.881333 | 24.789611 | 37.676074 |
| 294 | 708 | 20.232754 | 15.855947 | 36.090627 |
| 1315 | 914 | 23.129270 | 27.064374 | 50.196105 |

The per-loss records retain exact start/end samples, final render, both command
flags, verifier and executor read brackets, digest and lease. These are
command/capture correlations, not exact kernel drop instants. **Each loss remains
a loss; D-120 is not applied.** The measured finding is the guard size, with no
new tolerance or renewed request for an already-delivered timing surface.

The initial offline clock check rejected differing namespace IDs. Raw evidence
showed the same boot and exactly zero monotonic/boottime offsets across the
containers. A read-only [live agent `/proc` snapshot](producer-clock-after-restart.json)
confirmed that the source agent and node packet recorder share the post-restart
namespace. The member sender has a separate namespace with the same exact zero
clock origin. The analyzer now validates that basis and rejects offset/boot
mismatches; it adds no allowance. The same read is integrated for future runs.
This collector correction did not replace the running product or probe binary.
See the offset contract in [Linux time_namespaces(7)](https://man7.org/linux/man-pages/man7/time_namespaces.7.html).

## Whole-node expiry failure outside guard brackets

Generation-30 lease `bfeb6b9b-8fa6-47ed-85cd-9a4034d6f75e` was issued at
15:07:17.994901292Z, expiring 15:08:17.994901292Z. Its retained request anchor
is `42775508568707` ns, so **D=`42835508568707` ns**. The producer and node-held
deadline agree; no new lease was accepted during the expiry capture.

All **202 probes** have independent ingress/read evidence; the packet capture
reports zero drops. There are **65 positives and 137 drops**:

- **96 early drops**, packet IDs 1–96. First ingress is **D−8,098.843780 ms**;
  forwarding resumes at **D−3,298.889928 ms**. **95** are outside every completed
  guard; packet 96 overlaps the recovering install.
- **65 subsequent positives**. Last positive egress is **D−99.972452 ms**,
  satisfying the unchanged D−101 ms bound by 1.027548 ms.
- **41 final expiry drops**, beginning **D−62.863882 ms**. There is no positive
  at/after D. Nevertheless, the overall predicate **fails** because positives
  occur after the earlier drops. The early interval is not removed or forgiven.

The [focused failure index](expiry-failure-correlation.json) and
[all loss windows](expiry-timing-analysis.json) retain the deciding sequence:

1. Apply 955 succeeds at 15:08:01.179209101Z, followed by verified live reads 956
   and 957. The last of these reads ends at 15:08:04.325685104Z.
2. Live verifier 958 fails at 15:08:04.337377684Z, followed by blocked audit 959.
   Real report `01a19280-e9a9-7747-887f-3ba2aee9176e` carries
   **GENERATION_MISMATCH** and `safety_check_failed` while its accepted lease
   remains valid. The failed verifier skips the precision text stage; its zero
   samples are preserved, not called a successful read.
3. The first periodic kernel read at 15:08:08.230764210–15:08:08.259486099Z sees
   **generation-0 baseline, empty allowed_v4, serving 0**. Its heartbeat remains
   generation 30 with `safety_check_failed`.
4. Apply 961 installs generation 30 again: exact install
   `42832131549058`–`42832146531158` ns, release
   `42832146540788`–`42832160422196` ns. Verified Apply/live reads follow;
   packet 97 forwards. The lease and its original D are unchanged.

**Next owner: w2-agent-core**, specifically `node/agent/engine.go`
`Step` / `finishServingStep` / `verifyCurrentServing` and
`node/watchdog/watch.go` `Watch.needsBaseline`, with netlab supporting the
verifier diagnostics. Reconcile the control-loss I/O, heartbeat and concurrent
baseline sequence. Watchdog participation is an inference: its log is empty,
and Engine timing does not record an external `ApplyBaseline` writer or its
command bracket. No actor is invented from baseline presence alone. Those
product paths are outside this lane; that item stops at the evidence hand-off.

## Recovery, D-106 and D-120

ISO-F6-01 passes all six assertions. Serving generation 30 → labelled foreign
tc injection → dirty generation 31 remains rejected → complete removal → clean
generation 32, matching populated lease, verified Apply/live read and counted
recovery. Injection→contained observation ≤**444.013107 ms**; removal→link-up
≤**657.675093 ms**, serving ≤**8.505691 s**, counted completion ≤**17.667501 s**.
Admission bytes increase **175,420 → 178,960**. These are observation upper
bounds including polling/probe time, not product guarantees. No restart,
route/address repair, link raise or D-100 override assists this recovery.

Node `any`: **2,386 parsed/reported packets, zero drops**. Two unsnatted redirected
UDP/9 packets and one ordinary SNAT packet precede confirmed quarantine; no
uplink UDP/9 packet occurs from confirmed down until removal. All 14 foreign-tc
rejected reads belong to the labelled harness injection. No unsolicited rejected
tc object appears. The original follow-up 6 creator remains unknown.

D-106: **557/557** wire digests recompute, and all match a retained producer
digest. Historical rejected evidence remains separate from current objects.
The [receiver](receiver-capture.json) retains 562 observation POSTs: 549 HTTP 202,
seven 502 and six 429 responses. D-087 has zero qualifying accepted reports,
and its recipe was **not executed** because the real central seam is missing.
No new duration replaces historical **4,767 / 5,021 / 5,025 ms**.

The separate AT10 stream prints **PASS under unchanged D-120**: 100 reloads,
101 verified Applies, 50 datagrams, **one UDP loss / zero TCP timeouts**.
Packet **12**, generation **1020**, is at the guard boundary; the next datagram
succeeds and all conditions (i)–(v) pass. Its outer command bracket is
`1793458402431275719`–`1793458402456886806` wall ns (**25.610760 ms**), with
render→installed **9.737676 ms**. Maximum guard and render→installed across
that stream are **50.129072 / 29.194891 ms**. The complete
[per-loss verdict](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt)
and analyzer inputs are retained. D-120 is never widened to cover the renewal
or expiry diagnostic.

[D-093 reconciliation](d093-reconciliation.md): all 15 classifications agree;
13 open and two retired within portal simulator scope. The clone/freeze wire
fixture is retained without expanding its acceptance claim.

## Gates and cleanup

Owned lint/test/build, handover, root go-lint, gofmt, diff and committed-evidence
hygiene pass; [actual output tails](validation-results.json). The assembled gate
returns 2 for the measured failed/blocked states, not only blocked scope.
Mechanism suites passed in 22.488 s and 19.399 s. Five text captures had only
terminal blank lines removed; D-120 analysis recomputed byte-identically afterward.
After teardown: **containers=0, images=0, networks=0, volumes=0** under the owned
prefix; [inventory](cleanup-inventory.json). [Five routed requests](requests.md#follow-up-9-requests--2026-10-31).

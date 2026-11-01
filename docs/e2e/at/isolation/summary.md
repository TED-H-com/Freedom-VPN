# Isolation follow-up 13 — 2026-11-01

One assembled run **`38074149-1137-4782-80fd-8669c36c6fcb`**, runtime source
**`c521cb0018844eac1de24ee243c858518bb1a5e3`**, source dirty **false**.
Product baseline **`334d56e466c7689fdbf16e078041afbfbd9bf4fb`**, including
agent-core 32 (#376). Driver **1079.13 s**; full foreground command
**1216.858 s**, exit **2** because operator rows remain blocked.
This is the next independent sample after `1c25ee44`, not an A/B comparison.

| Case | Result: pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **pass: 9 / 0 / 0** |
| [AT11](AT11.json) | **blocked: 2 / 1 / 0** |
| [AT12](AT12.json) | **pass: 10 / 0 / 0** |
| [AT13](AT13.json) | **blocked: 6 / 1 / 0** |

Spec §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04, ENT02, SEC01,
DNS03. AT11's native platform matrix and AT13's full continuous fault matrix
remain [carried operator rows](requests.md#follow-up-13-requests--2026-11-01).
No production code or acceptance requirement changed in this lane.

## D-098, baseline and evidence identity

First commit **`822d56b0`** archives all **108** root files from run `1c25ee44`,
byte-identical to the baseline, in [history/follow-up-12](history/follow-up-12/archive.json).
The new manifest pins the compressed producer and its decompressed bytes.
Earlier archives are untouched. Handover reads immutable snapshots; mutable
root path strings are rejection inputs. No D-132 one-file exception edit was
needed. Historical `1c25ee44` remains **blocked on assertion 6**.

[Preflight](host-preflight.json) retains `vmstat 5 7` and the sanitized suite
inventory. Six intervals: runnable **0–5**, idle **76–84%**, I/O wait **12–17%**;
no other suite tree was present before admission. The runner rechecked before
each stack and after startup. This is not a quiet-host qualification or causal
explanation. [RUN.md](RUN.md) records the exact baseline and `go env GOCACHE`.
The runtime remained on this baseline while main advanced independently.

## ISO-F6-01: all six assertions pass in this sample

The [readiness boundary](control-observation-readiness.json) took
**4.592933845 s** within the existing 90-second restoration budget: an accepted
current generation-31 serving report, then a read-only outbox count of **0**.
No timeout was increased. Injection began at **11:22:51.990023802 UTC**;
removal began at **11:22:54.035207731 UTC**. All **three** dirty kernel samples
retained the foreign flower/mirred objects, link down and `serving 0`.

Two actual HTTP **202** receipts identify dirty candidate **32**, with applied
ACK **31**, forwarding **false**, and observation times
**11:22:53.710206071** and **11:22:53.779398728 UTC**. Their accepted times are
**11:22:53.727447807** and **11:22:53.791570583 UTC**, both before removal.
Each has `stage=apply_candidate`, code `TC_FOREIGN_FILTER`,
`candidate_generation=32 verified=false`, and original failure time
**11:22:53.664950190 UTC**.

[Original-command join](candidate-rejection-join.json): receiver transactions
**1498/1499**, desired transaction **1496**, producer process
`7a5c4424-c702-4c90-ba3a-e207319e33ff`, sequence **2329**, commands **0–36**.
This is a failed **pre-install** candidate; its installed verifier is unavailable.
It is used only for the candidate-rejection requirement. The retained rejected
read/quarantine and the later clean Apply/live-verifier/digest/topology/packet
requirements keep their separate evidence. All six original assertion names
and requirements remain in [recovery-verdict.json](recovery-verdict.json).
The harness removed the filter only after the accepted boundary; the maximum
12 samples at 500 ms was unchanged. No restart or topology repair assisted recovery.

The observation-key defect was independently reproduced and fixed by agent-core
32; this run does not attribute its introduction to #369. No local failure or
later recovery substitutes for a central receipt. The old missing report is
not retrospectively supplied by this run.

## D-087, D-120, D-104 and renewal

D-087 passes on the exported lease-only seam, fixed generation **15**, with
byte-equal invariants, natural empty/populated signed leases, accepted reports
and independent counted forwarding **2578→5208 bytes**. True→accepted serving
is **760.594944 ms**. [Wire audit](member-grants-audit.json): **270** original
hashes and **34** desired responses. No generation forcing, fabricated report
or lease, projector stop or transport substitute drove this transition.

[D-120](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt)
passes unchanged: **101** verified Applies, **62** datagrams, **0 UDP losses**,
**0 TCP timeouts**. It does not waive any renewal or expiry diagnostic loss.

[Whole-node D-104](deadline-verdict.json) passes: **161 positives / 41 drops**;
last positive **D−99.528349 ms**, first drop **D−62.521970 ms**. **One pre-D
loss** remains counted; literal zero pre-D loss is false. The conservative
[D−101 ms, D] transition retains the 100 ms installation reserve and 1 ms
truncation; no post-D positive is permitted. Finite-member expiry also passed
while its longer node lease remained valid.

[Renewal](continuity-analysis.json): **1396 probes / 1369 positives / 27 losses**,
**two** actual renewals and **70** completed guards. Every lost ingress overlaps
a completed outer guard; **0** are outside. All **27 losses remain losses**;
overlap is not an exact kernel drop timestamp or causal proof. Historical
renewal losses and follow-up 9's unattributed early outage remain unchanged.

## Completed guards and per-Apply commands

**7879** contiguous source records, **952 Applies**, **one failed pre-install
Apply**, **zero install overruns**. All **952** Apply host/lifecycle contexts
are available. **53,632** commands join by original run/boot/sequence/index;
**53,625** report `ok=true`. The seven other commands are conntrack revoke
exit-1 results within successful Applies, retained separately from the failed
candidate. Exit/signal/CPU/switch samples are available for all commands;
no cancellation callback was observed. Exact child-start and hidden host
ancestors remain unavailable. No scheduler or failure cause is inferred from
these observations. D-106: **627/627** canonical digests recompute and match
producer reads.

Nearest-rank milliseconds over completed guards, p50 / p90 / p99 / max:

| Independent sample / measure | n | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|---:|
| Follow-up 13 outer | 951 | 31.476965 | 40.656124 | 48.912754 | 55.460784 |
| Follow-up 13 install | 951 | 16.201976 | 22.493861 | 27.189380 | 29.107159 |
| Follow-up 13 render→installed | 951 | 16.302182 | 22.578520 | 27.277997 | 29.208894 |
| Follow-up 12 outer | 949 | 31.733360 | 41.314613 | 49.795099 | 54.955270 |
| Follow-up 12 install | 949 | 16.616649 | 22.812681 | 28.987567 | 31.429430 |
| Follow-up 12 render→installed | 949 | 16.747072 | 22.945306 | 29.084397 | 31.716162 |

Incomplete/failed operations remain in totals, outside completed-guard
populations. The historical maximum is not a ceiling. See
[producer analysis](producer-timing-analysis.json) and [command summary](command-lifecycle-summary.json).
The **127,502,155-byte** producer export is published losslessly as a
**6,495,737-byte** gzip, byte-compared with the original export. Both SHA-256
forms are pinned only at [the immutable stream archive](history/follow-up-13-stream/archive.json).
No record is sampled away.

## Agent-core 32 regressions and D-135 / corrected D-136

[Hold review](agent-core32-regression.json): **two** unique accepted generation-
mismatch reports, both forwarding=false with `policy_missing` and
`STALE_GENERATION` at `stage=lease`. One is before the ACK catches up; the other
has ACK **25** / signed lease **24**, with **steps=1, elapsed_ms=0, alarm=false**.
**No alarm was observed**. The retained streak never demonstrates the 3-Step
or 30-second trigger; this run does not claim that threshold was exercised.
The baseline remains. Another `policy_missing` report alone is explicitly
unclassified rather than assumed to be a stale-lease hold.

[Baseline review](policy-missing-analysis.json): **18** policy-missing windows;
**15** populated old/new-generation windows are consistent with stale leases.
Baseline→next successful Apply ranges **202.758066–1078.354515 ms**. Every one
has a later successful live verifier and serving heartbeat. No diagnostic
packet ingresses cover these 15 brackets, so their packet loss and independent
egress are **unavailable, not zero**. Every separately captured **27 renewal /
41 expiry loss** stays counted. Ordinary handling is baseline→ACK→renewal→
Apply→verify→heartbeat; only retained Apply-error recovery uses a verified
deny-all candidate. Recovery is required, and the install budget does not
bound the outage. Desired-accept/branch events are not exported, so per-call
cause labels remain unproven.

Direct terminal agent stops use pidfd SIGTERM and exit readiness. Source
SIGTERM→exit upper bound **9.117561 ms**; clone **13.659497 ms**, both below the
**13 s cooperative bound**. Both retained the same stored lease ID and original
monotonic anchor. Exit status was already unavailable when /proc was sampled;
no exit code is invented. These are two sampled stops, not a worst-case-budget
proof or a lease-eligibility claim. [Raw closure](receiver-closure.json).

Ordinary Docker stop→start completion **10.247527346 s** retained the lease
and recovered counted forwarding. Its container SIGTERM→die was
**10.086659773 s**, with SIGKILL and exit **137**. The clone fixture's container
SIGTERM→die was **20.152971455 s**, also SIGKILL / **137**, with stored lease
survival. These are container PID1 events, separate from agent-process stop
qualification; later events are teardown after terminal agent stops.
[Container events](container-lifecycle-events.json), [stop summary](container-stop-summary.json).

**No control I/O after cancel (M-1): not observable here.** There were no proxy
observation completions in the measured terminal signal→exit intervals, but
the stream lacks the actual Step-context cancellation and agent-side POST
completion timestamps. Zero proxy completions cannot establish that contract.

## Validation and remaining scope

Area lint/tests/build, amd64/arm64 cross-compilation, handover claims, whole-
module lint, formatting, diff and decompressed-evidence hygiene are recorded
in [validation-results.json](validation-results.json). The assembled exit 2
retains both operator blocks. No physical arm64, native platform matrix or
full fault-matrix pass is inferred. Four text captures have only terminal
blank lines normalized, with originals private; JSON/JSONL producer and wire
bytes and D-120 verdict are retained. The final and independent owned-prefix
inventories show **zero containers, images, networks and volumes**.

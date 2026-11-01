# Isolation follow-up 12 — 2026-11-01

One assembled run **`1c25ee44-1872-4cce-a9f5-d52491d87095`**, runtime source
**`99e58bff0cf7276e36cb04f35605835208c8638e`**, source dirty **false**.
Product baseline **`5bd751026073444087b32ac2a7115f8566104f8b`**, after
netlab 22 (#365) and agent-core 31 (#369). Driver **1074.90 s**; full command
**1207.405 s**, exit **2**. This is the next independent sample after edd986c4,
with the new TEST ONLY lifecycle producer and shutdown change. It is not an
A/B comparison or a qualified quiet-host result.

| Case | Result: pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **blocked: 8 / 1 / 0** |
| [AT11](AT11.json) | **blocked: 2 / 1 / 0** |
| [AT12](AT12.json) | **pass: 10 / 0 / 0** |
| [AT13](AT13.json) | **blocked: 6 / 1 / 0** |

AT10's D-087 transition passes; its ISO-F6-01 row is **blocked, 5/6 assertions**,
by missing dirty-generation observation evidence. AT11 and AT13 retain their
separate operator gaps. [Three dated hand-offs](requests.md#follow-up-12-requests--2026-11-01).
Spec §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04, ENT02, SEC01, DNS03.
No acceptance predicate or product code was changed in this lane.

## D-098 and measurement identity

First commit **`b597f995`** archives all **103** root files of edd986c4,
byte-identical against baseline main, with SHA-256 pins in
[history/follow-up-11/archive.json](history/follow-up-11/archive.json).
Earlier archives are untouched. Handover reads immutable evidence; its mutable
path strings are rejection inputs. No D-132 exception edit was necessary.
Evidence pins are only at immutable history paths.

[Preflight](host-preflight.json): `vmstat 5 7` and sanitized `pgrep -af
'gates.sh|mk/at-'`. Six interval samples: runnable **0–1**, idle **82–87%**,
iowait **8–13%**; one other suite. Admission was checked before each stack and
after startup. These observations do not establish a failure's cause.
[RUN.md](RUN.md) records the baseline and actual `go env GOCACHE` output.

## D-087 and D-120

The exported lease-only TEST ONLY seam kept generation **13**
and invariants byte-equal. False → natural signed empty lease → real accepted
HTTP 202 `AWAITING_MEMBER_GRANTS` report → true → new populated signed lease
and accepted serving report → independently counted native forwarding.
True→accepted serving **243.153733 ms**; admission bytes **2632→5264**.
[Original-wire audit](member-grants-audit.json): **261** hashes,
**34** desired responses. No fabricated observation or lease, forced
generation, projection stop or transport hold drove the seam.

D-120 **AT10 PASS under D-120**: **101** verified Applies, **49** datagrams,
**one** retained boundary loss, **zero** TCP timeouts. The original
[verdict](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt)
retains datagram 48, its unique qualifying guard and next datagram success.
D-120 does not apply to renewal or expiry diagnostics.

## Whole-node D-104 and renewal

Whole-node D-104 **pass**, D=`110488811839293` ns:
**161 positives / 41 drops**, last positive **D-99.604363 ms**,
first drop **D-62.610491 ms**. **One pre-D ingress loss**
remains counted; literal zero pre-D loss is false. The conservative transition
is **[D−101 ms, D]**, the 100 ms reserve plus 1 ms truncation. The D+100 ms
first-drop observation bound never permits a post-D positive.
[Exact verdict](deadline-verdict.json) and [producer joins](expiry-timing-analysis.json).

Renewal: **1396 probes / 1360 positives / 36 losses**, **two actual accepted
renewals**, **70 completed guards**. All **36** lost ingresses overlap completed
guard brackets; **zero** are outside. Every loss stays counted. Overlap is not
an exact kernel drop timestamp or causal proof. [Continuity](continuity-analysis.json).
Follow-up 11's independent 18 losses and follow-up 10's 30 remain separate.
Historical follow-up 9's 96 early drops / 65 subsequent positives / 41 final
drops remains failed; its initial baseline writer remains unattributed.

## Command lifecycle and completed guards

**7915** contiguous records, **956 Applies**, **7 failed Applies**,
**0 install overruns**; all **956 host contexts** and per-Apply lifecycle blocks
are available. **53754 command records** join by run/boot/sequence/index.
Start/return/wait, cancellation, before/after resources, kind/phase, started/ok,
exit/signal and availability fields are validated. Missing evidence says
unavailable; exact child-start and hidden host ancestors are unavailable in
all records. No cancellation callback was observed. CPU/switch samples are
available; they do not prove contention causality. Causes remain unproven.

The **127,807,956-byte** raw stream is retained losslessly as
[producer-operations.jsonl.gz](producer-operations.jsonl.gz), **6,497,248 bytes**;
decompression was compared byte-for-byte with the export. Indexed inventories
avoid duplicating every successful command in [producer-timing-analysis.json](producer-timing-analysis.json);
failed Applies retain their full records. [Command summary](command-lifecycle-summary.json).
Historical replay retains every prior result, adding only unavailable lifecycle
fields. Offline representation changes do not change the sampled binaries.

Nearest-rank p50 / p90 / p99 / max, milliseconds, completed guards only:

| Population / measure | n | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|---:|
| Follow-up 12 / Outer guard | 949 | 31.733360 | 41.314613 | 49.795099 | 54.955270 |
| Follow-up 12 / Install command | 949 | 16.616649 | 22.812681 | 28.987567 | 31.429430 |
| Follow-up 12 / Render→installed | 949 | 16.747072 | 22.945306 | 29.084397 | 31.716162 |
| Follow-up 11 edd986c4 / Outer guard | 945 | 31.545345 | 41.568870 | 48.385975 | 56.774935 |
| Follow-up 11 edd986c4 / Install command | 945 | 16.520153 | 22.588465 | 28.098082 | 30.353789 |
| Follow-up 11 edd986c4 / Render→installed | 945 | 16.615574 | 22.743450 | 28.185461 | 30.431648 |

Completed guard does not imply later verification success. Incomplete guards
are excluded from these populations, never from failure totals. An install
overrun does not release the guard; a baseline can replace it while closed.
The historical 52.596424 ms outer maximum is not a ceiling.

All seven failed Applies belong to producer `252fd542-4dae-4f26-89fd-e06c870c72c0`,
sequences **2311, 2320, 2329, 2338, 2347, 2356, 2365**. Each is generation 30,
pre-install, verifier unavailable, incomplete guard, 37 command records,
cause unproven. The labelled foreign-tc scenario and agent logs retain
`TC_FOREIGN_FILTER`. Four additional nonzero command exits are `conntrack`
revoke commands (exit 1, signal 0, context none→none) within successful Applies;
those command results are not silently promoted to failed Applies.

## ISO-F6-01: new observation gap, no pass carried forward

**Five of six assertions pass**: rejected read/quarantine, removal of injected
objects, clean generation and matching lease reaching Apply, verified/live
digests with rejection history, and restored topology/counting. No assistance.
The sixth assertion remains blocked. All twelve dirty-generation kernel samples
retain flower/mirred, link down and `serving 0`. Their central observation is
still generation 29 at **09:55:00.452272372 UTC**. Generation 30 was delivered
at **09:56:19.012930679 UTC**, followed by the seven failed Applies above.
There is **no generation-30 TC_FOREIGN_FILTER observation** in the receiver.
The generation-29 rejection observed at **09:56:18.153188288 UTC** is accepted
at **09:56:41.080290184 UTC**, after removal at **09:56:25.090642995 UTC**.

[Exact gap index](iso-f6-01-observation-gap.json), [original wire](receiver-capture.json),
[kernel snapshots](foreign-redirect.json), [unchanged six predicates](recovery-verdict.json).
Repeated observation 502s during control loss precede this window; backoff and
first-deciding-body retention are consistent with `node/agent/observe.go`.
The exact queue state was not exported; no stronger causal conclusion follows.
Local failed Applies or later recovery cannot replace the missing predicate.
No second assembled run was used to replace this blocked sample.

The later clone phase has one separate `qdisc/read_failed` rejection at
**09:57:54.450854944 UTC**, candidate 31, with no foreign objects recorded.
It remains in the outside-injection list; no foreign-object creator or cause
is inferred. D-106: **616/616** digests recompute and match producer reads.
D-093 remains 15 agreements, 13 open and two retired only in simulator scope.

## D-135 / corrected D-136

Ordinary stale-lease handling installs the node-wide baseline, ACKs, renews,
then Applies, verifies and heartbeats. Only recovery from a retained Apply error
uses a verified deny-all candidate. Baseline preserves revocation; it does not
restore the old generation. Recovery requires a successful Apply plus later
checks. The 100 ms installation budget supplies no outage/retry ceiling.

[Policy-missing review](policy-missing-analysis.json) retains **19** windows;
**15** have populated old leases and a different generation on the next
successful Apply, consistent with the stale-lease path. Baseline→Apply spans
**205.950370–264.804339 ms**. Each has a later successful live verifier and
serving heartbeat. No renewal/expiry diagnostic ingresses fall inside those
15 windows: packet loss and independent egress there are **unavailable, not
zero**. Every one of the separately captured 36 renewal and 41 expiry losses
remains counted. Desired-accept/branch events are not exported, so individual
stale-lease cause labels remain unproven. Every window, recovery endpoint and
coverage limit is retained; no deny-all candidate is invented for ordinary
baseline calls.

## Stops, stored leases and cleanup

[Ordinary restart](restart-lifecycle.json): Docker stop **10.088571 s**,
stopped SQLite snapshot **52.343 ms**, stop→start completion **10.213515 s**.
The identical stored lease ID and original monotonic anchor survived. Counted
recovery passed. Its exact Docker signal/exit events were no longer retained,
so graceful agent shutdown is unavailable, not inferred from recovery.

[Clone fixture](identity-freeze.json): whole call **21.475699 s**; actual Docker
finished→started interval **1.332763 s**; stored lease and anchor survived.
[Observed events](container-lifecycle-events.json) show SIGTERM→SIGKILL in
**20.007193 s**, exit **137**. This is a forced container stop, not a graceful
agent shutdown result. Terminal agent-service stops took **456.515 / 456.543 ms**
for source/clone; both retained their lease ID/anchor ([closure](receiver-closure.json)).
Lease survival does not establish serving eligibility. Subsequent container
termination events are stack teardown after those agent stops.

The assembled final inventory and an independent repeat both show **zero**
owned containers, images, networks and volumes. [Validation](validation-results.json)
retains actual output tails. Area lint/test/build (amd64/arm64), handover, whole
module lint, formatting, diff and decompressed-evidence hygiene pass. Only
terminal blank lines in five text captures were normalized, with original
copies private; the D-120 verdict and all JSON/JSONL source bytes are retained.
No physical arm64 or full operator fault/platform result is inferred.

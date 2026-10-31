# Isolation run procedure — Follow-up 8

```sh
make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build
make -f mk/at-isolation.mk at-isolation
python3 tests/e2e/at/isolation/run.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/isolation/run.py make go-lint
```

The binding is `w4-isolation` / port base `36000`. Every created Docker object
uses `fvpn-w4-isolation-*`. The foreground Python runner checks
`pgrep -af 'gates.sh|mk/at-'` before the lab and before the assembled stack,
waiting while two other suite trees are active. Its explicit `finally` teardown
uses no shell cleanup traps. TMPDIR/GOTMPDIR are unique disk directories under
`.agent/tmp`; no host network changes or sudo are used.

The lab first runs the merged netlab 18 AT10 reload scenario and the D-104
precision qualification, then the owned AT10–AT13 mechanism tests. The same
unmodified production source builds the assembled control/worker/node images.
The owned driver runs through a Go overlay, without copying or editing the
shared driver. A blocked or failed case produces make exit 2 after writing all
records. Mechanism passes never promote assembled acceptance (§19.2).

## D-120

`reload-stream/TestAT10_AT13_ReloadRevokeBaseline/` retains the mandatory
netlab analyzer inputs, its complete analysis and printed verdict. The original
100 reloads, one outstanding datagram, 1.5 s timeout and 200 ms sleep remain.
Only ingress inside/boundary of exactly one verified outer guard bracket is
eligible for tolerance; guard upper bound and render→installed must each be
≤100 ms, the next datagram must succeed, TCP timeouts must be zero, and at most
five UDP losses are permitted. This AT10 continuity rule is distinct from the
D-104 installation bound. No netlab implementation is edited by this lane.

## ISO-F6-03 / D-104

The fleet API sets a TEST ONLY 60 s ceiling and a capacity generation requests
renewal. A 70 s continuous UDP probe window records repeated real renewals
before control/worker are stopped. The harness copies the node's lease/anchor
SQLite state read-only, retaining signed issue/expiry times, monotonic send
anchor, and computed monotonic deadline. It restores the ceiling through the
API and retains the audit trail.

The final expiry trace uses 100 ms probe cadences phased at D−101 ms and at an
arbitrary 37 ms offset. The explicit D−101 ms sample supplies the boundary
measurement for both. Each TEST ONLY UDP probe has a unique source-port packet
ID, scheduled/send/read-completion CLOCK_MONOTONIC timestamps, and an 80 ms
read window. A separate node AF_PACKET reader records header-only ingress on
fvmb0 and egress on fvup0 using the same monotonic clock. Its final socket
statistics must show zero drops. Node `any` tcpdump independently records both
interfaces and its capture loss summary (D-108). These are userspace receipt
times, not hardware timestamps.

`deadline-verdict.json` requires last positive ≥ D−101 ms, first drop ≤ D+100 ms,
no positive at/after D, no later positive after the first drop, and complete
send/ingress/read evidence. Missing ingress never proves denial. Socket echo
loss with observed egress is retained and does not masquerade as a forwarding
drop. The tolerance is never increased for sampling cadence.

The receiver retains real renewal transactions and accepted/rejected observation
POSTs. Periodic precision-set and heartbeat reads retain start/end brackets.
`nft-passive-receipts.jsonl` retains passive netlink guard add/delete
receipt times and generation markers. It does **not** expose exact Apply start,
final-render start or executor-command completion; those missing producer
samples remain an explicit instrumentation request. Repeated-renewal packet
loss and guard receipts are retained even if the final expiry boundary passes.

## ISO-F6-01 / D-096

From a counted serving state, inject a labelled TEST ONLY clsact/flower/mirred
redirect. Capture rejected read/time, quarantine, heartbeat and node `any`
packets. Publish a dirty newer generation while the object remains; verify
non-serving and continued foreign-object presence. Remove the whole injected
clsact, retain fresh qdisc/filter reads, then publish another newer capacity
generation. Let ordinary agent Steps recover without restart or topology
writes. Native NetBird owns the IPv4 addresses/routes; the harness never
flushes/restores routes or raises the member interface.

`review_followup8.py` derives the six assertions in
[G2 follow-up 14](../../../poc/G2-followup14.md) from real wire bodies, signed
leases, kernel/topology snapshots and the final counted member control. It
recomputes D-106 SHA-256 over the exact netlab projection
`{reads, objects, truncated, tc}`. Total objects are never called foreign-object
counts. Historical `errors[stage=tc_evidence]` remain separate from the latest
verified `tc_evidence`. Completed Apply is cross-checked against the read-only persisted applied model,
member-path state, installed kernel generation and matching signed lease. The
mandatory fresh live read supersedes source=apply before observation emission.
A missing successful Apply/live proof stays unproved.

## D-087

The transparent loopback mTLS receiver is installed before enrollment and
retained over source restarts. Heartbeat, poll and observation periods are 1 s;
its queue drains every 100 ms. It retains observation/renewal bodies and
responses, never credential responses or request headers. Complete receiver
files use the shared stream exporter, avoiding diagnostic-output limits.

Agent-core 24's clean-node recipe requires `SetMemberGrantsReady(false/true)`
on its **fake control** while publication, generation, topology and bindings
remain intact. There is no exported assembled public-API counterpart. This
lane does not synthesize observations or edit projector-owned rows to emulate
it. Naturally emitted candidate reports are retained for review; an unrelated
fault or clone transition cannot substitute for that recipe.

## Archives

D-098 final snapshots are `history/follow-up-6/` (run
`bf671f3a-c6ac-4824-9107-425c4c5a5f02`, 44 original files) and
`history/follow-up-7/` (run `041e9e48-79a6-45dd-a19d-8b5a63290655`, 52 files).
Each has an `archive.json` SHA-256 manifest. They contain corrected final
verdicts, including assisted recovery, not only bootstrap snapshots.
Handover already reads the follow-up 6 archive and immutable follow-up 7
fixtures; no living-file claim pin moved. The path-string rejection regression
in `handover_test.go` does not read the living AT10 record.

## This run's passive-recorder correction

The originally compiled recorder expected a `nftables` wrapper, while this nft
version emits top-level event objects. A diagnostic read observed its empty file; the original export path also
ran only on receiver-export failure, so that empty file was not exported.
Both harness defects are corrected for future runs. A separately
labelled read-only `monitor_followup8.py` recorder was attached before restart,
renewal, expiry and foreign-tc recovery. It reconnects across container restarts,
records host CLOCK_MONOTONIC receipt times in `nft-passive-receipts.jsonl`, and
ends on teardown. These include Docker stream delivery latency and never stand
in for renderer/executor samples. Earlier missing events are not reconstructed.
The parser fix is in the owned helper for future runs; no running product binary
or test action was replaced.

# Follow-up 44: D-150 post-merge timing declaration — 2026-11-03

Committed before any measured leg. TEST ONLY same-host comparison of follow-ups
37–43 under D-133, D-138 and provisional D-150. No production change.

- A: `567080be1969bf22782617e78c7605c2c92a52a2`.
- B: `38420dadef6ae9d0236288eef533bc1ddee0da53`, production unchanged from
  `b42ed80b0dfaa184f20303f13a90a226925228e2`.
- #429 merge: `28e9e41c`. The command below returned exit 0 and **empty output**.
  It compares every non-test Go file outside tests and docs, including additions
  and deletions. The current main's following roadmap commit changes no Go code.

```sh
git diff 38420dadef6ae9d0236288eef533bc1ddee0da53 28e9e41c -- '*.go' ':!**/*_test.go' ':!tests/**' ':!docs/**'
```

- Reused A archive SHA-256: `3b0c7791dcdd4a8e0c79819bc4c55c67963805f69e775ab04c38d41f690134b9`.
  All 18,889 tree blobs reverified after fresh extraction; no mismatches.
- Reused B archive SHA-256: `5933e9a86278143e841e5316fdf3a8c025c6f7a24ce0d27a9356f9e8226b74fe`.
  All 18,939 tree blobs reverified after fresh extraction; no mismatches.
- The two timestamp-only overlays are byte-identical to 43's declared hashes.
  Only the local overlay mapping paths change. All old runners, tools, tests,
  archives and results remain unchanged. The final 43 results-tool hash is
  checked against 43's validation record (which records its public redaction fix).
- [Pins](followup44-timing-pins-2026-11-03.json) freeze sources, fixtures,
  tools, tests, overlays and separately built clock probe binaries. Every
  archive/tree/overlay and declared tool/probe is checked before each window.
- Prefix: `fvpn-w2-agent-core-f44`. Fresh output: `.agent/f44/timing/`.
  Fresh execution trees: `.agent/tmp/f44-timing/{A,B}`. No earlier evidence is overwritten.

## Windows and admission

Two reservations only, each with a maximum ninety-minute foreground wait.
Record every refusal and elapsed wait. Reserve with
`bash "$FVPN_HOLD_SH" --reserve 30 "agent-core f44 D-150"`.
Wait sixty seconds after success, then preflight. Window 1 runs A1,B1,A2,B2;
window 2 runs A3,B3,A4,B4, at least ten minutes after window 1 releases.
Every leg runs full `make -f mk/node.mk node-test-netlab`, all thirteen scenarios,
with no test filter. Never start with less than seven minutes left. Stop a
running leg thirty seconds before expiry. Release promptly when its last leg
ends. Commit each window's receipts before starting the next window. No third
reservation, replacement leg or retry within a reservation.

Admission must print exactly `free` before each leg and before every Go test,
make and Docker command, including recursive invocations. During a leg this
session runs only the foreground runner and its declared suite/monitor children;
all streams go to files. This Codex process and every other lane's session are
foreign CPU. No unrelated host load is paused or newly excluded.

## D-150 changes and unchanged qualification

Four-second monitoring intervals use `/proc` PID/start identity and the exact
39/43 exclusions: runner descendants; own full container cgroup IDs; pid 2 and
its kernel children; dockerd/containerd/containerd-shim*/docker-proxy/runc.
Shared PostgreSQL and all other processes remain foreign. vmstat is context
only. Excluded CPU classes, contributors and departed PIDs are retained.
Processes that start and exit entirely between samples remain a sampling limit.

Foreign CPU above 0.5 cores **marks** the interval without stopping or
disqualifying its leg. Each mark contains UTC, CLOCK_MONOTONIC endpoints,
foreign cores and its five largest foreign processes (pid, comm, ticks).
Another lane's suite or fvpn container, unreadable monitoring state or a sample
interval over five seconds still stops and disqualifies. The shared `fvpn-pg`
container is not a competing lane container, and is never altered.

All five span types use strict overlap: start < marked end AND end > marked
start. Touching endpoints do not overlap. Exclude overlaps from distributions;
retain their identities and count exclusions per type, leg and arm. Failed
operations remain reported, including their marks. No packet predicate changes.

A leg is pooled only when the unchanged non-CPU qualification holds and at
least half its whole-serving-Step spans survive. An empty Step population is
not pooled. A leg below the threshold remains fully reported. Suite failures
are never silently removed from a pooled leg.

D087/D104 failures with a located failing operation overlapping a marked
interval are reported with that interval and its processes and do not stop the
item. An unmarked failure stops it. Unlocated failure predicates conservatively
stop: a mark elsewhere in a long test cannot excuse them. The frozen analyzer's
install-bound assertion is resolved to its producer render/install endpoints.
The same test failing in two pooled B legs is a B failure, even if marked.
Other suite failures, a B-only failure or an attributable shift stop this item
and route a fix to a later follow-up; node/enforce remains untouched.

## Clocks and measurements

Every monitoring sample records explicit `clock_gettime_ns(CLOCK_MONOTONIC)`
next to UTC, including both interval endpoints. Before each leg's scenarios,
a separate probe built from that frozen arm calls `timeanchor.Mono()` between
two direct kernel CLOCK_MONOTONIC reads inside the lab container. The runner
brackets the probe with its own CLOCK_MONOTONIC reads, three times. Each probe
must fit that host bracket, have the same time namespace inode as the runner
and host init, and report zero monotonic/boottime namespace offsets. A mismatch
stops the timing item. This is measured evidence, not an assumption about Docker.
The probe is not linked into production or either measurement overlay.

The five measurement boundaries remain exactly 43's: complete outer guard,
install command, render to installed, whole serving Step (including deferred
work), and separate pre-Apply accounting. Accounting is never presented as the
command guard. The equally applied timestamp overlay and its serialization
cost remain explicitly instrumented diagnostics, not zero-overhead production
measurements. Complete spans from failed operations remain eligible unless
marked. Incomplete/aborted stages remain separate failures.

Report nearest-rank p50/p90/p99/max/count over retained spans per leg and pooled
arm, exclusions, every mark/failure, packet expiry and renewal predicates,
source/binary/fixture identity and all qualification samples. D-138 permits
attribution only with at least two pooled legs per arm, the same direction in
each window and a code explanation. A single tail does not establish a shift.

The twelve new synthetic tests cover marking and top-five selection, unchanged
CPU exclusions and stopping rules, sample clocks and namespace mismatch,
admission, strict overlap for all five spans, pooling at exactly one half,
operation-level failure marks and repeated pooled B failures. All 23 new and
inherited synthetic tests pass before declaration.

Environment: CGO_ENABLED=0, GOTOOLCHAIN=local, GOMAXPROCS=4,
GOFLAGS=`-p=4 -buildvcs=false -overlay=<pinned arm overlay>`, disk
TMPDIR/GOTMPDIR in the execution tree's `.agent/tmp`, GOCACHE obtained using
`go env GOCACHE` (public spelling `~/.cache/go-build`). Python uses -B.
No shell cleanup traps, host networking commands or production edits.

A pooled B pass of all thirteen scenarios is the production lab gate. If none
passes, run an ordinary uninstrumented `node-test-netlab` on main after the last
release using this follow-up's prefix. It is functional only. All other required
gates run outside reservations. Finish with zero owned Docker inventory.

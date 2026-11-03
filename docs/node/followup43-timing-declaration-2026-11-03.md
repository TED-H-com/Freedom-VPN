# Follow-up 43: D-133 timing declaration — 2026-11-03

Committed before any measured leg. TEST ONLY same-host D-133/D-138 comparison
for follow-ups 37–42; no production change, physical arm64 or field claim.

- A: current main `567080be1969bf22782617e78c7605c2c92a52a2`.
- B: post-r26 import and sole main merge `38420dadef6ae9d0236288eef533bc1ddee0da53`.
  Production is unchanged from `b42ed80b0dfaa184f20303f13a90a226925228e2`.
- Order: A1, B1, A2, B2; full `node-test-netlab`, all thirteen scenarios each.
- Prefix: `fvpn-w2-agent-core-f43`; fresh output `.agent/f43/timing/`.
- A archive SHA-256: `3b0c7791dcdd4a8e0c79819bc4c55c67963805f69e775ab04c38d41f690134b9` (18889 matching blobs).
- B archive SHA-256: `5933e9a86278143e841e5316fdf3a8c025c6f7a24ce0d27a9356f9e8226b74fe` (18939 matching blobs).

Archives remain under the output; extraction is `.agent/tmp/f43-timing/{A,B}`.
Every tree blob was verified and is checked again before each window.
The [pins](followup43-timing-pins-2026-11-03.json) identify every runner,
overlay and fixture separately from each original tree. Follow-up 39's runner
and all earlier timing records remain unchanged. r26's six proofs passed
before and after the sole merge. No later merge follows this declaration.

## Windows and qualification

The foreground runner reserves with
`bash "$FVPN_HOLD_SH" --reserve 30 "agent-core f43 A,B,A,B"`.
Thirty minutes maximum. Wait at most sixty minutes, recording every refusal
and elapsed wait. Drain sixty seconds after success, then preflight. Run
A1,B1,A2,B2 back to back and release promptly at the window's end, before
expiry. Do not start a leg with less than seven minutes left; stop a running
leg thirty seconds before expiry. This session runs no parallel work during
a leg, and runner streams go to files. This Codex ancestor counts as foreign.

Each invocation executes one window so its receipts/results can be committed
immediately on release. Missing legs may use only one more reservation,
at least ten minutes after the first release. No third reservation or
same-reservation retry. D-138 pools qualified legs per arm across these windows;
no follow-up 39 leg qualifies here.

Qualification is unchanged from follow-up 39:

- Entry admission prints exactly `free`. Admission also precedes every Go
  test, make and Docker command, including recursive tools.
- Four-second samples; any interval above five seconds disqualifies.
- No foreign suite in `pgrep -af 'gates.sh|mk/at-|go test|node-test-netlab|netlab-test'`,
  excluding only runner descendants with PID/start identity.
- No other lane's `fvpn-*` container. Pre-existing shared `fvpn-pg` is not a
  lane suite, is never changed, and counts as foreign CPU.
- At most 0.5 foreign CPU cores in every interval. Exclude only runner
  descendants with PID/start identity; own-container processes matched by
  full container ID; pid 2/kernel children; and dockerd, containerd,
  containerd-shim*, docker-proxy and runc. Record excluded classes separately.
  Other Codex, lane and unrelated host activity counts. vmstat is context only.
- Retain corrected admitted foreground Docker events and initial inventory;
  join workers. No monitoring Docker call competes with its make child.
- Stop/disqualify for foreign suite/container, CPU breach, unreadable state
  or interval over five seconds. Preserve every refusal/failure/partial leg.
  Successive `/proc` samples cannot reconstruct processes that start and
  exit entirely between samples; retain departed PIDs, as follow-up 39 did.
- A qualified D087/D104 failure remains failed and stops the timing item.
  Other qualified suite failures also stop it. A B-only failure or attributable
  shift is reported with both distributions and routed to a later follow-up;
  no production fix or replacement of failed legs occurs here.

## Measurement boundaries

The existing producer supplies complete outer guard (install start through
release end), install command and render-to-installed brackets. Completed
brackets from failed operations remain included; incomplete stages and every
failed operation are separate. Packet expiry and renewal retain D087/D104
assertions with no added loss tolerance.

A separately pinned TEST ONLY Go build overlay adds the missing Step and
pre-Apply timestamps. Archived product files remain unchanged; node/enforce
is untouched. Both arms use the same `followup43_spans.go.txt` fixture and
exact-match insertion script. Binary identities distinguish this overlay
from the original product sources.

- Whole serving Step is CLOCK_MONOTONIC entry through original Step return,
  including deferred verification, heartbeat and cancellation. Include Steps
  serving at entry or exit; retain all other Steps and returned errors too.
  The new Step record's serialization is after its measured endpoint.
- Pre-Apply accounting starts just before `wasServing`, before binding
  settlement, Baseline/final sampling, fresh sampling preparation and intent
  fsync. It ends immediately before `plane.Apply`. Intervening host sampling
  and conntrack reservation are included; it is not a pure fsync benchmark.
  The same anchors exist in A and B; work absent from A stays absent.
  Failed early returns are separately retained as aborted boundaries.
- The existing TestMode/TestTimingPath gate controls all spans. No new writer
  runs inside Apply. A deferred pre-Apply record write runs after enforcement
  and is included in whole-Step cost. These are equally instrumented lab
  measurements, not a zero-instrumentation production cost claim.

Both overlays compile. Existing D104/F29 measurement proofs pass on both,
and five synthetic monitor/overlay tests pass. Report nearest-rank
p50/p90/p99/max and counts for all five spans per leg and pooled qualified
arm, all failed operations, packet predicates, source/binary/fixture pins
and qualification samples. If either arm has no qualified leg, draw no
comparison. D-138 attribution requires repeated direction in each window
and a code explanation; a single window or tail does not establish a shift.

Environment: CGO_ENABLED=0, GOTOOLCHAIN=local, GOMAXPROCS=4,
GOFLAGS=`-p=4 -buildvcs=false -overlay=<pinned arm overlay>`, disk
TMPDIR/GOTMPDIR in each archive's `.agent/tmp`, GOCACHE from `go env GOCACHE`
(public spelling `~/.cache/go-build`). Python uses -B/no bytecode. Foreground
commands only, no shell cleanup traps and no host networking changes.

A qualified B leg passing all thirteen scenarios fulfills the functional lab
gate. Otherwise run one ordinary uninstrumented B `node-test-netlab` after
the final release; it is functional only, never a timing leg. All other gates
run outside reservations. Leave zero owned Docker inventory. Commit each
window's receipts before the final dated results, outcome and validation.

# Follow-up 39: timing declaration (2026-11-03)

This declaration is committed before any measured run. TEST ONLY same-host
comparison under D-133 / D-138; no real-device or physical arm64 claim.

- A: current main `339db17215db583330a2395f0484f2bdad9681e7`.
- B: final production tree `edc2f387c8f21c4cf36dfcf513af1263e202b95c`, containing that same main.
- Order: A1, B1, A2, B2; full `node-test-netlab` in every leg.
- Docker prefix: `fvpn-w2-agent-core-f39` for every leg.
- Inputs: immutable Git archives, verified against every tree blob before use.
- A archive SHA-256: `85936b92a8a81865ced98ef05dee1555533423cecbbb9c71166ca5ef5c57c640` (18128 blobs).
- B archive SHA-256: `848f1f6db3430d12d13cc6864d655094f438b007a2df71b75c7e50fe5363eb76` (18143 blobs).

The executable runner is `tests/netlab/agent/followup39_timing.py` at B.
Raw archives, inputs, admission receipts, process samples, logs, operation
streams, binaries' hashes, image identity and all results remain under
`.agent/f39/timing/`. Archives execute in `.agent/tmp/f39-timing/` with disk
TMPDIR/GOTMPDIR, CGO_ENABLED=0, GOTOOLCHAIN=local, GOMAXPROCS=4,
GOFLAGS='-p=4 -buildvcs=false' and GOCACHE from `go env GOCACHE`.

## Reservation and qualification

Before measuring, reserve thirty minutes with
`bash "$FVPN_HOLD_SH" --reserve 30 "agent-core f39 A,B,A,B"`.
Record every refusal and elapsed wait, up to ninety minutes. After success,
wait sixty seconds before preflight. Release immediately when the window's
last leg ends. Current main is checked again after admission to the window.
A second reservation, if needed, is the only retry window and begins at least
ten minutes after release. D-138 pools qualified legs per arm across windows.

Every leg requires admission to print `free`. At preflight and in-leg intervals
of four seconds (never accepted above five seconds), record:

- `pgrep -af 'gates.sh|mk/at-|go test|node-test-netlab|netlab-test'`, excluding
  only the runner's descendants; no other lane's suite may appear.
- Running containers; no other lane's `fvpn-*` container may appear. The
  existing shared `fvpn-pg` is not a lane suite and is never changed.
- CPU from `/proc/<pid>/stat` utime+stime divided by interval wall time:
  foreign CPU must remain at or below 0.5 cores in **every** interval.
- Separate excluded CPU totals for runner descendants, own container cgroups
  matched by full container ID, kernel threads parented by pid 2, and
  dockerd/containerd/containerd-shim*/docker-proxy/runc. All other processes,
  including shared PostgreSQL, count as foreign. PID start identity prevents
  reuse from inheriting a descendant exclusion.
- vmstat as context only; it never qualifies or disqualifies a leg.

Foreign suite/container appearance, a CPU breach, missing readable monitoring
state, or a sampling interval over five seconds stops and disqualifies that
leg. No retry occurs in the same reservation. A leg is also stopped before
the reservation expires. Every refusal, failure and partial leg is retained.
A D087/D104 failure on a qualified leg remains a failure and stops this timing
item; no repeat is used to replace it. All other qualified suite failures
likewise remain visible. A qualified B pass of all thirteen scenarios also
fulfills the production lab gate.

Report per-leg and pooled p50/p90/p99/max for complete guards, install commands
and render-to-installed intervals, all failed operations, packet renewal and
expiry predicates, source/binary/fixture identities, and qualification samples.
Report only comparisons supported by admitted legs. The binding-change cost
includes withdrawal and durable metering before Apply; command-only guard
measurements are not a direct measurement of that entire boundary.

## First reservation retained; monitoring correction (2026-11-03)

Reservation 1 succeeded at 11:20:35 UTC after 330.438 seconds and eleven
refusals. Following the sixty-second drain, A1 entered at 11:21:40 UTC with
`free` admission and a qualifying preflight. At its first in-leg Docker read,
the admission script reported this runner's own child make as competing work.
The leg stopped after 4.016 seconds, before any scenario; it is **unqualified**.
Its log, receipts and result remain in `window1/A1`. Inventory was empty and
the reservation released at 11:21:44 UTC. No leg was retried in that reservation.

The same immutable A/B archives will be used in the sole remaining reservation.
Before taking it, the monitoring harness is corrected to admit a foreground
Docker event stream and initial inventory before starting make. The stream is
owned, terminated and joined by the runner. It supplies running-container IDs
and also detects a foreign container that starts and exits between samples.
No in-leg Docker monitoring invocation competes with its own make sibling.
CPU, process and interval criteria are unchanged. Recursive make calls now
explicitly retain the admission wrapper through MAKE. Four synthetic monitor
tests pass. The revised runner SHA-256 is `a9431ffee5be67e12f7fee4f97c7735127ae5818472e7d623d13f3df0e6468e7`; the old runner is preserved
in B's archive and commit. This tooling change does not alter either product
tree, binary input, qualifier, or scenario.

The resumed driver reads prior results and the release receipt, starts at
reservation 2, and enforces ten minutes from that release (not earlier than
11:31:44 UTC). It cannot obtain a third reservation. All four legs are still
owed; no first-window result is pooled as qualified.

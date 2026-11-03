# F27-01 diagnostic declaration — 2026-11-03

TEST ONLY. One diagnostic window on task-start main
`8a775315f232c9142d8a884b5be5d7d7b88addb0`, with additive lab observation.
No production enforcement or agent change. No A/B and no retry, including a
failed or incomplete attempt. The mandatory full-suite gate is a separate gate,
not a replacement diagnostic window. Historical B1 remains FAIL.

- Run `TestAT10_AT13_ReloadRevokeBaseline` once: initial generation 1000,
  then exactly 100 reloads (1001–1100), as in follow-up 27's comparison.
- Keep `stream.py` byte-identical: one persistent TCP connection to the
  synthetic responder's port 9001; each serial cycle sends `tN` plus newline,
  performs one `recv(128)`, then the existing UDP echo on port 9000, then sleeps
  200 ms. Each socket operation retains its 1.5 s timeout. No reconnect,
  TCP_NODELAY, retry loop, deadline extension or changed tail rule.
- TCP failure means the unchanged D-120 clause (iv) predicate: any recorded
  failed TCP read, including timeout/EOF. The existing analyzer and its tests
  stay byte-identical. Diagnostic observations cannot turn a failure into PASS.
- Opt-in `FVPN_NETLAB_TCP_OBSERVE=1` adds independent TCP captures at origin,
  node and responder, both directions, including handshake, data, ACKs and
  retransmissions. Retain only scrubbed derived headers: tuple, flags, absolute
  sequence/ACK, segment length, TCP options and kernel realtime timestamps.
  These use the same realtime clock as guard brackets; retain paired
  CLOCK_MONOTONIC anchors and uncertainty, never label conversion a kernel
  monotonic packet timestamp. Require complete capture counts and zero drops.
- Read the probe's conntrack and socket timer state before and after each
  Apply, with read start/end stamps. Do not insert reads between guard install
  and release; the snapshots enclose the swap, not its atomic commit instant.
  Retain absence/errors explicitly. Snapshot cost outside Apply remains a
  measurement limitation and is not production D-133 cost.
- For every retransmitted sequence range and TCP failure, correlate every
  observation with install start/end, release start/end and verifier brackets.
  Track request and reply through all four endpoint/node stages in each
  direction. Pure repeated ACKs are reported separately, not called data
  retransmissions. Distinguish preexisting outstanding bytes from later probes
  reusing the same socket. Header evidence must identify missing hops and ACK
  progress, observed retry delays and available RTO/backoff state before a
  causal claim. Unobserved/ambiguous evidence stays unresolved.
- An explanation requires identifying SYN, SYN-ACK, request data, reply or ACK
  loss and direction; showing why retransmission did not restore a read by
  1.5 s; and distinguishing guard overlap, conntrack change, socket/deadline
  behavior and missing capture. Outer command overlap alone is not proof of
  an exact nft commit or a particular drop rule. No-failure results include
  the count of segments touching/crossing guard brackets and do not explain B1.
- Admit only after hold reports `free`, no other lane's actual
  `gates.sh` / `mk/at-` / `go test` process, and all six current `vmstat 5 7`
  rows have r ≤ 3 and id ≥ 80 (ignore since-boot row). Retain every preflight;
  wait at most 90 minutes. Expiry is a blocked window, never a quietness waiver.
  Recheck immediately before the foreground test. Record post-window context.

D-098 initial check: all 163 / 165 / 446 payload artifacts in follow-ups
25 / 26 / 27 match their original manifests, complete tree coverage and gzip
original hashes. Recheck at completion. New evidence stays in this directory.

Any production fix, harness-pattern change or D-120 change is a proposal only.
Render/install ownership is netlab; agent behavior belongs to agent-core.

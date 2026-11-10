# Boundary probe failure diagnostics — 2026-11-10

The isolation observer now retains a bounded diagnostic for a failed UDP socket
creation or send: the operation stage, numeric OS errno when available, and
whether the error reports a timeout. It never publishes the raw error string,
endpoint addresses or arbitrary wrapped text. The existing probe schedule,
80 ms deadline, payload and strict deadline evaluator are unchanged.

The offline producer review distinguishes incomplete packet evidence from a
verified network drop. After validating both traces, a missing ingress produces
an explicit incomplete report containing every probe, its recorded timestamps,
the missing packet IDs and the original deadline verdict. Missing ingress is
represented as null. Historical records without error details retain null
error diagnostics; the old errno is not reconstructed.

All four derived timing reports are replaced with incomplete/fail records:
`producer-timing-analysis.json`, `expiry-failure-correlation.json`,
`continuity-analysis.json` and `expiry-timing-analysis.json`. This prevents an
inherited or preliminary summary from retaining a misleading nearest-guard or
network-loss classification. The producer CLI and full review CLI return a
nonzero result without a traceback for this specific incomplete-evidence case.
No nonexistent ingress is correlated with an Apply, kernel state or deadline
region. Duplicate probes, malformed clocks or intervals, dropped captures,
and local socket/send failures that contradict captured packets remain rejected.

## Offline verification

- A copy of the immutable Follow-up 27 run reproduces the previous traceback and
  missing reports. With the correction, both CLIs emit failure diagnostics and
  exit nonzero. All 273 expiry probes remain, packet 145 still lacks ingress,
  and the original deadline verdict still fails. Archive hashes are unchanged.
- Synthetic renewal socket and send failures exercise the full review CLI and
  verify that all four reports remain explicitly incomplete, with every probe
  retained and no network-loss count or nearest-guard attribution.
- A real occupied loopback UDP source port proves the socket errno path;
  injected write failure proves the send path without sending external traffic.
  Payload success, timeout behavior and bounded error projection are checked.
- The Python correlation suite passes 28 tests. The Go command tests pass all
  three tests through the ordinary isolation package test registration.
- A complete successful committed trace replayed through the baseline and new
  full review CLIs produces byte-identical copies of all four timing reports.

This work completes the diagnostic part of Follow-up 27's second request. It
does not resolve the historical socket failure's underlying cause, change that
run's acceptance status, or add a new live/network/platform acceptance result.
Clone readiness was completed separately; the IPv6 and full fault-transition
matrices retain their existing limits.

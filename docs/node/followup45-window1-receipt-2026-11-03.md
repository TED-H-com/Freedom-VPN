# Follow-up 45 window 1 receipt — 2026-11-03

Reservation succeeded immediately after 0.071510 seconds. After the required
60-second wait, A1 preflight stopped because the suite-process pattern matched
the parent shell's declaration-writing command, which included the literal
`node-test-netlab`. That parent is outside the runner's descendant set. This
was an invocation error, not another lane's suite or a product failure.
No scenario or timing span ran; B1/A2/B2 were not started. The single foreign
CPU interval was marked at 1.344228 cores and remains reported with its five
largest processes. No foreign lane container, unreadable state or long interval
was found. The reservation was released at 2026-11-03T22:47:53.544637+00:00. No retry occurs in this reservation.

The [results](followup45-window1-results-2026-11-03.md) and JSON companion retain
the sample, mark, zero-span distributions and raw evidence hashes. The runner
and qualification rules remain unchanged. Window 2 will use a standalone
foreground runner invocation after at least ten minutes from this release.

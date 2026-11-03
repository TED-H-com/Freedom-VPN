# Full-suite monitor placement correction

The first ten reservation requests were refused; no run occurred. The next
reservation was granted and the topology started, but **no full-suite Go test
launched**. Its nested Go wrapper was admitted at 11:45:35.902 UTC. The sibling
monitor's Docker inventory was refused because caller-based admission saw the
suite's `make … netlab-test` outside the monitor's ancestry. The wrapper then
waited for fresh quiet evidence that this sibling could not collect.

The blocked make subtree was canceled, preserving its output and admission
records. Interrupting make bypassed its ordinary teardown, so the first inventory
cleanup found the still-running owned lab. A separate ordinary `netlab-down`
then removed it; [the final inventory](gate2-blocked-inventory.json) proves zero
owned objects or scratch. No other container was touched.

The owned tooling now places the continuous monitor inside the admitted Go
wrapper's ancestry and runs the real Go test as a foreground child. Every Docker
and Go invocation still obtains `free`; no hold-script change or bypass is used.
The reservation runner retains the initial preflight; the suite wrapper performs
a fresh post-startup preflight, four-second checks during the entire suite, and
a final covering interval. Both monitors exclude the reservation runner's
explicit PID and descendants, not an expanded class of unrelated processes.

This is a prerequisite correction before any full-suite test execution. The
single F27-01 diagnostic is already spent and is not rerun. Its executed monitor
scripts remain preserved, and its unqualified result remains unchanged.

# Follow-up 45 window 2 receipt — 2026-11-03

The second reservation succeeded immediately (0.070016 seconds),
600.071204 seconds after the first release.
After the 60-second wait, A3 passed non-CPU preflight. All three clock
measurements passed with zero offsets and aligned brackets in different
namespaces. The full lab command then exited 2 after 13.992014 seconds,
before any scenario. B3/A4/B4 were not started. Both reservations are released;
no third reservation or replacement timing run is allowed.

The frozen harness's `prepareLab` calls Docker inspect with `CombinedOutput()`
and requires exactly `true`. The inherited timing wrapper prints admission
`free` to stderr; the combined output is `free\ntrue\n`. A synthetic execution
of the actual wrapper reproduces this rejection while separate stdout is
correct. This is a harness transport failure, not an A or B product failure.
The dry preflight tested separate Docker stdout/stderr and correctly passed,
but did not cover this combined-stream consumer. No clock failure occurred.

The [results](followup45-window2-results-2026-11-03.md) and JSON companion retain
all samples, marks, clock/probe/binary/fixture identities and zero-span metrics.
No producer operation ran. The runner remains at its committed declaration
hash; no failed leg is retried or requalified. A later runner follow-up must
preserve admission receipts without contaminating either Docker stream used
by the frozen suite. The required ordinary main lab runs after release with
the previously used ordinary gate wrapper, whose admission output is captured
in its own receipt log. It is functional only.

# Follow-up 43 timing results — 2026-11-03

TEST ONLY. All times below are milliseconds. Unqualified legs are retained and excluded from pooling.

| Leg | Window | Qualified | Scenarios | Outer guard p99 / max | Serving Step p99 / max | Failed operations |
| --- | --- | --- | --- | --- | --- | --- |
| A1 | 1 | False: qualification breach (CPU, suite, container, unreadable state or interval); see samples | 1/13 | 37.742 / 37.742 | 205.757 / 205.757 | 13 |
| A1 | 2 | False: qualification breach (CPU, suite, container, unreadable state or interval); see samples | 1/13 | 81.643 / 81.643 | 1636.113 / 1636.113 | 13 |

## Window 1 A1

| Span | n | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- |
| outer_guard | 13 | 29.965 | 34.779 | 37.742 | 37.742 |
| install_command | 13 | 13.108 | 18.552 | 19.828 | 19.828 |
| render_to_installed | 13 | 13.200 | 18.655 | 19.927 | 19.927 |
| whole_serving_step | 14 | 180.247 | 202.923 | 205.757 | 205.757 |
| pre_apply_accounting | 16 | 0.096 | 4.047 | 4.243 | 4.243 |

## Window 2 A1

| Span | n | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- |
| outer_guard | 71 | 31.970 | 40.296 | 81.643 | 81.643 |
| install_command | 71 | 15.332 | 23.255 | 33.598 | 33.598 |
| render_to_installed | 71 | 15.417 | 23.337 | 33.664 | 33.664 |
| whole_serving_step | 72 | 199.100 | 729.853 | 1636.113 | 1636.113 |
| pre_apply_accounting | 74 | 0.086 | 0.278 | 4.261 | 4.261 |

## Qualified pooled A

| Span | n | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- |
| outer_guard | 0 | — | — | — | — |
| install_command | 0 | — | — | — | — |
| render_to_installed | 0 | — | — | — | — |
| whole_serving_step | 0 | — | — | — | — |
| pre_apply_accounting | 0 | — | — | — | — |

## Qualified pooled B

| Span | n | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- |
| outer_guard | 0 | — | — | — | — |
| install_command | 0 | — | — | — | — |
| render_to_installed | 0 | — | — | — | — |
| whole_serving_step | 0 | — | — | — | — |
| pre_apply_accounting | 0 | — | — | — | — |

The JSON companion retains every failed operation, packet predicates, source/binary/fixture pins,
qualification samples, refusals and releases. Only qualified legs enter the arm pools.
Attribution requires repeated direction in each window and a code explanation; see the outcome record.

## Outcome

**Neither A nor B has a qualified leg. No A/B comparison or attribution is
supported. D-133 remains owed.** Both authorized reservations are exhausted.
B1, A2 and B2 were not started because A1 tripped the CPU stop rule in both
windows; no same-window retry or third reservation was taken.

Window 1 reserved at 17:44:03 UTC (0.035 s wait; no refusals), drained sixty
seconds and passed preflight. A1 ran 36.144 s and passed one scenario before
foreign CPU reached 0.691858 cores (278 ticks/4.018165 s). A foreign python3
process contributed 250 ticks. Release was 17:45:45 UTC.

Window 2 reserved at 17:55:45 UTC (0.037 s wait; no refusals), 600.037 s after
the first release, then drained sixty seconds and passed preflight. A1 ran
92.415 s and passed one scenario before foreign CPU reached 1.311445 cores
(527 ticks/4.018469 s). The largest contributors were an unrelated process
(387 ticks) and supervise.py (103 ticks). Release was 17:58:23 UTC. These
processes were not excluded or paused. No foreign suite/container, unreadable
state or over-five-second interval caused either stop.

Each partial leg retained thirteen failed enforcement operations from the
fixture's injected failures/blocked audits: three Apply and ten Verify calls.
No scenario failure was printed before interruption. Neither completed the
D087/D104 packet case, so expiry and renewal predicates are unavailable,
not passing. All raw streams, including interrupted Step/accounting records,
are retained and hash-pinned. The partial durations above are descriptive
only and never enter either qualified pool.

Both windows ended with zero owned container/image/network/volume inventory.
The separate uninstrumented B production gate is recorded in the outcome
and validation records; it is not a timing leg.

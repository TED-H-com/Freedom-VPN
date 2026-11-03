# Follow-up 43 timing results — 2026-11-03

TEST ONLY. All times below are milliseconds. Unqualified legs are retained and excluded from pooling.

| Leg | Window | Qualified | Scenarios | Outer guard p99 / max | Serving Step p99 / max | Failed operations |
| --- | --- | --- | --- | --- | --- | --- |
| A1 | 1 | False: qualification breach (CPU, suite, container, unreadable state or interval); see samples | 1/13 | 37.742 / 37.742 | 205.757 / 205.757 | 13 |

## Window 1 A1

| Span | n | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- |
| outer_guard | 13 | 29.965 | 34.779 | 37.742 | 37.742 |
| install_command | 13 | 13.108 | 18.552 | 19.828 | 19.828 |
| render_to_installed | 13 | 13.200 | 18.655 | 19.927 | 19.927 |
| whole_serving_step | 14 | 180.247 | 202.923 | 205.757 | 205.757 |
| pre_apply_accounting | 16 | 0.096 | 4.047 | 4.243 | 4.243 |

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

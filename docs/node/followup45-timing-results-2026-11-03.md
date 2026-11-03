# Follow-up 45 timing results — 2026-11-03

TEST ONLY. Nearest-rank times in milliseconds; marked overlaps are excluded from all five distributions.

| Leg / window | Pooled / reason | Scenarios | Marked intervals | Retained / excluded whole Step | Outer guard p99 / max | Whole Step p99 / max | Failed operations |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A1 / 1 | False: preflight failed | 0/13 | 1 | 0 / 0 | — / — | — / — | 0 |
| A3 / 2 | False: no whole-serving-Step spans | 0/13 | 1 | 0 / 0 | — / — | — / — | 0 |

## Window 1 A1

| Span | Retained | Excluded | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- | --- |
| outer_guard | 0 | 0 | — | — | — | — |
| install_command | 0 | 0 | — | — | — | — |
| render_to_installed | 0 | 0 | — | — | — | — |
| whole_serving_step | 0 | 0 | — | — | — | — |
| pre_apply_accounting | 0 | 0 | — | — | — | — |

## Window 2 A3

| Span | Retained | Excluded | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- | --- |
| outer_guard | 0 | 0 | — | — | — | — |
| install_command | 0 | 0 | — | — | — | — |
| render_to_installed | 0 | 0 | — | — | — | — |
| whole_serving_step | 0 | 0 | — | — | — | — |
| pre_apply_accounting | 0 | 0 | — | — | — | — |

## Pooled A

| Span | Retained | Excluded | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- | --- |
| outer_guard | 0 | 0 | — | — | — | — |
| install_command | 0 | 0 | — | — | — | — |
| render_to_installed | 0 | 0 | — | — | — | — |
| whole_serving_step | 0 | 0 | — | — | — | — |
| pre_apply_accounting | 0 | 0 | — | — | — | — |

## Pooled B

| Span | Retained | Excluded | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- | --- |
| outer_guard | 0 | 0 | — | — | — | — |
| install_command | 0 | 0 | — | — | — | — |
| render_to_installed | 0 | 0 | — | — | — | — |
| whole_serving_step | 0 | 0 | — | — | — | — |
| pre_apply_accounting | 0 | 0 | — | — | — | — |

## Marked intervals

| Leg | Monotonic start / end (ns) | Foreign cores | Five largest foreign processes (pid, comm, ticks) |
| --- | --- | --- | --- |
| A1 | 69511919110913 / 69515936286803 | 1.344228 | 2836578, python3, 495; 601179, MainThread, 25; 34216, claude, 5; 2834949, agy, 2; 831, systemd-journal, 1 |
| A3 | 70192287273182 / 70194241303721 | 0.527116 | 2854297, python3, 76; 601179, MainThread, 3; 2834949, agy, 3; 1, systemd, 2; 3013, systemd, 2 |

The JSON companion retains every failed operation, exclusion, packet predicate, clock measurement,
source/binary/fixture identity, qualification sample, refusal and release.
D-138 requires at least two pooled legs per arm, repeated direction in both windows and a code explanation.

Both arms have zero pooled legs; no comparison is permitted. Six planned legs never started.
A1 stopped at preflight; A3 stopped in harness preparation before scenarios.
All five distributions are empty, with zero excluded spans and no producer failed operations.
These zeros are absence of scenario data, not evidence of successful operation.

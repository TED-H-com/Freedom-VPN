# Follow-up 46 timing results — 2026-11-03

No measured legs or reservations. Dry diagnostics below are never timing evidence. Nearest-rank times in milliseconds; marked overlaps are excluded from all five distributions.

| Leg / window | Pooled / reason | Scenarios | Marked intervals | Retained / excluded whole Step | Outer guard p99 / max | Whole Step p99 / max | Failed operations |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A-dry1 / dry1 | False: dry; never timing evidence | 0/13 | 0 | 0 / 0 | — / — | — / — | 0 |
| A-dry2 / dry2 | False: dry; never timing evidence | 1/13 | 4 | 61 / 4 | 79.417 / 79.417 | 815.631 / 815.631 | 13 |
| A-dry3 / dry3 | False: dry; never timing evidence | 0/13 | 1 | 0 / 0 | — / — | — / — | 0 |

## Window dry1 A-dry1

| Span | Retained | Excluded | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- | --- |
| outer_guard | 0 | 0 | — | — | — | — |
| install_command | 0 | 0 | — | — | — | — |
| render_to_installed | 0 | 0 | — | — | — | — |
| whole_serving_step | 0 | 0 | — | — | — | — |
| pre_apply_accounting | 0 | 0 | — | — | — | — |

## Window dry2 A-dry2

| Span | Retained | Excluded | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- | --- |
| outer_guard | 60 | 4 | 28.155 | 41.354 | 79.417 | 79.417 |
| install_command | 60 | 4 | 14.295 | 23.396 | 26.828 | 26.828 |
| render_to_installed | 60 | 4 | 14.372 | 23.473 | 26.910 | 26.910 |
| whole_serving_step | 61 | 4 | 193.127 | 247.671 | 815.631 | 815.631 |
| pre_apply_accounting | 63 | 4 | 0.092 | 0.238 | 3.607 | 3.607 |

## Window dry3 A-dry3

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
| A-dry2 | 72476378506424 / 72480396253707 | 1.590443 | 3196368, python3, 538; 2239763, python3, 69; 601179, MainThread, 10; 34216, claude, 3; 3130069, codex, 3 |
| A-dry2 | 72480396253707 / 72484413989175 | 1.677562 | 3196368, python3, 538; 601179, MainThread, 92; 2239763, python3, 13; 34216, claude, 4; 3196112, codex, 4 |
| A-dry2 | 72484413989175 / 72488434219332 | 1.353156 | 3196368, python3, 481; 3197608, python3, 11; 601179, MainThread, 9; 2537, tailscaled, 6; 2995322, codex, 5 |
| A-dry2 | 72564780190979 / 72568799068993 | 0.763895 | 3212636, codex, 266; 3213014, MainThread, 12; 601179, MainThread, 9; 34216, claude, 4; 2995322, codex, 4 |
| A-dry3 | 72658322335930 / 72662340176894 | 0.836270 | 3216019, run_l5_qwen.py, 167; 2239763, python3, 70; 3216008, python3, 29; 3216010, stage_new.py, 27; 601179, MainThread, 15 |

The JSON companion retains every failed operation, exclusion, packet predicate, clock measurement,
source/binary/fixture identity, qualification sample, refusal and release.
D-138 requires at least two pooled legs per arm, repeated direction in both windows and a code explanation.

All eight intended measured legs were unstarted because the full dry prerequisite failed.
Both pooled arms have count zero for all five metrics; no comparison or attribution is permitted.

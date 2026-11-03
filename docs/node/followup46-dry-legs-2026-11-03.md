# Follow-up 46 timing results — 2026-11-03

TEST ONLY dry diagnostics, never pooled or timing evidence. Nearest-rank times in milliseconds; marked overlaps are excluded from all five distributions.

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

## Dry-round disposition

Round 1: admission returned free, but initial monitoring found four foreign
lane containers. A was refused before the suite; B was not started.
Round 2: admission and initial monitoring passed. The frozen harness accepted
the wrapper streams, clock checks passed, and A completed the ISO-F6 scenario.
At 88.405 seconds the unchanged suite-process check detected an unrelated
waiting shell carrying a suite command. The runner stopped A and removed all
owned Docker inventory. B was not started. This is a qualification stop, not a
scenario failure or a new harness defect. Whole-Step spans and admission
receipts were retained, including 13 failed producer operations from setup and
fault injection; their full diagnostic details are in the JSON.
Round 3: after a foreground wait found both inventories clear, preflight
observed an unrelated gate process. A was refused; B was not started.

All three rounds used the standalone invocation in the audit, the same runner
SHA-256 adfcf5d93d8b8e30d3cae3c27d71aea5dec5b8688b9b9c49b9d3131a18788063,
and no reservation. There was no runner revision or post-failure relaxation.
All receipts are under .agent/f46/timing/, with per-leg admission command,
stdout/stderr, exits and UTC timestamps. Every attempted round is retained.

The maximum three rounds are exhausted without a complete A/B dry pair.
No measurement declaration is authorized, no reservation is requested and no
measured leg starts. D-133/D-150 remains open. The missing B dry runs are
explicitly unexecuted, not successful and not B failures. Dry distributions
above describe the partial harness exercise only and cannot support comparison.

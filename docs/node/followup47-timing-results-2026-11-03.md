# Follow-up 47 timing results — 2026-11-03

TEST ONLY dry diagnostics, never timing evidence. Nearest-rank times in milliseconds; marked overlaps are excluded from all five distributions.

| Leg / window | Pooled / reason | Scenarios | Marked intervals | Retained / excluded whole Step | Outer guard p99 / max | Whole Step p99 / max | Failed operations |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A-dry1 / dry1 | False: dry; never timing evidence | 12/13 | 5 | 140 / 10 | 56.486 / 67.369 | 5171.205 / 5177.661 | 29 |

## Window dry1 A-dry1

| Span | Retained | Excluded | p50 | p90 | p99 | max |
| --- | --- | --- | --- | --- | --- | --- |
| outer_guard | 139 | 8 | 29.914 | 41.071 | 56.486 | 67.369 |
| install_command | 139 | 8 | 16.223 | 22.199 | 28.356 | 42.232 |
| render_to_installed | 139 | 8 | 16.394 | 22.287 | 28.442 | 42.323 |
| whole_serving_step | 140 | 10 | 193.390 | 210.566 | 5171.205 | 5177.661 |
| pre_apply_accounting | 140 | 8 | 0.097 | 0.195 | 4.039 | 4.625 |

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
| A-dry1 | 76722810184941 / 76726832822279 | 1.382178 | 27682, python3, 510; 601179, MainThread, 30; 34216, claude, 3; 2171, systemd-logind, 2; 2984, cloudflared, 1 |
| A-dry1 | 76956136764693 / 76960162013771 | 1.013602 | 76235, run_l5_qwen.py, 219; 76221, python3, 71; 76351, MainThread, 38; 76223, stage_new.py, 37; 601179, MainThread, 9 |
| A-dry1 | 76960162013771 / 76964186057561 | 0.852376 | 77379, run_l5_qwen.py, 202; 7112, supervise.py, 104; 601179, MainThread, 10; 34216, claude, 4; 77445, python3, 3 |
| A-dry1 | 76964186057561 / 76968210216957 | 1.028786 | 78690, run_l5_qwen.py, 170; 78241, codex, 163; 601179, MainThread, 45; 78593, MainThread, 12; 77445, python3, 5 |
| A-dry1 | 76968210216957 / 76972234195566 | 0.685888 | 78241, codex, 220; 2239763, python3, 36; 601179, MainThread, 9; 34216, claude, 2; 4185695, codex, 2 |

## Preflight waits

A-dry1: 4.026 seconds; waited=False; clear.

The JSON companion retains every failed operation, exclusion, packet predicate, clock measurement,
source/binary/fixture identity, qualification sample, refusal and release.
D-138 requires at least two pooled legs per arm, repeated direction in both windows and a code explanation.

## Not started

- B-dry1 / dry1: A scenario failure stopped timing item.
- A1 / 1: dry prerequisite failed; declaration withheld; no reservation.
- B1 / 1: dry prerequisite failed; declaration withheld; no reservation.
- A2 / 1: dry prerequisite failed; declaration withheld; no reservation.
- B2 / 1: dry prerequisite failed; declaration withheld; no reservation.
- A3 / 2: dry prerequisite failed; declaration withheld; no reservation.
- B3 / 2: dry prerequisite failed; declaration withheld; no reservation.
- A4 / 2: dry prerequisite failed; declaration withheld; no reservation.
- B4 / 2: dry prerequisite failed; declaration withheld; no reservation.

No pooled legs in either arm; no comparison or attribution.

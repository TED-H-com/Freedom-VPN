# Follow-up 47 timing results — 2026-11-03

TEST ONLY dry diagnostics, never timing evidence or pooled. Nearest-rank times in milliseconds; marked overlaps are excluded from all five distributions.

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

## Dry reservation outcome

One reservation, granted on its first request after 0.060 seconds at
2026-11-04T00:47:00.202151Z (2026-11-03 EST). After the mandatory 60-second
settle and clear 4.028-second preflight, A started at 00:48:04.389070Z.
A completed at 00:52:11.332843Z and release succeeded at 00:52:12.151923Z.
No preflight activity wait, reservation refusal or running qualification breach.
All 13 scenarios completed; 12 passed and TestLabExternalMeterReset failed at
frozen scenario_az_external_test.go:88: external reset not recovered in 12.393s.
Observed open reason after_final_sample, counter bytes 165612, discontinuity 0,
generation 3. This is a scenario assertion, not an admission/wrapper failure.
No root cause or causal timing explanation is established.

B-dry1 was not started. The scenario failure stops this timing item; no second
dry reservation is permitted for it. No declaration or measured reservation
follows. All three clock samples match; whole-Step output and admission receipts
exist. The runner hash is the committed audit hash. Inventory is empty after
cleanup. Dry distributions above are diagnostic only. All failed producer
operations, marks/top-five, exclusions and packet predicates remain in JSON.

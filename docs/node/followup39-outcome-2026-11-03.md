# Follow-up 39 outcome — 2026-11-03

The r22 fixes pass the full production lab and all non-lab gates. The required
A,B,A,B comparison is **incomplete**: both permitted reservations ended with an
unqualified A1. No qualified leg is pooled, and no performance equivalence or
regression conclusion is claimed. The structural serving-change cost remains
as described in [the judgments](followup39-r22-judgments-2026-11-03.md).

The [declaration](followup39-timing-declaration-2026-11-03.md) was committed as
`cd5082a0` before measurement. A is main `339db17215db583330a2395f0484f2bdad9681e7`;
B is `edc2f387c8f21c4cf36dfcf513af1263e202b95c`, containing that same main.
All 18,128 A blobs and 18,143 B blobs matched the extracted archives. The
runner correction was committed as `c5ffb344`; it changes monitoring tooling,
not either frozen product tree. Main advanced after the measured window;
these results remain pinned to the declared sources.

## Reserved comparison

| Window | Reservation wait | Result | Duration | Qualification |
| --- | ---: | --- | ---: | --- |
| 1 | 330.438 s, eleven refusals | A1 stopped before a scenario | 4.016 s | Initial `free`/preflight passed; first monitoring Docker admission rejected this runner's own child make |
| 2 | 120.189 s, four refusals | A1 stopped during D087/D104 after recovery passed | 136.621 s | Foreign CPU reached 0.550750 cores over 4.012710 s |

Window 1 released at 11:21:44 UTC. The second reservation was first requested
at 11:31:44 UTC, preserving the ten-minute separation; it succeeded at
11:33:44 UTC. Each successful reservation was followed by the required
sixty-second drain and preflight. Window 2 released at 11:37:06 UTC. Neither
window retried a leg. No third reservation was taken.

The first admission conflict was reproduced with a harmless simulated make
process. The revised runner uses an initial admitted Docker inventory and an
admitted, owned event stream, joined at the end. It detects even a foreign
container that starts and stops between samples. Recursive make calls retain
the admission wrapper. Four synthetic monitor tests pass. All original logs
and the first runner version remain retained.

Window 2 recorded 34 in-leg samples, with maximum interval 4.020653 s. No
foreign suite, foreign lane container or unreadable process sample appeared.
Its final interval contained 221 foreign CPU ticks at 100 ticks/s; an unrelated
Codex process contributed 179 of those ticks. This exceeds the fixed threshold,
so the leg was stopped regardless of the source of that work. Peak excluded
CPU was recorded separately: runner descendants 0.047290 cores, own containers
0.074661, kernel threads 0.291162, container runtime 0.119462. vmstat is retained
as context and did not decide qualification.

All failed/refused attempts remain in `.agent/f39/timing/`, including the
original archives, source pins, admission/refusal/release receipts, CPU and
process samples, container events, binary/image hashes and suite logs. The
interrupted A1's mounted scratch was copied read-only after container cleanup:
104 completed guard intervals and every written failed-operation row remain
available. D087/D104 did not finish in that leg; it has no test verdict.

## Independent production gate

The unchanged frozen B tree ran the full command with
`FVPN_LAB_PREFIX=fvpn-w2-agent-core-f39`. It passed all thirteen standard
scenarios in 234.202 s (Go package 221.027 s). The optional D116 installer-marker
scenario skipped because its separate opt-in was not set. This run is an
ordinary admitted production gate, **not** a qualified comparison leg.

The production run retains 136 completed guards, including failed operations
that installed and released a guard. Nearest-rank milliseconds:

| Metric | p50 | p90 | p99 | Maximum |
| --- | ---: | ---: | ---: | ---: |
| Complete outer guard | 31.859 | 42.895 | 89.559 | 112.302 |
| Install command | 15.985 | 23.646 | 48.469 | 83.279 |
| Render through installed | 16.083 | 23.766 | 48.557 | 83.531 |

All 26 failed-operation records remain visible; they include the suite's fault
and cancellation paths and are not removed from the evidence. D087/D104 passes:
expiry captured 97 positive packets and 15 drops, zero positives at/after D,
last positive D−100.361 ms, first drop sent D−63.861 ms, and no early drops
outside the allowed predicate. Renewal captured 1,518 probes with 17 losses,
all 17 ingress times inside a completed outer guard; capture drops were zero.
These are standalone diagnostics, not a comparison against the interrupted A1.

## Validation and limits

The [validation JSON](followup39-validation-2026-11-03.json) contains every gate,
before-fail receipt, source/archive pin, guard distribution, failed operation,
packet predicate and qualification summary. The full node lint/unit/integration
and amd64/arm64 builds, unchanged consumers, R16–R19 reviews, handover, root
pinned lint, vet, hygiene and plain module tests pass. The r20/r21/r22 converted
proofs pass, including the real 10+3-second Boot/Close blocking matrix.

The original r22 finding cases all failed before edits; portable main proofs
confirm the pre-existing live wrong-member path, failed hand-off bypass,
initialization latch and false probe-clear receipt. Reachability corrections,
D-141 decisions and all five restart windows are detailed in the judgments.
Follow-up 38's original record is preserved with dated additive corrections.
No dependency, enforcement, updater, installer or netlab-owned followup12 test
was edited. Physical arm64 and real-network acceptance remain unmeasured.

## Requests to orchestrator — 2026-11-03

The quantitative serving Step/Apply comparison remains owed. Both permitted
reservations are exhausted, with zero qualified complete legs; B1/A2/B2 did
not start. Schedule a follow-up declared quartet with unrelated agent startup
work quiet for the measured intervals. Keep the 0.5-core rule and the retained
failed attempts. This lane makes no comparative performance claim and takes
no further reservation under follow-up 39.

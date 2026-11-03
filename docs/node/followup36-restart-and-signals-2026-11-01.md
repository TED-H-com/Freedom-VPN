# Follow-up 36: restart set equality and signal lifetime

Baseline: `2f80722a98c24cf0ba18fdf898bdb607c18d1764` (main after #393).
This closes the agent adoption gap described in [follow-up 35](followup35-diagnostic-and-restart-2026-11-01.md).
It covers ISO01/ISO02, §8.1/§8.2/§16.3 and the restart mechanism in AT13
(§19), plus MTR03/MTR04 crash consistency (§10.5–§10.7).
It does not change historical acceptance results.

## Adoption requires unchanged sets

`retained_applied` already stores the entire installed `enforce.Model`,
including `DeniedV4`, `ExtraDeniedV4` and `HostV4`. `loadRuntime` restores it
before Boot attempts adoption. No public configuration field or agentstore
schema change is needed.

`adoptStoredServing` now compares those three fields against the candidate
before verifying or publishing it as serving. **Any changed set refuses
adoption**, including shrink and replacement. Even a shrink leaves the
kernel unequal to the candidate; recording the candidate as installed would
be false. Comparison ignores ordering, duplicates and prefix host bits, and
interprets nil `DeniedV4` as the enforcement spec defaults. Otherwise each
field's prefix membership must match independently. Equivalent overlapping
prefix decompositions are conservatively treated as changes.

Refusal uses Boot's existing baseline path. A busy node lock or failed control
renewal can delay the first full Apply, but cannot restore the refused
serving claim. That Apply installs the candidate's configured sets before
serving. An unchanged process replacement still adopts, preserving its
original signed deadline and the existing open meter epoch. No Step, Apply,
hot reload, watcher, timeout or stop-sequence change is included.

Proofs:

- The former `TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth/same_boot_adoption`
  gap branch now requires refusal and baseline, and retains the real
  Apply/verifier first-transaction rendering check. It fails on the baseline
  with `changed sets adopted: serving=true` and passes with this change.
- `TestAT13_F36_RestartSetChanges` covers growth, shrink, replacement,
  unchanged and reordered/duplicate sets in all three model fields; it holds
  the node lock through the first Step, then releases it for the first Apply.
- `TestAT13_F36_ChangedSetsControlTimeout` expires the real shared control
  budget while renewal is held, with the caller still alive for local work.
- `TestAT13_F36_RetainedModelContainsSets` reads actual persisted metadata.
- `TestLabF36_AT13_RestartChangedSetsStopsEstablishedFlow` keeps one UDP
  socket to the TEST ONLY destination `203.0.113.11:9000` across graceful
  restart. It checks an ASSURED conntrack entry before and after, denies the
  destination in the replacement config, holds renewal through Boot and a
  control timeout, then releases it for full Apply. The baseline, later
  rules, heartbeat, operation timeline and per-send results are retained.
  Successful sends after Boot's measured baseline completion are forbidden.
  The process-launch-to-baseline interval is measured, not described as an
  instantaneous kernel transition or a new timing acceptance qualification.

## Why isolation 15 could exit by signal 15

Both suspected windows exist on the baseline:

1. `open`/`agent.Start` runs before `signal.NotifyContext` is installed.
2. Defer order calls `stop()` before `eng.Close()`, restoring default TERM
   handling while the bounded checkpoint/withdrawal work is still running.

The second window explains the terminal source's observed outcome. The
[matched parent receipt](../e2e/at/isolation/stop-receipt-review.json) identifies
an agent started at 15:10:48.570546932Z, directly signalled at
15:10:52.841917494Z, and observed dead by 15:10:52.913479463Z (71.529909 ms).
The [agent log](../e2e/at/isolation/agent-final.log.txt) shows active Steps,
lease/usage work and completed baseline operations through 15:10:52.905Z.
That rules out the pre-Start signal window for this process.

The direct sampler in `tests/e2e/at/isolation/observer_main.go` writes
`pause-agent` **before** sending its pidfd TERM. The supervisor polls that
marker and `agent-service.sh` sends another TERM from `reap_agents`. Thus the
sampler and supervisor can deliver two TERMs while Step unwinds and Close
runs. The 0.2-second poll plus process scan is compatible with this receipt's
71.5 ms observation. The clone's 6.3 ms exit can finish before that second
send. The receipt does not timestamp the supervisor's TERM or Close entry;
this is source-and-timeline attribution, not a recovered instruction trace.

Devstack 26's PID1 handler uses this same supervised stop path, keeps the
waiter/watchdog alive, and permits the shipped 15-second agent budget inside
a 25-second container grace. Container exit zero is handler completion, not
agent exit zero. In particular, the container event list is not evidence
that its TERM reached this exact launch before the sampler. No event from a
separate container lifecycle is treated as the first TERM in this proof.

`run` now installs signal handling before opening the engine and unregisters
it only after Close returns. `TestAT13_F36_SignalLifetime` injects TEST ONLY
startup and Close gates into the command's real lifecycle and signals only
its child process. Both cases exit by signal 15 with baseline signal placement;
both complete Close and exit zero with the fix. The private opener interface
adds no runtime configuration or global mutable hook. Close's three-second
local budget and the shipped 15-second stop budget are unchanged.

## Durable state if Close is interrupted

A signal-15 exit is not evidence that shutdown checkpointing completed.
It can omit the last sample or withdrawal checkpoint and leave the previous
heartbeat. It does not itself roll back committed SQLite/WAL transactions,
replenish #388's pre-Apply conntrack reservation, or turn an unfinished
withdrawal into an acknowledged hand-off. The heartbeat is an atomic but
potentially stale observation, not a process-exit receipt. Restart and the
watchdog must continue validating it and the actual kernel state.

`TestAT13_F36_InterruptedCloseDurability` kills a real engine child by TERM
while its restart checkpoint is reading counters. Independent reopened
connections show identical desired/lease/retained metadata, pending revoke
and attempt count, immutable usage payloads/cursors and heartbeat. No closed
withdrawal checkpoint appears. The last committed usage is 4096 bytes; the
8192-byte same-boot counter survives in the fixture, and a successor accounts
for exactly 8192 total without rewriting earlier events. This tests durable
recovery, not survival of real counters after a reboot. If the kernel counters
also disappear, existing meter gap/epoch handling applies; no zero-loss claim
is made for that combined failure. The historical receipt alone cannot tell
whether its final sample had committed.

## Remaining enforcement work

**2026-11-01 — Netlab 26:** add live `denied_v4`/`host_v4` content checks in
`node/enforce/verify.go`. Agent equality compares stored installed intent
with new intent; it cannot detect a set edited underneath a running agent.
Keep these restart/first-Apply tests green. That verifier file is read-only
for this lane. Physical arm64 execution remains unmeasured; both binary
architectures are required cross-builds.

## Recorded lab result and aggregate gate failure

The [live result](followup36-lab-evidence-2026-11-01.json) records 101 replies
before the baseline, then **0 of 56** replies from baseline through the Apply
observation and **0 of 12** afterward. The one ASSURED conntrack tuple remains
present. Boot's baseline completed 47.528272 ms after replacement launch;
renewal remained held for 4.4 seconds, and the first recorded full Apply and
serving heartbeat followed its release. These are local observations, not
new deadline tolerances. The focused privileged run also passed.

The final full `node-test-netlab` aggregate **failed** its unchanged
`TestLabD087_D104_GrantRenewalExpiryCorrelation` assertion. All other executed
cases, including the new F36 proof and ordinary same-config restart, passed.
The [failure evidence](followup36-existing-timing-failure-2026-11-01.json)
records two early drops during the **old process's** successful guarded Apply:
ingress at monotonic 133986076217541 and 133986113082169, inside the install
bracket 133986043449505–133986137893333. Both precede successor launch at
133986338258849. The original process logged retained serving and the
successor logged successful adoption. Later positives violate the analyzer's
all-positives-before-first-drop rule. Last positive egress was D−100.320190 ms;
no positive reached or exceeded D. The assertion and its failed result remain
unchanged. A previous attempt also failed it; the first attempt passed that
case but failed the new fixture's initial ASSURED prerequisite.

**2026-11-01 — Orchestrator / agent-core and netlab timing owners:** route the
existing D087/D104 guarded-Apply loss for separate Step/Apply investigation
and any required D-133/D-138 experiment. This follow-up changes neither
Step/Apply nor that gate's stop sequence/assertion, and does not claim the
aggregate gate is green. Full raw attempts remain in the worktree report.

## Same-host D-133 comparison, judged under D-138 (resume)

The 2026-11-01 resume required attribution before acceptance because both
same-set adoption and process shutdown execute in this scenario. That
requirement supersedes the earlier scope-based routing paragraph above:
unchanged Step/Apply code and failures on older trees do not establish that
this delivery's failure is pre-existing or caused by host contention.

A was frozen at `2f80722a98c24cf0ba18fdf898bdb607c18d1764`; B at
`7d34c885df9ae22afa87061da7dbbbd9465955ed`. Separate archives were checked
against all 15,139 / 15,147 Git blobs, with zero mismatches. Each arm's agent,
watchdog and lab-control binaries were built from its own tree. Source-tree,
archive and binary hashes, platform information, every preflight and the
historical failure vectors are in the
[JSON companion](followup36-comparison-2026-11-01.json).

### Window 1 (2026-11-01)

The fixed window was **16:40:18–18:10:18 UTC (90 minutes)**, including source
preparation. Preflights ran every 60 seconds: no competing
`gates.sh|mk/at-|go test` process roots, and all six current `vmstat 5 7`
intervals with r≤3 and id≥80, ignoring the since-boot row. Load averages were
recorded and never used for admission. No other lane or unrelated scan was
touched. Exit samples were to use the same criteria.

**No leg was admitted.** There were 88 rejected preflights, 87 completed
within the window. The final in-flight sample finished at 18:10:37.032 UTC;
it admitted no suite. Twenty-three samples met both CPU thresholds, but
none had a clear process filter at both ends. Across all samples r ranged
0–71 and idle 7–90%. Matches included chain gates, isolation's preflight and
assembled run, members' assembled run, and Go-test wrappers. Thus the wait
consumed the whole window; there is no test-start or exit-host receipt.
The empty arm pools are missing measurements, not zero-duration results.

| Planned leg | Source | Full suite | Qualified guards | p50 / p90 / p99 / max | D087/D104 offsets |
| --- | --- | --- | --- | --- | --- |
| A1 | A | NOT RUN: admission blocked | unavailable | unavailable | unavailable |
| B1 | B | NOT RUN: A1 not admitted | unavailable | unavailable | unavailable |
| A2 | A | NOT RUN: A1 not admitted | unavailable | unavailable | unavailable |
| B2 | B | NOT RUN: A1 not admitted | unavailable | unavailable | unavailable |

There are no executed excluded legs and no new failed operations to list.
The planned command was the full `node-test-netlab` target with
`FVPN_LAB_PREFIX=fvpn-w2-agent-core-f36`, without a test filter. B's extra
F36 scenario was to remain present. The logging wrapper only delegates Go,
pins binaries and records startup/image receipts. Neither arm's frozen code,
D087/D104 assertion, stop sequence nor 15-second budget was changed. At the
end, owned containers, images, networks and volumes were all **zero**.

### Earlier failed full suites

For completeness, the earlier full-suite distributions remain below. They
are **not qualified A/B legs** and do not enter either pool. Full vectors and
failed operation identities/timings are retained in the JSON companion;
all percentiles use nearest rank over completed guards. The final run
includes F36's separate operation stream as well as the common suite stream.
Failed operations are listed separately even when a completed guard exists.

| Earlier run | Completed guards | Outer guard p50/p90/p99/max (ms) | Install command p50/p90/p99/max (ms) | Render→installed p50/p90/p99/max (ms) | Failed Apply / Verify operations |
| --- | ---: | --- | --- | --- | ---: |
| First | 133 | 31.966174/39.816663/46.827518/71.983020 | 15.984134/21.509328/24.598415/30.575210 | 16.101879/21.659681/24.713591/30.697356 | 3 / 21 |
| Second | 133 | 33.587840/43.171730/123.603359/202.016770 | 16.480458/22.912299/59.437072/61.490650 | 16.768147/23.019281/59.868038/61.581252 | 7 / 25 |
| Final | 137 | 32.798564/42.723797/130.814135/162.814760 | 15.891386/23.582555/87.919388/94.443828 | 16.002329/23.702675/88.039348/94.528286 | 3 / 23 |

| Earlier run | D087/D104 | First dropped send − D (ms) | First dropped ingress − D (ms) | Last positive egress − D (ms) | Positive at/after D |
| --- | --- | ---: | ---: | ---: | ---: |
| First | PASS; full suite failed F36 prerequisite | -63.874886 | -63.489732 | -100.699794 | 0 |
| Second | FAIL; F36 prerequisite also failed | -2900.871004 | -2900.751914 | -100.337473 | 0 |
| Final | FAIL; F36 passed | -2900.741565 | -2900.448278 | -100.320190 | 0 |

### Window 2 (2026-11-02)

After the host power loss, both sources were frozen again under disk-backed
`.agent/tmp`; every source blob and rebuilt binary matched the original
pins. The fresh window began when the runner started, **04:24:43.304 UTC on
2026-11-03** (23:24 EST on November 2), with a 05:54:43.304 UTC deadline.
The experiment finished at **04:49:48.821 UTC**, after **25m05.517s**, without
extending or reusing window 1. The interrupted November 1 non-privileged gate
has no completion credit; all required gates are rerun after this comparison.

The first A1 full suite passed, but its exit process filter matched gate
words inside the two explicitly permitted read-only `agy` prompt arguments.
Its quiet CPU sample does not undo that exclusion: **A1 stays excluded**.
The pending B1 preflight was interrupted without a lab leg running. The
process filter was corrected to ignore only those `agy` parent commands
*before* ancestor deduplication, so an actual test/gate child still blocks
admission. Raw matches and ignored-parent receipts are preserved. One
replacement preflight then failed on r=4; the next qualified. The original
window deadline and frozen sources did not change.

A clean **R1-A1, R1-B1, R1-A2, R1-B2** quartet then completed. All four full
suites passed, and all eight entry/exit samples qualified. Across those
samples r≤2, idle≥93%, and there were no competing test/gate roots. There
were 12 completed host samples in window 2: two rejected entry samples and
the excluded A1 exit are retained. This is entry/exit qualification, not a
separate host sample at each traffic boundary. The permitted read-only
sessions were not stopped or modified.

Every leg ran the full `node-test-netlab` command under the required prefix,
from its own frozen tree. Full make durations were 201.721s for excluded A1,
then 197.836s / 210.628s / 202.332s / 206.558s for the clean quartet. Each
built agent/watchdog/lab-control binary matched its prebuilt arm hash.
All five lab image **manifests** were
`sha256:745805651420cd7b79ef5fe9fd777f3f4d34fe2e827f45c0fa2ac9f1ca91a83f`
and their **configs** were
`sha256:21b4abdf89d0f30c1ccbf9b7cffbc8d03b40f15a44e851da494af6a532a10fae`.
The per-build attestation index IDs differ and are retained separately.
The earlier November 1 failed runs used a different manifest/config, also
pinned in the JSON. Their logs show the same Ubuntu base digest and cached
RUN/COPY steps; cross-date image-byte equality is not claimed. Equality of
A and B's image contents is verified within window 2.

Both B legs passed the F36 established-flow scenario, each with zero replies
over 56 baseline-period and 12 post-Apply probes. The D087/D104 packet
predicate passed on every executed leg. All earlier failed traces, including
the second and final November 1 full-suite failures, remain failed.

| Leg | Full suite | Entry/exit quiet | Guards | Outer guard p50/p90/p99/max (ms) | Install command p50/p90/p99/max (ms) | Render→installed p50/p90/p99/max (ms) | Failed Apply / Verify |
| --- | --- | --- | ---: | --- | --- | --- | ---: |
| A1 | PASS | excluded | 133 | 31.638472/43.366430/81.118175/115.455282 | 15.979506/22.035845/43.965854/87.346461 | 16.106267/22.121843/44.044408/87.422248 | 3 / 20 |
| R1-A1 | PASS | yes | 134 | 28.119186/40.128976/83.300831/85.593776 | 15.023103/22.097518/44.587532/59.671868 | 15.104500/22.177264/44.674277/59.856973 | 2 / 21 |
| R1-B1 | PASS | yes | 137 | 30.880035/44.053924/89.978113/97.547402 | 16.155996/23.367825/51.869154/58.679008 | 16.256861/23.466554/51.943341/58.749101 | 2 / 22 |
| R1-A2 | PASS | yes | 134 | 31.487043/40.964485/53.222453/130.998259 | 16.405993/21.693952/34.046134/92.063025 | 16.485874/21.772253/34.126395/92.155256 | 2 / 20 |
| R1-B2 | PASS | yes | 137 | 30.811822/42.478066/70.783198/72.847820 | 15.814621/24.874160/36.800861/41.754771 | 15.890595/24.952211/36.880877/41.835771 | 2 / 22 |

| Leg | First dropped send − D (ms) | First dropped ingress − D (ms) | Last positive egress − D (ms) | Positive at/after D | Renewal losses / probes |
| --- | ---: | ---: | ---: | ---: | ---: |
| A1 | -63.821650 | -63.280490 | -100.370954 | 0 | 19 / 1518 |
| R1-A1 | -63.918636 | -63.463138 | -100.329794 | 0 | 29 / 1518 |
| R1-B1 | -63.913853 | -63.376600 | -100.199628 | 0 | 33 / 1518 |
| R1-A2 | -63.827166 | -63.438278 | -100.291497 | 0 | 1 / 1518 |
| R1-B2 | -63.923158 | -63.868805 | -100.365590 | 0 | 11 / 1518 |

| Leg / sample | UTC | Load average (1/5/15 min) | r range | Idle range (%) | Competing roots (before/after) |
| --- | --- | --- | --- | --- | --- |
| A1 / entry | 04:25:18.496252 | 2.38/1.29/0.81 | 0–2 | 88–99 | 0/0 |
| A1 / exit | 04:29:11.484466 | 0.76/0.99/0.80 | 0–1 | 93–99 | 2/2 |
| R1-A1 / entry | 04:32:31.453416 | 0.80/0.91/0.81 | 0–1 | 98–99 | 0/0 |
| R1-A1 / exit | 04:36:20.531300 | 1.04/0.95/0.85 | 0–1 | 98–99 | 0/0 |
| R1-B1 / entry | 04:36:51.826719 | 0.63/0.85/0.82 | 0–0 | 95–99 | 0/0 |
| R1-B1 / exit | 04:40:53.695992 | 0.73/0.75/0.78 | 0–1 | 94–99 | 0/0 |
| R1-A2 / entry | 04:41:24.953374 | 0.71/0.74/0.78 | 0–1 | 98–99 | 0/0 |
| R1-A2 / exit | 04:45:18.527253 | 1.91/1.14/0.92 | 0–0 | 97–99 | 0/0 |
| R1-B2 / entry | 04:45:49.787460 | 1.38/1.09/0.91 | 0–2 | 93–98 | 0/0 |
| R1-B2 / exit | 04:49:47.628049 | 0.54/0.99/0.93 | 0–1 | 98–99 | 0/0 |

| Qualified pool | Legs | Guards | Outer guard p50/p90/p99/max (ms) | Install command p50/p90/p99/max (ms) | Render→installed p50/p90/p99/max (ms) |
| --- | --- | ---: | --- | --- | --- |
| A full | R1-A1, R1-A2 | 268 | 29.814018/40.636893/83.300831/130.998259 | 15.821677/21.825302/44.587532/92.063025 | 15.891907/21.890971/44.674277/92.155256 |
| A common suite | R1-A1, R1-A2 | 268 | 29.814018/40.636893/83.300831/130.998259 | 15.821677/21.825302/44.587532/92.063025 | 15.891907/21.890971/44.674277/92.155256 |
| A D087/D104 stream | R1-A1, R1-A2 | 216 | 30.209809/40.964485/76.703148/130.998259 | 16.012593/22.097518/44.587532/92.063025 | 16.097729/22.177264/44.674277/92.155256 |
| B full | R1-B1, R1-B2 | 274 | 30.811822/43.042151/81.867651/97.547402 | 16.097573/23.872381/41.986525/58.679008 | 16.169507/23.970746/42.070338/58.749101 |
| B common suite | R1-B1, R1-B2 | 270 | 30.811822/43.042151/81.867651/97.547402 | 16.098111/23.872381/41.986525/58.679008 | 16.235478/23.970746/42.070338/58.749101 |
| B D087/D104 stream | R1-B1, R1-B2 | 216 | 30.677462/43.674504/81.867651/97.547402 | 16.097573/24.332882/41.986525/58.679008 | 16.169507/24.417359/42.070338/58.749101 |

The qualified pools contain all **542 completed guards**: A=268, B=274.
B's total includes four guards from the additional F36 stream; the common
suite and equal-sized D087/D104 subsets are also shown. The two failed
Applies in each qualified leg never reached installation (`install_start_ns`
is zero). Their full incomplete timings and every failed live read remain
separate in the JSON. No operation failure is erased by a passing suite.

B's pooled full-suite p50/p90 are higher, while p99/max are lower. The
per-pair median direction reverses: B1 exceeds A1, while B2 is below A2.
B retains 44/3,036 renewal losses versus A's 30/3,036; all recorded loss
ingresses are inside completed guards, which is a timestamp correlation,
not a waiver or causal diagnosis. Single-TERM restart stop durations are
A=214.840538/230.735142ms and B=234.588042/214.546437ms, also reversing
direction between pairs. No source change adds work to ongoing Apply guards:
the set comparison runs on Boot adoption, and signal registration lifetime
changes startup/Close. These measurements do not establish a code-explained
repeated timing shift. No general latency improvement or zero-loss claim is
made, and the withdrawn two-leg envelope rule is not used.

The runner wrote `.agent/f36-legs-done` immediately after final exit
qualification and teardown, before non-privileged gates:

```text
done 2026-11-03T04:49:48Z legs=5
```

Owned containers, images, networks and volumes were zero after every leg
and at completion. No assertion, stop sequence, 15-second budget, Step or
Apply implementation changed during either window.

**D-138 verdict: no attributable B regression is established.** Both arms
pass the full suite under qualified quiet conditions. Under the requested
both-arms-pass rule, the earlier D087/D104 failures are attributed to host
contention; this is comparison attribution, not a controlled causal
reproduction. There is no B-only test failure or demonstrated code-explained
repeated shift requiring a production fix. The historical failures and all
losses stay recorded with their original results. Request 2's attribution is
closed; request 1 remains the already-briefed netlab 26 verifier content
check. This resume changes evidence and reporting only.

## Correction (2026-11-03, follow-up 37)

Review r20 (finding H-1) showed that the "Durable state if Close is
interrupted" claim above holds only when the next same-boot process
**adopts** the retained rules. That section's proof built the successor's
meter directly with the pre-crash bindings and did not boot a second
`Engine`. On the **refused-adoption** path (for example the changed-set
refusal this follow-up introduced), the successor's meter had no bindings:
the bytes the retained rules counted while no process sampled became a
`mapping conflict` gap at the first hand-off or rebinding Apply. They were
neither attributed exactly once nor lost silently, but they were not
recovered.

Follow-up 37 restores the retained installed model's bindings when
`loadRuntime` reads `retained_applied`, and settles the open epoch against
them right after Boot's baseline when adoption is refused. The corrected
statement is: on the same boot, the pending counter delta is attributed
exactly once whether the successor adopts or refuses; after a boot-ID change
the epoch still closes as a power-loss gap and nothing carries over. Proofs,
the remaining gaps and r20's other findings are in
[follow-up 37](followup37-r20-adoption-review-2026-11-03.md). The text above
is unchanged.

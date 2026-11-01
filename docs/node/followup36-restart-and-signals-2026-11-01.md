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

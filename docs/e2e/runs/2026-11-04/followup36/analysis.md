# Follow-up 36 — D-162 fixture cadence and consumer audit

2026-11-04. Run **`c17589fa-7e48-4dee-b574-7e29aca1effa`** observed
**32 pass / 1 blocked / 0 fail**, acceptance make exit **2**. The unchanged
scheduled-job evaluator returned **PASS (exit 0)**. Every scenario's status
matches follow-up 35. `member-ipv6` remains blocked and listed; AT29 guest-clock
rollback remains operator-run. No scenario or full run was retried.

The regular-node fixture is now explicitly 5/5/5 seconds. Before teardown, the
database yielded **123 accepted observations**, with successive-receipt
intervals **min 0.081075 / median 5.162159 / max 10.254596 seconds**. This does
not claim uninterrupted freshness: the run's final 622.437456 seconds has no
new accepted receipt. All three exercised node writers succeeded in one attempt
each, with **zero 412s**. The good updater acquired after a **446 ms wait** on
the sampled original agent, started its candidate, cleared deferred recovery
and served again; the bad artifact was rejected. Stale-then-live contention is
**unobserved**. The consumer audit identifies one missing retry outside this
lane, in isolation's explicit operator-assist script.

The acceptance command ran **14:32:49.364796–14:56:48.636097 UTC**,
**1439.271 seconds**. See [all scenario statuses](acceptance-summary.md),
[complete public log](05-acceptance.txt), [checks](checks.md), and the detailed
evidence below. Run records were committed first in `feaf3ba6`; this analysis
does not alter their outcomes.

## Fixture change and scope

D-162 adds only `observation_seconds: 5` to `writeAgentConfigs`' shared map,
with a **TEST ONLY** comment naming the decision. Both prepared files now
explicitly use heartbeat/poll/observation **5/5/5 seconds**. The product default
remains 30 seconds, the installer remains 5/5/5, and all acceptance predicates,
including the strict 15-second freshness bound, are unchanged.
`TestD162_PreparedAgentCadence` checks the serialized fields, production config
loading and the unchanged default after removing the override.
`TestD162_InstallerCadenceUnchanged` checks the actual existing installer
transformation, preservation of unrelated fields and malformed-input failure.
The installer transform was extracted solely to make that offline proof possible.
The other code change adds this record directory to the existing hygiene scan.

Members 23 [#473](https://github.com/TED-H-com/Freedom-VPN/pull/473) identified
the missing fixture setting. Targeted reads of
`docs/e2e/at/members/F23-observation-config.json` and
`F23-observation-windows.json` show its 30-second default and roughly 30-second
accepted-report spacing. Its retained later diagnostic for dual 1280 index 46
reads at **12:41:30.912797 UTC**, against a latest receipt at
**12:41:08.306704 UTC**, age **22.606093 seconds**. That is a later diagnostic,
not an invented age for the original query. Those records and failed/blocked
conditions are not rewritten or re-scored here. This full acceptance run is not
a rerun of members' dual-1280 row.

## Baseline and execution controls

Baseline **`52b93845b5cfeac1dd63e04c700857fcd098a7f6`** includes
[agent-core 51 / #470](https://github.com/TED-H-com/Freedom-VPN/pull/470),
[updater 26 / #471](https://github.com/TED-H-com/Freedom-VPN/pull/471), and
members 23 / #473. All three PR bodies were read; their identities and body
digests are retained in [source-inputs.txt](source-inputs.txt).
Measured code is **`f6edbda9909268106f7a8e08c26aab8254ae1481`**, including
D-162 commit `d8a989f5` and the hygiene inclusion commit `f6edbda9`.
This is the **first run with agent-core 51, updater 26 and D-162 together**.
Where a record cannot separate their effects, no individual causal attribution
is made.

The command was `FVPN_E2E_RUN_LABEL=followup36 make -f mk/e2e.mk e2e-test`.
No opt-in follow-up campaign or forced ETag conflict was enabled. No contention
was staged, no reservation was taken, and no scenario or full run was retried.
The [before-run record](before-run.txt) retains baseline, measured commit and
the actual `go env GOCACHE` result **`<home>/.cache/go-build`**. TMPDIR and
GOTMPDIR were on disk at `<worktree>/.agent/tmp`; no cache location was assumed.

The first required gate waited **1260 seconds (21 minutes)** for admission.
The acceptance preflight added **zero seconds of contention wait** and took
**30.140 seconds**, including the required `vmstat 5 7`. Thus the pre-acceptance
admission sleep totaled **1260 seconds**, below the 120-minute limit. See
[command admission](command-admission.jsonl),
[host admission](host-admission.txt), [vmstat](host-vmstat.txt), and the
[early foreign-process sample](early-host-inventory.txt). The exact requested
`ps -eo pid,etimes,args` and bracketed grep inventory ran before acceptance;
public foreign-process fields are only pid, comm and start ticks. The earlier
foreign run was gone at admission and is not labeled an overlap with this run.

## Fifteen-second observation reads

[freshness-reads.txt](freshness-reads.txt) records **zero executed regular-node
15-second observation-window reads** in this ordinary full acceptance path.
`TestE2E_Acceptance` does not call `servingGate26` or `survivorHold30`; their
opt-in native/recovery/publisher campaigns were skipped, as retained in the
verbose acceptance log. There is therefore no per-read age to report for
those predicates. The database receipt timeline is a separate measurement;
ages at hypothetical reads are not manufactured from it. AT members' strict
queries were audited from source, not executed by this lane. D-162's config
and measured cadence do not by themselves close the members' live finding.

## Accepted observation cadence from the stack database

The read-only [query](observation-query.sql) selected this run's regular node
and every `received_at` in the half-open acceptance-command interval. It ran
against the owned stack's database at **15:06:59.943235 UTC**, before teardown
started at **15:08:23.137847**. The [query output](accepted-observations.txt)
retains report/incarnation IDs, observed/received times, generation and forwarding
boolean, with no report payload. The [calculation](observation-cadence.txt)
retains all 122 successive intervals and the source digest. Decimal arithmetic
uses PostgreSQL's microsecond timestamps.

| Measurement | Observed value |
|---|---|
| Accepted regular-node observations in run window | **123** |
| Successive `received_at` intervals | **122** |
| Minimum / median / maximum interval | **0.081075 / 5.162159 / 10.254596 s** |
| First accepted receipt | **14:34:14.765916 UTC** |
| Last accepted receipt | **14:46:26.198641 UTC** |
| Run tail after last accepted receipt | **622.437456 s**, with no later accepted row inside the run window |

[prepared-cadence.txt](prepared-cadence.txt) retains both actual prepared-file
digests and explicit heartbeat/poll/observation 5/5/5 fields. Reports are due
at the configured interval or a state change; scheduling, uploads and backoff
still apply. There are also forty measured intervals of at least nine seconds.
No interval is discarded to make the setting look exact. The last receipt
precedes control stop at 14:46:27.869039249; the subsequent control-loss period
and brief restart/revocation tail have no additional accepted receipt in this
window. [observation-coverage.txt](observation-coverage.txt) explicitly retains
the leading/trailing portions, which are not intervals between two rows.

The fixture override and receipt spacing are measured. This first combined
agent-core 51 / updater 26 / D-162 run cannot separate their effects on every
state-change report or gap. A configured five-second cadence is not a guarantee
of exact periodic receipts or an excuse for a failed freshness predicate.

## Conditional writes exercised

[conditional-writes.txt](conditional-writes.txt) retains all three node-write
receipts, including every attempt, ETag, key, body digest and UTC bounds.
Each was accepted on its first attempt; **three writes, three attempts, zero
412 answers**. None needed a reread. The run did not force a conflict.

| Record / writer | Method | If-Match | Attempts | HTTP results | 412s |
|---|---|---|---|---|---|
| [publish.json](publish.json), capacity | PATCH | `"48"` | 1 | 202 | 0 |
| [publish.json](publish.json), publication | PUT | `"49"` | 1 | 202 | 0 |
| [revoke.json](revoke.json), node revocation | POST | `"340"` | 1 | 202 | 0 |

These are all exercised node/exit If-Match writers. Acceptance of a command
does not claim its asynchronous operation is complete; the unchanged scenario
read-backs provide that evidence. The offline conflict tests passed and retain
their three-write/ten-second bounds, identical body/key, exact error-code
restriction, reread evidence and no implicit transport replay. Zero live
conflicts does not prove the faster cadence has no conflict risk.

## D-160 and D-133 evidence

[admission-timing.jsonl](admission-timing.jsonl) and
[timing-review.txt](timing-review.txt) show **30 admitted timed sections**,
**2031 wrapped commands inside them**, **zero checks/waits inside sections**,
zero command/section mismatches, and zero held answers during acceptance.
Every section has a preceding free answer. All start/end intervals are retained.
The [286 host samples](host-overlap.txt) found **zero foreign overlaps**; no
D-133 confounded window was observed. Activity entirely between five-second
samples cannot be excluded. The later held answer before export is after the
acceptance command's end and does not confound its measured sections.

| Section | Admitted UTC | Start UTC | End UTC |
|---|---|---|---|
| update-good | 14:45:40.149432 | 14:45:40.153712736 | 14:46:01.887592383 |
| update-bad | 14:46:02.016890 | 14:46:02.020065432 | 14:46:12.651743242 |
| lease-expiry | 14:46:12.721160 | 14:46:12.724825970 | 14:56:35.670155973 |

## Updater 26: complete output, wait and candidate recovery

Both scenario records retain complete per-invocation combined output, PID,
UTC bounds and exit status. [updater-output-retention.txt](updater-output-retention.txt)
checks that the good invocation is unchanged in the later bad-update archive
and gives output byte counts and digests. The offline two-invocation retention
test passed in the acceptance log.

| Record | Invocation / PID | UTC start–end | Exit | Complete output / wait |
|---|---|---|---|---|
| [update-good.json](update-good.json) | `invocation.YQEMZXKV` / 186378 | 14:45:49.043793385–14:45:59.988545964 | **0** | One 148-byte JSON wait line, reproduced below; acquired after **446 ms** |
| [update-bad.json](update-bad.json) | `invocation.hHQ2yCIu` / 195558 | 14:46:07.995705033–14:46:11.002748759 | **1** | `fvpn-updater: updater: HASH_MISMATCH: artifact` plus newline (47 bytes); **no wait record** |

The good invocation's complete output is this single line followed by newline:

```json
{"time":"2026-11-04T14:45:49.495505375Z","level":"INFO","msg":"update lock wait","command":"run","waited_ms":446,"acquired":true,"budget_ms":10000}
```

[lock-wait-samples.txt](lock-wait-samples.txt), extracted from the original
[lock/heartbeat trace](update-recovery-heartbeats.txt.gz), identifies **PID 894 /
fvpn-agent**, executable `/usr/local/bin/fvpn-agent`, holding `update.lock`
at **14:45:49.155531064** and **14:45:49.367913641**, within the good invocation's
wait. At **14:45:49.584005669**, the holder is **186378 / fvpn-updater**.
Those are sampled owners, not a reconstruction of every 20 ms acquisition
answer. **Stale-then-live contention is unobserved**: the samples retain no
lock metadata or individual stale/live answers. No contention was staged.

The record proves a first-attempt ErrBusy and later successful acquisition
under the current wait implementation. It does not exercise or establish five
consecutive stale answers, a live-answer stale-count reset, or exhaustion of
the ten-second budget. The bad invocation reached artifact verification and
wrote no wait record, consistent with first-attempt acquisition; it supplies
no wait interval or wait holder. Its target candidate did not start, and the
good candidate remained the running executable. Rejection occurred before
switching, so `rolled_back=false` is not a failed rollback claim.

The trace, [agent log](update-recovery-agent.txt) and
[selected exact samples/log lines](recovery-proof.txt) establish the good path:

1. **Candidate starts:** boot log begins **14:45:54.570**. At
   **14:45:54.577655841**, PID **189738** executes
   `/opt/vpn-node/releases/01a1a6fb-5025-7002-810c-b276cfc37c8a/fvpn-agent`,
   with parent updater **186378** still holding the lock, forwarding latch
   set, and heartbeat `serving=0, reason=boot_unverified`.
2. **Boot recovery deferred:** at **14:45:54.789907759**, the same candidate
   reports `recovery_required`, generation **97**, while its parent holds the
   lock. Agent baseline logs show `updater_barrier`. The runtime contract
   (`node/agent/engine.go:546`, `updaterecover.go:30`, `measurement.go:70`)
   attempts recovery before the latch check and leaves recovery blocked on
   parent ownership; the latch takes precedence in the baseline cause.
   This is the observed D-155 deferred path interpreted with that contract,
   not a fabricated recovery-child exit log.
3. **Parent releases:** last sampled parent hold is
   **14:45:59.782333852**; the next sample **14:45:59.996747024** has no holder.
   The updater invocation ends between them at **14:45:59.988545964**.
4. **Deferral clears by 14:46:04.707:** the first `policy_missing` baseline
   entry has that time. Recovery has precedence over `policy_missing`, so it
   is no longer blocking by then. Exact recovery-child completion time is
   unlogged; clearance is bounded after the parent-held phase and by that
   event. The meter epoch opens at **14:46:04.837**.
5. **First reported member service:** heartbeat written
   **14:46:05.436664574**, sampled **14:46:05.451947393**, shows
   `serving=1, reason=serving, forwarding_blocked=0`, generation **99**, same
   candidate PID. `update-good.json` separately records the exact executable,
   local success, configured health probe and verified traversal. The later
   lease scenario supplies actual HTTP 200 and increased member counters;
   heartbeat/meter opening alone is not treated as a counted member flow.

Brief incomplete PID/executable reads during the old agent's stop remain in
the raw trace and selected transcript; they are not valid ownership evidence.
The observer is read only and was joined before acceptance returned.

Compared with follow-up 35, both updater **statuses remain pass** with exits
0 and 1. The material new observation is the good invocation's 446 ms wait
and sampled ordinary-agent holder. This first run combines **agent-core 51,
updater 26 and D-162**; its records cannot assign the occurrence of that
collision to one change. They do not separately prove agent-core 51's canceled
per-call Boot-context, unchanged-state binding or one-early-renewal branches.
Nor do they isolate updater 26's stale/live reset changes from the wait behavior
already present in updater 25.

## Lease expiry after candidate service

All scenario values are from [lease-expiry.json](lease-expiry.json); selected
fields are also retained in [selected-scenario-fields.txt](selected-scenario-fields.txt).
Local heartbeat and baseline times come from [recovery-proof.txt](recovery-proof.txt)
and its original trace/log sources.

- **Counted flow:** readiness polling starts **14:46:12.724854677** and finds
  member counters after **2.343563389 seconds**. The controlled origin returns
  HTTP 200, `controlled_origin_http_200_and_counted=true`, and admission bytes
  increase **17086 → 19718**. Established TCP and UDP each have a positive
  reply before the fault.
- **Control stop:** requested **14:46:27.623512030**, completed
  **14:46:27.869039249**.
- **D:** signed desired policy generation **103** expires at
  **14:56:25.259547047 UTC**. Its installed monotonic deadline is
  **127627708978464 ns**. The pre-stop heartbeat wall/monotonic pair maps that
  deadline to **14:56:25.257603476**, within 1.944 ms of signed D. This is the
  earlier desired-policy boundary; it does not isolate expiry of a longer
  member lease. Monotonic time is the enforcement reference.
- **Closed forwarding:** the existing two-second rule poll waits
  **9m58.275120649s**, records **558 → 0 member accept rules**, and finishes
  at **14:56:26Z** (whole-second precision). Agent `policy_expired` baseline
  writing begins **14:56:25.282**, succeeds at **14:56:25.435**, and a second
  write succeeds at **14:56:25.494**. The first closed heartbeat is written
  **14:56:25.508501520**, about **249 ms after signed D**. Its sample begins
  **14:56:25.498603330** and reads the heartbeat later within that sample.
  The preceding sample contains a serving heartbeat written at
  **14:56:21.933750756**. That stale heartbeat does not show forwarding after
  D. These are baseline/heartbeat/rule observations, not an exact
  last-forwarded-packet time or a subsecond cutoff guarantee.
- **Closed-path quiet checks:** senders stay active. TCP attempts increase
  **1201 → 1204**, UDP attempts **1205 → 1211**; TCP echoes and UDP replies
  each stay **1197**, upload bytes stay **231905**, download bytes stay
  **177265**. Both `active_senders_after_cutoff` and
  `established_replies_and_bytes_stopped` are true.
- **Control restart:** requested **14:56:32.458748521**, ready check succeeds
  **14:56:34.622965742**, `control_restarted=true`.

This preserves the existing ENT02 / ISO02 / SEC01 control-loss subcase of
§19 AT29 and §16.3. The guest-clock rollback row stays
[operator-run](../../../at29-guest-clock.md). Neither a host nor guest clock
was changed. The row also passed in follow-up 35; changed sample timings alone
are not attributed to any one of the three combined changes.

## Every observed row and attribution

No scenario changes status from follow-up 35. Each link below is this run's
own record. Faster receipt opportunities and a measured good-update wait do
not warrant re-scoring a row, claiming a newly passed AT case, or attributing
unchanged statuses to one merged change. This is the first run with
agent-core 51, updater 26 and D-162 together; the updater discussion identifies
the changed wait observation and the limits of causal attribution.

| Scenario record | Follow-up 35 | Follow-up 36 |
|---|---|---|
| [stack-health](stack-health.json) | pass | pass |
| [control-metrics](control-metrics.json) | pass | pass |
| [gateway-principals](gateway-principals.json) | pass | pass |
| [admin-login](admin-login.json) | pass | pass |
| [backup](backup.json) | pass | pass |
| [threat-feed-central](threat-feed-central.json) | pass | pass |
| [serving-catalogue](serving-catalogue.json) | pass | pass |
| [portal-billing-return](portal-billing-return.json) | pass | pass |
| [billing](billing.json) | pass | pass |
| [release-register](release-register.json) | pass | pass |
| [enroll-bootstrap](enroll-bootstrap.json) | pass | pass |
| [agent-supervisor](agent-supervisor.json) | pass | pass |
| [desired-state](desired-state.json) | pass | pass |
| [threat-feed-node](threat-feed-node.json) | pass | pass |
| [node-restart](node-restart.json) | pass | pass |
| [publish](publish.json) | pass | pass |
| [forwarding](forwarding.json) | pass | pass |
| [usage](usage.json) | pass | pass |
| [member-traffic](member-traffic.json) | pass | pass |
| [established-transports](established-transports.json) | pass | pass |
| [forced-relay](forced-relay.json) | pass | pass |
| [exit-dns](exit-dns.json) | pass | pass |
| [member-ipv6](member-ipv6.json) | blocked | blocked |
| [worker-restart](worker-restart.json) | pass | pass |
| [device-revoke](device-revoke.json) | pass | pass |
| [native-scale-hook](native-scale-hook.json) | pass | pass |
| [gateway-privacy](gateway-privacy.json) | pass | pass |
| [threat-feed-tamper](threat-feed-tamper.json) | pass | pass |
| [update-good](update-good.json) | pass | pass |
| [update-bad](update-bad.json) | pass | pass |
| [lease-expiry](lease-expiry.json) | pass | pass |
| [revoke](revoke.json) | pass | pass |
| [second-exit](second-exit.json) | pass | pass |

## Consumer audit (source reading)

Audit source is measured commit `f6edbda9` on baseline `52b93845`.
[The source index](consumer-source-index.txt) lists the searches across all
`tests/e2e/**`, including AT packages, Python-generated Go and `driver.go.txt`.
Paths below are relative to `tests/e2e/`. Offline test data and historical
record-formatters do not become live reads. The regular-node and node2 fixtures
both inherit the prepared config; installer-only paths are identified separately.

`node/agent/observe.go:29` makes a report due on the configured interval **or a
state change**. Outbox/backoff still applies. Consequently 5 s is a fixture
setting, not a promise of exactly five seconds between accepted rows.
`internal/nodes/observations.go:302` increments the node version when updating
an accepted observation. Periodic reports now offer roughly six times as many
opportunities for a GET/write version race; this does not imply six times as
many observed 412s. Neither this setting nor a conflict retry proves forwarding.

### Observation reads, windows, waits, counts and rates

“Can change” below means the sampled value, time to success, or eventual result
can differ with the same predicate. It does not attribute any live outcome.
No acceptance predicate, 15 s bound, wait duration, report shape, byte-count
assertion or scheduled expected-blocked list was changed.

| Consumer / source line | Read or predicate | Can 5 s change the result, and why? |
|---|---|---|
| `publish_test.go:35,48` | 90 s / 2 s poll of latest ready/passed baseline; latest row and peer-event count diagnostic | Yes: a newer valid report can satisfy the wait earlier; the snapshot/event count can differ. The publication operation and observed-active gate remain mandatory. |
| `restart_scenario_test.go:58` | Post-restart `EXISTS`, same node/incarnation, `received_at >= started` | Yes: receipt can arrive sooner inside the fixed restart wait. It cannot stand in for the heartbeat/kernel/identity checks. |
| `traffic_scenario_test.go:95,130` | Latest report diagnostics; first ready receipt joined to first subsequent populated lease; reports the delay | Timing/value only: a different receipt can move the measured delay; no periodic report-count threshold. |
| `device_revoke_test.go:145` | Latest report on the revoke diagnostic path | Diagnostic changes only; counted revoked/sibling traffic remains decisive. |
| `followup18_live_test.go:299`; `followup18_native_test.go:266` | Latest report snapshot; exact report-ID/generation accepted ACK lookup | Latest snapshot can change; exact ID lookup cannot be replaced by a newer row. A required receipt may arrive earlier. |
| `followup20_live_test.go:76,154`; `member_grants.go:34` | Wire observation HTTP 202 after seam completion, exact provisional/serving shape; 3 min wait | Yes: more qualifying report opportunities, but state transition reports were already immediate. No fixed report count or cadence substituted. |
| `followup26_live_test.go:53,134` | Export every accepted row; 90 s / 1 s serving wait, latest current-incarnation report **under 15 s**, matching lease/generation/device/heartbeat/counters | Yes: more exported rows and fewer periodic stale phases; every non-age condition remains required. |
| `fixtures16_live_test.go:213,222,231`; `followup18_native_test.go:88,641`; `followup26_live_test.go:252`; `followup27_native_test.go:20,79`; `followup27_live_test.go:52,183`; `followup30_live_test.go:146,413`; `followup32_live_test.go:72` | Call sites of the same strict servingGate26 freshness predicate in opt-in native, recovery, stream, survivor and publisher campaigns | Same age/receipt effect as servingGate26 above; none is silently added to the ordinary full acceptance run. |
| `followup27_live_test.go:63` | One latest report joined to matching populated lease/generation and >90 s remaining, after servingGate26 | Yes: reported state catches up sooner; no explicit report-age window. |
| `followup30_live_test.go:165,180,306,354` | 60 s wait for latest **under 15 s** report with two populated members and >150 s lease; later exact hold/post-mutation report joins | Yes: freshness/precondition wait and first qualifying receipt time can change. Report, incident, generation, survivor and suspension predicates remain. |
| `generation_hold.go:66,67` | All hold samples and all node observations, ordered by receipt | Export size/count changes. Downstream per-report validity can expose an additional invalid sample; report frequency itself is not an alarm or escalation proof. |
| `namespace_restart.go:94,113` | First serving receipt after start, then all receipts through the full 90 s restart window; used for node and node2 | Yes: first-receipt latency and report count change. The full window, unchanged generation and counted traffic stay required. |
| `fleet_snapshot.go:61,146` | Admin API readiness waits for retained incarnation, after service/restart | Indirectly yes: observation-driven ready projection can arrive earlier. It does not count reports or assert a reporting rate. |
| `analyze_followup26.py:29,128` | Retained reports whose **observed_at** lies between serving completion and capture end; counts/lists closed reports | Yes: a denser report history can reveal a closed state. It is an offline receipt join, not packet-exact close timing. |
| `at/members/forwarding_test.go:71,75` | 90 s wait; latest current-incarnation report **under 15 s** with matching populated lease, generation, device and live local evidence | Yes: D-162 removes the fixture's regular 30 s / 15 s phase mismatch; safety and counted transfer remain independent. |
| `at/members/d146_live_test.go:72,130`; `serving_diagnostic.go:58` | One strict **under 15 s** serving SQL read per D-146 inspection; if absent, a separate later diagnostic computes age | Yes: the one-shot freshness result can change. The later diagnostic remains a later read, not proof of the original failed condition. |
| `at/members/populated_hold_test.go:119,162,194,252` | 60 s precondition wait, **under 15 s** latest report and >150 s populated lease; first hold report, accepted post-removal alarm, post-release serving receipt | Yes: age and first accepted sample can change, with exact lease/incident/member checks unchanged. Applies to selected regular/node2 roles. |
| `at/members/d146_live_test.go:152`; `path_evidence_live_test.go:137`; `path_evidence.go:205` | All receipts in half-open `[row start,row end)`; filters and exports metadata only | More rows and different intervals; diagnostic only. This does not re-score original D-146 predicates. |
| `at/members/followup_test.go:241` | Restart readiness requires at least one report after restart | Yes: time to a positive count can change; no fixed expected count. |
| `at/members/native_test.go:272`; `forwarding_test.go:384` | Latest post-revoke forwarding/reasons and typed report checks | Yes: newest state may appear earlier in the bounded health wait; independent counted sibling/replacement transfers remain required. |
| `at/members/generation_hold_test.go:30` | Every accepted hold report joined to its own lease; validate each report's stale marker/safety and audit correlation | More samples can reveal a failure or better support correlation. Counts cannot replace valid content or revocation evidence. |
| `at/members/second_exit_test.go:140,270`; `restart_readback_test.go:22`; `restart_predicate_test.go:64,119` | node2 restart latest post-start report; all 90 s observations grouped into contiguous states; report count >0, grantless/populated counts, no foreign-filter errors | Inherited 5 s can change sample counts and first serving latency. D-067 still requires an actually observed grantless lease/report; a populated first lease stays unexercised. |
| `at/members/provisional_test.go:78` | Offline regression reads the frozen follow-up-9 restart capture | No live D-162 effect: historical bytes and expected counts remain unchanged. Its report-shape evaluator is also used by the live restart assessment above. |
| `at/secops/publication_live_test.go:18,42`; `followup15_withdrawal_test.go:121` | Current observation equals node `last_observed_at`; own lease join; production forwarding-evidence evaluator | Yes: latest projection/lease and report check freshness can change. No new acceptance age value chosen. |
| `at/secops/followup5_live_test.go:167`; `followup5_records.py:89` | First post-revoke receipt within 40 s; requires nonempty next-observation evidence plus counted survivor | Yes: fixed-window coverage may improve; revoke/forwarding validity cannot be inferred from cadence. |
| `at/secops/followup10_probes_test.go:95` | All post-start applied/error rows | Counts/timeline can change; every retained error remains visible. |
| `at/secops/followup8_live_test.go:1194`; `followup8_records.py:273` | Count >0 after restore completion within 30 s; finishes RTO measurement | Yes: 30 s cadence was at the wait boundary; 5 s can reduce the reporting component of RTO. Not faster service proof by itself. |
| `at/secops/followup10_live_test.go:478,551,591`; `followup9_records.py:163,400` | During 240 s restore hold: report with nonzero unacked usage; after recovery: count >0 and zero-unacked same-epoch report within 45 s | Yes: denser sampling may catch a transient nonzero state or converge sooner. Usage counters and replay safety are unchanged; they are not observation counts. |
| `at/secops/followup11_restore_test.go:85`; `followup13_restore_test.go:58,92` | Post-restore count >0 within 45 s; retained-generation receipt and exact first-offer ACK ID | Yes for receipt timing/window coverage; exact ID/retained generation still required. Pre-observation probes stop the agent, so cadence cannot bypass their hold. |
| `at/secops/body_boundaries_test.go:18` | Latest real payload used as input to schema/body-boundary requests | Payload snapshot may differ; no report cadence/count assertion. Body-size and HTTP boundaries remain fixed. |
| `at/secops/rate_classes_live_test.go:43,66,78` | Deliberately invalid `{}` reports from **node B**; first 429 at burst+1, Retry-After 1, global class metric delta 1, <500 ms | No direct shared subject bucket: real agent is node A. A denser regular-agent load can affect scheduling/global metrics if it itself rate-limits, so the exact-rate result remains measured and must not be normalized. It is not a periodic observation-rate test. |
| `at/enrollment/freeze_test.go:174,254` | Latest regular-node report/lease during 200 ms freeze snapshots, report replay validation and 30 s post-restart receipt wait (`freeze_test.go:289`) | Yes: accepted freeze/coherent state may be visible sooner. Cutoff is based on local heartbeat/forwarding and configured heartbeat, which stays 5 s. |
| `at/enrollment/handoff_test.go:534,577` | **Installer node:** latest report after start and fresh local heartbeat (<15 s); current-incarnation upload diagnostics | No D-162 direct effect: installer already uses 5/5/5. This 15 s bound is on the local heartbeat, not regular-node accepted-observation age. |
| `at/updates/live_test.go:155,645`; `authority_test.go:170,204`; `preresult_test.go:155,262,298` | Ready baseline `EXISTS` within 2 min; latest node/fleet receipts; paused-agent authority/result-ordering payload tests and receipt-change checks | Readiness and ordinary latest snapshots can change. Paused-agent synthetic protocol cases retain explicit report-ID/status/authority assertions; no fixed observation total. |
| `at/updates/restated_test.go:100,118` | Stop agent; wait `OnlineSeconds+30` until neither node observations nor fleet state has a receipt inside configured online window | The last receipt time can move the start of expiry, but agent is stopped and the same full online window is waited. No threshold change required. |
| `at/usage/live_test.go:859`; `live_helpers_test.go:535` | Latest unpublished-ready within 90 s; accepted serving generation within 100 s with local heartbeat/counters | Yes: next valid row may appear earlier; target generation and local proof stay required. |
| `at/usage/live_helpers_test.go:567,623,650,729,748,792` | D-067 baseline/max receipt; first ready provisional within 45 s; first subsequent failed/serving state; count of LEASE_EXPIRED reports must be zero; ready→populated bound 65 s | Yes: timestamps, coverage and forbidden-error count can change. More samples can reveal an error; do not discount it. Lease count/60 s TTL and 65 s bound remain unchanged. |
| `at/usage/live_serving_test.go:48`; `live_precision_test.go:207` | First serving report after ready; post-publication current-incarnation passed/serving receipt | Yes: first report timing/availability changes. Counted byte equality and accounting predicates do not. |
| `at/isolation/driver.go.txt:259,402,430,440` | Latest diagnostics and all bootstrap history, including awaiting-report count | More records/counts, no fixed expected total. The lane's receiver overrides all intervals to **1 s before enrollment** (`observer_main.go:119`), so its ordinary main-node run does not inherit D-162's 5 s; auxiliary nodes still may. |
| `at/isolation/driver.go.txt:1190,1236` | Serving report exists for the actual lease | The main node already runs at 1 s; auxiliary inherited configs can yield earlier receipts. Lease/forwarding content remains mandatory. |
| `at/isolation/driver.go.txt:1546,1913,1940`; `scheduled.go:45`; `convergence.go:50` | 90 s control-readiness requires a post-start observation and drained queue; three distinct 250 ms reads in a coherent provisional lease window; latest observation snapshot | Arrival/queue timing and sample availability can change; three reads are not three distinct reports. The main-node 1 s override is installed before enrollment and is unchanged. |
| `at/isolation/driver.go.txt:2075,2417,2424,2540,2788` | Wire accepted provisional/serving evidence, accepted-report ID list, latest report for recovery, all accepted hold samples with ID dedup | Counts/history can change under inherited cadence; ordinary main-node 1 s override remains 1 s. No fixed report-rate predicate; validity of each report and original event times stays binding. |
| `at/isolation/driver.go.txt:3082,3194,3213` | Populated matching lease with >50 s left, then first post-fault/restart hold report in bounded waits | Receipt time can change; exact incident/error/generation still required. pre-enrollment 1 s receiver override prevents a direct D-162 cadence change on the main node. |

The source search found no regular-node acceptance assertion requiring an exact
periodic report count or an exact 30 s report rate. Heartbeat expiry, traffic
rates, usage-event counts, lease-renewal timing, capture samples and HTTP rate
limits are separate measurements. Increasing observation frequency may still
change load or expose previously unsampled states; none of those outcomes is
excused or re-scored here. Historical Python review scripts consume their
retained rows and do not query a new live cadence.

### Conditional node/exit writers

The shared `node_write.go:95` helper retries only **HTTP 412 with
RESOURCE_VERSION_CONFLICT**. It re-GETs the resource, accepts already-intended
*desired* fields without claiming observed completion, otherwise validates the
new ETag and retries the identical body/key. It stops after three writes or ten
seconds, including all rereads, and retains each attempt/status/reread/time.
Transport errors, redirects and other responses do not trigger retries.
The offline `TestAPI01_NodeWriteConflict`, deadline, no-implicit-replay and
exported-writer tests already prove those boundaries. The publisher's
`TestAPI01_PublisherProofOptionalConflictVersion` tests retain their assertions.

| Client / source line | Writes and 412 behavior | Effect of 5 s / action |
|---|---|---|
| `publish_test.go:88,105` | Capacity PATCH and publication PUT; shared helper; scenario `node_write` receipts | More ETag races possible, already bounded/re-read/recorded. No change. |
| `scenarios_test.go:863` | Node revoke POST; shared helper; scenario receipt | Same; no change. |
| `followup18_native_test.go:226`; `followup27_live_test.go:93`; `followup30_live_test.go:231` | Capacity PATCH; shared helper; own scenario receipts | Same; no missing retry. Separate opt-in scenarios, not automatically added to this run. |
| `exports_live_test.go:353`; `installer_exports_test.go:274` | Clone reconcile and installer reenroll revoke; shared helper; caller note receipts | Same for regular node; installer cadence unchanged. |
| `stack_exports.go:77` | Exported `Stack.WriteNode`; shared helper returns full receipt to caller | All AT wrapper clients below retain receipt; the empty internal note callback does not remove the returned receipt. |
| `at/enrollment/node_write_test.go:18,23,28,44`; `live_test.go:881` | Capacity PATCH, publication PUT, revoke POST; exported helper | Retry present, with caller receipts. More regular-node conflicts possible; hand off audit, no edit. |
| `at/members/identity_nodes_test.go:195,202` | `nodeMutation` for capacity/region PATCH, publication PUT and revoke POST; exported helper | Retry present with every command receipt, including populated-hold and node2 callers. Hand off, no edit. |
| `at/secops/node_write_test.go:13,19`; `counted_capacity_test.go:106`; `followup4_live_test.go:55`; `followup5_live_test.go:57`; `followup7_live_test.go:102`; `inputs_test.go:337`; `live_test.go:680` | Capacity, publication, revoke; exported helper | Retry present with receipts. Hand off, no edit. |
| `at/usage/live_test.go:722,729` | `nodeMutation`: capacity, publication/disable, revoke and DELETE retirement, including reenroll/cleanup helpers | Exported helper and receipt; retry present. Hand off, no edit. |
| `at/updates/installer_test.go:424` | DELETE retirement with `retiredNode` desired-state predicate | Exported helper and `fixture_retirement_node_write` receipt; installer cadence unchanged. Shared publisher/revoke paths inherit their retries. |
| `at/isolation/driver.go.txt:655,866,1403,2553,2956,3116` | Publication and capacity changes, including change/restore/hold; shared helper | Bounded retry present, all calls pass receipt callbacks. Inherited or explicit 1 s cadence can conflict; no edit. |
| **`at/isolation/assist.py:22`** | Python-generated Go directly GETs then capacity PATCHes with If-Match; **one attempt, no reread/retry** | **Can lose the write on a report ETag race.** Out-of-owned-path request below; never executed in this run. Its explicit assisted-run classification must remain. |

No independent If-Match writer to an exit resource was found in `tests/e2e/**`;
publication uses the node resource. Other If-Match clients update members,
fleet configuration or policy activation, whose versions do not increment on a
node observation (`followup30_live_test.go:285`, `followup20_live_test.go:49`,
`serving.go:225`, `feed_scenario_test.go:68`, and corresponding AT clients).
They are outside this node/exit race. No owned node/exit consumer was missing a
bounded retry, so no redundant retry layer was added. Tests and run receipts
exercise the existing implementation faithfully.

## Scheduled verdict, cleanup, validation and records

The scheduled job's evaluator was run locally with the list as it is:

```sh
go run ./tests/e2e/cmd/fvpn-e2e-nightly-gate -summary docs/e2e/runs/2026-11-04/followup36/acceptance-summary.md -test-exit-code 2 -test-log docs/e2e/runs/2026-11-04/followup36/05-acceptance.txt
```

[nightly-gate.md](nightly-gate.md) and [complete output](12-nightly-gate.txt)
record **PASS**, CLI exit **0**, pass=32, expected_blocked=1, unblocked=0,
unexpected_blocked=0, fail=0, error=0. The list's sole entry remains
`member-ipv6`, with its preflight SHA-256 unchanged. No workflow was dispatched.

After acceptance, export admission was held for **600 seconds (10 minutes)**,
14:56:59–15:06:59 UTC. It ran after free and before teardown. Including the
first gate's 1260-second wait and a 60-second final-review wait, total
admission sleep was **1920 seconds (32 minutes)**; acceptance preflight itself
added zero. The [through-nightly snapshot](admission-summary.txt),
[final admission summary](final-admission-summary.txt), and
[command ledger](command-admission.jsonl) retain the waits. This post-run wait is outside the acceptance window and its
cadence query, and is not an observed D-133 overlap.

`make -f mk/e2e.mk e2e-down` completed with exit 0 in **47.165 seconds**.
Compose initially reported networks in use; the full cleanup then removed
owned auxiliary peers. Actual [inventory](06-owned-inventory.txt) at
**15:09:38.457418 UTC** proves **zero owned containers, images, networks and
volumes**. No other lane's Docker objects were changed.

[checks.md](checks.md) and [gate receipts](gate-receipts.txt) retain command
arguments, UTC bounds, exits and real output tails: e2e lint, isolation overlay
lint, tagged vet across every AT package, all non-stack e2e tests including the
nightly gate, handover, root go-lint, whole-module gofmt, whitespace and the
committed-evidence hygiene check all pass. Root lint used `go-with-disk-tmp=`
because disk TMPDIR/GOTMPDIR were already set, avoiding a new shell cleanup trap.
No outside-area gate failure was observed. A final ad hoc scope check initially
compared moving branch tips after `main` advanced; the proper merge-base plus
staged-file check passed. [Process notes](process-notes.txt) retain that
correction; no outside-path file was edited. Final documentation is checked again
for whitespace and evidence hygiene before the final commit.

No new artifact exceeds 20 MB. The 1,190,460-byte heartbeat trace is additionally
published as lossless gzip (34,241 bytes), with both forms' SHA-256 and verified
byte equality in [compression-receipts.txt](compression-receipts.txt).
[integrity.sha256](integrity.sha256) covers measured/supporting artifacts;
derived analysis/check summaries and the finalizing command ledger are excluded
to avoid circular hashes. Public command logs use `<worktree>`/`<home>` and
`$FVPN_HOLD_SH` placeholders; [publication receipts](log-publication-receipts.txt)
record their hashes and transformations. The nightly Markdown's surplus EOF
blank lines were removed, with hashes in [artifact-format-receipts.txt](artifact-format-receipts.txt);
its complete CLI output is retained separately. Earlier follow-up and AT records
are unchanged. No production default, acceptance predicate or dependency changed.

## Coverage and remaining limits

D-162 is a TEST ONLY fixture correction, not a new product reporting default.
§14.5 / API01 conflict behavior remains covered by offline helper tests and
these three live successful writes; live conflict retry was not needed.
Updater/candidate evidence covers the existing §13.5 / D-053 and D-155 path;
the 446 ms wait is a D-158 observation with the stale/live limits above.
The lease row covers the existing §16.3 / §19 AT29 control-loss subcase.
Every other scenario retains its own requirement/AT references and outcome.
No new arm64 hardware or per-platform native IPv6 proof is claimed. The members'
strict 15-second live query awaits its owner's separately authorized campaign.
There is no unresolved product-spec question introduced by this fixture change.

### Requests to orchestrator

2026-11-04 — Route the following source-audit results to the owning AT lanes;
all referenced paths are relative to `tests/e2e/` and were read only here.
The full consumer table above is the hand-off for every listed window/count
consumer, including those whose existing predicates need no change.

| Owner | File and line | Requested follow-up |
|---|---|---|
| Isolation | **`at/isolation/assist.py:22`** | The explicit operator-assist capacity PATCH still uses a single GET/If-Match attempt. Adopt the shared bounded node-write helper, retain every attempt/reread, and add an offline conflict test. Preserve `assisted=true`; this is not authorization to run the assist or re-score any campaign. |
| Members | `at/members/forwarding_test.go:71`; `d146_live_test.go:72,130`; `populated_hold_test.go:119`; `second_exit_test.go:140,270`; `identity_nodes_test.go:195` | Request 1's fixture change is delivered. Keep the 15 s bound, report/lease joins and all traffic checks; assess F22-D146 only in a separately authorized owner run. Existing node-write retry is present. Retain newly visible report states/counts instead of normalizing them to the old cadence. |
| Enrollment | `at/enrollment/freeze_test.go:174,254`; `handoff_test.go:534`; `node_write_test.go:18` | Review denser regular-node receipt/freeze/restart coverage. Installer cadence and heartbeat bound are unchanged; shared write retries already exist. |
| Security-ops | `at/secops/followup8_live_test.go:1194`; `followup10_live_test.go:478,551,591`; `followup13_restore_test.go:58,92`; `rate_classes_live_test.go:43`; `node_write_test.go:13` | Preserve RTO/window/error/counter and exact rate assertions. More receipts can change observed RTO or transient coverage; invalid-request rate tests are a separate subject. Shared write retries already exist. |
| Usage | `at/usage/live_helpers_test.go:567,623,650,729,748,792`; `live_test.go:722,859` | Preserve provisional/serving joins, timing bounds, forbidden LEASE_EXPIRED count and accounting assertions. More samples may expose an error. Shared write retries already exist. |
| Updates | `at/updates/live_test.go:155,645`; `authority_test.go:170`; `preresult_test.go:155,262`; `restated_test.go:100`; `installer_test.go:424` | Preserve exact report-ID/authority assertions and full stopped-agent expiry window. Installer retirement already has bounded retry. This run's wait evidence is described above; stale-to-live remains unobserved. |
| Isolation | `at/isolation/driver.go.txt:129,1546,1913,2283,2417,3082,3194`; `observer_main.go:119` | The main agent's pre-enrollment override remains 1 s, so D-162 does not directly change its normal cadence. Audit auxiliary inherited configs and preserve all coherent-window, sample and incident checks. Normal node writers already use bounded retry. |

2026-11-04 — Carry forward **AT29 guest-clock rollback as operator-run** and
**member-ipv6 as listed**. No owner record, acceptance predicate, expected-blocked
entry, product default or historical evidence was edited. D-162 remains the
orchestrator's provisional fixture decision; any owner reversal should change
that decision explicitly, not widen freshness predicates.

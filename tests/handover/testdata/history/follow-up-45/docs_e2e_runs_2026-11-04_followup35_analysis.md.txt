# Follow-up 35 — candidate recovery and first service

2026-11-04. **F30-01 is closed by updater 24, agent-core 49 and updater 25**
with acceptance run `8a436a87-5efe-4c18-a7ec-72be170cb69b`: **32 pass / 1 blocked /
0 fail**. The candidate started under its parent's update lock, deferred recovery,
cleared recovery after release, served members, and then passed `lease-expiry`.
The unchanged scheduled-job evaluator returned **PASS**. `member-ipv6` remains
blocked and listed. No scenario or acceptance run was retried.

This run **did not exercise a contended acquisition**: both updater invocations
wrote no wait record. It therefore does not establish that the new ten-second
budget caused the changed `update-good` outcome. The functional candidate path
is proved; the earlier ordinary-agent contention is not reproduced.

## Source, admission and execution

- Baseline: `5c21e670596f5b0b7029f6a02648305a24f0f22a`, including updater 25
  [#463](https://github.com/TED-H-com/Freedom-VPN/pull/463). Updater 24
  [#446](https://github.com/TED-H-com/Freedom-VPN/pull/446) and agent-core 49
  [#454](https://github.com/TED-H-com/Freedom-VPN/pull/454) were already present.
- Measured code: `9fbff94c84b0d29c2d78f420e76726f7c9a706ad`. The launcher retains
  each invocation's complete combined output, PID, UTC start/end and exit status.
  `update-good` and `update-bad` predicates and the nightly list are unchanged.
- Command: `FVPN_E2E_RUN_LABEL=followup35 make -f mk/e2e.mk e2e-test`.
  UTC **09:39:25.935949–10:03:20.352841**, **1434.417 seconds**, make exit **2**.
  The nonzero acceptance result is the one blocked IPv6 row, not a failed row.
  See [summary](acceptance-summary.md) and [complete public log](05-acceptance.txt).
- [Before-run record](before-run.txt) contains the baseline, measured commit and
  `go env GOCACHE` result, `<home>/.cache/go-build`. TMPDIR and GOTMPDIR used
  `<worktree>/.agent/tmp` on disk. No cache path was hard-coded.
- **Pre-run contention wait: 660 seconds (11 minutes)**. Preflight took
  **690.891 seconds**, including the required 30-second `vmstat 5 7` sample.
  A separate earlier vet admission waited about 180 seconds; it is outside the
  acceptance run. No reservation was taken.
- [Host admission](host-admission.txt) records every preflight answer and foreign
  process match, including the free/no-match recheck after [vmstat](host-vmstat.txt).
  The exact requested `ps -eo pid,etimes,args` / bracketed grep inventory was run;
  public foreign-process fields are only pid, comm and start ticks.
- [Five-second host samples](host-overlap.txt) found **no foreign overlap** in
  this run. There is no observed D-133 confounded window. Sampling does not
  exclude activity entirely between samples.

## D-160: measured sections

[The command/section ledger](admission-timing.jsonl) and
[its audit](timing-review.txt) show **30 named sections**, **2041 wrapped commands
inside them**, **zero admission checks inside them**, and **zero held answers
throughout acceptance**. Every section has its own preceding free answer and
UTC start/end. The intervals conservatively encompass scenario setup, the first
deadline/start read, every sample and deferred cleanup. Related billing and
member-traffic rows share named sections; each scenario record names its section.
The complete list of names and intervals is in the audit.

| Section | Admitted (UTC) | Start (UTC) | End (UTC) |
|---|---|---|---|
| update-good | 09:52:21.887429 | 09:52:21.891633601 | 09:52:38.460819430 |
| update-bad | 09:52:38.583792 | 09:52:38.587332266 | 09:52:47.179831608 |
| lease-expiry | 09:52:47.248949 | 09:52:47.253042930 | 10:03:07.361757244 |

The Go runner admits before exporting its section name. Child wrappers inherit
that name and execute without checking or waiting for admission. An attempted
nested admission is rejected. Outside sections, wrappers admit each tool,
including recursive make. The original host Go guard can still reject work;
no timed scenario waits or retries on its behalf. See
[the harness contract](../../../ADMISSION.md). Offline tests prove held/free
waiting before a section and a host becoming held inside a section without a
new check, wait or contaminated tool-output stream. Both tests also pass in the
full run's verbose log.

**Follow-up 34 had no recorded admission wait inside a timed section.** Its
entire acceptance interval, **05:37:34.607929–06:01:08.734172 UTC**, contains
**2029 recorded checks, all free**. Its first denial was **06:01:57.594346**,
after acceptance ended; teardown admission cleared at **06:27:58.872950**.
The [audit](followup34-admission-audit.txt) cites the preserved prior ledger and
[prior public admission summary](../followup34/admission-summary.txt). The three
previously disclosed missing setup-sub-make checks remain an historical gap;
this is not a claim that those checks existed. No earlier record was rewritten.

## Updater output and candidate path

Both scenario JSON records contain the cumulative `updater_invocations`
observation, including complete output without excerpting it. The
[retention receipt](updater-output-retention.txt) verifies that update-bad still
contains update-good's identical receipt and output. The offline two-invocation
test proves mixed stdout/stderr, a missing terminal newline, exits 0/3 and both
start/end receipts survive. The latest-output compatibility files remain.

| Record | Invocation / PID | Start–end UTC | Exit | Complete output / wait record |
|---|---|---|---|---|
| [update-good.json](update-good.json) | `invocation.uoern8lE` / 178784 | 09:52:28.264063978–09:52:37.470593496 | **0** | **0 bytes**; latest output was readable; no wait record |
| [update-bad.json](update-bad.json) | `invocation.yIuYeknH` / 186533 | 09:52:43.478106458–09:52:46.494609725 | **1** | `fvpn-updater: updater: HASH_MISMATCH: artifact` followed by newline; no wait record |

Both reached their updater operation, so absence of the D-158 record here means
first-attempt acquisition, not an unobserved wait. **There is no wait interval
and hence no holder to attribute during a wait.** The separate lock samples
positively identify PID 178784 as `fvpn-updater` during the good operation, PID
186533 during the bad operation, and the candidate's own subsequent agent holds.
They do not retrospectively assign a holder to follow-up 34's failed instant.
`update-bad`'s false `running_candidate_executable` is relative to the bad target;
the successfully installed good candidate remained running.

The [lossless lock/heartbeat trace](update-recovery-heartbeats.txt.gz),
[agent log](update-recovery-agent.txt) and
[selected exact samples/log lines](recovery-proof.txt) establish this sequence:

1. **Parent lock:** sample **09:52:28.339161088** identifies lock holder **178784 /
   fvpn-updater**, while old agent **895** is still running. The parent remains
   the sampled holder through **09:52:37.440685257**. The next sample,
   **09:52:37.654699088**, has no holder. Its exit receipt ends at
   **09:52:37.470593496**.
2. **Candidate started:** agent log line 1 begins boot at **09:52:31.986**.
   Sample **09:52:32.026315972** identifies PID **181291** executing
   `/opt/vpn-node/releases/01a1a5ee-f13e-7155-bea1-b22ce5a34a9f/fvpn-agent`,
   with parent 178784 still holding the lock. Its boot heartbeat is non-serving.
3. **Boot recovery deferred:** at **09:52:32.243095448**, that same candidate
   reports `serving=0`, `reason=recovery_required`, generation 93 while the
   parent owns the lock. The agent log records `updater_barrier` during boot
   and the subsequent parent-held pass. This is the D-155 deferred-recovery path:
   `node/agent/engine.go:544` calls `recoverOnce` before reading the latch;
   `updaterecover.go:40–60` leaves recovery blocked when the update lock is held.
   `baselineReason` gives the latch precedence, explaining the log's
   `updater_barrier` rather than a separate `updater_recovery` label. This state
   interpretation uses that runtime contract together with the measured boot,
   parent ownership and heartbeat; it is not a logged child-exit event.
4. **Deferral cleared by 09:52:42.112:** agent log line 18 has
   `cause=policy_missing`. `node/agent/measurement.go:70–84` gives recovery
   precedence over this cause, so recovery is no longer blocked by that log
   event. This follows the observed parent release. The exact recovery-child
   completion timestamp is not logged; the evidence bounds clearance after the
   parent-held phase and by **09:52:42.112**.
5. **First service:** agent log line 22 opens the meter epoch at
   **09:52:42.247**. The first serving heartbeat is written at
   **09:52:42.902971300** and sampled at **09:52:42.906726464**, generation **95**,
   `serving=1`, `reason=serving`, `forwarding_blocked=0`, same PID **181291**.
   The meter log alone is not treated as a successful member flow. The following
   lease scenario proves an authenticated HTTP 200 and increased member
   counters, with its captured member connection beginning **09:52:50.821575**.

`update-good.json` independently records the exact candidate executable,
`locally_succeeded=true`, configured probe issuance and verified traversal.
The observer acquired no lock and changed no gate. Short missing-PID/executable
reads during the old process's stop are retained in the original trace; they
are not interpreted as a holder or service state.

## Lease expiry after candidate service

All scenario fields below are from [lease-expiry.json](lease-expiry.json).
Heartbeat/agent times come from the same [recovery trace](recovery-proof.txt).

- **countedFlow returned true:** `controlled_origin_http_200_and_counted=true`;
  member admission bytes increased **30333 → 32964**. The readiness poll began
  **09:52:47.253068283**, found the counters, and took **1.293275146 seconds**.
  Established TCP and UDP each had a positive reply before the fault.
- **Control stop:** requested **09:52:59.831981062**, completed
  **09:53:00.135616864**.
- **D:** generation **98**, signed desired-policy expiry
  **10:02:57.306114086 UTC**. The installed monotonic deadline was
  **110019755610444 ns**. This is the desired-policy boundary, not an isolated
  proof that a longer member lease expired. Converting the sampled heartbeat's
  wall/monotonic pair gives about **10:02:57.304235879**, within 1.879 ms of the
  signed wall deadline; the monotonic deadline is the enforcement reference.
- **Forwarding closed:** the existing two-second rule poll waited
  **9m58.301032150s**, then recorded **558 → 0 member accept rules** and
  `wait_finished_at=10:02:58Z` (whole-second precision). Agent baseline write
  for `policy_expired` begins **10:02:57.812** and succeeds **10:02:57.955**.
  The first closed heartbeat is written **10:02:58.052170124**, about **746 ms
  after signed D**, and sampled **10:02:58.110610199**. The preceding sample
  carries a stale serving heartbeat written at **10:02:53.035427572**; it is
  not evidence of packets forwarding after D. These are observations of
  baseline/heartbeat/rules, **not an exact last-forwarded-packet timestamp or
  a subsecond cutoff guarantee**.
- **Closed-path quiet checks:** active senders continue. TCP attempts increase
  **1201 → 1204**, UDP attempts **1205 → 1211**, while TCP echoes remain **1196**,
  UDP replies **1196**, upload bytes **239265**, and download bytes **182826**.
  `active_senders_after_cutoff=true` and
  `established_replies_and_bytes_stopped=true`.
- **Control restart:** requested **10:03:04.755285301**, ready check succeeds at
  **10:03:06.925194492**, `control_restarted=true`.

This covers the existing ENT02 / ISO02 / SEC01 control-loss subcase of §19 AT29.
The independent guest-clock rollback row remains
[operator-run](../../../at29-guest-clock.md); no guest or host clock was changed.

## Every observed row and attribution

Only `update-good` changes status from follow-up 34. The records now prove the
candidate's D-155 deferral/clearance implemented by updater 24 and agent-core 49,
which follow-up 34 did not reach. **No merged change can be assigned as the
cause of this run avoiding its earlier acquisition collision**: updater 25 is
present, but neither invocation used its wait budget. No other causal merge
attribution is made.

| Scenario | Follow-up 34 | Follow-up 35 |
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
| [update-good](update-good.json) | blocked | pass |
| [update-bad](update-bad.json) | pass | pass |
| [lease-expiry](lease-expiry.json) | pass | pass |
| [revoke](revoke.json) | pass | pass |
| [second-exit](second-exit.json) | pass | pass |

## Nightly verdict, cleanup and checks

The scheduled job's unchanged evaluator was run locally; no workflow was
triggered or dispatched:

```sh
go run ./tests/e2e/cmd/fvpn-e2e-nightly-gate -summary docs/e2e/runs/2026-11-04/followup35/acceptance-summary.md -test-exit-code 2 -test-log docs/e2e/runs/2026-11-04/followup35/05-acceptance.txt
```

[Verdict](nightly-gate.md): **PASS**, CLI exit **0**, pass=32,
expected_blocked=1, unblocked=0, unexpected_blocked=0, fail=0, error=0.
[Complete evaluator output](13-nightly-gate.txt) is retained. The sole expected
entry is still `member-ipv6`; no nightly entry was added or changed.

`make -f mk/e2e.mk e2e-down` completed with exit 0 in **47.598 seconds**, without
an admission wait. Although Compose initially reported networks still in use,
the full teardown removed the owned auxiliary peers. The subsequent actual
[inventory](06-owned-inventory.txt), at **10:05:45.877729 UTC**, contains **zero
owned containers, images, networks and volumes**.

[Gate commands, exits and real output tails](checks.md) include e2e lint,
isolation lint, tagged vet across all AT packages, non-stack units, handover,
root go-lint, whole-module gofmt and evidence hygiene. Root lint used the
existing `go-with-disk-tmp=` override because TMPDIR/GOTMPDIR were already on
disk; it did not install a shell cleanup trap. No outside-area gate failed.

Process corrections are preserved in [notes](process-notes.txt): a unit-test
admission waiter was mistakenly queued while vet waited, then stopped before
Go execution; the actual gate ran only after vet completed. The first hygiene
check rejected wording in a publication receipt that accidentally matched the
home-path filter; the unchanged test passed after that wording was fixed.
The nightly generator's surplus EOF blank line also failed staged whitespace;
only that blank line was removed, with both hashes retained. None of these was
an acceptance attempt or a scenario retry.

[Compression receipts](compression-receipts.txt) retain SHA-256 for both forms
of the complete lock trace (1,169,445 original bytes, 33,559 gzip bytes), with
lossless round-trip verification. Compression also preserves trailing sample
spaces. No new file exceeds 20 MB. [Publication receipts](log-publication-receipts.txt)
and [record hashes](integrity.sha256) distinguish formatting/path normalization
from unchanged measured data. Runtime was amd64; release preparation builds
amd64 and arm64, with no real arm64-device claim.

The update fixture covers UPD02 / UPD05 at §13.4, §13.5 and §13.9, and the
existing §19 AT23 / AT25 / AT27 slices. It is not a full multi-device or
physical-arm64 acceptance campaign. SEC02 evidence hygiene is checked below.

### Requests to orchestrator

2026-11-04 — Update F30-01 tracking and the downstream handover to **closed by
updater 24, agent-core 49 and updater 25**, citing run
`8a436a87-5efe-4c18-a7ec-72be170cb69b`. Preserve the attribution limit: this run
proves candidate deferred recovery and service, but has no contended updater
acquisition and cannot credit the wait budget for avoiding the prior collision.
Keep AT29 guest-clock rollback operator-run and `member-ipv6` listed. No new
outside-path implementation request or nightly quarantine entry is needed.

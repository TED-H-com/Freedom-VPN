# Follow-up 34 — acceptance on the merged updater and agent fixes

2026-11-04. Run `af4c130b-5029-401b-bd0f-e8afaad3d37f` completed once:
**31 pass / 2 blocked / 0 fail**, make exit **2**. `lease-expiry` passed;
`update-good` was blocked by updater exit **3 before candidate start**.
Consequently this run does **not** prove F30-01's candidate recovery path.
No scenario was retried and no acceptance predicate changed.

## Source, admission and limits

- Base: `33f0ef8e891525e291d60eec9d513bda07f57c82`, including updater 24
  [#446](https://github.com/TED-H-com/Freedom-VPN/pull/446), agent-core 48
  [#447](https://github.com/TED-H-com/Freedom-VPN/pull/447), agent-core 49
  [#454](https://github.com/TED-H-com/Freedom-VPN/pull/454), and wire-nodes 35
  [#443](https://github.com/TED-H-com/Freedom-VPN/pull/443).
- Measured code: `ab0899e6799a4d992c87215c5a6705e7e2fa506f`, after comment
  correction `80cbcfc2`. Added observations record the existing poll, control
  stop/start and quiet samples; a joined read-only container observer samples
  the lock holder and heartbeat. It never acquires the lock or changes a gate.
- Command: `FVPN_E2E_RUN_LABEL=followup34 make -f mk/e2e.mk e2e-test`.
  UTC **05:37:34.607929–06:01:08.734172**, **1414.126 s**; the acceptance test
  itself took **1360.50 s**. See [complete sanitized log](05-acceptance.txt).
- [Pre-run record](before-run.txt): `go env GOCACHE` returned
  `<home>/.cache/go-build`; TMPDIR/GOTMPDIR were `<worktree>/.agent/tmp`.
  Baseline, measured source and the unchanged list's SHA-256 were recorded
  before launch. List SHA-256: `b31adc8d5d1906a91f4556c9715a9f254fd7ae6416e8c9d3d486e13f61847314`.
- No other assembled-run/lab match appeared at admission or after the required
  `vmstat 5 7`. **Contention wait: 0 s** (initial check elapsed **0.064 s**);
  the mandatory vmstat sampling took 30 s. See [admission](host-admission.txt)
  and [seven vmstat samples](host-vmstat.txt).
- The requested `ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test'`
  was run before launch. Monitoring also checked tagged test processes and
  excluded this runner's ancestors/descendants. Published foreign process
  metadata is limited to pid, comm and start ticks.
- **D-133: confounded.** Foreign gate/test processes appeared after launch,
  first sampled at **05:41:01.011401Z**, with further overlap during the control
  outage. [All five-second samples](host-overlap.txt) retain the windows; absence
  in a sample does not exclude shorter intervening activity. No timing comparison
  or isolation claim is made and no overlap-triggered retry was attempted.
- **Admission-wrapper gap:** the outer make was admitted, but GNU Make used
  absolute `/usr/bin/make` for its three setup sub-makes (`e2e-down`, `e2e-up`,
  `e2e-browser`), bypassing the PATH wrapper. Their Go/Docker/npm commands were
  separately admitted, and all admission checks before acceptance ended returned
  free. The local wrapper's argv[0] was corrected for subsequent commands. The
  three missing per-make checks are a process limitation, not fabricated checks.
  Later teardown admission deferrals are recorded separately from test results.
- Runtime was amd64. The release preparation built both architectures; there
  was no physical arm64 run. No reservation, workflow dispatch, push, PR or tag.

## Every scenario, compared with follow-up 29

The old result remains at
[follow-up 29 after-full](../../2026-11-03/followup29/after-full/acceptance-summary.md).
The [current summary](acceptance-summary.md) and linked records are authoritative.

| Scenario | Follow-up 29 | This run |
|---|---|---|
| [stack-health](stack-health.md) | pass | pass |
| [control-metrics](control-metrics.md) | pass | pass |
| [gateway-principals](gateway-principals.md) | pass | pass |
| [admin-login](admin-login.md) | pass | pass |
| [backup](backup.md) | pass | pass |
| [threat-feed-central](threat-feed-central.md) | pass | pass |
| [serving-catalogue](serving-catalogue.md) | pass | pass |
| [portal-billing-return](portal-billing-return.md) | pass | pass |
| [billing](billing.md) | pass | pass |
| [release-register](release-register.md) | pass | pass |
| [enroll-bootstrap](enroll-bootstrap.md) | pass | pass |
| [agent-supervisor](agent-supervisor.md) | pass | pass |
| [desired-state](desired-state.md) | pass | pass |
| [threat-feed-node](threat-feed-node.md) | pass | pass |
| [node-restart](node-restart.md) | pass | pass |
| [publish](publish.md) | pass | pass |
| [forwarding](forwarding.md) | pass | pass |
| [usage](usage.md) | pass | pass |
| [member-traffic](member-traffic.md) | pass | pass |
| [established-transports](established-transports.md) | pass | pass |
| [forced-relay](forced-relay.md) | pass | pass |
| [exit-dns](exit-dns.md) | pass | pass |
| [member-ipv6](member-ipv6.md) | blocked | blocked |
| [worker-restart](worker-restart.md) | pass | pass |
| [device-revoke](device-revoke.md) | pass | pass |
| [native-scale-hook](native-scale-hook.md) | pass | pass |
| [gateway-privacy](gateway-privacy.md) | pass | pass |
| [threat-feed-tamper](threat-feed-tamper.md) | pass | pass |
| [update-good](update-good.md) | pass | blocked |
| [update-bad](update-bad.md) | pass | pass |
| [lease-expiry](lease-expiry.md) | blocked | pass |
| [revoke](revoke.md) | pass | pass |
| [second-exit](second-exit.md) | pass | pass |

Only two statuses changed. `update-good` changed pass → blocked: the current
record's exit 3 is the explicit live-lock busy status introduced by #446.
The run does not establish which merged change, if any, caused this contention;
exit-code mapping is not proof of a newly introduced contention defect.
`lease-expiry` changed blocked → pass, but the original process stayed running
because the good update never started a candidate. Therefore its new pass cannot
be attributed to #454's candidate deferral/retry or used to close F30-01.
The observed sustained service during control loss is consistent with #447's
local-probe fix; this run alone does not isolate its causal contribution from
all other merged changes. No retrospective cause is invented.

## Updater outcomes and candidate recovery

[update-good.json](update-good.json): batch 202, signed desired instruction seen,
**exit_code=3**, `running_candidate_executable=false`, `locally_succeeded=false`,
`configured_probe_issued=false`, zero authenticated probe replies and zero grant
samples. The scenario remains **blocked** under its unchanged predicate.
The exit contract is documented in
[updater 24](../../../../updates/node-followup24-2026-11-03.md).

[update-bad.json](update-bad.json): **exit_code=1**, `HASH_MISMATCH=true`,
`locally_succeeded=false`, `running_candidate_executable=false`, and retained
`fvpn-updater: updater: HASH_MISMATCH: artifact`. The tampered artifact was
rejected and the scenario **passed**. It did not start another candidate.

[update-recovery-heartbeats.txt](update-recovery-heartbeats.txt) contains 2,960
samples from 05:50:18.963263722Z to 06:01:00.560195821Z. Every sample names
**agent PID 876**. There is no `recovery_required` heartbeat and no forwarding
latch in this window. During the good-update interval it samples PID 876 itself
holding update.lock at 05:50:27.694129866Z, 05:50:27.903036880Z and
05:50:28.125726771Z. The actual failed acquisition instant was not separately
captured, so those samples are not an exact contention-duration measurement.

[The agent log](update-recovery-agent.txt) records policy_missing baseline work
around 05:50:27 and subsequent meter epoch opening; it contains no candidate
boot recovery event in this window. **Candidate recovery deferral, its clearance,
and first member service by that candidate were not reached.** There is no
clearance timestamp to supply. This distinguishes an unexercised fix from a
failed recovery fix. [Agent-core 49's note](../../../../node/followup49-2026-11-03.md)
explains the intended D-155 path; this run cannot supply its missing live proof.
The watchdog log was read successfully and was empty, retained as
[update-recovery-watchdog.txt](update-recovery-watchdog.txt).

## lease-expiry: admission, D, quiet checks and recovery

All scenario fields below are in [lease-expiry.json](lease-expiry.json) and
[its readable record](lease-expiry.md).

1. `countedFlow`'s 90-second readiness poll started **05:50:38.407521373Z**,
   found admission in **243.761994 ms**, and the counted fetch succeeded.
   Before control stop, the node had **558 member accept rules**, generation 96,
   heartbeat `serving=1`, `reason=serving`.
2. Control stop was requested **05:50:51.720957715Z**, and returned
   **05:50:52.036088650Z**. [Docker read-back](control-outage-container.txt)
   confirms stopped, exit 0, `FinishedAt=05:50:51.807901149Z`.
3. Heartbeat's installed **D = 95489861608303 monotonic ns**. Using that
   heartbeat's paired wall/monotonic samples gives **06:00:47.410233392Z**.
   The signed desired policy expires at **06:00:47.412266442Z**.
   This is the earlier policy bound, **not expiration of the longer lease**:
   [the issued lease snapshot](control-outage-deadlines.txt) has node expiry
   **2026-11-05T05:50:47.683761946Z** and earliest member lease/effective-service
   end **2026-11-04T06:39:43.614406Z**. The first
   [lease extraction](control-outage-issued-leases.txt) queried the absent
   member `expires_at` key (null); the later extraction explicitly uses
   `lease_end`. Both observations are retained.
4. The unchanged two-second rule poll waited **9m56.775499755s** after control
   stop and found **0 member accept rules**; its second-resolution completion
   record is **06:00:48Z**. The agent log records its first `policy_expired`
   baseline begin at monotonic **95489893967035** (**D + 32.358732 ms**), and
   successful end at **95490039177172** (**D + 177.568869 ms**). The first
   closed heartbeat is monotonic **95490091856257** (**D + 230.247954 ms**),
   wall **06:00:47.640480206Z**, sampled at **06:00:47.642163577Z**:
   `serving=0`, `reason=policy_expired`, verified baseline, generation 96.
   These are state/write observations; the harness did not capture an exact
   last-positive packet or prove a packet-level cutoff bound. Kernel timeout
   behavior must not be inferred from a later baseline write. The window is
   additionally confounded under D-133.
5. Closed-path checks passed with senders still active. Across the two samples,
   TCP/UDP replies stayed **1193 / 1193**, upload/download bytes stayed
   **230975 / 176578**, while TCP attempts increased **1198 → 1201** and UDP
   attempts **1202 → 1209**. Both `active_senders_after_cutoff` and
   `established_replies_and_bytes_stopped` are true.
6. Control restart was requested **06:00:55.403248933Z**, and readiness was
   confirmed **06:00:57.725503990Z**. The same PID 876 next reports service at
   wall **06:00:59.615113403Z**, monotonic **95502066485053**, generation 97,
   observed at **06:00:59.700324599Z**. This is recovery after control restart,
   not candidate boot-recovery clearance.

This is the existing AT29 control-stop subscenario. It does not complete all of
AT29 (§19), test standalone expiry of the longer lease, execute guest-clock
rollback, or establish an unconfounded performance bound. Security-ops 15's
repeated guest request is already answered by
[the existing operator-run response](../../../RESPONSES.md#security-ops--2026-10-31)
and [the guest procedure](../../../at29-guest-clock.md); no new work is added.

## Quarantine decision and F30-01

The list was unchanged for the whole acceptance invocation. Because
`lease-expiry` passed, its entry is eligible for the requested removal; the
`member-ipv6` entry must remain byte-identical. The before/after gate outputs and
actual list edit are appended below when performed on these same run records.

**F30-01 remains open for missing candidate proof.** The prompt's closure branch
assumes that the successful lease row follows the real candidate update path.
Here update-good stopped at ExitBusy. Removing a passed row's quarantine is
not proof that updater 24 and agent-core 49 recovered a candidate. This conflict
was raised during execution; absent a different disposition, the truthful
hand-off keeps the finding open and requests a separately authorized future
candidate proof. No scenario predicate or result was changed to obtain a pass.

## Requests to orchestrator — 2026-11-04

- Updater / agent-core: investigate the blocked good-update invocation (exit 3,
  no candidate or health challenge), using update-good and the lock/heartbeat
  trace above. Decide how to obtain the missing D-155 candidate proof in a future
  authorized run. This session used its one full invocation and did not rerun.
- Handover/status: retain F30-01 as unproved/open despite the lease row's pass;
  do not describe this run as live proof of #446 + #454 candidate recovery.
  Historical r29/follow-up-29 evidence and handover's frozen claims stay intact.
- Retain D-133 confounding and the three setup-sub-make admission omissions
  when reviewing the run. Neither is silently converted into a clean gate.

Publication trims terminal line whitespace from Docker progress, vmstat and lock-observer text; timestamps and values are unchanged. [Normalization hashes](publication-normalization.txt) identify pre-trim and published bytes. The first staged whitespace check found those capture spaces and was corrected before the records commit.

## Completed quarantine and gate decision

The unchanged-list command was:

```sh
go run ./tests/e2e/cmd/fvpn-e2e-nightly-gate -summary docs/e2e/runs/2026-11-04/followup34/acceptance-summary.md -test-exit-code 2 -test-log docs/e2e/runs/2026-11-04/followup34/05-acceptance.txt
```

[Before removal](nightly-gate-before.md), the verdict is **fail**, exit 1:
pass=31, expected_blocked=1 (`member-ipv6`), unblocked=1 (`lease-expiry`),
unexpected_blocked=1 (`update-good`), fail=0, error=0. This is the scheduled
job's local evaluator on the actual log/exit status, not a dispatched workflow.

The passed `lease-expiry` entry was then removed. The member-ipv6 JSON object
was compared byte-for-byte against the original and is unchanged (SHA-256
`c017b0a1232be1873486aee5c7cddcf2e15e581145a3743c489f039023958f8a`). It remains the sole listed entry. Thus r29 L-2's stale
live citations and ordering note are gone with the entry. The frozen historical
copies deliberately retain the original text.

[After removal](nightly-gate-after.md), the **same run** still gets **fail**,
exit 1: pass=31, expected_blocked=1, unblocked=0, unexpected_blocked=1,
fail=0, error=0. No entry was added for update-good and no result was suppressed.

The existing E2E nightly-gate tests used the live list for historical follow-up
29 cases. They now load the byte-identical frozen
`tests/e2e/testdata/nightly_expected_blocked_followup29.json`; all their existing
case assertions are retained. A separate test checks the current list contains
only the unchanged member-ipv6 entry. Handover's own frozen list, tests and
follow-up 29 records were not changed. Required test results are in
[checks.md](checks.md).

**F30-01 is not closed by this run:** updater 24 and agent-core 49 are merged,
but the good update never started the candidate. The lease quarantine removal
answers the observed row only. The candidate proof gap remains routed above.

## Cleanup and final verification

`make -f mk/e2e.mk e2e-down` completed successfully; see
[teardown output](06-e2e-down.txt). Admission held the next Docker command after
compose teardown; [separate admission summary](admission-summary.txt) records
the exact first-denial/first-free windows. Deferral is not a test result.
The [final D-103 inventory](owned-docker-inventory.txt) is **0 containers,
0 images, 0 networks, 0 volumes**. Shared PostgreSQL and other prefixes were
untouched. Every new evidence file is below 20 MiB, so lossless gzip is not
required. Existing historical records remain byte-identical.

An auxiliary post-run preparation script stopped because the new E2E fixture directory did not yet exist. The directory was created, and continuation resumed after the already recorded unchanged-list gate. No acceptance invocation or completed gate was repeated for this preparation error.

The final staged whitespace check found an extra terminal blank line in each generated nightly verdict Markdown. These were trimmed; [hashes](gate-publication-normalization.txt) retain both forms, and the full CLI outputs remain unchanged. The repeated staged check passed.

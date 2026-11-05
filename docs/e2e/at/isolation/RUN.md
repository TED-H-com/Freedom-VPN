> [!IMPORTANT]
> **Follow-up 27 live run status — NEWEST ATTEMPT, FAILED (`d69f5652-aed6-4fdd-9e7c-edface751237`):**
> The single permitted Follow-up 27 assembled invocation (`run_id` `d69f5652-aed6-4fdd-9e7c-edface751237`, `2026-11-05T10:11:41.296179Z`–`2026-11-05T10:33:52.446700Z`, `1331.15` s, `source_commit: 37b3c25f60194e0d2261cf9aca30dfe8fc5d93be`, `baseline/launch_main: aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`, `source_dirty: false`, `invocations: 1`, `reruns: 0`, `lab_included: true`, `confounded: false`) **failed** (`make` exit `2`; [`scheduled-job.json`](history/follow-up-27-run/scheduled-job.json) status `fail` on `AT13` `short whole-node lease expiry` (`packet 145 missing send/ingress/read` after `result: "socket_error"`); auxiliary [`clone-transport-review.json`](history/follow-up-27-run/clone-transport-review.json) status `not established` with 8/9 criteria `established` and `consumer_latch_held` `not established`).
>
> **Root-record decision:** Canonical root `AT10`–`AT13` and `support`/`reload-stream` payloads in `docs/e2e/at/isolation/` remain byte-identical to pre-run `37b3c25f60194e0d2261cf9aca30dfe8fc5d93be` and retain the last passing full run (**Follow-up 25**, `f25`, `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db`, backed by [`history/follow-up-25/archive.json`](history/follow-up-25/archive.json)). Every Follow-up 27 run artifact is sealed under [`history/follow-up-27-run/`](history/follow-up-27-run/archive.json); see [`## Follow-up 27 — 2026-11-05`](#follow-up-27--2026-11-05-newest-attempt-failed) below and [`followup27-2026-11-05.md`](followup27-2026-11-05.md).

# Isolation run procedure — Follow-up 19 (2026-11-03)

One assembled invocation only, after the D-147 and unchanged lab rows. Baseline main `27ff6392`; measured source `458b7a89`. No rebase, reset, branch recreation, reservation, GitHub workflow, push or tag.

## Procedure and receipts

1. D-098 archive first: `f5a5a425`, [140 byte-identical files](history/follow-up-18/archive.json). Check the inherited note 1 separation and unchanged `memberAcceptRule`; run the 23 reader tests and commit note 2's `retire` addition.
2. Diagnose and retain the inherited failed D-147 record; commit all embedded runner changes before each retained build. Recreate the owned lab after inherited captures and between standalone attempts. The final focused lab on `3e12f913` passes all five tests, with the existing four unchanged and in the same session. [Dated detail](d147-related-frag-needed-2026-11-03.md).
3. Run `make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build`. Invoke Python with `-B` or `PYTHONDONTWRITEBYTECODE=1`; use `.agent/tmp` for TMPDIR and GOTMPDIR, and discover GOCACHE with `go env GOCACHE`.
4. Before every Go test, make or Docker command, run `bash "$FVPN_HOLD_SH"`; wait up to 90 minutes for free. Check `pgrep -af 'gates.sh|mk/at-'` before startup and again afterward; never start with two other suite roots. Record `vmstat 5 7` before the assembled invocation. Use only owned `fvpn-w4-isolation-*` objects.
5. Run `make -f mk/at-isolation.mk at-isolation` exactly once, with `FVPN_AT_EVIDENCE_DIR` pointing to a fresh empty disk directory. This invocation includes its own separate mechanism rerun. Retain all outcomes and overlap receipts; make no retry under D-133. Publish only after it completes and teardown reports zero owned inventory.
6. Run handover tests, root `make go-lint` with the disk runner and `go-with-disk-tmp=` (no shell trap), gofmt, diff check and committed-evidence hygiene. Preserve all failed records and archive pins. Update the dated document, requests and `.agent/REPORT.md` / `.agent/last.md`; commit without pushing.

## Recorded outcome

[Receipt](follow-up-19-run-receipt.json): run `70beb28e-4766-4004-932b-510fd01f986a`, **22:08:56.001043Z–22:13:50.863036Z**, **294.862 s**, exit **2**. Source dirty false. Lab including D-147 passes. Assembled AT10/11/12/13 are **blocked**, with respectively **0/2/0, 0/2/0, 0/3/0, 0/5/0** pass/blocked/fail checks. The shared capacity PATCH returned 412; serving publication did not complete. The real-node expiry schedule was not executed and no historical result was substituted.

Admission was free (0.038 / 0.037 s checks); an earlier lab reservation wait lasted 30 s. Initial members-gate overlap is **confounded under D-133**; [samples](suite-overlap-monitor.jsonl), [preflight](host-preflight.json). No retry. [Inventory](cleanup-inventory.json) is zero. [All observed rows and limits](summary.md), [owner handoffs](requests.md).

## Retained follow-up 18 procedure

The following procedure/outcome is historical; links point into its archive. D-140/D-142 predicates remain unchanged, but no such sweep occurred in follow-up 19.

# Isolation run procedure — Follow-up 18

Baseline **`090bb97b88b4062073a487c82676d0e1faca4fac`** (including isolation 17 #410,
scheduled run `41640196805` on `487c698e6602dd721391cfdb0e5205170e114000`, and
previous night's run `41456092596` on `a6d56cff`).

D-098 archives all **138** root files and subtrees from run
`0ec14b39-467e-4a37-a239-1a33ef12eb71` byte-identically with SHA-256 pins at
[follow-up 17](history/follow-up-17/archive.json). All earlier archive files,
including `follow-up-17-stream`, are untouched. Handover claims already read
immutable paths; no D-132 exception edit is needed.

## Scheduled nightly diagnosis and harness fix

In scheduled run **`41640196805`** (`run_id` `775bc692-e6ef-4490-8056-ba6cfa818b69`),
two unlisted rows blocked:
1. **`AT13` — `short whole-node lease expiry` (`blocked`, `positive=false`):**
   `deadline()` stopped `fvpn-w4-isolation-control` and `fvpn-w4-isolation-worker`
   at `13:18:54Z` and registered a deferred `docker start` without awaiting
   control readiness (`x.waitControl()`). When `deadline()` returned at
   `13:19:22.5917021Z`, `defer x.wholeNodeExpiry()` immediately invoked
   `x.r.lab.login(x.ctx, "e2e-admin", "admin")` and its own `defer x.fleetAudits`
   while `fvpn-control` was still booting (serving its first request at
   `13:19:30.47655856Z`), receiving `502 Bad Gateway` from `nginx/1.30.0`.
2. **`AT10` — `foreign ingress redirect containment and D-096 recovery` (`blocked`, `positive=true`):**
   Because `wholeNodeExpiry()` returned before stopping control, `x.waitControl()`
   was never called and `control-observation-readiness.json` was never written.
   When `foreignRedirect()` ran at `13:19:47Z`–`13:19:57.4376555Z`, it performed a
   one-shot `os.ReadFile` on the absent readiness file and blocked without
   waiting, even though `receiver-capture.json` and `lease-history.json` show
   observations were already current and retry-drained by `13:19:40.222691621Z`.

Both rows share a single owned harness defect (`Outcome (a)` in
`tests/e2e/at/isolation/driver.go.txt` and `scheduled.go`), fixed by awaiting
`x.waitControl()` after `deadline()`'s `docker start` and verifying current
post-start `ControlObservationReady` before `wholeNodeExpiry()` login, before
`wholeNodeExpiry()`'s deferred `fleetAudits`, and in `foreignRedirect()`. See
[dated nightly diagnosis](history/follow-up-18/nightly-diagnosis-2026-11-03.md).

## Predeclared D-140 schedule and D-142 scope

Exactly one whole-node expiry sweep in one assembled run; no retry. Retain the
corrected follow-up 16 schedule: 101 phased and 101 arbitrary_37ms original
probes, same-run calibration capture copied at D−3 s, planning at D−1 s from
at least 20 original-cadence packets. Let spread=max−min member-send→node-ingress
latency and transit=max ingress→egress latency. Choose spacing
`ceil_0.1ms(max(0.5ms, spread + 2*transit + 0.1ms))`. Record every input and the
chosen spacing before the boundary.

**D-142 applies to this sample.** Order probes by observed ingress. Judge each
adjacent pair starting with whichever comes earlier in that order: the last
positive or the first drop, continuing through the end of the sweep.
`EvaluateDeadline` and its tests are byte-identical and own the verdict.

## Recorded outcome

One run **`9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`**, measured commit
**`c248db966d6870182a7c87cf7e410d7524bf56ff`**, source dirty **false**. Started
**2026-11-03T14:05:18Z**, ended **14:29:55Z**; full **1477 s**, driver
**1316.05 s**, exit **2**. GOCACHE before and after: **`$HOME/.cache/go-build`**.
Admission checks (`bash "$FVPN_HOLD_SH"` and `pgrep -af 'gates.sh|mk/at-'`) were
`free` before and after. Inventory zero.

The 103 calibration packets selected **5.8 ms**. All **234** probes are retained,
with **zero overlaps**. D-142 qualification and unchanged evaluator pass
(`164 positives / 70 drops`); last positive **D−100.551358 ms**, first drop
**D−93.753451 ms**. Both `AT13 short whole-node lease expiry` and `AT10 foreign
ingress redirect containment and D-096 recovery` **pass**. D-104 mechanism passes;
all original seven hold categories and central-age fallback pass. Pure
classification remains not exercised. AT11/AT13 remain operator-blocked on the
two listed rows. Six agent exits are 0; no install overrun. [Full result](history/follow-up-18/summary.md).

## Follow-up 20 procedure — 2026-11-03 (additive)

The sections above describe historical runs; their original linked records are frozen in [follow-up 19](history/follow-up-19/archive.json). Current procedure: archive first (`f4f3ad6e`), commit all six shared-helper migrations and early-abort diagnostics, then run lane lint/test/build. Require `$FVPN_HOLD_SH` to print `free` before every Go test, make and Docker command, capturing admission separately from command streams. Wait up to 90 minutes; no reservation. Record `vmstat 5 7`, baseline commit and `ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-'` before the run, and recheck suites after startup.

Use `.agent/tmp` for TMPDIR/GOTMPDIR, `go env GOCACHE`, Python `-B`, and a fresh empty evidence directory. Invoke `FVPN_AT_SKIP_LAB=1 make -f mk/at-isolation.mk at-isolation` once. This explicit skip leaves default/scheduled behavior unchanged and preserves all root lab records. Report overlaps as D-133 confounded and retry nothing. Only an owned harness abort before the first acceptance check permits one recorded rerun. Final gates: handover, root go-lint, formatting, diff check, committed-evidence hygiene and zero owned Docker inventory. [Dated record](followup20-2026-11-03.md).

### Follow-up 20 recorded outcome

Run `c52bea6c-cd91-4086-a779-656b01277dcc` on `037f4418`, 00:06:02.834855Z–00:32:12.735944Z on 2026-11-04 UTC (2026-11-03 EST), 1569.901 s, exit 2. No lab, no rerun. 26 pass / 2 blocked / 1 fail. Initial admission 0.035 s; no other-suite overlap sampled; final owned inventory zero. The collector failure did not stop the main driver before acceptance, so no rerun exception applied. Post-run recording fixes `4fc47dd1` have offline coverage only. [Full outcome, limits and handoffs](followup20-2026-11-03.md).

## Follow-up 21 — 2026-11-03 EST (additive)

Archive first: `8bd664bc`, 165 original root/support files. Commit M-1 and predeclare unchanged D-140/D-142 schedule: `fe2c68ec`. Run lane lint/test/build before the assembled invocation. Require `$FVPN_HOLD_SH` free before every make, Go test and Docker command; keep admission in a separate stream. Wait up to 120 minutes; no reservation. Record `vmstat 5 7`, baseline and `ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test'`; publish foreign processes only as pid, comm and ticks. Use disk scratch under `.agent/tmp`, Python `-B`, discovered GOCACHE and a fresh evidence directory.

The single authorized `make -f mk/at-isolation.mk at-isolation` invocation is consumed: `7599576e-339c-49c9-8bf5-0f710818cbda`, **02:50:38.278643Z–03:17:20.956680Z on 2026-11-04 UTC**, lab included, source clean, **26 pass / 3 blocked / 0 fail**, exit 2. No pre-acceptance owned-harness abort occurred; no retry. Admission took 0.054 s; 54 overlap samples clear. Zero owned inventory. The three blocks are native platform, full continuous fault matrix and member-expiry lease-bound prerequisite. [Detailed results and qualification](followup21-2026-11-03.md). Final gate receipts are retained in `history/follow-up-21-validation/`.

## Follow-up 22 — 2026-11-04 EST (additive)

Archive first `dfad736e` (173 byte-identical files), then commit fresh-renewal schedule and offline proofs `0b95e69a`. Require `$FVPN_HOLD_SH` free before every make, Go test and Docker command, with separate admission streams. Record `vmstat 5 7` and `ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test|[e]2e-test'`. Wait up to 120 minutes, no reservation; disk scratch under `.agent/tmp`, Python `-B`, cache discovered with `go env GOCACHE`. No lease/grant edit or predicate change.

The one authorized assembled invocation is consumed: `7b0db5f2-eab5-4534-824a-b54781eed3fd`, **06:01:16.716438Z–06:27:35.324871Z**, clean measured source `0b95e69a`, **27 pass / 2 blocked / 0 fail acceptance**, with **failed D-120 lab TCP continuity** and command exit **2**. Wait **1414.753 s** after another suite started during vmstat; no sampled run overlap; no rerun eligible. Both expiry rows pass; D-142 qualifies the whole-node bracket. Inventory zero. [Full results](followup22-2026-11-04.md). Later `2460932f` only normalizes schedule metadata to UTC and is verified offline; original live bytes are retained.


## Follow-up 23 — 2026-11-04 EST (current procedure)

D-120 retains the UDP ingress/bracket bounds and next-datagram-success rule. Under D-161, a TCP timeout is accepted only when complete checked combined captures attribute every lost transmission of the request and echo bytes to exactly one verified ≤100 ms guard bracket, render→installed is ≤100 ms, the bytes and in-order echo arrive on the same connection before trace end, and no FIN or reset occurs. Any other TCP failure, including a timeout with missing, incomplete or unparsed TCP evidence, fails clause (iv). UDP losses plus guard-attributed TCP timeouts must total at most five per 100 reloads.

[D-098 archive, D-160 audit and predeclared run](followup23-2026-11-04.md). Earlier procedure sections and results remain historical. Check host admission before the top-level command; nested Make/Docker calls do not wait inside timed sections. The assembled invocation includes netlab 36’s analyzer and complete combined captures. Preserve D-104/D-140/D-142 bounds, diagnostic exclusions and every historical failure.


### Follow-up 23 recorded outcome

Archive `29c2e8be`, D-160 audit/D-161 procedure `89accd8d`, then the one assembled invocation above. Run `328ab9e0-7ff4-495a-81c1-410ae8a7d876`: **10:29:20.007992Z–10:53:10.483784Z**, **1430.476 s**, **27 pass / 2 blocked / 0 fail**, included D-120 **pass**, exit **2**. No other-run wait (0.175 s admission checks); 48 overlap samples clear. No retry eligible or performed. Both expiry rows pass; D-142 qualifies; member expiry is M-1’s second live pass. Four internal admission checks precede timed sections; no script admission overlaps the lab campaign or assembled driver. Zero owned inventory. [All results, conflicts, restarts and limits](followup23-2026-11-04.md). The authorized live invocation is consumed.

## Follow-up 24 — 2026-11-04 living procedure

Before a new assembled run, archive the living records under immutable history with SHA-256 pins. The finite-member plan now creates its one new grant with **D=floor(A+44s)** from a fresh accepted 60-second renewal. This is a TEST ONLY fixture plan, tested for binding-bearing issue A+[10,34]s, acceptance within 2s and setup by A+34s; it is not a production latency bound. The signed member/service deadlines must equal D, an accepted serving lease must expire after D+25s and be seen before D−5s, and the control stop must complete before D. Existing grants and leases are never changed. Slower setup or acceptance blocks. Five denied member probes still require six local serving/non-baseline reads and a live node lease through the final probe.

Scheduled mode is locally usable: set `FVPN_AT_SCHEDULED=1` and `FVPN_AT_EVIDENCE_DIR=.agent/nightly-records` as the workflow does, using a fresh destination. D-163 maps only the populated-holds review's named pure_stale_generation coverage gap when its own same-run result has every other item passing and no failure or join problem. The review remains incomplete and keeps exit 1. `scheduled-job.md` lists the reason and owner, and asks to remove an entry when it becomes unnecessary. Any other gap/error stays red. D-144 acceptance mapping, expected-blocked.json, the workflow, every review exit, and D-120/D-161/D-140/D-142 predicates remain unchanged.

Use `$FVPN_HOLD_SH` before each top-level Go test, Make or Docker command, with admission output captured separately. Record `vmstat 5 7` and the requested process inventory; wait for all other suites up to the authorized total limit. Do not add admission waits inside timed sections. Record any later overlap as D-133 confounded and obey the run-specific retry limit. The follow-up-24 single invocation has already been consumed. [Diagnosis, source commits, full results and validation](followup24-2026-11-04.md).


## Follow-up 25 — predeclared F24-01 handling (2026-11-04)

The [dated plan and diagnosis](followup25-2026-11-04.md) apply to one assembled
scheduled-mode run after the archive and planner/reviewer tests are committed.
Run `make -f mk/at-isolation.mk at-isolation` with `FVPN_AT_SCHEDULED=1` and a
fresh private `FVPN_AT_EVIDENCE_DIR`. Before launch, record baseline/source,
`vmstat 5 7` and the requested process search; wait for `$FVPN_HOLD_SH` to print
`free` and for other assembled/lab/e2e runs to finish, at most 120 minutes.
No reservation. Keep admission output outside parser streams. Monitor overlap;
any observed foreign run marks the window confounded under D-133. No acceptance
retry; only an own-harness stop before the first acceptance check permits one rerun.

The D−1 s checkpoint still uses the same calibration copy and both unchanged
original cadences. Pure `boundaryplan.Plan` preserves the maximum-transit formula,
median correction, grid and exclusions. No sample is discounted. Every plan has
an added-schedule array; only a nonempty array can be plan-qualified. An empty
result records its reason, spacing, cadence gaps and every candidate's excluding
original sends. This is an explicit unqualified result, not successful coverage.
The reviewer still applies D-142 to every adjacent captured pair from the earlier
of last positive/first drop through sweep end. It preserves the independent
packet verdict and writes a failing review for malformed or unqualified plans.
An exit 1 remains a scheduled error; F24-01 is not covered by D-163.
The follow-up 24 member-expiry schedule and predicate remain unchanged.


### Follow-up 25 execution receipt

The one authorized invocation is consumed: `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db`, **15:47:50Z–16:11:27Z**,
**1416.556s**, exit **0**, scheduled **pass**. No retry. The recorded plan above
was used unchanged. The host preflight had no additional wait after the earlier
751.046s offline-gate reservation wait; 48 overlap samples found no foreign run.
[Complete receipt and every observed row](followup25-2026-11-04.md),
[original preflight](history/follow-up-25-run/preflight.json),
[original vmstat](history/follow-up-25-run/vmstat.txt).


## Follow-up 27 — 2026-11-05 (NEWEST ATTEMPT, FAILED)

Archive first: `48a7db4f11e62982ceadf23884d09796b9d09d3b` ([218 Follow-up 25 root/support/reload-stream files](history/follow-up-25/archive.json)), then commit consumer-owned runtime/stop/clone collectors, reviewers, redaction hardening and offline tests (`42012fc428618e23e11a0103983aff370de808c0` / `3cfd4e451c7bb78e8cca5511c8c6ac40dccb6220`) and integrate `main` `aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f` into clean head `37b3c25f60194e0d2261cf9aca30dfe8fc5d93be`.

Before launch, record host preflight ([`host-preflight.json`](history/follow-up-27-run/host-preflight.json): `ext4` launcher/worktree, `vmstat_exit: 0`, `MemAvailable: 172665452 kB`, `processes: []`), Docker preflight ([`docker-preflight.json`](history/follow-up-27-run/docker-preflight.json): `overlayfs` on `ext4`, empty owned `fvpn-w4-isolation*` inventory), and quiet-window admission ([`prelaunch-admission.json`](history/follow-up-27-run/prelaunch-admission.json): 3 free checks across 30 s, `0.057`, `30.136`, `30.635` s, `foreign_processes: []`).

### Follow-up 27 execution receipt (NEWEST ATTEMPT, FAILED)

The single permitted assembled invocation is consumed and **failed**: `d69f5652-aed6-4fdd-9e7c-edface751237` ([`follow-up-27-run-receipt.json`](history/follow-up-27-run/follow-up-27-run-receipt.json), [`invocation.json`](history/follow-up-27-run/invocation.json)), **2026-11-05T10:11:41.296179Z–2026-11-05T10:33:52.446700Z**, **1331.15 s**, `source_commit: 37b3c25f60194e0d2261cf9aca30dfe8fc5d93be` (`source_dirty: false`), `baseline/launch_main: aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`, `invocations: 1`, `reruns: 0`, `lab_included: true`, `confounded: false`, `make` exit **2**, [`scheduled-job.json`](history/follow-up-27-run/scheduled-job.json) status **fail**.

- **In-suite admission, overlap and teardown:** [`suite-admission.jsonl`](history/follow-up-27-run/suite-admission.jsonl) records 4 free checks (`0.02`, `0.022`, `0.021`, `0.024` s) with `other_roots: []` and `timing_sensitive: []`. [`overlap.json`](history/follow-up-27-run/overlap.json) records 45 samples with `foreign_processes: []`. [`cleanup-inventory.json`](history/follow-up-27-run/cleanup-inventory.json) records empty owned `fvpn-w4-isolation*` inventory across containers, images, networks and volumes.
- **Acceptance and lab summary (`26` pass / `2` blocked / `1` fail):** [`AT10.json`](history/follow-up-27-run/AT10.json) `9/0/0` pass, [`AT11.json`](history/follow-up-27-run/AT11.json) `2/1/0` blocked, [`AT12.json`](history/follow-up-27-run/AT12.json) `10/0/0` pass, [`AT13.json`](history/follow-up-27-run/AT13.json) `5/1/1` fail on `short whole-node lease expiry` (`packet 145 missing send/ingress/read` after `result: "socket_error"` in [`whole-node-probes.json`](history/follow-up-27-run/whole-node-probes.json), [`deadline-verdict.json`](history/follow-up-27-run/deadline-verdict.json), and [`boundary-review.json`](history/follow-up-27-run/boundary-review.json); `review_followup10.py` then raised `ValueError: duplicate probe or missing ingress` in [`assembled.log.txt.gz`](history/follow-up-27-run/assembled.log.txt.gz)). Included netlab, `D-120`/`D-161`, `D-104`, and `AT13` `lease member expiry under control loss` all passed; `9` unique logical node-write operations (`idempotency_key`s), each with `1` attempt and HTTP `202` (`0` `412` conflicts), are stored across `13` receipt occurrences (`8` in `AT10.json` — `3` check-local + `5` root observations; `2` in `AT12.json`; `3` in `AT13.json`) due to `4` cross-record copies.
- **Auxiliary stop and clone reviews:** [`stop-receipt-review.json`](history/follow-up-27-run/stop-receipt-review.json) is `reviewed` (`all_parent_receipts_matched: true`, all 6 stops `exit_code: 0`, `signal: null`). [`clone-transport-review.json`](history/follow-up-27-run/clone-transport-review.json) is `not established` (`review_clone.py` exit `1`): 8 of 9 criteria are `established` (`clone_created`, `identity_fingerprints_match`, `final_agent_stop`, `executing_release_config` on `/etc/vpn-node/agent-run.json` with `snat: 192.0.6.23`, `corrected_transport` with 5 accepted `http_status: "202"` submissions, `central_conflict_frozen`, `source_freeze_response_and_report`, `clone_stop_log_retention`), while `consumer_latch_held` is `not established` because `after-clone-creation` in [`clone-runtime.json`](history/follow-up-27-run/clone-runtime.json) (`read_start: 10:32:59.621346473Z`, `read_end: 10:32:59.726700358Z`; inner `snapshot.read_start: 10:32:59.674883482Z`, `snapshot.read_end: 10:32:59.723398052Z`) completed ~`107` ms before the parent-waiter `StartedAt` (`10:32:59.833688125Z`, sampled before `cmd.Start()` in `tests/e2e/internal/agentwait/wait.go:59-60`) for restarted source PID `911`, leaving the first source process snapshot `unavailable` (`agent_pid` `"file absent"`, `process_coherent: false`).
- **Root-record retention:** Canonical root `AT10`–`AT13` and `support`/`reload-stream` payloads in `docs/e2e/at/isolation/` remain byte-identical to pre-run `37b3c25f60194e0d2261cf9aca30dfe8fc5d93be` (`f25`, `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db`). All 135 Follow-up 27 run archive files are pinned in [`history/follow-up-27-run/archive.json`](history/follow-up-27-run/archive.json); see [`followup27-2026-11-05.md`](followup27-2026-11-05.md).

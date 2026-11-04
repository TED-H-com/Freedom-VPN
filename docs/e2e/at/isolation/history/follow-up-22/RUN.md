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

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
[dated nightly diagnosis](nightly-diagnosis-2026-11-03.md).

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
two listed rows. Six agent exits are 0; no install overrun. [Full result](summary.md).

# ISO-F6-03 agent hand-off — 2026-10-31

## Attribution from isolation 9

The retained evidence **does not identify the initial generation-0 writer**.
It rules out the proposed process-restart boundary and shows that the Engine's
verification-failure withdrawal followed, rather than initiated, the mismatch.
Watchdog involvement is plausible, not proven. No actor or timestamp is supplied
to fill that gap.

Sources: [failure correlation](../e2e/at/isolation/expiry-failure-correlation.json),
[producer operations](../e2e/at/isolation/producer-operations.jsonl),
[producer analysis](../e2e/at/isolation/producer-timing-analysis.json), and
[continuous kernel/heartbeat reads](../e2e/at/isolation/whole-node-continuous-reads.json).
All times below are UTC on 2026-10-31. The two retained clock-identity captures
report the same boot/time namespace and zero MONOTONIC/BOOTTIME offsets.

| Process run prefix | Sequence range | First / last snapshot wall time |
| --- | --- | --- |
| `5d95b245` | 1–1918 | 14:54:38.411499563 / 15:02:38.248042285 |
| `0b191fdd` | 1–1361 | 15:02:38.925582154 / 15:10:18.890608753 |
| `0af802e4` | 1–10 | 15:10:20.806148037 / 15:10:24.997354652 |

The failure lies entirely inside the second run. Its exact sequence is:

| Snapshot | Source/result | Wall time | Anchor mono-before (ns) |
| --- | --- | --- | --- |
| 953 | live, verified | 15:07:27.990943320 | 42785507314445 |
| 954 | renewal, no new lease | 15:08:01.077020886 | 42818593392021 |
| 955 | Apply, verified generation 30 | 15:08:01.179209101 | 42818695580230 |
| 956 | live, verified | 15:08:01.200731124 | 42818717102121 |
| 957 | live, verified | 15:08:04.325685104 | 42821842056180 |
| 958 | live, `GENERATION_MISMATCH` | 15:08:04.337377684 | 42821853748806 |
| 959 | blocked audit, failed | 15:08:04.405077877 | 42821921449036 |
| 960 | renewal, no new lease | 15:08:14.580953605 | 42832097324748 |
| 961 | Apply, verified generation 30 | 15:08:14.665674404 | 42832182045542 |

The generation-bearing nft read in 957 is bracketed by
`42821800904040–42821803162429`; 958's read is
`42821844519732–42821846014079`. Those are command brackets, not kernel mutation
timestamps: the successful verifier also does later precision/tc reads.
There is no intervening Engine Apply, failed verifier, or process-run change.
The unchanged Step path cannot take its verification-rejection baseline before
the failure in 958. After 958 it does withdraw and wait for a later Step, which
unnecessarily prolongs the observed outage until 961.

The first retained independent baseline read is
15:08:08.230764210–15:08:08.259486099. It has generation 0, an empty allowed set,
and heartbeat `mono_ns=42821921476024`, `serving=0`, `baseline_ok=1`,
`policy_expires_mono_ns=0`, generation 30, reason `safety_check_failed`.
That heartbeat describes the **subsequent withdrawal**, not the watchdog's
input when it decided to write. The last baseline read at
15:08:33.040969758–15:08:33.069798759 is after expiry and has `lease_expired`.
The watchdog log is empty; no contemporaneous watchdog input/decision or
baseline writer receipt survives.

The original deadline is D=`42835508568707`. All 96 early losses remain
failures: first ingress D−8,098.843780 ms, recovery D−3,298.889928 ms;
95 are outside every completed guard. The final 41 drops begin at ingress
D−62.863882 ms. No later positive passes D, but positives after the first
outage fail D-104. Nothing here reclassifies that run as passing.

## Reproduction and changes

`TestAT13_ISOF603_ExactProducerReplay955To961` reads a TEST ONLY extraction of
the original 955–961 clocks/operations. Fake control signs this test's own
generation-30 60-second lease at the original request anchor; no captured
account identity or key is copied. An emulated kernel executes the real
enforcement producer and exports its real `TimingEvidence()` samples. The
test injects the observed generation-0 state before 958, deliberately without
assigning the historical writer. The replay clock drives Engine decisions;
new producer timestamps are never substituted with historical timestamps.

Running the test against an untouched `main` source snapshot fails the
same-lease repair assertion. The same-boot restart test also fails against
that snapshot. The fixed replay queues the failed verifier report before
repair, applies generation 30 with the original lease/anchor in the same
verification pass, and adds no Engine baseline on successful repair. Failed
repair still closes; expiry at D still closes. D-126 check precedence and the
`GENERATION_MISMATCH` wire result are unchanged.

Changes within the agent lane:

- A short same-boot process replacement adopts the live rules only after
  signed desired/lease verification, original anchor/generation checks,
  fresh previous heartbeat, current release eligibility, fresh probe/DNS,
  full kernel verification, topology check and meter ownership/health.
  Adoption does not install rules or reset a timeout. Reboot, clock anomaly,
  expiry, frozen identity, uncertain observation response and updater barriers
  retain fail-closed behavior.
  The verifier uses the retained installation time sample, checked against
  the same-boot lease anchor, with authorization rebuilt from signed documents.
  This preserves the kernel's original `timeout` check while checking its
  decreasing `expires` against the live read and unchanged absolute deadline.
  Brief contention with the healthy watchdog's presence read is retried under
  a bounded lock wait, rather than being treated as failed authorization.
- An ordinary clean stop checkpoints the last usage delta and retains the
  open meter epoch. Kernel counters continue across the process gap and the
  successor uses the existing same-boot continuity path. Updater withdrawal
  still closes the epoch through the full final-sample barrier.
- A valid same-generation live mismatch immediately attempts full verified
  repair, after queuing the rejected-read observation. Foreign-tc failures
  still quarantine; a failed repair does not bypass any safety gate.
- Process cancellation does not supply kernel evidence. Serving reads use
  an independent bounded context, including the final Step read; actual drift
  or expiration of that read's own deadline still closes forwarding.
- Control work shares a budget of `min(5s, watchdog_stale/3)` per Step.
  Only HTTP calls use this deadline; local persistence keeps the caller's
  context, so an outage cannot masquerade as a SQLite failure.
  Verified heartbeats are published before control I/O and before the Apply
  lock is released. An exhausted control budget with unchanged documents and
  valid, verified rules retains them without a redundant guarded Apply.
- The watchdog reads the heartbeat before sampling now. The former order
  can mistake a concurrently replaced heartbeat for time travel; a deterministic
  test reproduces that race. It takes the shared Apply lock, re-evaluates the
  heartbeat before a write, and bounds its kernel read. Genuine stale/malformed
  heartbeat, clock/boot change, elapsed policy and missing/unreadable baseline
  still fail closed. These fixes are not proof of the historical writer.

## Evidence contract for isolation 10

Keep `test_mode=true` and the existing `test_timing_path`. Its JSONL now adds
`source=heartbeat` (the exact written state and write success) and
`source=baseline` (cause, actual start/end samples, success). Process UUID,
boot UUID and contiguous sequence numbering cover these records too. Existing
Apply/live/blocked timing objects and D-106 wire evidence are unchanged.
All ns values in the JSONL remain decimal strings.

The agent's normal stderr/journal records `baseline write`, writer, cause,
begin/end mono timestamps and success. Direct boot/planned/probe/quarantine
calls are bracketed. The existing SystemPlane executor also captures baseline
nft commands performed inside Apply rollback/quarantine in memory; those
records/logs are emitted **after Apply returns**, outside its guard.

The watchdog's normal stderr/journal records `baseline decision` and
`baseline write` with cause, heartbeat mono, policy deadline, generation and
boot identity. A decision canceled by a refreshed heartbeat is logged too.
Causes include `heartbeat_stale`, `heartbeat_unreadable`, `heartbeat_in_future`,
`boot_changed`, `clock_unavailable`, `baseline_unverified`, `policy_expired`,
`serving_without_deadline`, `baseline_missing`, `baseline_unreadable`.
An nft error is reported as unreadable, not invented evidence of absence.

For the assembled harness, retain `agent.log`, `watchdog.log`, the
complete producer JSONL across every process run, and independent kernel and
ingress/egress reads. The local agent suite retains these under the selected
`FVPN_AGENT_EVIDENCE_DIR/assembled/`, including `expiry.json.restart.json` with
real stop/start samples. Extend consumers to accept the two additional
record kinds without dropping them or inventing sequence gaps.

Isolation 10 should see no generation-0 writer or multi-second dropout under
an otherwise valid verified policy across a short restart. Continue counting
**every** ingress loss, including guarded installation losses and losses
before D near the conservative kernel cutoff. The requested literal zero
pre-D loss result must be measured; it is not inferred from a no-baseline
proof or from a passing D-104 bounded-expiry predicate.

## Updates F10-01 consult — 2026-10-31

The release-root `current/fvpn-agent` check in `node/agent/release.go` is the
intended contract, matching `packaging/installer/layout.go:agentBin()`.
`bin/fvpn-agent` is a build output, not the installed release layout. The
candidate also checks the UUID-named current target, that the running
executable and active file are the same file, the updater journal's current
release, the canonical digest of its stored manifest envelope, the manifest
release UUID and the node-agent component version. Bootstrap exceptions are
limited to the configured initial release with an absent or provably empty
journal; durable update history requires the real current-release record.
Resolution is once per process so an old executable cannot claim a switched
candidate. Devstack's fixture alignment is the appropriate fix; no release
identity code change is made here.

## Measurements and gates

The retained intermediate assembled run under
`.agent/f28-evidence/after-process-final/partial/` exposed an additional restart
defect: SIGTERM interrupted control I/O and canceled the final verifier's
commands. Run `a9ba3607`, snapshot 852 at 16:26:51.406232090Z had failed command
reads; snapshots 853–855 and 857 then recorded agent quarantine/baseline writes
at 16:26:51.689060242–16:26:51.788572845Z. The successor `e18297d7` wrote
`boot_unverified` in snapshot 1 at 16:26:51.829930652Z. The independent packet
collector failed with `Network is down`. This is evidence for that new run's
writer, **not attribution of isolation 9**. The independent bounded read and
its cancellation regression test address this path. The failed run is retained.

The final process suite artifacts are under
`.agent/f28-evidence/final-process/assembled/`. With D=`48190127802169`,
the restart began at `48187156409996` (D−2,971.392173 ms) and launched its
successor 71.929253 ms later. The agent logged retained serving at
16:37:29.706Z and adopted serving at 16:37:29.783Z, with the same deadline.
There was **no baseline write between restart and D** and no restart dropout.

The unchanged expiry predicate passed: **97 positives, then 15 drops**;
last positive D−100.359051 ms, first dropped send D−63.791150 ms, first dropped
ingress **D−63.277263 ms**, no positive at/after D or after the first drop.
There were **two drops before D**, so the requested literal zero pre-D loss
result is **not achieved**. Both are retained, consistent with the existing
conservative kernel cutoff; no D-104 allowance or timestamp was changed.
All 15 client timeouts also lack observed egress.

Renewal counted **1,518 probes, 1,458 delivered, 60 losses**. Forty-one lost
ingresses lie inside completed outer guards; 19 do not. The trace identifies
failed Apply **215 at 16:35:50.844593376Z**, final-render→installed
**135.896293 ms**, exceeding the unchanged 100 ms install bound. Its actual
`apply_rollback` baseline accounts for an additional loss window. It is not
reclassified as a successful guard. Two natural renewals were observed.

| Actual run | Completed guards | Outer median / worst (ms) | Install-command worst (ms) | Render→installed worst (ms) |
| --- | ---: | ---: | ---: | ---: |
| unchanged-main reference | 117 | 30.363491 / 54.486995 | 27.933875 | 28.026939 |
| first successful live suite | 116 | 30.941086 / 150.413539 | 71.351652 | 71.419691 |
| final assembled process suite | 102 | 30.819062 / 141.650207 | 97.371275 | 97.455300 |

The reference's 70 production Go files in agent/enforce/watchdog/meter match
branch base `0df1716d`. Reference artifacts are in
`.agent/f28-evidence/before-live/`; live artifacts above are in `after-live/`.
The earlier assembled correlation in `after-process-verified/assembled/`
also passed expiry, with two pre-D drops and **160.616422 ms** worst outer
guard, but its full suite failed the subsequently corrected unpublished
health-challenge fixture. Its three renewal-window failed installs and all
90 renewal losses remain retained. No enforcement bracket source changed;
these measured maxima do **not** meet the historical 52.596424 ms ceiling.
Inner guard time in the final process run was at most 0.010853 ms; the long
brackets are in the nft commands, not logging inserted between them. The
trace alone does not establish the cause of that command latency.

The full final process suite passed (193.922 s). The final live-suite repeat
**failed** at `followup21_netlab_test.go:162`: Apply **11**, snapshot at
16:38:43.479084850Z, had render `48260791516485` and installed-by
`48260943641788`, **152.125303 ms** against the 100 ms bound. It took the
existing rollback at `node/enforce/apply.go:179–189`; guard release did not
occur. D-133 held attribution pending equal-host measurements, with artifacts in
`.agent/f28-evidence/final-live/` and output in
`.agent/f28-gates/live-complete.log`. The earlier successful live suite is
retained but does not erase this failed final rerun. Other final gate output
is recorded in `.agent/REPORT.md`.

Node lint/unit/integration/build gates, unchanged updater/installer consumers,
plain `go test ./...` with default package parallelism, root lint/vet, and the
explicit R16/R17 gate passed. Agent and watchdog cross-compiled for amd64 and
arm64; real arm64 execution is not claimed. Final task-prefix Docker inventory
is zero. That was the pre-resume live-gate state; the D-133 measurements below supersede it.

### D-133 A/B resume — 2026-10-31

**Decision: D-133 case 3, both versions pass under the amended quiet-CPU
criterion. The earlier 152.125303 ms live-gate failure is attributed to host
contention for this delivery.** It is not reproduced on unchanged main or on
`036a4829`; all four selected live gates and all four full agent suites pass.
The earlier ownership-based stop is withdrawn. No install-bound overrun occurs
in these eight traces, including their failed operations. No production code,
install bracket, 100 ms bound, packet predicate or timestamp changed in this
resume.

The initial foreground wait recorded load1 15.30 with a devstack gate chain,
then 9–13 without a matching chain. The orchestrator's amendment identified
persistent I/O-blocked scan threads as the reason load average could not reach
6. That unrelated scan was left untouched. The amended check is no external
`gates.sh|mk/at-|go test` process and six five-second vmstat samples with
`r <= 3` and `id >= 80`, discarding the since-boot row. The retained load,
blocked-thread and idle measurements below distinguish this from the earlier
16:38Z host state (orchestrator evidence: four gate chains, two other agent
sessions, load1 about 16). Load average alone is not the comparison criterion.

A was rebuilt from a fresh `git archive 0df1716d` outside this worktree;
all **6,090 Git blobs** matched that commit. B is the unchanged production/test
source of `036a4829`. The same live runner and full agent make target ran in
A/B/A/B order, with compiler concurrency 4 and disk TMPDIR/GOTMPDIR. Source
identity, tree and archive hashes are in `source-identities.json`. The archive
has no Git metadata: its first process-suite build stopped on VCS stamping
before any test or Apply. Both process builds then used `-buildvcs=false`.
The original build failure is retained. Test binaries use the runner's same
`-p=4` settings on A and B. No reference source was patched.

Both commits run `TestD085_LiveReadNetlab` and the full agent suite, including
`TestLabD087_D104_GrantRenewalExpiryCorrelation`. Their committed scenarios
include each version's restart behavior: B additionally proves verified
adoption and performs a process restart during assembled expiry. These are
whole-suite distributions, not a claim that their process histories are
identical. Every completed guard and failed Apply is retained.

Artifacts are under `.agent/f28-d133-evidence/`, gate output under
`.agent/f28-d133-gates/`. The selected A1 live/process directories are
`A1r-live` / `A1p-process`; B1 uses `B1-*`, A2 uses `A2-*`, B2 uses `B2r-*`.
The first `A1-live` and `B2-live` attempts passed their tests but failed the
post-live CPU check substantially; both remain retained and were repeated
before proceeding. Brief other CPU-criterion failures paused the next suite
with automated re-checks on a 60-second cadence (the first manual A1
post-leg re-check began 83 seconds after the prior sample start). No external
suite was observed during the selected tests; pgrep matches for this task's own tests were identified by
process ancestry. An initial shell merely quoting the pgrep pattern in prose
was recorded as a text match, not treated as a running test.

| Leg (commit) | Live / full agent gate | UTC before → qualified after | Load1; r/b/id min–max; D-state threads, before → after |
| --- | --- | --- | --- |
| A1 (`0df1716d`) | PASS / PASS | 17:14:24 → 17:24:19 | 11.81; r=0–3, b=2–12, id=86–87%; D=3 → 11.88; r=0–1, b=11–12, id=87%; D=11 |
| B1 (`036a4829`) | PASS / PASS | 17:24:49 → 17:31:45 | 11.70; r=0–3, b=0–12, id=87%; D=12 → 11.34; r=0–3, b=2–12, id=87%; D=12 |
| A2 (`0df1716d`) | PASS / PASS | 17:34:26 → 17:41:19 | 10.64; r=3, b=0–6, id=86–87%; D=2 → 11.02; r=0–3, b=3–12, id=87%; D=3 |
| B2 (`036a4829`) | PASS / PASS | 17:45:24 → 17:51:19 | 10.50; r=1–3, b=1–12, id=83–87%; D=2 → 11.22; r=0–2, b=12, id=87%; D=12 |

All times are UTC on 2026-10-31. Each CPU range uses six five-second samples; the initial since-boot vmstat row is discarded. The after column uses the qualifying re-check when the immediate post-leg sample failed; every failed sample is listed below. Full `/proc/loadavg`, raw vmstat and pgrep output are retained.

Each quartet below is **p50 / p90 / p99 / max in milliseconds**, nearest rank over **all completed guards**, including the suite’s recovery phases. Failed operations remain separate; no failed bracket becomes a completed guard.

| Leg / suite | Completed guards | Outer guard | Install command | Render→installed | Failed Apply run-prefix / sequence |
| --- | ---: | --- | --- | --- | --- |
| A1 live | 117 | 26.300401 / 35.169083 / 38.074182 / 40.355095 | 12.797440 / 18.371892 / 24.237096 / 25.492805 | 12.939853 / 18.427315 / 24.319517 / 25.580029 | `5e461bf0/126`, `287facb6/3`, `287facb6/5` |
| A1 agent | 99 | 27.007684 / 37.626002 / 47.888882 / 47.888882 | 13.814066 / 19.590287 / 23.977357 / 23.977357 | 13.919701 / 19.701046 / 24.259678 / 24.259678 | none |
| B1 live | 117 | 27.583266 / 38.682088 / 51.672900 / 57.924611 | 13.322533 / 21.139924 / 27.784300 / 29.006529 | 13.379571 / 21.285369 / 27.845963 / 29.065518 | `c5eadc57/12`, `74bfd99e/7`, `74bfd99e/15`, `74bfd99e/22` |
| B1 agent | 103 | 29.385620 / 39.512521 / 43.600958 / 45.599321 | 14.477619 / 21.679326 / 24.866790 / 25.322706 | 14.619248 / 21.748142 / 24.970069 / 25.438686 | none |
| A2 live | 117 | 26.520890 / 34.715628 / 40.343014 / 41.614374 | 13.153604 / 19.028590 / 24.212385 / 24.757228 | 13.223928 / 19.090655 / 24.266692 / 24.804036 | `8a9b69ca/126`, `6a937bf5/3`, `6a937bf5/5` |
| A2 agent | 99 | 28.570859 / 36.034524 / 49.752900 / 49.752900 | 15.876186 / 20.004907 / 28.804759 / 28.804759 | 15.961162 / 20.095574 / 28.896648 / 28.896648 | none |
| B2 live | 117 | 28.064229 / 38.811592 / 50.464151 / 52.020393 | 14.455816 / 20.389862 / 25.755958 / 27.931510 | 14.531400 / 20.714561 / 25.798024 / 28.300645 | `4826ec8c/12`, `6910a891/7`, `6910a891/15`, `6910a891/22` |
| B2 agent | 103 | 28.165894 / 35.738524 / 41.244924 / 41.907740 | 13.597263 / 19.484211 / 24.371206 / 25.201834 | 13.752792 / 19.577714 / 24.452528 / 25.292246 | none |

Failed Apply identities and actual operation brackets (CLOCK_MONOTONIC ns):

| Leg / suite | Full run UUID / sequence | Snapshot UTC | Operation start–end (ns) | Install / release bracket |
| --- | --- | --- | --- | --- |
| A1 live | `5e461bf0-ec7a-4f5e-8253-85e39ac24543/126` | 2026-10-31T17:15:02.889900277Z | 50440366242982–50440406267046 | not reached (all render/install/release samples are 0) |
| A1 live | `287facb6-3abb-4f9e-8b17-5d62b377c666/3` | 2026-10-31T17:15:03.510787017Z | 50440995924793–50441027154836 | not reached (all render/install/release samples are 0) |
| A1 live | `287facb6-3abb-4f9e-8b17-5d62b377c666/5` | 2026-10-31T17:15:03.627714942Z | 50441112178410–50441144083390 | not reached (all render/install/release samples are 0) |
| B1 live | `c5eadc57-7dc7-46ae-8bd2-6714557b99ad/12` | 2026-10-31T17:25:27.815881628Z | 51065285041843–51065332248676 | not reached (all render/install/release samples are 0) |
| B1 live | `74bfd99e-b0b3-4b86-9d26-f1279aaf9628/7` | 2026-10-31T17:25:28.46606616Z | 51065940785616–51065982434614 | not reached (all render/install/release samples are 0) |
| B1 live | `74bfd99e-b0b3-4b86-9d26-f1279aaf9628/15` | 2026-10-31T17:25:28.634206488Z | 51066098325553–51066150574447 | not reached (all render/install/release samples are 0) |
| B1 live | `74bfd99e-b0b3-4b86-9d26-f1279aaf9628/22` | 2026-10-31T17:25:28.771866422Z | 51066241514426–51066288232115 | not reached (all render/install/release samples are 0) |
| A2 live | `8a9b69ca-e5ca-4317-98e4-11226c889913/126` | 2026-10-31T17:35:04.455026382Z | 51641928509234–51641971391189 | not reached (all render/install/release samples are 0) |
| A2 live | `6a937bf5-dff9-420b-8f35-4e7250a89e89/3` | 2026-10-31T17:35:05.125722391Z | 51642607546993–51642642089571 | not reached (all render/install/release samples are 0) |
| A2 live | `6a937bf5-dff9-420b-8f35-4e7250a89e89/5` | 2026-10-31T17:35:05.232478451Z | 51642720126788–51642748847063 | not reached (all render/install/release samples are 0) |
| B2 live | `4826ec8c-b6e2-4a10-86f9-b50c109fd781/12` | 2026-10-31T17:46:03.675699602Z | 52301148945090–52301192067330 | not reached (all render/install/release samples are 0) |
| B2 live | `6910a891-9719-4e5a-ba49-f1274ab6e492/7` | 2026-10-31T17:46:04.306590771Z | 52301784198421–52301822959430 | not reached (all render/install/release samples are 0) |
| B2 live | `6910a891-9719-4e5a-ba49-f1274ab6e492/15` | 2026-10-31T17:46:04.467952933Z | 52301940276111–52301984321705 | not reached (all render/install/release samples are 0) |
| B2 live | `6910a891-9719-4e5a-ba49-f1274ab6e492/22` | 2026-10-31T17:46:04.616529759Z | 52302098516173–52302132894232 | not reached (all render/install/release samples are 0) |

CPU-criterion failures retained (no subsequent suite began until a qualifying re-check):

| Capture | UTC | Load1 | r min–max | b min–max | idle % min–max | D-state threads |
| --- | --- | ---: | --- | --- | --- | ---: |
| A1-host: live-after/process-before | 17:13:05 | 10.19 | 0–28 | 11–12 | 68–87 | 11 |
| A1p-host: leg-after/process-after | 17:21:56 | 11.63 | 0–4 | 5–11 | 83–87 | 11 |
| A1p-host: leg-after-recheck | 17:23:19 | 10.80 | 0–1 | 11–12 | 78–87 | 11 |
| A2-host: leg-before/live-before | 17:32:26 | 11.19 | 0–5 | 2–11 | 81–87 | 2 |
| A2-host: leg-before/live-before-retry | 17:33:26 | 11.49 | 0–4 | 0–12 | 77–87 | 3 |
| A2-host: leg-after/process-after | 17:40:19 | 11.12 | 0–4 | 1–12 | 86–87 | 1 |
| B1-host: leg-after/process-after | 17:30:45 | 12.17 | 0–4 | 3–12 | 87 | 12 |
| B2-host: live-after/process-before | 17:43:57 | 9.58 | 0–9 | 9–12 | 64–87 | 11 |

The listed failed Applies are the live suite's deliberate foreign-tc/restart
negative cases. Each stops before final render/install, with no completed
guard; they are not hidden as successful samples. All four assembled traces
have zero failed Applies. Raw `operations.jsonl` and per-suite
`d133-distributions.json` retain the entire sample distribution and every
original deadline field, not only these percentiles.

This result does **not** claim identical latency distributions. B's live
outer p99 is 51.672900 / 50.464151 ms versus A's 38.074182 / 40.343014 ms;
B's live outer medians are about 4.9% / 5.8% higher. The corresponding
install-command maxima are 29.006529 / 27.931510 ms versus
25.492805 / 24.757228 ms. In the assembled suite, B's outer p99 is lower in
both repeats (43.600958 / 41.244924 versus 47.888882 / 49.752900 ms), and its
second install maximum is lower (25.201834 versus 28.804759 ms). These
measurements do not show a material install-overrun regression repeated across
the live and assembled scenarios; they do show a live-suite tail difference
that remains visible in the hand-off.

The named candidate paths were reviewed: successful install/release executor
calls are unchanged; baseline receipts are buffered only for rollback and
serialized after Apply; heartbeat writes occur after Apply returns; watchdog
presence reads are serialized by the shared lock; restart verification and
meter checkpointing occur outside the install guard. No added agent I/O sits
between installed-by and release-start. B's largest measured gap between those
samples is 0.010348 ms across both suites. This code review supports the
measurement interpretation; file ownership alone supplies no attribution.

The **52.596424 ms historical outer maximum is still not established**:
B1 live reaches **57.924611 ms** (run `74bfd99e-b0b3-4b86-9d26-f1279aaf9628`,
Apply 137 at 17:25:38.360143220Z). Its actual outer bracket is
`51075801274543–51075859199154`, comprising a 29.006529 ms install command,
0.002777 ms between commands and a 28.915305 ms release command. B2 live's
maximum is 52.020393 ms. No bracket source or tolerance was changed to obtain
the passes, and this resume does not turn the historical maximum into a new
allowance.

| Leg | Renewal delivered / probes; losses inside / outside guards | Last positive / first dropped send / first dropped ingress relative to D (ms) | Pre-D ingress drops |
| --- | --- | --- | ---: |
| A1 | 1508 / 1518; 10 / 0 | -100.109794 / -63.798798 / -63.083586 | 2 |
| B1 | 1502 / 1518; 16 / 0 | -100.285460 / -63.783329 / -63.314461 | 2 |
| A2 | 1486 / 1518; 32 / 0 | -100.164852 / -63.840533 / -63.285562 | 2 |
| B2 | 1489 / 1518; 29 / 0 | -100.373132 / -63.884386 / -63.512896 | 2 |

All four assembled expiry runs contain 97 positives then 15 drops, with no
positive after the first loss or at/after D. The two pre-D drops in each run
remain failures for the literal zero-pre-D requirement. B1 restarted at
D−2,982.110548 ms and launched its successor after 78.029325 ms; B2 restarted
at D−2,971.881932 ms and launched its successor after 78.125716 ms. Neither
Engine trace nor watchdog log records a baseline write between restart and D.
B2's first dropped ingress is **D−63.512896 ms**. All 29 B2 renewal losses
are inside completed outer guards, and every loss remains counted. In the
live 1 ms probe diagnostic the A1/B1/A2/B2 losses are respectively
999 / 1,045 / 1,038 / 1,111 out of 76,000 probes per run; none is waived.

The README startup, stored-lease, retained-model, restart, watchdog and meter
sections now match this contract. The dated release-root identity consult
above stands unchanged. The prior failed live run and every earlier artifact
remain retained; the final live gate state is now **PASS** on `036a4829`.
The node/unit/integration/build, updater/installer and R16/R17 results from
that exact production/test source remain applicable. Resume repository and
formatting checks are recorded in `.agent/REPORT.md`.

### Requests to orchestrator

2026-10-31 (D-133 resume), request 1: isolation 10 must retain the added Engine
baseline/heartbeat records and watchdog reason logs. The historical isolation
9 writer remains unattributed; do not infer it from this successful restart
proof or relabel the original failed row.

2026-10-31 (D-133 resume), request 2: coordinate the remaining literal zero
pre-D loss requirement and historical 52.596424 ms outer maximum with the
enforcement lane. Both A and B still have two pre-D expiry drops; B1's live
outer maximum is 57.924611 ms. These measurements do not request a tolerance
or establish an enforcement-owned install-overrun defect. The D-133 live gate
passes, and its earlier ownership-based escalation is withdrawn. Production
`node/enforce/*` changes remain outside this lane.

2026-10-31: the former README request is completed in this resume under the
orchestrator's explicit `docs/node/README.md` ownership clarification.

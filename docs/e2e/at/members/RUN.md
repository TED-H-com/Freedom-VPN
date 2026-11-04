Final verification: [gate results, link proof and archive audit](F22-verification.md). All required non-live gates pass; live exit 2 retains blocked cases.

# Follow-up 22 run ledger — 2026-11-04

Baseline `55f2765dac6e1c6871045ee6182d6cc23ccf8a33`; measured harness `8a6dcf34248f035a3a460fcdee3e1e19e80c9137`; main at preflight `0d9448800b9e50392b1365a54fed89d4fdb333ba`. PR bodies #454 and #461 were read. #461 implements D-157's one opportunity after central accepts the clean report tied to a verified closed recovery; it is not diagnosis-only. No product or shared-hook change is made by this lane.

**3 PASS / 7 BLOCKED / 0 FAIL; 702 passed / 8 blocked / 0 failed subchecks**. [Run analysis](followup22-2026-11-04.md), [all original subchecks](F22-subchecks.md). One pre-check harness failure followed by the one authorized rerun; no case retry. The current run's ten case JSON records retain original status/actual/expected fields.

`go env GOCACHE` returned `<home>/.cache/go-build`. No cache override. Disk scratch under `.agent/tmp`; Python `-B`, foreground commands, no reservation or new shell cleanup trap. Prefix `FVPN_E2E_PREFIX=w4-members`, port base `24000`; setup uses two resolvers and kernel observation. Both attempts use admitted `make -f mk/e2e.mk e2e-up`, `make -f mk/at-members.mk at-members`, exports, then `e2e-down`. Shared Postgres and foreign objects are untouched.

Preflight at `2026-11-04T08:19:50.370960+00:00`: `vmstat 5 7`, `go env GOCACHE`, and `ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test|[e]2e-test'`. Exact process arguments remain private; published foreign processes contain only pid, comm and ticks. [Host receipt](F22-host-preflight.json), [attempt-1 host receipt and failure](F22-attempt1.json).

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 0  0 3076952 36525932 2573020 137527888   10   39 19058 32467 25507   22  4  2 93  1  0  0
 0  0 3076952 36560924 2573028 137527876    0    0     0   101 2689 3801  0  0 100  0  0  0
 0  0 3076952 36569500 2573036 137522916    0    0     0  2126 3426 5429  0  0 99  0  0  0
 0  0 3076952 36567020 2573044 137522320    0    0     0   906 3040 4187  0  0 99  0  0  0
 0  0 3076952 36567612 2573052 137522604    0    0     0   690 2389 3540  0  0 99  0  0  0
 0  0 3076952 36567624 2573060 137523700    0    0   354   127 2500 3585  0  0 99  0  0  0
 0  0 3076952 36567624 2573068 137523700    0    0     0    52 2380 3440  0  0 99  0  0  0
```

## Attempts, admission and overlap

| Attempt | Foreground wrapper interval (UTC) | Outcome |
|---|---|---|
| 1, source `a52bad1d` | 07:52:29.002138–08:14:57.830369 | Exit 2 before `go test` or the first acceptance check: owned `admission.py` raised `ValueError: No closing quotation`. No acceptance UUID/result. Cleanup exit 0 and empty inventory. |
| 2, source `8a6dcf34` | 08:19:50.295004–09:08:52.970545 | Run `3eb3960d-c411-4648-8d77-bba40b436ae3`; honest make exit 2 for blocked cases. Acceptance command 08:20:54.441869–09:08:08.816545; live test 2,800.89 s. Cleanup exit 0 and empty owned inventory. |

The first attempt waited **840 s** in 60-second suite-admission sleeps and **420.304 s** in the nested tool's admission loop: **1,260.304 s (21 min 0.304 s)** accounted admission wait, below the total 120-minute limit. The two suite poll windows, including check overhead, were 600.726 s and 240.342 s. The retry needed no admission sleeps. Both startup and post-startup gates checked for foreign runs; no reservation was taken. The first attempt's foreign-activity windows are retained and confounded under D-133; no acceptance measurement occurred in that attempt.

The only observed retry overlap is **09:08:22.475474Z–09:08:52.654917Z**, during teardown after acceptance ended. Marked **D-133 confounded**, with zero intersecting measured case envelopes. Five-second process sampling is a bounded observation, not proof that shorter foreign work never occurred. No retry was made for this overlap or for any case result.

The retry baseline is `55f2765d`, including #454 and #461, but its preflight main ref is `0d944880`: netlab follow-up 35 #462 merged while this session was active. [Baseline drift and disposition](F22-baseline-drift.json). No result is attributed to or presented as a measurement of that newer main.

## Reproduction and record handling

The normal entry point remains `make -f mk/at-members.mk at-members`. The private foreground runner supplied disk `TMPDIR`/`GOTMPDIR`, admitted wrappers for each make/go/Docker invocation, and `FVPN_AT_MEMBERS_SECTION_HOOK`/`FVPN_AT_MEMBERS_SECTIONS` for the D-160 case envelopes and nested scope journal. It exported original agent/watchdog logs between cases and at teardown, then stopped and removed only `fvpn-w4-members-*` inventory. No shared helper changed.

Offline postprocessing uses `python3 -B tests/e2e/at/members/followup22_records.py`, guarded by the exact measured source and run UUID. It copies existing outcomes and receipt attempts; it does not execute acceptance or recalculate a live predicate. Oversize original case and recovery evidence remains lossless gzip with compressed and uncompressed SHA-256 sidecars. Plain `AT02.md` is a document index for its original gzip record. Native pcap bytes stay private; public capture exports contain headers, counts and hashes only.


[Admission checks and answers](F22-admission.json), [timed-section boundaries](F22-timed-sections.json), [five-second process monitor](F22-process-monitor.json), [cleanup](F22-cleanup.json), [gate receipts](F22-gates.json). Every tool admission stream is separate from tool-consumed output, including Docker JSON. D-160 uses admitted no-wait case envelopes covering nested timers and concurrent persistent probes; nested scopes and the original producer/request/sample timestamps remain retained. Observed overlap is D-133 confounded and never retried.

---

Historical ledger below retains its original outcomes.

# Follow-up 21 run ledger — 2026-11-03

Run `5715939e-3943-48f1-8bd5-c869bf7c91c2`; baseline **`9e70951872105139213ed39c808704fdab315112`**, source/archive commit **`dbfccc6fc7d076ec73f4f8575eb27efe39337a23`**. Main at preflight was the baseline. Agent-core 48 (#447) and members 20 (#445) are included. Source/predicates were not changed during execution. D-098 first commit `dbfccc6f` preserves all **81** starting root files byte-identically: [archive](history/follow-up-18-run2/INDEX.md), [SHA-256 pins](history/follow-up-18-run2/SHA256SUMS). Earlier archives are untouched; no handover claim reader required a D-132 edit.

**One assembled run; no rerun. 2 PASS / 8 BLOCKED / 0 FAIL; 705 / 8 / 0 subchecks.** Live make exit **2** is retained. Wrapper `2026-11-04T02:50:12.408311+00:00`–`2026-11-04T04:12:46.722862+00:00`, **4954.315 s**, including waiting, startup and teardown. The acceptance make starts at **03:23:19.935 UTC**, last case check **04:09:58 UTC**; `TestATMembers_Live` elapsed **2772.85 s**. [Every result](history/follow-up-21/F21-results.json), [every subcheck](history/follow-up-21/F21-subchecks.md).

Initial hold **1621.105 s**, post-startup wait **300.410 s**, total pre-acceptance waiting **1921.515 s (32 min 1.515 s)**, below 120 minutes. No reservation. `bash "$FVPN_HOLD_SH"` output is captured on its own stream before every make/Go/Docker invocation, including nested calls; held checks wait 60 seconds. A conservative check also held `go env GOCACHE`. MAKE is pinned to the local admission wrapper for recursive calls. No admission exit 125 was treated as a test result.

Other gates appeared during startup and during teardown. Those windows remain **D-133 confounded**, with no retry. [Exact first/last samples and pid/comm/ticks](history/follow-up-21/F21-process-monitor.json), [window index and waits](history/follow-up-21/F21-admission.json). No foreign-suite sample falls within acceptance checks; five-second sampling cannot exclude shorter activity or establish full CPU/A-B qualification. Cleanup's additional admission hold is in the receipts. No foreign process was paused, signalled or reconfigured.

`go env GOCACHE`: **`<home>/.cache/go-build`**, home-relative; no cache override. Disk scratch under `.agent/tmp`; Python `-B`; no new shell cleanup trap. Prefix `fvpn-w4-members-`, port base `24000`. [Teardown and independent four-kind inventory](history/follow-up-21/F21-cleanup.json): **exit 0, zero owned containers/networks/volumes/images**. Shared Postgres was untouched.

Setup used `FVPN_E2E_PREFIX=w4-members`, `FVPN_E2E_PORT_BASE=24000`, `FVPN_E2E_TWO_RESOLVERS=true`, `FVPN_E2E_KERNEL_OBSERVE=1` and `E2E_RESET=1`. The admitted foreground sequence was `make -f mk/e2e.mk e2e-up`, the single `make -f mk/at-members.mk at-members`, producer/log export, then `make -f mk/e2e.mk e2e-down` and independent owned inventory.

Preflight `vmstat 5 7`, begun **2026-11-04T02:50:12.483732+00:00** before the admission wait:

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 1  0 2999288 20059036 2994960 153700184   13   47 23181 36045 24273   21  4  2 94  1  0  0
 0  0 2999288 20053620 2994968 153700332    0    0     4   167 2958 4928  1  0 99  0  0  0
 0  0 2999288 19833380 2995124 153711200    0    0  1702  1150 5042 6105  1  0 99  0  0  0
 0  0 2999288 19904560 2995132 153711212    0    0     0   570 2885 4156  0  1 99  0  0  0
 1  0 2999288 19851336 2995156 153765888    0    0     0  2686 62791 9335  4  1 93  2  0  0
 0  0 2999288 19876204 2995164 153728420    0    0     1  4861 5022 8222  1  0 98  1  0  0
 0  0 2999288 19750980 2995180 153757816    0    0    23 13371 37759 25144  3  1 96  0  0  0
```

Refreshed `vmstat 5 7`, **2026-11-04T03:23:23.727713+00:00–2026-11-04T03:23:53.750567+00:00**, while the accepted test was setting up before its first case:

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 4  1 3019884 19359564 2823172 152521912   13   47 22664 35554 24351   21  4  2 94  1  0  0
 1  0 3019884 18331956 2823192 152518404    0    0    15 29606 24068 30746  6  3 89  2  0  0
 2  0 3019884 18417880 2823204 152518652    0    0     0  1770 22672 21684  6  2 92  0  0  0
 2  0 3019884 19343832 2823212 152535240    0    0     0  8101 23538 21264  5  2 93  0  0  0
 2  0 3019884 18376596 2823224 152610436    0    0 15038   550 26257 66061  5  3 92  0  0  0
 1  0 3019884 18912828 2823232 152627060    0    0     2 20344 21487 91585  4  2 94  0  0  0
 0  0 3019884 19192716 2823244 152658536    0    0     3  7099 18766 24484  3  2 94  0  0  0
```

Both snapshots also ran the requested process command:

```sh
ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test'
```

Raw arguments remain private; public foreign processes are only pid, comm and ticks. [Preflight](history/follow-up-21/F21-host-preflight.json), [refreshed snapshot](history/follow-up-21/F21-host-refreshed.json).

Actual live result tail (Go marks blocked cases non-green):

```text
--- FAIL: TestATMembers_Live (2772.85s)
    --- FAIL: TestATMembers_Live/AT05 (15.74s)
    --- FAIL: TestATMembers_Live/AT01 (81.84s)
    --- PASS: TestATMembers_Live/AT02 (934.06s)
    --- PASS: TestATMembers_Live/AT03 (213.11s)
    --- FAIL: TestATMembers_Live/AT04 (583.08s)
    --- FAIL: TestATMembers_Live/AT08 (107.81s)
    --- FAIL: TestATMembers_Live/AT20 (99.64s)
    --- FAIL: TestATMembers_Live/AT21 (34.67s)
    --- FAIL: TestATMembers_Live/AT22 (370.32s)
    --- FAIL: TestATMembers_Live/AT09 (323.27s)
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	2801.534s
```

Gates run in order: area lint; assembled run; offline members; uncached handover; root lint; root vet; gofmt; diff check; committed-evidence hygiene. Root lint uses `make go-lint go-with-disk-tmp=` with the wrapper's disk TMPDIR/GOTMPDIR, retaining the exact checks without the root shell cleanup trap. All nine gate receipts are complete. [Case analysis and limitations](history/follow-up-21/followup21-2026-11-03.md), [requests](history/follow-up-21/requests.md).

Completed gates (original output tails and log pins in [F21-gates.json](history/follow-up-21/F21-gates.json)):

| Gate | Exit | Elapsed seconds |
|---|---:|---:|
| 01-area-lint | 0 | 2.041 |
| 02-assembled | 2 | 4954.315 |
| 03-offline-members | 0 | 28.338 |
| 04-handover | 0 | 4.137 |
| 05-root-lint | 0 | 4.036 |
| 06-vet | 0 | 0.449 |
| 07-gofmt | 0 | 0.133 |
| 08-diff | 0 | 0.458 |
| 09-hygiene | 0 | 81.997 |

The committed-evidence hygiene gate passed against records commit `d70f3e16`, scanning 250 acceptance JSON records and 523 Markdown documents. Its 81.997 s includes a separate 60.083 s admission wait; the test itself took 21.61 s. This post-run wait is separate from the 32 min 1.515 s before acceptance and 120.120 s before cleanup. Reproduce the additive case index with `python3 -B tests/e2e/at/members/followup21_records.py`; it refuses another run/source and never changes a case verdict.

---

Historical follow-up 18 ledger below retains every original outcome; its evidence links now point to the archive.

# Follow-up 18 run ledger

Run `50117c68-d1b3-431d-a2be-cd94d27a2447`, measured harness `4eb5d7d32c449da7689e749fd8ecb208657937dd`, baseline `27ff63928222c9642e4f5e180cf225108327a1e5` (#432). Main at preflight: `27ff63928222c9642e4f5e180cf225108327a1e5`. The measured source was not rebased or changed during the live run.

**2 PASS / 6 BLOCKED / 2 FAIL**, **699 passed / 5 blocked / 4 failed subchecks**. Second authorized assembled ten-case run, live wrapper exit **2**; no third run. [Original case results](history/follow-up-18-run2/F18-results.json), [all AT09 subchecks](history/follow-up-18-run2/F18-AT09-subchecks.md).

The foreground wrapper ran **2026-11-03T21:05:48.316204+00:00–2026-11-03T21:55:03.324790+00:00**, **2955.009 s**, including admission, startup and teardown. Its initial host-admission wait was **0.036 s**, below the 90-minute limit. The separate suite-admission receipt is retained below. No reservation was taken. The hold script was invoked as `bash "$FVPN_HOLD_SH"` before every make, go test and Docker command, including nested calls; held checks polled every 60 seconds. Admission was checked again after stack startup.

```json
{
  "wait_seconds": 0.027,
  "waited_for_isolation": false,
  "polls": [
    {
      "elapsed_seconds": 0.027,
      "other_suites": 0,
      "isolation_active": false
    }
  ]
}
```

The five-second monitor retained **573 samples** with **0 external-suite samples** after separating this runner's own child make. No external-suite overlap observed; five-second sampling cannot exclude processes entirely between samples. [Original process monitor](history/follow-up-18-run2/F18-process-monitor.json), [admission and cleanup receipts](history/follow-up-18-run2/F18-admission-and-cleanup.json).

No separate diagnostic command was attempted during run 2. Run 1's canceled diagnostic remains historical in its archive. No other lane's process was touched.

`go env GOCACHE` returned `<home>/.cache/go-build` (home-relative publication); no cache override. Exact output stays private. Scratch uses disk under `.agent/tmp` via Python `TemporaryDirectory`; no shell cleanup trap was introduced. Prefix `fvpn-w4-members-`, port base `24000`. Owned teardown and the subsequent inventory are recorded in the receipts; shared Postgres and other lanes were untouched.

Preflight **2026-11-03T21:05:48.415486+00:00** ([original receipt](history/follow-up-18-run2/F18-host-preflight.json)):

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 0  0 2944476 18497164 2900256 155685272   17   62 29446 40430 22306   20  3  2 94  1  0  0
 0  0 2944476 18496584 2900264 155684856    0    0     0   188 3141 4447  0  1 99  0  0  0
 0  0 2944476 18495868 2900272 155684872    0    0     0    38 2852 4877  0  0 99  0  0  0
 1  0 2944476 18525192 2900284 155652684    0    0     0   420 2824 4312  0  0 100  0  0  0
 0  0 2944476 18524700 2900292 155652688    0    0     0   306 2744 4367  0  0 100  0  0  0
 1  0 2944476 18520804 2900300 155652692    0    0     0   273 2942 4533  0  0 100  0  0  0
 0  0 2944476 18518372 2900312 155655332    0    0     0    48 2755 4057  0  0 99  0  0  0
```

`pgrep -af 'gates.sh|mk/at-'` returned empty stdout immediately before the run; its SHA-256 is retained in the admission receipt.

Live output tail:

```text
--- FAIL: TestATMembers_Live (2828.59s)
    --- FAIL: TestATMembers_Live/AT05 (15.72s)
    --- FAIL: TestATMembers_Live/AT01 (56.10s)
    --- FAIL: TestATMembers_Live/AT02 (967.08s)
    --- PASS: TestATMembers_Live/AT03 (214.36s)
    --- FAIL: TestATMembers_Live/AT04 (600.58s)
    --- PASS: TestATMembers_Live/AT08 (111.40s)
    --- FAIL: TestATMembers_Live/AT20 (101.10s)
    --- FAIL: TestATMembers_Live/AT21 (34.70s)
    --- FAIL: TestATMembers_Live/AT22 (370.82s)
    --- FAIL: TestATMembers_Live/AT09 (347.53s)
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	2847.732s
Private producer export node exit: 0
Private producer export node2 exit: 0
Owned stack teardown exit: 0
```

Required gate order: area lint; live at-members; offline members; uncached handover; root Go lint; root vet; gofmt; diff check; amd64/arm64 helper builds; committed-evidence hygiene. The root lint invocation is `make go-lint go-with-disk-tmp=` within the Python disk wrapper, disabling the root shell cleanup trap while retaining its checks. [Every attempt and real output tail](history/follow-up-18-run2/F18-gates.json).

| Gate | Exit | Elapsed seconds | Initial host wait seconds |
|---|---:|---:|---:|
| 01-run2-area-lint | 0 | 1.651 | 0.035 |
| 01b-run2-reader-tests | 1 | 0.497 | 0.035 |
| 01c-run2-area-lint | 0 | 1.608 | 0.035 |
| 01d-run2-reader-tests | 0 | 0.513 | 0.034 |
| 02-assembled | 2 | 2955.009 | 0.036 |
| 03-offline-members | 0 | 21.592 | 0.036 |
| 04-handover | 0 | 4.538 | 0.035 |
| 05-root-lint | 0 | 4.616 | 0.037 |
| 06-vet | 0 | 0.462 | 0.037 |
| 07-gofmt | 0 | 0.178 | 0.037 |
| 08-diff | 0 | 0.58 | 0.036 |
| 09-helper-builds | 0 | 3.331 | 0.036 |
| 10-hygiene | 0 | 19.43 | 0.037 |
| 11-final-handover | 0 | 3.886 | 0.035 |
| 12-final-hygiene | 0 | 19.344 | 0.036 |

D-098 archive commit `50f4403b` precedes the harness commit. All 77 original root files from run 81b638e8, including gzip and pin files, are preserved byte-for-byte under [follow-up 17](history/follow-up-17/INDEX.md); all earlier archives remain unchanged. No handover claims test reads a moved living-root file, so no D-132 test-pointer edit was needed.

Orchestrator note 1 (2026-11-03 15:58 EST) authorized the reader correction and this second run. The reader now accepts the production retire phase with all lifecycle checks unchanged. D-146 qualification remains as committed before run 1: successful receive-only TCP bytes avoid the former D-137 downgrade only with complete same-run clamp, handshake, outer-packet and no-PTB evidence. Original failures, D-135/D-136 predicates and historical results remain unchanged. Postprocessing only indexes retained records. [Results](history/follow-up-18-run2/summary.md), [limitations](history/follow-up-18-run2/BLOCKERS.md), [requests](history/follow-up-18-run2/requests.md).

Run 1 (`118271c1`, harness `b4317bd4`) is frozen in archive commit `71dc60ac`: [72 original root files and pins](history/follow-up-18-run1/INDEX.md). Its observed result remains 0 PASS / 3 BLOCKED / 7 FAIL and 442 / 0 / 35 subchecks. The proven retire rejection and 16 original export-prefix matches explain the joined reader failures; no other historical verdict is rewritten. [Reader value audit and tests](history/follow-up-18-run2/F18-reader-audit.md). The first synthetic test attempt passed its synthetic cases but failed to frame the indented archived reproduction as JSONL; compacting that test input fixed the reproduction, with no parser relaxation.

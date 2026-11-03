# Follow-up 18 run ledger

Run `50117c68-d1b3-431d-a2be-cd94d27a2447`, measured harness `4eb5d7d32c449da7689e749fd8ecb208657937dd`, baseline `27ff63928222c9642e4f5e180cf225108327a1e5` (#432). Main at preflight: `27ff63928222c9642e4f5e180cf225108327a1e5`. The measured source was not rebased or changed during the live run.

**2 PASS / 6 BLOCKED / 2 FAIL**, **699 passed / 5 blocked / 4 failed subchecks**. Second authorized assembled ten-case run, live wrapper exit **2**; no third run. [Original case results](F18-results.json), [all AT09 subchecks](F18-AT09-subchecks.md).

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

The five-second monitor retained **573 samples** with **0 external-suite samples** after separating this runner's own child make. No external-suite overlap observed; five-second sampling cannot exclude processes entirely between samples. [Original process monitor](F18-process-monitor.json), [admission and cleanup receipts](F18-admission-and-cleanup.json).

No separate diagnostic command was attempted during run 2. Run 1's canceled diagnostic remains historical in its archive. No other lane's process was touched.

`go env GOCACHE` returned `<home>/.cache/go-build` (home-relative publication); no cache override. Exact output stays private. Scratch uses disk under `.agent/tmp` via Python `TemporaryDirectory`; no shell cleanup trap was introduced. Prefix `fvpn-w4-members-`, port base `24000`. Owned teardown and the subsequent inventory are recorded in the receipts; shared Postgres and other lanes were untouched.

Preflight **2026-11-03T21:05:48.415486+00:00** ([original receipt](F18-host-preflight.json)):

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

Required gate order: area lint; live at-members; offline members; uncached handover; root Go lint; root vet; gofmt; diff check; amd64/arm64 helper builds; committed-evidence hygiene. The root lint invocation is `make go-lint go-with-disk-tmp=` within the Python disk wrapper, disabling the root shell cleanup trap while retaining its checks. [Every attempt and real output tail](F18-gates.json).

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

Orchestrator note 1 (2026-11-03 15:58 EST) authorized the reader correction and this second run. The reader now accepts the production retire phase with all lifecycle checks unchanged. D-146 qualification remains as committed before run 1: successful receive-only TCP bytes avoid the former D-137 downgrade only with complete same-run clamp, handshake, outer-packet and no-PTB evidence. Original failures, D-135/D-136 predicates and historical results remain unchanged. Postprocessing only indexes retained records. [Results](summary.md), [limitations](BLOCKERS.md), [requests](requests.md).

Run 1 (`118271c1`, harness `b4317bd4`) is frozen in archive commit `71dc60ac`: [72 original root files and pins](history/follow-up-18-run1/INDEX.md). Its observed result remains 0 PASS / 3 BLOCKED / 7 FAIL and 442 / 0 / 35 subchecks. The proven retire rejection and 16 original export-prefix matches explain the joined reader failures; no other historical verdict is rewritten. [Reader value audit and tests](F18-reader-audit.md). The first synthetic test attempt passed its synthetic cases but failed to frame the indented archived reproduction as JSONL; compacting that test input fixed the reproduction, with no parser relaxation.

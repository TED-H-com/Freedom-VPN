# Follow-up 18 run ledger

Run `118271c1-616e-46a2-a744-d67b00fb94fb`, measured harness `b4317bd4e4ad178609b71d3fd4fbb986cb9947db`, baseline `27ff63928222c9642e4f5e180cf225108327a1e5` (#432). Main at preflight: `27ff63928222c9642e4f5e180cf225108327a1e5`. The measured source was not rebased or changed during the live run.

**0 PASS / 3 BLOCKED / 7 FAIL**, **442 passed / 0 blocked / 35 failed subchecks**. One assembled ten-case run, live wrapper exit **2**; no live retry. [Original case results](F18-results.json), [all AT09 subchecks](F18-AT09-subchecks.md).

The foreground wrapper ran **2026-11-03T19:42:19.219056+00:00–2026-11-03T20:54:26.543577+00:00**, **4327.325 s**, including admission, startup and teardown. Its initial host-admission wait was **0.035 s**, below the 90-minute limit. The separate suite-admission receipt is retained below. No reservation was taken. The hold script was invoked as `bash "$FVPN_HOLD_SH"` before every make, go test and Docker command, including nested calls; held checks polled every 60 seconds. Admission was checked again after stack startup.

```json
{
  "wait_seconds": 0.025,
  "waited_for_isolation": false,
  "polls": [
    {
      "elapsed_seconds": 0.025,
      "other_suites": 0,
      "isolation_active": false
    }
  ]
}
```

The five-second monitor retained **846 samples** with **0 external-suite samples** after separating this runner's own child make. No external-suite overlap observed; five-second sampling cannot exclude processes entirely between samples. [Original process monitor](F18-process-monitor.json), [admission and cleanup receipts](F18-admission-and-cleanup.json).

A separate optional diagnostic was held against this lane's active make and was canceled before any container read. Those held polls are retained in the ledger; they do not establish external contention or a delay inside the runner. No other lane's process was touched.

`go env GOCACHE` returned `<home>/.cache/go-build` (home-relative publication); no cache override. Exact output stays private. Scratch uses disk under `.agent/tmp` via Python `TemporaryDirectory`; no shell cleanup trap was introduced. Prefix `fvpn-w4-members-`, port base `24000`. Owned teardown and the subsequent inventory are recorded in the receipts; shared Postgres and other lanes were untouched.

Preflight **2026-11-03T19:42:19.314442+00:00** ([original receipt](F18-host-preflight.json)):

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 0  0 3004076 19850768 2877476 154420620   18   67 31968 43551 21116   19  3  2 94  1  0  0
 0  0 3004076 19842940 2877484 154424636    0    0     0   431 3409 4482  0  1 99  0  0  0
 1  0 3004076 19829256 2877504 154425580    0    0     0  3439 3368 4326  0  0 99  0  0  0
 2  0 3004076 19072192 2877512 154426660    0    0     0   556 11034 56088  3  1 96  0  0  0
 1  0 3004076 19072192 2877520 154427324    0    0     0   432 70388 443654  4  2 94  0  0  0
 1  0 3004076 19087956 2877528 154427656    0    0     0    78 61368 398650  4  2 94  0  0  0
 0  0 3004076 19728300 2877536 154427644    0    0     0  2526 15706 84925  2  1 97  0  0  0
```

`pgrep -af 'gates.sh|mk/at-'` returned empty stdout immediately before the run; its SHA-256 is retained in the admission receipt.

Live output tail:

```text
--- FAIL: TestATMembers_Live (4202.28s)
    --- FAIL: TestATMembers_Live/AT05 (16.34s)
    --- FAIL: TestATMembers_Live/AT01 (76.89s)
    --- FAIL: TestATMembers_Live/AT02 (1550.27s)
    --- FAIL: TestATMembers_Live/AT03 (206.47s)
    --- FAIL: TestATMembers_Live/AT04 (833.71s)
    --- FAIL: TestATMembers_Live/AT08 (548.88s)
    --- FAIL: TestATMembers_Live/AT20 (118.67s)
    --- FAIL: TestATMembers_Live/AT21 (17.73s)
    --- FAIL: TestATMembers_Live/AT22 (371.39s)
    --- FAIL: TestATMembers_Live/AT09 (454.12s)
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	4220.189s
Private producer export node exit: 0
Private producer export node2 exit: 0
Owned stack teardown exit: 0
```

Required gate order: area lint; live at-members; offline members; uncached handover; root Go lint; root vet; gofmt; diff check; amd64/arm64 helper builds; committed-evidence hygiene. The root lint invocation is `make go-lint go-with-disk-tmp=` within the Python disk wrapper, disabling the root shell cleanup trap while retaining its checks. [Every attempt and real output tail](F18-gates.json).

| Gate | Exit | Elapsed seconds | Initial host wait seconds |
|---|---:|---:|---:|
| 01-area-lint | 0 | 2.378 | 0.035 |
| 01b-area-lint | 0 | 1.644 | 0.035 |
| 02-assembled | 2 | 4327.325 | 0.035 |

D-098 archive commit `50f4403b` precedes the harness commit. All 77 original root files from run 81b638e8, including gzip and pin files, are preserved byte-for-byte under [follow-up 17](history/follow-up-17/INDEX.md); all earlier archives remain unchanged. No handover claims test reads a moved living-root file, so no D-132 test-pointer edit was needed.

The new D-146 qualification is the only live verdict change: successful receive-only TCP bytes avoid the former D-137 downgrade only with complete same-run clamp, handshake, outer-packet and no-PTB evidence. Original failures, D-135/D-136 predicates and historical results remain unchanged. Postprocessing only indexes retained records. [Results](summary.md), [limitations](BLOCKERS.md), [requests](requests.md).

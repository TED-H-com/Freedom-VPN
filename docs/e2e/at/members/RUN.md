# Follow-up 17 run ledger

Run `81b638e8-be59-4996-9594-14dd402ebb86`, measured harness `3ba0856487288b9be3cf81f3e3afa37b0fa43a47`, baseline `a7497ddf0b08805e4cddfc0e3b5f47399205054e` (the assigned #406 tree, including #400/#401/#403/#406).

**2 PASS / 6 BLOCKED / 2 FAIL**, **691 passed / 7 blocked / 5 failed subchecks**. One assembled ten-case run; live make exit **2**.

Main advanced during admission to `3573f396d60836f3f203ab5f883575c96a0ec45c` (#407 handover). Its 34-file delta is documentation, handover fixtures/tests and secops scope evidence; no runtime source changed. The running tree was not rebased. Build records honestly retain `source_dirty=true` from generated evidence; no measured Go source changed during the assembled run.

The foreground wrapper ran **2026-11-03T08:06:07.607781Z–08:55:03.714784Z**, **2936.107 s**, including admission, startup and teardown. The live test took **2758.08 s**, package **2767.176 s**. All ten cases belong to this one run; none was retried. Teardown exited 0; independent inventory at **08:56:06.916900Z** found zero owned containers, networks, volumes or images. Shared Postgres and other lanes were untouched. [Admission/cleanup receipts](F17-admission-and-cleanup.json).

Area lint's initial host hold waited **120.113 s**, polling at 60 seconds; a netlab timing suite was observed then. After final source preparation, area lint passed again. The assembled runner then waited **60.055 s** for the active handover chain to finish. The priority/concurrency check admitted zero other suites, sampled `vmstat 5 7`, checked `pgrep -af 'gates.sh|mk/at-'`, and checked again after startup. Scheduling used observed process state and the authorized hold script, not an assumed owner completion time. Every make, go test and Docker command, including nested calls, checked that the script printed `free`; none was held during the live period.

The [five-second process monitor](F17-process-monitor.json) has **557 samples**, zero chain-gate-active samples. Its suite count can include this runner's own child make; the chain-gate flag is separate. No chain overlap was observed, and no confounded interval is fabricated. This sampling does not exclude a process entirely between samples. Historical follow-up 16's contention window is unchanged.

`go env GOCACHE` returned `${HOME}/.cache/go-build` (home-relative publication). Exact output and raw pgrep remain private. No cache override. Scratch uses disk under `.agent/tmp` via Python `TemporaryDirectory`; no shell cleanup trap was introduced. Prefix `fvpn-w4-members-`, port base `24000`.

Preflight **2026-11-03T08:07:07.699127+00:00** ([original receipt](F17-host-preflight.json)):

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 2  0 521060 2638636 1915920 171170352    3   37 107944 109473 25339   21  3  2 93  1  0  0
 0  0 521060 3349308 1915928 171170864    0    0     0 14483 23372 136491  2  1 93  3  0  0
 1  0 521060 3349112 1915936 171170452    0    0     0  8061 4868 8763  0  0 99  1  0  0
 4  0 579580 1428224 1880132 171420172    0 11606     1 43060 67461 35594  5  3 90  2  0  0
 3  0 607700 2677828 1815296 171610172    0 5615    49  8474 71139 36551  8  5 87  0  0  0
 2  0 607700 3689696 1803224 170928276    0    0   836 435802 38543 83128  6  4 89  0  0  0
 0  1 607700 3625824 1803540 170928808    0    0    59 259930 17142 84836  2  1 96  0  0  0
```

Live output tail:

```text
--- FAIL: TestATMembers_Live (2758.08s)
    --- FAIL: TestATMembers_Live/AT05 (16.25s)
    --- FAIL: TestATMembers_Live/AT01 (76.08s)
    --- FAIL: TestATMembers_Live/AT02 (926.85s)
    --- PASS: TestATMembers_Live/AT03 (215.73s)
    --- FAIL: TestATMembers_Live/AT04 (583.10s)
    --- PASS: TestATMembers_Live/AT08 (111.65s)
    --- FAIL: TestATMembers_Live/AT20 (98.73s)
    --- FAIL: TestATMembers_Live/AT21 (36.25s)
    --- FAIL: TestATMembers_Live/AT22 (370.27s)
    --- FAIL: TestATMembers_Live/AT09 (313.92s)
FAIL github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members 2767.176s
Private producer export node exit: 0
Private producer export node2 exit: 0
Owned stack teardown exit: 0
```

The required foreground gate order is area lint; live at-members (above, exit 2); members offline tests; uncached handover tests; root Go lint; root vet; gofmt; diff check; amd64/arm64 helper builds; committed-evidence hygiene. The root invocation uses `make go-lint go-with-disk-tmp=` inside the Python disk wrapper to disable the root Makefile's shell cleanup trap without skipping checks. Final offline receipts follow below.

[Results](summary.md), [archive](history/follow-up-16/INDEX.md), [commit mapping](history/follow-up-16/PUBLISHED-COMMITS.md), [derivation commands](PROCEDURES.md#follow-up-17-retained-evidence).

Post-run root lint initially rejected the live-only `memberHTTPRecord` declaration as unused in the untagged build (`U1000`). Commit `05c5798d` moves the identical declaration into the existing `e2e`-tagged test file and removes the now-unused import. It changes no behavior or predicate. The live measurement remains pinned to `3ba08564`; both build modes are rechecked without repeating the run. The initial failed lint receipt is retained.

Final gate receipts ([all attempts and real output tails](F17-gates.json)):

| Gate | Exit | Elapsed seconds |
|---|---:|---:|
| Area lint | 0 | 1.448 |
| One assembled run | 2 | 2936.107 |
| Offline members (recheck) | 0 | 253.650 |
| Uncached handover (recheck) | 0 | 2.730 |
| Root go-lint (recheck) | 0 | 1.379 |
| Root go vet | 0 | 0.373 |
| Exact gofmt assertion | 0 | 0.114 |
| Diff whitespace | 0 | 0.305 |
| Five helpers × amd64/arm64 | 0 | 3.037 |
| Committed-evidence hygiene | 0 | 11.698 |

The offline recheck waited **240.181 s** for host admission, then passed. The initial gofmt shell assertion contained one extra literal `)` and returned 1 without a formatting finding; the exact corrected assertion returned 0. The initial lint U1000 and this invocation error remain in the ledger. Committed-evidence hygiene scanned **220 acceptance JSON records and 416 Markdown documents**, including lossless gzip bytes and pins. No failed live predicate was retried.

Archive/link verification: all 81 original archive files still match baseline bytes and SHA-256. Across current root documents, the new archive and additive references, **336** Markdown targets were checked. One original operator-checklist reference resolves outside owned paths and is listed in [requests](requests.md); the archive index supplies a working direct link. Other checked targets resolve, and plain AT02 remains present. Earlier archives remain untouched.

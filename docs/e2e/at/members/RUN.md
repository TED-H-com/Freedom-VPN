# Follow-up 16 run ledger

Run `0ffb355a-945b-4198-a551-3794120a686c`, measured harness `3a8b61594b447d164bb7c9481545e207f8688ec2`, baseline `d640ce96d9145e57a60e04e5875e049070f30b96`.

**2 PASS / 6 BLOCKED / 2 FAIL**, with **679 passed / 6 blocked / 7 failed subchecks**. One assembled ten-case run; live make exit **2**. No assembled run was repeated after the reboot.

D-098 is the first separate commit `56eb3c0b`: all **64 root files** of e80bb592, including gzip bytes and sidecars, are byte-identical in [follow-up 15](history/follow-up-15/RUN.md), with [SHA-256 pins](history/follow-up-15/SHA256SUMS). Every earlier archive is untouched. Handover claim tests read frozen evidence and explicitly reject living root pins; no D-132 one-file exception was needed. The plain [AT02.md index](AT02.md) is retained for Markdown links.

**D-133 contention:** the orchestrator reported CPU-heavy handover gates at **2026-11-01 17:32:43–17:37:12 UTC**. Every timing, failure and anomaly overlapping that window is **contention-confounded**, with `cause_status=unproven`; measured values and verdicts stand. [Complete time and step index](F16-contention.json) includes 46 AT02 steps completed in the window and cumulative references in later records. Finite-expiry withdrawal/restoration, the second re-grant, SIGTERM/usage stop/resume, node2 preparation, alarm recovery and sibling removal overlap. The 4,724-send stream, initial AT01 selection, sole-member/retained-safety holds and fresh AT09 measurements are outside this reported window.

The measured baseline includes #385, #386, #388, #389 and #394, then #395/#396. Main advanced during admission to `97912766d885b0cbe549ab0a7f781e991ef832df` (#397); its only delta from the measured baseline was `docs/ROADMAP.md` and `tests/e2e/internal/counterread/read_test.go`. No runtime source changed and the running tree was not rebased. The measured harness is pinned above; later harness fixes are explicitly for the next follow-up.

The orchestrator records live start **2026-11-01T17:24:15Z** (12:24:15 EST); the first case starts 17:24:28Z and the final case's last step is **18:08:22Z**. The test reports **2639.99 s**, package **2644.440 s**. The run overlaps the reported contention window; it did not run outside it. Teardown exit 0 is retained in the original log; a wall-clock timestamp for teardown completion was not retained. Owned inventory was verified zero after two leftover networks were removed before the poweroff; the orchestrator confirms it remains zero after reboot. No Docker command or new stack was needed on resume.

Isolation's timing-sensitive run came first: waited **2931.736 s (48 min 51.736 s)**. Another suite began during the subsequent vmstat sampling; admission stopped that wrapper with exit 75 **before Docker startup or any acceptance case**. [Stopped preflight](F16-host-preflight-admission-stop-1.json). The next admission waited **240.537 s (4 min 0.537 s)**, then allowed one other suite; startup admission rechecked concurrency. [Actual preflight](F16-host-preflight.json). No third integration suite was deliberately started.

`go env GOCACHE` returned `${HOME}/.cache/go-build` (home component redacted for public-repository hygiene; exact output and raw pgrep are retained privately, with original preflight SHA-256). No cache override or purge. TMPDIR/GOTMPDIR are unique disk directories under `.agent/tmp`. Prefix `w4-members`, port base `24000`.

Preflight at **2026-11-01T17:23:01.823528Z** records `pgrep -af 'gates.sh|mk/at-'`, `go env GOCACHE` and `vmstat 5 7`:

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 2  5  51544 43105348 12010552 115639416    0    0 19649 28973 38793   35  8  3 79 10  0  0
 2  5  51544 43035944 12010552 115666164    0    0     0  7905 61456 6809  7  1 83  8  0  0
 1  5  51544 43310584 12010552 115694340    0    0     2 12426 63259 8572  6  1 85  8  0  0
 2  5  51544 43691808 12010552 115715556    0    0     0  8818 61386 6140  6  1 85  8  0  0
 1  5  51544 43966092 12010552 115742100    0    0     0 34854 57018 6087  6  1 85  8  0  0
 1  5  51544 44036444 12010552 115769824    0    0     0  6637 74360 66600  7  2 83  8  0  0
 0  4  51544 44136444 12010552 115755448    0    0     2  6682 82837 63867  6  2 85  7  0  0
```

Actual live tail:

```text
--- FAIL: TestATMembers_Live (2639.99s)
    --- FAIL: TestATMembers_Live/AT05 (23.01s)
    --- FAIL: TestATMembers_Live/AT01 (58.91s)
    --- FAIL: TestATMembers_Live/AT02 (889.21s)
    --- PASS: TestATMembers_Live/AT03 (204.77s)
    --- FAIL: TestATMembers_Live/AT04 (571.11s)
    --- PASS: TestATMembers_Live/AT08 (98.93s)
    --- FAIL: TestATMembers_Live/AT20 (82.87s)
    --- FAIL: TestATMembers_Live/AT21 (32.11s)
    --- FAIL: TestATMembers_Live/AT22 (368.67s)
    --- FAIL: TestATMembers_Live/AT09 (304.44s)
FAIL    github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members    2644.440s
Owned stack teardown exit: 0
```

## Resume — 2026-11-02

The host powered off after run/teardown completion, interrupting offline verification. Retained JSON/JSONL and gzip pins verified after reboot; no truncated or missing measurement was repaired or invented. The original capture manifest verified. The resumed packet review joins late initial-selection SYNs and the ::1-source response, retains AT09's four failures, and annotates every observation overlapping the orchestrator's contention window. The three harness corrections (skip redundant unsuspension, query member.patch, re-authenticate before AT09) apply to future runs only.

Offline gates resume in the required order after area lint, without repeating gate 2. Before every `go test`, `make` or Docker invocation the authorized host-admission script must print `free`; a held result is retried every 60 seconds. All commands are foreground and sequential. No shell cleanup trap is used: `disktmp.py` owns disk scratch; `go-with-disk-tmp=` disables the root Makefile's trap wrapper without skipping its checks.

```sh
# Host admission before each heavy command below.
python3 -B tests/e2e/at/members/disktmp.py make -f mk/at-members.mk at-members-lint
# Gate 2 already complete: make -f mk/at-members.mk at-members (exit 2). DO NOT repeat.
python3 -B tests/e2e/at/members/disktmp.py go test ./tests/e2e/at/members/...
python3 -B tests/e2e/at/members/disktmp.py go test -count=1 ./tests/handover/...
python3 -B tests/e2e/at/members/disktmp.py make go-lint go-with-disk-tmp=
python3 -B tests/e2e/at/members/disktmp.py go vet ./...
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
python3 -B tests/e2e/at/members/disktmp.py make -f mk/at-members.mk at-members-build
python3 -B tests/e2e/at/members/disktmp.py go test ./tests/e2e/at/members/... -run '^TestCommittedEvidenceHygiene$' -count=1 -v
```

<!-- offline-gates -->
All resumed offline gates passed in order after admission (2026-11-02/03 EST). Each admission check returned `free` on its first poll; there were no held wait cycles. Gate 2 remains the original exit-2 run and was not repeated.

| Gate | Check | Exit | Resume start (UTC) |
|---|---|---:|---|
| 1 | area lint | 0 | 2026-11-03T05:01:07.716907+00:00 |
| 3 | members offline tests | 0 | 2026-11-03T05:01:23.872050+00:00 |
| 4 | handover, uncached | 0 | 2026-11-03T05:01:57.428758+00:00 |
| 5 | root Go lint | 0 | 2026-11-03T05:02:36.599377+00:00 |
| 6 | root vet | 0 | 2026-11-03T05:03:28.689280+00:00 |
| 7 | tracked Go formatting | 0 | 2026-11-03T05:03:52.611046+00:00 |
| 8 | diff whitespace | 0 | 2026-11-03T05:03:52.794888+00:00 |
| 9 | five helpers, linux amd64 + arm64 | 0 | 2026-11-03T05:04:01.757885+00:00 |
| 10 | committed-evidence hygiene | 0 | 2026-11-03T05:04:33.358653+00:00 |

The first resumed members-test invocation exited 1 because the archive verifier treated two `.sha256.json` sidecars as case records and demanded run IDs. The archive bytes and hashes were correct. The verifier now distinguishes sidecars and validates compressed acceptance records with both pins and their original run ID. Area lint and the members tests then passed; no archive or live verdict changed. The initial failed offline invocation remains in the private gate log.

Actual output tails:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	7.611s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	2.796s
=== RUN   TestCommittedEvidenceHygiene
    evidence_test.go:65: scanned 210 acceptance JSON records plus 387 Markdown documents
--- PASS: TestCommittedEvidenceHygiene (6.99s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	6.996s
testing: warning: no tests to run
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s [no tests to run]
```

Area lint, root lint, vet, formatting, diff check and both architecture builds returned 0 with no output. Cross-builds cover the lane probe, fixture-probe, member-session, native-observe and lease-read helpers; physical arm64 remains unmeasured. No dependency changed.

Integrity checks verified all 64 archived root files against their baseline bytes, 124 capture-manifest pins, all 169 contention-index source pins, 46 private pcaps and complete producer sequences (node 4,366 / node2 2,854). Earlier archives are unchanged. Every new file above 20,000,000 bytes has a verified lossless gzip pair; no oversized new plain file remains.
<!-- /offline-gates -->

Agent-core 34's durable reservation is included in the **`host.before_apply.end_ns` → `timing.start_ns`** bracket before Apply. The [reservation join](F16-pre-apply-reservations.json.gz) retains the original producer records, revoked-device request/command bounds, surviving-sibling baseline-to-recovery windows and both AT04 suspension monitors. Original `phase=revoke` commands establish a nonempty delete batch. Absence of such a command cannot exclude an early failed reservation/Apply. The bracket also includes common setup; it is not an isolated reservation latency. Request-to-install and baseline-to-Apply bounds include it once, without adding it twice.

The inventory retains 792 Apply records and 19 records with original revoke commands. For AT02's revoked-device batch (producer sequence 398), the pre-Apply bracket is **1.264829 ms**; the before-command snapshot → restricting-install end bound is **3.270914141–3.270914407 s** (after-command snapshot → install: 2.651206713–2.651206979 s). For AT04 suspension, the queued batches have pre-Apply brackets **1.465935 ms** (node) and **1.762922 ms** (node2); suspension input → restricting-install end bounds are respectively **2.445555565–2.445555932 s** and **2.446532587–2.446533072 s**. These command bounds include reservation and are not exact packet cutoff instants. All 61 retained baseline-to-recovery intervals contain their full pre-Apply bracket.

New evidence over 20,000,000 bytes is lossless gzip with compressed and uncompressed SHA-256/byte pins. The plain AT02.md navigation document links the compressed Markdown capture and annotated JSON. Historical archives remain byte-identical; root F15 files now resolve through follow-up-15 links. Retained raw packets and producer streams stay private; published captures contain headers/hashes only. [Derivation protocol](PROCEDURES.md#follow-up-16-retained-evidence-and-resume).

[Results](summary.md); [hold joins](F16-populated-holds.md); [AT09 failure review](F16-AT09-failure-review.json); [one new seam request and seven carried groups](requests.md).

# Follow-up 15 run ledger

Run `e80bb592-27ed-4ab5-a8c3-6433e7278938`, compiled harness `cc6e0298a791ed6c41b268f1634f543446f8c562`, baseline `9dde5eeb60a6cb6b9b10974c3100ef1ea1352698` (current main at admission; #375, #376, #377, #381 and #382 included).

**2 PASS / 6 BLOCKED / 2 FAIL**, with **644 passed / 6 blocked / 2 failed subchecks**. One assembled ten-case run; live make exit **2**.

D-098 was the first, separate commit `90cf72cf`: all **70 root files** of run 258ba504 are byte-identical under [follow-up 14](history/follow-up-14/RUN.md), with [SHA-256 pins](history/follow-up-14/SHA256SUMS). Earlier archives are untouched. Handover claim tests read frozen evidence; no D-132 outside-owned pointer edit was needed.

`go env GOCACHE` returned `/data/ted-h-cache/go-build`; no cache override or purge. Scratch is disk-backed under `.agent/tmp`. Prefix `w4-members`, port base `24000`, two resolvers.

Preflight `2026-11-01T12:02:25.968996+00:00`; teardown finished `2026-11-01T12:40:15.495141+00:00`. Full wrapper time 2269.509 s, live test 2190.00 s. All three admission checks saw exactly one other suite. Original pgrep output remains private because it includes other lanes' command paths. The [host receipt](F15-host-preflight.json) records the commands and sanitized inventory.

Host state from `vmstat 5 7`:

```text
procs -----------memory---------- ---swap-- -----io---- -system-- -------cpu-------
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st gu
 4  8  51604 12247360 11370916 148539392    0    0  4997 30018 37960   35  8  3 79 11  0  0
 2 12  51604 12444176 11370936 148704292    0    0     8 107150 50016 66314 12  3 79  6  0  0
 4 12  51604 11798392 11370936 149126632    0    0    26 118211 43441 117735  6  2 79 13  0  0
 2 13  51604 11594052 11370944 149288784    0    0    33 111413 44680 101114  7  4 76 13  0  0
 1 12  51604 11622116 11370944 149317076    0    0     0 12820 22655 42116  2  1 84 12  0  0
 0 12  51604 11666776 11371004 149329080    0    0     0  1630 24708 46092  2  2 84 12  0  0
 2 12  51604 11651780 11371076 149300512    0    0     0 13259 78259 60986  6  3 80 11  0  0
```

```sh
make -f mk/at-members.mk at-members-lint
# Preflight: vmstat 5 7; pgrep -af 'gates.sh|mk/at-'; go env GOCACHE
python3 -B tests/e2e/at/members/disktmp.py python3 -B tests/e2e/at/members/assembled.py
# assembled.py runs: make -f mk/at-members.mk at-members
# Remaining commands run with a unique disk TMPDIR/GOTMPDIR under .agent/tmp.
go test ./tests/e2e/at/members/...
make go-lint
go vet ./...
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
make -f mk/at-members.mk at-members-build
go test ./tests/e2e/at/members -run '^TestCommittedEvidenceHygiene$' -count=1 -v
```

All gates run sequentially in the requested foreground order. Area lint and gates 3–8 exit 0; live make exits 2. Final committed-evidence hygiene also exits 0 (200 acceptance JSON records scanned after decompression). No dependency change. Helper cross-builds cover the lane probe and the existing fixture-probe, member-session and native-observe helpers for linux/amd64 and linux/arm64; physical arm64 remains unmeasured.

Actual live output tail:

```text
--- FAIL: TestATMembers_Live (2190.00s)
    --- FAIL: TestATMembers_Live/AT05 (15.36s)
    --- FAIL: TestATMembers_Live/AT01 (53.52s)
    --- FAIL: TestATMembers_Live/AT02 (434.44s)
    --- PASS: TestATMembers_Live/AT03 (179.54s)
    --- FAIL: TestATMembers_Live/AT04 (615.56s)
    --- PASS: TestATMembers_Live/AT08 (109.81s)
    --- FAIL: TestATMembers_Live/AT20 (82.03s)
    --- FAIL: TestATMembers_Live/AT21 (32.11s)
    --- FAIL: TestATMembers_Live/AT22 (369.57s)
    --- FAIL: TestATMembers_Live/AT09 (291.87s)
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	2193.094s
make: *** [mk/at-members.mk:14: at-members] Error 1
Owned stack teardown exit: 0
```

Package output:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	3.624s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
```

New evidence larger than **20,000,000 bytes** is lossless gzip with SHA-256 and byte counts for both forms in `.gz.sha256.json`. The readers and hygiene scan decompress and verify both pins. AT02 JSON is **62,234,933 bytes** uncompressed and **3,371,553 bytes** compressed. Historical archives remain byte-identical, including the original large files. [Compression contract and derivation](PROCEDURES.md#follow-up-15-evidence-and-regression-readback).

The direct AT02 pause/SIGTERM request-to-observed-exit upper bound is **33.548444 ms**, within **13 s**. The retained launch ID/PID/start-ticks join resolves to parent wait status **0**, exit code **0**, no signal; the stopped lease survives unchanged. This is one cooperative sample, not a worst-case or hard scheduling guarantee. The existing supervisor stop/start follows the observed signal and retains the selected release. **No control I/O after cancel: not observable here**; this stream does not expose the Step cancellation / observation-POST completion boundary.

Across the assembled run, **three accepted hold samples / three incidents** carry `policy_missing` and `STALE_GENERATION` at `stage=lease`, with forwarding false, valid summaries (`steps=1`, `alarm=false`), and publication replay cause `stale_generation`. All end on `generation_matched`; **zero alarm reports and zero escalation audit events**. Alarm triggering and retained-safety precedence are **not exercised**, never passed by absence. [Cumulative final incidents, exact reports and lease IDs](F15-AT09-generation-holds.json).

**AT02 / wire-nodes 34 request 2:** its cumulative snapshot contains only the earlier bootstrap incident `01a196fe-5ac5-7a68-ade3-aebefed4bd4c`, report `01a196fe-5abe-7786-a1f4-2b254d4fbd74`, lease `5ce8d59c-9878-41c1-966a-624896cca11d` (applied 3 / lease 2, empty leased-member set). It ended before AT02 began. **No accepted stale-lease hold intersects AT02's revocations.** Sole-member `members_revoked` closure, removal of one member while an eligible sibling survives, and retained source-binding/TC priority during a hold are each **not exercised**. Fifteen original revocation/withdrawal audit rows retain resource/member IDs and surrounding member-state snapshots. Device/grant withdrawal is not member suspension/closure. [Central join and coverage](F15-AT02-central-hold-join.json), [revocation/member joins](F15-AT02-revocation-members.json).

The rows are consistent with a **node-wide** hold and the contract's absence of a per-member hold field; they do not prove the unexercised revocation combinations. Incident closure alone never claims service recovered. The two later samples (including one lease with eight members) likewise report node-wide forwarding false.

| Original action | Command duration | Stored lease evidence |
|---|---:|---|
| AT02 supervisor stop after direct signal | 0.315547960 s | unchanged closed storage |
| AT02 supervisor resume | 0.470609069 s | new producer retains prior-run lease before renewal |
| AT08 disabled-node Docker restart | 10.107214827 s | new producer retains prior-run lease before renewal |
| AT09 selected namespace stop | 5.157674441 s | unchanged closed storage |
| AT09 namespace restart + observation | 90.551346091 s | includes required 90-second capture; prior-run lease retained before renewal |

[Exact original transitions and producer joins](F15-agent-transitions.json). Docker restart/namespace-stop SIGTERM-to-agent-exit intervals are **not observable here**: the retained parent receipt is unavailable. Their command durations are not graceful shutdown measurements or comparisons against the 13 s process budget.

**682d4434 is now live-qualified:** the forced relay row passes with **Relayed=2 / Direct=0**, followed by same-socket relay-to-direct recovery. The routed hop is added only afterward. Follow-up 14's relay FAIL remains recorded and confounded; it is not reclassified.

The routed IPv6 row passes **64,064 bytes in 1 ms**, with one oversized ingress, one type-2/code-0 PTB seen at router and sender, learned PMTU 1280, and **16** router fragment egresses received. The routed IPv4 row passes **64,064 bytes in 1 ms**: **25 DF=0 oversized datagrams → 50 fragments / 25 groups**, all received, no PTB. The known-PMTU IPv6 row separately passes **64,064 bytes in 1 ms**.

Receive-only rows remain limitations: dual-underlay MTU 1280 is **FAIL**, with **12,012 bytes in 12,011 ms**; IPv6 receive-only transfers **64,064 bytes in 1 ms** but remains **BLOCKED**. Neither is upgraded to a supported-PMTU pass. [Original independent rows](F15-D137-native-paths.json), [packet analyses](history/follow-up-15-captures/e80bb592-27ed-4ab5-a8c3-6433e7278938/analysis.json). Historical F8-02 failures remain unattributed; IPv4 overlay payloads do not prove IPv6 overlay support (D-123).

AT01 remains **BLOCKED**. Its original first selection has **five member SYNs, zero node/origin inner packets, zero origin requests**, with generation **5** and **six member accept rules** before and after. Captures are complete and zero-drop (**201 / 200 / 45** member/node/origin records). Cause is **unproven**; neither the node's sampled generation nor this failure locates the missing native hop. Devstack 25's campaign is consumed as evidence, not a repair on main. [Original attempt/readiness](F15-F11-01-first-selection.json), [native analysis](history/follow-up-15-captures/e80bb592-27ed-4ab5-a8c3-6433e7278938/initial-selection-analysis.json).

All **three D-077 re-grant attempts** pass this run. Every attempt retains its original readiness, member/node/origin packet observations, origin receipt and device counters. These are the captures the harness already made; detailed new WireGuard captures are not invented. `cause_status=unproven`; the earlier F14-01 remains failed and assigned to devstack's native investigation. A passing first selection or re-grant closes neither F11-01 nor F14-01. [All attempts](F15-D077-regrant-attempts.json).

AT04 retains **1,377 withdrawal samples with zero fresh parse-failure receipts**. Retention remains enabled: a rejection keeps `parse_failure` plus private raw/error files at mode 0600, never counter evidence and never published. Netlab 23's decision is consumed: every duplicate owned-chain definition, identical or not, is rejected as typed `DuplicateChainError`; its **0 / 3,300** lab result leaves origin unproven. Counterread's content-free `duplicate_chain` surface remains assigned to devstack's next follow-up. Both historical F13-01 failures stay failed, with no retry or predicate relaxation. [Fresh receipt index](F15-F13-01-private-receipts.json), [prior original receipt](history/follow-up-14/F14-F13-01-private-receipts.json).

Teardown exit 0. Final inventory: **0 owned containers / 0 networks / 0 volumes / 0 images**. Generated state and helper artifacts removed; required private pcaps, producer streams and parser files remain in ignored disk scratch. Root-owned state cleanup used an owned self-removing container, existing public PostgreSQL image, no network and only the owned state mount. Shared PostgreSQL, other lanes, host networking and caches were untouched. No sudo or new shell cleanup trap.

[Results](summary.md); [seven carried groups, no new requests](requests.md).

Final hygiene output:

```text
=== RUN   TestCommittedEvidenceHygiene
    evidence_test.go:56: scanned 200 acceptance JSON records plus documentation
--- PASS: TestCommittedEvidenceHygiene (2.82s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	2.826s
```

Archive verification: all 70 root files match their baseline bytes and SHA-256 pins; all earlier archives unchanged. The 102 new capture pins verify. Python gzip round-trip, corrupted-pin rejection and stale-file cleanup pass. New published evidence has no uncompressed file above 20,000,000 bytes.

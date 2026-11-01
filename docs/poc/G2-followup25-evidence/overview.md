# Follow-up 25: D-120 measurement end

TEST ONLY harness evidence. Production enforcement, the RELATED ICMP source
guard, D-120 clauses (i)–(v), the five-loss limit, and the 100-reload campaign
are unchanged. Real arm64 execution remains unmeasured.

## Retention and historical verdicts

[D-098 initial receipt](d098-initial.json) verifies all 609 follow-up 24 files,
complete manifest coverage, and the manifest digest
`ccbb579ff6c205dc8a5fab159c7acf06be201d4efa7656fbb82562357904b0ee`.
No earlier evidence tree was written.

[Analyzer identity](analyzer-identity.json) pins the entire analyzer and its
entire existing test file, byte-identical to task-start main. Replaying A2's
original inputs produces identical [before](a2-before-verdict.txt) and
[after](a2-after-verdict.txt) results: exit 1, generation 1087, sequence 57,
`(iii): next datagram missing`. Both runs use the same unchanged analyzer;
“after” does not append a synthetic successor to the historical input.
A2, the earlier branch attempt, and `gate2-replay` remain failed.

## Measurement boundary

`trace_capture.finish_stream` reads the completed Apply records and requires
the initial generation 1000 plus reloads 1001–1100. The last Apply must be
verified with a successful guard install/release and a release endpoint no
later than the completed Apply. Only then does it request the stream boundary.
The retained `stream-a1.log.trace-finish.json` includes that final bracket and
the request timestamp.

The stream is serial: each UDP send finishes its read before another UDP
send. Thus at an ordinary loop boundary every earlier lost datagram already
has a successor read; only the last datagram can lack one. If that last read
failed, record exactly one more TCP/UDP cycle. That is sufficient to observe
the immediate successor without changing campaign size or retrying any
sequence. If the final read already succeeded, no extra cycle is needed.

A failed successor remains the immediate successor and a real clause (iii)
failure. The count bound prevents trying a third datagram to get a better
result: the failed successor itself remains a lost tail with `missing`, and
`trace_end.end_reason` records `missing`. A TCP failure before that successor's
UDP send also leaves the original loss missing. The unchanged analyzer sees
all failures. No loss, TCP error, guard violation, or capture error is excused.
The underlying stream continues after measurement for the existing revocation
test, but later reads cannot alter the closed measurement.

The parent also has a 15-second watchdog. The pending cycle and one successor
each contain at most four socket operations with 1.5-second timeouts and a
0.2-second pause: 12.4 seconds, leaving 2.6 seconds for overhead. This is a
bounded measurement allowance, not a promise under arbitrary host stalls.
Expiry fails the run and preserves incomplete evidence; it fabricates neither
a read nor a successful boundary. An aborted/unfinished guard likewise fails,
with the available traces still saved.

## Offline proof

`tests/netlab/scripts/trace_end_test.py` executes the real stream loop with
synthetic sockets and a stop request arriving during the last lost read.
It performs no network I/O. [Successful successor](offline-ok.json) shows the
old one-datagram prefix failing as missing and the new two-datagram stream
passing. [Timed-out successor](offline-TimeoutError.json) and
[mismatched successor](offline-wrong_echo.json) retain failed/missing verdicts
with no third recorded send. Additional tests cover a TCP-blocked successor,
a tail whose earlier losses already have successors, absent stop requests,
missing/failed/unclosed final guards, and watchdog expiry.

## Live protocol

[Predeclared plan](attempt-plan.json): one full-suite live attempt, containing
one 100-reload scenario. Every attempt is retained; no A,B,A,B comparison is
owed for this harness-only change. No run is discarded for host state.

The foreground [receipt harness](harness/run.py) records `vmstat 5 7`, load
averages and `pgrep -af 'gates.sh|mk/at-'` before launch. All six measured
5-second samples must have r ≤ 3 and idle ≥ 80 to qualify as quiet; the first
vmstat CPU row is retained separately as the since-boot aggregate. Polling
stops after a quiet window or 60 minutes, then the attempt runs either way.
Suite admission waits while at least two matching suite processes exist and
rechecks immediately before and after startup. Public process receipts keep
PID/kind and the raw-output digest; raw arguments remain in ignored scratch
to avoid publishing other lanes' private paths.

## Lossless publication and reproduction

New JSON/JSONL larger than 1,000,000 bytes is published as deterministic gzip
(`mtime=0`). [Retention pins](retention-pins.json) record `bytes`/`sha256` for
the published compressed bytes and `uncompressed_bytes`/`uncompressed_sha256`
for the exact original bytes. Smaller JSON remains plain. No earlier tree is
compressed. Linked files use their retained path; source input paths inside
command receipts describe the original execution.

Command-log path substitution occurs before public retention. Where readable
text trims trailing whitespace, a `.raw.gz` sibling preserves the exact public
text before trimming; both forms are pinned. JSON/JSONL bytes are never
normalized. The committed-evidence test decompresses gzip before JSON/hygiene
checks and verifies both sets of hashes. It rejects truncated gzip and an
incorrect uncompressed pin.

Run all commands through `python3 tests/netlab/scripts/disktmp.py` for private
disk TMPDIR/GOTMPDIR. GOCACHE was obtained from `go env GOCACHE`.

- Offline tests: `python3 -B -m unittest discover -s tests/netlab/scripts -p trace_end_test.py`.
- Analyzer replay: use either `a2-*-command.json`; choose a fresh output path.
- Full live suite: `make -f mk/netlab.mk netlab-test FVPN_LAB_PREFIX=fvpn-w1-netlab-f25`, with a fresh `FVPN_NETLAB_EVIDENCE_DIR`, after the host/suite checks.
- Compressed analysis: decompress `at10-apply.jsonl.gz` into a new disk scratch directory, then pass it to the unchanged analyzer alongside the retained stream and capture files.
- Publication: `harness/run.py seal` compresses only this follow-up's new large JSON/JSONL and writes its manifest. Delivered evidence must not be overwritten by a later run.

AT10/AT13 support ISO01/ISO02 and §8.1/§8.2/§19's controlled reload and isolation
evidence. This follow-up improves measurement completeness; it makes no new
production capability or broader platform acceptance claim.

## Recorded live result

The single [full-lab attempt](gate2.json) passed in **242.681 seconds** including
setup and cleanup (Go suite: **231.726 seconds**). The reload scenario passed
with **100 reloads, 101 verified Applies, 62 datagrams, zero UDP losses and
zero TCP timeouts**. See the [D-120 verdict](live-attempt1/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt)
and [end-condition audit](live-end-audit.json). The maximum guard bracket was
51.260124 ms and maximum render-to-installed time was 30.796864 ms. All reads
were complete. The final release preceded the stop request and trace end.

This live stream had no lost tail and used zero extra cycles. The extra-cycle
path is established by the offline stream-loop proofs, not by a live loss in
this attempt. No second live attempt was run to obtain a preferred sample.

[Host qualification](host/quiet-result.json) took **1,412.128 seconds**, across
47 windows; all 46 rejected windows remain retained. The qualifying window had
max r=2 and min idle=81%. Load moved from 10.49/12.42/12.34 to
10.49/12.22/12.28 (one/five/fifteen minutes; values rounded).
[Suite admission](host/admission.json) took 0.022 seconds and found no matching
suite. The immediate prestart and [post-start check](host/poststart.json) also
found none. Host state did not alter acceptance criteria.

[Cleanup](cleanup.json) proves zero owned containers, images, networks and
volumes, with the scratch binary and directory removed. [Final D-098](d098-final.json)
rechecks 609/609 files and unchanged production/consumer/analyzer/campaign
sources.

The [Apply stream](live-attempt1/TestAT10_AT13_ReloadRevokeBaseline/at10-apply.jsonl.gz)
is **720,796 compressed bytes** for **13,646,265 original bytes**, a 94.72%
reduction. Both hashes are in the manifest. [Publication replay](publication-replay.json)
decompresses that file into disk scratch and runs the unchanged analyzer
against the published text captures: its complete result is byte-identical
to the original live analysis, and D-120 still passes.

## Final gates

| Gate | Result | Seconds | Receipt |
|---|---|---:|---|
| 1 — lane lint/unit/integration/build | PASS | 5.879 | [gate1](gate1.json) |
| 2 — full privileged lab | PASS | 242.681 | [gate2](gate2.json) |
| 3a — agent/updater unchanged | PASS | 83.450 | [gate3-agent](gate3-agent.json) |
| 3b — usage/counterread unchanged | PASS | 5.337 | [gate3-e2e](gate3-e2e.json) |
| 4 — evidence hygiene / D-098 | PASS | 1.944 | [gate4-completion](gate4-completion.json) |
| 5 — handover | PASS | 1.672 | [gate5-completion](gate5-completion.json) |
| 6 — root go-lint | PASS | 1.356 | [gate6](gate6.json) |
| 7 — go vet ./... | PASS | 0.328 | [gate7](gate7.json) |
| 8 — tracked Go formatting | PASS | 0.110 | [gate8](gate8.json) |
| 9 — git diff --check | PASS | 0.007 | [gate9](gate9.json) |

The initial [hygiene failure](gate4.json) is retained. The existing AT12
producer emits a Go struct dump under a `.json` filename.
[Publication naming](publication-names.json) records the rename to
`counters.txt` with identical bytes; it adds no new JSON-validation exception
and changes no producer source. Before sealing a fresh full-suite export,
apply that naming correction. The original [handover pass](gate5.json) is also
retained; hygiene and handover were then repeated in order against the
corrected publication. All other required gates passed on their first run.

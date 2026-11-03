# Follow-up 26 — D-104 precision fixture / isolation F16-01

TEST ONLY. This is a fixture disposition, not a production timing change or an
acceptance promotion. Isolation run `83b457ef` stays failed: the assertion
stopped it before sockets, Apply, and the mechanism trace. Its eight pairs are
orchestrator-supplied input, not a recovered trace.

The fixture correction is justified by source and live behavior. In the sole
resumed campaign, all 1,000 exact timeout round trips passed, but the old expiry
adjacency check rejected four listings. One normal transaction directly proved
different allocation bases; an unchanged-set control independently proved
per-element dump-clock sampling. The replacement accepted all 1,000 listings
without changing the deadline or packet predicate. F16-01's exact event remains
unattributed because its sorted pairs cannot distinguish these two mechanisms.

## What the old assertion proved

Follow-up 15 / commit `9b4383a1` (#290), its roadmap row, and
[the original contract](../G2-deadline-contract-2026-10-31.md) qualified a 1 ms
set-element quantum on `7.0.0-38-generic` / nft `1.0.9-1ubuntu0.1`. The old
line 34 required eight exact original timeouts, 1001–1008 ms. Line 35 additionally
required every remaining-expiry difference to be exactly 1 ms. The intended
claim was sub-second resolution, avoiding the known whole-second JSON loss.

Exact original timeouts already carry the 1 ms round-trip proof: the kernel
converts both timeout and explicit expiration from milliseconds into jiffies,
and converts stored timeout and remaining expiration back through the same
integer conversion. A coarser duration conversion cannot preserve all eight
consecutive values. Timeout does not age, so this part needs no simultaneous
clock samples. It qualifies the requested millisecond interface on this kernel;
it is not a claim about arbitrary kernels or physical arm64.

Line 35 also imposed an unnecessary condition on two independently sampled
clocks. With configured expires `C_i`, allocation sample `B_i`, and dump sample
`R_i`, all in the kernel's effective millisecond units:

`E_i = C_i + B_i - R_i` (while the element is live).

Thus the adjacent step is `1 + ΔB_i - ΔR_i`. One transaction's atomic visibility
does not make either group of clock reads simultaneous.

## Mechanism and alternative causes

The upstream version corresponding to the lab's kernel family calls the
allocation helper separately for each new element, before commit. That helper
reads the clock and adds the converted explicit expiration. The transaction's
own timestamp is a different field; it is not used for this initialization.
Commit activates these new elements without resetting their expiration.
Updates to existing elements have a separate path, which this fixture does
not exercise. See [kernel API source, v7.0](https://github.com/torvalds/linux/blob/v7.0/net/netfilter/nf_tables_api.c#L6688),
and the exact locations/digests in [sources.json](sources.json). This is upstream
source analysis, not a claim to have recovered the distribution binary's full
build provenance; the retained experiments establish behavior on the running
lab kernel.

The kernel also reads the clock **per element during a dump**, subtracting it
from the stored expiration. Its hash traversal is not address order.
[nft 1.0.9's netlink reader](https://git.netfilter.org/nftables/tree/src/netlink.c?h=v1.0.9#n1530)
sorts the result by key before printing. Consequently even a positive extra
step in sorted output does not identify insertion as its unique cause.
For F16-01, ages `2,2,2,2,2,2,1,1` fit both a later allocation of elements 7–8
and an earlier dump of 7–8, or a combination. The original run has neither
allocation timestamps nor raw dump order, so its individual cause remains
unproven. The orchestrator's allocation reading describes a real mechanism,
but the supplied eight pairs cannot uniquely establish it for that attempt.

Rounding is distinct: [kernel conversion](https://github.com/torvalds/linux/blob/v7.0/kernel/time/time.c#L766)
uses integer arithmetic; exact 1001–1008 read-back rules out coarse conversion
for this qualification. Default [nft text formatting](https://git.netfilter.org/nftables/tree/src/datatype.c?h=v1.0.9#n1020)
emits the millisecond remainder; the Python parser uses integer unit sums.
The installed matching-release [kernel build headers](kernel-headers.json)
also specify `CONFIG_HZ=1000`; for that configuration both conversions are
exact at integer milliseconds. This supports, rather than substitutes for,
the live round-trip measurement. No floating-point rounding occurs in this path. JSON's separate division by
1000 explains its lost precision and is why the text read remains mandatory.
Tick-boundary sampling can produce the observed step without changing the
quantum; calling it rounding does not justify a shared-clock assumption.

## Disposition and proofs

`deadline_precision.check_precision` keeps the **exact same eight-timeout
assertion**. It replaces only the adjacent-expiry equality with individual
live-countdown checks: `0 < expires_i <= configured_expires_i`. It continues
to require all eight pairs and retains the text and JSON listings in the trace.
The installed member read, deadline window, 100 ms install/guard bound, phased
100 ms packet cadence, 80 ms observation window and every packet predicate are
unchanged. No production file, D-104 predicate, D-140 predicate, or follow-up 27
verifier-content work is changed.

The existing installation argument already bounds every element's allocation
by `installed_by`: each expiration base lies inside the measured installation
bracket. It does not require equal bases. The 100 ms reserve and guard-release
check therefore retain their original meaning; this fixture correction adds
no allowance to the serving deadline.

The [offline replay](replay.py) compiles the original two assertions from the
task-start Git object. It exercises F16-01 before/after, all 19 retained listings
found from follow-up 15 onward (including immutable archive duplicates), and
complete deliberately quantized 2/4/10/1000 ms listings.
[Replay results](replay.json): F16-01 fails before and passes after; all 19
historical listings pass both; all four coarse listings fail both. These are
offline listing verdicts and do not reclassify any containing run. The small Python unit
suite also covers missing/reordered timeouts and expired/extended countdowns.

## Experiment protocol

[probe.py](probe.py) runs only inside a new owned lab-image container with
`--network none`. It uses the original add-set/add-eight-elements transaction
and normal text listing. Five additional read-only netlink dumps follow the
text listing, before the JSON read and set removal. The plan fixes 1,000
transactions without load injection, two padded transactions, and 2,000 dumps
of one unchanged set. Every trial and raw netlink reply is retained; no failure
is retried away. Later work per trial can affect subsequent ambient scheduling,
so the observed frequency describes this finite campaign, not a platform rate.

A raw dump preserves traversal order. If all allocation bases were equal,
`configured_expires - returned_expires` could only increase in that order.
A decrease proves unequal allocation bases; dump-clock advancement cannot
produce it. Persistent offsets across repeated dumps provide another check.
The padded controls put 5,000 unrelated set insertions between two groups
within one transaction, then reverse the groups. These are explanatory
controls, excluded from the normal-condition rate. Repeated reads of an
unchanged set isolate dump-time changes without any new allocation.

## Measured results — sole resumed campaign

[Experiment receipt](experiment.json), [preflights](admission/experiment.json),
[post-run process check](monitor/experiment.json), and
[derived analysis](mechanism-analysis.json) retain the complete result. The
campaign took 24.311 s after qualification; no other-lane overlap was observed.
Kernel `7.0.0-38-generic`, nft `1.0.9-1ubuntu0.1`. The original interrupted
attempt contributes no measurement.

| Observation | Result |
|---|---:|
| Exact 1001–1008 ms timeout round trips | 1,000 / 1,000 |
| Old adjacency-check failures | 4 / 1,000 (0.4%) |
| Replacement-check failures | 0 / 1,000 |
| Direct allocation-split witnesses in normal transactions | 1 / 1,000 (0.1%, witnessed lower bound) |
| Exact F16-01 age-vector shape recaptured | 0 / 1,000 |
| Normal trials with changing relative ages across subsequent dumps | 16 / 1,000 |
| Padded controls proving a split, in opposite group orders | 2 / 2 |
| Unchanged-set dumps with nonuniform ages | 2 / 2,000; the other 1,998 were uniform |

The four original-check failures are zero-based trial indices **70, 281, 406,
758**. Across the 7,000 adjacent pairs in normal text listings, six steps were
2 ms, four were 0 ms and 6,990 were 1 ms. This reproduces the extra-step failure
class, but not F16-01's exact split after element 6. No attempt was repeated for
a better result. “Normal” here means the original eight-element transaction
without padding or injected load in the quiet isolated tools container; it
uses a fresh `inet fvpn` table, not the full assembled topology/baseline.
The finite observed rates are not estimates of an assembled-run failure rate.
The other three rejected text listings are not assigned individual causes from
subsequent dumps; those are different reads.

**Allocation witness:** normal trial index 758 lists expires
`699,700,701,703,704,705,706,707`. The first raw dump returns key 3 before key 6:
key 3 has configured expires 703 and observed 701 (age 2 ms), then key 6 has
configured 706 and observed 705 (age 1 ms). Time advancing during a dump cannot
make age decrease if bases are equal. Hence key 6's allocation base is at least
1 ms later than key 3's. The same offset persists through all five dumps.
This directly establishes split allocation timing on the running kernel,
independent of nft text ordering. See [all normal replies](mechanism-normal.jsonl.gz),
including raw netlink bytes and order.

**Transaction controls:** inserting 5,000 unrelated elements between the two
groups inside one `nft -f -` transaction creates at least a 2 ms base separation
in both directions. Forward order gives older elements 1–6; reverse order gives
older elements 7–8. Raw age decreases are key 6 → 8 (4 → 2 ms) and key 8 → 3
(4 → 2 ms), respectively. Atomic publication therefore does not reset these
new elements to one common transaction/commit base. The allocation source
identifies the exact initialization operation; the live controls corroborate
its placement before commit. [Padded replies](mechanism-padded.jsonl.gz).

**Dump-clock witness:** one unchanged set is read 2,000 times. Dump index 398
has age 13 ms for every element. Dump 399 reads keys `6,3,8,7,2,5` at age 13,
then keys `1,4` at age 14; dump 400 has age 14 for all. Dump 399's entire
userspace request/reply bracket is 27,827 ns. A second analogous crossing
occurs at dump 693. There are no intervening writes, and the relative-age
vector returns to uniform immediately after each crossing. This proves the
listing clock alternative on the lab kernel; nft subsequently sorting the
keys obscures that temporal order. [All unchanged-set replies](mechanism-dump.jsonl.gz).

Both duration conversion and raw netlink data agree with the exact integer
millisecond text path. Clock advancement at allocation and at dump is observed;
coarse conversion or a text-parser rounding error is not needed to explain
these failures. Neither live control supplies the missing raw order/timestamps
from the original `83b457ef` run, which remains failed and causally ambiguous.

## Reproduction and retention

D-098 was checked first: 163/163 files, complete coverage, both gzip forms,
manifest SHA-256 `61331386b3e8c4cd86f644dbd141aea6232d1b036e10ce7a9647541c0afc13a0`.
All earlier trees remain byte-identical. New artifacts live only here.

Run host commands through `python3 tests/netlab/scripts/disktmp.py`. The
foreground [receipt runner](run.py) calls the orchestrator's admission script
(selected by `FVPN_F26_HOLD_SCRIPT`) and checks
`pgrep -af 'gates.sh|mk/at-|go test'` against actual executable arguments.
Prompts and shell command text are not runs. The resumed experiment and full
netlab gate require three consecutive 60-second free windows; all commands
are checked again every 60 seconds while running. There is no wait deadline,
and other lanes' processes are never touched. Obtain
GOCACHE with `go env GOCACHE`; the actual values belong in the ignored report,
while public receipts identify the queried cache by digest.

For fresh experiments, use a new evidence destination: delivered files must
not be overwritten. Build `tests/netlab` with an owned image tag, run it with
an owned container name, `--privileged --network none` and a read-only worktree
mount, then invoke the three probe modes via `docker exec`. Remove that
container and image afterwards. `run.py gate NAME COMMAND...` records a
foreground gate; the final full suite uses
`make -f mk/netlab.mk netlab-test FVPN_LAB_PREFIX=fvpn-w1-netlab-f26`.
`run.py seal` pins new artifacts and losslessly compresses large JSON.

## Resume (2026-11-02)

The orchestrator reports that the host powered off at 13:13 EST on November 1
and returned at 22:29 EST on November 2. The interrupted experiment is **UNRUN**.
Its setup/admission receipts live only in [interrupted-2026-11-01](interrupted-2026-11-01/status.json);
no measurements are claimed from them. No completed experiment output survived.
The owned mechanism container was observed exited with code 137 and removed
before reuse. Source analysis, original-input replay and implementation commit
`5f79644d` survived. [D-098 was rechecked](d098-resume.json) before resuming.

[The resumed plan](resume-plan.json) fixes the original counts and decision rule,
pins the probe and analysis source before execution, and permits one campaign
with no retry for a better result. The new [campaign driver](campaign.py)
creates the isolated container only after quiet qualification, executes all
three modes in order, and removes it in Python cleanup.

### Admission wrapper correction (2026-11-03)

Before gate 2 started, `hold.sh` matched the **queued** `netlab-test` text in
our two waiting Python wrappers. No lab command was running. The
[exact public match](hold-wrapper-false-positive.txt) and
[disposition](wrapper-disposition.json) preserve the false admission hold.
Only the owned waiting child was interrupted; its disk-temp parent unwound.
The original preflight receipt is retained as
[gate2-queued-wrapper](admission/gate2-queued-wrapper.json). This is an abandoned
preflight, not a failed or retried lab attempt.

The owned runner now accepts a command plan file, so a waiting process carries
no future gate command in its arguments. It still requires `hold.sh` to print
`free` and applies the same real-process checks. The actual `make` and `go test`
children remain visible once launched. No private admission-script edit or
other-lane process change was made. Use `run.py plan gate2-command.json` (with
the full evidence-relative path) and the same disk-temp wrapper.

## Required gates and cleanup

All nine required gates passed in order. Gate 1 includes 29 Python tests and
amd64/arm64 builds. The full lab package took 227.651 s (235.272 s including
setup/teardown/checks), after three quiet windows; no other-lane overlap was
observed. Its unchanged D-104 trace has 26 positives and 7 drops, last positive
D−100.703983 ms, first drop D−0.742217 ms, render→installed 12.810495 ms, and
precision read 1.290625 ms. No later or post-deadline positive occurred.
[Live trace](live-attempt1/TestD104_AT13_DeadlineContinuity/trace.json).

| Gate | Result | Seconds | Receipt |
|---|---|---:|---|
| 1 | PASS | 9.135 | [gate1](gate1.json) |
| 2 | PASS | 235.272 | [gate2](gate2.json) |
| 3 | PASS | 8.016 | [gate3](gate3.json) |
| 4 | PASS | 2.124 | [gate4](gate4.json) |
| 5 | PASS | 4.943 | [gate5](gate5.json) |
| 6 | PASS | 5.030 | [gate6](gate6.json) |
| 7 | PASS | 0.387 | [gate7](gate7.json) |
| 8 | PASS | 0.170 | [gate8](gate8.json) |
| 9 | PASS | 0.071 | [gate9](gate9.json) |

The isolation consumer passed without edits. Root lint, vet, formatting and
diff checks passed. [Cleanup](cleanup.json) records zero owned Docker objects
and absence of all known own build manifests. [Final D-098](d098-final.json)
again validates 163/163 follow-up 25 files and every older G2 tree.
[Finish environment](environment-finish.json) records the queried cache digest
and unchanged cache location; the literal path is only in the ignored report.

The AT12 producer still emits a Go struct dump named `counters.json`; this new
publication retains its exact bytes under `counters.txt`, with the
[hash/name mapping](publication-names.json). No producer or older evidence was
edited. Large new JSON/JSONL uses lossless gzip and dual byte/hash pins; text
whitespace presentation retains exact public raw bytes alongside when needed.

The private `hold.sh` matcher correction is an advisory hand-off dated
2026-11-03. The owned wrapper workaround is complete; no external edit blocks
this delivery. Physical arm64, other kernels and suspend/resume remain
unmeasured. Follow-up 27 remains deferred.

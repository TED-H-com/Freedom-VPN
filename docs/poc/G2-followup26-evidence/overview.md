# Follow-up 26 — D-104 precision fixture / isolation F16-01

TEST ONLY. This is a fixture disposition, not a production timing change or an
acceptance promotion. Isolation run `83b457ef` stays failed: the assertion
stopped it before sockets, Apply, and the mechanism trace. Its eight pairs are
orchestrator-supplied input, not a recovered trace.

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

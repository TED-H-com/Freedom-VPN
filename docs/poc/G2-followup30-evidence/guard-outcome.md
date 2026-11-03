# Guard drops attributed; B1 remains failed and unresolved

The one declared window completed: **100 reloads, 101 verified Applies, 118 TCP
reads, zero TCP read failures**. All 381 TCP segments correlate across endpoint
and node captures. Seven missing node egresses each have exactly one matching
NFLOG drop and no contradictory egress: five original request data packets,
one request retry, and one ACK. Every matched verdict is
`inet fvpn fv_deadline_guard iifname "fv-fh" drop`, member → responder, inside
its generation's install-command bracket. The [seven-row ledger](drop-ledger.md)
names generations **1002, 1018, 1019, 1041, 1069, 1070 and 1093**, packet identities
and command endpoints. No missing probe TCP segment remains unattributed.

**The window is unqualified for host quietness and is spent.** One interval,
11:28:42.462–11:28:46.458 UTC, overlaps collector readiness at 11:28:45.263 and
the first TCP send at 11:28:45.459. It reports **2.478498 foreign cores**, above
0.5. No other lane suite or lab container was seen. Earlier startup intervals
also exceeded 0.5; the after-startup preflight waited 12.103 seconds and accepted
a fresh quiet check before launch, but the next interval breached. The raw
[qualification](guard-window-qualification.json), [all four-second checks](guard-window-quiet.jsonl)
and [overlap derivation](window-summary.json) are retained. No retry or quietness
waiver occurred. Positive rule/packet evidence remains recorded; no quiet-host
performance claim is made.

## What this proves, and the limit for B1

This directly demonstrates the proposed guard-loss mechanism for this window.
Cycle 26's original request is dropped in generation 1018, its retry at
**+208.154 ms** in generation 1019, and its next retry at **+416.030 ms** traverses
the path. The read completes at **416.739885 ms**, the longest of 118 reads.
Cycle 92 also loses an ACK; the responder retransmits its reply 207.464 ms later.
All seven retransmissions and the repeated ACK remain in the complete analysis.
No socket reconnect, changed probe, relaxed timeout or analyzer change was used.

This **does not explain B1's failed read**. B1's sequence 43 still times out at
**1,501.407973 ms** against the unchanged 1.5-second timeout. Its send overlaps
1064's install bracket; that guard's release ends only **14.469408 ms** after the
send. Fifteen guard brackets overlap the read, including +199.467–223.614,
+404.030–427.552 and +821.378–857.635 ms ([retained B1 timeline](b1-context.json)).
Those alignments can support a repeated-loss/backoff hypothesis, but B1 has no TCP
packet or NFLOG capture to identify its request, reply, retries or pending timer.
This run's successful 416.740 ms read cannot fill that historical gap. F27-01
remains unresolved, B1 remains FAIL, and follow-up 28's gate 2 remains FAIL.

## Collector and diagnostic cost

The [live prerequisite](collector-outcome.md) passed before this reservation.
This window captured and decoded **18 NFLOG records**, with **18 received / 18
captured / zero kernel drops**. All 18 have 1,000 ns pcap precision and a kernel
timestamp; none has an NFLOG sequence attribute. Seven are the probe TCP drops;
the other eleven comprise nine deny-probe TCP packets, one spoof-source ICMP
packet and the single D-120 UDP boundary loss. The independent TCP captures also
report zero receiver/kernel drops and no unmatched observations. Absence of a
sequence attribute prevents a general claim of lossless kernel logging; the
positive probe matches and finite collector check supply the stated evidence.

Observed executor timings across all 101 Applies:

| Interval | Median (ms) | p95 (ms) | Maximum (ms) |
|---|---:|---:|---:|
| Guard install command | 13.684706 | 24.470368 | 69.025183 |
| Guard release command | 13.153761 | 22.889335 | 35.959668 |
| Install start → release end | 28.471576 | 43.438670 | 94.665010 |
| Installed-by → release start | 0.005087 | 0.008232 | 0.014273 |
| Producer install start → release end | 28.493114 | 43.459978 | 94.684882 |

The decorated script adds exactly **138 bytes per Apply**. The 18 records contain
1,044 IP bytes; derived JSONL is 9,975 bytes. The discarded private raw pcap was
3,664 bytes. Collector start/stop command times were **62.928116 / 57.258627 ms**;
internal start-to-ready was **11.070288 ms**. Read-only TCP/conntrack snapshots
outside Apply total **5.367775832 s**. Executed scripts/hashes and both timing
sources are retained for every Apply. The outer brackets bound command execution,
not exact atomic commit times. These observations bound this instrumented,
unqualified run; **incremental overhead against an uninstrumented control is not
isolated**. Production change/cost is zero; this is no production D-133 pass.

The [command output](guard-window.txt), [receipt](guard-window.json),
[binary digest](guard-window-binary.json), [guard analysis](guard-analysis.json),
[complete TCP analysis](tcp-analysis.json.gz) and [cleanup inventory](guard-window-inventory.json)
are retained. Cleanup leaves zero owned containers, images, networks, volumes or
task scratch. D-120 passed with its one explicitly classified UDP boundary loss;
no historical result is replaced.

## Proposal only — separate D-133 work

A hitless guard hand-over is a candidate for a new design/comparison assignment:
keep the last verified permitted path until one atomic replacement establishes
the complete new policy, while still closing forwarding on invalid/expired state,
late commit, crash, verifier failure and rollback. The design must show how it
preserves deadline, revocation, baseline and failed-Apply guarantees before any
implementation. Removing the guard or granting a blanket established-flow
exception is not justified by these observations.

Any implementation needs a predeclared D-133 A/B/A/B comparison against the exact
current guard on an admitted quiet host, with the same 100-reload probe, unchanged
1.5-second timeout and D-120 analyzer, all old failures retained. Compare complete
Apply/install/release/verifier distributions and observed loss/read deadlines;
exercise crash, late-commit, expired-lease and rollback failure paths, and record
amd64 plus arm64 evidence. Instrumentation must be identical between legs, with
its own cost disclosed. A B-only timeout or unqualified leg cannot be accepted
by substituting a later clean run. No such production change is made here.

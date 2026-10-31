# Isolation follow-up 10 — 2026-10-31

Current assembled run **`bfcc3880-dc02-485d-990c-13811d6e8835`**, live source `cd2e89746b710eda1580d2c822ceff3f636fb013`,
source dirty **false**, product baseline `96ec0630` (through PR #344).
The assembled driver ran **1,096.70 seconds**; the full command took
**1,226.199 seconds** and returned **2 (lab=0, stack=1)**. All production paths
remain unchanged. The two earlier attempts remain separate
immutable evidence: [first attempt](first-attempt.md), [second attempt](second-attempt.md).
No later pass reclassifies a failed sample.

| Case | Result: pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **pass: 9 / 0 / 0** |
| [AT11](AT11.json) | **blocked: 2 / 1 / 0** |
| [AT12](AT12.json) | **pass: 10 / 0 / 0** |
| [AT13](AT13.json) | **blocked: 6 / 1 / 0** |

Spec §19 AT10–AT13, §8.2–8.3, §16.3; ISO01–ISO04, ENT02, SEC01, DNS03.
The two carried operator items are stated once in the current
[requests](requests.md#follow-up-10-requests--2026-10-31); they remain open.
Mechanism evidence and helper cross-compiles do not establish physical arm64.

## D-098 / D-132 and retained attempts

Before any root evidence rewrite, commit `5e9cb21d` archived all **86** follow-up
9 root files byte-identically, including SHA-256 pins, under
[history/follow-up-9](history/follow-up-9/archive.json). All 86 also compare equal
to their original main bytes. Follow-ups 6/7/8 are untouched: **44/52/80** pins,
including netlab 20's three exact follow-up 8 digests. Handover uses immutable
claims; its living AT10/AT13 path strings are rejection-test data, not file
reads. No D-132 handover change or pin move was needed.

The [first archive](history/follow-up-10-attempt-1/archive.json) pins 92 original
files and explicitly labels 23 inherited files as not new measurements.
Its finite-member setup was incompatible with the fixed 60-second node window;
a separate real install-bound failure occurred during renewal, and subsequent
expiry/recovery lacked their required positive controls. It retains D-087 pass
at 2,035 ms, 49 failed Applies and 279 renewal losses.
The old reviewer rejected mixed clock/namespace inputs. Current review refuses
unexecuted expiry windows, and generated artifacts were cleared before reruns.

The [second archive](history/follow-up-10-attempt-2/archive.json) pins 98 files.
Its initial counted flow incorrectly followed the stability wait, allowing
ordinary initial FUP projection to change generation 12→13 before false.
The exact fixed-generation predicate failed; true was cleanup only, with no
new serving duration. Its D-104 pass, seven renewal losses and eight failed
pre-install Applies remain separate measurements. The corrected third run
counts first and then waits for stability. Both earlier attempts ended with
zero owned Docker inventory.

## AT10 on the real D-087 trigger

A fresh dedicated `FVPN_E2E_GRANT_SEAM=1` runtime uses wire-nodes 33's reader
factory through devstack 20's exported loopback adapter. The TEST ONLY offline
window was set to **60 seconds before enrollment** and remained fixed.
Initial native counted traffic precedes **65 seconds of identical invariants**,
allowing ordinary initial FUP projection to settle. The node is published,
unfrozen, ready/passed, has bindings, and holds applied=desired **generation 13**.
Real accepted observations retain baseline, meter and tunnel checks.

False completed at **19:09:31.409608131 UTC**. A natural signed empty lease
`bb2890af-dd8a-4626-b536-bbaaa9ba2f10` completed at **19:09:32.347800630**.
The actual **HTTP 202** provisional receipt completed at **19:09:32.471404691**:
ready/passed, forwarding false, exactly `awaiting_member_grants`,
`AWAITING_MEMBER_GRANTS` on lease/qos/source_binding and exactly one error with
that code at `stage=lease`. It names that empty lease; expiry or a safety failure
does not supply the receipt.

Only then did true complete at **19:09:32.571782557**. A new populated signed
lease `ecafb4a5-6440-4bd0-ace4-df3e88a7e523` completed at **19:09:33.390501686**;
a distinct accepted serving report completed at **19:09:33.534230637**.
True→accepted serving is **962 ms** (**962.448080 ms** from original timestamps).
The subsequent native counter increases **2,633→5,266 bytes**. Member, node
ingress and uplink each capture 16 packets with zero capture drops; first
uplink packet **19:09:35.490018** is after the serving receipt.

The independent [audit](member-grants-audit.json) verifies **297 original wire
hashes**, three desired responses during the transition, identical ETags and
byte-equal before/provisional/after invariants: envelope, generation, bindings,
ready/passed state, publication, freeze, entitlement and device outputs.
[Original wire](member-grants-wire.json), [capture sequence](member-grants-transition.json)
and [actual node acceptance joins](member-grants-timing.json) retain separate
server, node and observation samples. No scheduled slot is treated as acceptance.
No compiler change, projector stop, forced generation, freeze, unpublish,
transport hold, synthetic report or synthetic lease drives this transition.

| Independent sample | True→accepted serving, ms |
|---|---:|
| Historical isolation | 4,767 / 5,021 / 5,025 |
| Devstack 20 | 3,980 / 4,112 |
| Isolation 10 first attempt | 2,035 |
| Isolation 10 second attempt | No qualifying transition |
| Isolation 10 current run | **962** |

The separate 100-reload stream passes **unchanged D-120**: 101 verified Applies,
45 datagrams, one eligible UDP loss, zero TCP timeouts. Packet 37 at generation
1059 is a boundary loss; the next datagram succeeds. Its outer guard is
45.873661 ms and render→installed 26.081340 ms; maxima across the stream are
50.399956 / 30.949933 ms. All original conditions (i)–(v) remain required.
[Printed verdict](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt).
D-120 is never applied to the separate renewal/expiry diagnostics.

## New expiry and renewal measurements

D-104 packet predicate: **pass**, D=`58085626424739` ns,
**161 positives / 41 drops**. Last positive offset
**-99.655342 ms**, first drop offset
**-63.564818 ms**. There are no later or post-D positives; packet capture drops are zero.
The original signed deadline and
D−101 ms / D+100 ms bounds are unchanged. See the [packet verdict](deadline-verdict.json),
[producer correlation](expiry-failure-correlation.json) and
[all loss brackets](expiry-timing-analysis.json).
**One pre-deadline ingress loss remains counted**;
literal zero-pre-deadline loss is **false**.

Historical follow-up 9 remains **failed: 96 early drops / 65 subsequent positives /
41 final expiry drops** in its [immutable verdict](history/follow-up-9/deadline-verdict.json).
Its first baseline writer remains **unattributed**. A new bounded-expiry pass or
restart adoption cannot resolve that historical row or strict ISO-F6-03.
Fresh baseline rows name only fresh agent writes; watchdog decisions and writes
come from retained reason logs, never inferred from a generation-0 read alone.

The independent 80 ms renewal diagnostic has **1,396 probes /
1,366 positives / 30 losses**, two actual
accepted renewals and 69 completed guards.
**30** lost ingresses lie in completed guard brackets;
**0** remain outside those brackets. Every loss remains a
loss. [Exact renewal correlation](continuity-analysis.json) retains original
Apply, render, install/release, verifier, baseline and heartbeat samples.

## Completed-guard distributions and failed Applies

The original [producer stream](producer-operations.jsonl) retains **8,088**
contiguous snapshots. Source counts: baseline=128, heartbeat=3949, renewal=73, apply=975, live=2926, blocked_audit=37.
New baseline/heartbeat kinds are included in sequence and timing validation.
The retained watchdog log explicitly names its bootstrap `heartbeat_unreadable`
write at 19:06:25 UTC; it contains no later write. This is fresh evidence only.
Reason logs preserve pre-restart, pre-clone and final segments; concatenated
[agent](agent.log.txt) and [watchdog](watchdog.log.txt) logs retain those originals.

Nearest rank over completed guards only, milliseconds:

| Population | n | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|---:|
| Outer guard | 968 | 31.085118 | 40.528987 | 48.851468 | 52.927957 |
| Matching install command | 968 | 16.109632 | 22.319402 | 27.699284 | 30.619698 |

All seven failed Applies reject before install during the labelled foreign-tc
scenario; no install-bound overrun occurs in this sample. Failed Applies are
listed separately; full command flags, partial samples and
verifier brackets remain in [producer-timing-analysis.json](producer-timing-analysis.json):

- Process `f5e8c05d-bf49-4021-8351-509b40d1c7d6`, sequence **2310**; Apply `58113062105883`–`58113114374641` ns, completed guard=false.
- Process `f5e8c05d-bf49-4021-8351-509b40d1c7d6`, sequence **2317**; Apply `58114055110218`–`58114100292478` ns, completed guard=false.
- Process `f5e8c05d-bf49-4021-8351-509b40d1c7d6`, sequence **2325**; Apply `58114252138103`–`58114308910214` ns, completed guard=false.
- Process `f5e8c05d-bf49-4021-8351-509b40d1c7d6`, sequence **2332**; Apply `58115244686480`–`58115301221571` ns, completed guard=false.
- Process `f5e8c05d-bf49-4021-8351-509b40d1c7d6`, sequence **2340**; Apply `58116256699009`–`58116309545044` ns, completed guard=false.
- Process `f5e8c05d-bf49-4021-8351-509b40d1c7d6`, sequence **2347**; Apply `58117253417988`–`58117307302601` ns, completed guard=false.
- Process `f5e8c05d-bf49-4021-8351-509b40d1c7d6`, sequence **2354**; Apply `58118248366905`–`58118300918350` ns, completed guard=false.

Earlier completed-guard samples remain independent:

| Attempt / population | n | p50 ms | p90 ms | p99 ms | max ms |
|---|---:|---:|---:|---:|---:|
| First / outer | 678 | 31.634581 | 41.827776 | 51.362069 | 57.630034 |
| First / install | 678 | 16.459780 | 22.524217 | 28.101933 | 30.792366 |
| Second / outer | 931 | 30.749663 | 38.828002 | 46.306456 | 49.844866 |
| Second / install | 931 | 15.779444 | 21.512728 | 26.901864 | 30.439422 |

The first attempt's 49 failed installs are **excluded from completed-guard
percentiles and retained separately**, including Apply 712's 215.890796 ms
render→installed bound violation and baseline 713's explicit `apply_rollback`
writer. The second attempt's eight pre-install rejections occur in the labelled
foreign-tc scenario. Full sequence lists are in the two attempt documents.

Agent-core 28's separate D-133 A/B reference measured live outer-guard p99
51.67 / 50.46 ms versus unchanged main's 38.07 / 40.34 ms, and assembled
43.60 / 41.24 ms versus 47.89 / 49.75 ms. These are separate samples; this run
is not a controlled A/B experiment and supplies no causal performance claim.

## Recovery, digests and reconciliation

ISO-F6-01 **passes all six assertions** in the
[recovery verdict](recovery-verdict.json), with genuine observations, populated
lease, durable applied state, kernel topology and counted forwarding.
Injection→containment ≤445.418286 ms;
removal→link-up ≤652.416785 ms, serving
≤8485.502956 ms, counted completion
≤17728.987910 ms. These are observation upper bounds,
not product guarantees. Recovery assistance=false. The original runtime log
prints the preliminary recovery check as blocked; the existing six-assertion
review joins durable verified Apply and fresh live evidence before the final
AT10 record is written. That final review passes; no packet predicate changes.
The node-any capture has 2,392 packets and zero capture drops: two redirected
packets and one ordinary SNAT packet precede confirmed containment, and no
uplink packet appears after containment until removal.
There are 12 retained rejected reads and
0 unsolicited rejected reads. The original
unretained follow-up 6 foreign-object creator remains unknown.

D-106: **654/654** canonical wire digests recompute;
**654** also match original producer reads.
[D-093 reconciliation](d093-reconciliation.md) preserves all 15 agreements,
13 open / two retired only within simulator scope; the new grant seam changes
none of those simulator claims.

## Gates and cleanup

Exact foreground gate results and output tails are in
[validation-results.json](validation-results.json). The assembled command's
exit 2 preserves the two current operator gaps. Owned lint/test/build, handover,
root go-lint, formatting, diff and committed-evidence hygiene all **pass**.
Only four current text captures had terminal blank lines normalized; the D-120
JSON replay is byte-identical afterward. All original JSON wire and archive
bytes remain untouched. Final owned-prefix [inventory](cleanup-inventory.json):
**containers=0, images=0, networks=0, volumes=0**.
Four items are routed under [Requests to orchestrator](requests.md#follow-up-10-requests--2026-10-31).

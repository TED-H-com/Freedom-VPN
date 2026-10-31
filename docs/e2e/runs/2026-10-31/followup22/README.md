# Follow-up 22 — native IPv6 path and first route selection

TEST ONLY, Linux amd64. This is endpoint mechanism evidence for §8.3 / §19
AT09 (EXIT02 / APP01 / NFR01) and initial-route diagnostics for §19 AT01.
It is not four-platform, physical arm64, QUIC or full PMTU acceptance.

D-098 was first: `716f8144` preserves **63** follow-up 21 evidence files with
[immutable SHA-256 pins](../../../followup21-archive.sha256) and byte-identical
ignored disk copies. The two loose members analysis files have immutable
copies under `followup21-archive`. Earlier follow-ups, devstack 20's `cce1a688`
and members 11's IPv6 row are unchanged.

## First campaign: retained failure and explicit-path control

Run [004d6156](004d6156-3a1f-408c-ab1a-087a971965c3/ipv6-asymmetric.md), source
`ede844d4`, is **12 pass / 0 blocked / 1 fail**, process exit **1**. The original
members mutation order remains IPv4/1500 → IPv6/1500 → dual/1500 → dual/1280,
each with native down/up, connected-peer observation and three-second settle.
It then adds separately labeled IPv6/1280 asymmetric and explicit-path controls.
IPv4 overlay traffic is carried over the observed underlay; IPv6Block is not
an explanation for loss of that IPv4 stream.

| Row | Selected underlay | Validated bytes | Elapsed ms |
|---|---|---:|---:|
| matrix4 | IPv4 P2P | 64,064 | 2 |
| matrix6 | IPv6 P2P | 64,064 | 2,088 |
| matrixDual | IPv4 P2P | 64,064 | 1 |
| matrix1280 | IPv6 P2P | 64,064 | 1 |
| ipv6-asymmetric | IPv6 P2P | **10,010** | **12,007** |
| ipv6-explicit1280 | IPv6 P2P | **64,064** | **3** |

The original four-row sequence passed this attempt. Its 1280 row sent only
1,152-byte reverse data packets. The additional IPv6 asymmetric row failed;
it is not represented as the historical members failure or as a failed
original four-row replay. The failing configuration still lowers only member
eth0/eth1. The selected node underlay stays 1500, both native tunnels 1280.

[Complete header analysis](004d6156-3a1f-408c-ab1a-087a971965c3/headers/analysis.json)
retains exact packet numbers, TCP ranges, multiset outer-packet hashes and
per-hop clock bounds. In the failed row:

- **23 / 23** reverse outer IPv6 datagrams of **1,360 bytes** leave node eth2
  and are absent at member eth1. Every smaller selected-endpoint datagram
  arrives, with no unmatched receiver datagrams.
- First missing TCP range **[3516855168,3516856396)** is **1,228 bytes**.
  Origin and node `fvup0` ingress / `fvmb0` egress retain it and retransmissions;
  member tunnel ingress does not. Loss is after node underlay egress and before
  member underlay ingress, beyond enforcement.
- All **52** upload fragment pairs (**1,280 + 136** bytes, second offset 1,232)
  arrive. There is **no PTB or IPv4 fragmentation-needed** at any capture.
- Member/node/origin capture counts are **600 / 524 / 122**, respectively;
  all match decoded counts, with **zero kernel drops**, truncations or errors.

The explicit fixture adds only a sender `/128` route to the selected member
endpoint, with **MTU 1280**, then deletes that exact route. Before/after route
readbacks show `metrics: [{mtu: 1280}]` during the control and its removal.
No global interface/tunnel MTU, MSS, enforcement, health or transport code changes.
This is prior knowledge of path MTU, chosen to isolate sender packet sizing
without a new router or ICMP policy. It does **not** test reactive PTB discovery.

The explicit-path capture contains one reverse fragment pair, both received.
Crucially, it belongs to a retransmission of the **older timed-out flow** (port
49350): the formerly missing range reaches member tunnel packet **34**, at
`1793483101.237016`. The new successful bulk flow is port **44410** and its
reverse packets are smaller. These are separate observations: the route
constraint makes a previously oversized retransmission deliverable, and a new
64,064-byte transfer completes. No PTB appears; no claim of learned ICMP PMTU
or recovery of the already timed-out application is made.

This is an **asymmetric receive-link fixture artifact**, not a measured routed
1280 bottleneck or a certified real member-platform link. Receive-MTU rejection
is supported by the size-selective loss and explicit-route control; no kernel
drop-reason trace was taken. If future real-platform evidence establishes a
known 1280 receiver, peer/path-MTU signaling and reverse packetization belong
to the pinned NetBird/WireGuard transport integration, not `node/enforce`.
The native binary is selected in `deploy/devstack/Dockerfile.node`; the owned
fixture changes the member link in `tests/e2e/native_underlay.go`. This task
neither chooses a production MTU nor asserts a production repair.

## F11-01: first use is observed, historical five are not reconstructed

[Initial selection](004d6156-3a1f-408c-ab1a-087a971965c3/initial-selection.md)
starts captures before selecting the product-created exit. It retains route
and selected-peer readbacks through selection, the original two-second settle,
a fixed-source-port **20001** request with a **5-second dial / 6-second HTTP**
bound, node counters and authenticated origin receipt. It runs before the
ordinary counted-flow retry. No reconnect, serving repair or manufactured
health state is inserted.

This attempt has **one SYN**, sequence **3425414161**: member tunnel packet 13
at `1793483042.644275`, node tunnel packet 95 at `1793483042.648243`, node uplink
packet 102 at `1793483042.648309`, origin packet 27 at `1793483042.648310`.
The request succeeds in **6 ms** and both device counters increase. Native
status is initially pending and later reports **Relayed**; an absent ICE
endpoint is not a proven transport failure. This is a fresh first-use sample,
not the complete members 11 history. The five historical SYNs/retries were
not reproduced and cannot be identified in these new pcaps. No node fault
is assigned. Their original first-unconfirmed interval remains member tunnel
egress → selected node ingress until members 12 captures that failing sequence.

The first campaign has complete packets but incomplete origin link/route
metadata (`ip` was absent in that image), and lacks `wg` readbacks. These missing
reads are explicitly false, not inferred successful. Subsequent observer work
adds origin iproute2, the public WireGuard diagnostic utility, correct selected
tunnel name, native kernel-interface flag, relay endpoint metadata and sampling
during the first transfer. The first campaign is retained unchanged.

## Export, reproduction and boundaries

[Shared export contract](../../../scenarios/harness-exports.md#follow-up-22--selected-native-path-capture-test-only)
defines `StartNativeCapture`, `Sample`, `Stop` and the separate
`ConstrainSelectedNativePeer1280` control. Members owns adoption in
`tests/e2e/at/members/forwarding_test.go`; it is not edited here.

All raw pcaps remain in ignored `.agent/tmp/followup22/<run>/<row>/<role>.pcap`.
Public files contain headers, opaque address tokens and packet hashes only.
`*.bounds.jsonl` joins each original packet number/wall timestamp to an
**empirical** monotonic offset interval using the same-boot clock pairs.
This assumes no intervening wall step; offset stability and microsecond pcap
quantization are recorded. It is not a hardware timestamp or an exact route
commit time. Matching encrypted hashes proves outer receipt; time proximity
alone does not assign an encrypted datagram to an inner stream.

Use disk TMPDIR/GOTMPDIR under `.agent/tmp`, `CGO_ENABLED=0`,
`GOTOOLCHAIN=local`, and `GOCACHE` from `go env GOCACHE`. Check
`pgrep -af 'gates.sh|mk/at-'` before startup/live and after startup; wait while
two other integration suites are active. Every invocation is foreground.

```sh
make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E_FOLLOWUP22=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup22Native$'
make -f mk/e2e.mk e2e-down
python3 -B tests/e2e/analyze_native_capture.py <immutable-run-directory>
```

The asymmetric failure deliberately keeps the acceptance command nonzero;
do not relax its 64,064-byte criterion. The remaining required gates and actual
exits are retained in `gates.md`. The attempted second startup was interrupted
when two other AT suites were observed; no second live runner started then.

The DNS third node and installer continue to share origin slot `.24`.
**Keep separate reset stacks**, as documented in follow-up 21. No slot layout
change is needed for this native campaign. Shared `fvpn-pg` is never touched;
all owned Docker objects must be removed after each run.

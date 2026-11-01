# Broader fixture capture (TEST ONLY)

Run `cce1a688-6af5-483c-bd67-2d167c51c04b`, source `98a4b4607cef592a4b00f79aa623c15682cc6ef4`; original scenario status **fail**.

The full raw record, process windows, interface/route readbacks, TCP_INFO and
[endpoint samples](../cce1a688-6af5-483c-bd67-2d167c51c04b/native-matrix-inputs.md) are retained.
Raw pcaps remain private; every exported record below has payload-free headers
and campaign-stable endpoint tokens. There is no separate bottleneck router.
The member underlay is the configured bottleneck; no global MTU/MSS change.

| Row | Selected underlay before / after | Member port | Validated bytes / ms | Member read / write / lines / partial / error | Origin read / write / lines / partial / error |
|---|---|---:|---|---|---|
| broader1500 | ipv4 / ipv4 | 36434 | 64064 / 2 | 64387 / 64108 / 64 / 0 / none | 64108 / 64387 / 64 / 0 / eof |
| broader1280 | ipv4 / ipv4 | 44552 | 10010 / 12011 | 10063 / 64108 / 10 / 0 / timeout | 58604 / 57345 / 58 / 0 / none |

## Completeness and SHA-256 receipts

Snaplen 128 intentionally truncates payload. First/last are packet timestamps;
the scenario separately records process launch/stop/observed-stop bounds.
Captured packet sizes can reflect GSO/GRO; oversized capture lengths alone
do not prove on-wire MTU rejection.

| File / headers | Bytes | SHA-256 | Records / filter / kernel drops | Snap-truncated | First / last Unix UTC |
|---|---:|---|---|---:|---|
| [broader1280-member](cce1a688-6af5-483c-bd67-2d167c51c04b-broader1280-member.headers.jsonl) | 33627 | `8681f3198c9aa0c32514dd4fcac1c9220d6c7ccd5d3b4077b9b2f0db54b4e529` | 252 / 252 / 0 | 169 | 1793463997.432675 / 1793464009.669175 |
| [broader1280-node](cce1a688-6af5-483c-bd67-2d167c51c04b-broader1280-node.headers.jsonl) | 54910 | `8c7e18e318e8d3b4da2f386a0047366d9b40903c26c15927203d21f8699c660d` | 411 / 429 / 0 | 289 | 1793463997.433665 / 1793464009.669169 |
| [broader1280-origin](cce1a688-6af5-483c-bd67-2d167c51c04b-broader1280-origin.headers.jsonl) | 11179 | `25074b2b993939023fc75c4e74d1aa5800338799c27e57880005f2ad85b276e8` | 86 / 86 / 0 | 60 | 1793463997.434241 / 1793464009.485645 |
| [broader1500-member](cce1a688-6af5-483c-bd67-2d167c51c04b-broader1500-member.headers.jsonl) | 36932 | `b97a06d1c5e8b65968697be7352be740d8075892264a04a8f026cb6c83c0eb71` | 271 / 275 / 0 | 229 | 1793463993.462323 / 1793463994.639508 |
| [broader1500-node](cce1a688-6af5-483c-bd67-2d167c51c04b-broader1500-node.headers.jsonl) | 54802 | `fbd32620d3d577684f3bbc5c2eb62ced66c68c7781bf0c11921514253d4d5169` | 407 / 413 / 0 | 334 | 1793463993.462334 / 1793463994.639521 |
| [broader1500-origin](cce1a688-6af5-483c-bd67-2d167c51c04b-broader1500-origin.headers.jsonl) | 15607 | `368970f6cb22aca0149958e15907df92f8830759944144b7fc499b369121ae95` | 121 / 121 / 0 | 87 | 1793463994.637015 / 1793463994.639522 |

All files end on complete records with zero header errors and zero unparsed
records. Decoded counts equal tcpdump captured counts. Differences between
filter and captured counts are retained above, not declared globally lossless.
IPv6 refusal/no-uplink-leak controls are separate from the required IPv4 bulk
completion and do not establish a native ctv6 hit or IPv6 overlay service.

## Failure localization

The bound was **at most three fresh broader campaigns**, stopping at the first
instrumented MTU-1280 bulk failure. Attempt **1** captured it, so attempts 2/3
were not run. This preserves the broader fixture's original three underlay
reconnects followed by dual 1500 and dual 1280 inputs without an extra reconnect.
Both selected paths remain direct IPv4 before/after. The required IPv4 bulk
fails; both independent IPv6 refusal/no-uplink-leak controls pass.

The initial TCP MSS advertisements are **1240 / 1460** in both rows and at all
three capture points. The successful 1500 response uses **64 × 1,006-byte** TCP
segments. The failed connection has member port **44552**, origin initial
sequence **1784138848**, and a first missing range
**[1784148912, 1784150140)** (1,228 bytes). Origin and node tunnel egress retain
that range in the initial coalesced response and in repeated retransmissions;
member tunnel ingress never contains it. The member instead receives later
out-of-order data and repeatedly ACKs **1784148912**, with final SACK blocks
**[1784157064, 1784165008)** and **[1784150140, 1784150924)**. This distinguishes
missing stream data from a fixture that merely miscounts fully received echoes.

| Missing-range retransmission | Origin packet / Unix UTC | Node tunnel packet / Unix UTC | Node outer packet / IP identification |
|---|---|---|---|
| Initial fast retransmit | 67 / 1793463997.439897 | 286 / 1793463997.439900 | 294 / 18625 |
| RTO 1 | 80 / 1793463997.644643 | 325 / 1793463997.644673 | 326 / 18676 |
| RTO 2 | 81 / 1793463998.052639 | 332 / 1793463998.052667 | 333 / 18784 |
| RTO 3 | 82 / 1793463998.900650 | 343 / 1793463998.900689 | 344 / 19082 |
| RTO 4 | 83 / 1793464000.564801 | 352 / 1793464000.564857 | 353 / 20230 |
| RTO 5 | 84 / 1793464003.829653 | 367 / 1793464003.829709 | 368 / 21948 |

The paired outer packets follow those tunnel retransmissions toward the member.
Their captured IPv4 length is **1,340**, DF=false, MF=false, offset=0. Across the
failed row the node emits **19** such outer datagrams; none is present at the
member. All **17** reverse outer packets of length **1,132** have corresponding
member receipts. Upload captures at both ends retain **52 fragment pairs**
(first IP length 1,276 / offset 0 / MF=true, last length 84 / offset 1,256 /
MF=false). Thus the successful upload fragmentation is distinct from the
unfragmented missing reverse traffic. Neither capture contains IPv4
fragmentation-needed or IPv6 PTB. Neighbor discovery is not PTB.

Original interface readbacks identify member **7=fvnb0**, node **6=fvmb0** and
**3=fvup0**. At 1280, member eth0/eth1/fvnb0 and node fvmb0 are 1280; node
underlays/uplink and origin link are 1500. The boundary is downstream of the
node tunnel egress and before the member's captured ingress. Receive-link MTU
rejection is supported as an inference; the exact kernel/virtual-link drop
reason is **not measured**. GSO/GRO and the explicit capture/filter-count
differences above preclude claiming a globally lossless wire capture. No
central/nft/tc or native-client production fix is asserted.

## Endpoint and socket evidence

At timeout the member read **10,063 stream bytes** (3-byte greeting plus ten
1,006-byte echo lines), validated **10,010 payload bytes**, and has no partial
line. It wrote **64,108** bytes and its TCP_INFO shows **64,109 ACKed**, including
the SYN. The origin socket received **64,109**, while its application had read
**58,604** and successfully written **57,345** bytes: **58 complete input lines**,
with the current response write blocked. The final retained origin sample is
at 16:26:50.187550638Z; it is the last observed sample, not a fabricated final
close/EOF. There are **51** periodic origin samples for this connection.

The final origin TCP_INFO has **10,063 ACKed**, **29,472 not sent**, **17 unacked**,
**8 lost**, **9 SACKed**, **11 total retransmissions**, congestion window **1**,
and a nonzero send window **59,392**. The missing data and duplicate ACK/SACKs
precede the application write backpressure. The socket evidence supports a
transport stall rather than a zero-window receiver or a completed transfer
misreported by the read loop. It does not identify the physical drop mechanism.

Per-I/O instrumentation can affect response segmentation. This run shows that
instrumentation does **not always eliminate the failure**; it does not reproduce
the exact historical 12,012- or 11,011-byte boundary. All prior failures and the
two passing follow-up 19 instrumented campaigns remain unchanged.

The origin image lacks `ip` and `ss`; those attempted command errors are retained.
Its interface MTU is read from sysfs and its active socket state from TCP_INFO.
No origin route/ss output is claimed.

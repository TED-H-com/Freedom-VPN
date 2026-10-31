# Fresh native reconnect capture (TEST ONLY)

Run `956c335d-0449-44ac-aa63-544154456397`, source `706d7cf6cec8c00c209dbe029eaa7825921f12ac`. The original 11,011-byte row has no matching capture here; this is a new same-order sequence, not its replacement. The complete campaign is 7/0/0, exit 0.

| Row / underlay selection | Member port | Validated bytes / ms | Member read / write | Origin read / write | Lines / partial |
|---|---:|---|---|---|---|
| matrix4 / ipv4 | 37946 | 64064 / 3 | 64387 / 64108 | 64108 / 64387 | 64 / 0 |
| matrix6 / ipv6 | 37962 | 64064 / 8 | 64387 / 64108 | 64108 / 64387 | 64 / 0 |
| matrixDual / ipv4 | 35902 | 64064 / 1 | 64387 / 64108 | 64108 / 64387 | 64 / 0 |
| matrix1280 / ipv4 | 37850 | 64064 / 1 | 64387 / 64108 | 64108 / 64387 | 64 / 0 |

Each transfer retains the 12-second bound and 64,064-byte validated criterion.
Member TCP_INFO is sampled before close: no timeout/mismatch, no outstanding
bytes, no loss/retransmission, nonzero window. Origin final samples record EOF
after all 64 lines and 64,387 response stream bytes were written and ACKed;
there is no application stall in this control. Each successful connection ends
before the 250-ms periodic sample interval, so these are final pre-close socket
samples rather than invented in-flight time series. A longer connection enables
the same periodic origin samples. Raw endpoint fields and actual MTUs are in
[native-mtu-comparison](../956c335d-0449-44ac-aa63-544154456397/native-mtu-comparison.md).

Dual 1500 and dual 1280 separately prove a reachable controlled IPv6 listener,
member refusal, and no IPv6 TCP/9447 egress on node fvup0 during capture. IPv4
bulk must still complete. These member refusals occur before tunnel ingress;
they do not claim a ctv6 hit or IPv6 overlay service. The kernel drop proof
remains netlab's separate D-123 evidence.

## Completeness and receipts

All 12 files end on complete records, with zero header decode errors and no
unparsed records. All packet counts below equal the stopped tcpdump captured
counts, and all report zero kernel drops. Each member capture reports four
more packets received by the filter than captured; that difference is retained,
not declared a globally lossless capture. The complete TCP application/ACK
control is independently established. Snaplen 128 intentionally truncates
payload. Capture launch/stop/observed-stop times are recorded in the scenario;
all processes stopped before export. First/last below are packet timestamps.

| File / header decode | Bytes | SHA-256 | Records / snap-truncated | First / last packet (Unix UTC) |
|---|---:|---|---|---|
| [matrix1280-member](956c335d-0449-44ac-aa63-544154456397-matrix1280-member.headers.jsonl) | 45324 | `f56a99cde7c1aa4d4c978253edc4bcf670abbbf51625c438a84bfecd6d4080fd` | 343 / 236 | 1793456396.853663 / 1793456396.980555 |
| [matrix1280-node](956c335d-0449-44ac-aa63-544154456397-matrix1280-node.headers.jsonl) | 70454 | `dfe6b780fdef864a4c65fbc51c5be765bf2f1c1cec795fde7245d943efca06bd` | 534 / 386 | 1793456396.978864 / 1793456397.365663 |
| [matrix1280-origin](956c335d-0449-44ac-aa63-544154456397-matrix1280-origin.headers.jsonl) | 19775 | `a451be5455ef3acebce773931a17488d0d79c8eae3c8ad9d0b9a8d3545efc2b9` | 155 / 108 | 1793456396.978926 / 1793456396.980568 |
| [matrix4-member](956c335d-0449-44ac-aa63-544154456397-matrix4-member.headers.jsonl) | 36332 | `720bf898456aeb5b16923c7dd2103ec72819858c0a3069d238d834f92f881e29` | 269 / 222 | 1793456375.596707 / 1793456375.728554 |
| [matrix4-node](956c335d-0449-44ac-aa63-544154456397-matrix4-node.headers.jsonl) | 52870 | `df63e6fb75580dcf49107b07ce357074be25e4539c312f11f780a098d738fd28` | 398 / 316 | 1793456375.724427 / 1793456376.060019 |
| [matrix4-origin](956c335d-0449-44ac-aa63-544154456397-matrix4-origin.headers.jsonl) | 14783 | `d16995822698c69472a91e62adb5fb5184eaca6e397d91a1a5294837068703f1` | 118 / 77 | 1793456375.724500 / 1793456375.728573 |
| [matrix6-member](956c335d-0449-44ac-aa63-544154456397-matrix6-member.headers.jsonl) | 29378 | `a6b200a1c7a3720a58c045d551e9062c6fc3e7bff47f7c65a31518afeb4bb21f` | 213 / 184 | 1793456381.647841 / 1793456382.968885 |
| [matrix6-node](956c335d-0449-44ac-aa63-544154456397-matrix6-node.headers.jsonl) | 40536 | `5d42018f964229ab4507a4ef364d4acb2af51e1a068a94bb9c2573e83a1bfc27` | 297 / 253 | 1793456381.647847 / 1793456383.154081 |
| [matrix6-origin](956c335d-0449-44ac-aa63-544154456397-matrix6-origin.headers.jsonl) | 9999 | `3be18ebbca0d40125ee076473da4ce16da3c54761de5e2ba5cf0a4012e86fdbd` | 77 / 56 | 1793456382.857747 / 1793456382.866023 |
| [matrixDual-member](956c335d-0449-44ac-aa63-544154456397-matrixDual-member.headers.jsonl) | 39420 | `66208e521ce982826266e503559db65cc0cc05c12a1df4fc785a4ff43040e738` | 292 / 241 | 1793456389.827008 / 1793456390.127527 |
| [matrixDual-node](956c335d-0449-44ac-aa63-544154456397-matrixDual-node.headers.jsonl) | 59046 | `79134af308dcdaa45c2fc0d708856e85023b8de3bd5ae05bb81c21280a795a63` | 444 / 354 | 1793456389.948590 / 1793456390.127530 |
| [matrixDual-origin](956c335d-0449-44ac-aa63-544154456397-matrixDual-origin.headers.jsonl) | 17439 | `791981effd1235c919e4cda141dbfb6307d522b8d284b026e99e41c084755448` | 138 / 93 | 1793456389.948656 / 1793456389.950231 |

## Size-dependent boundary still open

At matrix1280 the member eth0/eth1/fvnb0 are 1280, node fvmb0 is 1280,
node underlays/uplink and origin link are 1500. All 64 response TCP segments
are 1,006 bytes at origin, node and member. Their node→member outer IPv4 UDP
packets are 1,132 bytes, unfragmented, and arrive at the member. In the upload
direction, both member and node capture the same 52 fragment pairs: first
IP length 1,276 / MF=true / offset 0 and last length 84 / MF=false / offset
1,256. This proves fragmentation and delivery in that direction for this
control; it does not prove fragmentation of the failed reverse direction.

The [retained failed flow](../capture-decode/README.md) instead retransmits
a missing 1,228-byte TCP segment; associated node→member outer IPv4 packets
are 1,340 bytes, unfragmented, and absent at the member. The different response
segmentation explains why a passing control does not disprove a size-dependent
boundary. Receive-link MTU rejection remains an inference, not a measured
kernel drop reason. The exact old failure has no origin endpoint samples or
capture drop statistics. Its 22,022-byte failure and the original 11,011-byte
members failure stay failed. No product/native MTU or MSS change is justified
by this successful control alone.

The bottleneck is a direct member bridge link, not a separate router. These
captures cover both underlays, member tunnel, node tunnel/uplink and controlled
origin; no nonexistent router hop or unobserved ICMP emission is invented.

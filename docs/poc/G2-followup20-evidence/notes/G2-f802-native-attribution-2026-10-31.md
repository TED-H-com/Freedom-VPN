# F8-02 native attribution — follow-up 19, 2026-10-31

**No single transport cause is established by the available captures.**
Members' original IPv4-selected **11,011/64,064-byte** reconnect transfer stays
failed. Devstack's earlier IPv4-only reconnect returned **22,022/64,064** in
12,010 ms; its later independent control returned **64,064/64,064** in 2 ms.
The latter does not clear the former. No product or NetBird MTU, MSS, nft
policy or dependency change is justified or made here.

Scope: AT09 / EXIT02 / APP01 / NFR01 (§19), MTU policy §8.3 and SEC02.
Sources are the [original members row](../e2e/at/members/history/follow-up-8/AT09.json),
[devstack 18 overview](../e2e/runs/2026-10-31/followup18/README.md),
[2454c833 captures](../e2e/runs/2026-10-31/followup18/2454c833-ae93-48e0-96b0-00dee7cc3117/native-mtu-comparison.json),
[daf790fc captures](../e2e/runs/2026-10-31/followup18/daf790fc-81f7-49a2-8a65-f3ec8faec656/native-mtu-comparison.json)
and [the prior kernel measurement](G2-f802-mtu1280-2026-10-31.md).
Input SHA-256 pins are in [source-pins.json](G2-followup19-evidence/source-pins.json).
No peer IDs, native keys or packet payloads are copied into this report.

## What the packet evidence establishes

For the failed `2454c833` `ipv4_1280` connection, member source port 41478:

| Captured Unix time | Location/direction | TCP evidence |
| --- | --- | --- |
| 1793442773.708862 | member fvnb0 Out | SYN seq 2418164341, MSS 1240, IP length 60 |
| 1793442773.709098 | node fvmb0 In | Same SYN seq/MSS/length |
| 1793442773.709128 | node fvup0 Out | Same SYN after SNAT, MSS still 1240 |
| 1793442773.709156 | node fvup0 In | SYN-ACK seq 10709237, ack 2418164342, MSS 1460 |
| 1793442773.709162 | node fvmb0 Out | Same SYN-ACK/MSS after reverse NAT |
| 1793442773.709198 | member fvnb0 In | Same SYN-ACK, MSS still 1460 |

This proves **no MSS rewriting at the observed node hops** for that handshake.
Member MSS 1240 is consistent with inner IPv4 MTU 1280; it is an endpoint
advertisement, not evidence of an enforcement clamp. Timestamp options are
present. The successful `daf790fc` `ipv4_1280` handshake (port 55090, SYN seq
4196169297, SYN-ACK seq 59143023) has the same 1240/1460 advertisements at all
four node/member directions, at 1793443234.027042–1793443234.027565.

| Candidate cause | Evidence and limit |
| --- | --- |
| MSS clamp | Not observed: unchanged SYN and SYN-ACK MSS through the node in failed and successful rows. Enforcement renders no clamp. No basis for adding one. |
| Suppressed ICMP need-fragmentation | Neither row's retained ICMP summary contains IPv4 type 3/code 4 or an IPv6 PTB. Absence at these observation points does not distinguish no emission, lost error, or suppression upstream. Neighbor/router discovery is not PTB. |
| Interface MTU mismatch | Both rows read back member eth0=1280, eth1=1280, fvnb0=1280; node fvmb0=1280, fvup0=1500, eth0/eth2=1500. Encapsulation adds overhead, so equal inner/outer limits warrant investigation, but outer datagram sizes, fragmentation/reassembly and the actual bottleneck path are not in the published headers. The same layout also completed successfully. |
| TCP retransmission stall | Not provable from these exported summaries: the TCP display filter retains SYN only, omitting ordinary seq/ack/SACK, retransmissions, zero windows, FIN/RST and data lengths. Post-transfer ss sees the failed connection in FIN-WAIT-2 with no retained active socket TCP_INFO; it does not reconstruct the stall. |
| Harness read-count artefact | The client uses bufio.ReadString in a 64-line loop, not one recv. It validates each `ECHO ` line and counts 1001 payload bytes only for complete matching lines. Thus 11,011 and 22,022 mean 11 and 22 complete echoes, excluding prefixes and any partial final line. It collapses read errors and mismatch to bulk_echo_incomplete; without endpoint partial-read/error/write counts, an application-side stall remains possible. This is not evidence that all 64 echoes arrived. |

The origin fixture reads bounded 1024-byte lines and writes each echo; the
1000-character line fits that bound. Static source inspection does not prove
what the origin actually read/wrote during either failure. Aggregate nft byte
counters include retransmitted headers and other traffic; they cannot replace
end-to-end application receipt. Incomplete transfers remain failures.

The prior kernel-only test completed both IPv4 MTU-1280 transfers and proved
related fragmentation-needed traversal in both directions with zero tc drops.
Its bottleneck is the **exit link** at 1280, while its WireGuard interface is
1420. Devstack lowers the **member underlay**, retaining a 1280 native tunnel
and a 1500 exit link. These are different paths, not contradictory measurements.
The lab neither reproduces nor disproves the original native failure.

## D-123 and family interpretation

V1.0 overlay mode is **IPv6Block by design**. IPv6 overlay transfer failure is
expected, not a product defect or an IPv6 MTU/PMTU acceptance row. Both
`meta nfproto ipv6 ... ctv6 drop` rules stay untouched. The prior lab's SYN/PTB
counter proof establishes that expected overlay block.

For this follow-up the IPv6-family failure entries in devstack's matrix are
retained as expected/outside the v1 IPv6 service acceptance scope, never routed
as a request to enable IPv6 forwarding. There is a crucial attribution limit:
`ObserveSelectedNativeExit.chosen_family` is the **underlay candidate family**;
its captured echo SYNs are inner IPv4 (even in rows labelled ipv6). It is not
proof that those native failures incremented ctv6. No IPv6-policy cause is
invented for them, and they are not used to dismiss the IPv4 F8-02 failure.

## Missing capture and D-102 request to devstack

The committed JSON/Markdown contains decoded **SYN/MSS + ICMP summaries**, not
the complete pcap. `followup18_native_test.go` captured TCP/UDP/ICMP but exported
text with `icmp or icmp6 or (tcp port 9446 and tcp[tcpflags] & tcp-syn != 0)`.
Raw pcaps were exported privately into devstack's `.agent/tmp`; none exists in
this worktree. Cross-worktree access is outside the lane contract.

The first requested existing pair is the failed IPv4-only reconnect:

| Run / row / capture | Bytes | Retained SHA-256 |
| --- | --- | --- |
| 2454c833 / ipv4_1280 / member (0) | 35536 | a5b7059d092c1a7352769fa1be874d0820982cd610165aa141e20f773e59f9ec |
| 2454c833 / ipv4_1280 / node (1) | 54892 | 17f0652be93b2a94844a056798df221fcd0c11673f98287f8b84381cfc8cd953 |
| daf790fc / ipv4_1280 / member control (0) | 44388 | 2f66fa0e576f68e19d74796c2c30f44327e72c108660de63cf00ede08588efdc |
| daf790fc / ipv4_1280 / node control (1) | 67414 | ca9390d8bfa02febe5b2e9403b60d0f2263a0baacafe435b582eb27f23aaa71d |

**Request:** devstack should decode these verified files locally and export
payload-free TCP timestamp/interface/direction/seq/ack/SACK/window/flags/IP and
TCP length headers for the full failed flow plus its control, and outer UDP/IP
length/DF/fragment-offset/identification headers at both tunnel endpoints.
Include the complete capture start/stop/statistics; the early run lacks the
later run's stats/flush proof. The later IPv4 control reports 334 member and
509 node packets, zero kernel drops, but its SYN-only text cannot be compared
to those total counts. Do not expose WireGuard payloads, keys or peer IDs.

The **original 11,011-byte members reconnect** has no supplied matching pcap.
If the retained failed pair is incomplete or does not establish causality,
capture that exact mutation/reconnect sequence with synchronized member tunnel
and underlay, node tunnel and uplink, controlled origin and bottleneck-router
headers. Include actual interface MTUs and capture completeness at each hop,
plus endpoint cumulative read/write bytes, completed lines, partial final-line
length, error category and in-flight TCP_INFO before socket close. These
separate missing data/ACKs or PMTU signals from an application stall. Keep the
12 s deadline, validated 64,064-byte criterion and each failed attempt.

Devstack owns `tests/e2e/followup18_native_test.go`, its two fixture endpoints
and its `requests.md`; no such file is edited here. No specific node/enforce
transport fix is supported. This request supplies the exact missing evidence
rather than selecting a global MTU or a speculative MSS workaround.

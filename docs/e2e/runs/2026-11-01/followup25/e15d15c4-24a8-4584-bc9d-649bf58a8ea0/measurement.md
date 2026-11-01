# e15d15c4-24a8-4584-bc9d-649bf58a8ea0 — kernel / ipv4 (TEST ONLY)

Original result: **pass**. [Full immutable summary](first-selection-summary.json), [verified capture/trace analysis](kernel-path-analysis.json), [original measurement](initial-selection.json).

| Measurement | Observation |
|---|---|
| SYNs: member tunnel / all node capture records | 1 / 2 |
| First origin packet from first SYN, bounded ms | 205.034107–205.038215 |
| Original HTTP completion / timeout | 209 ms; available=True |
| Independent origin HTTP receipts | 1 |
| Forwarding generation; serving; upload/download bytes, before | 5; serving=True; 0/0 |
| Forwarding generation; serving; upload/download bytes, after | 5; serving=True; 2345/2464 |
| Node `::1`-source egress response records | 0 |

Observed `uses_kernel_interface`: member **[True]**, node **[True]**. Every selected peer status read valid/present: {'member': True, 'node': True}.

| Response reference | Peer | Total type-4 records | Before | After | Bounds overlap |
|---|---|---:|---:|---:|---:|
| first node egress type-2 | member | 22 | 0 | 22 | 0 |
| first node egress type-2 | node | 22 | 0 | 22 | 0 |
| first node `::1` egress type-2 | member | 22 | unknown | unknown | unknown |
| first node `::1` egress type-2 | node | 22 | unknown | unknown | unknown |

Egress reference means SLL2 OUTGOING (packet type 4). Loopback HOST records are still retained in the full analysis. Type-4 counts include multiple proxy hops and are structural records, not authenticated or unique messages. With no reference response, ordering is unknown. Origin time is first captured inner packet, separately from the independent HTTP receipt.

| Peer | Changed sample | WireGuard endpoint | Matched selected peer | Latest handshake (Unix seconds) | Native remote |
|---|---|---|---|---|---|
| member | before-selection | addr-32bcd336456e1b8c:57481 (ipv4) | True | 0 | unavailable |
| member | first-transfer-0 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 0 | unavailable |
| member | after-first-transfer | addr-459868bfedca20a7:51820 (ipv4) | True | 1793528523 | addr-459868bfedca20a7:51820 (ipv4) |
| node | before-selection | addr-32bcd336456e1b8c:54869 (ipv4) | True | 0 | unavailable |
| node | after-first-transfer | addr-129fe9318e066262:51820 (ipv4) | True | 1793528523 | addr-129fe9318e066262:51820 (ipv4) |

Changes above are sampled transitions. The original record retains every sample and its command/clock bounds; no intervening endpoint history is invented.

| Namespace | Source | Counter | First sample | Final sample |
|---|---|---|---:|---:|
| member | snmp4 | IpInAddrErrors | 0 | 0 |
| member | snmp4 | IpInDiscards | 0 | 0 |
| member | snmp4 | IpInHdrErrors | 0 | 0 |
| member | snmp4 | IpInUnknownProtos | 0 | 0 |
| member | snmp4 | UdpInCsumErrors | 0 | 0 |
| member | snmp4 | UdpInErrors | 0 | 0 |
| member | snmp4 | UdpNoPorts | 1 | 1 |
| member | snmp6 | Ip6InAddrErrors | 0 | 0 |
| member | snmp6 | Ip6InDiscards | 20 | 21 |
| member | snmp6 | Ip6InHdrErrors | 0 | 0 |
| member | snmp6 | Udp6InCsumErrors | 0 | 0 |
| member | snmp6 | Udp6InErrors | 0 | 0 |
| member | snmp6 | Udp6NoPorts | 0 | 0 |
| member | nstat4 | IpExtInNoRoutes | 0 | 0 |
| member | nstat4 | IpExtInTruncatedPkts | 0 | 0 |
| member | nstat4 | IpInAddrErrors | 0 | 0 |
| member | nstat4 | IpInDiscards | 0 | 0 |
| member | nstat4 | IpInHdrErrors | 0 | 0 |
| member | nstat4 | IpInUnknownProtos | 0 | 0 |
| member | nstat4 | UdpInCsumErrors | 0 | 0 |
| member | nstat4 | UdpInErrors | 0 | 0 |
| member | nstat4 | UdpNoPorts | 1 | 1 |
| member | nstat_hdr_errors | Ip6InHdrErrors | 0 | 0 |
| node | snmp4 | IpInAddrErrors | 0 | 0 |
| node | snmp4 | IpInDiscards | 0 | 0 |
| node | snmp4 | IpInHdrErrors | 0 | 0 |
| node | snmp4 | IpInUnknownProtos | 0 | 0 |
| node | snmp4 | UdpInCsumErrors | 0 | 0 |
| node | snmp4 | UdpInErrors | 0 | 0 |
| node | snmp4 | UdpNoPorts | 1 | 2 |
| node | snmp6 | Ip6InAddrErrors | 0 | 0 |
| node | snmp6 | Ip6InDiscards | 44 | 44 |
| node | snmp6 | Ip6InHdrErrors | 0 | 0 |
| node | snmp6 | Udp6InCsumErrors | 0 | 0 |
| node | snmp6 | Udp6InErrors | 0 | 0 |
| node | snmp6 | Udp6NoPorts | 0 | 0 |
| node | nstat4 | IpExtInNoRoutes | 0 | 0 |
| node | nstat4 | IpExtInTruncatedPkts | 0 | 0 |
| node | nstat4 | IpInAddrErrors | 0 | 0 |
| node | nstat4 | IpInDiscards | 0 | 0 |
| node | nstat4 | IpInHdrErrors | 0 | 0 |
| node | nstat4 | IpInUnknownProtos | 0 | 0 |
| node | nstat4 | UdpInCsumErrors | 0 | 0 |
| node | nstat4 | UdpInErrors | 0 | 0 |
| node | nstat4 | UdpNoPorts | 1 | 2 |
| node | nstat_hdr_errors | Ip6InHdrErrors | 0 | 0 |

Counter read success and all original namespace counters are retained in the JSON. Missing values are unavailable, never zero.

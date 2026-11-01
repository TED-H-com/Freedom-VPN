# 48dbf84e-00e5-4250-84d2-0a36b074ed1f — kernel / ipv4 (TEST ONLY)

Original result: **fail**. [Full immutable summary](first-selection-summary.json), [verified capture/trace analysis](kernel-path-analysis.json), [original measurement](initial-selection.json).

| Measurement | Observation |
|---|---|
| SYNs: member tunnel / all node capture records | 5 / 5 |
| First origin packet from first SYN, bounded ms | absent |
| Original HTTP completion / timeout | 5004 ms; available=False |
| Independent origin HTTP receipts | 0 |
| Forwarding generation; serving; upload/download bytes, before | 0; serving=False; 0/0 |
| Forwarding generation; serving; upload/download bytes, after | 0; serving=False; 0/0 |
| Node `::1`-source egress response records | 0 |

Observed `uses_kernel_interface`: member **[True]**, node **[True]**. Every selected peer status read valid/present: {'member': True, 'node': True}.

| Response reference | Peer | Total type-4 records | Before | After | Bounds overlap |
|---|---|---:|---:|---:|---:|
| first node egress type-2 | member | 6 | unknown | unknown | unknown |
| first node egress type-2 | node | 6 | unknown | unknown | unknown |
| first node `::1` egress type-2 | member | 6 | unknown | unknown | unknown |
| first node `::1` egress type-2 | node | 6 | unknown | unknown | unknown |

Egress reference means SLL2 OUTGOING (packet type 4). Loopback HOST records are still retained in the full analysis. Type-4 counts include multiple proxy hops and are structural records, not authenticated or unique messages. With no reference response, ordering is unknown. Origin time is first captured inner packet, separately from the independent HTTP receipt.

| Peer | Changed sample | WireGuard endpoint | Matched selected peer | Latest handshake (Unix seconds) | Native remote |
|---|---|---|---|---|---|
| member | before-selection | addr-32bcd336456e1b8c:47246 (ipv4) | True | 0 | unavailable |
| member | first-transfer-0 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 1793528109 | unavailable |
| member | first-transfer-5 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 1793528109 | addr-459868bfedca20a7:51820 (ipv4) |
| member | after-passive-tail | addr-459868bfedca20a7:51820 (ipv4) | True | 1793528109 | addr-459868bfedca20a7:51820 (ipv4) |
| node | before-selection | addr-32bcd336456e1b8c:59805 (ipv4) | True | 0 | unavailable |
| node | first-transfer-1 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 1793528109 | unavailable |
| node | first-transfer-4 | addr-129fe9318e066262:51820 (ipv4) | True | 1793528109 | unavailable |
| node | first-transfer-5 | addr-129fe9318e066262:51820 (ipv4) | True | 1793528109 | addr-129fe9318e066262:51820 (ipv4) |

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
| member | snmp6 | Ip6InDiscards | 21 | 22 |
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
| node | snmp4 | UdpNoPorts | 1 | 1 |
| node | snmp6 | Ip6InAddrErrors | 0 | 0 |
| node | snmp6 | Ip6InDiscards | 43 | 45 |
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
| node | nstat4 | UdpNoPorts | 1 | 1 |
| node | nstat_hdr_errors | Ip6InHdrErrors | 0 | 0 |

Counter read success and all original namespace counters are retained in the JSON. Missing values are unavailable, never zero.

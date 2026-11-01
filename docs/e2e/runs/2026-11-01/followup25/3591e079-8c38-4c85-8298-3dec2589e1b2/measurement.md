# 3591e079-8c38-4c85-8298-3dec2589e1b2 — userspace / ipv6 (TEST ONLY)

Original result: **pass**. [Full immutable summary](first-selection-summary.json), [verified capture/trace analysis](kernel-path-analysis.json), [original measurement](initial-selection.json).

| Measurement | Observation |
|---|---|
| SYNs: member tunnel / all node capture records | 1 / 2 |
| First origin packet from first SYN, bounded ms | 103.774205–103.777834 |
| Original HTTP completion / timeout | 106 ms; available=True |
| Independent origin HTTP receipts | 1 |
| Forwarding generation; serving; upload/download bytes, before | 5; serving=True; 0/0 |
| Forwarding generation; serving; upload/download bytes, after | 5; serving=True; 2357/2462 |
| Node `::1`-source egress response records | 0 |

Observed `uses_kernel_interface`: member **[True]**, node **[False]**. Every selected peer status read valid/present: {'member': True, 'node': True}.

| Response reference | Peer | Total type-4 records | Before | After | Bounds overlap |
|---|---|---:|---:|---:|---:|
| first node egress type-2 | member | 20 | 20 | 0 | 0 |
| first node egress type-2 | node | 20 | 20 | 0 | 0 |
| first node `::1` egress type-2 | member | 20 | unknown | unknown | unknown |
| first node `::1` egress type-2 | node | 20 | unknown | unknown | unknown |

Egress reference means SLL2 OUTGOING (packet type 4). Loopback HOST records are still retained in the full analysis. Type-4 counts include multiple proxy hops and are structural records, not authenticated or unique messages. With no reference response, ordering is unknown. Origin time is first captured inner packet, separately from the independent HTTP receipt.

| Peer | Changed sample | WireGuard endpoint | Matched selected peer | Latest handshake (Unix seconds) | Native remote |
|---|---|---|---|---|---|
| member | before-selection | addr-32bcd336456e1b8c:55960 (ipv4) | True | 0 | unavailable |
| member | first-transfer-0 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 0 | unavailable |
| member | after-first-transfer | addr-837a7935ba5539b1:51820 (ipv6) | True | 1793532122 | unavailable |
| member | after-passive-tail | addr-837a7935ba5539b1:51820 (ipv6) | True | 1793532122 | addr-837a7935ba5539b1:51820 (ipv6) |
| node | before-selection | unavailable | False | None | unavailable |
| node | after-first-transfer | unavailable | False | None | unavailable |
| node | after-passive-tail | unavailable | False | None | addr-5f910114cdbd19f5:51820 (ipv6) |

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
| node | snmp6 | Ip6InDiscards | 44 | 46 |
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

## Native daemon handshake telemetry

Separate cached native-status field; it does not replace a failed WireGuard CLI read. Earlier missing samples stay missing.

| Peer | Changed sample | Daemon-reported last handshake (UTC) |
|---|---|---|
| member | after-first-transfer | 2026-11-01T11:22:02.034234597Z |
| member | after-passive-tail | 2026-11-01T11:22:02.034234597Z |
| node | after-first-transfer | 2026-11-01T11:22:02.033917159Z |
| node | after-passive-tail | 2026-11-01T11:22:02.033917159Z |

# 2cf62be6-7684-449d-8f8f-814da0f64cd6 — userspace / ipv6 (TEST ONLY)

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

Observed `uses_kernel_interface`: member **[True]**, node **[False]**. Every selected peer status read valid/present: {'member': True, 'node': True}.

| Response reference | Peer | Total type-4 records | Before | After | Bounds overlap |
|---|---|---:|---:|---:|---:|
| first node egress type-2 | member | 6 | unknown | unknown | unknown |
| first node egress type-2 | node | 0 | unknown | unknown | unknown |
| first node `::1` egress type-2 | member | 6 | unknown | unknown | unknown |
| first node `::1` egress type-2 | node | 0 | unknown | unknown | unknown |

Egress reference means SLL2 OUTGOING (packet type 4). Loopback HOST records are still retained in the full analysis. Type-4 counts include multiple proxy hops and are structural records, not authenticated or unique messages. With no reference response, ordering is unknown. Origin time is first captured inner packet, separately from the independent HTTP receipt.

| Peer | Changed sample | WireGuard endpoint | Matched selected peer | Latest handshake (Unix seconds) | Native remote |
|---|---|---|---|---|---|
| member | before-selection | addr-32bcd336456e1b8c:52588 (ipv4) | True | 0 | unavailable |
| member | first-transfer-0 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 1793531335 | unavailable |
| member | first-transfer-1 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 1793531335 | unavailable |
| member | first-transfer-5 | addr-060f0ef186d1db5b:3128 (ipv4) | True | 1793531335 | addr-837a7935ba5539b1:51820 (ipv6) |
| member | after-passive-tail | addr-837a7935ba5539b1:51820 (ipv6) | True | 1793531335 | addr-837a7935ba5539b1:51820 (ipv6) |
| node | before-selection | unavailable | False | None | unavailable |
| node | first-transfer-1 | unavailable | False | None | unavailable |
| node | first-transfer-5 | unavailable | False | None | addr-5f910114cdbd19f5:51820 (ipv6) |

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
| member | snmp6 | Ip6InDiscards | 21 | 21 |
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
| node | snmp6 | Ip6InDiscards | 43 | 43 |
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
| member | first-transfer-1 | 2026-11-01T11:08:55.63321328Z |
| member | first-transfer-5 | 2026-11-01T11:08:55.63321328Z |
| member | after-passive-tail | 2026-11-01T11:08:55.63321328Z |
| node | first-transfer-1 | 2026-11-01T11:08:55.63303838Z |
| node | first-transfer-5 | 2026-11-01T11:08:55.63303838Z |

# Native transport and DNS evidence — Follow-up 9

Run `11e305fe-6360-4dbd-95b1-67274aa4988c`, harness `49fbb48e013b824c4ce107d3cb551db79b915b42`, product baseline `7e044c71` (agent-core 25 #312, devstack 18 #313, dns-publish 8 #309 and netlab 18), assigned `w4-members / 24000`.

All family labels below refer to underlay availability. The payload destination is IPv4. D-123 permits an IPv6Block/no-leak assertion and IPv4 completion at MTU 1280; it supplies no IPv6 end-to-end MTU/PMTU row. Independent no-leak and IPv4 ICMP traversal evidence: [netlab follow-up 18](../../../poc/G2-f802-mtu1280-2026-10-31.md).

| Native row | Selected underlay before → after | Bytes / requested | Milliseconds | Upload / download delta | Result |
|---|---|---:|---:|---:|---|
| native ipv4 MTU 1500 transfer | ipv4 → ipv4 | 64064 / 64064 | 2 | 66612 / 68035 | pass |
| native ipv6 MTU 1500 transfer | ipv6 → ipv6 | 64064 / 64064 | 1 | 66924 / 68087 | pass |
| native dual MTU 1500 transfer | ipv6 → ipv6 | 64064 / 64064 | 1 | 66196 / 68191 | pass |
| native dual MTU 1280 transfer | ipv4 → ipv4 | 64064 / 64064 | 1 | 67652 / 68139 | pass |

The original F8-02 failure (11,011/64,064 bytes in 12,011 ms) remains archived and unattributed. Devstack 18 captured failures with either underlay family, with and without reconnect. Netlab completes IPv4 at 1280; neither owner establishes whether the original native partial transfer is a NetBird defect or fixture echo/read behavior. See [dated requests](requests.md).

[Complete restart capture](AT09-restart.json) retains all 90 seconds, boot/final nft/tc, heartbeats, central observations and redacted agent log. Historical rejections are retained; no topology/state/generation repair is performed.

Restart: desired 30 → 30, applied 30 → 30; restart `2026-10-31T12:06:31.816656225Z`, fresh serving `2026-10-31T12:06:47.545258Z` (15.729 s), first counted byte `2026-10-31T12:06:53.190469871Z` (21.374 s). Composite recovery/lease assertion: **fail** because it includes the D-067 mixed-state predicate. The generation, new populated lease, fresh serving and counted-traffic evidence close F8-01; F9-01 preserves the separate reporting-contract failure.

| AT20 native query | Backend row / scope | RCODE | Answer types | SOA / TTL | Result |
|---|---|---:|---|---|---|
| RR case 01 TCP=false; rr QTYPE 1 | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 02 TCP=false; rr QTYPE 28 | upstream control (not synthetic filtering);  | 0 | [28] | False / 0 | pass |
| RR case 03 TCP=false; alias QTYPE 5 | upstream control (not synthetic filtering);  | 0 | [5] | False / 0 | pass |
| RR case 04 TCP=false; service QTYPE 64 | upstream control (not synthetic filtering);  | 0 | [64] | False / 0 | pass |
| RR case 05 TCP=false; service QTYPE 65 | upstream control (not synthetic filtering);  | 0 | [65] | False / 0 | pass |
| RR case 06 TCP=false; feed-01 QTYPE 1 | F8-A / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 07 TCP=false; feed-01 QTYPE 28 | F8-AAAA / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 08 TCP=false; blocked-cname QTYPE 5 | F8-CNAME / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 09 TCP=false; feed-01 QTYPE 64 | F8-SVCB / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 10 TCP=false; feed-01 QTYPE 65 | F8-HTTPS / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 11 TCP=false; blocked-alias QTYPE 1 | F8-A / F8-TTL5; C | 3 | [] | True / 5 | pass |
| RR case 12 TCP=false; sub QTYPE 1 | F8-A / F8-TTL5; W | 3 | [] | True / 5 | pass |
| RR case 13 TCP=false; notfeed-01 QTYPE 1 | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 14 TCP=false; absent QTYPE 1 | upstream control (not synthetic filtering);  | 3 | [] | True / 5 | pass |
| RR case 01 TCP=true; rr QTYPE 1 | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 02 TCP=true; rr QTYPE 28 | upstream control (not synthetic filtering);  | 0 | [28] | False / 0 | pass |
| RR case 03 TCP=true; alias QTYPE 5 | upstream control (not synthetic filtering);  | 0 | [5] | False / 0 | pass |
| RR case 04 TCP=true; service QTYPE 64 | upstream control (not synthetic filtering);  | 0 | [64] | False / 0 | pass |
| RR case 05 TCP=true; service QTYPE 65 | upstream control (not synthetic filtering);  | 0 | [65] | False / 0 | pass |
| RR case 06 TCP=true; feed-01 QTYPE 1 | F8-A / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 07 TCP=true; feed-01 QTYPE 28 | F8-AAAA / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 08 TCP=true; blocked-cname QTYPE 5 | F8-CNAME / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 09 TCP=true; feed-01 QTYPE 64 | F8-SVCB / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 10 TCP=true; feed-01 QTYPE 65 | F8-HTTPS / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 11 TCP=true; blocked-alias QTYPE 1 | F8-A / F8-TTL5; C | 3 | [] | True / 5 | pass |
| RR case 12 TCP=true; sub QTYPE 1 | F8-A / F8-TTL5; W | 3 | [] | True / 5 | pass |
| RR case 13 TCP=true; notfeed-01 QTYPE 1 | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 14 TCP=true; absent QTYPE 1 | upstream control (not synthetic filtering);  | 3 | [] | True / 4 | pass |

RCODE 0 with zero answers and SOA is NODATA, never NXDOMAIN. Synthetic filtering uses the five-second profile; [BACKEND.md §2.1.1–§2.1.4](../../../dns/BACKEND.md) defines each table row. The positive and absent-upstream controls are separately labeled; cache phase/TTL observations remain in AT20.json.

| Case / query | Backend row | RCODE | Answer types | SOA / TTL | Result |
|---|---|---:|---|---|---|
| AT21 fallback block policy | F8-A / F8-TTL5 | 3 | [] | True / 5 | pass |
| AT22 privacy sentinel query | F8-A / F8-TTL5 | 3 | [] | True / 5 | pass |
| AT22 node last-good block response | F8-A / F8-TTL5 | 3 | [] | True / 5 | pass |

All platform/QUIC unavailable selectors remain blocked. [OPERATORS.md](OPERATORS.md) retains the required real-device procedures. Native distribution remains TEST ONLY. DNS wording closure does not close IDNA, platform, production distribution, URL privacy or retention gaps.

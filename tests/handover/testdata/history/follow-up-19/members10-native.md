# Native transport and DNS evidence — Follow-up 10

Run `2be2d32f-bfd9-46f5-bf29-afaa6617ae94`, harness `731be5a7af7657660d96c240695747a23a3f5c96`, product baseline `d3e9a0f1` (agent-core 27 #325, wire-nodes 32 #321, netlab 19 #318; devstack 18 and dns-publish 8 retained), assigned `w4-members / 24000`.

All family labels below refer to underlay availability. The payload destination is IPv4. D-123 permits an IPv6Block/no-leak assertion and IPv4 completion at MTU 1280; it supplies no IPv6 end-to-end MTU/PMTU row. Independent no-leak and IPv4 ICMP traversal evidence: [netlab follow-up 18](../../../poc/G2-f802-mtu1280-2026-10-31.md).

| Native row | Selected underlay before → after | Bytes / requested | Milliseconds | Upload / download delta | Result |
|---|---|---:|---:|---:|---|
| native ipv4 MTU 1500 transfer | ipv4 → ipv4 | 64064 / 64064 | 1 | 67392 / 68191 | pass |
| native ipv6 MTU 1500 transfer | ipv6 → ipv6 | 64064 / 64064 | 1 | 66612 / 68191 | pass |
| native dual MTU 1500 transfer | ipv4 → ipv4 | 64064 / 64064 | 1 | 66924 / 68087 | pass |
| native dual MTU 1280 transfer | ipv4 → ipv4 | 64064 / 64064 | 1 | 67236 / 68243 | pass |

The original F8-02 failure (11,011/64,064 bytes in 12,011 ms) remains archived and unattributed. Devstack 18 captured failures with either underlay family, with and without reconnect. [Netlab 19](../../../poc/G2-f802-native-attribution-2026-10-31.md) establishes no single transport cause. Devstack 19 owns the exact-sequence header decode/capture and endpoint read/write counts. A later MTU-1280 pass is no fix proof. See [dated requests](requests.md).

[Complete restart capture](AT09-restart.json) retains all 90 seconds, boot/final nft/tc, heartbeats, central observations and redacted agent log. Historical rejections are retained; no topology/state/generation repair is performed.

Restart: desired 29 → 29, applied 29 → 29; restart `2026-10-31T14:00:14.473094656Z`, fresh serving `2026-10-31T14:00:30.224689Z` (15.752 s), first counted byte `2026-10-31T14:00:35.830146766Z` (21.357 s). Recovery assertion: **pass**.

**F10-04:** the requested exact generation-30 reproduction was not obtained;
the unchanged full sequence produced 29. The same-generation recovery and D-126
composition pass do not waive this qualification gap or full AT09 blockers.
The mixed report at receiver time `14:00:15.201584Z` retains both reasons,
`TC_FOREIGN_FILTER` on all three dependent checks and the separate AWAITING
lease error. All three provisional reports pass; none claims `lease_expired`.

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

**F10-03 cache failure:** the initial positive answer has TTL 10, but the
before-TTL query at `13:52:12.96280019Z` already returns the changed private
address class. The negative control remains NXDOMAIN/SOA TTL 1; after-TTL
controls pass. [Capture](F10-03-cache.json) retains the authority-only change and
query results. The table above remains unchanged in its expected RCODE/SOA/TTL
semantics; no failed cache row is promoted by passing RR filtering rows.

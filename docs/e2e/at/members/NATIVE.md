# Native transport and DNS evidence — Follow-up 8

Run `69f92f49-73dc-4779-8101-dfb6a802a96b`, harness `d79accd9c693aed83713072c925b3b5005b5684c`, product baseline `6848add` (#284), assigned `w4-members / 24000`. The tables below are derived from the current AT JSON, not historical fixture runs.

| AT09 row | Bytes | Selected family before → after | Upload delta | Download delta | Status |
|---|---:|---|---:|---:|---|
| native ipv4 MTU 1500 transfer | 64064 | ipv4 → ipv4 | 65936 | 68139 | pass |
| native ipv6 MTU 1500 transfer | 64064 | ipv6 → ipv6 | 66820 | 68035 | pass |
| native dual MTU 1500 transfer | 64064 | ipv6 → ipv6 | 66924 | 68191 | pass |
| native dual MTU 1280 transfer | 11011 | ipv4 → ipv4 | 66296 | 45963 | fail |

These are underlay-family and finite TCP/MTU measurements. They do not establish IPv6 overlay forwarding, QUIC or a complete PMTU campaign.

| Unavailable selector | Status | Fixture declaration |
|---|---|---|
| native linux quic | blocked | requires a pinned QUIC client/server at the controlled origin; UDP echo is not QUIC |
| native android tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |
| native ios tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |
| native windows tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |
| native macos tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |

Issued resolver list: `["192.0.4.53", "192.0.4.54"]`; observed native endpoints: `["192.0.4.53:53", "192.0.4.54:53"]`. Native DNS enabled: True; system uses native stub: True; identical applied document: True. Distribution into the peer-restricted native group is TEST ONLY.

The shared 28-row RR matrix includes positive and label-boundary controls. Direct blocked CNAME/SVCB return empty NODATA; other blocked query types return NXDOMAIN. SOA presence and TTL are recorded independently. DNS owners still need to confirm the wording.

| Transport | Fixture case | Type | RCODE / interpretation | Answer types | SOA present / TTL | Status |
|---|---|---|---|---|---|---|
| UDP | rr | A | 0 / NOERROR | [1] | False / 0 | pass |
| UDP | rr | AAAA | 0 / NOERROR | [28] | False / 0 | pass |
| UDP | alias | CNAME | 0 / NOERROR | [5] | False / 0 | pass |
| UDP | service | SVCB | 0 / NOERROR | [64] | False / 0 | pass |
| UDP | service | HTTPS | 0 / NOERROR | [65] | False / 0 | pass |
| UDP | feed-01 | A | 3 / NXDOMAIN | [] | True / 5 | pass |
| UDP | feed-01 | AAAA | 3 / NXDOMAIN | [] | True / 5 | pass |
| UDP | blocked-cname | CNAME | 0 / NODATA | [] | True / 5 | pass |
| UDP | feed-01 | SVCB | 0 / NODATA | [] | True / 5 | pass |
| UDP | feed-01 | HTTPS | 3 / NXDOMAIN | [] | True / 5 | pass |
| UDP | blocked-alias | A | 3 / NXDOMAIN | [] | True / 5 | pass |
| UDP | sub | A | 3 / NXDOMAIN | [] | True / 5 | pass |
| UDP | notfeed-01 | A | 0 / NOERROR | [1] | False / 0 | pass |
| UDP | absent | A | 3 / NXDOMAIN | [] | True / 5 | pass |
| TCP | rr | A | 0 / NOERROR | [1] | False / 0 | pass |
| TCP | rr | AAAA | 0 / NOERROR | [28] | False / 0 | pass |
| TCP | alias | CNAME | 0 / NOERROR | [5] | False / 0 | pass |
| TCP | service | SVCB | 0 / NOERROR | [64] | False / 0 | pass |
| TCP | service | HTTPS | 0 / NOERROR | [65] | False / 0 | pass |
| TCP | feed-01 | A | 3 / NXDOMAIN | [] | True / 5 | pass |
| TCP | feed-01 | AAAA | 3 / NXDOMAIN | [] | True / 5 | pass |
| TCP | blocked-cname | CNAME | 0 / NODATA | [] | True / 5 | pass |
| TCP | feed-01 | SVCB | 0 / NODATA | [] | True / 5 | pass |
| TCP | feed-01 | HTTPS | 3 / NXDOMAIN | [] | True / 5 | pass |
| TCP | blocked-alias | A | 3 / NXDOMAIN | [] | True / 5 | pass |
| TCP | sub | A | 3 / NXDOMAIN | [] | True / 5 | pass |
| TCP | notfeed-01 | A | 0 / NOERROR | [1] | False / 0 | pass |
| TCP | absent | A | 3 / NXDOMAIN | [] | True / 4 | pass |

| Cache observation | Positive answer TTLs / class | Negative RCODE / SOA TTL (or new answer) | Status |
|---|---|---|---|
| cache initial | [10] / ['other_ipv4'] | 3 / 4; answers [] | pass |
| positive/negative cache before TTL | [9] / ['other_ipv4'] | 3 / 3; answers [] | pass |
| positive/negative cache after TTL | [10] / ['private_ipv4'] | 0 / 0; answers [1] | pass |

| Rebinding type | Private DNS answer classes | Node listener connected | Member connected / available / error | Status |
|---|---|---|---|---|
| 1 | ['private_ipv4'] | True | False / True /  | pass |
| 28 | ['private_ipv6'] | True | False / False / connect_unavailable | pass |

The IPv6 member connect attempt reports `connect_unavailable`; the node-side IPv6 listener works. This is a failed member connection, not evidence of an established IPv6 overlay path.

| Native OS lookup | Milliseconds | Available | Status |
|---|---:|---|---|
| automatic secondary fallback | 2503 | True | pass |
| both down bounded failure | 12001 | False | pass |
| native DNS recovery | 41 | True | pass |

Full cases retain the blockers and failures in [summary.md](summary.md). Exact observations are in [AT09.json](AT09.json), [AT20.json](AT20.json), [AT21.json](AT21.json) and [AT22.json](AT22.json).

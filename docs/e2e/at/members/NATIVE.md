# Native transport and DNS evidence — Follow-up 11

Run `7edb263f-1f6e-4f94-9e8e-26099f96a699`, harness `33abb2fd2c2fc95826c6e69eb434045522d6f7ef`, product baseline `96ec0630e4834e13a3a14d65553ba8108bb5da55` (devstack 20 #343, agent-core 28 #344, netlab 20 #336; dns-publish 8 #309). Assigned `w4-members / 24000`, two resolvers. The full mutation order is unchanged.

Desired/applied **30 → 30 / 30 → 30**; serving in **6.032389860 s**, first counted byte in **11.399224550 s**. Recovery row: **pass**.

[Complete restart capture](AT09-restart.json) retains every original heartbeat, observation, boot/final nft/tc read and redacted agent log. Agent-core 28 adds same-boot adoption and bounded serving verification; see [its hand-off](../../../node/followup28-iso-f6-03-2026-10-31.md). A changed sequence or timing is recorded as measured, not attributed from ownership alone.

| Native row | Selected underlay before → after | Bytes / requested | Milliseconds | Upload / download delta | Result |
|---|---|---:|---:|---:|---|
| native ipv4 MTU 1500 transfer | ipv4 → ipv4 | 64064 / 64064 | 2 | 65988 / 66891 | pass |
| native ipv6 MTU 1500 transfer | ipv6 → ipv6 | 64064 / 64064 | 1 | 66820 / 68139 | pass |
| native dual MTU 1500 transfer | ipv4 → ipv4 | 64064 / 64064 | 1 | 67028 / 68087 | pass |
| native dual MTU 1280 transfer | ipv6 → ipv6 | 13013 / 64064 | 12010 | 66216 / 38241 | fail |

D-123: family labels are underlay availability; all bulk payloads are IPv4. IPv6Block is unchanged. F8-02's historical failure stays failed. [Devstack 20's capture-cce1a688](../../runs/2026-10-31/followup20/capture-cce1a688/README.md) has 10,010 / 64,064 bytes in 12,011 ms at MTU 1280, attempt 1 of 3: 19 reverse outer IPv4 datagrams of length 1,340 absent at the member; all 17 of length 1,132 arrive; no PTB. Netlab 21 owns interpretation. A passing members bulk row is no fix proof.

| Unavailable selector | Status | Exact resource reason |
|---|---|---|
| native linux quic | blocked | requires a pinned QUIC client/server at the controlled origin; UDP echo is not QUIC |
| native android tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |
| native ios tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |
| native windows tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |
| native macos tcp | blocked | requires the named real platform and operator client; no emulation by the Linux container |

| AT20 native query | Backend row / scope | RCODE | Answer types | SOA / TTL | Result |
|---|---|---:|---|---|---|
| RR case 01 TCP=false | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 02 TCP=false | upstream control (not synthetic filtering);  | 0 | [28] | False / 0 | pass |
| RR case 03 TCP=false | upstream control (not synthetic filtering);  | 0 | [5] | False / 0 | pass |
| RR case 04 TCP=false | upstream control (not synthetic filtering);  | 0 | [64] | False / 0 | pass |
| RR case 05 TCP=false | upstream control (not synthetic filtering);  | 0 | [65] | False / 0 | pass |
| RR case 06 TCP=false | F8-A / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 07 TCP=false | F8-AAAA / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 08 TCP=false | F8-CNAME / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 09 TCP=false | F8-SVCB / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 10 TCP=false | F8-HTTPS / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 11 TCP=false | F8-A / F8-TTL5; C | 3 | [] | True / 5 | pass |
| RR case 12 TCP=false | F8-A / F8-TTL5; W | 3 | [] | True / 5 | pass |
| RR case 13 TCP=false | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 14 TCP=false | upstream control (not synthetic filtering);  | 3 | [] | True / 5 | pass |
| RR case 01 TCP=true | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 02 TCP=true | upstream control (not synthetic filtering);  | 0 | [28] | False / 0 | pass |
| RR case 03 TCP=true | upstream control (not synthetic filtering);  | 0 | [5] | False / 0 | pass |
| RR case 04 TCP=true | upstream control (not synthetic filtering);  | 0 | [64] | False / 0 | pass |
| RR case 05 TCP=true | upstream control (not synthetic filtering);  | 0 | [65] | False / 0 | pass |
| RR case 06 TCP=true | F8-A / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 07 TCP=true | F8-AAAA / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 08 TCP=true | F8-CNAME / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 09 TCP=true | F8-SVCB / F8-TTL5; D | 0 | [] | True / 5 | pass |
| RR case 10 TCP=true | F8-HTTPS / F8-TTL5; D | 3 | [] | True / 5 | pass |
| RR case 11 TCP=true | F8-A / F8-TTL5; C | 3 | [] | True / 5 | pass |
| RR case 12 TCP=true | F8-A / F8-TTL5; W | 3 | [] | True / 5 | pass |
| RR case 13 TCP=true | upstream control (not synthetic filtering);  | 0 | [1] | False / 0 | pass |
| RR case 14 TCP=true | upstream control (not synthetic filtering);  | 3 | [] | True / 3 | pass |

Positive/upstream and synthetic filtering rows remain distinct. RCODE 0 with zero answers and SOA means NODATA. [BACKEND.md §2.1.1–§2.1.4](../../../dns/BACKEND.md) defines the exact per-type response, including SOA TTL 5.

| Cache query | Class / RCODE | Remaining TTL / SOA TTL | Native ms | Responder | Cache evidence | Authority serial sampled after query |
|---|---|---|---:|---|---|---:|
| cache initial: negative | [] / 3 | [] / 2 | 1 | primary | unknown | 1001 |
| cache initial: positive | ['other_ipv4'] / 0 | [10] / 0 | 1 | primary | miss_authority_exchange_observed | 1001 |
| positive/negative cache before TTL: negative | [] / 3 | [] / 1 | 1 | primary | unknown | 1002 |
| positive/negative cache before TTL: positive | ['other_ipv4'] / 0 | [9] / 0 | 1 | primary | inferred_hit_no_authority_exchange | 1002 |
| positive/negative cache after TTL: formerly_negative | ['other_ipv4'] / 0 | [5] / 0 | 1 | primary | miss_authority_exchange_observed | 1002 |
| positive/negative cache after TTL: positive | ['private_ipv4'] / 0 | [10] / 0 | 1 | primary | miss_authority_exchange_observed | 1002 |

[Complete query captures](F11-AT20-cache.json) retain native query/wrapper start and completion, resolver receipts, authority exchanges, SOA serial/time and TTL. Native probe milliseconds and broader wrapper bounds are distinct. Unique matching metadata identifies a responder; multiple matches remain ambiguous. An observed authority exchange proves a miss. No exchange supports only an **inferred** positive cache hit, never a private AdGuard cache flag. Negative policy responses may be local, so cache state is unknown unless separately attributable. `authority_serial_after_query` is sampled separately, never a cached A record's serial.

The historical 2,502 ms query and premature changed answer remain failed and unattributed in [follow-up 10](history/follow-up-10/F10-03-cache.json). [AT02 diagnostics](F11-AT02-withdrawal.json) retain every read category and bracket; [full producer stream](AT02-enforcement-timing.json) includes original operations, renewals, baseline and heartbeat records. These are command brackets, not exact kernel commit timestamps. The historical unreadable sample remains in [follow-up 10](history/follow-up-10/F10-02-withdrawal.json).

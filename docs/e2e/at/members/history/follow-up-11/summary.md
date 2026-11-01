# Follow-up 11 acceptance results

Run `7edb263f-1f6e-4f94-9e8e-26099f96a699`, harness `33abb2fd2c2fc95826c6e69eb434045522d6f7ef`, product baseline `96ec0630e4834e13a3a14d65553ba8108bb5da55` (devstack 20 #343, agent-core 28 #344, netlab 20 #336; dns-publish 8 #309). Assigned `w4-members / 24000`, two resolvers. The full mutation order is unchanged.

**2 complete cases pass / 6 blocked / 2 fail; 558 pass / 5 blocked / 3 fail subchecks.**

The new **finite-expiry withdrawal row passes**: 186 successful reads,
both reader categories `none`, last reply increments 408.006282 ms before
deadline. Across both AT02 withdrawals, 214 samples / 428 producer snapshots
join 798 original records without gaps. Full AT02 still fails D-077 readiness
and the admitted-sibling ingress control (F11-02/F11-03).

The new **AT20 before-TTL row passes**, with 39/39 complete traces. Primary
returns the old positive class with TTL 9 and no authority exchange (inferred
hit only), then the changed class after TTL with an observed exchange. Negative
cache state stays unknown; the authority serial after a query is separate.

**Full AT09 fails** its dual/1280 bulk row: 13,013/64,064 bytes in 12,010 ms,
selected direct IPv6 underlay before/after. All three MTU-1500 controls pass.
F8-02 remains with netlab 21. AT01 also retains a new first-hop blocker
(F11-01); later passing traffic does not replace it. There are nine pending
request groups in [requests.md](requests.md).

| Case / evidence | Requirements | Full status | Pass / blocked / fail |
|---|---|---|---:|
| [AT01](AT01.json) | §19.1 §19 AT01 APP01 ID01 MEM01 | blocked | 16 / 0 / 0 |
| [AT02](AT02.json) | §19.1 §19 AT02 MEM01 MEM02 MEM03 SEC01 | fail | 121 / 0 / 2 |
| [AT03](AT03.json) | §19.1 §19 AT03 BILL01 BILL03 ENT01 | pass | 92 / 0 / 0 |
| [AT04](AT04.json) | §19.1 §19 AT04 BILL02 ENT02 MEM03 | blocked | 99 / 0 / 0 |
| [AT05](AT05.json) | §19.1 §19 AT05 NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 |
| [AT08](AT08.json) | §19.1 §19 AT08 NODE02 NODE03 EXIT01 EXIT03 | pass | 31 / 0 / 0 |
| [AT09](AT09.json) | §19.1 §19 AT09 EXIT02 APP01 NFR01 | fail | 37 / 5 / 1 |
| [AT20](AT20.json) | §19.1 §19 AT20 DNS01 DNS02 ISO04 | blocked | 81 / 0 / 0 |
| [AT21](AT21.json) | §19.1 §19 AT21 DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 |
| [AT22](AT22.json) | §19.1 §19 AT22 SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 |

Desired/applied **30 → 30 / 30 → 30**; serving in **6.032389860 s**, first counted byte in **11.399224550 s**. Recovery row: **pass**.

D-129 closes F10-04: follow-up 10's unchanged full sequence recovered at generation 29 (serving 15.751594344 s, first counted byte 21.357052110 s, three provisional reports under D-126). Generation 30 is not a contract value. The original generation-30 F8-01 closure stays in [follow-up 9](history/follow-up-9/RUN.md). No forced generation or state/topology repair is used.

F10-02 and F10-03 remain historically failed and unattributed in the [35-file follow-up 10 archive](history/follow-up-10/SHA256SUMS). A new pass is a separate measurement, not a repair of the unreadable sample or missing DNS responder. Follow-up 8/9 archives, handover snapshots and external provenance remain unchanged.

The complete AT09 case requires every selector row. QUIC and Android/iOS/Windows/macOS remain resource-dependent; no passing recovery or bulk subset certifies them. [requests.md](requests.md) retains exact owner boundaries. D-123 measures IPv4 payload over selected underlays, with IPv6Block; no IPv6 service/PMTU claim is added.

DNS assertions retain [BACKEND.md §2.1.1–§2.1.4](../../../dns/BACKEND.md): blocked A/AAAA/HTTPS are NXDOMAIN, direct CNAME/SVCB are NODATA, with SOA TTL 5 for the filtering profile. IDNA, distribution, Private DNS/DoH, rendered screens, physical arm64, privacy/retention and commercial feed approval remain separate resource gaps. D-119 keeps immediate replacement separate from natural finite expiry to 0/3.

Failed checks in this run:

- AT02: D-077 re-grant counted forwarding
- AT02: strict post-withdrawal fvmb0 ingress injection
- AT09: native dual MTU 1280 transfer

[Native and DNS evidence](NATIVE.md), [run ledger and gates](RUN.md).

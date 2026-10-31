# Follow-up 10 acceptance results

Run `2be2d32f-bfd9-46f5-bf29-afaa6617ae94`, harness `731be5a7af7657660d96c240695747a23a3f5c96`, product baseline `d3e9a0f1` (agent-core 27 #325, wire-nodes 32 #321, netlab 19 #318; devstack 18 and dns-publish 8 retained), assigned `w4-members / 24000`.

**2 complete cases passed; 6 blocked; 2 failed.** Subchecks: 560 pass / 5 blocked / 2 fail.

**AT09:** all 38 executed checks pass, including the D-126 mixed-report predicate;
five platform/QUIC selectors and the wider platform matrix remain blocked.
The complete original mutation order yielded **desired/applied 29 → 29**:
serving in **15.752 s**, first counted byte in **21.357 s**, 88 heartbeat samples
over **90.354 s**, and the initial `TC_FOREIGN_FILTER` retained. **F10-04: this
is not exact generation-30 qualification**; no extra mutation forced that number.
The original generation-30 capture remains in follow-up 9 and is replayed offline.
The MTU-1280 IPv4 row passes (64,064 bytes / 1 ms); F8-02 remains unattributed.

**New failures remain failed:** F10-02 is AT02's one unreadable withdrawal sample
among 234; replies stopped before expiry and counters stabilized, but the
all-reads-valid predicate fails. F10-03 is AT20's changed positive answer before
TTL; the negative and after-TTL controls pass. Full captures and dated ownership
are in [requests.md](requests.md). No failed assertion was retried or relaxed.

| Case / evidence | Requirements | Status | Pass / blocked / fail | Owner / boundary |
|---|---|---|---:|---|
| [AT01](AT01.json) | §19.1; §19 AT01; APP01 ID01 MEM01 | blocked | 18 / 0 / 0 | Operator/platform resources; OPERATORS.md |
| [AT02](AT02.json) | §19.1; §19 AT02; MEM01 MEM02 MEM03 SEC01 | fail | 121 / 0 / 1 | F10-02 unreadable withdrawal sample; members/devstack |
| [AT03](AT03.json) | §19.1; §19 AT03; BILL01 BILL03 ENT01 | pass | 92 / 0 / 0 | Members; billing evidence |
| [AT04](AT04.json) | §19.1; §19 AT04; BILL02 ENT02 MEM03 | blocked | 99 / 0 / 0 | Portal/devstack/operator: rendered screens |
| [AT05](AT05.json) | §19.1; §19 AT05; NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 | Packaging/devstack/operator: boot sources, physical arm64 |
| [AT08](AT08.json) | §19.1; §19 AT08; NODE02 NODE03 EXIT01 EXIT03 | pass | 31 / 0 / 0 | Members; publication/readback |
| [AT09](AT09.json) | §19.1; §19 AT09; EXIT02 APP01 NFR01 | blocked | 38 / 5 / 0 | F9-01 closed; F10-04 exact-generation gap; F8-02 and platform resources |
| [AT20](AT20.json) | §19.1; §19 AT20; DNS01 DNS02 ISO04 | fail | 80 / 0 / 1 | F10-03 before-TTL cache failure; IDNA/distribution/platform resources |
| [AT21](AT21.json) | §19.1; §19 AT21; DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 | Native integration/operator: production distribution, Private DNS/DoH |
| [AT22](AT22.json) | §19.1; §19 AT22; SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 | Devstack/usage/database/backup: retention/URL; business feed approval |

D-119 (2026-10-31): retire the inherited “3/0 → 2/1 deadline transition” wording. Natural finite-grant expiry is 0/3 with no other covering grant. Billing calls `ReplaceFiniteGrant` at a period boundary; the harness retains immediate 3/0 → 2/1 → 3/0 replacement and separately measures genuine expiry/restoration. Plan v1.0 has no scheduled plan-change contract.

D-123: all native bulk payloads use IPv4; IPv4/IPv6/dual labels select **underlay** availability. V1.0's IPv6Block policy drops forwarded IPv6. [Netlab 18](../../../poc/G2-f802-mtu1280-2026-10-31.md) supplies the independent no-leak control and IPv4 fragmentation-needed traversal measurements. There is no end-to-end IPv6 MTU/PMTU row or claim. A standalone pass cannot close the combined native AT09 matrix.

DNS wording retired 2026-10-31 by [dns-publish follow-up 8's measured table](../../../dns/BACKEND.md): blocked A/AAAA/HTTPS are NXDOMAIN; direct CNAME/SVCB are NODATA. F8-TTL5 requires SOA TTL 5 for this profile. [NATIVE.md](NATIVE.md) links each fresh native row to observed RCODE, answer types and SOA TTL. Positive/upstream/cache controls remain separate. IDNA, distribution, platform and retention gaps are unchanged.

F4-08 / F4-09: never defined; retired 2026-10-31. The follow-up 4 ledger is F4-01 … F4-07 and F4-10. [requests.md](requests.md) records dated owner actions; [BLOCKERS.md](BLOCKERS.md) retains platform/operator references.

D-098: follow-up 8 run `69f92f49` remains [byte-identical](history/follow-up-8/RUN.md), with [32-file SHA256SUMS](history/follow-up-8/SHA256SUMS). Its failed measurements are historical and unchanged. [RUN.md](RUN.md) contains current gate exits.

Failed checks retained:

- AT02: persistent withdrawal member-a3
- AT20: positive/negative cache before TTL

D-126 / D-128: [agent-core 27](../../../node/followup27-members-2026-10-31.md) confirms deliberate cumulative composition, no reporter race or producer fix. [Wire-nodes 32](https://github.com/TED-H-com/Freedom-VPN/pull/321) preserves the exact JSON value/JCS/digest with the existing 202 receipt. F9-01 is closed by the focused predicate/replay; recovery alone never passes full AT09. Follow-up 9 remains [byte-identical](history/follow-up-9/RUN.md), with [33-file SHA256SUMS](history/follow-up-9/SHA256SUMS).

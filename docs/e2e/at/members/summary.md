# Follow-up 9 acceptance results

Run `11e305fe-6360-4dbd-95b1-67274aa4988c`, harness `49fbb48e013b824c4ce107d3cb551db79b915b42`, product baseline `7e044c71` (agent-core 25 #312, devstack 18 #313, dns-publish 8 #309 and netlab 18), assigned `w4-members / 24000`.

**3 complete cases passed; 6 blocked; 1 failed.** Subchecks: 560 pass / 5 blocked / 2 fail.

| Case / evidence | Requirements | Status | Pass / blocked / fail | Owner / boundary |
|---|---|---|---:|---|
| [AT01](AT01.json) | §19.1; §19 AT01; APP01 ID01 MEM01 | blocked | 18 / 0 / 0 | Operator/platform resources; OPERATORS.md |
| [AT02](AT02.json) | §19.1; §19 AT02; MEM01 MEM02 MEM03 SEC01 | pass | 122 / 0 / 0 | Members; D-119 resolved, no scheduled-change contract |
| [AT03](AT03.json) | §19.1; §19 AT03; BILL01 BILL03 ENT01 | pass | 92 / 0 / 0 | Members; billing evidence |
| [AT04](AT04.json) | §19.1; §19 AT04; BILL02 ENT02 MEM03 | blocked | 99 / 0 / 0 | Portal/devstack/operator: rendered screens |
| [AT05](AT05.json) | §19.1; §19 AT05; NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 | Packaging/devstack/operator: boot sources, physical arm64 |
| [AT08](AT08.json) | §19.1; §19 AT08; NODE02 NODE03 EXIT01 EXIT03 | pass | 31 / 0 / 0 | Members; publication/readback |
| [AT09](AT09.json) | §19.1; §19 AT09; EXIT02 APP01 NFR01 | fail | 36 / 5 / 2 | Agent-core/wire-nodes: F9-01 mixed-state semantics; devstack: historical F8-02 attribution; platform/operator |
| [AT20](AT20.json) | §19.1; §19 AT20; DNS01 DNS02 ISO04 | blocked | 81 / 0 / 0 | Devstack: IDNA positive wire; native distribution/platform; DNS wording retired |
| [AT21](AT21.json) | §19.1; §19 AT21; DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 | Native integration/operator: production distribution, Private DNS/DoH |
| [AT22](AT22.json) | §19.1; §19 AT22; SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 | Devstack/usage/database/backup: retention/URL; business feed approval |

**AT09:** F8-01 recovers at unchanged generation 30 (fresh serving 15.729 s; first counted byte 21.374 s). The dual-underlay/1280 IPv4 transfer completes 64,064 bytes in 1 ms. The case still fails on F9-01: mixed provisional/TC-rejection codes fail the D-067 check and its composite recovery assertion. The original 11,011-byte F8-02 cause remains unattributed; this successful row does not establish a fix. [Measurements](NATIVE.md), [capture](AT09-restart.json), [owner request](requests.md).

D-119 (2026-10-31): retire the inherited “3/0 → 2/1 deadline transition” wording. Natural finite-grant expiry is 0/3 with no other covering grant. Billing calls `ReplaceFiniteGrant` at a period boundary; the harness retains immediate 3/0 → 2/1 → 3/0 replacement and separately measures genuine expiry/restoration. Plan v1.0 has no scheduled plan-change contract.

D-123: all native bulk payloads use IPv4; IPv4/IPv6/dual labels select **underlay** availability. V1.0's IPv6Block policy drops forwarded IPv6. [Netlab 18](../../../poc/G2-f802-mtu1280-2026-10-31.md) supplies the independent no-leak control and IPv4 fragmentation-needed traversal measurements. There is no end-to-end IPv6 MTU/PMTU row or claim. A standalone pass cannot close the combined native AT09 matrix.

DNS wording retired 2026-10-31 by [dns-publish follow-up 8's measured table](../../../dns/BACKEND.md): blocked A/AAAA/HTTPS are NXDOMAIN; direct CNAME/SVCB are NODATA. F8-TTL5 requires SOA TTL 5 for this profile. [NATIVE.md](NATIVE.md) links each fresh native row to observed RCODE, answer types and SOA TTL. Positive/upstream/cache controls remain separate. IDNA, distribution, platform and retention gaps are unchanged.

F4-08 / F4-09: never defined; retired 2026-10-31. The follow-up 4 ledger is F4-01 … F4-07 and F4-10. [requests.md](requests.md) records dated owner actions; [BLOCKERS.md](BLOCKERS.md) retains platform/operator references.

D-098: follow-up 8 run `69f92f49` remains [byte-identical](history/follow-up-8/RUN.md), with [32-file SHA256SUMS](history/follow-up-8/SHA256SUMS). Its failed measurements are historical and unchanged. [RUN.md](RUN.md) contains current gate exits.

Failed checks retained:

- AT09: D-067 revision 11 provisional lease observation
- AT09: same-generation D-067 restart recovery and lease sequence

# Follow-up 8 acceptance results

Run `69f92f49-73dc-4779-8101-dfb6a802a96b`, harness `d79accd9c693aed83713072c925b3b5005b5684c`, product baseline `6848add` (#284), assigned `w4-members / 24000`. **3 complete cases passed; 6 blocked; 1 failed.** Subchecks: 558 pass / 5 blocked / 4 fail. Each complete-case status retains its failures and external blockers.

| Case | Requirements / spec | Status | Subchecks pass / blocked / fail |
|---|---|---|---:|
| [AT01](AT01.json) | §19.1; §19 AT01; APP01 ID01 MEM01 | blocked | 18 / 0 / 0 |
| [AT02](AT02.json) | §19.1; §19 AT02; MEM01 MEM02 MEM03 SEC01 | pass | 122 / 0 / 0 |
| [AT03](AT03.json) | §19.1; §19 AT03; BILL01 BILL03 ENT01 | pass | 92 / 0 / 0 |
| [AT04](AT04.json) | §19.1; §19 AT04; BILL02 ENT02 MEM03 | blocked | 99 / 0 / 0 |
| [AT05](AT05.json) | §19.1; §19 AT05; NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 |
| [AT08](AT08.json) | §19.1; §19 AT08; NODE02 NODE03 EXIT01 EXIT03 | pass | 31 / 0 / 0 |
| [AT09](AT09.json) | §19.1; §19 AT09; EXIT02 APP01 NFR01 | fail | 34 / 5 / 4 |
| [AT20](AT20.json) | §19.1; §19 AT20; DNS01 DNS02 ISO04 | blocked | 81 / 0 / 0 |
| [AT21](AT21.json) | §19.1; §19 AT21; DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 |
| [AT22](AT22.json) | §19.1; §19 AT22; SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 |

The finite grant hook performs an immediate 3/0 → 2/1 downgrade and a 3/0 restoration. Genuine deadline expiry is a separate measured transition to 0/3 because this member has no fallback entitlement. [AT02-TRACE.md](AT02-TRACE.md) indexes the evidence; D-119 (2026-10-31) retires the inherited “3/0 → 2/1 deadline transition” wording: period-boundary replacement belongs to billing; Plan v1.0 has no scheduled plan-change contract.

AT09 records the selected exit family before and after each 64,064-byte transfer, upload/download counters, MTU 1500/1280, and explicit unavailable declarations for QUIC and four non-Linux platforms. AT20–AT22 use the native OS stub and exact two-resolver list, with TEST ONLY distribution. Direct blocked CNAME/SVCB are empty NODATA; blocked A/AAAA/HTTPS are NXDOMAIN. DNS-wording part retired 2026-10-31 by dns-publish follow-up 8’s per-type table; other gaps remain. Every RR case retains RCODE, SOA TTL and answer types.

**AT09 failures:** second exit fresh serving after restart; D-067 revision 11 provisional lease observation; same-generation D-067 restart recovery and lease sequence; native dual MTU 1280 transfer.

[BLOCKERS.md](BLOCKERS.md) rechecks F4-03 through F4-10 against landed PRs. [requests.md](requests.md) records owner actions. F4-08 / F4-09: never defined; retired 2026-10-31. [RUN.md](RUN.md) contains actual gate exits and output tails. Follow-up 7 final run `297faa31` is [archived byte-identically](history/follow-up-7/RUN.md), with [SHA256SUMS](history/follow-up-7/SHA256SUMS). [Diagnostic attempt 1](history/follow-up-8/attempt-1/RUN.md) is preserved separately and supplies no current pass claim.

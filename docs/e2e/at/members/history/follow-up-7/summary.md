# Follow-up 7 acceptance results

Run `297faa31-2f90-4aab-8cf0-6818629f46c8` on `w4-members / 24000`, product baseline `d9e28f8` (includes #240, #244 and D-081 erratum #247), harness `10a5f3891aa8fde2d3a2f158d97bfcb0a4c7670b`.
**3 complete cases passed; 7 blocked; 0 failed.** Executed subchecks: 471 pass / 0 fail. These claims use only this final run; the interrupted [fixture failure](history/follow-up-7/attempt-1/RUN.md) and [meter-sequence collision](history/follow-up-7/attempt-2/RUN.md) are retained separately.

| Case | Requirements | Status | Subchecks pass / fail | Deciding observation |
|---|---|---|---:|---|
| [AT01](AT01.json) | APP01 ID01 MEM01 | subchecks pass / blocked on F4-03 | 18 / 0 | Portal/native identity and Linux counted forwarding before/after pause; four native platforms and network moves remain. |
| [AT02](AT02.json) | MEM01 MEM02 MEM03 SEC01 DATA01 | pass | 106 / 0 | One final-slot winner; 3→2→3 slot reuse; revoked fvmb0 injection hits ctunku while sibling is counted. D-077 fence/list/cursor/re-grant/replay and D-081 limit denial are recorded separately. |
| [AT03](AT03.json) | BILL01 BILL03 ENT01 DATA01 | pass | 93 / 0 | Ten identical paid deliveries and six event orders preserve one exact paid interval; paid native device is counted before fencing and after explicit HTTP re-grant. |
| [AT04](AT04.json) | BILL02 ENT02 MEM03 | subchecks pass / blocked on F4-03/04 | 86 / 0 | Refund, real expiry and suspension terminate persistent TCP on both exits and deny new flows; member UDP and rendered suspended/expired states remain. |
| [AT05](AT05.json) | NODE01 NODE04 ENR01 ENR04 PKG01 | subchecks pass / blocked on F4-10 | 30 / 0 | Installer enrollment/replay and signed architecture artifacts pass; preinstalled/cloud-init boot sources and physical arm64 remain. |
| [AT08](AT08.json) | NODE02 NODE03 EXIT01 EXIT03 | pass | 39 / 0 | A/B group isolation, disabled restart and republish carry operation readback and actual traffic evidence. |
| [AT09](AT09.json) | EXIT02 APP01 NFR01 | subchecks pass / blocked on F4-03 | 27 / 0 | D-067 same-generation restart: 36.997 s to serving-ready, 41.729 s to first counted byte; forced relay/QUIC/PMTU and physical second NAT remain. |
| [AT20](AT20.json) | DNS01 DNS02 ISO04 | subchecks pass / blocked on F4-05/07 | 26 / 0 | Typed resolver delivery and controlled rule answers pass; native DNS configuration and complete RR/cache/rebinding fixtures remain. |
| [AT21](AT21.json) | DNS03 DNS04 APP01 | subchecks pass / blocked on F4-05 | 14 / 0 | Resolver corruption/last-good recovery and two explicit filtering resolver endpoints pass; native automatic fallback and Private DNS/DoH require operator coverage. |
| [AT22](AT22.json) | SEC02 DNS01 DNS04 | subchecks pass / blocked on F4-06 | 32 / 0 | Privacy scans, guarded audit retention and scheduled feed last-good subsets pass; remaining retention/file/backup/URL surfaces and commercial-feed approval remain. |

[D-077](D077.md) records every HTTP status, operation identity, fence count and wire body location. Successful 202 responses are confirmed by worker device-list readback and counted forwarding. The fence input is explicit SQL in the isolated lab database, matching migration 00188; other business state is produced through the product.

AT09's `serving_ready_at` is `2026-10-31T04:27:04.060261733Z`: **36.997 s**, versus follow-up 6's **36.976 s** (+0.021 s). The first counted byte is at `2026-10-31T04:27:08.792505245Z` (41.729 s). Desired generation remains `24`. The provisional observation check records `awaiting_member_grants`, one lease-stage `AWAITING_MEMBER_GRANTS`, and absence of `lease_expired`. Full AT09 remains blocked on F4-03; this measured lab recovery is not a transport or hardware guarantee.

The worker RestartPolicy note is closed: this run observes `unless-stopped` for worker/control/NetBird and no longer manually starts the worker during scheduled feed refresh. F4-03/04/05/06/07/10 remain external blockers. See [requests.md](requests.md), [BLOCKERS.md](BLOCKERS.md), [RUN.md](RUN.md) and [OPERATORS.md](OPERATORS.md).

Follow-up 6 is preserved under [history/follow-up-6](history/follow-up-6/RUN.md). Native platform, physical arm64 and second-NAT rows remain **needs operator run**.

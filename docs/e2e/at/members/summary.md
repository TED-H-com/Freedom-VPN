# Follow-up 12 acceptance results

Run `b98b78ee-9162-419e-ae2f-d1d9a618a3e8`, compiled harness `65e18e67f482d4c51dd3f86a8c7db08a1ab62f33` (clean source), product baseline `45209609d33d567c5f04d49441799cdd3efac59b`. This baseline includes #345–#353, netlab 21 #355 and agent-core 29 #356. Assigned `w4-members / 24000`, two resolvers. `go env GOCACHE` returned `/data/ted-h-cache/go-build`; no cache override or purge was used.

**Eight fresh cases: 0 pass / 4 blocked / 4 fail; 468 pass / 0 blocked / 14 failed subchecks.** Full-case blockers are separate from subcheck counts. AT01 and AT09 were skipped by instruction, and their root JSON/Markdown and native capture files remain byte-identical to follow-up 11. They are not included in these fresh totals.

| Fresh case | Requirements | Full status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | §19.1 §19 AT05 NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 |
| [AT02](AT02.json) | §19.1 §19 AT02 MEM01 MEM02 MEM03 SEC01 | blocked | 150 / 0 / 0 |
| [AT03](AT03.json) | §19.1 §19 AT03 BILL01 BILL03 ENT01 | fail | 70 / 0 / 1 |
| [AT04](AT04.json) | §19.1 §19 AT04 BILL02 ENT02 MEM03 | fail | 60 / 0 / 7 |
| [AT08](AT08.json) | §19.1 §19 AT08 NODE02 NODE03 EXIT01 EXIT03 | fail | 27 / 0 / 4 |
| [AT20](AT20.json) | §19.1 §19 AT20 DNS01 DNS02 ISO04 | fail | 80 / 0 / 2 |
| [AT21](AT21.json) | §19.1 §19 AT21 DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 |
| [AT22](AT22.json) | §19.1 §19 AT22 SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 |

Preserved only: [AT01](history/follow-up-11/AT01.json) blocked **16 / 0 / 0** (F11-01); [AT09](history/follow-up-11/AT09.json) failed **37 / 5 / 1** (F8-02). Neither was re-observed in this run.

**F11-02 closes with a separate fresh measurement.** Both AT02 D-077 positive probes wait for populated grants, accepted serving state, intended binding and an original same-boot producer heartbeat. Generations 10/21 join producer run `3065df99` sequences 219/602; both counted transfers pass. [Predicate and rows](AT02.md), [exact evidence](F12-F11-02-readiness.json). Run 7edb263f's failed row stays archived.

**F11-03 remains blocked: no failed Apply observed.** All 89 AT02 Applies succeed and retain agent-core 29 host/command fields. Maximum render→installed is 26.889651 ms; maximum completed guard interval is 51.260362 ms. No failed-Apply recovery is certified. The admitted stream retains 3,976 attempts, 3,976 ingresses, 3,955 egresses and **21 losses**, with zero sender/read errors and zero socket drops. Thirteen loss ingress times lie in completed guards, eight outside. Those correlations do not prove kernel drop causes. D-135 now measures baseline→next successful Apply, preserving every loss and revocation. A diagnostic contention label never changes a verdict; cause remains unproven. [Full capture and original Applies](F12-AT02-recovery.json).

**F12-01:** the eight outside-guard losses lie inside a separate **236.970251 ms** `policy_missing` baseline→successful generation-13 Apply bracket, containing nine total losses. First post-Apply ingress with egress is 9.654891 ms later. Records 362–366 show successful baselines, a populated renewal and a verified Apply, not a failed-Apply rollback. Agent-core/netlab interpretation is requested; underlying cause remains unproven. [Original window](F12-01-policy-missing.json).

**F12-02:** the cumulative producer JSONL crosses the shared diagnostic reader's 1 MiB cap. AT03 stops at readiness; AT04/AT08 retain their readiness failures. A read-only call reproduces the explicit export limit while `ExportContainerFile` streams 1,591,639 bytes. The owned reader correction is commit `7b949b68`, after this run's compiled harness; it has offline and read-only export validation, not live qualification. AT04's separate 85-second secondary TCP monitor ends before the delayed refund input; its failed row and zero-time sentinel remain, with no withdrawal-latency inference. [Capacity evidence](F12-02-stream-limit.json), [seven AT04 failures](F12-AT04-failures.json). No repeat run replaces any failure.

**AT20 stays failed, 80 / 0 / 2.** The positive sequence uniquely matches primary: TTL 10→8 with no new exchange, then a new answer after expiry with a fetch (inferred F9-HIT). **F12-03:** the negative half starts with remaining SOA TTL 2; a same-primary positive refetch 1.228672687 s later is compatible with whole-second expiry uncertainty (F9-ROUND). No responder switch or outstanding completion is observed. Internal expiry versus invalidation is unproven; the unchanged combined before-TTL assertion remains failed. **F12-04:** UDP case 01 returns native SERVFAIL in 5,001 ms despite positive receipts from both resolvers. Its responder and delivery/client-processing cause remain unattributed. F9-OTHER and F9-INFLIGHT are considered without claiming fallback or an AdGuard cache defect. Actual before/after settings, complete request histories, outstanding sets, TTLs and query bounds are retained. [AT20 interpretation](AT20.md), [cache sequence](F12-AT20-cache.json), [negative window](F12-03-negative-cache.json), [SERVFAIL exchanges](F12-04-native-servfail.json).

AT02 finite expiry retains **193/193** successful reads, last TCP/UDP reply increments observed **85.203130 ms before D**. Both withdrawals retain 221 samples and 442 joined snapshots in an 812-record producer stream. D-134's [D−101 ms,D] window is not an early-loss waiver; sampled reply increments are not exact cutoff timestamps. AT21/AT22 have no failed checks but keep resource/business blockers. [Withdrawal evidence](F12-AT02-withdrawal.json).

D-098 archive commit **201dcd1d** came first: all 43 run-7edb263f root files are byte-identical in [follow-up 11](history/follow-up-11/RUN.md), pinned by [SHA256SUMS](history/follow-up-11/SHA256SUMS); `attempt-1/` and older history are unchanged. D-132 permits evidence pins only in immutable `history/`. F10-02/F10-03 and run 2be2d32f remain failed/unattributed. The historical generation-13 failures keep their old expectation and unproven cause.

Twelve pending request groups are dated in [requests.md](requests.md). [Run ledger and gates](RUN.md).

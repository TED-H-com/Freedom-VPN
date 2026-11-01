# Members acceptance — Follow-up 13

Run `4182dd3d-7f1f-453e-8c1f-265d097c3abc`, clean compiled harness `d49b535adf831b6823cd05bb47495437fbf6ee63`, product baseline `de6d2e0246892b5744f68864c3d6d2854945c40a`
(includes agent-core 30 #357, devstack 22 #362, netlab 22 #365).
`go env GOCACHE` returned `/data/ted-h-cache/go-build`; no cache override or purge.
Assigned `FVPN_E2E_PREFIX=w4-members`, `FVPN_E2E_PORT_BASE=24000`, two resolvers.

**2 PASS / 6 BLOCKED / 2 FAIL**, with **616 passed / 7 blocked / 3 failed subchecks**.
One regular ten-case run, including AT01 and AT09; live make exit **2**.

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | §19.1 §19 AT05 NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 |
| [AT01](AT01.json) | §19.1 §19 AT01 APP01 ID01 MEM01 | blocked | 22 / 0 / 0 |
| [AT02](AT02.json) | §19.1 §19 AT02 MEM01 MEM02 MEM03 SEC01 | fail | 136 / 0 / 1 |
| [AT03](AT03.json) | §19.1 §19 AT03 BILL01 BILL03 ENT01 | pass | 95 / 0 / 0 |
| [AT04](AT04.json) | §19.1 §19 AT04 BILL02 ENT02 MEM03 | fail | 115 / 0 / 2 |
| [AT08](AT08.json) | §19.1 §19 AT08 NODE02 NODE03 EXIT01 EXIT03 | pass | 33 / 0 / 0 |
| [AT20](AT20.json) | §19.1 §19 AT20 DNS01 DNS02 ISO04 | blocked | 84 / 0 / 0 |
| [AT21](AT21.json) | §19.1 §19 AT21 DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 |
| [AT22](AT22.json) | §19.1 §19 AT22 SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 |
| [AT09](AT09.json) | §19.1 §19 AT09 EXIT02 APP01 NFR01 | blocked — under provisional D-137 | 50 / 7 / 0 |

- **F11-01:** first selection succeeds with one SYN, node/origin receipt and counted traffic. All three captures are complete (269 / 292 / 77 records, zero drops). Type-4 transport data appears before a loopback-source IPv6 response arrives after the successful request. This differs from devstack 22's five-SYN/no-type-4 reproduction; neither historical cause nor a fix is inferred. The original prehistory and sleep/wake branch remain. [Own capture and comparison](F13-F11-01-forwarding.json).
- **AT02 under provisional D-135 + D-136:** 95 successful Applies, zero failed Applies; F11-03 stays **blocked on no failed Apply observed**. All 61 baseline calls are paired to the next successful Apply, with original rows/run identities. Three overlap the sibling window; the other 58 have command recovery evidence and explicitly unavailable sibling coverage. **4,029 attempts / 4,029 ingresses / 4,008 egresses / 21 losses**; 13 inside completed guards, eight outside. No losses are excluded. Sequences **402–404** are successful `policy_missing` baselines on the generation-11 lease, **405** renews to 13, **406** applies successfully. The first bracket is **219.395230 ms**, contains nine losses (one inside a guard, eight outside), and has first subsequent egress **14.828914 ms** after Apply. Repeated baseline brackets overlap and are never summed as unique losses. The no-loss row stays FAIL, even though recovery is measured. [Full accounting](F13-AT02-recovery.json), [original transition](F13-AT02-policy-missing.json).
- **F12-02 live-qualified:** all **22 readiness checks** across AT03/AT04/AT08 and **12 monitor stream checks** pass. AT03 streams 1,160,894–1,277,148 bytes; AT08 reaches **2,455,378 bytes / 2,262 records**. AT04's primary refund monitor has 196 samples with **392 exact producer identities beyond 1 MiB**, cumulative offsets 1,407,676–1,527,951. Secondary streams are reported at their actual smaller sizes when applicable, never padded. [Qualification](F13-F12-02-stream-qualification.json), [immutable stream and failed monitor](history/follow-up-13-stream/4182dd3d-7f1f-453e-8c1f-265d097c3abc/SHA256SUMS). Run b98b78ee's eleven readiness failures and early-finished monitor remain failed in history.
- **F13-01:** AT04 has two separate failed persistent-withdrawal checks: primary refund and secondary suspension each contain one `nft_parse` / `parse` sample with command exit 0. The remaining status reads succeed; observed reply cutoffs are 3.558769767 s and 2.596756019 s, with active senders and stable counters. Incomplete reads keep both rows failed. Raw nft output/parser errors were not retained, so their cause is unproven. [Failed rows, neighbors and producer windows](F13-01-counter-parse.json); shared reader/parser changes are handed off.
- **F12-03 quiet window passes:** six seconds without fixture queries (measured 6.004276509 s), then fresh negative SOA TTL **5 → 3**, same uniquely matched primary, approximately **1.221 s** between query completions, before the receipt-plus-four-second conservative boundary. No new authority fetch or outstanding completion is recorded. The combined before-TTL assertion is unchanged and passes. No shared fixture, cache setting or flush changed. [Exact window](F13-F12-03-negative-cache.json). Historical TTL 2 / 1.228672687 s F12-03 stays failed. F12-04's original SERVFAIL and positive resolver receipts remain unchanged and owned by devstack; this run does not diagnose them.
- **AT09 verdict: BLOCKED under provisional D-137.** Known-PMTU 1280 completes **64,064 bytes in 1 ms**, with route metric 1280 before/after traffic, exact removal and **11 delivered reverse IPv6 fragment pairs**. Reactive PTB remains pending devstack 23 on this tested baseline. Both receive-only 1280 rows also transfer **64,064 bytes in 1 ms**, but remain explicit **PMTU black-hole limitation rows, never PASS**. Reverse data packets are only 1,132 bytes on IPv4 / 1,152 bytes on IPv6, so oversized loss was not reproduced; no PTB appears. Historical F8-02 failures stay failed. [Separate rows and captures](F13-D137-native-paths.json).

[Run and gates](RUN.md). [Ten pending request groups](requests.md).

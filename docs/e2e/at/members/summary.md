# Follow-up 15 acceptance results

Run `e80bb592-27ed-4ab5-a8c3-6433e7278938`, compiled harness `cc6e0298a791ed6c41b268f1634f543446f8c562`, baseline `9dde5eeb60a6cb6b9b10974c3100ef1ea1352698` (current main at admission; #375, #376, #377, #381 and #382 included).

**2 PASS / 6 BLOCKED / 2 FAIL**, with **644 passed / 6 blocked / 2 failed subchecks**. One assembled ten-case run; live make exit **2**.

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | §19.1 §19 AT05 NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 31 / 0 / 0 |
| [AT01](AT01.json) | §19.1 §19 AT01 APP01 ID01 MEM01 | blocked | 19 / 0 / 0 |
| [AT02](AT02.json.gz) | §19.1 §19 AT02 MEM01 MEM02 MEM03 SEC01 | fail | 142 / 0 / 1 |
| [AT03](AT03.json) | §19.1 §19 AT03 BILL01 BILL03 ENT01 | pass | 97 / 0 / 0 |
| [AT04](AT04.json) | §19.1 §19 AT04 BILL02 ENT02 MEM03 | blocked | 118 / 0 / 0 |
| [AT08](AT08.json) | §19.1 §19 AT08 NODE02 NODE03 EXIT01 EXIT03 | pass | 35 / 0 / 0 |
| [AT20](AT20.json) | §19.1 §19 AT20 DNS01 DNS02 ISO04 | blocked | 89 / 0 / 0 |
| [AT21](AT21.json) | §19.1 §19 AT21 DNS03 DNS04 APP01 | blocked | 13 / 0 / 0 |
| [AT22](AT22.json) | §19.1 §19 AT22 SEC02 DNS01 DNS04 | blocked | 40 / 0 / 0 |
| [AT09](AT09.json) | §19.1 §19 AT09 EXIT02 APP01 NFR01 | fail | 60 / 6 / 1 |

**682d4434 is now live-qualified:** the forced relay row passes with **Relayed=2 / Direct=0**, followed by same-socket relay-to-direct recovery. The routed hop is added only afterward. Follow-up 14's relay FAIL remains recorded and confounded; it is not reclassified.

The routed IPv6 row passes **64,064 bytes in 1 ms**, with one oversized ingress, one type-2/code-0 PTB seen at router and sender, learned PMTU 1280, and **16** router fragment egresses received. The routed IPv4 row passes **64,064 bytes in 1 ms**: **25 DF=0 oversized datagrams → 50 fragments / 25 groups**, all received, no PTB. The known-PMTU IPv6 row separately passes **64,064 bytes in 1 ms**.

Receive-only rows remain limitations: dual-underlay MTU 1280 is **FAIL**, with **12,012 bytes in 12,011 ms**; IPv6 receive-only transfers **64,064 bytes in 1 ms** but remains **BLOCKED**. Neither is upgraded to a supported-PMTU pass. [Original independent rows](F15-D137-native-paths.json), [packet analyses](history/follow-up-15-captures/e80bb592-27ed-4ab5-a8c3-6433e7278938/analysis.json). Historical F8-02 failures remain unattributed; IPv4 overlay payloads do not prove IPv6 overlay support (D-123).

AT02 retains **93 successful Applies, zero failed Applies**, with command lifecycle records. All **65 baseline calls** are paired to a successful recovery Apply; ten brackets overlap the continuous capture. The stream records **4,131 attempts / 3,882 node ingresses / 3,854 egresses**. All **277 attempts without observed egress** count: **249 absent at node ingress**, plus **28 observed ingress-without-egress (12 inside completed guards / 16 outside)**. The missing ingress sequences are **1108–1356**; their send times and guard placement are unavailable. The 28 located losses are not the full loss total. Capture reports zero kernel drops/read errors, but the missing-ingress gap makes `measurement_valid=false`; the original D-135/D-136 row remains **FAIL**. All covered brackets retain a post-Apply egress; overlapping brackets are never summed as unique losses. Cause remains unproven.

[Original recovery accounting](F15-AT02-recovery.json.gz), [missing-ingress receipts](F15-AT02-missing-ingress.json), [policy-missing windows](F15-AT02-policy-missing.json.gz). Corrected D-136 remains baseline → ACK → renew → Apply → verify → heartbeat for ordinary stale leases. Only a retained Apply error enables the verified deny-all candidate. There is no finite outage ceiling. D-139 option (b) is a design track, absent from this baseline; no loss waiver. F11-03 remains blocked on no failed Apply observed.

The direct AT02 pause/SIGTERM request-to-observed-exit upper bound is **33.548444 ms**, within **13 s**. The retained launch ID/PID/start-ticks join resolves to parent wait status **0**, exit code **0**, no signal; the stopped lease survives unchanged. This is one cooperative sample, not a worst-case or hard scheduling guarantee. The existing supervisor stop/start follows the observed signal and retains the selected release. **No control I/O after cancel: not observable here**; this stream does not expose the Step cancellation / observation-POST completion boundary.

Across the assembled run, **three accepted hold samples / three incidents** carry `policy_missing` and `STALE_GENERATION` at `stage=lease`, with forwarding false, valid summaries (`steps=1`, `alarm=false`), and publication replay cause `stale_generation`. All end on `generation_matched`; **zero alarm reports and zero escalation audit events**. Alarm triggering and retained-safety precedence are **not exercised**, never passed by absence. [Cumulative final incidents, exact reports and lease IDs](F15-AT09-generation-holds.json).

**AT02 / wire-nodes 34 request 2:** its cumulative snapshot contains only the earlier bootstrap incident `01a196fe-5ac5-7a68-ade3-aebefed4bd4c`, report `01a196fe-5abe-7786-a1f4-2b254d4fbd74`, lease `5ce8d59c-9878-41c1-966a-624896cca11d` (applied 3 / lease 2, empty leased-member set). It ended before AT02 began. **No accepted stale-lease hold intersects AT02's revocations.** Sole-member `members_revoked` closure, removal of one member while an eligible sibling survives, and retained source-binding/TC priority during a hold are each **not exercised**. Fifteen original revocation/withdrawal audit rows retain resource/member IDs and surrounding member-state snapshots. Device/grant withdrawal is not member suspension/closure. [Central join and coverage](F15-AT02-central-hold-join.json), [revocation/member joins](F15-AT02-revocation-members.json).

The rows are consistent with a **node-wide** hold and the contract's absence of a per-member hold field; they do not prove the unexercised revocation combinations. Incident closure alone never claims service recovered. The two later samples (including one lease with eight members) likewise report node-wide forwarding false.

AT01 remains **BLOCKED**. Its original first selection has **five member SYNs, zero node/origin inner packets, zero origin requests**, with generation **5** and **six member accept rules** before and after. Captures are complete and zero-drop (**201 / 200 / 45** member/node/origin records). Cause is **unproven**; neither the node's sampled generation nor this failure locates the missing native hop. Devstack 25's campaign is consumed as evidence, not a repair on main. [Original attempt/readiness](F15-F11-01-first-selection.json), [native analysis](history/follow-up-15-captures/e80bb592-27ed-4ab5-a8c3-6433e7278938/initial-selection-analysis.json).

All **three D-077 re-grant attempts** pass this run. Every attempt retains its original readiness, member/node/origin packet observations, origin receipt and device counters. These are the captures the harness already made; detailed new WireGuard captures are not invented. `cause_status=unproven`; the earlier F14-01 remains failed and assigned to devstack's native investigation. A passing first selection or re-grant closes neither F11-01 nor F14-01. [All attempts](F15-D077-regrant-attempts.json).

AT04 retains **1,377 withdrawal samples with zero fresh parse-failure receipts**. Retention remains enabled: a rejection keeps `parse_failure` plus private raw/error files at mode 0600, never counter evidence and never published. Netlab 23's decision is consumed: every duplicate owned-chain definition, identical or not, is rejected as typed `DuplicateChainError`; its **0 / 3,300** lab result leaves origin unproven. Counterread's content-free `duplicate_chain` surface remains assigned to devstack's next follow-up. Both historical F13-01 failures stay failed, with no retry or predicate relaxation. [Fresh receipt index](F15-F13-01-private-receipts.json), [prior original receipt](history/follow-up-14/F14-F13-01-private-receipts.json).

New evidence larger than **20,000,000 bytes** is lossless gzip with SHA-256 and byte counts for both forms in `.gz.sha256.json`. The readers and hygiene scan decompress and verify both pins. AT02 JSON is **62,234,933 bytes** uncompressed and **3,371,553 bytes** compressed. [AT02.md](AT02.md) is a navigation document, separate from the pinned captures it links; the hygiene scan treats it as documentation. Historical archives remain byte-identical, including the original large files. [Compression contract and derivation](PROCEDURES.md#follow-up-15-evidence-and-regression-readback).

D-098 was the first, separate commit `90cf72cf`: all **70 root files** of run 258ba504 are byte-identical under [follow-up 14](history/follow-up-14/RUN.md), with [SHA-256 pins](history/follow-up-14/SHA256SUMS). Earlier archives are untouched. Handover claim tests read frozen evidence; no D-132 outside-owned pointer edit was needed.

[Run ledger and gates](RUN.md); [carried owner assignments](requests.md).

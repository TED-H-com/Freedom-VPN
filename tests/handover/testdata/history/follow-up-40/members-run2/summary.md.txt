# Follow-up 18 acceptance results — run 2

Run 2 `50117c68-d1b3-431d-a2be-cd94d27a2447`, measured harness `4eb5d7d32c449da7689e749fd8ecb208657937dd`, baseline `27ff63928222c9642e4f5e180cf225108327a1e5`. **2 PASS / 6 BLOCKED / 2 FAIL; 699 passed / 5 blocked / 4 failed subchecks.**

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 31 / 0 / 0 |
| [AT01](AT01.json) | APP01 ID01 MEM01 | blocked | 19 / 0 / 0 |
| [AT02](AT02.json.gz) | MEM01 MEM02 MEM03 SEC01 | fail | 191 / 0 / 2 |
| [AT03](AT03.json) | BILL01 BILL03 ENT01 | pass | 98 / 0 / 0 |
| [AT04](AT04.json) | BILL02 ENT02 MEM03 | blocked | 113 / 0 / 0 |
| [AT08](AT08.json) | NODE02 NODE03 EXIT01 EXIT03 | pass | 35 / 0 / 0 |
| [AT20](AT20.json) | DNS01 DNS02 ISO04 | blocked | 89 / 0 / 0 |
| [AT21](AT21.json) | DNS03 DNS04 APP01 | blocked | 13 / 0 / 0 |
| [AT22](AT22.json) | SEC02 DNS01 DNS04 | blocked | 40 / 0 / 0 |
| [AT09](AT09.json) | EXIT02 APP01 NFR01 | fail | 70 / 5 / 2 |

| Receive-only row | Selected family | Transfer / upload / evidence | Payload bytes / ms | Member SYN MSS sent → received | Origin SYN-ACK MSS sent → received | Reverse sent / received / max sent / max received IP bytes |
|---|---|---|---:|---|---|---:|
| [native dual MTU 1280](history/follow-up-18-captures/50117c68-d1b3-431d-a2be-cd94d27a2447/matrix-dual-1280-d146/analysis.json) | ipv4 | pass / pass / pass | 64064 / 1 | 1240 → 1160 | 1460 → 1160 | 73 / 73 / 1132 / 1132 |
| [native ipv6 MTU 1280](history/follow-up-18-captures/50117c68-d1b3-431d-a2be-cd94d27a2447/matrix-ipv6-1280-d146/analysis.json) | ipv6 | pass / pass / pass | 64064 / 1 | 1240 → 1160 | 1460 → 1160 | 74 / 74 / 1152 / 1152 |
| [native ipv4 MTU 1280](history/follow-up-18-captures/50117c68-d1b3-431d-a2be-cd94d27a2447/matrix-ipv4-1280-d146/analysis.json) | ipv4 | pass / pass / pass | 64064 / 1 | 1240 → 1160 | 1460 → 1160 | 72 / 72 / 1132 / 1132 |

The existing dual and IPv6 inputs and original byte/counter/family predicate are unchanged. The additional IPv4-only 1280 row makes IPv4 selection explicit independently of the dual row. The upload check uses the original bulk connection and requires 64,064 payload bytes after subtracting its 44-byte MEMBER greeting, with all 64 exact echoes proving acceptance at the origin. D-146 removes the D-137 downgrade only when every same-run qualification condition passes; an original failed transfer stays failed. Missing evidence or an advertisement already at/below 1160 cannot qualify the clamp. All three original analyses qualify: every selected reverse datagram matches at sender and member ingress, and both PTB types are absent at member, node and origin. Reverse payloads remain the fixture's 1006-byte echo writes; the endpoint MSS rewrites are the direct evidence that the clamp operated.

Receive-only UDP and QUIC without PTB remain documented limitations. The smaller-MSS control is **not exercised**: the unchanged fixture has no TCP_MAXSEG input; [Netlab 34's 1000 → 1000 control](../../../poc/G2-followup34-2026-11-03.md) stands. Clamp operation at 1280 on relay, real platforms, other encapsulations, IPv6 overlay and physical arm64 remain unverified. D-123 IPv6Block is unchanged.

AT09 records serving readiness **6.021246702 s**, counted recovery **12.548980167 s**, and desired generation **66 → 66**. [Every original AT09 subcheck](F18-AT09-subchecks.md), including D-067 and the combined restart, retains its measured status; [original restart records](F18-AT09-restart.json). These timing values alone identify no particular merged change from agent-core 37–43 (#429). Any narrower attribution requires the original records. Namespace destruction/recreation does not prove warm-kernel adoption. The D-067 record has 2 provisional observations, stage=lease error counts [0, 0], forwarding-block reasons [['safety_check_failed'], ['safety_check_failed']], and saw_lease_expired=False. Captured errors include [('apply', 'TC_FOREIGN_FILTER'), ('initial_quarantine', 'TC_FOREIGN_FILTER'), ('probe', 'HEALTH_CHECK_FAILED'), ('quarantine', 'TC_FOREIGN_FILTER'), ('tc_evidence', 'TC_FOREIGN_FILTER')]. These are original failure observations, not a causal attribution to a merged change.

AT02 retains **6250 sends**, **0 send errors**, **983 absent node ingresses**, and **259 ingress-without-egress losses**. All **1242** losses count. [Every original send](F18-AT02-every-send.json), [original recovery](F18-AT02-recovery.json.gz), [reservation joins](F18-pre-apply-reservations.json.gz). D-135/D-136 remain provisional; no loss waiver or finite outage ceiling is introduced. The original evaluator records measurement_valid=False, failed_apply_in_capture=3, failed_apply_bracketed=0, and cause_status=unproven.

AT01's original first selection is counted=False, HTTP status 0, origin receipt=False, with 0 origin requests. Its joined first-SYN transit is 5.654543001–5.654549869 s. [Original headers and observations](F18-loopback-and-first-selection.json) retain cause_status=unproven. F11-01 remains OPEN and upstream-dependent; later success closes neither F11-01 nor historical F14-01.

Orchestrator note 1 (2026-11-03 15:58 EST) authorized this second run after archiving run 1. Commit `4eb5d7d3` adds only the missing production retire phase to the reader and preserves its other validation. [Closed-list audit and regression tests](F18-reader-audit.md). D-146, all timeouts and the case order are unchanged. No third run is authorized or performed.

Run 1 `118271c1`, measured harness `b4317bd4`, remains **0 PASS / 3 BLOCKED / 7 FAIL; 442 / 0 / 35 subchecks**. [Archive commit 71dc60ac](history/follow-up-18-run1/INDEX.md) retains 72 byte-identical root files, gzip and pins, plus the original captures. The [retire rejection and 16 exact export-prefix joins](history/follow-up-18-run1/F18-lifecycle-phase-compatibility.json) prove the reader defect where joined; they do not explain every independent failure. Its [finite-grant restoration gap](history/follow-up-18-run1/F18-AT09-precondition.json), restart failures and unexercised D-146 rows remain recorded.

D-098 first commit `50f4403b` archives run 81b638e8's 77 root files byte-identically, including gzip and sidecars: [archive](history/follow-up-17/INDEX.md), [pins](history/follow-up-17/SHA256SUMS). No handover claims test reads a moved root file, so no D-132 pointer edit was needed. Earlier archive bytes remain unchanged. Historical e80bb592, 0ffb355a and 81b638e8 outcomes stay unchanged; their F8-02 statuses remain failed. [Historical MSS/small-write analysis](F18-historical-81b638e8-MSS.json) explains the 64 separate 1006-byte echo segments and 1132/1152-byte outer datagrams; it changes no historical verdict.

[All failed checks](F18-results.json), [D-146 evidence](F18-D146.json), [grant restoration and prerequisites](F18-AT09-precondition.json), [qualification](F18-qualification.json), [hold joins](F18-populated-hold-joins.json), [run/gates](RUN.md), [carried requests](requests.md). F11-01 stays **OPEN and upstream-dependent (D-148)**; no product change.

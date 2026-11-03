# Follow-up 18 acceptance results

Run `118271c1-616e-46a2-a744-d67b00fb94fb`, measured harness `b4317bd4e4ad178609b71d3fd4fbb986cb9947db`, baseline `27ff63928222c9642e4f5e180cf225108327a1e5` (#432). **0 PASS / 3 BLOCKED / 7 FAIL**, **442 passed / 0 blocked / 35 failed subchecks**. One assembled ten-case run; no live retry.

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 31 / 0 / 0 |
| [AT01](AT01.json) | APP01 ID01 MEM01 | blocked | 23 / 0 / 0 |
| [AT02](AT02.json.gz) | MEM01 MEM02 MEM03 SEC01 | fail | 136 / 0 / 11 |
| [AT03](AT03.json) | BILL01 BILL03 ENT01 | fail | 71 / 0 / 1 |
| [AT04](AT04.json) | BILL02 ENT02 MEM03 | fail | 51 / 0 / 9 |
| [AT08](AT08.json) | NODE02 NODE03 EXIT01 EXIT03 | fail | 31 / 0 / 3 |
| [AT20](AT20.json) | DNS01 DNS02 ISO04 | fail | 29 / 0 / 2 |
| [AT21](AT21.json) | DNS03 DNS04 APP01 | blocked | 6 / 0 / 0 |
| [AT22](AT22.json) | SEC02 DNS01 DNS04 | fail | 38 / 0 / 2 |
| [AT09](AT09.json) | EXIT02 APP01 NFR01 | fail | 26 / 0 / 7 |

D-098 first commit `50f4403b` archived all **77** run-81b638e8 root files, byte-identical including gzip and sidecars. [Frozen archive](history/follow-up-17/INDEX.md), [SHA-256 pins](history/follow-up-17/SHA256SUMS). Earlier archives are untouched. No handover claims test reads a moved living-root file, so no D-132 pointer change was needed.

| Receive-only row | Selected underlay | Transfer / upload / evidence | Bytes / ms | Member SYN MSS sent → received | Origin SYN-ACK MSS sent → received | Reverse sent / received / largest IP bytes |
|---|---|---|---:|---|---|---:|
| native dual MTU 1280 | not observed | not exercised / not exercised / unavailable | — | — | — | — |
| native ipv6 MTU 1280 | not observed | not exercised / not exercised / unavailable | — | — | — | — |
| native ipv4 MTU 1280 | not observed | not exercised / not exercised / unavailable | — | — | — | — |

D-146 qualifies only the measured TCP rows with all evidence in the same run. The original bulk byte/counter/family predicate is unchanged. The added upload check requires 64,064 payload bytes on the same fresh connection after subtracting the 44-byte greeting, plus all 64 exact echoes proving origin acceptance. It was not exercised in this campaign. Missing or ambiguous evidence retains D-137 blocked; a failed original byte predicate stays failed. Receive-only UDP/QUIC without PTB remain documented limitations. Smaller-MSS control is **not exercised**: the unchanged bulk helper has no TCP_MAXSEG input; netlab 34's 1000 → 1000 control stands. Relay at 1280, real platforms, extra encapsulations and physical arm64 remain unverified. D-123 IPv6Block is unchanged.

An additional IPv4-only 1280 row was added alongside the unchanged dual and IPv6 rows. **All three are unexercised in this run:** AT09 returned at the admitted-device prerequisite before applying any matrix input. The last AT02 finite grant ended at 20:01:12 UTC; readiness failed and its early return bypassed the normal restoration. AT09 later returned HTTP 200 with zero visible exits. [Original member-ID and source-path join](F18-AT09-precondition.json). No new endpoint MSS, underlay family, outer-size, no-PTB or bidirectional transfer result exists for these rows. The [D-146 inventory](F18-D146.json) records the missing rows explicitly. The evidence collector remains committed but has no live qualification from this campaign.

AT09's original D-067 and combined restart predicates both fail. The capture records serving readiness **6.021907659 s**, but desired generation changed **43 → 44** and there is **no counted recovery**. The original zero timestamp/seconds are unset sentinels, not instantaneous recovery. The local fresh-serving check and complete same-generation capture check also fail. [Every AT09 subcheck](F18-AT09-subchecks.md) retains its original status. The records include TC_FOREIGN_FILTER, a failed fvmb0 qdisc read and incomplete quarantine; they identify no particular change from agent-core 37–43 (#429), so no causal attribution is made. The namespace is destroyed/recreated; this is not a warm-kernel adoption proof.

AT02 retains **10265 sends**, **0 send errors**, **985 absent node ingresses** and **257 ingress-without-egress losses**. All losses count; none is excluded by a guard or baseline label. [Original recovery](F18-AT02-recovery.json), [every-send join](F18-AT02-every-send.json), [reservation brackets](F18-pre-apply-reservations.json). D-135/D-136 remain unchanged; no finite outage ceiling or availability waiver.

[All original failed checks and case counts](F18-results.json), [member and hold qualification](F18-qualification.json), [incident/report/audit joins](F18-populated-hold-joins.json), [re-grant attempts](F18-D077-regrant-attempts.json), [first selection](F18-loopback-and-first-selection.json). F11-01 stays **OPEN, upstream-dependent** under D-148; no product, pin or architecture change.

Historical e80bb592, 0ffb355a and 81b638e8 outcomes remain unchanged. [Additive analysis of 81b638e8](F18-historical-81b638e8-MSS.json) records unchanged MSS 1240 / 1460 and 64 separate 1006-byte reverse echo payloads: inner 1058 bytes pads to 1072, giving outer 1132 / 1152 for IPv4 / IPv6. That small-write pattern explains finite success without a clamp; no earlier loss is attributed or erased.

[Run, host state and gates](RUN.md), [carried requests](requests.md), [all AT09 subchecks](F18-AT09-subchecks.md).


The measured lifecycle consumer rejects producer command 28, phase `retire`, at sequence 582. The [original byte-pinned join](F18-lifecycle-phase-compatibility.json) retains the command and **16 exact export-prefix matches**. Its whitelist lacks this production phase. The original rejected trace is unavailable, not a record of zero Applies. Supplemental raw-export joins preserve the observations but never repair the original D-135/D-136 verdict. This does not by itself explain every final central-readiness failure or packet loss.

The [same-member precondition join](F18-AT09-precondition.json) retains the last AT02 grant (`valid_until=2026-11-03T20:01:12Z`), subsequent failed readiness checks, absence of the normal restoration, and AT09's HTTP 200 / zero-exit result. The measured `finiteExpiry` source returns on a failed positive probe before restoring the grant. AT09 records the admitted-device prerequisite blocker and performs no native matrix row. No family, MSS, outer-size, no-PTB or bidirectional-byte result is available for dual/IPv6/IPv4 1280. This is a coverage gap, not a measured clamp failure.

The run reached its existing 70-minute context limit during the final AT09 prerequisite/readback work. Its final central hold join failed with unavailable readback; all ten case records are nevertheless marked complete by the original runner. No timeout, input or predicate was lengthened, no restoration was injected, and no run was retried. The four controlled populated-hold scenarios are **not exercised**. The separate cumulative AT02 incident does not substitute for them.

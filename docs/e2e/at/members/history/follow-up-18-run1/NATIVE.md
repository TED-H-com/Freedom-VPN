# Follow-up 18 native evidence

| Receive-only row | Selected underlay | Transfer / upload / evidence | Bytes / ms | Member SYN MSS sent → received | Origin SYN-ACK MSS sent → received | Reverse sent / received / largest IP bytes |
|---|---|---|---:|---|---|---:|
| native dual MTU 1280 | not observed | not exercised / not exercised / unavailable | — | — | — | — |
| native ipv6 MTU 1280 | not observed | not exercised / not exercised / unavailable | — | — | — | — |
| native ipv4 MTU 1280 | not observed | not exercised / not exercised / unavailable | — | — | — | — |

D-146 qualifies only the measured TCP rows with all evidence in the same run. The original bulk byte/counter/family predicate is unchanged. The added upload check requires 64,064 payload bytes on the same fresh connection after subtracting the 44-byte greeting, plus all 64 exact echoes proving origin acceptance. It was not exercised in this campaign. Missing or ambiguous evidence retains D-137 blocked; a failed original byte predicate stays failed. Receive-only UDP/QUIC without PTB remain documented limitations. Smaller-MSS control is **not exercised**: the unchanged bulk helper has no TCP_MAXSEG input; netlab 34's 1000 → 1000 control stands. Relay at 1280, real platforms, extra encapsulations and physical arm64 remain unverified. D-123 IPv6Block is unchanged.

[Full D-146 index](F18-D146.json), [all native original checks](F18-D137-native-paths.json), [historical small-packet analysis](F18-historical-81b638e8-MSS.json), [follow-up 17 narrative](history/follow-up-17/NATIVE.md).


The measured lifecycle consumer rejects producer command 28, phase `retire`, at sequence 582. The [original byte-pinned join](F18-lifecycle-phase-compatibility.json) retains the command and **16 exact export-prefix matches**. Its whitelist lacks this production phase. The original rejected trace is unavailable, not a record of zero Applies. Supplemental raw-export joins preserve the observations but never repair the original D-135/D-136 verdict. This does not by itself explain every final central-readiness failure or packet loss.

The [same-member precondition join](F18-AT09-precondition.json) retains the last AT02 grant (`valid_until=2026-11-03T20:01:12Z`), subsequent failed readiness checks, absence of the normal restoration, and AT09's HTTP 200 / zero-exit result. The measured `finiteExpiry` source returns on a failed positive probe before restoring the grant. AT09 records the admitted-device prerequisite blocker and performs no native matrix row. No family, MSS, outer-size, no-PTB or bidirectional-byte result is available for dual/IPv6/IPv4 1280. This is a coverage gap, not a measured clamp failure.

The run reached its existing 70-minute context limit during the final AT09 prerequisite/readback work. Its final central hold join failed with unavailable readback; all ten case records are nevertheless marked complete by the original runner. No timeout, input or predicate was lengthened, no restoration was injected, and no run was retried. The four controlled populated-hold scenarios are **not exercised**. The separate cumulative AT02 incident does not substitute for them.

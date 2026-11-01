# af718646: re-grant timeout with serving intact

This fresh IPv6 D-077 re-grant request fails after valid readiness under **passive native capture, with no node or member nft tracer on the re-grant path**. It reproduces the deadline-window symptom separately from the first-selection observer closure. It does not establish the cause of historical run 258ba504, whose preceding full members population is different.

The fence was inserted at 12:50:29.153887690Z, withdrawal observed at 12:50:40.907016570Z, and explicit authorization requested at 12:50:40.973831210Z. The same-key replay returned the same operation. Populated serving readiness passed at **12:50:43.393539572Z**, generation **7**, and the second fresh-heartbeat gate passed at **12:50:48.217481565Z**. The generation-7 lease had one member and expired the following day. The node had already recovered from the earlier first-selection tracer before fencing began.

| Event (UTC) | Native and central evidence |
|---|---|
| Before fencing | Both peers connected with completed old handshakes. |
| After withdrawal | Both NetBird peers and matching WireGuard peers absent. |
| Before first request | Peers/configuration return, WireGuard handshake is 0 on both sides, NetBird connected=false. The selected origin route uses member `fvnb0`; both WireGuard endpoint routes still use loopback. |
| 12:50:50.510393 | First member SYN, packet 306. Five SYNs are emitted during the request. |
| 12:50:50.510492 → 50.714347 | Type-1 message hash `63527caa…94bea` joins member loopback to node loopback, sender index 4200933091. |
| 12:50:50.714513 → 50.714518 | Matching type-2 receiver index 4200933091, hash `987558f4…d04d2e2`, leaves node `eth2` and reaches member `eth1` with an IPv6 loopback source. |
| During the original 5.003-second request | No matching inner packet at node or origin. Member `Ip6InHdrErrors` increments 0 → 1; node stays 0. Both kernel handshake samples remain 0. NetBird becomes connected and endpoints move to native IPv6 while the kernel handshake remains incomplete. |
| At timeout | Both device byte counters are unchanged: upload 2409, download 2463; kernel generation remains 7 and the sampled heartbeat remains serving=true. No failed Apply or baseline writer event follows the re-grant serving gate. |
| 12:50:56.180953 → 56.181211 | A later native IPv6 type-1/type-2 exchange has no loopback source; the final peer samples record completed handshakes at Unix second 1793537456. |
| 12:50:56.181478–56.181579 | The five SYNs finally arrive at node `fvmb0`. The first matched SYN transit bound is **[5,671.079, 5,671.087] ms**, beyond the HTTP deadline. |
| 12:50:56.181539–56.181844 | The origin sees late SYN/SYN-ACK and reset packets (six captured packets), but receives no HTTP request. These late packets do not turn the original failed request into a pass. |

The producer's post-SYN live reads and Applies all succeed at generation 7 (sequences 123–138), with serving heartbeats. The earlier `policy_missing` baselines belong to fence/re-grant convergence before readiness; they are not post-readiness closures. All accepted central reports and checks remain available in [central-record.json](central-record.json).

**Supported localization:** the original deadline is consumed before native WireGuard handshake completion and member-data delivery, across the loopback-to-IPv6 endpoint transition. The loopback-source response and member IPv6 header-error increment are captured together. This passive experiment does not trace the exact kernel discard callsite or authenticate the structural WireGuard headers, and it does not prove the same cause for 258ba504. No grant repair, Apply retry, NAT workaround, kernel change or NetBird pin change is justified from this record alone.

**Owner hand-off:** NetBird/native transport integration and the members acceptance owner should assess this native handshake/endpoint transition using the original full F14-01 population. The F11-01 transport option remains Ted's decision; this separate failure is not assigned the tracer-closure cause. See [original re-grant record](regrant-session.json), [all peer/route/handshake samples](session-intervals.json), and [packet/writer joins](serving-packet-joins.json).

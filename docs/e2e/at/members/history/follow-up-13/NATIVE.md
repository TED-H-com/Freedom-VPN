# Native paths — Follow-up 13

AT09 verdict: **BLOCKED under provisional D-137**. Run `4182dd3d`,
clean harness `d49b535a`, product baseline `de6d2e02`. IPv4 overlay payloads;
D-123 IPv6Block remains. Underlay family is reported independently.

| Row | Selected underlay | Bytes / requested | ms | Verdict |
|---|---|---:|---:|---|
| IPv4 / 1500 | IPv4 P2P | 64,064 / 64,064 | 1 | pass |
| IPv6 / 1500 | IPv6 P2P | 64,064 / 64,064 | 1 | pass |
| Dual / 1500 | IPv4 P2P | 64,064 / 64,064 | 1 | pass |
| Dual / receive-only 1280 | IPv4 P2P | 64,064 / 64,064 | 1 | limitation, blocked; never PASS |
| IPv6 / receive-only 1280 | IPv6 P2P | 64,064 / 64,064 | 1 | limitation, blocked; never PASS |
| Selected-native-peer known PMTU 1280 | IPv6 P2P | 64,064 / 64,064 | 1 | supported row pass under provisional D-137 |
| Reactive PTB router hop | unavailable on this baseline | — | — | pending devstack 23 |

**AT09 verdict: BLOCKED under provisional D-137.** Known-PMTU 1280 completes **64,064 bytes in 1 ms**, with route metric 1280 before/after traffic, exact removal and **11 delivered reverse IPv6 fragment pairs**. Reactive PTB remains pending devstack 23 on this tested baseline. Both receive-only 1280 rows also transfer **64,064 bytes in 1 ms**, but remain explicit **PMTU black-hole limitation rows, never PASS**. Reverse data packets are only 1,132 bytes on IPv4 / 1,152 bytes on IPv6, so oversized loss was not reproduced; no PTB appears. Historical F8-02 failures stay failed. [Separate rows and captures](F13-D137-native-paths.json).

[Eight complete capture sessions](history/follow-up-13-captures/4182dd3d-7f1f-453e-8c1f-265d097c3abc/README.md)
have 24 zero-drop namespace captures. Known-PMTU sender fragmentation is measured;
reactive PMTU is not. No global MTU/MSS clamp or production fix was introduced.
The two successful receive-only probes send smaller reverse data packets, so this
run does not reproduce historical oversized reverse loss and cannot close F8-02.

The complete original restart preserves generation 30, fresh serving at
15.850473740 s and first counted byte at 21.440684604 s. See
[fresh restart capture](F13-AT09-restart.json). Five platform/QUIC selectors remain
blocked with their original resource reasons; physical roaming/NAT and end-to-end
IPv6 PMTU are unclaimed. AT01 is a separate successful first-selection sample;
[F11-01 comparison](F13-F11-01-forwarding.json) does not transfer devstack's cause.


The lane's latest preserved receive-only failure remains **13,013 / 64,064 bytes
in 12,010 ms**, `bulk_echo_incomplete`, in [run 7edb263f](history/follow-up-12/AT09.json).
Separately, [devstack 22's owner run](../../runs/2026-10-31/followup22/9b3a89e7-fa86-426c-9737-ce55bf66aa77/analysis.md)
records **16,016 / 64,064 bytes in 12,011 ms**, with 15 reverse 1,360-byte
IPv6 packets lost and no PTB. Those historical failed rows remain failed;
the fresh smaller-packet transfers do not replace them or transfer their cause.

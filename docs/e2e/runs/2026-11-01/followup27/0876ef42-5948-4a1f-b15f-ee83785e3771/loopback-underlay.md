# IPv6 data-source event, separate from handshake retry

2026-11-01. [Complete joins](loopback-underlay.json), [handshake / endpoint timeline](transport-timeline.json), and [all stream intervals](stream-intervals.csv).

The first selection passed (3.534703–3.559238 ms from member tunnel egress to node ingress). All 7,297 stream sends reached node ingress. Eight subsequent node-to-member **type 4 data** messages had source `::1` on the IPv6 underlay. Each opaque message hash joins node `eth2` egress to member `eth1` ingress. The member IPv6 header-error counter increased by eight in four bounded sampling windows. No type 2 response with a loopback source crossed the underlay in this run.

| WireGuard counter | Node packet / Unix seconds | Member packet / Unix seconds | Hash prefix |
|---:|---|---|---|
| 9 | 314 / 1793547190.447274 | 648 / 1793547190.447281 | `89d276a8bced8ed0` |
| 10 | 322 / 1793547190.467754 | 660 / 1793547190.467758 | `9bd2b204defc6574` |
| 11 | 329 / 1793547190.488340 | 664 / 1793547190.488344 | `d5bbeea8515fa95f` |
| 12 | 337 / 1793547190.509032 | 670 / 1793547190.509042 | `3484104d3f8c3bfc` |
| 13 | 344 / 1793547190.529118 | 674 / 1793547190.529126 | `cd0bcfb146cee597` |
| 14 | 352 / 1793547190.549714 | 679 / 1793547190.549717 | `d39640523293e7af` |
| 15 | 553 / 1793547191.454083 | 860 / 1793547191.454091 | `368b797bb9899442` |
| 16 | 782 / 1793547192.457961 | 1071 / 1793547192.457975 | `868482b09c4fdae6` |

All eight data headers use receiver index `1448874228`, matching the sender index of the earlier member response `21fd4c5cfe8b561fc57647792b019257319e6c24bbab5e383d80d1bad56b326b` at Unix `1793547187.602114` on loopback. This is a structural key-index join, not cryptographic authentication by the observer.

The node WireGuard endpoint is already IPv6 in `after-passive-tail` (15:33:09.750616442Z); the member WireGuard endpoint remains loopback until `stream-9` (15:33:12.79438189Z). NetBird reports P2P at both earlier samples. At Unix `1793547193.463370`, node packet 957 / counter 17 uses the expected non-loopback node IPv6 source token `addr-837a7935ba5539b1`; member packet 1235 receives the same bytes 11 µs later. No handshake initiation or response occurs between these data messages. The next captured handshake is the node-initiated IPv6 rekey at Unix `1793547308.466707`.

The evidence associates the source anomaly with the asymmetric endpoint handover and bounded header-error increments. It does not identify a kernel drop callsite or establish the cause of the source selection. There is no member→node ingress stall or five-second handshake retry here. The encrypted inner data is not decoded or attributed to a particular application reply. Historical F11-01, AT02 and F14-01 causes remain unchanged. No repair, reconnect, NAT, kernel, probe or NetBird-pin change was made.

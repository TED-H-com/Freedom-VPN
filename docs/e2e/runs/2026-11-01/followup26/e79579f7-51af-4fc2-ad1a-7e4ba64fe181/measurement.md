# e79579f7-51af-4fc2-ad1a-7e4ba64fe181

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:59:29.310889965Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [104.631, 104.635] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

No baseline writer bracket in this capture (the first campaign attempt predates producer export).

Native samples: 11. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T13:00:13.881799435Z; accepted generation 8. Kernel counter generation before → after: 8 → 8.

First matching SYN member-egress → node-ingress interval: [103.923, 103.927] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 114 | policy_missing | true | -17746.716 … -17706.928 |
| 115 | policy_missing | true | -17701.480 … -17672.682 |
| 116 | policy_missing | true | -17657.319 … -17630.447 |
| 134 | policy_missing | true | -7743.539 … -7709.502 |
| 135 | policy_missing | true | -7704.735 … -7677.023 |
| 136 | policy_missing | true | -7659.884 … -7628.804 |

Native samples: 30. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:59:39.68018448Z → 2026-11-01T12:59:58.818712086Z, accepted generation 6. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

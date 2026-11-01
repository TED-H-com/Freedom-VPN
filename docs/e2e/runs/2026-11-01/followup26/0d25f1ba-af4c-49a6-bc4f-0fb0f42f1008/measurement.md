# 0d25f1ba-af4c-49a6-bc4f-0fb0f42f1008

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:57:15.805021306Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [105.038, 105.043] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

No baseline writer bracket in this capture (the first campaign attempt predates producer export).

Native samples: 11. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:57:55.748499296Z; accepted generation 8. Kernel counter generation before → after: 8 → 8.

First matching SYN member-egress → node-ingress interval: [203.521, 203.527] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 105 | policy_missing | true | -13068.061 … -13026.682 |
| 106 | policy_missing | true | -13022.436 … -12993.950 |
| 107 | policy_missing | true | -12976.031 … -12941.760 |
| 117 | policy_missing | true | -8072.493 … -8034.985 |
| 118 | policy_missing | true | -8031.389 … -8003.118 |
| 119 | policy_missing | true | -7986.048 … -7957.044 |

Native samples: 25. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:57:25.987188599Z → 2026-11-01T12:57:46.118199695Z, accepted generation 6. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

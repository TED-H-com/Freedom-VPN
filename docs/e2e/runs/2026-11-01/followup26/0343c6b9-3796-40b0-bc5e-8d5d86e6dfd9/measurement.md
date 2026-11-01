# 0343c6b9-3796-40b0-bc5e-8d5d86e6dfd9

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:55:06.456662305Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [3.262, 3.266] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

No baseline writer bracket in this capture (the first campaign attempt predates producer export).

Native samples: 11. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:55:36.752647217Z; accepted generation 9. Kernel counter generation before → after: 9 → 9.

First matching SYN member-egress → node-ingress interval: [3.230, 3.236] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 102 | policy_missing | true | -18252.014 … -18218.965 |
| 103 | policy_missing | true | -18215.982 … -18184.739 |
| 104 | policy_missing | true | -18169.418 … -18146.048 |
| 114 | policy_missing | true | -13249.169 … -13216.797 |
| 115 | policy_missing | true | -13213.447 … -13183.165 |
| 116 | policy_missing | true | -13167.939 … -13146.135 |
| 126 | policy_missing | true | -8249.823 … -8215.817 |
| 127 | policy_missing | true | -8212.326 … -8189.869 |
| 128 | policy_missing | true | -8174.811 … -8140.745 |

Native samples: 30. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:55:16.76251655Z → 2026-11-01T12:55:21.874953043Z, accepted generation 6. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

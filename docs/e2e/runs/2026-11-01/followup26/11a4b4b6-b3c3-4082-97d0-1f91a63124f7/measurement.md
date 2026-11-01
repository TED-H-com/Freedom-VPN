# 11a4b4b6-b3c3-4082-97d0-1f91a63124f7

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:39:08.930368447Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [105.119, 105.160] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:39:13.149112+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 905.040000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | 845.406 … 879.196 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | 883.200 … 904.971 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:39:39.264840817Z; accepted generation 8. Kernel counter generation before → after: 8 → 8.

First matching SYN member-egress → node-ingress interval: [105.675, 105.679] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 85 | policy_missing | true | -23253.436 … -23212.980 |
| 86 | policy_missing | true | -23209.187 … -23182.216 |
| 87 | policy_missing | true | -23166.829 … -23142.825 |
| 105 | policy_missing | true | -13248.032 … -13199.576 |
| 106 | policy_missing | true | -13196.241 … -13172.227 |
| 107 | policy_missing | true | -13156.278 … -13130.967 |
| 117 | policy_missing | true | -8244.269 … -8204.527 |
| 118 | policy_missing | true | -8200.935 … -8176.531 |
| 119 | policy_missing | true | -8160.409 … -8135.113 |

Native samples: 35. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:39:15.874796136Z → 2026-11-01T12:39:18.979583324Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

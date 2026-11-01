# c1f20cc7-a51c-48dc-9c55-e538603169b5

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:29:58.973902222Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [105.385, 105.389] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:30:03.320597+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 841.426000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | 774.581 … 810.994 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | 814.900 … 841.317 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:30:24.154989622Z; accepted generation 7. Kernel counter generation before → after: 7 → 7.

First matching SYN member-egress → node-ingress interval: [203.902, 203.907] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 85 | policy_missing | true | -17983.761 … -17952.063 |
| 86 | policy_missing | true | -17948.594 … -17923.807 |
| 87 | policy_missing | true | -17909.365 … -17885.036 |
| 105 | policy_missing | true | -7985.138 … -7954.751 |
| 106 | policy_missing | true | -7951.381 … -7924.364 |
| 107 | policy_missing | true | -7909.879 … -7879.344 |

Native samples: 30. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:30:06.037832384Z → 2026-11-01T12:30:09.145340246Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

# 528e1feb-9898-4405-9396-dd8bc69d87de

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:43:32.320844767Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [3.441, 3.445] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:43:36.438985+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 711.436000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | 632.730 … 675.019 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | 681.058 … 711.335 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:43:57.463224677Z; accepted generation 7. Kernel counter generation before → after: 7 → 7.

First matching SYN member-egress → node-ingress interval: [3.515, 3.521] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 85 | policy_missing | true | -18275.175 … -18234.039 |
| 86 | policy_missing | true | -18229.930 … -18204.753 |
| 87 | policy_missing | true | -18189.011 … -18151.677 |
| 105 | policy_missing | true | -8278.597 … -8248.235 |
| 106 | policy_missing | true | -8244.207 … -8219.287 |
| 107 | policy_missing | true | -8203.185 … -8178.478 |

Native samples: 30. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:43:39.276153876Z → 2026-11-01T12:43:42.385605293Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

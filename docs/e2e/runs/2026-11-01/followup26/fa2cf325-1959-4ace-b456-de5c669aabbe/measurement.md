# fa2cf325-1959-4ace-b456-de5c669aabbe

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:41:12.314603625Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [3.822, 3.826] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:41:17.075776+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 1472.019000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 53 | verification_ISOLATION_POLICY_INVALID | true | 1398.658 … 1437.728 |
| 54 | verification_ISOLATION_POLICY_INVALID | true | 1447.315 … 1471.946 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:41:38.602025159Z; accepted generation 7. Kernel counter generation before → after: 7 → 7.

First matching SYN member-egress → node-ingress interval: [4.657, 4.664] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 65 | policy_missing | true | -17619.860 … -17577.119 |
| 66 | policy_missing | true | -17573.764 … -17538.645 |
| 67 | policy_missing | true | -17523.639 … -17495.756 |
| 85 | policy_missing | true | -7596.227 … -7543.794 |
| 86 | policy_missing | true | -7539.373 … -7494.965 |
| 87 | policy_missing | true | -7474.339 … -7437.486 |
| 105 | policy_missing | true | 2393.975 … 2437.031 |
| 106 | policy_missing | true | 2440.735 … 2477.633 |
| 107 | policy_missing | true | 2494.362 … 2525.325 |

Native samples: 29. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:41:19.119213554Z → 2026-11-01T12:41:23.225537788Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

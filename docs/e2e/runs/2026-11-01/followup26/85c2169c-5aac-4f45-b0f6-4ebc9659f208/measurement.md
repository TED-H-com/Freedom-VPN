# 85c2169c-5aac-4f45-b0f6-4ebc9659f208

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:32:21.512164912Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [4.781, 4.785] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:32:25.868794+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 922.777000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | 837.354 … 876.792 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | 886.670 … 922.692 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:32:46.671609692Z; accepted generation 7. Kernel counter generation before → after: 7 → 7.

First matching SYN member-egress → node-ingress interval: [5.032, 5.037] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 85 | policy_missing | true | -18009.238 … -17968.982 |
| 86 | policy_missing | true | -17965.438 … -17929.033 |
| 87 | policy_missing | true | -17913.903 … -17888.437 |
| 105 | policy_missing | true | -8005.850 … -7976.792 |
| 106 | policy_missing | true | -7973.311 … -7947.495 |
| 107 | policy_missing | true | -7931.672 … -7900.652 |

Native samples: 30. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:32:28.400655902Z → 2026-11-01T12:32:31.518893311Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

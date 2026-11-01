# 65260e25-8aa5-4e7c-994f-51525514e801

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:52:51.289371773Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [5.481, 5.486] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:52:56.320339+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 1629.870000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | 1543.611 … 1585.091 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | 1595.495 … 1629.773 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:53:17.51849868Z; accepted generation 7. Kernel counter generation before → after: 7 → 7.

First matching SYN member-egress → node-ingress interval: [3.511, 3.517] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 85 | policy_missing | true | -18362.411 … -18323.428 |
| 86 | policy_missing | true | -18320.487 … -18296.682 |
| 87 | policy_missing | true | -18281.613 … -18255.484 |
| 105 | policy_missing | true | -8364.504 … -8331.801 |
| 106 | policy_missing | true | -8328.701 … -8300.320 |
| 107 | policy_missing | true | -8283.906 … -8256.985 |

Native samples: 30. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:52:58.400433016Z → 2026-11-01T12:53:02.51497275Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

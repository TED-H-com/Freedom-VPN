# 86153543-1251-48f4-b368-cf11e5b0939d

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **fail**. Member SYNs / node ingress SYNs / origin receipts: 5 / 5 / 0.

This is the explicitly delayed historical-tracer fault probe, not first-selection acceptance.

Admission gate: pass at 2026-11-01T13:07:37.952889164Z; accepted generation 5. Kernel counter generation before → after: 0 → 0.

First matching SYN member-egress → node-ingress interval: [104.785, 104.796] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T13:07:42.701216+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | -11272.753000 |
| 2026-11-01T13:07:47.789143+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply, ISOLATION_POLICY_INVALID / apply_candidate | -6184.826000 |
| 2026-11-01T13:07:52.809862+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply, ISOLATION_POLICY_INVALID / apply_candidate | -1164.107000 |
| 2026-11-01T13:07:58.21688+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply, ISOLATION_POLICY_INVALID / apply_candidate | 4242.911000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | -11356.729 … -11307.967 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | -11298.422 … -11272.822 |
| 78 | apply_rollback | true | -6275.319 … -6263.899 |
| 79 | verification_ISOLATION_POLICY_INVALID | true | -6245.278 … -6220.363 |
| 80 | verification_ISOLATION_POLICY_INVALID | true | -6210.589 … -6184.922 |
| 84 | apply_rollback | true | -1279.341 … -1257.586 |
| 85 | verification_ISOLATION_POLICY_INVALID | true | -1238.029 … -1205.991 |
| 86 | verification_ISOLATION_POLICY_INVALID | true | -1195.563 … -1164.238 |
| 90 | apply_rollback | true | 4152.791 … 4163.295 |
| 91 | verification_ISOLATION_POLICY_INVALID | true | 4183.487 … 4209.277 |
| 92 | verification_ISOLATION_POLICY_INVALID | true | 4212.910 … 4242.820 |

Native samples: 38. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

counted initial path unavailable; no re-grant attempted

## After observer removal

Recovery gate: **pass**, 2026-11-01T13:08:02.599870796Z → 2026-11-01T13:08:03.708677353Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

# 68591f0c-0fa0-414f-8187-590d8ce11bc5

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T11:56:42.620731749Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [3.854, 3.858] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T11:56:46.643016+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 470.740000 |

These are report times, not exact kernel close instants.

No baseline writer bracket in this capture (the first campaign attempt predates producer export).

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T11:56:57.740026437Z; accepted generation 8. Kernel counter generation before → after: 8 → 8.

First matching SYN member-egress → node-ingress interval: [5.211, 5.215] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T11:57:01.667769+00:00 | 8 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 1349.899000 |

These are report times, not exact kernel close instants.

No baseline writer bracket in this capture (the first campaign attempt predates producer export).

Native samples: 10. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

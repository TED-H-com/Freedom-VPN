# ce8ccbe1-ed7d-4fc0-b1ed-786ef0e7578c

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **fail**. Member SYNs / node ingress SYNs / origin receipts: 5 / 5 / 0.

Admission gate: pass at 2026-11-01T12:48:18.542313158Z; accepted generation 5. Kernel counter generation before → after: 5 → 0.

First matching SYN member-egress → node-ingress interval: [10443.666, 10443.671] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:48:23.604196+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 1738.063000 |
| 2026-11-01T12:48:28.720218+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply, ISOLATION_POLICY_INVALID / apply_candidate | 6854.085000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | 1650.453 … 1704.071 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | 1707.811 … 1737.964 |
| 78 | apply_rollback | true | 6743.462 … 6765.123 |
| 79 | verification_ISOLATION_POLICY_INVALID | true | 6784.212 … 6826.047 |
| 80 | verification_ISOLATION_POLICY_INVALID | true | 6829.444 … 6854.007 |

Native samples: 26. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

counted initial path unavailable; no re-grant attempted

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:48:34.358377123Z → 2026-11-01T12:48:38.950282588Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

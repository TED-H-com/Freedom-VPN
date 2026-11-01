# a3154028-4296-4a61-afba-fee20fc0aa5f

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **fail**. Member SYNs / node ingress SYNs / origin receipts: 5 / 5 / 0.

This is the explicitly delayed historical-tracer fault probe, not first-selection acceptance.

Admission gate: pass at 2026-11-01T13:09:21.015726709Z; accepted generation 5. Kernel counter generation before → after: 0 → 0.

First matching SYN member-egress → node-ingress interval: [4.293, 4.300] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T13:09:24.787207+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | -12279.668000 |
| 2026-11-01T13:09:29.901644+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply, ISOLATION_POLICY_INVALID / apply_candidate | -7165.231000 |
| 2026-11-01T13:09:34.905849+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply, ISOLATION_POLICY_INVALID / apply_candidate | -2161.026000 |
| 2026-11-01T13:09:39.973498+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply, ISOLATION_POLICY_INVALID / apply_candidate | 2906.623000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 44 | verification_ISOLATION_POLICY_INVALID | true | -12349.132 … -12315.910 |
| 45 | verification_ISOLATION_POLICY_INVALID | true | -12306.498 … -12279.728 |
| 49 | apply_rollback | true | -7255.296 … -7243.915 |
| 50 | verification_ISOLATION_POLICY_INVALID | true | -7227.303 … -7198.760 |
| 51 | verification_ISOLATION_POLICY_INVALID | true | -7188.960 … -7165.294 |
| 55 | apply_rollback | true | -2272.901 … -2253.678 |
| 56 | verification_ISOLATION_POLICY_INVALID | true | -2232.339 … -2200.103 |
| 57 | verification_ISOLATION_POLICY_INVALID | true | -2190.351 … -2161.122 |
| 61 | apply_rollback | true | 2740.838 … 2756.370 |
| 62 | verification_ISOLATION_POLICY_INVALID | true | 2778.436 … 2808.769 |
| 63 | verification_ISOLATION_POLICY_INVALID | true | 2876.047 … 2906.492 |

Native samples: 38. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

counted initial path unavailable; no re-grant attempted

## After observer removal

Recovery gate: **pass**, 2026-11-01T13:09:45.653870107Z → 2026-11-01T13:09:50.174077862Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

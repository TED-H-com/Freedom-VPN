# 2d9dc8ee-50ca-4ad1-9ed9-f077041f2f64

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:27:36.668660623Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [104.674, 104.698] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:27:40.868664+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 881.074000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 52 | verification_ISOLATION_POLICY_INVALID | true | 800.564 … 845.971 |
| 53 | verification_ISOLATION_POLICY_INVALID | true | 856.111 … 881.004 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:28:06.963581273Z; accepted generation 9. Kernel counter generation before → after: 9 → 9.

First matching SYN member-egress → node-ingress interval: [104.792, 104.800] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 64 | policy_missing | true | -23248.250 … -23215.525 |
| 65 | policy_missing | true | -23212.615 … -23187.048 |
| 66 | policy_missing | true | -23172.368 … -23144.466 |
| 84 | policy_missing | true | -13241.975 … -13215.062 |
| 85 | policy_missing | true | -13211.670 … -13189.871 |
| 86 | policy_missing | true | -13174.920 … -13145.321 |
| 96 | policy_missing | true | -8241.019 … -8208.326 |
| 97 | policy_missing | true | -8204.842 … -8182.148 |
| 98 | policy_missing | true | -8167.106 … -8140.779 |

Native samples: 35. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:27:43.664883964Z → 2026-11-01T12:27:46.776228213Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

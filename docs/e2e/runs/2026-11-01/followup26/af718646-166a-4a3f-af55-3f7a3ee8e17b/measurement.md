# af718646-166a-4a3f-af55-3f7a3ee8e17b

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T12:50:18.161212225Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [106.846, 106.850] ms.

| Accepted report observed time | Applied generation | Block reasons | Error code / stage | Relative to first SYN (ms) |
|---|---|---|---|---:|
| 2026-11-01T12:50:22.20283+00:00 | 5 | safety_check_failed | ISOLATION_POLICY_INVALID / apply | 615.136000 |

These are report times, not exact kernel close instants.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 73 | verification_ISOLATION_POLICY_INVALID | true | 539.222 … 580.628 |
| 74 | verification_ISOLATION_POLICY_INVALID | true | 590.309 … 615.050 |

Native samples: 7. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **fail**. Member SYNs / node ingress SYNs / origin receipts: 5 / 5 / 0.

Admission gate: pass at 2026-11-01T12:50:43.393539572Z; accepted generation 7. Kernel counter generation before → after: 7 → 7.

First matching SYN member-egress → node-ingress interval: [5671.079, 5671.087] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 85 | policy_missing | true | -18367.114 … -18330.199 |
| 86 | policy_missing | true | -18326.857 … -18293.891 |
| 87 | policy_missing | true | -18273.888 … -18239.690 |
| 105 | policy_missing | true | -8355.059 … -8321.626 |
| 106 | policy_missing | true | -8318.425 … -8294.063 |
| 107 | policy_missing | true | -8278.994 … -8248.580 |

Native samples: 49. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T12:50:25.203935767Z → 2026-11-01T12:50:28.329535459Z, accepted generation 5. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

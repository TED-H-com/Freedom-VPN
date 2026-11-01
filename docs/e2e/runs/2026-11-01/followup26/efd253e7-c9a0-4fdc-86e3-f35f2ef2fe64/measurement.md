# efd253e7-c9a0-4fdc-86e3-f35f2ef2fe64

TEST ONLY. Original requests and failed prerequisites are retained. Central history is in [central-record.json](central-record.json); exact packet and writer joins are in [serving-packet-joins.json](serving-packet-joins.json); peer states and transit bounds are in [session-intervals.json](session-intervals.json).

## initial-selection

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T13:02:13.216353747Z; accepted generation 5. Kernel counter generation before → after: 5 → 5.

First matching SYN member-egress → node-ingress interval: [104.991, 104.995] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

No baseline writer bracket in this capture (the first campaign attempt predates producer export).

Native samples: 11. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': True, 'node': True}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## regrant-session

Outcome: **pass**. Member SYNs / node ingress SYNs / origin receipts: 1 / 1 / 1.

Admission gate: pass at 2026-11-01T13:02:38.503913744Z; accepted generation 8. Kernel counter generation before → after: 8 → 8.

First matching SYN member-egress → node-ingress interval: [104.526, 104.534] ms.

No accepted non-serving report after this serving gate within the capture. This is not continuous-state proof.

| Writer sequence | Baseline cause | Succeeded | Relative writer bracket to first SYN (ms) |
|---:|---|---|---|
| 102 | policy_missing | true | -13265.640 … -13221.905 |
| 103 | policy_missing | true | -13218.633 … -13186.283 |
| 104 | policy_missing | true | -13168.376 … -13133.225 |
| 114 | policy_missing | true | -8275.538 … -8240.211 |
| 115 | policy_missing | true | -8236.844 … -8211.154 |
| 116 | policy_missing | true | -8196.611 … -8172.603 |

Native samples: 26. All sampled peers connected: {'member': False, 'node': False}. All sampled WireGuard peers matched: {'member': False, 'node': False}. Per-sample handshake timestamps, endpoints and route selections remain in the session supplement; a connected peer is not proof of delivered member data.

## After observer removal

Recovery gate: **pass**, 2026-11-01T13:02:23.615053769Z → 2026-11-01T13:02:27.729368587Z, accepted generation 6. This does not replace the original request result.

Raw packet/log/listing/producer bytes stay private. Public hashes and header-only exports are checked by the analyzer. Bounds assume no intervening clock step and include measured offset envelopes plus pcap quantization. No encrypted datagram is assigned to an inner TCP stream by timing alone.

# Transport timeline: 1ecee8fe-69a4-4a38-8f72-aa392ee88eb1

Source: `d616bb1410d7aea35e9d5bad98f751bed7324839`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 4.766312–4.812435 ms.

Stream: 7358 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793547758.622896 | 1 / `d1c25bf1a7102efb` | member / lo / 471 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:37267` | IPv4 |
| 1793547758.625331 | 1 / `8264c8b6c2ed898b` | node / lo / 129 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793547758.626032 | 1 / `8264c8b6c2ed898b` | member / lo / 494 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547758.626135 | 2 / `e2c677ecad9eb579` | member / lo / 500 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793547758.626504 | 2 / `e2c677ecad9eb579` | node / lo / 146 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547758.827523 | 1 / `d1c25bf1a7102efb` | node / lo / 308 | `addr-5f910114cdbd19f5:51820` → `::1:51820` | IPv6 |
| 1793547758.827648 | 2 / `7d66b24b6e5b039d` | node / eth2 / 309 | `::1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547758.827656 | 2 / `7d66b24b6e5b039d` | member / eth1 / 666 | `::1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547879.031865 | 1 / `04a556882f6807d7` | node / eth2 / 17910 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547879.031868 | 1 / `04a556882f6807d7` | member / eth1 / 16369 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547879.032186 | 2 / `c0d25150fa6d1842` | member / eth1 / 16370 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547879.032193 | 2 / `c0d25150fa6d1842` | node / eth2 / 17911 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

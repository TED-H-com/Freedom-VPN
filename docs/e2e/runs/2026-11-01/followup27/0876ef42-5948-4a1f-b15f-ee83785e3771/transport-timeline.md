# Transport timeline: 0876ef42-5948-4a1f-b15f-ee83785e3771

Source: `cd190549aec32f8902c8b85d03f96ebd49a7dfa9`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 3.534703–3.559238 ms.

Stream: 7297 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793547187.599503 | 1 / `50535efb8cc92c46` | member / lo / 426 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:58859` | IPv4 |
| 1793547187.601423 | 1 / `3f0d3273aee2b6dd` | node / lo / 108 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793547187.602014 | 1 / `3f0d3273aee2b6dd` | member / lo / 457 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547187.602114 | 2 / `21fd4c5cfe8b561f` | member / lo / 460 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793547187.602282 | 2 / `21fd4c5cfe8b561f` | node / lo / 130 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547187.702924 | 1 / `50535efb8cc92c46` | node / lo / 274 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547187.703041 | 2 / `523276b0a557160c` | node / lo / 275 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793547187.703272 | 2 / `523276b0a557160c` | member / lo / 588 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547308.466707 | 1 / `82247c0a56c783a4` | node / eth2 / 18798 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547308.466711 | 1 / `82247c0a56c783a4` | member / eth1 / 17032 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547308.466984 | 2 / `5306b53daa3daccc` | member / eth1 / 17034 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547308.466986 | 2 / `5306b53daa3daccc` | node / eth2 / 18799 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

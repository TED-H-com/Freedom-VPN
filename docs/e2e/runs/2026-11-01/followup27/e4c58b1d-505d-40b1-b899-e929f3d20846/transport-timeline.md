# Transport timeline: e4c58b1d-505d-40b1-b899-e929f3d20846

Source: `9d7df53501ee798b6cbe954771dccfda3640ca43`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 105.912113–105.972855 ms.

Stream: 7304 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793545731.232288 | 1 / `03d7cc40b8597941` | member / lo / 379 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:41975` | IPv4 |
| 1793545731.234570 | 1 / `396ded9c9bf4558a` | node / lo / 125 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:54424` | IPv4 |
| 1793545731.335661 | 1 / `03d7cc40b8597941` | node / lo / 260 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793545731.335872 | 2 / `4a01861fbc684aea` | node / lo / 261 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793545731.336585 | 2 / `4a01861fbc684aea` | member / lo / 525 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793545851.346435 | 1 / `5c54623766fecfba` | member / eth0 / 11157 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |
| 1793545851.346442 | 1 / `5c54623766fecfba` | node / eth1 / 15752 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |
| 1793545851.346663 | 2 / `53dea75d9a129a53` | node / eth1 / 15753 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |
| 1793545851.346670 | 2 / `53dea75d9a129a53` | member / eth0 / 11158 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

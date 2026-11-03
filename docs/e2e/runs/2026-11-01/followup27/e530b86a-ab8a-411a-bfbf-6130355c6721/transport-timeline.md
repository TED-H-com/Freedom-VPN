# Transport timeline: e530b86a-ab8a-411a-bfbf-6130355c6721

Source: `cd190549aec32f8902c8b85d03f96ebd49a7dfa9`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 104.935676–104.965715 ms.

Stream: 7342 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793546900.123450 | 1 / `67111729aafd11f8` | member / lo / 468 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:49083` | IPv4 |
| 1793546900.125497 | 1 / `136ecf72fb3c7585` | node / lo / 123 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:53382` | IPv4 |
| 1793546900.227903 | 1 / `67111729aafd11f8` | node / lo / 214 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546900.228012 | 2 / `05f73341a64a8577` | node / lo / 215 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793546900.228158 | 2 / `05f73341a64a8577` | member / lo / 568 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547020.240246 | 1 / `2615d71adde7e157` | member / eth1 / 16347 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547020.240251 | 1 / `2615d71adde7e157` | node / eth2 / 17829 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547020.240359 | 2 / `80f8dda5411d2806` | node / eth2 / 17830 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547020.240363 | 2 / `80f8dda5411d2806` | member / eth1 / 16348 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

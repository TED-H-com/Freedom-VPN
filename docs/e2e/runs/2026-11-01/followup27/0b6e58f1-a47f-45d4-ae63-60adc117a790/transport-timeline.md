# Transport timeline: 0b6e58f1-a47f-45d4-ae63-60adc117a790

Source: `fcf04fd5dcd5fbba91a7c0c635822c26636fcd12`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 106.703229–106.727972 ms.

Stream: 7351 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793546038.624503 | 1 / `e5fe5280cd587442` | member / lo / 396 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:59569` | IPv4 |
| 1793546038.626410 | 1 / `62181df889ef0f2f` | node / lo / 128 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:59909` | IPv4 |
| 1793546038.729286 | 1 / `e5fe5280cd587442` | node / lo / 277 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546038.729434 | 2 / `5ddbdc1a7b5f7f5d` | node / lo / 278 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793546038.730106 | 2 / `5ddbdc1a7b5f7f5d` | member / lo / 544 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546158.730596 | 1 / `a1b52fd2d13201e8` | member / eth0 / 12733 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |
| 1793546158.730602 | 1 / `a1b52fd2d13201e8` | node / eth1 / 18107 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |
| 1793546158.730722 | 2 / `b3c4d42e32bd4cfd` | node / eth1 / 18110 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |
| 1793546158.730731 | 2 / `b3c4d42e32bd4cfd` | member / eth0 / 12734 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

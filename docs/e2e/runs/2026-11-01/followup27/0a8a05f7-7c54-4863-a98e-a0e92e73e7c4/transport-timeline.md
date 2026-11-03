# Transport timeline: 0a8a05f7-7c54-4863-a98e-a0e92e73e7c4

Source: `b43fb15af52ed54aa4e13a600a3b2d92469e189b`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 3.350965–3.369257 ms.

Stream: 7337 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793546329.405121 | 1 / `82890730c780a1e4` | member / lo / 379 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:60676` | IPv4 |
| 1793546329.406914 | 1 / `d38ef35ce742057c` | node / lo / 134 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793546329.407300 | 1 / `d38ef35ce742057c` | member / lo / 402 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546329.407454 | 2 / `4f04a542ded55c35` | member / lo / 412 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793546329.407918 | 2 / `4f04a542ded55c35` | node / lo / 164 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546329.507725 | 1 / `82890730c780a1e4` | node / lo / 401 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546329.507828 | 2 / `96c91bbae56e409b` | node / lo / 402 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793546329.507974 | 2 / `96c91bbae56e409b` | member / lo / 646 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546449.807066 | 1 / `ce091cdea7fa7cde` | node / eth1 / 18207 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |
| 1793546449.807072 | 1 / `ce091cdea7fa7cde` | member / eth0 / 12853 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |
| 1793546449.807292 | 2 / `ec0ea6e053134099` | member / eth0 / 12854 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |
| 1793546449.807298 | 2 / `ec0ea6e053134099` | node / eth1 / 18208 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

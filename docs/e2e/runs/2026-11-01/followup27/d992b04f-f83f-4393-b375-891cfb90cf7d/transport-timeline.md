# Transport timeline: d992b04f-f83f-4393-b375-891cfb90cf7d

Source: `757052242f7a5beeb2526a44194af8e257dc99da`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 104.996068–105.044803 ms.

Stream: 7308 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793546603.708984 | 1 / `ac041cbb3df72a37` | member / lo / 386 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:37175` | IPv4 |
| 1793546603.711422 | 1 / `3128e35fca20c0dd` | node / lo / 115 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:35337` | IPv4 |
| 1793546603.813299 | 1 / `ac041cbb3df72a37` | node / lo / 277 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546603.813423 | 2 / `d5da48234fdc9db8` | node / lo / 278 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793546603.813617 | 2 / `d5da48234fdc9db8` | member / lo / 555 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793546723.822254 | 1 / `edaff9efad4452b1` | member / eth0 / 11179 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |
| 1793546723.822262 | 1 / `edaff9efad4452b1` | node / eth1 / 15766 | `addr-129fe9318e066262:51820` → `addr-459868bfedca20a7:51820` | IPv4 |
| 1793546723.822382 | 2 / `80b5184e44e68e5e` | node / eth1 / 15769 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |
| 1793546723.822388 | 2 / `80b5184e44e68e5e` | member / eth0 / 11180 | `addr-459868bfedca20a7:51820` → `addr-129fe9318e066262:51820` | IPv4 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

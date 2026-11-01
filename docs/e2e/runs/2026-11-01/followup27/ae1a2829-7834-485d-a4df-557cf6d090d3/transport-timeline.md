# Transport timeline: ae1a2829-7834-485d-a4df-557cf6d090d3

Source: `cd190549aec32f8902c8b85d03f96ebd49a7dfa9`.

[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)

First selection: **pass**, member/node/origin SYNs 1 / 1 / 1.
First SYN member-egress → node-ingress: 3.585356–3.630897 ms.

Stream: 7322 socket-write attempts; 0 send errors; 0 absent at node ingress; 0 ingress gaps ≥1 s; 0 deliveries whose upper bound exceeds 1 s.
Missing ranges: `[]`. All loss is counted, including isolated and terminal loss.

| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |
|---|---|---|---|---|
| 1793547451.716846 | 1 / `e85074a28d6d78ba` | member / lo / 456 | `127.0.0.1:51820` → `addr-32bcd336456e1b8c:37250` | IPv4 |
| 1793547451.718879 | 1 / `259059536c5f17bc` | node / lo / 116 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793547451.719493 | 1 / `259059536c5f17bc` | member / lo / 484 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547451.719596 | 2 / `36383e6dff6b525b` | member / lo / 488 | `127.0.0.1:51820` → `addr-060f0ef186d1db5b:3128` | IPv4 |
| 1793547451.720112 | 2 / `36383e6dff6b525b` | node / lo / 156 | `addr-060f0ef186d1db5b:3128` → `127.0.0.1:51820` | IPv4 |
| 1793547451.920478 | 1 / `e85074a28d6d78ba` | node / lo / 284 | `addr-5f910114cdbd19f5:51820` → `::1:51820` | IPv6 |
| 1793547451.920671 | 2 / `6081d9e36e671cf5` | node / eth2 / 285 | `::1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547451.920681 | 2 / `6081d9e36e671cf5` | member / eth1 / 640 | `::1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547467.316916 | 1 / `3597bf0a0d7c7ff3` | member / eth1 / 838 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547467.316926 | 1 / `3597bf0a0d7c7ff3` | node / eth2 / 384 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547467.317174 | 2 / `b359069e55646867` | node / eth2 / 385 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547467.317184 | 2 / `b359069e55646867` | member / eth1 / 839 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547587.337133 | 1 / `362988c94572fb8a` | member / eth1 / 16481 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547587.337140 | 1 / `362988c94572fb8a` | node / eth2 / 17964 | `addr-5f910114cdbd19f5:51820` → `addr-837a7935ba5539b1:51820` | IPv6 |
| 1793547587.337499 | 2 / `c44fa2dd6a603e46` | node / eth2 / 17965 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |
| 1793547587.337506 | 2 / `c44fa2dd6a603e46` | member / eth1 / 16482 | `addr-837a7935ba5539b1:51820` → `addr-5f910114cdbd19f5:51820` | IPv6 |

Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.

Accepted closed reports after serving admission: 0. Capture drops/read errors/truncations are zero on all three points; 8 private receipts verified.

The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.

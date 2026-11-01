# Follow-up 13 native capture archive

Run `4182dd3d-7f1f-453e-8c1f-265d097c3abc`, compiled harness `d49b535a`,
product baseline `de6d2e02`. TEST ONLY, Linux amd64. All eight capture sessions
(24 namespace captures) are complete, with zero kernel drops, truncations or
header errors. Raw pcaps remain private; this archive contains metadata, packet
headers, opaque address/message hashes and empirical monotonic intervals.
`SHA256SUMS` pins this immutable subtree (D-132).

`*-original-capture.json` preserves the exact exported snapshots. Analyzer inputs
map `before-transfer` to `before-first-transfer`, and wrap the matrix's transfer
object as `bulk`; these are structural adaptations only. Original AT rows are
unchanged. `later-selection-1` is the original sleep/wake transfer on TCP port
20002, distinct from the first-selection port 20001. `analysis.json` retains
shared decoder/analyzer source hashes. Reproduction uses the owned
`tests/e2e/at/members/followup13_evidence.py` and private pcaps, writing disk scratch.

First selection has one SYN and successful origin/counter evidence, with type-4
WireGuard messages through the local proxy path. A loopback-source IPv6 response
arrives after the successful request. This does not reproduce the historical
five-SYN failure and does not identify its cause.

All six bulk rows transfer 64,064 bytes in 1 ms. Receive-only 1280 rows remain
limitations, never supported-path passes. Their reverse data packets are 1,132
bytes (IPv4) and 1,152 bytes (IPv6); no oversized reverse loss was reproduced.
The separate known-PMTU IPv6 control delivers 11 reverse fragment pairs after
installing the selected sender route metric 1280; readbacks prove exact removal.
No PTB or IPv4 fragmentation-needed occurs at any retained point. This proves
known route-MTU sender fragmentation, not reactive PMTU discovery. All AT09
interpretation is under provisional D-137; historical F8-02 failures stay failed.

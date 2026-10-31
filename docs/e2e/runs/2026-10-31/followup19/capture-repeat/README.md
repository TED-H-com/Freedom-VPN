# Additional native reconnect capture (TEST ONLY)

Run `4e1997a8-d3ea-4291-b2f6-c100992b4fda`, source `587e5267`, passes 7/0/0, exit 0.
This one additional fresh attempt followed the separate 12,012-byte failure
in the DNS fixture campaign. All four reconnect transfers again pass at
64,064 validated bytes in 1 ms, with 64 complete echoes, zero partial line,
64,387 member stream bytes read / 64,108 written, and no member TCP loss or
retransmission. Origin final samples agree on all request/response bytes.
Both dual rows pass IPv6 refusal and no uplink probe leakage.

The original deadline, payload and per-line application writes are unchanged.
Per-I/O instrumentation adds scheduling overhead and can affect TCP segmentation;
successful instrumented controls do not prove absence of the intermittent
failure. No failed instrumented transfer was captured in these two attempts.
The retained failed pair remains the failure-side packet evidence.

All 12 pcaps match the raw receipt byte/SHA pins, stop on complete records,
have zero unparsed records or header errors, and match tcpdump captured counts.
All have zero kernel drops; member filter counts exceed captured counts by
four in each row. Process start/stop bounds and endpoint samples are retained
in the [scenario record](../4e1997a8-d3ea-4291-b2f6-c100992b4fda/native-mtu-comparison.md).
Snaplen 128 intentionally excludes most payload; no payload is decoded here.

| File / full header decode | Bytes | SHA-256 | Records / snap-truncated | First / last packet (Unix UTC) |
|---|---:|---|---|---|
| [matrix1280-member](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix1280-member.headers.jsonl) | 44652 | `365ca6547c5d0528b834f29c0174330391cf1e700e886aa0ef8b070c8f4ed2ef` | 337 / 234 | 1793457043.029311 / 1793457043.150163 |
| [matrix1280-node](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix1280-node.headers.jsonl) | 65723 | `7cee916282215a193081329ad8c984a09e8229b0c5427294009da8b7fb4963de` | 500 / 355 | 1793457042.532515 / 1793457043.497577 |
| [matrix1280-origin](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix1280-origin.headers.jsonl) | 17439 | `4033fea3bbe2e5f976a3f63644ef036caf822563a1635b77431afc4d2755de07` | 138 / 93 | 1793457043.148251 / 1793457043.150189 |
| [matrix4-member](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix4-member.headers.jsonl) | 38692 | `381842f3e2abca43995b1019e05be4473009f2c223894b0516185e9f650c423f` | 285 / 239 | 1793457022.195514 / 1793457022.462705 |
| [matrix4-node](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix4-node.headers.jsonl) | 57407 | `748cb7d376e98c7a0d9b8ca77ef62953f6cfb03c9bddbf7943f6ac9c02685771` | 429 / 348 | 1793457022.305496 / 1793457022.525693 |
| [matrix4-origin](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix4-origin.headers.jsonl) | 16855 | `c2743a09e0124d7b1d0847547fc261e445c4a4826440b1a66ffe6517a7cab9a0` | 132 / 92 | 1793457022.305554 / 1793457022.307423 |
| [matrix6-member](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix6-member.headers.jsonl) | 37944 | `dadfda1fcd4b3be99c0d2b819727915f20e04518b95bfc044b3db9bbd73475d9` | 281 / 231 | 1793457028.660777 / 1793457029.185244 |
| [matrix6-node](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix6-node.headers.jsonl) | 61542 | `308aeb17077bcac235826b54cb2a4617bdcd327e36ae755269a0f3be73b4f7d3` | 459 / 375 | 1793457029.183229 / 1793457029.462183 |
| [matrix6-origin](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrix6-origin.headers.jsonl) | 18903 | `306418c4c2d250ca6cafe1af4402b83efee6825e5210e40d3a07b80ca498baa3` | 147 / 105 | 1793457029.183296 / 1793457029.185278 |
| [matrixDual-member](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrixDual-member.headers.jsonl) | 37716 | `a67937d225f3f99dbfd279fb75fe67eb30401726196c22989aa7bfd4d04480b3` | 279 / 231 | 1793457036.048492 / 1793457036.168927 |
| [matrixDual-node](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrixDual-node.headers.jsonl) | 59238 | `fc646c20836b9c279b8ca474e03153c44a212a1d3030587548b95f122004cd6d` | 443 / 359 | 1793457036.166920 / 1793457036.421605 |
| [matrixDual-origin](4e1997a8-d3ea-4291-b2f6-c100992b4fda-matrixDual-origin.headers.jsonl) | 17751 | `900ac8a74938839c337a2590810fd66dad662634b41b0017a0c1236d2fddf0dd` | 139 / 97 | 1793457036.166979 / 1793457036.168939 |

# Follow-up 23 DNS delivery — 2026-11-01

Run `9bb2098e-1c6f-48a8-9fbd-fd99bd9fc49e`, source
`1d396065c432485875add24a00761608afb293cc`: **11 pass / 0 blocked / 1 fail**,
exit **1**. The failed `native-dns-delivery` record is retained unchanged.
Its three application queries are positive; its capture-statistics gate fails
because the existing parser accepted plural `packets` but not singular `packet`.
The one-packet secondary was incorrectly marked statistics-unavailable.

[Same stopped-capture statistics readback](capture-statistics-readback.json)
retains the original capture UUID, exact statistics text and SHA/size: **1 packet
captured / 1 packet received / 0 dropped**. No traffic was repeated and no pcap
was modified. The parser now accepts both forms; missing statistics remain an
error. [Derived delivery analysis](dns-delivery-analysis.json) explicitly uses
that supplement without rewriting the original record or its failed verdict.
It verifies statistics SHA/size, capture UUID and decoded record count. All five
capture files are complete, without drops or truncation: member **58**, node
**88**, origin **15**, primary **18**, secondary **1**.

## Positive IDNA wire fixture (F4-05 / F4-07)

`ProbeNativeIDNA` converts `bücher.example.invalid` to the expected A-label
`xn--bcher-kva.example.invalid` using the lookup profile, then calls the original
native-stub probe. The authority supplies a positive controlled-origin A record.

| Query | Wire ID | Client RCODE / types | Remaining TTL | Duration |
|---|---:|---|---:|---:|
| RR A UDP | 53718 | 0 / A | 5 | 2 ms |
| IDNA A UDP | 22849 | 0 / A | 5 | 1 ms |
| IDNA A TCP | 64801 | 0 / A | 4 | 1 ms |

All answers match the expected controlled-origin address. Both IDNA queries
have independent positive primary receipts, complete lifecycle history and
wire joins. This is positive native wire evidence, not a Unicode browser/OS
UI test, automatic production DNS distribution, or complete AT20–AT22 acceptance.

## F12-04: the delivery segment is now observable; original cause remains open

The fresh RR query succeeds and does **not** reproduce members 12's historical
5,001 ms SERVFAIL. The original historical responder and cause remain
**unattributed / unproven**. No repeat query replaces that failed row.

The new RR's exact primary reply digest is observed at:

- Primary eth0 egress **packet 4**, **08:29:34.784288Z**.
- Node fvup0 ingress **14**, **08:29:34.784292Z**.
- Node fvmb0 egress **15**, **08:29:34.784309Z**.
- Member fvnb0 ingress **15**, **08:29:34.784393Z**.

The native stub then emits a positive response on member loopback **packet 16**,
**08:29:34.784466Z**, with the same transaction ID. The full message digest changes:
the upstream reply has an OPT record, while the local response has no additional
record and a different encoding/length. The successful client result independently
confirms processing in this fresh query. An identical full-message hash is not
required across native DNS reserialization.

IDNA UDP replies similarly join primary **8** → node **28/29** → member tunnel
**21**, then local stub **22**. TCP joins primary **14** → node **58/59** → member
tunnel **40**, then local stub **46**. These captured messages fit in single
segments; the decoder does not claim general TCP reassembly.

For the historical F12-04 failure, the still-unobserved interval is now precisely
instrumentable: successful relay socket write → relay egress → selected-node
forwarding → member native tunnel ingress → native local-stub response → client
result. Resolver receipt alone stops at the first boundary. Both positive
historical receipts cannot name the source of the client's RCODE 2. Adopt
`StartNativeDNSCapture` around the original members sequence to determine which
boundary fails there; neither this successful query nor the pin's timeout code
can reconstruct that missing historical evidence. No AdGuard SERVFAIL, cache
TTL failure, transport drop or native parsing cause is asserted.

F12-03's negative zone, TTL, cache size, phase transitions and historical verdict
were not changed. No quiet-window retry or negative-cache mutation was added.

# Follow-up 3 acceptance summary

Initial member traffic is counted end to end. The node lease follows applied
generation without a restart. After the first device revocation, the node lacks
`conntrack` and falls back to its drop baseline; subsequent paid/member and
publication traffic stays blocked. See [forwarding observation](forwarding-observation.md)
and [requests](requests.md).

Current records: **242 passing subchecks, 0 current failed subchecks**.
AT08 retains **1 failed and 30 passing prior checks**; its earlier
HTTP 422 is not erased. All ten complete ATs remain blocked for the stated
conditions. The initial AT02 pass is superseded by the observed post-revoke
failure; its positive quota and forwarding evidence remains unchanged.

| AT | Complete result | Evidence | Remaining condition |
|---|---|---|---|
| AT01 | blocked | [AT01.json](AT01.json), [AT01.md](AT01.md) | Four platforms and network move; initial Linux forwarding/wake proven |
| AT02 | blocked | [AT02.json](AT02.json), [AT02.md](AT02.md) | Post-revoke enforcement: missing conntrack (F3-02) |
| AT03 | blocked | [AT03.json](AT03.json), [AT03.md](AT03.md) | Paid-member forwarding after F3-02; financial checks passed |
| AT04 | blocked | [AT04.json](AT04.json), [AT04.md](AT04.md) | F3-02 prevents positive persistent TCP; UDP/all-node fixtures absent |
| AT05 | blocked | [AT05.json](AT05.json), [AT05.md](AT05.md) | Three boot flavours and physical arm64 |
| AT08 | blocked | [AT08.json](AT08.json), [AT08.md](AT08.md) | F3-02 A/B forwarding; prior restore 422 retained |
| AT09 | blocked | [AT09.json](AT09.json), [AT09.md](AT09.md) | Second runnable exit, relay/QUIC/PMTU and physical NAT |
| AT20 | blocked | [AT20.json](AT20.json), [AT20.md](AT20.md) | Native DNS disabled; RR/cache/rebinding matrix incomplete |
| AT21 | blocked | [AT21.json](AT21.json), [AT21.md](AT21.md) | Second filtering resolver and native fallback/DoH/DoT |
| AT22 | blocked | [AT22.json](AT22.json), [AT22.md](AT22.md) | Full URL/file/raw-usage/backup retention surfaces |

| Operator scope | Status | Script |
|---|---|---|
| Windows, macOS, iOS, Android | needs operator run | [OPERATORS.md](OPERATORS.md) |
| Physical arm64 / three source hardware | needs operator run | [OPERATORS.md](OPERATORS.md) |
| Second physical NAT / candidate production hardware | needs operator run | [OPERATORS.md](OPERATORS.md) |

D-056 node delivery and B9 nullable attribution pass their live subchecks.
The real 300-second feed refresh fails with `zero_rules` and preserves the
last-good digest; node-local NXDOMAIN with TTL 5 remains. This is not a native
member DNS or complete privacy pass. See [RUN.md](RUN.md) for both attempts,
provenance, actual command tails and cleanup.

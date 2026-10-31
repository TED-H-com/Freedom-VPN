# Forwarding observations — Follow-up 10

Run `2be2d32f-bfd9-46f5-bf29-afaa6617ae94`, harness `731be5a7af7657660d96c240695747a23a3f5c96`, product baseline `d3e9a0f1` (agent-core 27 #325, wire-nodes 32 #321, netlab 19 #318; devstack 18 and dns-publish 8 retained), assigned `w4-members / 24000`.

Counted HTTPS requires the controlled origin receipt, selected-exit SNAT and device upload/download increases. All snapshots remain in the AT JSON records.

| AT | Persistent cohort | Event UTC | Last TCP / UDP reply delta (s) | Bound (s) | Result |
|---|---|---|---:|---:|---|
| AT02 | member-a3 | 2026-10-31T13:34:37.640762923Z | 0.312 / 0.312 | 5 | pass |
| AT02 | member-a3 | 2026-10-31T13:36:39Z | -0.012 / -0.012 | 5 | fail |
| AT04 | member-paid | 2026-10-31T13:41:22.330573086Z | 3.401 / 3.401 | 60 | pass |
| AT04 | member-paid2 | 2026-10-31T13:41:22.330573086Z | 3.597 / 3.597 | 60 | pass |
| AT04 | member-expiry1 | 2026-10-31T13:45:25Z | 0.065 / 0.065 | 60 | pass |
| AT04 | member-expiry2 | 2026-10-31T13:45:25Z | -0.265 / -0.265 | 60 | pass |
| AT04 | member-suspend1 | 2026-10-31T13:48:11.413681234Z | 2.034 / 2.034 | 60 | pass |
| AT04 | member-suspend2 | 2026-10-31T13:48:11.413681234Z | 1.867 / 1.867 | 60 | pass |

TCP and UDP keep their established sockets. Post-bound samples prove senders remain active while replies/counters stop or policy removes counters. Poll timestamps bound observations, not wire delivery. AT04 exercises both exits.

AT09 transfer counters and selected-family observations are indexed in [NATIVE.md](NATIVE.md). Under D-123 those family selectors describe the underlay; the payload is IPv4. No QUIC, physical roaming, second NAT, full PMTU or IPv6 overlay forwarding claim follows. The separate revoked-source ingress test uses positive controls and never restores a withdrawn production route to manufacture denial.

F10-02 remains failed because the finite-expiry `member-a3` window contains
one unreadable sample, despite pre-deadline last replies and stable final
counters. The table's failed row is not a post-expiry leakage claim and is not
promoted by the other passing withdrawal rows. [Capture](F10-02-withdrawal.json).

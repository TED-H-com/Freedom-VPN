# Forwarding observations — Follow-up 9

Run `11e305fe-6360-4dbd-95b1-67274aa4988c`, harness `49fbb48e013b824c4ce107d3cb551db79b915b42`, product baseline `7e044c71` (agent-core 25 #312, devstack 18 #313, dns-publish 8 #309 and netlab 18), assigned `w4-members / 24000`.

Counted HTTPS requires the controlled origin receipt, selected-exit SNAT and device upload/download increases. All snapshots remain in the AT JSON records.

| AT | Persistent cohort | Event UTC | Last TCP / UDP reply delta (s) | Bound (s) | Result |
|---|---|---|---:|---:|---|
| AT02 | member-a3 | 2026-10-31T11:40:23.854501216Z | 0.936 / 0.936 | 5 | pass |
| AT02 | member-a3 | 2026-10-31T11:42:25Z | -0.158 / -0.158 | 5 | pass |
| AT04 | member-paid | 2026-10-31T11:47:20.239400279Z | 3.450 / 3.450 | 60 | pass |
| AT04 | member-paid2 | 2026-10-31T11:47:20.239400279Z | 3.468 / 3.468 | 60 | pass |
| AT04 | member-expiry1 | 2026-10-31T11:51:26Z | 0.059 / 0.059 | 60 | pass |
| AT04 | member-expiry2 | 2026-10-31T11:51:26Z | -0.250 / -0.250 | 60 | pass |
| AT04 | member-suspend1 | 2026-10-31T11:54:22.437571017Z | 2.337 / 2.337 | 60 | pass |
| AT04 | member-suspend2 | 2026-10-31T11:54:22.437571017Z | 1.887 / 1.887 | 60 | pass |

TCP and UDP keep their established sockets. Post-bound samples prove senders remain active while replies/counters stop or policy removes counters. Poll timestamps bound observations, not wire delivery. AT04 exercises both exits.

AT09 transfer counters and selected-family observations are indexed in [NATIVE.md](NATIVE.md). Under D-123 those family selectors describe the underlay; the payload is IPv4. No QUIC, physical roaming, second NAT, full PMTU or IPv6 overlay forwarding claim follows. The separate revoked-source ingress test uses positive controls and never restores a withdrawn production route to manufacture denial.

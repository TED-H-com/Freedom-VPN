# Forwarding observations — Follow-up 11

Run `7edb263f-1f6e-4f94-9e8e-26099f96a699`, harness `33abb2fd2c2fc95826c6e69eb434045522d6f7ef`, product baseline `96ec0630e4834e13a3a14d65553ba8108bb5da55` (devstack 20 #343, agent-core 28 #344, netlab 20 #336; dns-publish 8 #309). Assigned `w4-members / 24000`, two resolvers. The full mutation order is unchanged.

Counted HTTPS requires an origin receipt, selected-exit SNAT and both named device-counter increases. Captured attempts remain separate from successful forwarding.

| AT | Persistent cohort | Event UTC | Last TCP / UDP reply delta (s) | Bound (s) | Reads / failed | Result |
|---|---|---|---:|---:|---:|---|
| AT02 | member-a4 | 2026-10-31T18:32:39.672151678Z | 0.974625849 / 0.974625849 | 5 | 28 / 0 | pass |
| AT02 | member-a4 | 2026-10-31T18:34:37Z | -0.408006282 / -0.408006282 | 5 | 186 / 0 | pass |
| AT04 | member-paid | 2026-10-31T18:39:28.40922031Z | 3.589907078 / 3.589907078 | 60 | 197 / 0 | pass |
| AT04 | member-paid2 | 2026-10-31T18:39:28.40922031Z | 3.376495706 / 3.376495706 | 60 | 252 / 0 | pass |
| AT04 | member-expiry1 | 2026-10-31T18:43:32Z | -0.019940244 / -0.019940244 | 60 | 345 / 0 | pass |
| AT04 | member-expiry2 | 2026-10-31T18:43:32Z | -0.132194045 / -0.132194045 | 60 | 360 / 0 | pass |
| AT04 | member-suspend1 | 2026-10-31T18:46:28.478325244Z | 2.489762799 / 2.489762799 | 60 | 194 / 0 | pass |
| AT04 | member-suspend2 | 2026-10-31T18:46:28.478325244Z | 2.597296362 / 2.597296362 | 60 | 205 / 0 | pass |

The established TCP/UDP sockets and active senders persist through each cutoff. Poll timestamps bound observations, not wire delivery; the nominal 250 ms cadence also includes reader execution time. AT02 retains reader categories and validates all same-boot before/after producer joins against its full stream. AT04 exercises both exits; no node2 timing-completeness claim is added.

AT01's initial path lacks node/origin receipt (F11-01). AT02's early D-077 positive probe runs during an empty-grant lease (F11-02). The strict revoked-source injection succeeds, but its admitted sibling control hits baseline during recorded generation-13 failed applies (F11-03). [Captures and dated owners](requests.md). No later passing transfer replaces those rows, and no withdrawn route is restored to manufacture denial.

[NATIVE.md](NATIVE.md) retains AT09 bulk results and selected families. D-123 limits these to IPv4 payload over the selected underlay; no QUIC, physical roaming, second NAT, complete PMTU or IPv6 overlay claim follows. Historical F10-02's unreadable finite-expiry sample remains failed and unattributed in [follow-up 10](history/follow-up-10/F10-02-withdrawal.json); this run's successful reader measurement is a separate row.

# Forwarding observations — Follow-up 8

Run `69f92f49-73dc-4779-8101-dfb6a802a96b`, harness `d79accd9c693aed83713072c925b3b5005b5684c`, product baseline `6848add` (#284), assigned `w4-members / 24000`. [AT JSON records](summary.md) retain exact before/after snapshots. Counted HTTPS requires the authenticated controlled origin receipt, selected-exit SNAT and upload/download counter increases. Negative rows need positive controls.

| AT | Persistent cohort | Event UTC | Last TCP / UDP reply delta (s) | Bound (s) | Assertion |
|---|---|---|---:|---:|---|
| AT02 | member-a4 | 2026-10-31T08:53:13.073851341Z | 0.944 / 0.944 | 5 | pass |
| AT02 | member-a4 | 2026-10-31T08:55:49Z | -0.842 / -0.842 | 5 | pass |
| AT04 | member-paid | 2026-10-31T09:01:58.446450493Z | 3.479 / 3.479 | 60 | pass |
| AT04 | member-paid2 | 2026-10-31T09:01:58.446450493Z | 3.385 / 3.385 | 60 | pass |
| AT04 | member-expiry1 | 2026-10-31T09:06:03Z | 0.169 / 0.169 | 60 | pass |
| AT04 | member-expiry2 | 2026-10-31T09:06:03Z | -0.559 / -0.559 | 60 | pass |
| AT04 | member-suspend1 | 2026-10-31T09:09:04.520234854Z | 2.306 / 2.306 | 60 | pass |
| AT04 | member-suspend2 | 2026-10-31T09:09:04.520234854Z | 2.510 / 2.510 | 60 | pass |

TCP never reconnects and UDP retains its socket. Samples after the bound prove senders remain active while replies and counters stop (or the policy removes the counter). The 250 ms polling cadence bounds observations; it is not wire timing. AT04 includes both exits for refund, paid expiry and suspension.

AT09 bulk rows retain the selected exit connection/family before and after, exact bytes and both device counters. A four-row TCP/MTU test does not establish QUIC, physical roaming, second NAT or ICMP-black-hole PMTU. Explicit unavailable rows remain blocked.

Fenced native HTTPS and strict injected ingress are separate: fence withdrawal removes the native route; AT02 revoked-source injection enters fvmb0 directly and checks ctunku versus the sibling counter and controlled origin. No withdrawn route or policy is restored to manufacture a denial proof.

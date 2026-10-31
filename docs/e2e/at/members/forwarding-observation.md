# Forwarding observations — Follow-up 7

Run `297faa31-2f90-4aab-8cf0-6818629f46c8` only. A counted transfer requires the authenticated controlled origin to receive the probe from the selected node's SNAT address, a CA-verified HTTP 200 receipt, and increases in that device's upload and download counters. Tables below use decimal counter differences, not synthetic usage-ingest bytes. Negative rows record observed denial; they are not failed checks by themselves.

| AT | Native peer | Exit | UTC observation | HTTP | Counted | Origin requests | Upload / download delta (B) |
|---|---|---|---|---:|---|---:|---:|
| AT01 | member | node | 2026-10-31T04:06:16Z | 200 | True | 1 | 2325 / 2413 |
| AT01 | member | node | 2026-10-31T04:06:31Z | 200 | True | 1 | 2265 / 2413 |
| AT02 | member | node | 2026-10-31T04:07:17Z | 200 | True | 1 | 2357 / 2464 |
| AT02 | member | node | 2026-10-31T04:07:47Z | 0 | False | 0 | 0 / 0 |
| AT02 | member | node | 2026-10-31T04:08:19Z | 200 | True | 1 | 2305 / 2412 |
| AT02 | member | node | 2026-10-31T04:08:49Z | 200 | True | 1 | 2357 / 2465 |
| AT02 | member-a3 | node | 2026-10-31T04:09:02Z | 200 | True | 1 | 2305 / 2413 |
| AT02 | member-a4 | node | 2026-10-31T04:09:14Z | 0 | False | 0 | 0 / 0 |
| AT02 | member-a2 | node | 2026-10-31T04:09:27Z | 200 | True | 1 | 2305 / 2464 |
| AT02 | member-a2 | node | 2026-10-31T04:09:50Z | 0 | False | 0 | 0 / 0 |
| AT02 | member | node | 2026-10-31T04:10:39Z | 200 | True | 1 | 2317 / 2412 |
| AT02 | member-a4 | node | 2026-10-31T04:10:52Z | 200 | True | 1 | 2357 / 2464 |
| AT02 | member-a4 | node | 2026-10-31T04:11:22Z | 200 | True | 1 | 2377 / 2413 |
| AT03 | member-paid | node | 2026-10-31T04:13:45Z | 200 | True | 1 | 2317 / 2411 |
| AT03 | member-paid | node | 2026-10-31T04:13:58Z | 200 | True | 1 | 2305 / 2412 |
| AT03 | member-paid | node | 2026-10-31T04:14:29Z | 0 | False | 0 | 0 / 0 |
| AT03 | member-paid | node | 2026-10-31T04:14:46Z | 200 | True | 1 | 2305 / 2413 |
| AT04 | member-paid2 | node2 | 2026-10-31T04:16:13Z | 200 | True | 1 | 2305 / 2413 |
| AT04 | member-paid | node | 2026-10-31T04:16:25Z | 200 | True | 1 | 2305 / 2464 |
| AT04 | member-paid | node | 2026-10-31T04:16:46Z | 0 | False | 0 | 0 / 0 |
| AT04 | member-paid2 | node2 | 2026-10-31T04:16:59Z | 0 | False | 0 | 0 / 0 |
| AT04 | member-expiry1 | node | 2026-10-31T04:17:49Z | 200 | True | 1 | 2305 / 2465 |
| AT04 | member-expiry2 | node2 | 2026-10-31T04:18:14Z | 200 | True | 1 | 2305 / 2463 |
| AT04 | member-expiry1 | node | 2026-10-31T04:19:55Z | 0 | False | 0 | 0 / 0 |
| AT04 | member-expiry2 | node2 | 2026-10-31T04:20:07Z | 0 | False | 0 | 0 / 0 |
| AT04 | member-suspend1 | node | 2026-10-31T04:20:58Z | 200 | True | 1 | 2409 / 2465 |
| AT04 | member-suspend2 | node2 | 2026-10-31T04:21:58Z | 200 | True | 1 | 2265 / 2412 |
| AT04 | member-suspend1 | node | 2026-10-31T04:22:17Z | 0 | False | 0 | 0 / 0 |
| AT04 | member-suspend2 | node2 | 2026-10-31T04:22:30Z | 0 | False | 0 | 0 / 0 |
| AT08 | member | node | 2026-10-31T04:23:12Z | 200 | True | 1 | 2305 / 2412 |
| AT08 | member-b1 | node | 2026-10-31T04:23:25Z | 0 | False | 0 | 0 / 0 |
| AT08 | member-b1 | node | 2026-10-31T04:23:42Z | 200 | True | 1 | 2305 / 2412 |
| AT08 | member | node | 2026-10-31T04:23:55Z | 0 | False | 0 | 0 / 0 |
| AT08 | member-b1 | node | 2026-10-31T04:24:12Z | 0 | False | 0 | 0 / 0 |
| AT08 | member-b1 | node | 2026-10-31T04:24:36Z | 0 | False | 0 | 0 / 0 |
| AT09 | member | node | 2026-10-31T04:25:50Z | 200 | True | 1 | 2265 / 2411 |
| AT09 | member | node2 | 2026-10-31T04:26:03Z | 200 | True | 1 | 2369 / 2465 |
| AT09 | member | node2 | 2026-10-31T04:27:04Z | 200 | True | 1 | 2265 / 2412 |
| AT09 | member | node | 2026-10-31T04:27:22Z | 200 | True | 1 | 2305 / 2412 |

D-077's fenced native probes lose their selected route and receive no origin response, with unchanged target counters and no observed `fvmb0` packets. This proves denial on the native path; it is not a separate injected-frame test of a fenced source. Explicit admin re-grant restores a route, member-list authorization and counted traffic. Exact bodies and before/after snapshots are in [D077.md](D077.md), [AT02.json](AT02.json) and [AT03.json](AT03.json).

AT02's revoked-source injection is a separate assertion: `Stack.ObserveMemberIngress` injects directly on `fvmb0` after member DELETE. The revoked frame increments `ctunku` and reaches neither the uplink nor origin; the admitted sibling increments its own upload counter and reaches both. No withdrawn NetBird route or policy is restored to force this proof.

AT04 records persistent TCP withdrawal at the actual refund/expiry/suspension boundary on both exits. AT09 retains its complete lease/observation timeline and same-generation recovery times. Native UDP/QUIC, forced relay, PMTU and physical networks remain F4 blockers. The updater exception probe supplies no member acceptance claim.

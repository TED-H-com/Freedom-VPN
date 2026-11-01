# Forwarding observations — Follow-up 12

Run `b98b78ee`, compiled harness `65e18e67`. Counted HTTPS requires an origin receipt, selected-exit SNAT and named device-counter increases after the [exact readiness predicate](AT02.md). Readiness failure stops the positive probe; it is not evidence that a sent packet failed.

| AT | Cohort | Event UTC | Last TCP / UDP reply delta (s) | Target (s) | Samples | Result |
|---|---|---|---:|---:|---:|---|
| AT02 | member-a3 | 2026-10-31T21:59:22.745247458Z | 0.326458120 / 0.326458120 | 5 | 28 | pass |
| AT02 | member-a3 | 2026-10-31T22:01:26Z | -0.085203130 / -0.085203130 | 5 | 193 | pass |
| AT04 | member-paid2 | 2026-10-31T22:09:19.760245577Z | 3.351259294 / 3.351259294 | 60 | 759 | pass |

AT02 records both independent counter/status categories and producer joins without retrying failed reads. Reply increments and polling bounds are not exact wire cutoff times. D-134's conservative expiry window does not waive early/renewal loss. The strict revoked UDP/9 source drops and its admitted control passes; continuous UDP/40002 independently retains all 21 losses. D-135 remains blocked on no failed Apply observed, with the separate policy-missing baseline bracket filed as F12-01.

AT03, AT04 and AT08 retain 11 failed readiness checks after the cumulative producer stream exceeds the old diagnostic reader's cap. The subsequent streaming-reader correction is offline-validated only. AT04's secondary persistent TCP monitor expires before refund input; its failed result contains a zero first-failure sentinel, not a meaningful negative withdrawal latency. The independent TCP/UDP persistent sender above has 759 reads and a 3.351259294 s last-reply delta. This does not rescue the separate failed control or cover missing primary/expiry/suspension paths. [Exact failures](F12-AT04-failures.json).

AT01/AT09 and [NATIVE.md](NATIVE.md) remain follow-up 11 evidence. F11-01 waits for devstack 22 route-selection exports; F8-02 waits for its native captures after netlab 21's IPv6 receive-link-MTU reproduction. No re-observation was launched. D-123 still limits native payload claims to IPv4 over selected underlays; physical platforms, QUIC and complete PMTU remain separate.

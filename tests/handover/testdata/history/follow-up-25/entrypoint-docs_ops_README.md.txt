# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 24](../handover/follow-up-24.md) (2026-11-01 EST):** Through #369: D-136 corrected; routed 1280 hop delivered; usage AT14–AT20 pass; members AT02 fails; AT04 fails on F13-01; graceful agent stop fixed. D-135 / D-136 / D-137 / D-138 remain provisional; historical failures stay failed.
Previous cut: [follow-up 23](../handover/follow-up-23.md), through #362.

## Follow-up 24 acceptance snapshot (2026-11-01 EST)

Fixed main **5bd75102**, through **#369**, all #363–#369. Nothing merged after
the cutoff at task start. r18 and devstack 24 are running, with no results in
this cut. [Full evidence and dated hand-offs](../handover/follow-up-24.md).

| Lane | Verified delivery and remaining boundary |
|---|---|
| Security-ops 14 (#363) | Interval-start periods in PITR/rate-class fixtures; offline month/year-boundary proof; replay per retained period. Product ingestion unchanged, no live suite, recorded rows unchanged. |
| Handover 23 (#364) | #357–#362 at bc011b6f; 16 subtests / 6 rows / 18 snapshots / 421 file checks; month-boundary incident; D-136 / D-137 provisional. |
| Netlab 22 (#365) | 132-file D-098 archive; opt-in TEST ONLY lifecycle/render capture; two 90-minute windows, no qualified second B agent leg. D-138 accepts observation; D-135 assessment, every loss retained. Requests 1–2 delivered by #369. |
| Usage AT 7 (#366) | 19-file 71b1a160 archive; interval/summary/held-snapshot period fixes and offline UTC-boundary proof; November b8a6e9e1 AT14–AT20 pass; 0 requests. |
| Devstack 23 (#367) | Routed 1280: IPv6 PTB / learned PMTU, IPv4 DF=0 transit fragmentation. F11-01 traced to NetBird 0.82.0 loopback proxy; positive IDNA; F12-04 not reproduced. 3 requests. |
| Members 13 (#368) | 52-file b98b78ee archive; 4182dd3d 2 pass / 6 blocked / 2 fail. AT02 61 baseline brackets / 21 losses, no failed Apply; AT04 F13-01 parser failures. F12-02 live-qualified, F12-03 quiet-window pass, historical rows stay failed. Known-PMTU passes; receive-only limitations. 10 requests. |
| Agent-core 31 (#369) | D-136 correction; 80 transitions / 1,013 losses retained in two populations, no ceiling. TestLab04 cancellation fix and recorded graceful exit; local completion at most 10 s, control I/O cancellable. TEST ONLY caller requires both gates. Intersection serving is for r18 review only; no failed live timing threshold, no D-133 run. |

D-135 / D-136 / D-137 / D-138 stay provisional. Owner decisions: D-135 +
D-136 confirmation, D-137 hardening option, D-138 overturn option. F13-01 raw
bytes/parser error and F11-01 drop investigation are assigned to devstack 24;
F12-04 original-query instrumentation and routed-hop adoption to members.
No historical failure is promoted. Install-overrun cause, F9-INFLIGHT,
F18-01 intended negative, ISO-F6-03 writer, Dependabot #22 and
blocked_response_ttl owner value retain their boundaries.

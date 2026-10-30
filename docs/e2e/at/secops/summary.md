# Follow-up 6 — measured admission and control diagnostics

Measured main `00e1d0c`; devstack #170 and control #169. This is a shared development host, not candidate hardware.

| Phase | Users/devices simultaneously authorized | Admission wall seconds | Per-peer admission p50/p95/p99 ms | Reads HTTP 200 | Read p50/p95/p99 ms | Foreign devices |
|---|---|---|---|---|---|---|
| profile | 20/60 | 904.360 | 14892.523 / 16451.805 / 18000.086 | 600/600 | 9.616 / 16.413 / 17.969 | 0 |
| helper | 20/60 | 300.386 | not exposed / not exposed / not exposed | 600/600 | 10.721 / 47.604 / 56.136 | 0 |

The first profile is unchanged sequential 20×3. The second uses exported `ScaleNativeMembers` (three peers per user concurrently); the helper does not return individual admission latencies, hence nulls. Both use grants and product admission, no projected-row writes. Gateway socket/FD inventory is in [capacity.json](capacity.json) and [42 timed samples](gateway-inventory.json). The initial profile/observer worker count used unavailable `ps`, so those zeros are invalid; the helper uses `/proc` and observes two workers. Final established TCP / FD counts were 398 / 441 (profile) and 385 / 429 (helper). Socket TIME_WAIT entries are separate and are not live worker connections. Read windows are 20 readers × 30 seconds at one request per second.

SECOPS-10 admission subset: **passed**. Full AT31 stays blocked: per-peer counted serving traffic, concurrent sampling/policy/relay/UDP, node process/cgroup telemetry, and candidate hardware are not measured here. The previous counted two-device evidence remains [history](history/follow-up-5/capacity.json).

AT28 limiter subset: **passed**. [Limiter records](limiter.json) retain anonymous refusals with the intervening admin and certified lease responses, metrics and the 8 MiB boundary. This is one actual address, not an IPv6 table-fill simulation. Control follow-up 18 was not merged on measured main.

SECOPS-16: **open**. [Scan evidence](secret-scan.json) includes protected pre-delete streams; a positive recognized device-flow key count keeps the all-stream finding open even when gateway diagnostics are value-free. No key values or matched lines are public.

AT29: finite-grace preparation and D-065 setter remain requested. AT30: unchanged; backup sources did not change since #165. All full AT28–AT32 statuses remain blocked with their exact scope in the case records. See [requests](requests.md) and [run instructions](RUN.md).

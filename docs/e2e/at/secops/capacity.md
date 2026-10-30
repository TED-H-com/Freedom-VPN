# Follow-up 7 — measured lab subsets

Application base `fb3ea32` (#195); harness `443cbb0`; run `ab96acac-8713-485a-9c29-88f20110e2e5`. Shared development host, **not candidate hardware**. Source, build/image digests, policy revision, incarnations, UTC times and commands are in each JSON record.

| Measurement | Actual |
|---|---|
| Simultaneous native cohort | 20 users / 60 devices / 60 bindings |
| Sequential admission wall time | 900.75031848 s |
| Admission p50 / p95 / p99 | 14933.613 / 16653.033 / 17203.99 ms |
| Counted peers during traffic | 60 / 60 |
| Read p50 / p95 / p99 | 9.84 / 132.105 / 292.114 ms |
| Read statuses / wrong member responses | {'200': 5960} / 0 |
| Node telemetry samples / errors / cadence | 82 / 0 / 5 s |
| Node CPU p95 (100%=one core) | 13.208400837956692% |
| Peak memory / open FDs / conntrack | 114642944 bytes / 136 / 1738 |
| Proxy and body-limit subset | passed |
| Paid / grant / lease cutoff and recovery subsets | passed / passed / passed |
| Fleet maximum restored to 86400 via API | True |
| Ordinary log scan / protected artifact lifecycle | passed / passed |

[Capacity](capacity.json) contains per-device bytes, fixed load profile, policies, usage watermarks, gateway inventory and the second-exit attempt. [Node telemetry](node-telemetry.json) contains the full fixed-cadence series. [Deadlines](deadlines.json) retains every fault sample and signed bound. [Limiter](limiter.json) retains exact response sequences and metrics. [D-066 scan](secret-scan.json) separates protected credentials from ordinary streams. The previous run is preserved byte-for-byte under [history/follow-up-6](history/follow-up-6/archive-index.json).

All full AT28–AT32 cases remain **blocked** for the requirements listed in their records; physical hardware/platform/clock rows are **needs operator run**. SECOPS-04's API/config inputs and SECOPS-10's admission facility are delivered; this does not turn unmeasured transports or a refused provider state into passes. AT30 is unchanged; no new physical restore run or RTO/RPO claim. See [requests](requests.md) and [run instructions](RUN.md).

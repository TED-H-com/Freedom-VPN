# Held historical-tracer fault probes

These two separately predeclared fresh-stack diagnostics deliberately hold the historical node tracer across twelve one-second sample intervals after proven admission. They are **not first-selection acceptance**. Each keeps its failed request; no re-grant request is launched after the counted-path prerequisite fails.

| Family / run | Proven admission | First verification baseline relative to first SYN (ms) | Member→node first SYN (ms) | Kernel generation before/after request | Recovery |
|---|---|---|---|---|---|
| [ipv4 / 86153543](86153543-1251-48f4-b368-cf11e5b0939d/measurement.md) | 2026-11-01T13:07:37.952889164Z, generation 5 | -11356.729 … -11307.967 | 104.785 … 104.796 | 0 / 0 | 2026-11-01T13:08:03.708677353Z, generation 5 |
| [ipv6 / a3154028](a3154028-4296-4a61-afba-fee20fc0aa5f/measurement.md) | 2026-11-01T13:09:21.015726709Z, generation 5 | -12349.132 … -12315.910 | 4.293 … 4.300 | 0 / 0 | 2026-11-01T13:09:50.174077862Z, generation 5 |

In each run all five SYNs reach node ingress and the origin has no HTTP receipt. The accepted reports name `safety_check_failed` with `ISOLATION_POLICY_INVALID` at stage `apply`; the exact NFT replay identifies the tracer table, netdev ingress and unscoped `mangle`. The producer first records a failed live verification and successful `verification_ISOLATION_POLICY_INVALID` baseline writes, then failed candidate Applies/rollbacks while the observer remains installed. Published desired and central applied generations do not fall to zero: the node's deny baseline carries the local generation-0 marker. Live lease/member windows are retained in each central record.

The first SYN reaches the IPv4 node in about 105 ms, well inside the original five-second HTTP bound, while the node has already been closed for over eleven seconds. This provides a closure fault case distinct from the natural ce8ccbe1 timeout's delayed native delivery. The normal observer controls omit node marking and have no initial-capture closure. Neither diagnostic substitutes for transport acceptance, nor assigns a cause to historical rows lacking central history.

Millisecond values are rounded; exact bounds, all reported reasons, writer sequences, leases and source-binding/QoS checks remain in the linked JSON records.

# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 22](../handover/follow-up-22.md) (2026-10-31 EDT):** Through #356: shared installer loopback fixed; F8-02 mechanism captured on both underlays; install-overrun cause unproven. D-134 expiry semantics and provisional D-135 node-wide rollback are recorded; historical failures stay failed.
Previous cut: [follow-up 21](../handover/follow-up-21.md), through #351.

## Follow-up 22 acceptance snapshot (2026-10-31 EDT)

Fixed main cutoff `45209609`, through **#356**. Handover ran no new live
acceptance. See [full evidence](../handover/follow-up-22.md) and the
[decision / findings register](../handover/README.md#follow-up-22-decisions-register-2026-10-31-edt).
Devstack 22, members 12, isolation 11, updates 12 and netlab 22 were not merged
at start; their results are excluded. D-135 remains provisional, pending the
owner's confirmation.

| Delivery | Measured scope at the cutoff |
|---|---|
| Devstack 21 (#352) | 114 follow-up 20 files archived; updates F11-01 fixed, shared-wrapper 33b9621a 3 / 0 / 0. DNS cba88bbc 13 / 0 / 0, 36 / 36 complete traces, 85 requests / 70 retained identities; export validated, F9-INFLIGHT not reproduced. Members F11-01 interval bounded; F11-03 install stage established, cause unproven. F18-01 intended D-125 negative; first origin-slot .24 collision retained. |
| Installer 8 (#353) | No behaviour change; field-by-field producer/consumer match and literal-JSON regression. Per-distinct-archive cost corrected; F10-01 CLOSED, privileged 13 / 13. Distinct-endpoint CA / private mirrors, physical arm64, power loss, production remain unmeasured. |
| Handover 21 (#354) | Frozen #351 cut: 21 subtests / 7 rows / 13 snapshots / 350 owner-archive files. |
| Netlab 21 (#355) | 82 prior files archived. F8-02 IPv6 fixture: 19 reverse 1,360-byte packets absent, zero capture drops / no PTB; one mechanism explains both underlays, native capture pending. D-134 reserve [D−101 ms, D], full guard distributions; no predicate change. F11-03 baseline→recovery measured with every loss counted. |
| Agent-core 29 (#356) | Controlled CPU contention does NOT reproduce install overrun: max render→installed 53.133 ms; historical cause unproven. Opt-in TEST ONLY host context, 1,874 Apply brackets audited, D-106 unchanged. Node-wide rollback until next successful Apply, no outage ceiling; S2 later member_path_unconfigured after successful verifier. AT02 stays failed. No D-133 A/B triggered. |

The [operator changes](../handover/operator-guide.md#operator-changes-at-356)
cover D-134 expiry-window semantics and provisional D-135 rollback only.
Updates F11-01 and installer-side F10-01 are closed. F8-02 native capture,
members F11-01 / F11-02 / F11-03, install-overrun cause, F9-INFLIGHT,
ISO-F6-03 writer, Dependabot #22 and production blocked_response_ttl remain
open; F18-01 remains the intended negative. Historical failures stay failed
and unattributed. The snapshots below retain their original cutoffs.

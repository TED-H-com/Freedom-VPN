# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 23](../handover/follow-up-23.md) (2026-11-01 EDT):** Through #362: month-boundary fixture fixed; regular updates pass; members F11-02 closed; native IPv6 F8-02 and five-SYN pattern reproduced. D-135 / D-136 / D-137 remain provisional; historical failures stay failed.
Previous cut: [follow-up 22](../handover/follow-up-22.md), through #356.

## Follow-up 23 acceptance snapshot (2026-11-01 EDT)

Fixed main cutoff `bc011b6f`, through **#362**, after #361. Handover ran no new
live acceptance. See [full evidence](../handover/follow-up-23.md) and the
[decision / findings register](../handover/README.md#follow-up-23-decisions-register-2026-11-01-edt).
Security-ops 14 and netlab 22 / members 13 / devstack 23 / agent-core 31 /
usage AT 7 were absent from main at start; their results are excluded.

| Delivery | Measured scope at the cutoff |
|---|---|
| Agent-core 30 (#357) | Month-boundary TestUsageCheckpointRoundTrip fixed with one 2026-11-01T00:00:00Z clock and period from interval start; 87-file sweep, one fix, six other-lane findings routed. Product validation unchanged. |
| Handover 22 (#358) | Frozen #356 cut: 19 subtests / 5 rows / 22 snapshots; 114-file devstack-20 and 82-file netlab-20 inventories; D-134, provisional D-135. |
| Updates 12 (#359) | 43 files archived; 2babf9b6 regular AT23–AT27 5 / 5 through shared TEST ONLY wrapper, installer 27 / 27. F11-01 closed by regular run; F11-02 closed #351; F18-01 intended negative. First path-bug attempt retained; 0 requests. |
| Isolation 11 (#360) | 100 files archived; 49 historical failed installs replay with host / verification unavailable, cause unproven. edd986c4 AT10 9 / 0 / 0 and AT12 10 / 0 / 0 pass, AT11 / AT13 operator-blocked. Fresh D-104 passes, all losses counted; 952 Applies / zero install overruns, seven intended pre-install rejections. 2 requests. |
| Members 12 (#361) | 43 files archived; b98b78ee F11-02 closed by populated-grant rows, F11-03 blocked (89 Applies succeeded). New F12-01 stale-lease baseline bracket / nine losses, F12-02 timing-export cap, F12-03 negative-cache and F12-04 UDP SERVFAIL. Streamed reader not live-qualified; 12 requests. |
| Devstack 22 (#362) | 63 files archived; native IPv6 F8-02 16,016 / 64,064 bytes, 15 oversized reverse losses, no PTB. Sender constraint passes by fragmentation. Five-SYN pattern reproduced with loopback-source handshake response, no type-4 data. Capture exports; 2 requests. |

[Operator additions](../handover/operator-guide.md#operator-changes-at-362)
cover provisional D-136 stale-lease baseline semantics and D-137 AT09 scope.
D-135 + D-136 require confirmation together; D-137's lower-tunnel-MTU / MSS
hardening remains an owner option. Historical failures stay failed and
unattributed, including F8-02's receive-only silent-drop limitation. The
incident lesson is to derive a half-open interval's month from its own start,
never pair a calendar literal with the wall clock. Earlier snapshots below
retain their cutoffs.

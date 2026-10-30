# Dev-stack owner requests — follow-up 11

Evidence: [follow-up 11 verification](followup11-verification.md), source
`1b2f358` on product base `5555772`, run
`76f0725c-2264-4021-a553-60f6c921e32b`. Only `e2e` / `18000` is exercised.
Previous follow-up 10 records remain archived, including `followup10-merged/`.

## Stale-client revocation ingress — blocked

Owner: **w3-devstack-e2e with w1-netlab** for the isolated ingress fixture.
[Device-revoke](runs/2026-10-30/device-revoke.md) establishes two native counted
flows and a further counted flow while the revoked client's NetBird userspace
service is SIGSTOPped. DELETE is 202; generation 8 projects ready/passed; the
exit agent PID is unchanged. The client retains its `fvnb0` exit route.

The post-apply node capture nevertheless reports `0 packets captured`, and
`ctunku` stays `0 → 0`. Revoked bytes freeze `5262 → 5262`, conntrack is empty,
and origin receipt is zero. Sibling and replacement requests remain counted;
member accept rules remain; client SIGCONT cleanup succeeds. The later node
NetBird inventory no longer contains the revoked peer. That inventory supports
upstream transport withdrawal but does not time its removal. No node forwarding
bug or node-side drop is inferred from zero node ingress.

**Missing primitive:** a controlled tunnel-ingress transport that survives the
remote NetBird peer withdrawal while the actual agent applies its new signed
state, with the old device source attributed at `fvmb0` and read-only named
counter observations. A stale route on the client alone is insufficient. Do not
restore the revoked peer/group/binding or edit enforcement to manufacture a pass.
DELETE-to-first-node-drop remains **not observed**. This is distinct from the
resolved conntrack executable/apply failure and the resolved route-discovery race.
See the [scenario scope](scenarios/device-revoke.md).

## Enrollment binding — ready for harness adoption

Owner: **w4-at-enrollment**. Replace the old `66000` binding with
`FVPN_E2E_PREFIX=w4-enroll FVPN_E2E_PORT_BASE=12000`. Both lower bases are legal;
6000 remains reserved. Tests preserve every existing port/subnet and reject
66000. No other owner's stack was changed or started. The
[table](RUN.md#assigned-port-slots) is authoritative for this lab layout.

## Gateway trust, finite grace and resource hooks — ready for secops

Owner: **w4-at-secops** for the broader AT28/AT29/AT31 campaigns.
The [gateway-principals record](runs/2026-10-30/gateway-principals.md) proves two
real browser members use separate connection principals through the gateway,
and rotating untrusted XFF does not bypass the address limiter, through nginx
or directly to control. Both layers retain the product's default rates. The
agent mTLS listener remains direct; worker_connections is a separate budget.

`FVPN_E2E_GRACE_SECONDS=90` on fresh prepare writes a finite TEST ONLY payment
grace; default 0 is unchanged. Restart both control and worker after changing
an existing members config. The prepare-only CLI readback and product-config
unit checks pass; a live paid-to-grace deadline is not claimed. Record real
provider truth and reconciliation, then a positive counted flow and projected
grace/deadline evidence as described in [RUN.md](RUN.md).
`max_offline_lease_seconds` remains the **w3-wire-nodes** D-065 admin API scope.

`StartNodeTelemetry(ctx, 5*time.Second)` is exported through labstack. The
[member record](runs/2026-10-30/member-traffic.md) has 9 samples; the
[60-peer admission record](runs/2026-10-30/native-scale-hook.md) has 61, both
with zero read errors. Samples include CPU, memory, process FDs and namespace
conntrack counts. Adopt the full series and workload timestamps, not just maxima.
These node-container numbers exclude separate resolver/control containers and
are not a complete mixed-traffic hardware certification.

`PauseNativeControl` is separate from `PauseAgent`: it pauses a member's NetBird
service, preserves kernel transport while possible, and must always be resumed.
The protected native log scan hook and gateway query-redaction replay remain
available. The 60-peer run reports 60 identity matches/authorized peers/protected
scans with zero gateway connection-budget errors or leaked query sentinels.

## Existing hooks and remaining operator work

`StartSecondExit` and `EnrollAt(..., "node2")` remain the real second-node path;
use normal publication APIs rather than a synthetic portal node row.
Both update scenarios retain the real signed candidates and the dedicated
non-member §13.9 peer; this run observes 29 authenticated replies, 2233 bytes in
each probe counter, a maximum 44-second timeout, unchanged nonzero member totals
and empty exception sets after the answer. There are zero probe omissions.

Other boot flavours, physical arm64, physical NAT/network moves, forced relay,
PMTU, wider resolver topology and the full retention/clock/grace campaign remain
with their harness/operator owners. Host clock rollback is never a lab action.
No product, contract or migration workaround is introduced.

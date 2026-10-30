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

## Follow-up 12 — shared harness exports

Live evidence is under
[followup12/e2e](runs/2026-10-30/followup12/e2e/) and
[followup12/e2e-exports](runs/2026-10-30/followup12/e2e-exports/).
Both runs are `TestE2E_HarnessExports` with `FVPN_E2E_EXPORTS=1` on port base
18000. Prefix `e2e` is run `8b97ae7f-ea01-4fba-9a34-930e330684ea` (297s).
Prefix `e2e-exports` is run `fa5650f3-d238-496e-b7d4-718ce705fefb` (286s).
Each summary is 15 pass, 1 blocked, 0 fail, Go exit 0. `source_commit` on
those records is `a36bb84f1dafc7526c021177026b4d558e273cd0`, the merge the
working tree was sitting on when the runs executed. The export fixes in this
follow-up were uncommitted at that moment. `clone-conflict` stays blocked.
The inventory is [RUN.md](RUN.md#follow-up-12-export-inventory).

### Updates F5-02 — ready to consume

Owner: **w4-at-updates**, follow-up 8. Import `SetupUpdaterProbe` and
`UpdaterProbe.Observe` through `tests/e2e/internal/labstack` with `-tags e2e`.
Put `Config` into the real updater's `probe_peer` and `AgentConfig` into
`agent_config`, including a recovery invocation that should exercise health.
Call `Observe` around that invocation. Rerun AT23 and AT25. Do not read the
0700 journal, construct a peer, alias the agent config, or synthesize a
challenge. `configured_probe_issued` is a dev-stack record boolean from the
in-node watcher; it is not a second product exception and it does not copy
the peer ID. Both live `update-good` records have that boolean true, zero
probe omissions, and `probe_traversal_verified` true. An artifact hash failure
still happens before health and cannot prove D-063 issuance.

### Enrollment ENR-04 and ENR-07 — hooks exported

Owner: **w4-at-enrollment**. `JoinEnrolledIdentity` and
`EnrollmentPeerInventory` are the ENR-04 surface. Both live
`export-enrollment` records discarded a 201, recorded 409
`ENROLLMENT_KEY_MISMATCH` for the other key, then replayed the same bytes as
200 enroll and 200 one-time bootstrap with `joined` true. Inventory
`complete` is true: one setup key, `setup_key_uses` 1, one projected peer,
`correlation_groups` 0. The names hash the `enrollment.consumed` audit
operation id. Discovery deletes the correlation group after binding, so zero
groups after that cleanup is the steady state. This does not supply ENR-03
incarnation history.

`MachineIdentityFingerprints` and `StartClonedIdentityFixture` are the ENR-07
fixture. Both live `export-clone` records show equal machine-id, agent SPKI
and NetBird public-key hashes, with the source stopped, durable state copied
before clone boot, and the source resumed. `clone-conflict` remains blocked:
`ObserveEndpoint` has no production caller, so conflict rows stay 0 and
`mapping_frozen` stays false. A successful clone setup is not AT07 safety.
Do not mark AT06 or AT07 passed from these records.

### ENR-05 — usage acknowledgement hold (not implemented)

Owner: **w4-usage** (usage agent and ledger), then devstack once that
contract exists. Enrollment follow-up 13 stays blocked on this.

Confirm the upload endpoint and status that withhold only usage
acknowledgements while production and sampling continue, and an approved
read-only local ledger view. The snapshot must expose immutable event hashes,
decimal sequence and epoch identity, and the actual contiguous ACK watermark,
including the same watermark after a restart. The enrollment harness already
names `HoldUsageAcknowledgements(ctx, slot, hold)`, `UsageLedgerSnapshot(ctx, slot)`
and `CountedMemberTraffic`. `PauseAgent` stops the producer and is not this
hook. Never edit or delete outbox rows to manufacture a hold.

### ENR-06 — installer re-run and controlled reinstall (not implemented)

Owner: **packaging / installer**, then devstack once that invocation exists.
Enrollment follow-up 13 stays blocked on this.

Supply the real installer invocation and the disposable Ubuntu storage and
service prerequisites for an idempotent re-run that preserves identity and
the outbox. Say how a controlled reinstall receives a fresh API-authorized
enrollment after revocation while the stable Node and its history remain.
An old-certificate mTLS probe must stay in private storage; export only the
HTTP/TLS rejection. The enrollment harness already names
`RerunInstaller(ctx, slot)` and `ControlledReinstall(ctx, slot, enrollment)`.
A container restart, key erasure, or copied identity rows are not installer
or reinstall proof.

### SECOPS-16 — per-stream final results

Owner: **w4-at-secops**, follow-up 7. `NativeScale.Streams` now carries each
stream: name, result `scanned`, `missing`, `error` or `not_started`,
`NativeLogScan` including `DeviceFlowKeys`, a fixed safe error class, and
`Deleted`. `JoinNativeMember` keeps an existing scan error class when delete
also fails. Both live `export-native-streams` records have two streams,
mode `0:600`, `device_flow_keys` 1, `deleted` true, the deleted path missing,
and a foreign container refused. File contents are not exported. The
sequential profile already has exact counts through `ScanNativeJoinLog` and
`JoinNativeMember`. Secops still needs to read `Streams` rather than the
totals alone, and to attribute the protected pre-delete login file separately
from logs and captures. A positive device-flow key count remains a positive
count; this export does not close the all-stream zero-canary finding.

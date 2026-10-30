# Follow-up 7 run instructions

Run from this worktree with Go 1.27.3 and Docker:

```sh
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
make -f mk/at-secops.mk at-secops-followup7
make go-lint
go vet -tags e2e ./tests/...
```

The live target refuses an occupied namespace. It fixes `FVPN_E2E_PREFIX=w4-secops`,
`FVPN_E2E_PORT_BASE=30000`, `FVPN_E2E_GRACE_SECONDS=90` and `E2E_RESET=1`, then uses
the existing `mk/e2e.mk` for fresh preparation/start and unconditional teardown.
Grace is a prepare-time TEST ONLY option: both control and worker read it at
startup. It is not live reload. The live test is bounded to 65 minutes.

The test first measures two actual container TCP addresses through the one
trusted gateway. A tiny owned probe runs in those namespaces; credentials are
passed through protected stdin, and only status/code/Retry-After metadata is
returned. It saturates anonymous enrollment from A, requires a valid enrollment
replay HTTP 200 from B, then repeats directly with rotating forged XFF values.
It creates no routes, groups or policies for the proxy test.

The signed serving baseline comes from the exported lab helper. API-created
plans, grants and publication supply the unchanged sequential 20-user × 3-peer
native profile. SQL only observes projections. Each counted peer must select a
product-created exit, route through `fvnb0`, receive HTTPS 200 from the controlled
origin, match its authenticated node-SNAT receipt and increase both exact-device
admission counters. The capacity window is fixed at 60 seconds warmup followed
by 300 seconds measurement: 20 portal readers at 1/s, 60 HTTPS peers at 1/s,
real agent sampling and two API capacity-policy changes. The exported node
sampler runs every five seconds during admission and traffic; the full series,
errors and actual timing jitter are retained. `StartSecondExit` / `EnrollAt`
supply a subsequent two-exit attempt; no forced-relay result is inferred.

AT29 uses `GET /api/v1/admin/fleet-config`, its strong ETag, and `PUT` with
`max_offline_lease_seconds=60`, a TEST ONLY reason and UUID Idempotency-Key.
The same-key replay is observed. A deferred GET/ETag/PUT restores 86400 even if
a case cannot proceed. Existing leases are never rewritten: the record reads
the newly issued signed window before faulting. Source payment snapshots and
finite grants supply service deadlines. A `past_due` source snapshot is attempted
for grace; a provider refusal is retained as a blocked precondition, never
replaced with SQL. During control-loss faults the own worker is paused to exclude
NetBird reconciliation as the cause of cutoff. Cleanup resumes worker, agent
and watchdog and starts control before restoring the fleet maximum. Repeated
HTTPS is separate from established TCP/UDP and guest clock rollback.

D-066 separates the native device-flow credential artifact from ordinary logs,
captures and readable files. The exact `JoinNativeMember` pre-delete scans retain
count, mode/owner and key-line count; a post-join exported scan proves deletion.
Ordinary streams require zero canaries. No values or matched lines are public.
`ScaleNativeMembers` final per-stream attribution remains pending devstack
follow-up 12; the previous aggregate run is not rewritten as complete coverage.

Records are generated after teardown:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tests/e2e/at/secops/followup7_records.py
```

The target returns 2 while any full acceptance case remains blocked. Inspect the
Go live-test result independently. Physical candidate hardware, real member
platforms and guest clock tests remain [operator runs](operator-runs.md). No
physical PITR rerun is required here: backup sources are unchanged since #165.
Previous records are preserved with hashes under `history/follow-up-6/`.
The workflow dependencies are unchanged.

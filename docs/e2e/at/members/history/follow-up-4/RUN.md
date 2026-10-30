# Follow-up 4 run ledger

Assigned instance: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Product tree: `e0a84fb` (merged #170). Harness commit:
`a511927c4962b21cc90d04dc372f11f848cf6478`; attempt 2 and supplements include
owned harness corrections with `source_dirty=true`. Every JSON retains source,
exact image/release/binary hashes and the test executable hash; supplements use
`followup_build`. No product, contract, migration or shared lab source changed.
The workflow is unchanged; no dependency changed.

| Execution | Run UUID | Duration / exit | Result |
|---|---|---|---|
| Fresh attempt 1 | `13ccb8ab-9edf-4b0a-90f9-c3758ad4ec5f` | 936.455 s; Go 1 / make 2 | 2 pass, 7 blocked, 1 fail (AT21 expired recent-auth) |
| Fresh attempt 2 | `d89eda32-84ef-4582-b669-053d5d9abadd` | 1324.862 s; Go 1 / make 2 | 3 pass, 6 blocked, 1 fail (AT09 empty restart lease) |
| Scheduled partial readback | same second stack | 380.413 s; exit 0 | Feed last-good and immutable restart timeline; full cases remain as classified |
| Observation-only readback | same second stack | 12.403 s; exit 0 | No publication/identity/projection repair; no feed mutation |

Both startups succeeded on the first call and both teardowns exited 0. Primary
records are immutable under [attempt 1](history/follow-up-4/attempt-1/attempt.json)
and [attempt 2](history/follow-up-4/attempt-2/attempt.json). The earlier run is
preserved under [follow-up 3](history/follow-up-3/RUN.md).

Current status is **three passed / seven blocked**. AT09's failed recovery check
remains in `prior_checks` and the untouched primary archive; observation-only
reclassification to blocked on F4-01 is not a passing retry. No failed prior
result is silently superseded. See [summary](summary.md), [forwarding](forwarding-observation.md)
and [requests](requests.md).

Between attempts, the harness (a) waits through the documented partial
withdrawal state for the node ACK, (b) reauthenticates through real OIDC per case,
(c) stops all second-exit namespace holders and requires fresh post-restart
serving evidence, (d) measures the old device before revoke and denial again
after disabled restart, and (e) adds paid expiry/suspension TCP cohorts on both
exits. The first attempt's 403, early partial sample, and uncounted restart probe
are retained. No limiter, auth window or D-050 state was changed.

F4-01 is a new observed restart condition: the node receives a fresh empty
same-generation member lease and stays on its baseline; later independent DNS
generation changes restore a populated lease. This is separate from the now
resolved missing conntrack and native exit-selection findings. AT02/AT03 and
ordinary two-exit forwarding are proven; AT09 recovery is not.

## Commands

Run the following lifecycle twice for independent fresh attempts:

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
# Preserve the complete primary JSON/Markdown before supplements.
make -f mk/at-members.mk at-members-readback
FVPN_AT_MEMBERS_OBSERVE_ONLY=1 make -f mk/at-members.mk at-members-readback
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
```

This run used the two supplemental targets on attempt 2 only. The full target
is deliberately non-green for blocked complete cases. Per-AT Markdown pages
are concise summaries of the final JSON; raw assertions/timelines remain there
and primary pages stay untouched in the archives.

## Actual output tails

Attempt 1 startup, exit 0:

```text
 Container fvpn-w4-members-member Starting
 Container fvpn-w4-members-member Started
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
e2e stack is up
```

Attempt 1 suite, make exit 2:

```text
    live_test.go:156: AT21: fail; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:156: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (936.45s)
    --- FAIL: TestATMembers_Live/AT05 (14.69s)
    --- FAIL: TestATMembers_Live/AT01 (103.32s)
    --- PASS: TestATMembers_Live/AT02 (158.21s)
    --- PASS: TestATMembers_Live/AT03 (131.13s)
    --- FAIL: TestATMembers_Live/AT04 (189.12s)
    --- FAIL: TestATMembers_Live/AT08 (162.24s)
    --- FAIL: TestATMembers_Live/AT09 (122.71s)
    --- FAIL: TestATMembers_Live/AT20 (34.49s)
    --- FAIL: TestATMembers_Live/AT21 (15.60s)
    --- FAIL: TestATMembers_Live/AT22 (0.52s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	936.455s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Attempt 2 startup, exit 0:

```text
 Container fvpn-w4-members-adguard Started
 Container fvpn-w4-members-control Waiting
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
e2e stack is up
```

Attempt 2 suite, make exit 2:

```text
    live_test.go:158: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:158: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (1324.85s)
    --- FAIL: TestATMembers_Live/AT05 (16.36s)
    --- FAIL: TestATMembers_Live/AT01 (95.86s)
    --- PASS: TestATMembers_Live/AT02 (165.62s)
    --- PASS: TestATMembers_Live/AT03 (151.90s)
    --- FAIL: TestATMembers_Live/AT04 (430.31s)
    --- PASS: TestATMembers_Live/AT08 (178.79s)
    --- FAIL: TestATMembers_Live/AT09 (225.52s)
    --- FAIL: TestATMembers_Live/AT20 (22.77s)
    --- FAIL: TestATMembers_Live/AT21 (31.00s)
    --- FAIL: TestATMembers_Live/AT22 (1.24s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	1324.862s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Scheduled partial readback, exit 0:

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (380.40s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	380.413s
```

Observation-only readback, exit 0:

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (12.40s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	12.403s
```

`make -f mk/at-members.mk at-members-lint at-members-build at-members-test`
exits 0 (gofmt, tagged vet, pinned staticcheck v0.8.2; linux/amd64 and arm64
probe builds; offline test):

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	(cached)
```

`make go-lint` and `go vet -tags e2e ./tests/...` both exit 0 with empty stdout.
There are no application/frontend/dependency changes. The shared startup builds
the actual Portal, and AT03 uses its existing browser probe.

Attempt 1 teardown, exit 0:

```text
 Network fvpn-w4-members Removing
 Network fvpn-w4-members-origin Removing
 Network fvpn-w4-members Resource is still in use
 Network fvpn-w4-members-origin Resource is still in use
 Volume fvpn-w4-members-nb Removed
 Volume fvpn-w4-members-pgbackrest-repo Removed
 Volume fvpn-w4-members-pg Removed
e2e stack is down
```

Attempt 2 teardown, exit 0:

```text
 Network fvpn-w4-members Removing
 Network fvpn-w4-members-origin Removing
 Network fvpn-w4-members Resource is still in use
 Network fvpn-w4-members-origin Resource is still in use
 Volume fvpn-w4-members-pg Removed
 Volume fvpn-w4-members-nb Removed
 Volume fvpn-w4-members-pgbackrest-repo Removed
e2e stack is down
```

Compose initially reports its bridges in use by the hook-created second exit;
the same shared teardown then removes all extra prefix objects. Final verified
inventory (after a network-disabled own-prefix container removes private
root-owned generated state):

```json
{
  "containers": 0,
  "networks": 0,
  "volumes": 0,
  "images": 0
}
```

Neutral shared build caches are retained. No host networking, other prefix,
shared admin database, push, PR or merge was touched.

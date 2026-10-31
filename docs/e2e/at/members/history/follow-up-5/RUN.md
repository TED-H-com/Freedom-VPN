# Follow-up 5 run ledger

Assigned instance: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Product tree: `dde4aa6` (merged #193, D-067 provisional grant-less leases).
Harness commit: `d6f49fd228f535ad45a7a9c4625cf933ae100216`; the primary suite
ran with a clean tree (`source_dirty=false`, `vcs.modified=false`), and the
scheduled readback target recorded `followup_build` (`source_dirty=true` only
because the primary suite had just written the new `docs/e2e/at/members/AT*.json`
evidence files). Every JSON retains source, exact image/release/binary hashes
and the test executable hash. No product, contract, migration or shared lab
source changed. The workflow is unchanged; no dependency changed.

| Execution | Run UUID | Duration / exit | Result |
|---|---|---|---|
| Fresh complete suite (`at-members`) | `16aa2d1b-d46f-4e88-a509-a8a9c9076006` | 1233.987 s; Go 1 / make 2 | 3 pass (`AT02`, `AT03`, `AT08`), 7 blocked (`AT01`, `AT04`, `AT05`, `AT09`, `AT20`, `AT21`, `AT22`), 0 failed subchecks (382 passing subchecks) |
| Scheduled partial readback (`at-members-readback`) | same stack | 380.181 s; exit 0 | Feed last-good (`AT22`), second-exit privacy (`AT22`), and immutable restart lease/observation timeline (`AT09`); full cases remain as classified |

Startup (`e2e-up`) succeeded on the first call and teardown (`e2e-down`) exited
0. Previous Follow-up 4 records are preserved under
[history/follow-up-4](history/follow-up-4/RUN.md) and Follow-up 3 under
[history/follow-up-3](history/follow-up-3/RUN.md).

Current status is **three passed / seven blocked**, with **zero failed
subchecks**. **F4-01 is resolved by PR #193 (D-067):** after stopping and
starting `node2`, `node2-adguard` and `node2-netns` at
`2026-10-30T19:36:50.392336111Z`, central issues three provisional zero-grant
leases on unchanged desired generation `25` (`19:36:52.908587+00:00`,
`19:36:58.749086+00:00`, `19:37:08.853644+00:00`) while `node2` is not yet
`ready`, and then issues a 7-grant lease on generation `25` at
`19:37:29.064693+00:00` once `safety_status: "passed"` at `19:37:24.045812Z`
transitions `node2` to `ready`. Serving readiness (`18` kernel
`member_accept_rules` and accepted `forwarding: true` observation) is restored
in **`39.236 s`** (`19:37:29.628457149Z`), and the first counted member byte
arrives in **`43.982 s`** (`19:37:34.374304456Z`, `device_upload_bytes: "0" -> "2305"`,
`device_download_bytes: "0" -> "2411"`, `fvmb0` `10 to_origin / 7 from_origin`)
on the same desired generation `25` with no policy change or admin action. See
[summary](summary.md), [forwarding](forwarding-observation.md) and
[requests](requests.md).

## Commands

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
make -f mk/at-members.mk at-members-readback
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
```

The full target is deliberately non-green for blocked complete cases. Per-AT
Markdown pages are concise summaries of the final JSON; raw assertions and
timelines remain in `AT*.json`.

## Actual output tails

Startup (`e2e-up`), exit 0:

```text
 Container fvpn-w4-members-adguard Started
 Container fvpn-w4-members-control Waiting
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
e2e stack is up
```

Complete suite (`make -f mk/at-members.mk at-members`), make exit 2 (3 passed, 7 blocked, 0 failed subchecks):

```text
    live_test.go:165: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:165: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (1233.97s)
    --- FAIL: TestATMembers_Live/AT05 (16.45s)
    --- FAIL: TestATMembers_Live/AT01 (96.12s)
    --- PASS: TestATMembers_Live/AT02 (162.81s)
    --- PASS: TestATMembers_Live/AT03 (140.42s)
    --- FAIL: TestATMembers_Live/AT04 (466.15s)
    --- PASS: TestATMembers_Live/AT08 (126.34s)
    --- FAIL: TestATMembers_Live/AT09 (170.61s)
    --- FAIL: TestATMembers_Live/AT20 (18.41s)
    --- FAIL: TestATMembers_Live/AT21 (32.18s)
    --- FAIL: TestATMembers_Live/AT22 (1.32s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	1233.987s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Scheduled partial readback (`make -f mk/at-members.mk at-members-readback`), exit 0:

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (380.17s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	380.181s
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

Teardown (`e2e-down`), exit 0:

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

Compose initially reports its bridges in use by the hook-created second exit and
extra native peers; the same shared teardown then removes all extra prefix
objects. Final verified inventory (after a network-disabled own-prefix container
removes private root-owned generated state):

```json
{"containers": 0, "networks": 0, "volumes": 0, "images": 0}
```

Neutral shared build caches are retained. No host networking, other prefix,
shared admin database, push, PR or merge was touched.

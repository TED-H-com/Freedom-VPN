# Follow-up 6 run ledger

Assigned instance: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Product tree: `9c37a38` (merged #219 `device-revoke` `ObserveMemberIngress` export, #213 & #216 contract revision 11 `awaiting_member_grants` / `AWAITING_MEMBER_GRANTS`, #193 D-067 provisional grant-less leases).
Harness commit: `159d6171739a07c645a4c1cb6e208cf1797e8234`; the primary suite
ran with a clean tree (`source_dirty=false`, `vcs.modified=false`), and the
scheduled readback target recorded `followup_build` (`source_dirty=true` only
because the primary suite had just written the new `docs/e2e/at/members/AT*.json`
evidence files and updated the readback worker check). Every JSON retains source, exact image/release/binary hashes
and the test executable hash. No product, contract, migration or shared lab
source changed. The workflow is unchanged; no dependency changed.

| Execution | Run UUID | Duration / exit | Result |
|---|---|---|---|
| Fresh complete suite (`at-members`) | `80db103b-622c-42b1-946d-4628daaa213d` | 1305.124 s; Go 1 / make 2 | 3 pass (`AT02`, `AT03`, `AT08`), 7 blocked (`AT01`, `AT04`, `AT05`, `AT09`, `AT20`, `AT21`, `AT22`), 0 failed subchecks (384 passing subchecks across suite + readback) |
| Scheduled partial readback (`at-members-readback`) | same stack | 379.384 s; exit 0 | Feed last-good (`AT22`), second-exit privacy (`AT22`), and immutable restart lease/observation timeline (`AT09`); full cases remain as classified |

Startup (`e2e-up`) succeeded on the first call and teardown (`e2e-down`) exited
0. Previous Follow-up 5 records are preserved under
[history/follow-up-5](history/follow-up-5/RUN.md), Follow-up 4 under
[history/follow-up-4](history/follow-up-4/RUN.md), and Follow-up 3 under
[history/follow-up-3](history/follow-up-3/RUN.md).

Current status is **three passed / seven blocked**, with **zero failed
subchecks**.
- **AT02 strict post-withdrawal ingress (#219):** after real member DELETE of `member-a2` (`01a18efa-99e8-75d7-9a6c-47f05b5c55e9`, `100.104.160.29`) and `SyncDevice` convergence on generation `8`, `ObserveMemberIngress` on `fvmb0` (`tun-mirred`, link type `65534`, source port `40000`, destination `192.0.4.10:9`) proves that an injected frame from the revoked member source increments `ctunku` from `{packets: 0, bytes: 0}` to `{packets: 1, bytes: 29}` (`+1` packet, `+29` bytes) with `0` forwarded bytes (`uplink_empty: true`, `origin_empty: true`) and leaves all device counters unchanged, while an injected frame from the admitted sibling `member` (`01a18ef9-ae93-7a1c-b934-20bed403bf2f`, `100.104.242.187`) increments `u_01a18ef9ae937a1cb93420bed403bf2f` from `{packets: 31, bytes: 6967}` to `{packets: 32, bytes: 6996}` (`+1` packet, `+29` bytes) and `d_01a18ef9ae937a1cb93420bed403bf2f` from `{packets: 22, bytes: 7283}` to `{packets: 23, bytes: 7340}` (`+1` packet, `+57` bytes on origin ICMP reply), appears on both `fvup0` and `origin`, and leaves `ctunku` and all other device counters unchanged.
- **AT09 in D-067 / revision 11 terms (#193, #213, #216):** after stopping and
  starting `node2`, `node2-adguard` and `node2-netns` at
  `2026-10-30T23:00:03.796010637Z`, central issues three provisional zero-grant
  leases with 1-minute TTL on unchanged desired generation `23` (`23:00:04.349901+00:00`,
  `23:00:09.419186+00:00`, `23:00:19.549439+00:00`) while `node2` is not yet
  `ready`, during which observations (`23:00:04.375940436Z` and `23:00:34.753993516Z`)
  report `forwarding_block_reasons: ["awaiting_member_grants"]` (plus `safety_check_failed`
  initially; `saw_lease_expired: false`), `error_code: "AWAITING_MEMBER_GRANTS"` on
  `lease`, `qos`, and `source_binding` checks, and `lease_stage_errors_count: 1`,
  and then issues a 7-grant lease on generation `23` at `23:00:39.805556+00:00`
  once `safety_status: "passed"` at `23:00:34.753993516Z` transitions `node2` to
  `ready`. Serving readiness (`18` kernel `member_accept_rules` and accepted
  `forwarding: true` observation) is restored in **`36.976 s`**
  (`23:00:40.771707223Z`), and the first counted member byte arrives in
  **`41.695 s`** (`23:00:45.491347556Z`, `device_upload_bytes: "0" -> "2305"`,
  `device_download_bytes: "0" -> "2461"`, `fvmb0` `11 to_origin / 7 from_origin`)
  on the same desired generation `23` with no policy change or admin action. See
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
    live_test.go:167: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:167: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (1305.11s)
    --- FAIL: TestATMembers_Live/AT05 (16.29s)
    --- FAIL: TestATMembers_Live/AT01 (96.08s)
    --- PASS: TestATMembers_Live/AT02 (193.54s)
    --- PASS: TestATMembers_Live/AT03 (143.19s)
    --- FAIL: TestATMembers_Live/AT04 (511.23s)
    --- PASS: TestATMembers_Live/AT08 (126.15s)
    --- FAIL: TestATMembers_Live/AT09 (167.21s)
    --- FAIL: TestATMembers_Live/AT20 (18.21s)
    --- FAIL: TestATMembers_Live/AT21 (31.89s)
    --- FAIL: TestATMembers_Live/AT22 (1.32s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	1305.124s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.003s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Scheduled partial readback (`make -f mk/at-members.mk at-members-readback`), exit 0:

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (379.38s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	379.384s
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

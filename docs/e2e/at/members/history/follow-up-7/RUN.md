# Follow-up 7 run ledger

Assigned instance: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Task assigned 2026-10-30 EDT; final execution took place on 2026-10-31 UTC/EDT.
Product baseline: `d9e28f8`, including devstack #240, wire-members #244 and the D-081 contract erratum #247. Harness source: `10a5f3891aa8fde2d3a2f158d97bfcb0a4c7670b`.
The primary suite started from a clean committed tree (`build.source_dirty=false`). The same-stack readback records its own `followup_build`; generated acceptance records account for the dirty tree at that point. Every JSON retains exact source, test executable, container image, release and binary hashes.

| Execution | Run UUID | Duration / exit | Result |
|---|---|---|---|
| Fresh complete suite (`at-members`) | `297faa31-2f90-4aab-8cf0-6818629f46c8` | 1416.798 s; Go 1 / make 2 | 3 passed / 7 blocked / 0 failed complete cases |
| Scheduled partial readback (`at-members-readback`) | same stack/run | 380.125 s; exit 0 | 471 passing / 0 failed subchecks across primary + readback; original complete-case classifications retained |

The full target deliberately exits nonzero while complete cases remain blocked. Current results are in [summary.md](summary.md), with the remaining F4-03/04/05/06/07/10 resources in [BLOCKERS.md](BLOCKERS.md).

## D-077 / D-081

[D077.md](D077.md) indexes every recorded response, status, operation ID and fence count. The normal path and all specified negatives are executed on the running production HTTP API. A restore-style fence insert is 0→1; denied mutations retain it; an explicit successful re-grant is 1→0. The original authorized-device grant returns the existing operation, confirmed by unchanged operation identity/pre-bump generation. Worker device-list readback and fresh counted HTTPS establish completion. Same-key replay preserves the operation and device version. The device-limit response is 409 `DEVICE_LIMIT_REACHED`, matching D-081 and #247.

Fenced native HTTPS has no authenticated origin receipt, no fvmb0 arrival and no device counter increase after route withdrawal. This is separate from AT02's explicit revoked-source ingress injection and counted sibling control. Neither an old succeeded operation nor a 202 alone supplies forwarding evidence.

## AT09 / D-067

Same-generation restart serving readiness is `2026-10-31T04:27:04.060261733Z`, **36.997 seconds** after start, compared with follow-up 6's **36.976 seconds** (+0.021 seconds). First counted byte: `2026-10-31T04:27:08.792505245Z` (**41.729 seconds**). Desired generation remains `24`. AT09.json retains the immutable lease/observation timeline, provisional zero-grant leases, `awaiting_member_grants`, one lease-stage `AWAITING_MEMBER_GRANTS`, and no `lease_expired`.

This is measured lab recovery; full AT09 remains blocked on F4-03. It does not establish physical second-NAT or forced relay/QUIC/PMTU coverage.

## Retained attempts and fixture corrections

- [Follow-up 6](history/follow-up-6/RUN.md) is preserved verbatim. Its 36.976-second result is used only as a comparison, never as current pass evidence.
- [Attempt 1](history/follow-up-7/attempt-1/RUN.md), `63a07f95-6a73-4f6c-95d9-f10a5a22de8c`, stopped at a harness bootstrap-admin ID lookup failure. The fixture now resolves the actor from the current authenticated admin session without publishing credentials.
- [Attempt 2](history/follow-up-7/attempt-2/RUN.md), `c8939ad8-ec4f-413a-aac3-1a4bf5d35b32`, exposed an inherited synthetic positive usage event racing the live meter sequence. The server correctly rejected the collision. Positive replay now uses an actual accepted payload and asserts unchanged receipts, samples and conflict count. The synthetic unknown-device negative runs during a bounded supervisor stop and the same release resumes afterward.

Both interrupted attempts were torn down and reset. Their records are historical; every current pass comes from the final run. Readback now preserves primary failures instead of demoting them to blocked.

## Commands

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
make -f mk/at-members.mk at-members-readback
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
make -f mk/at-members.mk at-members-lint at-members-build at-members-test
make go-lint
mkdir -p .agent/tmp
TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp" CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
make go-test
```

The area targets create and remove unique disk-backed scratch under ignored `.agent/tmp`. The primary timeout is 40 minutes (internal context 35 minutes); the assigned-stack CI job is 60 minutes. No application, contract, migration, dependency or shared lab source changed.

## Actual output tails

Fresh startup, exit 0:

```text
 Container fvpn-w4-members-member Starting
 Container fvpn-w4-members-member Started
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
e2e stack is up
```

Complete live target, make exit 2:

```text
=== RUN   TestATMembers_Live/AT20
    live_test.go:161: AT20: blocked; see docs/e2e/at/members/AT20.json
=== RUN   TestATMembers_Live/AT21
    live_test.go:161: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:161: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (1416.79s)
    --- FAIL: TestATMembers_Live/AT05 (15.38s)
    --- FAIL: TestATMembers_Live/AT01 (92.35s)
    --- PASS: TestATMembers_Live/AT02 (297.67s)
    --- PASS: TestATMembers_Live/AT03 (197.39s)
    --- FAIL: TestATMembers_Live/AT04 (471.02s)
    --- PASS: TestATMembers_Live/AT08 (141.83s)
    --- FAIL: TestATMembers_Live/AT09 (143.08s)
    --- FAIL: TestATMembers_Live/AT20 (14.11s)
    --- FAIL: TestATMembers_Live/AT21 (35.81s)
    --- FAIL: TestATMembers_Live/AT22 (1.25s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	1416.798s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.001s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Scheduled readback, exit 0:

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (380.12s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	380.125s
```

Area lint, amd64/arm64 probe builds and offline tests, exit 0:

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	(cached)
```

Whole-module `make go-lint` and `go vet ./...` each exited 0 with empty stdout. `make go-test` exited 0:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.172s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.169s
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/at	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab	[no test files]
```

Teardown, exit 0:

```text
 Volume fvpn-w4-members-nb Removing
 Volume fvpn-w4-members-pgbackrest-repo Removing
 Network fvpn-w4-members-origin Removing
 Network fvpn-w4-members Removing
 Network fvpn-w4-members Resource is still in use
 Network fvpn-w4-members-origin Resource is still in use
 Volume fvpn-w4-members-pg Removed
 Volume fvpn-w4-members-pgbackrest-repo Removed
 Volume fvpn-w4-members-nb Removed
e2e stack is down
```

Compose can first report bridges still in use by extra native peers/second exit; the shared teardown subsequently removes those assigned-prefix objects. Final inventory verifies zero assigned-prefix containers, networks, volumes and images. Generated secret-bearing state and scratch were removed using only the assigned worktree and an assigned-prefix, network-disabled cleanup container where needed. Neutral shared build caches and the shared test database were left intact. No push, PR, main rewrite or host networking change was performed.

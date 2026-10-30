# Follow-up 1 live run

Fresh assigned stack: `w4-members / 24000`. Product binaries were built from
`629766c18d1bdd63a336972f95ae2b41734a242b` with Go 1.27.3 and `vcs.modified=false`.
The main acceptance code was commit `3262e1a`; supplemental readbacks used
`8a90082` and `483d1ce` on the same images and node incarnation. The JSON records
retain all image and binary digests, UTC step times, applied generation and
policy references. Supplemental `source_dirty=true` reflects generated evidence
and documentation, not application edits.

**115 final subchecks passed; all ten full ATs remain blocked.** No platform,
arm64 or second-NAT execution is claimed. See [README.md](README.md) for the AT
table, [AT02-TRACE.md](AT02-TRACE.md) for the resolved native admission path,
and [BLOCKERS.md](BLOCKERS.md) for current owner requests.

Native subject linkage, worker projection, final-slot race, cross-owner existing
device access, revoke-slot reuse, real-binding usage replay, hosted checkout,
webhook replay/order/signature checks, grant expiry and suspension all ran live.
AT04's expiry projection changed nine seconds after the real finite interval
ended; this is not a measured TCP/UDP termination time.

The node-mTLS desired-state gate returned 503 `DEPENDENCY_UNAVAILABLE` with
“Release support policy required.” The active support-policy count was zero,
node peer ID null, applied generation zero, and NetBird route count zero.
This is an observed baseline block, not a failed claim about #75/#77 readiness.

The feed worker fetched a pinned self-authored TEST ONLY source. After the real
300-second cadence plus scheduling sweep, an empty update produced `zero_rules`
and preserved one content row and the original hash. Generated DNS settings
were restored to 21600 seconds. A null-attribution profile reference exposed
B9; an explicitly attributed revision/profile published successfully. Node
resolver-document delivery still awaits contracts-r5 and nodes/agent follow-ups.

Fresh `e2e-up`: exit 0. Only one stack startup was attempted.

```text
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
e2e stack is up
```

Full suite: `make -f mk/at-members.mk at-members`, exit 2 (Go test exit 1).
Blocked full ATs deliberately remain non-green:

```text
=== RUN   TestATMembers_Live/AT22
    live_test.go:135: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (203.64s)
    --- FAIL: TestATMembers_Live/AT05 (4.05s)
    --- FAIL: TestATMembers_Live/AT01 (23.37s)
    --- FAIL: TestATMembers_Live/AT02 (49.41s)
    --- FAIL: TestATMembers_Live/AT03 (19.91s)
    --- FAIL: TestATMembers_Live/AT04 (27.30s)
    --- FAIL: TestATMembers_Live/AT08 (44.58s)
    --- FAIL: TestATMembers_Live/AT09 (0.05s)
    --- FAIL: TestATMembers_Live/AT20 (27.08s)
    --- FAIL: TestATMembers_Live/AT21 (4.26s)
    --- FAIL: TestATMembers_Live/AT22 (0.52s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	203.650s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	[no test files]
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

The initial supplemental run exposed the null-attribution profile prerequisite
as a failed combined publication assertion (exit 2, 363.26 seconds). It did
complete the scheduled invalid-update experiment. The assertion was separated
into the observed HTTP prerequisite and an explicit-attribution positive
control; the existing feed state was then inspected without changing its clock
or any projection. Final diagnostic command:

```sh
FVPN_AT_MEMBERS_FEED_READBACK=1 make -f mk/at-members.mk at-members-readback
```

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (6.55s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	6.556s
```

That diagnostic pass does not change any complete AT to passed.

Verification: `make go-lint`, `go vet -tags e2e ./tests/...`, area
lint/staticcheck/test and amd64/arm64 probe builds all exit 0. Root lint and vet
produce no stdout. Whole-module `go test -p 1 -parallel 2 ./...` exits 0:

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-nbbootstrap	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-origin	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.007s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/at	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab	[no test files]
```

`e2e-down` exits 0. Compose initially reports the bridge still in use by the
extra clients; the stock prefix cleanup then removes those clients and bridge.
The test cleanup helper was also made independent of the canceled parent
context for future runs. No host network changes were made.

```text
 Container fvpn-w4-members-netbird Removed
 Volume fvpn-w4-members-nb Removing
 Volume fvpn-w4-members-pg Removing
 Network fvpn-w4-members-origin Removing
 Network fvpn-w4-members Removing
 Network fvpn-w4-members Resource is still in use
 Volume fvpn-w4-members-nb Removed
 Volume fvpn-w4-members-pg Removed
 Network fvpn-w4-members-origin Removed
e2e stack is down
```

Private generated state was deleted with a prefix-named, network-disabled
cleanup container; its image tag was removed. Final inventory:

```json
{"containers": 0, "networks": 0, "volumes": 0, "images": 0}
```

The reported tests used no shared PostgreSQL admin database; all live SQL was
against this stack's own database. No projections were seeded (D-050), and no
application or lab source was modified. No push or merge was performed.

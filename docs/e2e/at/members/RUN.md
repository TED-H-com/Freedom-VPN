# Verified live run

Fresh assigned stack: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Source under test: `4e500392333b00caf380ca155b7e2c722447044d`. Per-AT JSON records
contain service binary SHA-256/VCS settings, 12 runtime image IDs, UTC times,
actual node incarnation and policy references. Build `vcs.modified=true` is
reported because generated evidence/documentation changed in the worktree;
there are no application edits. Evidence is synthetic and de-identified.

**75 executed subchecks succeeded; all 10 complete ATs remain blocked.**
The target deliberately exits nonzero for incomplete acceptance. This is not a
claim that device quota, paid reconciliation or forwarding passed. Native and
physical requirements remain needs operator run; see OPERATORS.md.

Fresh `e2e-up` exited 0:

```text
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
e2e stack is up
```

`make -f mk/at-members.mk at-members` exited 2 (Go test exited 1):

```text
--- FAIL: TestATMembers_Live (131.02s)
    --- FAIL: TestATMembers_Live/AT05 (4.05s)
    --- FAIL: TestATMembers_Live/AT01 (4.85s)
    --- FAIL: TestATMembers_Live/AT02 (4.30s)
    --- FAIL: TestATMembers_Live/AT03 (18.76s)
    --- FAIL: TestATMembers_Live/AT04 (15.07s)
    --- FAIL: TestATMembers_Live/AT08 (44.62s)
    --- FAIL: TestATMembers_Live/AT09 (0.05s)
    --- FAIL: TestATMembers_Live/AT20 (27.06s)
    --- FAIL: TestATMembers_Live/AT21 (4.26s)
    --- FAIL: TestATMembers_Live/AT22 (0.52s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	131.028s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	[no test files]
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Verification: `make go-lint` exit 0; `go vet -tags e2e ./tests/...` exit 0
(both produced no stdout); area vet/staticcheck, offline check and amd64/arm64
probe builds exit 0. Whole-module `go test -p 1 -parallel 2 ./...` exit 0:

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-origin	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.005s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab	[no test files]
```

Assigned `e2e-down` exited 0:

```text
 Network fvpn-w4-members Removing
 Network fvpn-w4-members-origin Removing
 Volume fvpn-w4-members-nb Removed
 Volume fvpn-w4-members-pg Removed
 Network fvpn-w4-members Removed
 Network fvpn-w4-members-origin Removed
e2e stack is down
```

Generated private state was removed with an isolated, prefix-named cleanup
container, which was also removed along with its temporary image tag. Final
prefix inventory (actual command result):

```json
{"containers": 0, "networks": 0, "volumes": 0, "images": 0}
```

No application/lab changes, host network changes, production values, push or
merge into main. Full owner-specific requests: [BLOCKERS.md](BLOCKERS.md).

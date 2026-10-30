# Follow-up 3 live run

Binding: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Base main was `b8aa690`; first harness commit is `ac93203dbabc05aa3c1c70f644370a205a704f16`.
The final full run used that source plus the owned harness corrections described
below. Every record retains source-dirty state, exact runtime image IDs, binary
VCS metadata, test executable SHA-256 and probe SHA-256 where used. Product code
was unchanged. Supplemental builds are separate `followup_build` fields.

| Attempt | Run UUID | Result | Evidence |
|---|---|---|---|
| Fresh stack 1 | `c2eb408c-7d5e-4ef0-893b-348f101a448e` | Interrupted, make exit 2 / recipe 143: harness fixture-key collision | [attempt record](history/follow-up-3/attempt-1/attempt.json) |
| Fresh stack 2 | `f81a8365-7839-4756-bee2-9bb2a5506cdd` | All ten cases ran, 536.259 s; make exit 2 / Go exit 1 | [primary records](history/follow-up-3/attempt-2/attempt.json) |
| Scheduled partial readback | same second stack/images/incarnation | 379.446 s, exit 0 | Current AT02/AT08/AT20/AT22 JSON |
| Final observation-only readback | same second stack/images/incarnation | 12.254 s, exit 0 | Current AT02/AT03/AT04/AT08/AT20/AT22 JSON |

Both fresh startups succeeded on their first attempt. The first suite was stopped
when its full UUID made payment test usernames exceed the shared IdP helper's
40-character key boundary, dropping the order suffix and reusing a member.
This is a harness failure; no provider/product defect is inferred. Its completed
AT01/AT02/AT05 and incremental AT03 records are preserved. Other ATs did not run
on that attempt. The fix keeps a short fixture key separately from the full run
UUID and asserts unique members for all six payment-order permutations.

The first attempt also retained unsuccessful immediate native route selection.
The second waits for an advertised route before selecting through the native CLI.
It proves counted initial member traffic and pause/unpause recovery. The first
real revocation then exposes the missing `conntrack` image dependency: later
packets reach node ingress but hit its fail-closed baseline. Full scope and
owners are in [summary.md](summary.md), [forwarding-observation.md](forwarding-observation.md)
and [requests.md](requests.md). No package or projection workaround was used.

The primary second run initially classified AT02 as passed, AT08 as failed and
eight cases as blocked. The subsequent immutable apply-failure evidence makes
complete AT02 blocked after revocation; its original positive checks are intact.
AT08's 422 after restart is retained in `prior_checks` and the primary archive.
The partial path never upgrades it to a full pass. Current records contain 242
passing subchecks; AT08 additionally retains 30 passing and one failed prior
checks. All ten complete ATs remain blocked on stated conditions/operator scope.

Harness corrections after the primary run: publication polling waits for a
terminal operation rather than its retryable intermediate failed step; typed
resolver evidence waits for the current issued profile; readback links the
post-revoke packet trace and missing dependency. The supplemental path is now
read-only apart from the explicitly requested scheduled feed experiment.
No agent-only restart workaround or sidecar restart is used.

Aggregate packet observers initially returned early when `poll` was interrupted;
actual `elapsed_ms` is retained, so silence is not treated as proof of absence.
The observer now resumes on EINTR. Its final container smoke ran for
9034 ms. Positive
HTTPS responses plus device counters prove forwarding independently of these
shortened capture windows. No raw packet or credential was persisted.

## Commands and actual output tails

Run from the repository root:

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
make -f mk/at-members.mk at-members-readback
FVPN_AT_MEMBERS_OBSERVE_ONLY=1 make -f mk/at-members.mk at-members-readback
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
```

Second startup, exit 0:

```text
 Container fvpn-w4-members-control Waiting
 Container fvpn-w4-members-member Starting
 Container fvpn-w4-members-member Started
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
e2e stack is up
```

First full-suite attempt (interrupted):

```text
=== RUN   TestATMembers_Live
=== RUN   TestATMembers_Live/AT05
    live_test.go:150: AT05: blocked; see docs/e2e/at/members/AT05.json
=== RUN   TestATMembers_Live/AT01
    live_test.go:150: AT01: blocked; see docs/e2e/at/members/AT01.json
=== RUN   TestATMembers_Live/AT02
=== RUN   TestATMembers_Live/AT03
Terminated
make: *** [mk/at-members.mk:13: at-members] Error 143
```

Second full-suite attempt, make exit 2:

```text
    live_test.go:153: AT20: blocked; see docs/e2e/at/members/AT20.json
=== RUN   TestATMembers_Live/AT21
    live_test.go:153: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:153: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (536.25s)
    --- FAIL: TestATMembers_Live/AT05 (16.59s)
    --- FAIL: TestATMembers_Live/AT01 (90.58s)
    --- PASS: TestATMembers_Live/AT02 (82.80s)
    --- FAIL: TestATMembers_Live/AT03 (164.41s)
    --- FAIL: TestATMembers_Live/AT04 (74.29s)
    --- FAIL: TestATMembers_Live/AT08 (85.31s)
    --- FAIL: TestATMembers_Live/AT09 (0.66s)
    --- FAIL: TestATMembers_Live/AT20 (12.75s)
    --- FAIL: TestATMembers_Live/AT21 (4.22s)
    --- FAIL: TestATMembers_Live/AT22 (0.49s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	536.259s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.001s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Scheduled partial readback, exit 0:

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (379.44s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	379.446s
```

Observation-only readback, exit 0:

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (12.25s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	12.254s
```

`make -f mk/at-members.mk at-members-lint at-members-build at-members-test`
exits 0; gofmt, vet, pinned staticcheck v0.8.2 and linux/amd64+arm64 builds pass.
`make go-lint` and `go vet -tags e2e ./tests/...` exit 0 with empty stdout.
No dependency or workflow change was needed.

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	(cached)
```

Teardown exits 0. Private generated state was removed using an own-prefix,
network-disabled cleanup container. Shared neutral build caches are retained.

```text
 Volume fvpn-w4-members-pgbackrest-repo Removing
 Volume fvpn-w4-members-nb Removing
 Network fvpn-w4-members-origin Removing
 Network fvpn-w4-members Removing
 Volume fvpn-w4-members-pg Removed
 Volume fvpn-w4-members-pgbackrest-repo Removed
 Volume fvpn-w4-members-nb Removed
 Network fvpn-w4-members-origin Removed
 Network fvpn-w4-members Removed
e2e stack is down
```

Final assigned-prefix inventory:

```json
{"containers": 0, "images": 0, "networks": 0, "volumes": 0}
```

No host network change, shared database mutation, push or merge was performed.
The previous follow-up is preserved under [history/follow-up-2](history/follow-up-2/RUN.md).

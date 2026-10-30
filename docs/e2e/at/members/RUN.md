# Follow-up 2 live run

Final fresh stack: `w4-members / 24000`, product and main harness source
`4ebf248f02d95da663223d2b922e8a8f21c59a6c`. Binary build metadata records
Go 1.27.3 and `vcs.modified=false`. Supplemental checks used `b43613d` and
`75e7e33` on those same image digests and node incarnation. Each record retains
its original build plus the supplemental build identity; source-dirty on a
supplement reflects evidence/doc edits, not product modifications.

**202 current subchecks pass, zero current subchecks fail; all ten complete ATs
remain blocked.** AT08 also retains 29 prior checks, including three failed
assertions, with their original UTC times in `prior_checks`. No platform,
physical arm64, second physical NAT or complete acceptance pass is claimed.

| AT | Final status | Evidence |
|---|---|---|
| AT01 | blocked / operator subsets: needs operator run | [AT01.json](AT01.json), [AT01.md](AT01.md) |
| AT02 | blocked | [AT02.json](AT02.json), [AT02.md](AT02.md) |
| AT03 | blocked | [AT03.json](AT03.json), [AT03.md](AT03.md) |
| AT04 | blocked | [AT04.json](AT04.json), [AT04.md](AT04.md) |
| AT05 | blocked / operator subsets: needs operator run | [AT05.json](AT05.json), [AT05.md](AT05.md) |
| AT08 | blocked | [AT08.json](AT08.json), [AT08.md](AT08.md) |
| AT09 | blocked / operator subsets: needs operator run | [AT09.json](AT09.json), [AT09.md](AT09.md) |
| AT20 | blocked | [AT20.json](AT20.json), [AT20.md](AT20.md) |
| AT21 | blocked / operator subsets: needs operator run | [AT21.json](AT21.json), [AT21.md](AT21.md) |
| AT22 | blocked | [AT22.json](AT22.json), [AT22.md](AT22.md) |

The shared boot hook built real amd64/arm64 release artifacts, registered and
verified them, and activated valid support. API-created groups/plans, capacities,
real shared-subject admission, final-slot race, existing-device cross-owner
rejection, revoke-slot reuse and real-binding usage receipt/replay ran live.
Synthetic usage bytes test receipt validation, not measured gateway usage.

Billing now proves one paid interval/invoice after ten identical deliveries,
all six created/paid/failed permutations on independent checkouts, invalid
signature denial, forged return non-grant, and the actual rendered Portal
return page. Cancellation and partial refund preserve paid time; full refund
revokes effective service while retaining invoice/refund history. Real paid
expiry and successful payment after manual suspension also pass. The JSON
records exact periods, revisions, reconciliation state and observed expiry
latency; none is a measurement of TCP/UDP stop latency.

D-056/D-061 delivery **works with the controlled baseline**: typed node evidence
reports `applied`, the node mTLS fetch is 200 and canonical bytes match its digest.
The scheduled feed experiment rejects a comment-only update with `zero_rules`,
retains one content row and the original hash, and the node still returns
NXDOMAIN with TTL 5 for a name that resolved before activation. Query history
and statistics are disabled; service logs, metrics, audit and in-memory backup
sentinel checks pass. The guarded audit purge removes only the expired fixture.

The remaining serving gate is the inherited lease-refresh problem. During
normal operation, the original lease stayed at generation 1 with zero members
while desired/applied generations and native bindings advanced. Publication
recorded agent-policy and route readbacks independently of its incomplete
`data_path_probe`; no counted flow or denial/termination timing is claimed.

AT08 additionally found `entrypoint-node.sh` returns to `waiting_token` on
restart despite an existing durable identity. Agent, NetBird and watchdog
supervision do not resume. The documented agent-only `Stack.Service(start)`
hook allowed independent resolver checks to continue; after that restart a
later lease did contain members. This is neither uninterrupted lease renewal
nor a reboot/forwarding pass. See [requests.md](requests.md), F2-01 and F2-05.

## Attempts and corrected harness assumptions

There were two fresh bring-ups, both successful; no startup retry loop. The
first full attempt took 428.049 seconds and exposed an empty-profile mismatch:
the applier's fallback allowed probe is `example.com`, while the controlled
CoreDNS answers NXDOMAIN outside its fixture zone. The final baseline instead
uses an API allow rule for existing `feed-01.example.invalid`. Revisions reuse
one active DNS definition rather than create ambiguous active definitions.

The second full run completed all cases. AT08's initial PATCH encountered a
heartbeat-driven 412; bounded retries now refresh ETag and use a fresh command
key. The supplemental publication cycle exercised A/B regroup, disable,
restart and restoration. Its missing nft table exposed the reboot blocker.
A route assertion also incorrectly required NetBird masquerading; production
requires `masquerade=false` because node enforcement owns NAT. The corrected
readback verifies that setting plus `skip_auto_apply=true` and an explicit peer.
These earlier failures remain visible, not silently relabeled as passes.

The final observation-only run rechecks current state without another
publication mutation or 300-second feed experiment. Its successful exit is a
partial result. Whole AT results remain blocked. Host-local resolver queries
are never represented as native member DNS or member forwarding.

## Actual command results

Fresh final `e2e-up`: exit 0.

```text
 Container fvpn-w4-members-control Waiting
 Container fvpn-w4-members-member Started
 Container fvpn-w4-members-control Healthy
 Container fvpn-w4-members-node Starting
 Container fvpn-w4-members-node Started
 Container fvpn-w4-members-adguard Starting
 Container fvpn-w4-members-adguard Started
e2e stack is up
```

`make -f mk/at-members.mk at-members`: make exit 2, Go exit 1.

```text
=== RUN   TestATMembers_Live/AT20
    live_test.go:148: AT20: blocked; see docs/e2e/at/members/AT20.json
=== RUN   TestATMembers_Live/AT21
    live_test.go:148: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:148: AT22: blocked; see docs/e2e/at/members/AT22.json
--- FAIL: TestATMembers_Live (442.16s)
    --- FAIL: TestATMembers_Live/AT05 (14.51s)
    --- FAIL: TestATMembers_Live/AT01 (132.89s)
    --- FAIL: TestATMembers_Live/AT02 (52.23s)
    --- FAIL: TestATMembers_Live/AT03 (69.17s)
    --- FAIL: TestATMembers_Live/AT04 (64.25s)
    --- FAIL: TestATMembers_Live/AT08 (94.30s)
    --- FAIL: TestATMembers_Live/AT09 (0.05s)
    --- FAIL: TestATMembers_Live/AT20 (7.98s)
    --- FAIL: TestATMembers_Live/AT21 (2.22s)
    --- FAIL: TestATMembers_Live/AT22 (0.49s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	442.164s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	[no test files]
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

The scheduled supplemental run finished its feed checks but exited 2 because
of the now-preserved AT08 assertions:

```text
=== RUN   TestATMembers_FollowupReadback
    followup_test.go:90: AT08 readback contains a failed subcheck
--- FAIL: TestATMembers_FollowupReadback (390.62s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	390.627s
FAIL
make: *** [mk/at-members.mk:28: at-members-readback] Error 1
```

`FVPN_AT_MEMBERS_OBSERVE_ONLY=1 make -f mk/at-members.mk at-members-readback`:
exit 0.

```text
=== RUN   TestATMembers_FollowupReadback
--- PASS: TestATMembers_FollowupReadback (5.90s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	5.904s
```

Area lint (including pinned staticcheck), offline area test and amd64/arm64
probe builds exit 0. Root `make go-lint` and `go vet -tags e2e ./tests/...`
exit 0 with no stdout. No dependency, application, contract or shared lab file
was edited. Native/operator scripts remain [needs operator run](OPERATORS.md).

`e2e-down` exits 0; private generated state was removed through an own-prefix,
network-disabled cleanup container. Shared neutral image cache tags were retained.

```text
 Network fvpn-w4-members Removing
 Network fvpn-w4-members-origin Removing
 Volume fvpn-w4-members-pgbackrest-repo Removed
 Volume fvpn-w4-members-nb Removed
 Volume fvpn-w4-members-pg Removed
 Network fvpn-w4-members Removed
 Network fvpn-w4-members-origin Removed
e2e stack is down
```

Final prefix inventory:

```json
{"containers": 0, "images": 0, "networks": 0, "volumes": 0}
```

No push or merge was performed. All changes are in owned paths.

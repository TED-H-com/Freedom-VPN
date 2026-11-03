# Follow-up 42 outcome — 2026-11-03

## Follow-up 42

### Summary

| Finding or sub-trace | Verdict | Reachable | Introduced by (37, 38, 39, 40, 41 or pre-existing) | Fix |
| --- | --- | --- | --- | --- |
| H-1a Step entry, device/member sibling | Confirmed | A Step starts at the earlier binding deadline | 41 adds this entry call; underlying whole-node withdrawal is pre-existing | Re-decide; keep survivors until ordinary Apply on this tick |
| H-1b verifyCurrentServing, device/member sibling | Confirmed; review attribution corrected | Deadline before/during a live read, before evaluate/enforce | Pre-existing on main | Only withdraw when the fresh decision cannot serve; otherwise continue the Step |
| H-1b retained live read, device/member sibling | Confirmed | Deadline crossed in locked retained verification | 41 | Re-decide after read; if model changed, fall through to ordinary Apply |
| H-1 watchdog, device/member sibling | Confirmed additional consumer | Between earlier expiry and next heartbeat, including unlocked control I/O | Pre-existing | Heartbeat carries last admitted binding deadline, capped by all node limits; kernel expires each earlier element |
| H-1 false dash/empty reason | Confirmed consequence | Withdrawal followed by a serving decision but early return before Apply | Pre-existing verify path; 41 entry/retain variants | Remove erroneous withdrawal/return; use current blocking reasons when cached decision has none |
| M-1 post-Apply switch | Confirmed | Ruleset nudge rotates epoch after intent write | 41 retention predicate | Accept proven same counting interval; durably advance intent before retaining |
| M-1 sample discontinuity | Confirmed | Sampler rotates epoch with unchanged generation/bindings | 41 retention predicate | Same interval proof and durable-then-memory intent update |
| F32-01 renewal jitter | Confirmed | Every ordinary renewal check | Pre-existing | Uniform millisecond draw, cached and durably saved per lease |
| DeadlineMono comment | Confirmed stale description | Reader interpretation after per-device fix | 41 | Approved comment-only correction |

H-1 and M-1 are fixed on the existing branch, preserving 37–41 and every
prior merge. The unchanged r25 proofs were imported alone as bf68757f; all
six original failures and three held passes were recorded in 8abb8fa6, and
D-141 judgments were committed in 7744edd6 before fixes. H-1's underlying
whole-node expiry and verify-path early return predate 37–41 on main; 41
introduced the entry/retained variants. M-1's strict epoch equality is new in 41.

The single requested main merge is a4b76e09, taking the clean newer main
77626af4 (#427 handover 37, in addition to #424–#426). Final production is
`b42ed80b0dfaa184f20303f13a90a226925228e2`. All required gates pass. The full functional lab
passes 13 scenarios on its blob-verified archive with prefix
`fvpn-w2-agent-core-f42`. No reservation or D-133 timing measurement was run.

### Requirement and AT coverage

ISO01/ISO02 and §16.3: device and member sibling expiry at entry, initial
verification and retained verification Applies survivors with no intermediate
baseline. Last binding, node lease, desired state, node offline window and
member service limits still stop admission. The watchdog stays open for a
healthy survivor during unlocked control work, but closes on stale heartbeat
or last expiry. r24 H-3 and 41's deadline proofs remain green.

MTR01–MTR04 (§10.3–§10.7), AT10/AT13/AT16 mechanisms: expired-only removal
checkpoints outgoing bytes without closing the interval, preserves exact
counts over restart, and fails closed on settlement failure. Both r25 epoch
rotations retain without Apply; final sample, closed head and boot boundaries
reject retention. Failed durable intent refresh withdraws truthfully. r23's
restart/accounting proofs and all earlier review tests pass unchanged.

F32-01: uniform 1 ms resolution, default lead 108–132 seconds. Both endpoints
and midpoint, all 24,001 default points, rejection sampling, same-lease reuse,
next lease, failed/short read, lease replay, restart, configuration clamp and
lease/jitter transaction rollback pass. amd64 and arm64 cross-builds pass;
real arm64/field-network acceptance is not claimed.

### Files

- node/agent/{deadlines,serving,engine,decide,boot_heartbeat,restart,observe}.go:
  separate earliest wake from whole-node deadline, same-tick surviving Apply,
  expired-only accounting settlement and truthful reporting/diagnostics.
- node/agent/retain.go and engine.go: counting-interval retention and durable
  then in-memory intent refresh, preserving all prior gates.
- node/agent/sync.go and node/internal/agentstore/state.go: uniform jitter,
  lease-scoped caching and atomic lease/jitter persistence using existing meta.
- node/agent/followup42_*_test.go, node/internal/agentstore/followup42_test.go,
  unchanged imported node/agent/review_r25_test.go: deterministic proofs.
- node/watchdog/heartbeat.go: deadline contract comment; watchdog rules unchanged.
- node/enforce/model.go: expressly approved DeadlineMono comment only.
- docs/node/followup42-{r25-judgments,consumers,outcome,validation}-2026-11-03
  and the before-fail text: dated additive records. No earlier record rewritten.

No dependency, updater, installer or usage-lab fixture changes by this follow-up.
All non-owned changes from the merge are upstream main, not lane edits.

### How to run

Require `free` from `bash "$FVPN_HOLD_SH"` before every go test, make and Docker
command. Recursive make/test/Docker operations also use admission wrappers.
Exit 125 is admission refusal, never a test result. No reservation. Use
CGO_ENABLED=0, GOTOOLCHAIN=local, TMPDIR/GOTMPDIR=.agent/tmp and GOCACHE from
`go env GOCACHE` (public spelling `~/.cache/go-build`). Python uses -B.
Foreground commands only; no shell cleanup traps were introduced or used.

```sh
make -f mk/node.mk node-lint node-test node-test-integration node-build
go test -count=1 ./node/updater/... ./packaging/installer/... ./node/enforce/...
go test -count=1 -v ./node/agent ./node/enforce -run '^TestReviewR1[6-9]_'
go test -count=1 -v ./node/agent ./node/agent/cmd/fvpn-agent ./node/watchdog ./node/enforce -run '^Test(R2[0-5]_|ReviewR2[0-5]_|ReviewR11_ClearFailureNextStep)'
go test -count=1 ./tests/handover/...
make go-lint 'go-with-disk-tmp=set -eu;'
go vet ./...
make -f mk/node.mk node-test-netlab FVPN_LAB_PREFIX=fvpn-w2-agent-core-f42
go test ./...
```

Root lint's override removes only its stock cleanup trap; the same pinned
staticcheck/vet/gofmt checks run with disk scratch. The lab archive contains
exactly the final production tree, with optional timing export disabled.

### Self-check results

| Gate | Result | Receipt |
| --- | --- | --- |
| final-node-lint-build | PASS (0) | `.agent/f42/final-node-lint-build.log` |
| final-node-test | PASS (0) | `.agent/f42/final-node-test.log` |
| final-node-integration | PASS (0) | `.agent/f42/final-node-integration.log` |
| final-consumers | PASS (0) | `.agent/f42/final-consumers.log` |
| final-reviews-r16-r19 | PASS (0) | `.agent/f42/final-reviews-r16-r19.log` |
| final-reviews-r20-r25 | PASS (0) | `.agent/f42/final-reviews-r20-r25.log` |
| final-handover | PASS (0) | `.agent/f42/final-handover.log` |
| final-go-lint | PASS (0) | `.agent/f42/final-go-lint.log` |
| final-vet | PASS (0) | `.agent/f42/final-vet.log` |
| final-evidence | PASS (0) | `.agent/f42/final-evidence.log` |
| final-fixture-hygiene | PASS (0) | `.agent/f42/final-fixture-hygiene.log` |
| production-netlab | PASS (0) | `.agent/f42/production/suite.log` |
| final-all-tests | PASS (0) | `.agent/f42/final-all-tests.log` |

Formatting, git diff --check, owned-path/public hygiene, ancestry and prior
record preservation pass. The configured integration database was used; only
test-owned databases are created/dropped by dbtest. No shared database or
other lane's Docker object was changed. Owned container/image/network/volume
inventory is empty after cleanup. r25 is byte-identical to 1afecb29, and every
prior review file is byte-identical to the delivered 41 tip.

Real output tails (ordinary test-reported durations, not D-133 measurements):

```text
final-node-test:
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.298s
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.117s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/agentstore	0.199s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/controlclient	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol	0.032s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	1.073s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/meter	0.673s

final-node-integration:
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	134.918s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.286s

final-consumers:
ok  	github.com/TED-H-com/Freedom-VPN/node/updater	19.055s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater	0.785s
ok  	github.com/TED-H-com/Freedom-VPN/packaging/installer	7.434s
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	1.281s

final-reviews-r16-r19:
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	19.457s
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.220s

final-reviews-r20-r25:
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	42.908s
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.079s
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.011s
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.012s

final-handover:
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.524s

final-all-tests:
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	1.121s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usageupload	1.105s
ok  	github.com/TED-H-com/Freedom-VPN/node/nodelock	0.033s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater	26.364s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater	0.824s
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.131s
ok  	github.com/TED-H-com/Freedom-VPN/packaging/installer	8.618s
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.605s

production-netlab:
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/agent	217.727s
```

Lab archive SHA-256: `e5e44ecd1e8954924230e502ad9ed05a78bf64f54451abf69c4b76e6028706a9`;
18919 verified blobs. Binary hashes, image pin, exact scenario names and
receipt hashes are in followup42-validation-2026-11-03.json. The separate
opt-in D116 installer-marker test skips unless enabled; all required scenarios
pass. No qualified timing/performance conclusion is drawn from this suite.

Retained development failures: the initial new accounting fixture let the
uploader ACK events before an unacked-only ledger assertion; holding its test
uploads unavailable fixes the fixture without weakening assertions. An initial
jitter compile error and one S1008 lint simplification were corrected. Original
r25 before-fails and these receipts remain recorded, not represented as passes.

### Decisions made

The complete consumer inventory is docs/node/followup42-consumers-2026-11-03.md.
Earliest PolicyExpires remains the agent wake/retention boundary; the watchdog
receives the last installed binding deadline after every member/node cap.
Per-device kernel timeouts remain authoritative while control I/O is unlocked.
A deadline crossed during retained verification is re-decided before skip.

Expired-only same-generation removals checkpoint the outgoing identities
before ordinary Apply; all other identity changes retain baseline/final-sample
settlement. This does not implement intersection serving or promise hitless
Apply. Heartbeats and reports use actual serving state and current blocking
reasons. The zero decision before first evaluation keeps its legitimate dash;
Boot's dedicated heartbeat already names boot_baseline/boot_unverified.

Epoch rotations keep memory and disk together on the old continuity-valid
intent until retention refreshes both, durable first. Restart can interpret
either side of that write. Failed persistence withdraws; real interval
boundaries never qualify. Equal-epoch ticks avoid a historical query/write.

Jitter is uniformly sampled in milliseconds once per lease, atomically saved
with it, and restored after restart. Random failure/legacy rows use zero offset.
Its upper bound is 1.1 times renewLead's capped base: at the half-window cap,
at most 0.55 of the window (the pre-existing one-second base floor remains).
The grant-less D-067 renewal schedule is unchanged.

### Spec questions

None newly unresolved. D-104's pre-existing conservative installation reserve
and early-expiry precision margin remain unchanged. Kernel admission never
extends past a binding or its member/node limits; exact zero-early-expiry or
hitless transition guarantees are not added.

### Known gaps and later work

D-133 A,B,A,B for 37–42 remains owed and unrun. Physical arm64 and field
network evidence remain separate. Failed kernel commands still require the
existing fail-closed retries and kernel/watchdog backstops; command failure
is not proof of completed withdrawal. No performance claim is made here.

### Requests to orchestrator

- **2026-11-03 — Timing:** schedule the owed D-133 A,B,A,B for 37–42 after
  reviewing this follow-up, in a quiet slot. No reservation was taken here.
- **2026-11-03 — Scope:** none pending. The approved enforcement comment is
  corrected; no other owner-path edit is requested.

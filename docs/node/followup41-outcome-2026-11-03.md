# Follow-up 41 outcome — 2026-11-03

See [judgments](followup41-r24-judgments-2026-11-03.md),
[Apply inventory](followup41-apply-inventory-2026-11-03.md), and
[validation receipts](followup41-validation-2026-11-03.json).

### Summary

| Finding or sub-trace | Verdict | Reachable | Introduced by (37, 38, 39, 40 or pre-existing) | Fix |
| --- | --- | --- | --- | --- |
| H-1 repeated full Apply | Confirmed | Healthy unchanged 304 ticks | Pre-existing; timeout-only retain in 60270cd6 (28) | Full model/pending comparison under node lock, fresh serving and member-path verification |
| H-3 shared device deadline | Confirmed, mechanism corrected | Devices of one member with unequal ValidUntil, either order | Pre-existing; aggregation in ba1e3750 (25) | Member-only lease limits; renderer retains each binding's own ValidUntil cap |
| H-1 missing-element verification dependency | Confirmed | Live deletion of an authorized allow element | Pre-existing verifier omission, previously masked by periodic Apply | SystemPlane completeness check on the existing live nft snapshot |
| H-3 retained applied model | Expected audit state; C1 warning | Withdrawal and same-boot adoption | Pre-existing | Preserve it; require serving/current boot/verified evidence and no pending work to skip |
| H-3 Apply clock value copy | Expected local resampling; C1 warning | Every real Apply | Pre-existing | Compare clock mappings and absolute deadlines; no C1 eligibility added |
| H-2 fair-share recalculation | Expected behavior, not an agent defect | Independent member removal | Pre-existing | Original passing proof unchanged; C1 owner accounts for QoS changes |
| Original H-1 fixture's claimed 304 | Incorrect fixture premise | First post-setup poll changes effective expiry by 1 ms | Pre-existing review fixture, not 37–40 | Three-line setup correction prepared and proven via overlay; approval pending |
| Usage lab counting setup | One missed burst retained; exact loss cause undetermined | Instantaneous burst between serving observations | Pre-existing fixture assumption | Require actually counted traffic within bounded setup; keep ledger/ACK assertions |

Both runtime findings are fixed on the existing branch, preserving 37–40 and
all prior merges. The r24 proofs were imported byte-for-byte alone in
660ab392; their original before-fails were recorded, then D-141 judgments and
the full Apply-effect inventory were committed in 1ebc1fe3 before runtime
fixes. Main advanced to c98ea61c (#423) and merged cleanly once as 6b6a26b0.
The frozen production source is f1ed8f3bba25ed233ad81e290ad6c9068a506b89.
All 13 final functional lab scenarios pass; the first failed run is retained.

The required green-gate claim remains blocked by the original H-1 fixture,
which the prompt only authorizes adapting for API differences. Its fake server
returns fresh HTTP 200 envelopes and moves the first effective expiry by 1 ms;
correct full-model comparison Applies once. Ignoring that change would violate
the requested rule. Independent actual-304 checks pass, and the original H-1
assertions pass with the prepared setup correction in a test-only overlay.
The tracked review files remain identical to ba5d3004 pending the answer.
No D-133 performance measurement or reservation was taken. The mandatory
functional suite retains its existing deadline/correlation diagnostics.
After the source freeze, main gained only ROADMAP updates 4144e0cc and
d7867928. The one requested clean merge retains c98ea61c; production content
is unchanged by those later documentation updates.

### Requirement and AT coverage

ISO01/ISO02 (§8, §16.3), MTR01–MTR04 (§10.2–§10.9), §11 QoS, §13.9 probe
bounds and §14.7 lease/304 behavior; AT10, AT13 and AT16 mechanisms.
Tests cover full renderer-model changes at unchanged generation, genuine 304,
locked fresh verification and truthful heartbeat, ValidFrom/ValidUntil/member
and sibling boundaries, lease renewal, both device orders and several member
lease caps, deleted allow element/chain, foreign tc, and metering-marker drift.

Skipped ticks retain the counting identity and durable intent while samples
account for 11,264 bytes without gaps. Existing restart, shutdown, pending
handoff, persistence, quarantine and conntrack review proofs remain active.
Cross-builds cover linux/amd64 and linux/arm64; physical arm64 and field
network acceptance are not claimed. C1 and hitless expiry transitions remain
unimplemented; a planned expiry conservatively withdraws, settles usage, and
redecides surviving devices on the same Step.

### Files

- node/agent/{engine.go,retain.go}: full-model retention eligibility, pending
  gates, locked live verification, and same-tick expiry redecision.
- node/agent/decide.go: member-only deadlines and separate watchdog device
  boundaries, capped before monotonic conversion.
- node/agent/{allowed_presence.go,plane.go}: read-only expected-element
  completeness using the existing nft snapshot and verification timing record.
- node/agent/followup41_*_test.go, allowed_presence_test.go, and the imported
  node/agent and node/watchdog review_r24_test.go files: direct proofs.
- Earlier tests now trigger their Apply-only faults with a valid local QoS
  change. D-104 tests assert device timeouts separately from member limits;
  conntrack tests inspect a new Apply rather than an old saved request. No
  failure/safety assertion is weakened. The exact producer replay starts its
  replacement emulated kernel as unverified. Dated judgments precede changes.
- docs/node/followup41-{r24-judgments,apply-inventory,outcome}-2026-11-03.md
  and followup41-validation-2026-11-03.json: decisions, effects and receipts.
- tests/netlab/agent/scenario_at_usage_test.go: bounded counted-traffic setup;
  downstream ledger and ACK assertions remain unchanged.
- No dependency, updater, installer or enforcement-owner edit by this lane.
  Netlab 32's production and faithful nft/tc simulator changes arrived via main.
  Earlier records remain unchanged; new clarification is additive.

### How to run

Before each Go test, make and Docker command, run
`bash "$FVPN_HOLD_SH"` and require `free`. Exit 125 means not admitted, never
a test result. No reservation. Use CGO_ENABLED=0, GOTOOLCHAIN=local,
TMPDIR/GOTMPDIR=.agent/tmp and GOCACHE from `go env GOCACHE` (public spelling
`~/.cache/go-build`). Python uses -B. Foreground commands; no shell traps.

```
make -f mk/node.mk node-lint node-test node-test-integration node-build
go test -count=1 ./node/updater/... ./packaging/installer/...
go test -count=1 -v ./node/agent ./node/enforce -run '^TestReviewR1[6-9]_'
go test -count=1 -v ./node/agent ./node/agent/cmd/fvpn-agent ./node/watchdog ./node/enforce -run '^Test(R2[0-4]_|ReviewR2[0-4]_|ReviewR11_ClearFailureNextStep)'
go test -count=1 ./tests/handover/...
make go-lint 'go-with-disk-tmp=set -eu;'
go vet ./...
go test ./...
make -f mk/node.mk node-test-netlab FVPN_LAB_PREFIX=fvpn-w2-agent-core-f41
```

Root lint's override removes only its stock shell cleanup trap. All pinned
checks run on disk scratch. Recursive MAKE, Go tests and Docker use admission
wrappers. The lab uses a blob-verified archive of the production commit, with
the optional whole-suite timing collector unset.

### Self-check results

Original r24 before-fix: H-1 and H-3 fail; H-2, retained audit model,
monotonic/same-boot and watchdog held proofs pass. The independent actual-304
proof fails on b17c8c17 (Apply 2 -> 3 on its first tick) and passes after 41.
Deleted-element drift initially remained serving; the new completeness check
makes it withdraw. The distant-date development proof failed before its cap
correction. Failed runs and fixture panics remain in ignored logs and are not
reported as passing evidence.

| Gate | Result | Receipt |
| --- | --- | --- |
| final-node-lint-build | PASS (0) | `.agent/f41/final-node-lint-build.log` |
| final-node-test | FAIL: original H-1 fixture (2) | `.agent/f41/final-node-test.log` |
| final-node-integration | FAIL: original H-1 fixture (2) | `.agent/f41/final-node-integration.log` |
| final-consumers | PASS (0) | `.agent/f41/final-consumers.log` |
| final-reviews-r16-r19 | PASS (0) | `.agent/f41/final-reviews-r16-r19.log` |
| final-reviews-r20-r24 | FAIL: original H-1 fixture (1) | `.agent/f41/final-reviews-r20-r24.log` |
| final-handover | PASS (0) | `.agent/f41/final-handover.log` |
| final-go-lint | PASS (0) | `.agent/f41/final-go-lint.log` |
| final-vet | PASS (0) | `.agent/f41/final-vet.log` |
| final-evidence | PASS (0) | `.agent/f41/final-evidence.log` |
| final-fixture-hygiene | PASS (0) | `.agent/f41/final-fixture-hygiene.log` |
| final-all-tests | FAIL: original H-1 fixture (1) | `.agent/f41/final-all-tests.log` |
| production-netlab-attempt1 | FAIL: usage setup burst (2) | `.agent/f41/production-attempt1/suite.log` |
| production-netlab | PASS (0) | `.agent/f41/production/suite.log` |
| final-lab-setup-lint | PASS (0) | `.agent/f41/final-lab-setup-lint.log` |
| delivery-all-tests | FAIL: original H-1 fixture (1) | `.agent/f41/delivery-all-tests.log` |

The nonzero non-lab gates contain exactly one failing test each: the
untouched original H-1 proof. The main merge introduces no additional failure.
Original H-3, H-2, held watchdog and all other earlier review tests pass.
The proposed three-line H-1 setup correction passes through a Go test overlay
with original assertions unchanged; this is supplemental evidence, not a
substituted green canonical gate.

The first full lab passes 12 of 13: UsageLedger's single instantaneous UDP
burst did not move its admission counters. The exact cause is not established
by the retained snapshot, which shows serving rules afterward. The fixture
now requires actual counted traffic within a bounded retry window before its
unchanged byte-equality, no-recount, final-sample, ACK and watermark assertions.
This accounts for permitted guarded installs; it makes no hitless claim.
No runtime file changed. The subsequent full suite passes all 13 scenarios.

Real output tails:

```text
final-node-test:
FAIL	github.com/TED-H-com/Freedom-VPN/node/agent	130.799s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.294s
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.110s
final-node-integration:
FAIL	github.com/TED-H-com/Freedom-VPN/node/agent	132.712s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.291s
final-consumers:
ok  	github.com/TED-H-com/Freedom-VPN/node/updater	19.219s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater	0.809s
ok  	github.com/TED-H-com/Freedom-VPN/packaging/installer	7.328s
final-reviews-r16-r19:
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	19.616s
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.220s
final-reviews-r20-r24:
FAIL	github.com/TED-H-com/Freedom-VPN/node/agent	40.973s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.079s
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.019s
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.012s
final-handover:
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.211s
final-all-tests:
FAIL	github.com/TED-H-com/Freedom-VPN/node/agent	139.697s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.285s
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	1.401s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater	26.183s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater	0.800s
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.118s
ok  	github.com/TED-H-com/Freedom-VPN/packaging/installer	8.570s
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.492s
production-netlab:
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/agent	222.100s
```

Only the separate opt-in D116 installer-marker scenario skips. Source archive
SHA-256 is `76237d6b3fa79d60f72b95e795e48ff26f2d8f5029c2e438ea790cfd1f51d7b3`, with
18718 verified blobs. The optional timing collector is disabled. Binary/image
pins and exact scenario names are in the validation JSON. No D-133 arm ran;
the mandatory suite retains its ordinary deadline/correlation diagnostics.

Formatting, pinned staticcheck, vet, whitespace, owned-path/public hygiene and
prior-record preservation checks pass. Original r24 files remain byte-identical
to ba5d3004. Owned Docker containers/images/networks/volumes and owned lab
processes are empty. Final production binaries are identical to the first
lab's binaries; only the counting setup and dated record differ. Only docs/
evidence change after the final frozen production source; final delivery hash
is recorded below and in .agent/last.md.



The final plain whole-module check also ran after staging the outcome/evidence
documents, on production f1ed8f3b. It again has only the original H-1 fixture
failure; all other packages pass. Exact tail:

```text
--- FAIL: TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick (0.16s)
FAIL	github.com/TED-H-com/Freedom-VPN/node/agent	173.756s
```

### Decisions made

- Normalize only elapsed clock samples with an unchanged wall/monotonic
  mapping; compare every model field via DeepEqual. Compare actual assigned
  QoS classes and treat a successful Revoke as a consumed action. Separate
  PolicyExpires equality and live metering-marker verification are mandatory.
- Pending revocations/reservations, repairs/quarantine/probe cleanup, usage
  persistence/settlement, closed epochs, final barriers, sampler state, boot
  identity, intent/active epoch and kernel nudges prevent retention. Reading
  usage Head is new read-only work; 40's historical no-extra-SQL sentence does
  not describe 41. Healthy retained ticks do not rewrite usage intents.
- Verify nft/tc under the node lock and check member topology before the
  retained heartbeat. Initial/final Step reads still run. A failed read
  follows existing repair or withdrawal rules. Foreign tc stays quarantined
  until its owner removes it; no foreign object is deleted.
- The allow-element presence supplement adds no command/write and preserves
  existing nft/tc failure precedence. It accepts only the existing 101 ms
  early-expiry margin. No Apply/guard measurement record is fabricated on a
  skipped tick; live verification and heartbeat records remain.
- H-3 uses the existing renderer's per-binding cap. The review's statement
  that memberTimeout ignores ValidUntil is wrong. Watchdog expiry still wakes
  at the earliest active boundary, but no device limit shortens the member's
  lease fields or another device's installed lifetime.
- A development arithmetic review caught 41's initial distant-date overflow.
  Its 300-year validity proof failed on 0d920a28; judgment/test preceded the
  correction. Binding duration is now capped before adding the monotonic
  anchor, and that proof passes. This was an implementation correction in 41,
  not a defect introduced by 37–40.

### Spec questions

The requested wording that an element never expires before its intended
deadline must retain D-104's existing conservative installation reserve.
The current renderer expires a member element in (D−101 ms, D], not exactly
at signed time D. This follow-up removes the multi-minute sibling collapse
and refresh dependency without extending authorization or altering that
pre-existing enforcement contract. Probe lifetimes have their separate
whole-second/cap/hold rules. No exact zero-early-expiry guarantee is claimed.

### Known gaps and later work

The original H-1 fixture correction is pending; required unit/integration,
review and whole-module green status cannot be claimed until it is authorized
and rerun. The proposal changes only setup, with assertions unchanged.
D-133 A,B,A,B for 37–41 is owed and unrun. Physical arm64/field networks and
hitless C1 transitions remain unverified/unimplemented. Persistent kernel
command failures retain existing conservative reporting, retries and
kernel-expiry/watchdog backstops; failed commands are not proof of withdrawal.


The prepared setup addition immediately after H-1's manualUsage call is:

```go
// Keep the fake signer at this snapshot so later polls are actual 304s.
signedAt := r.eng.desired.IssuedAt.Add(time.Minute)
r.srv.SetClock(func() time.Time { return signedAt })
```

### Requests to orchestrator

- **2026-11-03 — H-1 proof fixture:** authorize the three-line setup correction
  in `.agent/f41/r24-h1-fixture.patch`. The prompt says “Adapt only what your
  branch's API requires”; this is a fixture correction, so it was asked
  separately and is still pending. The proposed original assertion already
  passes via `.agent/f41/proposed-r24-h1-setup.log`; the tracked original stays
  untouched. No production comparator workaround is requested or appropriate.
- **2026-11-03 — Enforcement comment:** ask the node/enforce owner to correct
  MemberLease.DeadlineMono's comment, which still describes folding binding
  limits into the shared member deadline. Runtime already caps each binding
  independently; no enforcement runtime change is needed for H-3.
- **2026-11-03 — Timing:** schedule the later quiet D-133 A,B,A,B for 37–41
  with fresh main/B freezes. Prior unrun drafts/evidence are historical; do
  not reuse an old B pin. No reservation or measurement was taken here.

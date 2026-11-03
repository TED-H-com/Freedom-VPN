# Follow-up 43 outcome — 2026-11-03

## Follow-up 43

### Summary

| leg | window | qualified (with reason if not) | scenarios passed | outer guard p99 and maximum | whole serving Step p99 and maximum | failed operations |
| --- | --- | --- | --- | --- | --- | --- |
| A1 | 1 | No: foreign CPU 0.691858 > 0.5 cores | 1/13 | 37.742 / 37.742 ms | 205.757 / 205.757 ms | 13 |
| A1 | 2 | No: foreign CPU 1.311445 > 0.5 cores | 1/13 | 81.643 / 81.643 ms | 1636.113 / 1636.113 ms | 13 |
| B1 | — | Not run: preceding A1 stopped in both windows | — | — | — | — |
| A2 | — | Not run: preceding A1 stopped in both windows | — | — | — | — |
| B2 | — | Not run: preceding A1 stopped in both windows | — | — | — | — |

**Neither arm has a qualified leg. D-133 remains owed; no A/B comparison or
attributable shift is supported.** Both authorized windows are exhausted.
The separately run uninstrumented B production lab passes all thirteen
required scenarios. All ordinary gates and the final plain `go test ./...`
pass. No production code changed, and node/enforce is untouched.

The branch preserves every 37–42 commit and prior main merge. r26's proofs
were imported alone in `fe256ba9`, byte-identical to review `5c6d6abe`.
The sole main merge is `38420dad`, taking `567080be1969bf22782617e78c7605c2c92a52a2`.
Frozen B is `38420dadef6ae9d0236288eef533bc1ddee0da53`, with production code
unchanged from `b42ed80b0dfaa184f20303f13a90a226925228e2`. The declaration and
runner were committed in `45a18791` before either reservation. Window 1
receipts were committed in `daa6dc44`, window 2 in `37fdda12`, and combined
results in `5cc09662`. No later merge, push, PR or tag was made.

Window 1 reserved at 17:44:03 UTC after 0.035 seconds, with no refusal.
After the sixty-second drain and qualifying preflight, A1 ran 36.144 seconds.
Foreign CPU reached 0.691858 cores (278 ticks over 4.018165 seconds; a foreign
python3 supplied 250 ticks). Release was 17:45:45 UTC. Window 2 reserved at
17:55:45 UTC after 0.037 seconds, with no refusal, 600.037 seconds after the
first release. After its drain/preflight, A1 ran 92.415 seconds. Foreign CPU
reached 1.311445 cores (527 ticks over 4.018469 seconds; the two largest
contributors supplied 387 and 103 ticks). Release was 17:58:23 UTC. No
foreign suite/container, unreadable state or interval over five seconds
caused either stop. No process was paused or newly excluded.

### Requirement and AT coverage

D-133/D-138: immutable archives, every-blob verification, declared A1/B1/A2/B2,
two bounded reservations, unchanged CPU/process/container qualification,
complete stopped-leg retention and five requested span distributions are
recorded. Both qualified pools have n=0; no regression or no-regression
judgment follows. Expiry/renewal traces were incomplete in both timing legs,
so their packet predicates are unavailable, not passing.

r26's six held proofs pass on the delivered 42 tip and after merging main:
whole-node deadline versus per-binding kernel cutoff; surviving siblings and
early-return truthfulness; exact expired-only accounting; same-interval epoch
retention and durable intent refresh; jitter bounds/persistence/clamping;
and the zero-decision/validation contract. ISO01/ISO02 (§16.3), MTR01–MTR04
(§10.3–§10.7), AT10/AT13/AT16 mechanisms and earlier R16–R26 proofs remain
covered. There is no change to a spec requirement or acceptance threshold.

The functional B lab passes D087/D104, lease gating, watchdog, kernel expiry,
verified restart, revoked/unpublished/update isolation, health attestation,
usage ledger and meter-reset scenarios. Its expiry predicate passes with
97 positives and 15 drops: last positive at D−100.437384 ms, first drop at
D−63.884857 ms, zero positive egress at/after D. Two actual renewals were
captured; 1517/1518 probes delivered. The one lost packet's ingress is inside
the install/outer-guard bracket. It remains a loss, with no D-120 tolerance,
no zero-loss claim and no timing qualification. These functional observations
cannot repair either failed timing window.

amd64 and arm64 node cross-builds pass. Physical arm64 and field networks
remain unmeasured. The optional D116 installer-marker test retains its
explicit opt-in skip; it is outside the thirteen required scenarios.

### Files

- `node/agent/review_r26_test.go`: unchanged imported proofs only.
- `tests/netlab/agent/followup43*`: new runner, timestamp-only build overlay,
  span fixture, offline analyzer and seven monitor/analysis tests.
- `docs/node/followup43-*`: dated declaration/pins, two cumulative window
  receipt snapshots, combined results (Markdown/JSON), outcome and validation.

Earlier timing records, old runner hashes, all previous review tests and
production files are preserved. No dependencies, node/enforce, node/updater,
installer, API contracts or another lane's paths were edited by this follow-up.
The sole merge's non-owned changes are upstream main.

### How to run

Every make, Go test and Docker command requires `free` from
`bash "$FVPN_HOLD_SH"`; recursive tools use admission wrappers. Exit 125
means refusal, never test failure. No such refusal occurred in these runs.
Use CGO_ENABLED=0, GOTOOLCHAIN=local, disk TMPDIR/GOTMPDIR=.agent/tmp,
GOCACHE from `go env GOCACHE` (public spelling `~/.cache/go-build`), Python
-B/no bytecode. Commands remain foreground, with no shell cleanup traps.

The timing declaration's two reservations are consumed. Do not restart this
experiment or repool its unqualified legs under a new label. The runner
ledger prevents a third reservation. A new timing follow-up needs its own
declaration/authorization and qualifying host conditions.

Ordinary commands used, each after admission and outside reservations:

```sh
make -f mk/node.mk node-lint node-test node-test-integration node-build
go test -count=1 ./node/updater/... ./packaging/installer/... ./node/enforce/...
go test -count=1 -v ./node/agent ./node/enforce -run '^TestReviewR1[6-9]_'
go test -count=1 -v ./node/agent ./node/agent/cmd/fvpn-agent ./node/watchdog ./node/enforce -run '^Test(R2[0-6]_|ReviewR2[0-6]_|ReviewR11_ClearFailureNextStep)'
go test -count=1 ./tests/handover/...
make go-lint 'go-with-disk-tmp=set -eu;'
go vet ./...
make -f mk/node.mk node-test-netlab FVPN_LAB_PREFIX=fvpn-w2-agent-core-f43
go test ./...
python3 -B -m unittest discover -s tests/netlab/agent -p 'followup43*_test.py'
```

The lint override removes only the stock shell cleanup trap; the pinned
staticcheck, gofmt and vet checks remain unchanged. The ordinary lab uses
frozen B without the overlay and with optional whole-suite timing disabled.

### Self-check results

| Gate | Result | Receipt |
| --- | --- | --- |
| r26-tip | PASS (0) | `.agent/f43/r26-tip.log` |
| final-node-lint-build | PASS (0) | `.agent/f43/final-node-lint-build.log` |
| final-node-test | PASS (0) | `.agent/f43/final-node-test.log` |
| final-node-integration | PASS (0) | `.agent/f43/final-node-integration.log` |
| final-consumers | PASS (0) | `.agent/f43/final-consumers.log` |
| final-reviews-r16-r19 | PASS (0) | `.agent/f43/final-reviews-r16-r19.log` |
| final-reviews-r20-r26 | PASS (0) | `.agent/f43/final-reviews-r20-r26.log` |
| final-handover | PASS (0) | `.agent/f43/final-handover.log` |
| final-go-lint | PASS (0) | `.agent/f43/final-go-lint.log` |
| final-vet | PASS (0) | `.agent/f43/final-vet.log` |
| final-evidence | PASS (0) | `.agent/f43/final-evidence.log` |
| final-fixture-hygiene | PASS (0) | `.agent/f43/final-fixture-hygiene.log` |
| production-netlab | PASS (0) | `.agent/f43/production/suite.log` |
| final-all-tests | PASS (0) | `.agent/f43/final-all-tests.log` |
| runner/overlay/analyzer tests | PASS (7) | `.agent/f43/runner-tests.log` |
| overlay A/B compile and D104/F29 proofs | PASS | `.agent/f43/overlay-{check,proofs}-{A,B}.log` |
| scope, ancestry, prior records, gofmt and diff hygiene | PASS | `.agent/f43/hygiene.json` |
| owned Docker inventory | Empty | `.agent/f43/inventory.json` |

Real output tails (ordinary test durations, not qualified timing):

```text
final-node-test:
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol	0.031s
?   	github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol/cmd/fvpn-labcontrol	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/node/internal/nodelock	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	1.054s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/meter	0.669s

final-node-integration:
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 -tags integration ./node/agent/...
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	137.750s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.286s

final-reviews-r16-r19:
    review_r19_intersection_test.go:66: current destination denial precedes the established download accept
--- PASS: TestReviewR19_Q4_CurrentRenderDoesNotRestrictEstablishedDownload (0.00s)
=== RUN   TestReviewR19_Q4_ExistingCounterIsNotRecreated
--- PASS: TestReviewR19_Q4_ExistingCounterIsNotRecreated (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.219s

final-reviews-r20-r26:
=== RUN   TestR24_M2_PostShapeClockCheckSkipsBaselineInApply
--- PASS: TestR24_M2_PostShapeClockCheckSkipsBaselineInApply (0.00s)
=== RUN   TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity
--- PASS: TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.012s

final-handover:
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.444s

final-all-tests:
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.703s
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	2.733s
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/at	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/cmd/cache-authority	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/internal/cachefixture	[no test files]

production-netlab:
PASS
ok  github.com/TED-H-com/Freedom-VPN/tests/netlab/agent 212.862s
```

Every imported proof and original production file matches its frozen source.
The public diff contains only owned paths, with no secret/private-location
hygiene matches. Earlier documents are byte-preserved. The test database is
managed by dbtest; shared PostgreSQL was never reconfigured. Every window
and the final lab end with zero owned containers, images, networks and
volumes. No reservation or worker remains running.

Retained development notes: an initial pre-declaration fixture-list generator
assumed follow-up 39's runner existed in A; enumeration was corrected per
arm before pinning. An initial overlay compile probe matched no tests;
actual D104/F29 proof selections subsequently passed on both arms. The
only post-declaration analyzer change redacts unrelated container names in
public receipts, while preserving local raw inventories and hashes. Its
current hash is recorded. The measurement runner/overlay stayed unchanged.

### Decisions made

Original archives are immutable and reverified before each window. The new
TEST ONLY build overlay is separately pinned and changes no production
source or enforcement logic. It measures original Step entry through return,
including deferred verification/heartbeat, and separately brackets the
pre-Apply accounting work through the immediate Apply call boundary.
Pre-Apply record serialization occurs after enforcement; whole-Step cost
includes it. These are instrumented diagnostics, not claims of zero overhead.

Completed failed-operation guard spans stay in the partial distributions.
Each interrupted leg retains thirteen failed enforcement operations (three
Apply and ten Verify, including intentionally blocked/negative fixture paths)
and no printed scenario failure. Aborted boundaries and returned errors are
retained separately. Qualified pooling never includes either stopped leg.
No D-138 attribution is possible with zero qualified legs in both arms.

### Spec questions

None new. No D087/D104 predicate, production threshold, CPU qualification,
process exclusion or window limit was weakened.

### Known gaps and later work

The D-133 A,B,A,B debt for 37–42 remains open. Neither A1 attempt qualified;
B1, A2 and B2 were never admitted. Both full packet predicates and any B Step
or accounting distribution are therefore missing from the qualified timing
comparison. The ordinary B lab is functional evidence only. Physical arm64,
real NetBird/AdGuard and field-network acceptance remain separate.

### Requests to orchestrator

- **2026-11-03 — D-133:** arrange a new declared comparison on a host/window
  that meets the unchanged foreign-CPU rule. Both authorized reservations
  here are exhausted; neither partial A leg may be reused as qualified.
  There is no attributable code regression to fix from these data.
- **2026-11-03 — Scope:** no external-owner code change is requested. The
  production lab and all other gates are delivered; node/enforce stays closed.

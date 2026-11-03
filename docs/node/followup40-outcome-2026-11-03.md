# Follow-up 40 outcome — 2026-11-03

The confirmed r23 accounting, interrupted hand-off and reporting defects are
fixed. The [judgments](followup40-r23-judgments-2026-11-03.md) distinguish each
reachable failure from fixture overclaims and record the pre-fix evidence.
All six original finding functions failed before edits; all six held functions
passed. Their unmodified import is commit `dcdc3d96`, matching `26ffc36c`.

H-1a still requires the genuine counter-reset gap: the recoverable total is
8192 bytes, not 9216. H-1b's absent idle counter is retained as a labeled guard;
the production-shaped zero-counter version already passed on 39. No healthy
renderer outage is claimed from that fixture. H-1c and the actual repair nudge
exercise additional restart provenance failures. Arbitrary old same-boot intent
is not accepted: the ledger must prove uninterrupted reset successors, without
a final sample, fresh baseline or reboot in between.

## Final mechanism and cost

Restart can use intent across a proven same-boot counting interval. A transient
marker-read failure does not erase durable attribution; a successful conflicting
marker still rejects it, and adoption retains all live verification gates.
This adds one read-only history query on restart after an epoch change.

Ledger closure is independent of updater checkpoint publication. Every failed
withdrawal retains pending settlement, and every admission after closure prepares
an open epoch. Baseline precedes fallible checkpoint invalidation. The original
second fail-closed Baseline attempt remains. The pre-Apply failure path retries
a failed Baseline within its existing local deadline.

Decision recomputation includes pending usage, persistence and serving changes.
Blocked Steps publish matching observations and heartbeats, and a successful
final sample does not erase the failure that caused that pass's withdrawal.
Healthy serving Steps add only boolean checks, with no additional sample,
source read, fsync or SQL write. No quantitative performance claim is made.

H-2's old false admission failure usually healed on the next successful hand-off
and subsequent Step (default cadence ten seconds plus work); repeated I/O
failure has no finite recovery bound. M-1's bytes remain in preserved counters,
but recovery now explicitly retains settlement rather than relying on a future
sample or binding change. When all kernel withdrawal calls fail, physical
withdrawal is unverified: reports carry Serving=false and BaselineOK=false;
retries and the existing kernel expiry/watchdog remain necessary.

## Functional production gate

Frozen source: `5d964d2a05111e86ca881f6c1a103e7617909ce5`, including main
`090bb97b88b4062073a487c82676d0e1faca4fac`. All 18,528 archive blobs were
verified. Archive SHA-256:
`17da6860c865f68f82069c597279e3ab27393475ecc3eafe3694e5bbd4643b72`.
Later delivery changes contain only this evidence and documentation.

The full `node-test-netlab` passed **all 13 required scenarios**, using
`FVPN_LAB_PREFIX=fvpn-w2-agent-core-f40`. The separate optional D116
installer-marker scenario skipped because its opt-in was unset. The optional
whole-suite timing collector was disabled. Standard functional deadline checks
still ran, without a comparative timing analysis or reservation.

Source, binary and image pins, scenario names, before-fail receipts, gate
results and cleanup inventory are in the
[validation JSON](followup40-validation-2026-11-03.json). Raw receipts remain
under `.agent/f40/`; the production log is `production/suite.log`.

Final node lint/unit/integration and amd64/arm64 builds pass. Unchanged
consumers, R16–R19, handover, root pinned lint, vet, evidence hygiene and plain
`go test ./...` pass. Build/integration runtime sources match the frozen lab
source; final module and evidence checks include the latest merged handover
changes. Owned Docker container, image, network and volume inventory is empty.

## Regression and record coverage

The converted r20–r23 proofs pass. The new matrix covers H-1a/b/c; all four
H-2 interruption modes; H-3a/b/b′; M-1; restart during each failure and after
recovery; and remove/re-add with real renderer calls that preserve named
counters. No r20–r22 assertion changed in 40. R16–R19, the unchanged
`node/agent/followup12_test.go`, updater and installer consumers remain covered.
The Boot blocking matrix and shipped stop-budget checks pass with unchanged
10-second Boot and 3-second Close budgets under TimeoutStopSec=15s. The existing
combined test allows 300 ms of scheduling tolerance; its full test duration also
includes fixture setup. This is not a new worst-case latency measurement.

Follow-up 39's judgments and outcome have dated additive corrections for every
confirmed limitation, including L-1's actual checkpoint invalidation ordering.
Original text, 37/38 corrections and every prior merge remain preserved. The
renderer is `render_nft.go`, with a forward-chain generation marker, preserved
named counters and `TestRender_Golden`; the review's alternative file names,
table comment and deletion claim are not used as evidence.

## Requests to orchestrator — 2026-11-03

Schedule the owed D-133 A,B,A,B for 37–40 in a later quiet slot with no other
lane session. The [draft declaration](followup40-timing-draft-2026-11-03.md)
lists the source, archive, runner namespace and measurement changes required
to use final 40 merged with the scheduled main as B. It is **not yet run**.
No reservation was taken in 40, and the prior attempts are preserved. There
is no additional cross-owner implementation request. Physical arm64 and field
network acceptance remain unmeasured.

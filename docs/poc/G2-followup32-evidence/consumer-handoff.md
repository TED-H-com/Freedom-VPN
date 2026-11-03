# Node consumer gate: retained failure and fixture hand-off

2026-11-03. [Full output](gate3-node.txt), [receipt](gate3-node.json).
`go test -count=1 ./node/... ./packaging/installer/...` exited 1 after 90.141s.
All listed packages except node/agent passed, including enforce, watchdog,
updater and installer. Seven top-level agent tests fail (their subtests remain
in the full output). This is not a green consumer gate or merge-ready claim.

The directly diagnosed incompatibility is in the unchanged
[`listedTable.applyLine`](../../../node/agent/followup12_test.go): its `add chain`
case unconditionally calls `braceBody`, rejecting the valid command
`add chain inet fvpn fv_deadline_guard`. The same new baseline's actual
nftables behavior is proven by the [live kernel check](lab-summary.md).
The fixture also replaces existing sets/chains on `add`, instead of implementing
nft's create-if-missing semantics. A faithful simulator must preserve existing
objects and accept empty regular chain declarations before delete/recreate.
No production nft syntax should be weakened to fit that parser.

[`renderedExecutor.TCBatch`](../../../node/agent/followup12_test.go) additionally
snapshots the entire desired model on every batch rather than interpreting
upserts versus deferred deletes. Its embedded
[`revocationExecutor`](../../../node/agent/revocation_executor_test.go) generates
tc listings from that model. It therefore cannot prove the new crash cut or old
class persistence until nft commit. That tc fixture needs separate ownership
and ordering treatment. The retained failure does not yet prove which further
cases would fail after repairing the nft parser; do not attribute all seven
failures to the old tc order or claim an unperformed follow-up run.

### Requests to orchestrator

**2026-11-03 — agent-core:** update the nft and tc simulation in
`node/agent/followup12_test.go` and, as needed, its embedded helper in
`node/agent/revocation_executor_test.go`; model idempotent add, bare regular
chains, atomic command batches and prepare/commit/retire ordering. Then rerun
the unchanged consumer gate. Although followup12_test.go is a listed ownership
exception, this task expressly requires consumers without edits and hands off
agent-core fixtures; it remains byte-identical. No agent/watchdog test was
imported or changed. Only this item stops here; all other gates continue.

### 2026-11-03 — 09:03 EST continuation: owned consumer fixture repair

The orchestrator clarified that “without edits” forbids weakening consumer
assertions, and explicitly authorized these test-only files:
`node/agent/followup12_test.go` and `node/agent/revocation_executor_test.go`.
No production file or other agent-core test was changed. The previous failure
and its hand-off remain historical evidence rather than being replaced.

The nft simulator now keeps existing tables/chains/sets on add, accepts bare
regular chains, and commits cloned mutable state only after every command in a
batch succeeds. Failed commands do not update installed-state metadata, the
admission-set expiry anchor or successful batch history. Re-adding an existing
set does not renew its elements. Actual production baseline batches pass the
simulator cold/warm/repeat behavior already proved by the retained live gate.

TC readback now reflects executed commands per interface rather than the desired
model. Preparation adds/replaces while retaining old classes and leaves;
retirement removes only the named objects; root cleanup removes its whole tree.
TC batches retain successful commands preceding an error, unlike nft's atomic
batch. Foreign-filter injection and production verification remain active.

The new tests observe real enforce.Apply calls in prepare/commit/retire/release
order. They prove that old classes still exist while old nft references remain,
that the committed candidate no longer references those classes, and that a
process-death cut before retirement leaves unreferenced classes that exact QoS
verification rejects. The next Apply or baseline reclaims them. No production
syntax, guard, verifier or consumer assertion was weakened.

The unchanged consumer command now **passes**, exit 0 in **89.056s**, on
`b79ec209b6520adcedc4d03bc02d75002ccdc2a3`. `node/agent` passed in 88.580s.
See the [rerun output](gate3-node-rerun.txt), [receipt](gate3-node-rerun.json),
[focused fixture proofs](consumer-fixture-proofs.txt) and
[test-only source hashes](consumer-source.json). The proof receipt records the
pre-commit parent plus exact fixture hashes; those hashes match the committed
source used by the consumer gate. No tests in other agent-core files failed,
and no existing consumer assertion was edited.

This completes the fixture request above under the orchestrator's scope
clarification. The first failed gate and original hand-off text remain intact.
The current [continuation gate receipts](consumer-gates.json) are separate from
the initial gate records. Gates 1–2 and the usage/counterread gate are retained;
the full lab is not rerun because production is byte-identical. C1 remains docs
only, with the owner's decision and other earlier future-work requests unchanged.

Gates **4–9 also pass** on the unchanged test source, in the required order:
[evidence hygiene](gate4-consumer.txt), [handover](gate5-consumer.txt),
[whole-module lint](gate6-consumer.json), [vet](gate7-consumer.json),
[formatting](gate8-consumer.json) and [whitespace](gate9-consumer.json).
The [final continuation audit](consumer-audit.json) checks source pins,
retained artifacts, local links and scope. The [final publication hygiene check](consumer-publication-hygiene.txt) also
passes after adding these receipts. The seven initial
consumer failures are resolved; no consumer-specific request remains open.

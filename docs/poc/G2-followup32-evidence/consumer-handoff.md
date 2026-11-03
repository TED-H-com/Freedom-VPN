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

# Follow-up 24 handover — 2026-10-31

Read the [isolation hand-off](followup24-isolation-2026-10-31.md) for the recovery
latch, revision 13 evidence, D-087 recipe and unchanged D-104 limitation. Read the
[updater/installer hand-off](followup24-updater-2026-10-31.md) for real durable
meter evidence, candidate identity and the remaining barrier/health integration.
Mechanism proofs do not replace assembled acceptance or physical arm64 runs.

## r16 M-1

A freeze decision immediately withdraws forwarding. Its incarnation-scoped
intent is durably written outside SQLite before the meta-row attempt, so a locked
SQLite writer cannot lose it on process restart. The meta write is retried on
every Step until committed, even when the memory latch is already set. An
observation that returned `identity_frozen` is never marked accepted before that
row exists. The sidecar never unfreezes an incarnation and cannot freeze a newly
re-enrolled incarnation with a different ID.

There is also a pre-HTTP durable attempt marker in the observation outbox. If a
process dies after central decides a freeze but before any local response/intent
write, or the sidecar write fails, the next process blocks forwarding and
withholds ACKs until it replays the uncertain observation. Thus a full disk does
not have to successfully write a new file to preserve fail-closed behavior.
A positive replay persists the freeze before acceptance; false/absent responses
cannot clear a decided freeze.

The imported r16 containment, D-100 and quarantine proofs remain regression
tests. The frozen-persist proof's lock/restart setup is preserved. Its original
final assertions unconditionally failed if the meta row was absent before
restart, including when a fixed restart recovered the intent. Those assertions
now require the fixed outcome: frozen after restart, meta row recovered, and no
forwarding after Step. This strengthens the positive regression without changing
the injected failure. Additional tests cover Step retries, withheld acceptance,
and a central freeze response lost before any local freeze write. Identity is
loaded before retrying the sidecar: two successive process exits before Boot,
with the meta-row failure still injected, must preserve the same incarnation.

The specifically requested `node/enforce/review_r16_quarantine_test.go` is the
only path outside the ownership block; it is imported under the explicit §6
instruction. No enforcement production code, updater code, installer tests,
central rate budget or HTTP contract was edited.

## Running and limits

Node make targets now place Go scratch under `.agent/tmp` on the worktree
filesystem because this host's `/tmp` is tmpfs. The live launcher uses Python
TemporaryDirectory and container disk scratch, with no shell cleanup trap.
All Docker objects use `fvpn-w2-agent-core-lab*`. Do not run an integration gate
while two other `gates.sh` or standalone AT make suites are active.

Required order: `make -f mk/node.mk node-lint node-test node-test-integration
node-build`, then `node-test-netlab`, then `node-test-live-netlab`, then root
`make go-lint`, `test -z "$(gofmt -l .)"`, and `git diff --check`. The root gate
also needs disk-backed TMPDIR/GOTMPDIR. The final `.agent/REPORT.md` retains real
exit codes and output tails, including any blocked installer result. No push,
PR, merge or production rollout is part of this lane.

The final ordered node gates, privileged netlab, live lab, root lint and diff
check exited 0. The literal recursive gofmt check exited 0 but reported an
unreadable inherited ignored AdGuard scratch directory; root lint verified all
tracked and unignored Go files. The unchanged installer proof exited 1 after
consuming the real checkpoint and reaching health/rollback: its enrollment-only
control returns 404 for desired state and lease. See the updater/installer note
for the terminal `recovery_required` evidence and owning-file requests. All
matching lane containers, images, networks and volumes were removed.

# AT23–AT27 updater health finding

The first-install `ROLLBACK_FAILED` is the correct fail-closed outcome after
health fails with no safe retained release. The underlying health failure is
an unwired updater health adapter. Being unpublished does not by itself make
an update unhealthy, and publishing the node cannot make this adapter pass.
The earlier `lab_precondition` classification was too narrow.

## Evidence and code path

The live [finding](update-good-finding.md) records the updater exit code,
stdout/stderr, transaction events, health JSON, artifact hash and retained
release pointers. Its first-install transaction reached `checking`, then
`rolling_back`, then `recovery_required`. `processes` and
`identity_and_state` passed; `internet_only`, `dns_policy` and `tunnel_probe`
failed. `last_good` and `rollback_release` were empty. The updater produced
no stdout/stderr for that terminal exit; the SQLite journal supplies the
diagnostic evidence. The separate devstack record at
`main:docs/e2e/runs/2026-10-30/update-good.md` corroborates this shape, but is
not this harness run's evidence.

References below are to the product code in this worktree; this task does
not change `node/updater`:

- `node/updater/host.go:109` documents the fallback; `host.go:151` wires
  `Health: NodeChecks{}` into the CLI engine.
- `node/updater/health.go:59`, `:63`, `:67` return `failed` unconditionally
  for isolation, DNS and tunnel probing. They inspect neither publication
  state nor actual service readiness. This also blocks a published node.
- `node/updater/health.go:28` checks the release tree hash for `processes`;
  this pass is not proof that all installed service versions actually run.
  The lab release contains markers and uses no-op service commands.
- `node/updater/health.go:71` collects all five checks;
  `node/updater/decide.go:229` requires all five to pass.
- `node/updater/apply.go:212` records health; `apply.go:214` begins rollback
  on failure. `apply.go:217` sets `last_good` only after a successful check.
- `node/updater/revoke.go:179` selects a safe retained rollback candidate;
  `node/updater/decide.go:33` returns `ROLLBACK_FAILED` without a last-good
  release. `node/updater/apply.go:238` sends an empty rollback target to
  recovery before the restore `SwitchSymlink` at `apply.go:266`. This
  journal does not establish a failed attempt to restore a symlink.
- `node/updater/engine.go:835` separately gates forwarding on the latest
  publication intent and serving/revocation policy. Health success need
  not publish an unpublished or disabled node.

## Contract interpretation

§13.8 explicitly permits updates and reports while disabled. §13.9 requires
process/version and durable-state checks, isolation/authorization, DNS and
a synthetic probe over the candidate tunnel/forwarding path. Its dedicated
authenticated probe identity gets only exact source, destination, protocol
and port exceptions. The probe can run while all member forwarding is
blocked, without publishing an unpublished/disabled exit. The exception
must expire or be removed and must not authorize member traffic.

The updater therefore needs baseline and candidate-path health independent
of member publication. A reduced process-only profile or skipped DNS,
isolation or tunnel checks would weaken §13.9. Keep the current rejection
until real evidence is available. Publication remains a separate, known
prerequisite blocker for the serving-node clauses of AT23/AT27; fixing it
alone will not resolve the stock updater health failures.

## Requests to orchestrator

Assign the updater owner the missing production health adapter and its
agent/probe integration (`node/updater/host.go:151`, `health.go:59–68`).
Implement the §13.9 checks for published, unpublished and disabled nodes
while preserving member forwarding restrictions and publication intent.
Validate actual service versions, baseline isolation and authorized
candidate-path DNS/tunnel probes, including cleanup and a negative test
that the exception cannot permit arbitrary member/host traffic. Exercise
first install without last-good and rollback with a verified last-good.

Acceptance for the fix: a healthy unpublished/disabled install reaches
`locally_succeeded` and records last-good while member forwarding remains
closed; a genuine health failure remains closed and records the failing
checks, and no safe rollback target still yields `recovery_required`.
Please preserve the original health failure in diagnostics alongside the
terminal rollback decision. Re-run AT23–AT27 after the adapter and separate
publication blocker are fixed. Do not resolve this by marking fallback
checks passed or hand-writing projector-owned observations (D-050).

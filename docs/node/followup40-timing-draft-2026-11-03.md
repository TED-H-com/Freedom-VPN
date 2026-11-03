# Draft D-133 declaration for 37–40 — 2026-11-03

**NOT YET RUN. No reservation is authorized or taken in follow-up 40.**
The orchestrator will schedule a quiet slot with no other lane session.
This draft prepares the next task; it does not extend either exhausted
follow-up 39 reservation or reuse its unqualified legs.

Source preparation required in the scheduled follow-up:

1. Freeze A as the then-current main and merge that exact main into the
   existing w2-agent-core branch, keeping 37–40 and all prior merges intact.
   Freeze B as the final delivered 40 tree including that merge. The functional
   source is `5d964d2a`, including main `090bb97b`; runtime fixes are
   `ef0f2951`, `05c39d5e` and `da63e733`. Use the final delivery SHA from the
   40 hand-off, not 39's
   `edc2f387`, and resolve both arm identifiers to full immutable SHAs.
2. Regenerate both archives, per-blob verification counts and SHA-256 values.
   Recheck main at admission. Commit the completed declaration and runner
   before any measured leg. Pin runner and fixture hashes separately from
   each arm's product tree. Do not manufacture an archive hash in this draft.
3. Run a newly declared A1,B1,A2,B2 quartet. All thirteen functional scenarios
   remain required. Any qualified failure remains a failure; no replacement
   leg conceals it. Physical arm64 and field acceptance remain separate.

Changes needed from `followup39_timing.py` before that future run:

- Replace PREFIX, OUT and TREES with the scheduled follow-up's owned Docker
  prefix and fresh evidence/archive directories; change both OUT assignments,
  including wrapped_tool's entry point. Do not overwrite `.agent/f39/timing`.
- Replace F39_TIMING_ROOT, F39_ORIGINAL_PATH and F39_LEG consistently in the
  driver and generated wrappers. Update the reservation label only when that
  later task authorizes it. The old script remains preserved with its hashes.
- Start a new window ledger with no imported qualified legs. Old refused and
  unqualified receipts remain linked as history. Declare the new reservation
  budget/retry policy in that task rather than silently resetting 39's budget.
- Feed prepare the new A/B SHAs. Keep full Git-blob verification, disk
  TMPDIR/GOTMPDIR, CGO_ENABLED=0, GOTOOLCHAIN=local and cache from `go env
  GOCACHE` (public spelling `~/.cache/go-build`). Retain admission before
  every Go test, make and Docker command and recursive MAKE wrapper propagation.
- Keep the corrected foreground Docker event stream/initial inventory, join
  all workers and remove owned objects on completion. Do not revert to the
  monitoring call that competed with its own make child in 39's first window.

Retain the declared quartet qualification rules: four-second samples, reject
intervals over five seconds, no foreign suite/container, at most 0.5 foreign
CPU cores in every interval, PID-start identity, separate documented exclusions,
vmstat as context only, and immediate disqualification on unreadable state.
Do not exclude other Codex/lane activity to rescue a leg. The quiet slot must
remove that activity before admission. Preserve every refused/failed attempt.

Retain guard/failed-operation streams and packet expiry/renewal predicates.
Declare measurement of the whole serving Step and the pre-Apply accounting
boundary separately from command guard time: the latter alone omits Baseline,
final sampling, fresh baseline preparation and the intent fsync. Follow-up 40
adds a restart-only continuity query and error-path retries; measure rather
than infer quantitative equivalence. No arm, sample or comparison was run
under this draft.

The final delivery SHA is recorded in follow-up 40's hand-off report; the
outcome records the frozen functional source. Substitute that delivered tree
(with the scheduled main merged) for B. These pins prepare a later declaration;
they do not authorize a measured run in follow-up 40.

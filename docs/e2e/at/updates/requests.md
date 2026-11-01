# Updates requests — follow-up 13, 2026-11-01

Run `759d244a-2dc4-460c-9f82-e37ce0ac7560`, product main `5bd751026073444087b32ac2a7115f8566104f8b`.

### Requests to orchestrator

2026-11-01 — **F13-01 → devstack**: add a reliable TEST ONLY exit-status
readback to `deploy/devstack/agent-service.sh` / `deploy/devstack/entrypoint-node.sh`
(or an equivalent owner-supported observation API). The regular supervisor can
reap the agent before a 1 ms `/proc` sampler sees the zombie. This run misses
node stop 1 (final_sampled) despite recording command success and retained lease.
The fixture owner should record the actual wait status and any SIGKILL escalation
without changing shutdown limits or AT predicates. Those files are outside this
lane, so only this measurement item is handed off; they were not edited.
The row stays unobserved, and this run is never retried for a pass. This is an
observation gap, not proof of agent-core or updater failure.

Request count: **1**.

### Owner closures and limits

- **F11-01 CLOSED:** the shared wrapper installs with exit 0; 27/27 installer
  checks, original generated reporting and the TEST ONLY loopback opt-in remain.
- **F11-02 CLOSED by #351:** retained verified archive versus guarded absent-cache
  fallback documentation remains aligned with installer 8. No new origin input.
- **F18-01 intended negative:** stopped-producer 1/1/2 and CHECKPOINT_FAILED remain;
  no failed checkpoint is relabeled as successful upgrade acceptance.

Historical direct durations/exit statuses for run 2babf9b6 were never recorded.
The [full comparison](agent-stops.md) gives every available larger-stage bracket
and labels missing data. No source edit or rerun can repair that historical gap.
All updater 19 deployment/fault/bound measurement gaps remain as listed in
[Follow-up 13](follow-up-13.md#registers-and-remaining-measurements).

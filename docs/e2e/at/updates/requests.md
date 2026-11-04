# Updates requests — follow-up 15, 2026-11-03

Last live run `2704600f-e076-4a59-9338-09befa2af70e` (2026-11-01, product main `f680d6ddb8fd19979c0471a2ce06ffc6e16c7cd2`); offline adoption follow-up 15 on 2026-11-03.

### Requests to orchestrator

2026-11-03 — **0 requests**. Devstack 33's (`PR #441`) exported node conflict-handling hand-off (`docs/e2e/RESPONSES.md` 2026-11-03 follow-up 33 and `docs/e2e/runs/2026-11-03/followup32/audit.md`) is answered offline in [follow-up-15-2026-11-03.md](follow-up-15-2026-11-03.md): `handoffRun.retire` (`tests/e2e/at/updates/installer_test.go`) now routes `DELETE /api/v1/admin/nodes/{node}` through `Stack.WriteNode` with `retiredNode` (`lifecycle_state == "retired"`), embeds `fixture_retirement_node_write` beside `fixture_retirement_http`, keeps the post-write `fixture_retired` observed-state check unchanged, and enforces package-wide coverage with `TestUPD02_NodeWriteAudit` (`tests/e2e/at/updates/node_write_audit_test.go`) and offline stub tests (`tests/e2e/at/updates/node_write_test.go`). Live proof waits for the next assembled run.

2026-11-01 — **0 requests**. Devstack 25/26's retained parent receipt and
aligned stop path are consumed; all current updater stops have their own
agent outcome. Full [stop comparison](agent-stops.md) and [run report](follow-up-14.md).

### Owner closures and limits

- **Devstack 32 / 33 node-write adoption ANSWERED (2026-11-03):** `handoffRun.retire` (`DELETE /api/v1/admin/nodes/{node}`) uses `Stack.WriteNode` with `NodeFieldsMatch(map[string]string{"lifecycle_state": "retired"})`, embeds `fixture_retirement_node_write` beside `fixture_retirement_http`, preserves the `fixture_retired` observed-state gate, and is guarded by `node_write_audit_test.go` and `node_write_test.go` ([follow-up-15-2026-11-03.md](follow-up-15-2026-11-03.md)).
- **F11-01 CLOSED:** shared-wrapper installed / exit 0, installer 27/27,
  generated reporting unchanged and TEST ONLY loopback opt-in retained.
- **F11-02 CLOSED:** #351's retained verified archive versus guarded uncached
  fallback contract remains aligned; no new origin input.
- **F18-01 intended negative:** stopped producer stays 1/1/2 and CHECKPOINT_FAILED.
- **Updates F13-01 historical:** run 759d244a first AT23 exit stays unobserved in
  the [immutable record](history/follow-up-13/requests.md). This run's matching
  parent exit-0 receipt is a new sample; it cannot fill that old gap. Members'
  F13-01 remains a different finding.

All updater 19 measurement gaps remain under
[remaining measurements](follow-up-14.md#registers-and-remaining-measurements).

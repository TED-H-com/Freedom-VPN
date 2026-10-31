# Offline catch-up and post-batch inheritance — follow-up 8

Run `71f2aa37-b0b9-4568-abe0-2ba7665377e4`, source `bc93ce430ebe8a50e5dfdaf4121352047fe89249` (on `main` `52dc77026cb168df5684ae2ccd5cbcadb4dbaeff`), evidence `2026-10-30T20:59:15Z`. Prefix `w4-updates`, port base 42000.

Raw evidence: [inheritance.json](inheritance.json), [AT24.json](AT24.json).

## 1. Natural offline snapshot and catch-up completion

- Primary node `01a18e94-1ede-7da3-b7c1-ae4f674321c0` (incarnation `01a18e94-2123-7ae8-9820-7a80fca3fe42`) was stopped cleanly and the harness waited for the configured `180s` online window (`nodes.json` `online_seconds=180`) to expire naturally without rewriting any database timestamp.
- Batch `01a18e9a-3257-7742-aa58-6a604f3132d7` for release `1.0.2` (`01a18e94-0df0-7894-9874-5ef3b6736565`, `set_fleet_default=true`, generation `5`) was accepted with `target_count=1`, `online_at_snapshot=0`, `offline_at_snapshot=1`, and target state `waiting_online` (`current_release_id=01a18e94-0630-7535-a302-8ebf65284d54`).
- After resuming the primary agent, signed desired state delivered job `01a18e9a-3258-7b5e-969f-67f988279a76`, `fvpn-updater` verified and applied `1.0.2` (`locally_succeeded`, all 5 D-053 health checks `passed`), and central confirmed batch `01a18e9a-3257-7742-aa58-6a604f3132d7` (`state="completed"`, target `state="succeeded"`, `unconfirmed=false`).

## 2. Post-batch new-node inheritance (`node2`)

- A second exit node container (`fvpn-w4-updates-node2`) was started and enrolled via `stack.StartSecondExit` and `stack.EnrollAt`, producing `node_id: "01a18e9b-4b6d-79af-b55c-7c21a5c26720"`, `incarnation_id: "01a18e9b-4c5d-7186-84d7-d2e8534f2c97"`.
- On its first desired-state sync, `node2` inherited the fleet default release `1.0.2` (`01a18e94-0df0-7894-9874-5ef3b6736565`) through its own separate `first_sync` batch `01a18e9b-4cc8-701b-be8a-48ee7188d747` and job `01a18e9b-4cc9-7003-9edd-0c917875f254`.
- `fvpn-updater` ran on `node2`, reached `locally_succeeded` with all 5 health checks `passed`, and central confirmed `node2`'s `first_sync` batch (`state="completed"`, `counts.succeeded=1`, `unconfirmed=false`).
- Provenance query confirms:
  - `scope`: `"first_sync"`
  - `first_sync_node_id`: `"01a18e9b-4b6d-79af-b55c-7c21a5c26720"`
  - `new_node_created_after_original_batch`: `true`
  - `original_target_count`: `1`
  - `new_node_in_original_snapshot`: `false`
  - `fleet_default_release_id`: `"01a18e94-0df0-7894-9874-5ef3b6736565"`

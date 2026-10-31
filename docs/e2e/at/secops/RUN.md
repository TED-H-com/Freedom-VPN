# Follow-up 9 run instructions

Run from this worktree with Go 1.27.3 and Docker:

```sh
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
make -f mk/at-secops.mk at-secops-followup9
make go-lint
go vet ./...
```

`at-secops-followup9` runs `tests/e2e/at/secops/followup9_run.py` in the foreground:

1. Verifies that zero `fvpn-w4-secops-*` Docker containers, volumes, or networks exist (`inventory_before`).
2. Brings up a fresh lab stack (`make -f mk/e2e.mk e2e-up` with `FVPN_E2E_PREFIX=w4-secops`, `FVPN_E2E_PORT_BASE=30000`, `E2E_RESET=1`).
3. Runs `TestAT30_Followup9_LiveRestoreReconcile` (`tests/e2e/at/secops/followup9_live_test.go`):
   - **Step 1 (`T0`)**: Seeds two paid members (`m1`, `m2`) with native devices on `fvpn-w4-secops-node`, runs counted traffic until usage checkpoints are ACKed (`L0`), captures SQLite + `fvpn-agent usage snapshot`, and takes a pgBackRest full backup (`T0`).
   - **Step 2 (`T1`)**: Runs additional counted traffic (`L1 > L0`), applies subscription cancellation + full refund on `m1` (`P1`), revokes `m2`'s second device `peerM2D2`, authorizes `peerM3D1` for post-`T0` member `m3` (`N1`), and archives WAL (`wal-switch`).
   - **Step 3 (Restore to `T0` + SECOPS-19 visibility)**: Verifies `restore-pitr.sh` primary-overwrite refusal (exit `2`), runs the 2-minute fail-safe negative without `freeze`/`begin-recovery`, then restores `T0` cleanly (`post-restore-checklist.sql` `6/6`) and executes the 5-step product restore protocol (`freeze` → `begin-recovery` → node archive replay → `complete-recovery`), capturing `node_observations.report_payload.usage` and `fvpn-agent usage snapshot -json` both during the restore hold (non-zero unacked) and after `complete-recovery` (`0` unacked), and measuring `RTO`.
   - **Step 4 (D-076 Fence, Re-Grant, Two Negatives & Reconciliation)**: Verifies `peerM2D2` is fenced (`device_authorization_fences` row, worker log line with 4 fields, worker counter `fvpn_device_restore_fence_decisions_total`, `device.authorization.external_withdrawal` audit, `GET /api/v1/member/devices` `status=pending`, data-plane withdrawn), exercises `AuthorizeDevice` re-grant (proving `T0` key replay preserves the fence while a fresh key clears the fence and restores single NetBird grant and counted flow), and verifies both negatives (central revoke `status=revoked` unfenced; post-restore fresh device enrollment `status=authorized` unfenced).
4. Unconditionally tears down all `fvpn-w4-secops-*` containers, volumes, networks, and restore images and verifies `inventory_after` is zero.
5. Renders records via `python3 tests/e2e/at/secops/followup9_records.py`.

Previous Follow-up 8 root records are archived with SHA-256 digests under `docs/e2e/at/secops/history/follow-up-8/`.

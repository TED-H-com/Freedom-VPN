# Follow-up 8 run instructions

Run from this worktree with Go 1.27.3 and Docker:

```sh
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
make -f mk/at-secops.mk at-secops-followup8
make go-lint
go vet ./...
```

`at-secops-followup8` runs `tests/e2e/at/secops/followup8_run.py` in the foreground:

1. Verifies that zero `fvpn-w4-secops-*` Docker containers, volumes, or networks exist (`inventory_before`).
2. Brings up a fresh lab stack (`make -f mk/e2e.mk e2e-up` with `FVPN_E2E_PREFIX=w4-secops`, `FVPN_E2E_PORT_BASE=30000`, `E2E_RESET=1`).
3. Runs `TestAT30_Followup8_LiveRestoreReconcile` (`tests/e2e/at/secops/followup8_live_test.go`):
   - **Step 1 (`T0`)**: Seeds two paid members (`m1`, `m2`) with native devices on `fvpn-w4-secops-node`, runs counted traffic until usage checkpoints are ACKed (`L0`), captures the node's retained SQLite store (`/var/lib/vpn-node/state/usage.db`), and takes a pgBackRest full backup + records `T0` UTC/LSN via `deploy/backup/backup.sh`.
   - **Step 2 (`T1`)**: Runs additional counted traffic (`L1 > L0`), applies a subscription cancellation + full refund on `m1` in `fvpn-w4-secops-payment` (`P1`), revokes `m2`'s second device `peerM2D2` (`DELETE /api/v1/member/devices/{id}`), authorizes two new `T0..T1` devices (`peerM3D1` for post-`T0` member `m3` and `peerM2D3` for `T0` member `m2`) (`N1`), and archives WAL (`wal-switch`).
   - **Step 3 (Restore to `T0`)**: Verifies `restore-pitr.sh` primary-overwrite refusal (exit `2`), runs a **2-minute fail-safe negative** on a restored `T0` database without `freeze`/`begin-recovery` (proving `T0..T1` archived events are skipped while idle and trigger `ErrAckRegress` on new traffic), then restores `T0` cleanly (`post-restore-checklist.sql` `6/6`) and executes the 5-step product restore protocol (`fvpn-usage-admin freeze` → `begin-recovery` → node archive replay → `complete-recovery`), measuring `RTO`.
   - **Step 4 (Reconciliation Observation)**: Observes `usage_replay` (`L2` vs `L1` and `L0`, `RPO`), `provider_reconciliation` (`Central L2` vs `P1`), and `peer_revocation_reconciliation` (`Central L2` vs `N1`).
4. Unconditionally tears down all `fvpn-w4-secops-*` containers, volumes, networks, and restore images (`make -f mk/e2e.mk e2e-down` + restore object cleanup) and verifies `inventory_after` is zero.
5. Renders records via `python3 tests/e2e/at/secops/followup8_records.py`.

Previous Follow-up 7 root records are archived with SHA-256 digests under `docs/e2e/at/secops/history/follow-up-7/`.

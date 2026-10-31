# AT30 Live Central Database Restore & Cross-System Reconciliation — Follow-up 8

Application base `6b98422`; harness `6350853010054cf656465e5e677c60dc29553cb5`; run `0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8`; driver `af67be8e-3c40-4728-95e2-159e2aa5a3f6`.
Window: `2026-10-30T22:00:15Z` → `2026-10-30T22:10:16Z`.

## 1. Checkpoints `T0` and `T1` (`L0`, `L1`, `P1`, `N1`, and Node Retained Records)

| Checkpoint / Artifact | Measured Value |
|---|---|
| `T0` pgBackRest full backup label | `20261030-220248F` (`type=full`) |
| `T0` restore target UTC / LSN | `2026-10-30T22:02:57.160042Z` / `0/7001278` |
| `L0` central usage checkpoint (`highest_contiguous_seq` / `max_seen_seq`) | `2` / `2` (`ledger_epoch=7e3c2c31-adec-45d5-ab75-45dd611d5416`) |
| `L0` central `usage_events` / `usage_samples` / total bytes | `2` / `4` / `28014` bytes |
| Node retained store at `T0` (`/var/lib/vpn-node/state/usage.db`) | `next_seq=3`, `acked_through=2`, `events=2`, `archived=2`, `outbox=0` |
| `T1` UTC after moving world forward | `2026-10-30T22:04:53.811409Z` (`T1 - T0 = 116.651 s`, LSN `0/12003BB0`) |
| `L1` central usage checkpoint (`highest_contiguous_seq` / `max_seen_seq`) | `5` / `5` (`ledger_epoch=7e3c2c31-adec-45d5-ab75-45dd611d5416`) |
| `L1` central `usage_events` / `usage_samples` / total bytes | `5` / `10` / `78951` bytes |
| Node retained store at `T1` (`/var/lib/vpn-node/state/usage.db`) | `next_seq=6`, `acked_through=5`, `events=5`, `archived=5`, `outbox=0` |
| `P1` fake payment provider truth (`m1` / `m2`) | `m1`: `/e2e/snapshot` `past_due` probe returned HTTP `400`; applied `status=canceled, refund_minor=1000` + `charge.refunded` webhook (`202`) → `provider_status=canceled`, `is_active=False`, `refund_records=1`, `entitlement.state=expired`, `device_limit=3`; `m2`: `active` (`paid_invoice=1`) |
| `N1` NetBird & peer changes in `T0..T1` | `peerM2D2` (`01a18ed5-a66c-709d-b5f0-a71fb1deefec`, peer `dbih7mf8k4c00083h1n0`) revoked (`DELETE` HTTP `202`, `device.revoke` audit `01a18ed7-226a-766f-bd6a-6f51faa4c0ca`, `t1_observed_authorized=False`); new `T0..T1` peers authorized: `peerM3D1` (`01a18ed7-b80a-7103-bd08-6e6b18a38226` for post-`T0` member `01a18ed7-982c-705f-b064-378df9dfbf47`) and `peerM2D3` (`01a18ed7-f21e-702f-b39d-38a9dca758b0` for `T0` member `01a18ed5-048e-7432-9a29-8d4168e74564`) |

### Node Retained-Record Exposure (`node/` Source Finding)

- node/agent/observe.go:111-116 sends a zeroed api.UsageOutboxStatus{} in POST /api/v1/agent/observations, and no node HTTP/CLI endpoint exposes retained ACKed usage records. The node persists all usage envelopes in /var/lib/vpn-node/state/usage.db (SQLite WAL tables: meta, events, ledger_epochs, replay_unavailable), keeping ACKed rows with state='archived' for DefaultArchiveRetain = 72h so ReconcileCheckpoint can un-archive and replay them when central rotates ledger_epoch during recovery.

## 2. Restore Command Sequence, Fail-Safe Negative, and Measured `RTO`

### 2.1 Primary-Overwrite Refusal & 2-Minute Fail-Safe Negative (Before `freeze` / `begin-recovery`)

- **Primary-overwrite refusal**: `restore-pitr.sh --isolated-container fvpn-w4-secops-postgres` exited `2` (`refused=True`).
- **2-minute fail-safe negative (`120.12 s`)**: Restored `T0` into `fvpn-w4-secops-restore-neg-live` and started `control` + `worker` **without** running `freeze` or `begin-recovery`:
  - At `T0` restore: central `highest_contiguous_seq=2`, `max_seen_seq=2`, `usage_events=2`.
  - After `25 s` idle: central `highest_contiguous_seq=2`, `max_seen_seq=2`, `usage_events=2` (`Uploader.due()` is `false` because events `3..5` are `state='archived'` on the node).
  - After post-`T1` traffic burst (`seq=6`) and `120 s` total window: central `highest_contiguous_seq=2`, `max_seen_seq=6`, `usage_events=3` (events `3..5` skipped!), while node SQLite store has `next_seq=7`, `acked_through=5`, `events=6`, `archived=5`, `outbox=1` (`seq=6` stuck in `outbox` because central returned `highest_contiguous_seq=2 < acked_through=5`, triggering `ErrAckRegress`).
  - **Why**: Without freeze/begin-recovery rotating ledger_epoch, events 3..5 remain state='archived' on the node (acked_through=5). While idle, Uploader.due() is false and nothing is sent. Once post-T1 traffic creates event > 5 in outbox, the node uploads only the new event under the old ledger_epoch; central advances max_seen_seq while returning highest_contiguous_seq=2, which the node rejects with ErrAckRegress ('usagestore: ack watermark cannot move backwards in the same epoch').

### 2.2 Clean Restore to `T0` + Product Restore Protocol (`docs/usage/restore-runbook.md` §2 Steps 1–5)

| Step | Command / Action | Measured Result |
|---|---|---|
| PITR + Checklist | `deploy/backup/restore-pitr.sh` to `2026-10-30T22:02:57.160042Z` + `deploy/backup/post-restore-checklist.sql` | `3.64 s`, `6/6` checklist sections passed |
| Step 1 (`freeze`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... freeze --actor ... --reason ...` | `0.06 s`, `usage_ledger_state.mode = frozen` |
| Step 2 (`begin-recovery`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... begin-recovery --restored-to 2026-10-30T22:02:57.160042Z --affected-since ...` | `0.07 s`, `mode = reconciling`, new `ledger_epoch = 01a18eda-3ac3-7ae6-bd3c-9a8b8a4e8d18` |
| Steps 3–4 (Node Replay) | Start `control` + `worker`; node receives `409 LEDGER_EPOCH_MISMATCH`, fetches `GET /api/v1/agent/usage-checkpoint`, un-archives `3..5`, and replays | `90.05 s`, `highest_contiguous_seq = 7`, `max_seen_seq = 7` |
| Step 5 (`complete-recovery`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... complete-recovery --actor ...` | `0.06 s`, `mode = normal`, `reconciling_rollups_remaining = 0` |
| Readiness Verification | `GET /api/v1/member/me` (`200`), `GET /api/v1/member/usage` (`200`), fresh `node_observations` row (`True`) | **Protocol RTO (Step 1 `freeze` → ready): `105.94 s`**; **Total Outage RTO (central stop → ready): `111.51 s`** |

## 3. Usage Replay (`L2` vs `L1` and `L0`) and Measured `RPO`

| Metric | Value |
|---|---|
| Status | **passed** |
| `N0` (`T0` highest contiguous seq) / `N1` (`T1` highest contiguous seq) | `2` / `5` |
| `L2` replayed events `1..N1` / post-`T1` events `> N1` | `5` / `2` |
| Canonical `payload_hash` matches across `1..N1` | `5 / 5` |
| `L2` bytes `1..N1` (matches `L1` `78951`) / post-`T1` bytes | `78951` / `15257` |
| Lost `T0..T1` ACKed events / Double-counted events / `usage_ledger_gaps` | `0` / `0` / `0` |
| Pre-replay database RPO (`T1 - T0`) / Post-replay service usage RPO | `116.651 s` / `0 s` (`0` bytes lost within the node's 72h archive retention window) |

## 4. Provider Reconciliation (`Central L2` vs `P1`)

| Metric | Value |
|---|---|
| Status | **passed** |
| `m1` (`sub_e2e_1`, canceled + full refund in `T0..T1`) in `L2` | `provider_status=canceled`, `is_active=False`, `entitlement.state=expired`, `device_limit=3`, `refund_rows=1`, `http_200_after_reconcile=False` |
| `m2` (`sub_e2e_2`, unchanged active subscription) in `L2` | `provider_status=active`, `is_active=True`, `entitlement.state=active`, `device_limit=3`, `paid_invoice_rows=1`, `entitlement_extended=False` |
| `billing.reconciled` audit events in `L2` | `58` (converged via worker `billing.reconcile` querying fake Stripe `GET /v1/subscriptions` & `GET /v1/invoices` without re-delivering the webhook) |

## 5. Peer & Revocation Reconciliation (`Central L2` vs `N1`)

| Scenario | Measured Outcome in `L2` |
|---|---|
| **(a) Device revoked in `T0..T1` (`peerM2D2`: `01a18ed5-a66c-709d-b5f0-a71fb1deefec`, peer `dbih7mf8k4c00083h1n0`)** | **Discrepancy (`status=blocked`)**: At `T1` before restore, `peerM2D2` was `status=revoked` in central (`t1_observed_authorized=False`). After restoring central DB to `T0` (`member_devices.status='authorized'`), `members.reconcile` → `SyncMember` → `SyncDevice` (`apps/worker/jobs_members.go:158`, `internal/devices/worker.go:72-93`) treated `T0` `member_devices.status='authorized'` as authoritative and called `SetAuthorization(peer, true)`, re-authorizing `peerM2D2` in NetBird (`l2_observed_authorized=True`, `l2_central_status=authorized`, `l2_in_active_device_bindings=True`). |
| **(b) Post-`T0` member's `T0..T1` peer (`peerM3D1` for member `01a18ed7-982c-705f-b064-378df9dfbf47`)** | **Passed (#205 verified)**: Member `01a18ed7-982c-705f-b064-378df9dfbf47` does not exist in `T0` DB. Across multiple `devices.discover` sweeps, central kept `central_member_device_rows_l2=0`, emitted exactly `1` `device.authorize` (`result=denied`, `reason=identity_unmapped`) audit row (`deduplicated_across_sweeps=True`), and `peerM3D1` could not forward (`http_200=False`). |
| **(c) Existing `T0` member's `T0..T1` peer (`peerM2D3` for member `01a18ed5-048e-7432-9a29-8d4168e74564`)** | **Converged**: `devices.discover` matched `peerM2D3` via `m2`'s `T0` `netbird_user_bindings` and re-admitted it (`rediscovered_rows_in_l2=1`, `status=authorized`). |

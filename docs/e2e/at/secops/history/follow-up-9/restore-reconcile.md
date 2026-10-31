# AT30 Live Central Database Restore, D-076 Fence & SECOPS-19 Visibility — Follow-up 9

Application base `205633f` (#231: #227 D-076 fence, #226 SECOPS-19 observation `UsageOutboxStatus`, #225 `fvpn-agent usage snapshot`); harness `205633f0de8b3befbd563a092b856c0bc089e9c9`; run `42396951-2d9a-49f4-95c3-45f33d9519d5`; driver `4e0cca84-5c8e-4b94-88d2-4bfc6843a0b0`.
Window: `2026-10-31T01:36:15Z` → `2026-10-31T01:46:57Z`.

## 1. Checkpoints `T0` and `T1` (`L0`, `L1`, `P1`, `N1`, and Node Retained Records)

| Checkpoint / Artifact | Measured Value |
|---|---|
| `T0` pgBackRest full backup label | `20261031-013901F` (`type=full`) |
| `T0` restore target UTC / LSN | `2026-10-31T01:39:06.450534Z` / `0/7000DA8` |
| `L0` central usage checkpoint (`highest_contiguous_seq` / `max_seen_seq`) | `1` / `1` (`ledger_epoch=3674561b-32a3-4e5b-ac37-f8c25057d65b`) |
| `L0` central `usage_events` / `usage_samples` / total bytes | `1` / `3` / `28015` bytes |
| Node retained store at `T0` (`/var/lib/vpn-node/state/usage.db` & `fvpn-agent usage snapshot`) | `next_seq=2`, `acked_through=1`, `events=1`, `archived=1`, `outbox=0` |
| `T1` UTC after moving world forward | `2026-10-31T01:40:36.751342Z` (`T1 - T0 = 90.301 s`, LSN `0/10000380`) |
| `L1` central usage checkpoint (`highest_contiguous_seq` / `max_seen_seq`) | `3` / `3` (`ledger_epoch=3674561b-32a3-4e5b-ac37-f8c25057d65b`) |
| `L1` central `usage_events` / `usage_samples` / total bytes | `3` / `7` / `78833` bytes |
| Node retained store at `T1` (`/var/lib/vpn-node/state/usage.db` & `fvpn-agent usage snapshot`) | `next_seq=4`, `acked_through=3`, `events=3`, `archived=3`, `outbox=0` |
| `P1` fake payment provider truth (`m1` / `m2`) | `m1`: `/e2e/snapshot` `past_due` probe returned HTTP `400`; applied `status=canceled, refund_minor=1000` + `charge.refunded` webhook (`202`) → `provider_status=canceled`, `is_active=False`, `refund_records=1`, `entitlement.state=expired`, `device_limit=3`; `m2`: `active` (`paid_invoice=1`) |
| `N1` NetBird & peer changes in `T0..T1` | `peerM2D2` (`01a18f9b-a2a0-728f-8482-b08cc9b1a3ee`, peer `dbikd1v8k4c000eb212g`) revoked (`DELETE` HTTP `202`, `device.revoke` audit `01a18f9d-1f25-77ca-9dfe-100b55016228`, `t1_observed_authorized=False`); post-`T0` member `m3` (`01a18f9d-47af-7dc8-96c8-59a6e34b69c7`) authorized `peerM3D1` (`01a18f9d-76df-729e-a5f3-349d700e823c`) |

## 2. Restore Command Sequence, Fail-Safe Negative, Measured `RTO`/`RPO`, and Comparison with Follow-up 8 (`#221`)

### 2.1 Primary-Overwrite Refusal & 2-Minute Fail-Safe Negative (Before `freeze` / `begin-recovery`)

- **Primary-overwrite refusal**: `restore-pitr.sh --isolated-container fvpn-w4-secops-postgres` exited `2` (`refused=True`).
- **2-minute fail-safe negative (`120.12 s`)**: Restored `T0` into `fvpn-w4-secops-restore-neg-live` and started `control` + `worker` **without** running `freeze` or `begin-recovery`:
  - At `T0` restore: central `highest_contiguous_seq=1`, `max_seen_seq=1`, `usage_events=1`.
  - After `25 s` idle: central `highest_contiguous_seq=1`, `max_seen_seq=1`, `usage_events=1` (`Uploader.due()` is `false` because events `2..3` are `state='archived'` on the node).
  - After post-`T1` traffic burst (`seq=6`) and `120 s` total window: central `highest_contiguous_seq=1`, `max_seen_seq=4`, `usage_events=2` (events `2..3` skipped!), while node SQLite store has `next_seq=5`, `acked_through=3`, `events=4`, `archived=3`, `outbox=1` (`ErrAckRegress` log lines: `0`).

### 2.2 Clean Restore to `T0` + Product Restore Protocol (`docs/usage/restore-runbook.md` §2 Steps 1–5)

| Step | Command / Action | Measured Result |
|---|---|---|
| PITR + Checklist | `deploy/backup/restore-pitr.sh` to `2026-10-31T01:39:06.450534Z` + `deploy/backup/post-restore-checklist.sql` | `7.17 s`, `6/6` checklist sections passed |
| Step 1 (`freeze`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... freeze --actor ... --reason ...` | `0.06 s`, `usage_ledger_state.mode = frozen` |
| Step 2 (`begin-recovery`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... begin-recovery --restored-to 2026-10-31T01:39:06.450534Z --affected-since ...` | `0.07 s`, `mode = reconciling`, new `ledger_epoch = 01a18f9f-c20a-7716-9f31-8b078efba999` |
| Steps 3–4 (Node Replay) | Start `control` + `worker`; node receives `409 LEDGER_EPOCH_MISMATCH`, fetches `GET /api/v1/agent/usage-checkpoint`, un-archives `2..3`, and replays | `84.75 s`, `highest_contiguous_seq = 4`, `max_seen_seq = 4` |
| Step 5 (`complete-recovery`) | `go run ./internal/usage/cmd/fvpn-usage-admin -config ... complete-recovery --actor ...` | `0.06 s`, `mode = normal`, `reconciling_rollups_remaining = 0` |
| Readiness Verification | `GET /api/v1/member/me` (`200`), `GET /api/v1/member/usage` (`200`), fresh `node_observations` row (`True`) | **Protocol RTO: `112.35 s`**; **Total Outage RTO: `121.88 s`** |

### 2.3 Measured `RPO` / `RTO` Comparison: Follow-up 9 vs Follow-up 8 (`#221`)

| Metric | Follow-up 8 (`#221`, `0f5fb60d`) | Follow-up 9 (`#231`, `42396951`) |
|---|---|---|
| Pre-replay database RPO (`T1 - T0`) | `116.651 s` | `90.301 s` |
| Post-replay service usage RPO | `0 s` (`0` bytes lost) | `0 s` (`0` bytes lost) |
| Protocol RTO (Step 1 `freeze` → ready) | `105.94 s` | `112.35 s` |
| Total outage RTO (central stop → ready) | `111.51 s` | `121.88 s` |

## 3. SECOPS-19 Visibility (`#226` `UsageOutboxStatus` in Observations & `#225` `fvpn-agent usage snapshot`)

| Phase / Surface | `unacked_event_count` / `unacked_count` | `unacked_payload_bytes` | `ledger_epoch` | `meter_epoch_id` / `meter_epochs` |
|---|---|---|---|---|
| **During restore hold (`reconciling`)** — `node_observations.report_payload.usage` | `3` | `"2352"` | `01a18f9f-c20a-7716-9f31-8b078efba999` | `01a18f9d-807e-78ee-8f36-aa1592b097e0` |
| **During restore hold (`reconciling`)** — `fvpn-agent usage snapshot -json` | `3` (`acked_through=1`, `last_event_seq=4`) | `"2352"` | `01a18f9f-c20a-7716-9f31-8b078efba999` | `5` epochs (`current=01a18f9d-807e-78ee-8f36-aa1592b097e0`) |
| **After `complete-recovery` & replay** — `node_observations.report_payload.usage` | `0` | `"0"` | `01a18f9f-c20a-7716-9f31-8b078efba999` | `01a18f9d-807e-78ee-8f36-aa1592b097e0` |
| **After `complete-recovery` & replay** — `fvpn-agent usage snapshot -json` | `0` (`acked_through=4`, `last_event_seq=4`) | `"0"` | `01a18f9f-c20a-7716-9f31-8b078efba999` | `5` epochs (`current=01a18f9d-807e-78ee-8f36-aa1592b097e0`) |

## 4. Usage Replay (`L2` vs `L1` and `L0`) and Provider Reconciliation (`Central L2` vs `P1`)

| Subsystem | Measured Outcome |
|---|---|
| **Usage Replay (`L2`)** | **passed**: `N0=1` → `N1=3`, replayed `3/3` events (`3/3` canonical SHA-256 matches, `78833` B), `1` post-`T1` event (`10160` B), `0` lost, `0` double-counted, `0` gaps. |
| **Provider Reconciliation (`P1` → `L2`)** | **passed**: `m1` converged to `provider_status=canceled`, `is_active=False`, `refund_rows=1`, `entitlement.state=expired`, `device_limit=3` via `60` `billing.reconciled` audits without webhook re-delivery; `m2` remained `active` (`entitlement_extended=False`). |

## 5. SECOPS-18 Closure Evidence: D-076 Restore Authorization Fence, Two Negatives, and Re-Grant

### 5.1 Restored `T0..T1` Revoked Device (`peerM2D2`) Fenced Instead of Re-Granted

| Proof Artifact | Measured Value |
|---|---|
| `device_authorization_fences` row (`SELECT`) | `count=1`: `device_id=01a18f9b-a2a0-728f-8482-b08cc9b1a3ee`, `fenced_at=2026-10-31T01:42:55.616525+00:00`, `observed_group_ids=[]` |
| Worker structured log line (`msg="device authorization unconfirmed during ledger restore"`) | `device_id=01a18f9b-a2a0-728f-8482-b08cc9b1a3ee`, `member_id=01a18f9a-fec8-7838-a19d-b716062c41b8`, `peer_id=dbikd1v8k4c000eb212g`, `ledger_mode=reconciling` |
| Worker Prometheus counter (`http://127.0.0.1:9091/metrics`) | `fvpn_device_restore_fence_decisions_total 2` (`value=2`) |
| Audit row (`device.authorization.external_withdrawal`) | `count=1`: `id=01a18f9f-c780-7969-ba2d-140363d91810`, `resource=member_device:01a18f9b-a2a0-728f-8482-b08cc9b1a3ee`, `result=succeeded`, `after_diff={"reason": "external_withdrawal", "peer_id": "dbikd1v8k4c000eb212g", "member_id": "01a18f9a-fec8-7838-a19d-b716062c41b8", "observed_groups": []}` |
| Member `GET /api/v1/member/devices` status | `status="pending"` (`observed_authorized=False`, `in_netbird_paid_group=False`) |
| Data-plane proof (`peerM2D2` vs sibling `peerM2D1`) | `peerM2D2` stays withdrawn (`counted=False`, `blocker="expected exactly one advertised product exit"`); sibling `peerM2D1` forwards counted traffic (`counted=True`) |

### 5.2 Re-Grant via `AuthorizeDevice` (`operation kind = member.device_authorize`)

- **Note**: re-authorization admin HTTP route pending contract revision 12 / D-077; exercised AuthorizeDevice (operation kind = member.device_authorize) via devices.Service.AuthorizeDevice against the live restored PostgreSQL and worker.
- **Replay of original `T0` idempotency key**: returned `Replayed=True` and left `device_authorization_fences` intact (`fence_rows_after_t0_replay=1`).
- **Fresh `AuthorizeDevice` (`operation_id=01a18f9b-a2a0-7f20-8e14-9bad5b83e8e6`, `kind=member.device_authorize`)**: deleted the `device_authorization_fences` row (`fence_rows_after_fresh_regrant=0`), bumped `resource_version` `1 → 2`, worker synced NetBird (`observed_authorized=True`, `operation_status=succeeded`), `GET /api/v1/member/devices` returned `status="authorized"`, and counted data-plane flow resumed (`counted=True`, delta `{'download': '1515', 'upload': '1066'}`).

### 5.3 Two Negatives & Unmapped Discovery Deduplication

| Negative / Scenario | Measured Result |
|---|---|
| **Negative 1 — Never-observed device enrolled after restore (`peerM2D3`: `01a18fa2-accf-7f00-a148-a5db4b5aa877`)** | **Fresh grant (no fence)**: `GET /api/v1/member/devices` → `status="authorized"`, `observed_authorized=True`, `fence_row_count=0`, `external_withdrawal_audit_count=0`, worker fence counter `2 → 2`, counted data-plane flow `counted=True`. |
| **Negative 2 — Device centrally revoked after restore (`peerM2D1`: `01a18f9b-67d0-7907-be68-e00da5689f7b`)** | **Revoked (not fenced, not `pending`)**: `DELETE /api/v1/member/devices/{id}` → HTTP `202` (`operation_status=succeeded`), `GET /api/v1/member/devices` → `status="revoked"`, `fence_row_count=0`, `external_withdrawal_audit_count=0`, worker fence counter `2 → 2`, data-plane withdrawn (`counted=False`). |
| **Post-`T0` unmapped member device (`peerM3D1` for `01a18f9d-47af-7dc8-96c8-59a6e34b69c7`)** | **Deduplicated (#205)**: `central_member_device_rows_l2=0`, `identity_unmapped_audit_count=1` (`deduplicated_across_sweeps=True`). |

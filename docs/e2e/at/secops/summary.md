# Follow-up 8 — AT30 live central database restore & cross-system reconciliation

Application base `6b98422` (#207); harness `6350853010054cf656465e5e677c60dc29553cb5`; run `0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8`. Shared development host, **not candidate hardware**.

| Measurement | Actual |
|---|---|
| `T0` pgBackRest full backup / restore target | `20261030-220248F` / `2026-10-30T22:02:57.160042Z` (LSN `0/7001278`) |
| `L0` → `L1` central ACKed usage sequence (`N0` → `N1`) | `2` (`28014` B) → `5` (`78951` B) |
| Node retained records at `T1` (`/var/lib/vpn-node/state/usage.db`) | `next_seq=6`, `acked_through=5`, `events=5`, `archived=5`, `outbox=0` (`node/agent/observe.go:111-116` reports zeroed `UsageOutboxStatus`; SQLite retains `archived` rows for 72h) |
| 2-minute fail-safe negative (before `freeze` / `begin-recovery`) | `120.12 s`: central ACK stayed `2`, `max_seen_seq=6`, `usage_events=3` (events `3..5` skipped, `seq=6` stuck in node `outbox` due to `ErrAckRegress`) |
| 5-step product restore protocol (`fvpn-usage-admin`) | `freeze` → `begin-recovery` (new `ledger_epoch=01a18eda-3ac3-7ae6-bd3c-9a8b8a4e8d18`) → node archive replay (`90.05 s`) → `complete-recovery` (`mode=normal`, `0` reconciling rollups) |
| Usage replay (`L2` vs `L1` and `L0`) | **passed**: `5/5` events (`5/5` canonical SHA-256 matches, `78951` B), `0` lost, `0` double-counted, `0` gaps |
| Pre-replay DB RPO (`T1 - T0`) / Post-replay service usage RPO | `116.651 s` / `0 s` |
| Protocol RTO (Step 1 `freeze` → ready) / Total outage RTO | `105.94 s` / `111.51 s` |
| Provider reconciliation (`Central L2` vs `P1`) | **passed**: `m1` converged to `canceled` + `refund=1` + `entitlement.state=expired` (`device_limit=3`) via `billing.reconcile`; `m2` stayed `active` (`entitlement_extended=False`) |
| Unmapped NetBird peer discovery denial (#205) | **passed**: `identity_unmapped` audit count = `1` across multiple `devices.discover` sweeps |
| `T0..T1` device revocation after restore (`peerM2D2`) | **blocked (`SECOPS-18`)**: `SyncDevice` re-authorized `peerM2D2` in NetBird from `T0` `member_devices.status='authorized'` (`l2_observed_authorized=True`, `l2_in_active_device_bindings=True`) |

Full details and command sequences are in [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json). Follow-up 7 records are archived byte-for-byte under [history/follow-up-7](history/follow-up-7/archive-index.json). See [requests](requests.md) and [run instructions](RUN.md).

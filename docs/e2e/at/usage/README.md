# Wave 4 Usage Acceptance Summary (`w4-at-usage` — Follow-up 3)

- **Run ID**: `e1f8c483`
- **Recorded At (UTC)**: `2026-10-31T02:39:12Z`
- **Source Commit**: `46081fb0e4011b91e9f9f93be23e04e0f223d42d` (dirty: `false`)
- **Stack Instance**: `FVPN_E2E_PREFIX=w4-usage` (`fvpn-w4-usage-*`), `FVPN_E2E_PORT_BASE=48000`
- **Primary Exit Node**: `node_id=01a18f2d-e7d6-770e-9b8c-edb2016a7db9`, `incarnation_id=01a18f2d-e830-7e38-84eb-493517bd4460`, `exit_id=01a18f2d-e7d6-770f-9d32-b5397d190043` (`fvpn-exit-1836ffa7f54af96f436bcb78`)
- **Second Exit Node (`AT19` & `ENR-03`)**: `node_id=01a18fd0-4188-77d3-9e94-5c5bdc8f8d29`, `incarnation_id=01a18fd1-f3d3-7868-9373-487f724b9df7` (initial: `01a18fd0-42b9-7e45-97db-c16ad22f59d0`), `exit_id=01a18fd0-4188-77d4-a466-c66dc72ab9d4` (`fvpn-exit-cd7bd7ca8917d3f58c624b1b`)
- **Non-Carrying Registered Node (`AT19`)**: `node_id=01a18fd1-3793-7120-9318-d306221ee728`, `incarnation_id=01a18fd1-3795-7559-b3ab-ff42bafcc873` (`desired_config_generation=0 -> 0`)
- **Admitted Member Devices**: `memberA` device `01a18f5f-5980-785f-9cfd-0e6860b04f5b`, `memberB` device `01a18f5f-944b-7178-8130-c32a1397fa0b`
- **Totals**: `0 pass`, `6 blocked`, `0 fail`
- **Run Archives**: [history/initial/](history/initial/) (#80), [history/followup2/](history/followup2/) (#214)

## Acceptance Matrix (AT14–AT19)

| Case | Title | Requirements | Status | Evidence |
|---|---|---|---|---|
| `AT14` | 轉發許可流量計量正確性與排除項 (Forward-Admission Traffic Metering & Exclusions) | `MTR01`, `MTR05`, `API01`, `DATA01` | `blocked` | [AT14.md](AT14.md) / [AT14.json](AT14.json) |
| `AT15` | 用量批次冪等、亂序缺口、衝突稽核與簽章檢查點恢復 (Idempotent Usage Ingest, Contiguous ACK, Conflict Audit & Signed Checkpoint Recovery) | `MTR02`, `MTR03`, `MTR04`, `API01`, `DATA01` | `blocked` | [AT15.md](AT15.md) / [AT15.json](AT15.json) |
| `AT16` | 節點計量器重啟、計數器重置、Outbox 佇列上限與安全停轉發 (Node Sampler Restart, Counter Reset, Outbox Caps & Safety Fail-Closed) | `MTR01`, `MTR02`, `MTR03`, `MTR04`, `UPD03` | `blocked` | [AT16.md](AT16.md) / [AT16.json](AT16.json) |
| `AT17` | UTC 自然月切分、延遲事件歸屬與時鐘回撥保護 (UTC Month Boundary, Late-Arriving Events & Clock Rollback Protection) | `MTR01`, `MTR02`, `MTR04`, `API01`, `DATA01` | `blocked` | [AT17.md](AT17.md) / [AT17.json](AT17.json) |
| `AT18` | 多階層速率整形、有效速率計算與宿主保底頻寬 (Multi-Tier Rate Shaping, Effective Rate & Host Reserve) | `FUP01`, `FUP02`, `FUP03` | `blocked` | [AT18.md](AT18.md) / [AT18.json](AT18.json) |
| `AT19` | 跨節點累計、Shadow/Enforce 模式切換、月初重置與臨時覆寫 (Cross-Node Fair-Use, Shadow/Enforce, Monthly Reset & Temporary Override) | `FUP01`, `FUP02`, `FUP03`, `FUP04`, `ENR03`, `API01` | `blocked` | [AT19.md](AT19.md) / [AT19.json](AT19.json) |

## Follow-up 3 Measured Timelines

### 1. AT19 Fair-Use Recompile & `fair_use.applying` Convergence (PR #222 Verified)

| Transition | Carrying `node` Gen | Carrying `node2` Gen | Non-Carrying `other` Gen | Clearing Observation | Wall-Clock (`ms`) | Tier 2 Rate (`5Mbit`/`2Mbit`) | Replay Gen After |
|---|---|---|---|---|---|---|---|
| `policy.activated` (`mode=enforce`) | `126 -> 127` | `4 -> 7` | `0 -> 0` | `5a9a70e5-b7d2-4c4d-9175-fc7acba16c2b` (`gen=7`, `2026-10-31T02:37:06.496862Z`) | `5387 ms` | `DesiredState 5Mbit/2Mbit (kernel tc blocked by FLOWTABLE_BYPASS)` | `127 / 7 / 0` |
| `fup.member_generation_changed` (usage threshold) | `128 -> 129` | `8 -> 9` | `0 -> 0` | `922edc3f-c2b6-4f4d-a4aa-3bea02b6fa8d` (`gen=9`, `2026-10-31T02:37:22.991386Z`) | `6151 ms` | `DesiredState 5Mbit/2Mbit (kernel tc blocked by FLOWTABLE_BYPASS)` | `129 / 9 / 0` |

### 2. D-067 Provisional-Lease Timing (`awaiting_member_grants` -> Populated Lease)

| Cycle | Ready Observation (`t_ready`) | Populated Lease (`t_lease`) | Apply Blocker After Populated Lease | Ready -> Lease (`ms`) | Max Allowed (`ms`) | Checks (`lease`/`qos`/`source_binding`) | `errors[]` (`stage=lease`, zero `LEASE_EXPIRED`) |
|---|---|---|---|---|---|---|---|
| `bring_up` | `01a18fcd-a1a4-7983-9fe3-aa16f11d129b` (`2026-10-31T02:33:00.584324Z`) | `677c5ad3-b5a0-44dc-b77b-98dec868eb3b` (`members=2`, `2026-10-31T02:33:05.605449Z`) | `FLOWTABLE_BYPASS` (`obs=01a18fce-194d-7c40-b1eb-246cda6a5222`) | `5021 ms` | `65000 ms` | `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` | `count=1` (`AWAITING_MEMBER_GRANTS`), `expired=false` |
| `agent_restart` | `01a18fce-aba1-7b56-bd5e-6dbdccd14664` (`2026-10-31T02:34:08.679543Z`) | `27108c64-ac6e-42b0-828f-8a8b6e1dd982` (`members=2`, `2026-10-31T02:34:13.70479Z`) | `FLOWTABLE_BYPASS` (`obs=01a18fcf-23b7-7718-9eb5-7655c3aae811`) | `5025 ms` | `65000 ms` | `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` | `count=1` (`AWAITING_MEMBER_GRANTS`), `expired=false` |

### 3. `ENR-03` Per-Incarnation Retained Usage (`GET /api/v1/admin/nodes/{id}/incarnations`, PR #210 Verified)

- **Node ID**: `01a18fd0-4188-77d3-9e94-5c5bdc8f8d29` (409 while active: `NODE_INCARNATION_ACTIVE`)
- **Retired Incarnation (`old`)**: `01a18fd0-42b9-7e45-97db-c16ad22f59d0` (`state=retired`, `retire_reason=reinstall`, retained `upload_bytes=1500`, `download_bytes=2000`, unchanged after new traffic: `true`)
- **Active Incarnation (`new`)**: `01a18fd1-f3d3-7868-9373-487f724b9df7` (`state=active`, initial `0/0`, after new traffic `upload_bytes=900`, `download_bytes=1100`)
- **Member Period Continuity (`usage_rollups` / `GET /api/v1/member/usage`)**: before re-enroll `total_bytes=109929`, after re-enroll `total_bytes=109929` (`unchanged=true`), after new traffic `total_bytes=111929` (`accumulated=true`)

## Upstream Product Blockers

- **Owner `w1-netlab` / `w3-devstack-e2e` (regression introduced in PR #220 `2473acf`)**: `VerifyServing` (`node/enforce/verify_foreign.go:107`) rejects Docker Engine's built-in `table ip nat` `POSTROUTING -> DOCKER_POSTROUTING -> xt target "SNAT"` rule inside `node-netns` and `node2-netns` with `FLOWTABLE_BYPASS` (`"table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt"`) whenever `Serve=true` (populated lease). See [requests.md](requests.md) for full root cause and exact lines.

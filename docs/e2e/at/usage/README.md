# Wave 4 Usage Acceptance Summary (`w4-at-usage` — Follow-up 3)

- **Run ID**: `eb79863a`
- **Recorded At (UTC)**: `2026-10-31T02:28:57Z`
- **Source Commit**: `ebfdb644d877961dbf5f8e1e5d6872ede7f27a34` (dirty: `false`)
- **Stack Instance**: `FVPN_E2E_PREFIX=w4-usage` (`fvpn-w4-usage-*`), `FVPN_E2E_PORT_BASE=48000`
- **Primary Exit Node**: `node_id=01a18f2d-e7d6-770e-9b8c-edb2016a7db9`, `incarnation_id=01a18f2d-e830-7e38-84eb-493517bd4460`, `exit_id=01a18f2d-e7d6-770f-9d32-b5397d190043` (`fvpn-exit-1836ffa7f54af96f436bcb78`)
- **Second Exit Node (`AT19` & `ENR-03`)**: `node_id=01a18fc6-fb50-793b-b824-718838f20496`, `incarnation_id=01a18fc8-90a6-781b-ab46-c0a6531d96ed` (initial: `01a18fc6-fc7d-73f6-bf0c-6d8abb6fd46b`), `exit_id=01a18fc6-fb50-793c-bff4-6d42db93bf9d` (`fvpn-exit-c862913a5eb3813c005357c0`)
- **Non-Carrying Registered Node (`AT19`)**: `node_id=01a18fc7-d74c-70f9-9fa0-d48d9f112449`, `incarnation_id=01a18fc7-d74f-7b21-8712-cfcadebddc63` (`desired_config_generation=0 -> 0`)
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
| `policy.activated` (`mode=enforce`) | `117 -> 119` | `4 -> 7` | `0 -> 0` | `139ab64b-59a5-4b60-84d7-73c8eddfbe9b` (`gen=7`, `2026-10-31T02:26:52.060175Z`) | `5384 ms` | `DesiredState 5Mbit/2Mbit (kernel tc blocked by FLOWTABLE_BYPASS)` | `119 / 7 / 0` |
| `fup.member_generation_changed` (usage threshold) | `119 -> 120` | `7 -> 8` | `0 -> 0` | `c14761e0-9137-49ef-9b15-f4d920babbe7` (`gen=8`, `2026-10-31T02:27:07.475006Z`) | `6141 ms` | `DesiredState 5Mbit/2Mbit (kernel tc blocked by FLOWTABLE_BYPASS)` | `120 / 8 / 0` |

### 2. D-067 Provisional-Lease Timing (`awaiting_member_grants` -> Populated Lease)

| Cycle | Ready Observation (`t_ready`) | Populated Lease (`t_lease`) | Apply Blocker After Populated Lease | Ready -> Lease (`ms`) | Max Allowed (`ms`) | Checks (`lease`/`qos`/`source_binding`) | `errors[]` (`stage=lease`, zero `LEASE_EXPIRED`) |
|---|---|---|---|---|---|---|---|
| `bring_up` | `01a18fc4-bea8-78fa-9e94-b33b845f3eaa` (`2026-10-31T02:23:18.190635Z`) | `c85de877-f661-41b7-8b66-d2bcac1288d3` (`members=2`, `2026-10-31T02:23:23.210452Z`) | `` (`obs=00000000-0000-0000-0000-000000000000`) | `5019 ms` | `65000 ms` | `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` | `count=1` (`AWAITING_MEMBER_GRANTS`), `expired=false` |
| `agent_restart` | `01a18fc5-99dc-72da-bd1f-8fe1d954859e` (`2026-10-31T02:24:14.305677Z`) | `6e38985a-51ac-4098-81d6-88e685e60f64` (`members=2`, `2026-10-31T02:24:19.330482Z`) | `` (`obs=00000000-0000-0000-0000-000000000000`) | `5024 ms` | `65000 ms` | `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` / `AWAITING_MEMBER_GRANTS` | `count=1` (`AWAITING_MEMBER_GRANTS`), `expired=false` |

### 3. `ENR-03` Per-Incarnation Retained Usage (`GET /api/v1/admin/nodes/{id}/incarnations`, PR #210 Verified)

- **Node ID**: `01a18fc6-fb50-793b-b824-718838f20496` (409 while active: `NODE_INCARNATION_ACTIVE`)
- **Retired Incarnation (`old`)**: `01a18fc6-fc7d-73f6-bf0c-6d8abb6fd46b` (`state=retired`, `retire_reason=reinstall`, retained `upload_bytes=1500`, `download_bytes=2000`, unchanged after new traffic: `true`)
- **Active Incarnation (`new`)**: `01a18fc8-90a6-781b-ab46-c0a6531d96ed` (`state=active`, initial `0/0`, after new traffic `upload_bytes=900`, `download_bytes=1100`)
- **Member Period Continuity (`member_usage_periods`)**: before re-enroll `total_bytes=97500`, after re-enroll `total_bytes=97500` (`unchanged=true`), after new traffic `total_bytes=99500` (`accumulated=true`)

## Upstream Product Blockers

- **Owner `w1-netlab` / `w3-devstack-e2e` (regression introduced in PR #220 `2473acf`)**: `VerifyServing` (`node/enforce/verify_foreign.go:107`) rejects Docker Engine's built-in `table ip nat` `POSTROUTING -> DOCKER_POSTROUTING -> xt target "SNAT"` rule inside `node-netns` and `node2-netns` with `FLOWTABLE_BYPASS` (`"table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt"`) whenever `Serve=true` (populated lease). See [requests.md](requests.md) for full root cause and exact lines.

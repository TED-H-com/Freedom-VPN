# Wave 4 Usage Acceptance Summary (`w4-at-usage` — Follow-up 2)

- **Run ID**: `9d561888`
- **Recorded At (UTC)**: `2026-10-30T21:31:58Z`
- **Source Commit**: `3caa49cb3337bfc8fe69a9d3234e863dc0ed7765` (dirty: `false`)
- **Stack Instance**: `FVPN_E2E_PREFIX=w4-usage` (`fvpn-w4-usage-*`), `FVPN_E2E_PORT_BASE=48000`
- **Primary Exit Node**: `node_id=01a18e52-b663-70aa-858c-12d0cd359d20`, `incarnation_id=01a18e52-b696-70ca-8f8c-c9cb0c48342c`, `exit_id=01a18e52-b663-70ab-aac1-f1979b9a59f4` (`fvpn-exit-3b8c7fda8e7fc29ea530aef8`)
- **Second Exit Node (`AT19`)**: `node_id=01a18eb7-e7d5-7e1d-bf67-60db74591a03`, `incarnation_id=01a18eb7-e911-7610-bd46-94cb598a66e9`, `exit_id=01a18eb7-e7d5-7e1e-bcb3-de9d2a90daee` (`fvpn-exit-cd3550c79db788833c58906b`)
- **Admitted Member Devices**: `memberA` device `01a18e53-b028-7001-8905-4d739aee9610`, `memberB` device `01a18e5f-5ae3-71c9-9ffe-67613fefcdf3`
- **Totals**: `5 pass`, `1 blocked`, `0 fail`
- **Initial Run Archive (#80)**: [history/initial/](history/initial/)

## Acceptance Matrix (AT14–AT19)

| Case | Title | Requirements | Status | Evidence |
|---|---|---|---|---|
| `AT14` | 轉發許可流量計量正確性與排除項 (Forward-Admission Traffic Metering & Exclusions) | `MTR01`, `MTR05`, `API01`, `DATA01` | `pass` | [AT14.md](AT14.md) / [AT14.json](AT14.json) |
| `AT15` | 用量批次冪等、亂序缺口、衝突稽核與簽章檢查點恢復 (Idempotent Usage Ingest, Contiguous ACK, Conflict Audit & Signed Checkpoint Recovery) | `MTR02`, `MTR03`, `MTR04`, `API01`, `DATA01` | `pass` | [AT15.md](AT15.md) / [AT15.json](AT15.json) |
| `AT16` | 節點計量器重啟、計數器重置、Outbox 佇列上限與安全停轉發 (Node Sampler Restart, Counter Reset, Outbox Caps & Safety Fail-Closed) | `MTR01`, `MTR02`, `MTR03`, `MTR04`, `UPD03` | `pass` | [AT16.md](AT16.md) / [AT16.json](AT16.json) |
| `AT17` | UTC 自然月切分、延遲事件歸屬與時鐘回撥保護 (UTC Month Boundary, Late-Arriving Events & Clock Rollback Protection) | `MTR01`, `MTR02`, `MTR04`, `API01`, `DATA01` | `pass` | [AT17.md](AT17.md) / [AT17.json](AT17.json) |
| `AT18` | 多階層速率整形、有效速率計算與宿主保底頻寬 (Multi-Tier Rate Shaping, Effective Rate & Host Reserve) | `FUP01`, `FUP02`, `FUP03` | `pass` | [AT18.md](AT18.md) / [AT18.json](AT18.json) |
| `AT19` | 跨節點累計、Shadow/Enforce 模式切換、月初重置與臨時覆寫 (Cross-Node Fair-Use, Shadow/Enforce, Monthly Reset & Temporary Override) | `FUP01`, `FUP02`, `FUP03`, `FUP04`, `API01` | `blocked` | [AT19.md](AT19.md) / [AT19.json](AT19.json) |

## Upstream Product Blockers

See [requests.md](requests.md) for the two `w3-policy-fup` product defects blocking `AT19`.

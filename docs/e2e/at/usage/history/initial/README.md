# Wave 4 Acceptance Evidence: Usage Metering, Ledger & Fair-Use (AT14–AT19)

This directory contains the machine-readable (`AT14.json`..`AT19.json`) and human-readable (`AT14.md`..`AT19.md`) acceptance evidence for Wave 4 task `w4-at-usage` (§19.1, §19.5).

## Stack Instance

- **Prefix**: `FVPN_E2E_PREFIX=w4-usage` (`fvpn-w4-usage-*`)
- **Port Base**: `FVPN_E2E_PORT_BASE=48000` (control `48443`, agent mTLS `49443`, IdP `48444`, origin `48445`, payment `48446`, NetBird `48480`, STUN `51478`, Postgres `53432`)
- **Source Commit**: `912d976b03b9948ba308cd6e4e97d0f931574510`

## Acceptance Matrix

| Case | Title | Requirements | Status | Evidence |
|---|---|---|---|---|
| `AT14` | 轉發許可流量計量正確性與排除項 (Forward-Admission Traffic Metering & Exclusions) | `MTR01`, `MTR05`, `API01`, `DATA01` | `blocked` | [AT14.md](AT14.md) / [AT14.json](AT14.json) |
| `AT15` | 用量批次冪等、亂序缺口、衝突稽核與簽章檢查點恢復 (Idempotent Usage Ingest, Contiguous ACK, Conflict Audit & Signed Checkpoint Recovery) | `MTR02`, `MTR03`, `MTR04`, `API01`, `DATA01` | `blocked` | [AT15.md](AT15.md) / [AT15.json](AT15.json) |
| `AT16` | 節點計量器重啟、計數器重置、Outbox 佇列上限與安全停轉發 (Node Sampler Restart, Counter Reset, Outbox Caps & Safety Fail-Closed) | `MTR01`, `MTR02`, `MTR03`, `MTR04`, `UPD03` | `blocked` | [AT16.md](AT16.md) / [AT16.json](AT16.json) |
| `AT17` | UTC 自然月切分、延遲事件歸屬與時鐘回撥保護 (UTC Month Boundary, Late-Arriving Events & Clock Rollback Protection) | `MTR01`, `MTR02`, `MTR04`, `API01`, `DATA01` | `blocked` | [AT17.md](AT17.md) / [AT17.json](AT17.json) |
| `AT18` | 多階層速率整形、有效速率計算與宿主保底頻寬 (Multi-Tier Rate Shaping, Effective Rate & Host Reserve) | `FUP01`, `FUP02`, `FUP03` | `blocked` | [AT18.md](AT18.md) / [AT18.json](AT18.json) |
| `AT19` | 跨節點累計、Shadow/Enforce 模式切換、月初重置與臨時覆寫 (Cross-Node Fair-Use, Shadow/Enforce, Monthly Reset & Temporary Override) | `FUP01`, `FUP02`, `FUP03`, `FUP04`, `API01` | `blocked` | [AT19.md](AT19.md) / [AT19.json](AT19.json) |

## Reproduce

```bash
# Unit and mechanism tests (no Docker required; uses FVPN_TEST_DATABASE_URL when set)
make -f mk/at-usage.mk at-usage-test

# Full live stack acceptance run on assigned instance w4-usage (port base 48000)
FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-usage.mk at-usage
FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/e2e.mk e2e-down
```

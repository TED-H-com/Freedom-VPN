# Freedom VPN 交付導覽

[English](DELIVERY.en.md) · **繁體中文** · [專案簡介](../README.zh-TW.md)

Freedom VPN 以自架 NetBird 和官方 App 為基礎，提供會員、出口節點與日常維運管理。
這次收尾保留已建立的會員、計量、限速與更新功能，集中完成整合、必要修正和文件整理。

**目前仍是開發與驗證版本，尚未達到正式營運條件。** 已有實作、已取得的驗證與待完成的
實機／部署條件分列如下；功能存在不代表所有平台及故障場景均已驗收。

## 現有功能與操作入口

| 使用情境 | 現有實作範圍 | 詳細入口 |
|---|---|---|
| 會員使用服務 | 登入、訂閱入口、裝置管理、出口選擇、用量與公平使用狀態 | [會員指南](handover/member-guide.md) |
| 管理會員與節點 | 會員資格、節點註冊與上架、撤權及非同步操作結果 | [營運指南](handover/operator-guide.md) |
| 管理 VPN 流量 | NetBird 連線協調、出口隔離、用量計量、限速與 DNS 政策 | [產品規格](spec/PROJECT_SPEC.zh-TW.md) |
| 維護服務 | 節點安裝、簽章更新、失敗回復、備份與還原 | [部署指南](handover/deploy.md)、[還原演練](handover/restore-drill.md) |
| 管理中央簽章金鑰與發行信任 | 初始化金鑰庫（`fvpn-vault-init`）、檢查與匯出公開信任（`fvpn-vault inspect`、`fvpn-vault export-trust`）、輪替與撤銷中央金鑰（`fvpn-vault-keys rotate`、`fvpn-vault-keys revoke`），以及建立、驗證與套用發行信任更新（`fvpn-release trust-update`、`fvpn-release trust-verify`、`UpdateReleaseTrust`） | [金鑰庫初始化](pki/initialize.md)、[金鑰庫檢查與匯出](pki/inspect.md)、[中央金鑰指令](pki/keys.md)、[金鑰輪替與還原](pki/rotation.md)、[發行流程](updates/release-process.md) |

## 從哪裡開始

1. **了解整套流程：**依 [開發環境啟停程序](e2e/README.md#bring-up) 使用隔離實驗環境。
   測試身分、付款與政策設定僅供驗證，不能直接作為正式環境。
2. **準備部署：**依 [部署指南](handover/deploy.md) 和 [參數校準表](handover/calibration.md)
   確認身分服務、網域、付款、方案及節點設定。
3. **日常操作：**管理員看 [營運指南](handover/operator-guide.md)，會員看
   [會員指南](handover/member-guide.md)；備份與故障回復依 [還原演練](handover/restore-drill.md)。
4. **維護程式：**先讀 [工程規範](../AGENTS.md)、[API 契約](contracts/README.md) 與相關元件文件。

## 交接指南中已過時的說明

[營運指南](handover/operator-guide.md) 與 [會員指南](handover/member-guide.md) 是封存的歷史交付切點，後續不再直接修改。若指南內容與下列文件不同，請以目前的文件與指令為準。

| 位置 | 指南原有說明 | 目前適用作法與文件 |
|---|---|---|
| [營運指南](handover/operator-guide.md) §8.1（第 1096–1104 行） | 呼叫 Go 方法 `vault.Rotate(ctx, purpose, validityDuration, overlapDuration)`、`vault.TrustBundle(ctx)` 與 `vault.Revoke(ctx, keyID)`，且舊金鑰狀態轉為 `retiring` | 使用 `fvpn-vault inspect`、`fvpn-vault export-trust`（[金鑰庫檢查與匯出](pki/inspect.md)）及 `fvpn-vault-keys rotate`、`fvpn-vault-keys revoke`（[中央金鑰指令](pki/keys.md)）；金鑰狀態為 `active`、`retired`、`revoked`，輪替與推展規則見 [金鑰輪替與還原](pki/rotation.md#central-signing-keys) |
| [營運指南](handover/operator-guide.md) §8.2（第 1106–1110 行） | 以新舊發行金鑰同時簽署單一 `fvpn-release-trust-v1` 輪替封套，並以 `vault.UpdateReleaseTrust(ctx, rotationEnvelopeBytes, now)` 套用 | 以 `fvpn-release trust-update`（每次僅指定 `-add`、`-retire` 或 `-revoke` 之一並以單一金鑰簽署）與 `fvpn-release trust-verify` 依連續版本號分步交接，中央以雙參數 `UpdateReleaseTrust(ctx, raw)` 套用；見 [發行流程](updates/release-process.md) 與 [金鑰輪替與還原](pki/rotation.md#release-verification-trust) |
| [營運指南](handover/operator-guide.md) 第 259 行（每日檢查表） | 直接讀取 `/var/lib/fvpn/vault/state.json` 與節點信任檔，檢查接近 `not_after` 的啟用金鑰 | 執行 `fvpn-vault inspect` 檢視每把金鑰的 `state`、`not_before`、`not_after` 與 `window_status`（[金鑰庫檢查與匯出](pki/inspect.md#metadata-inspection)） |
| [會員指南](handover/member-guide.md) §5.3（第 235、465 行） | 在 `shadow` 模式下，入口網站顯示 `would_apply_tier` 且 `throttled` 維持 `false` | 會員用量回應的 `FairUseStatus` 不含這兩個欄位；`shadow` 模式回傳 `state: normal`（級距、速率與原因為 `null`），且入口網站僅在 `state: adjusted` 時顯示調速提示（[API 契約](../contracts/openapi.yaml)、[入口網站文件](portal/README.md)） |

## 驗證範圍與已知限制

[合併紀錄](ROADMAP.md) 追蹤目前實作；[follow-up 49](handover/follow-up-49.md)
保存 #482–#485 的歷史交付切點，後續成果以各項驗收紀錄為準。
單元／整合測試、Linux 實驗環境、真實 App 裝置和正式部署各有不同的驗證範圍。
本導覽不宣告正式發行，也不以文件整理替代尚未完成的驗收。

節點保留的驗證結果 `serving_read` 已完成節點回報、中央儲存與投影、管理介面顯示及
診斷匯出（[#492](https://github.com/TED-H-com/Freedom-VPN/pull/492)、
[#493](https://github.com/TED-H-com/Freedom-VPN/pull/493)、
[#494](https://github.com/TED-H-com/Freedom-VPN/pull/494)）。它保留原始讀取時間與結果，
不能單獨視為目前可轉送流量的許可；原有安全檢查與租約判定仍適用。

| 事項 | 目前界線與驗收入口 |
|---|---|
| 官方 App | [NetBird 版本表](netbird/VERSIONS.md) 列出固定元件及候選 App；Windows、macOS、iOS、Android 的真機驗收仍待執行。 |
| 出口與會員隔離 | [D164 驗收](e2e/at/isolation/d164-acceptance.md) 記錄 2026-11-06 的完整實驗環境與組合驗證：27 項通過、2 項待驗，並關閉 F17-01 缺口。IPv6 平台與完整故障轉換矩陣仍待驗；[歷史隔離紀錄](e2e/at/isolation/summary.md) 保留原有失敗判定。 |
| 會員連線 | [會員驗收](e2e/at/members/README.md) 記錄上游連線與網路條件限制；伺服器端隔離不提供會員裝置的 OS kill switch。 |
| 更新與硬體 | [更新驗收](e2e/at/updates/summary.md) 和 [安全維運驗收](e2e/at/secops/summary.md) 記錄實體 arm64、候選硬體容量與獨立時鐘 guest 等實測缺口；交叉編譯不能替代硬體實測。 |
| 正式營運 | 正式 IdP／MFA、付款供應商、網域／TLS、DNS 清單與營運參數仍需設定及驗證；測試預設不代表正式政策。 |

較早的收尾紀錄中，[隔離驗收修正](e2e/at/isolation/followup27-2026-11-05.md)
保留失敗結果與證據；[網路實驗環境修正](poc/G2-followup37-2026-11-05.md)
提供封包歸屬與阻擋判定的局部本機驗證。後者不改寫原始排程失敗，兩者也不代表
整套系統已完成最終整合驗證。

## 保留的延伸空間

現有模組、API、資料與歷史證據持續保留。後續工作以具體需求或已證實的缺陷為依據，
不因已有設計提案就自動追加實作。

NetBird API 客戶端重用、其他架構調整及進階診斷留待後續評估，
必要的完成性修正另依實際問題處理。

[歷史交付與已知問題](handover/README.md) 保存完整脈絡，
[開發路線與合併紀錄](ROADMAP.md) 保存逐項交付。各驗收入口標明的結果與限制為準。

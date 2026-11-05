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

## 從哪裡開始

1. **了解整套流程：**依 [開發環境啟停程序](e2e/README.md#bring-up) 使用隔離實驗環境。
   測試身分、付款與政策設定僅供驗證，不能直接作為正式環境。
2. **準備部署：**依 [部署指南](handover/deploy.md) 和 [參數校準表](handover/calibration.md)
   確認身分服務、網域、付款、方案及節點設定。
3. **日常操作：**管理員看 [營運指南](handover/operator-guide.md)，會員看
   [會員指南](handover/member-guide.md)；備份與故障回復依 [還原演練](handover/restore-drill.md)。
4. **維護程式：**先讀 [工程規範](../AGENTS.md)、[API 契約](contracts/README.md) 與相關元件文件。

## 驗證範圍與已知限制

[當期工程紀錄](handover/follow-up-49.md) 彙整已合併的修正、驗證來源與保留的問題。
單元／整合測試、Linux 實驗環境、真實 App 裝置和正式部署各有不同的驗證範圍。
本導覽不宣告正式發行，也不以文件整理替代尚未完成的驗收。

| 事項 | 目前界線與驗收入口 |
|---|---|
| 官方 App | [NetBird 版本表](netbird/VERSIONS.md) 列出固定元件及候選 App；Windows、macOS、iOS、Android 的真機驗收仍待執行。 |
| 出口與會員隔離 | [隔離驗收](e2e/at/isolation/summary.md) 保留未完成場景及最新嘗試結果；歷史失敗維持原判定。 |
| 會員連線 | [會員驗收](e2e/at/members/README.md) 記錄上游連線與網路條件限制；伺服器端隔離不提供會員裝置的 OS kill switch。 |
| 更新與硬體 | [更新驗收](e2e/at/updates/summary.md) 和 [安全維運驗收](e2e/at/secops/summary.md) 記錄實體 arm64、候選硬體容量與獨立時鐘 guest 等實測缺口；交叉編譯不能替代硬體實測。 |
| 正式營運 | 正式 IdP／MFA、付款供應商、網域／TLS、DNS 清單與營運參數仍需設定及驗證；測試預設不代表正式政策。 |

本輪另有兩份收尾紀錄：[隔離驗收修正](e2e/at/isolation/followup27-2026-11-05.md)
保留失敗結果與證據；[網路實驗環境修正](poc/G2-followup37-2026-11-05.md)
提供封包歸屬與阻擋判定的局部本機驗證。後者不改寫原始排程失敗，兩者也不代表
整套系統已完成最終整合驗證。

## 保留的延伸空間

現有模組、API、資料與歷史證據持續保留。後續工作以具體需求或已證實的缺陷為依據，
不因已有設計提案就自動追加實作。

可選的 `serving_read` 契約已加入；中央儲存與顯示、診斷讀取及節點資料產生流程尚未
全部啟用。這些介面保留為後續擴充入口，欄位存在不代表功能已上線。NetBird API
客戶端重用、其他架構調整及進階診斷亦留待後續評估，必要的完成性修正另依實際問題處理。

[歷史交付與已知問題](handover/README.md) 保存完整脈絡，
[開發路線與合併紀錄](ROADMAP.md) 保存逐項交付。各驗收入口標明的結果與限制為準。

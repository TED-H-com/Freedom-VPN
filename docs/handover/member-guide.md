# Freedom VPN 會員使用手冊 / Member Guide (`docs/handover/member-guide.md`)

本文件包含繁體中文（預設語言，`zh-TW`）與英文（`en`）雙語完整說明（`AGENTS.md §3`、`§7.3`、`§8.1`–`§8.3`、`§9.2`、`§9.4`、`§10`–`§12`、`§16.3`、`§16.5`、`§18.4`–`§18.5`、`§19.1`）。
This document provides the complete Member Guide in Traditional Chinese (`zh-TW`, primary) followed by English (`en`, secondary).

---

# 第一部分：繁體中文會員手冊 (`zh-TW`)

## 1. 支援平台與候選客戶端版本 (`§3.2`, `§7.3`, `§18.5`, `APP01`, `docs/netbird/VERSIONS.md`)

Freedom VPN v1 設計採用官方 **NetBird 客戶端** 搭配 **Freedom VPN 會員入口網站（Member Portal）**（`https://vpn.example.com/member`）。

> **重要狀態說明（`§7.3`、`§19.1`、`docs/netbird/VERSIONS.md`）**：下表所列之四大原生平台版本目前均為 **候選版本（Candidate Versions — 待實測 / Needs Operator Run）**。Linux 容器環境之測試結果不代表桌面或行動端原生 App 相容性已驗證；正式支援聲明將於完成四平台實機驗收（`AT01`、`AT09`、`AT21`）並附上帶日期之實測紀錄後定案。

| 平台 | 候選客戶端版本 (`docs/netbird/VERSIONS.md`) | 目標作業系統基準 | 實機驗收狀態 (`AT01`, `AT09`, `AT21`) |
|---|---|---|---|
| **Windows** | NetBird daemon/CLI 與隨附桌面 UI `v0.82.0` | Windows 10 (22H2+) / Windows 11 (`amd64` / `arm64`) | **候選版本 — 尚未實測（Needs operator run，`AT01`、`AT09`、`AT21`）** |
| **macOS** | NetBird daemon/CLI 與隨附桌面 UI `v0.82.0` | macOS 14 Sonoma / macOS 15 Sequoia (Apple Silicon & Intel) | **候選版本 — 尚未實測（Needs operator run，`AT01`、`AT09`、`AT21`）** |
| **iOS / iPadOS** | NetBird iOS 客戶端 `v0.4.1`（`build 2`） | iOS / iPadOS 17.0+ | **候選版本 — 上游標示 `build 2` 為 TestFlight 測試版，尚未斷言 App Store 正式上架狀態；尚未實測（Needs operator run，`AT01`、`AT09`、`AT21`）** |
| **Android** | NetBird Android 客戶端 `v0.6.2`（獨立行動端版本號，非 `0.82.0`） | Android 12+ | **候選版本 — 尚未實測（Needs operator run，`AT01`、`AT09`、`AT21`）** |

---

## 2. 四大平台安裝、中央自動授權與連線步驟 (`§7.3`, `§9.2`, `§9.4`, `§18.4`, `§18.5`)

### 2.1 第一步：登入會員入口網站並確認有效方案 (`/member`, `/member/billing`)

1. 開啟瀏覽器前往 `https://vpn.example.com/member`，點擊 **登入（Sign In）** 並透過您的 OIDC 帳號完成驗證（`§9.1`、`§16.1`、`ID01`）。
2. 若尚未啟用訂閱，請至 **帳單與方案（Billing）** 分頁選擇方案並點擊 **前往結帳（Checkout）**（`POST /api/v1/member/checkout-sessions`，回傳 `201 Created` 結帳網址）。待付款供應商 webhook 對帳完成後，入口網站服務資格狀態會顯示為 **`Active`（有效）**（`§9.2`、`§9.3`、`BILL01`、`BILL03`；前端結帳回跳頁本身不直接代表已開通，`AT03`）。

### 2.2 第二步：設定 NetBird 管理網址、登入 OIDC 並由中央自動授權裝置 (`§7.3`, `§9.4`, `D-052`, `docs/members/README.md`)

> **中央自動授權機制與 `D-052` 單一 Issuer 公開 Subject 驗證（`§2 [S36]`、`§7.3`、`§9.4`、`§15.2`、`MEM01`、`MEM02`）**：會員入口網站 `/member/devices` **沒有**手動授權按鈕，API 亦無會員手動新增裝置端點。當您在官方 NetBird 客戶端將 **Management URL（管理伺服器網址）** 指向 `https://netbird.example.com` 並使用與會員入口網站 **相同之 OIDC 簽發者（Issuer）與帳號** 完成 SSO 登入後，中央 `fvpn-worker` 的 `devices.discover` 作業會比對 NetBird 使用者 ID 與中央 `identity_bindings.subject`（寫入 `netbird_user_bindings`，證明格式 `mapping_proof = "shared_issuer_public_subject:" + oidc_issuer`，排除服務帳號與停用帳號，且絕不以 Email 或顯示名稱猜測配對；若為 pairwise 或未知 subject 則以 `identity_unmapped` 失敗關閉並排除於清單外）。驗證通過後，中央在資料庫交易鎖定下檢查您的有效資格與裝置名額上限（`device_limit`），並透過背景作業（`member.device_authorize`）自動配發出口群組與 DNS 權限。您只需在 `/member/devices`（`GET /api/v1/member/devices`）檢視裝置狀態由 `pending`（套用中）自動變為 **`authorized`（已授權）**。

會員身分由相同的標準化 Issuer 與 Subject 決定（`internal/members/issuer/issuer.go` 的 `issuer.Canonical` 僅移除未編碼路徑尾端的字面值 `/`，保留 `%2F` / `%2f`，`docs/members/README.md`）；管理員更換入口網站的 OIDC Client ID 時會沿用原會員，不會建立新會員。若身分比對遭拒，請聯絡支援人員，勿改用相同 Email 的另一帳號嘗試合併（[身分規則](../control/AUTH.md)）。

#### A. Windows（候選版本 `v0.82.0`，待實測 `AT01`）
1. 安裝官方候選版 `v0.82.0` Windows 安裝程式。
2. 點擊系統列（System Tray）的 NetBird 圖示 -> **Settings（設定）** -> **Advanced（進階）**。
3. 在 **Management URL** 輸入 `https://netbird.example.com` 並儲存（或於 PowerShell 執行 `netbird up --management-url https://netbird.example.com`）。
4. 系統會開啟瀏覽器，請使用您的 Freedom VPN OIDC 帳號完成 SSO 登入。
5. 開啟會員入口網站 `https://vpn.example.com/member/devices` 檢視裝置列表，確認剛登入的 Windows 裝置已自動出現，且狀態由 `pending`（套用中）轉為 **`authorized`（已授權）**（若名額已滿，請先於該頁面撤銷不再使用的舊裝置，`AT02`）。
6. 在系統列 NetBird 圖示點擊 **Connect**，並於 **Exit Node / Networks** 選單手動勾選欲使用的出口節點（例如 `tw-tpe-01`，或於 CLI 執行 `netbird networks select <fvpn-exit-…>`；所有獨立出口預設關閉自動套用 `skip_auto_apply = true`，**裝置僅在手動選取出口後才會透過該出口轉發流量**——在 NetBird `0.82.0` 上 `netbird networks select all` 不會自動選取出口，`§7.2`、`§18.4`、`EXIT02`、[`docs/e2e/RUN.md`](../e2e/RUN.md) § "Native exit selection and traffic trace"）。

#### B. macOS（候選版本 `v0.82.0`，待實測 `AT01`）
1. 安裝官方候選版 `v0.82.0` macOS 套件並啟動應用程式（首次啟動時請允許 macOS 新增 VPN 設定檔與系統網路延伸模組）。
2. 點擊頂部選單列（Menu Bar）的 NetBird 圖示 -> **Preferences / Settings** -> **Advanced**。
3. 將 **Management URL** 設為 `https://netbird.example.com` 並儲存，接著點擊 **Connect** 完成瀏覽器 OIDC 登入。
4. 前往會員入口網站 `https://vpn.example.com/member/devices` 確認該 macOS 裝置已被中央自動納管且狀態顯示為 **`authorized`（已授權）**。
5. 於選單列 NetBird 圖示手動選擇出口節點（Exit Node，或於終端機執行 `netbird networks select <fvpn-exit-…>`；裝置僅在選取出口後才會透過出口轉發流量）。

#### C. iOS / iPadOS（候選版本 `v0.4.1 build 2` TestFlight，待實測 `AT01`）
1. 安裝候選版 **NetBird** iOS App（`v0.4.1, build 2`；注意：上游將 `build 2` 標示為 **TestFlight** 版本，尚未斷言 App Store 正式上架狀態，正式發布前須完成 `AT01` 實機確認，`docs/netbird/VERSIONS.md`）。
2. 開啟 App，前往 **Settings -> Change Server / Server**，輸入 `https://netbird.example.com`（不輸入 setup key）並儲存。
3. 點擊連線按鈕，於系統彈窗點擊 **Allow（允許新增 VPN 設定）**，並在彈出的瀏覽器視窗完成 OIDC SSO 登入。
4. 開啟 `https://vpn.example.com/member/devices` 確認您的 iPhone / iPad 已自動列於清單且狀態轉為 **`authorized`（已授權）**。
5. 回到 NetBird App 的 **Connection -> Exit node** 手動選擇已授權的出口節點（裝置僅在選取出口後才會透過出口轉發流量）。

#### D. Android（候選版本 `v0.6.2`，待實測 `AT01`、`AT21`）
1. 安裝候選版 **NetBird** Android App（`v0.6.2`）。
2. 開啟 App，前往 **Settings -> Profiles -> Add**，將 **Management Server** 設為自架網址 `https://netbird.example.com`（不輸入 setup key）並儲存。
3. 點擊連線按鈕，核准 Android VPN 連線權限並於瀏覽器完成 OIDC SSO 登入。
4. 前往 `https://vpn.example.com/member/devices` 確認您的 Android 裝置已由中央自動授權且狀態顯示為 **`authorized`（已授權）**，再於 App 內手動選擇出口節點（裝置僅在選取出口後才會透過出口轉發流量）。
5. **重要 DNS 設定（`§12.6`、`AT21`）**：Android 的系統「私人 DNS（Private DNS）」預設或開啟時會繞過 VPN 指派的 DNS 解析器。使用本服務之網域防護時，請至 Android **設定 -> 網路和網際網路 -> 私人 DNS（Private DNS）**（實際路徑依手機廠牌與系統版本而異），將其設為 **「關閉（Off）」**（詳見第 6 節說明）。

---

## 3. 手動驗證腳本 (`AT01`, `AT09`, `AT21`, `AT29`) (`§7.3`, `§8.1`–`§8.3`, `§12.1`, `§12.6`, `§16.3`, `§16.5`, `§19`)

> **量測狀態說明（`§19.1`、`docs/e2e/README.md`）**：以下腳本為受控驗收環境與實機測試的操作程序（`docs/e2e/at/members/OPERATORS.md`）。目前四平台客戶端行為（`AT01`、`AT09`、`AT21`）與線上/離線轉發切斷時效（`AT04`、`AT13`、`AT29`）均 **尚未實測（not yet measured — needs operator run）**。

### 3.1 `AT01`、`AT10`、`AT12`：驗證受控出口來源 IP 與跨會員隔離 (`§7.3`, `§8.1`, `§8.2`, `ISO01`, `ISO02`)

1. **驗證會員工作階段與有效資格（`GET /api/v1/member/me`）**：
   - `/api/v1/member/me` 回傳的是您的會員帳號與服務資格狀態（`status`、`effective_service_end`、`plan`），**並非**用來反映客戶端的出口公網 IP，且呼叫時必須附帶已認證的瀏覽器工作階段 Cookie（請透過權限 `0600` 的 curl 設定檔傳遞，避免將工作階段秘密留在指令歷史中，`§16.1`、`§16.2`）：
   ```bash
   # 建立 0600 設定檔（測試完成後刪除），內含：header = "Cookie: __Host-fvpn-member=<SESSION_TOKEN>"
   curl -fsS --config ~/.config/fvpn/session.curl https://vpn.example.com/api/v1/member/me
   ```
2. **以受控測試端點驗證出口來源 IP 切換（`AT01`、`AT09`）**：
   - 依據安全與衛生規範（`AGENTS.md §7`、`§19.1`），驗證出口 IP 時必須使用團隊受控的測試/實驗室回應端點（例如 `https://echo.test.example.invalid/ip`，回傳伺服器觀測到的連線來源 IP），不得對未授權第三方網站發起測試。
   - **步驟**：先在**未開啟隧道（或未勾選出口）**時查詢受控端點並記錄原始來源 IP；接著在 NetBird 客戶端**手動勾選目標出口節點**後再次查詢，確認觀測到的來源位址已切換為該出口節點的對外出口位址（例如文件示例位址 `192.0.2.20`）：
   ```bash
   # 連線前：顯示設備原本網路的來源位址
   curl -fsS https://echo.test.example.invalid/ip

   # 選取出口節點連線後：必須顯示所選出口節點的對外位址（如文件示例 192.0.2.20）
   curl -fsS https://echo.test.example.invalid/ip
   ```
3. **具備正向連通對照的跨會員隔離測試（`ISO01`、`AT12`）**：
   - 單純 `ping` 失敗若發生在隧道中斷時，不能證明隔離規則有效。因此測試跨會員阻擋時，必須**先執行正向連通控制組（Positive Reachability Control）**確認隧道轉發與授權 DNS 正常運作，且另一位測試會員裝置（文件示例 Overlay IP `100.64.0.11`）同時在線，再測試連線至 `100.64.0.11` 是否被拒絕：
   - **macOS / Linux 終端機指令（候選環境驗收腳本 — 尚未實測 / Needs operator run，`AT01`、`AT12`）**：
     ```bash
     # 1. 正向連通控制組：確認透過隧道可正常連線至受控測試端點與授權 DNS (53/tcp, 53/udp)
     curl -fsS https://echo.test.example.invalid/healthz
     nslookup example.com 100.64.0.53

     # 2. 跨會員隔離驗證：嘗試連線另一位在線會員裝置的 Overlay IP (示例 100.64.0.11) -> 必須 100% 丟棄
     ping -c 3 -W 2 100.64.0.11
     # 預期結果：100.0% packet loss（由 NetBird 存取政策與出口節點 inet fvpn 防火牆阻擋）
     ```
   - **Windows PowerShell 指令（候選環境驗收腳本 — 尚未實測 / Needs operator run，`AT01`、`AT12`）**：
     ```powershell
     # 1. 正向連通控制組：確認受控測試端點可達
     Invoke-RestMethod -Uri https://echo.test.example.invalid/healthz

     # 2. 跨會員隔離驗證：嘗試連線另一台在線會員裝置 Overlay IP (示例 100.64.0.11) -> 必須為 False
     Test-NetConnection -ComputerName 100.64.0.11 -Port 443
     ```

### 3.2 `AT09`、`AT29`：伺服器端 Fail-Closed 保護、終端無 Kill-Switch 限制與平台網路切換行為 (`§7.3`, `§8.2`, `§16.3`, `§16.5`, `ISO02`, `ENT02`)

1. **重要限制揭露：無終端完整 Kill-Switch 保證（`§7.3`、`§17.1`）**：
   - Freedom VPN v1 使用未客製之官方 NetBird 客戶端，**不提供完整的作業系統層級「永遠開啟 / 強制斷線保護（Always-On / Kill Switch）」模式**（與會員入口網站 `member.gettingStarted.killSwitchWarningBody` 警告一致）。
   - **伺服器端 Fail-Closed 與終端路由的區別**：出口節點的 `inet fvpn` 防火牆（Fail-Closed）僅保證**當封包進入 VPN 隧道到達出口節點時**，若節點隔離基線、租約或會員資格失效，**出口節點不會代為轉發流量**。但若您的手機或電腦因隧道中斷、應用程式關閉、睡眠喚醒或切換 Wi-Fi / 行動網路而失去 VPN 路由，客戶端作業系統**可能會改走原本的本地 ISP 網路直接連線**（`§7.3`）。
2. **三種伺服器端停止轉發機制與時效說明（`§16.3`、`§16.5`、`AT04`、`AT13`、`AT29`）**：
   - **節點本機 Watchdog 心跳逾時（`§16.3`）**：出口節點上的獨立守護程序 `fvpn-watchdog` 會檢查本機 `fvpn-agent` 的心跳檔案（預設設定 `watchdog_stale_seconds = 30` 秒）。若本機 `fvpn-agent` 卡死或異常退出，`fvpn-watchdog` 會將該節點的 `inet fvpn` 切換回 baseline drop 狀態，停止該節點的會員轉發。
   - **控制面在線時的主動撤權 / 停用（`§16.5`、`AT04`、`AT13`）**：當中央控制服務與出口節點均在線時，管理員停用出口、會員停權或撤銷裝置的**線上政策收斂設計目標以 60 秒為起點**（`§16.5` 非功能目標，並非瞬間生效，亦非離線保證）。
   - **控制面失聯時的離線到期界線（`§16.3`、`AT29`）**：當出口節點與中央控制面失聯時，無法即時收到新發生的撤權通知；節點核心透過 `inet fvpn` 的 `@allowed_v4` timeout 集合強制執行 `min(effective_service_end, node_lease_end, signed_server_time + max_offline_seconds)`（預設最長離線租約上限為 `86,400` 秒 / 24 小時），最早期限到達時由核心自動停止轉發，且不受系統牆鐘回調（clock rollback）延長。
   - **實測狀態（`DOC15`、`DOC16`、`#165`、`#170`、`#196`、`#201`、`#203`、`#231`）**：在目前 `main` 分支上，`deploy/devstack/Dockerfile.node` 已安裝 `conntrack` 與 `tcpdump`、`SystemPlane.MissingExecutables()`（`node/agent/plane.go`）已藉由 `enforce.RequireConntrack` 預檢 `conntrack`、`node/agent/sync.go` 會在套用新期望狀態世代的同一步驟立即更新服務租約（`#146`、`#153`、`#162`、`#166`），且於節點冷重啟收到無授權成員之暫時租約（`members: []`）時以 `D-067` 指數退避（1 次心跳後重試、倍增至上限 `60` 秒，狀態回報 `awaiting_member_grants`）加速恢復轉發（`#193`、`#198`、`#200`）。先前於 [會員驗收 follow-up 3（`#151`）](../e2e/at/members/README.md)（[`docs/e2e/at/members/requests.md`](../e2e/at/members/requests.md)）及早期 [devstack follow-up 9（`#153`）](../e2e/followup9-verification.md) 紀錄中因缺少 `conntrack` 或世代切換租約更新而標記為 `blocked` 的 `AT02`、`AT03`、`AT04`、`AT08` 項目均屬當時歷史快照，不再代表目前主線狀態（[會員驗收 follow-up 5（`#201`）与 follow-up 6（`#231`，執行紀錄 `80db103b-622c-42b1-946d-4628daaa213d`）](../e2e/at/members/summary.md) 已通過 `AT02`（`42/0` 子檢查，包含透過 `Stack.ObserveMemberIngress` 於 `fvmb0` 實測嚴格撤銷後入場阻擋：已撤銷裝置觸發 `ctunku` `0 -> 1` 且 `0` 轉發，同會員已授權旁支裝置則使 `u_` 由 `31 -> 32`）、`AT03`、`AT08`，並於 `AT09` 不變世代 `23` 冷重啟實測 `D-067` 暫時窗口 `36.976` 秒恢復 `ready`、以及 `41.695` 秒產生首筆計費位元組，其餘 `7` 個案例因實機/實驗室資源維持 `blocked`，全套 `3 pass / 7 blocked / 0 fail`、`384` 項子檢查通過）。歷史雙裝置基準證據固定封存於 [資安與營運驗收 follow-up 5 歷史封存（`#165`）](../e2e/at/secops/history/follow-up-5/README.md) 與 [devstack follow-up 10（`#170`）](../e2e/RUN.md)：(1) [`history/follow-up-5/counted-flow.json`](../e2e/at/secops/history/follow-up-5/counted-flow.json)（結案 `SECOPS-09`）驗證選取出口（`netbird networks select`）後之雙裝置計費轉發（`forward_admission_v1` `0 -> 399` 上傳 / `0 -> 678` 下載位元組、`usage_events = 2`、`usage_rollups = 4`，且無 `probe_up`/`probe_down` 診斷流量混入）；(2) [`history/follow-up-5/post-revoke.json`](../e2e/at/secops/history/follow-up-5/post-revoke.json)（通過 `F3-02` 一致性）、[`device-revoke.md`](../e2e/runs/2026-10-30/followup10-run3/device-revoke.md)（[`docs/e2e/requests.md`](../e2e/requests.md)）及 [devstack follow-up 12（`#196`、`#199`）](../e2e/scenarios/device-revoke.md) 驗證真實 `DELETE /api/v1/member/devices/{device_id}` 撤銷、NetBird peer 移除、`@allowed_v4` 成員位址清除、受控凍結客戶端三點 `tcpdump` 驗證（`peer_tx_packets > 0`、`mor0_rx_packets == 0`、`uplink_packets == 0`）與第 4 台裝置替換授權（撤銷前 `409 device_limit_reached`、撤銷後 `202 succeeded`）；(3) [`history/follow-up-5/deadlines.json`](../e2e/at/secops/history/follow-up-5/deadlines.json) 與 [2026-10-30 租約到期紀錄](../e2e/runs/2026-10-30/lease-expiry.json) 記錄雙裝置基準下之 `fvpn-watchdog` 心跳逾時接管（35 秒陳舊心跳觸發 `fv-watchdog-enforce`）、期望狀態到期（`policy_missing`）、租約到期（`lease_expired`）與元件層時鐘檢查（component clock tests，與獨立客體 VM 系統牆鐘回調 `clock_settime` 嚴格區分）；(4) [`history/follow-up-5/capacity.json`](../e2e/at/secops/history/follow-up-5/capacity.json) 記錄雙裝置計費轉發子集，且 `Stack.StartSecondExit` / `EnrollAt(..., "node2")`（[`docs/e2e/RUN.md`](../e2e/RUN.md) `second-exit`）提供雙真實出口節點實驗室掛鉤。目前主線最新量測數據載於 [資安與營運驗收 follow-up 7（`#203`，執行紀錄 `ab96acac-8713-485a-9c29-88f20110e2e5`）](../e2e/at/secops/summary.md)：於 [`deadlines.json`](../e2e/at/secops/deadlines.json) 透過 `D-065` 艦隊 API（`GET`/`PUT /api/v1/admin/fleet-config` 驗證 `428`/`412`/冪等重播/`fleet_config.update` 稽核，將 `max_offline_lease_seconds` 設為 `60` 秒並於測試後恢復為 `86400`）與有限寬限期準備（`FVPN_E2E_GRACE_SECONDS=90`）實測付費/授權/租約到期切斷（`495/869` -> `0/0` 位元組）與恢復（`0 -> 1320/2316` 位元組）；並於 [`capacity.json`](../e2e/at/secops/capacity.json) 與 [`node-telemetry.json`](../e2e/at/secops/node-telemetry.json) 實測 60 台原生 peer（`20` 位會員 × `3` 台裝置，`ScaleNativeMembers` `296.017` 秒）全數 `60/60` 產生計費流量（`node1` `39,600`/`69,480` 位元組、`node2` `7,920`/`13,896` 位元組）、`5,960/5,960` 次 `GET /api/v1/member/devices` 併發讀取全數回傳 `200`（`0` 筆跨會員誤配，`p50/p95/p99/max` = `25.81/96.32/149.94/302.18 ms`）及 `82` 筆 `5` 秒週期節點遙測樣本（`0` 錯誤）。**保留之實測限制與待辦資格條件（`AT28`–`AT32` 均維持「子集通過、全案 `blocked`」狀態）**：`AT29` 之 `past_due` 付款真值對齊（`SECOPS-17`）、既有連線中 TCP/UDP 雙向傳輸切斷、以及獨立可拋棄式客體 VM 系統牆鐘回調（`clock_settime`，維持 `needs operator run` 且不以元件層時鐘測試替代），連同 `AT31` 強制中繼（forced relay）與候選硬體認證、以及四大平台原生實機（`AT01`、`AT09`、`AT21`）及 24 小時完整 `AT29` 戰役，仍為 `blocked` / `needs operator run` 待實測項目。
3. **四平台斷線、睡眠喚醒與換網行為記錄要求（`§7.3`、`§18.5`、`APP01`、`AT01`、`AT09`）**：
   - 各平台客戶端在睡眠/鎖定、切換網路及所選出口被停用時的實際路由行為，須依 `docs/e2e/at/members/OPERATORS.md` 逐平台實測並記錄：

| 平台 | 測試動作 (`AT01` / `AT09`) | 觀測重點（是否恢復隧道、是否退回本地 ISP、所需秒數） | 目前狀態 |
|---|---|---|---|
| **Windows (`v0.82.0` 候選版)** | 睡眠 2 分鐘後喚醒；於兩個受控 Wi-Fi 網路間切換；連線中停止所選出口節點 | 記錄喚醒與換網後之出口選擇、受控 echo 來源 IP、DNS 解析路徑，以及出口停止時是否發生本地網路直連 fallback | **尚未實測（Not yet measured — 待執行 `AT01`、`AT09`）** |
| **macOS (`v0.82.0` 候選版)** | 睡眠 2 分鐘後喚醒；於兩個受控 Wi-Fi 網路間切換；連線中停止所選出口節點 | 記錄喚醒與換網後之出口選擇、受控 echo 來源 IP、DNS 解析路徑，以及出口停止時是否發生本地網路直連 fallback | **尚未實測（Not yet measured — 待執行 `AT01`、`AT09`）** |
| **iOS / iPadOS (`v0.4.1 build 2` 候選版)** | 鎖定螢幕 2 分鐘後喚醒（記錄為行動端背景暫停）；切換受控 Wi-Fi 網路；連線中停止所選出口節點 | 記錄喚醒與換網後之隧道狀態、受控 echo 來源 IP、DNS 解析路徑及出口中斷時之行為 | **尚未實測（Not yet measured — 待執行 `AT01`、`AT09`）** |
| **Android (`v0.6.2` 候選版)** | 鎖定/省電 2 分鐘後喚醒（一併記錄電池最佳化設定）；切換受控 Wi-Fi 網路；連線中停止所選出口節點 | 記錄喚醒與換網後之隧道狀態、受控 echo 來源 IP、DNS 解析路徑及出口中斷時之行為 | **尚未實測（Not yet measured — 待執行 `AT01`、`AT09`）** |

### 3.3 `AT20`、`AT21`：驗證 DNS 惡意網域攔截（`NXDOMAIN`）與白名單放行 (`§12.1`, `§12.4`, `§12.6`, `§12.8`, `DNS01`–`DNS04`)

- **macOS / Linux（`dig` 或 `nslookup`，候選環境驗收腳本 — 尚未實測 / Needs operator run，`AT20`、`AT21`）**：
  ```bash
  # 1. 查詢受封鎖之測試惡意網域（以文件示例解析器位址 100.64.0.53 為例）-> 預期回傳 NXDOMAIN
  nslookup malware.test.example.invalid 100.64.0.53

  # 2. 查詢正常網域與白名單覆寫網域 -> 預期回傳 NOERROR 與正常位址
  nslookup example.com 100.64.0.53
  ```
- **Windows PowerShell（候選環境驗收腳本 — 尚未實測 / Needs operator run，`AT20`、`AT21`）**：
  ```powershell
  Resolve-DnsName malware.test.example.invalid -Server 100.64.0.53
  # 預期結果：DNS name does not exist (RCODE=3 NXDOMAIN)
  Resolve-DnsName example.com -Server 100.64.0.53
  ```
- **四大平台系統解析器與瀏覽器路徑（`§12.6`、`AT21`）**：
  除指定解析器位址查詢外，各平台實機驗收（`AT21`，**尚未實測 / Needs operator run**）須在關閉 Android 私人 DNS 與關閉瀏覽器自訂安全 DNS 的情況下，直接以一般瀏覽器驗證系統預設 DNS 路徑確實經由受控解析器阻擋測試惡意網域。

---

## 4. 裝置數量上限與自助撤銷 (`/member/devices`, `§7.3`, `§9.4`, `MEM01`–`MEM03`, `ENT02`, `AT02`, `AT04`)

1. **預設裝置上限（`§9.4`、`§21`、`MEM02`）**：每位會員建議預設最多可同時擁有 **3 台已授權裝置**（`device_limit = 3`，實際上限依您的訂閱方案或管理員設定顯示於 `/member/devices` 的已用/上限配額）。
2. **達到上限時的行為（`409 DEVICE_LIMIT_REACHED`，`AT02`）**：當您在新裝置登入 NetBird 而已授權裝置數已達上限時，中央自動授權交易會在資料庫列鎖定下拒絕授予新裝置出口權限（紀錄 `DEVICE_LIMIT_REACHED`），該新裝置將停留在未授權狀態而無法使用 VPN 出口轉發（`§7.3`、`§9.4`、`AT02`）。
3. **自助撤銷舊裝置與生效時效（`§9.4`、`§16.3`、`§16.5`、`AT02`、`AT04`、`AT29`）**：
   - 前往 `https://vpn.example.com/member/devices`，找到不再使用的舊裝置，點擊 **撤銷（Revoke）**（呼叫 `DELETE /api/v1/member/devices/{device_id}`，建立非同步 `member.device_revoke` 操作）。
   - **撤銷生效時間說明**：點擊撤銷**並非瞬間在全球所有節點同步斷線**。在中央與出口節點皆在線的正常路徑下，系統會非同步撤除 NetBird 權限並下發新政策版本以終止既有與新連線（規格在線撤權目標以 **60 秒**為起點，`§16.5`；實際端到端切斷時間 **尚未實測 / not yet measured**，待執行 `AT02`、`AT04`、`AT29`）。若某台出口節點當時與控制面失聯，該節點上的轉發最長受既有核心租約/服務截止期限（`min(effective_service_end, node_lease_end, signed_server_time + max_offline_seconds)`，預設最長 `86,400` 秒，`§16.3`、`AT29`）約束。
   - 待 `/member/devices` 頁面上的撤銷操作狀態完成並釋出配額後，即可讓新裝置取得授權。

方案降低裝置名額時，中央保留最早授權的裝置，超額裝置轉為 `pending` 並撤除權限；資格恢復後，只會在現有名額內重新授權先前已授權且未撤銷、身分證明有效的裝置。在服務備份還原後，若某台裝置的授權狀態尚待營運人員確認，該裝置亦可能暫時顯示為 `pending`，直到營運人員完成確認為止，會員端的任何操作均不會改變此狀態。`D-077`（`#238`、`#244`；2026-10-30 EDT）新增 `authorization_unconfirmed` 標記，表示「需營運人員重新授權」。合約中此欄位為選填；省略表示伺服器尚未投影此資訊，不代表已確認。當日 `main` 的 `GET /api/v1/member/devices` 每筆都回傳此布林值：受還原圍欄保護者為 `pending` + `authorization_unconfirmed: true`，其他裝置為 `false`。會員無法自行重新授權；可聯絡營運人員或撤銷裝置（`D-076`、`#227`）。請在 `/member/devices` 確認目前狀態；詳見[會員與裝置規則](../members/README.md#quota-withdrawals-and-finite-binding-history)。

---

## 5. 用量統計與公平使用政策（FUP）顯示說明 (`/member/usage`, `§8.3`, `§10`, `§11`, `MTR01`–`MTR05`, `FUP01`–`FUP04`, `AT14`, `AT17`–`AT19`)

在會員入口網站 **用量與速度（Usage）** 分頁（`/member/usage`）中，您可以查看當月與每日的上傳、下載及總流量：

### 5.1 計量口徑（`forward_admission_v1`）與 v1 IPv6 封鎖模式 (`§8.3`, `§10.1`, `MTR01`, `AT11`, `AT14`)
- **正式計量位置（`§10.1`）**：`forward_admission_v1` 僅計算解開隧道後通過出口節點授權與隔離檢查、獲准進入轉發路徑（`inet fvpn` forward chain）之內層封包位元組（包含內層 IP 與傳輸層標頭，上行與下行各計一次）。在計量點之前被防火牆拒絕的封包、外層 WireGuard 封裝或節點控制通道流量，均不會計入會員用量；而在通過 admission hook 之後才因佇列（qdisc）或下游網路丟棄的封包則可能已計入，因此此數值為「已記錄的 VPN 轉送入場用量」，不等同應用程式最終下載完成的檔案大小（`§10.1`）。
- **v1 IPv6 阻擋模式說明（`§8.3`、`node/enforce`）**：雖然跨節點帳本與身分綁定之資料結構（schema）預留了 IPv4 與 IPv6 欄位，但 **Freedom VPN v1 的節點資料平面固定執行 `IPv6Block`（`block`）模式**——出口節點 `inet fvpn` 會無條件丟棄所有會員 IPv6 轉發流量（`meta nfproto ipv6 ... drop`），僅提供會員 **IPv4** 轉發服務。因此在 v1 中，您的實際轉發與計量位元組皆來自獲准通過的內層 IPv4 流量（`AT11`、`AT14`）。

### 5.2 結算週期、跨月歸屬與重置時間 (`§10.4`, `§10.9`, `§11.1`, `MTR04`, `FUP01`, `AT17`, `AT19`)
- 以 **UTC 自然月**（每月 1 日 `00:00:00Z` 至下月 1 日 `00:00:00Z` 半開區間）為統計與公平使用級距重置週期（`§10.4`、`§11.1`）。
- 跨月邊界與遲到事件依觀測區間歸屬對應月份，不會因上月遲到事件而延長本月調速（`§10.9`、`AT17`）；進入新 UTC 月份時，當期累計用量與調速級距自動從 `Tier 0` 重新計算（`§11.2`、`AT19`）。

### 5.3 公平使用（FUP）調速階層、共享容量限制與強制 Fail-Closed 例外 (`§11.1`–`§11.7`, `FUP01`–`FUP04`, `AT18`, `AT19`)
- **調速階層與設定底線（`minimum_rate`，`§11.1`、`§11.2`、`AT19`）**：當月跨全部裝置與出口節點累積總流量超過方案門檻時，系統在 `enforce` 模式下會依階層調降您在各出口節點的 HTB 速率上限（`Tier 1`、`Tier 2`...）。政策編譯器要求每個階層的設定上限皆不低於正數設定底線 **`minimum_rate`**（`down_bps > 0`、`up_bps > 0`），代表**公平使用調速本身不會因流量超額而將您的設定速率設為 `0 bps`**。
- **非保證實得頻寬、且受節點共享容量約束（`§11.3`、`§11.6`、`§11.7`、`FUP02`、`FUP04`、`AT18`）**：
  - `minimum_rate` 與各級距之 `down_bps` / `up_bps` 僅為出口節點 `tc` HTB 排程器的**設定速率上限/底線（Configured Shaping Floor / Ceiling）**，**並非**對您終端實得下載或上傳速度的保證。
  - **同出口多裝置共享（`§11.3`、`AT18`）**：同一位會員名下的多台已授權裝置若同時使用**同一台出口節點**，會共同分享該會員在該出口節點的單一 HTB 速率類別（不會每台裝置各拿一份滿額速率）。
  - **節點總容量共享（`§11.3`、`FUP04`）**：所有連線至同一台出口節點的會員共同受該節點父類別總容量上限（`node_capacity`）以及上游 ISP / Relay 線路條件限制；當節點總容量壅塞時，實際可得吞吐量可能低於個別級距上限。
- **強制 Fail-Closed 停止轉發之例外情形（`§8.2`、`§10.7`、`§10.8`、`§13.6`、`§16.3`、`ISO02`）**：
  - `minimum_rate` 絕不覆蓋安全、資格與計量完整性規則。當發生以下任一情況時，出口節點**必然會立即或於期限到達時停止您的流量轉發（Fail-Closed）**：
    1. 您的訂閱/人工授權到期、帳號被停權、裝置被撤銷，或節點離線服務租約（`forwarding_deadline`）到期（`§9.2`、`§16.3`、`AT04`、`AT29`）；
    2. 出口節點之 Internet-only 隔離檢查、DNS 基線檢查或 `fvpn-watchdog` 心跳檢查失敗（`§8.2`、`§16.3`、`AT10`、`AT13`）；
    3. 出口節點本機計量路徑異常（`METER_PATH_INVALID`）、本機 SQLite 無法寫入，或未獲中央 ACK 之 usage outbox 達到容量/時間保護上限（建議預設 `72 小時` 或 `256 MB`，`§10.3`、`§10.7`、`§10.8`、`AT16`）；
    4. 出口節點正處於程式更新的切換與健康檢查窗口（`applying` / `checking`，`§13.2`、`§13.6`）。
- **觀測模式（`shadow` mode，`§11.1`、`§11.2`、`AT19`）**：若系統處於 `shadow` 模式，頁面會顯示預估級距（`would_apply_tier`），但節點資料面仍維持正常未降速設定（`throttled: false`）。

### 5.4 資料品質標記說明 (`§10.6`–`§10.9`, `§18.4`, `AT16`, `AT17`, `AT30`)
- **`統計截至 (Aggregate as of)`**：中央帳本已完成投影彙總的資料時間點（`§10.6`）。
- **`對帳中 (Reconciliation in progress)`**：中央資料庫於備份還原後正在執行節點事件重播與對帳，數據將於對帳完成後定案（`§10.8`、`§17.3`、`AT30`）。
- **`含缺口 (Gap)` / `邊界估算 (Boundary Estimated)`**：`gap` 代表部分區間因節點異常斷電、重啟或超過保存窗口而存在無法重建之未取樣缺口；`boundary_estimated` 代表跨 UTC 月界區間依重疊時間比例整數分攤。系統**絕不會憑空捏造或虛構未知的流量位元組**（`§10.7`–`§10.9`）。

---

## 6. DNS 安全防護行為、資料最小化與技術限制 (`§8.1`, `§12.1`–`§12.7`, `DNS01`–`DNS04`, `SEC02`, `AT12`, `AT20`–`AT22`)

會員入口網站 `/member/dns-protection` 說明 DNS 防護範圍、限制與誤判回報方式；`/member/exits` 顯示可選出口及可用狀態。節點回報的型別化 `dns_policy_evidence` 位於管理員的節點詳細頁，會員頁面尚未顯示；需要確認特定出口的套用結果時，請支援人員查看該證據。入口網站行為以 [portal 文件](../portal/README.md) 為準。

### 6.1 受控解析器配置、防火牆規則與資料最小化 (`§8.1`, `§12.2`, `§12.6`, `§12.7`, `DNS03`, `SEC02`, `AT12`, `AT21`, `AT22`)
- **受控 DNS 解析器與 `inet fvpn` 規則（`node/enforce/render_nft.go`）**：
  - NetBird 會將本服務配置的受控 AdGuard Home 解析器位址（例如文件示例位址 `100.64.0.53` 與 `100.64.0.54`；實際位址依部署環境設定而定，`AGENTS.md §8`）下發給已授權的會員裝置（`§12.2`）。
  - 在出口節點資料平面（`inet fvpn` forward chain）中，規則僅精確放行已授權會員位址（`@allowed_v4`）連往受控解析器集合（`@resolver_v4`）的 **`UDP/53` 與 `TCP/53`** 查詢（計入獨立的 `dns_service_v1` 計數器，不計入會員上網流量），並**強制丟棄**連往解析器位址的所有非 53 埠流量（如管理埠 `8080` 或 SSH，`§8.1`、`DNS03`、`AT12`）。
  - **無全域 Port-53 DNAT 強制劫持**：出口節點 `inet fvpn` **並未**實作將任意外部 `UDP/53` 或 `TCP/53` 流量透過 DNAT 透明改寫至內部解析器的機制。會員裝置若手動指定外部 Port-53 DNS，其封包是否被阻擋或依政策處理，取決於環境所選定的外部 DNS 政策（`§12.6` 標記為 `待決策`，各平台實際行為 **尚未實測 / not yet measured**，列入 `AT21`）。
  - 受控解析器針對命中的惡意軟體、釣魚或管理員封鎖網域回傳 `NXDOMAIN`；當單台受控解析器故障時，僅能切換至另一台套用相同政策的受控解析器，**絕不會**靜默退回未過濾的公共 DNS（`§12.2`、`§12.5`、`AT21`）。
- **資料最小化與無完整查詢日誌（`SEC02`、`§12.7`、`§16.4`、`AT22`）**：
  - 受控 AdGuard Home 解析器在組態上關閉查詢日誌與統計功能（`querylog.enabled: false`、`querylog.file_enabled: false`、`statistics.enabled: false`），且一般服務日誌、指標（metrics）與中央資料庫**不記錄**您查詢過的完整網域名稱（qname）、造訪的完整網址（URL）或封包內容（`§12.7`）。
  - **誠實揭露（`§16.4`）**：本服務**不宣稱完全無日誌（not a complete no-log service）**——系統仍會保存必要的會員資格、裝置授權狀態、匿名化流量計量（`usage_events` / `usage_rollups`）與管理/安全稽核紀錄（`audit_events`）。

### 6.2 您必須了解的技術限制：DoT / DoQ 與 DoH 差異及防護邊界 (`§1.1`, `§12.1`, `§12.6`, `DNS04`, `AT21`)
我們堅持如實揭露網域層級 DNS 防護的技術邊界：
1. **區分 DoT / DoQ（連接埠 `853`）與 DoH（HTTPS 連接埠 `443`）的繞過與政策行為（`§12.6`）**：
   - **DNS-over-TLS（DoT，使用 `TCP/853`）與 DNS-over-QUIC（DoQ，使用 `UDP/853`）**：例如 Android 系統的「私人 DNS（Private DNS）」或特定系統加密 DNS 設定，走專用連接埠 `853`。
   - **DNS-over-HTTPS（DoH，使用一般 HTTPS `TCP/443` 或 `UDP/443`）**：例如瀏覽器（Chrome、Edge、Firefox、Brave）內建的「安全 DNS（Use Secure DNS）」或應用程式自建的 DoH 查詢，會將 DNS 查詢封裝在一般 HTTPS（`443` 埠）流量中，且常與一般網站共用 CDN 或 IP。
   - **實際影響與限制（`§12.6`、`AT21`）**：外部 Port-53 DNS、DoT/DoQ（`853` 埠）及已知加密 DNS 端點雖可依部署所選政策加以限制，但對於未受管（unmanaged）會員裝置上的瀏覽器 DoH（`443` 埠）或開啟中的 Android 私人 DNS，本服務**無法保證全面辨識或攔截**。當這類設定開啟時，查詢可能繞過 VPN 指派的受控解析器（導致惡意網域過濾不生效）或因政策阻擋而解析失敗（各平台實際行為 **尚未實測 / not yet measured**，待執行 `AT21`）。
   - **建議設定**：使用本服務之網域防護時，請將 Android「私人 DNS」設為 **「關閉（Off）」**，並將瀏覽器自訂「安全 DNS / DoH」關閉（改用系統預設解析器），再依 3.3 節確認解析路徑。
2. **不進行 TLS 中間人解密（No MITM）與網址路徑檢查（`§1.1`、`§12.1`）**：
   - Freedom VPN **絕不攔截或解密您的 HTTPS / TLS 流量**，因此僅能比對「網域名稱（Domain）」，無法檢查或過濾同一網域下的特定網頁路徑（`/path`）、查詢參數或下載檔案內容。
3. **不攔截純 IP 直連與本機區域網路（`§1`、`§12.1`、`§12.6`）**：
   - 若程式不查詢 DNS 而直接連線至外部公網 IP 位址，DNS 規則不會觸發；且伺服器端出口隔離僅管制進入 VPN 隧道的流量，不涵蓋您設備在自身本地 Wi-Fi / LAN 上的區網連線。

---

## 7. 客服支援與誤擋回報流程 (`§12.5`, `§12.6`, `§18.3`, `§18.4`, `AT20`, `AT22`)

當您遇到連線異常、帳單問題或認為某個正常網站被誤攔截時：
1. **收集診斷識別碼（請勿提供密碼或私鑰，`§16.2`、`§18.3`）**：
   - 在會員入口網站（`/member`）發生錯誤時，請複製畫面錯誤訊息中的 **`Request ID`**（或操作紀錄中的 **`Operation ID`**）與發生時間（UTC）。
   - **安全警告**：支援人員**永遠不會**向您索取 OIDC 密碼、工作階段 Cookie（`__Host-fvpn_session`）或任何私鑰檔案。
2. **回報誤攔截網域（False Positive）與生效時間說明（`§12.5`、`§12.6`、`D-054`、`D-056`、`AT20`、`AT22`）**：
   - 請提供完整網域名稱（例如 `sub.example.com`）與發生時間（UTC）。管理員確認後可於 `domain_rule_set` 政策新增白名單覆寫（`action: "allow"`，可覆蓋一般威脅清單誤擋，但不會解除強制 deny 或內網 IP 隔離，`§12.4`）。
   - **解除封鎖之實際生效時間（不作全域 60 秒保證，`§12.5`、`docs/dns/PUBLISH.md`、`node/agent/resolver.go`）**：白名單發布後的端到端生效時間取決於三個獨立環節：
     1. **政策發布與節點套用延遲（`D-056`）**：中央背景作業會將可用的解析器文件參照加入簽署狀態，無須另行發布出口；節點驗證大小與摘要後套用，並回報 `dns_policy_evidence`。只有套用目前有效政策所指定的文件後，`dns_policy` 檢查才能通過。支援人員可在管理員節點詳細頁確認結果；排程間隔不是端到端生效保證，實際延遲仍待量測（[節點接線](../nodes/WIRING.md)、[節點執行行為](../node/README.md)）；
     2. **受控解析器之封鎖回應快取 TTL（`blocked_ttl_seconds`）**：`dns_profile` 中的 `blocked_ttl_seconds` 為可調整設定（schema 範圍 `60..3600` 秒，`internal/dnspolicy` 預設 `300` 秒，`60` 秒僅為 `§12.5` 建議起點），僅控制封鎖回應的 DNS TTL 標頭，並非端到端生效 SLA；
     3. **終端作業系統、瀏覽器快取與既有連線（`§12.5`）**：即使受控解析器已載入新版本，您的作業系統 DNS 快取、瀏覽器快取或已建立的長連線仍可能暫時沿用舊結果。若管理員通知已發布白名單，建議您清除本機 DNS 快取、重啟瀏覽器或重新連線 VPN 隧道以加速重新解析。

---

# Part II: English Member Guide (`en`)

## 1. Supported Platforms and Candidate Client Versions (`§3.2`, `§7.3`, `§18.5`, `APP01`, `docs/netbird/VERSIONS.md`)

Freedom VPN v1 is designed to use the official **NetBird Client** paired with the **Freedom VPN Member Portal** (`https://vpn.example.com/member`).

> **Important Status Notice (`§7.3`, `§19.1`, `docs/netbird/VERSIONS.md`)**: Every native client version listed below is currently a **candidate version (`待實測` — needs operator run)**. Linux container lab tests do not establish native desktop or mobile client compatibility; production support claims require completed real-device acceptance runs (`AT01`, `AT09`, `AT21`) with dated evidence.

| Platform | Candidate Client Version (`docs/netbird/VERSIONS.md`) | Target OS Baseline | Device Acceptance Status (`AT01`, `AT09`, `AT21`) |
|---|---|---|---|
| **Windows** | NetBird daemon/CLI and bundled desktop UI `v0.82.0` | Windows 10 (22H2+) / Windows 11 (`amd64` / `arm64`) | **Candidate — not yet measured (needs operator run, `AT01`, `AT09`, `AT21`)** |
| **macOS** | NetBird daemon/CLI and bundled desktop UI `v0.82.0` | macOS 14 Sonoma / macOS 15 Sequoia (Apple Silicon & Intel) | **Candidate — not yet measured (needs operator run, `AT01`, `AT09`, `AT21`)** |
| **iOS / iPadOS** | NetBird iOS client `v0.4.1` (`build 2`) | iOS / iPadOS 17.0+ | **Candidate — upstream labels `build 2` as a TestFlight release and does not assert App Store availability; not yet measured (needs operator run, `AT01`, `AT09`, `AT21`)** |
| **Android** | NetBird Android client `v0.6.2` (separate mobile version; not `0.82.0`) | Android 12+ | **Candidate — not yet measured (needs operator run, `AT01`, `AT09`, `AT21`)** |

---

## 2. Step-by-Step Setup and Automatic Central Authorization Across All Four Platforms (`§7.3`, `§9.2`, `§9.4`, `§18.4`, `§18.5`)

### 2.1 Step 1: Sign In to the Member Portal and Confirm an Active Entitlement (`/member`, `/member/billing`)

1. Open `https://vpn.example.com/member` in your browser, click **Sign In**, and authenticate with your OIDC account (`§9.1`, `§16.1`, `ID01`).
2. Under **Billing**, select a subscription plan and click **Checkout** (`POST /api/v1/member/checkout-sessions`, which returns `201 Created` with `checkout_url`). Once the payment provider webhook is verified and reconciled, your entitlement status transitions to **`Active`** (`§9.2`, `§9.3`, `BILL01`, `BILL03`; the browser return page alone never activates access, `AT03`).

### 2.2 Step 2: Point NetBird to the Management URL, Sign In via SSO, and Verify Automatic Central Authorization (`§7.3`, `§9.4`, `D-052`, `docs/members/README.md`)

> **How Device Authorization and `D-052` Shared-Issuer Public-Subject Discovery Actually Work (`§2 [S36]`, `§7.3`, `§9.4`, `§15.2`, `MEM01`, `MEM02`)**: There is **no** manual authorization button on `/member/devices` and no member device-creation API endpoint in the 49-operation OpenAPI `1.9.0` contract. Once the server projects the mark, a device fenced after a ledger restore reads `status: pending` with `authorization_unconfirmed: true`; until then the field is absent. You cannot clear that flag, and there is still no member authorize button. When you point the official NetBird client's **Management URL** to `https://netbird.example.com` and sign in with the **same OIDC issuer and public subject** used in the Member Portal, `fvpn-worker`'s `devices.discover` handler matches the NetBird user ID to `identity_bindings.subject` under `oidc_issuer` and writes `netbird_user_bindings` (`mapping_proof = "shared_issuer_public_subject:" + oidc_issuer`, excluding service/blocked accounts and failing closed as `identity_unmapped` on pairwise or unknown subjects—never matching by email or display name). Central then checks your active entitlement and `device_limit` quota under a database transaction lock, and dispatches the asynchronous `member.device_authorize` synchronization job to grant exit and DNS permissions. You simply check `https://vpn.example.com/member/devices` (`GET /api/v1/member/devices`) to see your device appear and transition from `pending` to **`authorized`**.

Your member identity is the same canonical issuer and subject across a portal OIDC client-ID rotation (`issuer.Canonical` in `internal/members/issuer/issuer.go` trims only literal trailing `/` characters from the escaped URL path and preserves `%2F` / `%2f`, `docs/members/README.md`); the existing member is reused. If identity matching is refused, contact support instead of trying to link a different account by matching its email ([identity rules](../control/AUTH.md)).

#### A. Windows (Candidate `v0.82.0`, needs operator run `AT01`)
1. Install the candidate `v0.82.0` Windows installer.
2. Right-click the NetBird tray icon -> **Settings** -> **Advanced**, set **Management URL** to `https://netbird.example.com`, and save (or run `netbird up --management-url https://netbird.example.com` in PowerShell).
3. Complete the browser OIDC SSO sign-in when prompted.
4. Open `https://vpn.example.com/member/devices` and confirm your Windows device appears automatically and transitions from `pending` to **`authorized`** (if your device quota is full, revoke an older device on `/member/devices` first, `AT02`).
5. Click **Connect** in the NetBird tray menu and manually select your desired **Exit Node** (e.g., `tw-tpe-01`, or run `netbird networks select <fvpn-exit-…>` on the CLI; independent exits have `skip_auto_apply = true`, so **a device forwards traffic through an exit only after selecting it**—`netbird networks select all` does not select an exit on NetBird `0.82.0`, `§7.2`, `§18.4`, `EXIT02`, [`docs/e2e/RUN.md`](../e2e/RUN.md) § "Native exit selection and traffic trace").

#### B. macOS (Candidate `v0.82.0`, needs operator run `AT01`)
1. Install the candidate `v0.82.0` macOS package, launch NetBird, and approve the macOS VPN configuration and system extension prompts.
2. Click the NetBird menu bar icon -> **Preferences / Settings** -> **Advanced**, set **Management URL** to `https://netbird.example.com`, and click **Connect** to complete OIDC SSO sign-in.
3. Open `https://vpn.example.com/member/devices` and verify that central device synchronization has transitioned your Mac from `pending` to **`authorized`**.
4. Manually select an authorized exit node from the NetBird menu bar icon (or run `netbird networks select <fvpn-exit-…>`; a device forwards through the exit only after selecting it).

#### C. iOS / iPadOS (Candidate `v0.4.1 build 2` TestFlight, needs operator run `AT01`)
1. Install the candidate **NetBird** iOS client (`v0.4.1, build 2`; note: upstream labels `build 2` as a **TestFlight** release and does not assert App Store availability — verify the exact distribution channel and build during the `AT01` operator run, `docs/netbird/VERSIONS.md`).
2. Open the app, tap **Settings -> Change Server / Server**, enter `https://netbird.example.com` (do not enter a setup key), and tap **Save**.
3. Tap the connect button, approve the iOS VPN configuration prompt, and sign in with your OIDC account in the browser sheet.
4. Open `https://vpn.example.com/member/devices` and confirm your iPhone/iPad is listed and automatically transitions from `pending` to **`authorized`**.
5. In the NetBird app (`Connection -> Exit node`), manually select an authorized exit node (a device forwards through the exit only after selecting it).

#### D. Android (Candidate `v0.6.2`, needs operator run `AT01`, `AT21`)
1. Install the candidate **NetBird** Android app (`v0.6.2`).
2. Open **Settings -> Profiles -> Add**, set **Management Server** to Self-hosted `https://netbird.example.com` (no setup key), and save.
3. Tap Connect, accept the Android VPN permission prompt, and complete OIDC SSO sign-in.
4. Check `https://vpn.example.com/member/devices` to confirm your Android device is automatically **`authorized`**, then manually select an exit node in the app (a device forwards through the exit only after selecting it).
5. **Important DNS Step (`§12.6`, `AT21`)**: Android's system "Private DNS" bypasses VPN-assigned DNS resolvers when enabled. To use Freedom VPN's domain filtering, open Android **Settings -> Network & internet -> Private DNS** (exact menu path varies by OEM) and set it to **Off** (see Section 6 below).

---

## 3. Manual Verification Scripts (`AT01`, `AT09`, `AT21`, `AT29`) (`§7.3`, `§8.1`–`§8.3`, `§12.1`, `§12.6`, `§16.3`, `§16.5`, `§19`)

> **Measurement Status (`§19.1`, `docs/e2e/README.md`)**: The procedures below define the verification steps for controlled lab and native-client acceptance (`docs/e2e/at/members/OPERATORS.md`). Native four-platform runs (`AT01`, `AT09`, `AT21`) and live end-to-end forwarding cutoff timings (`AT04`, `AT13`, `AT29`) are **not yet measured (needs operator run)**.

### 3.1 `AT01`, `AT10`, `AT12`: Verify Controlled Exit Egress IP and Member-to-Member Isolation (`§7.3`, `§8.1`, `§8.2`, `ISO01`, `ISO02`)

1. **Authenticated Member Session Check (`GET /api/v1/member/me`)**:
   - `/api/v1/member/me` returns your account and entitlement JSON (`status`, `effective_service_end`, `plan`), **not** your client egress IP, and requires an authenticated session cookie passed via a `0600` curl config file (so session tokens are never exposed on the command line, `§16.1`, `§16.2`):
   ```bash
   # ~/.config/fvpn/session.curl (mode 0600) contains: header = "Cookie: __Host-fvpn-member=<SESSION_TOKEN>"
   curl -fsS --config ~/.config/fvpn/session.curl https://vpn.example.com/api/v1/member/me
   ```
2. **Verify Exit Egress IP via the Controlled Lab Origin (`AT01`, `AT09`)**:
   - Per `AGENTS.md §7` and `§19.1`, never probe third-party IP-echo websites. Query the **controlled test/lab origin** (for example `https://echo.test.example.invalid/ip`, which echoes the observed client source address) **before** enabling the exit route and **after** selecting the exit node, confirming that the observed source IP changes to the selected exit node's egress IP (e.g., documentation address `192.0.2.20`):
   ```bash
   # Before selecting exit node: returns your pre-tunnel underlay address
   curl -fsS https://echo.test.example.invalid/ip

   # After selecting exit node: must return the selected exit node's egress IP (e.g., 192.0.2.20)
   curl -fsS https://echo.test.example.invalid/ip
   ```
3. **Member-to-Member Isolation with a Positive Reachability Control (`ISO01`, `AT12`)**:
   - A failed ping alone does not prove isolation if the tunnel itself is down. Always execute a **positive reachability control** over the tunnel first (confirming that the controlled test origin `https://echo.test.example.invalid/healthz` and the configured resolver `100.64.0.53` on port `53` are reachable, and that a second member's device at documentation overlay address `100.64.0.11` is online), and then verify that traffic to `100.64.0.11` is dropped:
   - **macOS / Linux Terminal (Candidate verification script — not yet measured / needs operator run, `AT01`, `AT12`)**:
     ```bash
     # 1. Positive reachability control over the active tunnel
     curl -fsS https://echo.test.example.invalid/healthz
     nslookup example.com 100.64.0.53

     # 2. Member-to-member isolation check (100.64.0.11 = online second member peer) -> must drop 100%
     ping -c 3 -W 2 100.64.0.11
     # Expected: 100.0% packet loss (dropped by NetBird policy and exit-node inet fvpn rules)
     ```
   - **Windows PowerShell (Candidate verification script — not yet measured / needs operator run, `AT01`, `AT12`)**:
     ```powershell
     # 1. Positive reachability control
     Invoke-RestMethod -Uri https://echo.test.example.invalid/healthz

     # 2. Member-to-member isolation check against online second member peer (100.64.0.11)
     Test-NetConnection -ComputerName 100.64.0.11 -Port 443
     # Expected: TcpTestSucceeded : False
     ```

### 3.2 `AT09`, `AT29`: Server-Side Fail-Closed Gate, No Client Kill-Switch Guarantee, and Platform Transition Behavior (`§7.3`, `§8.2`, `§16.3`, `§16.5`, `ISO02`, `ENT02`)

1. **Explicit Limitation: No Complete Client Kill-Switch Guarantee (`§7.3`, `§17.1`)**:
   - Freedom VPN v1 uses the stock official NetBird client, which **does not provide a complete OS-level always-on / kill-switch mode** (matching the warning in Portal `member.gettingStarted.killSwitchWarningBody`).
   - **Server-Side Fail-Closed vs. Endpoint Routing**: The exit node's `inet fvpn` fail-closed rules guarantee only that **the exit node itself** stops forwarding packets that arrive over the tunnel when a lease, entitlement, or safety check fails. They do **not** lock down your device's local routing table: if the tunnel disconnects, the client app stops, the device wakes from sleep, or you switch between Wi-Fi and cellular networks, your operating system **may fall back to sending traffic directly over your local ISP** (`§7.3`).
2. **Three Distinct Server-Side Cutoff Mechanisms and Unmeasured Timings (`§16.3`, `§16.5`, `AT04`, `AT13`, `AT29`)**:
   - **Node-Local Watchdog Staleness (`§16.3`)**: On each exit node, `fvpn-watchdog` monitors the local `fvpn-agent` heartbeat file (configured `watchdog_stale_seconds = 30` seconds by default) and reverts the local `inet fvpn` table to baseline drop if the local agent stalls or crashes.
   - **Online Revocation / Disable (`§16.5`, `AT04`, `AT13`)**: When the control plane and exit node are online, administrative disable, member suspension, or device revocation has a spec design target starting at **60 seconds** (`§16.5` NFR target — not instantaneous, not a 30-second guarantee, and not applicable while a node is offline).
   - **Offline Lease / Service Deadline (`§16.3`, `AT29`)**: When an exit node cannot reach the control plane, it enforces the earliest of `effective_service_end`, `node_lease_end`, and `signed_server_time + max_offline_seconds` (configured default `86,400` seconds / 24 hours) via kernel nftables set timeouts (`@allowed_v4`), independent of wall-clock rollback.
   - **Measurement status (`DOC15`, `DOC16`, `#165`, `#170`, `#196`, `#201`, `#203`, `#231`):** On current `main`, `deploy/devstack/Dockerfile.node` installs `conntrack` and `tcpdump`, `SystemPlane.MissingExecutables()` (`node/agent/plane.go`) preflights `conntrack` via `enforce.RequireConntrack`, `node/agent/sync.go` renews the serving lease immediately in the step that applies a new desired-state generation (`#146`, `#153`, `#162`, `#166`), and after a node restart `fvpn-agent` treats a same-generation grant-less lease (`members: []`) as provisional under `D-067` (`#193`, `#198`, `#200`, reporting `awaiting_member_grants` and renewing after one heartbeat, then doubling, capped at `60 s`). The earlier `blocked` `AT02`/`AT03`/`AT04`/`AT08` records in [members acceptance follow-up 3 (`#151`)](../e2e/at/members/README.md) ([members requests](../e2e/at/members/requests.md)) and the initial [devstack follow-up 9 (`#153`)](../e2e/followup9-verification.md) snapshot are historical records captured before those fixes were re-verified together and do not represent current `main` status ([members follow-up 5 (`#201`) and follow-up 6 (`#231`, run `80db103b-622c-42b1-946d-4628daaa213d`)](../e2e/at/members/summary.md) pass `AT02` (`42/0` subchecks, including strict `fvmb0` ingress injection via `Stack.ObserveMemberIngress` where the revoked device raises `ctunku` `0 -> 1` with nothing forwarded while the admitted sibling moves `u_` `31 -> 32`), `AT03`, and `AT08`, and `AT09` measures the `D-067` provisional window at `36.976 s` to serving-ready and `41.695 s` to first counted byte after cold restart on unchanged generation `23`, with `7` cases blocked on lab/device resources — `3 pass / 7 blocked / 0 fail`, `384` passing subchecks). The follow-up 5 (`#165`) two-device baseline evidence is pinned to the immutable archive [security-ops follow-up 5 archive (`#165`)](../e2e/at/secops/history/follow-up-5/README.md) alongside [devstack follow-up 10 (`#170`)](../e2e/RUN.md): (1) [`history/follow-up-5/counted-flow.json`](../e2e/at/secops/history/follow-up-5/counted-flow.json) (`SECOPS-09` closed) records two-device counted member forwarding through the selected exit (`netbird networks select`, `forward_admission_v1` `0 -> 399` upload / `0 -> 678` download bytes, `usage_events = 2`, `usage_rollups = 4`, and zero `probe_up`/`probe_down` diagnostic leakage); (2) [`history/follow-up-5/post-revoke.json`](../e2e/at/secops/history/follow-up-5/post-revoke.json) (`F3-02` coherence passing), [`device-revoke.md`](../e2e/runs/2026-10-30/followup10-run3/device-revoke.md) ([`docs/e2e/requests.md`](../e2e/requests.md)), and [devstack follow-up 12 (`#196`, `#199`)](../e2e/scenarios/device-revoke.md) prove real `DELETE /api/v1/member/devices/{device_id}` revocation, NetBird peer deletion, `@allowed_v4` element removal, controlled stale-client 3-point `tcpdump` verification (`peer_tx_packets > 0`, `mor0_rx_packets == 0`, `uplink_packets == 0`), and 4th-device replacement (`409 device_limit_reached` before revoke -> `202 succeeded` after revoke); (3) [`history/follow-up-5/deadlines.json`](../e2e/at/secops/history/follow-up-5/deadlines.json) and [2026-10-30 lease-expiry](../e2e/runs/2026-10-30/lease-expiry.json) record the two-device baseline `fvpn-watchdog` stale-heartbeat cutover (`35s` stale age -> `fv-watchdog-enforce`), desired-state expiry (`policy_missing`), lease expiry (`lease_expired`), and component clock tests (kept distinct from disposable guest VM wall-clock rollback `clock_settime`); and (4) [`history/follow-up-5/capacity.json`](../e2e/at/secops/history/follow-up-5/capacity.json) records the two-device counted-capacity subset, while `Stack.StartSecondExit` / `EnrollAt(..., "node2")` ([`docs/e2e/RUN.md`](../e2e/RUN.md) `second-exit`) provides a second real exit lab hook. Historical secops follow-up 7 measurements are recorded in [security-ops follow-up 7 (`#203`, run `ab96acac-8713-485a-9c29-88f20110e2e5`)](../e2e/at/secops/summary.md): [`deadlines.json`](../e2e/at/secops/deadlines.json) exercises the `D-065` fleet API (`GET`/`PUT /api/v1/admin/fleet-config` with `428`/`412`/idempotent replay/`fleet_config.update` audit and `max_offline_lease_seconds: 60`, restored to `86400`) and finite grace (`FVPN_E2E_GRACE_SECONDS=90`) to verify paid/grant/lease cutoff (`495/869` -> `0/0` bytes at expiry) and recovery (`0 -> 1320/2316` bytes), and [`capacity.json`](../e2e/at/secops/capacity.json) and [`node-telemetry.json`](../e2e/at/secops/node-telemetry.json) verify `60/60` admitted native peers (`20` members × `3` devices, `ScaleNativeMembers` `296.017 s`) with `60/60` counted peers (`node1` `39,600`/`69,480` bytes, `node2` `7,920`/`13,896` bytes), `5,960/5,960` HTTP `200` `GET /api/v1/member/devices` reads (`0` wrong-member responses, `p50/p95/p99/max` = `25.81/96.32/149.94/302.18 ms`), and `82` node telemetry samples at `5 s` cadence (`0` errors). **Historical follow-up 7 qualifications (`AT28`–`AT32` were `blocked` with passed subsets; AT30 later passed in #265):** `AT29` `past_due` payment truth (`SECOPS-17`), active-load established TCP and UDP cutoff, and disposable guest VM kernel wall-clock rollback (`clock_settime`, kept distinct from component clock tests and remaining `needs operator run`), along with `AT31` forced-relay control and candidate hardware certification, native four-platform checks (`AT01`, `AT09`, `AT21`), and the full 24-hour `AT29` campaign remain `blocked` / `needs operator run` / qualified.
3. **Per-Platform Disconnect, Sleep/Wake, and Network-Switch Matrix (`§7.3`, `§18.5`, `APP01`, `AT01`, `AT09`)**:
   - Per `docs/e2e/at/members/OPERATORS.md`, each platform's disconnect, sleep/lock, and network-switch behavior must be recorded during operator runs:

| Platform | Required Test Actions (`AT01` / `AT09`) | Observations to Record (Tunnel Recovery, Local-ISP Fallback, Recovery Seconds) | Current Status |
|---|---|---|---|
| **Windows (`v0.82.0` candidate)** | Sleep 2 min -> wake; switch between two controlled Wi-Fi networks; stop selected exit node while connected | Selected exit, controlled echo source IP, DNS path, recovery time, and whether traffic falls back to the local ISP when the exit stops | **Not yet measured (pending `AT01`, `AT09` operator run)** |
| **macOS (`v0.82.0` candidate)** | Sleep 2 min -> wake; switch between two controlled Wi-Fi networks; stop selected exit node while connected | Selected exit, controlled echo source IP, DNS path, recovery time, and whether traffic falls back to the local ISP when the exit stops | **Not yet measured (pending `AT01`, `AT09` operator run)** |
| **iOS / iPadOS (`v0.4.1 build 2` candidate)** | Lock screen 2 min -> wake (recorded as mobile suspension, not desktop sleep); switch Wi-Fi networks; stop selected exit node | Tunnel state, controlled echo source IP, DNS path, and route/fallback behavior on wake and exit stop | **Not yet measured (pending `AT01`, `AT09` operator run)** |
| **Android (`v0.6.2` candidate)** | Lock / power-save 2 min -> wake (recording battery optimization setting); switch Wi-Fi networks; stop selected exit node | Tunnel state, controlled echo source IP, DNS path, and route/fallback behavior on wake and exit stop | **Not yet measured (pending `AT01`, `AT09` operator run)** |

### 3.3 `AT20`, `AT21`: Verify DNS Threat Blocking (`NXDOMAIN`) and Allow-List Resolution (`§12.1`, `§12.4`, `§12.6`, `§12.8`, `DNS01`–`DNS04`)

- **macOS / Linux (`nslookup` / `dig`, candidate verification script — not yet measured / needs operator run, `AT20`, `AT21`)**:
  ```bash
  # 1. Query a blocked test domain against a configured resolver (example 100.64.0.53) -> expected NXDOMAIN
  nslookup malware.test.example.invalid 100.64.0.53

  # 2. Query a normal or allow-listed domain -> expected NOERROR and an IP address
  nslookup example.com 100.64.0.53
  ```
- **Windows PowerShell (Candidate verification script — not yet measured / needs operator run, `AT20`, `AT21`)**:
  ```powershell
  Resolve-DnsName malware.test.example.invalid -Server 100.64.0.53
  # Expected: DNS name does not exist (RCODE=3 NXDOMAIN)
  Resolve-DnsName example.com -Server 100.64.0.53
  ```
- **System Resolver & Browser Path Across All Four Platforms (`§12.6`, `AT21`)**:
  In addition to direct `nslookup` queries, per-platform acceptance (`AT21`, **not yet measured / needs operator run**) must verify resolution through the OS system resolver and a standard browser with Android Private DNS and custom browser DoH turned Off, and then record behavior when Private DNS (DoT) or browser DoH is enabled.

---

## 4. Device Limits and Self-Service Revocation (`/member/devices`, `§7.3`, `§9.4`, `MEM01`–`MEM03`, `ENT02`, `AT02`, `AT04`)

1. **Default Device Limit (`§9.4`, `§21`, `MEM02`)**: By recommended default, each member can have up to **3 authorized devices** (`device_limit = 3`, configurable per plan and shown as `used / limit` on `/member/devices`).
2. **Limit Reached (`409 DEVICE_LIMIT_REACHED`, `AT02`)**: If you sign into NetBird on a new device when all of your device slots are already occupied, central automatic device authorization rejects the new peer under a row lock (`DEVICE_LIMIT_REACHED`, `AT02`) and does not grant it exit forwarding permissions (`§7.3`, `§9.4`).
3. **Self-Service Revocation and Cutoff Timing (`§9.4`, `§16.3`, `§16.5`, `AT02`, `AT04`, `AT29`)**:
   - Visit `https://vpn.example.com/member/devices` and click **Revoke** next to the device you want to remove (`DELETE /api/v1/member/devices/{device_id}`, which creates an asynchronous `member.device_revoke` operation).
   - **Propagation Timing (Not Instantaneous)**: Revoking a device is an asynchronous control-plane and node-policy operation, **not** an instantaneous global disconnect. When the control plane and exit nodes are online, the spec design target for revoking existing and new flows starts at **60 seconds** (`§16.5`; actual end-to-end cutoff timing is **not yet measured**, pending `AT02`, `AT04`, `AT29`). If an exit node is unreachable from the control plane, forwarding on that node remains bounded by the earliest kernel lease/service deadline (`min(effective_service_end, node_lease_end, signed_server_time + max_offline_seconds)`, default `86,400s`, `§16.3`, `AT29`).
   - Once the revocation operation reaches **`succeeded`**, the device slot is released for a new device.

When a plan reduces your device allowance, central keeps the oldest authorizations and moves excess devices to `pending` while withdrawing access. Restored entitlement readmits previously authorized, non-revoked devices with valid identity proof within the current quota. After a service restore, a device may show as `pending` until an operator confirms it, and nothing you do on the member side changes that. Under `D-077` (`#238`, `#244`; 2026-10-30 EDT), `authorization_unconfirmed: true` is the “needs operator re-authorization” mark on a `pending` device. The field is optional in the contract: absent means the server does not project it yet, not that authorization is confirmed. On that day, `main` always emits the boolean on `GET /api/v1/member/devices`, including `false` for devices without a fence. Members cannot re-authorize themselves; contact an operator or revoke the device (`D-076`, `#227`). Check `/member/devices` for current status; see [member and device rules](../members/README.md#quota-withdrawals-and-finite-binding-history).

---

## 5. Usage Metering and Fair-Use Policy (FUP) Display (`/member/usage`, `§8.3`, `§10`, `§11`, `MTR01`–`MTR05`, `FUP01`–`FUP04`, `AT14`, `AT17`–`AT19`)

### 5.1 Metering Definition (`forward_admission_v1`) and v1 IPv6-Block Mode (`§8.3`, `§10.1`, `MTR01`, `AT11`, `AT14`)
- **Formal Metering Point (`§10.1`)**: `forward_admission_v1` counts inner IP packet bytes (including inner IP and transport headers, once per direction) that have passed tunnel decapsulation, member binding verification, and Internet-only isolation checks at the exit-node `inet fvpn` admission hook. Packets dropped before the admission hook, outer WireGuard encapsulation, and node control traffic are excluded; packets dropped after the admission hook by `tc` shaping or downstream links may still be counted, so this figure represents **recorded VPN forwarding admission usage**, not application-layer delivered file size (`§10.1`).
- **v1 IPv6-Block Mode (`§8.3`, `node/enforce`)**: Although the usage ledger and binding schemas (`§10.1`, `§10.2`) include fields for IPv4 and IPv6 bindings, **Freedom VPN v1 data-plane enforcement (`node/enforce`) implements `IPv6Block` (`block`) mode only**: the `inet fvpn` forward chain unconditionally drops all member IPv6 traffic (`meta nfproto ipv6 ... drop`) and forwards authorized member **IPv4** traffic only. No IPv6 Internet forwarding path is enabled in v1 (`AT11`, `AT14`).

### 5.2 UTC Monthly Reset and Boundary Attribution (`§10.4`, `§10.9`, `§11.1`, `MTR04`, `FUP01`, `AT17`, `AT19`)
- Usage totals and Fair-Use tiers are evaluated over each **UTC calendar month** (`[YYYY-MM-01T00:00:00Z, next_month)`, `§10.4`, `§11.1`).
- Late-arriving events from a previous UTC month are attributed to that previous month and do not re-trigger throttling in the new month (`§10.9`, `AT17`). At the start of a new UTC month (`00:00:00Z`), the current period total and Fair-Use tier reset to `Tier 0` (`§11.2`, `AT19`).

### 5.3 Fair-Use Tiers, Configured Shaping Floor, Shared Capacity, and Mandatory Fail-Closed Exceptions (`§11.1`–`§11.7`, `FUP01`–`FUP04`, `AT18`, `AT19`)
- **Configured Shaping Floor (`minimum_rate`, `§11.1`, `§11.2`, `AT19`)**: When your cumulative monthly usage across all devices and exit nodes crosses a tier threshold in `enforce` mode, your per-exit HTB class ceiling (`down_bps` / `up_bps`) steps down to that tier's configured rate. Every tier is validated to stay at or above a positive configured **`minimum_rate`** (`> 0 bps`), meaning **Fair-Use tiering itself does not configure a `0 bps` rate for exceeding a byte threshold**.
- **Not a Guaranteed Delivered Speed & Subject to Shared Capacity (`§11.3`, `§11.6`, `§11.7`, `FUP02`, `FUP04`, `AT18`)**:
  - `minimum_rate` and tier rates are **configured `tc` HTB shaper parameters** on the exit node, **not** a guaranteed delivered download or upload speed.
  - **Per-Exit Multi-Device Sharing (`§11.3`, `AT18`)**: All authorized devices under your account connected to the **same exit node** share a single per-member HTB class on that exit node (`rate_scope = (member_id, node_id, direction)`).
  - **Shared Node Capacity Ceiling (`§11.3`, `FUP04`)**: All members on an exit node share the node's parent capacity ceiling (`node_capacity`: `qos.Up` / `qos.Down`) as well as physical uplink and Relay bandwidth; under contention, actual throughput can be lower than your individual class ceiling.
- **Mandatory Fail-Closed Exceptions Still Stop Forwarding (`§8.2`, `§10.7`, `§10.8`, `§13.6`, `§16.3`, `ISO02`)**:
  - `minimum_rate` never overrides safety, entitlement, or metering-integrity rules. An exit node **will stop forwarding your traffic (fail-closed)** whenever:
    1. Your subscription/entitlement or the node's offline serving lease (`forwarding_deadline`) expires, or your account/device is suspended or revoked (`§9.2`, `§16.3`, `AT04`, `AT29`);
    2. The exit node fails an isolation, DNS baseline, or `fvpn-watchdog` check (`§8.2`, `§16.3`, `AT10`, `AT13`);
    3. The node's local meter path fails (`METER_PATH_INVALID`), local SQLite storage becomes unwritable, or the unacknowledged usage outbox reaches its protection limit (`72 hours` or `256 MB` recommended default, `§10.3`, `§10.7`, `§10.8`, `AT16`);
    4. The node enters the disruptive switch/check window of a software update (`applying` / `checking`, `§13.2`, `§13.6`).
- **Shadow Mode (`§11.1`, `§11.2`, `AT19`)**: When `shadow` mode is active, the Portal displays your informational `would_apply_tier` while `throttled` remains `false`.

### 5.4 Quality Badges (`§10.6`–`§10.9`, `§18.4`, `AT16`, `AT17`, `AT30`)
- **`Aggregate as of`**: UTC timestamp of the latest usage projection watermark committed in the central ledger (`§10.6`).
- **`Reconciliation in progress`**: Shown while nodes verify signed checkpoints and replay retained events after a central database restore (`§10.8`, `§17.3`, `AT30`).
- **`Gap` / `Boundary Estimated`**: `gap` indicates that an exit node experienced an unscheduled power loss, reboot, or retention expiry leaving unrecorded intervals; `boundary_estimated` indicates an interval spanning a UTC month boundary that was split proportionally by elapsed time. Freedom VPN **never fabricates or guesses missing byte counts** (`§10.7`–`§10.9`).

---

## 6. DNS Behaviour, Data Minimization, and Explicit Limitations (`§8.1`, `§12.1`–`§12.7`, `DNS01`–`DNS04`, `SEC02`, `AT12`, `AT20`–`AT22`)

The Member Portal's `/member/dns-protection` page explains filtering scope, limitations and false-positive reporting; `/member/exits` shows exits and availability. Typed node-reported `dns_policy_evidence` is on the admin node-detail page and is not currently shown in member views. Ask support to inspect it for the selected exit. [Portal documentation](../portal/README.md) remains authoritative for the interface.

### 6.1 Configured Resolvers, Firewall Rules, and Data Minimization (`§8.1`, `§12.2`, `§12.6`, `§12.7`, `DNS03`, `SEC02`, `AT12`, `AT21`, `AT22`)
- **Configured DNS Resolvers and `inet fvpn` Rules (`node/enforce/render_nft.go`)**:
  - NetBird distributes the deployment's configured AdGuard Home resolver addresses (for example, documentation addresses `100.64.0.53` and `100.64.0.54` per `AGENTS.md §8`; actual addresses depend on `nodes.json:dns_resolver_ipv4`) to authorized member clients (`§12.2`).
  - In the exit node's `inet fvpn` forward chain (`node/enforce/render_nft.go`), `Render` permits `UDP/53` and `TCP/53` from authorized member addresses (`@allowed_v4`) to the configured resolver set (`@resolver_v4`, counted under `dns_service_v1`) and **drops** all non-port-53 traffic to `@resolver_v4` (`counterBadDNS`, blocking access to the resolver's admin UI or SSH, `§8.1`, `DNS03`, `AT12`).
  - **No Universal Port-53 DNAT Interception**: `inet fvpn` **does not** perform universal DNAT redirection of arbitrary external port-53 traffic. How manual external port-53 DNS queries from a client are handled depends on the deployment's selected external-DNS policy (`§12.6`, marked `待決策` / pending decision, and **not yet measured** under `AT21`).
  - Configured resolvers return `NXDOMAIN` for blocked malware, phishing, and admin-denied domains. If one controlled resolver becomes unavailable, failover uses only the other controlled resolver with the same policy—never an unfiltered public DNS resolver (`§12.2`, `§12.5`, `AT21`).
- **Data Minimization and No Query History Logging (`SEC02`, `§12.7`, `§16.4`, `AT22`)**:
  - AdGuard Home query logs and statistics are disabled in the candidate configuration (`querylog.enabled: false`, `querylog.file_enabled: false`, `statistics.enabled: false`). Service logs, metrics, and the central database never record full queried domain names (qnames), full URLs, or packet payloads (`§12.7`).
  - **Honest Scope (`§16.4`)**: Freedom VPN **does not claim to be a complete no-log service**—the platform retains necessary entitlement records, device authorization state, anonymized usage accounting (`usage_events` / `usage_rollups`), and administrative/security audit trails (`audit_events`).

### 6.2 Explicit Technical Limitations: DoT / DoQ vs. DoH and Filtering Boundaries (`§1.1`, `§12.1`, `§12.6`, `DNS04`, `AT21`)
1. **Distinguishing DoT / DoQ (`853`) from DoH (`443`) and Policy/Platform Dependence (`§12.6`)**:
   - **DNS-over-TLS (DoT, `TCP/853`) and DNS-over-QUIC (DoQ, `UDP/853`)**: Used by features such as Android "Private DNS" or dedicated encrypted-DNS profiles over port `853`.
   - **DNS-over-HTTPS (DoH, `TCP/443` and `UDP/443`)**: Used by browser "Use Secure DNS" settings (Chrome, Firefox, Edge, Brave) or application-level DoH resolvers over standard HTTPS port `443`, often sharing IP addresses and CDNs with ordinary websites.
   - **Behavior and Limitations (`§12.6`, `AT21`)**: While external port-53 DNS, DoT/DoQ (`853`), and known encrypted-DNS endpoints may be restricted according to the deployment's selected policy (`§12.6`), an unmanaged client device or browser using DoH over port `443` (or Android Private DNS when enabled) cannot be guaranteed to be intercepted and may either bypass VPN-assigned DNS filtering or fail to resolve depending on the active policy and OS/browser behavior (**not yet measured across all four native platforms; tracked under `AT21`**).
   - **Recommended Configuration**: Turn **Off** Android "Private DNS" and disable custom browser "Secure DNS / DoH" (use the system default resolver), then verify your resolution path using Section 3.3.
2. **No TLS Interception (No MITM) and No URL-Path Filtering (`§1.1`, `§12.1`)**:
   - Freedom VPN never intercepts TLS certificates or decrypts HTTPS traffic. Filtering operates strictly on **domain names**, not on `/path` segments, query strings, or downloaded file contents.
3. **No Direct-IP Blocking via DNS and Local LAN Boundary (`§1`, `§12.1`, `§12.6`)**:
   - Connections made directly to raw IP addresses without a DNS lookup do not traverse DNS rules, and server-side exit isolation applies only to traffic entering the VPN tunnel, not to your device's local Wi-Fi/LAN subnet.

---

## 7. Support Contact Flow and False-Positive Unblocking (`§12.5`, `§12.6`, `§18.3`, `§18.4`, `AT20`, `AT22`)

1. **Include Diagnostic IDs (Never Share Secrets, `§16.2`, `§18.3`)**:
   - When contacting support about an error in the Member Portal, include the **`Request ID`** shown in the error banner (or the **`Operation ID`** from `/member/devices`) along with the approximate UTC timestamp.
   - **Never** send your OIDC password, session cookies (`__Host-fvpn_session`), or private keys to support.
2. **Reporting False Positives and Unblock Propagation Timing (`§12.5`, `§12.6`, `D-054`, `D-056`, `AT20`, `AT22`)**:
   - Send the exact domain name (e.g., `app.example.com`) and UTC timestamp to support. Once approved, an administrator activates a new `domain_rule_set` policy revision with an `allow` rule (which overrides general threat-feed false positives, but never overrides an explicit admin `block` rule or IP-layer private-network isolation, `§12.4`).
   - **How Unblock Propagation Timing Works (No Universal 60-Second Guarantee, `§12.5`, `docs/dns/PUBLISH.md`, `node/agent/resolver.go`)**: End-to-end propagation of an `allow` override depends on three separate factors:
     1. **Publication and node apply delay (`D-056`)**: Central background reconciliation adds an eligible resolver-document reference to signed state without another exit-publication command. The node verifies size/digest, applies the document and reports `dns_policy_evidence`; `dns_policy` passes only after the exact document named by the in-force policy is applied. Support can inspect this on admin node detail. Scheduling cadence is not an end-to-end apply guarantee; actual delay still needs measurement ([node wiring](../nodes/WIRING.md), [node runtime](../node/README.md));
     2. **Configured Blocked-Response TTL (`blocked_ttl_seconds`)**: The `dns_profile` setting `blocked_ttl_seconds` (schema bounds `60..3600` seconds, default `300` seconds in `internal/dnspolicy`, `60` seconds as the recommended starting point in `§12.5`) governs only the DNS TTL header on blocked responses, not end-to-end propagation time;
     3. **Client OS, Browser Caches, and Existing Connections (`§12.5`)**: Even after controlled resolvers apply the new revision, your operating system DNS cache, browser DNS/socket cache, or already-open HTTPS/QUIC sessions may continue holding cached results until they expire or you flush your local DNS cache, restart the browser, or reconnect the VPN tunnel.

### 2026-10-31 更新 / update (D-087, #250)

無成員授權的租約仍持續排程續租，包括尚無 desired bindings 的情況。首次請求時段為一個 heartbeat 後（上限 60 秒），之後倍增至最多每 60 秒，且遵守租約提前續租時間。這是請求排程，不是 I/O 完成保證；取得有成員授權的租約還需要已發布且有合格綁定的 generation、已接受的 readiness，以及可回應的控制服務。`identity_frozen` 沒有會員可見欄位；會員只會看到裝置不再提供服務及 Fair-Use 摘要。

D-087 (#250): every accepted grant-less lease has a renewal schedule, even before desired bindings arrive. The first request slot is one heartbeat after the empty lease (capped at 60 s), then doubling, never more than 60 s of local waiting between requests; the lease renewal lead can bring a slot forward. A populated lease additionally needs a published generation with eligible member bindings, accepted readiness, and a responsive control service. A scheduled request slot is not completed I/O: ongoing work, request latency and server `Retry-After` can delay completion. Empty bindings do not stop renewal. `identity_frozen` has no member-visible surface; members see the device become non-serving and the Fair-Use summary. See the [dated acceptance snapshot](../ops/README.md#follow-up-14-acceptance-snapshot-2026-10-31-edt) for usage #257 and members #268. The members run did not exercise portal rendering of D-077.

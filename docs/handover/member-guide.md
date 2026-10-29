# Freedom VPN 會員使用手冊 / Member Guide (`docs/handover/member-guide.md`)

本文件包含繁體中文（預設語言，`zh-TW`）與英文（`en`）雙語完整說明（`AGENTS.md §3`、`§5`、`§10`–`§12`、`§18.2`、`§19.1`）。
This document provides the complete Member Guide in Traditional Chinese (`zh-TW`, primary) followed by English (`en`, secondary).

---

# 第一部分：繁體中文會員手冊 (`zh-TW`)

## 1. 支援平台與釘選客戶端版本 (`§5.2`, `docs/netbird/VERSIONS.md`)

Freedom VPN v1 採用官方簽章之 **NetBird 客戶端** 搭配 **Freedom VPN 會員入口網站（Member Portal）**（`https://vpn.example.com/member`）。請務必安裝以下經驗證之釘選版本：

| 平台 | 釘選客戶端版本 | 最低作業系統需求 |
|---|---|---|
| **Windows** | `0.82.0` | Windows 10 (22H2+) / Windows 11 (`amd64` / `arm64`) |
| **macOS** | `0.82.0` | macOS 14 Sonoma / macOS 15 Sequoia (Apple Silicon & Intel) |
| **iOS / iPadOS** | `0.4.1` (`build 2`) | iOS / iPadOS 17.0+ |
| **Android** | `0.6.2` | Android 12+ |

---

## 2. 四大平台完整安裝、授權與連線步驟 (`§5.1`, `§5.2`, `§18.2`)

### 2.1 第一步：登入會員入口網站並啟用訂閱 (`/member`, `/member/billing`)

1. 開啟瀏覽器前往 `https://vpn.example.com/member`，點擊 **登入（Sign In）** 並透過您的 OIDC 帳號完成驗證。
2. 若尚未啟用訂閱，請至 **帳單與方案（Billing）** 分頁選擇方案並點擊 **前往結帳（Checkout）**（`POST /api/v1/member/checkout-sessions`）。付款完成後，入口網站狀態會顯示為 **`Active`（有效）**。

### 2.2 第二步：依平台安裝 NetBird 客戶端並登入同一帳號

> **重要提醒**：在任何裝置上啟動 NetBird 後，必須先將 **Management URL（管理伺服器網址）** 設定為 `https://netbird.example.com`，並使用與會員入口網站 **完全相同** 的 OIDC 帳號登入，接著再到會員入口網站 `/member/devices` 點擊 **授權此裝置（Authorize Device）**，並等待狀態由 `Pending（套用中）` 變更為 **`Authorized / Applied（已生效）`**。

#### A. Windows (`0.82.0`)
1. 安裝官方 `netbird-installer_0.82.0_windows_amd64.exe`。
2. 點擊系統列（System Tray）的 NetBird 圖示 -> **Settings（設定）** -> **Advanced（進階）**。
3. 在 **Management URL** 輸入 `https://netbird.example.com` 並儲存（或於 PowerShell 執行 `netbird up --management-url https://netbird.example.com`）。
4. 系統會開啟瀏覽器，請使用您的 Freedom VPN 帳號完成 OIDC 登入。
5. 回到會員入口網站 `https://vpn.example.com/member/devices`，找到剛註冊的 Windows 裝置，點擊 **授權（Authorize）**，等待操作狀態變為 **`Succeeded（已生效）`**。
6. 在系統列 NetBird 圖示點擊 **Connect**，並於 **Exit Node / Networks** 選單勾選欲使用的出口節點（例如 `tw-tpe-01`）。

#### B. macOS (`0.82.0`)
1. 安裝官方 `NetBird_0.82.0_darwin.dmg` 並啟動應用程式（首次啟動時請允許 macOS 新增 VPN 設定檔與系統網路延伸模組）。
2. 點擊頂部選單列（Menu Bar）的 NetBird 圖示 -> **Preferences / Settings** -> **Advanced**。
3. 將 **Management URL** 設為 `https://netbird.example.com` 並儲存，接著點擊 **Connect** 完成瀏覽器 OIDC 登入。
4. 前往會員入口網站 `https://vpn.example.com/member/devices` 授權該 macOS 裝置，等待狀態顯示 **`Authorized（已生效）`**。
5. 於選單列 NetBird 圖示選擇出口節點（Exit Node）即可開始安全連線。

#### C. iOS / iPadOS (`0.4.1 build 2`)
1. 自 App Store 安裝官方 **NetBird** App（`0.4.1 build 2`）。
2. 開啟 App，點擊 **Change Server / Server**，輸入 `https://netbird.example.com` 並點擊儲存（Save）。
3. 點擊中間連線按鈕，於系統彈窗點擊 **Allow（允許新增 VPN 設定）**，並在彈出的瀏覽器視窗完成 OIDC 登入。
4. 開啟 `https://vpn.example.com/member/devices` 授權您的 iPhone/iPad，等待狀態顯示 **`Authorized（已生效）`**。
5. 回到 NetBird App 確認連線已開啟並已套用預設出口節點。

#### D. Android (`0.6.2`)
1. 自 Google Play 安裝官方 **NetBird** App（`0.6.2`）。
2. 開啟 App，點擊右上角齒輪（Server 設定），將 **Management URL** 設為 `https://netbird.example.com` 並儲存。
3. 點擊連線按鈕，核准 Android VPN 連線權限並於瀏覽器完成 OIDC 登入。
4. 前往 `https://vpn.example.com/member/devices` 授權您的 Android 裝置，等待狀態變為 **`Authorized（已生效）`**。
5. **重要 DNS 設定**：請至 Android **設定 -> 網路和網際網路 -> 私人 DNS（Private DNS）**，將其設為 **「關閉（Off）」**（詳見第 5 節說明），以確保 Freedom VPN 的惡意網站過濾正常運作。

---

## 3. 手動驗證腳本 (`AT01`, `AT09`, `AT21`) (`§19.1`, `§19 AT01, AT09, AT21`)

連線後，您可依下列驗收步驟驗證出口轉發、成員隔離、斷線保護（Kill Switch）與 DNS 防護是否正常運作：

### 3.1 `AT01`：驗證出口 IP 切換與成員間完全隔離 (`ISO01`–`ISO04`)

- **macOS / Linux 終端機指令** `[Verified on dev/test stack]`：
  ```bash
  # 1. 檢查外部出口 IP 是否已切換為 Freedom VPN 出口節點公網 IP (例如 192.0.2.20)
  curl -fsS https://vpn.example.com/api/v1/member/me

  # 2. 驗證成員與成員之間無法互連 (以另一台成員裝置的 Overlay IP 100.64.0.11 為例，必須 100% 逾時/封鎖)
  ping -c 3 -W 2 100.64.0.11
  # 預期結果：100.0% packet loss (封包被 NetBird 政策與出口節點 inet fvpn 防火牆雙重丟棄)
  ```
- **Windows PowerShell 指令**：
  ```powershell
  # 驗證另一台成員裝置 Overlay IP 無法連線 (必須顯示 False / 逾時)
  Test-NetConnection -ComputerName 100.64.0.11 -Port 443
  ```

### 3.2 `AT09`：驗證 Fail-Closed 斷線保護（無明文外洩）與自動重連 (`§7.5`, `§8.2`)

1. 在連線狀態下持續執行外部連線測試。
2. 當出口節點切換、租約到期（`86,400` 秒離線上限）或管理員停用該出口時，出口節點之 `inet fvpn` 資料平面與 `fvpn-watchdog` 會在 **30 秒內** 自動關閉轉發（Fail-Closed）。
3. **驗證標準**：在節點停止轉發或隧道重連期間，會員流量會被安全阻斷，**絕不會**退回本地 ISP 明文直連外洩真實 IP；當節點恢復或切換至健康出口後，流量自動恢復。

### 3.3 `AT21`：驗證 DNS 惡意網域攔截（`NXDOMAIN`）與白名單放行 (`§12.1`, `§12.4`)

- **macOS / Linux (`dig` 或 `nslookup`)** `[Verified on dev/test stack]`：
  ```bash
  # 1. 查詢受封鎖之測試惡意網域 -> 必須回傳 NXDOMAIN
  nslookup malware.test.example.invalid 100.64.0.53

  # 2. 查詢正常網域與白名單覆寫網域 -> 必須回傳 NOERROR 與正常 IP
  nslookup example.com 100.64.0.53
  ```
- **Windows PowerShell**：
  ```powershell
  Resolve-DnsName malware.test.example.invalid -Server 100.64.0.53
  # 預期結果：DNS name does not exist (RCODE=3 NXDOMAIN)
  Resolve-DnsName example.com -Server 100.64.0.53
  ```
- **iOS / Android**：
  開啟瀏覽器造訪正常網站（如 `https://example.com`）應正常載入；造訪被封鎖之釣魚/惡意測試網域應立即顯示無法解析網域（`DNS_PROBE_FINISHED_NXDOMAIN`）。

---

## 4. 裝置數量上限與自助管理 (`/member/devices`, `§5.1`, `AT02`, `AT03`)

1. **預設裝置上限**：每位會員預設最多可同時授權 **3 台裝置**（`device_limit = 3`，實際上限依您的訂閱方案或管理員設定顯示於 `/member/devices`）。
2. **達到上限時（`409 DEVICE_LIMIT_REACHED`）**：若您嘗試授權第 4 台裝置，系統會拒絕授權並提示已達裝置上限（`AT02`）。
3. **自助撤銷舊裝置（`AT03`）**：
   - 前往 `https://vpn.example.com/member/devices`。
   - 找到不再使用的舊裝置，點擊 **撤銷（Revoke）**。
   - 等待操作狀態由 `Pending` 變為 **`Revoked / Succeeded`**（系統會立即移除該裝置的出口轉發權限並停用其 NetBird Peer），即可釋出名額授權新裝置。

---

## 5. 用量統計與公平使用政策（FUP）顯示說明 (`/member/usage`, `§10`, `§11`)

在會員入口網站 **用量（Usage）** 分頁（`/member/usage`）中，您可以查看當月與每日的上傳、下載及總流量：

1. **計量口徑（`forward_admission_v1`）**：
   - 僅計算實際通過出口節點轉發驗證（`inet fvpn` forward 路徑）之會員 IPv4/IPv6 流量位元組（`§10.1`）。被防火牆丟棄的封包或控制通道訊號不會計入您的用量。
2. **結算週期與重置時間**：
   - 以 **UTC 日曆月**（每月 1 日 `00:00:00Z` 至下月 1 日 `00:00:00Z`）為統計與 FUP 重置週期（`§11.1`，跨月自動重置為 `Tier 0` 全速，`AT18`）。
3. **公平使用（FUP）降速階層與共享範圍**：
   - 當月累積總流量超過方案門檻時，系統會依階層調降速率上限（`Tier 1`、`Tier 2`...），但**保證不低於可用底線速率（`minimum_rate`）**，不會將已付費會員斷網（`§11.2`、`AT17`）。
   - **同出口共享限速（`§11.3`）**：同一位會員名下的多台裝置若同時連線至**同一台出口節點**，會共享該會員在該出口節點的頻寬上限。
   - 若系統處於 **觀測模式（`shadow` mode）**，頁面會顯示預估階層，但實際連線仍維持全速（`throttled: false`）。
4. **資料品質標記說明（`§10.6`、`§18.2`）**：
   - **`統計截至 (Aggregate as of)`**：中央帳本已完成連續確認（Watermark）的時間點。
   - **`對帳中 (Reconciliation in progress)`**：系統正在進行備份還原後的節點序列重播對帳，數據將於對帳完成後定案。
   - **`含缺口 (Gap)` / `估算 (Estimated)`**：代表部分區間因節點異常斷電或重啟而存在未完整取樣之缺口；系統**絕不會憑空捏造或補猜您的流量位元組**（`§10.6`）。

---

## 6. DNS 安全防護行為、隱私承諾與技術限制 (`§12.1`, `§12.6`, `§12.7`, `SEC02`)

### 6.1 我們提供的保護與零日誌隱私承諾 (`SEC02`)
- **惡意與釣魚網域攔截**：所有連線至出口節點的標準 DNS 查詢（`UDP/53` 與 `TCP/53`）皆由出口節點強制導向內部受保護的 AdGuard Home 解析器（`100.64.0.53`、`100.64.0.54`），針對已知惡意軟體與釣魚網域回傳 `NXDOMAIN`。
- **零 DNS 查詢日誌（`SEC02`）**：Freedom VPN **完全關閉** AdGuard Home 的查詢日誌（`querylog.enabled: false`）與統計紀錄，且中央資料庫與節點日誌**絕不記錄**您查詢過的完整網域名稱、造訪的網址（URL）或封包內容。

### 6.2 您必須了解的技術限制（DoH / DoT 繞過說明，`§12.6`）
我們堅持誠實揭露 DNS 防護的技術邊界，不作誇大宣稱：
1. **瀏覽器加密 DNS（DoH / Secure DNS）與系統私人 DNS 繞過**：
   - 若您的瀏覽器（Chrome、Edge、Firefox、Brave 的「安全 DNS / Use Secure DNS」）或作業系統（Android「私人 DNS / Private DNS」、iOS 第三方加密 DNS 描述檔）啟用了 **DNS-over-HTTPS (DoH)** 或 **DNS-over-TLS (DoT)**，DNS 查詢會被封裝在一般 HTTPS（連接埠 `443`）流量中直接發往外部供應商（如 `8.8.8.8` 或 `1.1.1.1`），從而**繞過** Freedom VPN 的 DNS 惡意網域過濾。
   - **建議設定**：為獲得完整防護，請將 Android「私人 DNS」設為**關閉（Off）**，並將瀏覽器的「安全 DNS」設為**使用目前的服務供應商（系統預設 DNS）**或關閉。
2. **不進行 TLS 解密（No MITM）與網址路徑檢查**：
   - Freedom VPN **絕不攔截或解密您的 HTTPS 憑證**，因此我們只過濾「網域名稱（Domain）」，無法過濾同一網域下的特定網頁路徑（URL Path）或網頁內容。
3. **不攔截純 IP 直連**：
   - 若惡意程式不查詢 DNS 而直接連線至外部 IP 位址，DNS 層過濾不會觸發。

---

## 7. 客服支援與問題回報流程 (`§18.2`)

當您遇到連線異常、帳單問題或認為某個正常網站被誤攔截時：
1. **收集診斷識別碼（請勿提供密碼或私鑰）**：
   - 在會員入口網站（`/member`）發生錯誤時，請複製畫面錯誤訊息中的 **`Request ID`**，或在 `/member/devices` 複製 **`Operation ID`**。
   - **安全警告**：客服人員**永遠不會**向您索取 OIDC 密碼、工作階段 Cookie 或任何私鑰檔案。
2. **回報誤攔截網域（False Positive）或惡意網域**：
   - 請提供完整網域名稱（例如 `sub.example.com`）與發生時間（UTC），管理員可透過 `domain_rule_set` 白名單覆寫（`allow`）在 1 分鐘內（`blocked_ttl_seconds = 60`）為全體會員解除誤封鎖。

---

# Part II: English Member Guide (`en`)

## 1. Supported Platforms and Pinned Client Versions (`§5.2`, `docs/netbird/VERSIONS.md`)

Freedom VPN v1 uses the official signed **NetBird Client** paired with the **Freedom VPN Member Portal** (`https://vpn.example.com/member`). Always install the pinned, verified client version for your operating system:

| Platform | Pinned Client Version | Minimum OS Version |
|---|---|---|
| **Windows** | `0.82.0` | Windows 10 (22H2+) / Windows 11 (`amd64` / `arm64`) |
| **macOS** | `0.82.0` | macOS 14 Sonoma / macOS 15 Sequoia (Apple Silicon & Intel) |
| **iOS / iPadOS** | `0.4.1` (`build 2`) | iOS / iPadOS 17.0+ |
| **Android** | `0.6.2` | Android 12+ |

---

## 2. Step-by-Step Setup Across All Four Platforms (`§5.1`, `§5.2`, `§18.2`)

### 2.1 Step 1: Sign In to the Member Portal and Activate Your Plan (`/member`, `/member/billing`)

1. Open `https://vpn.example.com/member` in your browser, click **Sign In**, and authenticate with your OIDC account.
2. Under **Billing**, select a subscription plan and click **Checkout** (`POST /api/v1/member/checkout-sessions`). Once completed, your entitlement status shows **`Active`**.

### 2.2 Step 2: Install NetBird, Point to the Management URL, and Authorize Your Device

> **Important**: On every platform, set the NetBird **Management URL** to `https://netbird.example.com` and sign in with the **exact same OIDC account** used in the Member Portal. Then open `https://vpn.example.com/member/devices`, click **Authorize Device**, and wait for the status to change from `Pending` to **`Authorized / Applied`**.

#### A. Windows (`0.82.0`)
1. Install `netbird-installer_0.82.0_windows_amd64.exe`.
2. Right-click the NetBird tray icon -> **Settings** -> **Advanced**, set **Management URL** to `https://netbird.example.com`, and save (or run `netbird up --management-url https://netbird.example.com` in PowerShell).
3. Complete the browser OIDC sign-in when prompted.
4. In `https://vpn.example.com/member/devices`, click **Authorize** next to your Windows device and wait for the operation to reach **`Succeeded`**.
5. Click **Connect** in the NetBird tray menu and select your desired **Exit Node** (e.g., `tw-tpe-01`).

#### B. macOS (`0.82.0`)
1. Install `NetBird_0.82.0_darwin.dmg`, launch NetBird, and approve the macOS VPN configuration and system extension prompts.
2. Click the NetBird menu bar icon -> **Preferences / Settings** -> **Advanced**, set **Management URL** to `https://netbird.example.com`, and click **Connect** to complete OIDC sign-in.
3. Authorize your Mac in `https://vpn.example.com/member/devices` and wait for **`Authorized`** status.
4. Select an exit node from the NetBird menu bar icon.

#### C. iOS / iPadOS (`0.4.1 build 2`)
1. Install **NetBird** (`0.4.1 build 2`) from the App Store.
2. Open the app, tap **Change Server / Server**, enter `https://netbird.example.com`, and tap **Save**.
3. Tap the connect button, approve the iOS VPN permission prompt, and sign in with your OIDC account.
4. Open `https://vpn.example.com/member/devices` to authorize your iPhone/iPad and wait for **`Authorized`** status.
5. Return to the NetBird app and verify the tunnel is connected.

#### D. Android (`0.6.2`)
1. Install **NetBird** (`0.6.2`) from Google Play.
2. Tap the gear icon (Server settings), set **Management URL** to `https://netbird.example.com`, and save.
3. Tap Connect, accept the Android VPN prompt, and complete OIDC sign-in.
4. Authorize your Android device at `https://vpn.example.com/member/devices` and wait for **`Authorized`** status.
5. **Important DNS Step**: Go to Android **Settings -> Network & internet -> Private DNS** and set it to **Off** (see Section 6 below) so Freedom VPN's protective DNS filtering takes effect.

---

## 3. Manual Verification Scripts (`AT01`, `AT09`, `AT21`) (`§19.1`, `§19 AT01, AT09, AT21`)

### 3.1 `AT01`: Verify Exit Egress IP and Member-to-Member Isolation (`ISO01`–`ISO04`)

- **macOS / Linux Terminal** `[Verified on dev/test stack]`:
  ```bash
  # 1. Confirm authenticated session and active state
  curl -fsS https://vpn.example.com/api/v1/member/me

  # 2. Verify member-to-member traffic is blocked (replace 100.64.0.11 with another member peer IP)
  ping -c 3 -W 2 100.64.0.11
  # Expected: 100.0% packet loss (blocked by both NetBird policy and exit-node inet fvpn rules)
  ```
- **Windows PowerShell**:
  ```powershell
  Test-NetConnection -ComputerName 100.64.0.11 -Port 443
  # Expected: TcpTestSucceeded : False
  ```

### 3.2 `AT09`: Verify Fail-Closed Kill Switch (No Cleartext Leak) and Reconnect (`§7.5`, `§8.2`)

1. Start a continuous ping or download test while connected through an exit node.
2. If an exit node's serving lease expires (`86,400s` max offline window), its health check fails, or `fvpn-watchdog` detects a stale heartbeat (`> 30s`), the exit node immediately disables member forwarding (`inet fvpn` fail-closed gate).
3. **Verification Criteria**: Traffic stops cleanly rather than leaking unencrypted over your local ISP, and automatically resumes once the exit node recovers or you switch to another published exit node.

### 3.3 `AT21`: Verify DNS Threat Blocking (`NXDOMAIN`) and Allow-List Resolution (`§12.1`, `§12.4`)

- **macOS / Linux (`nslookup` / `dig`)** `[Verified on dev/test stack]`:
  ```bash
  # 1. Query a blocked test domain -> must return NXDOMAIN
  nslookup malware.test.example.invalid 100.64.0.53

  # 2. Query a normal or allow-listed domain -> must return NOERROR and an IP address
  nslookup example.com 100.64.0.53
  ```
- **Windows PowerShell**:
  ```powershell
  Resolve-DnsName malware.test.example.invalid -Server 100.64.0.53
  # Expected: DNS name does not exist (RCODE=3 NXDOMAIN)
  Resolve-DnsName example.com -Server 100.64.0.53
  ```

---

## 4. Device Limits and Self-Service Revocation (`/member/devices`, `§5.1`, `AT02`, `AT03`)

1. **Device Limit**: By default, each member can authorize up to **3 devices** (`device_limit = 3`, configurable per plan).
2. **Limit Reached (`409 DEVICE_LIMIT_REACHED`)**: Attempting to authorize an additional device beyond your limit returns `DEVICE_LIMIT_REACHED` (`AT02`).
3. **Revoking a Device (`AT03`)**:
   - Visit `https://vpn.example.com/member/devices` and click **Revoke** next to any device you wish to remove (`DELETE /api/v1/member/devices/{device_id}`).
   - Once the revocation operation reaches **`Succeeded`**, the old device's tunnel access is terminated and the slot is immediately available for a new device.

---

## 5. Usage Metering and Fair-Use Policy (FUP) Display (`/member/usage`, `§10`, `§11`)

1. **Metering Definition (`forward_admission_v1`)**:
   - Only IPv4/IPv6 bytes admitted into the exit-node forwarding path (`inet fvpn`) are counted (`§10.1`). Dropped packets and control-plane traffic are never billed or counted.
2. **UTC Monthly Reset**:
   - Usage totals and Fair-Use tiers reset on the **1st of each UTC calendar month at `00:00:00Z`** (`§11.1`, `AT18`).
3. **Fair-Use Tiers & Per-Exit Shared Scope**:
   - If your monthly total exceeds a fair-use tier threshold, your rate is shaped to that tier's `down_bps` / `up_bps` ceiling, which is guaranteed to stay at or above a usable `minimum_rate` floor (`> 0 bps`, never a hard cutoff, `§11.2`, `AT17`).
   - All authorized devices belonging to your account that are connected to the **same exit node** share your member rate class on that exit node (`§11.3`).
   - When `shadow` mode is active, the Portal displays your informational tier while `throttled` remains `false`.
4. **Quality Badges (`§10.6`, `§18.2`)**:
   - **`Aggregate as of`**: Timestamp of the latest contiguous usage checkpoint committed from exit nodes.
   - **`Reconciliation in progress`**: Shown temporarily while nodes replay events following a central database restore.
   - **`Gap` / `Estimated`**: Indicates that an exit node experienced an unscheduled reboot or retention gap during the period; Freedom VPN **never invents or fabricates missing byte counts** (`§10.6`).

---

## 6. DNS Behaviour, Privacy Guarantee, and Explicit Limitations (`§12.1`, `§12.6`, `§12.7`, `SEC02`)

### 6.1 Domain-Level Protection and Zero Query Logging (`SEC02`)
- Standard DNS queries (`UDP/53` and `TCP/53`) from member tunnels are redirected by the exit node (`inet fvpn` DNAT) to Freedom VPN's hardened AdGuard Home resolvers (`100.64.0.53`, `100.64.0.54`), returning `NXDOMAIN` for blocked malware and phishing domains.
- **Zero DNS Query Logging (`SEC02`)**: AdGuard Home query logs and statistics are disabled (`querylog.enabled: false`, `statistics.enabled: false`). Freedom VPN never logs or stores the domains you query, the URLs you visit, or packet payloads.

### 6.2 Explicit Technical Limitations (DoH / Private DNS Bypass Stated Clearly, `§12.6`)
1. **Browser DoH ("Secure DNS") and OS Private DNS (DoT/DoH) Bypass**:
   - If your browser (Chrome/Firefox/Edge/Brave "Use Secure DNS") or operating system (Android "Private DNS" or iOS encrypted DNS profiles) sends encrypted DNS over HTTPS (`TCP/443` or `UDP/443`) directly to an external resolver (such as `1.1.1.1` or `8.8.8.8`), those queries bypass port-53 DNS interception and **will not be filtered** by Freedom VPN.
   - **How to Ensure Protection**: Turn **Off** Android "Private DNS" and set browser "Secure DNS" to **Off** or **Use current service provider (system default)**.
2. **No TLS Interception (No MITM) and No URL Path Filtering**:
   - Freedom VPN never intercepts TLS certificates or decrypts HTTPS traffic. Blocking operates strictly at the **domain name** level, not on specific URL paths or page contents.
3. **No Direct-IP Blocking via DNS**:
   - Connections made directly to raw IP addresses without a DNS lookup are not filtered by DNS policies.

---

## 7. Support Contact Flow (`§18.2`)

1. **Include Diagnostic IDs (Never Share Secrets)**:
   - When contacting support about an error in the Member Portal, include the **`Request ID`** shown in the error banner or the **`Operation ID`** from `/member/devices`, along with the approximate UTC timestamp.
   - **Never** send your OIDC password, session cookies, or private keys to support.
2. **Reporting False Positives (Wrongly Blocked Domains)**:
   - Send the exact domain name (e.g., `app.example.com`) to support. Once approved, an administrator adds an `allow` override in `domain_rule_set`, which takes effect within 60 seconds (`blocked_ttl_seconds = 60`).

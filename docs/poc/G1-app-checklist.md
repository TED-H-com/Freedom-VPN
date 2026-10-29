# G1 原生 App 人工驗收表 — 尚未執行 / 待實測

本表不代表平台已支援。每平台都須完成 AT01、AT08、AT09；只看到 route
或登入成功不能判定通過。英文字樣採官方 UI；如果實機不同，記錄實際字樣與
截圖，不猜測成功。不要把帳密、device code、PAT、完整 JWT、私鑰或真實
會員身分放進公開證據。使用 TEST ONLY 帳號與受控測試端點。

## 測試前填寫

| 項目 | 值 / 記錄 |
|---|---|
| 測試日期 UTC、測試員代號 | ____ / ____ |
| Server / Dashboard | 0.82.0 / 2.96.0；digest 依 `docs/netbird/VERSIONS.md` |
| Windows 候選 | daemon/CLI 0.82.0 + 同 release desktop UI；OS build ____；UI 顯示版本 ____ |
| macOS 候選 | daemon/CLI 0.82.0 + 同 release desktop UI；OS build / CPU ____；UI 顯示版本 ____ |
| Android 候選 | app 0.6.2；實際 versionCode ____；core ____；OS / 機型代號 ____ |
| iOS 候選 | app 0.4.1 **build 2 (TestFlight)**；core ____；OS / 機型代號 ____ |
| 管理 URL | `https://netbird.example.invalid` 為文件占位；實際受控 TLS URL 私下保存，證據使用此占位 |
| 共用 OIDC issuer / client scope 設計 | 證據參照 ____；Portal client ____；NetBird client ____；MFA 強制設定 ____ |
| 兩個獨立出口 | TEST-exit-A / TEST-exit-B；各自 route、node、incarnation 代號 ____ |
| 會員與群組 | TEST-member-A / TEST-member-B；A 可用兩出口，B 不可用 A 專屬出口 |
| Resolver | 兩個已驗證 resolver 代號 ____ / ____；政策版本 ____ |
| 受控資料端點 | TEST-IP-echo、TCP、UDP、QUIC、DNS fixture 代號 ____；禁止使用第三方 IP 查詢網站 |
| 網路情境 | 受控 Wi-Fi 1 / Wi-Fi 2、行動網路、雙 NAT、強制 relay fixture 代號 ____ |

此 repo 的 loopback HTTP dev stack 只能供本機機器測試。手機不能把
`127.0.0.1` 當遠端伺服器。真人測試須另有受控、可達、憑證受信任的 TLS
測試環境與 G2 enforcement；不要把 dev compose 改綁 `0.0.0.0`。
若商店無法安裝上列候選版本（尤其 iOS TestFlight），記錄「blocked」與
原因，先更新版本矩陣再測。不能將不同 mobile core 當成 Linux 0.82.0。

先由管理員確認：Default All-to-All 已移除、兩個 exit 各只有單一 node peer、
`skip_auto_apply=true`、`masquerade=false`、G2 SNAT/隔離有效、新 user 無 paid
auto-group、DNS 只發布給已授權 device、管理員 MFA 由 IdP 強制。

## Windows — 0.82.0

| 步驟 / 畫面與設定 | 預期 | 觀察 / 截圖檔名 |
|---|---|---|
| W1 安裝固定版本；開啟 NetBird → Settings → About。記錄 UI 與 daemon/CLI 版本。 | 與本表相符；不能讓自動更新換版。 | ____ / W1-____ |
| W2 首次 Welcome 選 **Self-hosted**，輸入完整管理 URL。若有舊 profile，Settings → Profiles 建立獨立測試 profile 並選取。 | 管理 URL 與受控 TLS endpoint 完全一致；無憑證錯誤。 | ____ / W2-____ |
| W3 Main window / tray → **Connect**；瀏覽器確認 issuer，登入 TEST-member-A，完成既定 MFA。 | 返回 App 顯示 Connected 與 overlay IP；第一次 device 尚無出口。 | ____ / W3-____ |
| W4 在控制服務核對 NetBird user/peer → member/device。檢查不同 OAuth client 的 subject 證據。完成名額檢查後授予此 device 的 paid groups。 | 身分映射用 issuer、scope、subject；不能用 email join。只有這個 device 被授權。 | ____ / W4-____ |
| W5 主視窗 **Exit Nodes** selector 或 tray exit selector。先記錄未選取，再選 TEST-exit-A。 | A/B 出口可見但不自動套用；選 A 後受控 echo 顯示 A 出口，記錄外部位址的文件代號。 | ____ / W5-____ |
| W6 同 selector 切到 TEST-exit-B，再取消選擇回 direct。 | 每次 selected exit 與實際 egress 一致；記錄切換延遲及既有 TCP 影響。 | ____ / W6-____ |
| W7 Settings → Network 記錄 DNS/routes/IPv6；Security 記錄 inbound/LAN；General 記錄 startup/Stay Connected After Quitting。 | 不啟用 SSH/remote jobs；依核准 IPv4 模式測試，不靠設定名稱宣稱無 leak。 | ____ / W7-____ |
| W8 執行下方共用矩陣全部 C1–C10。 | 每項有證據與結果；未通過則 Windows 不發布。 | ____ / W8-____ |

## macOS — 0.82.0

| 步驟 / 畫面與設定 | 預期 | 觀察 / 截圖檔名 |
|---|---|---|
| M1 安裝固定版本，允許 macOS 要求的 NetBird 背景服務/網路權限；menu bar NetBird → Settings → About。 | UI、daemon/CLI 版本相符；記錄 Intel/Apple Silicon。 | ____ / M1-____ |
| M2 Welcome → **Self-hosted**，輸入管理 URL；舊設定使用 Settings → Profiles 的獨立測試 profile。 | TLS URL 正確，無跳回 cloud。 | ____ / M2-____ |
| M3 Main window / menu bar → **Connect**，在瀏覽器完成 TEST-member-A 登入與既定 MFA。 | Connected；正確 user/peer；授權前無出口。 | ____ / M3-____ |
| M4 在中央核對 issuer/client-scoped identity proof 和 device，原子完成名額檢查並授權。 | 不能用相同 email 當證明；別的 device 不因登入而獲權。 | ____ / M4-____ |
| M5 主視窗或 menu bar **Exit Nodes** 選 A，再 B；對受控 echo 驗證每次出口。 | 初始未自動選取；名稱、selected exit、egress 一致。 | ____ / M5-____ |
| M6 selector 取消 exit 回 direct；Settings → Network / Security / General 記錄 DNS、IPv6 與關閉 UI 後行為。 | 不把關閉視窗當斷線；不把 server 隔離當 endpoint kill switch。 | ____ / M6-____ |
| M7 關蓋睡眠 2 分鐘、喚醒，再切另一受控 Wi-Fi；記錄每次 UI、route、DNS 與 echo。 | 有明確恢復/失敗結果，沒有未授權出口恢復。 | ____ / M7-____ |
| M8 執行 C1–C10 全部。 | 未完成或失敗時 macOS 不發布。 | ____ / M8-____ |

## iOS — app 0.4.1 (2)

下列標籤來自固定版 Swift UI source；官方安裝頁沒有完整操作圖，所以每張
實機畫面仍為待實測，不能把 source review 當成實機驗收。

| 步驟 / 畫面與設定 | 預期 | 觀察 / 截圖檔名 |
|---|---|---|
| I1 從授權的 TestFlight 測試取得 0.4.1 (2)；App → **Settings → About**。 | 記錄 Version、Core 與 build；若版本不可取得標 blocked。 | ____ / I1-____ |
| I2 **Settings → Change Server**，讀取清除舊 config 的提示並 Confirm。**Server** 欄輸入管理 URL；不填 setup key；按 **Change**，再 **OK**。 | 顯示 Server was changed；網址為受控 TLS URL。 | ____ / I2-____ |
| I3 **Connection** 頁按 Connect/NetBird 連線按鈕；允許 iOS 新增 VPN configuration，完成系統認證；允許開啟登入頁。 | VPN 權限已授予；登入頁 issuer 正確。 | ____ / I3-____ |
| I4 瀏覽器完成 TEST-member-A 登入與既定 MFA，回 Connection；中央核對 user/device 與跨 client 身分證據，再授權此 device。 | 授權前沒有 exit；不得以 setup-key member 替代。 | ____ / I4-____ |
| I5 Connection 頁 **Exit node** 卡片，先記錄 None/未選取，再選 A；對受控 echo 驗證。 | A/B 出現在選單，Auto Apply off；只在手動選後改 egress。 | ____ / I5-____ |
| I6 同卡片選 B，再選 **None**；Resources 頁核對收到的路由。 | B egress 正確，None 回原路徑；記錄實際顯示的出口名稱。 | ____ / I6-____ |
| I7 **Settings → VPN On Demand** 記錄規則（基線先關閉），**Advanced** 記錄 DNS/routes/IPv6。鎖屏 2 分鐘，Wi-Fi 切行動網路再回 Wi-Fi。 | 恢復結果有證據；On Demand 不能當完整 kill switch 保證。 | ____ / I7-____ |
| I8 執行 C1–C10 全部。 | 未完成或失敗時 iOS 不發布。 | ____ / I8-____ |

## Android — app 0.6.2

固定版 source 使用 Profiles；較舊官方安裝文章的 hamburger → Change Server
與此候選 UI 不完全相同。以此版本的實機畫面為準並保存差異。

| 步驟 / 畫面與設定 | 預期 | 觀察 / 截圖檔名 |
|---|---|---|
| A1 安裝官方固定 APK/對應商店版本；NetBird **Settings → About**。 | Version 0.6.2；記錄 versionCode、core、OS 與廠牌設定差異。 | ____ / A1-____ |
| A2 **Settings → Profiles**（或 Home profile selector → Manage Profiles）→ Add Profile；Profile Name 填 TEST ONLY。**Management Server → Self-hosted**，填管理 URL，儲存並切到此 profile。 | URL 可達，不能用 Save anyway 掩蓋 TLS/連線錯誤；不勾 Add this device with a setup key。 | ____ / A2-____ |
| A3 Home 按 Connect/NetBird 連線按鈕；Android **Connection request → OK**；瀏覽器完成 TEST-member-A 登入與既定 MFA。 | VPN 圖示、Connected、正確 user/peer。 | ____ / A3-____ |
| A4 中央核對 issuer/scope/subject mapping 和 device；完成名額檢查後授權。 | 授權前沒有出口；帳號登入不自動給整個 user paid group。 | ____ / A4-____ |
| A5 Home **Exit node → A**；觀察受控 echo；再選 B，再選 **None**。 | 初始 None selected；A/B egress 各自正確，None 回原路徑。 | ____ / A5-____ |
| A6 Android 系統 **Settings → Network & internet → Private DNS**（廠牌路徑記錄 ____）。先記錄原值；選 **Off** 作服務 DNS 基線。 | 受控查詢到已授權 resolver；不能只看 App Connected 判斷 DNS。 | ____ / A6-____ |
| A7 在受控測試網路設定 **Private DNS provider hostname** 指向測試者自有 DoT fixture，再測 DNS 路徑；測後恢復 Off。 | 明確記錄可能繞過 VPN DNS；UI/指引揭露限制，不宣稱可阻止所有 DoH/DoT。 | ____ / A7-____ |
| A8 鎖屏/省電 2 分鐘，Wi-Fi 切行動網路再回 Wi-Fi；記錄電池最佳化設定。執行 C1–C10 全部。 | 逐情境有恢復/失敗證據；未通過時 Android 不發布。 | ____ / A8-____ |

## 共用矩陣 — 每平台各做一次

每格填 `PASS/FAIL/BLOCKED；觀察；截圖/trace 代號`。公開證據只用文件位址
與測試代號。先保留私有原始證據，再人工去敏；不要上傳未去敏 debug bundle。

| 案例 / 步驟 | 預期與需記錄證據 | Windows | macOS | iOS | Android |
|---|---|---|---|---|---|
| C1 第二會員 B 登入，嘗試看/選 A 專屬出口 | B 不可見/不可轉送；user/peer/device 對照正確。 | ____ | ____ | ____ | ____ |
| C2 改 exit 顯示名稱與分配群組 | 記錄原生 App 顯示的實際名稱/更新時效；舊組移除、新組可用；不把穩定 machine label 當友善名稱驗收成功。 | ____ | ____ | ____ | ____ |
| C3 已選 A 時 WithdrawExit，再重啟 App / node | route 消失；G2 轉送實際停止；重啟不自行上架。記錄 API read-back、node ACK、既有 TCP/UDP 結果。 | ____ | ____ | ____ | ____ |
| C4 選 A 後撤銷 device paid group，再到期/停權 | route 與 DNS 授權撤除；既有流停止時效受租約約束；本機 lease/ACK 證據不可缺。 | ____ | ____ | ____ | ____ |
| C5 中央撤銷 device / delete peer；登出，再以有效新授權重新加入 | 舊身分不能恢復；新 peer/incarnation 依正式流程映射。裝置名額競爭另驗 AT02。 | ____ | ____ | ____ | ____ |
| C6 睡眠/喚醒與兩種網路切換 | 記錄每次 selected exit、connection type、恢復秒數、DNS/echo；不得假定選取偏好消失。 | ____ | ____ | ____ | ____ |
| C7 對兩個 exit 分別量 direct 與強制 relay | 逐對記錄 P2P/Relayed；切斷 relay、signal/management 分別記錄；不可用節點永久標籤代表路徑。 | ____ | ____ | ____ | ____ |
| C8 雙 NAT、大封包、TCP、UDP、QUIC、PMTU/MSS | 只用受控 endpoint；記錄 underlay、packet size、丟包、MTU、吞吐。不得採未測全域 MTU。 | ____ | ____ | ____ | ____ |
| C9 IPv4 服務 + IPv6 endpoint probes；斷隧道/失效出口 | 驗證核准 IPv6 阻擋模式；記錄原 ISP 回落/失敗，不能宣稱 NetBird 有完整 always-on kill switch。 | ____ | ____ | ____ | ____ |
| C10 DNS resolver A 故障、B 故障、變更 exit；復原 | 只切受控 resolver；無未核准公共 DNS fallback；記錄主從/並行實際行為及來源 device 辨識。 | ____ | ____ | ____ | ____ |

結論：Windows ____；macOS ____；iOS ____；Android ____。
阻擋缺陷與追蹤代號 ____。審核日期 ____；G1 可否關閉 ____。

來源：[desktop](https://docs.netbird.io/client/desktop-app)、
[Android install](https://docs.netbird.io/get-started/install/android)、
[Android v0.6.2 UI strings](https://github.com/netbirdio/android-client/blob/v0.6.2/app/src/main/res/values/strings.xml)、
[iOS v0.4.1 settings](https://github.com/netbirdio/ios-client/blob/v0.4.1/NetBird/Source/App/Views/iOS/iOSSettingsView.swift)、
[Private DNS limitation](https://docs.netbird.io/manage/dns/troubleshooting)。

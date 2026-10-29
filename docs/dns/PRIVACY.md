# DNS Privacy, Data Minimization & Member Disclosures (§12.1, §12.6, §12.7, SEC02, DNS04, AT22)

## 1. Data Minimization Inventory (§12.7, SEC02)

Freedom VPN enforces strict DNS data minimization across both the central control plane (`internal/dnspolicy`) and all filtering resolver instances (`AdGuard Home`).

### 1.1 Data NEVER Collected, Logged, or Persisted

| Prohibited Data Category | Enforcement Mechanism | Verification (`AT22`) |
|---|---|---|
| **Individual DNS Query Logs** (queried domain name, QTYPE, timestamp, client IP, or member ID) | `querylog.enabled: false`, `querylog.file_enabled: false`, `querylog.interval: 0` in `AdGuardHome.yaml` and enforced on every `Publish()` via `PUT /control/querylog_config`. | Verified via `GET /control/querylog` (`data: []`) and container filesystem grep showing no `querylog.json` or queried domain strings. |
| **Per-Domain or Per-Client Query Statistics** (top queried domains, top blocked domains, top clients) | `statistics.enabled: false`, `statistics.interval: 0` in `AdGuardHome.yaml` and enforced on every `Publish()` via `PUT /control/stats_config`. | Verified via `GET /control/stats` returning empty arrays and zeroed counters. |
| **Verbose Resolver Stdout/File Logs Containing Query Names** | `log.verbose: false`, `log.file: ""` in `AdGuardHome.yaml`. | Verified by inspecting `docker logs` after active test queries to confirm zero occurrences of queried sentinel domains. |
| **Client Runtime Discovery** (RDNS, ARP, WHOIS, DHCP, `/etc/hosts` client correlation) | `clients.runtime_sources.{whois,arp,rdns,dhcp,hosts}: false`. | Prevents resolver background probing or reverse-mapping overlay IPs to hostnames. |
| **HTTPS URLs, Paths, Headers, or Packet Payloads** | No TLS interception (`SCOPE01`); DNS resolvers only see port 53 domain name queries. | Architectural guarantee. |

### 1.2 Data Stored and Why

| Stored Artifact | Location | Retention & Purpose | Contains Member Query History? |
|---|---|---|---|
| **DNS Policy Profiles & Custom Rules** (`rule_id`, `domain`, `include_subdomains`, `action`, `reason`, `revision`) | Central control plane database & compiled bundle | Retained while active/versioned to compile deterministic filtering policies and support admin audit trails (`§12.4`). | **No** (contains only administrator-configured rules). |
| **List Source Metadata & Quality Metrics** (`source_id`, `version`, `content_sha256`, `rule_count`, `delta`, `parse_errors`, `last_update`, `smoke_test_passed`) | Central control plane & worker | Used to verify threat feed integrity, detect broken upstream feeds, and retain `lastGood` on failure (`§12.3`). | **No** (aggregate feed counters only). |
| **Compiled Resolver `user_rules` & `AdGuardHome.yaml`** | Resolver `/opt/adguardhome/conf/` | Required by the resolver daemon to enforce active block/allow policies and survive container restarts (`§12.5`). | **No** (contains only static policy rules and system config). |
| **Resolver Operational Health & Revision Status** (`applied_revision`, `smoke_passed`, `last_error`) | Control plane / worker health status | Proves whether all resolvers have converged on the desired policy revision (`all_applied`, `partial`, `failed`) (`§12.5`, `§16.4`). | **No** (smoke tests only query synthetic policy test domains). |
| **Ephemeral In-Memory DNS Response Cache** (`dns.cache_size: 4194304`) | Volatile RAM inside `AdGuardHome` process only | Caches upstream DNS responses strictly in memory for the duration of the record TTL (blocked responses use `blocked_response_ttl: 60` seconds) to prevent upstream amplification and reduce resolution latency. | **Disk: Never.** Unassociated with client IPs, never written to `/opt/adguardhome/work` or `/opt/adguardhome/conf`, not exposed via HTTP APIs when `querylog` and `statistics` are disabled, and wiped completely on process restart or rule refresh. |

---

## 2. Member-Facing Limitation Text in Traditional Chinese (`zh-TW`) (§12.1, §12.6, DNS04)

The following `zh-TW` copy accurately explains the technical boundaries of DNS domain protection per **§12.1**, **§12.6**, and **DNS04**, without exaggerating capabilities. It is ready for direct inclusion in `apps/portal` (Member & Admin UI).

### 2.1 Full Disclosure Copy (`zh-TW`)

> ### 網域安全防護說明與技術限制
>
> Freedom VPN 提供**網域層級（DNS）安全防護**，可協助攔截已知的惡意軟體、網路釣魚網域，並套用管理員設定的網域封鎖與放行規則。為確保您的連線隱私與安全，請了解以下防護範圍與技術限制：
>
> 1. **僅限網域層級防護，不進行 HTTPS 解密或內容審查**
>    本服務僅在「網域名稱解析（DNS）」階段判斷是否放行或封鎖整個網域（例如 `example.com` 或 `sub.example.com`）。我們**絕不**安裝憑證進行 TLS/HTTPS 中間人解密（MITM），因此無法檢查或過濾加密連線內的特定網址路徑（例如無法允許 `https://example.com` 卻單獨封鎖 `https://example.com/path`）、網頁內容、標頭或傳輸檔案。
>
> 2. **無法攔截直接使用 IP 位址的連線**
>    DNS 防護僅針對透過網域名稱發起的連線生效。若應用程式或惡意程式不經過 DNS 查詢，而是直接連線至目標的 IPv4 或 IPv6 位址（例如直接存取 `http://203.0.113.10`），DNS 過濾器將不會介入（但仍受 VPN 節點防火牆與會員隔離規則保護）。
>
> 3. **瀏覽器加密 DNS（DoH / DoT）與系統「私人 DNS」的影響**
>    - 若您的網頁瀏覽器（如 Chrome、Firefox、Edge、Brave）開啟了內建的「安全 DNS（DNS-over-HTTPS, DoH）」，並指向外部公開解析器（如 Cloudflare `1.1.1.1` 或 Google `8.8.8.8`），瀏覽器可能會繞過 VPN 指派的防護 DNS，或因節點防火牆封鎖外部 DNS 而導致無法解析。
>    - 若您的 Android 裝置於系統設定中將「私人 DNS（Private DNS / DNS-over-TLS）」設為特定外部主機名稱（例如 `dns.google`），系統 DNS 查詢將不會使用 VPN 的過濾規則，或可能無法連線。
>    - **建議設定**：使用 Freedom VPN 期間，請將系統「私人 DNS」設為「自動」或「關閉」，並於瀏覽器中停用自訂的外部安全 DNS（DoH），以確保網域安全防護正常運作。
>
> 4. **零查詢紀錄承諾（資料最小化）**
>    Freedom VPN 的 DNS 解析器已永久關閉查詢紀錄（Query Log）與網域統計功能。我們**不會**記錄、儲存或分析任何會員查詢過的網域名稱與瀏覽歷史。

---

### 2.2 Portal-Ready i18n Resource Snippet (`zh-TW`)

```json
{
  "dnsPolicy.disclosure.title": "網域安全防護說明與技術限制",
  "dnsPolicy.disclosure.summary": "Freedom VPN 提供網域層級（DNS）安全防護，可攔截已知的惡意與釣魚網域。我們不進行 HTTPS 解密，亦不記錄任何會員的 DNS 查詢歷史。",
  "dnsPolicy.disclosure.domainOnly.title": "僅限網域層級，不解密 HTTPS 網址路徑或內容",
  "dnsPolicy.disclosure.domainOnly.body": "防護僅作用於網域名稱（如 example.com），無法針對單一 HTTPS 網址路徑（如 https://example.com/path）或加密網頁內容進行過濾。",
  "dnsPolicy.disclosure.directIp.title": "不包含直接 IP 連線",
  "dnsPolicy.disclosure.directIp.body": "若應用程式略過 DNS 查詢並直接連線至目標 IP 位址，DNS 網域過濾將不會觸發。",
  "dnsPolicy.disclosure.dohPrivateDns.title": "瀏覽器安全 DNS（DoH）與 Android 私人 DNS 設定提醒",
  "dnsPolicy.disclosure.dohPrivateDns.body": "若瀏覽器啟用外部 DoH（DNS-over-HTTPS）或 Android 開啟自訂「私人 DNS（DoT）」，可能會繞過本服務的 DNS 防護或導致無法解析。建議將系統私人 DNS 設為「自動／關閉」並使用 VPN 預設 DNS。",
  "dnsPolicy.disclosure.privacy.title": "不保存任何 DNS 查詢紀錄",
  "dnsPolicy.disclosure.privacy.body": "解析器已關閉查詢日誌（Query Log）與統計功能，不會保存或追蹤您查詢過的任何網域名稱。"
}
```

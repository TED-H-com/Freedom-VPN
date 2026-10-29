# AdGuard Home Filtering Backend (§12.2, §12.5, §12.7)

## 1. Version Pin & License

### 1.1 Container Image Pin

| Component | Version | Image Digest |
|---|---|---|
| **AdGuard Home** | `v0.107.59` | `docker.io/adguard/adguardhome:v0.107.59@sha256:5d5e3aef39a8c8906a324f0c2da0a6123929e459b57c1e3f05e401dca5c7bf02` |
| **CoreDNS (Dev Upstream)** | `1.12.0` | `docker.io/coredns/coredns:1.12.0@sha256:40384aa1f5ea6bfdc77997d243aec73da05f27aed0c5e9d65bfa98933c519d97` |

Configuration schema version: `schema_version: 29`.

### 1.2 License Note

- **AdGuard Home** is licensed under the **GNU General Public License v3.0 (GPL-3.0)** ([AdGuardTeam/AdGuardHome LICENSE.txt](https://github.com/AdGuardTeam/AdGuardHome/blob/v0.107.59/LICENSE.txt)).
- Freedom VPN interacts with AdGuard Home exclusively over the network as an isolated out-of-process container/daemon via standard HTTP/JSON control APIs and UDP/TCP port 53 DNS queries. No AdGuard Home source code or GPL libraries are linked into `fvpn-control`, `fvpn-worker`, or `fvpn-agent`.
- **CoreDNS** (used in `deploy/dev/adguard` as the local authoritative upstream test server) is licensed under the **Apache License 2.0**.

---

## 2. Rule Syntax & Precedence Mapping

Freedom VPN compiles the 4-layer policy model (`internal/dnspolicy`) into AdGuard Home `user_rules` syntax ([S17]).

| Policy Layer | `action` | `include_subdomains` | Rendered AdGuard Rule | Semantics in `urlfilter` Engine |
|---|---|---|---|---|
| **Layer 1: `admin_block`** | `block` | `true` | `\|\|<domain>^$important` | Blocks `<domain>` and all subdomains `*.<domain>`. `$important` modifier overrides any `@@` exception rule. |
| **Layer 1: `admin_block`** | `block` | `false` | `\|<domain>^$important` | Single pipe `\|` anchors to the exact start of the hostname, matching `<domain>` only (not `sub.<domain>`). `$important` overrides `@@`. |
| **Layer 2: `admin_allow`** | `allow` | `true` | `@@\|\|<domain>^` | Unblocks `<domain>` and all subdomains `*.<domain>` against Layer 3 (`source_list`) rules, but yields to `$important` rules. |
| **Layer 2: `admin_allow`** | `allow` | `false` | `@@\|<domain>^` | Unblocks the exact hostname `<domain>` only against Layer 3 (`source_list`) rules, leaving `sub.<domain>` blocked if matched by Layer 3. |
| **Layer 3: `source_list`** | `block` | `true` | `\|\|<domain>^` | Standard threat/category feed block covering `<domain>` and all subdomains `*.<domain>`. |
| **Layer 3: `source_list`** | `block` | `false` | `\|<domain>^` | Standard threat/category feed block covering the exact hostname `<domain>` only (`hosts` file entries). |
| **Layer 4: `normal_resolution`** | `resolve` | N/A | *(No rule emitted)* | Unmatched queries resolve normally via configured upstreams. |

### 2.1 Why `$important` and Single-Pipe `|` Match `Evaluate` Exactly

In AdGuard Home's `urlfilter` DNS engine:
1. Rules carrying `$important` (`||domain^$important` or `|domain^$important`) have the highest priority in the matcher and beat exception (`@@`) rules.
2. Exception rules (`@@||domain^` or `@@|domain^`) have higher priority than normal blocking rules (`||domain^` or `|domain^`), unless the blocking rule has `$important`.
3. `||domain^` matches at any domain label boundary (`domain` and `*.domain`), whereas `|domain^` anchors the match to the beginning of the query hostname (`domain` only) and `^` terminates the hostname match at the end of the domain name.
4. When a query returns a `CNAME` record pointing to another domain, AdGuard Home evaluates the `CNAME` target against the same rule set and returns `NXDOMAIN` if the target domain is blocked (`AT20`).
5. When a domain is blocked with `blocking_mode: nxdomain`, AdGuard Home returns `RCODE=NXDOMAIN` for **all** DNS query types (`A`, `AAAA`, `HTTPS`, `SVCB`, `TXT`), ensuring no IPv6 or `HTTPS`/`SVCB` bypass path remains open (`AT20`).

### 2.2 Revision Metadata Header

Each compiled bundle embeds a deterministic metadata header at the top of `user_rules`:
```text
! fvpn-bundle-profile: <profile_id>
! fvpn-bundle-revision: <revision>
```
`AdGuardAdapter` parses `! fvpn-bundle-revision:` from `GET /control/filtering/status` to provide idempotent publishing and revision verification across resolvers.

---

## 3. Hardened Resolver Settings (§12.5, §12.7)

| Setting | Value | Spec Reference & Purpose |
|---|---|---|
| `bind_host` | `127.0.0.1` (dev) / Management IP (prod) | Restricts Admin UI and HTTP control API to the management plane (`§12.5`). |
| `dns.blocking_mode` | `nxdomain` | Returns standard `NXDOMAIN` response for blocked domains (`§12.5`). |
| `dns.blocked_response_ttl` | `60` (configurable, default 60s) | Short TTL so false-positive fixes (`admin_allow`) take effect rapidly without long client cache retention (`§12.5`). |
| `dns.upstream_dns` | Explicit upstream list only | Never falls back to unconfigured public resolvers (`§12.5`, `AT21`). |
| `dns.bootstrap_dns` | Explicit upstream IPs only | Prevents bootstrap leakage to public DNS. |
| `dns.FallbackDNS` / `fallback_dns` | `[]` (empty) | Disables unfiltered public DNS fallback on upstream timeout (`AT21`). |
| `dns.allowed_clients` | Overlay & local CIDR ranges (`100.64.0.0/10`, `127.0.0.0/8`, etc.) | Restricts recursive query access to authorized clients (`§12.5`). |
| `querylog.enabled` | `false` | Disables in-memory and API query logging (`§12.7`, `SEC02`, `AT22`). |
| `querylog.file_enabled` | `false` | Prevents creation or writing of `querylog.json` on disk (`§12.7`, `SEC02`, `AT22`). |
| `statistics.enabled` | `false` | Disables per-domain and per-client statistical counters (`§12.7`, `SEC02`, `AT22`). |
| `clients.runtime_sources.*` | `false` (`whois`, `arp`, `rdns`, `dhcp`, `hosts`) | Disables background client discovery and reverse DNS probing. |
| `log.verbose` | `false` | Prevents DNS query names from being logged to container stdout/stderr (`SEC02`). |

---

## 4. Control API Endpoints Used by `AdGuardAdapter`

| Endpoint | Method | Purpose |
|---|---|---|
| `/control/status` | `GET` | Health check and version inspection. |
| `/control/querylog_info` | `GET` | Read-back verification that `enabled: false`. |
| `/control/querylog_config` | `PUT` | Enforces `{"enabled": false, "anonymize_client_ip": true, "interval": 0}`. |
| `/control/querylog` | `GET` | Audited in integration tests to prove `data: []`. |
| `/control/stats_info` | `GET` | Read-back verification that `enabled: false`. |
| `/control/stats_config` | `PUT` | Enforces `{"enabled": false, "interval": 0}`. |
| `/control/stats` | `GET` | Audited in integration tests to prove zero top-queried/blocked domain records. |
| `/control/dns_info` | `GET` | Reads `blocking_mode` and `blocked_response_ttl`. |
| `/control/dns_config` | `POST` | Enforces `{"blocking_mode": "nxdomain", "blocked_response_ttl": 60}`. |
| `/control/filtering/status` | `GET` | Reads active `user_rules` and extracts `! fvpn-bundle-revision:`. |
| `/control/filtering/config` | `POST` | Enforces `{"enabled": true, "interval": 0}`. |
| `/control/filtering/set_rules` | `POST` | Atomically replaces `user_rules` with the compiled bundle (or `lastGood` on rollback). |
| `/control/filtering/refresh` | `POST` | Forces immediate reload of filter structures. |
| `/control/filtering/check_host` | `GET` | Smoke-tests candidate domains against the active `urlfilter` engine before marking publish complete. |

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

Layer 3 in the running publisher is sourced from `dns_feed_refresh_state.last_good_content` joined to immutable `dns_feed_contents`, not from a policy’s URL or a resolver-managed external list. D-054 bootstrap approval, bounded refresh and failure retention happen on the control plane. Resolver documents embed the normalized set and raw content hash; the enclosing canonical document hash covers every entry. Missing last-good rejects publication. Feed entries use the shared local-name isolation baseline, and include-subdomain entries cannot cover a public/private suffix. Only active, unexpired feed revisions are eligible; supersession does not repoint pinned profile revisions. See [FEEDS.md](FEEDS.md) and [PUBLISH.md](PUBLISH.md).

### 2.1 Why `$important` and Single-Pipe `|` Match `Evaluate` Exactly

In AdGuard Home's `urlfilter` DNS engine:
1. Rules carrying `$important` (`||domain^$important` or `|domain^$important`) have the highest priority in the matcher and beat exception (`@@`) rules.
2. Exception rules (`@@||domain^` or `@@|domain^`) have higher priority than normal blocking rules (`||domain^` or `|domain^`), unless the blocking rule has `$important`.
3. `||domain^` matches at any domain label boundary (`domain` and `*.domain`), whereas `|domain^` anchors the match to the beginning of the query hostname (`domain` only) and `^` terminates the hostname match at the end of the domain name.
4. Response filtering checks CNAME targets present in the upstream answer. A/AAAA/HTTPS queries whose returned chain reaches a blocked target yield NXDOMAIN. A CNAME query exposing that target yields NODATA. A query returning only an earlier, unblocked hop can still return that CNAME; querying the next hop stops at the blocked target. See the measured chain cases below (`AT20`, §12.4).
5. With `blocking_mode: nxdomain`, the pinned version returns NXDOMAIN for A/AAAA/HTTPS, but NODATA (NOERROR, empty answer, SOA authority) for CNAME/SVCB/TXT/MX/NS when filtering matches. ANY is rejected before filtering when `refuse_any: true`. The measured invariant is fail-closed for answers: no address answer, no SVCB/HTTPS target or hints, and no CNAME traversal completing to an address through the blocked name. This is not a promise of uniform NXDOMAIN, nor a claim about arbitrary untested response structures. Use the per-type rows below (§12.4–§12.5).

### 2.1.1 Expected responses — measured 2026-10-31

The exact §1.1 image was driven through **`node/dns.New(...).Apply`**, including read-back and its A smoke test. No replacement adapter was used. The [retained evidence](evidence/2026-10-31-response-shape/PROCEDURE.md) includes the actual `/control/dns_config` and `/control/filtering/set_rules` bodies, initial YAML, canonical-document contents, read-back, and 432 raw UDP/TCP probe exchanges. The selected setting is `blocking_mode: nxdomain`, `blocked_response_ttl: 60`; `refuse_any: true` is an initial resolver setting that the applier does not write. DNSSEC validation is disabled by the production body. Requests set RD and AD; the filtered responses clear AA and AD.

Scopes below are synthetic names, each verified to have upstream data before blocking:

- **D** — direct exact block: `blocked.example.com`, rule `|blocked.example.com^$important`.
- **W** — wildcard scope: `child.wild.example.com`, rule `||wild.example.com^$important` (`include_subdomains: true`, not a literal `*` in the product domain).
- **C** — chain: `chain.example.com → hop.example.com → blocked.example.com → target.example.com`. Only `blocked.example.com` is blocked.
- **H** — follow-on hop: `hop.example.com`, whose CNAME answer exposes the blocked target. This proves that the CNAME returned in row F8-CNAME-C cannot complete traversal to an address.

Both transports gave the following **fresh** replies. `0 / —` means zero answers of any type. A SOA entry is the sole authority RR unless stated otherwise. Row IDs are stable citation targets for acceptance reports.

| Row | QTYPE | Scope | RCODE | Answer count / types | Authority SOA / TTL | AA / AD |
|---|---|---|---|---|---|---|
| F8-A | A (1) | D, W, C, H | 3 NXDOMAIN | 0 / — | present / 60 | 0 / 0 |
| F8-AAAA | AAAA (28) | D, W, C, H | 3 NXDOMAIN | 0 / — | present / 60 | 0 / 0 |
| F8-CNAME | CNAME (5) | D, W, H | 0 NOERROR (NODATA) | 0 / — | present / 60 | 0 / 0 |
| F8-CNAME-C | CNAME (5) | C | 0 NOERROR | 1 / CNAME to `hop.example.com`, TTL 300 | absent; one upstream NS authority RR | 1 / 0 |
| F8-SVCB | SVCB (64) | D, W, C, H | 0 NOERROR (NODATA) | 0 / — | present / 60 | 0 / 0 |
| F8-HTTPS | HTTPS (65) | D, W, C, H | 3 NXDOMAIN | 0 / — | present / 60 | 0 / 0 |
| F8-TXT | TXT (16) | D, W, C, H | 0 NOERROR (NODATA) | 0 / — | present / 60 | 0 / 0 |
| F8-MX | MX (15) | D, W, C, H | 0 NOERROR (NODATA) | 0 / — | present / 60 | 0 / 0 |
| F8-NS | NS (2) | D, W, C, H | 0 NOERROR (NODATA) | 0 / — | present / 60 | 0 / 0 |
| F8-ANY | ANY (255) | D, W, C, H | 4 NOTIMP | 0 / — | absent | 0 / 0 |

F8-CNAME-C describes this authority's first-hop reply, not a synthetic blocking response: its TTL, NS authority and AA flag come from the upstream. Cache hits can clear AA and decrement the TTL. The harness clears the cache before each matrix query, and separately queries H. Do not require an arbitrary upstream to send AA or an NS authority RR, or describe a harmless first-hop CNAME as NODATA. A/AAAA/HTTPS queries for C still have zero answers.

### 2.1.2 Mode comparison — measured 2026-10-31

Each cell is **RCODE; answer count/type/value; SOA TTL**. `—` means no SOA. These rows apply to D/W/H and also C except its CNAME query, which retains F8-CNAME-C in all three modes. All synthetic responses have AA=0, AD=0. Null A/AAAA answer TTLs are 60. These are the same compiled rules; only `blocking_mode` changes in a copy of the captured production request body.

| QTYPE | `nxdomain` | `default` | `null_ip` |
|---|---|---|---|
| A | 3; 0; 60 | 0; 1 A `0.0.0.0`; — | 0; 1 A `0.0.0.0`; — |
| AAAA | 3; 0; 60 | 0; 1 AAAA `::`; — | 0; 1 AAAA `::`; — |
| CNAME | 0; 0; 60 | 0; 0; 60 | 0; 0; — |
| SVCB | 0; 0; 60 | 0; 0; 60 | 0; 0; — |
| HTTPS | 3; 0; 60 | 0; 0; — | 0; 0; — |
| TXT | 0; 0; 60 | 0; 0; 60 | 0; 0; — |
| MX | 0; 0; 60 | 0; 0; 60 | 0; 0; — |
| NS | 0; 0; 60 | 0; 0; 60 | 0; 0; — |
| ANY | 4; 0; — | 4; 0; — | 4; 0; — |

`nxdomain` follows §12.5's recommended mode and avoids manufacturing successful null-address answers. `default` and `null_ip` also suppress the real addresses in this matrix, but their A/AAAA responses are successful answers containing unspecified addresses. Neither mode gives uniform NXDOMAIN. No product setting, resolver-document format or publication semantics changed for this measurement; `default` is a backend comparison, not a new document option.

An empty NOERROR response without a SOA in the comparison modes advertises no SOA negative-cache TTL; do not attach a 60-second TTL to those cells.

### 2.1.3 SOA TTL and cache expectations — measured 2026-10-31

**F8-TTL5:** repeating every nxdomain row with a document whose `blocked_ttl_seconds` is 5 changes each synthetic SOA TTL from 60 to **5**, including NODATA. RCODE, answer types and AA/AD stay the same. F8-CNAME-C retains its upstream TTL 300; F8-ANY still has no SOA. Thus 60 **is honoured** for NODATA. There is no five-second negative-cache constant on this path.

The pinned [message constructor](https://github.com/AdguardTeam/AdGuardHome/blob/v0.107.59/internal/dnsforward/msg.go) explains the measurement: `genDNSFilterMessage` treats A/AAAA/HTTPS separately; `NewMsgNODATA` and `NewMsgNXDOMAIN` both call `genSOA`, which takes the SOA RR TTL from `BlockedResponseTTL()`. Its fallback is 3600 only when that setting is zero. SOA MINIMUM is 86400 in both measured runs; it is not the observed RR TTL. Negative caching uses the lesser of SOA TTL and MINIMUM, so the advertised negative lifetime here is 60 or 5 seconds ([RFC 2308 §5](https://www.rfc-editor.org/rfc/rfc2308#section-5)). Client caches and application behavior can delay an allowlist correction (§12.5); no immediate reconnection is promised.

The members run `69f92f49` explicitly created a [TEST ONLY 5-second profile](../e2e/at/members/AT20.md). Devstack follow-up 16 also used `blocked_ttl_seconds: 5` in its [fixture setup](../../tests/e2e/fixtures16_live_test.go). Their five-second SOAs therefore agree with F8-TTL5. The type-dependent NODATA convention is reproduced at **both** settings; a configuration difference explains the TTL, not the earlier uniform-NXDOMAIN wording.

### 2.1.4 Answers — 2026-10-31

**Devstack follow-up 16, request 1:** yes, §12.4–§12.5 operator text should explicitly name NODATA. The invariant is fail-closed for answers as stated in §2.1 item 5. In `dns-negative`, cite F8-A/F8-AAAA/F8-HTTPS for address/HTTPS blocks, F8-CNAME for the directly blocked CNAME name, F8-SVCB for SVCB, F8-A scope C for the blocked alias A query, and F8-A scope W for the subdomain. Cite F8-TTL5 for this fixture's TTL. Require the measured RCODE and empty answer, rather than accepting either 0 or 3 for CNAME/SVCB. The fixture's deliberately absent upstream name and upstream-cache expiry rows are separate from synthetic filtering and keep their own authority controls.

**Members follow-up 8 routing, F4-05/F4-07, run `69f92f49`:** the AT20 direct CNAME/SVCB observations are NODATA and agree with F8-CNAME/F8-SVCB plus F8-TTL5; HTTPS agrees with F8-HTTPS, not the SVCB row. Use F8-A for AT21's remaining-resolver A block and AT22's A sentinel/last-good block, with F8-TTL5 when the applied profile is 5 seconds. Report RCODE, answer count/types, SOA TTL and cache state separately. This closes the DNS wording question only: native distribution, IDNA positive-wire fixtures, platform Private DNS/DoH and the remaining AT22 privacy/retention campaign retain their existing owners. No overall AT20–AT22 verdict is promoted by this resolver-only measurement.

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
| `blocking_mode` (API; YAML `filtering.blocking_mode`) | `nxdomain` | Per-type NXDOMAIN/NODATA response in §2.1.1, with no equivalent address path in the measured cases (`§12.5`). |
| `blocked_response_ttl` (API; YAML `filtering.blocked_response_ttl`) | `60` (configurable starting default; production value 待決策) | Synthetic NXDOMAIN **and NODATA** SOA TTL; §2.1.3 measures 60 and 5 seconds. Does not override the TTL of an unblocked upstream CNAME or guarantee client cache eviction (`§12.5`). |
| `dns.refuse_any` (YAML) | `true` | ANY returns NOTIMP (4), no answers or SOA, before policy filtering; not written by `node/dns`. |
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
| `/control/dns_config` | `POST` | Legacy adapter sets mode/TTL. Production `node/dns` also writes upstream/bootstrap/fallback, protection, rate limit, ECS, DNSSEC, IPv6 and private-PTR settings; the exact body is retained in [requests.json](evidence/2026-10-31-response-shape/requests.json). |
| `/control/filtering/status` | `GET` | Reads active `user_rules` and extracts `! fvpn-bundle-revision:`. |
| `/control/filtering/config` | `POST` | Enforces `{"enabled": true, "interval": 0}`. |
| `/control/filtering/set_rules` | `POST` | Atomically replaces `user_rules` with the compiled bundle (or `lastGood` on rollback). |
| `/control/filtering/refresh` | `POST` | Forces immediate reload of filter structures. |
| `/control/filtering/check_host` | `GET` | Smoke-tests candidate domains against the active `urlfilter` engine before marking publish complete. |

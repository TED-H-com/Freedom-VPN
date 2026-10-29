# Candidate Threat & Category Feeds (§12.3, DNS01)

> [!IMPORTANT]
> **Commercial Licensing Pending Decision (`待決策`)**
> Per Project Spec §12.3, **no external threat or category feed is enabled in the default configuration** (`DefaultSourceRegistry()` ships empty). A feed may only be enabled in production after legal/business owner review transitions its `license_status` from `pending_decision` to `approved` (`ValidateDraft` rejects any profile referencing an unapproved source with `SOURCE_LICENSE_UNAPPROVED`).

---

## 1. Source Registry & Ingestion Architecture (§12.3)

Each feed in `internal/dnspolicy` (`ListSource`) records:
- `source_id`: Stable identifier (e.g., `src_urlhaus_domains`).
- `version` & `content_sha256`: Upstream version/ETag and exact SHA-256 of raw feed bytes.
- `license` & `license_status`: License identifier and approval state (`pending_decision`, `approved`, `rejected`).
- `commercial_use_allowed`, `redistribution_notes`, `attribution_notes`: Governance metadata required by §12.3.
- `https_url`: Fixed `https://` endpoint passed through `ValidateSourceURL` (rejects `http://`, embedded credentials, IP literals, `localhost`, `.local`, `.internal`, `.test`, `.invalid`, and `.example` to prevent SSRF).
- `format`: Supported parser format:
  - `plain_domains`: One domain per line (`#` or `!` comments ignored).
  - `hosts`: Standard hosts syntax (`0.0.0.0 domain` or `127.0.0.1 domain`; skips `localhost` and broadcast entries; sets `include_subdomains=false`).
  - `adblock`: Domain-only subset of AdBlock syntax (`||domain^`; skips unsupported modifiers, regexes, paths, or `@@` exception lines while counting skipped lines in `QualityMetrics.SkippedUnsupported`).
- `update_cadence`: `待決策` configurable duration (`DefaultSourceUpdateCadence = 6h`).
- `metrics` (`QualityMetrics`): `rule_count`, `delta`, `parse_errors`, `skipped_unsupported`, `last_update`, `smoke_test_passed`, and `smoke_test_detail`.
- **Fail-Safe Last-Good Retention (`LastGood`)**: If a scheduled fetch fails, exceeds `MaxSourceBytes` (`16 MiB` default), produces zero valid domains, or fails smoke validation, `SourceRegistry.Ingest` preserves `LastGood` so resolver compilation continues uninterrupted with the previous verified snapshot.

---

## 2. Candidate Threat & Category Feeds (`待決策`)

All candidate feeds below are marked **`待決策` (Pending Business Decision)** and require owner sign-off before enabling in a paid/commercial VPN service.

| Candidate Feed | Category | Format | License & Commercial Terms | Attribution & Redistribution Notes | Reference Link | Status |
|---|---|---|---|---|---|---|
| **abuse.ch URLhaus Hostfile / Domains** ([S16]) | `malware` | `hosts` / `plain_domains` | **CC0 1.0 Universal (Public Domain Dedication)**. Free for commercial and non-commercial use when consumed via public bulk feeds within fair-use rate limits (poll no more than once every 5–15 minutes). | No attribution legally required under CC0, though crediting `URLhaus / abuse.ch` in documentation is recommended. Bulk feed redistribution as a standalone raw list is discouraged. | https://urlhaus.abuse.ch/api/ | **待決策** |
| **HaGeZi Multi / Threat Intelligence DNS Blocklists** | `malware` / `phishing` / `ads_trackers` | `plain_domains` / `adblock` (`\|\|domain^`) | **GPL-3.0** (repository scripts/lists) aggregating upstream sources. Must verify all upstream sub-sources permit commercial VPN resolver use. | Requires preserving upstream attribution notices in documentation if redistributed. | https://github.com/hagezi/dns-blocklists | **待決策** |
| **AdGuard DNS Filter (`AdGuardSDNSFilter`)** ([S17]) | `ads_trackers` / `malware` | `adblock` (`\|\|domain^`) | **GPL-3.0**. Compiled on central server and rendered into internal AdGuard Home resolvers; raw list is not redistributed to end-user devices. | Retain GPL-3.0 credit and link to `AdGuardTeam/AdGuardSDNSFilter` in operator/product attributions. | https://github.com/AdGuardTeam/AdGuardSDNSFilter | **待決策** |
| **StevenBlack Unified Hosts** ([S29]) | `ads_trackers` / `malware` | `hosts` | **MIT License** (top-level repository and tooling). Note: Aggregates multiple third-party host lists whose individual terms must be audited before commercial deployment. | Preserve MIT copyright and permission notice in product documentation. | https://github.com/StevenBlack/hosts | **待決策** |
| **Spamhaus Domain Blocklist (DBL)** ([S15]) | `malware` / `phishing` | DNSBL / Zone feed | **Commercial Data Feed License Required**. Spamhaus explicitly prohibits using its free public DNSBL servers for commercial VPN services, ISP resolvers, or paid security products. | Requires executing a commercial Spamhaus Technology (SHTech) data feed agreement before any production integration. | https://www.spamhaus.org/blocklists/domain-blocklist/ | **待決策** (Requires paid license) |
| **OpenPhish / PhishTank** | `phishing` | `plain_domains` / JSON | **Commercial License Required for SaaS/VPN** (OpenPhish Community feed is non-commercial evaluation only; PhishTank requires commercial API key approval for high-volume commercial service). | Cannot be enabled without commercial subscription agreement. | https://openphish.com/ / https://phishtank.org/ | **待決策** (Requires paid license) |
| **OISD Blocklist** | `ads_trackers` / `phishing` | `adblock` (`\|\|domain^`) | **Proprietary / Custom Terms**. OISD restricts commercial redistribution or re-branding as a paid feature without explicit permission from the maintainer. | Written maintainer approval required prior to commercial VPN use. | https://oisd.nl/ | **待決策** |

---

## 3. Google Safe Browsing vs. Google Cloud Web Risk (`§12.3`, [S13], [S14])

Per Project Spec §12.3:
- **Google Safe Browsing API v4 ([S13])**: Terms of Service explicitly restrict usage to **non-commercial** purposes ([https://developers.google.com/safe-browsing/v4/terms](https://developers.google.com/safe-browsing/v4/terms)). Because Freedom VPN is a commercial membership service, **Google Safe Browsing v4 MUST NOT be used in production**.
- **Google Cloud Web Risk API ([S14])**: The commercially licensed Google Cloud offering providing equivalent malware, social engineering (phishing), and unwanted software lists ([https://cloud.google.com/web-risk/docs](https://cloud.google.com/web-risk/docs)).
  - **Status**: **`待決策` (Out of scope for v1 implementation)**.
  - **Architecture Note for Future Wave**: If approved by the business owner, Web Risk is a hash-prefix / URL-lookup API rather than a static zone file. It should be integrated either as a periodic central exporter into a local `ListSource` (for domain-level entries) or remain out of the critical DNS path so resolver latency and member query privacy (`SEC02`) are never compromised by per-query external API calls.

# Candidate Threat & Category Feeds (§12.3, DNS01)

> [!IMPORTANT]
> **Commercial Licensing Pending Decision (`待決策`)**
> Per Project Spec §12.3, **no external threat or category feed is enabled in the default configuration** (`NewDefaultSourceRegistry()` ships empty). D-054 defines approval as activation of a `threat_feed` policy revision carrying its confirmed commercial-use basis. There is no second approval flag in the running content pipeline. The in-memory registry’s `license_status` remains a compiler primitive; it does not authorize a production fetch.

---

## 1. Source Registry & Ingestion Architecture (§12.3)

Each feed in `internal/dnspolicy` (`ListSource`) records:
- `source_id`: Stable identifier (e.g., `src_urlhaus_domains`).
- `version` & `content_sha256`: Upstream version/ETag and exact SHA-256 of raw feed bytes.
- `license` & `license_status`: License identifier and approval state (`pending_decision`, `approved`, `rejected`).
- `commercial_use_allowed`, `redistribution_notes`, `attribution_notes`: Governance metadata required by §12.3.
- `https_url`: Fixed `https://` endpoint passed through `ValidateSourceURL` (rejects `http://`, embedded credentials, IP literals, `localhost`, `.localhost`, `.local`, and `.internal` to prevent SSRF) and fetched via `NewSourceHTTPClient` (which disables environment proxies, refuses all HTTP redirects in `CheckRedirect`, resolves DNS via `net.DefaultResolver.LookupNetIP` in `Transport.DialContext`, rejects any resolved address in loopback, RFC1918, CGNAT `100.64.0.0/10`, link-local, multicast, unspecified, ULA, or documentation prefixes before connecting, and dials the verified IP directly).
- `format`: Supported parser format:
  - `plain_domains`: One domain per line (`#` comments ignored).
  - `hosts`: Standard hosts syntax (`0.0.0.0 domain` or `127.0.0.1 domain`; skips `localhost` and broadcast entries; sets `include_subdomains=false`).
  - `adblock`: Domain-only subset of AdBlock syntax (`||domain^`; skips unsupported modifiers, regexes, paths, or `@@` exception lines while counting skipped lines in `QualityMetrics.SkippedUnsupported`).
- `update_cadence`: D-054 selects 6h by default, configurable and still 待實測; the effective interval also respects the revision’s `update_interval_seconds`.
- `metrics` (`QualityMetrics`): `rule_count`, `delta`, `parse_errors`, `skipped_unsupported`, `last_update`, `smoke_test_result`.
- **Fail-Safe Last-Good Retention (`LastGood`)**: If a scheduled fetch fails, exceeds `MaxSourceBytes` (`32,000,000` bytes hard ceiling in the production refresher), produces zero valid domains, or fails smoke validation, `SourceRegistry.Ingest` preserves `LastGood` so resolver compilation continues uninterrupted with the previous verified snapshot.

---

## D-054 production content pipeline

1. An active `threat_feed` revision is the only authorization to fetch its fixed HTTPS URL. Drafts and superseded revisions are not fetched. Activation writes a `dns.feed.refresh` outbox intent; the DNS worker runner repairs due work on startup and every minute. Multiple workers coalesce intents and use a durable per-revision attempt lease. HTTP runs outside database transactions. Activation and the outbox claim are checked again before committing the result.
2. Bootstrap checks SHA-256 of the original response bytes against the approved snapshot pin. A mismatch leaves `last_good_content` null and records `content_mismatch`. Correcting the pin requires a new approved revision, not editing the observation tables.
3. Later downloads may change hash without another approval. Every candidate still passes HTTPS/DNS-address SSRF protection, redirect refusal, bounded decompressed size, declared format, normalization, non-zero rule count and the removal guard. Any parser error rejects the candidate; unsupported adblock constructs are counted and skipped, and can never add an allow or `$important` block. Hosts entries match exact names.
4. The removal guard counts previous entries absent from the candidate, including their matching scope. Removing exactly 50% passes the default; removing more fails, even when unrelated additions preserve the total count. Accepted entries and aggregate parse metrics become an immutable `(feed_revision_id, content_sha256)` row in `dns_feed_contents`. Raw bodies are discarded. `kind` is `approved_snapshot` for bootstrap and `refreshed` afterwards. Re-fetching an existing hash does not create another row.
5. A transaction moves `dns_feed_refresh_state.last_good_content`, clears failures, appends `dns.feed.refreshed` with hash/count, and enqueues affected active profiles for publication. Failure leaves last-good untouched, increments consecutive failures and appends `dns.feed.refresh_failed` against the revision with the system actor and only a reason code. A crash or lost job lease leaves recoverable attempt state; it cannot commit stale content. Same-event replay does not refetch or duplicate the observation.
6. The publisher embeds the last-good entries as Layer 3 rules. No last-good means no publication. A changed hash produces a new immutable document generation; unchanged content and inputs preserve bytes and generation. Resolver syntax/read-back/wire smoke tests run at apply time. Feed parse metrics record `smoke_test_result=untested`: ingestion is not evidence of real resolver enforcement.

### Parameter registry (D-054)

Optional values live alongside upstreams in `dns_config_file`; no new required setting is introduced. Existing files remain valid.

| Parameter | Default / bound | Meaning |
|---|---|---|
| `feed_update_cadence_seconds` | 21600 (6h), range 300–604800 | Product cadence, 待實測. Effective interval is the greater of this and the approved revision’s `update_interval_seconds`, respecting the source’s polling limit. |
| `feed_max_removal_percent` | 50, range 0–50 | Maximum percentage of previous rules removable in one refresh. Zero forbids any removal. |
| `max_size_bytes` | Required in the revision; capped at 32,000,000 bytes | Counts decompressed response bytes, with a one-byte overflow check. |
| `MaxEmbeddedFeedBytes` | 524288 bytes | Hard ceiling for a normalized feed’s JSON entries; stricter than the legacy ingest limits. |
| Resolver document ceiling | 1048576 bytes | Existing canonical JSON bound, including all feeds and admin rules. Overflow preserves the old publication head. |
| Fetch timeout / attempt lease / scheduling sweep | 15s / 60s / 60s | Fixed recovery bounds; the attempt lease allows interrupted fetches to be recovered. |

```json
{
  "upstreams": ["192.0.2.53:53"],
  "feed_update_cadence_seconds": 21600,
  "feed_max_removal_percent": 50
}
```

TEST ONLY upstream above; it is not a public resolver default. Feed validity is checked on every attempt over `[valid_from, valid_until)`; an out-of-validity attempt fails and retains last-good. Existing policy/lease validity still governs serving. Feed failure does not silently authorize an unfiltered profile.

### Operator runbook: `dns.feed.refresh_failed`

Read the revision’s audit events and the following projection (use a read-only database role until the admin observation contract is added):

```sql
SELECT r.id AS feed_revision_id, r.payload->>'feed_code' AS feed_code,
       s.last_good_content, s.last_attempt_at, s.last_success_at,
       s.last_error_code, s.consecutive_failures, s.next_attempt_at,
       c.rule_count, c.size_bytes, c.fetched_at, c.kind, c.parse_metrics
FROM policy_revisions r
JOIN dns_feed_refresh_state s ON s.feed_revision_id = r.id
LEFT JOIN dns_feed_contents c ON c.feed_revision_id = s.feed_revision_id
                            AND c.content_sha256 = s.last_good_content
WHERE r.type = 'threat_feed';
```

- `content_mismatch`: bootstrap bytes differ from the reviewed snapshot. Confirm the intended upstream version and approve a new revision with the reviewed digest, or restore the approved upstream snapshot. Never bypass the pin.
- `ssrf_refused`, `fetch_failed`, `http_status`, `read_failed`: investigate endpoint availability, DNS/public-address resolution and TLS from the control plane. Redirects are forbidden. Keep credentials and response bodies out of diagnostics.
- `oversize`, `content_too_large`, `parse_error`, `zero_rules`, `delta_breach`: investigate the feed’s declared format, normalized count and recent upstream changes. Retain last-good. A legitimately replaced list can be reviewed as a new bootstrap snapshot in a new activated revision. Do not widen the removal bound beyond 50%.
- `outside_validity`: approve a current revision with a valid source/license window. The worker will not fetch outside the approved window.
- `dns.publish` retry/dead letter after a successful refresh: inspect aggregate document bounds and referencing profile composition. A successful content refresh is not applied resolver evidence. The old document remains until publication succeeds.

`fvpn_dns_feed_refresh_total{feed_code,result}` counts committed refresh observations; `result=success` or a bounded failure code. Counters reset on process restart. `fvpn_dns_feed_last_good_age_seconds{feed_code}` is seconds since the last successful verification and advances between attempts; it is `+Inf` before bootstrap, restored from persisted observations on startup, and the maximum age if active definitions share a feed code. Alert on failures, consecutive failures and stale/absent last-good. Inspect each revision in the database when codes overlap. Audit and persisted state are authoritative across worker restarts. Scheduler/storage failures fail the supervised runner and worker health instead of being silently ignored.

After repairing an upstream, allow the next cadence attempt; activation of a new revision triggers bootstrap immediately. Do not write the projection tables manually. The admin read endpoint, portal evidence and a devstack-hosted approved feed fixture require orchestrator follow-up. Production fetching keeps the SSRF guard enabled; tests inject a transport to local `httptest` TLS fixtures, with no public feed traffic.

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

# Adversarial Review `r4-dnsfeeds-review`: Threat-Feed Content Pipeline (`D-054`)

- **Branch:** `r4-dnsfeeds-review` (commit `e0a5756`)
- **Date:** 2026-10-29 (`2026-10-29 22:21:27 -0400`)
- **Reviewing model:** `agy/gemini-4-argon-high` (recorded in commit `e0a5756` `Agent:` trailer; not stated in `.agent/REVIEW.md` body)
- **Decision / scope reviewed:** `D-054` threat-feed content pipeline (`§12.1`–`§12.5`, `§14.5`, `§16`, `DNS01`, `DNS02`, `SEC02`, `AT20`, `AT22`; `internal/dnsfeeds`, `internal/dnspolicy`, `internal/dnspublish`, `internal/policies`, `apps/worker`)
- **Adopted proof files on `main`:**
  - [`internal/dnsfeeds/review_test.go`](../../internal/dnsfeeds/review_test.go) (adopted in `#101`)
  - [`internal/dnsfeeds/review_integration_test.go`](../../internal/dnsfeeds/review_integration_test.go) (adopted in `#101`)
  - [`internal/dnspolicy/review_test.go`](../../internal/dnspolicy/review_test.go) (adopted in `#101`)
  - [`apps/worker/review_integration_test.go`](../../apps/worker/review_integration_test.go) (adopted in `#101`)

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | F-01: Rolling `lastGood` baseline in `internal/dnsfeeds/service.go` (`checkDelta`) allows multi-step 49% dilution to wipe 100% of pinned rules and enforces no addition ceiling or public-suffix wildcard boundary. | closed | #101 |
| High | F-02: SSRF guard in `internal/dnspolicy/sources.go` and `internal/dnsfeeds/fetch.go` allows IPv4-mapped IPv6, NAT64, 6to4, Teredo, Benchmark/TEST-NET-2/3, non-443 ports, and IPv6 zone IDs, and misclassifies redirect SSRF failures as `transport_error`. | closed | #101 |
| High | F-03: Pre-parse `io.ReadAll` in `internal/dnsfeeds/fetch.go` (`readBounded`) buffers up to 28 MiB in memory before enforcing `MaxRules`, and `MaxFeedRules` (250k) vs `MaxTotalRules` (500k) in `internal/dnspolicy/sources.go` breaks two-feed profile compilation when local rules exist. | closed | #101 |
| High | F-04: `policy.activate` worker handling in `apps/worker/jobs_policies.go` and `internal/dnspublish/store/store.go` marks operations `succeeded` with empty `observed_refs` even when DNS profile publication or threat-feed bootstrap fails. | closed | #101 |
| Medium | F-05: Superseded and expired feed revisions in `internal/dnspublish/store/store.go` and `internal/dnsfeeds/service.go` remain compiled into active DNS profiles without fanning out refreshed revisions or recompiling on profile reactivation. | closed | #101 |
| Medium | F-06: `PolicyDefinition.threat_feed_status` in `internal/policies/store.go` is left unpopulated on API reads, `last_attempt_at` is overwritten by in-progress refreshes, and `internal/dnsfeeds/service.go` computes `last_good_age_seconds` across superseded revisions. | closed | #93, #101 |
| Low | F-07: `SourceRegistry.UpdateFromReader` in `internal/dnspolicy/sources.go` lacks last-good retention on parse errors and rejects valid lines carrying inline `#` comments. | closed | #101 |

## What Held

- **First-fetch sha256 pinning and immutable content storage (`internal/dnsfeeds/service.go`, `db/migrations/00181_dns_feed_contents.sql`):** When `lastGood == nil` on bootstrap (`RefreshActive` / `BootstrapRevision`), a candidate body whose sha256 disagrees with `expected_sha256` is rejected with `ReasonHashMismatch`, inserts zero rows into `dns_feed_contents`, leaves `last_good_content_id` `NULL`, and records `result = 'failed'`, `reason = 'hash_mismatch'` in `dns_feed_refresh_observations`. Once stored, `dns_feed_contents` is protected by `BEFORE UPDATE OR DELETE` trigger `dns_feed_contents_immutable` and PostgreSQL `CHECK` constraints (`rule_count > 0`, `size_bytes > 0`, `valid_until > fetched_at`, `content_sha256 ~ '^[0-9a-f]{64}$'`).
- **Single-step >50% shrinkage and zero-rule rejection (`internal/dnsfeeds/service.go`):** A single candidate revision that drops `rule_count` to `0` (`ReasonZeroRules`) or removes more than 50% of the immediately preceding revision (`removed*2 > len(lastRules)`, `ReasonRemovalDelta`) is rejected and retains `last_good_content_id`.
- **AdGuard rule injection and Punycode/IDNA normalization (`internal/dnspolicy/sources.go`, `internal/dnspolicy/normalize.go`, `internal/dnspolicy/adguard.go`, `internal/dnsfeeds/fetch.go`, `internal/dnspublish/document.go`):** Inputs containing AdGuard modifier syntax (`||example.com^$important`, `@@||example.com^`, `/regex/`), IPv4 literals (`192.0.2.1`), ports (`example.com:8080`), paths/schemes (`https://example.com/path`), control characters, underscores, or multi-wildcard prefixes (`*.*.example.com`) are rejected by `NormalizeDomain`. IDNA labels (`bücher.example.com` -> `xn--bcher-kva.example.com`) are punycode-normalized and deduplicated deterministically, and Layer-1/2 admin rules override Layer-3 (`feed`) rules.
- **Privacy and redaction (`SEC02`, `internal/dnsfeeds/service.go`, `internal/dnsfeeds/metrics.go`):** Source URLs, query parameters, response bodies, and domain patterns are never written to `slog`, `audit_events.after_diff`, `dns_feed_refresh_observations`, or Prometheus metric labels (metrics expose only `code` and bounded `reason` enums).

## Detailed Review Notes (Archived from `.agent/REVIEW.md`)

### `F-01` (High) — Rolling `lastGood` baseline in `checkDelta` allows two-step 49% dilution to wipe 100% of pinned rules and accepts unbounded rule additions
- **Where:** `internal/dnsfeeds/service.go:404-439` (`checkDelta`), `internal/dnsfeeds/service.go:258` (`refreshOne`), `internal/dnspolicy/normalize.go:42-145` (`NormalizeDomain`).
- **Scenario:** `checkDelta` compares each scheduled refresh only against the rolling `lastGood` (`removed*2 > len(lastRules)`), never against the initial `approved_snapshot` (`expected_sha256`), and imposes no cap on `added` rules. Two consecutive refreshes each removing 49% of the current rules replace 100% of the pinned baseline in two ticks, and `*.co.uk` (2 labels) passes `NormalizePattern`.
- **Proof:** `TestReview_F01_TwoStepDilutionWipes100PercentOfPinnedRules` (`internal/dnsfeeds/review_test.go`).
- **Fix (#101):** Enforced removal and addition delta bounds against both `last_good` and the pinned `approved_snapshot`, and rejected public-suffix wildcard patterns (`internal/dnspolicy/isolation.go`).

### `F-02` (High) — SSRF guard allows IPv4-mapped IPv6, NAT64, 6to4, Teredo, Benchmark/TEST-NET-2/3, non-443 ports, IPv6 zone IDs, and misclassifies redirect SSRF as `transport_error`
- **Where:** `internal/dnspolicy/sources.go:118-137` (`ValidateSourceURL`, `isDisallowedSourceAddr`), `internal/dnsfeeds/fetch.go:112-135, 169-174` (`Fetch`, `safeDialContext`).
- **Scenario:** `isDisallowedSourceAddr` did not call `addr.Unmap()` before checking IPv6 prefixes, allowing `::ffff:127.0.0.1`, `64:ff9b::a9fe:a9fe`, `2002:7f00:1::`, `2001:0::`, `198.18.0.1`, `198.51.100.1`, and `203.0.113.1`, allowed arbitrary TCP ports and IPv6 zone IDs, and classified redirect SSRF failures as `ReasonTransport` instead of `ReasonSSRFRefused`.
- **Proof:** `TestReview_F02_SSRFGuardBypassesAndPortAndRedirectClassification` (`internal/dnsfeeds/review_test.go`), `TestReview_F02_ValidateSourceURLAndSSRFAddrBypasses` (`internal/dnspolicy/review_test.go`).
- **Fix (#101):** Enforced TCP port 443 only, rejected unmapped/special/zoned IPv6 and reserved IPv4 ranges, and classified redirect SSRF refusals as `ssrf_refused`.

### `F-03` (High) — Pre-parse `io.ReadAll` buffers up to 28 MiB before rule cap, and `MaxFeedRules` (250k) vs. `MaxTotalRules` (500k) breaks 2-feed compilation
- **Where:** `internal/dnsfeeds/fetch.go:38-45, 198-211` (`DefaultLimits`, `readBounded`), `internal/dnspolicy/sources.go:25-33`.
- **Scenario:** `readBounded` buffered up to 28 MiB via `io.ReadAll` before `ParseRules` checked `MaxRules`, and two 250k-rule feeds plus any local rule exceeded `MaxTotalRules = 500,000`.
- **Proof:** `TestReview_F03_ParserMemoryAmplificationAndMaxRulesMismatch` (`internal/dnsfeeds/review_test.go`).
- **Fix (#101):** Replaced whole-body buffering with a bounded streaming line parser and reserved local-rule headroom in `MaxFeedRules`.

### `F-04` (High) — `policy.activate` operation succeeds when DNS publication or feed bootstrap fails, and drops `observed_refs`
- **Where:** `apps/worker/jobs_policies.go:17-60`, `internal/dnspublish/store/store.go:196-237`.
- **Scenario:** `registerPoliciesJobs` marked `policy.activate` operations `succeeded` with `observed_refs = []` even when `BootstrapRevision` or `PublishActiveProfile` failed.
- **Proof:** `TestReview_F04_PolicyActivateOperationUntruthfulOnDNSPublishOrFeedFailureAndDropsObservedRefs` (`apps/worker/review_integration_test.go`).
- **Fix (#101):** Recorded `dns_feed_refresh` and `dns_document` operation steps and failed the activation operation with a bounded failure code whenever bootstrap or publication failed.

### `F-05`, `F-06`, `F-07` (Medium / Low) — Superseded/expired feed revisions, `threat_feed_status` projection and metrics, and `SourceRegistry.UpdateFromReader`
- **Proofs:** `TestReview_F05_SupersededAndExpiredFeedRevisionCompiledIntoActiveProfileAndReactivationSkipped`, `TestReview_F06_ThreatFeedStatusUnpopulatedAndPrematureLastAttemptAt` (`internal/dnsfeeds/review_integration_test.go`), `TestReview_F06_MetricsLastGoodAgePoisonedBySupersededRevision` (`internal/dnsfeeds/review_test.go`), `TestReview_F07_SourceRegistryUpdateFromReaderOverwritesLastGoodOnParseErrors` (`internal/dnspolicy/review_test.go`).
- **Fix (#93, #101):** Added lineage fan-out and generation fencing (`db/migrations/00182_dns_feed_review.sql`), projected `PolicyDefinition.threat_feed_status` from completed observations on the active revision, filtered `last_good_age_seconds` to the active revision, and preserved last-good state and inline comment stripping in `SourceRegistry.UpdateFromReader`.

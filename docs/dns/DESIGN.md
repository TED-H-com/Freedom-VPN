# DNS Domain Policy Engine Design (§12, DNS01–DNS04, SEC02)

## 1. Overview & Source of Truth (§12.2)

Freedom VPN maintains its own internal DNS policy model in `internal/dnspolicy` (`Profile`, `Rule`, `ListSource`, `CompiledBundle`) as the single source of truth. Filtering backends (currently AdGuard Home `v0.107.59`) operate strictly behind a deterministic compiler and adapter (`AdGuardAdapter`, `MultiResolverPublisher`).

This design ensures:
- Reference evaluation semantics (`Evaluate(profile, name) Decision`) are independent of any vendor's proprietary config format.
- Policy validation, conflict detection, deterministic hashing, and rollback occur under our control before or during resolver publication.
- DNS filtering never loosens IP-layer isolation (`§8.1`, `§12.1`): member traffic can only reach the assigned overlay DNS resolvers on `UDP/53` and `TCP/53` via explicit `nftables` firewall exceptions before the `drop` rule.

---

## 2. Domain Normalization & IDNA Profile (§12.4)

`NormalizeDomain(raw string)` enforces strict syntactic and internationalized domain name validation before any rule is saved or evaluated:

1. **Input Rejection (`ValidationErrorCode`)**:
   - Empty or whitespace-only strings (`DOMAIN_EMPTY`)
   - ASCII control characters `< 0x20` or `0x7F` (`DOMAIN_CONTROL_CHAR`)
   - Whitespace characters (`DOMAIN_WHITESPACE`)
   - URL schemes (`http://`, `https://`), ports (`:443`), paths (`/`), query strings (`?`), fragments (`#`), or userinfo (`@`) (`DOMAIN_HAS_SCHEME`, `DOMAIN_HAS_PORT`, `DOMAIN_HAS_PATH`, `DOMAIN_HAS_QUERY`, `DOMAIN_HAS_FRAGMENT`, `DOMAIN_HAS_USERINFO`)
   - Wildcards (`*`) (`DOMAIN_HAS_WILDCARD`)
   - IPv4 or IPv6 literals, including bracketed literals (`DOMAIN_IP_LITERAL`)
2. **Trailing Root Dot & Label Splitting**:
   - Strips at most one trailing root dot (`.`); a double trailing dot (`example.com..`) is rejected as `DOMAIN_EMPTY_LABEL`.
   - Rejects leading dots (`.example.com`), consecutive internal dots (`a..b.com`), and bare Punycode prefixes (`xn--`).
3. **IDNA Profile & Rationale**:
   - Uses `golang.org/x/net/idna.New()` with:
     - `idna.MapForLookup()` (lowercasing and UTS #46 lookup mapping)
     - `idna.StrictDomainName(true)` (STD3 ASCII rules: only `a-z`, `0-9`, and `-` allowed in ASCII labels)
     - `idna.ValidateLabels(true)` (NFC normalization check, hyphen placement rules—no leading/trailing hyphen or `..--` in positions 3–4 except `xn--`, contextual joiner rules, and bidirectional Bidi rules)
     - `idna.VerifyDNSLength(true)` (1–63 bytes per ASCII label, 1–253 bytes total A-label domain length)
     - `idna.Transitional(false)` (strict IDNA2008 non-transitional processing so `ß` and `ς` are not silently collapsed into `ss` or `σ`, preventing spoofing ambiguity)
   - Additionally enforces RFC 5891 / RFC 5892 IDNA2008 registration code-point checks on converted U-labels, rejecting Unicode symbols/pictographs (such as emoji domains `💩.example.com`) that UTS #46 lookup mode might otherwise pass through to Punycode.
4. **Output (`NormalizedDomain`)**:
   - `Canonical`: Lowercase ASCII / Punycode (A-label) form stored for deterministic matching and bundle compilation.
   - `Display`: Original user input trimmed of surrounding spaces, retained for Admin UI display.

---

## 3. Rule Model, Subdomain Matching & 4-Layer Precedence (§12.2, §12.4)

### 3.1 Rule Structure

Each custom rule (`Rule`) follows §12.4:
```json
{
  "rule_id": "drule_01J...",
  "profile_id": "dnsprof_default",
  "domain": "login.example.com",
  "include_subdomains": true,
  "action": "allow",
  "reason": "ticket-1234 false positive override",
  "revision": "18"
}
```
Revisions use `pkg/decstr`-compatible decimal strings (`^(0|[1-9][0-9]*)$`, at most 20 digits, `<= 18446744073709551615`).

### 3.2 Label-Boundary Matching (`MatchesDomain`)

- `include_subdomains = false`: Exact match only (`q == rule.domain`). `example.com` matches `example.com`, **not** `sub.example.com`.
- `include_subdomains = true`: Matches the apex (`q == rule.domain`) and any subdomain on a full dot-separated label boundary (`strings.HasSuffix(q, "." + rule.domain)`).
- Lookalike protection: `example.com` **never** matches `notexample.com` or `badexample.com`.

### 3.3 Strict 4-Layer Precedence (`Evaluate`)

When evaluating a query name against a `Profile`, layers are checked strictly from highest to lowest priority:

| Priority | Layer (`Layer`) | Source | Action | Semantics |
|---|---|---|---|---|
| **1 (Highest)** | `admin_block` | Admin forced block rules (`action: "block"`) | `block` | Overrides all allow rules and threat feeds. |
| **2** | `admin_allow` | Admin explicit allow rules (`action: "allow"`) | `allow` | Overrides category/threat feeds (`source_list`), but **never** overrides `admin_block`. |
| **3** | `source_list` | Enabled threat & category feeds (`ListSource`) | `block` | Blocks domains listed in enabled feeds (subdomain-inclusive `||domain^` or exact `|domain^`). |
| **4 (Lowest)** | `normal_resolution` | Default fallback | `resolve` | Forwarded to configured authoritative/recursive upstream resolvers. |

### 3.4 Draft Validation & Conflict Rejection (`ValidateDraft`)

Before compilation or publication, `ValidateDraft(profile)` verifies:
- Profile ID, revision format, and rule count ceiling (`MaxRulesPerProfile`, default `500,000`).
- Every rule's `profile_id` matches the target profile, has a non-empty `reason`, and passes `NormalizeDomain`.
- **No "last write wins"**: Any duplicate or conflicting rules on the same canonical domain within the same layer (e.g., two `admin_block` rules or two `admin_allow` rules targeting the same domain) are rejected with `RULE_CONFLICT_SAME_LAYER`.
- Every referenced `ListSource` has `LicenseStatus == LicenseApproved` (rejecting `pending_decision` or `rejected` feeds with `SOURCE_LICENSE_UNAPPROVED`).

---

## 4. Publish Flow, Multi-Resolver Coordination & Failure Policy (§12.5, §16.4)

### 4.1 Single-Resolver Publish (`AdGuardAdapter.Publish`)

1. **Bundle Integrity & Syntax Check**:
   - Recomputes the SHA-256 of `BundleBytes` and validates every rendered AdGuard rule line (`ValidateAdGuardRuleSyntax`) **before** making any network calls.
2. **Idempotency Check**:
   - Reads the resolver's currently applied revision marker (`! fvpn-bundle-revision: <rev>`). If it already matches `bundle.Revision` and passes smoke tests, returns immediately with ` NoOp = true`.
3. **Snapshot & Apply**:
   - Captures the currently active rules from the resolver (`GET /control/filtering/status`) to seed `lastGood` if not already cached in memory.
   - Applies privacy/logging hardening (`querylog.enabled=false`, `statistics.enabled=false`), DNS blocking mode (`nxdomain`, `blocked_response_ttl=60`), and the compiled `user_rules` (`POST /control/filtering/set_rules` + `POST /control/filtering/refresh`).
4. **Post-Publish Smoke Verification**:
   - Exercises every `SmokeTestCase` embedded in the bundle against both:
     - AdGuard Home's `GET /control/filtering/check_host` control API, and
     - Real UDP/TCP wire DNS queries (`A` records) verifying `NXDOMAIN` for blocked domains and `NOERROR` for allowed/unlisted probe domains.
5. **Automatic Rollback to `lastGood`**:
   - If applying rules or any smoke test fails, `restoreLastGood` immediately re-applies the previous known-good rule set, refreshes the engine, and returns `RolledBack = true` with the failure cause.

### 4.2 Multi-Resolver Coordination (`MultiResolverPublisher`)

- Publishes to all configured resolvers (`resolver-1`, `resolver-2`) and aggregates individual `ResolverPublishResult` entries into a `MultiPublishReport`:
  - `all_applied`: Every resolver succeeded and verified the target revision.
  - `partial`: At least one resolver succeeded and at least one failed (and rolled back to `lastGood`). **Never** reported as full success.
  - `failed`: Every resolver failed.

### 4.3 Resolver Outage & Client Failover (`QueryWithFailover`)

- When the primary resolver (`resolver-1`) is unreachable or returns `SERVFAIL`, `QueryWithFailover` fails over to the secondary resolver (`resolver-2`).
- **Fail-Closed Guarantee**: If all configured resolvers are unreachable, resolution fails with `ErrAllResolversUnavailable`. There is **no** fallback to unconfigured public DNS servers (`8.8.8.8`, `1.1.1.1`, etc.).

---

## 5. Acceptance Criteria Coverage (AT20–AT22)

| Acceptance ID | Requirement Summary (§19) | Implementation & Verification |
|---|---|---|
| **AT20** | Known malicious/blocked domain returns `NXDOMAIN` (`0.0.0.0`/blocked response) on both resolvers; explicit allow rule overrides feed block and restores normal resolution; admin forced block overrides explicit allow. | Verified in unit tests (`TestVectorFile_Evaluation`, `TestAT20_DNS02_EvaluatePrecedenceAndLookalikes`) and against both live AdGuard Home containers (`TestAT20_DNS01_DNS02_IntegrationVectorsAndAppliedRevision`, `TestAT20_DNS02_IntegrationRecordTypesAndCNAMEChain`). Covers `A`, `AAAA`, `CNAME` chains to blocked targets, and `HTTPS`/`SVCB` records. |
| **AT21** | Primary resolver failure seamlessly fails over to secondary resolver while preserving filtering policy; when both resolvers fail, resolution fails closed without falling back to unfiltered public DNS; broken bundle on one resolver rolls back to `lastGood` and reports `partial`. | Verified in `TestAT21_IntegrationFailoverAndNoUnfilteredFallback`, `TestAT21_IntegrationBrokenBundlePartialRollback`, and `TestAT20_AT21_IntegrationDraftDoesNotAffectLive`. |
| **AT22** | Resolver configuration and runtime state store no per-member DNS query history (`querylog.enabled=false`, `querylog.file_enabled=false`, `statistics.enabled=false`). | Verified in `TestAT22_SEC02_IntegrationDataMinimization`: queries sentinel domains across `A`, `AAAA`, and `HTTPS`, then inspects `/control/querylog`, `/control/stats`, container logs, and all files in `/opt/adguardhome/work` and `/opt/adguardhome/conf` to confirm zero occurrences of queried names. |

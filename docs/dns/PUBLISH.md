# DNS policy publishing (§12.4, §12.5, §12.7, §17.1)

An activated `dns_profile` becomes a canonical resolver document, with a new immutable generation when referenced feed content changes. The worker stores that document. The `node/dns` applier applies those bytes to AdGuard Home, reads them back, and proves them with synthetic queries. The production node-agent transport integration is still required; the lab calls this applier directly. Member DNS is service traffic (`dns_service_v1` in `node/enforce`, §10.2). The applier does not emit usage events and does not change the nftables isolation rules (§8.2).

`internal/dnspolicy.CompileBundle` is unchanged. Its hash includes a wall-clock validity window, so two activations of the same rules do not match. The publish document is a separate artifact.

## Pipeline

1. An admin activates a `dns_profile` revision. The policies store writes `policy.activated` and the existing compiled bundle.
2. Migration `00180` inserts a second outbox row, `dns.publish`, for `payload.type = dns_profile` only. `policy.activated` stays available for desired-state compilation (§14.5). That trigger is the current fan-out. A later worker may fan out from the `policy.activated` handler instead.
3. `worker.Domains` registers `dns()`. It loads `dns_config_file` and fails startup with `config.SettingError{Name: "dns_config_file"}` when that file is missing or invalid. The handler in `internal/dnspublish/store` loads the profile, the referenced domain rules, and the threat feeds’ persisted last-good entries, then calls `dnspublish.Compile`. The document package does not import a database driver.
4. The RFC 8785 bytes and their SHA-256 are stored in `dns_resolver_documents`, keyed by `(profile_revision_id, policy_generation)` after migration `00181`. The revision id remains the id `desired_states.policy_refs` carries. Old content generations remain immutable and readable by their document ID through `dnspublish.DocumentReader`; `GetByGeneration` remains available to existing callers. `dns_publication_heads` points at the current document.
5. `node/dns` converges every configured resolver or restores the previous configuration. D-056 assigns authenticated document delivery and issued-reference authorization to the nodes owner, and consumption to the agent owner. The read API below supplies the reference and exact bytes. A publication reference alone is not applied resolver evidence.

Upstreams and bootstrap servers are deployment configuration, not fields of the policy payload and not a built-in default (D-028). `dns_config_file` points at a JSON file: `upstreams` is a non-empty list of `host:port` values, each an IPv4 or IPv6 literal. `bootstrap_dns` uses the same form and may be empty, in which case the document copies `upstreams`. It is never a public resolver. [`config.example.json`](config.example.json) is TEST ONLY and uses documentation prefixes (`192.0.2.0/24`, `2001:db8::/32`). The same rules plus the same file produce the same bytes. A config change is a different document.

D-054 activates feed ingestion when a `threat_feed` revision activates. Migration `00181` fans that activation out to `dns.feed.refresh`; the DNS worker’s supervised periodic runner repairs due refresh intents every minute. The refresh service fetches only active revisions, through the HTTPS/SSRF guard. Bootstrap must match the approved raw `content_sha256`. Subsequent content must pass size, parse, non-empty and set-removal guards. See [FEEDS.md](FEEDS.md) for parameters and operator handling.

The document embeds normalized entries from each referenced feed’s persisted last-good content, with its raw content hash and rule count. Missing last-good content rejects publication; metadata alone cannot produce an apparently filtered document. A failed refresh keeps the previous content and does not publish an empty replacement. A moved last-good atomically enqueues `dns.publish` for every active referencing profile. Duplicate events or a refresh returning identical content keep the same bytes and generation. Publication failures remain failed/retryable outbox work and retain the previous head.

Document generations are allocated above the global DNS maximum, serialized across workers. They are independent of the policy definition’s activation generation: a feed refresh never mutates the approved policy revision. An old activation cannot replace the current profile head. Embedded entries are capped at 512 KiB per feed; the complete canonical document retains its existing 1 MiB ceiling. Exceeding the combined ceiling fails publication without advancing its head. Large deployments need the separate authenticated artifact contract before raising this bound.

The DNS handler does not mark `policy.activate` succeeded or complete `desired_state_compile`; the separate policies worker owns that step. It records `observed_refs.dns_document` with the hash and generation while the operation remains pending (§14.5). A committed publication or withdrawal appends `dns.publish` or `dns.withdraw` with `audit.NewEvent` for the policy definition and again for the profile revision (D-029). The actor is the system singleton, the result is `succeeded`, and the operation's request id is copied when that row has one. The diff stores the hash, generation, status, and whether the head moved. It does not store domain names.

## Read API for delivery (D-056)

`internal/dnspublish` exports `DocumentReader`, `PublishedDocument`, and `ErrDocumentNotFound`. Construct the PostgreSQL implementation with `internal/dnspublish/store.NewDocumentReader(pool)`; it needs no publisher/upstream configuration.

```go
reader := store.NewDocumentReader(pool)
latest, err := reader.LatestPublished(ctx, profileDefinitionID)
version, err := reader.GetByID(ctx, documentID)
```

Both methods return `PublishedDocument` with `DocumentID`, `ProfileRevisionID`, `Generation`, `ContentSHA256`, `SizeBytes`, and `Canonical`. These map to the contract's `document_id`, `profile_revision_id`, `generation`, `content_sha256`, and `size_bytes`; `Canonical` is the exact stored RFC 8785 body. The generation is the DNS document generation, not the policy definition or desired-state generation. `SizeBytes` is bounded to 1–1048576 bytes. Reads verify normalization, privacy, canonical byte equality and the SHA-256 before returning content.

`LatestPublished` joins the active head to its exact revision, generation and hash in one database snapshot. It returns `ErrDocumentNotFound` when no head exists, the head is withdrawn, or its revision is no longer active (including while a replacement awaits publication). `GetByID` never follows a newer head and retains historical content after refresh or withdrawal. Missing IDs return the same sentinel. These are read-only operations with no fetch or publication side effects.

The nodes owner must authorize an ID against the current or immediately previous desired-state generation actually issued to the requesting node before serving it. Unknown and unissued IDs map to 404 `NOT_FOUND`. Send `Canonical` verbatim, without JSON re-encoding or a newline; set `ETag: "sha256:<ContentSHA256>"` and `Cache-Control: private, max-age=0`. The agent verifies the signed reference's exact size/hash before applying. Neither lookup grants delivery authorization.

## Document

`schema_version` is 1 and `kind` is `resolver_document`. `pkg/canonjson` produces the hashed bytes. `policy_generation` is stored beside the document, not inside it, so a second activation of the same rules matches byte for byte.

| Field | Rule |
|---|---|
| `profile_code` | Canonical profile code |
| `block_response` | `nxdomain` (default), `refused`, or `null_address` |
| `blocked_ttl_seconds` | Profile value; `0` becomes 60. Production TTL is 待決策 (§12.5) |
| `safe_search` | Always present. Default `false`. A raw `safe_search` boolean is honored if a future payload adds it. The current schema rejects unknown fields, so published profiles are `false` |
| `privacy` | `querylog_enabled`, `querylog_file_enabled`, and `statistics_enabled` are forced `false` (§12.7, §16.4) |
| `upstreams` | Non-empty, order kept, duplicates dropped |
| `bootstrap_dns` | Same cleanup. Empty means a copy of `upstreams` |
| `fallback_dns` | Always `[]`. There is no public-DNS fallback (§8.2, §12.5) |
| `rules` | IDNA ASCII, sorted by `admin_block` then `admin_allow`, then domain, then exact match before subdomains, then `rule_id` |
| `feeds` | Sorted by `feed_code`; categories sorted and unique; `entries` contains sorted, unique normalized domains and their `include_subdomains`; `rule_count` matches the entries; `content_sha256` identifies raw upstream content |

Rules render to AdGuard Home syntax through `dnspolicy.RenderAdGuardRule`, with comment headers:

```text
! fvpn-bundle-version: 1
! fvpn-profile-id: <profile_code>
! fvpn-policy-generation: <generation>
! fvpn-bundle-sha256: <hash>
```

Layer 3 entries render as `||domain^` for domain-list/adblock feeds and `|domain^` for hosts feeds. Explicit `@@` allows override them; `$important` admin blocks retain highest priority. Metadata comments include `! fvpn-feed: <feed_code> <hash> <rule_count>` for read-back evidence. Probes classify names by effective precedence, so an overridden feed entry is not mistakenly tested as blocked.

A head never moves to an older generation. Replaying the same revision and generation is a no-op when the hash matches. A withdrawn generation cannot be activated again. Documents are immutable; head updates must increase `resource_version`, and `policy_generation` cannot decrease.

## Evidence and dns_ok

`node/dns.Evidence` is the internal evidence returned to the agent adapter:

```json
{"policy_generation":"3","document_hash":"<64 hex>","applied_at":"...","verified":true,"feeds":[{"feed_code":"example-malware","content_sha256":"<64 hex>","rule_count":2}]}
```

Revision 5 adds `DnsPolicyEvidence` and `DnsFeedEvidence`. The agent owner maps `policy_generation` to `generation`, `document_hash` to `content_sha256`, adds the issued document ID, and maps the verified result to `outcome` and nullable `applied_at`. Existing per-feed `feed_code`, `content_sha256` and `rule_count` already match the contract; entries never appear in evidence. `NodeCheck.detail`, when present, must be bounded and redacted. The agent reports `checks[name=dns_policy].status = passed` only when `Evidence.Verified` is true (`Evidence.DNSOK`). That check is the `dns_ok` advertisement. When `verified` is false the agent must not advertise `dns_ok` and must not report this generation in `applied_policy_refs`. `DNSPolicyCheckName` is the contract value `dns_policy`.

`applied_at` is the node clock in UTC. The hash is the SHA-256 of the canonical bytes, not of the generation.

## Apply, failure, and withdraw

Apply verifies the document before any write. It then, on every resolver:

1. Turns the query log off (`enabled` and `file_enabled`) and turns statistics off (§12.7).
2. Sets safe search, including each engine, from `safe_search`.
3. Removes external filter lists. Those URLs are not restored later, so a rollback cannot re-enable a third-party list.
4. Sets upstreams, bootstrap, an empty fallback, the blocking mode, and the TTL. `null_address` is AdGuard `null_ip`.
5. Replaces user rules with the rendered document and clears the cache.

It does not change the client access list (§8.2). It reads the configuration back, then sends a synthetic A query for up to four block names and four allow names. Feed evidence survives rollback and describes the content actually restored. A block must be NXDOMAIN, REFUSED, or `0.0.0.0` according to the blocking mode. An allow must return a real address and must not be the block signal. Probe names are not included in errors (§16.4).

Any failure restores the previous snapshot on every resolver. The returned evidence describes that previous policy when it still verifies. Otherwise `verified` is false. The node must not advertise `dns_ok` in that case. A half-applied resolver is not left behind.

Withdraw writes a fail-closed configuration: filtering on, privacy off, safe search off, fallback empty, upstream and bootstrap `127.0.0.1:1`, and only a withdrawn marker plus the generation and hash. `127.0.0.1:1` is inside the resolver network namespace. It is not a public resolver. A synthetic query must not return an A or AAAA answer. Timeout, SERVFAIL, REFUSED, NXDOMAIN, and empty NOERROR are fail-closed. Success returns `verified: false`. If that probe still returns an address, the previous configuration is restored.

Withdrawing a hash that is not the current verified document is a no-op. An empty hash withdraws whatever is installed. A lower generation does not move the publication head.

## Lab

Unit tests do not need Docker or root. The lab does. It starts the pinned CoreDNS and two AdGuard Home containers on a bridge named `$FVPN_LAB_PREFIX-net`. Containers are not privileged and are not attached to the host network. DNS and HTTP are published only on `127.0.0.1`.

```sh
make -f mk/dns.mk dns-lab
# or
FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w3-dns-publish \
  go test -count=1 -tags=netlab -timeout 10m ./tests/netlab/dns/
```

`FVPN_LAB_PREFIX` must match `^fvpn-[a-z0-9][a-z0-9-]{0,40}$`. The make target defaults it to `fvpn-w3-dns-publish`. The test removes its containers, network, and temporary state directory when it finishes. `dns-verify` does not start Docker.

The scenario fetches a TEST ONLY approved feed from a local HTTPS fixture, applies its normalized content, and proves both feed names are blocked on both resolvers. It then adds an explicit allow and expects `blocked.example.com` (and a subdomain) to be NXDOMAIN on UDP and TCP from both resolvers, expects `safe-target.example.com` to answer `192.0.2.15`, checks that the query log and statistics are off, withdraws, and expects neither name to return an address.

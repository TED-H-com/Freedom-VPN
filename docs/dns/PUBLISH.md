# DNS policy publishing (§12.4, §12.5, §12.7, §17.1)

An activated `dns_profile` becomes one canonical resolver document. The worker stores that document. Each node applies the same bytes to AdGuard Home, reads them back, and proves them with a synthetic query. Member DNS is service traffic (`dns_service_v1` in `node/enforce`, §10.2). The applier does not emit usage events and does not change the nftables isolation rules (§8.2).

`internal/dnspolicy.CompileBundle` is unchanged. Its hash includes a wall-clock validity window, so two activations of the same rules do not match. The publish document is a separate artifact.

## Pipeline

1. An admin activates a `dns_profile` revision. The policies store writes `policy.activated` and the existing compiled bundle.
2. Migration `00180` inserts a second outbox row, `dns.publish`, for `payload.type = dns_profile` only. `policy.activated` stays available for desired-state compilation (§14.5).
3. `apps/worker` handles `dns.publish`. It loads the profile, the referenced domain rules, and the threat-feed identities, then calls `dnspublish.Compile`.
4. The RFC 8785 bytes and their SHA-256 are stored in `dns_resolver_documents`, keyed by `profile_revision_id`. That is the id `desired_states.policy_refs` already carries. `dns_publication_heads` points at the current document.
5. The node agent (not edited here) passes the document to `node/dns`. The applier converges every configured resolver or restores the previous configuration.

Upstreams and bootstrap servers are worker configuration (`dnspublish.Config`), not fields of the policy payload. The default upstream is `https://dns.example.com/dns-query` and is never dialed by the compiler. The same rules plus the same config produce the same bytes. A config change is a different document.

Threat-feed revisions store metadata and `content_sha256` only. The publisher does not fetch feed URLs (§12.3). The document pins `feed_code`, `content_sha256`, `format`, and categories. Feed domain bodies are not in the policy revision, so they are not rendered as filter rules.

The handler does not mark `policy.activate` succeeded. `desired_state_compile` stays pending. It records `observed_refs.dns_document` with the hash and generation while the operation remains pending (§14.5). Audit rows store the hash, generation, status, and whether the head moved. They do not store domain names.

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
| `feeds` | Sorted by `feed_code`. Categories sorted. No fetched names |

Rules render to AdGuard Home syntax through `dnspolicy.RenderAdGuardRule`, with comment headers:

```text
! fvpn-bundle-version: 1
! fvpn-profile-id: <profile_code>
! fvpn-policy-generation: <generation>
! fvpn-bundle-sha256: <hash>
```

A head never moves to an older generation. Replaying the same revision and generation is a no-op when the hash matches. A withdrawn generation cannot be activated again. Documents are immutable; head updates must increase `resource_version`, and `policy_generation` cannot decrease.

## Evidence and dns_ok

`node/dns.Evidence` is what the agent copies into its observation report:

```json
{"policy_generation":"3","document_hash":"<64 hex>","applied_at":"...","verified":true}
```

`NodeCheck` has no extra detail field. The agent reports `checks[name=dns_policy].status = passed` only when `Evidence.Verified` is true (`Evidence.DNSOK`). That check is the `dns_ok` advertisement. When `verified` is false the agent must not advertise `dns_ok` and must not report this generation in `applied_policy_refs`. `DNSPolicyCheckName` is the contract value `dns_policy`.

`applied_at` is the node clock in UTC. The hash is the SHA-256 of the canonical bytes, not of the generation.

## Apply, failure, and withdraw

Apply verifies the document before any write. It then, on every resolver:

1. Turns the query log off (`enabled` and `file_enabled`) and turns statistics off (§12.7).
2. Sets safe search, including each engine, from `safe_search`.
3. Removes external filter lists. Those URLs are not restored later, so a rollback cannot re-enable a third-party list.
4. Sets upstreams, bootstrap, an empty fallback, the blocking mode, and the TTL. `null_address` is AdGuard `null_ip`.
5. Replaces user rules with the rendered document and clears the cache.

It does not change the client access list (§8.2). It reads the configuration back, then sends a synthetic A query for up to four block names and four allow names. A block must be NXDOMAIN, REFUSED, or `0.0.0.0` according to the blocking mode. An allow must return a real address and must not be the block signal. Probe names are not included in errors (§16.4).

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

The scenario applies a document, expects `blocked.example.com` (and a subdomain) to be NXDOMAIN on UDP and TCP from both resolvers, expects `safe-target.example.com` to answer `192.0.2.15`, checks that the query log and statistics are off, withdraws, and expects neither name to return an address.

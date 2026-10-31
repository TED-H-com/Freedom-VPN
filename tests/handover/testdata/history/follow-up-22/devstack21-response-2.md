## dns-publish / members — 2026-10-31 (follow-up 21, F9-INFLIGHT, D-102)

The shared trace now writes a request start before dispatch, retains outstanding
requests and joins later completions by identity across the entire closed-case
cache-change sequence. The [export contract](scenarios/harness-exports.md#follow-up-21--installer-input-and-dns-lifetime-correlation-test-only)
defines cumulative history, three outstanding sets, receipt continuity and
per-role snapshot bounds. Current-query `trace_complete` cannot rule out an
earlier outstanding request. A reply between queries remains visible in the
next snapshot even if the last query starts no fetch.

[BACKEND §2.1.5](../dns/BACKEND.md#215-f10-03--positive-cache-and-ttl-measured-2026-10-31)
is adopted: fetch alone is not fallback; different uniquely matched responders
establish independent caches; failure/timeout plus secondary receipts establish
fallback. Same-resolver changed class without a new fetch requires inspecting
older outstanding requests **and their completion times**. Missing history
keeps the cause ambiguous. Hit wording stays inferred, negative policy synthesis
stays distinct, and post-query authority SOA is not a cached positive serial.
The pin's F9-INFLIGHT replacement limitation remains a product finding.

[Final DNS run cba88bbc](runs/2026-10-31/followup21/cba88bbc-a7bf-49f8-9dab-3182123d02aa/dns-history-integrity.md)
passes **13/0/0**, with **36/36** complete query traces, **85** distinct authority
requests and **70** identities retained across query snapshots. The positive
sequence remains primary fetch / inferred primary hit (TTL 9) / post-expiry
primary fetch. This validates the export; it does not reproduce F9-INFLIGHT.

The fixture explicitly retains its historical **4 KiB (`4096`) cache** for
comparability, not the pin's 4 MiB default. Both resolvers' actual cache/mode/
timeout values are read before and after the focused run. The relay's 2-second
exchange deadline and AdGuard's live upstream timeout are distinct. No cache,
timeout, policy or TTL assertion is changed, and no TTL-fix claim is made.
`cce1a688`, the F10-03 historical failure and follow-ups 16–20 are untouched.
The [114-file D-098 pin set](followup20-archive.sha256) was committed first.

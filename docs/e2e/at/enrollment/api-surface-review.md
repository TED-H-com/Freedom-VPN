# Enrollment API surface review — follow-up 9

2026-10-31. Public APIs provide token issuance, ETag/If-Match revocation,
same-node replacement, incarnation history, policy/grant/capacity setup and
publication. External revocation counts only after route/identity step read-back.
The installer runs exclusively through devstack wrappers.

D-116/#302 and D-118/#312 are exercised with strict four-chain enforcement,
a separate hookless ownership marker, held bytes and stopped rerun equality.
The current presenter uploads immutable original events under their retired
stream selector. Fresh assertions verify canonical bytes/hashes, once-only
samples, before-plus-held totals, stored-result replay and independent ACKs.

D-130/#330/#331 now permits the fresh gap to reach ACK 1, then the counted
550/358-byte event to reach ACK 2. Read-only central interval/sample/checkpoint
queries, API totals and coherent stopped SQLite copies corroborate the result.
No SQL transition, fabricated observation or manual installer invocation is used.

See [results](summary.md), [gap proof](current-stream-gap.md),
[AT20](../usage/AT20.md), [responses](../../RESPONSES.md) and [limits](requests.md).

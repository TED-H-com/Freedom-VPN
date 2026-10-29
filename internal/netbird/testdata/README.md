# TEST ONLY fixtures — NOT A SECRET

Recorded through the real HTTP API of the digest-pinned NetBird CE **0.82.0**
combined server on 2026-10-29. `route.json`, `group.json`, `peer.json`,
`policy.json` and `nameserver.json` preserve response fields/types. The
`_fixture` annotation is added by the scrubber, not returned by NetBird.
All upstream IDs, user IDs, peer names, host metadata, MACs and addresses are
replaced with TEST ONLY identifiers and documentation/overlay addresses; times
are fixed. The five fixture files contain no credentials or cryptographic keys.

Reproduce with the dev stack and `FVPN_NETBIRD_RECORD=1` integration run, then
`python3 -B deploy/dev/netbird/scrub-recordings.py`. Review every field manually.
The httptest fake merges real response templates with desired request fields,
models group/policy response ID objects, and injects lost responses or stale
read-back. Setup-key, user and minimal additional peer responses are synthetic
TEST ONLY values, not misrepresented as recordings of secrets.

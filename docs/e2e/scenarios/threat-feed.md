# Threat feed delivery (§12.5, AT22)

`threat-feed-central` approves and fetches a deterministic ten-entry hosts feed
from the isolated TLS origin, verifies normalized content and resolver-document
digests, and rejects a wrong-pin revision with last-good retention, an audit and
worker metrics. It reactivates the good feed afterward: a profile pinned to an
inactive feed is intentionally ineligible for node delivery.

`threat-feed-node` uses D-061 exactly as declared in `node/agent/dnsconfig.go`:
`dns.resolvers` contains `id`, `admin_url`, and `dns_addr`, and
`dns.resolver_credentials_file` points at the `{"username","password"}` JSON.
Only `agent-run.json` includes DNS. The entrypoint makes the regular credential
file root-owned mode 0600. AdGuard shares the node network namespace, so its
admin listener is `http://127.0.0.1:8080` and DNS is `127.0.0.1:53`. The central
resolver address is the node's lab interface. Query logs and statistics stay
disabled; the readback records only counts/hashes and a synthetic answer flag.

The current compiler emits resolver addresses but no `dns.resolver_document`.
The scenario waits for issuance, requests the actual published document through
the node's mTLS identity, and records **404 NOT_FOUND** and HTTP 200 node detail
with `dns_policy_evidence: null`. The authenticated local resolver readback is
HTTP 200, zero rules, and an unblocked synthetic response. **Owner:
w3-wire-nodes / internal/desiredstate**, which must select and issue the
reference and retain its issuance for handler authorization.

Node application and the tampered-document/last-good negative case remain
blocked on that same issuance path. No document reference is fabricated, no
signed desired state is edited and no projector row is seeded. When issuance
lands, the positive case must verify document ID/generation/digest/size, the
agent's `applied` evidence, the node-detail projection and an effective synthetic
block. Only after that can a same-length corrupted response demonstrate
`HASH_MISMATCH`, failed evidence and unchanged last-good rules. A central
wrong-pin failure alone does not prove that node behavior.

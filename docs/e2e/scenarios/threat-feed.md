# Threat feed delivery (§12.5, AT22)

`threat-feed-central` approves and fetches a deterministic ten-entry hosts feed
from the isolated TLS origin, verifies normalized content and resolver-document
digests, and rejects a wrong-pin revision with last-good retention, an audit and
worker metrics. It reactivates the good feed afterward and polls the active
revision and profile eligibility before node delivery. A profile pinned to an
inactive feed is intentionally ineligible for node delivery.

`threat-feed-node` uses D-061 exactly as declared in `node/agent/dnsconfig.go`:
`dns.resolvers` contains `id`, `admin_url`, and `dns_addr`, and
`dns.resolver_credentials_file` points at the `{"username","password"}` JSON.
Only `agent-run.json` includes DNS. The entrypoint makes the regular credential
file root-owned mode 0600. AdGuard shares the stable node network namespace, so its
admin listener is `http://127.0.0.1:8080` and DNS is `127.0.0.1:53`. The central
resolver address is the node's lab interface. Query logs and statistics stay
disabled; the readback records only counts/hashes and a synthetic answer flag.

`threat-feed-node` independently polls a verified signed reference and the
projected `dns_policy_evidence`. Pass requires matching document ID, generation
and SHA-256 with `outcome=applied`, provisioning `ready`, a real node-mTLS GET
whose body matches the signed size/digest, and an effective synthetic DNS block.
The runtime `dns` object is recorded without credential contents. The node and
AdGuard join the stable `node-netns` namespace so node restart preserves their
shared loopback listeners. Follow-up 14 (#127) supplies the periodic document
reconcile that was missing in follow-up 7.

`threat-feed-tamper` is a separate result. It records the positive delivery gate,
current last-good readback, whether a fault was injected, and its owner. A central
wrong-pin failure is never substituted for corrupted node-mTLS response evidence.
It remains blocked until a response corruption hook can present changed bytes
under the authentic signed digest and show failed evidence with unchanged
last-good rules. No signed reference or projector row is fabricated.

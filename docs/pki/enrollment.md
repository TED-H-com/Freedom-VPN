# Enrollment and proof protocol

## State and identity (§6.1–2, §14.5, §15.5.1)

`fvpn_enr_` followed by unpadded base64url of 32 cryptographically random bytes
is the enrollment token format (52 characters including the prefix). The
SHA-256 of the entire ASCII token is the only stored token representation.
It is supplied only in `Authorization: Enrollment <token>` over HTTPS. The
recognizable prefix is for secret scanners; no real token belongs in an example.

The contract calls the unused state `pending`. Terminal transitions are:

```text
pending -> consumed
pending -> expired (also derived at read/attempt time from the deadline)
pending -> revoked
```

Consumed enrollments retain immutable request ID, original public-key
fingerprint, request hash, incarnation, certificate and operation references.
They cannot be revoked as enrollments: revoke the node identity instead.
Get responses contain state/binding/expiry, metadata and the read-only stored
token digest `token_sha256` (`api.Enrollment.TokenSha256`, `*string`: 64-character
lowercase hex from `encode(token_hash, 'hex')`, distinct from
`bound_public_key_sha256`), never token plaintext (ENR-02, contract revision 10).
Admin create and its idempotent replay return the same `api.Enrollment` digest;
replay returns `token=null` and `secret_available=false`. If the first response
was lost, revoke the unused enrollment and create a new one with a new
idempotency key.

First use locks the reserved node and enrollment row, validates token state and
expiry, creates the incarnation, stores the public certificate, and records a
provisioning operation/outbox event in one transaction. The node becomes
`registered`, provisioning remains `applying`, and publication is `unpublished`.
The worker must provide evidence before setting `ready`. No second human
admission approval is introduced. A uniqueness failure rolls back everything.

An administrator can set `EnrollmentCreateRequest.node_id` to authorize AT07
reinstallation on the same Node. Creation locks that Node `FOR UPDATE`: a
missing Node returns `404 RESOURCE_NOT_FOUND`, a retired Node returns
`409 NODE_RETIRED`, and any unrevoked incarnation returns
`409 NODE_INCARNATION_ACTIVE`. It reserves only a new enrollment; it does not
clear lifecycle, revocation timestamps, journal denials or publication intent.
Existing exit metadata and historical identities remain attached to the Node.

`Enroll` checks again under the node lock, then inserts a fresh incarnation and
certificate, creates the provisioning operation, and consumes the bound
enrollment before updating `nodes.current_incarnation_id` and lifecycle to
`registered`. The operation/outbox and this transition commit together. A second
pending enrollment cannot replace the first one's active incarnation. Registered
Nodes whose old incarnation was separately revoked are also eligible; retirement
remains terminal. Reinstallation retains disabled publication until separately
authorized publishing and successful provisioning.

Migration `00111_pki_reinstall.sql` installs the narrow `pki_guard_terminal`
exception for `revoked` → `registered`: the pointer must select a different,
unrevoked incarnation belonging to the same Node, with its unrevoked certificate
and a matching consumed enrollment from the current transaction. The enrollment
must have been consumed before expiry and have the same public-key fingerprint;
no other incarnation may remain unrevoked. Bare lifecycle updates and heartbeat
reports have no such evidence and are rejected. `retired` cannot transition via
`revoked` to bypass this rule. Existing certificate and incarnation revocation
timestamps are never cleared. Goose Down restores the previous stricter guard
without deleting identity history; existing reenrollments stay recorded.

Replay still requires a valid CSR self-signature and request proof. The same
request ID, incarnation ID and key return the original certificate and
operation. A newly randomized CSR signature from that key is accepted;
initial metadata remains unchanged. A different key or request/incarnation ID
returns `ENROLLMENT_KEY_MISMATCH`. An expired token cannot be replayed, and a
revoked identity or expired/overlap-ended original certificate cannot be revived.
No reissue is necessary for a lost response; returning the stored certificate
keeps serial history bounded. Normal renewal has its own serial receipts.

D-029 audit records use UUID actors, `node:<uuid>` or `enrollment:<uuid>`
resources, and the generated `succeeded` result for committed local changes.
`enrollment.created` records `token_sha256` alongside `reason` in
`changes.after` (`After: {"reason": "", "token_sha256": "<64-hex>"}`,
`Fields: ["reason", "token_sha256"]`); `enrollment.consumed` records only
`reason` (ENR-02). Property names ending in `_sha256` are digests and are never
redacted by `internal/audit`, whereas `token`, `access_token`, `refresh_token`
and `token_hash` remain redacted.
Bootstrap claims refer to the stable Node; its incarnation remains in the
identity and bootstrap tables. Administrator service callers supply
`admin:<uuid>` (or a bare canonical administrator UUID), never a display label.
Operations use generated kinds `node.enroll`, `node.revoke` and `node.retire`,
start pending with required steps, and share the command's request ID with its
audit row. Enrollment's operation has `netbird_setup`, `peer_readback` and
`node_policy` steps. Local audit success does not assert that external work has
finished; the worker transitions operation steps only with read-back evidence.

## Exact proof format

The OpenAPI `EnrollRequest` description is authoritative:

1. Build the complete request with `enrollment_request_id`,
   `node_incarnation_id`, `csr_pem`, `platform` (including `os_version`) and
   `agent_release`. Do not include `proof` in the signed object.
2. Encode it with RFC 8785 JCS using `pkg/canonjson`. PEM newlines are part of
   the string: changing LF to CRLF changes the proof input. Property order and
   insignificant JSON whitespace do not change it. This typed Go API uses
   canonical lowercase hyphenated UUID strings; adapters should enforce that
   spelling before dispatch, or preserve raw JSON for proof verification.
3. Compute SHA-256 of those canonical UTF-8 bytes.
4. Sign the digest with the CSR's ECDSA P-256 key, using ASN.1 DER `(r,s)`
   encoding (`ecdsa.SignASN1`), and encode the result using padded standard
   base64. It is not base64url and not a raw 64-byte `r || s` signature.
5. Set `proof` to that base64 string. `VerifyProof` checks both signatures and
   derives SHA-256 of DER SubjectPublicKeyInfo itself.

The token is deliberately not in the signed object because the contract puts
it in Authorization. The proof binds the CSR, installation identity, request
ID, platform and agent version; TLS protects transport. It is not a substitute
for protecting the node private key. Unknown/oversized fields belong at the
schema-aware HTTP boundary; the service independently checks required IDs,
supported platform (Ubuntu 24.04, amd64/arm64), pinned release and CSR/proof bounds.

The public interoperability vector is
[`proof-v1.json`](../../internal/enrollment/testdata/proof-v1.json): an actual
CSR, proof, expected canonical string, SHA-256 digest and public-key fingerprint.
It is labeled **TEST ONLY - NOT A SECRET**. Its random key was discarded;
no private key or usable token is stored. `TestAT06_ProofVectors` verifies the
vector and rejects changed platform/CSR and noncanonical base64. Runtime tests
generate independent keys and use a separate map-based proof encoder.

## Worker event and bootstrap delivery (§6.4, §14.7)

`node.provision.requested`, schema version 1, has the exported Go payload
`enrollment.ProvisionEvent`:

```json
{
  "schema_version": 1,
  "node_id": "93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb",
  "incarnation_id": "d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0",
  "operation_id": "695a0f3b-99c3-48bc-97cf-9688f22e18e5",
  "one_off": true,
  "usage_limit": 1,
  "ephemeral": false,
  "expires_in_seconds": 86400,
  "group_role": "staging"
}
```

All UUIDs above are specification examples. The worker uses the stable
operation ID to correlate remote creation and reads back external key/peer
state before retrying. Exactly one local event is created per enrollment;
external at-least-once delivery still requires worker reconciliation. Never
put setup-key plaintext or external API credentials in an outbox payload.

`StoreBootstrap` verifies the active incarnation and provisioning operation,
then stores AES-GCM ciphertext, provider encryption-key ID, external setup-key
ID and bounded expiry. AAD is JCS of purpose `netbird-bootstrap-v1`, node,
incarnation, operation, external key ID and UTC expiry. PostgreSQL microsecond
precision is normalized before encryption. Moving ciphertext between nodes
or changing its bound metadata makes authentication fail.

`TakeBootstrap` authenticates the supplied TLS peer certificate, serializes
against node revocation, and atomically clears ciphertext/encryption-key ID
before decrypting outside the transaction and returning plaintext. At most
one caller succeeds. The expiry verdict comes only from PostgreSQL while the
row is locked: absent material gives retryable `409 BOOTSTRAP_NOT_READY`,
previously delivered material gives `410 BOOTSTRAP_CONSUMED`, and material
expired before delivery gives `410 BOOTSTRAP_EXPIRED`. Expiry erases the secret
without setting `delivered_at`, so later calls remain expired. Slow decryption
after a committed delivery cannot change that call to a 410. Crypto/response
failure after erasure does not restore the secret. Expired secrets are erased
on retrieval or by `ExpireBootstrap`.
Store retries for the same external key/operation do not resurrect erased data;
a different external key is a conflict. The external ID and processing metadata
remain for read-back. This deliberately handles a lost **enroll** response
differently from a lost **one-time bootstrap** response.

No plaintext is included in `EnrollResponse`. D-015 places single delivery at
`POST /agent/bootstrap`, over mTLS, with `Cache-Control: no-store`. See
[the wiring contract](../nodes/WIRING.md) for error codes and replay behavior.

## Revocation and conflicts

`RevokeNode` checks resource version and command idempotency, persists a terminal
incarnation journal denial, revokes every certificate/incarnation, cancels pending
enrollments, erases bootstrap material and records route/peer withdrawal work.
The durable journal survives an older PostgreSQL restore. A journal write
followed by a DB transaction failure intentionally leaves authentication denied;
retry the same revoke command to finish DB state and withdrawal work. A new
heartbeat, installer run, renewal or enrollment replay cannot undo this.

`ObserveEndpoint` and `ObserveEndpointTx` (`internal/enrollment/revoke.go`) use a
server-derived endpoint compatibility domain (D-072), never an untrusted body ID
or raw forwarding header:
- **Ingress callers**: On the mTLS agent listener (`internal/httpapi/middleware.go`),
  `httpapi.New` records the verified leaf certificate (`httpapi.ClientCertificate`)
  and the coarsened compatibility domain (`httpapi.ClientCompatibilityDomain`) in
  request-local metadata. The two per-request transactional agent write handlers—lease
  renewal (`POST /api/v1/agent/lease-renewals`) and observation submission
  (`POST /api/v1/agent/observations`)—call `Enrollment.ObserveEndpointTx` inside
  their request transaction before any automatic identity-mapping mutation. GET
  routes (`GET /api/v1/agent/desired-state`, `GET /api/v1/agent/resolver-documents/{id}`)
  never touch `node_identity_endpoints` or `node_identity_conflicts`.
- **Compatibility domain & limitation**: The client address is derived by the
  existing `requestRateKey` trusted-proxy logic (`X-Forwarded-For` is honoured
  only when the TCP peer is in configured `trusted_proxies`; otherwise `RemoteAddr`
  is used) and coarsened to **IPv4 `/24`** or **IPv6 `/48`** before server-side
  SHA-256 hashing into `endpoint_hash`. Raw IP addresses and CIDR strings are
  never stored or logged. Limitation: two clones operating behind the same `/24`
  or `/48` compatibility domain are not distinguished by this check.
- **Roaming vs. interleaving (`DefaultIdentityActiveWindow = 120s`)**: The default
  120 s active window is larger than the 60 s D-067 provisional lease renewal cap
  and the 30 s heartbeat interval (so a renewing or heartbeating node always
  refreshes inside the window) and shorter than a finite-member serving lease.
  A single switch `X → Y` (`Y` first seen while `X` was seen inside `activeWindow`)
  is treated as **roaming**: `X` remains the mapped `endpoint_hash` in
  `node_identity_endpoints`, `Y` is tracked as the candidate, and `Y` is promoted
  to `node_identity_endpoints` only once `X` has been quiet for a full
  `activeWindow`. **Interleaving**—`X` seen again while `Y` is still inside
  `activeWindow` (`X → Y → X`), `Y` and `X` both seen within one window twice, or
  a third incompatible domain `Z` seen while `X` and `Y` are inside the window—records
  one row in `node_identity_conflicts`, sets `mapping_frozen = true` (enforced
  irreversibly by database trigger `pki_mapping_frozen`), and emits one
  `node.identity.conflict` audit event on `node:<node_id>` with `changes.after`
  containing only `incarnation_id`, `conflict_count`, and `observed_at` (no
  hashes, addresses, or domains; later requests while frozen emit no additional
  audit event). The triggering HTTP request still returns its normal 2xx response.
- **Freeze honors & operator reconciliation**: Once `mapping_frozen = true`, all
  automatic external identity mapping updates (`node.peer.observe`,
  `node.peer.discover`, `node.provision.requested`, `EnsureNodePeer`,
  `DiscoverNodePeer`, and `PublishExit`) skip mutation (no last-writer-wins), and
  `SubmitObservation` applies the D-055 terminal incarnation state
  (`provisioning_state = 'failed'`, `safety_status = 'failed'`,
  `member_forwarding_observed = false`). This is intended: a cloned private key
  is an identity compromise. Reconciliation uses the existing operator path—`revokeNode`
  followed by a new enrollment (`createEnrollment` + `enrollNode`)—which creates
  a fresh unfrozen incarnation while preserving the historical conflict rows on
  the retired incarnation. There is no unfreeze operation.


An explicitly authorized reinstall uses the same stable Node and a new incarnation.
Incarnation journal tombstones are permanent; an old token/proof/certificate never
revives them. Migration 00111 permits only a newly consumed
bound enrollment to transition revoked→registered. Existing node-level journal
denials remain fail closed and require operator reconciliation; they are never
silently erased. Retirement cannot be undone.


## Management peer correlation (D-055)

The worker passes the stable enrollment **operation ID** to the NetBird adapter.
Its one-off, usage-limit-one, non-ephemeral setup key auto-enrolls the peer into
exactly two groups: shared `fvpn-unpublished-nodes` and an empty
`fvpn-enroll-peer-<hash>` unique to that attempt. The latter is only a correlation
record, never a policy/route authorization group. `node.peer.discover` reads
management's key usage and that group's membership, requires exactly one use
and one machine peer, and records `peer_readback` only after confirmed binding
and removal of the temporary group. A null agent `netbird.peer_id` therefore
requires no guessed identifier. Setup-key revocation also detaches and removes
the temporary group with read-back. Unknown/ambiguous management evidence never
binds a peer: transient absence retries within the worker budget; conflicts
freeze the incarnation and retain a safe operation/node/audit reason. Existing
bootstrap encryption, single delivery and revocation fences remain unchanged.
See [worker discovery and failure semantics](../nodes/WIRING.md#worker-jobs-and-read-back).

D-055 withdrawal now handles revocation before discovery too: a fresh one-use key
and a sole non-user, non-ephemeral correlation member permit peer deletion, with
GET 404 required before the correlation group is discarded. Multiple/unsafe
members preserve the group and fail with an operator-visible fixed reason.
After discovery, the durable attempt→peer withdrawal receipt preserves the same
proof through lost central commits. Revoking only the key is never reported as
having disconnected an enrolled peer (§5.2). Legacy one-group keys require explicit
revoke/re-enrollment; their shared staging membership cannot identify one node.

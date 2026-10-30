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
Get responses contain state/binding/expiry and metadata, never token plaintext.
Admin create replay returns the same enrollment with `token=null` and
`secret_available=false`. If the first response was lost, revoke the unused
enrollment and create a new one with a new idempotency key.

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

`ObserveEndpoint` uses a server-derived endpoint compatibility domain, not an
untrusted body ID or full destination list. A different hash within the active
window records a conflict and permanently freezes its mapping row, returning
`false` for all later automatic updates. The original endpoint is retained;
no last-writer-wins replacement happens. This does not infer physical device
identity from IP/MAC. Adapter policy must distinguish ordinary roaming from
incompatible simultaneous activity. The worker must honor this freeze before
mutating its external identity mapping. Resolution is explicit operator work;
there is no automatic unfreeze or new remote-shell capability.


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

# Key rotation and restore runbook

References: §13.4, §16.2, §17.4 item 1. All production lifetimes and overlap
durations must be chosen explicitly; the test constants are not production
recommendations.

## Central signing keys

1. Keep the provider vault and its monotonic state ledger outside PostgreSQL
   snapshot restore scope. Restrict the directory to the service account.
   Store current revocation records in an independently recoverable place.
2. Call `FileProvider.Rotate(ctx, purpose, validity, overlap)` for one of
   `desired-state`, `lease`, `recovery` or `ledger-epoch`. The provider generates
   a fresh key and atomically switches the active reference. The previous key
   becomes retired immediately; it is retained for verification until the
   earlier of its original expiry and rotation time plus overlap.
3. Export `TrustBundle` and deliver it through an already authenticated trust
   channel. Never accept trust from the same unauthenticated response as an
   unknown artifact. A signing-key rotation and a node trust-bundle rollout
   need coordination: a node lacking the new key fails closed. This service
   does not claim that writing a bundle means all nodes have accepted it.
4. Retain old public key records, including expiry and revocation. An old signer
   object selects the current purpose key on each call; it cannot pin a retired
   key. Restoring old private-key files does not override the current ledger.
5. For compromise, call `Revoke(ctx, keyID)` and distribute the updated bundle
   urgently. Revocation is terminal. Rotate to a fresh key if appropriate.
   Reverify authorization; an old signature is not ongoing permission to serve
   or install a release.

There is no transition from retired/revoked back to active and no snapshot
import API. Revisions are unsigned decimal strings and never wrap. File writes
are atomic and fsynced under a cross-process lock; a crash may leave an orphan
private-key file, but an unreferenced file is never a signer. Remove orphaned
files only after checking the current ledger under an operational maintenance
window. Do not delete ledger tombstones as routine cleanup.

## Release verification trust

Central services must never load a release signing private key. Provision the
initial trusted public key through `PinReleaseRoot` using an independently
verified installer/release trust root. This operation succeeds only once for a
new vault. Key IDs for release trust may be opaque strings; generated central
key IDs are UUIDs.

Subsequent `UpdateReleaseTrust` input is a `pkg/envelope` envelope signed by a
currently trusted release key. Its signed `ReleaseUpdate` has:

- `kind: "fvpn-release-trust-v1"`;
- a strictly increasing decimal-string `revision`;
- `keys`: public `KeyRecord` entries with `purpose: "release"`, validity,
  state and optional `revoked_at`.

The old trust store verifies the envelope before any update. Existing public
keys cannot be substituted, validity windows cannot expand, revoked keys
cannot be reinstated, and omitted old keys remain recorded. Store updates are
atomic. Use a two-step overlap: first authorize the new public key with the
existing key and distribute it; then authorize old-key retirement/revocation
with the new trusted key. A revoked/expired known signer invalidates the whole
authorization envelope, even when another signature would otherwise pass.
Emergency recovery from loss of every trusted root requires an independently
authorized trust-root recovery process, not acceptance of an unsigned update.

## CA and encryption keys

`Rotate(node-ca, ...)` creates a new provider-held P-256 key. Provision a matching
public CA certificate and construct a new `pki.CA`. The current PKI service is
configured for one root CA; a fleet-wide CA rollover requires explicit trusted
root distribution and adapter routing for old/new issuers, not just replacing
the signing reference. Plan the transition before retiring an issuing CA.
Development root certificates come from `CreateDevCA`; production CA issuance
and validity policy must be supplied by operations. Certificate overlap within
one CA is handled by `Renew` and the per-serial effective deadline.

`Rotate(bootstrap-secret, ...)` changes the encryption key for new ciphertext.
Retired encryption keys may decrypt already-stored bounded-lifetime secrets;
revoked encryption keys never decrypt. Do not destroy an old encryption key
until retained ciphertext is erased. If compromise requires immediate
revocation, pending deliveries fail closed and need worker reconciliation.
Nodes never receive these keys or the CA signing key.

## Database restore and authority loss

1. Stop admission/agent authorization and automatic external workers before
   restoring a database. Do not treat snapshot state as current authorization.
2. Recover the current provider ledger and independent node revocation journal
   first. `FileProvider` supplies both in its state file; keep its storage at
   the current revision. Do **not** restore that directory with an old database.
3. Reconcile enrollment consumption, certificate history, incarnation mappings,
   pending/erased bootstrap state, and NetBird operation/external key/peer IDs
   against newer audit/evidence before enabling service. Expire unresolved
   enrollments and erase unresolved bootstrap ciphertext. Never issue another
   remote setup key merely because an older DB snapshot lacks an external ID.
4. Reapply revocations and finish pending withdrawal operations. Local journal
   denials take precedence over database rows that still say registered.
5. Resume only after service identities and revocations are confirmed. Restore
   of historical usage is separate and does not authorize a revoked node.

A local file provider cannot detect rollback of **its own entire storage** by
an administrator or attacker, nor can it protect against host-root compromise.
No cryptographic scheme can infer newer state if every copy of it was rolled
back. Production KMS/HSM keys therefore require independently monotonic key
status/revocation storage or a nonrollbackable checkpoint, and the node denial
journal needs the same recovery discipline. This task implements the development
provider, not a production anti-rollback appliance. Multi-instance central
deployment must share one locked local vault/journal or use an appropriately
fenced production authority; divergent copies are not supported.

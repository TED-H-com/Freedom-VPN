# Key rotation and restore runbook

References: §13.4, §16.2, §17.4 item 1. All production lifetimes and overlap
durations must be chosen explicitly; the test constants are not production
recommendations.

## Central signing keys

1. Keep the vault and its monotonic ledger outside PostgreSQL snapshot restore
   scope, restricted to the service account, with independently recoverable
   revocations. Run `fvpn-vault inspect -vault EXISTING_VAULT` and obtain the
   ledger revision and the purpose's active key ID (or confirm there is none).
   Choose validity and overlap explicitly; neither has a default or a
   recommended production value.
2. Run `fvpn-vault-keys rotate -vault EXISTING_VAULT -purpose PURPOSE
   -expect-revision N -expect-active-key KEY_ID|none -validity DURATION
   -overlap DURATION` on one shell line, substituting the inspected values.
   Allowed purposes are `desired-state`, `lease`, `recovery` and `ledger-epoch`.
   Check the exit code and [result document](keys.md). The previous active key
   retires immediately; its verification window is capped at the earlier of
   its original expiry and rotation time plus overlap (with the existing
   `not_before + 1ns` minimum for an otherwise empty window). An overlap must
   cover documents signed by the previous key that nodes still hold; it cannot
   extend the key's original expiry. Central services sign with the new key
   immediately on their next signing call. Until a node holds the new bundle,
   it rejects new desired state, leases or recovery authorizations signed by
   that key. Rotation does not stage the new key for a later activation.
3. For desired-state, lease or recovery, run
   `fvpn-vault export-trust -vault EXISTING_VAULT -out NEW_FILE`
   ([inspection and export](inspect.md)), then deliver that public bundle
   through an already authenticated trust channel; follow the
   [node trust files and process actions](../node/install.md#node-trust-files).
   Never accept trust from
   the same unauthenticated response as an unknown artifact. Ledger-epoch is
   central-only: its rotation does not change the node bundle.
4. Confirm actual adoption and verification on the intended nodes. Writing or
   delivering a bundle alone does not establish adoption. Retain old public
   records, capped windows and revocations; account for retained documents
   before retiring their verification window. Existing provider signer handles
   select the current key on every signing call; stored envelopes can still
   name an earlier key. Follow the reference inventory below.
5. To revoke, inspect again, then run `fvpn-vault-keys revoke -vault
   EXISTING_VAULT -key-id KEY_ID -expect-purpose PURPOSE -expect-revision N`
   on one shell line. Add `-revoke-active` only when intentionally removing
   the active signer; the purpose then has no signer until a fresh rotation
   with inspected revision and `-expect-active-key none`. For compromise,
   distribute the updated revocation bundle urgently and confirm adoption
   using the [node trust files and process actions](../node/install.md#node-trust-files)
   (ledger-epoch has no node bundle change). Revocation is terminal; preserve
   its original time. Reverify authorization: an old signature is not ongoing
   permission to serve or install a release. See [all outcomes](keys.md) before
   repeating any command, especially after lost output or uncertain durability.

There is no transition from retired/revoked back to active and no snapshot
import API. Revisions are unsigned decimal strings and never wrap. File writes
are atomic and fsynced under a cross-process lock; a crash may leave an orphan
private-key file, but an unreferenced file is never a signer. The command's
`not_applied` outcome reports its key ID if it created such a file; the file
can be incomplete and is never removed automatically. Remove orphaned
files only after checking the current ledger under an operational maintenance
window. Do not delete ledger tombstones as routine cleanup.

## Key references outside the vault

The production composition uses the FileProvider for all four purposes, with
no central Ed25519 key-ID field in service configuration. No configuration edit
or service restart is needed to select a rotated central signer. This is based
on the production wiring below, not on the behavior of arbitrary injected
signers. The commands do not update any of these references automatically.

| Purpose | Production signing path and operator updates | Retained references |
|---|---|---|
| `desired-state` | `internal/nodes/httpwire/state.go` uses its purpose signer and `Runtime.Provider.SignEnvelope` for HTTP and worker compilation. No signer-ID setting changes; export/deliver/confirm the new node trust bundle. | `desired_state_envelopes.envelope` and node-persisted desired-state envelopes (agent store and updater journal) retain `signatures[].key_id`. A still-valid unchanged cached desired state can be served without a new signing call; rotation does not rewrite it. |
| `lease` | The same runtime constructs `leases.Service` with a provider purpose signer. No signer-ID setting changes; export/deliver/confirm node trust. | `serving_leases.key_id`, stored lease signatures/envelopes and node-persisted lease envelopes keep the issuing key ID. |
| `recovery` | `apps/control/wire_usage.go` and `apps/worker/jobs_usage.go` construct `usage.ProviderSigner`, which calls the provider per signature. No signer-ID setting changes; export/deliver/confirm node trust. | Signed usage checkpoints name their recovery key in `signatures[].key_id`. On a node, `ledger_epochs.checkpoint_json` in `node/internal/usagestore` stores that recovery-signed checkpoint; despite the table name, it is not signed with a ledger-epoch key. |
| `ledger-epoch` | The same usage wiring selects the provider per signature. No signer-ID setting changes, and no node bundle changes. Retain current central public key/revocation records for any central verification of historical recovery records; update any separately managed verifier trust manually. | `usage_recovery_records.signing_key_id` and `signed_envelope.signatures[].key_id` retain the issuing ledger-epoch key. |

`internal/config/usage.go`'s `SigningConfigFile`/`signing_config_file` is passed
as the vault directory to `OpenFileProvider`, not as a key ID. Nodes configuration
uses `provider_directory` for its vault. Ensure the inspected vault is the one
used by the relevant service; the command does not establish this for you.

The in-memory `desiredstate.SigningKey.KeyID` (`internal/desiredstate/interfaces.go`)
and `usage.Ed25519Signer.KeyID` (`internal/usage/signer.go`) do pin their keys.
Repository production composition does not construct those implementations;
they are used by fixtures/injected consumers. A custom consumer that chooses
them must arrange its own signer replacement; this command cannot rotate its
in-memory private material. The four production paths above need no such code
change.

`policy_revisions.signing_key_ref` (`internal/policies/store.go`,
`db/migrations/00002_nodes.sql`) is a configurable provenance reference with
default `central-policy-key`, not necessarily a vault key ID and not the active
signer selector. No change is required by the default wiring. If an operator
configured it to mean a concrete key ID, update the external reference supplied
for future policy revisions deliberately; preserve historical revision records.
`releases.signing_key_ref` names release provenance and is outside central-key
rotation.

The vault's `keys[].id`, exported trust's `keys[].key_id` and the initializer's
receipt also name keys. Vault state changes only through provider operations;
export a new trust file, retain old public records and preserve the receipt as
historical ceremony evidence rather than rewriting it. None of the stored
signed envelopes should be relabeled with a new key ID: their signature bytes
were made with the original key.

`node_bootstrap_secrets.encryption_key_id` names the bootstrap encryption key,
not a central Ed25519 signer. `Nodes.CAKeyID` / `ca_key_id` in
`internal/config/nodes.go` names the node CA key. Neither is changed by these
commands. Their rollovers still require the separate Go API procedures below.

## Release verification trust

Central services must never load a release signing private key. Provision the
initial trusted public key through [`fvpn-vault-init`](initialize.md) (or
`PinReleaseRoot`) using an independently verified installer/release trust root.
This operation succeeds only once for a new vault. Key IDs for release trust
may be opaque strings; generated central key IDs are UUIDs.

Subsequent release-root updates are `pkg/envelope` envelopes produced offline
with `fvpn-release trust-update` ([release process](../updates/release-process.md))
and signed by a currently trusted release key. Each signed `ReleaseUpdate` has:

- `kind: "fvpn-release-trust-v1"`;
- a strictly increasing decimal-string `revision`;
- `keys`: public `KeyRecord` entries with `purpose: "release"`, validity,
  state and optional `revoked_at`.

To apply a signed release trust update at the centre and perform a root
handover:

1. Run [`fvpn-vault inspect -vault EXISTING_VAULT`](inspect.md) to read the
   current `ledger_revision` and `release_revision`, and run
   `fvpn-vault export-trust -vault EXISTING_VAULT -out CURRENT_TRUST_FILE` to
   obtain a fresh public trust export.
2. Check the candidate update file offline against that fresh export and the
   inspected `release_revision`:
   `fvpn-release trust-verify -update UPDATE_FILE -trust CURRENT_TRUST_FILE -current-revision RELEASE_REVISION`.
   Confirm the signer, target record(s), `update_sha256` and
   `usable_root_after=true`.
3. Apply the update at the centre under the inspected `ledger_revision`:
   `fvpn-vault-keys apply-release-update -vault EXISTING_VAULT -update UPDATE_FILE -expect-revision LEDGER_REVISION`
   (see [command reference and outcomes](keys.md)). Add `-allow-no-usable-root`
   only for deliberate emergency removal of the last usable root. Check the exit
   code and the JSON result document (`outcome`, `update_sha256`, `change.keys`
   and `change.usable_root_key_ids`).
4. Run `fvpn-vault export-trust -vault EXISTING_VAULT -out NEW_TRUST_FILE` and
   deliver the new node trust store (`trust.json`) to nodes through an already
   authenticated channel. Nodes consume the exported trust store, never the
   update envelope. Follow the
   [node trust files and process actions](../node/install.md#node-trust-files).
5. Confirm actual node adoption across the fleet before starting the second step
   of a root handover. Only after nodes trust the newly added root should you
   produce, verify, apply and distribute a second update (signed by the new
   root) that retires or revokes the previous root.

The vault's current release records verify the envelope before any change.
Existing public keys cannot be substituted, validity windows cannot expand,
revoked keys cannot be reinstated, retired keys cannot become active again, and
omitted old keys remain recorded. A revoked or expired known signer invalidates
the whole authorization envelope, even when another signature would otherwise
pass. Because signature verification runs before the `already_applied` check,
repeating an applied update whose signer is still within `not_after` (including
an update that retired its own signer and is repeated before that `not_after`)
returns `already_applied` (exit `0`), whereas repeating an applied update that
revoked its own signer or whose signer's `not_after` has passed is `rejected`
with `reason: "signature"` (exit `10`), and `fvpn-release trust-verify` against
a fresh export rejects it too (`KEY_UNTRUSTED` for the revoked or expired
signer, or `update revision must exceed current revision` when
`-current-revision` is set to the applied `release_revision`). After an output
failure (exit `9`), a lost document or a rejection on a possible repeat, do not
repeat blindly: run `fvpn-vault inspect -vault EXISTING_VAULT` and compare
`release_revision` and the stored `release` records (`key_id`, `state`,
`not_before`, `not_after`, `revoked_at` and `public_sha256`) with the update's
`signed.revision` and `signed.keys` to recognise whether the update is already
applied.

What `fvpn-vault-keys apply-release-update` does **not** do:

- It binds the update to previous vault state only through `-expect-revision`
  (the signed update envelope itself carries only its proposed new
  `release_revision`, not a hash of prior trust state).
- It does not export the trust bundle or deliver anything to nodes.
- It provides no recovery path when every trusted release root is lost or
  revoked; emergency recovery from loss of every root requires an independently
  authorized trust-root recovery process, never acceptance of an unsigned update.

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

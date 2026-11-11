# Initialize a local FileProvider vault

`fvpn-vault-init` turns the existing central key ceremony into a one-shot command
for an operator-controlled staging environment. It uses the local development
FileProvider, not a KMS/HSM or a production anti-rollback authority. The provider
cannot detect rollback of its entire directory, and divergent replicas are not
supported. See [rotation and restore](rotation.md) before choosing its storage.

The command does not deploy services, contact providers, enroll nodes, generate
release private keys, rotate existing keys or resume partial initialization.
For existing central Ed25519 keys, use the separate
[conditional rotate and revoke commands](keys.md).
All policy inputs are explicit. A successful receipt proves local key operations
and generated artifacts; it does not prove target deployment or node adoption.

## Prepare the inputs

Build from the repository root:

```sh
make -f mk/pki.mk pki-build
```

This produces `bin/linux_amd64/fvpn-vault-init` and
`bin/linux_arm64/fvpn-vault-init`. Run the appropriate binary as the intended
service account in a trusted, operator-owned parent directory on persistent
storage. The destination must not exist, including as a symlink. Do not create
the destination first; its parent must already exist. No privileged service or
host configuration is performed by this command.

Generate the release signing key separately on the offline release workstation
with `fvpn-release keygen`. Independently verify the public trust document and
its key ID/fingerprint before transferring **only that public document** to the
central host. The initializer accepts exactly one current, nonrevoked,
release-only public root using the existing envelope trust-store format. It
rejects release private-key JSON. It does not authenticate the operator's trust
choice; compare the public fingerprint in the receipt with the independently
verified release root.

Write a JSON configuration. This complete example is **TEST ONLY**, not a
recommended production or business policy. Replace paths, CA common name and
all validity/overlap values with explicit decisions for the intended staging
ceremony. `release_trust_file` must be an absolute path. Durations use Go duration
syntax, must represent whole seconds and must fit `time.Duration`; validity is
at least one second, overlap is nonnegative and shorter than validity.

```json
{
  "provider": "file",
  "release_trust_file": "/secure/staging/release-trust.json",
  "release_key_id": "test-release",
  "ca": {
    "common_name": "TEST ONLY staging CA",
    "validity": "24h"
  },
  "purposes": {
    "node-ca": {"validity": "48h", "overlap": "0s"},
    "bootstrap-secret": {"validity": "24h", "overlap": "0s"},
    "desired-state": {"validity": "24h", "overlap": "0s"},
    "lease": {"validity": "24h", "overlap": "0s"},
    "recovery": {"validity": "24h", "overlap": "0s"},
    "ledger-epoch": {"validity": "24h", "overlap": "0s"}
  }
}
```

Exactly these six central purposes are required; release signing is never a
central purpose. Unknown or duplicate JSON fields, trailing data, omitted inputs
and invalid windows are rejected before destination creation. The CA common
name must contain 1–64 UTF-8 bytes without control characters or surrounding
whitespace. The command creates an operator-configured self-signed P-256 root
through the provider-held key; it does not call or relabel `CreateDevCA` and does
not accept an arbitrary external CA chain. CA validity cannot exceed the
node-ca key validity. The ceremony start precedes key creation, so its CA expiry
cannot run past that key's expiry.

## Initialize and inspect

For example, after preparing the parent directory and configuration:

```sh
bin/linux_amd64/fvpn-vault-init \
  -config /secure/staging/ceremony.json \
  -out /var/lib/fvpn/staging-authority
```

A successful command prints a short completion message and creates:

| Path within the destination | Contents |
|---|---|
| `vault/` | Existing FileProvider ledger, lock and six private key files. |
| `agent-ca.pem` | Public self-signed node CA certificate. |
| `trust.json` | Node public trust for desired-state, lease, recovery and release. |
| `receipt.json` | Final local completion record, public key IDs/windows/fingerprints and checks. |

Directories are created with mode0700; output files and provider private files
use mode0600 (the invoking umask can further restrict access). The receipt paths
are relative to the destination. Its `public_sha256` values hash the public
bytes stored by the provider: raw Ed25519 bytes for signing/release roots,
PKIX DER for node-ca. The CA certificate fingerprint hashes certificate DER;
the bundle fingerprint hashes the written JSON bytes. No private key, seed or
bootstrap encryption bytes appear in public artifacts or command output.

Before publishing `receipt.json`, the command signs and verifies with all four
central Ed25519 purposes, performs a bootstrap encryption/decryption roundtrip,
creates/verifies the CA signature and provider-key match, and reads back public
outputs. The node bundle excludes central-only ledger-epoch and all CA/encryption
private material. Provider open or metadata export alone is not a usable-key
check.

Use the receipt's CA key ID with the generated certificate and `vault/` directory
in the existing nodes configuration. Set the usage signing provider to the same
current authority. Choose runtime node certificate and renewal windows
separately. Deliver `trust.json` through an already authenticated channel and
verify actual node adoption. File creation is not runtime acceptance.

For an existing vault, use [`fvpn-vault inspect` and `export-trust`](inspect.md)
to inspect current metadata, optionally check key use, or publish a new public
node trust file without changing the vault.

## Failure and interruption

An existing destination is always refused, including repeat/concurrent attempts;
no completed purpose is rotated and no existing file is replaced. On a failure
after destination creation, retain the directory for manual inspection. The
command does not delete the vault, retry the ceremony or silently resume it.
Inspect the current ledger and the original failure before planning a separate
recovery action. Never remove a vault merely to make a retry succeed.

Each public output is written and synced to a temporary file, closed, then
published using a same-directory hard link that refuses an existing final name.
The receipt is published last, after its complete bytes are closed. A failure
before that publication cannot leave a final completion receipt. Temporary-name
cleanup after publication is best effort; a fully published receipt remains the
completion record even if a temporary link survives. A stdout failure after
publication returns failure but reports that the completed receipt remains.

This is not a transaction across all files or a power-loss durability guarantee.
An interruption can leave partial provider state, orphan keys or temporary files;
absence of a completion receipt requires inspection. A later conditional rotation
can also report `not_applied` with an unreferenced key ID; follow the
[manual orphan-file procedure](rotation.md#central-signing-keys), with no automatic
deletion or retry. A receipt records that the
ceremony completed at its timestamp, not that files remain unmodified or keys
remain valid forever. Preserve the current authority/revocation state separately
from PostgreSQL restore scope; follow the existing rotation/restore runbook for
maintenance and recovery. Do not restore an old vault with an old database.

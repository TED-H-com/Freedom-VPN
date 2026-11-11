# Inspect an existing local FileProvider vault

`fvpn-vault` provides read-only inspection and public node trust export for an
existing development/staging FileProvider. References: §13.4, §16.2, §17.2 and
§17.4 item 1. It never initializes, repairs, rotates, revokes or pins keys, and
never creates or rewrites files inside the vault. Use
[initialization](initialize.md) for a new vault and [rotation and restore](rotation.md)
for the separate Go API maintenance procedures.

FileProvider cannot detect rollback of its entire directory, host-root
compromise or divergent copies. These commands do not change those limits.
Keep the current authority and revocation records outside PostgreSQL restore
scope. An older readable vault is not evidence of current authority.

## Build and invoke

From the repository root:

```sh
make -f mk/pki.mk pki-build
```

The command is built as `bin/linux_amd64/fvpn-vault` and
`bin/linux_arm64/fvpn-vault`. Run it as the intended service account, using
operator-controlled directories on persistent storage.

Exactly two subcommands are available:

```text
fvpn-vault inspect      -vault EXISTING_VAULT [-check-keys] [-lock-timeout DURATION]
fvpn-vault export-trust -vault EXISTING_VAULT -out NEW_FILE [-lock-timeout DURATION]
```

| Flag | Meaning |
|---|---|
| `-vault` | Required existing vault directory. No default or environment fallback. A raw `..` path element is a usage error. After cleaning, only the final component must not be a symlink; ancestor symlinks are followed. |
| `-check-keys` | Inspect only; default false. Exercise current active central keys. `-check-keys=false` explicitly selects metadata only. |
| `-out` | Export only; required new file whose parent already exists. A raw `..` path element is a usage error. No overwrite or directory creation. |
| `-lock-timeout` | Positive Go duration, default `5s`; fractional values such as `250ms` are allowed. |

Repeated flags, unknown flags or subcommands, missing values, stray positional
arguments and nonpositive/invalid durations are usage failures. Parsing finishes
before accessing any vault or output path.

Both `-vault` and `-out` reject any `..` path element in the raw flag value,
before cleaning or filesystem access. The check is on complete elements, so
`..data` and `a..b` are valid names. This avoids lexical cleaning of `symlink/..`
selecting a different vault or destination from the one the OS resolves. Leading,
trailing and lone `..` elements are also rejected, including with trailing
separators. `.` elements, repeated separators and trailing separators are still
cleaned. A final vault symlink remains refused with a trailing `/` or `/.`.

Components above the final vault directory are resolved as the operating system
resolves them. The operator is responsible for the path above the vault, including
linked or mounted storage. `export-trust` compares resolved vault and destination
locations, so an alias cannot place an export inside the vault or its descendants.

These invocations are **TEST ONLY** examples with placeholder paths:

```sh
bin/linux_amd64/fvpn-vault inspect \
  -vault /secure/staging/authority/vault
bin/linux_amd64/fvpn-vault inspect \
  -vault /secure/staging/authority/vault -check-keys -lock-timeout 2s
bin/linux_amd64/fvpn-vault export-trust \
  -vault /secure/staging/authority/vault \
  -out /secure/staging/distribution/trust-next.json
```

The provider uses its existing exclusive lock even for reads. A running service
can contend with these commands. A single deadline covers initial open,
snapshots and all probes together; it is not restarted for each lock or purpose.
The provider polls lock contention every 5 ms. The deadline bounds lock waits,
not uninterruptible operating-system IO or output publication. No command retries
a changed inspection. A timeout is distinct from an unavailable vault.

Before opening the provider, the command holds `lock` through a Linux
`O_PATH | O_NOFOLLOW | O_CLOEXEC` descriptor and validates that held file's type
and permission bits with `fstat`. It must be regular with no group or other bits.
This neither follows a final symlink nor opens contents, blocks on a special
file, takes a lock, or changes anything. Symlinks and special files are refused.
The descriptor stays open until the command ends, preventing reuse of its inode
if the lock is deleted. Checkpoints compare a fresh `fstat` of the held file
against `Lstat` of the current path and the initially validated permission bits.
A lock deleted and created again is therefore detected, even with identical
content and mode. The command checks again after
the provider is open and the first snapshot is taken, and immediately before
writing a result document or invoking export publication. The final check follows
all probes and the last snapshot, where applicable. A missing, nonregular,
replaced (`os.SameFile` differs) or permission-changed lock produces exit 5 with
no document and no export file, including for metadata inspection and export.
Failed provider reads also check for a changed lock before reporting their error.

These checks detect a replacement that is in place at one of the three points.
Someone who can write inside the vault directory can replace the lock and put
it back between them, and can equally replace the ledger and key files. The
command does not defend against the vault's owner or system administrator and
does not bind the provider's own open to the validated file. Running services
use the provider directly, which does not perform these lock-file checks. The
provider's locking and validation remain unchanged.

## Metadata inspection

`inspect` without `-check-keys` reads public ledger metadata under the provider
lock and touches no private key file. Its snapshot includes ledger revision,
release revision, public key records, the count of denied identities and the
node bundle from the same locked read. Denied identity identifiers are never
returned.

The JSON document has `kind: "fvpn-file-vault-inspect-v1"`, `provider: "file"`,
`inspected_at`, `ledger_revision`, `release_revision`, `denied_identity_count`,
`key_use_checked`, `keys`, `purposes` and `result`. Revisions are unsigned decimal
strings. All times are UTC RFC 3339 with `Z`, preserving fractional seconds.

Every key includes `key_id`, `purpose`, `state`, `not_before`, `not_after`,
optional `revoked_at`, optional `public_sha256`, `window_status` and
`key_use_status`. States are the provider's `active`, `retired` and `revoked`.
Fingerprints are lowercase SHA-256 of raw Ed25519 public bytes or node-ca PKIX
DER, matching the initializer. Bootstrap-secret has no public fingerprint.
Validity windows are half-open `[not_before, not_after)`:

| `window_status` | Meaning |
|---|---|
| `current` | The inspection instant is within the window; this alone says nothing about revocation or private material. |
| `not_yet_valid` | The instant precedes `not_before`. |
| `expired` | The instant is at or after `not_after`. |

Each purpose summary contains `purpose`, `metadata_status`, `key_use_status`
and `result`. A central purpose passes metadata only with an active key whose
window contains the instant. Release passes with any non-revoked root in its
window, including a retired root still in overlap.

| `metadata_status` | Meaning |
|---|---|
| `pass` | A qualifying key is in its window. This is a metadata result only. |
| `not_yet_valid` | A qualifying non-revoked key exists, but its window has not started. |
| `expired` | A qualifying non-revoked key exists, but its window has ended. |
| `revoked` | No qualifying non-revoked key exists and a revoked record is present. |
| `no_active_key` | Central purpose has no active key and no revoked record; absent or retired records alone cannot sign. |
| `no_key` | No release root is recorded. |

When release has several records, summary precedence is `pass`,
`not_yet_valid`, `expired`, `revoked`, then `no_key`. Individual records retain
their own state and window status. For central purposes, retired keys are
ignored by metadata selection; an active key's window takes precedence over
older revoked records.

Without `-check-keys`, **every** key and purpose use status is `not_checked`,
and `key_use_checked` is false. A `pass` result and exit 0 prove only a consistent
ledger read with passing metadata. They do not establish that private keys
exist, match or can be used.

## Key-use inspection

With `-check-keys`, the command exercises each active, current central key using
the existing provider operations. It signs and verifies an in-memory probe for
desired-state, lease, recovery and ledger-epoch, performs a bootstrap seal/open
round trip, and verifies a CA signature over a probe digest with the snapshot's
P-256 public key. It never creates a certificate. Probes are domain separated
and omit consumer-required document fields; their signatures and ciphertext
are never printed or written. Checks require the snapshot's key ID and, where
there is a public key, verification against that public key.

| `key_use_status` | Meaning |
|---|---|
| `not_checked` | No key-use check was requested. |
| `not_applicable` | Checks were requested, but the key is retired, revoked, outside its window or a public-only release root; also used for a purpose with no eligible key. |
| `passed` | The eligible central key passed its actual provider operation and verification. |
| `failed` | The eligible central key failed its use check; no cause is inferred. |

A failure for one purpose does not prevent checking the others. Missing material,
wrong permissions, mismatched signing material and a window ending between calls
can have the same outward result. Bootstrap has no public-key binding: its
round trip proves the current named symmetric key can seal/open, not that it
matches historical ciphertext or an independently known secret.

A second locked snapshot follows the probes. If either revision or any key
record differs, the command emits no document and exits 5. Denied-identity count
changes are also rejected. If the second snapshot cannot be read, the command
emits no document and reports unavailability or timeout. No retry occurs.

Each purpose's `result` and the overall `result` are `pass` or `fail`. Overall
pass requires passing metadata for all seven purposes. With checks enabled,
it also requires `passed` use checks for all six central purposes. Retired and
revoked keys are never exercised or reported as usable. A passing document
records local evidence at its inspection instant, not future validity or what
any node currently trusts.

## Public trust export

`export-trust` takes one snapshot for its metadata and bundle. It reads no
private key files and performs no key-use check. The bundle contains only
desired-state, lease, recovery and release, including retained and revoked
public records. It excludes ledger-epoch, node-ca, bootstrap-secret and denied
identity identifiers. An empty bundle is refused. Missing current keys do not
prevent exporting a nonempty bundle: distributing revocation urgently remains
possible even when a purpose has no current non-revoked key.

The bundle is byte-compatible with the initializer's `trust.json`:
`json.MarshalIndent(bundle, "", "  ")` followed by exactly one newline. It is
parsed through `envelope.ParseNodeTrustStoreJSON` before creating any file.
The entire bundle and receipt are built in memory first.

The output's parent must exist. Its resolved directory must be outside the
vault and all its descendants, including through directory symlinks. Existing
regular files, directories, symlinks and dangling symlinks are refused. The
command opens the resolved destination directory, writes a mode0600 temporary
file there, syncs and closes it, then publishes via a hard link that refuses an
existing name. It reads the final file back and compares all bytes before
printing the receipt. A write, sync, close or link failure leaves no new final
name; a failed read-back removes the command's newly published name. Existing
targets remain unchanged, including one that wins a concurrent link race.
Temporary-name cleanup is best effort.

Use a trusted destination directory that other actors will not move or modify
during publication. Filesystem damage, external directory replacement or loss
of permission can defeat cleanup. This is not a transaction across files or a
power-loss guarantee. An interruption can leave a temporary link or a published
file without a receipt; inspect such artifacts before planning another export.

The receipt has `kind: "fvpn-file-vault-export-trust-v1"`, `provider: "file"`,
`exported_at`, `ledger_revision`, `release_revision`, `output` (the base name
only), `sha256` (the exact written bytes), `key_use_checked: false`,
`private_key_files_read: false`, `keys` and `purposes`. Each exported key includes
its ID, purpose, window, revoked flag and `public_sha256`. Each of the four node
purposes reports `current_non_revoked_key` as a boolean at the export instant.
False is informational and does not fail the export.

Both documents emit object fields in the order described above, keys sorted
lexicographically by key ID, and summaries in fixed purpose order. Inspection
uses node-ca, bootstrap-secret, desired-state, lease, recovery, ledger-epoch,
release; export uses desired-state, lease, recovery, release. The trust bundle
itself also sorts by key ID. Output is one indented JSON document plus a newline.

Deliver the bundle through an already authenticated trust channel, then verify
actual node adoption. A successful publication receipt proves local byte
publication only. It is neither future validity nor node adoption, and it does
not authenticate a restored vault as the latest authority.

## Failure and exit codes

| Exit | Meaning |
|---|---|
| `0` | Inspection result is pass, or trust publication and receipt output succeeded. |
| `1` | Inspection completed consistently, but its JSON result is fail. |
| `2` | Usage failure; no vault or output access. |
| `3` | Could not inspect: vault unavailable, or inspection could not be encoded. |
| `4` | Provider/lock deadline exceeded; no result document. |
| `5` | Lock identity/type/permissions changed during any invocation, or ledger metadata changed during checked inspection; no document or export file. |
| `6` | Could not prepare/encode/export public trust. |
| `7` | Stdout failed. After export, the verified published file remains. |

Stderr contains only fixed messages prefixed `fvpn-vault:`: the usage line,
`vault unavailable`, `vault lock deadline exceeded`, `cannot encode inspection`,
`changed during inspection`, `cannot prepare public trust export`,
`cannot encode export receipt`, `public trust publication failed`, `output failed`,
or `output failed; published file remains`. No raw OS/provider errors, operator
paths, contents, key material or ciphertext are printed. Unavailable vaults
initially include missing/wrong paths, missing lock/state, invalid lock types, forbidden
permissions and invalid ledgers. No group or other permission bits are permitted
on the vault directory, the lock or the state file. With `-check-keys`, the
provider also checks the private key files' type and permissions as it reads
them; a private-file failure is an inspected-and-failed use check, not evidence
of a specific cause. Metadata inspection and export never read those files.
The command does not guess which condition caused a provider failure.

Exit 1 includes the completed JSON result and no stderr. Other failures emit no
stdout, except that a stdout failure can leave a partial JSON document at the
consumer. In that case the exit code must be checked; a partial document is not
a receipt. A stdout failure after publication explicitly reports that the file
remains, and another export to the same name will be refused.

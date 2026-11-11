# Conditional central signing-key commands

`fvpn-vault-keys` changes an existing development/staging FileProvider vault.
It implements the central signing-key part of §13.4, §16.2, §17.2 and §17.4;
see the [rotation and restore procedure](rotation.md). Build with
`make -f mk/pki.mk pki-build`; binaries are
`bin/linux_amd64/fvpn-vault-keys` and `bin/linux_arm64/fvpn-vault-keys`.
Run as the vault's service account on persistent storage.

Exactly three subcommands are available:

```text
fvpn-vault-keys rotate -vault EXISTING_VAULT -purpose PURPOSE -expect-revision N -expect-active-key KEY_ID|none -validity DURATION -overlap DURATION [-lock-timeout DURATION]
fvpn-vault-keys revoke -vault EXISTING_VAULT -key-id KEY_ID -expect-purpose PURPOSE -expect-revision N [-revoke-active] [-lock-timeout DURATION]
fvpn-vault-keys apply-release-update -vault EXISTING_VAULT -update FILE -expect-revision N [-allow-no-usable-root] [-lock-timeout DURATION]
```

Obtain the expected ledger revision and active key ID from
[`fvpn-vault inspect -vault EXISTING_VAULT`](inspect.md). Copy them into the
arguments deliberately. These commands never discover expectations, refresh
stale ones, retry a mutation, export trust, distribute files or change service
configuration. A successful command is local vault evidence, not evidence that
any node has adopted new trust.

## Flags and validation

| Flag | Meaning |
|---|---|
| `-vault` | Required existing vault directory; no default or environment fallback. |
| `-purpose` | Rotate only: required `desired-state`, `lease`, `recovery` or `ledger-epoch`. |
| `-expect-purpose` | Revoke only: required purpose from the same four values; must match the named key. |
| `-expect-revision` | Required canonical unsigned decimal string, `0` through `18446744073709551615`; no sign, whitespace or leading zeroes. |
| `-expect-active-key` | Rotate only: required lowercase canonical UUID, or literal `none` to require no active record. An expired active record still counts as active. |
| `-validity` | Rotate only: required positive Go duration. No default or recommended production value. |
| `-overlap` | Rotate only: required nonnegative Go duration. No default or recommended production value. |
| `-key-id` | Revoke only: required lowercase canonical UUID. |
| `-revoke-active` | Revoke only: explicit permission to revoke the active record. Default false; `-revoke-active=false` also means false. |
| `-update` | Apply-release-update only: required regular file (at most `1 MiB` / `1048576` bytes, `canonjson.MaxBytes`) containing a signed release trust update envelope from `fvpn-release trust-update`. |
| `-allow-no-usable-root` | Apply-release-update only: explicit permission to apply an update that leaves no release root usable at the decision instant. Default false; `-allow-no-usable-root=false` also means false. |
| `-lock-timeout` | Optional positive Go duration; default `5s`. |

Durations can be fractional (including nanoseconds), must fit Go's
`time.Duration`, and use Go duration syntax; `0` is valid only for overlap.
UUID syntax is checked, not UUID version bits. Missing, repeated or unknown
flags, stray arguments, invalid values and forbidden purpose arguments are
usage failures before any filesystem access. For `apply-release-update`, a
missing, non-regular (for example a directory or FIFO), unreadable or oversized
(`> 1 MiB`) `-update` file is also a usage failure before the vault is opened;
once read, the file is validated against `contracts/schemas/trust/release-update-envelope.schema.json`
(`packaging/release.ValidateTrustUpdateEnvelope`) before opening the vault, and
a schema failure returns `outcome: "rejected"` (`reason: "schema"`, exit `10`)
without opening the vault. A key whose actual purpose is forbidden in `rotate`
or `revoke` is refused under lock even if an allowed expected purpose was given.
`node-ca` and `bootstrap-secret` cannot be changed with this tool; `release`
records are changed only through `apply-release-update`.

A raw `-vault` value containing a complete `..` path element is a usage failure,
before cleaning or access, including leading, trailing, repeated-separator and
`symlink/../vault` forms. Names such as `..data` remain valid. `.` elements and
repeated/trailing separators are cleaned. The final directory component must
not be a symlink, including paths ending in `/` or `/.`; this is an unavailable
vault. Symlinks above the final component are followed as the operating system
resolves them. Operators are responsible for those parent paths.

## Preconditions and state transitions

Rotation checks the revision, allowed purpose and exact active ID (or absence)
under the same cross-process lock as the write, before generating key material
or creating a file. Refusal leaves the vault unchanged. The new key is active
immediately with `[rotation_time, rotation_time + validity)`. The previous
active key becomes retired, with expiry equal to the earlier of its current
expiry and `rotation_time + overlap`. If that value is not after its start,
expiry becomes `not_before + 1ns`, preserving the existing provider rule. A
retired window never grows; overlap cannot extend a key's original lifetime.
An identical repeated rotation is refused and creates no second key. Concurrent
commands with identical expectations permit only one ledger change.

Revocation checks, in order:

1. The key exists, its purpose matches, and that purpose is allowed.
2. An already-revoked key returns `already_revoked` without changing anything,
   regardless of the supplied revision or `-revoke-active`.
3. Otherwise, the ledger revision must match.
4. Revoking an active record requires `-revoke-active`.

A retired key can be revoked without that flag. Revocation is terminal and its
recorded time never changes on repetition or later rotation. Revoking the
active key leaves the purpose without a signer. After inspecting the new
revision, rotation with `-expect-active-key none` can create a fresh signer;
it never reinstates the revoked key. Ledger revisions increment once per
change and never wrap.

`apply-release-update` validates the update file against the embedded schema
before opening the vault, then checks under the vault lock, in order:

1. The envelope signature verifies against the ledger's `release` records
   (active, retired and revoked) for purpose `release` at one reading of the
   provider's clock.
2. If the ledger's `release_revision` equals the update's `revision` and every
   record in the update is already stored in the ledger identically, the call
   returns `already_applied` without changing anything, regardless of
   `-expect-revision` or `-allow-no-usable-root`.
3. Otherwise, the ledger's `revision` must equal `-expect-revision`
   (`precondition_refused` otherwise).
4. Payload, revision (`update.revision > release_revision`) and per-record
   transition rules must hold (purpose `release`, unique IDs and public keys,
   `not_before < not_after`, immutable public bytes for existing IDs,
   non-expanding validity windows, terminal revocation with immutable
   `revoked_at`, no `retired` to `active` transition), and at least one release
   root must remain usable at the decision instant unless `-allow-no-usable-root`
   is set (`rejected` otherwise, with reason class `signature`, `payload`,
   `revision`, `record` or `no_usable_root`).
5. Every record in the update is applied, `release_revision` takes the update's
   revision, the ledger `revision` increases by one, and the ledger is saved.

Central services choose the current key on each new signing call. For
desired-state, lease, recovery and release, nodes lacking the updated trust
bundle reject signatures made with the new key or continue accepting an unrevoked
copy. Export, authenticate delivery and confirm adoption manually as described in
[rotation.md](rotation.md). Overlap covers documents signed by the previous key
that nodes still hold; it does not teach nodes to trust the new key. Ledger-epoch
keys are central-only and do not change the node bundle. Revocation takes effect
on node verification only when the node has the updated revocation information.

## Outcomes and next actions

When a conditional call returns an outcome, stdout receives exactly one
indented JSON document followed by a newline. These outcomes have empty
stderr, including nonzero outcomes. Exit 0 means a newly applied mutation or
successful no-change repetition (`outcome: "already_revoked"` or
`"already_applied"`). The command never exits with 1.

An exit status alone is not an outcome. Accept an outcome only together with
its documented output: the stdout document with empty stderr, or, for outcomes
without a document, the fixed stderr line and no stdout. Any other combination
(for example status 2 from a Go runtime failure or a status set by a wrapper)
must be handled like `unknown`: inspect before acting, never repeat blindly.
For exit 9, the fixed output-failure line identifies the lost outcome class;
stdout may contain a partial document, which is not a receipt.

| Exit | `outcome` | Meaning and next action |
|---|---|---|
| `0` | `applied` (rotate) | New key active, ledger save completed. Preserve the result and perform the required trust rollout. Do not repeat the rotation. |
| `0` | `applied` (apply-release-update) | Release trust update applied, ledger save completed. Export the trust bundle (`fvpn-vault export-trust`), deliver it to nodes, and confirm adoption before the next step of a root handover. |
| `0` | `revoked` (revoke) | Key revoked, ledger save completed. Distribute revocation urgently for node-held purposes. If the active key was revoked, the purpose has no signer until a fresh rotation. |
| `0` | `already_revoked` (revoke only) | Successful no-op. Recorded revocation time is returned unchanged. Preserve current revocation records and confirm trust rollout as needed. |
| `0` | `already_applied` (apply-release-update only) | Successful no-op. The update's `release_revision` and all of its records are already stored identically. Preserve the document and confirm trust rollout as needed. |
| `5` | `precondition_refused` | Nothing changed. No vault contents are echoed. Run `fvpn-vault inspect` (and for release updates check the update again with `fvpn-release trust-verify` against a fresh export) and decide on a new action; do not automatically refresh arguments. |
| `6` | `not_applied` | Failure before ledger replacement was attempted. If `unreferenced_key_id` is present, a private file was created and left behind, possibly incomplete. Inspect before retrying; do not remove anything automatically. |
| `7` | `durability_not_confirmed` | Applied/revoked state was observed in a fresh locked snapshot at the expected next revision, after a save error at/after rename. Preserve evidence and investigate storage durability. Do not rotate, revoke or re-apply blindly. |
| `8` | `unknown` | Replacement may have occurred; read-back failed or showed something else. No state is inferred. Inspect the vault and reconcile concurrent activity manually. Do not repeat blindly. |
| `10` | `rejected` (apply-release-update only) | Nothing changed. The update is not acceptable for this vault (`reason` is `schema`, `signature`, `payload`, `revision`, `record` or `no_usable_root`). Inspect the vault and check the update with `fvpn-release trust-verify` against a fresh trust export. |

`not_applied` with an unreferenced file points to the
[orphan-file procedure](rotation.md#central-signing-keys) and
[initialization failure guidance](initialize.md#failure-and-interruption).
An unreferenced file is never selected for signing. Under a maintenance window,
check the current ledger before deciding whether to remove it. Neither command
removes key files, imports state, un-revokes keys or deletes tombstones. A retry
with still-current expectations can succeed despite an orphan, leaving exactly
one active ledger key; the orphan remains until manual review.

A save error at the rename boundary is treated conservatively, including an
error returned by rename itself. Only a read-back matching the attempted change's
complete public record(s) and the expected next revision (plus `release_revision`
for `apply-release-update`) yields exit 7. A later concurrent mutation or another
revision can make the outcome unknown even if this command's change did occur.
Read-back is not a retry and cannot prove power-loss durability.

## Result documents

`kind` is `fvpn-file-vault-rotate-v1`, `fvpn-file-vault-revoke-v1` or
`fvpn-file-vault-apply-release-update-v1`.
All documents contain `kind`, `outcome` and a fixed `message`. Every
`apply-release-update` document also includes `update_sha256` (the lowercase hex
SHA-256 of the canonical envelope without trailing newline, matching
`fvpn-release trust-update` and `trust-verify`), and `rejected` documents
include `reason` (`schema`, `signature`, `payload`, `revision`, `record` or
`no_usable_root`). Documents never contain vault paths, private material,
private file contents or raw error strings. Revisions are JSON decimal strings;
times are UTC RFC 3339 with `Z`, preserving fractional seconds. Windows are
half-open.

Applied, revoked, already-revoked, already-applied and durability-not-confirmed
documents have `change`:

| Field | Contents |
|---|---|
| `old_revision`, `new_revision` | Ledger revisions before/after the change. For `already_revoked` and `already_applied`, both are the current revision and are equal. |
| `old_release_revision`, `new_release_revision` | `apply-release-update` only: release revisions before/after the change. For `already_applied`, both equal the stored `release_revision`. |
| `key` | `rotate` / `revoke` only: new active key (rotate), or revoked key (revoke). |
| `previous_key` | `rotate` / `revoke` only: Rotate: previous key **after** retirement/window capping, or `null`. Revoke: record **before** revocation. Already-revoked: `null`; previous state is not inferred. |
| `keys` | `apply-release-update` only: array of `{key, previous_key}` for every record in the update. `previous_key` is the stored record before the update (`null` for newly added keys and on `already_applied`, where previous state is not inferred). |
| `usable_root_key_ids` | `apply-release-update` only: sorted array of release root key IDs usable at the decision instant after the update (`[]` when none remain under `-allow-no-usable-root`). |
| `node_trust` | `nodes_require_new_bundle` for desired-state, lease, recovery and release; nodes need updated trust before accepting new-key signatures or enforcing new revocations. `node_bundle_unchanged` for ledger-epoch. |
| `purpose_has_no_signer` | `rotate` / `revoke` only: present as `true` only when this command revoked an active key. Otherwise omitted; absence makes no assertion about available signers. |

Each public key object has `key_id`, `purpose`, `state`, `not_before`,
`not_after`, `public_sha256`, and `revoked_at` when revoked. The fingerprint is
lowercase SHA-256 of raw Ed25519 public bytes, matching inspect/initialize.
No raw public key is needed in the result; use authenticated trust export for
public key distribution. A refusal or rejection has no `change` or vault key
identifiers.

`not_applied` optionally has `unreferenced_key_id`, only if exclusive file
creation succeeded. `unknown` has `read_back` of `failed` or `different` (plus
`attempted_key_id` for `rotate` and `revoke`). The identifier names the
attempted change, not confirmed state. `durability_not_confirmed` has
`read_back: "matched"` and the change details. No document says a node has
adopted the result.

## Failures without a document

| Exit | Fixed stderr message after `fvpn-vault-keys: ` | Next action |
|---|---|---|
| `2` | The usage line containing the three command syntaxes above, separated by ` \| `. | Correct arguments; the vault was not opened. |
| `3` | `vault unavailable` | Inspect storage/permissions and current authority. Nothing is created or repaired. |
| `4` | `vault lock deadline exceeded` | Resolve contention and inspect before deciding whether to retry. This invocation changed nothing. |
| `9` | `result output failed: OUTCOME; run fvpn-vault inspect; do not repeat blindly` | Run `fvpn-vault inspect` next. The change is not undone. |

`OUTCOME` is exactly the lost document's fixed `outcome` value from the table.
The prefix and a newline are always present. These messages form a closed set;
no OS/provider error text, flags, paths or key facts are appended. Exit 9 also
covers a short write, encoding failure and a closed stdout pipe. SIGPIPE is
ignored so a closed pipe produces this exit and message rather than signal
termination. A writer may already have received a partial document; check the
exit code. Output failure is never success, even if the mutation completed.

Unavailable includes missing vault/lock/state, a final vault symlink, invalid
ledger data and group/other permission bits on the directory, lock or ledger.
The pre-open lock check holds a Linux `O_PATH | O_NOFOLLOW | O_CLOEXEC` reference
without following a final symlink or opening contents. It requires an existing
regular file with no group or other permission bits, refuses symlinks, FIFOs,
directories and other special files, and keeps the reference alive until exit.

The conditional provider call separately checks the **file actually held
locked**, before evaluating preconditions: it must be the same identity as
`Lstat(vault/lock)`, regular, and have no group/other permission bits. Failure is
a typed lock-validation error reported as unavailable, with no mutation. An
acceptable replacement made before acquiring the lock is simply the current
vault lock. There are no later lock-reference rechecks in the mutating commands.
Existing provider methods retain their previous lock handling.

The deadline is shared by provider open and the conditional call. It bounds
waiting for the lock, not completion of filesystem IO once the lock is held.
A deadline that expires during a locked mutation does not cancel the save or
change its outcome. After an uncertain save, the sole follow-up is one locked
read-back snapshot with a **fresh** deadline of the same length. Read-back
failure, including its timeout, yields `unknown`, not the no-change timeout exit.

## Limits and recovery discipline

A party who can write in the vault directory can replace the lock after the
in-lock validation, and can equally replace the ledger and key files. These
commands are no defence against the vault's owner or root. FileProvider remains
a development/staging provider, with no protection against whole-directory
rollback, host-root compromise or divergent copies. Keep current key status and
revocations independently recoverable and outside database restore scope; an
old vault must not resurrect authority (§17.2, §17.4).

These commands do not prove real node adoption, runtime recovery, hardware
key protection or crash/power-loss durability. They provide no staging or
scheduled activation, no automatic overlap calculation, and no CA/bootstrap-secret
rollover. `apply-release-update` binds an update to the vault's current state
only through `-expect-revision` (the signed update envelope itself does not bind
previous trust state), does not export or distribute trust to nodes, and
provides no recovery path if every release root is lost or revoked. Choose
windows explicitly, account for retained signed documents, distribute trust over
an authenticated channel and confirm actual node behavior by hand. See
[key references outside the vault](rotation.md#key-references-outside-the-vault)
for which records and configuration remain unchanged.

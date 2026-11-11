# Release process

`fvpn-release` (`packaging/release/cmd/fvpn-release`) builds, signs, verifies and packs a node release. It runs on the build machine. It is not installed on nodes. Nodes hold the public release trust store only (§13.3, §16.2).

The release key is Ed25519 and its purpose is `release`. It is a different key from desired-state, lease and recovery keys. A key file whose purpose is not `release` is refused. The private key file is mode 0600. It never goes to a node, a package, a container image or the trust store that nodes ship.

Test keys in `contracts/vectors/test-keys.json` are labeled `TEST ONLY - NOT A SECRET`. Production keys are generated on the build machine and stay there.

## Layout of a release

`build-manifest` reads two artifacts, an SBOM, component versions and compatibility, and writes the canonical JSON of the `signed` object. `release_id` is a UUID. `product_version` is display text and is not a security decision. The schema requires both an `amd64` and an `arm64` artifact. The tool also requires both to be Ubuntu 24.04 `tar.zst`. A release missing either is not installable.

`sign` wraps that object in the envelope `{"signed","signatures"}` using `pkg/envelope`. The signature input is the RFC 8785 canonical JSON of `signed`. `manifest_sha256`, the value desired state carries, is the SHA-256 of the canonical JSON of the whole envelope, not of `signed` alone.

`verify` checks the envelope against a public trust store at a chosen time. The trust window is half-open `[not_before, not_after)`. A revoked, expired, not-yet-valid or wrong-purpose key fails closed. Optional `-artifact arch=path` checks that file's size and SHA-256 against the manifest.

`package` writes a reproducible `tar.zst` of a release tree: paths are sorted, owners are root, mtimes are the value passed in, and the zstd encoder uses one thread, a fixed window and a CRC. A release tree contains only regular files and directories (D-047). Symlinks, hard links and device files are rejected at pack time with an error that names the path, so the failure happens on the build machine and not on a node. The updater rejects symlinks, hard links and devices again at extract time. A directory entry that cleans to `.` is the archive root. Any other `.` or `..` segment is rejected at extract time.

The artifact format is `tar.zst` because that is the only value in the manifest schema. The standard library has no zstd decoder. The encoder and decoder are `github.com/klauspost/compress` (BSD-3-Clause, pure Go). The pin is the latest stable release at the time this package was added.

## Commands

From the repository root, with `CGO_ENABLED=0` and `GOTOOLCHAIN=local`:

```text
fvpn-release keygen -key-id release-YYYY -out release-key.json \
  -trust-out release-trust.json \
  -not-before 2026-01-01T00:00:00Z -not-after 2027-01-01T00:00:00Z

fvpn-release build-manifest -in release-input.json -out signed.json
fvpn-release sign -signed signed.json -key release-key.json -out manifest.json
fvpn-release verify -envelope manifest.json -trust release-trust.json \
  -at 2026-10-28T15:00:00Z \
  -artifact amd64=amd64.tar.zst -artifact arm64=arm64.tar.zst
fvpn-release package -dir ./tree -out amd64.tar.zst -mtime 0
```

`keygen` prints a reminder that the private key stays on the build machine. `-trust-out` writes public keys only.

`release-input.json` is a `release.BuildInput`: release id, product version, sequence, created time, source commit, component versions, compatibility (control API major, policy and usage schemas, minimum updater version, state schema from/to, whether the previous release can read the new state), rollback reserve and health profile, evidence URLs, and the two artifact paths and `https` URLs. Evidence URLs and artifact URLs must be `https`.

## Where the key lives

| Material | Where | On a node? |
|---|---|---|
| Release private key (`seed_b64`) | Build machine, mode 0600, outside the node image | No |
| Release public trust store | Built into the node image at `/etc/vpn-node/release-trust.json`, and shipped again when a key is rotated | Yes, public only |
| Desired-state, lease and recovery keys | Their own stores. Not this key | Public only |

Key rotation is a new trust-store entry with its own window. Nodes keep the previous public key until `not_after`. A revoked key is never trusted again. The updater rejects the whole envelope when any signature by a known key is revoked, out of window, the wrong purpose, or structurally bad (§16.2).

## Publishing

1. Build both Ubuntu 24.04 trees on the build machine. Do not build them on a node.
2. `package` each tree. Record the sha256 the command prints.
3. `build-manifest` with those artifacts, then `sign`.
4. `verify` the envelope and both files before upload.
5. Publish the manifest envelope and both artifacts at the `https` URLs in the manifest. The object store should treat them as immutable. A changed body under the same URL fails the node's hash check and is not retried forever.
6. The center puts `release_id` and `manifest_sha256` into desired state. It does not send the release private key, and it does not send an unsigned manifest.

A tampered byte in the envelope or in an artifact fails `verify`. The golden vector is `contracts/vectors/release-manifest-signing.json`. `product_version` in that vector is display only.

## Rotating release trust from the build machine

`trust-update` produces an offline authorization for one explicit `add`,
`retire` or `revoke` operation (§13.4, §16.2, §17.2). `trust-verify` checks that
file against a supplied public trust store. Both use the embedded
`trust/release-update-envelope.schema.json`; `sign` and `verify` remain limited
to release manifests. The release private seed stays on the build machine.

Applying an update at the centre is a separate step. Follow the centre's apply
operation and the [PKI rotation runbook](../pki/rotation.md); these commands do
not contact the centre or update its vault. Carry the public update file and
its `update_sha256` to the centre. After each successful apply, bring back the
centre's freshly exported node trust store and its current release revision.
Use that export as `-trust` for the next step. Do not build the next trust store
from the proposed update or treat offline success as an apply receipt.

The following paths, dates, windows and revisions are **TEST ONLY** placeholders.
The parent output directory must already exist. In production choose windows
explicitly and obtain the actual current release revision from the centre;
`-revision` must be that revision plus one, with no wrap at the uint64 maximum.
The exported public store itself does not carry that revision.

First generate B on the build machine, then have the currently trusted A
approve B. This example starts at centre release revision 0:

```sh
# TEST ONLY: substitute protected build-machine paths and chosen lifetimes.
fvpn-release keygen -key-id release-B -out ./example-build/release-B.key.json \
  -trust-out ./example-build/release-B.trust.json \
  -not-before 2026-01-01T00:00:00Z -not-after 2028-01-01T00:00:00Z
fvpn-release trust-update -key ./example-build/release-A.key.json \
  -trust ./example-build/centre-trust-r0.json -revision 1 \
  -add ./example-build/release-B.trust.json \
  -at 2026-11-01T12:00:00Z -out ./example-build/add-B-r1.json
fvpn-release trust-verify -update ./example-build/add-B-r1.json \
  -trust ./example-build/centre-trust-r0.json -current-revision 0 \
  -at 2026-11-01T12:00:00Z
```

Take `add-B-r1.json` and its hash to the centre for apply. Bring back the
confirmed revision 1 export as `centre-trust-r1.json`. Coordinate distribution
of the new trust store before switching release signers. An applied update
does not mean nodes have received the new trust store. No node consumes these
update envelopes; nodes consume the separately delivered trust store.

The second handover step has B retire A with an explicit overlap end. The
chosen end must be after A's existing `not_before` and no later than its
existing `not_after`; even keeping the original end requires stating it:

```sh
# TEST ONLY: assumes A's existing window contains this retirement end.
fvpn-release trust-update -key ./example-build/release-B.key.json \
  -trust ./example-build/centre-trust-r1.json -revision 2 \
  -retire release-A -not-after 2026-12-01T00:00:00Z \
  -at 2026-11-01T12:00:00Z -out ./example-build/retire-A-r2.json
fvpn-release trust-verify -update ./example-build/retire-A-r2.json \
  -trust ./example-build/centre-trust-r1.json -current-revision 1 \
  -at 2026-11-01T12:00:00Z
```

Carry that file and hash for apply and retrieve `centre-trust-r2.json`.
Retirement retains verification inside the narrowed window; it is not
revocation. If A must be revoked, B can authorize a separate update:

```sh
# TEST ONLY: obtain the latest export and revision again before production use.
fvpn-release trust-update -key ./example-build/release-B.key.json \
  -trust ./example-build/centre-trust-r2.json -revision 3 -revoke release-A \
  -at 2026-11-02T12:00:00Z -out ./example-build/revoke-A-r3.json
fvpn-release trust-verify -update ./example-build/revoke-A-r3.json \
  -trust ./example-build/centre-trust-r2.json -current-revision 2 \
  -at 2026-11-02T12:00:00Z
```

`trust-update` prints `kind`, `revision`, `signer_key_id`, `operation`,
`target_key_id` (a JSON string, so opaque IDs stay on one line) and
`update_sha256`. The new file is the canonical envelope plus one newline,
mode 0644, containing public data only. The hash is lowercase SHA-256 of the
canonical envelope, excluding the newline. Existing files, directories and
symlinks, including dangling symlinks, are refused without replacement.

`trust-verify` reads two files and writes none. It prints `kind`, `revision`,
`signer_key_ids` (comma-separated trusted signers, in signature order), the same
`update_sha256`, one `key_record=<canonical JSON>` line per record, and
`usable_root_after=true|false`. Each record contains `id`, `new`, `state`,
`not_before`, `not_after`, `window_narrowed` and `newly_revoked`. An unknown
signature is omitted from the signer list; any invalid signature by a known
key rejects the whole envelope even alongside a good signature.

`-at` is the single instant used for signer validity, added-key expiry, the
usable-root check and a new revocation's `revoked_at`. It defaults to current
UTC truncated to whole seconds. Explicit RFC 3339 offsets are normalized to
UTC; explicit fractional seconds are preserved. Windows are half-open:
`not_before <= at < not_after`. A past `-at` proves only validity at that past
instant, not validity when the centre receives the file.

The producer applies the change in memory and refuses to remove the last
usable release root. Retired roots still count until their window ends;
future or revoked roots do not count. `-allow-no-usable-root` permits deliberate
emergency removal of the last usable root, for example emergency revocation.
**Losing every trusted root is a recovery situation these commands do not
solve. No command accepts an unsigned replacement.** The flag bypasses only
this guard; it never bypasses schema, signer, key or transition checks.
`trust-verify` reports a false `usable_root_after` without failing verification.

A passing offline check does not mean the centre has applied or will accept
an update. The centre decides against its own ledger at apply time. In
particular, the offline check cannot know:

- The stored release revision unless the operator supplies `-current-revision`;
  even that value may be stale by apply time.
- Whether an existing non-revoked key is retired; the centre rejects a
  retired-to-active transition.
- The original revocation time of an already revoked key; the centre requires
  the exact stored `revoked_at` when that key is named again.
- Changes to the centre's trust, time or ledger after the export, or trust-store
  delivery and acceptance on nodes.

Every new-command flag may occur once. Unknown flags, positional arguments,
missing required flags, malformed revisions/times and incompatible operations
exit 2. Invalid files, failed verification, unsafe transitions, the usable-root
guard and write failures exit 1 with `fvpn-release: <error>` on stderr and no
result on stdout. Success exits 0; `-h`/`--help` exits 0 and prints help to
stderr. These conventions leave the existing subcommands unchanged.

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
`-revision` must be strictly greater than that revision; plus one is the ordinary
choice. Revisions never wrap at the uint64 maximum.
The exported public store itself does not carry that revision.

Choose release signer IDs matching `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`: 1–128
ASCII characters, starting with a letter or digit, then letters, digits, `.`,
`_`, `:` or `-`. The producer refuses an unsignable `-key` ID or a new `-add`
ID before signing. Existing opaque IDs remain valid targets for retirement
and revocation. This signable-ID restriction is producer-only; `trust-verify`
continues to report opaque record IDs accepted by the centre. The legacy
`keygen` still accepts IDs such as `release B`, but the legacy `sign` rejects
that ID when validating its manifest envelope. Choose a signable ID up front.

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

The following **TEST ONLY** blocks are stdout from a real run, except the
explicitly labeled keygen stderr block. Each `update_sha256` is an example from
runtime-generated keys and **varies between runs**; producer and verifier hashes
match within the same run. These are offline command outputs, not centre apply
receipts. The already pinned A in this run has the same 2026–2028 window as B.

`keygen` stdout:

<!-- trust-output:keygen -->

```text
key_id=release-B
purpose=release
```

`keygen` stderr:

<!-- trust-output:keygen-stderr -->

```text
fvpn-release: private key stays on the build machine; do not copy it to nodes
```

`trust-update -add` stdout:

<!-- trust-output:add -->

```text
kind=fvpn-release-trust-v1
revision=1
signer_key_id=release-A
operation=add
target_key_id="release-B"
update_sha256=95fecbaebef1bc3ac658e648ac042b44b14fe1745b05c4a80bd2eba5733600d1
```

`trust-verify` for the add file, before apply:

<!-- trust-output:verify-add -->

```text
kind=fvpn-release-trust-v1
revision=1
signer_key_ids=release-A
update_sha256=95fecbaebef1bc3ac658e648ac042b44b14fe1745b05c4a80bd2eba5733600d1
key_record={"id":"release-B","new":true,"newly_revoked":false,"not_after":"2028-01-01T00:00:00Z","not_before":"2026-01-01T00:00:00Z","state":"active","window_narrowed":false}
usable_root_after=true
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

`trust-update -retire` stdout:

<!-- trust-output:retire -->

```text
kind=fvpn-release-trust-v1
revision=2
signer_key_id=release-B
operation=retire
target_key_id="release-A"
update_sha256=c7de0ffe1fcf8fcc5e260081ede29e5c137ebc9b15ce462090a05c13e153dacd
```

`trust-verify` for the retirement file, before apply:

<!-- trust-output:verify-retire -->

```text
kind=fvpn-release-trust-v1
revision=2
signer_key_ids=release-B
update_sha256=c7de0ffe1fcf8fcc5e260081ede29e5c137ebc9b15ce462090a05c13e153dacd
key_record={"id":"release-A","new":false,"newly_revoked":false,"not_after":"2026-12-01T00:00:00Z","not_before":"2026-01-01T00:00:00Z","state":"retired","window_narrowed":true}
usable_root_after=true
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

`trust-update -revoke` stdout:

<!-- trust-output:revoke -->

```text
kind=fvpn-release-trust-v1
revision=3
signer_key_id=release-B
operation=revoke
target_key_id="release-A"
update_sha256=6b6e5d33b31d16b71f6eac3b6de2fe298949ceedfb682192f5c32c8be703b5b3
```

`trust-verify` for the revocation file, before apply:

<!-- trust-output:verify-revoke -->

```text
kind=fvpn-release-trust-v1
revision=3
signer_key_ids=release-B
update_sha256=6b6e5d33b31d16b71f6eac3b6de2fe298949ceedfb682192f5c32c8be703b5b3
key_record={"id":"release-A","new":false,"newly_revoked":true,"not_after":"2026-12-01T00:00:00Z","not_before":"2026-01-01T00:00:00Z","state":"revoked","window_narrowed":false}
usable_root_after=true
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
UTC. Both `-at` and `-not-after` accept at most nine fractional digits and
preserve that precision; more than nine is a usage error before any file read. Windows are half-open:
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
particular, the offline check cannot know or may decide differently:

- The stored release revision unless the operator supplies `-current-revision`;
  even that value may be stale by apply time.
- Whether an existing non-revoked key is retired; the centre rejects a
  retired-to-active transition.
- The original revocation time of an already revoked key; the centre requires
  the exact stored `revoked_at` when that key is named again.
- Changes to the centre's trust, time or ledger after the export, or trust-store
  delivery and acceptance on nodes.

The producer emits updates with one signature. Keep that single signature when
transporting the file. Offline verification checks the complete exported node
bundle: a second signature naming a known key of another purpose, such as a
lease key, rejects the whole envelope. The centre checks release roots only
and ignores that same non-release signature as unknown, so it can accept a file
that this stricter offline check rejects. Do not remove other-purpose keys from
the exported bundle to make the verdicts agree.

Every new-command flag may occur once. Unknown flags, positional arguments,
missing required flags, malformed revisions/times and incompatible operations
exit 2. Invalid files, failed verification, unsafe transitions, the usable-root
guard and write failures exit 1 with `fvpn-release: <error>` on stderr. Failures
before result output produce no stdout results. An output failure may leave
partial result lines, which must not be treated as a complete result. Success exits 0; `-h`/`--help` exits 0 and prints help to
stderr. These conventions leave the existing subcommands unchanged.

A result-output failure after `trust-update` publishes the file does not undo
publication. The complete, validated file remains unchanged; run `trust-verify`
with the same trust store and instant to verify it and recompute `update_sha256`.
Do not retry with the same output path expecting it to be replaced. These are
the fixed result-output diagnostics (each prefixed with `fvpn-release: `):

```text
trust-update: result output failed; published update file remains
trust-verify: result output failed
```

Filesystem diagnostics from these two commands contain neither operator paths
nor raw system error text. The complete fixed set is below (each also prefixed
with `fvpn-release: `). An unreadable, missing or non-file input is a read
failure. An inaccessible or absent staging parent is unavailable; failures
while writing/staging or linking the final file use `cannot be published`.

```text
-key cannot be read
-add cannot be read
-trust cannot be read
-update cannot be read
-out already exists
-out parent directory unavailable
-out cannot be published
```

Malformed private key contents retain the fixed message
`trust-update: cannot load release signing key`. Flag parsing uses
`invalid flags: unknown, repeated or malformed flag`, without quoting argument
values. The new precision and signer-ID diagnostics are:

```text
-at allows at most nine fractional digits
-not-after allows at most nine fractional digits
-key key_id cannot sign: must match ^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$
-add key_id cannot sign: must match ^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$
```

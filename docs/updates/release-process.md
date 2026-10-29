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

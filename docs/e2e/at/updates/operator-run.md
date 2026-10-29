# AT23 operator run: physical arm64 and the three join sources

Status: `needs operator run`. This host ran the amd64 lab containers and
cross-built `fvpn-updater` for `linux/arm64`. It did not boot a physical
arm64 node, a preinstalled box, a cloud Ubuntu host, or a third-party
coordinated Ubuntu host (§19 AT05, §19 AT23, NODE01). Do not mark those
steps passed from the lab record.

Use the same signed release the lab registered (manifest URL under
`https://releases.example.invalid/r/<release_id>/`, both `amd64.tar.zst` and
`arm64.tar.zst`, central `verification_state=verified`). Do not point a node
at a third-party Internet host.

## 1. Physical arm64 node

1. Install the node image on Ubuntu 24.04 arm64 hardware. Confirm
   `uname -m` prints `aarch64`.
2. Enroll with one unused installer token. The node must come up
   `unpublished`, with no per-node SSH and no per-batch approval.
3. Create one `all_nodes` batch for a release whose manifest lists both
   architectures. Leave the node online.
4. After the updater exits, record:
   - batch id, target row (`needs_attention`, `unconfirmed`, state)
   - journal artifact sha256 equals the manifest arm64 sha256 and does not
     equal the amd64 sha256
   - `/opt/vpn-node/current` names that release
   - health JSON and `error_code`
   - `nft list chain inet fvpn forward` (member accepts stay 0 until the
     node is published and serviceable)

Expected: the arm64 node downloads only the arm64 artifact. Success is a
real version plus health confirmation (§13.9, §13.10). A health failure
with `ROLLBACK_FAILED` and empty `last_good` is the same non-serving
precondition as the lab, not an architecture pass.

## 2. Three join sources (AT05 precondition of AT23)

Repeat the same enrollment and the same batch on each source. Each source
uses its own token. Afterwards the admin list shows three nodes, each
`unpublished` until an operator publishes it.

| Source | Action | Evidence to keep |
|---|---|---|
| Preinstalled box | Boot the image; it enrolls with the token baked into the installer flow | enrollment trace, package hash, node id, incarnation |
| Cloud Ubuntu 24.04 | Run the same installer on a cloud VM | enrollment trace, package hash, node id, incarnation |
| Coordinated third-party Ubuntu 24.04 | The owner installs; do not take a shell on their host | their enrollment trace and the central node row only |

Expected: one protocol, one agent, one policy, and the same publication
rules (§19 AT05). The batch snapshot includes every non-revoked node that
existed when the batch was created. A node enrolled after that batch takes
the fleet default and is absent from the old snapshot (§19 AT24).

Record UTC time, image digest, release id, policy revision, and incarnation
for each source. Redact tokens, setup keys, and hostnames.

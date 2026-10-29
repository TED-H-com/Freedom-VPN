# NetBird development version lock

Checked 2026-10-29. This is a **TEST ONLY** CE deployment, not a production
identity or forwarding approval. The latest stable upstream release at the
check was [v0.82.0](https://github.com/netbirdio/netbird/releases/tag/v0.82.0),
published 2026-10-29. Linux client and combined server are pinned together.
The separately released dashboard is v2.96.0 (2026-10-27).

| Component | Exact upstream image | OCI multi-platform index digest | License |
|---|---|---|---|
| Management + Signal + Relay + STUN + embedded IdP | `netbirdio/netbird-server:0.82.0` | `sha256:7b5b53b2dc8bc25639f791cf72d47cf86911894ff3f8a97cb79bab9be958109c` | NetBird server directories: AGPL-3.0 |
| Linux client | `netbirdio/netbird:0.82.0` | `sha256:eca5b274c635bfbb04ec6724a0712d1146190dd450eea199eeb83916fdd94965` | NetBird client: BSD-3-Clause |
| Dashboard | `netbirdio/dashboard:v2.96.0` | `sha256:340b13c00b84349572a2acc5605e907cb09250a9e6461c757097eabdb8e02f2c` | AGPL-3.0 |
| Loopback reverse proxy / STUN proxy | `nginx:1.30.0-alpine` | `sha256:0272e4604ed93c1792f03695a033a6e8546840f86e0de20a884bb17d2c924883` | nginx: BSD-2-Clause; image packages have their own notices |

The Dockerfiles contain these exact `tag@digest` references. They produce only
locally named `${FVPN_NB_PREFIX}-*` images. Each index has amd64 and arm64
manifests. A tag changing upstream cannot change the selected bytes. Digest
verification used `docker buildx imagetools inspect <image:tag>`; repeat that
command when reviewing a bump.

NetBird's [root license at v0.82.0](https://github.com/netbirdio/netbird/blob/v0.82.0/LICENSE)
excludes management, signal, relay and combined from the BSD grant. Their
[server license](https://github.com/netbirdio/netbird/blob/v0.82.0/combined/LICENSE)
and the [dashboard license at v2.96.0](https://github.com/netbirdio/dashboard/blob/v2.96.0/LICENSE)
are AGPL-3.0. Embedded Dex is compiled into the server, not a separate image;
the pinned NetBird `go.mod` uses `github.com/netbirdio/dex`
`v0.244.1-0.20261021152327-0319b501f1a6` (Apache-2.0).
See [Dex's source license](https://github.com/netbirdio/dex/blob/0319b501f1a6/LICENSE)
and [nginx's license](https://nginx.org/LICENSE).
These identify the principal components, not an exhaustive container SBOM.
Preserve upstream copyright notices, license texts and corresponding-source
links with redistributed release artifacts (§13.3); do not describe all
NetBird components as BSD. This task does not redistribute modified binaries.

## Layout and API decision

The [current quickstart](https://docs.netbird.io/selfhosted/selfhosted-quickstart)
recommends the combined server and embedded Dex. Separate management, signal,
relay and external-IdP installations remain documented for advanced deployments.
The embedded provider supports local users and external OIDC connectors.
We use generated local accounts for this lab; choosing a shared production
issuer and verifying Portal/NetBird subject mapping remain open (§2, §9.1).
CE does not provide Peer Approval, SCIM or active-active control services.

The adapter uses `/api/peers`, `/api/users` (read only), `/api/groups`,
`/api/policies`, `/api/routes`, `/api/setup-keys` and `/api/dns/nameservers`.
Development bootstrap additionally uses `/api/instance`, `/api/setup`,
`/api/accounts`, `/api/users` and `/api/users/{id}/tokens`.
All are HTTP APIs; nothing reads or writes NetBird's database directly.

[Routes is deprecated](https://docs.netbird.io/api/resources/routes) for general
network routing. [Networks replaces that use](https://docs.netbird.io/manage/networks/how-routing-peers-work),
but the pinned release still uses Routes for **exit nodes**; there is no documented
exit-node replacement. The adapter keeps this dependency isolated (§7.2).
The verified request uses one `peer`, omits `peer_groups`, sets `0.0.0.0/0`,
`skip_auto_apply=true`, and **`masquerade=false`**. Freedom VPN's G2 host layer
must supply SNAT after classification/metering. This differs from NetBird's
normal masquerading default and is not proof of Internet-only forwarding.

## Native app candidates (all 待實測)

| Platform | Exact candidate | Status |
|---|---|---|
| Windows | NetBird daemon/CLI v0.82.0; matching desktop UI bundled in that release | No Windows execution |
| macOS | NetBird daemon/CLI v0.82.0; matching desktop UI bundled in that release | No macOS execution |
| Android | [Android app v0.6.2](https://github.com/netbirdio/android-client/releases/tag/v0.6.2) | Separate mobile version; not v0.82.0 |
| iOS | [iOS v0.4.1, build 2](https://github.com/netbirdio/ios-client/releases/tag/v0.4.1) | Upstream labels it a TestFlight release; App Store availability not asserted |

Record the actual app build, bundled core and desktop UI version on devices.
Stop the acceptance run if those differ; update the matrix and rerun it. Linux
container success does not establish mobile core compatibility.

## Bumping

1. Read the candidate release notes, API changes, server/client/dashboard
   licenses and current self-hosting instructions. Keep the source links and
   hashes in `SOURCES.md` current. Check mobile versions independently.
2. Resolve all OCI index digests; confirm both architectures exist. Change
   Dockerfiles, Compose local tags, this table, fixture provenance and checklists
   together. Never replace pins with `latest`.
3. Reset the disposable stack; build, bootstrap twice, and run the complete
   integration suite. Re-record using `FVPN_NETBIRD_RECORD=1`, scrub with
   `scrub-recordings.py`, and review every resulting fixture field.
4. Run unit tests, vet, formatting and both architecture builds. Record actual
   failures, CLI output changes and the machine-evidence boundary in G1.
5. Repeat native platform acceptance and G2 network-path tests before expanding
   the production support matrix. Keep the old release available for rollback.

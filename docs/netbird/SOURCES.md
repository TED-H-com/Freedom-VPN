# Official-source review snapshot
Review date: 2026-10-29. These are concise engineering notes and SHA-256
fingerprints of the reviewed text/source, not redistributed copies of the
entire websites. URLs can change; the version lock and recorded API fixtures
remain the reproducible contract. The specification remains authoritative.
| Spec source | Official URL / pinned source | Review note | Snapshot SHA-256 |
|---|---|---|---|
| S01 | [source](https://docs.netbird.io/manage/peers/register-machines-using-setup-keys) | Machine setup keys support one-off use and auto-groups. Revoking a key does not disconnect its existing peers. | `977a271273731907d876de0f27048eedee1000ccce25563fd5f7ab27cde8b77f` |
| S02 | [source](https://docs.netbird.io/manage/peers/bootstrap-via-config-file) | Client state must be writable. Golden images leave the WireGuard private key empty; user devices should use SSO. | `91bc90a2e7299b95105f12ce85814274f9ef82b1efcc3a2bfcaaa9fb5dfbdb45` |
| S03 | [source](https://docs.netbird.io/manage/peers/approve-peers) | Peer Approval is unavailable in CE. Paid-device checks must be implemented by our controller. | `930855aa959b51fb7cb3dc34da86e8a5fce2292fa9788ff5a5c793c8c6f0300c` |
| S04 | [source](https://docs.netbird.io/api/resources/routes) | Routes is deprecated but still exposes the exit-node flags used by this adapter. Peer and peer_groups are mutually exclusive. | `fd141a1a1470fb1aef662adbefdd97a8fcb074628e989e5d6f1489e10bb7ebc0` |
| S05 | [source](https://docs.netbird.io/manage/networks/how-routing-peers-work) | Networks replaces general legacy routing, except exit nodes. Disabling masquerade preserves source addresses and requires an external return/SNAT design. | `78e98cc68edea73b04e078770781651ecb28b11ffd1715623c502f0b3b6f36ad` |
| S06 | [source](https://docs.netbird.io/use-cases/remote-access/exit-nodes) | Exit nodes use default routes and distribution groups. Auto Apply is an initial preference; explicit user selections persist. No complete kill-switch claim. | `dcf577126bdd9c371023f3934b156e9d0540c612ec3e9a5035454defbadce7ee` |
| S07 | [source](https://docs.netbird.io/manage/access-control/manage-network-access) | Policies allow access. Remove the initial broad policy before adding narrow rules; groups do not intrinsically isolate peers. | `dda6c6426957870306d444768cd875666d161c6c0283d35c17d685431eb9282a` |
| S08 | [source](https://docs.netbird.io/about-netbird/ports-and-firewalls) | Management, signal, relay and STUN have different connectivity needs. Our gateway uses HTTP/gRPC/WebSocket and UDP STUN in the private lab. | `fe038d8d6ff43c541c02c2ebc1d4117aa3a74254727402d327c156039a9658b2` |
| S09 | [source](https://docs.netbird.io/about-netbird/understanding-nat-and-connectivity) | Direct versus relay connectivity is per peer pair. Lab connectivity is not evidence of every public NAT situation. | `10c828b935456e87dc578e460b0c21f3187997db4c29d64eb0560a4755abb638` |
| S10 | [source](https://docs.netbird.io/client/environment-variables) | Native, userspace and netstack paths differ; environment options affect routing and marks. Do not infer G2 packet-path validation from successful enrollment. | `d05be31d9bee6d0e6e1edd04663df42aa705ad902b4f9e0665844d06b1677014` |
| S11 | [source](https://docs.netbird.io/manage/activity/traffic-events-logging) | Traffic-event logging has edition restrictions and excludes exit-node default-route flows. It is not the product usage ledger. | `de5608f12d1912fc2c57e825e53190bacb56fac06c623ebf78b880a7433d37cf` |
| S12 | [source](https://docs.netbird.io/manage/peers/auto-update) | Native automatic updates do not cover Linux node upgrades; package/version management remains our updater responsibility. | `f9da0c40ac6337ff730b2e34ecff48efa498244c75e407f97330ef01ac6b0fed` |
| S13 | [source](https://docs.netbird.io/manage/dns) | Nameserver groups distribute DNS settings. Resolver availability, fallback and split-DNS support require platform-specific tests. | `fade7697f982c6b5a58f5d48c5d55c387b680b19eb90b2eb859f80e212ba5398` |
| S14 | [source](https://docs.netbird.io/get-started/install/android) | Official Android setup documents self-hosted URL, VPN permission and browser login. Its older menu description differs from v0.6.2 source. | `869963b2570658779905fc2026e5c6751c123e25cd9ae975cf2b7b1edd13ac89` |
| S20 | [source](https://github.com/netbirdio/netbird/blob/v0.82.0/LICENSE) | Root BSD-3-Clause excludes management, signal, relay and combined, which use AGPL-3.0. | `8343fe949e32a30e2261c4f0967365271d08c4644e879dec5b998d1af666fdb3` |
| S21 | [source](https://github.com/netbirdio/dashboard/blob/v2.96.0/LICENSE) | Dashboard is AGPL-3.0, separately versioned from the client and server. | `4df3c306dddaaf4baffdff5ca820cc679ac8cd6dc263c6a74517783e42fa7a3b` |
| S27 | [source](https://docs.netbird.io/selfhosted/selfhosted-quickstart) | New self-hosted installs use the combined server with embedded IdP and separate dashboard. Generated local secrets are mandatory. | `b1897734e1b2712077941da8eb4ee625bdd3db37a0c2e3d6e0896287330c1fc7` |
| S29 | [source](https://docs.netbird.io/manage/dns/troubleshooting) | Android Private DNS can bypass VPN DNS configuration. Changing that setting and checking the actual resolver path is a manual acceptance step. | `5752b6691ee7ba1f15d884e83617ba541c7b344d8bb9b81a73f72714565f6b87` |
| S32 | [source](https://docs.netbird.io/selfhosted/identity-providers) | Embedded local users need no external IdP. Optional external OIDC connectors do not establish a shared Portal subject mapping by themselves. | `e5de004ef6ca7fbd5ad3f80bd8be10516a8673a38b78c8bacc19ba5e131ccc30` |
| S33 | [source](https://docs.netbird.io/client/desktop-app) | Desktop apps expose a self-hosted welcome flow, profiles, settings and exit-node selection. Each platform still needs real-device evidence. | `dca96d095faa548f045129158ac3693088f444ee12c93a8bb01525021e04c603` |
| S34 | [source](https://docs.netbird.io/selfhosted/enterprise) | CE core networking is available; Peer Approval, SCIM and active-active control-plane HA require commercial features. A self-hosted deployment is one account. | `699f591bd79ea0c693bdc9f1a1db5ab8dd30c938aab7a06444eb039c9bf9a9e4` |
| S35 | [source](https://docs.netbird.io/api/resources/setup-keys) | The setup-key API supports one-off type, usage_limit=1 and minimum expires_in=86400. Plaintext is not recoverable from list/read responses. | `59da1dcf80e769edcf981a47b9686fe2dd65a91b0a097b411923fdb6f3175ebb` |

The S27 URL in the spec (`/selfhosted/quickstart`) was unavailable through the
browser fetch; the current official `/selfhosted/selfhosted-quickstart` was read.
HTML page fingerprints use locally extracted visible text (including site
navigation), so site navigation changes can change the fingerprint without
changing the technical advice. License fingerprints are exact source bytes.

Additional pinned primary sources used to check actual behavior:

- [Combined server config](https://github.com/netbirdio/netbird/blob/v0.82.0/combined/config.yaml.example).
- [OpenAPI and generated API models](https://github.com/netbirdio/netbird/tree/v0.82.0/shared/management/http/api).
- [Automated owner/PAT setup](https://docs.netbird.io/selfhosted/automated-setup).
- [Embedded local user management](https://docs.netbird.io/selfhosted/identity-providers/local).
- [CE network_admin permissions](https://github.com/netbirdio/netbird/blob/v0.82.0/management/server/permissions/roles/network_admin.go).
- [Android v0.6.2 UI](https://github.com/netbirdio/android-client/tree/v0.6.2/app/src/main).
- [iOS v0.4.1 UI](https://github.com/netbirdio/ios-client/tree/v0.4.1/NetBird/Source/App/Views).

Runtime recordings are described in `internal/netbird/testdata/README.md`.
The lab needs no external API traffic after image/dependency acquisition;
version research and this review were separate public-documentation reads.

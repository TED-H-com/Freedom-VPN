# NetBird adapter contract

`internal/netbird` implements §7.2 for NetBird CE 0.82.0. It uses standard
`net/http` and adds no Go dependencies. The central worker owns the adapter and
its deployment-scoped token. Nodes and browsers never receive that token.

## Operations and API calls

All paths below are relative to a configured URL ending in `/api` (NetBird's
prefix, distinct from Freedom VPN's `/api/v1`). Every collection create first
reads the binding's external ID, if present, and searches the collection by its
stable managed name. Duplicate matches or a renamed bound resource fail with
`Conflict`. Successful writes are followed by a separate GET.

| Operation | Semantics and exact calls |
|---|---|
| `EnsureNodePeer(NodePeer)` | `GET /peers/{id}`; validate a non-ephemeral machine peer without a user ID, record node/incarnation mapping. The ID may come from a bound observation or D-055 discovery; the native agent may report null. REST cannot create a WireGuard peer; setup-key enrollment creates it. No name/email/IP matching. |
| `DiscoverNodePeer(enrollmentID, NodePeer)` | Under the deployment lock, read the one-off key and its dedicated correlation group; require exactly one peer and `used_times=1`, then GET that peer and require a non-ephemeral machine identity (`user_id=""`). Reject policy/route references to the group. Record the node/incarnation binding, detach the temporary auto-group from the key with PUT/GET, empty the group with PUT/GET, delete it and confirm GET 404. A repeated call resumes cleanup and returns the same binding. |
| `EnsureMemberDevice(MemberDevice)` | `GET /peers/{id}`, `GET /users`; require the expected external user, non-service/non-blocked user, no user auto-groups and non-ephemeral peer. Record verified member/user and device/peer bindings. Does not grant access. |
| `EnsurePaidGroup(groupID)` | `GET /groups`, `GET /groups/{id}`, then `POST /groups` if absent or `PUT /groups/{id}` for drift, then GET. First creation is empty. Existing peer memberships are preserved. |
| `PublishExit(Exit)` | Verify node and paid groups. Reconcile a one-node destination group and minimal ICMP policy with `/groups` and `/policies`; then `GET /routes`, `POST /routes` or `PUT /routes/{id}`, and `GET /routes/{id}`. Fixed single-peer IPv4 default route, enabled, metric 9999, no masquerade, no automatic application. |
| `WithdrawExit(exitID)` | Locate and `DELETE /routes/{id}`, confirm GET 404; then delete/read back the exit-specific policy. Keep the empty/reusable destination group. Retry is safe, including a previously lost DELETE response. |
| `ApplyDeviceAuthorization(device, paidGroupIDs)` | Reverify member identity for grants; revocation of a known immutable device binding remains possible even if its upstream user was blocked/deleted. Read all groups; replace only this peer's memberships in the reserved `fvpn-paid-*` namespace, using `PUT /groups/{id}` and GET. Remove obsolete memberships before adding new ones. Preserve all other peers. Empty desired set revokes paid exit and DNS distribution. |
| `DeletePeer(bindingKey)` | Read binding, `GET /routes` to reject deleting a published routing peer (direct or via peer groups); `DELETE /peers/{id}`, then GET 404. Remove mapping only after confirmed absence. Withdraw before delete. |
| `ReadObservedState()` | Fresh `GET /peers`, `/groups`, `/routes`, `/policies`, `/dns/nameservers`; return those observations plus UTC observation time. Never substitute desired state or the binding cache. This is a sequential snapshot, not an upstream transaction. |
| `CreateNodeSetupKey(enrollmentID)` | Reconcile the shared `fvpn-unpublished-nodes` and the initially empty `stableName("enroll-peer", enrollmentID)` correlation group; `GET /setup-keys` and known ID; `POST /setup-keys` only if absent, then GET. `type=one-off`, `usage_limit=1`, `expires_in=86400`, `ephemeral=false`, no extra DNS labels, exactly those two auto-groups. The correlation group grants no access and must never be referenced by a policy or route. |
| `RevokeNodeSetupKey(enrollmentID)` | Locate key; `PUT /setup-keys/{id}` with `revoked=true` and its existing auto-groups; GET verifies revoked; detach and remove the correlation group using the same confirmed cleanup as discovery, including when the key is already absent. Never un-revoke. Does not disconnect enrolled peers. |
| `RemoveDefaultPolicy()` | `GET /groups`, `/policies`; remove the initial `Default` policy referencing `All` and confirm GET 404. Reject any unexpected policy referencing `All`, even if disabled. Call before enrolling members. |
| `BootstrapAccess(AccessConfig)` | Verify resolver peer addresses using `GET /peers/{id}`. Reconcile resolver/admin/ops groups, unidirectional TCP/UDP 53 rules from paid groups to resolver peers, and an admin-to-ops TCP rule with explicit ports. `POST/PUT /dns/nameservers` plus GET publishes a primary resolver group only to paid groups. No domains, search domains or public fallback. |

DNS addresses must be exact overlay IPv4 addresses, and match their enrolled
resolver peers (§12.2). Two resolvers are the default in the lab/configuration
example; deployments can explicitly supply another nonempty list. Resolver
selection/failover and filtering belong to the DNS task. Admin peer IDs and ops
ports are explicit inputs; there is no broad default management rule. The
exit policy uses bidirectional ICMP because the pinned NetBird policy API
requires that for ICMP. It grants no TCP/UDP services on the exit host. This is
not a substitute for the G2 forwarding boundary (§8.2).

## Identity and persistence boundaries

The caller supplies `MemberDevice` only after verifying a `netbird_user_bindings`
row: deployment + external user ID → member ID, identity binding reference,
mapping proof/revision and verified timestamp (§2, §15.2). The adapter checks
actual peer ownership. It deliberately has no email argument and refuses to
map setup-key peers as members. Proof verification, OIDC issuer/audience and
client/sector subject matching are the identity service's responsibility.

The central DB must atomically enforce device quota and commit entitlement,
resource version, operation and outbox before calling this adapter (§9.4,
§14.5). The adapter's lock preserves shared group updates; it is **not** a
member device-limit transaction. Do not claim AT02's final-slot race is solved
by the two-device group-concurrency test here.

`BindingStore` models the relevant `external_bindings` fields (§15.3):

| Go field / binding kind | Central mapping |
|---|---|
| `DeploymentID` | provider=`netbird`, deployment ID; every key is deployment scoped |
| `Kind`, `InternalID` | internal resource type and immutable domain ID |
| `ExternalID` | opaque upstream ID; never parsed as an integer |
| `DesiredHash` | SHA-256 of the canonical normalized desired request; only confirmed state gets a nonempty value |
| `LastObservedAt` | UTC time of successful independent read-back; zero means write known, unconfirmed |
| `routes` / exit ID | `exits` → route |
| `groups` / `paid:<group ID>`, `exit:<exit ID>`, `unpublished-nodes`, `enroll-peer:<enrollment attempt ID>`, resolver/admin/ops keys | controlled group resources |
| `policies` / `exit:<exit ID>`, `dns:tcp`, `dns:udp`, `ops` | controlled policy resources |
| `dns/nameservers` / `resolvers` | resolver distribution resource |
| `peers` / `node:<node ID>:<incarnation ID>` | `node_incarnations.NetBird peer ref` |
| `peers` / `device:<device ID>` | `member_devices` external peer ref; bidirectional uniqueness prevents rebinding |
| `users` / member ID | adapter check of the verified `netbird_user_bindings` mapping |
| `setup-keys` / enrollment attempt ID | provisioning job's external key ID; no secret |
| `groups` / `enroll-peer:<enrollment attempt ID>` cleanup | temporary management-plane correlation; removed only after binding or revocation and confirmed group deletion |

The memory implementation is for tests/dev, with both internal and external
uniqueness. `nodes.NetBirdBindings` persists production mappings in `external_bindings`. All workers sharing a
deployment must share a lock implementation. `Lock` covers the read/modify/write
sequence for **all** objects in a deployment, so adding two different devices
to one paid group cannot overwrite one another. Implement a durable worker
lease/advisory lock without holding a request transaction across external I/O.

Names use `fvpn-<kind>-` plus 24 hex characters of SHA-256 of the internal ID.
They remain stable across retries and display-name changes. NetBird network IDs
are stable machine labels; the route description carries the friendly name.
Native app display of that friendly name remains a manual acceptance item.
A stable-name match is a recovery mechanism, never member identity proof.

## Retry, hashes and completion

The public API's `Idempotency-Key`, ETag, resource version and generation checks
belong to Control API/outbox. NetBird does not provide those guarantees for
these endpoints. The adapter uses immutable domain keys, serialized writers,
stable-name discovery, canonical desired/observed comparison and read-back.
It never claims exactly-once external delivery.

Only GET calls have bounded exponential retries with jitter (0–5 configured
retries, default zero). Retryable cases are transport failure, 429 and 5xx;
other HTTP failures are immediate. `Retry-After` accepts seconds or HTTP date;
when it exceeds the request-timeout budget, return it to the worker instead of
retrying early. The default per-request timeout is 10 seconds. Cancellation
also interrupts retry waits and lock acquisition. A worker should set an
operation deadline as well. Redirects are refused to avoid forwarding tokens.

POST/PUT/DELETE are **never** automatically resent. A lost response returns
`Unavailable`; the worker repeats the domain operation, which reads upstream
before deciding to write. A lost create response is adopted by stable name,
not followed by another create. Unknown duplicates, conflicting identity
mappings and drifted names require operator reconciliation. Absence of an old
bound ID releases that binding only after a confirmed 404 and empty name lookup.
Unexpected HTTP response shapes and read-back mismatches are `Unconfirmed`.

Comparisons project request fields, normalize peer/group objects into sorted
ID sets and omit response counters/timestamps/IDs. Canonical hashing uses the shared `pkg/canonjson`; it is not a signing API.
Bindings are evidence hints, not a substitute for GET. Paid memberships are
re-read directly; successful membership changes refresh the known group binding
hash. A cached hash must never be treated as authorization. A paid target must
have a known group binding; reconcile it with `EnsurePaidGroup` after restoring
the binding store. Revocation also scans reserved paid groups whose bindings
are absent, so lost bindings do not silently preserve access.

Never mark a central operation complete until both this external read-back
and the required node ACK/enforcement evidence exist (§14.5). A route appearing
in the management API alone does not prove a member's data path. The integration
test additionally observes client route distribution and selection. Node
forwarding, leases and existing-flow revocation remain G2/Agent work.

## Secrets and least privilege

`Config.Token` accepts a `SecretSource`; `TokenFile` checks regular-file mode
(no group/other permissions), caps size and rereads on each call for rotation.
Production URLs require HTTPS. `AllowInsecureHTTP` is an explicit dev-only
opt-in. The package has no logger, and error strings contain only a category
and HTTP status, never URLs, bodies, headers, token-source errors or certificates.
The HTTP client/transport supplied by a caller must also avoid payload logging.

The complete operation set requires peer read/delete, setup-key read/create/
update, group/policy/route/nameserver CRUD and user read. The pinned CE
`network_admin` role cannot write setup keys or delete peers. Consequently a
service user with built-in `admin` is the narrowest available built-in role
covering this adapter. It has no auto-groups and a 30-day dev token. Owner is
used only for bootstrap. Protect this central credential; scoped/custom roles
would be a future improvement, not an assumed CE feature. See the
[pinned role definitions](https://github.com/netbirdio/netbird/blob/v0.82.0/management/server/permissions/roles/network_admin.go).

`SetupSecret.Reveal()` exposes a setup key once to the enrollment caller; normal
formatting and JSON serialization redact it. It never enters `BindingStore`.
Retry returns the same external key ID with `SecretAvailable=false`; masked
GET values are never returned as a key. On lost plaintext, revoke the old key
and authorize a new enrollment attempt ID. Its 24-hour NetBird lifetime is
separate from the product's short-lived enrollment code (§6.4). A caller serving
this secret over HTTP must set `Cache-Control: no-store` (§16.2).

## Open questions and restoration

Production shared OIDC issuer, cross-client subject proof and admin MFA remain
to be selected and exercised. The local embedded provider is dev-only and is
not asserted to be a reusable Portal issuer. The node worker uses the DB-backed binding store, outbox claims and incarnation fences. These central proofs do not replace live data-path evidence. After restore, run observations,
reconcile mappings and revoke uncertain access; never republish solely because
an older backup says it was enabled (§17.4). Operators must reserve the managed
namespace: upstream has no CAS, so concurrent dashboard edits to these objects
cannot be made safe by a process-local lock.

Routes-to-Networks migration is pending an exit-node replacement upstream.
Automatic app-name display, native app behavior, direct/relay/MTU behavior,
resolver failover and G2 isolation are 待實測. See the G1 evidence and checklist.

## Enrollment correlation and cleanup (D-055)

Discovery never correlates a node by its hostname, address or observation timing.
Zero peers/unused key returns `NotFound` for bounded worker retry. Multiple peers,
more than one key use, user ownership, ephemeral peers, conflicting bindings or
policy/route references return `Conflict` with a fixed safe reason. Central
records terminal operation/audit evidence without inventing a peer binding.
The per-attempt group is `fvpn-enroll-peer-` plus the stable hash; the shared
`fvpn-unpublished-nodes` group and its peer membership survive cleanup. Binding
precedes cleanup so a crash cannot lose the proof; revoked keys always reject
discovery. A consumed one-off key may be `overused` or expired with exactly one
use. Neither discovery nor cleanup publishes a node or admits a member.
See [the adapter lifecycle and tests](../../internal/netbird/DISCOVERY.md) and
[worker fencing](../nodes/WIRING.md). Local HTTP fakes are TEST ONLY; native-client
and network behavior still requires lab evidence.

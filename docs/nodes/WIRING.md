# Node control and worker wiring

References: Project Spec §5–7, §13.1–2/10, §14.4–5, §15.3, §16.1–3,
§17.1. The current OpenAPI contract takes precedence over older service docs.

## Composition and operation list

`nodes(),` is the single registration line in each process's `Domains()` list.
`apps/control/wire_nodes.go` builds `internal/nodes/httpwire` once using the
process-owned pool, configuration and logger. Its `Built` supplies the strict
handlers, PKI authenticator and enrollment credential middleware to the
[D-026 composition seam](../control/COMPOSITION.md). The framework owns both
listeners, routing, response tracking and request IDs. All command handlers reuse
`httpapi.Transaction`, set durable resource references, and implement replay.
Standalone service callers retain their existing transaction/idempotency APIs.

The following transitional 501 handlers are removed:

| Operation | Behavior |
|---|---|
| `createEnrollment`, `getEnrollment`, `revokeEnrollment` | Issue a hash-only token, read its state, revoke unused tokens. Replayed creation never returns plaintext. |
| `enrollNode` | CSR proof, durable incarnation/certificate/provisioning operation; same proof returns the original certificate. |
| `consumeAgentBootstrap` | Authenticated single delivery: 409 not ready, one 200, then 410 consumed; expired undelivered material returns 410 expired. |
| `renewAgentCertificate` | New CSR and serial with bounded older-serial overlap; same renewal ID replays. |
| `listNodes`, `getNode`, `listMemberExits` | Registry and member-scoped exit views, including the current fleet job summary. |
| `updateNode`, `revokeNode`, `retireNode` | Strong If-Match under transaction locks; metadata 200 or pending operation 202; retirement retains history. |
| `setNodePublication` | All three intents use `exits.Service.SetPublicationTx` inside the HTTP command transaction; 202 returns the pending publication operation. |
| `getDesiredState` | Signed state from DB policy and D-030 entitlement readers, explicit release allowlist, authorized `NodeUpdateIntent`, whole-envelope ETag and 304. Published state includes eligible member policies. |
| `renewServingLease` | Signed 24-hour default maximum, bounded member grants, natural renewal-ID replay without restarting deadlines. |
| `submitObservation` | Immutable deduplicated receipt, then `ObserveReportTx` in the same transaction, fleet lock before node lock. |
| `listReleases`, `registerRelease` | Immutable signed manifests; runtime registration requests artifact verification and returns 202. |
| `createUpdateBatch`, `getUpdateBatch`, `retryUpdateBatch` | Immutable all-nodes snapshot, original acceptance replay, cursor targets, new retry attempts. |
| `submitUpdateResult` | Durable 202 ACK; identical result replays, changed payload returns 409 ACK_CONFLICT with retained conflict evidence. |

Agent keys now join the same command transaction as their domain writes (§14.5).
The transactional entry points delegate to the existing implementations rather
than duplicating ingestion logic. Natural report/result/renewal IDs continue to
work without a key.

| Keyed operation | First acceptance | Replay source (no writes) |
|---|---|---|
| `submitObservation` | `SubmitObservationTx`, 202 receipt, node resource | `node_observations.received_at`, authenticated node/incarnation + report ID |
| `submitUpdateResult` | `SubmitResultTx`, 202 receipt, node resource | Stored `update_results.ack`, authenticated node/incarnation + result ID |
| `renewServingLease` | `RenewTx`, 200 envelope, node resource | `leases.ReadRenewal`, the original `serving_leases.envelope` |

Receipts have no synthetic operation. Lease replay preserves the original lease
ID, expiry and signature. `RenewTx` holds node/member/entitlement SHARE locks
until the HTTP command commits; any command failure rolls back the lease,
member grants, audit and HTTP key together. The contract renewal route is
`POST /api/v1/agent/lease-renewals`.

Every API response passes through `apierr`/response tracking and `no-store`.
Certificate authentication uses the verified TLS certificate, current incarnation,
certificate history, effective validity, and independent revocation journal.
`X-Node-ID` and proxy certificate headers cannot select identity. The internal
principal ID selects the authenticated public certificate serial for renewal and
bootstrap; no private key is retrieved or transmitted.

## Publication transaction and remaining boundaries

`SetNodePublication` passes the authenticated principal scope, central request ID,
raw `If-Match` and decoded request to `exits.Service.SetPublicationTx` for
`published`, `unpublished` and `disabled` (§5.2, §14.5). The existing command wrapper
acquires the fleet lock before the service locks nodes and exits. It advances node
and exit resource versions and desired configuration generation once per accepted
command, creates the operation, enqueues `node.publication_changed`, and writes
the `exit:<uuid>` audit event through the core builders. `httpapi.SetResult`
completes the HTTP idempotency record in that same transaction. Replay reads the
stored operation without changing publication or advancing versions again.

Publishing creates `node.publish` with required pending steps `agent_policy`,
`netbird_route_readback` and `data_path_probe`. Unpublishing and disabling create
`node.unpublish` and `node.disable`; their data-path probe is explicitly not
required. These are pending operations, not proof of applied forwarding or routes.
The Tx method uses only the caller's transaction and its clock, so its service
receives the runtime pool and no compiler or NetBird adapter. The request performs
no external calls. Compilation on the following agent desired-state request uses
the merged DB readers and `fup.BuildNodeMemberRatePolicy`; unpublished and disabled
states have no member policies.

Only the explicit DNS configuration and release-support policy fail-closed 503s
remain in `internal/nodes/httpwire`. Publication acceptance is covered by
`TestAT08_HTTPPublicationIntentsReplayAndPreconditions` and command atomicity by
`TestAT08_HTTPPublicationRollsBackWithCommand`.

The contract contains no batch cancel operation or enrollment collection GET.
None is invented. Existing batch cancellation on revocation/retirement remains.
`UpdateTarget` includes both required booleans; `NodeUpdateSummary` has no such
fields in the current contract. The production worker now handles
`node.publication_changed` through `exits.Service.ExecuteOperation`, using the
runtime compiler and real NetBird adapter bridge. It returns the operation ID as
its durable receipt. Successful or superseded operations replay without effects;
a completed central withdrawal with an offline node retains `partially_succeeded`
until an agent observation confirms forwarding is off.

The spec has no §8.4; the applicable §8.1–3 were read. Its AT10 concerns network
isolation; lease-window evidence here corresponds to §16.3/AT29, not an AT10
packet-level pass.

## Worker jobs and read-back

`apps/worker/jobs_nodes.go` builds its registry and clients once from
`Dependencies.Pool`, `Dependencies.Config` and `Dependencies.Logger`. `NodeJobs`
also accepts explicit dependencies for integration tests. No domain opens a pool,
reads environment variables, or starts a listener/dispatcher/goroutine during
construction. `Built.Runners` supplies `RunNodeSchedule` to the process supervisor.

The nodes build creates one `Runtime.DesiredStateCompiler()` and shares it with
the publication executor and member jobs through `Built.Context` and
`membersruntime.WithCompiler`. All context decorators run before dispatch, so
this does not depend on domain build order or change the D-026/D-045 domain list.
The compiler uses the nodes signing provider, `desired_state_seconds`, and
`dns_resolver_ipv4` / `dns_resolver_ipv6`, with the DB entitlement and policy
readers. The members registry keeps ownership of `device.authorized`,
`device.revoked`, `member.entitlement_changed`, and `member.suspended`: after
NetBird authorization read-back, it recompiles affected node policies and records
an audit receipt before acknowledging the event. Compilation failure retries
without acknowledging success. A compiled generation is desired state; applied
generation and node operation steps still require agent observations (§14.5).

| Event | Durable evidence and external calls |
|---|---|
| `node.publication_changed` | Validates the operation/node and current claim, then runs the exits executor. Publication requires agent policy ACK, route/group/policy read-back and stored data-path probe evidence before `active`/`succeeded`. Unpublish/disable remove the route/access policy even offline; an absent agent ACK stays unconfirmed. |
| `node.provision.requested` | `CreateNodeSetupKey(operation_id)` locates the stable name and saved ID before creation. The adapter ensures staging group, creates one-off/usage_limit=1/non-ephemeral key with 86400-second external expiry, and GETs it to verify settings. Only then is bounded AES-GCM ciphertext stored and `netbird_setup` advanced. |
| `node.peer.observe` | Loads the immutable authenticated report; `EnsureNodePeer` GETs the reported peer and validates server/non-ephemeral identity. The one-time deployment/peer binding and provisioning steps commit after read-back. Frozen mappings and revoked/current-incarnation changes are denied. |
| `node.routes.withdraw` (revocation compatibility) | `WithdrawExit` locates route/policy, deletes, and confirms absence. Exit observed state, receipt and operation step are persisted under the current claim. |
| `node.peer.withdraw` (revocation compatibility) | Withdraws exit first, revokes setup keys by stable provisioning operation ID (including lost responses), then reads/binds known revoked-incarnation peers and `DeletePeer` verifies no remaining route reference and deletion. Active reinstall incarnations are excluded. |
| `release.verify` | `VerifyRelease` with `HTTPSFetcher` streams both architecture artifacts with size/hash/origin checks. Fenced terminal verification persists success or failure evidence; failed verification never marks its step successful. |
| `node.reconcile` | `fleetupdate.Reconcile`, expired-bootstrap erasure, then a delayed successor. |
| `fleet.update_created`, `fleet.first_sync_created`, `fleet.update_retry`, `fleet.release_support_changed`, `node.updated` | Reconcile persisted fleet state and seed/coalesce the periodic chain. Desired state is compiled when the authenticated node polls. |

The supervised runner seeds reconciliation immediately at startup and checks for
a missing chain each `reconcile_seconds`. Fleet locking coalesces seeds from
multiple processes and event-created successors. Runners only enqueue; the shared
dispatcher claims and completes work. Enqueue failures return to the supervisor;
cancellation stops the runner. Reconciliation never guesses success for
stuck/offline jobs.

`nodes.NetBirdBindings` aliases `devices.NetBirdBindings`, sharing the
[member/device binding convention](../members/README.md#netbird-binding-and-lock-namespace-convention):
provider `netbird`, reversible encoded keys, JCS SHA-256 resource UUIDs and
PostgreSQL deployment session-lock namespace 119. HTTP calls never hold a SQL
request transaction.
Transport loss after remote key creation may make the one-time secret unavailable:
the worker fails visibly and requires explicit enrollment recovery, never another
automatic key creation. No real NetBird deployment or device behavior is claimed.

Publication uses `httpwire.PublicationAdapter` because `netbird.Client` has a
different method set from `exits.NetBirdAdapter`. It resolves the current
incarnation and deployment, maps member-group UUIDs through `EnsurePaidGroup`,
and calls `Client.PublishExit`. The client preserves stable route bindings,
manual exit selection and source addresses, and reads back group/policy/route
writes. `ReadExitState` independently uses `ReadObservedState`, checks route
identity, peer, groups and forwarding options, and hashes actual read-back.
`WithdrawExit` deletes the route and access policy and confirms their absence;
it does not revoke the node's identity. The bridge matches the client's reserved
stable route name after a binding is deleted, with an integration test against
that client's implementation. It has no raw HTTP bypass and sanitizes failures
before they can become public operation messages.

The executor returns failed outcomes as an operation with a nil Go error. The
job uses the dispatcher's configured `worker_attempts` budget: a failed execution
remains `running` only when `event.Attempt` is below that budget, with failed
step/error evidence retained for the next retry. At or above the budget it stays
`failed` and still returns a job error, so D-040 dead-lettering leaves no delivery
receipt or false claim of progress (§14.5). If the worker dies after that failure,
the dispatcher's exhausted redelivery dead-letters without executing again.
Recovery requires an operator's fresh publication command with a new idempotency
key and current `If-Match`; the failed operation and evidence remain unchanged.
The job never fabricates observation.
A partial offline withdrawal is a completed central action with an unconfirmed
agent step, so replay does not repeat its external effects. The fake-adapter
worker tests and the real-client/local-HTTP test prove these orchestration paths;
they do not prove real NetBird deployment behavior or packet forwarding.

The two legacy withdrawal registrations are still used by
`internal/enrollment/revoke.go`, which creates `route_withdrawal` and
`peer_withdrawal` steps. They are retained for revocation/retirement compatibility;
removing them before that producer is migrated would strand key/peer revocation.
Publication commands exclusively use `node.publication_changed`.

The dispatcher records final delivery under its fenced claim. Release,
bootstrap and legacy withdrawal mutations also check the claim again in their
local transactions. Publication checks its claim before execution and when
retaining a retryable failure. The exits executor owns its step transactions and
honors the job context; those transactions do not yet accept the outbox claim
token. This wiring does not claim per-step zombie-worker fencing for that executor.

## Configuration

The central configuration's required `nodes_config_file` points to a strict JSON
document consumed by both domains through `Dependencies.Config.NodesConfigFile`.
Only the binary's central loader interprets the corresponding
`FVPN_NODES_CONFIG_FILE` environment override. Missing, malformed or unreadable
configuration/credentials fails startup with `config.SettingError` naming
`nodes_config_file`; there is no request-time configuration fallback.
Do not put key material or bearer tokens in JSON. Provision the local
FileProvider vault and public CA certificate explicitly; this development provider
is not a production HSM or independent anti-rollback appliance.

| JSON key | Requirement |
|---|---|
| `provider_directory`, `ca_certificate_file`, `ca_key_id` | Existing vault, public CA PEM and provider-held key reference. Never initialize a new authority silently. |
| `certificate_seconds`, `renewal_overlap_seconds` | Explicit certificate lifetime and nonnegative overlap less than lifetime. |
| `bootstrap_seconds` | Explicit local one-time delivery lifetime, independent of NetBird's external lifetime. |
| `installer_seconds`, `preinstalled_seconds` | Defaults 900 and 604800; configurable. |
| `authorization_seconds`, `online_seconds`, `stuck_seconds` | Explicit fleet authorization/online/watchdog windows. |
| `reconcile_seconds`, `fetch_seconds`, `desired_state_seconds` | Explicit reconciliation, HTTP fetch and desired-envelope durations. |
| `release_origins` | Exact HTTPS origins for immutable manifests/artifacts. |
| `dns_resolver_ipv4`, `dns_resolver_ipv6` | Fleet-wide designated resolver address arrays, validated by family; at least one address across the two arrays is required. No production default. |
| `netbird_url`, `netbird_deployment`, `netbird_token_file` | Adapter `/api` HTTPS URL, deployment scope and bearer-token file. |

A complete **TEST ONLY** configuration shape follows. Provision the vault, CA
key and certificate first; replace the test key reference with that provider's
returned key ID. The non-default time windows below are test values, not approved
production settings.

```json
{
  "provider_directory": "/var/lib/fvpn-control/signing",
  "ca_certificate_file": "/etc/fvpn-control/node-ca.pem",
  "ca_key_id": "test-ca-key-reference",
  "certificate_seconds": 86400,
  "renewal_overlap_seconds": 300,
  "bootstrap_seconds": 300,
  "installer_seconds": 900,
  "preinstalled_seconds": 604800,
  "authorization_seconds": 300,
  "online_seconds": 180,
  "stuck_seconds": 600,
  "reconcile_seconds": 30,
  "fetch_seconds": 30,
  "desired_state_seconds": 600,
  "release_origins": ["https://releases.example.invalid"],
  "dns_resolver_ipv4": ["100.64.0.53"],
  "dns_resolver_ipv6": ["2001:db8::53"],
  "netbird_url": "https://netbird.example.invalid/api",
  "netbird_deployment": "test-deployment",
  "netbird_token_file": "/etc/fvpn-control/netbird-token"
}
```

Resolver addresses come only from this deployment configuration, never a policy
revision payload. The node compiler receives them through
`desiredstate.WithDNSResolvers(ipv4, ipv6)` (see [README §6](README.md#6-desired-state-and-lease-decisions-follow-up-2)).
Both families participate in desired-envelope cache validation; changing either
invalidates the stored ETag. These are fleet-wide settings, with no per-node
override. A single-family deployment may omit the other array.

The fleet's `max_offline_lease_seconds` remains authoritative, default 86400.
Release trust is independently pinned in the vault and reloaded at each manifest
authorization, so an already constructed service cannot retain revoked trust.
Node trust bundles exclude central-only ledger-epoch keys. No release private
key is loaded. Mutable configured files require controlled provisioning and
process restart except the provider's current key/revocation ledger.

## Evidence interpretation

`needs_attention` is the stored watchdog flag OR failed/rolled_back/blocked OR
already_current without the target observed release. `unconfirmed` requires a
matching executing incarnation/generation and a central observation receipt
strictly after issuance and no older than the latest result. Unbound jobs remain
unconfirmed. Batch mapping, `Evidence` and `NodeVersionView` share the SQL
predicates. `ObserveReportTx` retains the actual receipt time instead of substituting
a later processing timestamp. No new evidence migration is needed.

D-048 `UpdateTarget.policy_generation` and `policy_refs` come from the accepted
result selected by the latest job's `observed_evidence.result_id`. `GetBatch`
joins that result in its existing cursor-page query, sharing `Evidence`'s lookup
semantics without a query per target. Missing result/history is explicit JSON
null. Reported `"0"` and known empty references `[]` remain values. Neither a
newer desired state nor a late historical result replaces accepted evidence.

Integration coverage lives under `internal/nodes/httpwire` and uses the real
httpapi middleware, test CA, isolated PostgreSQL databases and local HTTP fakes.
It covers enrollment replay/conflict/reinstall, one-time bootstrap, certificate
revocation/overlap expiry, snapshot stability and first sync, update-result
conflicts, observation dedupe and confirmation, signed control state/ETags,
lease deadlines and replay, NetBird setup read-back/retry, peer readiness, external
key/peer withdrawal and worker fencing.
Full packet isolation, throughput, real arm64 execution and native client
compatibility still require their separate labs.

The composition tests also exercise missing settings (including ignoring ambient
environment in domain builds), immediate/coalesced reconciliation seeding and
cancellation. A subprocess test runs both actual binaries with the production
member and node domains registered, enrolls over TLS, consumes bootstrap over
mTLS once, observes durable scheduled work and verifies clean SIGTERM shutdown.

D-039 limits domain integration tests to core plus the domains under test; only
composition and binary tests use the production `Domains()` list. Shared helper
`internal/nodes/nodestest.New(t, nodestest.NetBird{})` writes a strict **TEST ONLY**
configuration, ephemeral CA/vault and synthetic token under the test's private
disk temporary directory. Its default NetBird URL points at a local TLS fake that
rejects unexpected calls. Tests that exercise adapter requests pass their own
local TLS fake URL, deployment and token file and configure trust for it. The
returned `Path` supplies `NodesConfigFile`; `Nodes.CACertificateFile` supplies the
agent listener CA. The helper has no dependency on either application, changes
no environment variables, and adds no production defaults.

The HTTP desired-state path reads `NewDBPolicyReader(tx)` and
`NewDBEntitlementReader(tx)` in the same fleet-before-node transaction. The
entitlement reader uses D-030 serving projections and persisted FUP evidence;
unpublished nodes contain no member policies. The policy reader supplies active
revision references and the active `dns_publication_heads` generation. The adapter
adds the authorized fleet update intent and filters serving releases through the
explicit fleet allowlist minus revocations. Missing resolver configuration or
release-support policy remains fail closed.

Cache validation compares concrete revision references and the fully compiled
document, including generations, DNS, member eligibility, FUP rates and bindings.
It ignores only the candidate's new generation and issue/expiry timestamps after
checking the stored generation, signature and refresh deadline. The compiler's
signing boundary occurs before persistence: an unchanged document returns the
existing envelope without another desired-state/audit write. A policy revision,
DNS publication, FUP update or member suspension therefore cannot remain hidden
behind an otherwise fresh ETag.

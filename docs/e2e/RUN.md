# Shared lab hooks

Use `FVPN_E2E_PREFIX` and the assigned `FVPN_E2E_PORT_BASE` for your stack.
Runtime objects are `fvpn-<prefix>-*`; immutable per-run image IDs pin neutral
`fvpn-devstack-*` build-cache images. The devstack owner uses `e2e` / `18000`.
`e2e-down` selects that exact prefix. A name belongs to the prefix when it is
`fvpn-<prefix>` or `fvpn-<prefix>-<known token>` (compose service, `pg`,
`pgbackrest-repo`, `nb`, `member` and its peer suffix, `node2`, `node-clone`,
`restore`, `state-reset`). Prefix `e2e` therefore leaves `fvpn-e2e-exports-*`
alone, including while both stacks exist. Never use another owner's running
containers or the shared test PostgreSQL.

```sh
export CGO_ENABLED=0 GOTOOLCHAIN=local
export FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000
export FVPN_E2E_STATE="$PWD/.agent/e2e-state-$FVPN_E2E_PREFIX"
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
make -f mk/e2e.mk e2e-up E2E_RESET=1
```

`up.sh` calls `fvpn-e2e-prepare` to generate keys, trust, certificates and node
configuration. Prepare alone does **not** register a boot release or activate
release support. Use the shared Go helper after admin login, before delivering
any enrollment token:

```go
// Compile the harness with -tags e2e. adminClient has the lab CA, mapped
// loopback dialer and authenticated admin cookie jar. csrf comes from the
// X-CSRF-Token response header of GET /auth/session.
import e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"

baseline, err := e2e.PrepareServingBaseline(ctx, adminClient, csrf, false)
if err != nil {
    t.Fatal(err)
}
t.Logf("boot=%s support_revision=%s", baseline.Boot, baseline.SupportRevision)
```

The helper reads the prepared `FVPN_E2E_STATE`, builds the real agent, watchdog
and updater for amd64 and arm64, packages and signs boot `0.1.0`, uploads it to
the authenticated lab TLS origin, registers it through `POST /admin/releases`,
and polls release verification. It sets the boot ID in both node configurations.
It creates and activates a **non-empty** `release_support` policy through
`POST /admin/policies` and `POST /admin/policies/{definition_id}/activate`, using
the resource version read from `GET /admin/policies`, and polls active-revision
readback. `baseline.Digests` contains artifact, manifest-envelope and signed-object
SHA-256 values. Pass `true` as the final argument for real `1.0.1` and `1.0.2`
candidates; all returned release IDs are explicitly permitted to serve.
Call once per fresh stack: release sequence numbers are immutable.

Afterward, issue an installer enrollment through `POST /admin/enrollments` and
atomically write its returned token to
`$FVPN_E2E_STATE/node/var/e2e/enrollment.token`. The entrypoint handles enrollment,
one bootstrap, the fresh D-055 setup key and NetBird join. Poll product node
observations and readiness; the helper never forces these values. For D-057,
create a group through `POST /admin/groups`, use its ID in the plan policy's
`exit_group_ids` and publication request's `group_ids`, then poll the publication
operation. A 202 response is not completion. Do not replace an unavailable API
with hand-written group, readiness, entitlement, binding or publication rows.

The lab service manager restarts the selected candidate binary for D-053.
Manifest `components.netbird` is `0.82.0`, matching the daemon, and agent
heartbeat/poll intervals are both five seconds. Only the running agent writes
the health attestation. These settings are TEST ONLY.

## Assigned port slots

Offsets remain control `+443`, IdP `+444`, origin `+445`, payment `+446`,
NetBird gateway `+480`, agent mTLS `+1443`, STUN UDP `+3478`, PostgreSQL `+5432`.
Every base is 6000 apart. The existing eight slots keep exactly the same ports
and networks; the two lower bases append subnet slots instead of shifting them.
`66000` is rejected. Origin ranges below are TEST ONLY isolated bridges without
masquerading; they are not documentation prefixes or public traffic targets.

| Base | Owner / binding | Lab subnet | Origin subnet |
|---:|---|---|---|
| 6000 | Reserved; do not start | 172.28.39.0/24 | 192.0.12.0/24 |
| 12000 | Enrollment: `w4-enroll` | 172.28.38.0/24 | 192.0.11.0/24 |
| 18000 | Devstack: `e2e` | 172.28.30.0/24 | 192.0.3.0/24 |
| 24000 | Members | 172.28.31.0/24 | 192.0.4.0/24 |
| 30000 | Secops | 172.28.32.0/24 | 192.0.5.0/24 |
| 36000 | Isolation | 172.28.33.0/24 | 192.0.6.0/24 |
| 42000 | Updates | 172.28.34.0/24 | 192.0.7.0/24 |
| 48000 | Usage | 172.28.35.0/24 | 192.0.8.0/24 |
| 54000 | Agent | 172.28.36.0/24 | 192.0.9.0/24 |
| 60000 | Wire-nodes | 172.28.37.0/24 | 192.0.10.0/24 |

Each origin is `.10` in its listed subnet. This task starts only `e2e` / `18000`.
The enrollment harness owner must replace its old `66000` binding with `12000`.

## Gateway trust and client rate principals

The public nginx location appends `X-Forwarded-For` with
`$proxy_add_x_forwarded_for`. Prepared `central.json` trusts only the gateway's
lab address (`172.28.30.21/32` for `e2e`); no whole lab subnet is trusted.
Control uses the rightmost untrusted address, so a client-supplied prefix in the
header cannot choose a different principal. A direct untrusted connection to
control cannot choose one through that header either. With `trusted_proxies`
empty, every client through this gateway shares its connection-address bucket.
Clients sharing an actual source address still share pre-authentication limits.
The agent mTLS listener remains direct and ignores forwarded headers.
`worker_connections` is a separate nginx connection budget, not a control rate
limit. The live `gateway-principals` record uses only responses and control
metrics, never limiter tables.

## Finite payment grace (SECOPS-04 / AT29)

`fvpn-e2e-prepare -grace-seconds 90` writes `members.json:grace_seconds=90`
and a TEST ONLY reason. The default is `0` (disabled); valid values are
`0..31622400`, matching the members configuration bound. Through bring-up:

```sh
FVPN_E2E_GRACE_SECONDS=90 make -f mk/e2e.mk e2e-up E2E_RESET=1
```

This is a prepare-time option. Without reset, existing prepared state is reused
and the environment variable does not rewrite it. Reset is for a fresh stack,
not for changing a running stack's identity/secrets. Both control and worker
load members configuration at startup: when deliberately editing an existing
lab's `config/members.json`, restart both owned containers before the next
reconciliation; changing a file alone does not update either process. No node
restart is required for this setting; the node receives product-issued policy
and lease updates. Existing grace projections are not rewritten by the lab.

A harness records the configured seconds/reason, control and worker start times,
a paid provider snapshot and successful reconciliation, a positive counted
member flow, then the unpaid provider snapshot/event and read-only projected
`grace_from`/`grace_until`, desired/lease generation, and before/after counters
at the deadline. Do not infer payment or grace from webhook HTTP acceptance.
Only inject control-loss/short-deadline faults after the positive flow. This
option does not set `max_offline_lease_seconds`; D-065's API belongs to
w3-wire-nodes. Clock rollback remains a disposable-guest operator run.

## Portal and payment

`up.sh` builds the Portal with `npm ci` and `npm run build`. To reuse a build,
set `FVPN_E2E_PORTAL_DIST` to a directory containing `index.html` and its assets.
The existing gateway serves that build at the public origin. Its documented
TEST ONLY `/billing/return` hook redirects to the Portal's existing
`/member/billing?checkout=return` route. `/api/` and `/auth/` proxy to control
with upstream TLS certificate verification. This does not run the separate
Portal dev-stack fixture script or seed any of that script's rows.

See [billing](scenarios/billing.md) for authenticated provider-truth hooks and
[backup](scenarios/backup.md) for physical restore commands. The Portal browser
probe uses the existing `apps/portal` Playwright dependency and its Chromium
installation; cookies travel only through stdin and no trace is saved.

## Metrics and identity assumptions

Inside **your** control container, run this bounded HTTP probe:

```sh
docker exec "fvpn-$FVPN_E2E_PREFIX-control" \
  wget --server-response --timeout=3 --tries=1 -O /dev/null \
  http://127.0.0.1:9091/metrics
```

Prepared `metrics_address` is `127.0.0.1:9091`; compose does not override it for
control. Only worker sets `FVPN_METRICS_ADDRESS=:9092`. Metrics is plaintext
HTTP on loopback, not TLS and not reachable from the host or another container.
A counter may have no series before its first event; reachability is determined
by the HTTP response, with metric-family readback after the health request.

SECOPS-06 is covered by `TestAssumeTicketsStayOnTheirSession` in the lab IdP.
Two cookie jars assume different identities, each exchanges its own ticket,
and reuse returns 400. The authenticated hook issues a random one-use
`HttpOnly; Secure; SameSite=Lax` cookie, with an explicit ticket header for
non-browser drivers. No process-wide next-subject value exists. Production-like
portal/native admission uses embedded Dex instead; do not substitute the
legacy assumption endpoint for D-052 subject comparison.

Always finish with `make -f mk/e2e.mk e2e-down` and verify the prefix has no
containers, networks or volumes left. Neutral shared build-cache images stay.

## Follow-up 7: importable stack helpers

AT packages under `tests/e2e` can import
`github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack` with `-tags e2e`.
The package delegates to the acceptance implementation; no source overlay or
`.go.txt` copy is needed. Prepare/start the harness's assigned prefix first.
This task's commands use only `FVPN_E2E_PREFIX=e2e` and port base `18000`.

```go
stack, err := labstack.Open()
if err != nil { t.Fatal(err) }
defer stack.Close()
admin, err := stack.Login(t.Context(), "e2e-admin", "admin")
if err != nil { t.Fatal(err) }
baseline, err := labstack.PrepareServingBaselineWithOptions(
    t.Context(), admin.Client, admin.CSRF,
    labstack.BaselineOptions{Candidates: true},
)
if err != nil { t.Fatal(err) }
node, err := stack.Enroll(t.Context(), admin, "TEST ONLY harness exit")
if err != nil { t.Fatal(err) }
_ = baseline
_ = node
```

`PrepareServingBaseline` retains its existing signature. Both entry points
build/sign/register the real boot and optional candidate bundles for amd64 and
arm64, poll verification, configure the release identities, and activate a
non-empty `release_support` policy through the product API. They must run before
enrollment. `EnrolledNode` means enrollment/bootstrap/join completed; poll node
readiness and publication separately. Set a finite `capacity` with `PATCH
/admin/nodes/{id}` and a finite plan `rate_limit`; the agent cannot render QoS
without those inputs. The acceptance uses TEST ONLY 200 Mbps node capacity,
100 Mbps member caps and a 100 Mbps plan.

For individual candidates use `Stack.BuildRelease`, `RegisterRelease` and
`CandidateConfig`. `BuiltRelease.ManifestSHA256` is the canonical envelope
hash. `ReleaseOptions.AllowedReleaseIDs` controls manifest rollback
compatibility, independently of the serving policy. Its default encodes `[]`
(D-034e: no additional restriction). For a restricted candidate use:

```go
options := labstack.ReleaseOptions{
    AllowedReleaseIDs: labstack.RollbackAllowlist(lastGoodID, otherCompatibleID),
}
release, err := stack.BuildRelease(ctx, directory, "1.0.3", "4", "TEST-ONLY", options)
if err != nil { t.Fatal(err) }
if err := stack.RegisterRelease(ctx, admin, release); err != nil { t.Fatal(err) }
if err := stack.CandidateConfig(release.ID, "1.0.3"); err != nil { t.Fatal(err) }
```

The caller owns `directory` and deletes it after upload/registration. The
combined baseline helper also offers `IncludeBootInRollback` to include its
new boot ID in each candidate's explicitly restricted list. `ActivatePolicy`
creates, activates with If-Match, and polls policy revisions. `Call` accepts
product `/api/v1/` paths using the authenticated session. Never log session
cookies, CSRF values, manifest signing keys or native user credentials.

`Stack.Service(ctx, "start", releaseDirectory)` and `Service(ctx, "stop", "")`
operate only on the stack node. `PauseAgent(ctx, true)` persists the pause marker
so the entrypoint does not restart a stopped agent; pass false to resume. The
boot agent's recovery config is `/etc/vpn-node/updater-recover.json`, created
inside the node after enrollment from its real node/incarnation/boot IDs.
The one-shot `/var/lib/vpn-node/e2e/updater.json` remains the updater-run trigger
and is renamed after use. Recovery is never disabled or treated as success on
an error.

## SECOPS-10 native scale hook

Use the existing stack and a real API-created plan revision with a device limit
of at least three:

```go
ctx, cancel := context.WithTimeout(t.Context(), 45*time.Minute)
defer cancel()
counts, err := stack.ScaleNativeMembers(ctx, admin, planRevisionID, "secops", 20, 3)
if err != nil { t.Fatal(err) }
if counts.Users != 20 || counts.Authorized != 60 || counts.IdentityMatches != 60 {
    t.Fatalf("native scale incomplete: %+v", counts)
}
```

The hook creates 20 distinct embedded-issuer users and 60 native NetBird peers.
Each peer has its own container namespace, NetBird state and socket, using the
existing network and running node's immutable image. The entrypoint's
`FVPN_E2E_MEMBER_NAME` supplies a distinct hostname. Each join compares that
exact peer's NetBird user ID with the member's portal subject, then polls the
product mapping and `/member/devices` authorization with a four-minute bound.
Only counts and bounded final per-stream inventories escape the hook. No second Compose deployment or setup
key masquerading as a member is used. `StartNativePeer` and `JoinNativeMember`
are available separately. Prefix teardown removes all additional peer
containers and volumes. Acceptance runs the full **20 users × 3 peers** cohort. This proves native
admission and the lab gateway budget, not fairness or production hardware capacity.

## SECOPS-04 deadline faults

First prove a native controlled-origin HTTP response and an increase in that
device's `u_`/`d_` admission counters. Only then inject a short paid/grace/grant
deadline through product/provider inputs and pause the agent if required.
Read the resulting signed lease deadline before waiting for withdrawal; retain
the positive and negative counters. An initial fail-closed baseline cannot pass
this case. The lab does not alter projector-owned rows, forge leases, or move
the host clock. Clock rollback requires a disposable guest VM and remains an
operator test. The current counted-flow prerequisite is measured before each fault; see the
dated evidence for the exact signed deadline and measured withdrawal.

## Follow-up 8 restart, DNS and landing observables

Fresh prepared runtime configs already include the D-061 `dns` object shown in
[the threat-feed scenario](scenarios/threat-feed.md); enrollment configs omit it.
Keep the prepared runtime fields when using the shared enrollment helper. A private
harness-generated config that omits `dns` cannot apply an issued resolver document.
The entrypoint changes only the credential file's owner/mode to uid 0 / 0600;
its contents never enter evidence. `threat-feed-node` records the runtime `dns`
object, signed document reference, exact mTLS body digest/size and projected apply
outcome. Its result is independent of `threat-feed-tamper`.

[Node restart](scenarios/node-restart.md) resumes the existing durable identity
without another installer token. The smoke runs before member admission; it is
not a workaround for stale serving leases. Keep recovery enabled.

The gateway serves the built Portal at `/` with an explicit index and readable
public asset permissions. The login walker requires HTTP 200 on a portal landing;
404, 500 and callback errors are failures. `admin-login` records the landing status.

## Native exit selection and traffic trace

The pinned NetBird 0.82.0 CLI does not select an exit (`0.0.0.0/0`) by default;
`netbird networks select all` also leaves that exit unselected. After publication
and native device admission, list networks and explicitly select the single
product-created `fvpn-exit-...` identifier:

```sh
docker exec fvpn-e2e-member netbird networks list
# Set E2E_EXIT_NETWORK_ID to the identifier printed above.
docker exec fvpn-e2e-member netbird networks select "$E2E_EXIT_NETWORK_ID"
docker exec fvpn-e2e-member ip route get 192.0.3.10
```

Acceptance performs this selection with a 15-second route bound. It requires
`dev fvnb0`, a native overlay source, HTTPS 200, the authenticated origin probe's
arrival from the node's SNAT address, and increased device admission bytes.
Selection uses the actual API-published exit; it creates no route or group.
Only the controlled origin's TCP/443 packet headers are captured, inside the
member/node containers. No payloads or public-Internet traffic are collected.
The origin's `PUT|GET /e2e/traffic` hook requires `X-E2E-Secret` from
`origin.secret`, holds one synthetic probe ID, and records only its count and
source address. Ordinary health and release requests are excluded.

## TEST ONLY resolver transport fault

Prepared runtime agents connect to `https://127.0.0.1:9444`. The node entrypoint
runs `fvpn-e2e-nodeconfig resolver-proxy` there with a generated lab TLS
certificate, verifies client certificates against the real agent CA, and uses
that enrolled node's own mTLS identity upstream at
`https://agent.example.invalid:9443`. Enrollment and the updater's result
submission retain their normal endpoints. The proxy has no remotely accessible
fault-control endpoint.

The mode-0600 local marker `corrupt-resolver` in the node's hook directory contains a
document UUID or `*`. While present, only successful resolver-document GET
responses have one byte changed, preserving their length. Remove the marker to
restore transparent transport. `resolver-fault.json` records only document ID,
HTTP status, response count, size and the two SHA-256 digests. The harness arms
this hook before changing the approved lab feed source; it waits for the
worker's configured 300-second refresh/compilation and authentic signed reference.
Both the feed policy and the lab worker's `feed_update_cadence_seconds` are
300 (TEST ONLY); the production worker default remains six hours.
It never changes signatures, desired state, resolver cache or projector rows.

Long fault probes can exceed the configured 900-second recent-auth window.
Perform a fresh admin OIDC login before a protected action such as revoke;
the acceptance harness does so after control-loss withdrawal. Do not lengthen
the security window to make a long test pass.

## Protected native stream scan (SECOPS-16)

`labstack.Stack.ScanNativeJoinLog(ctx, container)` inventories the fixed
`/var/lib/vpn-node/e2e/netbird-up.log` inside an owned member container. It returns
`NativeLogScan{Present, Bytes, Mode, DeviceFlowKeys}` only (the last field counts
lines containing recognized device-flow keys); it does not export
matched lines, URLs, codes or raw content. `JoinNativeMember` always calls it
before removing the transient file and includes the result in `NativePeer.LogScan`.
The scale helper totals present and root-owned mode-0600 streams. Scanner owners
can call the export while a join is in progress and retain its aggregate result;
completed joins return their pre-delete inventory directly.

## Second runnable exit (F3-03)

After preparing the signed serving baseline, use the existing stack:

```go
if err := stack.StartSecondExit(ctx); err != nil { t.Fatal(err) }
second, err := stack.EnrollAt(ctx, admin, "TEST ONLY second exit", "node2")
if err != nil { t.Fatal(err) }
// Use second.NodeID with the normal node/group/publication APIs.
```

`StartSecondExit` pins the running node and resolver image IDs, creates
`node2-netns`, `node2-adguard`, and `node2` under the stack prefix, and uses the
existing two bridges. The second uplink is slot origin `.22`. It copies runtime
configuration and trust into a fresh `node2` directory, changes only that SNAT
address, and obtains a new identity/setup key through normal enrollment. No
identity, state database or readiness row is copied. `Enroll` still targets
`node`; `EnrollAt` accepts only `node` or `node2`. `e2e-down` removes both.
The hook does not itself publish an exit or prove simultaneous multi-exit traffic.
Boot-source variants, physical networks and physical arm64 remain operator runs.

## Dedicated updater probe (D-063/D-064)

Acceptance creates a separate one-off setup key, synthetic NetBird peer and
probe-only group through the lab management API. A separate transport policy
links that group only to a synthetic target group containing the actual node.
It gives only the probe group a
TEST ONLY `/32` controlled-origin route via the actual enrolled node. This does
not alter the product's published exit, member groups, devices or bindings.
The peer selects that narrow route. The updater receives its real peer ID,
overlay source `/32`, and isolated origin `/32` UDP/4443 in `probe_peer`.
The updater config explicitly sets `agent_config` to
`/etc/vpn-node/agent-run.json`; the default `agent.json` is not the lab's runtime
file. The #154 validator reads that file's control/resolver endpoints and the
node's actual interfaces at load, engine creation and challenge issuance. The
isolated origin must differ from every node/control/resolver address. The
existing validator remains authoritative; no denied-range exception is added.

`fvpn-e2e-nodeconfig probe-client <synthetic UUID>` sends bounded authenticated
UDP echo requests from that peer; `probe-watch` only reads nftables snapshots.
Both stop on `e2e/probe-stop` or a 180-second bound. Each 49-byte packet
has a synthetic UUID and a domain-separated HMAC-SHA256; the origin accepts only
the currently armed probe ID and the peer verifies the authenticated reply. The
per-run key is supplied by protected stdin to a mode-0600 peer file and removed
after capture. UDP/4443 is reachable only on the isolated origin bridge; it is
not published on the host. This avoids relying on a cold TLS handshake inside
the short real exception window. The observer records set
sizes/timeouts, diagnostic counters and member-counter totals, never challenge
nonces or peer IDs. Only the product installs/removes the exception and answers
the challenge. `update-good` requires traversal evidence in addition to the
real updater result; an attestation alone cannot satisfy this lab assertion.

For a long scale run, the optional final `ScaleNativeMembers` argument is an
admin-login callback `func(context.Context) (labstack.BrowserSession, error)`.
It performs a real login initially and every five minutes. Acceptance uses it
so grant commands stay inside the unchanged 900-second recent-auth window.
Three independent peers for each user join concurrently; users remain sequential.
The final `SimultaneousAuthorized` count reads all admitted device IDs in one
read-only query, in addition to each peer's identity and authorization checks.

Dynamic native peers and the second node namespace use the lab bridge's
`.128/25` allocation pool. Static services remain below `.128`; a new peer
cannot take a temporarily stopped control container's address during a fault.

## Stale native client control hook

`stack.PauseNativeControl(ctx, container, true)` stops only the named member
container's NetBird userspace process, whose PID the entrypoint saves privately.
Kernel WireGuard and installed routes remain. It does not pause the exit agent,
watchdog, or the container. Require a fresh counted request **after** pausing;
then issue the member DELETE through the API and inspect the node's new applied
generation, ingress capture, `ctunku`, per-device admission counters, origin
receipt and conntrack. Always defer `PauseNativeControl(..., false)` with a
bounded cleanup context. A paused CLI cannot answer `netbird networks list`;
read the kernel route directly instead. This is a controlled stale-client fault,
not a normal user workflow or a repair of a revoked binding.

## Node resource telemetry (AT31)

With `-tags e2e`, `tests/e2e/internal/labstack` exports `NodeTelemetry` and:

```go
collect, err := stack.StartNodeTelemetry(ctx, 5*time.Second)
if err != nil { t.Fatal(err) }
defer collect() // also stop on an early test exit
// Run bounded native admission or counted member traffic.
resources := collect() // stops the sampler and returns all observations
```

The primary acceptance run collects at five-second cadence during member
admission/traffic/usage polling and during the 20-user × 3-peer admission run.
`node_telemetry` sits in those scenario records beside their inventory and flow
evidence. Samples retain UTC time, cgroup v2 cumulative CPU microseconds,
`memory.current` bytes, all node-container process FD counts, process count and
the network namespace's `/proc/sys/net/netfilter/nf_conntrack_count`.
CPU percentage is a delta over actual sample timestamps, with 100% meaning one
fully occupied core; the first sample has no percentage. Docker exec is bounded
per sample. Failed reads increment `errors`, never become fabricated zeros.
The record includes the full series and configured cadence; scheduling jitter
is visible in its timestamps. The command reads only inside the owned node
container and emits no process arguments, FD paths, connection tuples or packet
contents. Numbers include the small observer overhead; memory includes cgroup
page cache. These are workload measurements, not an AT31 hardware certification.
The sampler needs cgroup v2 and readable namespace procfs; it is stopped and
collected before record writing and teardown.

## Follow-up 12 export inventory

Import `github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack` with
`-tags e2e`. All methods below belong to the existing `Stack`; no copied test
source or secondary Compose stack is required. Each operation validates the
prepared stack prefix. The smoke procedure is in
[shared harness exports](scenarios/harness-exports.md).

| Export | Inputs and result |
|---|---|
| `SetupUpdaterProbe(ctx, slot, EnrolledNode)` | `node` or `node2`; creates one dedicated setup-key peer and narrow transport route for this exact enrolled incarnation. Returns `*UpdaterProbe`. Call once per slot/fresh stack. |
| `UpdaterProbe.Config`, `.AgentConfig`, `.Notes` | Exact `probe_peer` JSON-compatible value and `/etc/vpn-node/agent-run.json`; put both into the actual updater invocation (including recovery invocations that should exercise health). Config contains public peer/address information, no setup key. `Notes` are the non-secret setup observations (`probe_not_member`, `probe_route_selected`, scope). Do not publish the raw peer ID. |
| `UpdaterProbe.Observe(ctx)` | Returns a collector. Start before the real updater; call once after completion, including errors. Generates authenticated lab traffic and reads nft state. `ProbeObservation.Verified` requires bounded exception elements, replies/origin receipt, probe counters, unchanged member counters and removal. It never writes a challenge, attestation or nft rule. |
| `GenerateEnrollmentIdentity(ctx, slot, name)` | Optional stack-local P-256 key, CSR and immutable signed request generator for a fresh waiting slot. Returns only `EnrolledIdentityInput` paths. Harnesses may stage their own equivalent files. |
| `StageEnrollmentToken(ctx, slot, input, token)` | Stages a real API-issued token via stdin into root-owned 0600 storage; never an argument/log. The caller must not record the token. |
| `JoinEnrolledIdentity(ctx, slot, input)` | Reads the caller's PKCS8 key, exact signed `api.EnrollRequest` (with CSR), and token at container paths under `/var/lib/vpn-node/e2e/harness/`. `DiscardResponse=true` consumes then discards a successful response before persistence/bootstrap. Replay with the same files; `Join=true` consumes the normal one-time mTLS bootstrap and joins NetBird. Returns only statuses, error code, request digest and public node/incarnation IDs. No agent or central database rows are manufactured. |
| `EnrollmentPeerInventory(ctx, admin, enrollmentID)` | Complete remote key/group/peer collections plus enrollment, audit and node API readback. The setup-key and correlation-group names are the product's `fvpn-enroll-<hash>` and `fvpn-enroll-peer-<hash>` of the `enrollment.consumed` operation id, not the enrollment id. Group peers may be id strings or objects. `Complete` requires that operation's one-off key used once, exactly one projected peer, and at most one correlation group. Discovery deletes that group after binding, so a zero group count after a successful readback is the cleaned state, not a missing correlation. A missing collection or operation id is an error, not a successful zero. This does not supply ENR-03's missing full incarnation-history API. |
| `MachineIdentityFingerprints(ctx, slot)` | SHA-256 of trimmed OS machine-id text, DER agent SPKI and raw NetBird X25519 public key. The native key is read from the client profile `/var/lib/netbird/default.json` inside the node. No private key, certificate or native config is returned. |
| `StartClonedIdentityFixture(ctx, "node")` | Requires unused `node-clone`; stops source, copies durable state, identity and `/var/lib/netbird` into private mounts before clone boot, then resumes source. The clone bind-mounts that profile at `/var/lib/netbird`. A clean `StartSecondExit` still leaves the image's `/var/lib/netbird` writable. Returns a fixture with `Observe(ctx)` for hashes, entrypoint stages and actual read-only conflict/freeze rows. Never publish this clone. Prefix teardown removes it. |
| `NativePeer.Stream`, `NativeScale.Streams` | Final per-stream scan inventory, at most 60 entries for 20×3. Contains synthetic stream name, `scanned/missing/error/not_started`, `NativeLogScan` (including `DeviceFlowKeys`), fixed safe error class and deletion outcome. A nonzero count is not a clean scan; missing/error is not zero. `JoinNativeMember` returns scanner/deletion errors instead of discarding them. |
| `InjectMemberIngress(ctx, iface, source, dest, port)` | Copies `fvpn-netlab` into `fvpn-<prefix>-node` and runs `inject` with an empty `-mechanism`. `fvmb0` (type 65534) selects `tun-mirred`. Returns `MemberIngressResult`: absolute `before`/`after` named counters and `deltas` for counters that moved. Does not write `inet fvpn`, call `wg`, or change the member address, qdisc, or routes. A non-admitted source moves `ctunku` by one packet. An admitted source moves `u_` plus the device UUID with dashes removed. UDP port 9 can also move that same device's `d_` counter when the origin returns ICMP unreachable; other devices' counters stay still. |
| `ObserveMemberIngress(ctx, source, dest)` | Calls `InjectMemberIngress` for `fvmb0` UDP port 9 while tcpdump watches node `fvup0` and the origin network namespace. Returns `MemberIngressProof` (the counter result plus both capture texts). |

`u_<hex>` is the product upload counter (`usageCounter`), not a traffic-class minor.
Compare packet deltas from `Deltas` or from `before`/`after`. The acceptance
device-revoke scenario records both deltas and the two captures under
`runs/2026-10-30/followup13/`. Members and isolation call these methods;
they do not shell out to `fvpn-netlab`.

Probe example (the caller keeps its existing updater invocation):

```go
probe, err := stack.SetupUpdaterProbe(ctx, "node", enrolled)
if err != nil { t.Fatal(err) }
config["probe_peer"] = probe.Config
config["agent_config"] = probe.AgentConfig
collect, err := probe.Observe(ctx)
if err != nil { t.Fatal(err) }
// Write/invoke the real updater using config; never synthesize health files.
// Call collect on every exit path after the invocation.
evidence := collect()
```

The observer is bounded to 180 seconds and must surround the health phase.
An artifact hash failure occurs before health and cannot prove D-063 issuance;
use a separate valid candidate for that assertion. The dev-stack update-good
record adds `configured_probe_issued`: true only when the in-node watcher sees
a fresh `health-challenge.json` `probe` object equal to `UpdaterProbe.Config`.
The watcher prints that boolean and does not copy the peer ID out. Source/destination guards remain the
product validator's responsibility and are never relaxed by the lab.

Protected native device-flow files are separate streams from gateway logs,
container logs and captures. Their final scan happens before deletion even when
a join fails or the caller cancels; cleanup has a separate ten-second bound.
Only counts/mode/result leave the scanner. Other stream inventories remain the
security-ops harness's responsibility.

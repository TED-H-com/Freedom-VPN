# Dev-stack acceptance

This stack is an end-to-end run of a node built from the repository against a
control plane built from the repository (§3, §5–§8, §10, §13, §17, §19). It is
a lab. Prices, the host reserve and policy inputs are TEST ONLY.
Shared setup calls and probe commands are documented in [RUN.md](RUN.md).

## Parameters

| Variable | Default | Meaning |
|---|---|---|
| `FVPN_E2E_PREFIX` | `e2e` | Runtime containers, networks and volumes use `fvpn-<prefix>-*`. |
| `FVPN_E2E_PORT_BASE` | `18000` | Each loopback port is this base plus a fixed offset. |
| `FVPN_E2E_GRACE_SECONDS` | `0` | Prepare-time payment grace; see restart semantics in RUN.md. |
| `FVPN_E2E_STATE` | `.agent/e2e-state-<prefix>` | Git-ignored secrets and rendered config for that prefix. |

Legal port bases are `6000 + 6000*n` for `n` from 0 through 9; `66000`
is rejected. Existing bases `18000..60000` retain their original subnets and
ports. The added `12000` slot belongs to `w4-enroll`; `6000` stays reserved.
See the [binding table](RUN.md#assigned-port-slots) for all owners and subnets.
This task uses only `e2e` / `18000`. Teardown matches only its own prefix:
exactly `fvpn-<prefix>`, or `fvpn-<prefix>-` plus a known service, volume, or
runtime token. Prefix `e2e` does not remove `fvpn-e2e-exports-*`. A native
peer whose name contains `exports` is still removed, because its token is
`member`.

Bring-up tags images only as `fvpn-<prefix>-*:run` and pins locally built
services to immutable image IDs before starting Compose. Docker's shared
layer cache remains reusable without moving another worktree's mutable tag.
Teardown removes the full owned `repository:tag`; it retains upstream cache
layers and pre-existing neutral images. Auxiliary nodes use the running
node's image ID.

## Loopback ports

The harness dials `127.0.0.1` at these ports and keeps the TLS name
`*.example.invalid`. Offsets are added to `FVPN_E2E_PORT_BASE`.

| Port | Offset | Container | Default |
|---|---:|---:|---:|
| Portal + control public (gateway) | 443 | 8443 | 18443 |
| agent mTLS | 1443 | 9443 | 19443 |
| IdP | 444 | 443 | 18444 |
| release origin | 445 | 443 | 18445 |
| payment | 446 | 443 | 18446 |
| NetBird gateway HTTPS | 480 | 443 | 18480 |
| gateway STUN | 3478 | 3478/udp | 21478 |
| Postgres | 5432 | 5432 | 23432 |

## Issuer

One deployment uses one OIDC issuer and public (non-pairwise) subjects for both
the portal and NetBird (D-052, §2 [S36], §14.2). Production must configure that
same issuer and subject contract for both clients; matching email addresses is
not identity evidence. The lab issuer is the embedded Dex issuer of
`netbird-server` 0.82.0:

`https://netbird.example.invalid/oauth2`

`netbird-server` 0.82.0 in the all-in-one layout cannot take an external token
issuer. Its combined config always enables the embedded IdP and sets the
management auth issuer to that IdP. There is no `config.yaml` field for a
different issuer. An identity provider added through the management API is only
a Dex connector, and the stored NetBird user id is Dex's encoded subject, not
the upstream `sub`. The lab therefore does the inverse of an external issuer:
control's `oidc_issuer` is the embedded issuer.

Dex on this image has two static clients, `netbird-dashboard` and
`netbird-cli`, and rejects client registration. The portal uses
`netbird-dashboard`. Prepare adds
`https://portal.example.invalid:8443/auth/callback` to
`server.auth.dashboardRedirectURIs`, which the server copies onto that client.
The CLI keeps `netbird-cli`. Both clients receive the same public `sub` for a
user, and that `sub` is the NetBird user id. This mirrors D-052's one-issuer
public-subject rule. It does not mirror a production external IdP; that
remains open for the production issuer choice.

Dex ID tokens do not carry `roles` or `amr`. Control requires `roles=admin`
for an administrator and `amr` containing `mfa`, and it sends the non-empty
client secret from `oidc_secret_file`. Dex's clients are public and reject
that secret. The lab IdP is a token adapter on two paths only,
`POST /oauth2/token` and `GET /oauth2/keys`. For an authorization-code grant
whose redirect is the portal callback, the adapter drops the client secret,
exchanges the code with Dex, checks Dex's signature, and re-signs an ID token
with the same `iss`, `sub`, and `nonce`, preserving `auth_time` when present
and otherwise using Dex's `iat`, plus TEST ONLY `roles` and `amr: ["mfa"]`.
The injected MFA claim is a lab fixture, not evidence of a real second factor.
The admin role is selected when the Dex profile name or
email is `E2E Admin` or `admin@example.invalid`. Every other grant, including
the CLI device-code grant, is proxied unchanged, so management still validates
Dex signatures with its in-process keys. `GET /oauth2/keys` returns Dex's keys
plus the adapter key `e2e-portal`. Discovery stays Dex's document; its
`jwks_uri` is the intercepted keys URL.

Bootstrap creates `e2e-admin`, `e2e-member`, and `e2e-member-b`. The harness
signs each browser in through Dex's password form as the user for that
session. Any other subject key is created through the management API and
stored under the state directory. The acceptance member and `netbird up` both
use `e2e-member`, so the portal `identity_bindings.subject` and the NetBird
user id are the same value.

`POST /e2e/assume` on the test IdP no longer stores one global next identity.
Each call keeps a one-time ticket in the `e2e_login` cookie and the
`X-E2E-Ticket` header, and `/authorize` reads that ticket. Portal login does
not use this path. The per-session ticket is the SECOPS-06 fix for callers
that still use the test IdP directly.

## Bring-up

From the repository root:

```sh
make -f mk/e2e.mk e2e-up
make -f mk/e2e.mk e2e-test
make -f mk/e2e.mk e2e-down
```

`e2e-up` writes a fresh secret set when `E2E_RESET=1` (which `e2e-test` sets),
builds `fvpn-control`, `fvpn-worker`, and the node binaries, and starts
PostgreSQL 17, the test IdP, a fake payment provider, NetBird CE, AdGuard
Home, CoreDNS, a release origin, one privileged node, and one member
container. Node and AdGuard share a stable namespace held by `node-netns`.
Bridges do not masquerade, so the lab has no working path to the
public Internet. Published ports listen on `127.0.0.1` only.
`e2e-down` removes that prefix and leaves upstream image tags in place.

`e2e-test` runs `go test -tags e2e ./tests/e2e` with `FVPN_E2E=1`. Without
that variable the scenario test skips, so `go test ./...` stays offline.
Pull requests run `e2e-lint` and `docker compose config`. The stack itself
runs on `workflow_dispatch` and the nightly schedule.

## Two stacks

A second prefix can run while the first is still up only on a legal base that
no other task is using. Tear each one down with its own prefix. Do not reuse
`24000` for that: the 2026-10-29 proof in
`docs/e2e/runs/2026-10-29/parallel-stacks.md` used prefix `side` on base
`24000`, and that base is now reserved for a W4 task.

`stack-health` for the default stack is `https://127.0.0.1:18443/readyz`.
Names inside each lab stay `*.example.invalid`. The control port is
`base + 443`.

## Scenario catalogue

| Evidence id | What the run drives | Spec |
|---|---|---|
| `stack-health` | control `/readyz`, IdP, origin, Postgres | §3, §17 |
| `gateway-principals` | distinct browser clients, forged-header refusals, response and metric evidence | §14.2, AT28/AT31 |
| `control-metrics` | bounded in-container HTTP probe of loopback metrics | §17.1 |
| `backup` | archive check, physical full backup and isolated PITR readback | §17.2, §17.3, AT30 groundwork |
| `serving-catalogue` | member group, Fair-Use and plan through APIs | §9, §14.2 |
| `portal-billing-return` | built Portal return route in Chromium | §18.4, AT03 groundwork |
| `billing` | linked provider snapshots, event replay, paid interval, cancellation and refunds | §9.2, §9.3, AT03/AT04 groundwork |
| `admin-login` | OIDC authorization code with MFA; the member session is inside `member-traffic` | §14.2 |
| `enroll-bootstrap` | installer token, bootstrap consumed once, NetBird join | §6, §19 AT06 |
| `desired-state` | signed desired-state file on the node | §5, §8 |
| `node-restart` | durable identity resume, new heartbeat, nft baseline and unchanged enrollment | §5, §6, F2-05 |
| `publish` | `PUT /admin/nodes/{id}/publication`, then poll the operation | §5.2, §14.2 |
| `forwarding` | forward-chain generation and `fvmb0` member accept rules | §5, §19 AT10 |
| `member-traffic` | API-created plan, grant, same-subject embedded-IdP join and device poll | §8, §19 AT12 |
| `native-scale-hook` | 20 distinct native users with three authorized peers each | SECOPS-10 |
| `usage` | `GET /member/usage` and real agent samples after a positive counted flow | §10, §19 AT14, AT15 |
| `release-register` | real packages, registration, verification and API-activated support policy | §13.2–§13.4 |
| `update-good`, `update-bad` | batch, `fvpn-updater run`, tampered artifact | §13, §19 AT23, AT25, AT27 |
| `lease-expiry` | stop control and read the forward chain again | §5, §19 AT29 |
| `threat-feed-central` | approved bootstrap, wrong-pin rejection, audit, worker metrics, resolver document | §12.3, §12.5, §19 AT22 groundwork |
| `threat-feed-node` | issued document, verified mTLS bytes, applied projection and synthetic block | §12.5, §19 AT22 |
| `threat-feed-tamper` | separate corrupted-response and last-good proof | §12.5, §19 AT22 |
| `device-revoke` | two counted native devices, member DELETE, ingress drop, frozen counters and surviving sibling | §8, §10, AT02, F3-02 |
| `gateway-privacy` | forced upstream failure with a synthetic auth query; value-free diagnostics | SECOPS-16, AT28 |
| `second-exit` | independent real enrollment, resolver and nft namespace via shared hooks | F3-03, AT09 groundwork |
| `revoke` | `POST /admin/nodes/{id}/revoke` | §5, §19 AT13 |

AT07 (reinstall of an existing node) is not executed. A `pass` record is only
a claim for the observations in that file. AT10 and AT29 pass only when the
forward chain's member accept rules change. Usage passes only after real counted traffic and agent samples; it does not
claim the separate exact-byte replay assertions of AT14/AT15.

## How to read evidence

Each scenario writes `docs/e2e/runs/<UTC date>/<id>.json` and a short `.md`
summary. Every record in one run carries the same `run_id` and `source_commit`;
the timestamp `at` is the scenario's write time. Replace a dated set only after
all scenarios finish. Status is `pass`, `blocked`, or `fail`. `blocked` means the run
reached a gap and recorded the response; it is not a pass. Observations are
names and short details. Tokens, setup keys, passwords, DSNs, and home paths
are redacted. Raw portal subjects and NetBird user ids are also redacted;
`same_subject` records the comparison made before redaction. The process exits
non-zero when any record is not `pass` (Go test exits 1; make exits 2).

## Portal fixture deviations

`apps/portal/e2e/dev-stack.mjs` is a separate portal test driver. Its fixture
rows do not prove acceptance of publication, device admission, or an updater:

- It hand-seeds the enrolled node's `exits` row to `published` / `active`
  after polling node readiness. The dev-stack acceptance runner instead calls
  the publication API and records its operation or rejection.
- `SEEDED_NODE_2_ID` is a synthetic second node/incarnation and exit with
  `ready`, `passed`, and `healthy` values, including an arm64 label. No second
  node process or arm64 execution is established by those rows.
- The single-column `desired_release_id` fixture writes a target directly to
  `nodes` because the node PATCH API does not set that field; creating a new
  all-nodes update batch would supersede its prearranged batch. This bypasses
  the product update intent path.
- Its `update_results` fixture inserts an accepted result and policy refs for
  the succeeded update job. Those fields exercise portal rendering, including
  policy evidence; they are not a result submitted by a running updater.

## Release and feed inputs

Boot `0.1.0`, candidate `1.0.1`, and tamper target `1.0.2` each contain real
agent, watchdog and updater binaries built from the recorded source commit
for linux/amd64 and linux/arm64, plus immutable version/source metadata.
Distinct artifact digests prevent the tamper target from reusing the good
candidate’s verified download cache. Each manifest is signed, registered and
verified. `release-register` records the SHA-256 of both each canonical manifest
envelope (the registered digest) and its `signed` object. Product version and runtime `agent_release` match; the boot
release ID and both candidates are in the serving allowlist. The shared
`PrepareServingBaseline` helper creates and activates that non-empty policy
through the API; no empty policy or fleet array is written in SQL. `update-bad`
flips one byte in the uploaded real archive after verification, preserving
its length. SBOM, notice and source-download documents are TEST ONLY
placeholders, not production release compliance evidence.

The node's lab service manager stops the boot agent and starts
`<installed-release>/bin/fvpn-agent` with that release's identity/config. A
repeated start is idempotent, as D-053 starts the candidate again after writing
its challenge. The product agent alone may write `health-attest.json`; the
harness never seeds health answers. A serving allowlist entry is insufficient
to invoke the updater: the harness waits for `signed.update.release_id`.

The lab worker explicitly configures `feed_update_cadence_seconds=300`
(TEST ONLY); its production default remains six hours. Prepare creates a deterministic hosts feed with ten unique synthetic entries,
mixed case and a duplicate, and computes its `content_sha256`. The lab TLS
origin serves it inside the isolated network. The scenario creates and
activates a pinned `threat_feed`, then a referencing `dns_profile`, through the
policy API. It polls the worker-owned content, refresh state, audit and
resolver publication rows, and checks the worker's two feed metrics. A second
revision with a wrong pin must fail bootstrap with `content_mismatch` and a
`dns.feed.refresh_failed` audit. Last-good is per revision: the new revision
stays null, while the prior approved content and resolver generation remain.
The document digest and normalized embedded rules are checked without
recording DNS queries. Periodic resolver reconciliation (#127) now issues the
node document reference. Positive node delivery and the tampered-response case
have separate records. D-061 credentials and the local AdGuard listener are
wired and checked. See [threat feed](scenarios/threat-feed.md). No feed is fetched from the public Internet.

## Known limits

- arm64 binaries are built and packaged from the same tree. This host executes
  only amd64; AT07 reinstall and existing-flow teardown are not measured.
- The agent uses the public listener for enrollment and a TEST ONLY loopback
  mTLS proxy when running, through two config files. The proxy authenticates
  to control with the enrolled node identity and can corrupt only resolver
  response bytes through a private local marker. The meter is inside `fvpn-agent`;
  `host_reserve_bps` is a required TEST ONLY value (D-034).
- Artifact and feed fetchers reject private, loopback, overlay and documentation
  addresses. The origin uses `192.0.<3+slot>.10` on an isolated bridge without
  masquerading; no product SSRF allowance or public traffic is introduced.
- Group, plan and release-support inputs use the product APIs. Both approved
  and supported release arrays contain the verified boot and candidate IDs.
  D-050 remains binding: the harness does not write identity/device projections,
  node readiness, publication, feed state or resolver generations. If group
  creation is unavailable, plan creation and live paid-period reconciliation
  remain blocked; no SQL catalogue substitute is used.
- Payment snapshot/event hooks and the built Portal are available to the
  harness owners; see [billing](scenarios/billing.md). Webhook acceptance alone
  never clears a payment gate. Physical PITR is available through the adopted
  backup image and dedicated repository; full AT30 recovery belongs to at-secops.
- Production D-052 identity mapping and D-055 node peer discovery are now in
  the tested main tree. The lab uses their writers and reports their observed
  rows; it no longer treats those implementations as absent. A matching
  subject alone is still insufficient evidence of device authorization.
- D-057 publication uses real groups and checks the operation's data-path step,
  including `member_forwarding_unconfirmed`. HTTP 202 and route readback alone
  cannot complete it. Native D-052 identity authorization is a separate gate.
  Agent follow-up 9 (#146) and wire-nodes follow-up 15 (#148) are included.
  The harness explicitly selects the published NetBird exit: default-route
  networks are not selected by the pinned native CLI. The latest verification
  compares lease/applied generations and actual admission counters, with a
  [hop-by-hop trace](followup9-forwarding-trace.md).
  Exact member counts, deadlines and heartbeat remain in `member-traffic`.
- Only a native controlled-origin request with a device admission counter
  increment can pass forwarding. Usage must come from the real sampler. Expiry
  faults run only after that positive flow; none is claimed from a baseline.
- D-061 resolver configuration uses the stable `node-netns` namespace shared by
  node and AdGuard, with root-owned mode-0600 credentials. The exact runtime
  `dns` object, signed reference and projected apply evidence are in
  `threat-feed-node`. `threat-feed-tamper` separately reports whether a corrupt
  mTLS response was injected; a central wrong pin is not node last-good proof.
- D-053 runs real candidates and records their actual runtime health answers.
  A successful update does not prove serving traffic or rollback to last-good.
- Shared importable helpers and a 20-users × 3-peers scale hook are documented
  in RUN.md. Acceptance executes the full 20 × 3 native admission cohort. Deadline faults,
  production hardware capacity and guest VM clock rollback remain with their
  harness/operator owners.
- The separate portal driver retains the four fixture deviations listed above.
  Those rows are not used by this acceptance runner and cannot clear its gates.
- Dated evidence is replaced as a complete set. The 2026-10-29 directory remains
  the earlier follow-up, including its parallel-stack proof; 2026-10-30 holds
  the latest complete run. Records with `blocked` keep the acceptance command
  non-zero and identify the responsible owner.

## Latest complete run — follow-up 11

[Follow-up 11 verification](followup11-verification.md) records the fresh run
`76f0725c-2264-4021-a553-60f6c921e32b`, source `1b2f358` on product base
`5555772`: **26 pass / 1 blocked / 0 fail**, Go exit 1 / make exit 2.
All 27 dated root record pairs belong to that run. The original follow-up 10
sets and its merged-tree run remain archived; their
[verification](followup10-verification.md) is historical evidence.

Gateway client principals, finite-grace prepare, all ten layout slots and node
resource sampling are verified. Bounded native route discovery now enables
counted member traffic, usage and expiry withdrawal. The stale native client
keeps its route and carries a counted flow while paused before DELETE, but the
post-DELETE node capture is empty. The independent ingress fixture remains
blocked with devstack/netlab; no kernel drop is claimed. See
[owner requests](requests.md) for exact evidence and harness adoption steps.

## Node image prerequisites (F3-02)

The node namespace needs kernel WireGuard, nftables, conntrack, traffic control
(HTB and fq_codel), forwarding, and `/dev/net/tun`. The image supplies pinned
NetBird 0.82.0, `ip`/`tc` from iproute2, `nft`, **conntrack**, CA certificates,
and `wget`; `tcpdump` is a lab diagnostic. Conntrack is a runtime prerequisite:
the product deletes revoked-source connections using `conntrack -D -s/-d`.
Installing it ad hoc or skipping that deletion cannot pass revocation.
The [device-revoke scenario](scenarios/device-revoke.md) exercises the packaged
executable without restarting the agent or clearing its state. Every networking
command executes inside the owned node/member namespace, never on the host.

## Gateway budget and protected diagnostics

The lab budget is 60 cohort peers (20 users × 3), plus up to six scenario/service
peers: primary member, revoked device, replacement, dedicated health probe and
two exits (66 peer slots total). Nginx has two workers
with 4096 connections each; this includes upstream sockets and long-lived
management/signal/relay streams. This is a TEST ONLY lab budget, not a product
capacity certification. HTTP logs contain only server name, method and status.
Native HTTP error logs are suppressed because their built-in format can append
a request URI including device-flow query values; process-level errors remain
on stderr. The controlled `/oauth2/e2e-log-fault` route proxies to a closed
loopback port and proves a 502 diagnostic without its synthetic query value.
`JoinNativeMember` scans each protected CLI stream before deletion and returns
aggregate inventory counts. No matched secret or URI leaves that scan.

The lab gateway appends `X-Forwarded-For`; control trusts only that hop's `/32`.
An empty trust list collapses gateway clients into one pre-authentication
connection bucket. The agent mTLS listener remains direct, and nginx
`worker_connections` is a separate resource limit. See the
[gateway scenario](scenarios/gateway-principals.md) and
[finite grace and telemetry hooks](RUN.md).

## Follow-up 15 scenario additions

| Scenario | Evidence required |
|---|---|
| [established-transports](scenarios/established-transports.md) | Persistent TCP + authenticated member UDP replies and per-device counters move, then stop on cleanup |
| [forced-relay](scenarios/forced-relay.md) | Connected relay path and traffic, then direct restoration on the same sockets |
| [exit-dns](scenarios/exit-dns.md) | TCP/UDP designated DNS traverses the exit and uses DNS-only counters |
| [member-ipv6](scenarios/exit-dns.md) | Explicit blocked native/platform matrix; no address-only pass |
| [harness exports](scenarios/harness-exports.md) | Usage hold/release and parsed snapshot, backlog/restart proof, real systemd installer wrappers |

Follow-up 15 records go directly under the UTC day's `followup15/` directory.
The earlier dated-root evidence is retained. Current hook contracts and open
scope boundaries are in [RUN.md](RUN.md) and the dated harness request notes.
The [2026-10-31 final run](runs/2026-10-31/followup15/README.md) records
acceptance 32 passed / 1 blocked / 0 failed and exports 21 passed / 0 blocked /
1 failed. IPv6 is blocked; real installer service startup is the failed case.

## Follow-up 16 shared fixtures

Answers to other lanes now live in [RESPONSES.md](RESPONSES.md) under D-102.
Only a lane's owner edits its `docs/e2e/at/<lane>/requests.md`.

The import surface is `tests/e2e/internal/labstack` with `-tags e2e`. Use the
assigned prefix and port slot from [RUN.md](RUN.md). The devstack owner runs:

```sh
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
export CGO_ENABLED=0 GOTOOLCHAIN=local
export FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000
FVPN_E2E_TWO_RESOLVERS=true make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E_FIXTURES16=1 go test -v -tags e2e -count=1 -timeout 35m \
  -run '^TestE2E_Followup16Fixtures$' ./tests/e2e
make -f mk/e2e.mk e2e-down
```

This is a fresh-stack campaign. It writes one directory per run UUID beneath
`docs/e2e/runs/<UTC-date>/followup16/`, preserving failed attempts. It does not
invoke the ENR-06 installer wrappers. Every record carries its run/source IDs;
fixture success does not pass an entire AT or an unexecuted platform.

### Finite plan-grant transition (members request 3)

Create the lower plan through `Stack.ActivatePolicy`, then call:

```go
transition, err := stack.ReplaceFiniteGrant(ctx, admin, memberID, lowerPlanRevision,
    time.Now().UTC().Add(time.Hour))
// Always export transition, including on error: a partial transition is useful evidence.
if err != nil { t.Fatal(err) }
if !transition.Projected { t.Fatal("worker projection incomplete") }
```

`memberID` and `lowerPlanRevision` are UUIDs, and `admin` is the authenticated
`BrowserSession`. The member must have a live finite grant and no primary
subscription. The hook resolves the actor from that exact active admin session,
reads the prepared members configuration, calls production
`members.Service.RevokeGrant` for the live grants, then `POST
/api/v1/admin/members/{id}/grants`. The public contract has no revoke route.
It polls the returned operation to `succeeded` and reads the entitlement/device
projection; it never writes projector-owned rows (D-050). Expired/excessive
replacement deadlines and non-grant-only members are rejected before revocation.

Exports include member/plan/operation IDs, revoked grant IDs, finite deadline,
separate `accepted`/`projected` flags, idempotent revoke/create replay results, before/after decimal-string revisions,
device limit, authorized/pending counts and observation time. An error after
revocation leaves the partial result visible; it is not an atomic plan swap.
The same hook restores the original higher plan. Reauthorization stays on the
production HTTP route. The assembled scenario checks three real native peers,
a 3-to-2 downgrade, rejected expired input and restoration to three.

### Native underlay and protocol/platform inputs

Call `PrepareNativeUnderlay(ctx)` on a fresh stack **before enrollment/apply**,
then `InstallFixtureProbe(ctx)` and `StartFixtureTargets(ctx)`. The former attaches a prefix-owned IPv6 bridge
to the actual node and member, and creates private rebinding targets inside the
node namespace. All host networking stays untouched. Adding interfaces to an
already serving node is intentionally not a supported setup path because the
agent must detect topology drift.

```go
selection, err := stack.ApplyNativeInputs(ctx, labstack.NativeFixtureInputs{
    Underlay: "ipv6", MTU: 1280, Protocol: "tcp", Platform: "linux",
})
if err != nil { t.Fatal(err) }
if !selection.Available { t.Skip(selection.Reason) }
observed, err := stack.ObserveNativeUnderlay(ctx)
```

| Input | Lab effect | Export / boundary |
|---|---|---|
| `Underlay: ipv4` | IPv6 peer UDP held on member underlay interfaces | Native direct/relay counts and candidate address-family counts; TCP control stays up |
| `Underlay: ipv6` | IPv4 peer UDP held; native IPv6 link available | NetBird may choose IPv6 or relay; its choice is observed, never prescribed |
| `Underlay: dual` | Both families available | Restores the family filter; no IPv6 overlay-forwarding claim |
| `MTU: 1280..1500` | Sets both member underlay interface MTUs | The campaign sends 64,064 bytes on one TCP connection at 1500 and 1280 with device counters; no global production MTU recommendation |
| `Protocol: tcp/udp` | Declares the workload used with existing member echo hooks | `ApplyNativeInputs` configures networking, not a workload generator; the campaign exercises TCP here, UDP uses `StartEstablishedTransports` |
| `Protocol: quic` | Returns unavailable with a reason before any lab mutation | A pinned real QUIC client/server at the controlled origin is still required; UDP echo is never substituted |
| `Platform: linux` | Uses the real Ubuntu amd64 native container | Actual runtime observation only |
| `Platform: android/ios/windows/macos` | Returns unavailable before mutation | Requires the named real platform/client and the operator campaign; Linux never passes these rows |

`SelectNativeFixture` is the side-effect-free validation/availability entry
point. Unknown values and MTUs outside the range are errors. Family holds expire
in 180 seconds if interrupted; selecting `dual` clears them immediately. Physical
second NAT, roaming, ICMP black holes, QUIC performance, and per-platform IPv6
leaks remain separate measurements. The pinned client's `--disable-ipv6` disables
the **overlay**, not the IPv6 underlay used here.

### Native two-resolver DNS and negative answers

`FVPN_E2E_TWO_RESOLVERS=true` is a prepare-time input. It publishes two exact
origin-network resolver addresses in production desired state and permits AAAA
answers in this lab's AdGuard configuration. Each address transports DNS to a
separate real node's product-managed filter. Call `StartSecondExit` / `EnrollAt`
and wait for both applied policy documents before:

```go
configuration, err := stack.ConfigureNativeDNS(ctx, admin, nativeDeviceID,
    []uuid.UUID{firstNodeID, secondNodeID})
answer, err := stack.ProbeNativeDNS(ctx, "", "rr.example.invalid", 1, false, "")
```

The hook requires identical applied document hashes and reads the two resolver
addresses from the issued desired state. It creates a TEST ONLY NetBird nameserver
group restricted to the admitted member, turns native DNS on, and verifies the
system uses the native stub. This is explicit lab distribution plumbing, not a
claim that product automatic nameserver distribution is wired. It does not
change admission, NetBird ACLs or AdGuard policy via a management write.

An empty resolver argument queries the native stub in `/etc/resolv.conf`;
an explicit `host:port` is a direct resolver probe. The fallback scenario uses
normal Go system resolution, stops the primary transport, checks the secondary
answer and block policy, stops both to require failure, then measures recovery.
Per-query milliseconds and the fault/recovery observations are exported. Resolver
content and query history are not logged. Android Private DNS and browser DoH
remain platform measurements.

The controlled zone declares A, AAAA, CNAME, HTTPS/SVCB, label-boundary and absent
name cases. The campaign activates block/allow rules through the product API and
records expected versus actual RCODE/answer presence for TCP and UDP. For blocked
direct CNAME/SVCB queries the pinned backend may return empty NODATA; A, AAAA,
HTTPS and CNAME-chain A still require NXDOMAIN. No usable answer is accepted in
either negative form. This does not imply all query types return the configured
NXDOMAIN code. Positive
controls prevent an absent authority record from passing a block assertion.
`SetDNSFixturePhase("private")` changes only the owned authority: the cache record
has TTL 10 seconds; the allowlisted rebinding A/AAAA have TTL 5 seconds. Both the SOA RR TTL and its negative minimum are 5 seconds; a formerly absent
record is added in the private phase to measure negative cache expiry. The
positive cache must survive before expiry and change afterward. Private targets
are behind the exit, outside the member's directly connected LAN. Listening
positive controls plus failed member connections establish the selected mode's
boundary. No private network is added to a DNS allow exception. The fixture
exports RR types, TTLs, address classes, expected-answer booleans and timings,
not queried names or browsing history. `SetDNSFixturePhase("initial")` restores
the authority file; existing caches still expire naturally.

### Retained third enrolling fleet target

After `PrepareServingBaseline` but before the fleet operation:

```go
retained, err := stack.RetainEnrollingThirdNode(ctx, admin)
// Run the fleet scenario while this real identity remains enrolling/offline.
booted, err := stack.BootEnrollingThirdNode(ctx, admin)
```

The `node3` slot receives a real preinstalled enrollment through the admin API.
The actual agent CLI runs once against a closed loopback port: it generates and
persists its key, CSR, enrollment request and pending incarnation, then returns
failure without consuming the token. The normal entrypoint remains held. The
central node is genuinely `enrolling/pending` with no bound incarnation, so fleet
snapshot consumers can exercise the enrolling-target branch. No database rows
are fabricated or reset.

The stopped directory snapshot retains those private files. Mid-run boot releases
the marker and completes the original request against the real control URL. The
export contains node ID, pending incarnation/request IDs, a key digest, central
unbound-incarnation evidence, before/after states, retention/boot timestamps,
and `eligible_at`: the first admin API observation of `ready` with the same key
and the formerly pending incarnation now bound centrally. The prepared
preinstalled-token deadline still applies; no expired token is bypassed.
This is a durable directory snapshot, not a VM snapshot or ENR-06 installer.
Publication, campaign membership and update assignment remain AT24/AT26/D-089
consumer assertions. All `node3-*` objects are included in exact-prefix teardown.

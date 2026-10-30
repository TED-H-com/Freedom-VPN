# Shared lab hooks

Use `FVPN_E2E_PREFIX` and the assigned `FVPN_E2E_PORT_BASE` for your stack.
Runtime objects are `fvpn-<prefix>-*`; immutable per-run image IDs pin neutral
`fvpn-devstack-*` build-cache images. The devstack owner uses `e2e` / `18000`.
Never use another owner's running containers or the shared test PostgreSQL.

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
Only aggregate counts escape the hook. No second Compose deployment or setup
key masquerading as a member is used. `StartNativePeer` and `JoinNativeMember`
are available separately. Prefix teardown removes all additional peer
containers and volumes. Acceptance runs only a **2 users × 2 peers smoke**;
this does not claim 20 × 3 capacity or fairness.

## SECOPS-04 deadline faults

First prove a native controlled-origin HTTP response and an increase in that
device's `u_`/`d_` admission counters. Only then inject a short paid/grace/grant
deadline through product/provider inputs and pause the agent if required.
Read the resulting signed lease deadline before waiting for withdrawal; retain
the positive and negative counters. An initial fail-closed baseline cannot pass
this case. The lab does not alter projector-owned rows, forge leases, or move
the host clock. Clock rollback requires a disposable guest VM and remains an
operator test. The current missing lease refresh prevents this prerequisite;
acceptance records the dependent expiry test as blocked without injecting it.

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

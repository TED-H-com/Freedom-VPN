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

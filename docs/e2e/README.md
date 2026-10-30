# Dev-stack acceptance

This stack is an end-to-end run of a node built from the repository against a
control plane built from the repository (§3, §5–§8, §10, §13, §17, §19). It is
a lab. Prices, the host reserve, and the release-support row it inserts are
TEST ONLY.

## Parameters

| Variable | Default | Meaning |
|---|---|---|
| `FVPN_E2E_PREFIX` | `e2e` | Runtime containers, networks and volumes use `fvpn-<prefix>-*`. |
| `FVPN_E2E_PORT_BASE` | `18000` | Each loopback port is this base plus a fixed offset. |
| `FVPN_E2E_STATE` | `.agent/e2e-state-<prefix>` | Git-ignored secrets and rendered config for that prefix. |

Legal port bases are `18000 + 6000*n` for `n` from 0 through 7. The step is
larger than the widest offset, so two legal bases cannot publish the same
host port. Slot `n` uses lab subnet `172.28.<30+n>.0/24` and release origin
`192.0.<3+n>.10`. `e2e-down` removes only names matching `^fvpn-<prefix>-`
for containers, volumes, and images, and `^fvpn-<prefix>(-|$)` for networks.
It does not match another prefix.

## Who may use which base

| Base | Prefix rule |
|---|---|
| `18000` | This lab's default. The acceptance run uses prefix `e2e` and this base only. |
| `24000`, `30000`, `36000`, `42000`, `48000` | Reserved for the W4 acceptance tasks. Do not start, stop, or reset those prefixes from this task. |
| `54000`, `60000` | Legal and currently unassigned. Do not start them unless a task owns that slot. |

Images use neutral `fvpn-devstack-*` tags, including
`fvpn-devstack-netbird:0.82.0`, so harnesses share the Docker build cache.
Bring-up pins locally built services to their immutable image IDs before
starting Compose; a concurrent worktree moving a shared `:local` tag cannot
change that run's selected image. Auxiliary update nodes use the running
node's image ID. Teardown retains the shared images.

## Loopback ports

The harness dials `127.0.0.1` at these ports and keeps the TLS name
`*.example.invalid`. Offsets are added to `FVPN_E2E_PORT_BASE`.

| Port | Offset | Container | Default |
|---|---:|---:|---:|
| control public | 443 | 8443 | 18443 |
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
container. Bridges do not masquerade, so the lab has no working path to the
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
| `admin-login` | OIDC authorization code with MFA; the member session is inside `member-traffic` | §14.2 |
| `enroll-bootstrap` | installer token, bootstrap consumed once, NetBird join | §6, §19 AT06 |
| `desired-state` | signed desired-state file on the node | §5, §8 |
| `publish` | `PUT /admin/nodes/{id}/publication`, then poll the operation | §5.2, §14.2 |
| `forwarding` | forward-chain generation and `fvmb0` member accept rules | §5, §19 AT10 |
| `member-traffic` | seeded plan, grant, same-subject embedded-IdP join, device poll, checkout | §8, §19 AT12 |
| `usage` | `GET /member/usage` and one idempotent usage-batch replay of a projected binding | §10, §19 AT14, AT15 |
| `release-register` | `fvpn-release` package, register, verify | §13.3, §13.4 |
| `update-good`, `update-bad` | batch, `fvpn-updater run`, tampered artifact | §13, §19 AT23, AT25, AT27 |
| `lease-expiry` | stop control and read the forward chain again | §5, §19 AT29 |
| `threat-feed-central` | approved bootstrap, wrong-pin rejection, audit, worker metrics, resolver document | §12.3, §12.5, §19 AT22 groundwork |
| `threat-feed-node` | explicit node-delivery gap (D-056) | §12.5, §19 AT22 |
| `revoke` | `POST /admin/nodes/{id}/revoke` | §5, §19 AT13 |

AT07 (reinstall of an existing node) is not executed. A `pass` record is only
a claim for the observations in that file. AT10 and AT29 pass only when the
forward chain's member accept rules change. AT14 and AT15 pass only when the
member total matches the uploaded bytes.

## How to read evidence

Each scenario writes `docs/e2e/runs/<UTC date>/<id>.json` and a short `.md`
summary. Every record in one run carries the same `run_id` and `source_commit`;
the timestamp `at` is the scenario's write time. Replace a dated set only after
all fifteen scenarios finish. Status is `pass`, `blocked`, or `fail`. `blocked` means the run
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
for linux/amd64 and linux/arm64. Each manifest is signed, registered and
verified. `release-register` records the SHA-256 of each canonical `signed`
manifest object. Product version and runtime `agent_release` match; the boot
release ID and both candidates are in the serving allowlist. `update-bad`
changes the uploaded real archive only after verification.

The node's lab service manager stops the boot agent and starts
`<installed-release>/bin/fvpn-agent` with that release's identity/config. A
repeated start is idempotent, as D-053 starts the candidate again after writing
its challenge. The product agent alone may write `health-attest.json`; the
harness never seeds health answers. A serving allowlist entry is insufficient
to invoke the updater: the harness waits for `signed.update.release_id`.

Prepare creates a deterministic hosts feed with ten unique synthetic entries,
mixed case and a duplicate, and computes its `content_sha256`. The lab TLS
origin serves it inside the isolated network. The scenario creates and
activates a pinned `threat_feed`, then a referencing `dns_profile`, through the
policy API. It polls the worker-owned content, refresh state, audit and
resolver publication rows, and checks the worker's two feed metrics. A second
revision with a wrong pin must fail bootstrap with `content_mismatch` and a
`dns.feed.refresh_failed` audit. Last-good is per revision: the new revision
stays null, while the prior approved content and resolver generation remain.
The document digest and normalized embedded rules are checked without
recording DNS queries. On-node feed delivery remains separately blocked on
D-056 (contracts revision 5 and node application); the central result does not
claim full AT22. No feed is fetched from the public Internet.

## Known limits

- arm64 is packaged and cross-compiled. This host does not boot an arm64 node.
- There is one agent `control_url`. Enrollment uses the public listener and
  the running agent uses the mTLS listener, via two config files.
- The release fetcher rejects RFC1918, loopback, CGNAT, and documentation
  ranges. Each stack's origin is `192.0.<3+slot>.10` on a bridge with
  masquerade off. There is no central test allowance for the origin URL.
- Activation of `release_support` does not by itself fill
  `approved_release_ids`. The run writes both allowlist arrays in SQL so a
  batch is not rejected as `RELEASE_NOT_APPROVED`, and it does not bump
  `generation`, so `expected_generation` still matches the API.
- Before enrollment the acceptance runner packages the tree-built agent,
  watchdog, and updater for amd64 and arm64 as a TEST ONLY boot release
  (`0.1.0`). It registers and verifies that release and both update fixtures,
  sets the boot UUID in the node configs, and includes all three IDs in the
  serving allowlist. `desired-state` records whether the boot ID is supported.
  Previously the non-empty update allowlist excluded the node's unset release
  ID, which correctly caused `release_supported=failed`. The `1.0.1` and
  `1.0.2` update archives remain marker fixtures for the updater exercise.
- Stripe's API host is fixed in the client. The lab publishes it as the
  Docker alias `api.stripe.com` with a certificate for that name. The lab
  seeds one plan whose provider is `stripe` and whose price id is `e2e-price`.
- The lab seeds one plan revision that includes an exit group and passes that
  `plan_revision_id` on the admin grant. The member container joins with
  `netbird up` and no setup key. The portal session and the device flow both
  sign in as `e2e-member` at the embedded issuer, and the harness unescapes
  the HTML form action so the password POST keeps the `state` query. It does
  not insert `member_devices`, `device_bindings`, or `netbird_user_bindings`.
  An empty `netbird_user_bindings` table is recorded as `writer not yet merged`
  while that writer is absent. The 2026-10-30 run recorded grant 202, checkout
  201, device path `/oauth2/device`, login submitted, stage `netbird_up`, and
  one peer of two with a NetBird user. `GET /api/v1/member/devices` was 200
  with zero devices. `same_subject=true` confirms that the portal subject and
  NetBird identity match before redaction. Stored row counts for this member
  were `netbird_user_bindings=0`, `member_devices=0`, and `device_bindings=0`.
  This is the missing production mapping writer, not an issuer mismatch.
  No binding was projected and no flow was counted, so AT12 is not claimed.
- After enroll the run calls `PUT /admin/nodes/{id}/publication` with intent
  `published` and `If-Match`. The 2026-10-30 call returned 422
  `NODE_NOT_SERVICEABLE` before an operation existed, with only
  `provisioning_state=applying (want ready)` in `failed_checks`. The accepted
  observation now reports `ready`, `safety_status=passed`, `data_health=healthy`,
  `tunnel_probe=passed`, and applied generation `1`. The stored node still has
  `provisioning_state=applying`, although desired and applied are both `1`.
  Its incarnation peer id and the report's NetBird peer id are null, and there
  are zero `node.peer.observe` events: the HTTP observer only enqueues that
  worker job for a non-null peer id. The worker also requires that id before
  its ready transition. This is the central wire-nodes blocker; the agent
  baseline fix from PR #75 is visible. `publication_observed` stays absent,
  so `agent_policy`, `netbird_route_readback`, and `data_path_probe` were not
  reached. `publish` includes the stored row snapshot without seeding a fix.
- `GET /api/v1/member/usage` is served. A usage batch is sent only when a
  desired-state `device_bindings` row matches an authorized device and covers
  the interval. The 2026-10-30 run did not send a batch. The member total
  stayed 0, so AT14 and AT15 are not claimed.
- A member accept rule is an `fvmb0` accept on the `inet fvpn` forward chain.
  NetBird `fvnb0` accepts in that chain do not count, and neither does
  `policy accept` on another chain. The 2026-10-30 forward chain stayed
  `policy drop`, generation 0, with the baseline marker and zero member
  accepts, so AT10 and AT29 are not claimed.
- The 2026-10-30 enroll record shows the node signal log `Connected` after
  join. An earlier lab run recorded a relay WebSocket 404 and one management
  `RST_STREAM`; this run's evidence does not repeat that measurement.
- Stock updater `NodeChecks` fail closed. While the node was not serving, the
  2026-10-30 good-release journal `health_json` was
  `{"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"}`.
  That update ended `recovery_required` with `ROLLBACK_FAILED`. The bad
  release was rejected with `HASH_MISMATCH: content changed during resume`;
  its record passes only the tampered-byte rejection observation. It has no
  journal health row and proves no successful rollback. `locally_succeeded`
  was not observed; full AT23, AT25, and AT27 completion is not claimed.
  The harness waits for the release in `signed.update.release_id` before
  invoking the updater; presence in `serving_release_ids` alone is insufficient.
- The meter runs inside `fvpn-agent`. There is no separate meter or
  time-anchor binary. `host_reserve_bps` is set in the generated agent file
  because the binary has no default (D-034).
- `docs/e2e/runs/<UTC date>/` is the latest run for that date. A later run on
  the same UTC date replaces those files. The acceptance table for this
  follow-up is `docs/e2e/runs/2026-10-30/`. `docs/e2e/runs/2026-10-29/` remains
  the follow-up 2 run, including `parallel-stacks.md`. The first recorded run
  of 2026-10-29 is the copy in `acc17fb`. A forward chain that stays on the
  baseline is not an AT10 or AT29 pass.

The final 2026-10-30 run is `6bf270e4-abb1-43ff-93eb-18d282fc7c2a`, from
source commit `5477de091f449047193fdc9114cfca4ced5b36e6`. All thirteen scenarios
completed: seven pass and six blocked, with no scenario labeled fail.
`TestE2E_Acceptance` took 197.36 seconds and exited 1 (`make` exited 2),
preserving the non-pass gate. These records replace the earlier attempts on
that date; they are not a combination of runs.

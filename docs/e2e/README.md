# Dev-stack acceptance

This stack is an end-to-end run of a node built from the repository against a
control plane built from the repository (§3, §5–§8, §10, §13, §17, §19). It is
a lab. Prices, the host reserve, and the release-support row it inserts are
TEST ONLY.

## Parameters

| Variable | Default | Meaning |
|---|---|---|
| `FVPN_E2E_PREFIX` | `e2e` | Every container, network, volume, and image is `fvpn-<prefix>-*`. |
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

Image names stay `fvpn-<prefix>-*` (`fvpn-e2e-postgres:17`, and the same shape
for the other services). A shared `fvpn-devstack-*` tag would collide when
two prefixes are up, so the prefix stays in the image name.

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
| `member-traffic` | seeded plan, grant, embedded-IdP user join, device poll, checkout | §8, §19 AT12 |
| `usage` | `GET /member/usage` and one idempotent usage-batch replay of a projected binding | §10, §19 AT14, AT15 |
| `release-register` | `fvpn-release` package, register, verify | §13.3, §13.4 |
| `update-good`, `update-bad` | batch, `fvpn-updater run`, tampered artifact | §13, §19 AT23, AT25, AT27 |
| `lease-expiry` | stop control and read the forward chain again | §5, §19 AT29 |
| `revoke` | `POST /admin/nodes/{id}/revoke` | §5, §19 AT13 |

AT07 (reinstall of an existing node) is not executed. A `pass` record is only
a claim for the observations in that file. AT10 and AT29 pass only when the
forward chain's member accept rules change. AT14 and AT15 pass only when the
member total matches the uploaded bytes.

## How to read evidence

Each scenario writes `docs/e2e/runs/<UTC date>/<id>.json` and a short `.md`
summary. Status is `pass`, `blocked`, or `fail`. `blocked` means the run
reached a gap and recorded the response; it is not a pass. Observations are
names and short details. Tokens, setup keys, passwords, DSNs, and home paths
are redacted. The process exits non-zero when any record is not `pass`.

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
- Stripe's API host is fixed in the client. The lab publishes it as the
  Docker alias `api.stripe.com` with a certificate for that name. The lab
  seeds one plan whose provider is `stripe` and whose price id is `e2e-price`.
- The lab seeds one plan revision that includes an exit group and passes that
  `plan_revision_id` on the admin grant. The member container joins with
  `netbird up` and no setup key. The harness completes the embedded IdP
  device flow and unescapes the HTML form action so the password POST keeps
  the `state` query. It does not insert `member_devices`, `device_bindings`,
  or `netbird_user_bindings`. The 2026-10-30 run recorded grant 202, checkout
  201, device path `/oauth2/device`, login submitted, stage `netbird_up`, and
  one peer of two with a NetBird user. `GET /api/v1/member/devices` was 200
  with zero devices. No binding was projected and no flow was counted, so
  AT12 is not claimed.
- After enroll the run calls `PUT /admin/nodes/{id}/publication` with intent
  `published` and `If-Match`. The 2026-10-30 call returned 422
  `NODE_NOT_SERVICEABLE` before an operation existed. The failed checks were
  `provisioning_state=applying (want ready)` and `safety_status=failed`.
  `publication_observed` stayed absent, so the poll of `agent_policy`,
  `netbird_route_readback`, and `data_path_probe` did not start.
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
  release has no journal row, so `HASH_MISMATCH` was not observed.
  `locally_succeeded` was not observed. AT23, AT25, and AT27 are not claimed
  from this run.
- The meter runs inside `fvpn-agent`. There is no separate meter or
  time-anchor binary. `host_reserve_bps` is set in the generated agent file
  because the binary has no default (D-034).
- `docs/e2e/runs/<UTC date>/` is the latest run for that date. A later run on
  the same UTC date replaces those files. The acceptance table for this
  follow-up is `docs/e2e/runs/2026-10-30/`. `docs/e2e/runs/2026-10-29/` remains
  the follow-up 2 run, including `parallel-stacks.md`. The first recorded run
  of 2026-10-29 is the copy in `acc17fb`. A forward chain that stays on the
  baseline is not an AT10 or AT29 pass.

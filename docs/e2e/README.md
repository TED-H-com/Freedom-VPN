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

Legal port bases are `18000 + 6000*n` for `n` from 0 through 7 (`18000`,
`24000`, `30000`, `36000`, `42000`, `48000`, `54000`, `60000`). The step is
larger than the widest offset, so two legal bases cannot publish the same
host port. Slot `n` uses lab subnet `172.28.<30+n>.0/24` and release origin
`192.0.<3+n>.10`. `e2e-down` removes only `fvpn-<prefix>` and `fvpn-<prefix>-*`.

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

A second prefix can run while the first is still up. Use another legal port
base. Tear each one down with its own prefix:

```sh
make -f mk/e2e.mk e2e-up
FVPN_E2E_PREFIX=side FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-up
FVPN_E2E_PREFIX=side FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
make -f mk/e2e.mk e2e-down
```

`stack-health` for the default stack is `https://127.0.0.1:18443/readyz`.
For base 24000 it is `https://127.0.0.1:24443/readyz`. Names inside each lab
stay `*.example.invalid`.

## Scenario catalogue

| Evidence id | What the run drives | Spec |
|---|---|---|
| `stack-health` | control `/readyz`, IdP, origin, Postgres | §3, §17 |
| `admin-login` | OIDC authorization code with MFA; the member session is inside `member-traffic` | §14.2 |
| `enroll-bootstrap` | installer token, bootstrap consumed once, NetBird join | §6, §19 AT06 |
| `desired-state` | signed desired-state file on the node | §5, §8 |
| `forwarding` | `inet fvpn` forward-chain policy and member accept rules | §5, §19 AT10 |
| `member-traffic` | seeded plan, grant, device setup key, checkout | §8, §19 AT12 |
| `usage` | `GET /member/usage` and one idempotent usage-batch replay | §10, §19 AT14, AT15 |
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
- The lab seeds one plan revision that includes an exit group, passes that
  `plan_revision_id` on the admin grant, and creates the member setup key in
  the NetBird group `fvpn-unpublished-nodes` (never the All group). The
  2026-10-29 run recorded grant 202, checkout 201, and the member container
  at `netbird_up`. No flow was counted, so AT12 is not claimed.
- `GET /api/v1/member/usage` is served. The node uploads one usage batch with
  an `Idempotency-Key` and repeats it. The 2026-10-29 replay returned the
  same `results`. The event was `rejected` / `VALIDATION_FAILED` /
  `binding_not_found`, and the member total stayed 0, so AT14 and AT15 are
  not claimed.
- A member accept rule is an `fvmb0` accept on the `inet fvpn` forward chain.
  NetBird `fvnb0` accepts in that chain do not count, and neither does
  `policy accept` on another chain. The 2026-10-29 forward chain stayed
  `policy drop` with the baseline marker and zero member accepts, so AT10
  and AT29 are not claimed.
- The node client log before the signal fix was `DeadlineExceeded` dialing
  `netbird.example.invalid`. After the fix the client log is `Connected`.
  The gateway does not serve the NetBird relay WebSocket; the client records
  HTTP 404 for `rels://netbird.example.invalid:443`. One management job
  stream logged `RST_STREAM` / `PROTOCOL_ERROR` and retried.
- Stock updater `NodeChecks` fail closed. The 2026-10-29 journal
  `health_json` has `internet_only`, `dns_policy`, and `tunnel_probe` failed,
  and `processes` and `identity_and_state` passed. The good update ended
  `recovery_required` with `ROLLBACK_FAILED`. The tampered artifact logged
  `HASH_MISMATCH`. The result outbox was still `pending`. `locally_succeeded`
  was not observed.
- The meter runs inside `fvpn-agent`. There is no separate meter or
  time-anchor binary. `host_reserve_bps` is set in the generated agent file
  because the binary has no default (D-034).
- `docs/e2e/runs/<UTC date>/` is the latest run for that date. A later run on
  the same UTC date replaces those files. The first recorded run of
  2026-10-29 is the copy in `acc17fb`. A forward chain that stays on the
  baseline is not an AT10 or AT29 pass.

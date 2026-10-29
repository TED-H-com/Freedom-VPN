# Dev-stack acceptance

This stack is the first end-to-end run of a node built from the repository
against a control plane built from the repository (§3, §5–§8, §10, §13, §17,
§19). It is a lab. Prices, the host reserve, and the release-support row it
inserts are TEST ONLY.

## Bring-up

From the repository root:

```sh
make -f mk/e2e.mk e2e-up
make -f mk/e2e.mk e2e-test
make -f mk/e2e.mk e2e-down
```

`e2e-up` writes a fresh secret set under `.agent/e2e-state` when `E2E_RESET=1`
(which `e2e-test` sets), builds `fvpn-control`, `fvpn-worker`, and the node
binaries, and starts PostgreSQL 17, the test IdP, a fake payment provider,
NetBird CE, AdGuard Home, CoreDNS, a release origin, one privileged node, and
one member container. Every container, network, volume, and image name starts
with `fvpn-w3-devstack-e2e`. Nothing is attached to the host network.
`e2e-down` removes those objects and leaves upstream image tags in place.

`e2e-test` runs `go test -tags e2e ./tests/e2e` with `FVPN_E2E=1`. Without
that variable the scenario test skips, so `go test ./...` stays offline.
Pull requests run `e2e-lint` and `docker compose config`. The stack itself
runs on `workflow_dispatch` and the nightly schedule.

The lab bridges are internal, so this Docker daemon does not publish host
ports. The harness dials the fixed container addresses: control
`172.28.30.30:8443` and `:9443`, IdP `172.28.30.11:443`, NetBird
`172.28.30.21:443`, release origin `192.0.3.10:443`, PostgreSQL
`172.28.30.10:5432`. Names inside the lab are `*.example.invalid`.

## Scenario catalogue

| Evidence id | What the run drives | Spec |
|---|---|---|
| `stack-health` | control `/readyz`, IdP, origin, Postgres | §3, §17 |
| `admin-login` | OIDC authorization code with MFA; the member session is inside `member-traffic` | §14.2 |
| `enroll-bootstrap` | installer token, bootstrap consumed once, NetBird join | §6, §19 AT06 |
| `desired-state` | signed desired-state file on the node | §5, §8 |
| `forwarding` | nftables `inet fvpn` and whether member forwarding is on | §5, §19 AT10 |
| `member-traffic` | grant, device list, checkout attempt | §8, §19 AT12 |
| `usage` | `GET /member/usage` | §10, §19 AT14, AT15 |
| `release-register` | `fvpn-release` package, register, verify | §13.3, §13.4 |
| `update-good`, `update-bad` | batch, `fvpn-updater run`, tampered artifact | §13, §19 AT23, AT25, AT27 |
| `lease-expiry` | stop control and read the nftables table again | §5, §19 AT29 |
| `revoke` | `POST /admin/nodes/{id}/revoke` | §5, §19 AT13 |

AT07 (reinstall of an existing node) is not executed. A `pass` record is only
a claim for the observations in that file.

## How to read evidence

Each scenario writes `docs/e2e/runs/<UTC date>/<id>.json` and a short `.md`
summary. Status is `pass`, `blocked`, or `fail`. `blocked` means the run
reached a product gap and recorded the response; it is not a pass.
Observations are names and short details. Tokens, setup keys, passwords, DSNs,
and home paths are redacted. The process exits non-zero when any record is
not `pass`.

## Known limits

- arm64 is packaged and cross-compiled. This host does not boot an arm64 node.
- `policies()` and `usage()` are not in control `Domains()`. Unpublished
  desired state can be signed after a `release_support` row exists. Publishing
  a node returns 503. `GET /member/usage` returns 501 `NOT_IMPLEMENTED`.
- The agent always sends `Idempotency-Key` on lease renewal, and renewal then
  returns 503. Forwarding therefore stays at the fail-closed baseline, and
  AT29 cannot show a lease that was on and then turned off.
- `fvpn-updater` health checks fail closed for isolation, DNS, and the tunnel
  probe. A good package is not expected to report `locally_succeeded`.
- There is one agent `control_url`. Enrollment uses the public listener and
  the running agent uses the mTLS listener, via two config files.
- The release fetcher rejects RFC1918, loopback, CGNAT, and documentation
  ranges. The origin is `192.0.3.10` on an isolated bridge (no default route), which is a
  routable unicast that the fetcher accepts. There is no central test
  allowance for the origin URL.
- Activation of `release_support` does not by itself fill
  `approved_release_ids`. The run writes both allowlist arrays in SQL so a
  batch is not rejected as `RELEASE_NOT_APPROVED`, and it does not bump
  `generation`, so `expected_generation` still matches the API.
- Stripe's API host is fixed in the client. The lab publishes it as the
  Docker alias `api.stripe.com` with a certificate for that name. There is
  no plan catalogue, so checkout is not expected to create a subscription.
- No HTTP route authorizes a member device. Authorization is worker discovery
  of a NetBird peer after an entitlement. The lab member is a separate
  NetBird user from the OIDC member, so a device may not appear.
- The meter runs inside `fvpn-agent`. There is no separate meter or
  time-anchor binary. `host_reserve_bps` is set in the generated agent file
  because the binary has no default (D-034).

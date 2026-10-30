# How to Run `w4-at-usage` (AT14–AT19)

## 1. Unit & Mechanism Verification (No Docker Required)

```bash
make -f mk/at-usage.mk at-usage-lint
make -f mk/at-usage.mk at-usage-test
make -f mk/at-usage.mk at-usage-build
```

## 2. Live Counted Serving Flow on Assigned Stack (`w4-usage` / Port Base `48000`)

```bash
FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-usage.mk at-usage
FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/e2e.mk e2e-down
```

## 3. Product-Only Bootstrap Path Verified in Run `9d561888` (`2026-10-30T21:31:58Z`)

1. `e2e.PrepareServingBaseline(ctx, admin.Client, admin.CSRF, false)` builds, signs, uploads, registers, and verifies `0.1.0` (`amd64` + `arm64`) and activates `release_support` (`01a18e52-a293-7e1b-b22e-ed9f7341ab29`).
2. `POST /api/v1/admin/groups` creates member exit group `01a18e52-a4e3-7981-983d-bcb37866f801`.
3. `POST /api/v1/admin/policies` + `POST /activate` activates `dns_profile` (`01a18eb6-339e-7a3b-ba55-2f765bcc4ed4`), `fair_use` (`01a18eb6-3807-7351-848f-452c32f360a3`), and `plan` (`01a18eb6-4060-7b2d-ba8b-0d73f503e881`) with `exit_group_ids=[01a18e52-a4e3-7981-983d-bcb37866f801]`.
4. `stack.Enroll` & `stack.EnrollAt(..., "node2")`, `PATCH /api/v1/admin/nodes/{id}` (finite capacity), and `PUT /api/v1/admin/nodes/{id}/publication` (`group_ids=[01a18e52-a4e3-7981-983d-bcb37866f801]`) publish both exits.
5. `POST /api/v1/admin/members/{id}/grants` + `stack.JoinNativeMember` + `netbird networks select <exit_net_id>` establishes counted member forwarding (`serving 1`, `fvpn_generation > 0`, positive `u_<dev>`/`d_<dev>` counters).

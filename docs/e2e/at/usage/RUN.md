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

## 3. Product-Only Bootstrap & Follow-up 3 Verification Path (Run `eb79863a`, `2026-10-31T02:28:57Z`)

1. `e2e.PrepareServingBaseline(ctx, admin.Client, admin.CSRF, false)` builds, signs, uploads, registers, and verifies `0.1.0` (`amd64` + `arm64`) and activates `release_support` (`01a18f2d-cfa5-7793-9c2f-deb612a82a89`).
2. `POST /api/v1/admin/groups` creates member exit group `01a18f2d-d1ff-71f9-8541-d8e4f3c71807`.
3. `POST /api/v1/admin/policies` + `POST /activate` activates `dns_profile` (`01a18fc3-7d70-719f-af74-2ae6bd8d7d38`), `fair_use` (`01a18fc5-8cb0-7a68-baff-83e78a2e56f0`), and `plan` (`01a18fc3-8a7e-75df-b322-bde7ce4a0557`) with `exit_group_ids=[01a18f2d-d1ff-71f9-8541-d8e4f3c71807]`.
4. `POST /api/v1/admin/members/{id}/grants` + `stack.JoinNativeMember` admits `memberA` and `memberB`.
5. `stack.Enroll` & `stack.EnrollAt(..., "node2")`, `PATCH /api/v1/admin/nodes/{id}` (finite capacity), and `PUT /api/v1/admin/nodes/{id}/publication` (`group_ids=[01a18f2d-d1ff-71f9-8541-d8e4f3c71807]`) publish both carrying exits while measuring D-067 Cycle 1 (`bring_up`).
6. `AT14–AT19` measure D-067 Cycle 2 (`agent_restart`), Fair-Use recompile & `fair_use.applying` clearing across both `policy.activated` and `fup.member_generation_changed`, and `ENR-03` per-incarnation retained usage across node re-enrollment (`GET /api/v1/admin/nodes/{id}/incarnations`).

# Upstream Blockers & Requests to Orchestrator (`w4-at-usage` — Follow-up 3)

## 1. Active Upstream Blocker (Regression on `main` since PR #220 `2473acf`)

### Blocker 1 (`w1-netlab` / `w3-devstack-e2e`): `VerifyServing` Rejects Docker Engine's `table ip nat` `POSTROUTING` Rule with `FLOWTABLE_BYPASS`

- **Affected Cases**: `AT14`, `AT16`, `AT17`, `AT18`, `AT19` (`AT15` passes `1/1`; all Follow-up 3 central/protocol mechanisms — PR #222 Fair-Use recompile & `fair_use.applying` convergence, PR #206/#213/#216 D-067 provisional-lease convergence, and PR #210 `ENR-03` per-incarnation retained usage — are verified live on `fvpn-w4-usage`).
- **Exact Code Locations**:
  - `node/enforce/apply.go:102` (`if d.Serve { if err := e.plane.VerifyServing(ctx); err != nil { ... } }`)
  - `node/enforce/verify_foreign.go:107` (`checkForeignEntry`: any `xt` statement inside any hook chain in any table fails with `CodeFlowtableBypass`: `"table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt"`)
  - `node/enforce/verify_foreign.go:177` (`walkTargetChains`: follows `jump DOCKER_POSTROUTING`, which contains `xt` target `"SNAT"` for `127.0.0.11` Docker embedded DNS)
- **Live Observation Evidence (Run `eb79863a`, Commit `ebfdb644d877961dbf5f8e1e5d6872ede7f27a34`)**:
  - When `fvpn-agent` on `node` (`01a18f2d-e7d6-770e-9b8c-edb2016a7db9`) or `node2` (`01a18fc6-fb50-793b-b824-718838f20496`) applies a generation under the D-067 provisional lease (`members: []`), `d.Serve == false` so `VerifyServing` is not called; the node applies cleanly and reports `provisioning_state = "ready"`, `safety_status = "passed"`, `forwarding_block_reasons = ["awaiting_member_grants"]` (observations `01a18fc4-bea8-78fa-9e94-b33b845f3eaa` on `bring_up` and `01a18fc5-99dc-72da-bd1f-8fe1d954859e` on `agent_restart`).
  - Within `5019 ms` (`bring_up`) / `5024 ms` (`agent_restart`), central `RenewTx` issues the populated serving lease (`members: 2`, `86400s` TTL). On the next apply, `d.Serve == true` so `e.plane.Apply` invokes `VerifyServing`. Because `node-netns` and `node2-netns` run on a Docker bridge network, the kernel namespace contains Docker Engine's `table ip nat` (`chain OUTPUT` and `chain POSTROUTING` jumping to `DOCKER_OUTPUT` / `DOCKER_POSTROUTING` with `xt target "DNAT"` / `"SNAT"` for `127.0.0.11`).
  - `VerifyServing` fails with `FLOWTABLE_BYPASS: table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt`, rolls back `table inet fvpn` to baseline (`fvpn_generation = 0`), and emits `safety_status = "failed"` (observations `00000000-0000-0000-0000-000000000000` and `00000000-0000-0000-0000-000000000000`).
- **Required Fix (`w1-netlab` or `w3-devstack-e2e`)**:
  - In `node/enforce/verify_foreign.go`, exempt Docker's loopback embedded-DNS chains (`DOCKER_OUTPUT` / `DOCKER_POSTROUTING` matching `127.0.0.11` in `table ip nat`), or in `deploy/devstack/entrypoint-node-netns.sh` remove the unused `DOCKER_OUTPUT`/`DOCKER_POSTROUTING` jumps from `table ip nat` inside the lab container namespace before starting `fvpn-agent` (`AGENTS.md` §4 forbids `w4-at-usage` from touching tables outside `inet fvpn` or editing `node/enforce` / `deploy/devstack`).

## 2. Resolved Since Follow-up 2 (#214)

1. **Resolved by PR #222 (`w3-policy-fup`) — Desired-State Recompile on `policy.activated` and `fup.member_generation_changed`**:
   - Verified live in `AT19`: activating `fair_use` in `mode=enforce` incremented `desired_config_generation` on both carrying exits (`node` `117 -> 119`, `node2` `4 -> 7`) and left the non-carrying node (`01a18fc7-d74c-70f9-9fa0-d48d9f112449`) at `0 -> 0`; usage ingestion crossing the Tier 2 threshold (`fup.member_generation_changed`) incremented `desired_config_generation` on both carrying exits (`node` `119 -> 120`, `node2` `7 -> 8`) and left the non-carrying node at `0 -> 0`. Replaying either event did not bump generations twice.
2. **Resolved by PR #222 (`w3-policy-fup`) — `pgNodeAckReader.GetMemberNodeAckStatus` Clearing `fair_use.applying`**:
   - Verified live in `AT19`: `GET /api/v1/member/usage` `fair_use.applying` stayed `true` while only `node` had acknowledged the new generation, and transitioned `true -> false` once `node2` also acknowledged (`5384 ms` on `policy.activated` via observation `139ab64b-59a5-4b60-84d7-73c8eddfbe9b`; `6141 ms` on `fup.member_generation_changed` via observation `c14761e0-9137-49ef-9b15-f4d920babbe7`).
3. **Verified on `main` (PR #206, #213, #216) — D-067 Provisional-Lease Convergence (`AWAITING_MEMBER_GRANTS`)**:
   - Measured on two real cycles (`bring_up`: `5019 ms` ready->populated-lease; `agent_restart`: `5024 ms` ready->populated-lease), both within `<= 65000 ms` (`60 s + 1 heartbeat`), reporting `AWAITING_MEMBER_GRANTS` on `lease`/`qos`/`source_binding` and 1 `errors[]` entry at `stage=lease` (never `LEASE_EXPIRED`).
4. **Verified on `main` (PR #210) — `ENR-03` Per-Incarnation Retained Usage (`GET /api/v1/admin/nodes/{id}/incarnations`)**:
   - Verified live on `node2` (`01a18fc6-fb50-793b-b824-718838f20496`): re-enrolling `node2` after counted traffic retired incarnation `01a18fc6-fc7d-73f6-bf0c-6d8abb6fd46b` (`retire_reason=reinstall`, retained `upload_bytes=1500`, `download_bytes=2000`) and activated incarnation `01a18fc8-90a6-781b-ab46-c0a6531d96ed` newest-first without altering `member_usage_periods` (`total_bytes=97500`), and subsequent traffic on `01a18fc8-90a6-781b-ab46-c0a6531d96ed` accumulated on the same member period (`total_bytes=99500`) while `01a18fc6-fc7d-73f6-bf0c-6d8abb6fd46b` stayed fixed.

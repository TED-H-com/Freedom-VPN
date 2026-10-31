# Upstream Blockers & Requests to Orchestrator (`w4-at-usage` — Follow-up 3)

## 1. Follow-up 3 Upstream Blocker (Regression Since PR #220 `2473acf`; Addressed on `main` After This Run)

### Blocker 1 (`w1-netlab` / `w3-devstack-e2e`): `VerifyServing` Rejects Docker Engine's `table ip nat` `POSTROUTING` Rule with `FLOWTABLE_BYPASS`

- **Current Status**: Addressed on `main` by PR #240 (`e9d6779`, devstack follow-up 14) and PR #241 (`a4191fa`, netlab follow-up 11, D-079), after the node image used for Follow-up 3. The observations below remain the evidence for this run's `blocked` results; the re-run on the fixed lab is usage Follow-up 4.
- **Affected Cases**: `AT14`, `AT15`, `AT16`, `AT17`, `AT18`, `AT19` (all 6 live cases require a counted serving flow on the exit data plane; all Follow-up 3 central/protocol mechanisms — `AT15` mTLS batch ingest/recovery, `AT17` UTC month boundary/Portal totals, PR #222 Fair-Use recompile & `fair_use.applying` convergence, PR #206/#213/#216 D-067 provisional-lease convergence, and PR #210 `ENR-03` per-incarnation retained usage — are verified live on `fvpn-w4-usage`).
- **Exact Code Locations in This Run**:
  - `node/enforce/apply.go:102` (`if d.Serve { if err := e.plane.VerifyServing(ctx); err != nil { ... } }`)
  - `node/enforce/verify_foreign.go:107` (`checkForeignEntry`: any `xt` statement inside any hook chain in any table fails with `CodeFlowtableBypass`: `"table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt"`)
  - `node/enforce/verify_foreign.go:177` (`walkTargetChains`: follows `jump DOCKER_POSTROUTING`, which contains `xt` target `"SNAT"` for `127.0.0.11` Docker embedded DNS)
- **Live Observation Evidence (Run `e1f8c483`, Commit `46081fb0e4011b91e9f9f93be23e04e0f223d42d`)**:
  - When `fvpn-agent` on `node` (`01a18f2d-e7d6-770e-9b8c-edb2016a7db9`) or `node2` (`01a18fd0-4188-77d3-9e94-5c5bdc8f8d29`) applies a generation under the D-067 provisional lease (`members: []`), `d.Serve == false` so `VerifyServing` is not called; the node applies cleanly and reports `provisioning_state = "ready"`, `safety_status = "passed"`, `forwarding_block_reasons = ["awaiting_member_grants"]` (observations `01a18fcd-a1a4-7983-9fe3-aa16f11d129b` on `bring_up` and `01a18fce-aba1-7b56-bd5e-6dbdccd14664` on `agent_restart`).
  - Within `5021 ms` (`bring_up`) / `5025 ms` (`agent_restart`), central `RenewTx` issues the populated serving lease (`members: 2`, `86400s` TTL). On the next apply, `d.Serve == true` so `e.plane.Apply` invokes `VerifyServing`. Because `node-netns` and `node2-netns` run on a Docker bridge network, the kernel namespace contains Docker Engine's `table ip nat` (`chain OUTPUT` and `chain POSTROUTING` jumping to `DOCKER_OUTPUT` / `DOCKER_POSTROUTING` with `xt target "DNAT"` / `"SNAT"` for `127.0.0.11`).
  - `VerifyServing` fails with `FLOWTABLE_BYPASS: table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt`, rolls back `table inet fvpn` to baseline (`fvpn_generation = 0`), and emits `safety_status = "failed"` (observations `01a18fce-194d-7c40-b1eb-246cda6a5222` and `01a18fcf-23b7-7718-9eb5-7655c3aae811`).
- **Fixes on `main` (`w3-devstack-e2e` / `w1-netlab`)**:
  - PR #240 (`e9d6779`) makes the lab node delete Docker's embedded-DNS `table ip nat` rules in `deploy/devstack/entrypoint-node.sh` before `fvpn-agent` starts.
  - PR #241 (`a4191fa`, D-079) classifies foreign rules scoped to loopback (`127.0.0.0/8`, `::1`) as `loopback_scoped` instead of `FLOWTABLE_BYPASS`.

## 2. Resolved Since Follow-up 2 (#214)

1. **Resolved by PR #222 (`w3-policy-fup`) — Desired-State Recompile on `policy.activated` and `fup.member_generation_changed`**:
   - Verified live in `AT19`: activating `fair_use` in `mode=enforce` incremented `desired_config_generation` on both carrying exits (`node` `126 -> 127`, `node2` `4 -> 7`) and left the non-carrying node (`01a18fd1-3793-7120-9318-d306221ee728`) at `0 -> 0`; usage ingestion crossing the Tier 2 threshold (`fup.member_generation_changed`) incremented `desired_config_generation` on both carrying exits (`node` `128 -> 129`, `node2` `8 -> 9`) and left the non-carrying node at `0 -> 0`. Replaying either event did not bump generations twice.
2. **Resolved by PR #222 (`w3-policy-fup`) — `pgNodeAckReader.GetMemberNodeAckStatus` Clearing `fair_use.applying`**:
   - Verified live in `AT19`: `GET /api/v1/member/usage` `fair_use.applying` stayed `true` while only `node` had acknowledged the new generation, and transitioned `true -> false` once `node2` also acknowledged (`5387 ms` on `policy.activated` via observation `5a9a70e5-b7d2-4c4d-9175-fc7acba16c2b`; `6151 ms` on `fup.member_generation_changed` via observation `922edc3f-c2b6-4f4d-a4aa-3bea02b6fa8d`).
3. **Verified on `main` (PR #206, #213, #216) — D-067 Provisional-Lease Convergence (`AWAITING_MEMBER_GRANTS`)**:
   - Measured on two real cycles (`bring_up`: `5021 ms` ready->populated-lease; `agent_restart`: `5025 ms` ready->populated-lease), both within `<= 65000 ms` (`60 s + 1 heartbeat`), reporting `AWAITING_MEMBER_GRANTS` on `lease`/`qos`/`source_binding` and 1 `errors[]` entry at `stage=lease` (never `LEASE_EXPIRED`).
   - Devstack follow-up 14 (PR #240, `e9d6779`) leaves the empty-`members` lease behaviour unchanged and does not work around it in the lab. `member-traffic` records `ready_to_populated_lease_seconds` for comparison with a usage-harness run.
4. **Verified on `main` (PR #210) — `ENR-03` Per-Incarnation Retained Usage (`GET /api/v1/admin/nodes/{id}/incarnations`)**:
   - Verified live on `node2` (`01a18fd0-4188-77d3-9e94-5c5bdc8f8d29`): re-enrolling `node2` after counted traffic retired incarnation `01a18fd0-42b9-7e45-97db-c16ad22f59d0` (`retire_reason=reinstall`, retained `upload_bytes=1500`, `download_bytes=2000`) and activated incarnation `01a18fd1-f3d3-7868-9373-487f724b9df7` newest-first without altering `usage_rollups` / `GET /api/v1/member/usage` (`total_bytes=109929`), and subsequent traffic on `01a18fd1-f3d3-7868-9373-487f724b9df7` accumulated on the same member period (`total_bytes=111929`) while `01a18fd0-42b9-7e45-97db-c16ad22f59d0` stayed fixed.
5. **Lab Supervision Updated by PR #240 (`e9d6779`, devstack follow-up 14, 2026-10-30)**:
   - `pause-agent` is the only agent stop path, addressing the agent stop/start race noted in Follow-up 2. The `worker`, `control`, and `netbird` services use `unless-stopped`.

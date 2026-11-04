# Follow-up 37 — D-141 Clone Control-Path Diagnosis, Shared Fixture Fix, and Consumer Audit

Date: `2026-11-04`

## 1. The Diagnosis (D-141)

### 1.1 The Clone's Control Path When the Source Carries Isolation's Observer

When the source container (`fvpn-w4-isolation-node`, `s.lab.node`) carries isolation's pre-enrollment observer (`installObserver`, `tests/e2e/at/isolation/driver.go.txt:129,2283-2301`), the clone container (`fvpn-w4-isolation-node-clone`) executes the following exact sequence:

1. **Initial devstack preparation (`tests/e2e/cmd/fvpn-e2e-prepare/main.go:386-411`):**
   - `writeAgentConfigs` writes `node/etc/agent-enroll.json` with `"control_url": "https://portal.example.invalid:8443"` (`tests/e2e/cmd/fvpn-e2e-prepare/main.go:401-402`) and `node/etc/agent-run.json` with `"control_url": "https://127.0.0.1:9444"` (`tests/e2e/cmd/fvpn-e2e-prepare/main.go:403-404`).

2. **Pre-enrollment observer installation on the source node (`tests/e2e/at/isolation/driver.go.txt:129,2283-2301`, `tests/e2e/at/isolation/observer_main.go:114-136`):**
   - Before enrollment and before `PrepareServingBaseline`, `installObserver` (`tests/e2e/at/isolation/driver.go.txt:2283-2301`) runs inside `x.r.lab.node`:
     - copies `/tmp/isolation-probe` to `/usr/local/bin/isolation-observer` in `x.r.lab.node`'s writable container rootfs (`tests/e2e/at/isolation/driver.go.txt:2284`);
     - renames `/usr/local/bin/fvpn-e2e-nodeconfig` to `/usr/local/bin/fvpn-e2e-nodeconfig-original` and installs a wrapper script at `/usr/local/bin/fvpn-e2e-nodeconfig` in `x.r.lab.node`'s writable container rootfs that intercepts `resolver-proxy` to start `fvpn-e2e-nodeconfig-original resolver-proxy &` (listening on `127.0.0.1:9444`) and `exec /usr/local/bin/isolation-observer observe-proxy` (listening on `127.0.0.1:9445` and forwarding upstream to `https://127.0.0.1:9444`, `tests/e2e/at/isolation/driver.go.txt:2285-2294`, `tests/e2e/at/isolation/observer_main.go:45-96`);
     - invokes `/usr/local/bin/isolation-observer configure-observation` (`tests/e2e/at/isolation/driver.go.txt:2295`), which executes `configureObservation()` (`tests/e2e/at/isolation/observer_main.go:114-136`) and rewrites `/etc/vpn-node/agent-run.json` (`state/node/etc/agent-run.json`) so that `"control_url"` is `"https://127.0.0.1:9445"` (`tests/e2e/at/isolation/observer_main.go:125`), `"heartbeat_seconds"`, `"poll_seconds"`, and `"observation_seconds"` are `1` (`tests/e2e/at/isolation/observer_main.go:120-122`), and `"test_timing_path"` is `"/var/lib/vpn-node/e2e/isolation-operations.jsonl"` (`tests/e2e/at/isolation/observer_main.go:126`).
   - Next, `x.prepare()` (`tests/e2e/at/isolation/driver.go.txt:133,2513`) calls `PrepareServingBaseline` (`tests/e2e/serving.go:48-127`), which calls `l.candidateConfig(id, spec.version)` (`tests/e2e/serving.go:111`, `tests/e2e/harness.go:551-566`) and `l.setBootRelease(id)` (`tests/e2e/serving.go:118`, `tests/e2e/harness.go:392-413`). Because `candidateConfig` reads `state/node/etc/agent-run.json` after `configureObservation()` has already modified it, `state/node/var/e2e/agent-<boot_release_id>.json` also inherits `"control_url": "https://127.0.0.1:9445"`.

3. **Clone creation (`tests/e2e/identity_exports.go:45-64`, `tests/e2e/second_exit.go:21-170`):**
   - `freezeIdentity` calls `x.stack.StartClonedIdentityFixture(x.ctx, "node")` (`tests/e2e/at/isolation/driver.go.txt:2664`), which calls `s.startSecondExit(ctx, true)` (`tests/e2e/identity_exports.go:60`) -> `s.startExitSlot(ctx, "node-clone", ".23", true, false)` (`tests/e2e/second_exit.go:21-28`).
   - `startExitSlot` stops `s.lab.node` (`tests/e2e/second_exit.go:49-52`), copies `s.lab.node:/etc/vpn-node/.` to `base/etc` (`tests/e2e/second_exit.go:59-61`), removes `boot-id` and `updater-recover.json` while keeping `machine-id` (`tests/e2e/second_exit.go:65-70`), and loops over `agent-enroll.json` and `agent-run.json` (`tests/e2e/second_exit.go:72-94`).
   - Before the fix in this follow-up, lines `72-94` of `tests/e2e/second_exit.go` updated **only** `datapath["snat"] = prefix + lastOctet` (`tests/e2e/second_exit.go:82-86`), leaving `control_url` untouched at `"https://127.0.0.1:9445"` in `base/etc/agent-run.json`.
   - Lines `98-100,108,167` of `tests/e2e/second_exit.go` inspect `s.lab.node`'s base image (`docker inspect --format={{.Image}}`, built from `deploy/devstack/Dockerfile.node`) and start `fvpn-w4-isolation-node-clone-netns` and `fvpn-w4-isolation-node-clone` from that unmutated base image. The writable container layer of `s.lab.node` (where `installObserver` placed `/usr/local/bin/isolation-observer` and the `/usr/local/bin/fvpn-e2e-nodeconfig` shell wrapper) is not part of the base image and is not copied into `node-clone`.
   - Lines `121-142` of `tests/e2e/second_exit.go` copy `/var/lib/vpn-node/.` to `base/var` (including `identity/{node.json,cert.pem,key.pem,setup.key}`, `state/agent.db`, `e2e/agent-selected`, and `e2e/agent-<boot_release_id>.json`), `/var/lib/netbird/.` to `base/netbird`, and `/opt/vpn-node/.` to `base/opt`, then remove `updater.json`, `pause-agent`, and `agent.pid` from `base/var/e2e`.
   - Lines `154-160` of `tests/e2e/second_exit.go` attach the clone presentation network (`fvpn-w4-isolation-node-clone-net`) and call `rewriteLabHost` (`tests/e2e/clone_network.go:81-108`) to pin `agent.example.invalid` in `base/etc/lab-hosts` to `pres.Control`.

4. **Clone entrypoint steps and helper calls (`deploy/devstack/entrypoint-node.sh`, `tests/e2e/cmd/fvpn-e2e-nodeconfig/`, `deploy/devstack/agent-service.sh`):**
   - `deploy/devstack/entrypoint-node.sh:34-44` generates `/etc/vpn-node/boot-id` and preserves the cloned `/etc/vpn-node/machine-id`.
   - `deploy/devstack/entrypoint-node.sh:47-55` appends `/etc/vpn-node/lab-hosts` to `/etc/hosts` (`agent.example.invalid` -> `pres.Control`).
   - `deploy/devstack/entrypoint-node.sh:64-73` starts `netbird service` and brings the cloned NetBird peer up.
   - `deploy/devstack/entrypoint-node.sh:77-83` sees non-empty `/var/lib/vpn-node/identity/{node.json,cert.pem,key.pem}`, records `{"stage":"resuming"}` in `/var/lib/vpn-node/e2e/status.json`, and skips `fvpn-agent enroll` (`deploy/devstack/entrypoint-node.sh:83-100`).
   - `deploy/devstack/entrypoint-node.sh:105` runs `/usr/local/bin/fvpn-e2e-nodeconfig` with no arguments (`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:21-22,89-132`), which reads `/var/lib/vpn-node/identity/node.json` and writes **only** `/etc/vpn-node/updater-recover.json` (with `AgentConfig: "/etc/vpn-node/agent-run.json"` at `tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:114`). It does **not** read or rewrite `control_url` in any agent config.
   - `deploy/devstack/entrypoint-node.sh:107` runs `/usr/local/bin/fvpn-e2e-nodeconfig dns-relay &` (`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:47-49,136-159`), which reads only `datapath.snat` from `/etc/vpn-node/agent-run.json`. It does **not** rewrite `control_url`.
   - `deploy/devstack/entrypoint-node.sh:108` runs `/usr/local/bin/fvpn-e2e-nodeconfig resolver-proxy &`. Because the clone runs the unmutated node image (`deploy/devstack/Dockerfile.node:7`), this invokes the real `resolverProxy()` (`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:50-51`, `tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:96-142`), which listens on **`127.0.0.1:9444`** (`tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:140`) and proxies mTLS requests upstream to `https://agent.example.invalid:9443` (`tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:120`).
   - `deploy/devstack/entrypoint-node.sh:115-130` starts `fvpn-watchdog` and enters the supervisor loop, which reads `/var/lib/vpn-node/e2e/agent-selected` (`/usr/local`) and calls `/usr/local/bin/agent-service.sh --supervisor start /usr/local` (`deploy/devstack/entrypoint-node.sh:122-125`).
   - `deploy/devstack/agent-service.sh:117-123,129-153` resolves `service_cfg="/etc/vpn-node/agent-run.json"` (or `/var/lib/vpn-node/e2e/agent-<release_id>.json` if `agent-selected` names a release) and runs `/usr/local/bin/fvpn-e2e-nodeconfig agent-wait /usr/local/bin/fvpn-agent /etc/vpn-node/agent-run.json` (`deploy/devstack/agent-service.sh:152`, `tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:23-25`).
   - `fvpn-agent` loads `/etc/vpn-node/agent-run.json` (`node/agent/cmd/fvpn-agent/main.go:140`, `node/agent/config.go:34,87-103`) and constructs its control client (`node/agent/engine.go:300-304`) with `ControlURL = "https://127.0.0.1:9445"`.

- **Answers to 1.1:**
  - **URL the clone's agent dials after every entrypoint step:** `https://127.0.0.1:9445`. Neither `deploy/devstack/entrypoint-node.sh` nor any subcommand of `tests/e2e/cmd/fvpn-e2e-nodeconfig` rewrites `control_url`.
  - **What listens on that port (`127.0.0.1:9445`) inside the clone:** **Nothing.** Inside the clone, `fvpn-e2e-nodeconfig resolver-proxy` listens on `127.0.0.1:9444` (`tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:140`), and `/usr/local/bin/isolation-observer` is not installed or running in the clone container.
  - **Whether the clone's agent can reach control at all:** **No.** Every control-plane request from the clone's `fvpn-agent` (`GET /api/v1/agent/desired-state`, `POST /api/v1/agent/heartbeat`, `POST /api/v1/agent/observations`, `POST /api/v1/agent/usage-batches`) fails immediately at TCP dial to `127.0.0.1:9445` (`connection refused`).

---

### 1.2 Historical Isolation Records (`docs/e2e/at/isolation/**/identity-freeze.json*`)

Below is every archived isolation clone window under `docs/e2e/at/isolation/history/follow-up-*/identity-freeze.json*` plus the current root record `docs/e2e/at/isolation/identity-freeze.json.gz` (30 files total across follow-ups 7 through 25):

| Record path | Run ID | Source commit | Status | `clone_observation` row count | Max `endpoint_rows` | Max `conflicts` | `mapping_frozen` ever `true`? | Clone `agent.log` / `watchdog.log` retained? |
|---|---|---|---|---:|---:|---:|---|---|
| `docs/e2e/at/isolation/history/follow-up-7/identity-freeze.json` | `041e9e48-fa38-47bf-b7cb-beac596b96fc` | `20cd2aa00668ffbdbf114f141ed5b45017da4d2a` | `requires receiver correlation` | 4 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-8/identity-freeze.json` | `dc56795f-17a5-4d30-9f5c-0cb573957f51` | `eb33d29de1d73204aa39a2b4dd5c768e139a6591` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-9/identity-freeze.json` | `3da720fb-ce84-49d0-942c-7e12979d1bc2` | `1d546635ac71ed8273704fb23f5e0c0d82e52b63` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-10-attempt-1/identity-freeze.json` | `9ef7e3b6-c862-4bf5-bb21-25b0fd5a319c` | `c85216f45e78c8b6bd65c8791ca6c58fb5f18dc4` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-10-attempt-2/identity-freeze.json` | `848f2fd2-bb87-4361-a6e7-b08c6f89f1b9` | `b73501c5cf661a243d7b3859d292d4e4f8465426` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-10/identity-freeze.json` | `a04bd600-f82f-4e64-9a04-b7843ef07784` | `b73501c5cf661a243d7b3859d292d4e4f8465426` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-11/identity-freeze.json` | `781f1a09-71c8-4510-bb80-0d106d440610` | `e3fb1860260d157d7d0d6b34d556c0d399c72ec8` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-12/identity-freeze.json` | `20cd64eb-4cd4-4d6b-afac-c7be5e77f716` | `8d9dcdf91a5d293e05e2e0bd3c1fa5539e6816fb` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-13/identity-freeze.json` | `a9496fd2-3bc3-4ab2-a18f-ab40b5ba7282` | `26c9a5e6d124653f438983ba72ccbfbe054e419f` | `requires receiver correlation` | 4 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-14/identity-freeze.json.gz` | `c8523d30-da7b-486d-bc37-dc68c08df77d` | `254791cb95e4959326efae31cf458edae1c69c25` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-15/identity-freeze.json.gz` | `0b9e4510-79a2-40b2-a151-2ba53c5750fb` | `30a79540e3e133ca89855a2a31b4a69edcb6cb12` | `requires receiver correlation` | 3 | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-16/identity-freeze.json.gz` | `83b457ef-ea16-490b-9923-1fea0df86b36` | `9ef8635a0b318619b28dfda9be5fb717d1d669fa` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-17/identity-freeze.json.gz` | `7b6bc7c8-f5fd-40e5-82bb-3eb57be8de51` | `1511c7e47cf512f8791c229bc9f3b888aaec821a` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-18/identity-freeze.json.gz` | `9f1bc0ff-fffa-46f7-ba13-8c1a158c3db5` | `820de790aa5f7d1da1440d04071cb25b19b60f13` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-19/identity-freeze.json.gz` | `9f1bc0ff-fffa-46f7-ba13-8c1a158c3db5` (retained from follow-up 18; follow-up 19 run `70beb28e` aborted at setup before `freezeIdentity`) | `820de790aa5f7d1da1440d04071cb25b19b60f13` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-20/identity-freeze.json.gz` | `abbf24eb-5a19-4bd7-8483-5aa3002850b0` | `8a8bc3c389a67827d4847a372da8fc312a98ca74` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-20-run/identity-freeze.json.gz` | `abbf24eb-5a19-4bd7-8483-5aa3002850b0` | `8a8bc3c389a67827d4847a372da8fc312a98ca74` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-21/identity-freeze.json.gz` | `d183b410-1695-4fb9-b8f7-e66838d16667` | `c85937dd42661bf596a35f8db44820477c50dbbb` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-21-run/identity-freeze.json.gz` | `d183b410-1695-4fb9-b8f7-e66838d16667` | `c85937dd42661bf596a35f8db44820477c50dbbb` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-22/identity-freeze.json.gz` | `7b0db5f2-e33e-4633-8ff8-cf433cf23db6` | `da437c6676810d5edff76306e9f2f98f464180be` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-22-run/identity-freeze.json.gz` | `7b0db5f2-e33e-4633-8ff8-cf433cf23db6` | `da437c6676810d5edff76306e9f2f98f464180be` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-23/identity-freeze.json.gz` | `4c9cfee4-fb5a-4cf8-af3c-ef2aa0dbafcf` | `ec997282b43723b26ebcdac7fa176cc052882f73` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-23-run/identity-freeze.json.gz` | `4c9cfee4-fb5a-4cf8-af3c-ef2aa0dbafcf` | `ec997282b43723b26ebcdac7fa176cc052882f73` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-24/identity-freeze.json.gz` | `df9dbad7-2df8-43be-bbaf-e3bb4a6443d1` | `1ebdc80be0d8a87b33a8ba1c6be4ac5ab3ca96dc` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-24-run/identity-freeze.json.gz` | `df9dbad7-2df8-43be-bbaf-e3bb4a6443d1` | `1ebdc80be0d8a87b33a8ba1c6be4ac5ab3ca96dc` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-25/identity-freeze.json.gz` | `bd73d7a4-57bf-4be4-aaf5-d1d6b99878ad` | `32245df209a1e3418193d55635877b517f01a830` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-25-run/identity-freeze.json.gz` | `bd73d7a4-57bf-4be4-aaf5-d1d6b99878ad` | `32245df209a1e3418193d55635877b517f01a830` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/identity-freeze.json.gz` (current root = follow-up 25) | `bd73d7a4-57bf-4be4-aaf5-d1d6b99878ad` | `32245df209a1e3418193d55635877b517f01a830` | `requires receiver correlation` | 90 | 1 | 0 | `false` | none |
| `docs/e2e/at/isolation/history/follow-up-8/provisional-receiver-analysis.json` (summary artifact) | `041e9e48-fa38-47bf-b7cb-beac596b96fc` (analyzing follow-up 7) | `20cd2aa00668ffbdbf114f141ed5b45017da4d2a` | `provisional` | 4 (`central_clone_reports_in_index: 11`) | 1 | 1 | `true` | none |
| `docs/e2e/at/isolation/history/follow-up-10/provisional-receiver-analysis.json` (summary artifact) | `a04bd600-f82f-4e64-9a04-b7843ef07784` | `b73501c5cf661a243d7b3859d292d4e4f8465426` | `provisional` | 3 | 1 | 1 | `true` | none |

- **Note on `endpoint_rows`:** `node_identity_endpoints` (`db/migrations/00110_pki_enrollment.sql:30`, `internal/enrollment/revoke.go:261-392`) stores one row per `incarnation_id` (recording the mapped `endpoint_hash`, optional `candidate_hash`, and `mapping_frozen`). Thus `SELECT count(*) FROM node_identity_endpoints WHERE incarnation_id=$1` (`CloneObservation.EndpointRows`, `tests/e2e/identity_exports.go:93`) has a maximum of `1` once the incarnation has observed at least one endpoint.
- **Note on clone `agent.log` and `watchdog.log` retention:** No isolation run retains `agent.log` or `watchdog.log` from `fvpn-w4-isolation-node-clone`. Both `retainReasonLogs("before-clone")` (`tests/e2e/at/isolation/driver.go.txt:2306-2331,2663`, which writes `agent-before-clone.log.txt` and `watchdog-before-clone.log.txt`) and `archiveReceiver` (`tests/e2e/at/isolation/driver.go.txt:2378-2388`, which writes `agent.log.txt` and `watchdog.log.txt`) read `/var/lib/vpn-node/e2e/{agent,watchdog}.log` exclusively from `x.r.lab.node` (`fvpn-w4-isolation-node`). The only retained clone runtime artifacts in isolation's evidence are the direct SIGTERM pidfd/parent-wait stop receipt in `receiver-closure.json*` (`stop-receipt-review.json` `direct_agent`) and the container start/stop lifecycle events in `container-lifecycle-events.json`.

---

### 1.3 Our Own Clone Scenario Without Isolation's Observer

- **Scenario names and implementation:** `export-clone` and `clone-conflict` (`tests/e2e/exports_live_test.go:193-274`, invoked by `TestE2E_SharedExports` at `tests/e2e/exports_live_test.go:108`).
  - `exportsClone` (`tests/e2e/exports_live_test.go:193-209`) calls `s.StartClonedIdentityFixture(ctx, "node")` (`tests/e2e/exports_live_test.go:196`), polls `fixture.Observe(ctx)` until `obs.SameIdentity && obs.SourceStage == "enrolled" && obs.CloneStage == "enrolled"` (`tests/e2e/exports_live_test.go:202-204`), writes the `export-clone` record, and calls `r.cloneConflict(ctx, s, b, fixture, node)` (`tests/e2e/exports_live_test.go:208`).
  - `cloneConflict` (`tests/e2e/exports_live_test.go:211-274`) verifies the clone's presentation network (`fvpn-e2e-node-clone-net`, `172.29.30.0/24`), polls `GET /api/v1/admin/nodes/{id}` (`readCloneConflict`, `tests/e2e/exports_live_test.go:285-313`) and `GET /api/v1/admin/audit-events` (`countConflictAudits`, `tests/e2e/exports_live_test.go:315-343`) until `view.Frozen && view.ConflictCount == 1 && view.LastConflict && view.Provisioning == "failed" && view.Forwarding == "false" && audits == 1`, waits 8 seconds to verify `identity_conflict_audits_after_wait == 1`, and then runs `reconcileClone` (`tests/e2e/exports_live_test.go:345-432`) to stop the clone, revoke the frozen node, re-enroll the source node via the entrypoint, and verify that `reconciled_mapping_frozen` is `false` and `reconciled_conflict_count` is `0`.
- **Latest run records and exact values (`run_id: 12d10041-9d33-4fbb-a4b0-9d2a21bce785`, `source_commit: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3`, `2026-10-31T04:28:50Z`–`2026-10-31T04:30:11Z`):**
  1. `docs/e2e/runs/2026-10-31/followup15/exports/export-clone.json` (`status: "pass"`):
     - `observations[0]` (`name: "clone_observation"`), captured immediately when `SourceStage` and `CloneStage` both reported `"enrolled"`:
       - `same_identity`: `true`
       - `endpoint_rows`: `1`
       - `conflicts`: `0` (in the two earlier follow-up 15 export runs where the clone's first request landed before the initial `export-clone` read — `docs/e2e/runs/2026-10-31/followup15/history/exports-first/export-clone.json` (`run_id: 926791b9-36ef-4051-9d44-cd7eafdc8792`) and `docs/e2e/runs/2026-10-31/followup15/history/exports-second/export-clone.json` (`run_id: 630b9f07-62d2-4db1-9737-9060732db1f1`) — `clone_observation` already showed `endpoint_rows: 1`, `conflicts: 1`, `mapping_frozen: true`)
       - `mapping_frozen`: `false`
       - `SourceStage`: `"enrolled"`
       - `CloneStage`: `"enrolled"`
  2. `docs/e2e/runs/2026-10-31/followup15/exports/clone-conflict.json` (`status: "pass"`):
     - `presentation_subnet`: `"172.29.30.0/24"`
     - `presentation_network`: `"fvpn-e2e-node-clone-net"`
     - `mapping_frozen`: `"true"`
     - `conflict_count`: `"1"`
     - `last_conflict_at_present`: `"true"`
     - `provisioning_state`: `"failed"`
     - `member_forwarding_observed`: `"false"`
     - `identity_conflict_audits`: `"1"`
     - `identity_conflict_audits_after_wait`: `"1"`
     - `reconciled_mapping_frozen`: `"false"`
     - `reconciled_conflict_count`: `"0"`
     - `reconciled_last_conflict_null`: `"true"`

---

### 1.4 Consumer Audit of `StartClonedIdentityFixture` and `startSecondExit` Across `tests/e2e/**`

`StartClonedIdentityFixture` (`tests/e2e/identity_exports.go:45-64`) is the sole caller of `s.startSecondExit(ctx, true)` (`tests/e2e/identity_exports.go:60`), and `StartSecondExit` (`tests/e2e/second_exit.go:20`) is the sole caller of `s.startSecondExit(ctx, false)`. Both delegate to `s.startExitSlot(ctx, slot, octet, clone, false)` (`tests/e2e/second_exit.go:26,28-170`), which is also called directly by `RetainEnrollingThirdNode` (`tests/e2e/fleet_snapshot.go:44`) for `node3`.

| Consumer (`file:line`) | Hook called | Can source carry a modified `control_url`? | What the consumer's records claim about the clone / second slot |
|---|---|---|---|
| `tests/e2e/identity_exports.go:45,60` (`StartClonedIdentityFixture`) | `s.startSecondExit(ctx, true)` | **Yes**, when called by a consumer that rewrote `/etc/vpn-node/agent-run.json` on `s.lab.node` before cloning (specifically `tests/e2e/at/isolation/driver.go.txt:2664`). | Shared fixture implementation; returns `*ClonedIdentityFixture` whose `Observe(ctx)` queries `node_identity_endpoints` (`endpoint_rows`, `mapping_frozen`) and `node_identity_conflicts` (`conflicts`). |
| `tests/e2e/second_exit.go:20,21,26,28` (`StartSecondExit`, `startSecondExit`, `startExitSlot`) | `s.startSecondExit(ctx, false)` / `s.startExitSlot(...)` | **Yes** in principle if a caller modified `s.lab.node`'s `/etc/vpn-node/agent-run.json` before starting `node2` or `node-clone` (currently only isolation modifies `s.lab.node`'s `agent-run.json`, and isolation calls `StartClonedIdentityFixture`). | Shared fixture implementation; prepares `<state>/<slot>/etc/agent-{enroll,run}.json` (and copies `<state>/<slot>/var` when `clone == true`) and starts `<prefix>-<slot>{-netns,-adguard,}`. |
| `tests/e2e/exports_live_test.go:196` (`exportsClone` -> `cloneConflict` at `:208`) | `s.StartClonedIdentityFixture(ctx, "node")` | **No.** Source `node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/runs/2026-10-31/followup15/exports/export-clone.json` and `clone-conflict.json` (`run_id: 12d10041-9d33-4fbb-a4b0-9d2a21bce785`): `same_identity: true`, `endpoint_rows: 1`, `conflict_count: "1"`, `mapping_frozen: "true"`, `provisioning_state: "failed"`, `member_forwarding_observed: "false"`, `identity_conflict_audits: "1"`, `identity_conflict_audits_after_wait: "1"`, and recovery after revoke + entrypoint re-enrollment. |
| `tests/e2e/at/enrollment/live_test.go:1427` (`clonedIdentity` in `AT07`) | `p.stack.StartClonedIdentityFixture(p.ctx, "node")` | **No.** Source `fvpn-w4-enroll-node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/at/enrollment/AT07.json` (`run_id: 891ee9b8-bf01-4583-a47a-9542c8b04c8a`, checks `cloned_identity_conflict: pass`, `serving_freeze_withdrawal: pass`, `serving_freeze_restart: pass`): `same_identity: true`, `endpoint_rows: 1`, `conflicts: 1`, `mapping_frozen: true`, `identity_conflict_audits: 1`, `identity_conflict_audits_after_wait: 1`, `conflict_count_stable: true`, `withdrawal_upper_bound_ms: 187`, and `recovery.clone_denied: true` (`403 IDENTITY_REVOKED`). |
| `tests/e2e/at/isolation/driver.go.txt:2664` (`freezeIdentity`) | `x.stack.StartClonedIdentityFixture(x.ctx, "node")` | **Yes (since isolation follow-up 16, `8afa69c5` / #399).** `installObserver` (`tests/e2e/at/isolation/driver.go.txt:129,2283-2301`) runs `isolation-observer configure-observation` (`tests/e2e/at/isolation/observer_main.go:114-136`), setting `control_url = "https://127.0.0.1:9445"` in `/etc/vpn-node/agent-run.json` (and by inheritance via `candidateConfig` at `tests/e2e/serving.go:111`, in `/var/lib/vpn-node/e2e/agent-<boot_release_id>.json`). | `docs/e2e/at/isolation/identity-freeze.json.gz` (`status: "requires receiver correlation"`, `owner: "ENR-02 / ENR-03; this is the consumer latch observation only"`): records `clone_observation` samples (`endpoint_rows`, `conflicts`, `mapping_frozen`). In follow-ups 7–15 (`control_url` was `https://127.0.0.1:9444`), recorded 3–4 rows with `conflicts: 1, mapping_frozen: true` (`last_row_label: "freeze latch held"`). In follow-ups 16–25 (`control_url` became `https://127.0.0.1:9445`), recorded 90 rows with `conflicts: 0, mapping_frozen: false`. Also records the clone agent's SIGTERM stop receipt in `receiver-closure.json.gz` (`stop-receipt-review.json` `direct_agent`) and source container snapshot stop/start (`stop-receipt-review.json` `clone_source`). |
| `tests/e2e/second_exit_test.go:16` (`secondExit` in `TestE2E_Acceptance`) | `s.StartSecondExit(ctx)` | **No.** Source `node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/runs/2026-11-04/followup36/second-exit.json`: enrolls independent `node2` (`clone = false`), not a clone. |
| `tests/e2e/exports_live_test.go:102` (`TestE2E_SharedExports`) | `s.StartSecondExit(ctx)` | **No.** Source `node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/runs/2026-10-31/followup15/exports/second-exit.json`: enrolls independent `node2` (`clone = false`) before `exportsClone` runs on `node-clone`. |
| `tests/e2e/fixtures16_live_test.go:99` (`TestE2E_Followup16Fixtures`) | `s.StartSecondExit(ctx)` | **No.** Source `node` keeps `control_url = "https://127.0.0.1:9444"`. | Enrolls independent `node2` (`second-resolver` record, `clone = false`), not a clone. |
| `tests/e2e/followup18_native_test.go:114` (`TestE2E_Followup18Native`) | `s.StartSecondExit(ctx)` | **No.** Source `node` keeps `control_url = "https://127.0.0.1:9444"`. | Enrolls independent `node2` (`second-exit-counted` and `namespace-restart-90`, `clone = false`), not a clone. |
| `tests/e2e/at/enrollment/live_test.go:890,1552` (`at07`, `peerUniqueness`) | `p.stack.StartSecondExit(p.ctx)` | **No.** Source `fvpn-w4-enroll-node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/at/enrollment/AT07.json` (`distinct_api_first_boot`) and `AT06.json` (`peer_uniqueness`): enrolls/races independent `node2` (`clone = false`), not a clone. |
| `tests/e2e/at/members/second_exit_test.go:22` (`secondExit`) | `l.hook.StartSecondExit(l.ctx)` | **No.** Source `fvpn-w4-members-node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/at/members/AT09.json`: enrolls independent `node2` (`clone = false`), not a clone. |
| `tests/e2e/at/updates/restated_test.go:152` (`inheritDefault`) | `stack.StartSecondExit(s.ctx)` | **No.** Source `fvpn-w4-updates-node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/at/updates/AT24.json`: enrolls independent `node2` (`clone = false`), not a clone. |
| `tests/e2e/at/usage/live_cases_b_test.go:262`, `tests/e2e/at/usage/live_helpers_test.go:1066` (`evalAT19`, `measurePerIncarnationRetainedUsage`) | `s.hook.StartSecondExit(s.ctx)` | **No.** Source `fvpn-w4-usage-node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/at/usage/AT19.json`: enrolls and re-enrolls independent `node2` (`clone = false`), not a clone. |
| `tests/e2e/at/secops/two_exit_live_test.go:31` (`twoExitTraffic`) | `stack.StartSecondExit(p.t.Context())` | **No.** Source `fvpn-w4-secops-node` keeps `control_url = "https://127.0.0.1:9444"`. | `docs/e2e/at/secops/`: enrolls independent `node2` (`clone = false`), not a clone. |
| `tests/e2e/fleet_snapshot.go:44` (`RetainEnrollingThirdNode`) | `s.startExitSlot(ctx, "node3", ".24", false, true)` | **No** on source `node`; after `startExitSlot`, lines `71-77` intentionally rewrite `node3/etc/agent-enroll.json` `control_url` to `"https://127.0.0.1:1"` while `node3` is held in `enrolling`, then restore `originalControlURL` in `BootEnrollingThirdNode` (`tests/e2e/fleet_snapshot.go:116-127`). | Enrolls independent `node3` (`clone = false`), not a clone. |

---

### 1.5 The Verdict

1. **Does the clone reach control in isolation's runs?**
   - **No, not since isolation follow-up 16** (commit `8afa69c5` / PR #399, first recorded in `docs/e2e/at/isolation/history/follow-up-16/identity-freeze.json.gz`, run `83b457ef-ea16-490b-9923-1fea0df86b36`).
2. **Since when and why:**
   - In **isolation follow-ups 7 through 15** (runs `041e9e48-fa38-47bf-b7cb-beac596b96fc` through `0b9e4510-79a2-40b2-a151-2ba53c5750fb`), `observeProxy()` in `tests/e2e/at/isolation/observer_main.go` listened on `127.0.0.1:9444` and `configureObservation()` left `control_url` at `"https://127.0.0.1:9444"`. When the clone booted without the observer wrapper, its own `fvpn-e2e-nodeconfig resolver-proxy` listened on `127.0.0.1:9444` and forwarded to `https://agent.example.invalid:9443` (`pres.Control` on `fvpn-w4-isolation-node-clone-net`). Thus the clone reached Central on `127.0.0.1:9444`, triggered `conflicts: 1` and `mapping_frozen: true`, and `freezeIdentity()` (`tests/e2e/at/isolation/driver.go.txt:2693-2696`) exited early after 3–4 rows (`last_row_label: "freeze latch held"`).
   - In **isolation follow-up 16 (`8afa69c5` / #399)**, to chain the observer proxy in front of `fvpn-e2e-nodeconfig-original resolver-proxy` on the source node so devstack 27's lease-hold seam on `:9444` remained in path, `tests/e2e/at/isolation/observer_main.go:48,95,125` moved `observeProxy()` to `127.0.0.1:9445` (forwarding to `https://127.0.0.1:9444`) and rewrote `cfg["control_url"] = "https://127.0.0.1:9445"` in `configureObservation()`.
   - Because `startExitSlot` (`tests/e2e/second_exit.go:72-94`) copied `/etc/vpn-node/agent-run.json` (and `/var/lib/vpn-node/e2e/agent-<boot_release_id>.json`) from the source node and rewrote only `datapath.snat`, every isolation run from **follow-up 16 (`83b457ef`) through follow-up 25 (`bd73d7a4`)** booted the clone with `control_url = "https://127.0.0.1:9445"`, where nothing listens inside the clone container. Consequently, every run from follow-up 16 through follow-up 25 exhausted all 90 polling iterations (`row_count = 90`) with `max(endpoint_rows) = 1`, `max(conflicts) = 0`, and `mapping_frozen = false`.
3. **Which recorded claims, if any, depend on it:**
   - **No `AT10`–`AT13` pass/blocked/fail verdict and no stop-receipt (`review_stops.py`) claim in follow-ups 16–25 is invalidated:**
     - `freezeIdentity()` (`tests/e2e/at/isolation/driver.go.txt:2658-2703`) does not score any `AT10`–`AT13` check in `x.records`; it writes only the auxiliary record `identity-freeze.json.gz` with `"status": "requires receiver correlation"` and `"owner": "ENR-02 / ENR-03; this is the consumer latch observation only"`.
     - `tests/e2e/at/isolation/review_stops.py:23-47` uses `identity-freeze.json.gz` only to verify the **source** container's `docker stop -t 25` snapshot stop/start (`label = "clone_source"`), and uses `receiver-closure.json.gz` (`label = "direct_agent"`) to verify that `stopCloneAgent` cleanly stopped the running clone `fvpn-agent` process via SIGTERM (`exit_code: 0`, `wait_source: "pidfd_poll_and_host_proc_reap"`).
   - **What did depend on it:** Isolation's auxiliary consumer-latch observation in `docs/e2e/at/isolation/identity-freeze.json.gz` (observing `conflicts: 1`, `mapping_frozen: true`, and the resulting `identity_frozen` fail-closed latch on the source node during `freezeIdentity`) went unexercised in follow-ups 16–25 because the clone never reached control. Meanwhile, the primary `ENR-02` / `ENR-03` / `AT07` clone-conflict and serving-freeze-withdrawal proofs remained exercised and passing in enrollment's `docs/e2e/at/enrollment/AT07.json` (`run_id: 891ee9b8-bf01-4583-a47a-9542c8b04c8a`) and devstack's `docs/e2e/runs/2026-10-31/followup15/exports/clone-conflict.json` (`run_id: 12d10041-9d33-4fbb-a4b0-9d2a21bce785`), where the source node does not carry isolation's observer.

---

## 2. The Shared Fixture Fix and Offline Test Proof

### 2.1 Shared Fixture Change (`tests/e2e/second_exit.go`)

- **Why the fix belongs in the shared fixture (`tests/e2e/second_exit.go`):**
  - `startExitSlot` (`tests/e2e/second_exit.go:28-154`) boots the second or cloned container (`node2` or `node-clone`) from the unmutated base Docker image of `s.lab.node` (`tests/e2e/second_exit.go:82-84,92,151`). Inside that container, `deploy/devstack/entrypoint-node.sh:108` starts the base image's `/usr/local/bin/fvpn-e2e-nodeconfig resolver-proxy` on `127.0.0.1:9444` (`localNodeControlURL = "https://127.0.0.1:9444"`), which proxies mTLS agent requests to `https://agent.example.invalid:9443` (`pres.Control` on `fvpn-<prefix>-node-clone-net` when `clone == true`).
  - Isolation's `freezeIdentity` (`tests/e2e/at/isolation/driver.go.txt:2658-2703`) only needs the clone to reach Central from its distinct `/24` presentation network so Central records `conflicts = 1` and `mapping_frozen = true`, causing the **source** node (`x.r.lab.node`) to receive `identity_frozen = true` through its own observer proxy on `:9445`. Isolation does not read observer timing logs from the clone.
- **Exact change in `tests/e2e/second_exit.go`:**
  - Extracted `prepareExitSlotConfigs(base, snat string, clone bool) error`, `rewriteExitConfigFile`, and `rewriteExitConfig` (`tests/e2e/second_exit.go:156-223`), called before container creation (`tests/e2e/second_exit.go:73-75`) and after copying `/var/lib/vpn-node/.` into `base/var` when `clone == true` (`tests/e2e/second_exit.go:124-126`).
  - Using `map[string]json.RawMessage` at both the top level and inside `datapath`, `prepareExitSlotConfigs`:
    - sets `datapath.snat` to `snat` (`prefix + lastOctet`) in `base/etc/agent-enroll.json`, `base/etc/agent-run.json`, and every `base/var/e2e/agent-*.json`;
    - sets `control_url` to `"https://127.0.0.1:9444"` in `base/etc/agent-run.json`, in every `base/var/e2e/agent-*.json`, and (when `clone == true`) in `base/etc/agent-enroll.json` (while preserving `base/etc/agent-enroll.json`'s enrollment `control_url` when `clone == false`);
    - keeps every other copied field (including `test_timing_path`, `heartbeat_seconds`, `poll_seconds`, `observation_seconds`, `release_id`, `agent_release`, `dns`, `host_reserve_bps`, and `datapath.{member_ingress,uplink}`), the clone's identity files (`var/identity/{node.json,cert.pem,key.pem}`), `/etc/vpn-node/machine-id`, and durable state (`var/state/agent.db`) byte-identical.

### 2.2 Offline Test (`tests/e2e/second_exit_test.go`)

- `TestD141_ClonePreparedConfigsDialContainerResolverProxy` (`tests/e2e/second_exit_test.go:15-218`) constructs a fake source configuration carrying isolation's observer settings (`"control_url": "https://127.0.0.1:9445"`, `"heartbeat_seconds": 1`, `"poll_seconds": 1`, `"observation_seconds": 1`, `"test_timing_path": "/var/lib/vpn-node/e2e/isolation-operations.jsonl"`, `"datapath": {"member_ingress": "fvmb0", "snat": "198.51.100.20", "uplink": "fvup0"}`) alongside `agent-enroll.json`, `var/e2e/agent-00000000-0000-7000-8000-000000000001.json`, cloned identity files, and durable state, and verifies:
  1. All prepared clone configs (`etc/agent-enroll.json`, `etc/agent-run.json`, and `var/e2e/agent-00000000-0000-7000-8000-000000000001.json`) name `"control_url": "https://127.0.0.1:9444"` and `"snat": "198.51.100.23"`.
  2. Every other top-level and `datapath` key is byte-for-byte unchanged (`bytes.Equal`), and restoring only `control_url` and `snat` reproduces the exact input file bytes.
  3. All cloned identity and durable state files (`etc/machine-id`, `var/identity/{node.json,cert.pem,key.pem}`, `var/state/agent.db`, `var/e2e/agent-selected`) are untouched (`bytes.Equal`).
  4. In clean second-exit mode (`clone == false`), `etc/agent-enroll.json` preserves `"https://portal.example.invalid:8443"` while `etc/agent-run.json` points to `"https://127.0.0.1:9444"`.
  5. Malformed JSON or a config missing `datapath` is rejected.

---

## 3. Carried Items and Hand-offs

- **Carried:**
  - `AT29` guest-clock rollback remains an operator-run row ([`docs/e2e/at29-guest-clock.md`](../../../at29-guest-clock.md)).
  - `member-ipv6` remains listed in `docs/e2e/nightly-expected-blocked.json`.
  - No live run was executed in this follow-up, and no historical record in `docs/e2e/` or `docs/e2e/at/**` was rewritten or re-scored.
- **Hand-offs (`### Requests to orchestrator`):**
  1. **`w4-at-isolation` (`tests/e2e/at/isolation/driver.go.txt:2658-2703`, `tests/e2e/at/isolation/driver.go.txt:2306-2331`):**
     - The shared clone fixture in `tests/e2e/second_exit.go` now rewrites the clone's prepared configs (`etc/agent-run.json`, `var/e2e/agent-*.json`, and `etc/agent-enroll.json`) to dial `https://127.0.0.1:9444` (served by the clone's own `fvpn-e2e-nodeconfig resolver-proxy`). Isolation's next authorized assembled run will exercise `freezeIdentity` live against this fix.
     - Optional isolation-owned observability improvement: `retainReasonLogs` (`tests/e2e/at/isolation/driver.go.txt:2306-2331`) and `archiveReceiver` (`tests/e2e/at/isolation/driver.go.txt:2378-2388`) currently retain `/var/lib/vpn-node/e2e/{agent,watchdog}.log` only from the primary node (`x.r.lab.node`). If isolation wants retained clone agent/watchdog logs in future runs, capture `/var/lib/vpn-node/e2e/{agent,watchdog}.log` from `x.r.lab.prefix + "-node-clone"` before `x.stopCloneAgent(fixture.Clone)` at `tests/e2e/at/isolation/driver.go.txt:2698`.

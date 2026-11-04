# Review r28: Diagnosis of F20-01 (`tunnel_probe` / `HEALTH_CHECK_FAILED` during short whole-node lease expiry under control stop)

- **Task:** `r28-expiry-probe-review`
- **Base commit:** `9be409f8c1297dcdda8c9d12d4b192d99c52a190` (`main` at `c359d4a9` + isolation follow-up 20's 5 commits; product code identical to run `c52bea6c-cd91-4086-a779-656b01277dcc` built at `037f4418` and current `main` `6548329f`)
- **Reviewer:** `agy/gemini-4-argon-high`

---

## 1. Diagnosis

| Premise | What the evidence shows | The cause | The class | The proof | The owner of the fix |
|---|---|---|---|---|---|
| **1. `tunnel_probe` status read** (`netbirdProber.Probe` -> `p.read(readCtx)` -> `exec.CommandContext(ctx, "netbird", "status", "--json").Output()`, `node/agent/probe.go:114-122, 178-192`) | **Failed (`statusOK = false`)** 5.000 s after `GET /api/v1/agent/desired-state` started against the stopped `control` container (`00:25:57.396Z` -> `00:26:02.367Z`), producing `detail="overlay_down"`, `error_code="HEALTH_CHECK_FAILED"`, `netbird.management_connected=false`, and `netbird.version=null` (`<unset>`) in observation `019a4c93-dd39-70be-aa64-3228b1cf3fa0` (`seq 84`, `observed_at=2026-11-04T00:26:02.441819388Z`). | `Engine.Step` (`node/agent/engine.go:667-668`) creates one shared 5 s `ioCtx` (`controlPassTimeout() = min(5s, stale/3)`, `node/agent/serving.go:230-236`) under `controlBudgetKey{}` (`controlContext(ctx)`). In `Step`, `syncDesired(ctx)` (`engine.go:737`, `sync.go:36`) blocks for the full 5 s `ioCtx` budget trying to reach the stopped `control` container, exhausting `controlContext(ctx)` (`context.DeadlineExceeded`). When `evaluate(ctx)` -> `refreshProbe(ctx)` runs next (`engine.go:768`, `attest.go:72`), `refreshProbe` calls `p.Probe(controlContext(ctx), ...)` (`node/agent/engine.go:948`) instead of `localContext(ctx)`. `exec.CommandContext` sees the already-expired `controlContext(ctx)` and fails immediately with `context.DeadlineExceeded` without running `netbird status --json`. | **product** | **[Proof test]** `TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry` and `TestR28_WhatHeld_RefreshProbeAndNetbirdProberContextPropagation` (`node/agent/review_r28_test.go:53-155`); **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-failure-wire-extract.json` (`attempt_timeline` `seq 83` & `seq 84`) and `agent.log.txt:295-298`. | `agent-core` |
| **2. NetBird management & signal connectivity** (`st.Management.Connected`, `node/agent/probe.go:86`) | **Held in the environment; never read by the agent.** In `seq 82` (`00:25:48.921Z`, before control stop) and again after `control` restarted (`00:27:06Z`), NetBird management was connected and version `0.82.0` was reported. During `00:26:02Z`–`00:26:58Z`, `netbird.version` was `<unset>` (`null`) in `seq 84` (`expiry-failure-wire-extract.json:79-85`), proving `statusOK` was `false` (`probe.go:152-155`) rather than `st.Management.Connected == false` on a parsed JSON payload. | `control` (`fvpn-control`, `172.28.30.30`) and `worker` (`fvpn-worker`, `172.28.32.31`) are separate containers from `netbird` (`netbirdio/netbird-server:0.82.0`, `172.28.31.20` on `underlay`, `deploy/devstack/compose.yaml:129-186, 212-279`). Stopping `control` and `worker` did not stop NetBird management, signal, or relay. | **product** (false attribution caused by Premise 1) | **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-failure-wire-extract.json` (`seq 82` vs `seq 84`, `netbird` object); **[Code]** `node/agent/probe.go:152-155` and `deploy/devstack/compose.yaml:129-186, 212-279`. | `agent-core` (resolved by fixing Premise 1) |
| **3. NetBird peer & relay state** (`st.Relays.Total == 0 \|\| st.Relays.Available > 0`, `node/agent/probe.go:90-98`) | **Held.** Even when `st.Relays.Available == 0`, `judgeProbe` (`node/agent/probe.go:96-98`) returns `(DataHealthDegraded, HealthCheckStatusPassed)` with `detail="overlay_partial"`, never `DataHealthFailed` / `HealthCheckStatusFailed` (`detail="overlay_down"`). Moreover, continuous data-plane probes from `a1` (`100.64.0.105`) to `origin` succeeded every ~15 ms across `fvnb0` until the agent's own `nft -f` baseline write at `00:26:02.373Z` (`deadline-verdict.json`). | Not a failure cause; `judgeProbe` only returns `(DataHealthFailed, HealthCheckStatusFailed)` with `detail="overlay_down"` on line 86 (`!statusOK \|\| !st.Management.Connected`). | **product** (not implicated) | **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/deadline-verdict.json` (`positive_count: 159`, `last_positive_mono_ns: "75404810483344"`); **[Code]** `node/agent/probe.go:85-100`; **[Proof test]** `TestJudgeProbe` (`node/agent/probe_test.go:15-55`). | `agent-core` (none; held) |
| **4. Egress probe target dial** (`p.dialAddr(ctx, addr, timeout)` -> `dialProbe`, `node/agent/probe.go:123-140, 194-209`) | **Not run in `F20-01`, but shares the same latent context bug.** In `seq 84`, `probe_source_detail` was `"Probe used no egress target; overlay status only."` (`expiry-failure-wire-extract.json:77`), and `statusOK` was already `false`. However, when an egress target is configured (`req.Targets` or `req.Egress`), `p.dialAddr(ctx, addr, timeout)` (`node/agent/probe.go:136`) also inherits `ctx` (`controlContext(ctx)`). | `refreshProbe` (`node/agent/engine.go:948`) passes `controlContext(ctx)` into `p.Probe(ctx, ...)`, which passes `ctx` into both `p.read(readCtx)` (`line 114`) and `p.dialAddr(ctx, addr, timeout)` (`line 136`). | **product** | **[Proof test]** `TestR28_WhatHeld_RefreshProbeAndNetbirdProberContextPropagation` (`node/agent/review_r28_test.go:118-155`); **[Code]** `node/agent/engine.go:948`, `node/agent/probe.go:114, 136, 198-204`. | `agent-core` |
| **5. Member ingress & uplink interfaces** (`p.memberUp(req.Ingress, req.Uplink)` on `fvnb0` and `eth0`, `node/agent/probe.go:141-144, 211-225`) | **Held.** Both `fvnb0` (`MemberIngress`) and `eth0` (`Uplink`) remained up with `net.FlagUp != 0`. If `memberUp` had failed while `statusOK && st.Management.Connected` held, `judgeProbe` (`node/agent/probe.go:99`) would have returned `(DataHealthDegraded, HealthCheckStatusFailed)` with `detail="overlay_partial"`, whereas `seq 84` recorded `data_health="failed"` and `detail="overlay_down"`. | Not a failure cause; `fvnb0`, `fvmb0`, and `eth0` did not change state during `wholeNodeExpiry`, and `source_binding`, `qos`, and `lease` checks all reported `passed` in `seq 84`. | **product** (not implicated) | **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-failure-wire-extract.json` (`seq 84`, `checks` array and `data_health: "failed"`); **[Code]** `node/agent/probe.go:90-100, 145-151`. | `agent-core` (none; held) |
| **6. Sibling AT13 check `memberExpiry` predicate under control loss** (`tests/e2e/at/isolation/driver.go.txt:1241-1285`) | **Masked the same `H-1` failure as a pass.** In AT13 `"lease member expiry under control loss"`, the driver stopped `control` and `worker` at `00:24:02Z` (`driver.go.txt:1241`). Exactly 5.0 s later (`00:24:07.139Z`), `DesiredState` timed out `controlContext(ctx)` and the agent installed the whole-node `safety_check_failed` baseline (`00:24:07.144Z` and `00:24:07.172Z`, `agent.log.txt:259-263`), leaving heartbeat `serving 0` / `reason safety_check_failed` (`AT13.json:1098`). | `memberExpiry` (`driver.go.txt:1274-1278`) only checked `nodeStillValid := time.Now().Before(bounds.Expires)` on the wall clock without checking that the node remained `serving 1` (or that an unexpired sibling member could still forward) through the 5 post-deadline member probes (`00:24:06Z`–`00:24:11Z`). Whole-node `safety_check_failed` supplied the denial that `memberExpiry` attributed to per-member expiry. | **fixture** (test predicate gap in `isolation`) | **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/agent.log.txt:259-275` and `AT13.json:871-1103` (`heartbeat: "...\nserving 0\ngeneration 31\nreason safety_check_failed"`); **[Code]** `tests/e2e/at/isolation/driver.go.txt:1241-1285`. | `isolation` |

---

## 2. Detailed Answers to Brief Questions (Q1–Q5)

### Q1. The sequence from the control stop to producer sequence 1933's baseline

- **Active lease & deadline `D`:**
  - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-early-baseline.json` (`node_lease` and `AT13_check.observations`):
    - `lease_id`: `72fd35da-5c75-432c-be6b-f4e21cb81939`
    - `generation`: `"33"`
    - `max_offline_seconds`: `60`, `member_count`: `3`
    - `issued_at`: `2026-11-04T00:25:43.406500722Z`
    - `expires_at` (`D`): `2026-11-04T00:26:43.406500722Z`
    - `mono_sent_ns`: `75385854784464`
    - `deadline_mono_ns` (`D_mono`): `75445854784464` (`mono_sent_ns + 60,000,000,000 ns`)
- **Control stop (`T_stop`):**
  - **[Code]** `tests/e2e/at/isolation/driver.go.txt:1392`: `exec.Command("docker", "stop", "--time", "2", x.r.lab.control, x.r.lab.prefix+"-worker").Run()`
  - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/container-lifecycle-events.json:93-128`: `fvpn-w4-isolation-control` and `fvpn-w4-isolation-worker` stopped; `control_stopped_at` recorded by the driver at `2026-11-04T00:25:57Z` (`D - 46.406s`, `[Record] expiry-early-baseline.json:47`).
- **Step-by-step timeline across the control stop:**
  1. **Start of `Engine.Step` (`00:25:57.367Z`, `T_stop + 0.37s`, `D - 46.039s`):**
     - **[Code]** `node/agent/engine.go:667-673`: `Step` creates `ioCtx, cancelIO := context.WithTimeout(ctx, e.controlPassTimeout())` where `controlPassTimeout()` is `5s` (`WatchdogStaleSeconds = 30`, `node/agent/serving.go:230-236`), attaches `ioCtx` under `controlBudgetKey{}` (`controlContext(ctx)`), and attaches a separate 10 s `localBudget` (`localContext(ctx)`). Because `ioCtx` was created at `00:25:57.367Z`, its 5.000 s deadline is `00:26:02.367Z`.
  2. **Last successful live verification (`producer sequence 1931`, `00:25:57.367Z`–`00:25:57.393Z`, `D - 46.013s`) and heartbeat (`producer sequence 1932`, `00:25:57.396Z`):**
     - **[Code]** `node/agent/engine.go:720`: `e.verifyCurrentServing(ctx)` runs at the top of `Step` using `localContext(ctx)` (`node/agent/serving.go:66`) before any control HTTP call.
     - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-early-baseline.json:76-92` (`producer seq 1931` & `1932`):
       - `seq 1931` (`source: "verify"`, `generation: "33"`, `mono_start_ns: "75399817021077"`, `mono_end_ns: "75399842900496"`, `wall: "2026-11-04T00:25:57.393446918Z"`, `verified: true`).
       - `seq 1932` (`source: "heartbeat"`, `serving: true`, `generation: "33"`, `reason: "serving"`, `written_mono_ns: "75399845676177"`, `policy_expires_mono_ns: "75445854784464"`, `wall: "2026-11-04T00:25:57.396231926Z"`).
       - During this window (`00:25:58Z`), the driver's post-stop positive check `earlyOK, before, after, earlyDetail := x.countedForward(x.a1)` (`driver.go.txt:1417`) succeeded (`bytes=182808->185389`, `[Record] expiry-early-baseline.json:53-55`).
  3. **`syncDesired(ctx)` blocks for the full 5 s `controlContext(ctx)` budget (`00:25:57.396Z`–`00:26:02.367Z`, `D - 46.010s` to `D - 41.039s`):**
     - **[Code]** `node/agent/engine.go:737` -> `node/agent/sync.go:36`: `e.client.DesiredState(controlContext(ctx), e.desiredETag)` sends `GET /api/v1/agent/desired-state` through the local telemetry/first-offer proxy chain (`127.0.0.1:9445` -> `127.0.0.1:9444` -> `172.28.30.30:9443`, `deploy/devstack/entrypoint-node.sh:214-243`).
     - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-failure-wire-extract.json:27-35` (`wire seq 83`): `method: "GET"`, `path: "/api/v1/agent/desired-state"`, `started_at: "2026-11-04T00:25:57.396605375Z"`, `duration_ms: 4971`, `error: "Get \"https://control.example.invalid:9443/api/v1/agent/desired-state\": context deadline exceeded"`.
     - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/agent.log.txt:295`: `time=2026-11-04T00:26:02.367Z level=WARN msg="desired state" err="control request ended"`.
     - **[Code]** `node/agent/sync.go:43-50`: Because `ctx.Err()` (the step/local context) is `nil`, `syncDesired` logs `"desired state"` and returns `nil` so the step can continue with the cached in-force desired state and lease. **However, `controlContext(ctx)` (`ioCtx`) is now permanently expired (`context.DeadlineExceeded`) for the remainder of this `Step`.**
  4. **`syncLease(ctx)` (`00:26:02.367Z`):**
     - **[Code]** `node/agent/engine.go:749` -> `node/agent/sync.go:104-110`: Either skipped if `!e.needRenew()` or, when `RenewLeadSeconds = 120` exceeds the 60 s lease window so `needRenew()` is `true`, `RenewLease(controlContext(ctx), ...)` (`sync.go:116`) fails in `0ms` before hitting the wire because `controlContext(ctx).Err()` is already `context.DeadlineExceeded`, logging `time=2026-11-04T00:26:02.367Z level=WARN msg="lease renewal" err="control request ended"` (`[Record] agent.log.txt:296`).
  5. **`evaluate(ctx)` -> `refreshProbe(ctx)` fails immediately on the expired `controlContext(ctx)` (`00:26:02.367Z`, `D - 41.039s`):**
     - **[Code]** `node/agent/engine.go:768` -> `node/agent/attest.go:72` -> `node/agent/engine.go:948`:
       ```go
       e.lastProbe = p.Probe(controlContext(ctx), ProbeRequest{...})
       ```
     - **[Code]** `node/agent/probe.go:114-122, 178-192`: `netbirdProber.Probe` derives `readCtx, cancel := context.WithTimeout(ctx, probeTimeout)` from the already-expired `controlContext(ctx)` and calls `exec.CommandContext(readCtx, "netbird", "status", "--json").Output()`. Because `readCtx.Err() == context.DeadlineExceeded`, `exec.CommandContext` returns immediately without starting the child process. `statusOK` is `false`, `judgeProbe` (`probe.go:86`) returns `(DataHealthFailed, HealthCheckStatusFailed)`, and `Probe` returns `ProbeResult{Ran: true, Check: HealthCheckStatusFailed, Health: DataHealthFailed, Detail: "overlay_down", NetbirdVersion: ""}`.
     - **[Code]** `node/agent/decide.go:246-268, 456-457, 496-504`: `Decide` sees `ProbeCheck == HealthCheckStatusFailed`, adds `safety_check_failed` to `Reasons`, appends `NodeError{Code: "HEALTH_CHECK_FAILED", Stage: "probe", Message: "tunnel_probe"}`, sets `Safety = failed`, `DataHealth = failed`, `Serve = false`, and clears `PolicyExpires = 0` — even though `lease`, `qos`, and `source_binding` all pass (`[Record] expiry-failure-wire-extract.json:36-96`, `seq 84`).
  6. **`enforce(ctx)` installs the fail-closed baseline (`producer sequences 1933 & 1934`, `00:26:02.373Z`–`00:26:02.440Z`, `40.996s` before `D`) and writes non-serving heartbeat (`producer sequence 1935`, `00:26:02.450Z`):**
     - **[Code]** `node/agent/engine.go:1142-1153`: Because `!e.decision.Serve`, `enforce` calls `e.handoff(ctx)` (which calls `e.baseline(ctx)` during active-epoch handoff) and then `e.baseline(ctx)` (`engine.go:1143, 1146`), both using `localContext(ctx)`.
     - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-early-baseline.json:93-129` and `agent.log.txt:297-300`:
       - `seq 1933` (`source: "baseline"`, `cause: "safety_check_failed"`, `mono_start_ns: "75404823390077"`, `mono_end_ns: "75404860484304"`, `wall: "2026-11-04T00:26:02.410982297Z"`, `succeeded: true`) — **`40.996s` before `D` (`00:26:43.406500722Z`)**.
       - Continuous data-plane probe flow from `a1` saw its last positive packet at `mono_ns = 75404810483344` and its first dropped packet at `mono_ns = 75404840687570` (`41.015s` before `D_mono = 75445854784464`, `[Record] deadline-verdict.json:4-13`), right inside `seq 1933`'s `nft -f` transaction (`75404823390077`..`75404860484304`).
       - `seq 1934` (`source: "baseline"`, `cause: "safety_check_failed"`, `mono_start_ns: "75404865319465"`, `mono_end_ns: "75404890567771"`, `wall: "2026-11-04T00:26:02.441068393Z"`, `succeeded: true`).
       - `seq 1935` (`source: "heartbeat"`, `serving: false`, `generation: "33"`, `reason: "safety_check_failed"`, `written_mono_ns: "75404900181623"`, `wall: "2026-11-04T00:26:02.450681154Z"`).
  7. **Outbox observation queued and flushed at the start of the next step (`00:26:02.441Z`–`00:26:02.451Z`):**
     - **[Code]** `node/agent/engine.go:791-792` & `node/agent/observe.go:296`: `observe(ctx)` queues report `019a4c93-dd39-70be-aa64-3228b1cf3fa0` (`observed_at: 2026-11-04T00:26:02.441819388Z`); `flush(ctx)` at the end of Step 1 returns immediately because `controlContext(ctx).Err() != nil`. Because Step 1 took `5.083s`, the next step starts immediately at `00:26:02.450Z` with a fresh `ioCtx` and calls `flush(ctx)` (`engine.go:736`), producing `wire seq 84` (`POST /api/v1/agent/observations` at `00:26:02.45158891Z`, `[Record] expiry-failure-wire-extract.json:36-96`).
  8. **Subsequent steps until and after `D` (`00:26:02.450Z`–`00:26:48.329Z`):**
     - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-early-baseline.json:130-310` and `agent.log.txt:301-355`: Every 5 s (`00:26:07.452Z`, `00:26:12.489Z`, `00:26:17.523Z`, `00:26:22.559Z`, `00:26:27.596Z`, `00:26:32.633Z`, `00:26:38.260Z`, `00:26:43.297Z`), `flush`/`syncDesired` exhausts the 5 s `controlContext(ctx)`, `refreshProbe` fails with `context.DeadlineExceeded`, and `Step` re-applies `baseline` with `cause=safety_check_failed` (`producer seq 1936`–`1959`).
     - On the first step that completes after `D` (`00:26:48.299Z`, `producer seq 1960`–`1961`), `Decide` adds `lease_expired` (`node/agent/decide.go:201`) ahead of `safety_check_failed` (`node/agent/decide.go:248`), so `e.baselineReason()` (`node/agent/measurement.go:88-89`) switches from `"safety_check_failed"` to `"lease_expired"` (`[Proof test] TestR28_WhatHeld_PostDeadlineReasonOrderingInDecideAndBaseline`).

---

### Q2. The probe

1. **What `tunnel_probe` checks and on which lines (`node/agent/probe.go`):**
   - **[Code]** `node/agent/probe.go:113-157` (`netbirdProber.Probe`):
     - **Lines 114–122 & 178–192 (`p.read`):** Runs `exec.CommandContext(readCtx, "netbird", "status", "--json").Output()` (with `readCtx, cancel := context.WithTimeout(ctx, probeTimeout)` where `probeTimeout = 2 * time.Second`, `line 23`) or reads `p.statusFile` if configured, and unmarshals JSON into `nbStatus`. Sets `statusOK = true` iff both the read and `json.Unmarshal` succeed.
     - **Lines 123–140 & 194–209 (`p.dialAddr` -> `dialProbe`):** If an egress probe target is present (`req.Targets` or `req.Egress`) and `!req.Rejected && statusOK && st.Management.Connected`, dials TCP to each target using `d.DialContext(ctx, "tcp", addr)` with a child timeout of `req.Timeout` (default `2s`).
     - **Lines 141–144 & 211–225 (`p.memberUp`):** If `req.Published` is true, verifies that `req.Ingress` (`fvnb0`) and `req.Uplink` (`eth0`) are non-empty, distinct, exist via `net.InterfaceByName`, and have `net.FlagUp != 0`.
     - **Lines 85–100 & 145–156 (`judgeProbe`):**
       - Line 86: `if !statusOK || !st.Management.Connected { return api.DataHealthFailed, api.HealthCheckStatusFailed }` (with `detail = "overlay_down"` at line 146).
       - Lines 90–95: If `relayOK && egressPass && memberPass`, returns `(api.DataHealthHealthy, api.HealthCheckStatusPassed)` (`detail = "overlay_up"` at line 148).
       - Lines 96–98: If `egressPass && memberPass` (only relays unavailable), returns `(api.DataHealthDegraded, api.HealthCheckStatusPassed)` (`detail = "overlay_partial"` at line 150).
       - Line 99: Otherwise (egress dial failed or published member interfaces down), returns `(api.DataHealthDegraded, api.HealthCheckStatusFailed)` (`detail = "overlay_partial"` at line 150).
2. **Which premise failed here:**
   - **The status read itself (`statusOK == false`) failed** because `exec.CommandContext(readCtx, "netbird", "status", "--json")` at `node/agent/probe.go:190` received an already-expired `context.Context` (`controlContext(ctx)` from `node/agent/engine.go:948`, which had `Err() == context.DeadlineExceeded`).
   - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-failure-wire-extract.json:79-85` (`seq 84`) confirms `statusOK == false`: in the observation's `netbird` status object, `version` is `<unset>` (`null`), whereas whenever `statusOK == true` (even if `st.Management.Connected == false`), `netbirdProber.Probe` (`node/agent/probe.go:153-155`) populates `NetbirdVersion: "0.82.0"` (verified by `[Code] node/agent/probe_test.go:179-182` and `[Record] expiry-failure-wire-extract.json:19-24` `seq 82`).
3. **How long after the control stop it failed, and what timing that delay matches:**
   - It failed **5.000 seconds** after the step began (`00:25:57.367Z` -> `00:26:02.367Z`), which is `4971 ms` after `GET /api/v1/agent/desired-state` started (`00:25:57.396Z` -> `00:26:02.367Z`, `[Record] expiry-failure-wire-extract.json:31-33`).
   - That 5.000-second delay matches **the agent's own per-step control-plane HTTP pass timeout** (`Engine.controlPassTimeout() = min(5*time.Second, stale/3) = 5*time.Second` for `WatchdogStaleSeconds = 30`, `[Code] node/agent/serving.go:230-236` and `node/agent/engine.go:667`), **not** any NetBird client keepalive or reconnect timer.

---

### Q3. The fixture

1. **What the `control` container runs, and whether stopping it stops NetBird's management, signal, or relay:**
   - **[Code]** `deploy/devstack/compose.yaml:212-279`: `control` runs `/usr/local/bin/fvpn-control` (`172.28.30.30` on `lab`, `172.28.32.30` on `core`) and `worker` runs `/usr/local/bin/fvpn-worker` (`172.28.32.31` on `core`).
   - **[Code]** `deploy/devstack/compose.yaml:129-186` & `deploy/devstack/nginx.conf`: NetBird's combined management, signal, and relay server (`netbirdio/netbird-server:0.82.0`) runs in the separate `netbird` container (`172.28.31.20` on `underlay`), fronted by the `gateway` nginx container (`netbird.example.invalid:443`).
   - Stopping `fvpn-w4-isolation-control` and `fvpn-w4-isolation-worker` (`tests/e2e/at/isolation/driver.go.txt:1392`) stops **only** Freedom VPN's own control API and background worker; `netbird` and `gateway` remain running (`[Record] docs/e2e/at/isolation/history/follow-up-20-run/container-lifecycle-events.json:93-128`).
2. **Would a real outage of the product's control plane stop the same things?**
   - **Yes.** In production (`§5.1`, `§17.1`), `fvpn-control` and `fvpn-worker` are the central Freedom VPN services, distinct from the NetBird overlay planes. When `fvpn-control` is unreachable or blackholed (so TCP connection attempts or TLS handshakes to `fvpn-control` time out after the 5 s `controlPassTimeout()`), any production node running `fvpn-agent` hits the exact same `controlContext(ctx)` exhaustion in `syncDesired` / `syncLease` and immediately fails `refreshProbe` (`node/agent/engine.go:948`) with `safety_check_failed` (`HEALTH_CHECK_FAILED`), tearing down member forwarding within 5 seconds regardless of how much time remains on its signed serving lease.

---

### Q4. The spec

1. **Should a node that holds a valid signed lease keep forwarding while the control plane is unreachable?**
   - **Yes — mandatory (`必須`).**
   - **[Spec]** `docs/spec/PROJECT_SPEC.zh-TW.md`:
     - `§5.1` (line 272): `"serving_lease（節點短期續租憑證，決定 Control API 失聯時還能服務多久）"`.
     - `§16.3` (lines 1471–1477): `"必須保存並驗證最新 serving_lease，於離線容忍窗內維持既有轉發，超過 expires_at 後 fail-closed。"`
     - `§17.1` (line 1500): `"自家 Control API 暫時不可用 | 新納管、綁定變更、續租與同步延後；正常節點可在現有租約內服務；已知有效服務截止仍自行生效"`.
     - `§13.9` (line 1030): `"控制通訊異常與 VPN 資料路徑異常分開回報，不得混為單一 offline。"`.
     - `§19 AT13` (line 1619) & `§19 AT29` (line 1635): `"AT29 | Control API 暫時中斷 | 既有有效租約內節點可服務；租約過期後停止新轉發；恢復後自動補齊狀態與用量"`.
     - **[Decision]** `docs/handover/README.md` (`D-140`, `D-142`) and **[Code]** `node/agent/serving.go:238-239`: `"The budget constrains network requests only. Local persistence and cleanup use the caller context, so a control outage is not reported as disk failure."`
2. **Does any spec text or decision make NetBird management connectivity a safety precondition for forwarding?**
   - For `F20-01`, NetBird management was **not** down; only `fvpn-control` was stopped. No spec question or owner decision is needed to fix `F20-01`.
   - For completeness on Q4's second clause: if a future test separately stops the `netbird` management container while existing WireGuard peer tunnels stay up, `§17.1` (line 1501) states `"NetBird Management / Signal 暫時不可用 | 新 peer 協調與狀態變更受影響；部分既有直連可能延續；系統需明確呈現 degraded"`, whereas `judgeProbe` (`node/agent/probe.go:86`) currently treats `!st.Management.Connected` as `(DataHealthFailed, HealthCheckStatusFailed)` (`safety_check_failed`). Because `netbird` stayed up in `F20-01`, that behavior did not cause `F20-01`, and we note it under Routing only as a separate future design note if AT29 later tests a `netbird` management outage.

---

### Q5. Across the run

1. **Is `F20-01` related to `r27`'s `H-1` (a missing `fvmb0` read as a foreign filter)? Did `fvmb0` or the member path change state here?**
   - **No.** `F20-01` is completely unrelated to `r27`'s `H-1` (`TC_FOREIGN_FILTER` on `fvmb0`).
   - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/agent.log.txt:295-355` vs `agent.log.txt:364`: During `wholeNodeExpiry` (`00:25:57Z`–`00:27:06Z`), `tc_readback` was clean and `qos` / `source_binding` checks passed (`[Record] expiry-failure-wire-extract.json:71-75`). The `TC_FOREIGN_FILTER` log entry at `00:27:48.478Z` (`agent.log.txt:364`) occurred 106 seconds later during the separate `foreignRedirect` check (`[Code] tests/e2e/at/isolation/driver.go.txt:1488-1520`), which intentionally injects a foreign `tc filter` (`pref 1`) on `fvmb0`.
2. **Did anything else in the run fail the same probe? Compare AT10 to AT12's control-loss steps, if any:**
   - **Yes — AT13's sibling check `"lease member expiry under control loss"` (`memberExpiry`, `tests/e2e/at/isolation/driver.go.txt:1241-1285`) failed the exact same probe for the exact same reason, and was masked as a pass (`M-1`):**
     - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/agent.log.txt:259-275` & `AT13.json:871-1103`: At `00:24:02Z`, `memberExpiry` stopped `control` and `worker` (`driver.go.txt:1241`). Five seconds later (`00:24:07.139Z`), `DesiredState` timed out `controlContext(ctx)` (`msg="desired state" err="control request ended"`), `refreshProbe` failed with `context.DeadlineExceeded`, and the agent wrote the whole-node `safety_check_failed` baseline at `00:24:07.144Z` and `00:24:07.172Z` (`agent.log.txt:261-264`).
     - Because `memberExpiry` (`driver.go.txt:1264-1281`) ran its 5 post-deadline probes of the expired finite member from `00:24:06Z` to `00:24:11Z` and checked only `nodeStillValid := time.Now().Before(bounds.Expires)` (`driver.go.txt:1276`) without checking that the node remained `serving 1` (or that unexpired sibling `a1` could still forward), `memberExpiry` passed even though `AT13.json:1098` recorded `"heartbeat": "...\nserving 0\ngeneration 31\nreason safety_check_failed"` and a baseline-only `forward_chain`!
   - **Why `AT10`–`AT12` and `generationHoldCut` did not hit it:**
     - `AT10`–`AT12` did not stop or pause the `control` container (`[Record] container-lifecycle-events.json:1-92`). Only `AT13` (`memberExpiry` at `00:24:02Z` and `wholeNodeExpiry` at `00:25:57Z`) stopped `control` and `worker`.
     - `generationHoldCut` (`[Code] tests/e2e/at/isolation/driver.go.txt:2878-2960`) would have paused `control` for 35 s with `docker pause` at line `2927`, which would have triggered the same 5 s `controlContext(ctx)` timeout -> `safety_check_failed`, but it returned `blocked` early at line `2922` before reaching `docker pause` (`[Record] generation-hold-control-cut.json:172-179`).
3. **The narrowest change, and whose:**
   - **Owner `agent-core` (`H-1`):** In `Engine.refreshProbe(ctx)` (`node/agent/engine.go:948`), pass `localContext(ctx)` (or `ctx`, which already carries the step's `localBudget` from `engine.go:671-673` while `netbirdProber.Probe` bounds its own CLI read and dial by `probeTimeout = 2s` / `req.Timeout`) instead of `controlContext(ctx)`.
   - **Owner `isolation` (`M-1`):** In `memberExpiry` (`tests/e2e/at/isolation/driver.go.txt:1274-1281`), require that the node remains serving (`heartbeat` `serving 1` and/or positive forwarding for an unexpired sibling member) through the 5 post-deadline finite-member probes so a whole-node baseline (`safety_check_failed` or otherwise) cannot satisfy a per-member expiry check.

---

## 3. Findings by Severity

### High

#### `H-1`: `Engine.refreshProbe` passes `controlContext(ctx)` to `Prober.Probe`, causing any 5-second control-plane HTTP timeout to fail `tunnel_probe` (`HEALTH_CHECK_FAILED`) and baseline the node before lease expiry

- **Where:** `node/agent/engine.go:948` (called from `Engine.evaluate` at `node/agent/attest.go:72, 76` during `Engine.Step` at `node/agent/engine.go:768`, after `e.flush(ctx)`, `e.syncDesired(ctx)`, and `e.syncLease(ctx)` at `node/agent/engine.go:736, 737, 749`).
- **Scenario:**
  1. A published node is serving member traffic under a valid signed `ServingLease` with monotonic deadline `D`.
  2. The central control API (`fvpn-control`) becomes unreachable such that HTTP requests block until the per-step control-plane pass budget expires (`ioCtx` with `controlPassTimeout() = min(5*time.Second, stale/3) = 5s`, `node/agent/engine.go:667-668`, `node/agent/serving.go:230-236`), rather than failing instantaneously.
  3. In `Engine.Step`, `syncDesired(ctx)` (`node/agent/sync.go:36`) or `syncLease(ctx)` (`node/agent/sync.go:116`) consumes the 5 s `controlContext(ctx)` budget and returns `nil` (`sync.go:43-50`) because the caller context (`ctx`) is still live.
  4. `evaluate(ctx)` calls `refreshProbe(ctx)`, which invokes `p.Probe(controlContext(ctx), ProbeRequest{...})` (`node/agent/engine.go:948`).
  5. Inside `netbirdProber.Probe` (`node/agent/probe.go:114-122, 136`), both `p.read(readCtx)` (`exec.CommandContext(readCtx, "netbird", "status", "--json").Output()`, `probe.go:190-191`) and `p.dialAddr(ctx, addr, timeout)` (`d.DialContext(ctx, "tcp", addr)`, `probe.go:198-204`) inherit the already-expired `controlContext(ctx)` (`context.DeadlineExceeded`) and fail immediately in `0ms`.
  6. `judgeProbe` (`node/agent/probe.go:86`) returns `(DataHealthFailed, HealthCheckStatusFailed)` with `detail="overlay_down"`. `Decide` (`node/agent/decide.go:246-268`) sets `Serve = false`, `Safety = failed`, `DataHealth = failed`, `Reasons = ["safety_check_failed"]`, `Checks[tunnel_probe] = {Status: "failed", Detail: "overlay_down", ErrorCode: "HEALTH_CHECK_FAILED"}`, and `enforce` (`node/agent/engine.go:1142-1153`) installs the fail-closed nftables baseline (`cause=safety_check_failed`) 5 seconds after control loss — violating `§16.3`, `§17.1`, `§13.9`, `AT13`, `AT29`, `D-140`, `D-142`, and `node/agent/serving.go:238-239`.
- **Proof:**
  - **[Proof test]** `TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry` in `node/agent/review_r28_test.go:53-114` (fails against base `9be409f8c1297dcdda8c9d12d4b192d99c52a190`):
    ```text
    === RUN   TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry
        review_r28_test.go:109: H-1: refreshProbe passed expired controlContext(ctx) (context deadline exceeded) to Prober.Probe after syncDesired timed out: serving=false safety=failed dataHealth=failed reasons=[safety_check_failed] probeDetail="overlay_down" probeCheckStatus=failed errors=[{Code:HEALTH_CHECK_FAILED Message:tunnel_probe ObservedAt:2026-11-04 01:17:19.776565599 +0000 UTC Stage:probe}] baselines=1->3 heartbeat={serving:false reason:safety_check_failed}
    --- FAIL: TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry (0.44s)
    ```
  - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/expiry-early-baseline.json` (`seq 1931`–`1935`), `expiry-failure-wire-extract.json` (`seq 83` & `seq 84`), and `agent.log.txt:295-300`.
- **Narrowest fix and owner:**
  - **Owner:** `agent-core`
  - **Fix:** In `node/agent/engine.go:948`, replace `controlContext(ctx)` with `localContext(ctx)` in `p.Probe(localContext(ctx), ProbeRequest{...})` so `tunnel_probe` runs under the step's local budget (with its own per-probe `probeTimeout = 2s` / `req.Timeout` in `node/agent/probe.go:114, 198`) and is never cancelled by a control-plane HTTP timeout.

---

### Medium

#### `M-1`: AT13 `memberExpiry` ("lease member expiry under control loss") does not check that the node remains serving during the 5 post-deadline member probes, allowing a whole-node `safety_check_failed` baseline to pass the per-member expiry check

- **Where:** `tests/e2e/at/isolation/driver.go.txt:1274-1281` (in `memberExpiry`, lines `1187-1285`).
- **Scenario:**
  1. `memberExpiry` stops `control` and `worker` at `driver.go.txt:1241` (`00:24:02Z`), waits until 2 s after the finite member's `effective_service_end` (`00:24:06Z`), and runs 5 TCP probes from the expired member (`00:24:06Z`–`00:24:11Z`, `driver.go.txt:1264-1271`).
  2. At `00:24:07.139Z` (during probe 0), `H-1` caused the agent to install a whole-node `safety_check_failed` baseline (`00:24:07.144Z` and `00:24:07.172Z`, `agent.log.txt:259-264`), dropping all member forwarding across the entire node and writing `serving 0` / `reason safety_check_failed` to `/run/vpn-node/heartbeat` (`AT13.json:1098`).
  3. At `driver.go.txt:1274-1278`, the comment states:
     ```go
     // All five member-denial probes must finish while the original node lease
     // is still valid, so whole-node expiry cannot supply this result.
     nodeStillValid := time.Now().Before(bounds.Expires)
     ```
     Because `nodeStillValid` only compares `time.Now()` against `bounds.Expires` (`00:24:48.724Z`) without checking that the node actually stayed serving (`serving 1` / non-baseline `forward` chain, or that an unexpired sibling member still forwarded) through the 5 probes, `memberExpiry` reported `passed: true` (`AT13.json:873`) even though whole-node `safety_check_failed` had torn down the entire node.
- **Proof:**
  - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/agent.log.txt:259-265`:
    ```text
    time=2026-11-04T00:24:07.139Z level=WARN msg="desired state" err="control request ended"
    time=2026-11-04T00:24:07.139Z level=WARN msg="lease renewal" err="control request ended"
    time=2026-11-04T00:24:07.144Z level=INFO msg="baseline write" writer=agent cause=safety_check_failed phase=begin mono_ns=75289594481690 generation=31
    time=2026-11-04T00:24:07.169Z level=INFO msg="baseline write" writer=agent cause=safety_check_failed phase=end mono_ns=75289619100852 succeeded=true
    ```
  - **[Record]** `docs/e2e/at/isolation/history/follow-up-20-run/AT13.json:871-1103`: `"name": "lease member expiry under control loss"`, `"status": "pass"`, `"passed": true`, while its own post-probe observation at lines `1097-1098` recorded a baseline-only `forward_chain` (`policy drop; ct state invalid drop ...`) and `"heartbeat": "...\nserving 0\ngeneration 31\nreason safety_check_failed"`.
- **Narrowest fix and owner:**
  - **Owner:** `isolation`
  - **Fix:** In `memberExpiry` (`tests/e2e/at/isolation/driver.go.txt:1274-1281`), in addition to `time.Now().Before(bounds.Expires)`, verify after the 5 finite-member denial probes that the node is still serving (`heartbeat` reports `serving 1` and the `forward` chain is not the fail-closed baseline, or an unexpired sibling member's `countedForward` succeeds) so whole-node baseline installation cannot pass the per-member expiry check.

---

## 4. Routing

- **`agent-core`:** Fix `H-1` in `node/agent/engine.go:948` by passing `localContext(ctx)` instead of `controlContext(ctx)` to `p.Probe(...)` in `Engine.refreshProbe`, and verify that `TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry` (`node/agent/review_r28_test.go:53`) passes alongside the existing `node/agent` suite.
- **`isolation`:** Fix `M-1` in `tests/e2e/at/isolation/driver.go.txt:1274-1281` (`memberExpiry`) so the check requires the node to remain in a serving state (`serving 1` / non-baseline `forward` chain) through the 5 post-deadline finite-member probes, then re-run AT13 (`memberExpiry` and `wholeNodeExpiry`) once `agent-core` lands `H-1`.
- **`devstack`:** No change required (`control` and `worker` are already properly separated from `netbird` in `deploy/devstack/compose.yaml:129-186, 212-279`, and stopping `control` accurately models a central control API outage).
- **`orchestrator`:** Route `H-1` to `agent-core` first, then `M-1` and the AT13 verification run to `isolation`; no spec decision from Ted is needed for `F20-01` because `§16.3`, `§17.1`, `§13.9`, `D-140`, and `D-142` already mandate that a `fvpn-control` outage must not fail `tunnel_probe` or withdraw serving before the signed lease deadline `D`. *(Optional future note for Ted if AT29 later tests stopping the `netbird` management container itself: should `judgeProbe` in `node/agent/probe.go:86` treat `st.Management.Connected == false` with a readable status and live member interfaces as `DataHealthDegraded` + `HealthCheckStatusPassed` per `§17.1` line 1501 rather than `HealthCheckStatusFailed`? Recommendation: keep `F20-01` scoped strictly to `H-1` now, and address `st.Management.Connected` only if/when a `netbird` management-stop test is added.)*

---

## 5. What Held

1. **Separation of `controlContext(ctx)` and `localContext(ctx)` and healthy `netbirdProber.Probe` execution when given `localContext(ctx)`:**
   - **Proof test:** `TestR28_WhatHeld_RefreshProbeAndNetbirdProberContextPropagation` (`node/agent/review_r28_test.go:118-155`).
   - Proves that when `ioCtx` (`controlBudgetKey{}`) has timed out with `context.DeadlineExceeded`, `localContext(ctx)` and the parent step context remain live (`Err() == nil`), and `netbirdProber.Probe(localContext(ctx), ...)` succeeds (`Check: passed`, `Health: healthy`, `Detail: "overlay_up"`) for both CLI status reads and egress TCP probes.
2. **Continuous serving across control-plane failures that do not exhaust `controlContext(ctx)`, and exact fail-closed transition to `lease_expired` at `D`:**
   - **Proof test:** `TestR28_WhatHeld_FastControlRefusalPreservesServingUntilDeadline` (`node/agent/review_r28_test.go:157-232`).
   - Proves that when control-plane requests fail fast (leaving `controlContext(ctx)` unexpired when `refreshProbe` runs), `Engine.Step` keeps `serving == true`, `heartbeat.Serving == true` (`Reason: "serving"`), and `baselines` unchanged at `D - 46s` and `D - 4s`, and transitions at `D` to `serving == false`, `Safety: passed`, `DataHealth: healthy`, `Reasons: ["lease_expired"]`, `Checks[lease]: {Status: failed, ErrorCode: LEASE_EXPIRED}`, and `Checks[tunnel_probe]: {Status: passed}`.
3. **Reason ordering between `safety_check_failed` and `lease_expired` before and after `D` (`Decide` and `Engine.baselineReason`):**
   - **Proof test:** `TestR28_WhatHeld_PostDeadlineReasonOrderingInDecideAndBaseline` (`node/agent/review_r28_test.go:234-283`).
   - Proves why `agent.log.txt` recorded `cause=safety_check_failed` on every step from `00:26:02.373Z` to `00:26:43.299Z` (before `D`) and switched to `cause=lease_expired` at `00:26:48.301Z` (after `D`): `Decide` appends `lease_expired` (`node/agent/decide.go:201`) ahead of `safety_check_failed` (`node/agent/decide.go:248`), and `e.policyExpires` was already cleared to `0` at the first `!d.Serve` step (`decide.go:267`, `engine.go:1074, 1145`), so `e.baselineReason()` (`node/agent/measurement.go:86-89`) returns `Reasons[0]`.
4. **Devstack container isolation between Freedom VPN's control plane and NetBird's overlay services:**
   - **Trace:** `deploy/devstack/compose.yaml:129-186, 212-279` and `docs/e2e/at/isolation/history/follow-up-20-run/container-lifecycle-events.json:93-128`.
   - Stopping `fvpn-w4-isolation-control` and `fvpn-w4-isolation-worker` left `netbird` (`172.28.31.20`) and `gateway` running and did not disturb the member WireGuard tunnel (`fvnb0`), which continued forwarding data-plane probe packets every ~15 ms until the agent's own `nft -f` baseline write at `00:26:02.373Z` (`deadline-verdict.json`).

---

## 6. Not Examined

- Live Docker / `netlab` container execution (prohibited by `.agent/BRIEF.md`: *"No lab, no stack and no Docker in this review"*).
- End-to-end NetBird daemon behavior when the `netbird` management container (`172.28.31.20`) itself is stopped or paused (not exercised by AT13 or `F20-01`).
- Central `apps/control`, `apps/worker`, and `apps/portal` packages (outside owned paths and not involved in node-side `tunnel_probe` evaluation during control loss).

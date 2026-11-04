# Scheduled isolation job diagnosis — Follow-up 18, 2026-11-03

## 1. Scope and inputs

Scheduled `at-isolation` workflow run [41640196805](https://github.com/TED-H-com/Freedom-VPN/actions/runs/41640196805) ran on `main` `487c698e6602dd721391cfdb0e5205170e114000` on 2026-11-03 (`12:58`–`13:23` UTC, `run_id` `775bc692-e6ef-4490-8056-ba6cfa818b69`), the first scheduled execution after Follow-up 17 (#404) enabled blocked-row reason/owner logging and the 7-day records-only artifact in `.agent/nightly-records/`.

Because the GitHub Actions artifact expires on 2026-11-10, all log lines and JSON record excerpts relied on for this diagnosis are copied inline below using repository-relative paths. No workflow was triggered, re-run, or dispatched.

In `41640196805`, `.agent/nightly-records/scheduled-job.json` recorded 28 rows:
- 24 rows `pass` (`mapping: "pass"`);
- 2 rows `blocked` on D-144's expected-blocked list (`mapping: "expected blocked"`):
  - `AT11` / `IPv6 platform matrix`;
  - `AT13` / `continuous crash/NetBird/reboot/updater transitions`;
- 2 unlisted rows `blocked` (`mapping: "unexpected"`), failing the scheduled job:
  1. `AT13` / `short whole-node lease expiry` (`positive=false`, `owner=w3-devstack-e2e`, `reason="admin login failed before fleet-config write; pre-enrollment fleet audit missing"`);
  2. `AT10` / `foreign ingress redirect containment and D-096 recovery` (`positive=true`, `owner=w2-agent-core / w1-netlab`, `reason="current accepted observation and drained retry queue not established after control restoration; no injection"`).

Relevant job log lines from `41640196805` (with runner home prefixes stripped to repository-relative paths):

```text
2026-11-03T13:19:22.5917021Z     isolation_owned_test.go:216: AT13 lease member expiry under control loss: pass (positive=true) owner=w2-agent-core reason=five new TCP requests after finite signed member deadline; control and worker stopped; admission bytes frozen=true; route retained throughout=true; node lease has a later deadline
2026-11-03T13:19:22.9982753Z     isolation_owned_test.go:216: AT13 short whole-node lease expiry: blocked (positive=false) owner=w3-devstack-e2e reason=admin login failed before fleet-config write; pre-enrollment fleet audit missing
2026-11-03T13:19:57.4376555Z     isolation_owned_test.go:216: AT10 foreign ingress redirect containment and D-096 recovery: blocked (positive=true) owner=w2-agent-core / w1-netlab reason=current accepted observation and drained retry queue not established after control restoration; no injection
2026-11-03T13:22:09.0168765Z Retained 30 operation snapshots; final read after the assembled central report export
2026-11-03T13:22:11.3431884Z     isolation_owned_test.go:2583: review: exit status 1 Traceback (most recent call last):
  File "tests/e2e/at/isolation/review_followup8.py", line 87, in <module>
    dirty_tc = candidate_rejections(transactions, read_operations(root), initial_gen,
  File "tests/e2e/at/isolation/candidate_evidence.py", line 21, in candidate_rejections
    lo, hi = wall_ns(start), wall_ns(end)
  File "tests/e2e/at/isolation/candidate_evidence.py", line 11, in wall_ns
    raise ValueError('UTC nanosecond timestamp required')
ValueError: UTC nanosecond timestamp required
2026-11-03T13:22:11.6553148Z     isolation_owned_test.go:204: AT10: blocked
2026-11-03T13:22:11.6553762Z     isolation_owned_test.go:204: AT13: blocked
--- FAIL: TestIsolationAssembled (1137.25s)
```

---

## 2. Row 1 diagnosis: `AT13 short whole-node lease expiry`

### 2.1 Evidence chain (`41640196805`, `run_id` `775bc692-e6ef-4490-8056-ba6cfa818b69`)

1. **Pre-enrollment fleet configuration write succeeded at startup (`13:03:15Z`).**
   `.agent/nightly-records/fleet-before-enrollment.json`:
   ```json
   {
     "detail": "200 etag=\"2\" body={\"default_max_offline_lease_seconds\":86400,\"max_offline_lease_seconds\":60,\"resource_version\":\"2\",\"updated_at\":\"2026-11-03T13:03:15.480666Z\"}",
     "offline_seconds": 60,
     "request_id": "01a1a181-b816-7bfc-a507-09ad7d2aab97",
     "status": 200
   }
   ```
   In `tests/e2e/at/isolation/driver.go.txt` (`prepare()`, lines `262`–`269` on `487c698e`), `x.putFleet(60, "TEST ONLY D-087 fixed offline window before enrollment")` returned `status == 200`, recorded `x.fleetRequestID = "01a1a181-b816-7bfc-a507-09ad7d2aab97"`, and wrote `fleet-before-enrollment.json`.

2. **Preceding row `lease member expiry under control loss` (`deadline()`) stopped `fvpn-w4-isolation-control` and `fvpn-w4-isolation-worker`, and restarted them in a `defer` without waiting for control readiness.**
   In `tests/e2e/at/isolation/driver.go.txt` (`deadline()`, lines `1122`–`1235` on `487c698e`):
   - Line `1123` registered `defer x.wholeNodeExpiry()` first (so in LIFO defer order, `wholeNodeExpiry()` runs last when `deadline()` returns).
   - Line `1195` stopped control and worker (`exec.Command("docker", "stop", "--time", "2", r.lab.control, r.lab.prefix+"-worker").Run()`) at `13:18:54Z`.
   - Line `1197` registered `defer func() { _ = exec.Command("docker", "start", r.lab.control, r.lab.prefix+"-worker").Run() }()` **without calling `x.waitControl()`** (unlike `wholeNodeExpiry()` at line `1262` and `generationHoldCut()` at lines `2857`/`2894`).
   - In `.agent/nightly-records/AT13.json` (line `1041`), `deadline()` completed its last post-deadline probe at `2026-11-03T13:19:22.526098782Z` and logged `AT13 lease member expiry under control loss: pass` at `2026-11-03T13:19:22.5917021Z`.
   - Returning from `deadline()` executed `docker start fvpn-w4-isolation-control fvpn-w4-isolation-worker` and immediately invoked `wholeNodeExpiry()`.

3. **Where admin login failed, and why on the GitHub runner vs. local assembled runs.**
   At the entry of `wholeNodeExpiry()` (`tests/e2e/at/isolation/driver.go.txt`, lines `1272`–`1277` on `487c698e`):
   ```go
   admin, err := x.r.lab.login(x.ctx, "e2e-admin", "admin")
   if err != nil {
       c.Actual = "admin login failed before fleet-config write"
       c.Owner = "w3-devstack-e2e"
       return
   }
   ```
   `x.r.lab.login` (`tests/e2e/at/isolation/driver.go.txt:210` on `487c698e`) calls `idplogin.PortalLogin(ctx, l.urls.Gateway, l.urls.IDP, sub, role)`, which sends `GET /auth/login` through `fvpn-w4-isolation-gateway` (`nginx/1.30.0`) to upstream `fvpn-control:8080`.
   Because `docker start` returns as soon as the container process is spawned — before `fvpn-control` finishes starting its HTTP listener inside the container — `x.r.lab.login` hit `nginx` while `fvpn-control:8080` was still unreachable, receiving HTTP `502 Bad Gateway` and returning an error in **`406.6 ms`** (`13:19:22.5917021Z` -> `13:19:22.9982753Z`).

   The agent and receiver records prove when `fvpn-control` actually became reachable after that `docker start`:
   - `.agent/nightly-records/agent.log.txt` (lines `269`–`273`):
     ```text
     time=2026-11-03T13:19:24.800Z level=WARN msg="desired state" err="control request ended"
     time=2026-11-03T13:19:24.800Z level=WARN msg=lease err="control request ended"
     time=2026-11-03T13:19:29.881Z level=WARN msg="desired state" err="control request ended"
     time=2026-11-03T13:19:29.881Z level=WARN msg=lease err="control request ended"
     ```
   - `.agent/nightly-records/receiver-capture.json` (lines `165002`–`165015`): the first HTTP response served by `fvpn-control` after `docker start` was `GET /api/v1/agent/desired-state` (`status: 200`) at `response_at: "2026-11-03T13:19:30.47655856Z"`, followed by `POST /api/v1/agent/lease-renewals` (`status: 200`) at `received_at: "2026-11-03T13:19:30.480533644Z"` (`.agent/nightly-records/lease-history.json`, line `2507`: `issued_at: "2026-11-03T13:19:30.485959809Z"`).
   - Thus on the GitHub runner, `fvpn-control` took **~7.9 s** after `deadline()` returned (`13:19:22.59Z` -> `13:19:30.47Z`) to begin serving requests.
   - By contrast, in Follow-up 17's local assembled run (`0ec14b39-467e-4a37-a239-1a33ef12eb71`, archived at `docs/e2e/at/isolation/history/follow-up-17/AT13.json`, lines `1041` and `1061`), `deadline()` completed its last probe at `2026-11-03T06:33:53.438240828Z` and `fvpn-control` came up fast enough on the local host after `docker start` for `x.r.lab.login` to succeed immediately, so `wholeNodeExpiry()` proceeded to its positive control (`06:35:19Z`), stopped control for the 60 s whole-node expiry window (`06:35:09.37Z`–`06:36:09.37Z`), and passed at `06:36:31Z`.

4. **Why `"pre-enrollment fleet audit missing"` is a direct consequence of the same unawaited `docker start` and failed login (not a separate cause).**
   In `tests/e2e/at/isolation/driver.go.txt` (lines `1256`–`1271` on `487c698e`):
   ```go
   c := iso.Check{Name: "short whole-node lease expiry", Status: iso.StatusBlocked, Owner: "w2-agent-core", Actual: "not measured"}
   var stopped bool
   defer func() {
       if stopped {
           _ = exec.Command("docker", "start", x.r.lab.control, x.r.lab.prefix+"-worker").Run()
       }
       if stopped && !x.waitControl() {
           c.Status = iso.StatusBlocked
           c.Actual += "; control restart incomplete"
       }
       if !x.fleetAudits(&c, x.fleetRequestID) {
           c.Status = iso.StatusBlocked
           c.Actual += "; pre-enrollment fleet audit missing"
       }
       x.add("AT13", c)
   }()
   ```
   Because `x.r.lab.login` failed at line `1273` before `stopped = true` (line `1339`), the `defer` skipped `if stopped && !x.waitControl()` at line `1262` and immediately called `x.fleetAudits(&c, x.fleetRequestID)` (`lines 1475–1491` on `487c698e`), which issued `GET /api/v1/admin/audit-events?request_id=01a1a181-b816-7bfc-a507-09ad7d2aab97&limit=5` through `fvpn-w4-isolation-gateway` using the pre-outage `x.admin` session and checked `err != nil || resp.Status != 200 || !strings.Contains(string(resp.Body), "fleet_config.update")`. Because `fvpn-control` was still booting (`~400 ms` after `docker start`), `nginx` returned `502 Bad Gateway`, recorded in `.agent/nightly-records/AT13.json` (lines `1070`–`1072`) and `.agent/nightly-records/boundary-review.json` (lines `23`–`25`):
   ```json
   {
     "name": "fleet_audit_01a1a181-b816-7bfc-a507-09ad7d2aab97",
     "detail": "502 <html>\r\n<head><title>502 Bad Gateway</title></head>\r\n<body>\r\n<center><h1>502 Bad Gateway</h1></center>\r\n<hr><center>nginx/1.30.0</center>\r\n</body>\r\n</html>"
   }
   ```
   Because `x.fleetAudits` returned `false` on `resp.Status == 502`, line `1266` (`if !x.fleetAudits(&c, x.fleetRequestID)`) appended `"; pre-enrollment fleet audit missing"` to `c.Actual` (`"admin login failed before fleet-config write"`). Both halves of the reason string were caused by `GET` requests hitting `fvpn-w4-isolation-gateway` (`502 Bad Gateway`) ~400 ms after `deadline()` ran `docker start` without waiting for `fvpn-control`.

---

## 3. Row 2 diagnosis: `AT10 foreign ingress redirect containment and D-096 recovery`

### 3.1 Evidence chain (`41640196805`, `run_id` `775bc692-e6ef-4490-8056-ba6cfa818b69`)

1. **What the harness waited for after control restoration, and for how long: 0 seconds.**
   - In `tests/e2e/at/isolation/driver.go.txt` on `487c698e`, `x.waitControl()` (`lines 1436–1473`) is the only function that writes `control-observation-readiness.json`.
   - In `41640196805`, `x.waitControl()` was **never called**:
     - `deadline()` restarted `control` and `worker` in its `defer` without calling `x.waitControl()`.
     - `wholeNodeExpiry()` returned at line `1276` before `stopped = true` (line `1339`), so its `defer` (`if stopped && !x.waitControl()`, line `1262`) did not call `x.waitControl()`.
     - Next, `generationHoldCut()` (`lines 2746–2898` on `487c698e`) ran its positive control probe at `13:19:32Z` (`generation-hold-control-cut.json`, line `165`: `serving 1`, `generation 32`), polled 15 s (`13:19:32Z`–`13:19:47Z`) for an open generation hold, found none, and returned `"not exercised"` at line `2848` before pausing control:
       ```json
       {
         "reason": "no new accepted open hold in the single 15-second attempt; no transport seam or retry added",
         "run_id": "775bc692-e6ef-4490-8056-ba6cfa818b69",
         "status": "not exercised"
       }
       ```
   - Consequently, `.agent/nightly-records/control-observation-readiness.json` was **never created** in `41640196805`.
   - When `foreignRedirect()` (`lines 2429–2581` on `487c698e`) ran next:
     - Its positive control probe (`x.proof(x.a1)`) succeeded between `13:19:47Z` and `13:19:57.297Z` (`.agent/nightly-records/AT10.json`, lines `1181`–`1190` and `1357`: `controlled_origin_http_200_and_counted: "true"`, `serving 1`, `generation 33`).
     - Then at lines `2432`–`2439`, `foreignRedirect()` performed a one-shot file read without calling `x.waitControl()`:
       ```go
       var readiness struct {
           Ready bool `json:"ready"`
       }
       rawReadiness, _ := os.ReadFile(filepath.Join(x.dir, "control-observation-readiness.json"))
       if json.Unmarshal(rawReadiness, &readiness) != nil || !readiness.Ready {
           c.Actual = "current accepted observation and drained retry queue not established after control restoration; no injection"
           return
       }
       ```
     - Because `control-observation-readiness.json` did not exist (`os.ErrNotExist`), `foreignRedirect()` waited **0 s** and immediately returned `blocked` at `13:19:57.4376555Z`.

2. **What the agent's records show for the accepted observation and the retry queue.**
   `.agent/nightly-records/receiver-capture.json` (`lines 165000–170655`) and `.agent/nightly-records/lease-history.json` (`lines 2506–2562`) show that the agent had already recovered, drained its outage retry queue, and established current accepted observations **17.21 seconds before** `foreignRedirect()` checked:
   - At `13:19:30.485959809Z`, the agent renewed its lease at `config_generation: "32"` (`lease_id: "bafa767d-73d7-4ac6-baa6-7b4679c27ab7"`).
   - At `13:19:32.991225774Z`, the agent fetched desired state `config_generation: "33"` (triggered by `generationHoldCut()`'s capacity patch) and renewed its lease at `13:19:33.160249011Z` (`lease_id: "6d9f78de-f748-48ff-b68e-3c96f802664d"`, `config_generation: "33"`).
   - Between `13:19:37.996396146Z` and `13:19:40.0081796Z`, the agent drained all 6 queued observation reports (`01a1a190-0d14-7acb-abe5-9af248958bf1` through `01a1a190-b7b7-7a4a-a893-a01781a50138`, all accepted with HTTP `202`).
   - At `13:19:40.222691621Z`, central accepted the first **current** post-restoration observation report (`01a1a190-bebb-7497-989f-36bfe8b78a8d`, `observed_at: "2026-11-03T13:19:40.218635399Z"`, `applied_config_generation: "33"`, `member_forwarding_enabled: true`, `safety_status: "passed"`), followed by 9 more current accepted reports every ~2 s through `13:19:57.239735093Z` (`01a1a191-0133-749d-b7ba-8023b690ddee`, `observed_at: "2026-11-03T13:19:57.236039403Z"`):
     ```json
     {
       "path": "/api/v1/agent/observations",
       "received_at": "2026-11-03T13:19:40.222691621Z",
       "response": {
         "accepted": true,
         "received_at": "2026-11-03T13:19:40.227146Z",
         "report_id": "01a1a190-bebb-7497-989f-36bfe8b78a8d"
       },
       "response_at": "2026-11-03T13:19:40.232429962Z",
       "status": 202
     }
     ```

3. **Classification: harness defect, plus a secondary review-script defect when `foreignRedirect()` does not inject.**
   - This is a **harness defect** in `tests/e2e/at/isolation/driver.go.txt`, not runner slowness or an agent/netlab defect: the agent's observation pipeline and retry queue were completely healthy and drained by `13:19:40.23Z`, 17.2 s before `foreignRedirect()` ran at `13:19:57.43Z`, but `foreignRedirect()` only checked whether an earlier step had written `control-observation-readiness.json` and never invoked `x.waitControl()` when that file was absent or stale.
   - In addition, when `foreignRedirect()` returned early with `no injection`, `finish()` still invoked `tests/e2e/at/isolation/review_followup8.py` (`driver.go.txt:183, 2583` on `487c698e`), which read `foreign-redirect.json` (`trace = read('foreign-redirect.json', {})`), looked up `injection = first('inject clsact mirred')` and `removal = first('remove all injected objects')`, defaulted `start, end = injection.get('start', '9999'), removal.get('end', '9999')` (`line 68`), and passed `first('publish dirty newer generation').get('at', start)` (`'9999'`) and `removal.get('start', end)` (`'9999'`) into `candidate_rejections(...)` -> `candidate_evidence.py:wall_ns('9999')` (`ValueError: UTC nanosecond timestamp required`, `run.log:2060–2069`).

---

## 4. Comparison with the previous night's run (`41456092596` on `a6d56cff`, 2026-11-02)

Job log lines from [41456092596](https://github.com/TED-H-com/Freedom-VPN/actions/runs/41456092596) (2026-11-02, `main` `a6d56cff`, before Follow-up 17 added reason logging or artifact upload; runner home prefixes stripped):

```text
2026-11-02T14:02:46.6471421Z     isolation_owned_test.go:215: AT13 node restart with durable identity: pass (positive=true)
2026-11-02T14:02:46.6704983Z     isolation_owned_test.go:215: AT13 continuous crash/NetBird/reboot/updater transitions: blocked (positive=true)
2026-11-02T14:06:09.4244519Z     isolation_owned_test.go:215: AT13 lease member expiry under control loss: pass (positive=true)
2026-11-02T14:06:09.7621571Z     isolation_owned_test.go:215: AT13 short whole-node lease expiry: blocked (positive=false)
2026-11-02T14:07:59.6065268Z     isolation_owned_test.go:215: AT10 foreign ingress redirect containment and D-096 recovery: blocked (positive=true)
2026-11-02T14:08:21.7217737Z Retained 27 operation snapshots; final read after the assembled central report export
2026-11-02T14:08:23.0033258Z     isolation_owned_test.go:203: AT11: blocked
2026-11-02T14:08:23.0033698Z     isolation_owned_test.go:203: AT13: blocked
--- FAIL: TestIsolationAssembled (949.10s)
+ python3 -B tests/e2e/at/isolation/review_followup10.py docs/e2e/at/isolation
Current expiry measurement unavailable; refusing inherited packet evidence
```

### What can and cannot be told from `41456092596`:

1. **Row 1 (`AT13 short whole-node lease expiry`):**
   - **What cannot be told from the status-only line alone:** Because `a6d56cff` logged only `AT13 short whole-node lease expiry: blocked (positive=false)` without `c.Actual` or uploaded JSON records, the status line by itself could not distinguish between the pre-`proof` early returns in `wholeNodeExpiry()` (`lab.login` at line `1273`, `member route via fvnb0` at line `1285`, lease renewal wait at line `1313`, or `docker stop` at line `1334` on `487c698e`), other than ruling out any return after `x.proof()` (`positive=false`).
   - **Why `41456092596` is completely consistent with the exact same cause as `41640196805`:**
     - In `41456092596`, `AT13 short whole-node lease expiry: blocked (positive=false)` was logged at `14:06:09.7621571Z`, **`337.7 ms`** after `AT13 lease member expiry under control loss: pass` (`14:06:09.4244519Z`).
     - In `41640196805`, the same gap was **`406.6 ms`** (`13:19:22.5917021Z` -> `13:19:22.9982753Z`).
     - The lease-renewal wait loop (`driver.go.txt:1297` on `487c698e`) has a 90 s budget (`context.WithTimeout(x.ctx, 90*time.Second)`) and polls every 1 s; the only path that exits in `337`–`407 ms` immediately after `deadline()`'s `docker start` is the immediate gateway `502 Bad Gateway` on `x.r.lab.login` (followed by `x.fleetAudits` in the `defer`).
     - Furthermore, `review_followup10.py` on `41456092596` printed `Current expiry measurement unavailable; refusing inherited packet evidence`, confirming `wholeNodeExpiry()` returned before writing `whole-node-packets.json`.

2. **Row 2 (`AT10 foreign ingress redirect containment and D-096 recovery`):**
   - **Crucial difference between `41456092596` (`a6d56cff`) and `41640196805` (`487c698e`):**
     - In `41456092596`, the final `TestIsolationAssembled` summary at `14:08:23.003Z` (`isolation_owned_test.go:203`) reported **only `AT11: blocked` and `AT13: blocked`** — `AT10` **passed** in `41456092596`.
     - Why did `41456092596` log `AT10 foreign ingress redirect containment and D-096 recovery: blocked (positive=true)` at `14:07:59.6065268Z` (`109.8 s` after `AT13 short whole-node lease expiry`) and then finish with `AT10: pass`?
     - Because `foreignRedirect()` initializes `c.Status = iso.StatusBlocked` (`driver.go.txt:2430` on `487c698e`) with `c.Actual = "foreign tc redirect contained and removed; D-096 recovery evidence recorded; final pass requires six-assertion review after receiver export"` (`line 2560`), logs `c` via `defer func() { x.add("AT10", c) }()` (`line 2431`), and then `finish()` runs `x.reviewFollowup8()` (`driver.go.txt:183, 2583–2611` on `487c698e`), which evaluates `recovery-verdict.json` and upgrades `AT10` to `pass`.
     - Why did `foreignRedirect()` execute its ~110 s injection and D-096 recovery in `41456092596` (`a6d56cff`) even though `wholeNodeExpiry()` had failed to run `x.waitControl()`? Because on `a6d56cff`, the scheduled workflow wrote directly into `docs/e2e/at/isolation/`, where `docs/e2e/at/isolation/control-observation-readiness.json` (`"ready": true`, committed in Follow-up 16) already existed on disk from `git checkout`. `foreignRedirect()` read that checkout-inherited file at line `2435`, saw `"ready": true`, and proceeded to run the foreign-redirect injection and recovery on the GitHub runner.
     - **That pass is not a qualified result.** Its readiness precondition came from a file in the checkout (`docs/e2e/at/isolation/control-observation-readiness.json`), committed by follow-up 16, not from that run. **No acceptance status changes.** No record on `main` came from that run.
     - When Follow-up 17 (`487c698e`) isolated scheduled runs into a clean `.agent/nightly-records` directory (`FVPN_ISOLATION_OUT=.agent/nightly-records`) so no committed `docs/e2e/at/isolation/` file could be inherited, the missing `x.waitControl()` call after `deadline()` exposed the latent defect: `.agent/nightly-records/control-observation-readiness.json` was never written, causing `foreignRedirect()` to abort before injection in `41640196805`.

---

## 5. Outcomes and fixes

Both unlisted rows are **Outcome (a): harness defects in owned paths (`tests/e2e/at/isolation/`)**. Neither row is a capability limitation of GitHub's runner, and neither row is added to `tests/e2e/at/isolation/expected-blocked.json`.

### 5.1 Fixes in `tests/e2e/at/isolation/`

1. **`tests/e2e/at/isolation/scheduled.go` (`lines 32–89`):**
   - Added `ControlReadinessSample`, `SampleControlReadiness(startedAt, desiredGeneration, s)`, and `ControlObservationReady(dir, runID, afterControlStart, wait)`:
     - `SampleControlReadiness` requires:
       - non-zero `startedAt` (`!startedAt.IsZero()`) and non-empty `desiredGeneration` (`desiredGeneration != ""`);
       - admin login (`s.LoginOK`);
       - `GET /api/v1/admin/fleet-config` returns HTTP 200 (`s.FleetConfigStatus == 200`);
       - an available observation queue with 0 pending (`s.QueueAvailable && s.QueuePending == 0`);
       - an accepted observation strictly after the wait's start (`s.ObservedAt.After(startedAt)`), whose applied generation equals the desired generation (`s.AppliedGeneration == desiredGeneration`), with member forwarding enabled (`s.MemberForwardingEnabled`).
       This is `main`'s `waitControl` predicate (`driver.go.txt:1443–1470` on `487c698e`) unchanged, plus one new requirement: the desired generation is non-empty (`desiredGeneration != ""`). Neither `SampleControlReadiness` (`scheduled.go`) nor `waitControl` (`driver.go.txt`) checks `safety_status`, `forwarding_block_reasons`, `kernel_ruleset`, or `readback`.
     - `ControlObservationReady` reads `control-observation-readiness.json` in `dir` and requires `ready == true`, rejects a different `run_id` only when both `runID` and the record's `run_id` are non-empty (`runID != "" && v.RunID != "" && v.RunID != runID`), and requires that the record's `started_at` is not before `afterControlStart` (`!afterControlStart.IsZero() && v.StartedAt.Before(afterControlStart)` is rejected). If absent, unready, from another `run_id`, or started before `afterControlStart`, it invokes `wait()` (`x.waitControl`) when non-nil and re-reads the persisted record.

2. **`tests/e2e/at/isolation/driver.go.txt`:**
   - In `deadline()` (`lines 1198–1202`), the deferred `docker start` now sets `x.lastControlStart = time.Now().UTC()` and calls `_ = x.waitControl()` before `defer x.wholeNodeExpiry()` executes.
   - In `wholeNodeExpiry()` (`lines 1264–1282`), `iso.ControlObservationReady(x.dir, x.records["AT13"].RunID, x.lastControlStart, x.waitControl)` is verified at entry before `x.r.lab.login(x.ctx, "e2e-admin", "admin")`, and in `wholeNodeExpiry()`'s `defer`, `if (stopped || !iso.ControlObservationReady(x.dir, x.records["AT13"].RunID, x.lastControlStart, nil)) && !x.waitControl()` ensures `x.waitControl()` runs before `x.fleetAudits(&c, x.fleetRequestID)` even when `wholeNodeExpiry()` returns before setting `stopped = true`.
   - In `waitControl()` (`lines 1447–1496`), `start := time.Now().UTC()` updates `x.lastControlStart = start` on entry, `iso.SampleControlReadiness` evaluates each poll sample (`lines 1484–1492`), and `"run_id": x.records["AT13"].RunID` is written into `control-observation-readiness.json` (`line 1494`).
   - In `foreignRedirect()` (`line 2455`), the one-shot `os.ReadFile` is replaced with `iso.ControlObservationReady(x.dir, x.records["AT10"].RunID, x.lastControlStart, x.waitControl)`, so if an earlier step skipped `waitControl()` or restarted control after the record's `started_at`, `foreignRedirect()` waits for readiness instead of blocking immediately, and rejects an inherited file from a different non-empty `run_id`.
   - In `generationHoldCut()` (`lines 2875, 2911`), `x.lastControlStart` is updated when `docker unpause` restores control and worker.

3. **`tests/e2e/at/isolation/review_followup8.py` (`lines 87–93, 122–130`):**
   - `review_followup8.py` computes `dirty_start = first('publish dirty newer generation').get('at') or injection.get('start')` and `dirty_end = removal.get('start') or removal.get('end')`, and skips the `candidate_rejections(...)` join when the dirty window's start or end is missing (`dirty_tc = candidate_rejections(transactions, read_operations(root), initial_gen, dirty_start, dirty_end) if dirty_start and dirty_end else []`), writing `accepted_candidates: []` to `candidate-rejection-join.json`. The unchanged six assertions in `checks` then write `passed: false` and `status: "blocked"` to `recovery-verdict.json`, and the script exits `0` instead of passing `'9999'` to `wall_ns()` and raising a Python traceback.

4. **Regression tests added:**
   - `TestAT13_AT10_Nightly41640196805ControlRestorationSequence` in `tests/e2e/at/isolation/scheduled_test.go` (`lines 42–195`): replays the `41640196805` timestamp sequence (`docker start` at `13:19:22.5917021Z`, gateway `502` on `GET /api/v1/admin/fleet-config`, queued outage report before `controlStart`, post-start report with `QueuePending: 1` at `13:19:38.423577389Z`, and first current drained report at `13:19:40.218635399Z`) and verifies that `SampleControlReadiness` and `ControlObservationReady` wait through the `502` / queue-drain window, reject missing readiness (`wait == nil`), reject foreign-`run_id` readiness (`"run_id": "83b457ef-ea16-490b-9923-1fea0df86b36"`, `"started_at": controlStart.Add(time.Second)`), reject pre-restart readiness (`"started_at": controlStart.Add(-time.Second)`), establish readiness via `waitFn` (`"started_at": controlStart`), and reuse the persisted record on subsequent checks without re-polling.
   - `test_unexecuted_foreign_redirect_writes_blocked_verdict_without_traceback` in `tests/e2e/at/isolation/nightly_test.py` (`lines 51–59`): runs `review_followup8.py` on a temporary directory with `receiver-capture.json` and no injection rows, and asserts `result.returncode == 0`, `verdict['passed'] == False`, and `verdict['status'] == 'blocked'`.

### 5.2 Local assembled verification (`9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`)

After archiving Follow-up 17's root snapshot (`0ec14b39-467e-4a37-a239-1a33ef12eb71`) byte-identically under `docs/e2e/at/isolation/history/follow-up-17/` per D-098, one local assembled verification run (`make -f mk/at-isolation.mk at-isolation`) was executed on `c248db966d6870182a7c87cf7e410d7524bf56ff` (`2026-11-03T14:05:18Z`–`2026-11-03T14:29:55Z`, `run_id` `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`, `source_dirty: false`):
- `deadline()` completed its last post-deadline probe at `2026-11-03T14:22:41.26199998Z` (`docs/e2e/at/isolation/AT13.json:1041`), restarted `control` and `worker`, and `waitControl()` established post-outage readiness before `wholeNodeExpiry()` called `lab.login`.
- **`AT13 short whole-node lease expiry`:** **`pass`** (`positive=true`, `docs/e2e/at/isolation/AT13.json:1054–1128`, `docs/e2e/at/isolation/boundary-review.json`, `docs/e2e/at/isolation/deadline-verdict.json`: `schedule_qualified: true`, `unchanged_evaluator_passed: true`, `164` positives, `70` drops, `last_positive_offset_ns: "-100551358"`, `first_drop_offset_ns: "-93753451"`, `fleet_audit_01a1a1bb-d9f3-7d51-b65f-2c57c2b9c17f`: `200`).
- **`AT10 foreign ingress redirect containment and D-096 recovery`:** **`pass`** (`positive=true`, `docs/e2e/at/isolation/AT10.json`, `docs/e2e/at/isolation/recovery-verdict.json`: `passed: true`, `status: "pass"`, 6/6 assertions passed).
- Only the two D-144 expected-blocked operator rows remained blocked (`AT11 IPv6 platform matrix` and `AT13 continuous crash/NetBird/reboot/updater transitions`).

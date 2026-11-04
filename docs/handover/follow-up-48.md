# Follow-up 48 — #478–#481; F24-01 closed, assist.py retry, F22-D146 answered, r32

Verified against main **b921c75f180beb6c0051d585a86d6651b2c8d71a**, through
**members 24 #481 / 884565c8dd973c2d9b16082556a7a3d5c77387ea**, on
**2026-11-04 EST**. Follow-up 47 (#479) ended at **#477 /
1c39a0e73825dbcf911dfd2bec6e75aed64c979b**. Exactly **four merges and four
final roadmap rows**: **#478, #479, #480, #481**; no unbriefed PR merged
inside the range.

**Running at briefing, status only:** **agent-core 53** (`w2-agent-core`),
launched with this cut to diagnose first and fix the `node/agent` test failures
observed in GitHub Actions `Node` workflow runs **41782791081, 41787173794,
41794009264, 41808885873, 41871089749 and 41885624816** (eight failures since
#447, of which two are review branches' own proofs and six fail tests that pass
on `main`, with the sixth added by the orchestrator), and to write **F17-01's
and F31-01's design notes** with no product change for those. A codex session
started it at **13:36 EST** and stopped at **13:38 EST** on its usage limit,
before any commit; it was relaunched at **13:40 EST** on agy
`gemini-4-argon-high`. No delivered or merged PR was present at the initial
read-only check (`w2-agent-core` had no PR). Later results are **after the
cutoff** and do not enter this cut. No date is promised.

**D-098 / D-109 / D-132:** `TestHandover_Followup48Claims` pins the four final
roadmap rows, verbatim PR excerpts, 23 frozen snapshots (nine #479 documents,
three producer dated notes and eleven `F24-*.json` records), 22 Go test
declarations on main, 22 frozen source readings, 62 archived artifacts, exact
JSON values decoded with `json.Decoder.UseNumber()`, review r32 provenance and
proof file, and all six open `Node` workflow failure runs. All lane-document
quotations use committed frozen copies. No claims read living isolation,
members, secops or usage evidence roots. Follow-up 47's rewritten inputs now
read frozen **#479 bytes** (`8a50c36c84628cce8b7c56e02d9123165d03ce84`); its
assertions and every earlier claims test (`41` functions
`TestHandover_Followup7Claims` through `TestHandover_Followup47Claims`) remain
unchanged.

## Handover 47 (#479): previous cut, actual inventory and session

[PR #479](https://github.com/TED-H-com/Freedom-VPN/pull/479) (merge
`8a50c36c84628cce8b7c56e02d9123165d03ce84`, roadmap row
`8f4374d3a6932d845317c994d6cd3de634cde37c`) records the **#474–#477 cut, up to
1c39a0e73825dbcf911dfd2bec6e75aed64c979b**. On main,
`TestHandover_Followup47Claims` has **17 subtests** (eleven direct `t.Run`
blocks and six claim groups in `followup47-claims.json`), **four final roadmap
rows and PR excerpts**, **13 byte-identical frozen inputs** (nine #474
documents and four producer documents), **20 Go test declarations on main**,
**20 frozen source readings**, **92 archived artifacts** and **38 exact JSON
values** decoded as `json.Number`. Its **40 earlier claims functions**
(`TestHandover_Followup7Claims` through `TestHandover_Followup46Claims`) stay
exact after reversing only the four input redirects in
`restoreFollowup47Assertions`. Its register contains **66 decisions** (D-163
added, D-162 updated, 64 byte-identical to #474) and **136 findings rows** (9
updated, 3 added, 124 byte-identical to #474), and
`docs/e2e/at/secops/scope-review.json` refreshed **238 digest values** across
ten owned paths. These are #479's verified counts, not this cut's changes.

The session, attributed to #479's body: **codex gpt-6-astra**, launched
**11:32 EST** on **2026-11-04** on #477's roadmap row **1c39a0e7**, delivered
**11:55 EST**; **9 commits and 45 owned files, +12,104/−309**. Final gates
waited about **four minutes** for host admission: three during the
orchestrator's isolation 25 reservation and one during a members run. Docs,
claims tests, frozen snapshots and digest values only; no reservation, Docker
or live measurement. Isolation 25 (#478, merged at `2026-11-04T16:47:51Z`) was
recorded as after-cutoff status only in
`tests/handover/testdata/history/follow-up-47/after-cutoff-status.json`.

## Isolation 25 (#478): D-098 archive, F24-01 closed and scheduled pass

[PR #478](https://github.com/TED-H-com/Freedom-VPN/pull/478) (merge
`014b59ab15288dc6bc053932104e26a8f3327e9e`, roadmap row
`df61fd68add1678729ef32ae1ef93719124035e3`) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-48/docs_e2e_at_isolation_followup25-2026-11-04.md.txt)
archive follow-up 24's run **df9dbad7-049d-4eaa-ab98-5490906cc3e9** (measured
source `56d814113913ddc6177d47d4a5160a65c93add25` from
`docs/e2e/at/isolation/history/follow-up-24-run/AT13.json` `build`): all **213
root, support and reload-stream files** are copied byte-identically under
`docs/e2e/at/isolation/history/follow-up-24/` with manifest
`docs/e2e/at/isolation/history/follow-up-24/archive.json` and verified alongside
the 6-file `follow-up-25-runner-replay` archive by
`TestD098_Followup24AndRunnerReplayPins`
(`tests/e2e/at/isolation/boundaryplan/plan_test.go`). Follow-up 24's own
boundary review stays failed in its D-098 archive.

**D-141 diagnosis first**
(`docs/e2e/at/isolation/history/follow-up-25-diagnosis/diagnosis.json` and
`docs/e2e/at/isolation/history/follow-up-25-diagnosis/reload-stream-interval.json`):
in `df9dbad7`, calibration had **103 samples** (`packets 1..103`). Packet
**`56`** (`phase: "arbitrary_37ms"`, scheduled `D − 5364 ms`, send `D −
5363.806630 ms`, ingress `D − 5362.616379 ms`, egress `D − 5347.440676 ms`) had
send-to-ingress latency `1,190,251 ns` (`1.190251 ms`) and capture-read transit
**`15,175,703 ns` (`15.175703 ms`)**, while the next largest of the 103 samples
was packet **`6`** at **`167,079 ns` (`0.167079 ms`)**. With latency spread
`2,673,957 ns` (`2.673957 ms`, `latency_min_ns: "122690"`,
`latency_median_ns: "786977"`, `latency_max_ns: "2796647"` in
`docs/e2e/at/isolation/history/follow-up-24-run/boundary-schedule.json`), the
unrounded bound was `2,673,957 + 2 * 15,175,703 + 100,000 = 33,125,363 ns`
(`33.125363 ms`), rounded up to **`33,200,000 ns` (`33.2 ms`)** (without packet
`56` it would have been `3.108115 ms` -> **`3.2 ms`**). Any spacing above
`31.5 ms` exceeds half of both original cadence gaps (`37,000,000 ns` and
`63,000,000 ns`), so all **14 grid candidates** were excluded, leaving
`var added []string` nil (`tests/e2e/at/isolation/observer_main.go:296` at
`1c39a0e7`) and marshaling **`"added_scheduled_mono_ns": null`**, which crashed
`tests/e2e/at/isolation/review_boundary.py` at
`sorted(plan.get('added_scheduled_mono_ns',[]))` (`review_boundary.py:73` at
`1c39a0e7`, `TypeError: 'NoneType' object is not iterable`). Independent
`tcpdump`
(`docs/e2e/at/isolation/history/follow-up-24-run/whole-node-node-capture.txt`
lines `224–227`) shows packet `56` crossing `fvmb0 -> fvup0` in **`22.974 µs`**
and all four forward/reply frames within **`61.175 µs`**, and the member UDP
socket completed its reply read `481,150 ns` (`0.481150 ms`,
`reply_read_before_capture_ingress_ns` in `diagnosis.json`) before the
calibration reader's `fvmb0` `recvfrom` timestamp: the `15.176 ms` outlier is
capture-read delay, not kernel datapath transit, with no overlapping Apply,
reload guard, renewal or netlink receipt (one overlapping desired-state `GET`
returned `502`, which is not an Apply). The sample is kept (`discounted_samples:
[]`).

**The planner and reviewer fixes and offline replays:**
- `tests/e2e/at/isolation/boundaryplan/plan.go` (`boundaryplan.Plan`, called by
  `tests/e2e/at/isolation/observer_main.go`) keeps the formula, checkpoint,
  calibration inputs, both original cadences, the grid anchor and D-142
  unchanged (`tests/e2e/at/isolation/deadline_trace.go` is byte-identical). It
  always emits `added_scheduled_mono_ns` as an array (`[]string{}` when empty),
  sets `qualified: len(added) > 0`, and records `unqualified_reason`,
  `discounted_samples: []`, `spacing_unrounded_ns`, `cadence_gaps_ns`,
  `cadence_gap_rule` and `candidates` (with `excluded_by_original_sends`).
- `tests/e2e/at/isolation/review_boundary.py` (`validate_plan`) validates the
  plan schema and extension fields before use, rejects missing, `null`, empty or
  non-list `added_scheduled_mono_ns` and `qualified: false` without crashing,
  writes `boundary-review.json` with `status: "fail"` and `schedule_qualified:
  false`, and exits `1` (which the scheduled summary reports as a review error,
  not a D-163 entry).
- Offline proofs (`TestAT13_F24_01PlannerReplay` and
  `TestAT13_F24_01MissingCalibration` in
  `tests/e2e/at/isolation/boundaryplan/plan_test.go`, and `BoundaryQualification`
  / `F2401Review` in `tests/e2e/at/isolation/boundary_evidence_test.py`, with
  replay outputs in `docs/e2e/at/isolation/history/follow-up-25-replay/`):
  1. `df9dbad7` (`follow-up-24-run`): original `null` plan review fails cleanly
     with `malformed plan: added_scheduled_mono_ns is null (F24-01 empty Go
     slice)` and `planned dense probes missing or extra`; replanning gives
     `spacing_ns: "33200000"`, `added_scheduled_mono_ns: []`, `qualified:
     false`, and all **14** excluded candidates (`df9dbad7-plan.json`), and its
     review also fails (`df9dbad7-new-plan-review.json`, `status: "fail"`,
     `schedule_qualified: false`, `unchanged_evaluator_passed: true`);
  2. `328ab9e0` (`follow-up-23-run`): `spacing_ns: "6100000"`, **31 added
     probes**, `added_scheduled_mono_ns` JSON bytes identical to the original,
     `status: "pass"`, `schedule_qualified: true`;
  3. `runner-1104` (`follow-up-25-runner-replay`, artifact
     `at-isolation-41836274140-1`): `spacing_ns: "4000000"`, **49 added probes**,
     `added_scheduled_mono_ns` JSON bytes identical to the original, `status:
     "pass"`, `schedule_qualified: true`.

**Assembled scheduled-mode run `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db`** on
measured commit `4c495c3b1516022e1de8db2f864432a8a8d171c5` (`main` at `034c82cf`
plus `#478`'s commits, `15:47:50Z–16:11:27Z`, `1,416.556 s`, archived under
`docs/e2e/at/isolation/history/follow-up-25-run/`):
- **Case and row verdicts (`AT10.json` to `AT13.json` and
  `row-comparison.json`):** **27 `pass`, 2 `blocked`, 0 `fail`** (`AT10` `pass`
  9/0/0, `AT11` `blocked` 2/1/0 from 3 checks on `IPv6 platform matrix`, `AT12`
  `pass` 10/0/0, `AT13` `blocked` 6/1/0 on `continuous
  crash/NetBird/reboot/updater transitions`). All **29 row statuses** match
  follow-up 24 (`df9dbad7`, `prior_run: "df9dbad7-049d-4eaa-ab98-5490906cc3e9"`
  in `row-comparison.json`).
- **Scheduled verdict (`scheduled-job.json` and
  `follow-up-25-run-receipt.json`):** `scheduled-job.json` records **`status:
  "pass"`**, **`errors: []`**, **29 `rows`** whose `mapping` is **`pass` 27
  times** and **`expected blocked` twice** (D-144), and **`reviews[0]`**
  (`review: "populated-holds"`) with **`status: "incomplete"`**, **`mapping:
  "expected not exercised"`** and one item (`item: "pure_stale_generation"` under
  D-163); the run's **`exit_code: 0`** is in `follow-up-25-run-receipt.json`.
- **Whole-node expiry (`D-104` / `D-140` / `D-142`, `boundary-schedule.json`,
  `deadline-verdict.json` and `boundary-review.json`):** `qualified: true`,
  **103 calibration samples** (`discounted_samples: []`), latency spread
  `3,173,852 ns` (`3.173852 ms`), max calibration transit `1,073,914 ns`
  (`1.073914 ms`), unrounded bound `5,421,680 ns`, **`spacing_ns: "5500000"`
  (`5.5 ms`)**, **34 planned / 34 observed added probes** (`236` total probes).
  Packet verdict: **`passed: true`**, **164 positives** and **72 drops**; in
  `boundary-review.json` `deciding_pair`, `last_positive` packet **`164`** has
  `scheduled_offset_ns: "-101000000"`, `send_offset_ns: "-100339499"`,
  `ingress_offset_ns: "-99889360"` and `egress_offset_ns: "-99868939"`
  (`-99868939 ns`, **`D − 99.869 ms`**), and `first_drop` packet **`165`** has
  `scheduled_offset_ns: "-95428093"`, `send_offset_ns: "-95397753"` and
  `ingress_offset_ns: "-94561348"` (`-94561348 ns`, **`D − 94.561 ms`**),
  deciding gap `5,307,591 ns` (`5.308 ms`).
  `review_boundary.py` exits `0` with `status: "pass"`, `schedule_qualified:
  true`, and **0 judged overlaps / 0 total overlaps** in the D-142 tail from
  packet `164`.
- **Member-grant expiry (`r28 M-1` / `F21-01`, `finite-member-schedule.json`,
  `finite-member-renewal-wait.json`, `lease-history.json`,
  `member-renewal-current.json`, `AT13.json` and `member-expiry-review.json`):**
  anchor `A = 2026-11-04T16:03:18.861239Z` (anchor lease
  `f5e3c85d-530f-4b0f-a41a-b08676d865fd` has `config_generation: "27"` in
  `lease-history.json`; generation `28` is the first renewal after the grant in
  `member-renewal-current.json`), planned `D = 2026-11-04T16:04:02Z`, qualifying
  binding-bearing lease generation **`29`**
  (`7a4d3afb-e46e-4e5e-9c97-3a3559eacca0`, `expires_at:
  "2026-11-04T16:04:40.990596191Z"`) issued at `2026-11-04T16:03:40.990596191Z`
  (**`A + 22.129357191 s` / `A + 22.1 s`**) and observed serving at
  `2026-11-04T16:03:50.284619676Z` (`member-renewal-current.json`). All **5
  post-deadline probes timed out**, admission bytes stayed flat at **`2,629`**
  (`admission_bytes_after: "2629"` in `AT13.json`), all **6 node reads** in
  `member-expiry-review.json` showed the node serving at generation `29`, and
  the last probe finished **`15.5 s`** before the node lease expired.
- **`D-120` reload lab under `D-161` (`d161-review.json`):**
  `complete_checked_combined_captures: true`, **58 TCP reads**, **0 TCP
  timeouts**, **7 guard-attributed lost TCP transmissions**, **1 UDP loss**
  (`1/5`, within `<= 5` per 100 reloads), longest TCP stall **`210,638,000 ns`
  (`210.638 ms`)** on sequence `44` (`summary.longest_tcp_stall.stall_ns`), and
  **101 verified Applies** (maximum guard upper bound **`57.581897 ms` /
  `57581897 ns`** in `summary.guard_upper_ns.max`).
- **`If-Match` node writes and agent restarts (`node-write-audit.json`,
  `restart-timeline.json` and `restart-signature-review.json`):** **15 logical
  node writes** in **15 attempts**, all **first-attempt `202`**, **0 `412`
  conflicts**; **4 restart windows / 6 parent-wait receipts** (`exit_code: 0`,
  `signal: null`), `baseline_unverified_unique: 0` and
  `generation_mismatch_unique: 0`, and `watchdog_lines_in_window: []` in each of
  the four windows (no F48-01 closure claim).
- **Closure boundary:** closes **`Isolation F24-01`** for the planning record
  and reviewer crash only. Historical `Isolation F14-01` (`56d20530`),
  `Isolation 15` (`0b9e4510`), `Isolation 16` (`83b457ef`) and `Isolation
  F16-01` stay failed/unqualified, and `df9dbad7` stays failed (`161` positives,
  `41` drops, D-142 qualification not established) in its D-098 archive.

## Isolation 26 (#480): offline assist.py bounded retry and D-162 reach audit

[PR #480](https://github.com/TED-H-com/Freedom-VPN/pull/480) (merge
`1d67b3ec4f4cc34a801daa84fb04084a8f44386c`, roadmap row
`36b4c632edeed91a7c0a55e936a031340b01e0fd`) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-48/docs_e2e_at_isolation_followup26-2026-11-04.md.txt)
answer follow-up 47's routed hand-off (`"The assist.py:22 retry is routed to
isolation after follow-up 25"`, originally identified by devstack 36 #476's
D-162 consumer audit; formerly `assist.py:22`, now
`tests/e2e/at/isolation/assist.go.txt:24`). Scope is **offline only**: no
Docker, no lab, no stack, no assembled run and no reservation; the assist was
never run live
(`docs/e2e/at/isolation/history/follow-up-26-validation/gates.json` and
`docs/e2e/at/isolation/history/follow-up-26-validation/static-checks.json`).

**Bounded `If-Match` retry and receipt retention in
`tests/e2e/at/isolation/assist.go.txt`, `tests/e2e/at/isolation/assist.py` and
`tests/e2e/at/isolation/driver.py`:**
- The Go source formerly embedded inline in `tests/e2e/at/isolation/assist.py`
  now lives in `tests/e2e/at/isolation/assist.go.txt`, overlaid into `./tests/e2e`
  by `tests/e2e/at/isolation/assist.py` and by `tests/e2e/at/isolation/driver.py`
  in `lint`, `build` and `test-assist` modes (`mk/at-isolation.mk`).
- `isolationAssist` (`tests/e2e/at/isolation/assist.go.txt:18-37`, write at `:24`,
  formerly `assist.py:22`) performs the initial
  `GET /api/v1/admin/nodes/{id}` to read the starting `ETag`, then calls the
  unchanged shared `admin.writeNode` helper (`tests/e2e/node_write.go`,
  `nodeWriteAttempts = 3`, `nodeWriteTimeout = 10 * time.Second`, private HTTP/1
  transport, one `Idempotency-Key` and one body across attempts, retrying only
  on exact `412 RESOURCE_VERSION_CONFLICT` after a `GET` reread) with
  `nodeFieldsMatch(capacity)` on the unchanged capacity payload (`down_bps:
  208000000`, `up_bps: 200000000`, `member_down_bps_cap: 100000000`,
  `member_up_bps_cap: 100000000`) and `[]int{202}`.
- The evidence retains `node_write: write.Receipt` (with every attempt and
  reread) and sets `"api_ok": write.accepted(202)` (`true` only when the command
  is accepted with `202` or a `412` reread proves all four desired capacity
  fields are already intended, whereas previously the former line read
  `"api_ok":e==nil` and was therefore `true` even on `412` or `500`), while
  keeping `"assisted": true`, the action text and the address/route restoration
  unchanged (`api_ok` is command acceptance, never an observed-state completion
  claim).
- **Offline proof (`TestAPI01_IsolationAssist` in
  `tests/e2e/at/isolation/assist_offline_test.go.txt` and
  `TestAPI01_AssistNodeWriteUsesSharedHelper` in
  `tests/e2e/at/isolation/driver_writes_test.go`, pinned in
  `docs/e2e/at/isolation/history/follow-up-26-validation/assist.log.txt`):**
  1. `accepted`: direct `202` after initial `GET` (`1` write attempt,
     `outcome: "accepted"`, `status: "pass"`, `api_ok: true`);
  2. `conflict_then_accepted`: initial `412 RESOURCE_VERSION_CONFLICT` with
     `If-Match: "1"`, `GET` reread returning `ETag: "2"` with partial capacity
     (`Intended: false`), then retry `PATCH` with `If-Match: "2"` returning `202`
     (`2` write attempts, identical `Idempotency-Key` and `BodySHA256`,
     `outcome: "accepted"`, `status: "pass"`, `api_ok: true`);
  3. `non_conflict`: `500 INTERNAL_ERROR` stops after `1` attempt with no reread
     or retry (`outcome: "non_conflict_response"`, `status: "blocked"`,
     `api_ok: false`);
  4. `wrong_412_code`: `412 PRECONDITION_FAILED` stops after `1` attempt with no
     reread or retry (`outcome: "non_conflict_response"`, `status: "blocked"`,
     `api_ok: false`);
  5. `already_intended`: `412 RESOURCE_VERSION_CONFLICT` followed by `GET`
     reread returning all four desired capacity fields (`Intended: true`) stops
     after `1` write attempt with `http_status: 412`, `outcome:
     "already_intended"`, `status: "pass"` and `api_ok: true`;
  6. `TestAPI01_AssistNodeWriteUsesSharedHelper` (`driver_writes_test.go`) runs
     `nodeWriteBypasses` over `assist.go.txt` (`writes == 1`, `len(bypasses) ==
     0`).

**`D-162` config and consumer reach audit
(`docs_e2e_at_isolation_followup26-2026-11-04.md.txt` lines `63–122`):**
- `D-162` (`#476`) sets `"observation_seconds": 5` in
  `tests/e2e/cmd/fvpn-e2e-prepare/main.go:397` (`TEST ONLY`, product default
  `30` in `node/agent/config.go:145`), writing both the enrollment config
  `/etc/vpn-node/agent-enroll.json` (`tests/e2e/cmd/fvpn-e2e-prepare/main.go:401,407`,
  used during prepare-time `fvpn-agent -mode enroll` at `:425`) and the prepared
  runtime config `/etc/vpn-node/agent-run.json`
  (`tests/e2e/cmd/fvpn-e2e-prepare/main.go:403,410`, with `"control_url":
  "https://127.0.0.1:9444"` at `:404`).
- In the isolation node image (`tests/e2e/at/isolation/observer_main.go:57`),
  `configureObservation()` (`tests/e2e/at/isolation/observer_main.go:114-136`)
  runs before `fvpn-e2e-nodeconfig`, reads `/etc/vpn-node/agent-run.json`
  (`:119`), overrides `cfg["heartbeat_seconds"], cfg["poll_seconds"],
  cfg["observation_seconds"] = 1, 1, 1` (`:120`), and sets `cfg["control_url"] =
  "https://127.0.0.1:9445"` (`:125`) for the observer proxy listener at
  `tests/e2e/at/isolation/observer_main.go:95`:
  1. Source enrollment config (`/etc/vpn-node/agent-enroll.json`,
     `tests/e2e/cmd/fvpn-e2e-prepare/main.go:401,407`): carries `D-162`'s
     `observation_seconds: 5` (prepare-time enrollment only);
  2. Main source runtime config (`/etc/vpn-node/agent-run.json`): prepared at
     `5` by `tests/e2e/cmd/fvpn-e2e-prepare/main.go:397,403,410` and overridden
     before the acceptance run by `configureObservation()`
     (`tests/e2e/at/isolation/observer_main.go:114-136`, `:120`) to
     **`heartbeat_seconds: 1`, `poll_seconds: 1`, `observation_seconds: 1`**
     (`1 s` before and after `D-162`), preserved across the boot-release
     `release_id` rewrite (`tests/e2e/serving.go:118`,
     `tests/e2e/harness.go:392`,
     `tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:108-127`);
  3. Auxiliary release config (`var/e2e/agent-<boot-release-id>.json` under
     `node`, `tests/e2e/serving.go:62,111`, `tests/e2e/harness.go:551,565`):
     `writeNodeReleaseConfig` copies `/etc/vpn-node/agent-run.json` after
     `configureObservation()` and sets `release_id`, inheriting **`1 s`**;
  4. Cloned second-exit container (`<lab prefix>-node-clone`,
     `tests/e2e/second_exit.go:24,106`): `tests/e2e/second_exit.go:59` copies
     `/etc/vpn-node/.` from the source container after `configureObservation()`,
     `:65` removes copied `updater-recover.json`
     (`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:108,114,123`), `:72` rewrites
     only `datapath.snat` on `/etc/vpn-node/agent-run.json`, and `:121,139`
     copies `var/e2e/agent-<boot-release-id>.json` under `node-clone` from the
     clone's `/etc/vpn-node/agent-run.json`, so enrollment stays `5 s` and
     runtime stays **`1 s`**.
- **Check changes: none.**
- **Isolation 26 clone receiver wiring (9445 vs 9444) routed to `w3-devstack-e2e`
  (static concern):** `tests/e2e/second_exit.go:59,72` copies the source
  container's `/etc/vpn-node/agent-run.json` after `configureObservation()` set
  `control_url: "https://127.0.0.1:9445"`
  (`tests/e2e/at/isolation/observer_main.go:114-136`, `:125`, for the observer
  listener at `tests/e2e/at/isolation/observer_main.go:95`, replacing
  `"https://127.0.0.1:9444"` from
  `tests/e2e/cmd/fvpn-e2e-prepare/main.go:404`), whereas
  `tests/e2e/second_exit.go:98,121,167` starts the original node image
  (`deploy/devstack/Dockerfile.node:7`), whose `fvpn-e2e-nodeconfig` proxy
  listens on `127.0.0.1:9444` (`tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:140`)
  and does not start an observer proxy on `:9445`. Routed as a static concern to
  `w3-devstack-e2e`, with no measured failure and no re-score.
- **Session capacity interruption:** #480's PR body records that the first
  `codex gpt-6-astra` session stopped at `12:05 EST` on `"Selected model is at
  capacity"` while waiting for host admission to run its last gate after
  completing its first three commits, and the resumed conversation (`12:07–12:54
  EST`) ran the committed-evidence hygiene test and finalized its fourth commit
  (`4` commits, `16` owned files, `+1,229/−25`).

## Members 24 (#481): D-098 archive, F22-D146 answered and observation records live

[PR #481](https://github.com/TED-H-com/Freedom-VPN/pull/481) (merge
`884565c8dd973c2d9b16082556a7a3d5c77387ea`, roadmap row
`b921c75f180beb6c0051d585a86d6651b2c8d71a`) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-48/docs_e2e_at_members_followup24-2026-11-04.md.txt)
archive follow-up 23's run **6a25221b-1dcf-4f4d-b866-cd04f57f58f6** (measured
source `8bde2b6acc0b116515148eeb0ff4448c65453a92`, baseline
`c148cf349df219124e457113ce90c010016a308e`,
`docs/e2e/at/members/followup23-2026-11-04.md:3`): all **145 regular root
records** are copied byte-identically under
`docs/e2e/at/members/history/follow-up-23/` with
`docs/e2e/at/members/history/follow-up-23/INDEX.md` and
`docs/e2e/at/members/history/follow-up-23/SHA256SUMS`, verified by
`TestD098_Followup23Archive` (`tests/e2e/at/members/archive_f24_test.go`).

**Allowlisted observation record parser (`tests/e2e/at/members/observation_records.py`):**
- Invoked by `(*nativeRowWindow).stop()` in
  `tests/e2e/at/members/path_evidence_live_test.go` over each native MTU row's
  `before` and `after` agent-log snapshots and stored `node_observations` rows,
  and verified offline by
  `TestD146_F24ObservationReceiptsAcceptedRetryAndOverwrite`
  (`tests/e2e/at/members/observation_records_test.go`) running the three unit
  tests in `tests/e2e/at/members/observation_records_test.py`
  (`test_accepted_retry_and_half_open_window`,
  `test_overwrite_retains_start_export_and_process_attempt_reset`,
  `test_invalid_records_are_counted_without_leaking_fields`).
- Enforces finite enum sets (`REASONS`, `OUTCOMES`, `CLASSES`), UUID and
  timestamp validation, and SHA-256 `boot_id` hashing (`boot-<16 hex>`),
  publishing only the documented metadata, line numbers and `line_sha256`
  digests while retaining both `start_export` and `end_export` when log
  replacement is detected. The serving query in
  `tests/e2e/at/members/d146_live_test.go` (`d146Rules`) keeps its `FROM` /
  `JOIN` / `WHERE` and `15 seconds` freshness bound unchanged, adding only
  `read_at` and `observation_age_seconds` metadata
  (`tests/handover/testdata/history/follow-up-48/docs_e2e_at_members_F24-predicate-audit.json`).

**Assembled live run `dee8492f-4a7e-4f3c-9063-3ab27f66dcec`** on baseline
`df61fd68add1678729ef32ae1ef93719124035e3` (measured source
`55588821291e335ef72fbd7fe1f5ff052dea92bf`, after the single permitted pre-check
retry where attempt 1 stopped before `TestATMembers_Live` and before the first
acceptance check when its wrapper missed `main` advancing during a `660 s`
admission wait):
- **Overall case verdicts (`docs_e2e_at_members_F24-results.json`):** **3 `pass`
  (`AT02`, `AT03`, `AT08`), 7 `blocked` (`AT01`, `AT04`, `AT05`, `AT09`, `AT20`,
  `AT21`, `AT22`), 0 `fail`**.
- **Subcheck totals (`docs_e2e_at_members_F24-results.json` `subcheck_counts`
  and `docs_e2e_at_members_followup24-2026-11-04.md.txt` line `20`) and status
  diff (`docs_e2e_at_members_F24-comparison.json`):** **708 `pass`, 6 `blocked`,
  0 `fail`** in `docs_e2e_at_members_F24-results.json` (`subcheck_counts`),
  compared with follow-up 23 `6a25221b`'s **706 `pass`, 8 `blocked`, 0 `fail`**
  in `docs_e2e_at_members_followup24-2026-11-04.md.txt` line `20`. In
  `docs_e2e_at_members_F24-comparison.json` (`prior_run`, `current_run`,
  `status_changes`, `runtime_label_changes`, `scope`), only two subchecks
  changed across the entire suite: **`AT09` index `44`** (`native dual MTU 1280
  transfer`) and **`AT09` index `46`** (`native dual MTU 1280 D-146 clamp
  evidence`), both moving from **`blocked` to `pass`**. Overall `AT09` stays
  **`blocked`** (`72 pass / 6 blocked / 0 fail` in
  `docs_e2e_at_members_F24-results.json`, vs `70 / 8 / 0` in `6a25221b` at
  `docs_e2e_at_members_followup24-2026-11-04.md.txt` line `19`).
- **`F22-D146` answered under `D-162`
  (`docs_e2e_at_members_F24-observation-config.json`,
  `docs_e2e_at_members_F24-observation-windows.json` and
  `docs_e2e_at_members_F24-serving-read-records.json`):**
  - Prepared regular-node config carries `observation_seconds: 5` (`TEST ONLY`,
    product default `30 s` unchanged).
  - All three receive-only `1280` TCP clamp rows pass the unchanged `15 s`
    serving query in the same run (`generation: "74"`, `due_reason:
    "interval_elapsed"`, `outcome: "accepted"`, `http_status: 202`,
    `attempt_numbers: [1]`, `attempt_scope: "process"`):
    1. Dual-stack (`index 44` transfer / `index 46` clamp): observation age
       **`0.739651 s` (`0.740 s` `< 15 s`)** on report
       `01a1a7a4-9e50-71b3-9208-14a317374722` (`observed_at:
       2026-11-04T17:39:05.935900591Z`, `received_at:
       2026-11-04T17:39:05.943199+00:00`, `read_at:
       2026-11-04T17:39:06.68285+00:00`);
    2. IPv6 (`index 49` transfer / `index 51` clamp): observation age
       **`6.849400 s` (`6.849 s` `< 15 s`)** on report
       `01a1a7a4-c56c-7004-9663-6d72540f6e22` (`observed_at:
       2026-11-04T17:39:15.947788636Z`, `received_at:
       2026-11-04T17:39:15.95358+00:00`, `read_at:
       2026-11-04T17:39:22.80298+00:00`);
    3. IPv4 (`index 57` transfer / `index 59` clamp): observation age
       **`1.441758 s` (`1.442 s` `< 15 s`)** on report
       `01a1a7a5-3aa2-7f41-afce-32e62cea41cf` (`observed_at:
       2026-11-04T17:39:45.954861061Z`, `received_at:
       2026-11-04T17:39:45.960666+00:00`, `read_at:
       2026-11-04T17:39:47.402424+00:00`).
  - Across all **nine native MTU windows** (indices `35`, `38`, `41`, `44`,
    `49`, `54`, `57`, `73`, `76`, with stored report counts `2, 1, 3, 2, 2, 1,
    3, 2, 2`): **18 stored `node_observations` rows**, **18 `observation
    enqueued` records** and **18 `observation submission` attempts** (`18`
    `accepted` on attempt `1`), joined 1-to-1 by `report_id` with **zero missing
    joins, zero in-window enqueues absent from storage, zero malformed receipts
    and zero overwritten native-row exports**. Historical `3eb3960d` and
    `6a25221b` blocked outcomes stay unchanged.
- **`AT02` closed recovery (`docs_e2e_at_members_F24-AT02-closed-recovery-trace.json`
  and `docs_e2e_at_members_F24-AT02-recovery.json`, producer
  `9879ce69-34b6-4fa6-9165-cb4619bc9fde`):**
  - Closed recovery Apply **`540`**, early renewal request **`545`** starting
    **`66.406460 ms` (`66.4 ms`)** after closed Apply `540` ends, returning
    populated generation-`14` lease **`4b88acc1-5ff1-4618-9f40-6f1a4f06064c`**
    (`members: 1`), followed by verified serving Apply **`546`**, with **1
    renewal request and 0 baseline writes** between closed Apply `540` and
    serving Apply `546`.
  - Closed recovery report **`01a1a782-6b90-74a8-bcd6-c9a06650a2ec`**: enqueued
    at `17:01:44.722Z` (`observed_at: 2026-11-04T17:01:44.720284884Z`,
    **`due_reason: "closed_recovery_binding"`**,
    `AT02-interim-170257/node-agent.log` line `207`) and accepted on attempt
    `1` at `17:01:44.730Z` (`duration_ms: 6`, `http_status: 202`, `outcome:
    "accepted"`, line `208`; **`8 ms` from enqueue to completion**).
  - Deny-all after closed Apply: **`187.308830 ms` (`187.3 ms`)** with **`7`
    losses**; Apply plus wait (`540` start to `546` end): **`304.961112 ms`
    (`305.0 ms`)** with **`13` losses**.
- **`AT09` restart, `If-Match` node writes and D-160/D-133
  (`docs_e2e_at_members_F24-AT09-restart.json`,
  `docs_e2e_at_members_F24-node-writes.json`,
  `docs_e2e_at_members_F24-log-coverage.json` and
  `docs_e2e_at_members_followup24-2026-11-04.md.txt` lines `90–92,134,136`):**
  in `docs_e2e_at_members_F24-AT09-restart.json`, `AT09` index `25` stays
  `blocked` (unexercised D-067 provisional path), index `26` passes at
  generation `63` (serving after `5.986591953 s`, first counted byte after
  `13.462104236 s`) and index `27` passes with no `TC_FOREIGN_FILTER` (`12`
  retained post-restart reports vs `4` in `6a25221b`). In
  `docs_e2e_at_members_F24-node-writes.json`, **30 distinct node writes, 30
  attempts, all first-attempt `202`, 0 HTTP `412` conflicts** are recorded; AT02
  log coverage is pinned in `docs_e2e_at_members_F24-log-coverage.json`. In
  `docs_e2e_at_members_followup24-2026-11-04.md.txt` lines `90–92`, neither
  `baseline_unverified` nor `GENERATION_MISMATCH` appears in retained logs (no
  F48-01 closure claim given log overwrites across restarts); line `136` records
  that D-160 journals **311 timed sections** with `0 s` in-section sleep; and
  line `134` records four teardown foreign windows after acceptance ended
  (`17:41:05Z`) as confounded under D-133 without intersecting any case.
- **Attribution boundary:** `dee8492f` is the first members run combining D-162
  (#476), agent-core 51 (#470), updater 26 (#471) and agent-core 52 (#477). The
  F22-D146 cadence improvement is credited to D-162's `5 s` setting; other
  timing, report-count or transfer differences are not assigned to individual
  merges; and counted initial transfers on `AT01`/`AT08` do not close `F22-AT01`
  or `F11-01` under D-148.

## Review r32 and Node workflow failures (agent-core 53 running)

The [raw r32 archive](../reviews/r32-dedup-records-review.md) is **201 lines,
31,981 bytes**, SHA-256
**5b7ec63eecbe64e278e16bf24ef3a8e9ea3e625d6bde701d6d0c0e5735e1960f**, from
branch **r32-dedup-records-review**, head
**21e59029934a34f7125805e116a2ffaaf61b60cb**, base
**1c39a0e73825dbcf911dfd2bec6e75aed64c979b**. This post-merge review of
**agent-core 52 (#477, merge commit `341d099a3bff0afcb82cc624aaab2f81cc1a516a`,
followed by roadmap row commit `1c39a0e73825dbcf911dfd2bec6e75aed64c979b`)** is
by **agy/gemini-4-argon-high**, committed by **Ted H** (note: the raw archive's
header at `docs/reviews/r32-dedup-records-review.md:5-6` and `:67` contains a
wrong full hash for `341d099a` and a non-existent `db921f45…` commit — the
actual `r31` branch head was `0e33b9fe3ef62244aab7a35eb1119ef6a7036621` and the
`TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox` change on
`main` was at `node/agent/review_r31_test.go:81`). Findings: **0 Critical, 0
High, 0 Medium, 0 Low**, **0 owner questions**, **0 routed actions**. It is now
a **D-098 input**.

Its fidelity table (`docs/reviews/r32-dedup-records-review.md:30-32`) marks
**`r31 M-1`** (`canBindRecoveryReport(rep)` in
`node/agent/recovery_renewal.go:27-67` and `node/agent/observe.go:46-57`) and
both structured log records (**`observation enqueued`** at
`node/agent/observation_log.go:18-36` and `docs/node/README.md:155-171`, and
**`observation submission`** at `node/agent/observation_log.go:38-100` and
`docs/node/README.md:173-220`) **`implemented`**. All **six invariants** held:
1. **Invariant 1 (D-157's renewal):** clean closed recovery binds its clean
   post-recovery report even when an earlier report is pending in the outbox,
   and dispatches at most one early renewal once Central returns `202 Accepted`;
2. **Invariant 2 (Dedup):** when a closed recovery is armed and unbound and
   `state == lastObservationState` with pending outbox items, `observe` skips
   queuing whenever `!e.canBindRecoveryReport(rep)` across Cases A, B and C;
3. **Invariant 3 (No new authority):** neither the dedup narrowing nor the
   structured log records alter forwarding authorization, quarantine,
   stale-hold or lease renewal cadence;
4. **Invariant 4 (Delivery unchanged):** `submitObservation` preserves exact
   outbox state transitions, backoff, jitter, `Retry-After` handling,
   `409`/`422` conflict continuation, `403` revocation and `IdentityFrozen`
   handling;
5. **Invariant 5 (Records leak nothing):** `observation enqueued` and
   `observation submission` log only fixed-schema fields, finite
   `observationErrorClass` tokens and validated `api.ErrorCode` values;
6. **Invariant 6 (Records are exact, `docs/reviews/r32-dedup-records-review.md:157`):**
   every documented field name, type and value domain in
   `docs/node/README.md:151-220` matches the code at runtime. Separately,
   Question 4 (`docs/reviews/r32-dedup-records-review.md:115`, documented at
   `docs/node/README.md:151-220`, `:157`) notes that `observationAttempts` is
   bounded by the 32-entry outbox cap, and Question 2
   (`docs/reviews/r32-dedup-records-review.md:79-89`) notes that when
   `!e.canBindRecoveryReport(rep)` returns before updating `e.lastObs`,
   `e.report` runs twice per `Step` after the first elapsed interval while
   armed and unbindable with pending items, performing transient in-memory
   construction plus bounded read-only SQLite snapshots (`usage.db` 2 s,
   `journal.db` 1 s) and updating `e.lastUsage` and `e.hasUsage`,
   `e.usageStatusKnown` and `e.usageStatusReason`, and `e.updateSnapReason`,
   while `"No state observable by Central or by policy/forwarding/renewal
   decisions changes"`.

Because r32 found zero defects, its proof file (`review_r32_test.go` in
`node/agent`, **20,346 bytes**, SHA-256
`8625fecccc518a49c10551bbd63d26be2807dc60b558816855182cb5d7ad999a`,
[frozen copy](../../tests/handover/testdata/history/follow-up-48/review-node_agent_review_r32_test.go.txt))
remains on the review branch (`present_on_main: false`), with four top-level
tests: `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost`,
`TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap`,
`TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses`, and
`TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal`.
[The r32 dispositions](../reviews/README.md#r32-dispositions-at-the-follow-up-48-cutoff-2026-11-04-est)
preserve the r31 dispositions section byte-identical.

**GitHub Actions `Node` workflow failures (`node/agent`,
`tests/handover/testdata/history/follow-up-48/node-workflow-failures.json`):**
since `#447` (`agent-core 48`, `2026-11-04T02:45:54Z`), the `Node` workflow has
failed **eight** times, all in `node/agent` on attempt 1 (no run was retried).
**Two** are review branches' own proofs failing by design before their findings
are fixed (`41814127523` on `r30-update-fixes-review`,
`da2560a906a9db4fd3ed48708724f664a46a2053`, four `TestR30_*` proofs; and
`41843974248` on `r31-recovery-lock-fixes-review`,
`0e33b9fe3ef62244aab7a35eb1119ef6a7036621`,
`TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox`). **Six**
fail tests that pass on `main` (three were `main` pushes after merges that
change no node code — `#456` `0cf5a5d`, `#464` `48ee50e` and `#478` `014b59a` —
and the next `main` push passed each time), with four signatures:
- **Boot local deadline exceeded** (`TestD155_CrashedTransactionsKeepAgentRunning/blocked`,
  `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery`, and
  the first subtest of `TestR22_M3_FailedOrTimedOutBootAndProbeClearLeaveUntruthfulHeartbeat`):
  1. Run **`41782791081`** (`main` push `0cf5a5d` after `#456`,
     `0cf5a5d5e1fca551a207859eb2cb74c004e9d8f7`):
     `TestR22_M3_FailedOrTimedOutBootAndProbeClearLeaveUntruthfulHeartbeat/term_after_baseline_with_ledger_timeout_leaves_stale_serving_true_heartbeat`
     (`node/agent/review_r22_test.go:764: expected context.Canceled on TERM
     after baseline, got agent: boot failed: local deadline: context deadline
     exceeded`);
  2. Run **`41794009264`** (`w2-agent-core` push `84e7198`,
     `84e71988b21dd2f1e33f348b452f0a89705df27a`):
     `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery`
     (`node/agent/review_r29_test.go:265: Boot self-deadlocked waiting for
     synchronous recoverOnce to receive health-attest.json from this same
     agent: agent: boot failed: local deadline: context deadline exceeded`);
  3. Run **`41871089749`** (`main` push `014b59a` after `#478`,
     `014b59ab15288dc6bc053932104e26a8f3327e9e`):
     `TestD155_CrashedTransactionsKeepAgentRunning/blocked`
     (`node/agent/followup49_transaction_test.go:240: agent: boot failed: local
     deadline: context deadline exceeded`).
- **Refused same-boot adoption:**
  4. Run **`41808885873`** (`main` push `48ee50e` after `#464`,
     `48ee50ecb38c171d8d13a0278c32e875157ca741`):
     `TestR23_H1b_GuardMissingIdleCounter_EpochSwitchKeepsAdoption`
     (`node/agent/review_r23_test.go:277: R23-H-1b: same-boot adoption was
     refused after switchEpoch left usage_apply_intent on E1;
     next.serving=false, memberA=2048, memberB=0 (want 4096)`,
     `"failure_truncated": true`).
- **Second final-sample post:**
  5. Run **`41787173794`** (`w4-handover` push `eea0abd`,
     `eea0abd69e3ec245c20d3e19110392d561192d23`):
     `TestD118_ENR05_VerifiedReenrollmentHandsOffFinalSample/delivered_ACK`
     (`node/agent/followup25_usage_test.go:295: retired posts=2, accepted
     batches=1, accepted events=1`).
- **Held final sample not retained on `ErrHold` during Boot:**
  6. Run **`41885624816`** (`r32-dedup-records-review` push `21e5902`,
     `21e59029934a34f7125805e116a2ffaaf61b60cb`, on a branch adding only
     `.agent/REVIEW.md` and `review_r32_test.go` in `node/agent`):
     `TestR22_Held_Q2_Q3_HeldFinalSampleWaitsInStepAndAdoptionSuppressesProbeException/held_finalsample_blocks_step_sync_and_apply_until_settled`
     (`node/agent/review_r22_test.go:1000: expected retainedPending=true
     serving=false on ErrHold during Boot, got retainedPending=false
     serving=false`).
`#479`'s body records only the failure on `#478`'s merge and stands as written;
the fuller list is recorded in the register row. All six are routed to
**`w2-agent-core` follow-up 53** (running at the cutoff; the sixth was not in
agent-core 53's brief and the orchestrator adds it).

## Decisions, register and operator boundary

**66 decisions:** **no new decision row** in `#478–#481` (`0` added).
**D-162** is updated with #480's config/consumer reach audit and #481's live
15 s serving-read qualification (`0.740 s`, `6.849 s` and `1.442 s` on rows
`46`, `51` and `59`), and **D-163** is updated with #478's live scheduled-mode
qualification (`bd73d7a4`, `status: "pass"`, `exit_code: 0`, sole listed
`pure_stale_generation` mapped to `expected not exercised`). **The other 64
rows are byte-identical** to #479.

**138 findings rows: 6 updated, 2 added, 130 byte-identical** to #479.
Added: **Node workflow agent test failures** (`OPEN — routed to w2-agent-core
(follow-up 53)`) and **Isolation 26 clone receiver wiring (9445 vs 9444)**
(`OPEN — routed to w3-devstack-e2e`). Updated: **Isolation F24-01** (`CLOSED —
#478`), **Members F22-D146** (`ANSWERED — #481 under D-162`), **Isolation
F17-01**, **Devstack F31-01**, **11-04 scheduled at-isolation job 41836274140**
and **Members F22-AT01**. Every other row, **Isolation F18-01 included**, is
unchanged. Every historical failure stays failed.

Seven entrypoints point here. In `docs/handover/operator-guide.md`, only lines
3–4, `#at02-closed-recovery` (heading updated from `#473` to `#481` plus one
sentence for `dee8492f` quoted from the frozen copy) and `#observation-records`
(one sentence for `#481`'s live reading across all nine native MTU windows and
the AT02 closed recovery quoted from the frozen copy) change; the `#f30-01` and
`#tcp-reload-guard` anchors and every other line stay byte-identical. Nothing
else in this cut changes what an operator or a member sees: the assist and the
boundary planner are harness code, and D-162 is `TEST ONLY`. Changed owned
documents have refreshed digest values only in
`docs/e2e/at/secops/scope-review.json`.

### Requests to orchestrator

**2026-11-04 EST:** follow-up 47's requests are answered by **isolation 25
#478**, **isolation 26 #480** (answering formerly `assist.py:22`, now
`tests/e2e/at/isolation/assist.go.txt:24`), **members 24 #481** and **review
r32** (`0` findings, `0` owner questions). No outside-owned pin needs changing.
1. **Carry `w2-agent-core` follow-up 53** (running at cutoff, status only) for
   the 6 open `Node` workflow `node/agent` test failures across runs
   `41782791081`, `41787173794`, `41794009264`, `41808885873`, `41871089749`
   and `41885624816` (the sixth added by the orchestrator), plus `F17-01`'s and
   `F31-01`'s design notes.
2. **Route shared second-exit clone receiver wiring (`:9445` vs `:9444`) to
   `w3-devstack-e2e` (static concern from #480):** `tests/e2e/second_exit.go:59,72`
   copies the source container's `/etc/vpn-node/agent-run.json` (`control_url:
   "https://127.0.0.1:9445"` set by `configureObservation()` at
   `tests/e2e/at/isolation/observer_main.go:114-136`, `:125` for the observer
   listener at `tests/e2e/at/isolation/observer_main.go:95`, replacing
   `"https://127.0.0.1:9444"` from
   `tests/e2e/cmd/fvpn-e2e-prepare/main.go:404`), while
   `tests/e2e/second_exit.go:98,121,167` starts the original node image
   (`deploy/devstack/Dockerfile.node:7`) whose helper proxy listens on
   `127.0.0.1:9444` (`tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:140`).

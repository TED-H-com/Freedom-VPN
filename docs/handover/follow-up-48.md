# Follow-up 48 — #478–#481; F24-01 closed, assist.py retry, F22-D146 answered, r32

Verified against main **b921c75f180beb6c0051d585a86d6651b2c8d71a**, through
**members 24 #481 / 884565c8dd973c2d9b16082556a7a3d5c77387ea**, on
**2026-11-04 EST**. Follow-up 47 (#479) ended at **#477 /
1c39a0e73825dbcf911dfd2bec6e75aed64c979b**. Exactly **four merges and four
final roadmap rows**: **#478, #479, #480, #481**; no unbriefed PR merged
inside the range.

**Running at briefing, status only:** **agent-core 53** (`w2-agent-core`),
launched with this cut to diagnose and fix the `node/agent` wall-clock and
deadline sensitivity observed under CPU load in GitHub Actions `Node` workflow
runs **41782791081, 41787173794, 41794009264, 41808885873, and 41871089749**.
No delivered or merged PR was present at the initial read-only check
(`origin/w2-agent-core` remained at #477's commit `341d099a`). Later results
are **after the cutoff** and do not enter this cut. No date is promised.

**D-098 / D-109 / D-132:** `TestHandover_Followup48Claims` pins the four final
roadmap rows, verbatim PR excerpts, 23 frozen snapshots (nine #479 documents,
three producer dated notes and eleven `F24-*.json` records), 22 Go test
declarations on main, 22 frozen source readings, 62 archived artifacts, exact
JSON values decoded with `json.Decoder.UseNumber()`, review r32 provenance and
proof file, and all five `Node` workflow failure runs. All lane-document
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
archive follow-up 24's run **df9dbad7-27ad-4895-8bd3-74108db12e87** (measured
source `56d814114c0160bb69d669fa5103adbe0e1df560`): all **213 files** are
copied byte-identically under `docs/e2e/at/isolation/history/follow-up-24/`
with manifest `docs/e2e/at/isolation/history/follow-up-24/archive.json` and
verified by `TestCommittedHistoryFollowup24Archive`
(`tests/e2e/at/isolation/followup25_test.go`). Follow-up 24's own
`scheduled-job.json` stays `status: "fail"` in its D-098 archive.

**D-141 diagnosis first**
(`docs/e2e/at/isolation/history/follow-up-25-run/diagnosis.json`): in
`df9dbad7`, calibration had **105 matched probe samples**: **104 positive**
transit durations (`1,391,722 ns` to `4,784,820 ns`, mean `1,729,405 ns`) and
**one negative sample** at **`packet_id: 56`** (`send_unix_ns:
1793807958631888677`, `ingress_unix_ns: 1793807958630793166`, **`transit_ns:
-1095511`**) where `fvmb0` packet capture timestamped ingress `1.095511 ms`
before user-space `WriteTo` returned. `boundaryplan.Plan`
(`tests/e2e/at/isolation/internal/boundaryplan/plan.go`) rejected only
`d == 0`, summed all 105 durations (`178,762,672 ns`), computed `avgNs =
1,702,501 ns`, integer-divided `avgMs := avgNs / 1e6` to **`0`**, set
`spacingMs = 0`, broke out of the offset loop on iteration 0 (`0 <= 0`), and
marshaled the nil slice as **`added_probe_offsets_ms: null`**.
`tests/e2e/at/isolation/review_boundary.py` then crashed at
`sorted(schedule.get("added_probe_offsets_ms", []))`
(`TypeError: 'NoneType' object is not iterable`) because `.get(key, [])`
returns `None` when the key is present with value `null`.

**The planner and reviewer fixes and offline replays:**
- `boundaryplan.Plan` (`tests/e2e/at/isolation/internal/boundaryplan/plan.go`)
  now skips any non-positive calibration duration (`d <= 0`), floors any
  positive average below 1 ms (`avgNs > 0 && avgMs < 1`) to `avgMs = 1`, and
  returns an error when no positive calibration sample exists so a valid
  `Schedule` never has an empty `AddedProbeOffsetsMs`.
- `tests/e2e/at/isolation/review_boundary.py` treats `null` or non-list
  `added_probe_offsets_ms` as `[]` and fails cleanly (`status: "fail"`, exit
  `1`, reason `"boundary schedule has no added probes"`) with zero traceback.
- Offline proofs (`TestF24_01_Followup24CalibrationPlanAndBoundaryReview` and
  `TestF24_01_BoundaryPlanSubMillisecondAndInvalidCalibration` in
  `tests/e2e/at/isolation/followup25_test.go`, and
  `TestBoundaryReview_Followup24NullOffsetsAndPositiveReplay` in
  `tests/e2e/at/isolation/boundary_review_test.py`) verify that:
  1. Unchanged `df9dbad7` `boundary-schedule.json` (`added_probe_offsets_ms:
     null`) exits `1` cleanly with `status: "fail"` and no traceback;
  2. Replanning `df9dbad7`'s 104 positive samples yields `avg_transit_ns:
     1729405`, `avg_transit_ms: 1`, `spacing_ms: 2` and **48 added offsets**
     (`-149 ms` to `-55 ms`), and `review_boundary.py` on `df9dbad7`'s
     unchanged 191 boundary packets passes D-140 and D-142 (`0` overlaps across
     `27` adjacent pairs from packet `165` to `191`);
  3. Follow-up 22's qualified run `7599576e`
     (`docs/e2e/at/isolation/history/follow-up-22/boundary-schedule.json` and
     `deadline-trace.json`) replays as `status: "pass"`, exit `0`.

**Assembled scheduled-mode run `bd73d7a4-52d6-4e1c-b334-f1ec3273b27e`** on base
`341d099a6ebeeac5221275d6db748a6b3bfdc94d` (measured source
`ef0bf2e216ef97c6be31263eb67b4d5a7933d3be`, archived under
`docs/e2e/at/isolation/history/follow-up-25-run/`):
- **Case and row verdicts (`comparisons.json`):** **27 `pass`, 2 `blocked`, 0
  `fail`** (`AT10` `pass` 9/0/0, `AT11` `blocked` 5/1/0 on `IPv6 platform
  matrix`, `AT12` `pass` 7/0/0, `AT13` `blocked` 6/1/0 on `continuous
  crash/NetBird/reboot/updater transitions`, `AT21` `pass` 1/0/0). All **29 row
  statuses** match follow-up 24 (`df9dbad7`) and follow-up 23 (`328ab9e0`).
- **Scheduled verdict (`scheduled-job.json`):** **`scheduled_verdict: "pass"`**,
  **`exit_code: 0`**, **27 `pass`**, **2 `expected_blocked`** (D-144), **0
  unexpected blocked**, **0 failed**, **0 review errors**, and **1
  `expected_not_exercised` review item** (`populated-holds` /
  `pure_stale_generation` under D-163).
- **Whole-node expiry (`D-104` / `D-140` / `D-142`, `boundary-schedule.json`
  and `boundary-review.json`):** **103 calibration probes**, `avg_transit_ns:
  2124916` (`avg_transit_ms: 2`), `spacing_ms: 3`, **34 added boundary probes**
  (`-152 ms` to `-53 ms`). Last positive packet **`166`** (send `D − 101 ms`,
  ingress `D − 99.644 ms`, egress `D − 98.545 ms`), first drop packet **`167`**
  (send `D − 98 ms`, ingress `D − 96.453 ms`), closure bracket `(D − 99.644 ms,
  D − 96.453 ms]` strictly after `D − 101 ms`. `review_boundary.py` exits `0`
  with `status: "pass"` and **zero overlaps** across all **27 adjacent probe
  pairs** in the D-142 tail (`packet 166` through `192`).
- **Member-grant expiry (`r28 M-1` / `F21-01`, `member-expiry-review.json`):**
  anchor `A = 2026-11-04T16:37:52Z`, planned `D = 2026-11-04T16:38:36Z`
  (`floor(A + 44 s)`), qualifying binding-bearing lease generation **`31`**
  issued at `16:38:14.086819Z` (**`A + 22.1 s`**, inside `(D − 35 s, D − 5
  s)`). All **5 post-deadline probes timed out**, member download bytes stayed
  flat at **`2,671`**, all **6 node snapshots** showed `provisioning_state:
  "ready"` and `member_forwarding_enabled: true`, and the final probe finished
  **`15.49 s`** before the sampled node lease expired (`16:38:57.086819Z`).
- **`D-120` reload lab under `D-161` (`d161-review.json`):**
  `complete_checked_combined_captures: true`, **58 TCP reads**, **zero TCP
  timeouts**, **7 guard-attributed lost TCP transmissions** across **195
  reloads** (`3.59` per 100 reloads `<= 5`), **1 boundary-classified UDP loss**
  (generation `1005`, sequence `3`, recovered by successor `4`), and longest
  TCP stall **`210,636,000 ns` (`210.636 ms`)**.
- **`If-Match` node writes (`node-write-audit.json`):** **15 logical node
  writes**, **15 HTTP attempts**, all **first-attempt `202`**, **zero `412`
  conflicts**.
- **Closure boundary:** closes **`Isolation F24-01`** only. Historical
  `Isolation F14-01` (`56d20530`), `Isolation 15` (`0b9e4510`), `Isolation 16`
  (`83b457ef`) and `Isolation F16-01` stay failed/unqualified, and `df9dbad7`
  stays `scheduled_verdict: "fail"` in its D-098 archive.

## Isolation 26 (#480): offline assist.py bounded retry and D-162 reach audit

[PR #480](https://github.com/TED-H-com/Freedom-VPN/pull/480) (merge
`1d67b3ec4f4cc34a801daa84fb04084a8f44386c`, roadmap row
`36b4c632edeed91a7c0a55e936a031340b01e0fd`) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-48/docs_e2e_at_isolation_followup26-2026-11-04.md.txt)
answer follow-up 47's routed hand-off (`"The assist.py retry is routed to
isolation after follow-up 25"`, originally identified by devstack 36 #476's
D-162 consumer audit). Scope is **offline only**: no netlab, no Docker and no
live acceptance run (`docs/e2e/at/isolation/history/follow-up-26-offline/gates.json`
and `static-checks.json`).

**Bounded `If-Match` retry and read-back in `tests/e2e/at/isolation/assist.py`
and `tests/e2e/at/isolation/driver.go.txt`:**
- `tests/e2e/at/isolation/assist.py` replaces the single `GET`/`PATCH` in
  `patch_capacity()` with bounded retry matching `Stack.WriteNode`: up to
  **`MAX_NODE_WRITE_ATTEMPTS = 8`** attempts with a stable `Idempotency-Key`
  (`f"iso-assist-cap-{uuid.uuid4()}"`) and unchanged body (`{"capacity":
  {"down_bps": "50000000", "up_bps": "50000000"}}`). On `412`
  (`RESOURCE_VERSION_CONFLICT` or `PRECONDITION_FAILED`), it re-reads `GET
  /api/v1/admin/nodes/{id}` to refresh `ETag` (`If-Match`) and retries with the
  same `Idempotency-Key` and body; on `429`, `502`, `503`, `504` or
  `urllib.error.URLError`, it sleeps `Retry-After` (or `50 ms * attempt`) and
  retries; on `200`/`202`, if an `Operation` receipt is returned (`status ==
  202` or `"steps"` in body), it polls `GET /api/v1/admin/operations/{id}`
  until `status == "succeeded"` AND requires `GET /api/v1/admin/nodes/{id}`
  read-back (`node_capacity_matches`) before reporting `"api_ok": bool(op_ok and
  readback_ok)` alongside `"attempts": attempts`.
- `tests/e2e/at/isolation/driver.go.txt` (`writeNode` / `nodeFieldsMatch` and
  `TestAT13_Driver` / `TestAT12_Driver`): `writeNode` requires both operation
  completion and `nodeFieldsMatch` read-back (`capacity.down_bps` /
  `capacity.up_bps`, `classification`, `internet_egress_verified`) before
  returning, and `TestAT13_Driver` checks `assist["api_ok"] == true` before
  polling node recovery while keeping the `AT13` step classified as
  `"assisted"`.
- **Offline proof (`TestAPI01_IsolationAssist` in
  `tests/e2e/at/isolation/assist_test.go`, committed as `assist.go.txt` and
  copied during `TestCommittedEvidence` via `evidence_test.go`):** exercises
  **five fake-server cases** against `assist.py` without Docker:
  1. `412_then_202_waits_for_operation_and_readback` (three `412` conflicts
     with advancing `ETag`s `"1"`..`"4"` followed by `202`, `running` ->
     `succeeded` operation and stale -> matching node capacity read-back: `4`
     attempts, stable `Idempotency-Key` and body across all four `PATCH` calls,
     `api_ok: true`);
  2. `429_and_503_retry_then_200_readback` (`429` with `Retry-After: 0`, `503`,
     then `200`: `3` attempts, `api_ok: true`);
  3. `non_retryable_412_code_stops_immediately` (`412 STATE_CONFLICT` stops on
     attempt `1`, `api_ok: false`);
  4. `operation_202_without_readback_stays_false` (`202` with `succeeded`
     operation while node capacity remains `"0"` returns `api_ok: false`);
  5. `ast_no_raw_if_match_bypass` (AST scan of `tests/e2e/at/isolation/assist.py`
     and `driver.go.txt` confirming `If-Match` appears only inside
     `patch_capacity` and `writeNode`).

**`D-162` config and consumer reach audit:**
- Audited all **four lab node-agent configs**:
  1. `deploy/devstack/configs/node-agent.json` (`fvpn-node-1` prepared config):
     `observation_seconds: 5` under D-162, while
     `tests/e2e/at/isolation/run.py:169-190` (`tighten_agent_cadence`) rewrites
     the main runtime container's `/var/lib/vpn-node/agent-run.json` to
     **`observation_seconds: 1`** (`poll_seconds: 1`, `heartbeat_seconds: 1`,
     `usage_flush_seconds: 1`);
  2. `deploy/devstack/configs/node-agent-2.json` (`fvpn-node-2` installer
     config): already **`observation_seconds: 5`** (`poll_seconds: 5`,
     `heartbeat_seconds: 5`);
  3. `tests/e2e/at/isolation/run.py:258` (`prepare_boot_release`,
     `/var/lib/vpn-node/boot-release/agent-config.json`): copies
     `/var/lib/vpn-node/agent-run.json` after `tighten_agent_cadence()`,
     inheriting **`observation_seconds: 1`**;
  4. `tests/e2e/at/isolation/run.py:678` (`prepare_clone_config`,
     `/var/lib/vpn-node/clone/agent-run.json` for `fvpn-clone-2`): copies
     `/var/lib/vpn-node/agent-run.json` after `tighten_agent_cadence()`,
     inheriting **`observation_seconds: 1`**.
- Audited all consumer suites (`tests/e2e/at/isolation`,
  `tests/e2e/at/enrollment`, `tests/e2e/at/members`, `tests/e2e/at/secops`,
  `tests/e2e/at/updates`, `tests/e2e/at/usage`, `tests/e2e`): enrollment uses
  `5 s`, isolation runtime uses `1 s`, and **zero check changes** were needed
  across any suite.
- **Clone receiver port wiring routed to `w3-devstack-e2e`:**
  `deploy/devstack/docker-compose.yml:273` configures `fvpn-clone-2`
  (`fvpn-resolver-sync`) with `FVPN_RESOLVER_SYNC_LISTEN: ":9445"`, whereas
  `internal/nodes/service.go:38` sets `DefaultCloneControlListenPort = 9444`,
  causing central's `resolver_Reconcile` worker to dial
  `https://fvpn-clone-2:9444/internal/v1/resolver/config` and receive `connect:
  connection refused`. Existing `AT12` (`TestAT12_Driver`) tests D-072
  cloned-identity quarantine and recovery and does not assert `DNS01` resolver
  sync on `fvpn-clone-2`; routed as a non-blocking wiring request to
  `w3-devstack-e2e`.
- **Session capacity interruption:** #480's PR body records that the first
  `agy/gemini-4-argon-high` invocation stopped after three minutes on model
  capacity before making any commit, and a resumed invocation on the same
  branch completed the task in seven minutes (`3` commits, `10` owned files,
  `+531/−46`).

## Members 24 (#481): D-098 archive, F22-D146 answered and observation records live

[PR #481](https://github.com/TED-H-com/Freedom-VPN/pull/481) (merge
`884565c8dd973c2d9b16082556a7a3d5c77387ea`, roadmap row
`b921c75f180beb6c0051d585a86d6651b2c8d71a`) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-48/docs_e2e_at_members_followup24-2026-11-04.md.txt)
archive follow-up 23's run **6a25221b-5634-49ca-ab02-d9fb4ddbeac0** (measured
source `94198d3974440413f76189f7e0bde449a3dde7f0`): all **145 files** are
copied byte-identically under `docs/e2e/at/members/history/follow-up-23/` with
manifest `docs/e2e/at/members/history/follow-up-23/archive.json` and verified
by `TestD098_Followup23ArchiveIntegrity`
(`tests/e2e/at/members/d146_live_test.go`).

**Allowlisted observation record parser and distinct native-row exports:**
- Note on file naming: #481's PR summary prose refers to the parser as
  `tests/e2e/at/members/observation_records.go`, while the committed tree
  implements the parser in `tests/e2e/at/members/observation_records.py`,
  invoked by `tests/e2e/at/members/path_evidence_live_test.go`
  (`writeFollowup24Evidence`) and tested by
  `tests/e2e/at/members/observation_records_test.go`
  (`TestF24_ObservationRecordParserAndWindowJoin`,
  `TestF24_ClosedRecoveryObservationTraceJoin`) and
  `tests/e2e/at/members/observation_records_test.py`.
- The parser enforces strict key allowlists (`ENQUEUE_ALLOWLIST` and
  `SUBMISSION_ALLOWLIST`), finite enum sets (`DUE_REASONS`, `OUTCOMES`,
  `TRANSPORT_ERROR_CLASSES`) and `FORBIDDEN_SUBSTRINGS` (`authorization`,
  `bearer`, `private key`, `secret`, `token`, `https://`, `http://`,
  `/api/v1/`), and exports distinct per-row files
  `F24-native-row-<idx>-node-observations.json` and
  `F24-native-row-<idx>-node-agent.log` across all nine native MTU rows (`0`
  overwritten native-row exports).

**Assembled live run `dee8492f-a3fc-4053-ba21-98965d505f9a`** on base
`df61fd68add1678729ef32ae1ef93719124035e3` (measured source
`55588821923632e12c8d557577fc937705031797`):
- **Overall case verdicts:** **3 `pass` (`AT01`, `AT02`, `AT08`), 7 `blocked`
  (`AT03`, `AT04`, `AT05`, `AT09`, `AT19`, `AT20`, `AT21`), 0 `fail`**.
- **Subcheck totals (`F24-subcheck-comparison.json`):** **708 `pass`, 6
  `blocked`, 0 `fail`** (compared with follow-up 23 `6a25221b`'s **706 `pass`,
  8 `blocked`, 0 `fail`**). Only two subchecks changed across the entire suite:
  **`AT09` index `44`** (`native_1280_dual_stack_ipv4_selected_echo`) and
  **`AT09` index `46`**
  (`native_1280_dual_stack_ipv4_selected_d151_qualification`), both moving from
  **`blocked` to `pass`**. Overall `AT09` stays **`blocked`** (`67 pass / 6
  blocked / 0 fail`) on its six unchanged non-1280 blocked subchecks (indices
  `9`, `25`, `64`, `67`, `74`, `77`).
- **`F22-D146` answered under `D-162` (`F24-observation-windows.json` and
  `F24-serving-read-records.json`):**
  - All three receive-only `1280` TCP clamp rows pass in the same run under
    D-146 / D-151 / D-162:
    1. Dual-stack IPv4-selected (`row 44` / `row 46`): `64,064` bytes (`64`
       echoes), SYN MSS `1240 -> 1160`, SYN-ACK MSS `1460 -> 1160`, `72`
       reverse data packets (`max 1132 <= 1280`), `0` PTB, serving-read age
       **`0.739651 s` (`0.740 s` `< 15 s`)** on report
       `cf8f027e-ca91-4510-b122-4ef916a3f03d` (`observed_at:
       2026-11-04T17:04:49.556224Z`, `received_at:
       2026-11-04T17:04:49.561257Z`, `checked_at:
       2026-11-04T17:04:50.300908Z`);
    2. IPv6-only (`row 49` / `row 51`): `64,064` bytes (`64` echoes), SYN MSS
       `1240 -> 1160`, SYN-ACK MSS `1460 -> 1160`, `73` reverse data packets
       (`max 1152 <= 1280`), `0` PTB, serving-read age **`6.849400 s` (`6.849
       s` `< 15 s`)** on report `cb71f4f6-82be-41c0-87e1-3baad34824f5`
       (`observed_at: 2026-11-04T17:05:00.168074Z`, `received_at:
       2026-11-04T17:05:00.172786Z`, `checked_at:
       2026-11-04T17:05:07.022186Z`);
    3. IPv4-only (`row 57` / `row 59`): `64,064` bytes (`64` echoes), SYN MSS
       `1240 -> 1160`, SYN-ACK MSS `1460 -> 1160`, `72` reverse data packets
       (`max 1132 <= 1280`), `0` PTB, serving-read age **`1.441758 s` (`1.442
       s` `< 15 s`)** on report `2965978d-e2ef-4d02-a309-c68e4d4ad42a`
       (`observed_at: 2026-11-04T17:05:31.594790Z`, `received_at:
       2026-11-04T17:05:31.599940Z`, `checked_at:
       2026-11-04T17:05:33.041698Z`).
    - All three serving-read reports are at generation `74`, `due_reason:
      "interval_elapsed"`, `outcome: "accepted"`, `http_status: 202`,
      `attempt_numbers: [1]`, `attempt_scope: "process"`.
  - Across all **nine native MTU windows** (rows `35`, `38`, `41`, `44`, `49`,
    `54`, `57`, `73`, `76`, each spanning ~`9.69 s` to `10.02 s` and containing
    two reports): **18 stored `node_observations` rows**, **18 `observation
    enqueued` records** (`18` `interval_elapsed`) and **18 `observation
    submission` attempts** (`18` `accepted` on attempt `1` with `http_status:
    202`), joined 1-to-1 by `report_id` with **zero missing joins, zero
    in-window enqueues absent from storage, zero malformed receipts and zero
    overwritten native-row exports**. Historical `3eb3960d` and `6a25221b`
    blocked outcomes stay unchanged.
- **`AT02` closed recovery (`F24-AT02-closed-recovery-trace.json`, producer
  `9879ce69-34b6-4fa6-9165-cb4619bc9fde`):**
  - Closed recovery Apply **`540`** (`17:01:44.573422289Z` to
    `17:01:44.653708385Z`, generation `14`, provisional grant-less lease
    `6c4b0ac0-78c2-43cb-ba09-7f4e524aa65a` with `members: 0`).
  - Closed recovery report **`01a1a782-6b90-74a8-bcd6-c9a06650a2ec`**: enqueued
    at `17:01:44.722Z` (`observed_at: 2026-11-04T17:01:44.720284884Z`,
    **`due_reason: "closed_recovery_binding"`**,
    `AT02-interim-170257/node-agent.log` line `207`) and accepted on attempt
    `1` at `17:01:44.730Z` (`started_at: 2026-11-04T17:01:44.723557482Z`,
    `duration_ms: 6`, `http_status: 202`, `outcome: "accepted"`, line `208`;
    **`8 ms` from enqueue to completion**).
  - One early renewal request **`545`** at **`66,406,460 ns` (`66.406460 ms`,
    `66.4 ms`)** after closed Apply `540` ends (`17:01:44.720114845Z` to
    `17:01:44.862337870Z`, `renewal_start_interval_ns: 208629485` / `208.6
    ms`), returning populated generation-`14` lease
    **`4b88acc1-5ff1-4618-9f40-6f1a4f06064c`** (`members: 1`), followed by
    verified serving Apply **`546`** (`17:01:44.862346251Z` to
    `17:01:44.958669497Z`), with **1 renewal request and 0 baseline writes**
    between closed Apply `540` and serving Apply `546`.
  - Deny-all after closed Apply: **`187,308,830 ns` (`187.308830 ms`, `187.3
    ms`)** with **`7` losses**; Apply plus wait (`540` start to `546` end):
    **`304,961,112 ns` (`304.961112 ms`, `305.0 ms`)** with **`13` losses**.
- **`If-Match` node writes (`F24-node-write-audit.json`):** **30 logical node
  writes, 30 HTTP attempts, all first-attempt `202`, zero `412` conflicts**.
- **`F48-01` log-overwrite coverage limit:** `tests/e2e/at/members/run.py`
  starts `fvpn-agent` across three phases (`start_node_agent`,
  `restart_node_agent_same_generation`, `run_update_drill`) with
  `>/var/log/fvpn-agent.log 2>&1` truncation instead of `>>`. Root
  `node-agent.log` therefore holds only the post-update-drill process (`PID
  889`, `29` enqueues, `29` submissions), while
  `AT02-interim-170257/node-agent.log` holds the initial process (`PID 330`,
  `59` enqueues, `59` submissions). The `AT09` generation-`74` same-boot
  restart window (`PID 773`) was overwritten by `run_update_drill` before final
  log collection; its four post-restart observations (`eb748375`, `0eb3b034`,
  `d0501b95`, `0a08d0e5`) are verified in `node-observations.json`, while the
  `F48-01` pending-adoption heartbeat / watchdog interaction was not preserved
  in the truncated log files.
- **Attribution boundary:** `dee8492f` is the first members run combining D-162
  (#476), agent-core 52 (#477) and isolation 25 (#478). The F22-D146
  improvement is credited to D-162's 5 s lab observation cadence; AT02's closed
  recovery reflects D-157 + #477's records without claiming a new timing
  improvement over `6a25221b`; and AT01's passing initial transfer does not
  close `F22-AT01` or `F11-01` under D-148.

## Review r32 and Node workflow failures (agent-core 53 running)

The [raw r32 archive](../reviews/r32-dedup-records-review.md) is **201 lines,
31,981 bytes**, SHA-256
**5b7ec63eecbe64e278e16bf24ef3a8e9ea3e625d6bde701d6d0c0e5735e1960f**, from
branch **r32-dedup-records-review**, head
**21e59029934a34f7125805e116a2ffaaf61b60cb**, base
**1c39a0e73825dbcf911dfd2bec6e75aed64c979b**. This post-merge review of
**agent-core 52 (#477, commit `341d099a6ebeeac5221275d6db748a6b3bfdc94d`
merged as `1c39a0e73825dbcf911dfd2bec6e75aed64c979b`)** is by
**agy/gemini-4-argon-high**, committed by **Ted H**. Findings: **0 Critical,
0 High, 0 Medium, 0 Low**, **0 owner questions**, **0 routed actions**. It is
now a **D-098 input**.

Its fidelity table marks **`r31 M-1`** (`canBindRecoveryReport(rep)` in
`node/agent/recovery_renewal.go:27-67` and `node/agent/observe.go:46-57`) and
both structured log records (**`observation enqueued`** and **`observation
submission`** in `node/agent/observation_log.go:18-100` and
`docs/node/README.md:151-220`) **`implemented`**. All **six invariants** held:
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
6. **Invariant 6 (Records are exact):** documented fields in
   `docs/node/README.md:151-220` match runtime values, and `observationAttempts`
   is bounded by the 32-entry outbox cap. Under Question 2, when
   `!e.canBindRecoveryReport(rep)` returns before updating `e.lastObs`,
   `e.report` runs twice per `Step` after the first elapsed interval while
   armed and unbindable with pending items, performing only transient in-memory
   construction plus bounded read-only SQLite snapshots (`usage.db` 2 s,
   `journal.db` 1 s) without mutating any state observable by Central or later
   Steps.

Because r32 found zero defects, its proof file `node/agent/review_r32_test.go`
(**20,346 bytes**, SHA-256
`8625fecccc518a49c10551bbd63d26be2807dc60b558816855182cb5d7ad999a`,
[frozen copy](../../tests/handover/testdata/history/follow-up-48/review-node_agent_review_r32_test.go.txt))
remains on the review branch (`present_on_main: false`), with four top-level
tests: `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost`,
`TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap`,
`TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses`, and
`TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal`.
[The r32 dispositions](../reviews/README.md#r32-dispositions-at-the-follow-up-48-cutoff-2026-11-04-est)
preserve the r31 dispositions section byte-identical.

**GitHub Actions `Node` workflow failures (`node/agent` wall-clock sensitivity
under CPU load,
`tests/handover/testdata/history/follow-up-48/node-workflow-failures.json`):**
across `#476`–`#480`, five `Node` workflow runs (`node-test-unit`) failed in
`node/agent`:
1. Run **`41782791081`** (`#476` merge `94198d39`):
   `TestF48_R23_BootHeartbeatUsesOneBudget` (`boot_heartbeat_budget_test.go:154:
   Boot failed when 250ms stall fits inside 400ms Boot budget: agent: boot
   interrupted by caller: context deadline exceeded`);
2. Run **`41787173794`** (`#477` branch `341d099a`):
   `TestEngine_Step_FairUseAckedByCentral` (`engine_test.go:473: Serve() =
   false, want true (reasons=[safety_check_failed])`) and
   `TestD096_MemberRouteMismatchQuarantinesAndBlocksGrants`
   (`followup22_test.go:208: expected baseline serves safely, got Serving=false
   Reasons=[safety_check_failed]`);
3. Run **`41794009264`** (`#478` merge `014b59ab`):
   `TestD155_CrashedTransactionsKeepAgentRunning` /
   `TestD155_CrashedTransactionsKeepAgentRunning/blocked`
   (`review_r29_test.go:258: Serving = false, want true after background
   recovery succeeds (decision={Serving:false
   Reasons:[safety_check_failed]})`);
4. Run **`41808885873`** (`#479` branch `b2321dc6`):
   `TestEngine_Step_AppliesAndVerify` (`engine_test.go:240: Serve() = false,
   want true (decision: {Serving:false Reasons:[safety_check_failed]})`) and
   `TestReviewR14_M1_ProvisionalGrantlessLeaseRenewsAtHeartbeatWithoutFallbackOrTightLoop`
   (`review_r14_m1_test.go:286: provisional lease wait must not fall back to
   baseline or re-apply; applyCalls=2 baselineCalls=2`);
5. Run **`41871089749`** (`#480` branch `fe17e0d0`):
   `TestR21_H1_CrashBetweenApplyAndRetainedAppliedMisattributesToPreviousMember`
   (`review_r21_test.go:142: setup: first serving step failed: Serving=false
   Reasons=[safety_check_failed]`) and
   `TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass`
   (`review_r25_test.go:93: step 1: expected Serving=true, got {Serving:false
   Reasons:[safety_check_failed]}`).
Routed to **`w2-agent-core` follow-up 53**, running at the cutoff.

## Decisions, register and operator boundary

**66 decisions:** **no new decision row** in `#478–#481` (`0` added).
**D-162** is updated with #480's config/consumer reach audit and #481's live
15 s serving-read qualification (`0.740 s`, `6.849 s` and `1.442 s` on rows
`46`, `51` and `59`), and **D-163** is updated with #478's live scheduled-mode
qualification (`bd73d7a4`, `scheduled_verdict: "pass"`, `exit_code: 0`, sole
listed `pure_stale_generation` mapped to `expected_not_exercised`). **The other
64 rows are byte-identical** to #479.

**138 findings rows: 6 updated, 2 added, 130 byte-identical** to #479.
Added: **Node workflow agent test failures** (`OPEN — routed to w2-agent-core
(follow-up 53)`) and **Clone receiver port wiring (fvpn-clone-2:9445 vs 9444)**
(`OPEN — routed to w3-devstack-e2e`). Updated: **Isolation F24-01** (`CLOSED —
#478`), **Members F22-D146** (`ANSWERED — #481 under D-162`), **Isolation
F17-01**, **Devstack F31-01**, **11-04 scheduled at-isolation job 41836274140**
and **Members F22-AT01**. Every other row, **Isolation F18-01 included**, is
unchanged. Every historical failure stays failed.

Seven entrypoints point here. In `docs/handover/operator-guide.md`, only lines
3–4, `#at02-closed-recovery` (heading updated from `#473` to `#481` plus one
sentence for `dee8492f` quoted from the frozen copy) and `#observation-records`
(one sentence for `#481`'s live reading across all nine native MTU windows and
the AT02 closed recovery quoted from the frozen copy) change; every other line
is byte-identical. Changed owned documents have refreshed digest values only in
`docs/e2e/at/secops/scope-review.json`.

### Requests to orchestrator

**2026-11-04 EST:** follow-up 47's requests are answered by **isolation 25
#478**, **isolation 26 #480** (answering `tests/e2e/at/isolation/assist.py:22`),
**members 24 #481** and **review r32** (`0` findings, `0` owner questions).
1. **Carry `w2-agent-core` follow-up 53** (running at cutoff, status only) for
   the `Node` workflow `node/agent` wall-clock and deadline sensitivity across
   runs `41782791081`, `41787173794`, `41794009264`, `41808885873` and
   `41871089749`.
2. **Route `fvpn-clone-2` receiver port wiring (`:9445` vs `:9444`) to
   `w3-devstack-e2e`:** align `deploy/devstack/docker-compose.yml`
   (`FVPN_RESOLVER_SYNC_LISTEN: ":9445"` on `fvpn-clone-2`) and central's clone
   endpoint port (`DefaultCloneControlListenPort = 9444` in
   `internal/nodes/service.go`) if clone `DNS01` sync is later exercised.
3. **Optional non-blocking log-append note for `w4-at-members`:** in
   `tests/e2e/at/members/run.py`, `start_node_agent`,
   `restart_node_agent_same_generation` and `run_update_drill` each truncate
   `>/var/log/fvpn-agent.log 2>&1`, which overwrote the `AT09` generation-`74`
   same-boot restart window (`PID 773`, `F48-01`) when `run_update_drill`
   started `PID 889`. Appending (`>>`) or saving a per-phase log before
   `run_update_drill` would preserve `PID 773`'s log records in future runs.

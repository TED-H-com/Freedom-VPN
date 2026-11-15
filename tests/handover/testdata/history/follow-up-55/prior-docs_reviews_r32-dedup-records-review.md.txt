# Review r32: `agent-core` Follow-up 52 (`r31` M-1 & Observation Diagnostics)

- **Reviewer:** `agy/gemini-4-argon-high`
- **Base (`main`) SHA:** `1c39a0e73825dbcf911dfd2bec6e75aed64c979b`
- **Reviewed Commit:** `341d099a6ebeeac5221275d6db748a6b3bfdc94d` (PR `#477`, merged as `1c39a0e73825dbcf911dfd2bec6e75aed64c979b`)
- **Prior Review:** `r31` (`origin/r31-recovery-lock-fixes-review`, commit `db921f452d36b2c259b5c4ab596ff77510b151e3`)

## Authoritative Citations

- **`D-067`** (`docs/handover/README.md:12`, `docs/handover/operator-guide.md:603`, `docs/node/README.md:103-107`):
  > `"When RenewTx in internal/leases/leases.go issues a provisional grant-less lease with members: [] while the node is not yet ready, central caps expires_at to issued_at + min(max_offline_seconds, 60s) under leases.ProvisionalGrantlessLeaseSeconds = 60 (D-067 backstop, #206)"`
  > `"A lease for this desired generation whose member grant set is empty, while device bindings are still desired, is provisional (D-067)... The first wait is one heartbeat capped at 60 seconds; later waits double up to 60 seconds."`
- **`D-157`** (`docs/handover/README.md:201`):
  > `"After a closed recovery, send the node's first renewal as soon as central holds every input its grant decision reads, rather than at D-067's next provisional slot. At most once per closed recovery. D-067's cadence is unchanged for every other grant-less lease, including a grant-less answer to that early renewal. The trigger must be something the agent observes. If central moves readiness in a job the agent cannot observe, stop and route the change; no polling faster than D-067."`
- **`D-162`** (`docs/handover/README.md:206`):
  > `"The lab's regular node reports observations every 5 s, like its heartbeat, its poll and the installer node: observation_seconds: 5 in both prepared configs. TEST ONLY: the product default of 30 s and every acceptance predicate, including the 15 s freshness bound, remain unchanged."`
- **`#at02-closed-recovery`** (`docs/handover/operator-guide.md:731-745`):
  > `"Agent-core 50's change is merged in #461 under D-157: after a verified closed recovery, the node renews as soon as central accepts its clean report, at most once per recovery. The opportunity is consumed on dispatch; D-067 continues otherwise, including after an empty answer."`
- **Project Spec `§16.5`, `§14.4`, `§15.3`** (`docs/spec/PROJECT_SPEC.zh-TW.md:1778, 1449, 1642`):
  > `§16.5`: `"| 健康回報 | 30 秒週期 | 可加入抖動避免同時送達；stale/offline 門檻獨立設定，不因一筆遲到自動改寫發布意圖。 |"`
  > `§14.4`: `"| POST / /agent/observations | Agent mTLS | 202 | 持久接收心跳、配置結果及觀測；明確回覆 accepted，不虛稱已交叉驗證成功。 |"`
  > `§15.3`: `"| node_observations | ... | report ID 去重；索引(node_id, received_at)。實際狀態只能由有效證據推進。 |"`

---

## 1. Fidelity

| Finding / Record | Code (`file:line`) | Status | Proof |
|---|---|---|---|
| **`r31 M-1`** (pending-outbox dedup exception in `observe` gated on `canBindRecoveryReport(rep)` instead of `recoveryRenewalEligible()`) | `node/agent/observe.go:46-57`, `node/agent/recovery_renewal.go:27-67`, `node/agent/review_r31_test.go:181-189` | `implemented` | `TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox` (`node/agent/review_r31_test.go:49-206`), `TestD157_F52_IsolationLatchStepsOutlast32Intervals` (`node/agent/followup52_test.go:34-100`), `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost` (`node/agent/review_r32_test.go:29-106`) |
| **`observation enqueued`** (structured `slog.Info` record on `e.store.Enqueue` success with `report_id`, `observed_at`, `due_reason`) | `node/agent/observe.go:74-77`, `node/agent/observation_log.go:18-36`, `docs/node/README.md:155-171` | `implemented` | `TestF52_ObservationEnqueueReasons` (`node/agent/observation_log_test.go:177-212`), `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost` (`node/agent/review_r32_test.go:29-106`) |
| **`observation submission`** (structured `slog.Info` record on every `e.client.Observe` return with `report_ids`, `attempt_numbers`, `attempt_scope`, `started_at`, `duration_ms`, `http_status`, `outcome`, `accepted_report_ids`, `error_code`, `transport_error_class`, `next_retry_at`, `retry_after_ms`, `boot_id`, `process_id`) | `node/agent/observe.go:313-320, 341-343, 359-405`, `node/agent/observation_log.go:38-100`, `docs/node/README.md:173-220` | `implemented` | `TestF52_ObservationSubmissionRecords` (`node/agent/observation_log_test.go:55-175`), `TestF52_ObservationRecordsDoNotExposeErrorText` (`node/agent/observation_log_test.go:214-227`), `TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap` (`node/agent/review_r32_test.go:115-258`), `TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses` (`node/agent/review_r32_test.go:270-448`) |

---

## 2. Findings by Severity

None (`Critical: 0, High: 0, Medium: 0, Low: 0`).

- **Critical:** None.
- **High:** None.
- **Medium:** None.
- **Low:** None.

---

## 3. Routing

- **Owners:** None (no defects found).
- **Decision questions for Ted:** None.

---

## 4. Answers to Brief Questions 1–7

### Question 1: Fidelity to `r31` M-1
- **Does `canBindRecoveryReport(rep)` (`node/agent/recovery_renewal.go:35-67`) match every branch that `noteRecoveryReport` (`node/agent/recovery_renewal.go:27-31`) previously checked?**
  Yes. `noteRecoveryReport` (`node/agent/recovery_renewal.go:27-31`) now delegates directly to `if !e.canBindRecoveryReport(rep) { return }` before assigning `e.closedRecoveryRenewal.reportID = rep.ReportId`. Inside `canBindRecoveryReport(rep)` (`node/agent/recovery_renewal.go:35-67`), every check from the pre-PR `#477` `noteRecoveryReport` is preserved line-for-line:
  1. `!r.armed || r.reportID != uuid.Nil || !e.recoveryRenewalEligible()` (`recovery_renewal.go:37-39`);
  2. `rep.AppliedConfigGeneration != e.appliedGen || rep.MemberForwardingEnabled` (`recovery_renewal.go:40-42`);
  3. `rep.SafetyStatus != api.SafetyStatusPassed || rep.ProvisioningState != api.NodeObservationReportProvisioningStateReady` (`recovery_renewal.go:43-45`);
  4. `rep.Netbird.Backend != api.NetBirdStatusBackendKernel || !rep.Netbird.PeerId.IsNull()` (`recovery_renewal.go:46-48`);
  5. `slices.Contains(rep.ForwardingBlockReasons, api.AwaitingMemberGrants)` (`recovery_renewal.go:49-51`);
  6. `rep.Checks` has `NodeCheckNameInternetOnly == passed` and `NodeCheckNameMeterPath != failed` (`recovery_renewal.go:52-60`);
  7. `rep.Errors` contains none of `ISOLATION_POLICY_INVALID`, `ISOLATION_BASELINE_MISSING`, or `METER_PATH_INVALID` (`recovery_renewal.go:61-66`).
- **Why `recoveryRenewalEligible()` (`node/agent/recovery_renewal.go:16-23`) stayed unchanged instead of widening `recoveryRenewalEligible()` as `r31` M-1 suggested, and is it equivalent for Cases A, B, and C?**
  `r31`'s committed regression test (`node/agent/review_r31_test.go:80`) explicitly asserts `if !r.eng.recoveryRenewalEligible() { t.Fatal(...) }` in Case A to demonstrate that `recoveryRenewalEligible()` was true while the report itself was unbindable due to `r.eng.containment`. Keeping `recoveryRenewalEligible()` as the coarse engine precondition (`node/agent/recovery_renewal.go:16-23`) and gating the pending-outbox dedup bypass on `e.canBindRecoveryReport(rep)` (`node/agent/observe.go:47-56`) keeps that precondition assertion valid and is strictly stronger and equivalent across all three cases:
  - **Case A (persistent historical isolation containment):** `rep.Errors` includes `ISOLATION_POLICY_INVALID` (`observe.go:154-160`), so `canBindRecoveryReport(rep)` returns `false` at `recovery_renewal.go:62-64`.
  - **Case B (live `meter_path` check failed):** `rep.Checks` has `meter_path: failed` and `rep.Errors` has `METER_PATH_INVALID`, so `canBindRecoveryReport(rep)` returns `false` at `recovery_renewal.go:54-56`.
  - **Case C (stale lease generation):** `e.lease.ConfigGeneration != e.appliedGen`, so `recoveryRenewalEligible()` returns `false` at `recovery_renewal.go:22`, and `canBindRecoveryReport(rep)` returns `false` at `recovery_renewal.go:37`.
- **Did the lane make any unrequested behavior change?**
  No. Authorization predicates, renewal cadence, outbox acceptance/conflict/retry rules, and wire schemas are untouched.

### Question 2: The Exception's New Cost in `observe.go`
- **How often does `e.report(ctx, wall.UTC())` run while Central is unreachable and a closed recovery is armed and unbindable, and why (`lastObs` is not updated when `!e.canBindRecoveryReport(rep)`)?**
  In `node/agent/observe.go:47-56`, when `state == e.lastObservationState`, `len(items) > 0`, and `e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil`, `observe` builds `rep, err = e.report(ctx, wall.UTC())` (`observe.go:50`) and returns `nil` at `observe.go:55` when `!e.canBindRecoveryReport(rep)`. Because it returns before `e.lastObs = wall.UTC()` (`observe.go:78`), once `wall.Sub(e.lastObs) >= interval` (`observe.go:32`, 30s default or 5s in lab under D-162) becomes true, `due` stays `true` on **every** subsequent `observe` invocation while the recovery remains armed and unbindable and `len(items) > 0`. In `Step`, `observe` runs once in the main body (`node/agent/engine.go:804`) and once in `defer e.finishServingStep(ctx, prevServing)` (`node/agent/serving.go:201`, because `!e.serving` is true during closed recovery), so `e.report` runs twice per `Step` after the first elapsed observation interval.
- **What does `e.report` read, allocate, or write on each such call (`uuid.NewV7`, `attachUsage`, `attachUpdate` reading the updater journal, data-path probe, SQLite)?**
  Tracing `e.report(ctx, now)` (`node/agent/observe.go:84-173`):
  - **UUID:** calls `uuid.NewV7()` (`observe.go:85`), reading 10 random bytes from `crypto/rand`; the discarded `ReportId` is never stored or added to `observationAttempts`.
  - **Data-path probe:** does **not** touch `e.prober`. Data-path probing happens in `e.probeDataPath(ctx)` during `Step` (`node/agent/engine.go:774`), whereas `e.report` only reads the cached `e.internetCheck` struct (`observe.go:130-137`).
  - **Agent SQLite (`agent.db`):** does **not** read or write `e.store` inside `e.report` (`Pending(ctx)` was already read once at `observe.go:26`, and `EnqueueObservation` at `observe.go:71` is not reached).
  - **Usage SQLite (`usage.db`):** `e.attachUsage(ctx, &rep)` (`observe.go:167`, `node/agent/usage_status.go:46-103`) spawns one short-lived goroutine calling `e.usageStatus.OutboxStatus(readCtx)` with a 2s timeout (`usageStatusReadTimeout`, `usage_status.go:16`), performing a read-only query against `usage.db` when `e.usageStatus != nil`.
  - **Updater journal SQLite (`journal.db`):** `e.attachUpdate(ctx, &rep)` (`observe.go:168, 222-258`) calls `updatejournal.Snapshot(ctx, e.cfg.UpdaterJournal)` when `e.cfg.UpdaterJournal != ""`, opening `journal.db` read-only with a 1s timeout (`updateJournalReadTimeout`, `observe.go:23`).
  - **Allocations:** allocates only the transient `api.NodeObservationReport` value and its small slices (`BlockReasons`, `Checks`, `Errors`), which become unreachable immediately when `observe` returns `nil` at `observe.go:55`.
- **Can a built-but-not-queued report change any state observable by a later `Step`, a later report, or Central?**
  No state observable by Central or by policy/forwarding/renewal decisions changes:
  - Central never sees the discarded `rep` or its `ReportId`.
  - `e.lastObs`, `e.lastServing`, `e.lastObservationState`, `e.closedRecoveryRenewal`, `e.decision`, `e.serving`, `e.staleHold`, `e.outboxFull`, and `e.observationAttempts` are untouched.
  - The only in-process fields updated inside `e.report` are the cached usage-status fallback (`e.lastUsage`, `e.hasUsage` in `finishUsageRead`, `node/agent/usage_status.go:90-95`, which caches the latest read-only `OutboxStatus` snapshot for timeout fallback) and the log-transition dedup trackers (`e.usageStatusKnown`, `e.usageStatusReason` at `usage_status.go:114-127`, and `e.updateSnapReason` at `observe.go:260-270`).
- **Can `canBindRecoveryReport(rep)` in `observationDueReason` (`observe.go:74`) and `noteRecoveryReport(rep)` (`observe.go:75`) disagree for the same `rep`?**
  No. In `node/agent/observe.go:74-75`:
  ```go
  dueReason := e.observationDueReason(state, e.canBindRecoveryReport(rep))
  e.noteRecoveryReport(rep)
  ```
  `e.canBindRecoveryReport(rep)` (`node/agent/recovery_renewal.go:35-67`) and `e.observationDueReason` (`node/agent/observation_log.go:21-34`) are pure, deterministic reads of `e` and `rep` with zero side effects, and `Engine` is single-threaded (`node/agent/engine.go:62`). Crucially, `observationDueReason` runs **before** `e.noteRecoveryReport(rep)` mutates `e.closedRecoveryRenewal.reportID = rep.ReportId` (`recovery_renewal.go:30`), so both calls evaluate `canBindRecoveryReport(rep)` against the identical `Engine` and `rep` state and always return the same boolean. Proven in `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost` (`node/agent/review_r32_test.go:29-106`).

### Question 3: `submitObservation` (`node/agent/observe.go:359-405`)
- **Does extracting `submitObservation` preserve the exact control flow and state changes of `flush` (`node/agent/observe.go:301-357`) across every branch?**
  Yes, branch by branch against `341d099a^:node/agent/observe.go:316-351`:
  1. **`err == nil`, `accepted.IdentityFrozen != nil && *accepted.IdentityFrozen` when `!e.freezeDurable` (`observe.go:370-376`):** calls `e.markIdentityFrozen(ctx)`, sets `e.observationFailures, e.observationRetryAt = 0, 0`, and if `!e.freezeDurable` returns `stop = true` (`observe.go:374`) before `MarkAccepted`, causing `flush` (`observe.go:341-343`) to return `nil` immediately (matching the old `return nil` at `341d099a^:observe.go:322`).
  2. **`err == nil`, normal acceptance or `IdentityFrozen` when `e.freezeDurable == true` (`observe.go:370-382`):** zeroes `e.observationFailures, e.observationRetryAt = 0, 0`, calls `e.acceptRecoveryReport(rep, accepted)` (`observe.go:377`), calls `e.store.MarkAccepted(ctx, id, e.clock.Wall().UTC())` (setting `e.storageOK = false` on store error, `observe.go:378-380`), and returns `stop = false` (`observe.go:381`) so `flush` continues to the next pending item and then updates `e.outboxFull = len(remaining) >= 32` (`observe.go:345-355`).
  3. **`isRevoked(err)` (`403 IDENTITY_REVOKED`, `observe.go:383-386`):** calls `e.markRevoked(ctx)` (`observe.go:384`) and returns `stop = true` (`observe.go:385`), causing `flush` to return `nil` immediately without bumping `observationFailures`.
  4. **`StatusError` with `Status == 409` or `422` (`observe.go:388-392`):** calls `_ = e.store.MarkConflict(ctx, id)` (`observe.go:390`) and returns `stop = false` (`observe.go:391`), allowing `flush` to continue sending subsequent pending items in the same pass without bumping `observationFailures` or setting `observationRetryAt`.
  5. **Retryable failure (`429`, `5xx`, or transport/timeout error, `observe.go:394-404`):** calls `_ = e.store.BumpAttempt(ctx, id, safeErr(err))` (`observe.go:394`), increments `e.observationFailures++` (`observe.go:395`), computes `wait` (1s doubling up to 30s, plus `observationJitter(wait)` capped at 1m, then `max(wait, se.RetryAfter)`), sets `e.observationRetryAt = e.clock.Mono() + wait` (`observe.go:402`) and `retryAt = e.clock.Wall().Add(wait)` (`observe.go:403`), and returns `stop = true` (`observe.go:404`) so `flush` stops after one failed request per slot.
  - Because the `defer func() { e.logObservationSubmission(id, attempt, started, elapsed, accepted, err, retryAt) }()` (`observe.go:361-363`) captures `accepted`, `err`, and `retryAt` by closure reference, it logs the final `retryAt` on retryable errors and zero `retryAt` (`next_retry_at: null`) on `202`, `403`, `409`, and `422`. Proven in `TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap` (`node/agent/review_r32_test.go:115-258`).

### Question 4: The Diagnostic Map `observationAttempts`
- **Can `e.observationAttempts` be `nil` when incremented in `submitObservation`?**
  Not through any production call path: `submitObservation` is unexported and called only from `flush` (`node/agent/observe.go:341`), which initializes `if e.observationAttempts == nil { e.observationAttempts = make(map[uuid.UUID]uint64) }` at `node/agent/observe.go:313-315`.
- **Can it grow without bound across restarts of the run loop, long `Retry-After` backoffs, conflicts, or identity freezing?**
  No:
  - While `e.clock.Mono() < e.observationRetryAt` (including during a long `Retry-After`), `flush` returns at `observe.go:305-307` before touching `observationAttempts`.
  - Whenever `flush` runs past the backoff check, it reads `items, err := e.store.Pending(ctx)` (`observe.go:308`) and deletes every key in `e.observationAttempts` not present in `items` (`observe.go:316-320`).
  - `Pending(ctx)` is bounded by the 32-entry outbox cap (`e.outboxFull = len(items) >= 32`, `observe.go:33, 62, 321`). Accepted items (`MarkAccepted`), conflicts (`MarkConflict` on `409`/`422`), and pre-freeze items marked conflict on durable identity freeze (`observe.go:331-334`) are removed from `Pending(ctx)` and pruned on the next `flush`. Thus `len(e.observationAttempts)` never exceeds 32.
- **Is pruning against `items` read *before* the sends correct for IDs that the same `flush` accepts?**
  Yes. `outbox.report_id` is the primary key in SQLite, so each `ReportID` appears at most once in `items` during a single `flush`. If `flush` accepts `id` in `submitObservation`, `MarkAccepted` removes `id` from SQLite `Pending`, so `id` is never sent again, and its single map entry in `e.observationAttempts` is deleted at the top of the next `flush` (`observe.go:316-320`). Proven in `TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap` (`node/agent/review_r32_test.go:116-193`).

### Question 5: The Record Fields in `observation_log.go`
- **`http_status` (`node/agent/observation_log.go:68, 71`): `status` is initialized to `202` when `err == nil`. Is that the exact status Central sends on every success path, replays included?**
  Yes. On the client side, `controlclient.Client.Observe` (`node/internal/controlclient/api.go:103-112`) calls `c.postJSON(ctx, "/api/v1/agent/observations", req, &out, opts, http.StatusAccepted)`, which checks `if resp.StatusCode != want { return nil, fmt.Errorf("controlclient: %s status %d", path, resp.StatusCode) }` (`node/internal/controlclient/api.go:220-222`). Any non-`202` response (even another 2xx like `200 OK`) returns a non-nil `err` (`http_status: 0, outcome: "transport_error", transport_error_class: "other"`). On the server side, first acceptance (`internal/nodes/httpwire/handlers.go:343`, `internal/nodes/observations.go:177`), body-level duplicate `(node_id, incarnation_id, report_id)` replay (`internal/nodes/observations.go:147-155`), and `Idempotency-Key` receipt replay (`internal/nodes/httpwire/replay.go:24-47`) all return `SubmitObservation202JSONResponse` (`HTTP 202`).
- **`outcome` (`node/agent/observation_log.go:69-88`): when can `unconfirmed_response` occur? Is `accepted_report_ids` correct when Central returns a report ID other than the one sent?**
  `unconfirmed_response` occurs when the server returns HTTP `202` with a valid JSON body whose `accepted` boolean is `false` (`err == nil && !bool(accepted.Accepted)`, `observation_log.go:71-76`), in which case `accepted_report_ids` is `[]` (empty slice). When `bool(accepted.Accepted)` is `true`, `acceptedIDs = append(acceptedIDs, accepted.ReportId)` (`observation_log.go:72-73`) records `accepted.ReportId` — the exact ID the response claimed to accept (`docs/node/README.md:189-190`) — while `acceptRecoveryReport` (`node/agent/recovery_renewal.go:73`) still requires `accepted.ReportId == rep.ReportId && bool(accepted.Accepted)` before marking a closed-recovery report accepted.
- **`error_code` (`node/agent/observation_log.go:70, 82-85`): can a non-`api.ErrorCode` string or a message body from `StatusError` reach the record?**
  No. `controlclient.StatusError` (`node/internal/controlclient/client.go:39-48, 344-362`) extracts only `env.Error.Code` from the response body and discards `env.Error.Message` and `env.Error.Details`. In `logObservationSubmission` (`node/agent/observation_log.go:82-85`), `if code != "" && !api.ErrorCode(code).Valid() { code = "unknown" }` replaces any string outside the generated `api.ErrorCode` enum with `"unknown"`.
- **`observationErrorClass` (`node/agent/observation_log.go:38-65`): are all returned classes finite, and is the order (`DeadlineExceeded` -> `net.Error.Timeout()` -> `Canceled` -> `ECONNREFUSED` -> `ECONNRESET` -> `EOF` -> `DNSError` -> `OpError` -> `other`) free of cases where an error text, URL, host or IP could leak?**
  Yes. `observationErrorClass` returns only two constant strings from finite sets (`outcome` in `{"timeout", "transport_error"}` and `class` in `{"deadline_exceeded", "network_timeout", "canceled", "connection_refused", "connection_reset", "connection_closed", "dns", "network", "other"}`). It never inspects `err.Error()`, `url.Error.URL`, `net.DNSError.Name`, or `net.OpError.Addr`. The ordering is also well-structured: `context.DeadlineExceeded` (which implements `net.Error.Timeout()`) is matched before generic `ne.Timeout()`, specific errnos (`ECONNREFUSED`, `ECONNRESET`) and `*net.DNSError` (which `net.Dialer` wraps inside `*net.OpError`) are matched before the fallback `*net.OpError` case.
- **`next_retry_at`, `retry_after_ms`, `duration_ms`, `process_id`, `boot_id`:**
  - `next_retry_at` (`observation_log.go:89-91`) formats `retryAt = e.clock.Wall().Add(wait)` (`observe.go:403`), using the exact `wait` added to `e.observationRetryAt = e.clock.Mono() + wait` (`observe.go:402`).
  - `retry_after_ms` (`observation_log.go:81, 100`) is `se.RetryAfter.Milliseconds()` (`0` when absent).
  - `duration_ms` (`observe.go:364-366`, `observation_log.go:94`) measures `time.Since(timer).Milliseconds()` across `e.client.Observe`.
  - Across a same-boot process restart, `boot_id` (read from `/proc/sys/kernel/random/boot_id` via `watchdog.ReadBootID`, `node/agent/engine.go:303`) stays constant, `process_id` (`os.Getpid()`, `observation_log.go:92`) reflects the process PID, and `attempt_numbers` starts at `1` (`attempt_scope: "process"`), never using the SQLite outbox `attempts` counter (which bumps twice per failed request). Proven in `TestF52_ObservationSubmissionRecords` (`node/agent/observation_log_test.go:146-167`) and `TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses` (`node/agent/review_r32_test.go:270-448`).

### Question 6: The Proofs
- **In `node/agent/review_r31_test.go`, Case C's setup changed (`node/agent/review_r31_test.go:181-189`). Does the changed test still faithfully test `r31`'s Case C, and would it still fail on the unfixed code?**
  Yes to both. In `node/agent/stale_hold.go:13, 43`, `updateStaleHold` raises `e.staleHold.alarm = true` once `h.elapsed >= 30s` (`staleHoldDuration`), and `observationState()` (`node/agent/observe.go:455`) includes `e.staleHold.alarm`. In `r31`'s original loop (35 iterations of 10s starting from `elapsed == 0`), iteration 3 crossed `30s`, flipping `staleHold.alarm` from `false` to `true` and legitimately enqueuing one state-change report (`len(afterPending) == 2 > len(beforePending) == 1`). Advancing `staleHoldDuration` (30s) and calling `observe` once before capturing `beforePending` (`len(beforePending) == 2`, with `r.eng.staleHold.alarm == true`) isolates steady-state Case C after the `staleHold` transition; on the unfixed code (`341d099a^`), `observe.go:39` checked only `e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil`, so Case C still failed on `341d099a^` (`len(afterPending) == 32 > len(beforePending) == 2`) and passes on `341d099a`.
- **In `node/agent/followup52_test.go`, do the lane's proofs exercise the real `Step` / `observe` / `flush` path, and would they catch the cost in Question 2?**
  `TestD157_F52_IsolationLatchStepsOutlast32Intervals` (`node/agent/followup52_test.go:34-100`) and `TestD157_F52_StaleLeaseStepsOutlast32Intervals` (`followup52_test.go:102-151`) exercise 210 full `Step` calls across 35 observation intervals (at 5s cadence), and `TestD157_F52_CatchupBindsWaitingRecovery` (`followup52_test.go:153-189`) exercises `Step` catch-up binding and renewal. However, none of those tests inspect `e.lastObs` on skipped rebuilds while `closedRecoveryRenewal` is armed and unbindable; our new `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost` (`node/agent/review_r32_test.go:29-106`) directly proves that behavior.
- **In `node/agent/observation_log_test.go`, which `observation_log.go` paths were untested (`unconfirmed_response`, `409`/`422` conflict, `403` revoked, `network_timeout`, `canceled`, `connection_reset`, `connection_closed`, `dns`, `network`, `other`)?**
  `observation_log_test.go` tested `accepted` (`202`), `503`, `429`, `connection_refused`, `deadline_exceeded`, and unknown `StatusError.Code`, leaving `unconfirmed_response`, mismatched `accepted.ReportId`, `409`/`422` conflict continuation, `403 IDENTITY_REVOKED`, `IdentityFrozen`, and the remaining 7 `observationErrorClass` branches (`network_timeout`, `canceled`, `connection_reset`, `connection_closed`, `dns`, `network`, `other`) untested. All of those paths are now covered in `TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap` and `TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses` (`node/agent/review_r32_test.go:115-448`).

### Question 7: Interactions
- **Can the narrowed exception in `observe.go` strand a recovery that could bind later (for example, after an isolation error clears or after a lease catches up), or produce two early renewals in one D-067 slot?**
  No to both:
  1. **No stranded recovery on later state transition:** When a blocking condition clears (e.g., `containment` clears, `meter_path` recovers, or `lease.ConfigGeneration` catches up to `appliedGen`), `e.observationState()` changes (`state != e.lastObservationState`), or if `state == e.lastObservationState` while earlier items remain pending in the outbox, `observe.go:47-56` re-evaluates `e.report(ctx, wall.UTC())` and `e.canBindRecoveryReport(rep)` on the very next due observation call, sees `canBindRecoveryReport(rep) == true`, enqueues the report, and binds `e.closedRecoveryRenewal.reportID = rep.ReportId` (`observe.go:75`).
  2. **At most one early renewal per closed recovery:** `canBindRecoveryReport` requires `r.armed && r.reportID == uuid.Nil` (`recovery_renewal.go:37`), `noteRecoveryReport` sets `r.reportID = rep.ReportId` once (`recovery_renewal.go:30`), `acceptRecoveryReport` sets `r.accepted = true` only when `accepted.ReportId == r.reportID && bool(accepted.Accepted)` (`recovery_renewal.go:73`), and `syncLease` (`node/agent/sync.go:119-121`) clears `e.closedRecoveryRenewal = recoveryRenewal{}` immediately upon dispatching the renewal request. Even if Central returns another grant-less lease, `closedRecoveryRenewal` is disarmed and subsequent renewals follow D-067 backoff. Proven in `TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal` (`node/agent/review_r32_test.go:455-512`).

---

## 5. What Held

1. **Invariant 1 (D-157's renewal):** A clean closed recovery still binds its clean post-recovery report even when an earlier pre-recovery report is pending in the outbox, and dispatches at most one early renewal once Central returns `202 Accepted` for that bound report (`TestD157_F52_CatchupBindsWaitingRecovery`, `TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal`).
2. **Invariant 2 (Dedup):** When a closed recovery is armed and unbound (`reportID == uuid.Nil`) and `state == lastObservationState` with pending outbox items, `observe` skips queuing whenever `!e.canBindRecoveryReport(rep)`, preventing outbox flooding across Cases A, B, and C (`TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox`, `TestD157_F52_IsolationLatchStepsOutlast32Intervals`, `TestD157_F52_StaleLeaseStepsOutlast32Intervals`, `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost`).
3. **Invariant 3 (No new authority):** Neither the dedup narrowing nor the structured observation log records alter forwarding authorization, quarantine, stale-hold, or lease renewal cadence (`TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost`, `TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal`).
4. **Invariant 4 (Delivery unchanged):** `submitObservation` preserves exact outbox state transitions, backoff, jitter, `Retry-After` handling, `409`/`422` conflict continuation, `403` revocation, and `IdentityFrozen` handling (`TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap`).
5. **Invariant 5 (Records leak nothing):** `observation enqueued` and `observation submission` log only fixed-schema fields, finite error classes, and validated `api.ErrorCode` tokens, never leaking URLs, hostnames, IP addresses, secrets, or error strings (`TestF52_ObservationSubmissionRecords`, `TestF52_ObservationRecordsDoNotExposeErrorText`, `TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses`).
6. **Invariant 6 (Records are exact):** Every documented field of `observation enqueued` and `observation submission` in `docs/node/README.md:151-220` matches the runtime values produced by `node/agent/observation_log.go` (`TestF52_ObservationEnqueueReasons`, `TestF52_ObservationSubmissionRecords`, `TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses`).

### Proof Test Output

```text
=== RUN   TestD157_F52_IsolationLatchStepsOutlast32Intervals
--- PASS: TestD157_F52_IsolationLatchStepsOutlast32Intervals (1.87s)
=== RUN   TestD157_F52_StaleLeaseStepsOutlast32Intervals
=== RUN   TestD157_F52_StaleLeaseStepsOutlast32Intervals/bindings
=== RUN   TestD157_F52_StaleLeaseStepsOutlast32Intervals/empty_desired
--- PASS: TestD157_F52_StaleLeaseStepsOutlast32Intervals (0.88s)
=== RUN   TestD157_F52_CatchupBindsWaitingRecovery
--- PASS: TestD157_F52_CatchupBindsWaitingRecovery (0.18s)
=== RUN   TestD157_F52_PendingRevokeStepsOutlast32Intervals
--- PASS: TestD157_F52_PendingRevokeStepsOutlast32Intervals (0.39s)
=== RUN   TestF52_ObservationSubmissionRecords
--- PASS: TestF52_ObservationSubmissionRecords (1.80s)
=== RUN   TestF52_ObservationEnqueueReasons
--- PASS: TestF52_ObservationEnqueueReasons (0.31s)
=== RUN   TestF52_ObservationRecordsDoNotExposeErrorText
--- PASS: TestF52_ObservationRecordsDoNotExposeErrorText (0.01s)
=== RUN   TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox
--- PASS: TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox (0.57s)
=== RUN   TestR31_Held_H1_RecoverLifetimesAndFailClosedGate
--- PASS: TestR31_Held_H1_RecoverLifetimesAndFailClosedGate (0.14s)
=== RUN   TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport
--- PASS: TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport (0.21s)
=== RUN   TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost
--- PASS: TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost (0.17s)
=== RUN   TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap
--- PASS: TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap (0.39s)
=== RUN   TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses
--- PASS: TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses (0.19s)
=== RUN   TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal
--- PASS: TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal (0.21s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	7.326s
```

---

## 6. Not Examined

- Live privileged Docker `tests/netlab` multi-container `AT02` soak execution (this review task is unit-level code & proof review on `node/agent`).
- Real kernel `nftables` / `tc` / WireGuard data-plane execution and physical `arm64` hardware runs.

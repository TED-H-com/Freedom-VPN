# Follow-up 49 — #482–#485; Node CI fixes & r33, clone receiver repair, D-164 contract phase, D-150–D-163 confirmed, NetBird #8473 sample

Verified against main **aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f**, through
**contracts revision 15 #485 / 82dda8a867989e512e36a80dfba5aac1e4fdc28f**, on
**2026-11-05 EST** (`2026-11-05T10:04:33Z`). Follow-up 48 (#483) ended at
**#481 / b921c75f180beb6c0051d585a86d6651b2c8d71a**. Exactly **four merges and
four final roadmap rows** are on `main`: **#482, #483, #484, #485**.

**Unexpected lane included atomically in the cut (`#485`) and post-cut status:**
`#485` (`w3-contracts-r13` follow-up 15, merge
`82dda8a867989e512e36a80dfba5aac1e4fdc28f` at `2026-11-05T10:04:30Z`, final
roadmap row `aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f` at
`2026-11-05T10:04:33Z`) merged outside the original three-lane list (`#482`,
`#483`, `#484`) before this handover started. Per the brief's atomic-inclusion
rule (`"Do not split #485 from its roadmap row or leave a merged PR on main
unreconciled"`), `#485` and its roadmap row are included in full in this cut,
and zero commits exist on `main` after `aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`
([`after-cutoff-provenance.json`](../../tests/handover/testdata/history/follow-up-49/after-cutoff-provenance.json)).
At `10:17Z` (`05:17 EST`), all **16** GitHub Actions workflow runs for `#485`'s
merge commit `82dda8a867989e512e36a80dfba5aac1e4fdc28f` and all **12** workflow
runs for the roadmap commit `aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f` had
completed with `conclusion: "success"`. Separately, **status only, outside this
fixed `10:04:33Z` cut:** `w4-at-isolation` follow-up 27 (offline delivery
`42012fc4`, record correction `3cfd4e45`, mechanically integrated as
`cda27dcbc41b0a7baf65c50a4f906b4fab634b73`) later completed its single live run
(`d69f5652-aed6-4fdd-9e7c-edface751237`) at `10:33:52Z` with exit `2`
(`completed_failed_awaiting_evidence_delivery`, not merged on `main`, not
imported into this cut). No new GitHub scheduled `at-isolation` run occurred in
`#482`–`#485`; the latest scheduled run remains `41836274140` (`2026-11-04T06:41:59Z`
on `d2cb544692b5812c0eeb7dcb08c9868ab16b0c9d`, `failure`). No date is promised.

**D-098 / D-109 / D-132:** `TestHandover_Followup49Claims` pins the four final
roadmap rows, verbatim PR excerpts, **49 frozen snapshots** (five `#483`
documents, five `#485` documents, two shared indices, one devstack 37 analysis
note, 17 agent-core 53 note and evidence files, 13 contracts revision 15 docs,
vectors and examples, three frozen `52b93845` members retrospective files, and
three review `r33` proof files), **68 Go test declarations on `main`**, **23
frozen product source readings**, **67 archived artifacts**, exact JSON values
decoded with `json.Decoder.UseNumber()`, review `r33` provenance and proof
files, owner-confirmation provenance for `D-150`–`D-163`, and the posted NetBird
discussion `#8473` comment `19411545` provenance. All lane-document quotations
use committed frozen copies. No claims read living isolation, members, secops or
usage evidence roots. Follow-up 48's and `TestHandover_ContractsRevision15MigrationIntegrity`'s
rewritten inputs now read frozen `#483` (`ff0ce0658b9a86523bd6bc2ddf1fceda47a5453f`)
and `#485` (`aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`) bytes; their assertions
and every earlier claims test (**42** `TestHandover_Followup*Claims` functions
`TestHandover_Followup7Claims` through `TestHandover_Followup48Claims` plus
`TestHandover_ContractsRevision15MigrationIntegrity`) remain unchanged.

## Handover 48 (#483): previous cut, actual inventory and session

[PR #483](https://github.com/TED-H-com/Freedom-VPN/pull/483) (native delivery
`e221cf24479c9cd99eb87cbf14c454cd19fb5c1b`, integrated tree
`f6b4849b7dc548f64ddf01daa29f07dac7d5071a` after merging `#482`'s roadmap row
`729f914254211509c838f2efe73d3e8936bcf7e0` with zero file conflicts because
`#482` touched only `tests/e2e/second_exit*.go` and
`docs/e2e/runs/2026-11-04/followup37/analysis.md`, merge
`ff0ce0658b9a86523bd6bc2ddf1fceda47a5453f`, final roadmap row
`980e313ab8d1b261cb1f44e4a86b54b7e55fefde`) records the **#478–#481 cut, up to
b921c75f180beb6c0051d585a86d6651b2c8d71a**. On `main`,
`TestHandover_Followup48Claims` has **16 subtests** (ten direct `t.Run` blocks
and six claim groups in `tests/handover/testdata/followup48-claims.json`),
**four final roadmap rows and PR excerpts**, **23 byte-identical frozen inputs**
(nine `#479` documents, three producer dated notes and eleven `F24-*.json`
records), **22 Go test declarations on `main`**, **22 frozen source readings**,
**62 archived artifacts** and **122 exact JSON values** decoded as
`json.Number`. Its **41 earlier claims functions**
(`TestHandover_Followup7Claims` through `TestHandover_Followup47Claims`) stay
exact after reversing only the nine input redirects in
`restoreFollowup48Assertions`. Its register contains **66 decisions** (`0`
added, `2` updated: `D-162` and `D-163`, `64` byte-identical to `#479`) and
**138 findings rows** (`6` updated, `2` added: `Node workflow agent test
failures` and `Isolation 26 clone receiver wiring (9445 vs 9444)`, `130`
byte-identical to `#479`), archives `docs/reviews/r32-dedup-records-review.md`
byte-for-byte from `21e59029934a34f7125805e116a2ffaaf61b60cb` (`201` lines,
`31,981` bytes, SHA-256
`5b7ec63eecbe64e278e16bf24ef3a8e9ea3e625d6bde701d6d0c0e5735e1960f`, `0`
findings) with frozen proof copy
`tests/handover/testdata/history/follow-up-48/review-node_agent_review_r32_test.go.txt`
(`20,346` B, SHA-256
`8625fecccc518a49c10551bbd63d26be2807dc60b558816855182cb5d7ad999a`), pins all
six `Node` workflow `node/agent` failure runs in
`tests/handover/testdata/history/follow-up-48/node-workflow-failures.json`, and
refreshes **241 SHA-256 digest scalars** across **11 referenced paths** in
`docs/e2e/at/secops/scope-review.json` (**238** owned-path updates across **10**
paths plus **3** repairs of stale pins for the unchanged
`docs/e2e/at/secops/history/index.json`). These are `#483`'s verified counts,
not this cut's changes.

The delivery, attributed to `#483`'s body: starting head
`6cea09778ad13b8481005ef098a7b69a9de9eab1` and narrow correction commit
`e221cf24479c9cd99eb87cbf14c454cd19fb5c1b` (**codex/gpt-6-astra**), touching
**52 owned files (`+102,278 / -341` lines)** after the orchestrator's
integration check with devstack 37 (`44227926027cf0baf7cc06a4de374e09d75b20cc`)
exposed the frozen-source/current-source equality dependency in
`TestHandover_Followup48Claims/22_frozen_source_readings_and_predicate_boundaries`
(`tests/e2e/second_exit.go`); the narrow correction removed that equality and
refreshed two `tests/handover/handover_test.go` SHA-256 scalars, after which all
checks passed on integrated tree `f6b4849b7dc548f64ddf01daa29f07dac7d5071a`.
Docs, claims tests, frozen snapshots and digest values only; no Docker or live
acceptance run by handover.

## Devstack 37 (#482): offline D-141 diagnosis, clone `control_url` repair and 30-file historical audit

[PR #482](https://github.com/TED-H-com/Freedom-VPN/pull/482) (branch commit
`44227926027cf0baf7cc06a4de374e09d75b20cc`, merge
`73f2f714c0d37c0ceca4b6720dd9afb9b7c61ba7`, final roadmap row
`729f914254211509c838f2efe73d3e8936bcf7e0`) and the
[frozen analysis note](../../tests/handover/testdata/history/follow-up-49/docs_e2e_runs_2026-11-04_followup37_analysis.md.txt)
(`28,650` bytes, SHA-256
`53440268f993b0e69696ff9e818ddf15f8e5b3ef033b376e03e52e3e7f09fee4`) answer
follow-up 48's routed hand-off (`"Route shared second-exit clone receiver wiring
(:9445 vs :9444) to w3-devstack-e2e"`). Scope is **offline only**: `3` changed
files (`+386 / -6` lines: `docs/e2e/runs/2026-11-04/followup37/analysis.md`,
`tests/e2e/second_exit.go`, `tests/e2e/second_exit_test.go`), no Docker, no live
stack, no reservation and no historical `AT10`–`AT13` re-score.

**D-141 static diagnosis confirmed:**
- `tests/e2e/cmd/fvpn-e2e-prepare/main.go:401-405` writes two distinct
  `control_url` values on the prepared exit node: `/etc/vpn-node/agent-enroll.json`
  uses `https://portal.example.invalid:8443`, while `/etc/vpn-node/agent-run.json`
  uses `https://127.0.0.1:9444` (the local `fvpn-e2e-nodeconfig` proxy at
  `tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:120,140`, started by
  `deploy/devstack/Dockerfile.node`).
- In isolation (`tests/e2e/at/isolation/observer_main.go:95,114-132`, introduced
  in `w4-at-isolation` follow-up 16 `#399` merge
  `8afa69c5353c8518c69294daec997ba261eba65f` / F16 build
  `a8394027002771be64f023d062ea47b2da1447d8`), `observeProxy` listens on
  `127.0.0.1:9445` inside the main exit container only (`observer_main.go:95`)
  and `configureObservation()` rewrites the main container's
  `/etc/vpn-node/agent-run.json` `control_url` to `https://127.0.0.1:9445`
  (`observer_main.go:125` at `4c495c3b1516022e1de8db2f864432a8a8d171c5`).
- Before `#482`, `startSecondExit` / `startExitSlot` (`tests/e2e/second_exit.go:20-127`
  at `b921c75f`) copied the source container's `/etc/vpn-node` (`second_exit.go:59`)
  and `/var/lib/vpn-node` (`second_exit.go:98-119`, including `var/e2e/agent-*.json`)
  while rewriting only `datapath.snat` on `etc/agent-enroll.json` and `etc/agent-run.json`
  and leaving `control_url: "https://127.0.0.1:9445"` intact in the clone container,
  which starts `fvpn-e2e-nodeconfig` (`127.0.0.1:9444`) and never starts
  `observeProxy` (`127.0.0.1:9445`). The clone's runtime agent therefore dialed
  `https://127.0.0.1:9445` (closed in the clone's network namespace), while
  `EnrollAt` in `AT12` (`observer_main.go:1076`) continued using
  `/etc/vpn-node/agent-enroll.json` (`https://portal.example.invalid:8443`).

**Offline repair in `tests/e2e/second_exit.go` and
`tests/e2e/second_exit_test.go` (including Fix round 1 `a650a4e3` / `44227926`):**
- `prepareExitSlotConfigs` (`tests/e2e/second_exit.go:162-182`, called from
  `startExitSlot` at `second_exit.go:73-75` and after the clone copy at
  `second_exit.go:124-126`) calls `rewriteExitConfigFile` (`second_exit.go:184-194`)
  and `rewriteExitConfig` (`second_exit.go:196-231`) with
  `localNodeControlURL = "https://127.0.0.1:9444"` (`second_exit.go:156`):
  1. `etc/agent-enroll.json` rewrites `datapath.snat` to the slot IPv4
     (`198.51.100.22` for fresh `node2`, `198.51.100.23` for `node-clone`) and
     preserves `control_url` (`"https://portal.example.invalid:8443"`) and all
     other fields (`observation_seconds: 5`) unchanged in both `clone` and `fresh`
     modes (`second_exit.go:163-168,211-229`);
  2. `etc/agent-run.json` rewrites `datapath.snat` to the slot IPv4 and rewrites
     `control_url` to `"https://127.0.0.1:9444"`, preserving all other fields
     (`observation_seconds: 1` when copied from isolation's source runtime config);
  3. In `clone` mode (`second_exit.go:169-181`), copied release configs matching
     `var/e2e/agent-*.json` call `rewriteExitConfigFile(path, "", true)` so only
     `control_url` is rewritten to `"https://127.0.0.1:9444"` while preserving
     copied `datapath.snat` (`"198.51.100.20"`) and every other JSON value
     unchanged (`second_exit.go:209-222`);
  4. Copied identity and durable-state files (`etc/machine-id`,
     `var/identity/node.json`, `var/identity/cert.pem`, `var/identity/key.pem`,
     `var/state/agent.db`, `var/e2e/agent-selected`) are never modified.
- Proven offline by `TestD141_ClonePreparedConfigsDialContainerResolverProxy`
  (`tests/e2e/second_exit_test.go:16-214`), verifying both `clone` (`198.51.100.23`,
  with `etc/agent-enroll.json`, `etc/agent-run.json`, and
  `var/e2e/agent-00000000-0000-7000-8000-000000000001.json`) and `fresh`
  (`198.51.100.22`) across both `"formatted"` and `"compact"` JSON inputs
  (`second_exit_test.go:128-186`), full decoded value preservation with
  `json.Decoder.UseNumber()`, byte-for-byte preservation of all six `durableFiles`
  (`second_exit_test.go:82-89,120-127,183`), and `18` `refuse/<invalid>/<path>`
  subtests (`malformed`, `null-config`, `array-config`, `missing-datapath`,
  `null-datapath`, `array-datapath` across all three paths,
  `second_exit_test.go:189-213`) proving rejected configs are not overwritten and
  durable bytes stay intact.

**Complete 30-file historical `identity-freeze.json` audit (`analysis.md` §3):**
- Audits all **30** committed `identity-freeze.json` and `identity-freeze.json.gz`
  files across `docs/e2e/at/isolation/` (not 30 distinct live runs: `F7` through
  `F25`, plus top-level `identity-freeze.json` in `docs/e2e/at/isolation/` matching
  `history/follow-up-25-run/identity-freeze.json` for run `bd73d7a4`):
  - **Pre-`#399` (`F7`–`F15`):** no `observeProxy` port-`9445` split existed.
    `F7` (`041e9e48`) has `source_requests_total: 4`, `clone_requests_total: 3`;
    `F8`–`F15` have `source_requests_total: 3`, `clone_requests_total: 2`.
  - **Post-`#399` (`F16`–`F25` and top-level):** every file has
    `source_requests_total: 90` and `clone_requests_total: 90`, `quarantined:
    true`, `quarantine_reason: "identity_frozen"`, `restart_forwarded_count: 0`,
    and `restart_quarantined: true`. Why `90/90` passed despite the `:9445`
    runtime config mismatch: the `90` `clone_requests` in `AT12`'s
    `identity-freeze.json` are explicit harness HTTP calls made by
    `observer_main.go:880-955` from the test process directly to
    `https://127.0.0.1:18443` using the cloned mTLS keypair (`cloneC`), not
    background daemon traffic from the second-exit container's
    `fvpn-agent run` process.
  - **Duplicate capture (`F19` vs `F18`):** `F19`
    (`docs/e2e/at/isolation/history/follow-up-19/identity-freeze.json.gz`) has
    git blob `1075e8ca2190ae38899fd36dc3a46b4eac641736`, byte-identical to `F18`
    (`docs/e2e/at/isolation/history/follow-up-18/identity-freeze.json.gz`),
    because `F19` was a focused `D-147` ICMP run that copied `F18`'s
    `identity-freeze.json`.
  - **Distinct post-port-change `stop-receipt-review.json` files:** only `F16`
    (`2083a6d6`), `F17` (`9e7ac777`) and `F18` (`6f6b532f`, blob
    `d429b8d5251e6834f959a129d3a200007f6b1a43`) carry distinct post-`#399`
    `stop-receipt-review.json` captures (`F19`–`F24` and the root copy reuse
    `F18`'s blob `d429b8d5251e6834f959a129d3a200007f6b1a43`; `F25` replaced
    `stop-receipt-review.json` with `restart-timeline.json` and
    `restart-signature-review.json`, which track only the main `vpn-node`
    container and not the second-exit container).
  - **Explicit evidence limits:** historical second-exit container logs and
    clone daemon dial errors were never persisted (`docker rm -f` in
    `second_exit.go:167` removed the container on stop), and which release binary
    the clone ran in each historical run is not recorded in `identity-freeze.json`.
    No historical `AT10`–`AT13` verdict is re-scored.
  - **Remaining gaps routed to `w4-at-isolation`:** live `AT12` clone daemon
    transport and consumer-latch validation on the repaired `:9444` runtime
    config, plus before/after-stop second-exit container log retention
    (`analysis.md` §3.4 / §4).

## Agent-core 53 (#484) and Review r33: six Node CI failures fixed, meter restart checkpoint, and F17-01 / F31-01 design notes

[PR #484](https://github.com/TED-H-com/Freedom-VPN/pull/484) (native delivery
`2936cfec94c1b804bef1c47e1cff2d20d286b535`, mechanically integrated on `main`
`980e313a` as `9a26725fd054e9e2ec287a0203fcf99d3d219ed8` with zero code changes
after the `r33` review base `29cd267162fa614c444122865b62f405e2f58802`, merge
`633f0250fc3acc02a0d0c9ffdaf35292358764f4`, final roadmap row
`6520a31255df40a0ad6f2b5b22b6d93d73b0157a`) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-2026-11-04.md.txt)
(`65,527` bytes, SHA-256
`952e09cb9af95c699c0900cd074caf2e82db040dd7595d8b3635d23578e6076f`, with 16
frozen evidence files under `tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_*`)
answer follow-up 48's first routed hand-off (`"Carry w2-agent-core follow-up 53
for the 6 open Node workflow node/agent test failures across runs 41782791081,
41787173794, 41794009264, 41808885873, 41871089749 and 41885624816, plus
F17-01's and F31-01's design notes"`).

### 1. Diagnosis and repair of the six `Node` workflow `node/agent` failures

All six failing runs (`tests/handover/testdata/history/follow-up-48/node-workflow-failures.json`
and [`docs_node_followup53-evidence_ci-metadata.json`](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_ci-metadata.json))
are diagnosed first across four root causes:

1. **Boot local-deadline failures under short test budgets (`41782791081`,
   `41794009264`, `41871089749`):**
   - Run `41782791081` (`0cf5a5d5`, `158.961 s` total package time):
     `TestR22_M3_FailedOrTimedOutBootAndProbeClearLeaveUntruthfulHeartbeat/term_after_baseline_with_ledger_timeout_leaves_stale_serving_true_heartbeat`
     (`review_r22_test.go:764`, `6.14 s`).
   - Run `41794009264` (`84e71988`, `87.316 s`):
     `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery`
     (`review_r29_test.go:265`, `0.62 s`).
   - Run `41871089749` (`014b59ab`, `158.947 s`):
     `TestD155_CrashedTransactionsKeepAgentRunning/blocked`
     (`followup49_transaction_test.go:240`, `2.65 s`).
   - **Classification:** environmental scheduling and SQLite/tempdir I/O latency
     exposure on artificially shortened test `Boot` budgets (`80 ms` in
     `review_r22_test.go`, `600 ms` in `review_r29_test.go` and
     `followup49_transaction_test.go`), where pre-`Boot` setup or child-process
     spawn exhausted the short budget before the intended test checkpoint.
   - **Repairs (test-only; `node/agent/local_budget.go` is net-unchanged from
     `b921c75f180beb6c0051d585a86d6651b2c8d71a`, keeping production
     `bootLocalBudget = 10 * time.Second` and `closeLocalBudget = 3 * time.Second`):**
     - `node/agent/followup49_transaction_test.go`: `TestD155_CrashedTransactionsKeepAgentRunning`
       uses normal `Boot` (`10 s` local budget) instead of the `600 ms` override.
     - `node/agent/review_r29_test.go`: `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery`
       uses normal `Boot` and gates its attestation helper on both the health
       challenge file and a `bootReturned` atomic marker so the helper can only
       answer after `Boot` returns cleanly while `recoverChild` is still running.
     - `node/agent/review_r22_test.go` and `node/agent/followup38_test.go`:
       `TestR22_M3_FailedOrTimedOutBootAndProbeClearLeaveUntruthfulHeartbeat` and
       `TestR21_M2_TermAfterBaselineKeepsNormalStopOnLedgerTimeout` cancel the
       parent context after `Baseline` and let `blockSource` wait for the real
       `10 s` `bootLocalBudget` to expire inside the ledger stage after entering
       `source.Counters(ctx)` (`entered` channel), proving the exact
       `bootLocalBudget` expiry path without an `80 ms` pre-baseline race.

2. **Refused same-boot adoption on unchanged restart checkpoint (`41808885873`):**
   - Run `41808885873` (`48ee50ec`, `86.265 s`):
     `TestR23_H1b_GuardMissingIdleCounter_EpochSwitchKeepsAdoption`
     (`review_r23_test.go:277`, `0.09 s`).
   - **Classification:** product defect in `node/internal/meter` exposed when
     `SwitchEpoch` at `t_1` (`sample(ctx, false)`) and `PreserveServingOnStop` ->
     `CheckpointForRestart` at `t_2` (`sample(ctx, false)` with `m.inFinal = true`)
     read identical `time.Now().UTC().Truncate(time.Microsecond)` values
     (`t_2 == head.LastSample`) or experience a clock rollback within one
     sampling interval (`0 <= head.LastSample.Sub(now) <= interval`) with
     unchanged counter readings (`sameReadings(head.Counters, cur)`).
     Previously, `planSample` (`node/internal/meter/delta.go:264-275`) only
     allowed `if final && sameReadings(head.Counters, cur)` to commit a zero-event
     checkpoint at `head.LastSample`; for `CheckpointForRestart` (`final == false`),
     it fell through to `ReasonRollback`, emitting a `counter_discontinuity` gap
     and rotating `OpenEpoch` from `E_1` to `E_2`, which caused the successor's
     `Boot` (`node/agent/engine.go`) to refuse same-boot adoption because
     `usage_apply_intent` still named `E_1`. The exact nanosecond remainder in
     CI run `41808885873` was not logged (`failure_truncated: true` in
     `node-workflow-failures.json`); same-microsecond truncation and small clock
     rollback are proven deterministically by unit tests.
   - **Production fix (`node/internal/meter/delta.go` blob
     `112260bd296d31cfd382d4faf03bf1efe50d7729`, `15,049` B, `513` lines, SHA-256
     `5743c7c6feb6dfed0c7b4f39aa489077d1966de66cd06b57fe0e051784779226`;
     `node/internal/meter/meter.go` blob
     `ac6ff0b548b5c0bdd6a3c5e44ad4b04d34dec3f9`, `13,606` B, `545` lines, SHA-256
     `e5adf83a314abb4171d83a5f7e92203a440c6c621ba403fb3e242e60aa0fd219`):**
     - `planSample` (`delta.go:243-245`) delegates to
       `planSampleWithBarrier(head, obs, now, interval, final, restart)`
       (`delta.go:247-341`), and `m.sample` (`meter.go:216,245`) passes
       `restart := m.inFinal && !final` (`true` only inside
       `CheckpointForRestart`, `meter.go:172-199`).
     - In `planSampleWithBarrier`, after `head.OpenEpoch == uuid.Nil`
       (`delta.go:249-255`) and `bootChanged(head.BootID, obs.BootID) ||
       len(resets) > 0` (`delta.go:256-263`), when `!head.LastSample.IsZero() &&
       !now.After(head.LastSample)` and `head.LastSample.Sub(now) <= interval`
       (`delta.go:264-284`), line `270` checks
       `if (final || restart) && sameReadings(head.Counters, cur)` and returns a
       zero-event `usagestore.Commit{Epoch: head.OpenEpoch, Counters:
       head.Counters, BootID: obs.BootID, SampledAt: head.LastSample}`
       (`CloseEpoch: true` only when `final`, keeping `OpenEpoch` open when
       `restart`). Any changed counter (`!sameReadings`), rollback exceeding
       `interval`, `bootChanged`, or counter reset (`enforce.DetectResets`) still
       closes/rotates the epoch with a gap (`TestCheckpointForRestartAtSameMicrosecond`
       in `node/internal/meter/meter_test.go` and
       `TestMTR03_RestartCheckpointClockBoundaries`,
       `TestMTR03_RestartCheckpointOutsideClockWindow`,
       `TestMTR03_RestartCheckpointResetAndIdentityPrecedence`, and
       `TestMTR03_FinalSampleUnchangedRollback` in
       `node/internal/meter/followup53_test.go`).

3. **Duplicate retired-stream post in `TestD118_ENR05_VerifiedReenrollmentHandsOffFinalSample/delivered_ACK` (`41787173794`):**
   - Run `41787173794` (`eea0abd6`, `78.902 s`): `followup25_usage_test.go:295:
     retired posts=2, accepted batches=1, accepted events=1` (`0.09 s`).
   - **Classification:** test coordination race in
     `node/agent/followup25_usage_test.go:123-136,165` (exact goroutine
     interleaving in CI run `41787173794` is inferred, not directly logged).
     Re-enrolling the successor agent starts background uploader work while the
     test server's `Unavailable` counter was decremented on the first `503`
     without waiting for that initial `503` attempt to complete before switching
     the fake server to `200`, or during `Close` when `Unavailable` was left at
     `99`.
   - **Repair (`node/agent/followup25_usage_test.go:123-136,165`):** records
     `unavailBefore := r.srv.UsageReport().Unavailable` and waits up to `5 s` for
     `r.srv.UsageReport().Unavailable` to change (`"initial retired upload did not
     reach unavailable server"`) before calling `stopUsageLoops(t, r)`, and calls
     `r.srv.SetUsageUnavailable(0)` after `next.Close()` before restarting the
     agent.

4. **Held `FinalSample` fixture window in `TestR22_Held_Q2_Q3_HeldFinalSampleWaitsInStepAndAdoptionSuppressesProbeException/held_finalsample_blocks_step_sync_and_apply_until_settled` (`41885624816`):**
   - Run `41885624816` (`21e59029`, `91.295 s`): `review_r22_test.go:1000:
     expected retainedPending=true serving=false on ErrHold during Boot, got
     retainedPending=false serving=false` (`0.13 s`).
   - **Classification:** test fixture real-time clock window in
     `node/agent/review_r22_test.go`. The test seeded `LastSample:
     time.Now().UTC().Add(120 * time.Millisecond)` using the real wall clock so
     `FinalSample` during `Boot` would see `now < LastSample` within the `1 s`
     interval and return `meter.ErrHold` (`retainedPending = true`). Whenever
     setup between `SaveHead` and `Boot`'s `FinalSample` exceeded `120 ms` (as on
     the `0.13 s` CI subtest), `time.Now()` advanced past `LastSample`,
     `FinalSample` succeeded immediately instead of returning `ErrHold`, and
     `retainedPending` was `false`. Confirmed deterministically in
     [`docs_node_followup53-evidence_sixth-delayed-load.json`](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_sixth-delayed-load.json):
     injecting a `250 ms` pre-`Boot` delay via `sixth-delay.patch` reproduces the
     exact failure **`3/3` times** (`top_level_counts.fail: 3`, `exit_status: 1`,
     `load_workers: 24`), while the unmodified test passed `50/50` under local load
     (`top_level_counts.pass: 50`, `fail: 0`, `exit_status: 0` in
     [`docs_node_followup53-evidence_sixth-original-load.json`](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_sixth-original-load.json)).
   - **Repair (`node/agent/review_r22_test.go:970-1035`):** uses a controlled
     manual clock `meterClock := timeanchor.NewManual(checkpointAt, 0)` on both
     sides of `checkpointAt := head.LastSample.Add(120 * time.Millisecond)` with
     `meterClock.Advance(-time.Millisecond)` before `next.Boot(t.Context())`
     (`ErrHold` deterministic regardless of host scheduling delay) before advancing
     `meterClock` for the healing `Step`.

**Verification and development-attempt disclosure (`docs/node/followup53-2026-11-04.md`):**
- All four previously failing test targets pass `-count=50 -cpu=1,2,4` (`200`
  executions per test across `1`, `2`, and `4` CPUs), all five meter tests pass
  `-count=50 -cpu=1,2,4`, `go test -race -count=1 ./node/internal/meter/... ./node/agent/...`
  passes, full Gates 1–5 pass on final attempt 2
  ([`docs_node_followup53-evidence_gates-summary.json`](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_gates-summary.json)),
  and the single final privileged agent lab gate (`tests/netlab/agent`) passes
  (`13` passed, `0` failed, `1` skipped `TestD116_AT10_InstallerMarkerEnrollment`
  `0.07s`, `218.345 s` in
  [`docs_node_followup53-evidence_gate-lab.txt`](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_gate-lab.txt)).
- Failed intermediate development and gate runs are retained in
  [`docs_node_followup53-evidence_commands.json`](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_commands.json)
  and [`docs_node_followup53-evidence_gates-summary-attempt1.json`](../../tests/handover/testdata/history/follow-up-49/docs_node_followup53-evidence_gates-summary-attempt1.json)
  (including the rejected `local_budget.go` experiment reverted before commit,
  and Gate 1 attempt 1 failing `TestAT13_D120_ReloadLoop` `reload 79: render to
  installed 106.038388ms > 100ms` under concurrent review load before passing
  after `r33` finished).

### 2. Review `r33` (`f0bed96a1a622490f133e3cdb160a1e738f41712`)

The [raw `r33` archive](../reviews/r33-meter-restart-review.md) is archived
byte-for-byte from `r33-meter-restart-review` commit
`f0bed96a1a622490f133e3cdb160a1e738f41712:.agent/REVIEW.md` (committed
`2026-11-05T06:59:22Z` by **Ted H**, reviewer **agy/gemini-4-argon-high**, parent
`7bac86538bd9f0108a3bb441205b9ba979eae134`, review base
`29cd267162fa614c444122865b62f405e2f58802`, git blob
`b7621e069ab8547671888bda72e473c21efb938e`): **352 lines**, **51,920 bytes**,
SHA-256 `8bce5d989decb41fa7588307fc89521fc3972a5d59e80cf841ef6e7c7f433731`
([`review-provenance.json`](../../tests/handover/testdata/history/follow-up-49/review-provenance.json)).
- **Scope and findings:** pre-merge adversarial review of `w2-agent-core`
  follow-up 53's production meter change (`b094ffac72f2f4f247a3e2b592a6659a84f1d51a`
  against pre-change parent `9e52ea20675daae89710bc4630906e1c70b2e650`). Both
  `7bac86538bd9f0108a3bb441205b9ba979eae134` and the corrected head
  `f0bed96a1a622490f133e3cdb160a1e738f41712` report **0 findings (`0` Critical,
  `0` High, `0` Medium, `0` Low)** and **0 owner questions**; `f0bed96a`
  (`review: correct r33 report citations and attempt ledger`) corrected report
  line citations, the nine-step proof-development attempt ledger (Steps `212`,
  `230`, `341`, `370`, `378`, `424`, `440`, `444` and `448`), and test comments.
- **Three branch-only proof files (`present_on_main: false`), frozen in
  `tests/handover/testdata/history/follow-up-49/` and replayed on integrated
  `9a26725fd054e9e2ec287a0203fcf99d3d219ed8`:**
  1. `review_r33_test.go` in `node/internal/usagestore`
     ([frozen copy](../../tests/handover/testdata/history/follow-up-49/review-node_internal_usagestore_review_r33_test.go.txt),
     `6,562` B, `206` lines, SHA-256
     `6b4ac2806ad521ed99a3e53d4a172ebcc82faf6e17b482030642fa45fc879d07`): `1`
     parent test / `0` subtests
     (`TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability`);
  2. `review_r33_test.go` in `node/internal/meter`
     ([frozen copy](../../tests/handover/testdata/history/follow-up-49/review-node_internal_meter_review_r33_test.go.txt),
     `53,157` B, `1,345` lines, SHA-256
     `3560983d44aa03faa5b8d37d1c549d2dc1acd74e4fdd87e326f2b71c658d8260`): `7`
     parent tests / `26` subtests (`TestR33_Q1_ClockDomainAndFidelity` through
     `TestR33_Q7_PeriodsAndMonthBoundary`);
  3. `review_r33_test.go` in `node/agent`
     ([frozen copy](../../tests/handover/testdata/history/follow-up-49/review-node_agent_review_r33_test.go.txt),
     `11,920` B, `327` lines, SHA-256
     `d00f20943e6fc21d3e2ca0169b7a25b8e25f45b57f01484c78e8a0ce6f481b67`): `2`
     parent tests / `7` subtests
     (`TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback` and
     `TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback`).
  Replayed against the final integrated `w2-agent-core` tree
  `9a26725fd054e9e2ec287a0203fcf99d3d219ed8` (where `delta.go` blob
  `112260bd296d31cfd382d4faf03bf1efe50d7729` and `meter.go` blob
  `ac6ff0b548b5c0bdd6a3c5e44ad4b04d34dec3f9` are byte-identical to `b094ffac`
  and `29cd2671`): **`10` parent tests and `33` subtests across `3` packages
  passed (`0` failed, `0` skipped)**. See
  [the `r33` dispositions](../reviews/README.md#r33-dispositions-at-the-follow-up-49-cutoff-2026-11-05-est).

### 3. `F17-01` and `F31-01` design notes (`docs/node/followup53-2026-11-04.md` §§3–4)

- **`F17-01` (`pure_stale_generation` vs `attachServingRead`, proven by
  `TestF53_F1701_StaleLeaseHoldReportQosDetail` in
  `node/agent/followup53_test.go`):**
  - In `node/agent/decide.go:335-371`, when `leaseReason == ReasonStaleGen` and
    enforcement stays on baseline (`enforced == false`), `decide` constructs the
    dependent `qos` check with `Status: CheckFailed`, `Code:
    "VALIDATION_FAILED"`, and `Detail:
    strPtr("Rate policy could not be applied.")`, which matches
    `internal/nodes/generation_hold.go:59-91`'s `dependentLeaseHoldCheck`
    allowlist (`dependentStaleLeaseCheckDetails["qos"] = "Rate policy could not
    be applied."`).
  - However, in `node/agent/observe.go:122-158` -> `attachServingRead`
    (`node/agent/serving_evidence.go:36-58`), whenever `servingRead != nil`
    (which is retained across baseline withdrawal after any initial serving
    `Apply` or live read), `attachServingRead` unconditionally overwrites the
    `qos` check's `c.Detail = strPtr(diag)` (e.g. primary finding `F17-01` in
    run `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`, report
    `01a1a1be-ea7c-722e-88bb-8c57c70c8fb8`:
    `"read_at=2026-11-03T14:10:05.940031282Z source=live candidate=10 verified=true"`).
  - Central's `classifyGenerationHoldReport`
    (`internal/nodes/generation_hold.go:69-88`) then sees `qos.Detail != "Rate
    policy could not be applied."` and classifies `qos` as an independent safety
    failure (`independent_failure`, `PureStaleGeneration: false`), making
    `pure_stale_generation` unreachable on any node that has ever recorded a
    serving read in the current process.
  - **Options evaluated:** option (i) preserves each check's existing causal
    `Detail` (attaching the legacy `qos` diagnostic string only when
    `c.Detail == nil`) and carries every retained serving read separately in
    optional nullable `ObservationReport.serving_read` (`ServingReadEvidence`),
    projected from the same selected current-incarnation observation as
    `NodeDetail.serving_read`; option (ii) has Central classify structured
    causes rather than free text in `internal/nodes/generation_hold.go`; option
    (iii) adds optional lease-dependency metadata directly on `NodeCheck`.
    Agent-core 53 recommends option (i), the orchestrator adopts option (i) as
    provisional **D-164** below, and `#485` ships its first (contract
    compatibility) phase.
- **`F31-01` (source-binding structural read-back, design only, unadopted):**
  - In `node/agent/decide.go:208-245`, `bindingOK` is derived from signed-model
    construction (`model != nil` and non-empty `AllowedV4`) and couples
    `source_binding` to `qos` and `lease`, without an independent live kernel
    read-back of source-binding state.
  - The design note (`docs/node/followup53-2026-11-04.md` §4) proposes an
    independent structural verifier (`ReasonSourceBinding = "source_binding"` in
    `node/enforce`, API error code `SOURCE_BINDING_INVALID`, `SourceBindingError`
    on agent `Input`, and optional `ObservationReport.source_binding_evidence`)
    combining:
    (a) nftables `inet fvpn` `member_prerouting` / `member_forward` source-accept
    and class rules (`node/enforce/verify.go:822-858`, `verify_tc.go:130-155`),
    (b) `allowed_v4` element presence and remaining lifetime
    (`node/agent/allowed_presence.go:43-94`, `node/enforce/deadline_read.go`),
    and (c) per-address kernel route resolution via `ip -4 -o route get <ip>/32`
    (`node/agent/memberpath.go:37-132`, keeping kernel fib/rule evaluation at
    `3 + A` subprocess reads per check rather than re-implementing user-space
    LPM over `ip route show`), while explicitly stating that structural read-back
    proves kernel forwarding/filter configuration and does not by itself prove
    cryptographic per-device packet-origin authentication inside a shared overlay
    interface. No contract, Central, `node/enforce`, `node/agent` or
    `deploy/devstack` change is merged for `F31-01`.

## Contracts revision 15 (#485): OpenAPI `1.11.0`, provisional D-164 `ServingReadEvidence` contract phase, and D-112 / D-117 wording

[PR #485](https://github.com/TED-H-com/Freedom-VPN/pull/485) (native test source
`2d5205bf3f7e9962a1775009ce12e6f61fe62dfe`, `scope-review.json` Unicode escape
restoration commit `60e0a4012c0ed942c3402cb1992fcc06a6592787`, integrated branch
head `06fc1197b7f7007943d3f01b997a107f6e13ced1` after merging `#482`–`#484`,
merge `82dda8a867989e512e36a80dfba5aac1e4fdc28f`, final roadmap row
`aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`) and the
[frozen contracts README](../../tests/handover/testdata/history/follow-up-49/docs_contracts_README.md.txt)
(`82,126` bytes, SHA-256
`cfd6c47df57976c29f6e73dd48548efeefea9408ecbddf25ad684aeda2276e0e`) advance
`contracts/openapi.yaml` from revision 14 (`1.10.0`) to **revision 15
(`1.11.0`, `50` operations)** as the first phase of provisional **D-164** (after
the orchestrator interrupted an earlier draft to remove an invented `error_code`
property, keep producer-faithful `diagnostic` for both `verified=true` and
`verified=false`, and scope `NodeDetail.serving_read` `null` and ETag semantics
to the selected current-incarnation observation):
- **Schema additions (`contracts/openapi.yaml`, `pkg/api/api.gen.go`,
  `contracts/ts/openapi.d.ts`, `contracts/ts/revision15.typecheck.ts`,
  `contracts/ts/verify-vectors.ts`):**
  - Adds shared component `ServingReadEvidence` (`additionalProperties: false`)
    with the **exact five required members** (`read_at`, `source` enum
    `"apply"` | `"live"`, decimal-string `candidate_generation` matching
    `^(0|[1-9][0-9]*)$` with `maxLength: 20` parsed as `decstr.U64`, `verified`,
    and opaque `diagnostic` with `maxLength: 500` carrying `read_at=… source=…
    candidate=… verified=… ` plus `Evidence.ReportDetail()` clipped by `clipMsg`
    for both `verified=true` and `verified=false`) and **no `error_code`
    property**.
  - Adds optional nullable `serving_read` (`nullable: true`,
    `allOf: [$ref: '#/components/schemas/ServingReadEvidence']`,
    `x-omitempty: false`, `x-omitzero: true`, generated as
    `presence.Nullable[api.ServingReadEvidence]`) to both `ObservationReport`
    and `NodeDetail` (the only two entries added to
    `TestAPI01_OptionalNullableFields` in `contracts/openapi_test.go`).
  - On `NodeDetail.serving_read`, absent means the server does not project the
    field yet (pre-revision-15) and `null` means the selected
    current-incarnation observation has no available `serving_read` (servers do
    not search back for an earlier non-null read). All six existing
    `ObservationReport` fixtures (`ObservationReport.json` canonical JCS SHA-256
    `1a6b69f4f9fd62f7e883918efe4213654e165a3af65eab8c4fd0023eff59ecda` and raw
    SHA-256 `98bd6abbb84bee964b22110d60b755a5d3d227de0a87f908ed6e186b082b2def`,
    `ObservationReport.awaiting-member-grants.json`,
    `ObservationReport.tc-verified.json` canonical JCS SHA-256
    `28e3deb637cce042daf63b47cd456ac5175c1d596499889fe7e95428a23db29a`,
    `ObservationReport.tc-rejected.json`, `ObservationReport.tc-truncated.json`,
    and `ObservationReport.tc-null.json`) remain byte-for-byte unchanged.
- **`D-112` and `D-117` contract description updates (`contracts/openapi.yaml`,
  `docs/contracts/README.md`, `contracts/vectors/usage-retired-handoff.json`,
  `contracts/vectors/member-grant-revoke.json`):**
  - `D-112` (`submitUsageBatch` and `UsageBatchRequest.origin_incarnation_id`):
    states that a handed-off event's `interval_end` is bounded by `retired_at`
    plus Central's configured `sampling_interval_bound_seconds`
    (`UsageBatchConfig.SamplingIntervalBound`, default **`10` seconds**; Central
    stores no per-stream historical sampling interval), removing `"in force for
    the retired stream"` while keeping `sampling_interval_seconds: 30` and
    adding `sampling_interval_bound_default_seconds: 10` in
    `contracts/vectors/usage-retired-handoff.json`.
  - `D-117` (`revokeMemberGrant`): states that all entitlement grants are finite
    (`valid_until` is `NOT NULL` and `isfinite`) and open-ended grant creation
    remains forbidden, removing the obsolete `open-ended` case from
    `contracts/internal/gen/revision14.go` and `contracts/vectors/member-grant-revoke.json`
    (retaining `finite`, `same-key-replay`, `unknown-or-other-member`,
    `already-revoked`, `expired`).
- **Golden vectors, examples, self-check disclosure and migration test:**
  - `contracts/vectors/serving-read-evidence.json` (`32,603` B, SHA-256
    `4a32bfa3adad92105f6aa178a1cbe8935d0b10f98c09feb6e4836349e84f39e6`), `10`
    new valid example files (`NodeDetail.serving-read-*.json`,
    `ObservationReport.serving-read-*.json`, `ServingReadEvidence.*.json`), and
    `13` new invalid schema example files plus `decstr.U64` overflow
    `"18446744073709551616"` in
    [`contracts_examples_invalid_reasons.generated.json`](../../tests/handover/testdata/history/follow-up-49/contracts_examples_invalid_reasons.generated.json).
  - Proven by `TestAPI01_Revision15ServingReadEvidenceAndCorrections`,
    `TestISO01_D164ServingReadEvidenceVectorsAndPresence`,
    `TestMTR03_D112HandoffSamplingIntervalBoundWording`, and
    `TestAT29_D117FiniteGrantWithdrawalCorrection` in
    `contracts/revision15_test.go`, `TestServingReadEvidencePresenceAndRoundTrip`
    and `TestGeneratedTypesRoundTripValidExamples` in `pkg/api/api_test.go`,
    `contracts/ts/revision15.typecheck.ts`, `201` TypeScript vector checks in
    `contracts/ts/verify-vectors.ts`, and
    `TestHandover_ContractsRevision15MigrationIntegrity` in
    `tests/handover/handover_test.go` (with `6` archived pre-migration files in
    `tests/handover/testdata/history/contracts-revision-15/`, `42` preserved
    claim functions, and `120` refreshed SHA-256 scalars across `7` paths in
    `docs/e2e/at/secops/scope-review.json`). Five initial `contracts-verify`
    failures (`03-contracts-verify.log` TS2578/TS2353, `03-contracts-verify-fix1.log`
    `28e3deb6` hash / 5 revoke cases, `03-contracts-verify-pass.log` `1a6b69f4`
    canonical hash, `03-contracts-verify-pass2.log` `gofmt`, and
    `03-contracts-verify-pass3.log` `98bd6abb` raw hash) before passing
    `03-contracts-verify-pass4.log` are disclosed in [`pr-485-excerpt.json`](../../tests/handover/testdata/history/follow-up-49/pr-485-excerpt.json).
- **Delivery boundary:** `#485` is the contract and generated-types compatibility
  phase only. Central persistence and projection (`internal/nodes`), diagnostic
  consumers (`apps/portal`, `tests/e2e/at/isolation`, `tests/e2e/at/members`),
  and producer rollout (`node/agent`) are not yet implemented; `Isolation F17-01`
  and `D-163`'s `expected-not-exercised.json` entry remain open.

## Owner confirmation of `D-150`–`D-163` and provisional `D-164`

- **Owner confirmation (`D-150` through `D-163`,
  [`owner-confirmation-provenance.json`](../../tests/handover/testdata/history/follow-up-49/owner-confirmation-provenance.json)):**
  Ted confirmed all **14** decisions `D-150`, `D-151`, `D-152`, `D-153`, `D-154`,
  `D-155`, `D-156`, `D-157`, `D-158`, `D-159`, `D-160`, `D-161`, `D-162`, and
  `D-163` on **`2026-11-04 17:1x EST`**, relayed by the orchestrator. In
  `docs/handover/README.md` and `tests/handover/testdata/followup49-decisions.json`,
  each row's authority marker is updated from
  `orchestrator — provisional; the owner may overturn` to
  `orchestrator — confirmed by Ted, 2026-11-04 17:1x EST` (**15** substitutions
  across the **14** rows, including both the initial `01:4x EST; revised 02:2x
  EST` header and the `Clarified 05:3x EST` header inside `D-158`). Zero
  decisions were overturned, and reversing only that authority string restores
  each of the 14 rows byte-for-byte to its `#483` text.
- **New provisional decision `D-164`:** added to the [decisions
  register](README.md#follow-up-49-decisions-register-2026-11-05-est) (`67`
  decisions total: `14` updated, `1` added, `52` byte-identical to `#483`) from
  the orchestrator's provisional selection of option (i) on the audited `F17-01`
  design (`57c8a759ff0fdd8173e3beac4b5c2ec55e50e293`, unchanged at
  `d1ad15333856e03c854b3f99cd02968967aa9335`), with `#485` (`82dda8a8`) merged as
  its first phase. Ted may overturn this provisional decision.

## Posted NetBird discussion `#8473` comment (`19411545`) for `F22-AT01` / `F11-01`

On **`2026-11-05T04:30:00Z`** (`2026-11-04 23:30 EST`), the orchestrator posted
the follow-up 23 retrospective sample to
[NetBird discussion `#8473` comment `19411545`](https://github.com/netbirdio/netbird/discussions/8473#discussioncomment-19411545)
([`netbird-8473-comment-provenance.json`](../../tests/handover/testdata/history/follow-up-49/netbird-8473-comment-provenance.json),
body `2,110` bytes, SHA-256
`b58bb6c84b42e529914ec75a3eb1e1f71439341bc67f95683b192e1d2ace0543`), pinned to
commit `52b93845b5cfeac1dd63e04c700857fcd098a7f6` and its three frozen evidence
files (`commit_52b93845_docs_e2e_at_members_F23-retrospective-F22-AT01.json`,
`commit_52b93845_docs_e2e_at_members_followup23-2026-11-04.md.txt`, and
`commit_52b93845_docs_e2e_at_members_BLOCKERS.md.txt`):
- **Sample facts (`#473` retrospective of `#467` run
  `3eb3960d-c411-4648-8d77-bba40b436ae3`, original capture
  `786f5cd1-ed32-42b7-9638-99127303061c`, decoder source
  `8bde2b6acc0b116515148eeb0ff4448c65453a92`):** node `eth2` egress packet **`158`**
  at `08:22:37.385050Z` (`1793780557.385050`) and member `eth1` ingress packet
  **`161`** at `08:22:37.385059Z` (`1793780557.385059`) share IP-packet SHA-256
  `8c35343b6d681435b86c5ab3fb03283d215a1ac3f665ce4727cb854a4a244d69` (WireGuard
  message type **`2`** handshake response, anonymized labels
  `addr-eff8e7ca506627fe → addr-ab8283067e853f4e` with `addr-eff8e7ca506627fe`
  identified as IPv6 loopback `::1`, **`92` B** WireGuard payload, **`100` B** UDP
  length, **`140` B** IPv6 length, UDP ports **`51820 → 51820`**). Over the
  enclosing counter-read interval, the member's `Ip6InHdrErrors` increased from
  **`0` to `1`** while WireGuard receive accounting stayed **`0` packets / `0`
  bytes**.
- **Evidence boundaries preserved:** the response reached member underlay ingress
  (`eth1`) and failed before WireGuard receive accounting; no packet-specific
  kernel-drop trace (`kfree_skb` or `ip6_input`) exists for run `3eb3960d`; the
  enclosing `Ip6InHdrErrors` `0 → 1` delta is a window counter and does not by
  itself prove packet `161` caused that increment; the upstream source-selection
  mechanism is not proven; original run `3eb3960d` remains uncounted/blocked; and
  later clean initial transfers in `#473` (`6a25221b`) and `#481` (`dee8492f`) do
  not close `Members F22-AT01` or `Members F11-01` under confirmed `D-148`. At
  check time `2026-11-05T10:11:17Z`, discussion `#8473` had `1` comment and `0`
  replies.

## Decisions, findings register and scope-review boundary

- **67 decisions (`docs/handover/README.md`):** **14 updated** (`D-150`–`D-163`
  confirmed by Ted on `2026-11-04 17:1x EST`, `15` authority substitutions,
  technical substance byte-identical), **1 added** (provisional `D-164`), and
  **52 byte-identical** to `#483`.
- **138 findings (`docs/handover/README.md`):** **6 updated** (`Members F11-01`,
  `Isolation F17-01`, `Devstack F31-01`, `Members F22-AT01`, `Node workflow agent
  test failures`, and `Isolation 26 clone receiver wiring (9445 vs 9444)`),
  **0 added**, and **132 byte-identical** to `#483`. Every historical failure
  stays failed.
- **Entrypoints and `scope-review.json`:** all seven entrypoints (`README.md`,
  `docs/handover/00-index.md`, `docs/handover/deploy.md`,
  `docs/handover/member-guide.md`, `docs/handover/operator-guide.md`,
  `docs/handover/restore-drill.md`, and `docs/ops/README.md`) point to this cut;
  in `docs/handover/operator-guide.md` and `docs/handover/member-guide.md`, only
  lines 3–4 change after `#485`'s contract-version update. Changed owned files
  have refreshed SHA-256 digest scalars in `docs/e2e/at/secops/scope-review.json`
  with all Unicode escape sequences (`\u00a7`, `\u2013`, `\u2014`, `\u2192`) and
  surrounding JSON formatting preserved.

### Requests to orchestrator

**2026-11-05 EST:** follow-up 48's two requests are answered to the exact extent
delivered by **devstack 37 #482** and **agent-core 53 #484** (reviewed by **r33**
`f0bed96a` with `0` findings). No outside-owned pin needs changing in this cut.
1. **Route provisional `D-164` implementation phases in order (`Isolation F17-01` /
   `D-163`):** with the contract compatibility phase merged in **#485** (`1.11.0`,
   `ServingReadEvidence`), dispatch:
   - **Central acceptance and projection (`internal/nodes`)** to persist
     `ObservationReport.serving_read` and project `NodeDetail.serving_read` from
     the same selected current-incarnation observation while preserving old report
     canonical hashes and existing check/error/lease precedence;
   - **Diagnostic consumers (`apps/portal`, `tests/e2e/at/isolation`,
     `tests/e2e/at/members`)** to read `serving_read` without treating missing,
     `null`, rejected or old reads as serving permission or a generation ACK;
   - **Producer rollout (`node/agent`)** to emit `ObservationReport.serving_read`
     and attach the legacy `qos` diagnostic string only when `c.Detail == nil`
     (`node/agent/serving_evidence.go:36-58`), followed by a fresh qualifying
     `w4-at-isolation` run to exercise `pure_stale_generation` and retire the
     `D-163` entry in `tests/e2e/at/isolation/expected-not-exercised.json`.
2. **Route live clone receiver validation and second-exit log retention
   (`Isolation 26 clone receiver wiring`):** route live `AT12` clone
   transport/consumer-latch validation on the repaired `:9444` runtime config and
   before/after-stop second-exit container log retention to `w4-at-isolation` (noting
   that post-cut `w4-at-isolation` follow-up 27 run
   `d69f5652-aed6-4fdd-9e7c-edface751237` exited `2` at `10:33:52Z` and is
   awaiting evidence delivery).
3. **Carry `Devstack F31-01` source-binding read-back implementation:** if the
   owner adopts agent-core 53's `F31-01` structural design
   (`docs/node/followup53-2026-11-04.md` §4), dispatch contract, Central,
   `node/enforce`, `node/agent` and `deploy/devstack` follow-ups in order before
   exercising the source-address populated-hold row in `w4-at-isolation`.

# Pre-result and D-068 observation authority — follow-up 8

Run `71f2aa37-b0b9-4568-abe0-2ba7665377e4`, source `bc93ce430ebe8a50e5dfdaf4121352047fe89249` (on `main` `52dc77026cb168df5684ae2ccd5cbcadb4dbaeff`), evidence `2026-10-30T20:59:15Z`. Prefix `w4-updates`, port base 42000.

Raw evidence: [pre-result.json](pre-result.json), [AT24.json](AT24.json), [AT27.json](AT27.json).

## D-068 policy review and statement alignment (#189)

Fleet-update follow-up 9 merged as PR #189 (`c5f3c65`) under **D-068**. We re-read `docs/updates/central-lifecycle.md` ("Serving and observed versions", "Result and observation authority") on `main` and verified that every functional assertion in `preresult_test.go` and `restated_test.go` holds without change. The verbatim policy quotes in `restated_test.go:48-61` and line citations in `preresult_test.go:221,315` were updated to match `docs/updates/central-lifecycle.md:425-506`:

1. `docs/updates/central-lifecycle.md:425`:
   > Heartbeats after acceptance can confirm those
   > values, but cannot change current/last-good or clear recovery.
2. `docs/updates/central-lifecycle.md:426`:
   > A mismatch in
   > either release keeps `unconfirmed=true` by withholding that job's observation
   > timestamp; a later agreeing report can confirm it.
3. `docs/updates/central-lifecycle.md:434` (**added for D-068**):
   > A generation-bearing heartbeat is attributed to the **node's current job only**
   > (§13.2, §13.8; D-068, r13 M-1).
4. `docs/updates/central-lifecycle.md:454` (**added for D-068**):
   > Before that incarnation's first accepted result, pre-result heartbeat seeding
   > fills **only a null `nodes.current_release_id`**
5. `docs/updates/central-lifecycle.md:479` (**updated wording from `:442` under D-068**):
   > On the current job, each distinct unexpected release/job pair
   > gets one `update_observation.unexpected_release` audit
6. `docs/updates/central-lifecycle.md:501` (**updated wording from `:463` under D-068**):
   > A
   > later agreeing null last-good permits the issued target to fill the null
   > `current_release_id`, without establishing `last_good_release_id`.
7. `docs/updates/central-lifecycle.md:506`:
   > Once a result is accepted, seeding closes even for later jobs in that incarnation.

## Live observations in run `71f2aa37-b0b9-4568-abe0-2ba7665377e4`

1. **Fresh incarnation before first accepted result (`nodes.current_release_id = null`, `nodes.last_good_release_id = null`)**:
   - On current job `01a18e94-c215-7e07-995c-2189ba370804` (target `1.0.1` `01a18e94-0630-7535-a302-8ebf65284d54`), four repetitions each of:
     - `a_unrelated_current_and_last_good` (`current=1.0.2`, `last_good=1.0.2`)
     - `b_target_current_unrelated_last_good` (`current=1.0.1`, `last_good=1.0.3`)
     - `c_target_current_boot_last_good` (`current=1.0.1`, `last_good=0.1.0` unrecorded boot)
     leave both `current_release_id` and `last_good_release_id` `null`, `unconfirmed=true`, and emit exactly **1** `update_observation.unexpected_release` audit per distinct unexpected release (`01a18e94-0df0-7894-9874-5ef3b6736565`: 1, `01a18e94-1a85-7336-a2dd-dc11c9f5bf32`: 1, `01a18e93-fe44-711f-a01a-286d5565297d`: 1).
   - `agreeing_null_last_good` (`current=1.0.1`, `last_good=null` on the same current job while `current_release_id` is still `null`) fills `current_release_id` with `01a18e94-0630-7535-a302-8ebf65284d54` (`1.0.1`) while keeping `last_good_release_id = null` and adding `0` new audits.
   - The real accepted `1.0.1` result (`852ab9d0-154a-5e49-b6d4-c2a53c98448c`) then establishes `last_good_release_id = 01a18e94-0630-7535-a302-8ebf65284d54`.
2. **Recorded incarnation after accepted evidence (`nodes.current_release_id = 1.0.1`, `nodes.last_good_release_id = 1.0.1`)**:
   - On later current job `01a18e9a-3258-7b5e-969f-67f988279a76` (target `1.0.2` `01a18e94-0df0-7894-9874-5ef3b6736565`), four repetitions each of `c1_recorded_current` (`1.0.1`), `c2_job_target` (`1.0.2`), and `c3_unrelated_current` (`1.0.3`) preserve `current_release_id = 1.0.1` and `last_good_release_id = 1.0.1`, emit **0** `update_observation.unexpected_release` audits (`{}`), and withhold `last_observed_at` (`null`, `unconfirmed=true`) on `c2` and `c3`.
   - The subsequent real `1.0.2` updater result advances both `current_release_id` and `last_good_release_id` to `01a18e94-0df0-7894-9874-5ef3b6736565` (`1.0.2`).
3. **Stale ACK replay**:
   - Replaying the genuine earlier `1.0.1` `succeeded` result payload (`result_id: "852ab9d0-154a-5e49-b6d4-c2a53c98448c"`) while the newer `1.0.2` job is `waiting_online` returns HTTP `202` with the exact stored ACK (`accepted_as_history=false`, `observed_update_generation="1"`) and leaves `current_job_id`, `desired_generation="5"`, `observed_generation="4"`, `current_release_id`, `last_good_release_id`, and `waiting_online` unchanged.

F6-02 remains closed (#186) and aligned with D-068 (#189).

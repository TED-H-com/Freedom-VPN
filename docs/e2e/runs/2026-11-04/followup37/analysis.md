# Follow-up 37 — D-141 clone transport diagnosis and offline fixture correction

Date: `2026-11-04`. Fix round 1 corrects this follow-up's account. Earlier dated
notes, acceptance records and verdicts remain unchanged. No live run is claimed.

## 1. Diagnosis (D-141)

**The static port mismatch is supported.** The copied runtime config points to
an unserved port under the inspected clone boot path. The retained later captures
show zero conflicts and no freeze; they do not contain an identified clone
`connection refused` observation. Actual clone dial errors and its selected agent
release are **not recorded** in the evidence identified here.

Unless a different revision is stated, every source citation in section 1 is to
**baseline b921c75f**, not current branch line numbers. The inventory is extracted
from that committed tree. The request is isolation 26's
[Requests to orchestrator](../../../at/isolation/followup26-2026-11-04.md#requests-to-orchestrator).

### 1.1 Static clone control path

1. Preparation writes the portal enrollment URL `https://portal.example.invalid:8443`
   and runtime URL `https://127.0.0.1:9444`
   (`tests/e2e/cmd/fvpn-e2e-prepare/main.go:401-405`). Isolation installs its observer
   before preparation (`tests/e2e/at/isolation/driver.go.txt:129-133`).
   `installObserver` copies the observer and replaces the helper in `/usr/local/bin`;
   its wrapper starts the original resolver proxy before the observer
   (`tests/e2e/at/isolation/driver.go.txt:2283-2295`). `observeProxy` forwards to 9444
   and listens on 9445 (`tests/e2e/at/isolation/observer_main.go:48`,
   `tests/e2e/at/isolation/observer_main.go:95`). `configureObservation` sets
   `control_url` to `https://127.0.0.1:9445` in the source run config
   (`tests/e2e/at/isolation/observer_main.go:114-132`).
2. `prepare` calls `PrepareServingBaseline`
   (`tests/e2e/at/isolation/driver.go.txt:264-273`). The wrapper delegates to
   `PrepareServingBaselineWithOptions` (`tests/e2e/serving.go:42-46`), which calls
   `candidateConfig` and `setBootRelease` (`tests/e2e/serving.go:111-118`).
   `candidateConfig` inherits the source run config and writes compact JSON to
   `var/e2e/agent-<release>.json` (`tests/e2e/harness.go:551-566`). `setBootRelease`
   also rewrites the etc enrollment/run configs, changing `release_id`
   (`tests/e2e/harness.go:392-413`); isolation is not their only writer.
3. `freezeIdentity` calls `StartClonedIdentityFixture`
   (`tests/e2e/at/isolation/driver.go.txt:2664`), which calls `startSecondExit(ctx, true)`
   (`tests/e2e/identity_exports.go:60`). `startExitSlot` stops the source and copies
   `/etc/vpn-node`; it removes the old boot/recovery files, preserves the clone's
   machine-id, and changes only the etc configs' `datapath.snat` at baseline
   (`tests/e2e/second_exit.go:49-59`, `tests/e2e/second_exit.go:65-94`).
   The source image ID is used for the namespace and node containers
   (`tests/e2e/second_exit.go:98`, `tests/e2e/second_exit.go:108`,
   `tests/e2e/second_exit.go:167`). The copy also includes `/var/lib/vpn-node`,
   `/var/lib/netbird` and `/opt/vpn-node`, with the existing runtime-trigger cleanup
   (`tests/e2e/second_exit.go:121-142`). It does not copy `/usr/local/bin`'s observer
   or wrapper. Saying the entire writable layer is excluded would be wrong:
   compose binds only `/etc/vpn-node` and `/var/lib/vpn-node`
   (`deploy/devstack/compose.yaml:355-356`).
4. The clone's presentation network and control host mapping are retained
   (`tests/e2e/second_exit.go:154-160`). Its network name is
   `<prefix>-node-clone-net` (`tests/e2e/clone_network.go:122`). The original image
   installs the original helper (`deploy/devstack/Dockerfile.node:7`).
5. In entrypoint order: lab hosts are appended (`deploy/devstack/entrypoint-node.sh:47-55`),
   machine-id is preserved/generated and boot-id read (`deploy/devstack/entrypoint-node.sh:64-71`),
   then durable identity determines resume versus enrollment
   (`deploy/devstack/entrypoint-node.sh:77-100`). Enrollment, when needed, runs at
   line 96, before the proxy. The helper prepares recovery config at line 105,
   DNS relay starts at 107, resolver proxy at 108, watchdog at 110, and the
   supervisor loop at 118. Selected release is read at 122–125 and started at 126.
   NetBird service starts later at 137/139, followed by `netbird up` at 158.
   These line references all name `deploy/devstack/entrypoint-node.sh` at baseline.
6. The no-argument helper writes updater recovery config, not agent `control_url`
   (`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:89-132`). DNS relay reads the run
   config's SNAT (`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:136-158`). Resolver
   dispatch is at `tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:62-63`; the proxy
   forwards to `https://agent.example.invalid:9443` and listens on 9444
   (`tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:120`,
   `tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:140`). These boot steps do not repair
   the copied runtime URL.
7. `release_config` selects `/etc/vpn-node/agent-run.json` for `/usr/local`, or the
   copied release-specific config for a release directory
   (`deploy/devstack/agent-service.sh:117-123`). `_sup_config` is passed to the
   agent waiter (`deploy/devstack/agent-service.sh:132`,
   `deploy/devstack/agent-service.sh:152`). Both config branches inherit 9445 in
   this path. Which branch the old clone actually selected is **not recorded**.

Thus the baseline clone has a runtime URL for 9445 while this image's proxy
serves 9444. This static diagnosis does not establish the exact runtime error,
request sequence or network health of any historical clone.

The revision check uses the observer source at each revision, not today's line
numbers applied to old code:

| Revision | Observer listener / config override | Source at that revision |
|---|---|---|
| `9031ebd4` (F15 build) | 9444; no `control_url` override | `tests/e2e/at/isolation/observer_main.go:94`; `tests/e2e/at/isolation/observer_main.go:113-133` |
| `a8394027` (F16 build) | 9445; override to 9445; upstream 9444 | `tests/e2e/at/isolation/observer_main.go:94`; `tests/e2e/at/isolation/observer_main.go:124`; `tests/e2e/at/isolation/observer_main.go:47` |
| `8afa69c5353c8518c69294daec997ba261eba65f` (#399 merge) | 9445; override to 9445; upstream 9444 | `tests/e2e/at/isolation/observer_main.go:94`; `tests/e2e/at/isolation/observer_main.go:124`; `tests/e2e/at/isolation/observer_main.go:47` |
| `4c495c3b` (F25 build) | 9445; override to 9445; upstream 9444 | `tests/e2e/at/isolation/observer_main.go:95`; `tests/e2e/at/isolation/observer_main.go:125`; `tests/e2e/at/isolation/observer_main.go:48` |

### 1.2 Committed identity-freeze inventory

There are **30 files**, including three `-stream` copies, not 30 independent
experiments. Paths in this table are relative to `docs/e2e/at/isolation/`.
No identity-freeze blob contains its own `run_id` or `build`. The ID/build source
column identifies the join. Counts include rows with `clone_read_error` when they
carry `clone_observation`; that is not a successful read. Maxima and ever-frozen
are calculated over the observation objects. Every file has status
`requires receiver correlation`. Clone agent/watchdog logs: **not recorded** in
any listed capture; the retention limits are explained below.

| Identity-freeze path | Run ID | Build | Total rows | Rows containing `clone_observation` | Max `endpoint_rows` | Max `conflicts` | Ever frozen | ID/build source |
|---|---|---|---:|---:|---:|---:|---|---|
| `history/follow-up-7/identity-freeze.json` | `041e9e48-79a6-45dd-a19d-8b5a63290655` | `d1a90ea60abd853f404e1df571dd3a5871d89b5f` | 4 | 3 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-8/identity-freeze.json` | `7779d83c-2f78-4699-ac88-8fcc59e8cf87` | `dafb8f4d5b143dc1b425c471678d39a841033006` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-9/identity-freeze.json` | `2be2797b-7a28-4e9f-a1fe-5fffa4c1b047` | `e496d826c04054bd718186de078b8f972353fb94` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-10/identity-freeze.json` | `bfcc3880-dc02-485d-990c-13811d6e8835` | `cd2e89746b710eda1580d2c822ceff3f636fb013` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-10-attempt-1/identity-freeze.json` | `17c14696-3eb1-469d-8aab-f947f6f425e4` | `2da5acfbbe08f094fa46dc91802bf75012481f71` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-10-attempt-2/identity-freeze.json` | `5e9e4041-d942-4562-9e99-489bf735636a` | `d9b770e06a3510880236eed57fe229465947951e` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-11/identity-freeze.json` | `edd986c4-16d2-4aab-b92f-baa19f8bd829` | `2304f77a733978405a35eea3b8d0a075027ce88e` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-12/identity-freeze.json` | `1c25ee44-1872-4cce-a9f5-d52491d87095` | `99e58bff0cf7276e36cb04f35605835208c8638e` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-13/identity-freeze.json` | `38074149-1137-4782-80fd-8669c36c6fcb` | `c521cb0018844eac1de24ee243c858518bb1a5e3` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-14/identity-freeze.json` | `56d20530-8042-4d0c-8156-de5a47ce8b61` | `716a16335cb5335473416740c76a75ce95536796` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-15/identity-freeze.json` | `0b9e4510-776f-4c51-9e66-500404ce9f7e` | `9031ebd4044059db25ea4ea7a397b87e6d6c6aa5` | 3 | 2 | 1 | 1 | `true` | same-dir AT13.json |
| `history/follow-up-16/identity-freeze.json.gz` | `83b457ef-ea16-490b-9923-1fea0df86b36` | `a8394027002771be64f023d062ea47b2da1447d8` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-16-stream/identity-freeze.json.gz` | `83b457ef-ea16-490b-9923-1fea0df86b36` | `a8394027002771be64f023d062ea47b2da1447d8` | 90 | 90 | 1 | 0 | `false` | same-dir archive.json + receiver-capture.json.gz |
| `history/follow-up-17/identity-freeze.json.gz` | `0ec14b39-467e-4a37-a239-1a33ef12eb71` | `de9682a96a296a3b79686db029b3e28f493f08f4` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-17-stream/identity-freeze.json.gz` | `0ec14b39-467e-4a37-a239-1a33ef12eb71` | `de9682a96a296a3b79686db029b3e28f493f08f4` | 90 | 90 | 1 | 0 | `false` | same-dir archive.json + receiver-capture.json.gz |
| `history/follow-up-18/identity-freeze.json.gz` | `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04` | `c248db966d684d58b4781349f71f8394fcb64d9b` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-18-stream/identity-freeze.json.gz` | `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04` | `c248db966d684d58b4781349f71f8394fcb64d9b` | 90 | 90 | 1 | 0 | `false` | same-dir archive.json + receiver-capture.json.gz |
| `history/follow-up-19/identity-freeze.json.gz` | `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04` | `c248db966d684d58b4781349f71f8394fcb64d9b` | 90 | 90 | 1 | 0 | `false` | F18 AT13.json + identical F18 blob; F19 README.md |
| `history/follow-up-20/identity-freeze.json.gz` | `c52bea6c-cd91-4086-a779-656b01277dcc` | `037f44187aaac256c82682de01af2ef61bbb90f0` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-20-run/identity-freeze.json.gz` | `c52bea6c-cd91-4086-a779-656b01277dcc` | `037f44187aaac256c82682de01af2ef61bbb90f0` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-21/identity-freeze.json.gz` | `7599576e-339c-49c9-8bf5-0f710818cbda` | `fe2c68ecbf1ef1481d744c998be95710d28be954` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-21-run/identity-freeze.json.gz` | `7599576e-339c-49c9-8bf5-0f710818cbda` | `fe2c68ecbf1ef1481d744c998be95710d28be954` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-22/identity-freeze.json.gz` | `7b0db5f2-eab5-4534-824a-b54781eed3fd` | `0b95e69ae89c68e0fcfbf1fbbfe6774b2e1948db` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-22-run/identity-freeze.json.gz` | `7b0db5f2-eab5-4534-824a-b54781eed3fd` | `0b95e69ae89c68e0fcfbf1fbbfe6774b2e1948db` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-23/identity-freeze.json.gz` | `328ab9e0-7ff4-495a-81c1-410ae8a7d876` | `89accd8da1bc3cf44093d3cc0061624f3d603002` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-23-run/identity-freeze.json.gz` | `328ab9e0-7ff4-495a-81c1-410ae8a7d876` | `89accd8da1bc3cf44093d3cc0061624f3d603002` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-24/identity-freeze.json.gz` | `df9dbad7-049d-4eaa-ab98-5490906cc3e9` | `56d814113913ddc6177d47d4a5160a65c93add25` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-24-run/identity-freeze.json.gz` | `df9dbad7-049d-4eaa-ab98-5490906cc3e9` | `56d814113913ddc6177d47d4a5160a65c93add25` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `history/follow-up-25-run/identity-freeze.json.gz` | `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db` | `4c495c3b1516022e1de8db2f864432a8a8d171c5` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |
| `identity-freeze.json.gz` | `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db` | `4c495c3b1516022e1de8db2f864432a8a8d171c5` | 90 | 90 | 1 | 0 | `false` | same-dir AT13.json |

F7 has 4 total / 3 clone-observation rows. F8–F15, including both F10 attempts,
have 3 / 2. Their last row is `rows[-1].label = "freeze latch held"`, without
`clone_observation`: the append is at `tests/e2e/at/isolation/driver.go.txt:2693-2696`.
All retained F16–F25 captures have 90 / 90, but **not every run in that interval
created a clone**. F19's own `AT13.json` identifies run
`70beb28e-4766-4004-932b-510fd01f986a`, build
`458b7a89f07c22dd4e66f839195b6549db4b883d`. Its
`history/follow-up-19/follow-up-19-findings.json` records F19-02 setup failure
and the secondary diagnostic “setup exited before clone creation”. Its README
still names F18. Its identity-freeze file is F18's identical compressed blob
`1075e8ca2190ae38899fd36dc3a46b4eac641736`, not an F19 clone attempt. The three
stream copies, later `-run` copies and root are duplicate retention, not new runs.
F14 and F15 are plain JSON. There is no F25 archive identity-freeze file outside
`follow-up-25-run` and root. Provisional receiver analyses are not identity-freeze
records and are excluded.

`endpoint_rows` counts database rows for the incarnation, not independently
reachable endpoints (`tests/e2e/identity_exports.go:90`). The table's primary key
is `incarnation_id` (`db/migrations/00110_pki_enrollment.sql:30-33`);
`candidate_hash` is added by
`db/migrations/00187_identity_roaming_and_readiness_grace.sql:5-6`.

Retention is narrower than the former account claimed. The before-clone call
(`tests/e2e/at/isolation/driver.go.txt:2659`) only rotates/truncates logs inside
the source (`tests/e2e/at/isolation/driver.go.txt:2309-2314`). The final call
(`tests/e2e/at/isolation/driver.go.txt:2378`) writes segments from that source
(`tests/e2e/at/isolation/driver.go.txt:2316-2330`) and aggregates them
(`tests/e2e/at/isolation/driver.go.txt:2379-2387`). It does not read clone
agent/watchdog logs. Other clone runtime data does exist: fingerprints and
`CloneStage` come from the clone (`tests/e2e/identity_exports.go:78-84`), plus
stop receipts and lifecycle evidence. These do not substitute for a dial-error log.

### 1.3 Earlier devstack and enrollment samples

`TestE2E_HarnessExports` calls `exportsEnrollment` and `exportsClone`
(`tests/e2e/exports_live_test.go:22`, `tests/e2e/exports_live_test.go:69-70`).
`exportsClone` polls `fixture.Observe(wait)` for matching identity and enrolled
stages, then calls `cloneConflict` (`tests/e2e/exports_live_test.go:193-217`).
The latter's predicate requires `Object`, `Frozen`, `ConflictCount >= 1`, failed
provisioning, forwarding false and one audit
(`tests/e2e/exports_live_test.go:240-246`). It does not poll `LastConflict`.
`reconcileClone` stops/revokes and re-enrolls, then checks the new incarnation's
unfrozen state (`tests/e2e/exports_live_test.go:345-355`,
`tests/e2e/exports_live_test.go:427-442`).

The latest retained export sample is run `12d10041-9d33-4fbb-a4b0-9d2a21bce785`,
source `4fb5fa88ac400605ac08ee42c8eac99e936ec2a3`. Both records below have status
`pass`, but their observations differ:

| Committed record | `at` | Values from that record |
|---|---|---|
| `docs/e2e/runs/2026-10-31/followup15/exports/export-clone.json` | `2026-10-31T04:28:50Z` | Parsed `observations[0].detail`: `same_identity:true`, `endpoint_rows:1`, `conflicts:0`, `mapping_frozen:false`, `SourceStage:"enrolled"`, `CloneStage:"enrolled"` |
| `docs/e2e/runs/2026-10-31/followup15/exports/clone-conflict.json` | `2026-10-31T04:30:11Z` | Named observation detail strings: `conflict_count:"1"`, `mapping_frozen:"true"`, `provisioning_state:"failed"`, `member_forwarding_observed:"false"`, `identity_conflict_audits:"1"`, `identity_conflict_audits_after_wait:"1"`, `reconciled_mapping_frozen:"false"`, `reconciled_conflict_count:"0"` |

Earlier `history/exports-first/export-clone.json` and
`history/exports-second/export-clone.json` under that same followup15 directory
have runs `926791b9-36ef-4051-9d44-cd7eafdc8792` and
`630b9f07-62d2-4db1-9737-9060732db1f1`, respectively. Both parsed observations
have `endpoint_rows:1`, `conflicts:1`, `mapping_frozen:true`. The causal ordering
of the first clone request relative to the initial read is **not recorded**.

Enrollment's `docs/e2e/at/enrollment/AT07.json`, run
`891ee9b8-bf01-4583-a47a-9542c8b04c8a`, covers
`2026-10-31T14:59:07Z`–`2026-10-31T15:16:38Z`. Its
`cloned_identity_conflict` check is `pass`; `detail.observe_enrolled` has
`same_identity:true`, `endpoint_rows:1`, `conflicts:1`, `mapping_frozen:true`.
These are earlier retained samples without isolation's observer, not live reruns
during F16–F25 or this follow-up. They do not validate the corrected fixture.

### 1.4 Consumer inventory

Search scope: all `tests/e2e/**`, including `.go.txt`. The table lists direct
fixture callers and shared wrappers. Source-config remarks describe the inspected
caller setup, not newly measured URLs. All baseline line references remain
explicitly pinned to b921c75f. Any future caller can supply a modified source
runtime URL; isolation is the identified observer override in these callers.

| Caller at baseline | Hook / source configuration | Record provenance and scope |
|---|---|---|
| `tests/e2e/identity_exports.go:60` — `StartClonedIdentityFixture` | `startSecondExit(ctx, true)`; inherits caller's source | `Observe` reads fingerprints, stages and central endpoint/conflict/freeze rows; source at `tests/e2e/identity_exports.go:68-90` |
| `tests/e2e/second_exit.go:20-26` — `StartSecondExit`, `startSecondExit` | false for fresh node2, true for node-clone; delegates to `startExitSlot` | Shared implementation, no acceptance verdict of its own |
| `tests/e2e/exports_live_test.go:196` — `exportsClone` | Clone; ordinary prepared runtime URL, no isolation observer | `export-clone.json` and separate `clone-conflict.json` in section 1.3; do not transfer 1/true from the latter to the former |
| `tests/e2e/at/enrollment/live_test.go:1427` — `clonedIdentity` | Clone; no isolation observer | `docs/e2e/at/enrollment/AT07.json`, `cloned_identity_conflict`; earlier sample described above |
| `tests/e2e/at/isolation/driver.go.txt:2664` — `freezeIdentity` | Clone; observer-modified URL inherited through source and candidate configs | Section 1.2 inventory: early final `rows[-1].label` is `freeze latch held`; later retained captures are zero-conflict/unfrozen. Auxiliary consumer latch only. Distinct post-change stop reviews cover F16, F17, F18 only (section 1.5) |
| `tests/e2e/second_exit_test.go:16` — `secondExit` | `StartSecondExit`; ordinary source config | `docs/e2e/runs/2026-11-04/followup36/second-exit.json`; independent node2, not a clone |
| `tests/e2e/exports_live_test.go:102` — `exportsEnrollment` | `StartSecondExit`; ordinary source config | `docs/e2e/runs/2026-10-31/followup15/exports/export-enrollment.json`; independent enrollment before `exportsClone` |
| `tests/e2e/fixtures16_live_test.go:99` — `runFixtureCampaign`, called by `TestE2E_Followup16Fixtures` at `tests/e2e/fixtures16_live_test.go:25` | `StartSecondExit`; ordinary source config | `Record.ID = "second-resolver"` at `tests/e2e/fixtures16_live_test.go:102`; independent node2 resolver |
| `tests/e2e/followup18_native_test.go:114` — `TestE2E_Followup18Native` | `StartSecondExit`; ordinary source config | `second-exit-counted` / `namespace-restart-90` records at `tests/e2e/followup18_native_test.go:124-138`; independent node2 |
| `tests/e2e/at/enrollment/live_test.go:890`, `tests/e2e/at/enrollment/live_test.go:1552` — `at07`, `peerUniqueness` | `StartSecondExit`; no isolation observer | `docs/e2e/at/enrollment/AT07.json` / `distinct_api_first_boot`; `docs/e2e/at/enrollment/AT06.json` / `peer_uniqueness`; fresh node2 |
| `tests/e2e/at/members/second_exit_test.go:22` — `secondExit` | `StartSecondExit`; no isolation observer | Shared member second-exit check, including `docs/e2e/at/members/AT09.json`; fresh node2, no clone claim |
| `tests/e2e/at/updates/restated_test.go:152` — `inheritDefault` | `StartSecondExit`; no isolation observer | `docs/e2e/at/updates/AT24.json`; new-node default inheritance, no clone claim |
| `tests/e2e/at/usage/live_cases_b_test.go:262`, `tests/e2e/at/usage/live_helpers_test.go:1066` — `evalAT19`, `measurePerIncarnationRetainedUsage` | `StartSecondExit`; no isolation observer | `docs/e2e/at/usage/AT19.json`; independent node2 enrollment/re-enrollment and usage |
| `tests/e2e/at/secops/two_exit_live_test.go:31` — `twoExitTraffic` | `StartSecondExit`; no isolation observer | `p.metadata["two_exit_traffic"]` at `tests/e2e/at/secops/two_exit_live_test.go:18`; two independent exits, no clone claim |
| `tests/e2e/fleet_snapshot.go:44` — `RetainEnrollingThirdNode` | Direct `startExitSlot` for fresh held node3 | Reads enrollment config and changes a local decoded copy's URL, writing separate `agent-retain.json` for one-shot enrollment (`tests/e2e/fleet_snapshot.go:71-86`). Does not change `agent-enroll.json`. `BootEnrollingThirdNode` removes the hold and boots, without restoring a URL (`tests/e2e/fleet_snapshot.go:134-145`) |

### 1.5 Verdict and stop-receipt limits

The F15/F16 revision comparison establishes the port/config change. Later retained
clone windows fit the static transport defect, but a retained clone dial-error
observation has not been identified. F19 was setup-blocked and copied F18's
capture. The former shorthand “every run 16–25” was too broad.

`freezeIdentity` writes an auxiliary consumer-latch record through `x.extra`
(`tests/e2e/at/isolation/driver.go.txt:2673-2678`,
`tests/e2e/at/isolation/driver.go.txt:2700-2703`), not a score in `x.records`.
The early captures record a freeze; the later captures do not establish that
latch. This is a narrow statement about that function and those records. All
historical AT verdicts remain unchanged; this follow-up does not re-score them.

Post-port-change distinct `stop-receipt-review.json` samples are only F16, F17
and F18. There is also an earlier F15 review. F19–F24 and root reuse F18's exact
blob `d429b8d5251e6834f959a129d3a200007f6b1a43`, whose `run_id` is
`9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`. These files establish no distinct stop
review for F19–F25.

`review_stops.py` reads `clone_source` from the source-container stop inside
identity-freeze (`tests/e2e/at/isolation/review_stops.py:23-36`), and
`direct_agent` from plain `receiver-closure.json`
(`tests/e2e/at/isolation/review_stops.py:37-47`). `archiveReceiver` invokes
`agent-stop-sample` on source and clone
(`tests/e2e/at/isolation/driver.go.txt:2341-2357`). For F16, F17 and F18, the clone
row's parsed `receiver-closure.json` output contains a parent wait receipt with
`exit_code:0`; joining its launch ID, PID and start ticks to the matching
`direct_agent` review row supports that recorded stop outcome only. It does not
prove successful control transport. The output's `outcome_source` is:

> parent wait receipt; escalation sidecar is intent only; pidfd timestamps bound observation, not exact kernel exit

This key is produced by `tests/e2e/at/isolation/observer_main.go:519` and is in the
closure output, not a top-level stop-review key.

## 2. Authorized fixture correction

The correction is confined to the shared fixture. The existing image selection,
presentation network, durable copy and runtime-trigger cleanup stay as before.
The required field scope is:

| Prepared file | Existing slot setup | Additional change |
|---|---|---|
| `etc/agent-enroll.json` | Keep etc `datapath.snat` rewrite | None to `control_url`, in clone and fresh modes; preserve portal enrollment URL |
| `etc/agent-run.json` | Keep etc `datapath.snat` rewrite | `control_url` to `https://127.0.0.1:9444` |
| Copied `var/e2e/agent-<release>.json` | Preserve copied `datapath.snat` and all other values | Only `control_url` to `https://127.0.0.1:9444` |
| Identity, machine-id, durable state and other copied files | Preserve baseline clone behavior | No new changes |

The previous correction overreached on cloned enrollment URLs and copied release
SNAT. JSON reserialization preserves non-target **values**, not formatting or
untargeted bytes. Identity and durable-state files must remain byte-for-byte
unchanged. Copied release SNAT remains a known gap: IPv4 validation in baseline
`node/agent/config.go:250-252` does not establish that the address is appropriate
for every use. No product change or release redesign is authorized here.

Fixture/test correction and its new gate results are pending at this diagnosis
commit. The existing test is one function,
`TestD141_ClonePreparedConfigsDialContainerResolverProxy`, in the `e2e`-tagged
`tests/e2e/second_exit_test.go`; default no-tag tests do not execute it. The offline
extension must cover compact release JSON as written by `candidateConfig`,
formatted JSON, the complete field matrix, full non-target value equality,
durable byte preservation, malformed-input refusal and fresh enrollment URL
preservation. No prior report output is accepted as this session's test result.

## 3. Carried items and requests

AT29 guest-clock rollback remains [operator-run](../../../at29-guest-clock.md).
`member-ipv6` stays listed in `tests/e2e/nightly_expected_blocked.json`.

### Requests to orchestrator

2026-11-04 — After merge, request isolation validation of the corrected clone
transport and consumer latch in a separately authorized assembled run. This is
future validation, not evidence already obtained. Request clone agent/watchdog
log retention around the real `archiveReceiver` / `agent-stop-sample` boundary
(baseline `tests/e2e/at/isolation/driver.go.txt:2341-2357`), before stopping and
with shutdown output retained afterward as appropriate. Isolation owns those
changes. Carry the copied-release-SNAT gap and the unchanged AT29/member-ipv6
dispositions. No AT file is edited here.

# Backup and Restore Drill Record (`docs/handover/restore-drill.md`)

**目前交付：**[交付導覽](../DELIVERY.md)整理現有功能、操作入口與驗證範圍。下列 follow-up 快照保留各自當時的範圍與結果。

**Current delivery:** The [delivery guide](../DELIVERY.en.md) covers current capabilities, operating entrypoints and verification scope. The follow-up snapshots below retain their historical scope and outcomes.

**Archived cut:** [follow-up 51](follow-up-51.md), #524–#526 at `66de6a24`; source-backed changes, verification boundaries and carried gaps.
**Earlier archived cut:** [follow-up 50](follow-up-50.md), #486–#518 and #520–#523 at `fe9bb436`; source-backed changes, verification boundaries and carried gaps.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

This document provides the repeatable backup, point-in-time restore (PITR), and post-restore reconciliation procedure (`§9.3`, `§17.2`, `§17.3`, `§17.4`, `§19 AT30`), together with the measured RPO and RTO evidence from the Wave 4 verification drill.

---

## 1. Recovery Objectives (`§17.2`)

| Subsystem | RPO Target | RTO Target | Architectural Guarantee |
|---|---|---|---|
| **Central PostgreSQL 17** (Members, Entitlements, Nodes, Policies, Audit) | `≤ 15 minutes` | `≤ 4 hours` (service target; unmeasured here) | Continuous WAL archiving (`archive_timeout = 300s`) + daily base backups. |
| **Usage Ledger (`usage_events`, `usage_checkpoints`, `usage_rollups`)** | **Effective RPO `0` only within the available node archive window**; outside it, replay can be unavailable | `≤ 4 hours` (service target; unmeasured here) (ingest resumes; reconciliation runs asynchronously) | Nodes retain ACKed payloads at least `72 hours` after local ACK (`DefaultArchiveRetain`); no node configuration sets a longer window. The unacknowledged outbox blocks new member forwarding at `72 hours` oldest-event age or decimal `256 MB` payload, keeps those events and resends; the archive has a separate `1000000000`-byte engineering budget. A verified new `ledger_epoch` permits replay of retained post-snapshot events without double-counting. Shortening the window requires updated parameters, recovery documents and risk notes ([usage storage](../node/usage-store.md#ack), `§10.8`). |
| **NetBird Management Store** | `≤ 15 minutes` | `≤ 4 hours` (service target; unmeasured here) | Reconciliation of known resources follows central desired state; it is not global orphan cleanup. Carried gap: post-restore orphan inventory/revocation, as described in Phase 5.2. |
| **Signing Vault (`/var/lib/fvpn/staging-authority/vault`) & PKI Revocations** | **`0` (Zero Loss)** | Immediate | Stored outside PostgreSQL snapshots (`§16.2`, `§17.2`). Never overwritten by a database rollback. |

---

## 2. Wave 4 AT30 Restore Drill Record (`2026-10-29`)

### 2.1 Drill Scenario & Topology

On `2026-10-29`, the Wave 4 AT30 point-in-time restore and reconciliation drill was executed against PostgreSQL 17 (`fvpn_w4_handover_drill`) and verified alongside `TestAT30_RestoreSnapshotRecoveryReplayAndReplayUnavailable` and `TestAT30_HTTPRecoveryFreezeEpochRotationReplayAndAuditPagination`:

1. **Pre-incident baseline**:
   - Member `0196ed00-0000-7000-8000-000000000101` (`active`), Device `0196ed00-0000-7000-8000-000000000201` (`100.64.0.10`, `binding_revision=1`).
   - Exit Node `0196ed00-0000-7000-8000-000000000301`, Incarnation `0196ed00-0000-7000-8000-000000000401`.
   - Node emitted usage events `event_seq = 1..5` (`forward_admission_v1`, `seq * 1000` upload bytes, `seq * 2000` download bytes per 1-minute window starting `2026-10-29T18:00:00Z`).
   - Database snapshot (`restored_to = 2026-10-29T18:02:00Z`) contained only `event_seq = 1..2` (`3,000` upload bytes, `6,000` download bytes, `highest_contiguous_seq = 2`, epoch `1f7b1c2a-48b1-4ab2-a75a-b125bec8e0ba`).
   - Events `3..5` were ACKed prior to the simulated central DB loss. On the node, `event_seq = 3` remained in the local SQLite archive, while `event_seq = 4..5` simulated local retention expiry (`retention_expired`), followed by new post-restore traffic `event_seq = 6` (`6,000` upload bytes, `12,000` download bytes).

### 2.2 Local Fixture Measurements (Service RTO Unmeasured)

| Metric | Spec Target (`§17.2`) | Wave 4 Measured Value | Result |
|---|---|---|---|
| **Snapshot Window Lag (Raw DB RPO before node replay)** | `≤ 15 minutes` | `3 minutes` (`18:02:00Z` snapshot vs `18:05:00Z` pre-crash watermark) | **PASS** |
| **Effective Usage Ledger RPO (Retained node archive `seq 1..3`)** | `0 bytes lost / 0 double-counted` | **`0 bytes lost / 0 bytes double-counted`** (`seq 1..2` deduplicated, `seq 3` replayed and accepted) | **PASS** |
| **Unrecoverable Node Range (`seq 4..5`) Handling** | Honest `gap=true` (`0` invented bytes) | **`gap=true`**, `2` zero-byte `replay_unavailable` tombstones recorded, `0` fabricated bytes | **PASS** |
| **Local schema + snapshot restore command duration** | Local command only | `342 ms` (isolated PostgreSQL 17 schema + state load) | Local duration only |
| **Local ledger recovery fixture duration** | Local fixture only | **`64 ms`** (`406 ms` local fixture including migrations) | Local duration only |

These millisecond durations exclude rebuilding secrets, service settings, external reconciliation and user probes. They do not establish the 4-hour service RTO in §17.2. The physical PITR evidence is [the backup drill record](../ops/backups/drill-2026-10-29.md).

### 2.3 Recorded Drill Timeline & Verified Command Outputs

- **Migration invariant — Verify migrations on the restored target (`fvpn-migrate status`)**:
  Every file in `db/migrations/*.sql` for the deployed release must be applied on the restored target, and `fvpn-migrate status` must report no pending version. Use the migration binary built from that same release and confirm `current_database` identifies the restored target. [`deploy/backup/test/review_proof.sh`](../../deploy/backup/test/review_proof.sh), run by `make -f mk/backup.mk backup-review-proof`, derives the expected versions and count from that glob and compares them with status from its isolated PITR target; no inventory is copied into this runbook. See Phase 2.2 for the operator commands.

- **`2026-10-29T22:23:20.348Z` — Step 1: Freeze usage ingestion (`fvpn-usage-admin freeze`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    freeze \
    --actor 0196ed00-0000-7000-8000-000000000999 \
    --reason "AT30 point-in-time restore drill"
  ```
  Output:
  ```json
  {
    "actor": "0196ed00-0000-7000-8000-000000000999",
    "reason": "AT30 point-in-time restore drill",
    "status": "frozen"
  }
  ```

- **`2026-10-29T22:23:20.371Z` — Step 2: Rotate `ledger_epoch` and enter `reconciling` mode (`fvpn-usage-admin begin-recovery`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    begin-recovery \
    --actor 0196ed00-0000-7000-8000-000000000999 \
    --reason "AT30 restore from 2026-10-29T18:02:00Z snapshot" \
    --restored-to 2026-10-29T18:02:00Z \
    --affected-since 2026-10-29T18:00:00Z
  ```
  Output:
  ```json
  {
    "recovery_id": "01a189c2-b133-7b6a-9006-2d8d7a99895f",
    "previous_ledger_epoch": "1f7b1c2a-48b1-4ab2-a75a-b125bec8e0ba",
    "new_ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
    "mode": "reconciling",
    "reason": "AT30 restore from 2026-10-29T18:02:00Z snapshot",
    "actor": "0196ed00-0000-7000-8000-000000000999",
    "signed_record": {
      "signed": {
        "actor": "0196ed00-0000-7000-8000-000000000999",
        "affected_since": "2026-10-29T18:00:00Z",
        "created_at": "2026-10-29T22:23:20.371746519Z",
        "new_ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
        "previous_ledger_epoch": "1f7b1c2a-48b1-4ab2-a75a-b125bec8e0ba",
        "reason": "AT30 restore from 2026-10-29T18:02:00Z snapshot",
        "recovery_id": "01a189c2-b133-7b6a-9006-2d8d7a99895f",
        "restored_to": "2026-10-29T18:02:00Z",
        "stream_count": 1
      },
      "signatures": [
        {
          "key_id": "01a189c2-afc3-7077-bd89-df83873cc3d8",
          "algorithm": "ed25519",
          "signature_b64": "xdNE2jfT0g0Ll3oMmlEgctQ+ajJSVMzZnmRsrjo0D2pT2UlwHfFQBtTKzywoOBemLx5y2KptS6RqEYOiq3l1Bg=="
        }
      ]
    },
    "recomputed_streams": 1,
    "created_at": "2026-10-29T22:23:20.371746519Z"
  }
  ```

- **`2026-10-29T22:23:20.383Z` — Step 3: Node pulls signed checkpoint (`GET /api/v1/agent/usage-checkpoint`) and replays `seq 1..3` + `replay_unavailable [4, 5]` + `seq 6`** `[Verified on dev/test stack]`:
  Signed checkpoint verified against the node `recovery` trust key:
  ```json
  {
    "signatures": [
      {
        "algorithm": "ed25519",
        "key_id": "01a189c2-afbf-7250-b72a-a29091e7e92e",
        "signature_b64": "5/jgJgmCJ8/u14FKMCnC3Qfd2fLQZ622olWZIqv9NkGDDo3lVDym9cRXgAJZIqNje63hLh5FjR7+2wR8jtsEAg=="
      }
    ],
    "signed": {
      "highest_contiguous_seq": "2",
      "issued_at": "2026-10-29T22:23:20.383726806Z",
      "ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
      "max_seen_seq": "2",
      "missing_ranges": [],
      "node_id": "0196ed00-0000-7000-8000-000000000301",
      "node_incarnation_id": "0196ed00-0000-7000-8000-000000000401",
      "recovery_record_id": "01a189c2-b133-7b6a-9006-2d8d7a99895f"
    }
  }
  ```
  Batch ingest response (`POST /api/v1/agent/usage-batches`):
  ```json
  {
    "blocked_seq": null,
    "highest_contiguous_seq": "6",
    "ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
    "node_id": "0196ed00-0000-7000-8000-000000000301",
    "node_incarnation_id": "0196ed00-0000-7000-8000-000000000401",
    "received_at": "2026-10-29T22:23:20.385579324Z",
    "results": [
      { "error": null, "event_seq": "1", "status": "duplicate" },
      { "error": null, "event_seq": "2", "status": "duplicate" },
      { "error": null, "event_seq": "3", "status": "accepted" },
      { "error": null, "event_seq": "6", "status": "accepted" }
    ],
    "max_seen_seq": "6"
  }
  ```

- **`2026-10-29T22:23:20.412Z` — Step 4: Exit reconciling mode (`fvpn-usage-admin complete-recovery`) and verify member rollup (`fvpn-usage-admin drill-down`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    complete-recovery \
    --actor 0196ed00-0000-7000-8000-000000000999

  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    drill-down \
    --member-id 0196ed00-0000-7000-8000-000000000101 \
    --period 2026-10
  ```
  Verified totals: `upload_bytes = "12000"` (`1000 + 2000 + 3000 + 6000`), `download_bytes = "24000"` (`2000 + 4000 + 6000 + 12000`), `total_bytes = "36000"`, `gap = true`, `reconciliation_in_progress = false`.

### 2.5 Measured on the Lab — Live `AT30` Second Half (`#221`, `2026-10-30`, run `0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8`, [`docs/e2e/at/secops/AT30.md`](../e2e/at/secops/AT30.md), [`docs/e2e/at/secops/restore-reconcile.md`](../e2e/at/secops/restore-reconcile.md))

In security-ops follow-up 8 (`#221`, `2026-10-30`, run `0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8` on live stack `w4-secops` / port base `30000`), central PostgreSQL was restored to the `T0` checkpoint (`2026-10-30T22:07:20.427256Z`, `highest_contiguous_seq = 2`) while the retained exit node (`node1`), the payment provider (`paymock`), and NetBird (`N1`) remained at `T1` (`2026-10-30T22:09:17.078556Z`, `highest_contiguous_seq = 5`):
- **Isolated `T0` restore & fail-safe negative (`120.12 s`)**: `deploy/backup/restore-pitr.sh` + `deploy/backup/verify.sh` restored and verified the `T0` checkpoint in **`3.64 s`** (`6/6` post-restore checklist sections passed; primary-overwrite refusal exited `2`). Starting `fvpn-control` against the restored `T0` database **without** running `freeze` / `begin-recovery` for a measured **`120.12 s`** (`>= 2 minutes`) proved the fail-safe negative: while idle (`seq 3..5` already archived on the node at `T1`), `UploadOnce` skipped archived events, and when post-`T1` traffic generated `seq=6`, central returned `highest_contiguous_seq=2` under the old `ledger_epoch` and the agent rejected the response with `ErrAckRegress` (`highest_contiguous_seq regressed from 5 to 2 in epoch ...`), keeping `seq=3..5` stranded until `ledger_epoch` was rotated.
- **Five measured restore-protocol steps (`105.94 s` protocol RTO, `111.51 s` total outage RTO, `116.651 s` pre-replay DB RPO, `0 s` post-replay service usage RPO)**:
  1. **`fvpn-usage-admin freeze --reason "AT30 restore drill"`** (with `fvpn-control` and `fvpn-worker` stopped): set `usage_ledger_state.mode = 'frozen'` (`0.08 s`).
  2. **`fvpn-usage-admin begin-recovery --reason "AT30 restore to T0" --checkpoint-at "2026-10-30T22:07:20.427256Z" --key-id <recovery-key-id> --private-key-file <recovery.key>`**: rotated `ledger_epoch` (`7232c191-6989-4246-be92-cf0d74a15091` -> `31fa2c6f-a685-4919-92b4-41bf7614c158`), created `usage_recovery_records` (`130fd33b-5da0-4051-9bd9-5d2859a49c2f`), recomputed `usage_checkpoints` watermarks (`highest_contiguous_seq = 2`), and entered `mode = 'reconciling'` (`0.11 s`).
  3. **Start `fvpn-control` and `fvpn-worker` so retained nodes reconcile and replay**: `node1` received `409 LEDGER_EPOCH_MISMATCH`, fetched the signed checkpoint from `GET /api/v1/agent/usage-checkpoint`, verified its Ed25519 signature, un-archived locally retained `seq=3..5` (`archived=1 -> 0`), and replayed `seq=1..6` (`seq=1..2` `duplicate`, `seq=3..6` `accepted`) — **`5/5` (`T0..T1`) events replayed with `5/5` canonical SHA-256 matches, `0` lost, `0` double-counted, `0` gaps (`gap=false`)**, reducing service usage RPO from the **`116.651 s`** (`T1 - T0`) database window to **`0 s`**.
  4. **`fvpn-usage-admin complete-recovery --recovery-id 130fd33b-5da0-4051-9bd9-5d2859a49c2f`**: marked recovery `completed`, cleared `reconciliation_in_progress`, and returned `usage_ledger_state.mode` to `'normal'` (`0.09 s`).
  5. **Post-recovery readiness checks and external provider / NetBird reconciliation**: `/healthz = 200`, `/readyz = 200`, worker `/readyz = 200`, node `control_status = online` and `safety_status = passed`; `billing.reconcile` converged `m1` from restored `T0` `active` to `subscriptions.status = 'canceled'`, `refund_rows = 1`, and `entitlement.state = 'expired'` (`member.access.withdrawn` audited, peers removed from `fvpn-paid-*`). Measured protocol RTO (`T2 - T3`) was **`105.94 s`** (within the `15 min` target) and total outage RTO (`T1 - T3`) was **`111.51 s`**.

### 2.6 Historical follow-up 10 measurement — `AT30` passed (`#265`)

**Historical follow-up 10 measurement — AT30 passed again (#265), run `0e8ff9a4-4fa1-4154-ae35-847a5be4d985` (`0e8ff9a4`), product `d9e28f8`, on the shared development host:** pre-replay database RPO **95.507504 s**, post-replay service usage RPO **0 s**, protocol RTO **93.532554563 s**, total outage RTO **106.868537734 s**. These are shared-host measurements, not production objectives. Historical #242 and #221 measurements remain separately dated below. SECOPS-16/17 closed against #240 (`past_due` admitted only while `paid` stays true); SECOPS-20 closed against #241 (D-079); SECOPS-18/19 revalidated, SECOPS-05 retained, D-077 proved through HTTP. SECOPS-21 stays open. AT29/AT31 were deferred to devstack follow-up 15 exports; those exports do not by themselves promote the full AT cases. Source: [AT30 Follow-up 10 archive](../e2e/at/secops/history/follow-up-10/AT30.md) and [exact JSON](../e2e/at/secops/history/follow-up-10/restore-reconcile.json). Security-ops follow-up 11 (#274) archived all 104 files byte-identical under D-098; its live AT29 / AT31 / AT30 runs were not executed. D-103 assigns the existing `fvpn-w4-secops-*` namespace to follow-up 12.

#### Historical follow-up 9 (#242)

Security-ops follow-up 9 ran on application base `205633f` during `2026-10-31T01:36:15Z`–`2026-10-31T01:46:57Z` (2026-10-30 EDT), run `42396951-2d9a-49f4-95c3-45f33d9519d5`. **AT30 passed live: D-076 fence proven, SECOPS-19 visibility, replay `3/3`, RPO `0 B`, total RTO `121.88 s` (`121.9 s`).** Sources: [`AT30.md`](../e2e/at/secops/AT30.md), [`restore-reconcile.md`](../e2e/at/secops/restore-reconcile.md), and [machine-readable evidence](../e2e/at/secops/history/follow-up-9/restore-reconcile.json).

| Measurement | Follow-up 9 (`#242`) |
|---|---|
| Isolated T0 restore / checklist | `7.17 s`, `6/6` passed; primary-overwrite refusal exit `2` |
| Fail-safe negative without epoch rotation | `120.12 s`; archived `seq 2..3` stayed stranded, central contiguous watermark stayed `1` while max seen advanced to `4`; `ErrAckRegress` log count `0` |
| Freeze / begin-recovery / replay / complete-recovery | `0.06 s` / `0.07 s` / `84.75 s` / `0.06 s` |
| Pre-replay database RPO | `90.301 s` |
| Post-replay service usage RPO | `0 s` / `0 B` |
| Protocol / total outage RTO | `112.35 s` / `121.88 s` |
| Retained ledger comparison | `L0=1` → `L1=3`; `3/3` events and canonical SHA-256 matches (`78833` B), plus one post-T1 event (`10160` B); zero lost, double-counted, or gaps |

**SECOPS-18 closure:** the device revoked between T0 and T1 (`peerM2D2`) stayed withdrawn after restore: fence count `1`, one `device.authorization.external_withdrawal` audit, worker log `device authorization unconfirmed during ledger restore` with four fields, `fvpn_device_restore_fence_decisions_total = 2`, member status `pending`, and no counted forwarding while its sibling forwarded. Replaying the original T0 idempotency key kept the fence. Fresh `AuthorizeDevice` (`member.device_authorize`) cleared it (`count=0`) and restored counted flow (`download: "1515"`, `upload: "1066"`). The never-observed post-restore device (`peerM2D3`) was authorized with no fence and counted traffic; the centrally revoked negative (`peerM2D1`) stayed revoked, unfenced, and without counted traffic. Entitlement restoration does not clear a fence.

**SECOPS-19 closure:** during the reconciling hold, `node_observations.report_payload.usage` reported `unacked_event_count = 3` and `unacked_payload_bytes = "2352"`; `fvpn-agent usage snapshot -json` reported `unacked_count = 3` and the same payload bytes. After complete-recovery and replay both reached zero events and `"0"` bytes. Billing reconciliation converged the canceled member to expired entitlement with one refund; the other member remained active without an entitlement extension. SECOPS-05 also closed in this run.

This run exercised the service-level `AuthorizeDevice` against the live database and worker; it predates the admin HTTP route and `NodeDetail.usage_outbox`. Those are now served by `#238` / `#244` and `#243` respectively. Use `GET /api/v1/admin/devices?authorization_unconfirmed=true` and `POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize` as described in Phase 6.3. The newer outbox projection uses `db/migrations/00189_node_incarnation_usage_outbox.sql`; it is informational, null before a usage observation, and included in the node ETag.

**Post-restore generation verification (`SECOPS-21`, `#245`):** **SECOPS-21 remains open. Symptom:** security-ops follow-up 10 (#265) saw first post-restore offer generation `11` while the node retained `14`, despite #245's floor over `nodes.desired_config_generation`, `nodes.applied_config_generation` and stored desired-state history. Older offers raise `ErrStaleGeneration`. **Rule (D-095, #270):** stop control/worker, restore T0, and freeze-ingest before starting either service on the restored database. The compiler holds the first post-restore compile until a fresh observation from the current incarnation arrives after `usage_ledger_state.mode` left `normal`. The observable hold is `pending fresh observation after restore` (retryable `503 DEPENDENCY_UNAVAILABLE`, node drift `desired_state_restore_observation`, operation step `fresh_observation`). **Verification:** after a fresh applied `14` report, require first new offer `15`, node acceptance without `ErrStaleGeneration`, then a separate applied ACK and converged `config_generation.desired` / `config_generation.applied`. Completing recovery alone must not release the hold. D-099 is fixed at the HTTP layer in wire-nodes follow-up 29 (#277). `CompileTx` always proposes `max(desired, applied, stored)+1`; cache reuse is allowed for the cached generation or its ordinary one-step proposal only when no higher applied/stored generation exists and no restore observation is outstanding. Policies follow-up 8 (#286) promotes `TestAT30_D095_HTTPRestoreCacheProbe`: the exact 503 hold precedes offer `15` above retained `14`; the pre-#277 negative control fails with `304 / generation 11`. The fixture ACKs are simulated. Do not close SECOPS-21 from the compiler tests alone. Live closure requires the first delivered offer, a matching node-local accepted envelope/ETag, successful application and a separate real applied ACK; absence of a stale-generation log is insufficient. See the [HTTP fix note](../nodes/2026-10-31-secops-21-handover.md) and [live closure checklist](../policies/2026-10-31-secops-21-handover.md).

---

## 3. Repeatable AT30 Restore & Reconciliation Checklist (`§17.2`–`§17.4`)

Execute every step in order during a real disaster recovery event or scheduled quarterly restore drill.

### Phase 1: Isolate Central Services and Preserve Out-of-Band State

Carried gap: production custody and independent anti-rollback remain unimplemented by FileProvider. Preserve independent authority evidence; the staging inspection below cannot prove whole-directory non-rollback.

- [ ] **1.1 Stop `fvpn-control` and `fvpn-worker`** so no new mutations, outbox jobs, or usage ACKs occur against the damaged database `[production only]`:
  ```bash
  systemctl stop fvpn-control.service fvpn-worker.service
  ```
  *(Note: Exit nodes continue forwarding traffic for existing members while their serving leases and member service deadlines remain valid (`86,400s` default max offline lease, `§16.3`), and buffer new usage events in local SQLite (`§10.5`, `§10.8`).)*

- [ ] **1.2 Verify `/var/lib/fvpn/staging-authority/vault` Integrity (`§16.2`, `§17.2`)**:
  Preserve the current authority outside the PostgreSQL restore. For a staging FileProvider, run `fvpn-vault inspect -vault /var/lib/fvpn/staging-authority/vault -check-keys` against the actual configured vault path and review the [key-use inspection](../pki/inspect.md#key-use-inspection) results. This checks current active central keys locally; it cannot prove that the whole directory was not rolled back. Preserve newer revocations and rotations using independent evidence and follow [rotation and restore](../pki/rotation.md); production custody and anti-rollback are not implemented by FileProvider.

### Phase 2: Restore PostgreSQL 17 and Freeze Usage Ingestion

Carried gap: native PITR below requires the PostgreSQL/restore repository environment described in [backup §6.2](../ops/backups.md#62-restoring-into-a-clean-host-data-directory-production-host-replacement); a production delivery procedure for that environment is absent.

D-095 (#270), restore runbook Step 1: keep control and worker stopped, restore T0, then run freeze-ingest and verify success **before services start on the restored database**. Starting the worker before freezing leaves the original race open; see §2.6.

- [ ] **2.1 Restore PostgreSQL 17 to the Target Point in Time (`restored_to`)**:
  - **Physical PITR (`pgBackRest`, recommended)** `[production only]`:
    ```bash
    # Run as the backup OS account with the documented repository environment.
    # Use a new empty path on disk; keep the primary and its pg1-path unchanged.
    /usr/local/lib/fvpn/backup/restore-pitr.sh \
      --stanza fvpn --target-time "2026-10-29T18:02:00Z" \
      --primary-dir /var/lib/postgresql/17/main \
      --target-dir /var/lib/postgresql/17/restore-candidate
    ```
    Start the candidate with an isolated configuration/socket and no public listener, or use `verify.sh --host-repo --primary-host` for a `--network none` copy. Check migrations and the read-only checklist against the candidate. Only after validating that copy, preserve the old cluster and explicitly switch the service configuration to the candidate; do not overwrite the old data. Steps 2.2 onward use the checked candidate's connection configuration.
  - **Logical Restore (`pg_restore`)** (`docs/db/README.md`, `F-02`–`F-04`):
    `[production only]` — this logical restore path remains unverified. `guard_append_only()` guards `UPDATE`/`DELETE`/`TRUNCATE` on `audit_events`, `billing_records`, `usage_adjustments`, `usage_events`, `usage_samples`, `projection_receipts`, `usage_event_payloads`, `usage_ledger_gaps` and `fup_evaluations`; on `usage_rollups` it guards `DELETE`/`TRUNCATE`. `usage_checkpoints` instead has a separate `forbid_delete()` guard for `DELETE`/`TRUNCATE`.

    No setting provides a general bypass of `guard_append_only()`, and it does not read `fvpn.restore`. Its sole exception returns `OLD` for a `DELETE` on `audit_events`, `usage_event_payloads` or `usage_rollups` when `current_user` owns `public.purge_retained(text,timestamptz,text)` and `fvpn.retention_purge` is `on`. Every other guarded operation raises, including every `TRUNCATE`. The authorized retention function manages that setting; see the [retention contract](../db/README.md#roles-grants-and-retention).

    Other triggers use `owner_setting('fvpn.restore')`, which requires both the setting to be `on` and `current_user` to own `public.owner_setting(text)`. It allows historical replay by skipping the node-table snapshot lock, target-insert transaction/eligibility checks, and comparison with the current eligible node population; the stored batch `target_count` must still match its restored targets. It also skips node-lease current-incarnation/offline-deadline checks, member-lease transaction/eligibility/deadline checks, the operation guard on historical `INSERT` (not `UPDATE`), and the billing service-interval check on authorized `member_devices` inserts/updates and `serving_lease_members` inserts. It is not a general constraint or trigger disable switch.

    The [database restore contract](../db/README.md#restore-of-historical-snapshots) describes the existing replay prerequisites. Carried gap: validating the dump, target schema and dependency ordering for this example remains [carried work](follow-up-51.md#existing-procedures-linked-and-carried-gaps); the example command is retained without a new restore procedure:
    ```bash
    pg_restore --single-transaction -U fvpn_migrator -d fvpn /backups/fvpn-logical.dump
    ```

- [ ] **2.2 Verify Schema Migration Status (`fvpn-migrate status`)** `[Verified on dev/test stack]`:
  ```bash
  FVPN_DATABASE_URL="${RESTORED_MIGRATOR_DSN:?supply the restored candidate migrator DSN}" /usr/local/bin/fvpn-migrate status
  FVPN_DATABASE_URL="${RESTORED_MIGRATOR_DSN:?supply the restored candidate migrator DSN}" /usr/local/bin/fvpn-migrate up
  FVPN_DATABASE_URL="${RESTORED_MIGRATOR_DSN:?supply the restored candidate migrator DSN}" /usr/local/bin/fvpn-migrate status
  ```
  Supply the separate `fvpn_migrator` operator credential for the checked restored candidate, as in [deployment database roles](deploy.md#32-least-privilege-database-roles-docsdbreadmemd); do not use either runtime DSN. Confirm every file in the deployed release's `db/migrations/*.sql` is applied on that target and the final status reports no pending version, as described in §2.3.

- [ ] **2.3 Freeze Usage Ingestion Before Starting `fvpn-control` (`§17.3 Step 1`)** `[Verified on dev/test stack]`:
  ```bash
  FVPN_DATABASE_URL_FILE="${RESTORED_RECOVERY_DSN_FILE:?supply the restored candidate recovery DSN file}" \
    /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    freeze \
    --actor "${ADMIN_ACTOR_UUID}" \
    --reason "PITR restore from snapshot ${RESTORED_TO_UTC}"
  ```
  `RESTORED_RECOVERY_DSN_FILE` names an operator-provisioned, service-readable mode-`0600` file containing the `fvpn_recovery` DSN for the checked restored candidate; use the same override for begin-recovery and complete-recovery below. `FVPN_DATABASE_URL_FILE` overrides the control-role default ([Central configuration](../control/README.md)).

  While `mode = 'frozen'`, `POST /api/v1/agent/usage-batches` returns `503 DEPENDENCY_UNAVAILABLE` (`retryable: true`) and never ACKs events against stale watermarks.

### Phase 3: Rotate `ledger_epoch` and Start Central Services (`§17.3 Steps 2–6`)

- [ ] **3.1 Begin Ledger Recovery (`fvpn-usage-admin begin-recovery`)** `[Verified on dev/test stack]`:
  ```bash
  FVPN_DATABASE_URL_FILE="${RESTORED_RECOVERY_DSN_FILE:?supply the restored candidate recovery DSN file}" \
    /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    begin-recovery \
    --actor "${ADMIN_ACTOR_UUID}" \
    --reason "Restore from ${RESTORED_TO_UTC} snapshot" \
    --restored-to "${RESTORED_TO_UTC}" \
    --affected-since "${AFFECTED_SINCE_UTC}"
  ```
  Verify in the JSON output that:
  1. `new_ledger_epoch` differs from `previous_ledger_epoch`.
  2. `mode` is `"reconciling"`.
  3. `signed_record` carries a valid `ed25519` signature from the central `ledger-epoch` key.

- [ ] **3.2 Start `fvpn-control` alone; keep `fvpn-worker` stopped** `[production only]`:
  ```bash
  systemctl start fvpn-control.service
  curl -fsS https://vpn.example.com:8443/readyz
  ```
  Before starting the worker or opening public traffic, re-apply recorded post-snapshot node/device revocations, administrative suspensions and grant withdrawals using [backup Step 7.1](../ops/backups.md#step-7-reconcile-entitlements-revocations-publication-intents--fleet-updates-174-secops-05). Node revocation commits lifecycle/publication intent and certificate/incarnation revocations (and persists the local identity fence before commit); device revocation commits revoked status. Suspension and grant withdrawal commit their intent and recomputed entitlement in the accepting transaction. The worker's first reconciliation therefore reads those committed changes. An accepted operation still does not prove remote withdrawal.

  Carried gap: restricted recovery startup is not implemented. Control alone still serves agent requests and may issue leases from restored authorization while newer intent is being re-applied; stopping the worker is not a global forwarding fence, and D-076 requires an earlier authorized observation plus a current withdrawal. The repository has no startup mode that completes usage replay, entitlement computation and policy synchronization before enabling the automatic worker as spec §17.4 step 5 requires. The order here reduces exposure but does not establish that fail-closed recovery boundary.

  Once the recorded withdrawal intents are committed, start `fvpn-worker.service` as in backup Step 7.2. Verify each accepted operation's terminal result and observed enforcement before opening public traffic; continue the checks below with that ordering.

### Phase 4: Reconcile Payments and Member Entitlements (`§9.3`, `§17.4`)

A point-in-time database restore may roll back webhook receipts (`inbox_events`), billing history (`billing_records`), or subscription state transitions that occurred between `restored_to` and the incident.

- [ ] **4.1 Verify Automatic Billing Reconciliation (`billing.reconcile`)**:
  After step 3.2's withdrawal-intent ordering, `fvpn-worker` runs `billing.reconcile` every `members.json:reconcile_seconds` (`900s` in the example). It schedules reconciliation for subscriptions already stored in the restored database, including canceled ones; processing reads current provider subscription, invoice and refund state.
  Carried gap: inventory and reconstruction of provider subscriptions absent from the restored rows. This sweep does not establish complete recovery of missed provider events. For known provider events and stored subscriptions:
  1. Replay unacknowledged webhook events from the Stripe Dashboard (`Developers -> Webhooks -> Send/Resend`) covering `[restored_to - 1h, now]`. Webhook handling deduplicates by `(source, external_event_id)` in `inbox_events` and never creates duplicate charges (`BILL01`).
  2. Inspect `subscriptions`, `entitlements`, and `members` for any subscription updated during the RPO window:
     ```sql
     SELECT id, member_id, external_subscription_id, provider_status, paid_through, updated_at
     FROM subscriptions
     ORDER BY updated_at DESC LIMIT 50;
     ```
- [ ] **4.2 Re-apply Post-Snapshot Admin Suspensions or Manual Grants (`§9.2`, `§9.3`, `§17.4`)**:
  Confirm that step 3.2 re-applied every recorded suspension and grant withdrawal before worker startup. Use [explicit grant withdrawal](../members/README.md#explicit-grant-withdrawal-revision-14-at29), including its required reason and accepted-operation read-back. A restored grant is not evidence that it remains authorized. Re-issue a recorded grant only after checking later withdrawals, using the finite grant fields in the [API contract](../../contracts/openapi.yaml). Track accepted operations and observed enforcement separately.

### Phase 5: Reconcile NetBird Overlay, Node PKI, and DNS State (`§5.2`, `§6.4`, `§12.2`, `§17.4`)

Central PostgreSQL is the source of truth for desired state; NetBird and AdGuard Home are downstream enforcement targets (`§3`, `§15`).

- [ ] **5.1 Re-revoke Any Post-Snapshot Node Incarnations (`§6.4`, `§17.2`)**:
  Confirm that step 3.2 re-applied recorded node revocations/retirements before worker startup through `POST /api/v1/admin/nodes/{node_id}/revoke` or `DELETE /api/v1/admin/nodes/{node_id}` using the [operator revocation procedure](operator-guide.md). The accepting transaction commits withdrawal intent; the local identity fence is persisted before that commit. NetBird peer/route cleanup is asynchronous. Check operation completion and remote read-back before declaring withdrawal observed. Never re-enable a revoked identity to retrieve usage.
- [ ] **5.2 Verify NetBird Node & Member Peer Reconciliation (`node.reconcile`, `members.reconcile`)**:
  The worker has separate node, member and discovery jobs; they do not implement a global orphan-peer or published-route/policy drift cleanup:
  - `node.reconcile` runs on `nodes.json:reconcile_seconds`. It reconciles generation holds, updates fleet job state/attention, and expires bootstrap secrets in the database. Fleet reconciliation does not read or delete NetBird peers.
  - `members.reconcile` runs on `members.json:device_sync_seconds`, recomputes member entitlements and calls `SyncMember` over the member's existing `member_devices` rows. `SyncDevice` applies the intended authorization and verifies it with peer read-back. Under [D-076](../members/README.md#restored-authorization-does-not-re-grant-d-076), an otherwise eligible authorized device with a previously authorized observation and an unexplained current withdrawal is fenced after confirming that withdrawal. Existing fences keep `allow=false`; fenced devices appear pending until explicit re-authorization. A new fence appends `device.authorization.external_withdrawal` with reason `external_withdrawal`. The warning `device authorization unconfirmed during ledger restore` and `fvpn_device_restore_fence_decisions_total` apply when ledger mode is `frozen` or `reconciling`; fencing itself is not restricted to those modes. Keep the fence through restore and follow step 6.3 after `complete-recovery`.
  - The separate `devices.discover` sweep skips userless and ephemeral peers. An unknown peer with an eligible verified member identity can enter `ObserveAndAuthorize`; absence from the restored device table does not imply deletion.
  - Carried gap: inventory and revocation of peers authorized after the restore point, including peers absent from restored records. No complete procedure for that inventory/revocation is supplied here; these sweeps must not be treated as proof that every such peer, route or policy has been withdrawn.
  List failed and in-flight operations with SQL; investigate failures and operations that remain pending or running:
  ```sql
  SELECT id, kind, status, resource_type, resource_id, error_code, created_at
  FROM operations
  WHERE status IN ('pending', 'running', 'failed', 'partially_succeeded')
  ORDER BY created_at DESC;
  ```
  For each returned ID, an authenticated administrator can retrieve details with `GET /api/v1/admin/operations/{operation_id}` (`operationId: getAdminOperation`). The contract has no operations-list endpoint. A `202` response to a mutation records acceptance; verify the operation's terminal result and observed state before declaring recovery complete.
- [ ] **5.3 Verify AdGuard Home DNS Policy Convergence (`dns.publish`)**:
  Inspect published heads and their immutable resolver documents; confirm applied state separately by resolver read-back:
  ```sql
  SELECT h.definition_id, h.profile_revision_id, h.policy_generation,
         h.status, h.document_sha256, d.created_at
  FROM dns_publication_heads h
  JOIN dns_resolver_documents d ON d.profile_revision_id = h.profile_revision_id
                               AND d.policy_generation = h.policy_generation;
  ```

Portal follow-up 16 (#255) supplies `/admin/devices` with authorization-unconfirmed, status and member filters, filter-bound cursors, a confirmed Re-authorize action, a neutral receipt for an existing operation, and a separate paginated read-back until the grant is visible. The node detail also has the usage-outbox block. Operators may use the portal page or the API above; a 202 receipt alone is not a completed grant.

### Phase 6: Node Usage Replay and Complete Recovery (`§10.8`, `§17.3 Steps 7–10`)

- [ ] **6.1 Monitor Node Checkpoint Pull & Replay (`§17.3 Steps 7–9`)**:
  As each online exit node attempts `POST /api/v1/agent/usage-batches` with the pre-restore `ledger_epoch`, `fvpn-control` returns `409 LEDGER_EPOCH_MISMATCH` (`current_ledger_epoch` in details). The node automatically:
  1. Calls `GET /api/v1/agent/usage-checkpoint` and verifies the `ed25519` signature using its `recovery` trust bundle.
  2. Replays all locally retained events with `event_seq > highest_contiguous_seq` (plus any `missing_ranges`) under the new `ledger_epoch`. Events already present in the restored snapshot return `status: "duplicate"` and are never double-counted.
  3. Reports any purged sequence ranges in `replay_unavailable` (accepted only while `mode = 'reconciling'`, `D-027`), which inserts zero-byte tombstones and sets `gap = TRUE` on affected member rollups.
  Check stream convergence in PostgreSQL:
  ```sql
  SELECT node_id, incarnation_id, ledger_epoch, highest_contiguous_seq, max_seen_seq, updated_at
  FROM usage_checkpoints
  ORDER BY updated_at DESC;
  ```

- [ ] **6.2 Complete Ledger Recovery (`fvpn-usage-admin complete-recovery`)** `[Verified on dev/test stack]`:
  Once all active nodes have adopted `new_ledger_epoch` and drained their replay queues, exit reconciling mode:
  ```bash
  FVPN_DATABASE_URL_FILE="${RESTORED_RECOVERY_DSN_FILE:?supply the restored candidate recovery DSN file}" \
    /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    complete-recovery \
    --actor "${ADMIN_ACTOR_UUID}"
  ```
  This marks the `usage_recovery_records` entry `completed`, clears `reconciliation_in_progress` on `usage_rollups` (preserving `gap = TRUE` where `replay_unavailable` occurred), and sets `usage_ledger_state.mode = 'normal'`.

- [ ] **6.2a Verify post-restore generation (`SECOPS-21`, `#245`)**: After the retained node reports its applied generation, trigger/observe the first compile and check `GET /api/v1/admin/nodes/{id}`: `config_generation.desired` (desired_generation) must exceed that reported applied generation. Agent logs must stop reporting `ErrStaleGeneration`. Once applied, the two generations may be equal. See §2.6.

- [ ] **6.3 Review Devices Fenced as Authorization-Unconfirmed (`D-076`, `D-077`, `D-081`, `#227`, `#238`, `#242`, `#244`)**:
  Immediately after `complete-recovery`, review devices fenced as authorization-unconfirmed and re-authorize explicitly where the member is entitled (`D-076`, `D-077`, `D-081`, `#227`, `#238`, `#242`, `#244`, `SECOPS-18`, `db/migrations/00188_device_authorization_fences.sql`, [`docs/members/README.md`](../members/README.md#restored-authorization-does-not-re-grant-d-076)):
  1. List every device needing re-authorization after the restore via **`GET /api/v1/admin/devices?authorization_unconfirmed=true`** (`listAdminDevices`, `#238`, `#244`, ordered by `(member_id, device_id)` with an opaque filter-bound keyset cursor, where each `AdminDevice` carries `authorization_unconfirmed: true` and `authorization_unconfirmed_since` set to `device_authorization_fences.fenced_at`), correlated with the worker log line `device authorization unconfirmed during ledger restore` (`device_id`, `member_id`, `peer_id`, `ledger_mode`, counted in `fvpn_device_restore_fence_decisions_total`), `GET /api/v1/admin/audit-events?resource_type=member_device&resource_id=<device_id>` (`action=device.authorization.external_withdrawal`, where the reason is recorded alongside the worker log line), or by joining `device_authorization_fences` (`device_id`, `fenced_at`, `observed_group_ids`) to `member_devices` (`id`, `member_id`, `peer_id`, `status`):
     ```sql
     SELECT f.device_id, d.member_id, d.peer_id, d.status, f.fenced_at, f.observed_group_ids
     FROM device_authorization_fences f
     JOIN member_devices d ON d.id = f.device_id
     ORDER BY f.fenced_at DESC;
     ```
  2. Entitlement restoration does not clear a fence. For each fenced device where the member is still entitled and the device was not revoked after `restored_to`, issue an explicit re-authorization via **`POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize`** (`authorizeMemberDevice` / `AuthorizeDevice`, admin session, CSRF, new `Idempotency-Key`, no request body, `#238`, `#244`): it deletes `device_authorization_fences` in the same transaction, writes a `device.authorize` audit event on `member_device:<device_id>`, enqueues `device.sync`, and returns `202 Accepted` with operation kind `member.device_authorize` (returning the device's existing `member.device_authorize` operation when `member_devices.status` is still `authorized` — confirm completion by checking `GET /api/v1/admin/devices` or `GET /api/v1/member/devices` for `status: authorized` and `authorization_unconfirmed: false` after the worker sync; `404 NOT_FOUND` if the device is not the member's; `409 STATE_CONFLICT` if `revoked`; **`409 DEVICE_LIMIT_REACHED`** per `D-081` if over the member's plan allowance; a revoke via `DELETE /api/v1/member/devices/{device_id}` also clears the fence; see [`operator-guide.md`](operator-guide.md#71-suspending-and-reactivating-a-member-patch-apiv1adminmembersmember_id) §7.1).

---

## 7. Automated Docker PITR Drill Names (`tests/backup`, `FVPN_BACKUP_DRILL_PREFIX`)

`TestAT30_BackupPITRDrill` (`go test -tags integration ./tests/backup/`, also `make -f mk/backup.mk backup-drill`) builds a PostgreSQL 17 + pgBackRest image and runs the primary, the isolated restore, and `verify.sh` on one Docker prefix. Concurrent drills must not share that prefix. The repository-wide integration gate and a second local run can execute at the same time.

Set `FVPN_BACKUP_DRILL_PREFIX` to choose the prefix. The value must match `fvpn-w4-ops-backups-<lowercase-run-id>`: the family name, then one or more hyphen-separated segments of lowercase letters and digits, at most 63 characters. `.github/workflows/backup.yml` sets `fvpn-w4-ops-backups-gha<run_id>-<run_attempt>`.

When the variable is unset, the test draws 4 bytes from `crypto/rand` and uses `fvpn-w4-ops-backups-` plus those bytes as 8 lowercase hex digits.

The test creates and deletes only objects owned by that prefix (the name equals the prefix, or starts with the prefix plus `-`). It registers the cleanup with `t.Cleanup`, so a failure removes the same set. It does not remove the bare family, a neighboring run, or `fvpn-pg`.

| Object | Name |
|---|---|
| Image tag | `<prefix>-postgres:17` |
| Primary container | `<prefix>-primary` |
| Primary volume | `<prefix>-primary-pgdata` |
| Repository volume | `<prefix>-repo` |
| Restore container | `<prefix>-restore` |
| Restore volume | `<prefix>-restore-pgdata` |
| `verify.sh` prefix | `<prefix>-verify` |
| Network name reserved for cleanup | `<prefix>-net` |

The primary and the restore container join Docker's built-in `none` network (`--network none`, zero published host ports). `<prefix>-net` is not created. Cleanup still removes a network of that name if one was left behind.

Stale objects from a killed drill are not removed by the next random-prefix run. Remove the whole `fvpn-w4-ops-backups-*` family, and only that family, with:

```bash
make -f mk/backup.mk backup-drill-clean
```

Do not run `backup-drill-clean` while a drill is still using a `fvpn-w4-ops-backups-*` prefix. The test never calls that target.

Evidence is written only with `FVPN_BACKUP_DRILL_RECORD=1`; normal integration runs do not change `docs/`. CI exposes this only as the manual `workflow_dispatch.record_evidence` input. Archive ages and local command durations are reported separately; service RTO remains unmeasured.

### Follow-up 16 restore evidence (2026-10-31 EDT)

Security-ops follow-up 12 (#301), run `dd845a7f-2660-431c-a8e6-c513108cc07d`,
passes AT30 on the shared development host: pre-replay database RPO **106.45 s**,
post-replay service usage RPO **0 s**, protocol RTO **66.90 s**, total outage RTO
**74.53 s**, replay **5 events / 5 hashes / 0 lost / 0 duplicate / 0 gaps**.
D-095's exact fresh-observation hold is proved. SECOPS-21 remains open: the
first probe/stored generation 16 exceeds retained 15, but first delivery,
matching local acceptance and the applied ACK remain unproven; capture is
pending devstack follow-up 18 / wire-nodes follow-up 32. Earlier measurements
remain historical, and no candidate-hardware guarantee follows. See the
[current acceptance snapshot](../ops/README.md#follow-up-16-acceptance-snapshot-2026-10-31-edt).

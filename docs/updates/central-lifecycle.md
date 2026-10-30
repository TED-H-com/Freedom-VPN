# Central release and fleet updates

This service implements UPD01–UPD05, Project Spec §13, §14.2/4/5 and §15.5.6.
HTTP handlers, desired-state signing and the local updater belong to their owners.
The generated HTTP types remain the wire contract.

## Registration and storage checklist

1. Provision a release-purpose `pkg/envelope.TrustStore` from the independently
   trusted deployment configuration. Never import a key from a submitted manifest.
2. Construct `releases.NewValidator(trust, origins)`. Origins are exact HTTPS
   authorities, including ports: for example `https://releases.example.invalid`.
   No suffix matching, credentials, fragments or arbitrary member locations.
   Artifact, manifest, SBOM, notices and source URLs must all pass this allowlist.
3. Pass authenticated admin identity, UUID idempotency key and the raw JSON
   `ReleaseRegisterRequest` to `Service.Register`. Raw bytes preserve signature
   validation and reject duplicate JSON members. The service enforces
   `canonjson.MaxBytes` and the request schema, extracts only the raw envelope,
   verifies its signature/schema, then decodes `api.ReleaseRegisterRequest`.
   The hash is SHA-256 of the JCS
   **whole envelope**, as specified by the shared signing vector.
4. Registration verifies Ed25519 trust purpose/window/revocation, the embedded
   JSON Schema, decimal ranges, Ubuntu 24.04 amd64 and arm64 coverage, duplicate
   platform records, control API major 1, and policy/usage schema version 1.
   Per-device updater and local state compatibility still require node preflight.
5. Invalid input returns a safe `apierr.Error`. When canonical JSON is available,
   only its release ID, manifest hash and error code enter
   `release_registration_rejections`, with `eligible=false`. Untrusted manifests
   do not become registered releases. A different body for an existing release ID
   returns `STATE_CONFLICT`. Under decision D-013 (§15.3), each product version
   also belongs to exactly one immutable release: a second release ID with the
   same `product_version` returns 409 `STATE_CONFLICT`. The base
   `releases.version` UNIQUE constraint remains in force. Product versions are
   display labels, never an ordering or authorization rule.
6. With `Service.VerifyArtifacts=true`, registration creates a pending operation
   and `release.verify` outbox event, returns 202, and remains ineligible.
   `VerifyRelease` streams every artifact through a size limit and SHA-256 check
   outside database transactions. Only a pending verification can commit its
   result; a crash before commit can safely retry. Failed verification remains
   ineligible. Re-registration with a new idempotency key and the same manifest
   returns 409 `STATE_CONFLICT` with `details.verification_state="failed"` after
   failure, 202 with the existing operation while pending, or 201 once verified.
   Exact command replay retains its original ACK and operation (§14.5).
   The current implementation makes one fetch attempt per worker
   invocation and records a fetch error as failed; there is no infinite retry.
7. Use `NewHTTPSFetcher(validator, measuredTimeout)` in production. It disables
   environment proxies, rejects redirects, rejects private/special DNS answers
   and dials the checked address directly while retaining HTTPS hostname checks.
   The `Fetcher` interface is for equivalent controlled storage adapters and
   tests; implementations must preserve these origin and network restrictions.
8. Activate `release_support` through the policy owner and call
   `ApplySupportTx` in that same transaction. Approved IDs must be a subset of
   serving IDs; revoked IDs are disjoint and cannot be resurrected. Registration
   alone never approves installation. `Eligible` rechecks current signature
   trust whenever a batch, retry or node intent authorizes work.

The list response contains architecture, components, compatibility, verification
state, approval/revocation, fleet default and serving IDs. Verification state maps
`pending` to `verifying` and `failed` to `rejected` in the HTTP contract. No
lexical comparison of product versions participates in authorization (§13.2).

## Batch lifecycle

`CreateBatch` (also exposed as `SetFleetRelease`) accepts the generated
`api.UpdateBatchCreateRequest`. Under a fleet row lock, one transaction checks
idempotency, expected fleet generation and eligibility; snapshots all existing
nodes except revoked/retired; allocates monotonically increasing per-node
update generations and attempt 1 jobs for bound identities; changes the fleet default; and writes an
operation, audit, outbox event and progress projection. Any failure rolls back
all of these. A stale generation returns 409 `STATE_CONFLICT`.

The snapshot includes offline, unpublished, disabled and enrolling nodes. Missing
execution incarnations remain null until first authenticated intent retrieval binds
these targets once and creates attempt 1. Before binding, an enrolling target has
no job, `attempts=0`, `current_job=null`, and counts as waiting_online. Its reserved
generation still supersedes older work. A newer batch supersedes an unbound older
target without creating a phantom execution job. The snapshot's original
incarnation is never changed. Jobs retain the base NOT NULL target and execution
incarnation, composite FK, forward-phase and frozen-terminal-evidence guards.

Creation takes the base SHARE lock on `nodes` before reading the population, so
concurrent insertion, enrollment or lifecycle changes cannot evade membership.
The base deferred `all_nodes` count and membership validator is unchanged, and
snapshot membership is still sealed in its creation transaction.

`NodeUpdateIntent` returns `Work{Intent api.UpdateIntent, Artifact
api.ManifestArtifact}` for an mTLS `Identity`. Only `Intent` belongs in the
existing signed desired-state `update` field. The selected artifact allows the
compiler to check the node platform without modifying that field's schema.
The caller also reads the node's latest desired release; an older applying job
may legitimately name a different release while finishing its transaction.
Intent refresh returns the stored first `issued_at` and the stored extended
`authorization_valid_until`, including PostgreSQL timestamp precision. The signed
intent therefore agrees with the authorization row on every retrieval.

Queued/downloading/verifying jobs become superseded when a newer target appears.
Applying/checking jobs finish first while the desired pointer already names the
newest job. A reinstall gets independent first-sync work with a higher generation;
old incarnation work cannot be transferred. Nodes created after the snapshot
also get the fleet default in a separate one-node system batch with
`reason='first_sync'`. Intent/result batch and target IDs identify that actual
system snapshot; they are non-null. This never enlarges an earlier admin batch
or changes the fleet default/generation.

Internally, system batches use `scope='first_sync'`, a reserved system actor and
`first_sync_node_id`. A database CHECK requires exactly one target, that actor
and reason; a deferred guard requires exactly the named eligible node with its
current bound incarnation. The base snapshot-insert and SHARE-lock guards also
apply. This narrowly extends the base batch scope; public create requests still
accept only `all_nodes`, whose full-population validator remains mandatory.

The fleet lifecycle trigger cancels unfinished jobs and sets their target
`target_state` in the node owner's revocation/retirement transaction. It also
cancels the current unbound enrolling target. The base guards allow these states
but do not themselves initiate cancellation. Already-terminal targets, snapshot
identities and accepted result evidence remain unchanged. Revoked identities cannot obtain more work or submit control results.
No update code writes `exits.publication_intent`.

`Retry` accepts the generated retry request and authenticated command. It creates
new attempts only for current, retryable failed targets, keeps their release and
snapshot generation, and retains all old attempts. Each retry has its own operation
and exact job references. Explicit selections outside the batch, nonretryable
states, different incarnations and targets with newer work are rejected. The
control adapter turns the returned operation ID into the framework's Operation
response; 202 is acceptance, never completion.

## Result acceptance decision table (§13.8)

| Situation | Acceptance and durable effect |
|---|---|
| mTLS/body node or incarnation differ | `INCARNATION_MISMATCH`; no result accepted |
| Identity revoked or retired | `IDENTITY_REVOKED`; no new control authorization |
| Job/target/batch/attempt/generation differ | Reject, generally `STALE_GENERATION`; unknown job is `RESOURCE_NOT_FOUND` |
| Result payload hash or authorized manifest differs | `HASH_MISMATCH` |
| No recorded job authorization, expired start or expired pre-apply progress | `AUTHORIZATION_EXPIRED`; acquire fresh intent |
| Same result ID, same canonical payload | Return the exact stored ACK, including original received time |
| Same result ID, different canonical payload | Commit each distinct conflict hash and its audit once, return 409 `ACK_CONFLICT`; preserve original ACK |
| Terminal result lacks actual version or finish time | Reject; no success counted |
| Pre-health failed result has null health, an error code and the unchanged attempt release | Accept as failed, with needs_attention; never count success |
| Failed result lacks an error code at any stage | `VALIDATION_FAILED` with a durable rejection receipt; do not change the job or serving version |
| Failed result at/after check, or another terminal result, lacks health | Reject; no completion inferred |
| Succeeded/already_current without target version, all health checks, state schema or durable usage checkpoint | Reject; no success counted |
| Rolled back to unsupported/revoked release | `MINIMUM_VERSION_VIOLATION`, persist recovery attention; forwarding stays forbidden |
| Rolled back without a verified, trusted, readable restored manifest, restored health, candidate ID/failure, or membership in a nonempty rollback allowlist | Reject; never count rollback as target success |
| Older applying/checking job finishes | Finish that job's ledger, ACK as history, leave newest pointer/generation intact |
| Superseded/terminal job reports late, progress goes backwards, or already_current arrives after download began | Append history; preserve terminal ledger/current pointer |
| Validated succeeded/failed/rolled_back result for the current job arrives while its ledger is still queued | Commit the terminal job/target state and normal current-result ACK; no earlier progress delivery is required |
| Current terminal result with a greater generation, or differing versions at the same generation | Advance actual and last-good release according to result evidence, even if a heartbeat seeded another version before the result |
| Same-generation heartbeat already supplied identical versions, or the observed high-water is newer | Save result and ACK without rewriting the version; `applied_to_current_observation=false` |
| Report claims forwarding while disabled, disruptive, unsupported or recovery-required | `NODE_NOT_SERVICEABLE`; never rewrite publication intent |

All accepted results, their ACK, generation-conditional observation, job evidence,
batch projection and audit commit before an ACK is returned. The result digest
available in the existing contract is `manifest_sha256`; there is no separate
artifact-digest field in `UpdateResult`. Actual artifact verification belongs to
the updater, and its test evidence must accompany release qualification.

The updater can deliver only its durable terminal result after completing the
local transaction (§13.6, §13.10.3; AT26). A current `rolled_back` result therefore
closes a queued target with its restored release, original failure code and the
existing attention/counting rules. `accepted_as_history=false` does not imply
`applied_to_current_observation=true`: identical same-generation values need no
rewrite, and the newer-generation fence remains in force. A later authenticated
observation with the matching job, generation and versions clears `unconfirmed`.

Heartbeat ingestion does not promote job phases. Result acceptance is independent
of that optional progress path. The base SQL guard permits rollback only from
applying/checking; after all result evidence validation, the result transaction
traverses applying and writes the final rollback in the same savepoint. Only the
terminal state is committed, with one result receipt and its audit. It does not
invent a progress report, loosen a database guard, or publish an intermediate
batch state. Rejected evidence, superseded jobs and frozen terminal jobs never
enter this transition.

An ACK already stored as history before this correction remains immutable on
replay. It does not repair that older queued target retroactively. Start a fresh
batch to exercise the corrected producer path; do not edit a sealed result or
rewrite its original receipt.

`testdata/at26-direct-rollback.json` in the fleetupdate package is the unchanged
AT26 producer capture, including passing restored checks and failed candidate
tunnel probe. The regression rebinds only job/release identity and authorization
times to isolated test data; it sends no applying/checking result before rollback.

## Counting example (§13.12)

A batch snapshots six nodes. Initially: queued=4, waiting_online=2; denominator=6,
state=waiting. Later: succeeded=2, already_current=1, failed=1, waiting_online=2;
still six and waiting. Retrying the failed target gives queued=1 in place of
failed=1, with attempt 2; the denominator is still six. When that retry succeeds,
one offline node rolls back and the other is revoked: succeeded=3,
already_current=1, rolled_back=1, cancelled=1; total six and
completed_with_exceptions. Four targets achieved the requested release.

Each target contributes exactly once: its cancelled disposition, otherwise its
latest attempt, or waiting_online/superseded if it has no job yet. Unbound targets
are included in cursor pages, timestamps and the same fixed denominator. The HTTP contract folds
cancelled_revoked/cancelled_retired into `cancelled`, and superseded_incarnation
into `superseded`; target rows preserve the exact reason. Any waiting_online
bucket takes display precedence over running work. All-terminal snapshots are
completed only if every target succeeded or was already_current. Empty snapshots
complete with zero targets. Cursor pages never change whole-batch counts.
Target `node_name` comes from the exit display name and is null only when the
node has no display name.

## Serving and observed versions

`ServingAllowed` implements §5.1/§13.2 and deliberately has no desired-version
argument. It requires valid identity, ready provisioning, valid policy and
isolation, member entitlement, published intent, a supported nonrevoked actual
release, and no applying/checking/recovery phase. `NodeVersionView` exposes the
version predicate separately so callers cannot mistake it for full permission.

For heartbeat integration, prefer `ObserveReportTx` using the existing
`api.ObservationReport`. The node owner inserts its validated, deduplicated
`node_observations` receipt in that transaction first. The adapter uses its
receipt time plus authenticated job/generation and observed time; it adds no wire
sequence field. `ObserveTx`/`Observe` are also available to integrations that
already maintain a trustworthy monotonic per-incarnation observation sequence.
Old sequences/generations cannot overwrite newer actual versions. Before the
incarnation's first accepted result, a bound observation may seed current only
from the current job's target or an already recorded current/last-good release.
It preserves last-good and can flag that recorded release as revoked. After an
accepted result, observations preserve the accepted versions and recovery; disagreements stay
unconfirmed. Missing observations stay unconfirmed.

If an agent reports a generation ahead of restored central state (§17.4), intent
retrieval refuses automatic first-sync/downgrade. An explicit new batch allocates
above the greatest desired/observed generation. Restore operators must reconcile
unknown journal/job IDs with durable audit evidence before resuming automatic work.

## Worker and integration wiring

- `release.verify` outbox: invoke `VerifyRelease`, then complete the fenced outbox
  claim. The operation carries verified digest evidence or failure.
- `fleet.update_created`, `fleet.first_sync_created`, `fleet.update_retry` and
  `fleet.release_support_changed`: wake desired-state compilation. Durable state
  also supports polling, so delivery loss does not erase work.
- Periodically call `Reconcile`. It recomputes batch/retry operations, marks
  unobserved queued jobs waiting_online, and sets needs_attention on jobs whose
  applying/checking phase exceeds `Config.StuckWindow`. It never guesses failure
  or success for stuck jobs.
- `Evidence(targetID)` exposes needs_attention, unconfirmed, last_result_at and
  last_observed_at. `UpdateTarget` already has both timestamps; both booleans are now exposed by the current contract and share one evidence
  predicate with batch mapping.
- Configure `AuthorizationTTL`, `OnlineWindow`, `StuckWindow` and fetch timeout
  explicitly. No production value is invented for §21 pending measurements.
- Transactions touching both areas should lock fleet_config before node rows.
  Snapshot creation takes the `nodes` SHARE lock before reading its members.
  The shared `db.InTx` retries deadlocks/serialization errors with bounded jitter.
  Services perform no external calls inside request transactions.
- Handlers render errors with `pkg/apierr.Write`; these packages contain no HTTP
  handlers. Authentication/CSRF, If-Match for policy activation and role checks
  remain the framework/policy owner's responsibility.

`ApplySupportTx` writes policy arrays and a separate support generation. The
policy owner should attach its active revision when projecting
`FleetReleaseState.release_support_revision_id`; the core schema currently has
no such pointer, so the release list returns null for that optional provenance.
The initial implementation serializes fleet mutations and scans batches during
reconciliation; measure production fleet volume before replacing this with
incremental projections.

## Database guard compatibility

Migration 00160 extends the base schema rather than replacing its job state or
identity guards. Derived job `node_id` and `release_id` are backfilled, populated
from their target on insert and protected by an additional immutable-fields
trigger. The base job target/incarnation/generation composite FK remains intact.

The additive result/retry tables and current-job pointer validate job references
with immediate constraint triggers using `FOR KEY SHARE`. Base jobs cannot be
deleted, truncated or change identity, so these references cannot become
orphaned. This preserves the base truncate guard's SQLSTATE 23514; an inbound FK
would cause PostgreSQL's earlier SQLSTATE 0A000 check to mask that guard. No base
FK, NOT NULL, state transition or terminal evidence restriction is removed.


HTTP and worker composition, including the receipt-transaction framework gaps,
is documented in [node wiring](../nodes/WIRING.md). The response mapping and
`Evidence` now share attention/confirmation predicates; `NodeVersionView` uses
the same confirmation predicate. Observations preserve central receipt time.

## Operation and audit evidence (D-029)

Create, first-sync, retry and release verification operations use control-core's
`operations.Create` and `operations.Transition`, with generated operation kinds
`update_batch.create`, `update_batch.retry` and `release.register`. Batch steps
separately record the persisted snapshot and target outcomes; retry operations
record their attempts. A batch with exceptions has a failed target step and a
successful snapshot step, so its operation is partially_succeeded. An entirely
successful batch/retry requires actual persisted target evidence.

Verification failure records a failed `verify` step with a safe error code.
`operations.Store.Get` exposes that code with a fixed, safe top-level error
message. Fetcher error text, URLs and payloads do not enter the operation response.
Every domain audit writer constructs a typed event with `audit.NewEvent` and
persists it with `audit.AppendEvent` in the action's transaction (§16.4).
Actors use contract types and nonzero UUIDs. Verification and first-sync use the
existing system singleton `00000000-0000-0000-0000-000000000003`; node reports
use their authenticated node UUID. Results are `succeeded` or `failed`, including
`failed` for conflicting results and refused rollbacks. Actions such as
`release.verify` and `update_result.accept` describe the audited action and are
independent of operation-kind enums.
Resource types are release, update_batch, update_target and node. Release support
changes audit each affected release, including removed releases. Request IDs and
operation IDs retain traceability; worker verification retains the registering
operation's request ID. Job/result UUIDs stay in allowlisted diff fields on real
node/target resources; they are never relabeled as other entity types. Free-form
reasons are excluded from audit diffs. The integration fixtures read every event
back through `audit.Store.List` and operations through `operations.Store.Get`,
assert zero anomalies, and verify actor IDs, resource relationships and correlation.

## Applied policy evidence (§13.10.3)

Contract revision 3 keeps `UpdateResult.policy_generation` required. It names the
actually applied desired-state `config_generation`, independently of job generation.
Migration 00161 derives an exact numeric column from the immutable result payload
and stores `policy_refs` from the matching node/incarnation/generation in
`desired_states`. Unavailable history leaves refs null; it never substitutes the
latest policy. Accepted old-job history and conflict payloads cannot overwrite
accepted evidence. `Evidence(targetID)` exposes the accepted job's policy generation
and references, including integers beyond JavaScript's safe range.

D-048 added nullable `policy_generation` and `policy_refs` to UpdateTarget.
`GetBatch` and `Evidence(targetID)` now use the same accepted-result evidence.
UpdateResult does not carry policy_refs; no undeclared wire fields are added.


## Terminal evidence and receipts (D-062)

The shared `api.UpdateStage` definition follows §13.6: claim, download, verify,
preflight, prepare, block_forwarding, final_sample, stop_services, migrate,
switch and start precede the health profile. The wire stage `check` corresponds
to the job/journal phase `checking`. Check, commit, rollback, report and unknown
or null stages do not qualify for the pre-health exception. Classification uses
those generated enum members, never alphabetical comparisons.

A pre-health `failed` result has `health: null`, a non-null `error_code`, a valid
`finished_at`, and `current_release_id` equal to the release recorded when the
job was first authorized. Central snapshots that version in the job's existing
`observed_evidence.attempt_release_id` and preserves it when storing subsequent
result evidence. Reissuing an intent never changes the snapshot. It accepts the
failure and marks the job/target failed with `needs_attention=true`. It does not
fabricate passing health checks or infer that the candidate became current.
A failed result at/after `check` still requires health. Preexisting issued jobs
without a recorded baseline fail closed for this exception; a new authorized
attempt/batch supplies that evidence.

The recorded harness pre-check failure is also exercised against this rule:
the target and job both retain the failed state, original error code and
`needs_attention`. Harness rollback checks must come from the restored release's
attestation; failed candidate checks cannot stand in for them. Corrected health
evidence uses a new result ID, since a rejected result's original payload and
outcome remain immutable.

A `rolled_back` result names the restored release in `current_release_id`, the
candidate in `previous_release_id`, the candidate failure in `error_code`, and
the restored release's checks in `health`. Central stores these fields unchanged
in the immutable result payload. The restored release must be serving, not
revoked, verified, trusted, and schema-readable. An empty manifest rollback
allowlist imposes no additional release restriction; a nonempty list must
contain the restored release (D-034e). Approval to target a new install is not
an additional rollback-serving requirement. Optional `candidate_health` carries
the failed candidate's checks; it never substitutes for restored `health` and
is not required for accepting a rollback. Both profiles remain stored exactly
as received in the immutable result payload.

TEST ONLY JSON vectors live in
[pre-health failure](../../internal/fleetupdate/testdata/d062-pre-health-failure.json),
[restored rollback](../../internal/fleetupdate/testdata/d062-rollback.json), and
[replayed rejection](../../internal/fleetupdate/testdata/d062-rejected-result.json).
Tests bind their identities, authorization timestamps and hashes to actual jobs.
The rejected vector describes an otherwise well-formed rollback to a release
that is not serving; re-enabling that release cannot change the old rejection.

| HTTP status | Result-submission meaning | Retry semantics |
|---|---|---|
| 202 | Persisted accepted progress/result; exact replay returns the original ACK and received_at | Retire that result from the node outbox after durable receipt |
| 400 | Malformed request | Permanent rejection; journal the precise API code |
| 401 | Authentication is unavailable/invalid | Permanent for this submission; preserve diagnostic evidence |
| 403 | Identity revoked or incarnation mismatch | Permanent rejection; no work under that identity |
| 404 | Job/node is unknown | Permanent rejection; investigate identity/restore state |
| 422 | Invalid evidence, hash, health, schema or serving policy | Permanent rejection; preserve the precise API code |
| 409 | Generation/ordering/authorization or result-ID content conflict | retryable=true; reconcile authorized work, and preserve immutable result IDs/payloads |
| 429 | Rate/concurrency limit | retryable=true and Retry-After; back off before replay |
| 5xx | Persistence/dependency failure; no durable acceptance promised | Retain the original outbox entry; follow code/backoff guidance |

For authenticated, job-bound input, `update_results` stores either the original
ACK or a safe rejection object in its existing `ack` JSON column. A rejection
never updates accepted job evidence or target counts. The domain runs projection
work under a savepoint: rejected projection work rolls back, while the original
code/message/retryable/details can commit as a receipt. Replaying the same result
ID/hash returns that original outcome; different content returns ACK_CONFLICT.
A fresh service instance has the same behavior. Request IDs remain per-request
tracing metadata rather than part of the cached rejection.

Schema/hash/identity checks before a trustworthy job binding remain fail-closed
validation, not persisted receipts. A nonexistent job cannot be referenced by
this inbox's FK. Durable replay of such early rejections across a restore or
contract change requires an unbound rejection receipt owned by the HTTP/core
persistence layer; no schema is weakened here.

`SubmitResult` commits domain-owned rejection receipts before returning their
API error. `SubmitResultTx` writes in the supplied transaction: its caller must
commit a recorded rejection outcome separately from fatal transaction failures.
The current generic HTTP command wrapper rolls back every returned error. Core
must add that distinction before HTTP rejection replay has the same durable
semantics as the standalone domain; it must also replay stored errors as errors,
never deserialize them into a success ACK. Authentication and rate limiting
remain core-owned. Current rate-limit middleware already emits Retry-After.

## Result and observation authority (r9 / D-062)

Accepted results own serving versions and recovery for their authorized job,
attempt and generation (§13.8). Heartbeats after acceptance can confirm those
values, but cannot change current/last-good or clear recovery. A mismatch in
either release keeps `unconfirmed=true` by withholding that job's observation
timestamp; a later agreeing report can confirm it. A null `update.job_id` leaves
the high-water and versions unchanged, just like a null generation. Proofs:
`TestReviewR9_D062_ObservationGenerationAndServingRelease`,
`TestReviewR9_D062_JoblessAndInvalidVocabulary`, and
`TestReviewR9_D062_RecoveryAndRetryAttemptAuthority`.

Before that incarnation's first accepted result, a heartbeat may seed current
only with the **current job's target** or a release already recorded in the
node's current/last-good columns (§13.2, §13.8; r11). A superseded target alone
is insufficient. Last-good is preserved until an accepted result establishes
it; repeatedly reporting a seeded target cannot promote it to last-good.
Unexpected releases are ignored for the version columns and withhold the job's
observation timestamp, leaving `unconfirmed=true`. Recovery uses the recorded
last-good, never an unexpected heartbeat claim. Each distinct unexpected
release/job pair gets one `update_observation.unexpected_release` audit with
node actor/resource, result `failed`, and only `job_id`/`release_id` evidence.
The existing fleet transaction lock serializes the audit lookup and append;
repeated or concurrent reports add no duplicate audit, while a different release
or job records new evidence. Proofs: the adopted
`TestReviewR11_HeartbeatSeedsAnyReleaseUntilAResult`,
`TestReviewR11_PreResultTargetSeedPreservesLastGood`,
`TestReviewR11_UnexpectedReleaseAuditBound` (D-029 read-back), and
`TestReviewR11_SupersededTargetCannotSeed`.

The wire path additionally requires a validated job and its exact generation;
it cannot seed from a jobless report. Once a result is accepted, seeding closes
even for later jobs in that incarnation. A current terminal result can correct
an earlier same-generation seed and establish last-good; that write returns
`applied_to_current_observation=true` even when current was already seeded.
The trusted sequence adapter retains its separate restore high-water behavior,
without moving versions after acceptance.
`TestReviewR9_D062_PreResultSeedAndTerminalAuthority` and
`TestUPD02_ResultBindingExpiryAndRestore` cover these bounds. No schema
relaxation or new projection column is needed.

A conflicting result records one row and one audit per
`(result_id, conflicting_hash)` in the same transaction. Replaying that hash
still returns `ACK_CONFLICT` with the original result untouched, and adds neither
row nor audit. Different hashes remain distinct evidence. This is the bound in
`TestReviewR9_D062_ResultIdentityReplayAndReceiptBounds`; its four distinct
hashes plus five repeated submissions produce four rows and four audits.

Every `failed` result requires an `error_code`, including `check`, later stages
and a null stage. Missing causes return permanent 422 `VALIDATION_FAILED` through
the existing bounded rejection receipt; replay returns the same rejection and
cannot change the job or version. `TestReviewR9_D062_EvidenceCompleteness` proves
the check-stage case; `TestReviewR9_D062_FailedRequiresErrorAtEveryStage` covers
every generated stage plus null, including durable rejection replay.

The domain validates any non-null observation `update.state` and `update.stage`
with the generated contract enum validators before projecting the receipt.
Unknown values return `VALIDATION_FAILED` and the caller's observation
transaction rolls back, including confirmation, even if the generation is null.
`TestReviewR9_D062_ObservationGenerationAndServingRelease` and
`TestReviewR9_D062_JoblessAndInvalidVocabulary` test the domain path independently
of the HTTP validator, including an invalid stage with a valid state.

## Target result evidence (contract revision 7)

The admin batch GET projects three fields onto each `UpdateTarget` from its
latest attempt's latest **current accepted receipt**. The stored ACK must have
`accepted=true` and `accepted_as_history=false`. Rejected receipts and later
history, including older generations, cannot replace or erase these fields.
`applied_to_current_observation=false` does not exclude a current result: a
same-generation heartbeat may already have supplied the observed version.

| Field | Source and null behavior |
|---|---|
| `previous_release_id` | The receipt's previous release; for rollback this is the candidate that failed |
| `health` | The receipt's checks, including the restored release's checks on rollback; null stays null |
| `candidate_health` | The receipt's candidate checks only for `rolled_back`; explicit null on every other state, even if its payload contains checks |

All three keys are serialized with explicit null when there is no current
receipt, including unbound enrolling targets and newly retried attempts. An
`ARTIFACT_INVALID` failure before checking retains `health: null`; central never
manufactures passing checks. The fields describe the accepted receipt, not a
new health evaluation or a heartbeat-derived profile.

The page query selects the receipt in one lateral join, scoped to the latest
job, without per-target round trips. It filters current acceptance before
ordering receipts by central receipt time. This also preserves earlier current
progress evidence when an older applying job later finishes its ledger as
history: that ledger's `observed_evidence` pointer alone is not sufficient.
Retries never borrow a previous attempt's evidence, and cursor paging retains
the fixed snapshot counts (§13.8, §13.12).

`TestUPD05_TargetResultEvidenceAdminHTTP` exercises the real admin GET with the
[AT26 producer capture](../../internal/fleetupdate/testdata/at26-direct-rollback.json),
success, a pre-check artifact failure and no result. The history, current-receipt
selection, and paging/rejection/retry tests in `internal/fleetupdate` cover the
receipt fences using real domain commands and isolated databases (D-050).

## Null update generations (D-062)

A heartbeat's null `update.generation` means no update state, regardless of any
other update metadata. A null `update.job_id` also cannot attest update state.
Neither claims generation zero nor confirms, fails or
demotes a target. It records the receipt time while preserving current/last-good
versions and the monotonic generation high-water mark. A subsequent heartbeat
with the authorized job's generation can confirm that target; a larger generation
cannot confirm an older target either.

`VersionView.ObservedGeneration` is nullable. Node view, `GetBatch` and `Evidence`
share a predicate requiring a non-null generation exactly equal to the job's
and a matching incarnation and sufficiently recent receipt. Only the report's
exact job can gain a matching timestamp, so an older retry attempt at the same
generation cannot confirm the current attempt. The job's observation
timestamp must also match that receipt; reports below a job's generation never
advance its observation evidence. A null receipt after
local success therefore leaves the target succeeded but unconfirmed; it cannot
change its denominator or result bucket. A later null receipt also removes
confirmation without undoing the successful result.

The existing numeric cache is NOT NULL and remains an allocation/ordering
high-water mark. A generationless receipt clears the nullable
observation-incarnation marker, so the projection stays null even after raw
heartbeat retention. A later generation-bearing observation restores that
identity only without lowering the high-water mark. This neither coerces null to
zero nor changes a base schema constraint. `ObserveReportTx` requires the
immutable receipt in the same transaction. Direct `Observe`/`ObserveTx` callers
provide an explicit generation; they are not the nullable wire adapter. The
current contract's update block has generation/job_id/stage/state; actual release
is still the top-level current_release_id until a contract revision adds the
nested member. No undeclared observation field is accepted here.

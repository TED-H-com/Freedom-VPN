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
   validation and reject duplicate JSON members. The hash is SHA-256 of the JCS
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
   ineligible. The current implementation makes one fetch attempt per worker
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
| Same result ID, different canonical payload | Commit conflict hash and audit, return 409 `ACK_CONFLICT`; preserve original ACK |
| Terminal result lacks actual version, manifest digest, health or finish time | Reject; no success counted |
| Succeeded/already_current without target version, all health checks, state schema or durable usage checkpoint | Reject; no success counted |
| Rolled back to unsupported/revoked release | `MINIMUM_VERSION_VIOLATION`, persist recovery attention; forwarding stays forbidden |
| Rolled back without compatible, approved, trusted manifest and healthy checks | Reject; never count rollback as target success |
| Older applying/checking job finishes | Finish that job's ledger, ACK as history, leave newest pointer/generation intact |
| Superseded/terminal job reports late, progress goes backwards, already_current arrives after download began, or rollback arrives before applying | Append history; preserve terminal ledger/current pointer |
| Current terminal result, no heartbeat since authorization | May update observed release if the stored observed generation is not newer |
| Heartbeat already arrived after authorization | Save result and ACK; await a new heartbeat for the current-version observation |
| Report claims forwarding while disabled, disruptive, unsupported or recovery-required | `NODE_NOT_SERVICEABLE`; never rewrite publication intent |

All accepted results, their ACK, generation-conditional observation, job evidence,
batch projection and audit commit before an ACK is returned. The result digest
available in the existing contract is `manifest_sha256`; there is no separate
artifact-digest field in `UpdateResult`. Actual artifact verification belongs to
the updater, and its test evidence must accompany release qualification.

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
Old sequences/generations cannot overwrite newer actual versions. A revoked
reported last-good flags recovery. Missing observations stay unconfirmed.

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
  last_observed_at. `UpdateTarget` already has both timestamps; the first two
  booleans need a contract-owner addition before HTTP/UI exposure.
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

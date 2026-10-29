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
   returns `STATE_CONFLICT`. Product version is display-only and may repeat.
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
update generations and attempt 1 jobs; changes the fleet default; and writes an
operation, audit, outbox event and progress projection. Any failure rolls back
all of these. A stale generation returns 409 `STATE_CONFLICT`.

The snapshot includes offline, unpublished, disabled and enrolling nodes. Missing
incarnations remain null until first authenticated intent retrieval binds them
once. The snapshot's original incarnation is never changed. `update_targets`
retain the database's sealed-transaction membership and immutable count guards.

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
also get independent jobs (`batch_id=null`, `target_id=null`) from the fleet
default and never enlarge an existing batch.

The lifecycle trigger cancels unfinished jobs immediately in the node owner's
revocation/retirement transaction. Snapshot identities and accepted result history
remain. Revoked identities cannot obtain more work or submit control results.
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
| Superseded/terminal job reports late, or progress goes backwards | Append history; preserve terminal ledger/current pointer |
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

Each target contributes its latest attempt exactly once. The HTTP contract folds
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

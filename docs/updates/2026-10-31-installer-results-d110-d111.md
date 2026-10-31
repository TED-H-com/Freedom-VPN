# Installer results: D-110 and D-111 (2026-10-31)

This follow-up closes r16 M-2/L-1 and the central acceptance side of updates
follow-up 9 F9-04. It implements UPD02/UPD05 and the receipt, generation and
rollback rules in Project Spec §13.8, §13.9 and §13.10.3. The endpoint remains
`POST /api/v1/agent/update-results`, with `source: installer`.

## D-110: one receipt identity per incarnation and release

`internal/installerjob.ID(releaseID, manifestSHA256, incarnationID)` uses UUIDv5
(Go's `uuid.NewSHA1`) in two steps. UUIDs use canonical lowercase text; the
SHA-256 digest uses lowercase hexadecimal:

```text
claim_namespace = UUIDv5(NameSpaceOID,
    "fvpn-installer-apply|" + release_id + "|" + manifest_sha256)
job_id = UUIDv5(claim_namespace, manifest_sha256 + "|" + incarnation_id)
NameSpaceOID = 6ba7b812-9dad-11d1-80b4-00c04fd430c8
```

Central first verifies release eligibility and the manifest digest, then derives
the job ID using that digest and the authenticated current incarnation. A new
receipt with a different job ID returns `409 STATE_CONFLICT` with
`details.reason = "job_id_mismatch"`; no new error code is introduced. Rejected
IDs create no installer receipt, acceptance audit, or node version increment.
Immutable release content and the existing `(node_id, job_id)` primary key bound
new receipts to one per incarnation and release. Existing historical receipts
are not deleted or rewritten.

Replay precedes the new derivation check: the same job and payload hash return
the stored ACK, including receipts accepted before D-110 or a later release
revocation. Different evidence for an accepted job remains `409 ACK_CONFLICT`.
The authenticated node/incarnation checks still precede replay. Central job IDs
cannot be used for installer evidence. The existing #277 HTTP keyed replay is
reused without changes.

The node resource version advances once if either release pointer or the node's
visible update summary changes. The writer compares the same projection used by
node list/detail: a current central job wins; otherwise the newest accepted
receipt of the current incarnation wins. A hidden installer receipt with no
version change does not advance the ETag. Exact replay does not advance it.

## Compatibility correction for updater follow-up 15

On the starting main revision, `node/updater/pending.go` hashes the **entire
canonical pending record**, including its local `request_id`, and uses that hash
in the final UUID name. It does not use the manifest digest there. The claim
namespace above matches main, but the final D-110 name is a deliberate change:
central never receives the pending-record bytes or their hash.

`TestD110_MainUpdaterClaimNamespaceCompatibility` invokes main's actual
`ApplyPending`, reads its durable result, and reproduces the legacy ID. The test
also demonstrates that this legacy ID differs from D-110's manifest-based ID,
and accepts the coordinated follow-up 15 switch to the new ID.
`TestD110_StableManifestIdentity` pins the D-110 formula and input binding.

The updater lane must switch to `installerjob.ID` in follow-up 15 before new
installer receipts can be accepted by this server. It must keep local pending
record hashing for staging integrity, but must not use staging paths, random
request IDs, or pending-record hashes to create new central receipt identities.
Retries must retain the original sealed result bytes. A previously accepted
legacy receipt can still replay; a legacy receipt not yet accepted will receive
`job_id_mismatch`. Coordinate that transition with this lane; do not claim unchanged live compatibility with main's old updater.

## D-111: an eligible seed can establish a missing baseline

When central `current_release_id` is null, a pre-health installer failure may
report a distinct eligible seeded release as current. Eligibility means the
release manifest is trusted and the catalog marks the release verified, approved,
supported and nonrevoked. Central validates the normal pre-health shape (null
health, an error code, completion time and unchanged seeded release), stores the
failure evidence, and sets both `current_release_id` and `last_good_release_id`
to that baseline. It does not promote the failed candidate, invent a health pass,
or grant member forwarding.

A successful result still needs candidate version, healthy checks, durable usage
checkpoint and compatible schema. A rollback still needs its failure code,
candidate in `previous_release_id`, compatible allowed rollback target and healthy
checks. Both establish the reported eligible release when the baseline is null;
rollback also corrects both pointers when central still records the departed
candidate (r16 M-2). A known baseline is not replaced by a pre-health failure.

All pointer writes obey the central observed generation high-water. Older
installer evidence is accepted as history without moving either pointer, and
installer generations never advance the central high-water or alter central
jobs, batch results, confirmations, publication intent or operations. Seeded
`last_good_release_id` records the D-111 baseline convention; null health remains
null and is not evidence that health checks passed.

## Regression and acceptance checks

- Adopted r16 self-promotion, rollback and cardinality proofs. The rollback proof
  retains its version assertions and the recorded candidate, but re-enrolls
  before the independent rollback receipt: D-110 forbids a second random job
  ID for the same incarnation and release.
  The redirect/byte-verification proof is under `node/updater`, owned by that lane.
- D-111 tests exercise failed pre-health, rolled-back and succeeded results for
  amd64/arm64 identities, ineligible baselines and candidate promotion rejection,
  immutable failure evidence, exact domain replay, and HTTP keyed/new-key replay.
- D-110 tests exercise wrong/random/foreign-incarnation IDs, one retained receipt
  and audit, stable ETags, replacement incarnations, central summary precedence,
  equal/newer generation fences and null-baseline fencing.
- `TestD110_InstallerMigrationDown190Up` executes Goose `DownTo(190)` then `Up()`
  against a dbtest database and checks acceptance, replay, conflict, projection,
  audit and immutable update/delete/truncate guards. Migration 00191 already
  removes triggers, index, then table; no schema change is needed.
- The independent nodes `TestSECOPS19_D077` round-trip through 188 remains a gate.

The updates AT lane rechecks F9-04 with the coordinated updater: install → verify
and seed baseline → fail staged upgrade before health → submit and keyed replay.
Both requests must return the stored 202 ACK, the baseline must be current and
last-good, and failure health stays null. Recheck succeeded and rolled-back
outcomes, and prove repeated random job IDs leave one row/audit with a stable
ETag. These are central integration proofs; this follow-up does not claim a new
live device or native arm64 run.

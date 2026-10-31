# Usage store

`node/internal/usagestore` is the node's append-only usage ledger (§10.5, §10.8, §13.7, §17.3). It is a SQLite database in WAL mode with `synchronous=FULL`. The default path is `/var/lib/vpn-node/state/usage.db` (`usagestore.DefaultPath`). Tests and the agent pass another path.

One database belongs to one node, with one current incarnation and retained retired streams. Opening it as another node always returns `ErrIdentity`. An incarnation change also returns `ErrIdentity` unless the caller explicitly supplies `Options.RetirePreviousAt` after verified re-enrollment (D-105). The file is retained.

## Transaction

`Commit` writes the delta event, its `event_seq`, and the counter checkpoint in one transaction (§10.5). A crash before `COMMIT` leaves neither the event nor the checkpoint. A crash after `COMMIT` leaves both. The checkpoint never moves ahead of the event: a byte increase without an event returns `ErrCheckpointWithoutEvent`.

`event_seq` starts at 1 for the incarnation and is never reused or reset when the meter epoch changes (§10.6). The durable range stops at `2^63-2` because SQLite integers are signed. Counter bytes, packets and binding revisions are decimal text, so values through `2^64-1` survive.

The stored payload is the RFC 8785 event, including `payload_sha256`. The hash is `canonjson` over the event with `payload_sha256` removed, matching `contracts/vectors/usage-event-payload-sha256.json`. Resends return those bytes unchanged.

## Tables

| Table | Role |
|---|---|
| `meta` | Identity, `next_seq`, open meter epoch, boot id, ledger epoch, contiguous ACK, `blocked_seq`, gate, `purged_through` |
| `epochs` | Meter epochs, including closed ones kept for audit |
| `counters` | Absolute counter checkpoint per meter epoch |
| `events` | Outbox and replay archive. `state` is `outbox` or `archived` |
| `gaps` | Gap reason for `quality=gap` |
| `ledger_epochs` | Verified recovery checkpoints. Superseded rows stay |
| `replay_unavailable` | Inclusive ranges this node can no longer replay |
| `upload_batches` | Exact request body and its `Idempotency-Key` |
| `retired_streams` | Original incarnation, locally observed retirement time, ACK, last sequence, verified ledger epoch, blocker, purge watermark and whole-origin final code |
| `retired_events` | Original payloads and hashes, archive state and final per-event receipts, keyed by incarnation and sequence |
| `retired_replay_unavailable` | Original stream's retained recovery gap ranges |
| `retired_history` | Original meter epochs, gaps and verified checkpoint evidence retained at rotation |
| `handoff_rejections` | One local audit entry per original incarnation/sequence; sequence zero identifies a whole-origin refusal |

## ACK

`ApplyACK` moves a contiguous prefix into the archive (§10.6). A sequence counts only when it is already archived, declared replay-unavailable, or marked `accepted` or `duplicate` in this result and still stored. `blocked_seq` and a `conflict` or `rejected` result stop the prefix. The ACK never goes backwards inside one ledger epoch (`ErrAckRegress`). A batch that names a different ledger epoch returns `ErrLedgerEpoch` and writes nothing; the caller fetches a signed checkpoint.

Archived payloads stay for at least 72 hours after the local ACK (`DefaultArchiveRetain`, §10.8, §16.4). `PurgeArchive` deletes only a prefix of archived rows. Unacknowledged rows are never purged. `purged_through` remembers the highest purged sequence so a later checkpoint can call that hole `retention_expired`.

## Caps

`EvaluateGate` implements §10.8 across the current and retired streams, including retained final rejections. `OutboxStatus` and the CLI snapshot describe only the current stream. `Health.ForwardingAllowed` is false when any cap is hot:

| Reason | Stop | Resume |
|---|---|---|
| `outbox_age` | Oldest unacked event age ≥ 72h | Age under 80% of the cap |
| `outbox_bytes` | Unacked canonical payload ≥ 256 decimal MB | Under 80% |
| `archive_bytes` | Replay archive ≥ its own budget (default 1 decimal GB; the spec requires a separate budget and does not set the number) | Under 80% |
| `disk_space` | Free space under `MinFreeBytes` | Free space ≥ floor + floor/4 |
| `persist_failed` | `NotePersistFailure` until `NotePersistSuccess` | The flag clears |

`MinFreeBytes` defaults to 0. The production floor is 待實測 and this package does not invent one. When it is set, the footprint is every file in the database directory whose name is the database file or starts with it (main, WAL, shared-memory index). A large archive does not consume the unacked budget.

The stopped flag is stored in `meta`, so a process restart keeps the gate closed until the resume line.

## Outbox status

`OutboxStatus` fills `api.UsageOutboxStatus` for an observation (§10.5, §10.6, §10.8). It is a read of the persisted stream. It does not call `EvaluateGate`, it does not stat the disk, and it does not update `gate_stopped` or `persistFailed`. Two calls with no commit and no ACK between them return the same value. The database file, its WAL, and its shared-memory index are unchanged.

`EvaluateGate` is the call that persists the forwarding gate. The agent run loop reaches it through `meter.Health`. That is where a hot cap sets `gate_stopped` and where a stopped gate resumes once every tripped meter is under 80%. An observation does not take that step, and neither does the uploader's size check. A backlog that is already over a cap stays `gate_stopped = 0` until `EvaluateGate` runs. A gate that is already stopped stays stopped until `EvaluateGate` sees the resume line.

`ledger_epoch` on that status is the central generation `ApplyCheckpoint` adopted. It is null until that verified checkpoint, including when `central_acked_through` is already greater than zero. A batch ACK advances the contiguous watermark and does not store the epoch named in the result (§10.5 step 5, §10.6, §17.3). Null means this node has not accepted a verified ledger generation. The watermark is still the highest contiguous ACK; it is not yet bound to a generation. `fvpn-agent usage snapshot` prints the same null. `UsageBatchRequest.ledger_epoch` already says the field is null before the first checkpoint.

## Checkpoint

`ApplyCheckpoint` verifies the envelope with `envelope.PurposeRecovery` against the node trust store (`NewNodeTrustStore` / `ParseNodeTrustStoreJSON`). Verification fails closed (`ErrNoTrust`, `ErrBadCheckpoint`). A node bundle cannot carry the central-only `ledger-epoch` purpose.

Inside the current ledger epoch the ACK never falls. A lower watermark is accepted only for a ledger epoch this node has not accepted before. Events above the new watermark return to the outbox with their original payloads and sequence numbers. The previous `ledger_epochs` row stays, with `superseded_at` set. Replaying a historical epoch returns `ErrEpochReplay`.

`next_seq` becomes at least `max_seen_seq+1`, so a hole the node never had is not reused. Inclusive missing ranges the node cannot fill are stored as `replay_unavailable` (`local_data_lost`, or `retention_expired` at or below `purged_through`).

## Fault injection

`Options.Hook` may return `ErrInjectedCrash` at `commit.insert`, `commit.checkpoint`, `commit.before_commit`, `commit.after_commit`, `ack.before_commit` and `ack.after_commit`. A hook must not call back into the store. Steps before commit roll the transaction back. `commit.after_commit` and `ack.after_commit` run after the commit is durable.

## Reading the local ledger

`fvpn-agent usage snapshot` prints the ledger without writing it. It is safe while `fvpn-agent run` is serving. It does not take the node update lock (`/var/lib/vpn-node/update.lock`). It opens `state_dir/usage.db` with SQLite `mode=ro` and `query_only`, waits at most one second (`usagestore.SnapshotBusyTimeout`) if the file is busy, and reads the schema version, the cursor, the meter epochs and every unacknowledged payload in one deferred read transaction. It does not create the file, migrate it, checkpoint it, or call the upload or ACK paths.

The ledger stores member and device UUIDs, period ids, decimal byte counters and the canonical event. It does not store tokens, private keys, or raw member browsing data (no DNS names, URLs or packet contents). The command does not print those either, and a failure is one fixed sentence with no stack trace and no file path.

```text
fvpn-agent usage snapshot -config /etc/vpn-node/agent-run.json
fvpn-agent usage snapshot -config /etc/vpn-node/agent-run.json -json
```

The text form is an operator table. `-json` is one compact JSON object and a trailing newline. Two snapshots with no commit and no ACK between them are byte-identical, in either form.

### What the numbers mean

| Field | Meaning |
|---|---|
| `schema_version` | Snapshot format version, still 1. The reader accepts SQLite schema 1 and 2 without migrating either. |
| `node_id`, `node_incarnation_id` | The current stream in this file (§10.6). |
| `ledger_epoch` | Central ledger generation the watermark belongs to (§17.3). JSON `null` (text `none`) until a verified checkpoint is applied. A batch ACK does not set it. |
| `meter_epochs` | Counter generations in open order. `current` is the open epoch. `closed` epochs stay so a reinstall can show the id did not reset (§10.7). |
| `last_event_seq` | Highest sequence this incarnation has assigned (`next_seq - 1`), as a decimal string. It does not reset on restart, counter rebuild or meter-epoch change. A purged archive row can leave this above the highest row still on disk. |
| `acked_through` | Contiguous central watermark, decimal. Unacknowledged events have a greater sequence. The same ledger epoch never lowers it (§10.6, §10.8). |
| `unacked_count` | Outbox rows. |
| `unacked_payload_bytes` | Sum of those rows' stored canonical payload lengths, decimal. This is the §10.8 byte total, not a file size. |
| `unacked_events` | One object per outbox row, in sequence order. |

Each unacknowledged event has `event_seq`, `period_ids` (first-seen order), `records` and `payload_sha256`. Each record copies `member_id`, `device_id`, `binding_revision`, `period_id`, `upload_bytes` and `download_bytes` from the stored payload. `payload_sha256` is the immutable hash: SHA-256 of the canonical payload with that field removed, checked against the stored column and against the exact bytes the uploader resends. A gap with no device records has an empty `records` array. In the text table that row uses `-` for the record columns.

An installer re-run that keeps this file and incarnation should show the same hashes, the same sequences, the same meter-epoch ids and the same watermark after `fvpn-agent run` starts again. The watermark in that snapshot is the one the store had persisted, not a value reconstructed from memory (§10.6, §10.8).

### Consistency

`ApplyACK` updates the outbox rows and `acked_through` in one SQLite transaction, and commits that transaction before it returns (§10.6). The snapshot's single read transaction therefore shows either the pre-ACK watermark and the pre-ACK outbox, or the post-ACK watermark and the post-ACK outbox. It does not show a new watermark with the old outbox, or the reverse. WAL mode lets this read proceed while `fvpn-agent run` holds its own writer. The snapshot process does not share that process's mutex, and it does not hold a write lock across the read.

### Exit codes

| Exit | Meaning | Stderr |
|---|---|---|
| 0 | Snapshot on stdout | empty |
| 1 | The snapshot was read but stdout could not be written | `fvpn-agent: usage snapshot failed` |
| 2 | Bad invocation, or the config file is not usable | `usage: ...` or `fvpn-agent: configuration error` |
| 3 | The usage database file is not there. Nothing is created | `fvpn-agent: usage store is absent` |
| 4 | SQLite stayed busy for the whole one-second wait | `fvpn-agent: usage store is locked` |
| 5 | The file is unreadable, not this schema, or fails the hash check | `fvpn-agent: usage store is corrupt` |
| 64 | `host_reserve_bps` is missing or zero, the same rule as `run` | the configuration sentence used by `run` |

### Lab contract

`UsageLedgerSnapshot` in the lab is the stdout of:

```text
fvpn-agent usage snapshot -config <agent-run.json> -json
```

The devstack harness diffs that byte string. It must not edit or delete outbox rows to build the view. Field names and types:

| JSON field | Type |
|---|---|
| `schema_version` | number |
| `node_id` | string, canonical UUID |
| `node_incarnation_id` | string, canonical UUID |
| `ledger_epoch` | string UUID, or `null` |
| `meter_epochs` | array of `{meter_epoch_id, state}` where `state` is `current` or `closed` |
| `last_event_seq` | decimal string |
| `acked_through` | decimal string |
| `unacked_count` | number |
| `unacked_payload_bytes` | decimal string |
| `unacked_events` | array of `{event_seq, period_ids, records, payload_sha256}` |
| `records[]` | `{member_id, device_id, binding_revision, period_id, upload_bytes, download_bytes}`, all strings |

Decimal strings match `^(0|[1-9][0-9]*)$`. `payload_sha256` is 64 lowercase hex digits. The object key order is the order in this table.


## Retired-stream hand-off (D-105, ENR-05)

`Open(path, nodeID, newIncarnationID, Options{RetirePreviousAt: observedAt})`
atomically moves the previous stream into the retired tables and starts the new
incarnation at sequence 1, ACK 0, with no meter baseline or verified ledger epoch.
The previous meter and uploader must be stopped and the caller must hold the
node lock. `observedAt` is the local observation of retirement/re-enrollment;
central uses its own `revoked_at`, so this timestamp never authorizes a window.
A repeated open of the new incarnation is a no-op. A foreign node or reuse of an
already retired incarnation fails closed. SQLite schema 1 migrates to 2 in place.

The old incarnation, highest ACK, meter epoch IDs, canonical payload bytes and
hashes remain available without its certificate. `RetiredStreams`,
`RetiredUnacked` and `HandoffRejections` expose the durable evidence. The new
meter never inherits the retired counter baseline. Rotation and ACK writes have
before/after-commit crash hooks (`retire.before_commit`, `retire.after_commit`,
`ack.before_commit`, `ack.after_commit`). Reopening after either crash neither
loses the tail nor creates a second retired stream.

`ApplyRetiredACK` changes only the named original stream. A final per-event
rejection and its audit commit together. The audit stores origin, current
presenter, sequence, allowlisted error code and local timestamp; it stores no
remote error message, certificate or event body. Whole-origin refusals are also
recorded once. Rejected payloads remain unacknowledged; accepted payloads behind
a rejected sequence remain stored without falsely advancing the contiguous ACK.
They continue to count against the unacknowledged capacity/age caps. Purging
removes only retained archived prefixes after the normal retention window and
keeps each original stream's ACK and purge watermark.

Production adoption is active as of 2026-10-31: agent-core follow-up 25 added
`RetirePreviousAt: e.clock.Wall()` to `Engine.ensureUsage` for the verified current
identity. See its [D-118 activation evidence](usage-upload.md#d-118-activation).
The normal final meter barrier and stopped old loops precede adoption; reopening
the unchanged incarnation does not retire it again.

Retained recovery gaps follow the v1 [reported, never re-fetched rule](usage-upload.md#retained-recovery-gaps).
`retired_replay_unavailable` holds pending origin ranges until that origin's
contiguous ACK covers them; `retired_history` retains the retirement-time copy
afterward. Applying a current-incarnation checkpoint never modifies either.
An origin with only gaps and no pending event stays local because revision 14
requires a non-empty events array; no payload or ACK is fabricated.

The enrollment AT lane can capture `UsageLedgerSnapshot` before re-enrollment
and sum its held records by original incarnation, member, device, binding
revision, period and direction using decimal integers. After hand-off, central's
attributed total must equal its committed **before + held** total exactly once,
including retries and values above 2^53. Compare original sequences, hashes and
historical attribution against the captured payloads. The post-enrollment CLI
snapshot describes the new stream; its zero/new ACK is not the retired ACK.
Use `RetiredStreams` or the original-stream central result for the latter. A
window rejection is an audited unresolved original event, not accepted held
bytes. These are package/HTTP assertions; real ENR-05 reinstall/network evidence
remains with enrollment follow-up 8 and usage AT follow-up 5, as do real arm64
runs. Package tests and cross-compilation do not replace that evidence.

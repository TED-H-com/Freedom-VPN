# Usage store

`node/internal/usagestore` is the node's append-only usage ledger (§10.5, §10.8, §13.7, §17.3). It is a SQLite database in WAL mode with `synchronous=FULL`. The default path is `/var/lib/vpn-node/state/usage.db` (`usagestore.DefaultPath`). Tests and the agent pass another path.

One database belongs to one node incarnation. Opening it as a different node or incarnation returns `ErrIdentity`. The stream is not reset and the file is not deleted.

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

## ACK

`ApplyACK` moves a contiguous prefix into the archive (§10.6). A sequence counts only when it is already archived, declared replay-unavailable, or marked `accepted` or `duplicate` in this result and still stored. `blocked_seq` and a `conflict` or `rejected` result stop the prefix. The ACK never goes backwards inside one ledger epoch (`ErrAckRegress`). A batch that names a different ledger epoch returns `ErrLedgerEpoch` and writes nothing; the caller fetches a signed checkpoint.

Archived payloads stay for at least 72 hours after the local ACK (`DefaultArchiveRetain`, §10.8, §16.4). `PurgeArchive` deletes only a prefix of archived rows. Unacknowledged rows are never purged. `purged_through` remembers the highest purged sequence so a later checkpoint can call that hole `retention_expired`.

## Caps

`EvaluateGate` implements §10.8. `Health.ForwardingAllowed` is false when any cap is hot:

| Reason | Stop | Resume |
|---|---|---|
| `outbox_age` | Oldest unacked event age ≥ 72h | Age under 80% of the cap |
| `outbox_bytes` | Unacked canonical payload ≥ 256 decimal MB | Under 80% |
| `archive_bytes` | Replay archive ≥ its own budget (default 1 decimal GB; the spec requires a separate budget and does not set the number) | Under 80% |
| `disk_space` | Free space under `MinFreeBytes` | Free space ≥ floor + floor/4 |
| `persist_failed` | `NotePersistFailure` until `NotePersistSuccess` | The flag clears |

`MinFreeBytes` defaults to 0. The production floor is 待實測 and this package does not invent one. When it is set, the footprint is every file in the database directory whose name is the database file or starts with it (main, WAL, shared-memory index). A large archive does not consume the unacked budget.

The stopped flag is stored in `meta`, so a process restart keeps the gate closed until the resume line.

## Checkpoint

`ApplyCheckpoint` verifies the envelope with `envelope.PurposeRecovery` against the node trust store (`NewNodeTrustStore` / `ParseNodeTrustStoreJSON`). Verification fails closed (`ErrNoTrust`, `ErrBadCheckpoint`). A node bundle cannot carry the central-only `ledger-epoch` purpose.

Inside the current ledger epoch the ACK never falls. A lower watermark is accepted only for a ledger epoch this node has not accepted before. Events above the new watermark return to the outbox with their original payloads and sequence numbers. The previous `ledger_epochs` row stays, with `superseded_at` set. Replaying a historical epoch returns `ErrEpochReplay`.

`next_seq` becomes at least `max_seen_seq+1`, so a hole the node never had is not reused. Inclusive missing ranges the node cannot fill are stored as `replay_unavailable` (`local_data_lost`, or `retention_expired` at or below `purged_through`).

## Fault injection

`Options.Hook` may return `ErrInjectedCrash` at `commit.insert`, `commit.checkpoint`, `commit.before_commit`, `commit.after_commit`, `ack.before_commit` and `ack.after_commit`. A hook must not call back into the store. Steps before commit roll the transaction back. `commit.after_commit` and `ack.after_commit` run after the commit is durable.

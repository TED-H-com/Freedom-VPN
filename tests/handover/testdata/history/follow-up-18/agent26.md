# Follow-up 26 to updater follow-up 17 — 2026-10-31

D-125 adds `barrier_id` to the agent's node-local meter checkpoint. There is no
HTTP contract, generated API type or signed-document change. The file remains
`state_dir/meter.checkpoint` unless `meter_checkpoint` selects another path.

```json
{"closed_meter_epoch_id":"019a0000-0000-7000-8000-000000000001","last_durable_event_seq":"17","central_acked_through":"12","new_meter_epoch_id":"","barrier_id":"019a0000-0000-7000-8000-000000000002"}
```

The example IDs and watermarks are TEST ONLY. Publication includes one final
newline, uses mode 0600, fsyncs a temporary file, atomically renames it, then
fsyncs the containing directory. The four existing members keep their order
and bytes when the optional identity is empty. `Store.ClosedCheckpoint` still
knows nothing about identities; only the agent sets `BarrierID` at publication.

## Exact correlation and ordering

The updater persists an opaque identity before blocking and writes it to
`forwarding.blocked` as `<identity>\n`. The agent reads that latch, strips only
one trailing newline and requires `uuid.Validate` to pass. It preserves the
UUID's spelling. **Equality means byte-equal UUID strings with the updater's
persisted identity**, not equivalent parsed UUIDs, case folding, UUID order,
mtime, inode replacement, boot ID, sequence advancement or close generation.
The updater must retain its own before/after latch checks, deadline and boot
checks from follow-up 16, and consume the checkpoint from one file snapshot.

For a new identity, the producer pauses and drains sampling, installs and
verifies the member baseline, refuses serving, takes the final sample and
commits the epoch close in SQLite. It reads the completed close and watermarks
in one store snapshot, rechecks that the latch is unchanged, and publishes only
then (§13.6 steps 7–8). The baseline leaves the control channel available.
A failed withdrawal, sample, close or file write cannot mark that barrier done.
A latch replaced or withdrawn during the close cannot acknowledge the new
identity. A subsequent pass starts the barrier again.

`finalDone` is keyed by identity. A second block after a quiet already-closed
epoch repeats withdrawal and finalization and publishes a fresh snapshot for
the second identity. It can have exactly the same epoch and watermarks: there
is no manufactured event, usage, sequence or epoch. It never just relabels the
previous file. If uploads have advanced ACKs between distinct barriers, a fresh
snapshot can reflect that advancement.

## Withdrawal and restart

When the latch disappears, the agent removes checkpoint evidence and fsyncs its
parent before resuming admission. The serving path also invalidates the file
under the node lock before Apply. A failure to remove it withholds admission.
A valid checkpoint therefore is not a reusable authorization after unblock.

On restart with the same live identity, a completed final-sample epoch stays
closed. The agent repeats verified withdrawal/finalization and atomically
republishes the original checkpoint bytes. Matching the saved identity,
closed epoch and durable sequence allows reuse of its ACK snapshot even if the
uploader has since advanced the live ACK. The next epoch opens from real
counters before a later serving Apply, using the existing PrepareSampling path.
No open epoch or another identity's checkpoint qualifies as that saved snapshot.

With the latch absent, restart removes stale evidence and publishes nothing.
With malformed, unreadable or non-regular latch content, forwarding remains
blocked and no checkpoint is published. A close can still be durable locally;
that is not acknowledgement of a valid updater identity. Agent restart is not
proof of unchanged host boot: the updater must enforce its persisted boot and
original wait deadline itself. Crash before publication can retry from the
completed store close without generating another event.

## Consumer meaning and compatibility

A matching `barrier_id` attests to the agent's verified withdrawal and completed
durable close for that exact block. The closed epoch and two sequence values
are local ledger evidence; `new_meter_epoch_id` is empty. It does not attest to
candidate health, service restart, successful switch, new-epoch creation,
central acceptance beyond the captured ACK, current publication authority or
permission to forward. The updater retains responsibility for its journal,
recovery, bounded wait, stop/switch/health sequence and failure vocabulary
(`CHECKPOINT_FAILED` / `STATE_CONFLICT`).

A legacy four-member checkpoint is **uncorrelated** for updater 17 and later,
even when it is newly written, has a later watermark or follows a quiet block.
Updater 16's strict decoder rejects the five-member form by design, so this
producer remains fail closed until updater 17 consumes it. The unchanged
updater and installer unit suites are required regression gates, not a claim
of successful five-member FileMeter update acceptance.

`TestMTR03_UPD05_DurableFileMeterBarrier` drives the real meter/store, records
an actual 4,096-byte final delta, checks the echoed identity and durable cursor,
and asserts updater 16 rejection. `TestD125_Rule*` covers fresh block, failed
withdrawal, latch replacement during close, quiet second block, invalidation
before admission, restart with and without the latch, and malformed content.

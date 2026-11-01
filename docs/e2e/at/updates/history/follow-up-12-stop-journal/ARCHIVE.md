# Follow-up 12 retained journal readback — 2026-11-01

Read-only extraction from this worktree's retained node and node2 SQLite journals,
before the Follow-up 13 stack reset. Transaction UUIDs match run
`2babf9b6-86a3-489d-a678-0bd447f2f8b0` in the byte-identical follow-up-12 archive.
Only txn_id, seq, stage and at are published; no private state or credentials.
The installer container's journal was not retained after teardown.

These timestamps bound larger stages. They are not direct command durations,
and cannot establish process exit status or exclude a supervisor SIGKILL.
The original archive remains untouched; this is a separately pinned readback.

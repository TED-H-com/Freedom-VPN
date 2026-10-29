// Package usagestore is the node usage ledger (§10.5, §10.8, §13.7).
//
// One SQLite database in WAL mode holds the counter checkpoint, the
// per-incarnation event stream, the unacknowledged outbox, the acknowledged
// replay archive, gap rows, and the ledger-epoch watermark. A sample's delta
// event, its event_seq and the new checkpoint commit in a single transaction.
// The checkpoint never moves without that event.
//
// event_seq starts at 1 for the node incarnation and is never reset when the
// meter epoch changes (§10.5, §13.7). Resends read the stored canonical
// payload bytes; nothing rebuilds an event (§10.5).
//
// The default path is /var/lib/vpn-node/state/usage.db (AGENTS.md §4). Tests
// pass a temporary path.
package usagestore

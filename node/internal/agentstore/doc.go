// Package agentstore is the node agent's durable SQLite state (§3, §13.5,
// §15). It keeps identity references, the accepted desired-state generation,
// the last verified desired-state envelope, lease state, observation outbox
// rows and the enrollment request. The usage meter (usage.db) and the updater
// (update journal) keep their own databases; this package does not create
// their tables.
//
// Writes use WAL and synchronous=FULL. A safety decision is committed before
// the agent acknowledges it. The private key, the enrollment token and the
// NetBird setup key are not stored here.
package agentstore

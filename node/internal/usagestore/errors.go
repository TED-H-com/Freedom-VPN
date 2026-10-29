package usagestore

import "errors"

// Sentinel errors. Callers classify with errors.Is.
var (
	// ErrIdentity means the database was created for a different node or
	// incarnation. Streams are not mixed (§10.6, §13.7).
	ErrIdentity = errors.New("usagestore: database belongs to another node incarnation")
	// ErrCheckpointWithoutEvent means a commit would move counter bytes
	// without an event in the same transaction (§10.5).
	ErrCheckpointWithoutEvent = errors.New("usagestore: checkpoint advanced without an event")
	// ErrCounterRegress means a non-gap commit lowered a counter on the
	// same meter epoch (§10.7).
	ErrCounterRegress = errors.New("usagestore: counter went backwards without a gap event")
	// ErrDeltaMismatch means a complete event does not account for the
	// counter increases (§10.5).
	ErrDeltaMismatch = errors.New("usagestore: event bytes do not match counter deltas")
	// ErrAckRegress means a checkpoint or batch result would lower the
	// contiguous ACK inside the same ledger epoch (§10.8, §17.3).
	ErrAckRegress = errors.New("usagestore: ACK regression in the same ledger epoch")
	// ErrLedgerEpoch means a batch named a different ledger epoch. The
	// caller fetches a signed checkpoint before retrying (§10.8, §17.3).
	ErrLedgerEpoch = errors.New("usagestore: ledger epoch mismatch")
	// ErrEpochReplay means a checkpoint reused a ledger epoch that is no
	// longer current. Only a new verified epoch may lower the watermark.
	ErrEpochReplay = errors.New("usagestore: historical ledger epoch cannot lower the watermark")
	// ErrBadCheckpoint means the signed checkpoint failed verification or
	// does not bind this node incarnation (§17.3).
	ErrBadCheckpoint = errors.New("usagestore: usage checkpoint rejected")
	// ErrInjectedCrash is returned by a test fault hook. A crash before
	// commit rolls the transaction back. A crash after commit leaves the
	// transaction durable.
	ErrInjectedCrash = errors.New("usagestore: injected crash")
	// ErrClosed means the store has been closed.
	ErrClosed = errors.New("usagestore: closed")
	// ErrNoTrust means a checkpoint arrived and no recovery trust store is
	// configured. Verification fails closed (§17.3).
	ErrNoTrust = errors.New("usagestore: no recovery trust store")
)

// Fault steps. A hook that returns an error aborts the surrounding
// operation at that boundary (§10.5 crash consistency, AT16).
const (
	StepInsert          = "commit.insert"
	StepCheckpoint      = "commit.checkpoint"
	StepBeforeCommit    = "commit.before_commit"
	StepAfterCommit     = "commit.after_commit"
	StepACKBeforeCommit = "ack.before_commit"
	StepACKAfterCommit  = "ack.after_commit"
)

// Hook is a test fault injector. Production leaves it nil.
// A hook must not call back into the store.
type Hook func(step string) error

package usagestore

import (
	"time"

	"github.com/google/uuid"
)

// Counter is one absolute named-counter reading stored in the checkpoint
// (§10.5). Bytes and packets are the kernel totals, not deltas.
type Counter struct {
	Name            string
	DeviceID        uuid.UUID
	Direction       string
	Packets         uint64
	Bytes           uint64
	MemberID        uuid.UUID
	BindingRevision uint64
}

// DraftRecord is one device's non-negative delta for an interval (§10.1).
type DraftRecord struct {
	MemberID        uuid.UUID
	DeviceID        uuid.UUID
	BindingRevision uint64
	PeriodID        string
	Upload          uint64
	Download        uint64
}

// Draft is an event before event_seq and payload_sha256 are assigned.
// Quality is complete, gap or boundary_estimated (§10.7, §10.9).
// GapReason is required iff Quality is gap. LastCheckpoint may be nil.
type Draft struct {
	EpochID        uuid.UUID
	Start          time.Time
	End            time.Time
	Quality        string
	GapReason      string
	LastCheckpoint *time.Time
	Records        []DraftRecord
	Detail         string
}

// NewEpoch opens a meter epoch (§10.7). ID is chosen by the caller so a
// crashed attempt does not reuse it.
type NewEpoch struct {
	ID     uuid.UUID
	Reason string
	BootID string
}

// Commit is one durable sample (§10.5). Event, sequence and the checkpoint
// write share one transaction. Leave Event nil only when no counter byte on
// the same epoch changes (a quiet interval, a clean epoch close, or a fresh
// baseline).
type Commit struct {
	Event       *Draft
	Counters    []Counter
	Epoch       uuid.UUID
	CloseEpoch  bool
	CloseReason string
	Open        *NewEpoch
	BootID      string
	SampledAt   time.Time
}

// Head is the durable meter cursor the sampler reads before it plans.
type Head struct {
	OpenEpoch       uuid.UUID
	OpenReason      string
	BootID          string
	LastCloseReason string
	HasHistory      bool
	Counters        []Counter
	LastSample      time.Time
	NextSeq         uint64
	AckedThrough    uint64
	LedgerEpoch     uuid.UUID
	BlockedSeq      *uint64
}

// StoredEvent is one stream row. Payload is the canonical JSON to resend,
// byte for byte (§10.5).
type StoredEvent struct {
	Seq       uint64
	EpochID   uuid.UUID
	Start     time.Time
	End       time.Time
	Quality   string
	Payload   []byte
	Hash      string
	State     string
	CreatedAt time.Time
	AckedAt   time.Time
}

// BatchACK is the contiguous-ACK portion of a usage batch result (§10.6).
// LedgerEpoch is recorded only when it matches the verified epoch. A batch
// result is not a signed checkpoint and cannot change the epoch or lower
// the ACK (§17.3).
type BatchACK struct {
	LedgerEpoch uuid.UUID
	Highest     uint64
	// Blocked is nil when the result has no blocked_seq.
	Blocked *uint64
	Results []EventResult
}

// EventResult is one per-event outcome (§10.6).
type EventResult struct {
	Seq    uint64
	Status string
	Code   string // allowlisted error code for a retired-stream rejection audit
}

// Event status strings, matching the contract.
const (
	StatusAccepted  = "accepted"
	StatusDuplicate = "duplicate"
	StatusConflict  = "conflict"
	StatusRejected  = "rejected"
)

// Gate reasons. Empty means member forwarding may proceed.
// The control channel is not affected (§10.7, §10.8).
const (
	ReasonOutboxAge    = "outbox_age"
	ReasonOutboxBytes  = "outbox_bytes"
	ReasonDiskSpace    = "disk_space"
	ReasonPersistFail  = "persist_failed"
	ReasonArchiveBytes = "archive_bytes"
)

// Health is the forwarding-gate signal (§10.7 last paragraph, §10.8).
type Health struct {
	ForwardingAllowed bool
	Reason            string
	Detail            string
	UnackedBytes      uint64
	UnackedCount      int
	OldestUnacked     time.Time
	ArchiveBytes      uint64
	DBBytes           uint64
	FreeBytes         uint64
	AckedThrough      uint64
	LastSeq           uint64
	BlockedSeq        *uint64
	LedgerEpoch       uuid.UUID
	MeterEpoch        uuid.UUID
	PersistFailed     bool
}

// Defaults from §10.5 and §10.8. MB is decimal (§10.8).
const (
	DefaultPath            = "/var/lib/vpn-node/state/usage.db"
	DefaultUnackedMaxAge   = 72 * time.Hour
	DefaultUnackedMaxBytes = 256 * 1_000_000
	DefaultArchiveRetain   = 72 * time.Hour
	// DefaultArchiveMaxBytes is an engineering budget. The spec requires a
	// separate archive budget and does not set the number (§10.8).
	DefaultArchiveMaxBytes = 1_000_000_000
	resumeNum              = 4
	resumeDen              = 5
)

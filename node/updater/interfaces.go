package updater

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// Platform is the node this updater is running on (§3.2, §13.4).
type Platform struct {
	OSID            string `json:"os_id"`
	OSVersion       string `json:"os_version"`
	Architecture    string `json:"architecture"`
	UpdaterVersion  string `json:"updater_version"`
	ControlAPIMajor int    `json:"control_api_major"`
	PolicySchema    int    `json:"policy_schema"`
	UsageSchema     int    `json:"usage_schema"`
	StateSchema     int    `json:"state_schema"`
}

// Node is the local identity. The private key stays in the identity
// directory; this struct holds only ids (§13.7).
type Node struct {
	ID          uuid.UUID `json:"id"`
	Incarnation uuid.UUID `json:"incarnation"`
	BootID      uuid.UUID `json:"boot_id"`
	Revoked     bool      `json:"revoked"`
}

// Work is one verified update job plus the serving policy that was current
// when the job was accepted (§13.2, §13.10.2).
type Work struct {
	Intent  api.UpdateIntent `json:"intent"`
	Serving []uuid.UUID      `json:"serving"`
	// Revoked is the unsigned release-support list supplied by the agent.
	// A legacy envelope, which omits signed revoked_release_ids, uses only
	// this list plus whatever signed ids were already persisted.
	Revoked []uuid.UUID `json:"revoked"`
	// SignedRevoked is the signed revoked_release_ids member. Nil means the
	// member was absent. A non-nil empty slice is a signed empty set: it
	// adds nothing and does not clear the persisted set (§13.4).
	SignedRevoked *[]uuid.UUID `json:"signed_revoked,omitempty"`
	// PolicyGeneration is the desired-state config_generation of this
	// envelope. It is not the update job generation (D-043).
	PolicyGeneration decstr.U64 `json:"policy_generation"`
	// PolicyKnown is true when PolicyGeneration was taken from a verified
	// desired state, including generation 0. A zero generation without this
	// flag is not applied and is not reported.
	PolicyKnown bool `json:"policy_known,omitempty"`
}

// Source supplies the latest verified desired state. The agent may pass it
// in, or the updater may have fetched it over mTLS (§13.6 step 1).
type Source interface {
	Current(ctx context.Context) (Work, error)
}

// UsageCheckpoint is the metering barrier record (§10.7, §13.6 step 8).
type UsageCheckpoint struct {
	ClosedMeterEpochID  string     `json:"closed_meter_epoch_id"`
	LastDurableEventSeq decstr.U64 `json:"last_durable_event_seq"`
	CentralAckedThrough decstr.U64 `json:"central_acked_through"`
	NewMeterEpochID     string     `json:"new_meter_epoch_id"`
}

// Meter is the metering barrier. FinalSample is idempotent while member
// forwarding stays blocked: a crash that repeats the call must not invent
// bytes or burn a second event sequence (§10.7, §13.7). OpenEpoch may return
// a new id; the updater stores the first id in the journal note epoch-<txn>
// and does not call OpenEpoch again for that transaction. The meter must not
// write the id into the mutable state directory (D-034c).
type Meter interface {
	FinalSample(ctx context.Context) (UsageCheckpoint, error)
	OpenEpoch(ctx context.Context) (newEpochID string, err error)
}

// ServiceManager stops and starts the replaceable node services. Both calls
// are idempotent (§13.6 steps 9 and 11). The updater never executes files
// from the manifest; Start only receives the release directory.
type ServiceManager interface {
	Stop(ctx context.Context) error
	Start(ctx context.Context, releaseDir string) error
}

// Gate is the outer member-forwarding block (§13.6 step 7). It does not
// change publication intent. Block is idempotent.
type Gate interface {
	Block(ctx context.Context) error
	Unblock(ctx context.Context) error
}

// Migrator applies a journaled state migration (§13.6 step 9, §13.7).
// It is idempotent. On failure the previous release must still be able to
// read the state, unless the error is ErrStateUnsafe.
type Migrator interface {
	Migrate(ctx context.Context, fromSchema, toSchema int, migrationID, stateDir string) error
}

// ReleaseView is what a health check is allowed to see (§13.9).
type ReleaseView struct {
	ReleaseID        string
	Dir              string
	Manifest         *api.ReleaseManifest
	IdentityDir      string
	StateDir         string
	ExpectedIdentity string
	// ExpectedState is the SHA-256 of the immutable pointers and state-schema
	// snapshot, not a hash of the mutable state directory (D-034c).
	ExpectedState string
	// ImmutableRecord is the live encoding of that snapshot. Empty means the
	// caller is not asking for the pointers check.
	ImmutableRecord string
	ExpectedTree    string
}

// HealthChecks are the five §13.9 layers. A status other than passed, or a
// non-nil error, fails the candidate. not_run fails closed.
type HealthChecks interface {
	Processes(ctx context.Context, v ReleaseView) (api.HealthCheckStatus, error)
	PersistentData(ctx context.Context, v ReleaseView) (api.HealthCheckStatus, error)
	Isolation(ctx context.Context, v ReleaseView) (api.HealthCheckStatus, error)
	DNSPolicy(ctx context.Context, v ReleaseView) (api.HealthCheckStatus, error)
	TunnelProbe(ctx context.Context, v ReleaseView) (api.HealthCheckStatus, error)
}

// Accepted is the part of a central update-result ACK the outbox stores
// (§13.10.3). Local success is not acceptance.
type Accepted struct {
	HTTPStatus        int
	APICode           string
	AcceptedAsHistory bool
	Raw               json.RawMessage
}

// ResultSink delivers one outbox payload. conflict is ACK_CONFLICT (§13.8):
// the payload stays as evidence and is not resent. DeliveryError preserves
// HTTP/API diagnostics and identifies permanent rejections and retry deadlines.
// Other transient errors leave the entry pending.
type ResultSink interface {
	Submit(ctx context.Context, payload []byte) (ack Accepted, conflict bool, err error)
}

// Hooks are test seams. AfterStage runs after the stage is durable. AfterPhase
// runs at claimed, downloading, verifying and verified, before the next step.
type Hooks struct {
	AfterStage func(stage string) error
	AfterPhase func(phase string) error
}

// Faults injects disk-full errors. They are test-only and default off.
type Faults struct {
	DiskFullDownload bool
	DiskFullExtract  bool
}

// DownloadConfig is the bounded retry policy (§13.4). The numbers are
// suggested defaults pending measurement (待實測), not a verified SLO.
type DownloadConfig struct {
	MaxAttempts    int           `json:"max_attempts"`
	Initial        time.Duration `json:"initial"`
	MaxWait        time.Duration `json:"max_wait"`
	AttemptTimeout time.Duration `json:"attempt_timeout"`
	ContentRetries int           `json:"content_retries"`
}

// DefaultDownload is the suggested retry policy (待實測).
func DefaultDownload() DownloadConfig {
	return DownloadConfig{
		MaxAttempts:    5,
		Initial:        200 * time.Millisecond,
		MaxWait:        30 * time.Second,
		AttemptTimeout: 60 * time.Second,
		ContentRetries: 1,
	}
}

func (c DownloadConfig) normalize() DownloadConfig {
	if c.MaxAttempts < 1 {
		c.MaxAttempts = 1
	}
	if c.Initial <= 0 {
		c.Initial = 200 * time.Millisecond
	}
	if c.MaxWait <= 0 {
		c.MaxWait = 30 * time.Second
	}
	if c.AttemptTimeout <= 0 {
		c.AttemptTimeout = 60 * time.Second
	}
	if c.ContentRetries < 0 {
		c.ContentRetries = 0
	}
	return c
}

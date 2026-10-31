package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

const (
	// ProjectionNameMemberUsage is the canonical projection name stored in
	// projection_receipts and usage_projection_generations (§10.9, §15.4).
	ProjectionNameMemberUsage = "member_usage"

	// MeteringVersionForwardAdmissionV1 is the v1 metering definition (§10.1).
	MeteringVersionForwardAdmissionV1 = "forward_admission_v1"

	// MaxBatchEvents is the contract limit on events per batch (§10.5).
	MaxBatchEvents = 1000

	// SamplingIntervalBound is the D-112 central bound for D-105 retired-stream
	// hand-off. Nodes must sample at an interval no greater than this bound.
	SamplingIntervalBound = 10 * time.Second

	// ledgerStateSingletonID is the fixed UUID of the usage_ledger_state singleton row.
	ledgerStateSingletonID = "00000000-0000-0000-0000-000000000130"

	// fleetConfigSingletonID is the fixed UUID of the fleet_config singleton row
	// used as the audit resource for usage ledger recovery lifecycle events (Decision D-051).
	fleetConfigSingletonID = "00000000-0000-0000-0000-000000000001"

	// ErrCodeUint64Overflow is the detail/log code attached to InternalError when a
	// rollup or adjusted byte total exceeds math.MaxUint64 on read (§10.4).
	ErrCodeUint64Overflow = "usage_uint64_overflow"
)

// LedgerMode represents the central usage ledger operating mode (§17.3).
type LedgerMode string

const (
	LedgerModeNormal      LedgerMode = "normal"
	LedgerModeFrozen      LedgerMode = "frozen"
	LedgerModeReconciling LedgerMode = "reconciling"
)

// NodeIdentity is the authenticated node identity extracted from the mTLS
// client certificate by the control plane (§10.6, §14.4).
type NodeIdentity struct {
	NodeID            uuid.UUID
	NodeIncarnationID uuid.UUID
}

// LedgerState holds the singleton ledger epoch, operating mode, active recovery
// record (if any), and active projection generation.
type LedgerState struct {
	LedgerEpoch                uuid.UUID  `json:"ledger_epoch"`
	Mode                       LedgerMode `json:"mode"`
	ModeReason                 string     `json:"mode_reason"`
	ActiveRecoveryID           *uuid.UUID `json:"active_recovery_id,omitempty"`
	ActiveProjectionGeneration uint64     `json:"active_projection_generation"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

// Config configures the central usage ledger Service.
type Config struct {
	Pool              *pgxpool.Pool
	Signer            Signer
	LedgerEpochSigner Signer
	Clock             func() time.Time
	// SamplingIntervalBound overrides the D-112 central hand-off window bound.
	// Zero selects the recommended 10-second default; negative values are invalid.
	SamplingIntervalBound time.Duration
}

// Service implements the central usage ledger, checkpoints, projection,
// adjustments, queries, and restore/replay workflows (§10, §15.4, §17.3).
type Service struct {
	pool                  *pgxpool.Pool
	signer                Signer
	ledgerEpochSigner     Signer
	clock                 func() time.Time
	samplingIntervalBound time.Duration
}

// NewService constructs a central usage ledger Service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Pool == nil {
		return nil, errors.New("usage: database pool is required")
	}
	bound := cfg.SamplingIntervalBound
	if bound < 0 {
		return nil, errors.New("usage: sampling interval bound must be positive")
	}
	if bound == 0 {
		bound = SamplingIntervalBound
	}
	clock := cfg.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		pool:                  cfg.Pool,
		signer:                cfg.Signer,
		ledgerEpochSigner:     cfg.LedgerEpochSigner,
		clock:                 clock,
		samplingIntervalBound: bound,
	}, nil
}

func (s *Service) now() time.Time {
	return s.clock().UTC()
}

// GetLedgerState returns the current central usage ledger state.
func (s *Service) GetLedgerState(ctx context.Context) (LedgerState, error) {
	if s.pool == nil {
		return LedgerState{}, apierr.New(apierr.DependencyUnavailable, "usage ledger database unavailable")
	}
	var (
		st      LedgerState
		modeStr string
		genText string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT s.current_epoch,
		       s.mode,
		       s.mode_reason,
		       e.recovery_record_id,
		       g.generation::text,
		       s.updated_at
		FROM usage_ledger_state s
		JOIN usage_ledger_epochs e ON e.id = s.current_epoch
		JOIN usage_projection_generations g
		  ON g.projection_name = $2 AND g.state = 'active'
		WHERE s.id = $1
	`, ledgerStateSingletonID, ProjectionNameMemberUsage).Scan(
		&st.LedgerEpoch,
		&modeStr,
		&st.ModeReason,
		&st.ActiveRecoveryID,
		&genText,
		&st.UpdatedAt,
	)
	if err != nil {
		return LedgerState{}, apierr.Wrap(err, apierr.DependencyUnavailable, "usage ledger database unavailable")
	}
	gen, err := parseNumericUint64(genText)
	if err != nil {
		return LedgerState{}, fmt.Errorf("usage: parse active projection generation: %w", err)
	}
	st.Mode = LedgerMode(modeStr)
	st.ActiveProjectionGeneration = gen
	st.UpdatedAt = st.UpdatedAt.UTC()
	return st, nil
}

func (s *Service) getLedgerStateInTx(ctx context.Context, tx pgx.Tx) (LedgerState, error) {
	return s.queryLedgerStateInTx(ctx, tx, "")
}

func (s *Service) lockLedgerStateForShareInTx(ctx context.Context, tx pgx.Tx) (LedgerState, error) {
	return s.queryLedgerStateInTx(ctx, tx, "FOR SHARE OF s")
}

func (s *Service) lockLedgerStateForUpdateInTx(ctx context.Context, tx pgx.Tx) (LedgerState, error) {
	return s.queryLedgerStateInTx(ctx, tx, "FOR UPDATE OF s")
}

func (s *Service) queryLedgerStateInTx(ctx context.Context, tx pgx.Tx, lockClause string) (LedgerState, error) {
	var (
		st      LedgerState
		modeStr string
		genText string
	)
	q := `
		SELECT s.current_epoch,
		       s.mode,
		       s.mode_reason,
		       e.recovery_record_id,
		       g.generation::text,
		       s.updated_at
		FROM usage_ledger_state s
		JOIN usage_ledger_epochs e ON e.id = s.current_epoch
		JOIN usage_projection_generations g
		  ON g.projection_name = $2 AND g.state = 'active'
		WHERE s.id = $1
	`
	if lockClause != "" {
		q += "\n\t\t" + lockClause
	}
	err := tx.QueryRow(ctx, q, ledgerStateSingletonID, ProjectionNameMemberUsage).Scan(
		&st.LedgerEpoch,
		&modeStr,
		&st.ModeReason,
		&st.ActiveRecoveryID,
		&genText,
		&st.UpdatedAt,
	)
	if err != nil {
		return LedgerState{}, fmt.Errorf("usage: read ledger state in tx: %w", err)
	}
	gen, err := parseNumericUint64(genText)
	if err != nil {
		return LedgerState{}, fmt.Errorf("usage: parse active projection generation: %w", err)
	}
	st.Mode = LedgerMode(modeStr)
	st.ActiveProjectionGeneration = gen
	st.UpdatedAt = st.UpdatedAt.UTC()
	return st, nil
}

// checkNodeIncarnation verifies that the authenticated mTLS node identity exists
// and is neither revoked nor superseded in nodes / node_incarnations (§10.6, §14.4).
func (s *Service) checkNodeIncarnation(ctx context.Context, nodeID, incarnationID uuid.UUID) error {
	var (
		lifecycleState string
		currentIncID   *uuid.UUID
		incRevokedAt   *time.Time
	)
	err := s.pool.QueryRow(ctx, `
		SELECT n.lifecycle_state, n.current_incarnation_id, i.revoked_at
		FROM nodes n
		LEFT JOIN node_incarnations i ON i.node_id = n.id AND i.id = $2
		WHERE n.id = $1
	`, nodeID, incarnationID).Scan(&lifecycleState, &currentIncID, &incRevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(apierr.Forbidden, "unknown node identity")
	}
	if err != nil {
		return apierr.Wrap(err, apierr.DependencyUnavailable, "usage ledger database unavailable")
	}
	if lifecycleState == "revoked" || lifecycleState == "retired" || incRevokedAt != nil {
		return apierr.New(apierr.IdentityRevoked, fmt.Sprintf("node is %s", lifecycleState))
	}
	if currentIncID == nil || *currentIncID != incarnationID {
		return apierr.New(apierr.IncarnationMismatch, "node_incarnation_id does not match current node incarnation")
	}
	return nil
}

func parseNumericUint64(s string) (uint64, error) {
	u, err := decstr.Parse(s)
	if err != nil {
		return 0, err
	}
	return u.Uint64(), nil
}

func parseByteTotalUint64(s, field string) (uint64, error) {
	u, err := decstr.Parse(s)
	if err != nil {
		if errors.Is(err, decstr.ErrOverflow) {
			return 0, uint64OverflowError(field, s)
		}
		return 0, err
	}
	return u.Uint64(), nil
}

func uint64OverflowError(field, value string) error {
	return apierr.Wrap(
		fmt.Errorf("usage: %s=%s: %w", field, value, decstr.ErrOverflow),
		apierr.InternalError,
		"usage byte total overflows uint64",
	).WithDetail("detail_code", ErrCodeUint64Overflow).
		WithDetail("field", field)
}

func formatUint64(v uint64) string {
	return decstr.U64(v).String()
}

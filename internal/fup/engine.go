package fup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

var (
	// ErrMemberStateNotFound is returned when no FUP evaluation exists yet for a member and period.
	ErrMemberStateNotFound = errors.New("fup: member fair-use state not found")
	// ErrInvalidOverride is returned when an admin override has invalid parameters.
	ErrInvalidOverride = errors.New("fup: invalid admin override parameters")
)

// EnforcedState describes the internal state stored in `fup_member_states.enforced_state`.
type EnforcedState string

const (
	// EnforcedStateNormal indicates Tier 0 normal service.
	EnforcedStateNormal EnforcedState = "normal"
	// EnforcedStateAdjusted indicates FUP tier rate shaping (`tier_index > 0`).
	EnforcedStateAdjusted EnforcedState = "adjusted"
	// EnforcedStateOverride indicates an active time-limited admin override.
	EnforcedStateOverride EnforcedState = "override"
)

// LedgerWatermark captures the usage.GetMemberPeriodTotal watermark snapshot
// recorded during a Fair-Use evaluation (§11.1, §11.2).
type LedgerWatermark struct {
	LedgerEpoch          uuid.UUID               `json:"ledger_epoch,omitempty"`
	ProjectionGeneration decstr.U64              `json:"projection_generation,omitempty"`
	AggregateAsOf        time.Time               `json:"aggregate_as_of"`
	Streams              []usage.StreamWatermark `json:"streams,omitempty"`
}

// UsageSnapshot represents a member's aggregated usage across all nodes for an accounting period
// along with the ledger watermark (`AggregateAsOf`) and any negative ledger adjustment metadata (§10.4, §11.1).
type UsageSnapshot struct {
	MemberID              uuid.UUID
	PeriodID              string
	PeriodStart           time.Time
	PeriodEnd             time.Time
	TotalBytes            decstr.U64
	AggregateAsOf         time.Time
	MeteringVersion       string
	HasNegativeAdjustment bool
	AdjustmentReason      string
	Watermark             *LedgerWatermark
}

// UsageReader reads a member's period total across all nodes with its ledger watermark.
type UsageReader interface {
	GetMemberPeriodUsage(ctx context.Context, memberID uuid.UUID, periodID string) (UsageSnapshot, error)
}

// NewUsageServiceReader adapts `*usage.Service.GetMemberPeriodTotal` into a `UsageReader`,
// propagating its `LedgerWatermark` (`LedgerEpoch`, `ProjectionGeneration`, `Watermarks`, `AggregateAsOf`)
// into every Fair-Use computation (§11.2).
func NewUsageServiceReader(svc *usage.Service, pool *pgxpool.Pool) UsageReader {
	return &usageServiceReader{svc: svc, db: pool}
}

type usageServiceReader struct {
	svc *usage.Service
	db  *pgxpool.Pool
}

func (r *usageServiceReader) GetMemberPeriodUsage(ctx context.Context, memberID uuid.UUID, periodID string) (UsageSnapshot, error) {
	tot, err := r.svc.GetMemberPeriodTotal(ctx, memberID, periodID)
	if err != nil {
		return UsageSnapshot{}, err
	}
	asOf := time.Now().UTC()
	if tot.AggregateAsOf != nil && !tot.AggregateAsOf.IsZero() {
		asOf = tot.AggregateAsOf.UTC()
	} else if tot.LastObservedAt != nil && !tot.LastObservedAt.IsZero() {
		asOf = tot.LastObservedAt.UTC()
	}
	wm := &LedgerWatermark{
		LedgerEpoch:          tot.LedgerEpoch,
		ProjectionGeneration: tot.ProjectionGeneration,
		AggregateAsOf:        asOf,
		Streams:              tot.Watermarks,
	}
	snap := UsageSnapshot{
		MemberID:        memberID,
		PeriodID:        tot.Period,
		PeriodStart:     tot.PeriodStart,
		PeriodEnd:       tot.PeriodEnd,
		TotalBytes:      tot.TotalBytes,
		AggregateAsOf:   asOf,
		MeteringVersion: tot.MeteringVersion,
		Watermark:       wm,
	}
	if r.db != nil {
		var negReason string
		err = r.db.QueryRow(ctx, `
			SELECT reason
			FROM usage_adjustments
			WHERE member_id = $1 AND period = $2 AND (upload_bytes + download_bytes) < 0
			ORDER BY created_at DESC
			LIMIT 1
		`, memberID, tot.Period).Scan(&negReason)
		if err == nil {
			snap.HasNegativeAdjustment = true
			snap.AdjustmentReason = negReason
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return UsageSnapshot{}, fmt.Errorf("fup: query usage_adjustments: %w", err)
		}
	}
	return snap, nil
}

// MemberPolicyContext holds the active plan and fair-use policy revision resolved for a member.
type MemberPolicyContext struct {
	MemberID          uuid.UUID
	PlanRevisionID    *uuid.UUID
	PlanRateLimit     *api.RatePair
	FairUseRevisionID uuid.UUID
	FairUsePolicy     api.FairUsePayload
}

// MemberPolicyReader resolves the active fair-use policy and plan rate limit for a member.
type MemberPolicyReader interface {
	GetMemberFairUsePolicy(ctx context.Context, memberID uuid.UUID, now time.Time) (MemberPolicyContext, error)
}

// NodeAckStatus summarizes how many serving nodes for a member have acknowledged
// at least the member's current enforced FUP policy generation (§11.2, §11.6, FUP03).
type NodeAckStatus struct {
	TotalServingNodes int
	AckedNodes        int
	PendingNodes      int
}

// NodeAckReader queries per-node acknowledgment of a member's enforced FUP policy generation.
type NodeAckReader interface {
	GetMemberNodeAckStatus(ctx context.Context, memberID uuid.UUID, generation decstr.U64) (NodeAckStatus, error)
}

// MemberOverride represents an admin rate override for a member (§11.1, FUP04).
type MemberOverride struct {
	ID               uuid.UUID  `json:"id"`
	MemberID         uuid.UUID  `json:"member_id"`
	DownBps          int64      `json:"down_bps"`
	UpBps            int64      `json:"up_bps"`
	Priority         int        `json:"priority"`
	ValidFrom        time.Time  `json:"valid_from"`
	ValidUntil       time.Time  `json:"valid_until"`
	Reason           string     `json:"reason"`
	ActorID          uuid.UUID  `json:"actor_id"`
	CreatedAt        time.Time  `json:"created_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevokedBy        *uuid.UUID `json:"revoked_by,omitempty"`
	RevocationReason *string    `json:"revocation_reason,omitempty"`
}

// ComputeInput specifies the parameters for a single FUP evaluation run.
type ComputeInput struct {
	MemberID         uuid.UUID
	PeriodID         string
	CurrentPeriodID  string
	AllowRelaxation  bool
	RelaxationReason string
	Actor            string
	Now              time.Time
}

// EvaluationResult holds the complete outcome of `Engine.Compute` for both shadow and enforce modes.
type EvaluationResult struct {
	StateID               uuid.UUID                   `json:"state_id"`
	EvaluationID          uuid.UUID                   `json:"evaluation_id"`
	MemberID              uuid.UUID                   `json:"member_id"`
	PeriodID              string                      `json:"period_id"`
	ValidFrom             time.Time                   `json:"valid_from"`
	ValidUntil            time.Time                   `json:"valid_until"`
	PolicyRevisionID      uuid.UUID                   `json:"policy_revision_id"`
	Mode                  api.FairUsePayloadMode      `json:"mode"`
	WouldApplyTierIndex   int                         `json:"would_apply_tier_index"`
	WouldApplyTierName    api.LocalizedText           `json:"would_apply_tier_name"`
	WouldApplyDownBps     int64                       `json:"would_apply_down_bps"`
	WouldApplyUpBps       int64                       `json:"would_apply_up_bps"`
	WouldApplyPriority    int                         `json:"would_apply_priority"`
	WouldApplyReason      api.MemberRatePolicyReason  `json:"would_apply_reason"`
	EnforcedState         EnforcedState               `json:"enforced_state"`
	EnforcedTierIndex     int                         `json:"enforced_tier_index"`
	EnforcedTierName      api.LocalizedText           `json:"enforced_tier_name"`
	EnforcedDownBps       *int64                      `json:"enforced_down_bps,omitempty"`
	EnforcedUpBps         *int64                      `json:"enforced_up_bps,omitempty"`
	EnforcedPriority      *int                        `json:"enforced_priority,omitempty"`
	EnforcedReason        *api.MemberRatePolicyReason `json:"enforced_reason,omitempty"`
	Generation            decstr.U64                  `json:"generation"`
	GenerationIncremented bool                        `json:"generation_incremented"`
	Relaxed               bool                        `json:"relaxed"`
	RelaxationReason      string                      `json:"relaxation_reason,omitempty"`
	LatePeriodEvent       bool                        `json:"late_period_event"`
	TotalBytes            decstr.U64                  `json:"total_bytes"`
	AggregateAsOf         time.Time                   `json:"aggregate_as_of"`
	OverrideID            *uuid.UUID                  `json:"override_id,omitempty"`
	OverrideValidUntil    *time.Time                  `json:"override_valid_until,omitempty"`
	Watermark             *LedgerWatermark            `json:"ledger_watermark,omitempty"`
	ImpactSummary         map[string]any              `json:"impact_summary"`
	ComputedAt            time.Time                   `json:"computed_at"`
}

// AdminMemberFairUseStatus exposes both the member-visible `FairUseStatus` and the
// admin-visible shadow/enforcement diagnostics and per-node ACK summary (§11.2, §11.6, FUP03).
type AdminMemberFairUseStatus struct {
	MemberStatus        api.FairUseStatus          `json:"member_status"`
	Mode                api.FairUsePayloadMode     `json:"mode"`
	EnforcedState       EnforcedState              `json:"enforced_state"`
	Generation          decstr.U64                 `json:"generation"`
	TotalBytes          decstr.U64                 `json:"total_bytes"`
	WouldApplyTierIndex int                        `json:"would_apply_tier_index"`
	WouldApplyTierName  api.LocalizedText          `json:"would_apply_tier_name"`
	WouldApplyRate      api.RatePair               `json:"would_apply_rate"`
	WouldApplyPriority  int                        `json:"would_apply_priority"`
	WouldApplyReason    api.MemberRatePolicyReason `json:"would_apply_reason"`
	OverrideID          *uuid.UUID                 `json:"override_id,omitempty"`
	OverrideValidUntil  *time.Time                 `json:"override_valid_until,omitempty"`
	TotalServingNodes   int                        `json:"total_serving_nodes"`
	AckedNodes          int                        `json:"acked_nodes"`
	PendingNodes        int                        `json:"pending_nodes"`
	NodeSyncSummary     string                     `json:"node_sync_summary"`
	LastComputedAt      time.Time                  `json:"last_computed_at"`
}

// SetOverrideInput holds parameters for creating a time-limited admin override (FUP04).
type SetOverrideInput struct {
	MemberID   uuid.UUID
	DownBps    int64
	UpBps      int64
	Priority   int
	ValidFrom  time.Time
	ValidUntil time.Time
	Reason     string
	ActorID    uuid.UUID
	Now        time.Time
}

// RevokeOverrideInput holds parameters for revoking an active admin override early.
type RevokeOverrideInput struct {
	OverrideID uuid.UUID
	MemberID   uuid.UUID
	ActorID    uuid.UUID
	Reason     string
	Now        time.Time
}

// Engine evaluates member fair-use policies, tracks shadow and enforced states,
// manages time-limited admin overrides, and produces member/admin status views.
type Engine struct {
	db           *pgxpool.Pool
	usageReader  UsageReader
	policyReader MemberPolicyReader
	ackReader    NodeAckReader
	modeOverride *api.FairUsePayloadMode

	// In-memory state fallback when db is nil (used by pure unit tests with no DB).
	mu             sync.RWMutex
	memStates      map[string]*EvaluationResult
	memEvaluations []*EvaluationResult
	memMaxGen      map[uuid.UUID]decstr.U64
	memOverrides   map[uuid.UUID][]*MemberOverride
}

// NewEngine constructs a Fair-Use Policy Engine.
// If `db` is non-nil and any reader interface is nil, a PostgreSQL-backed reader is used.
func NewEngine(db *pgxpool.Pool, usageReader UsageReader, policyReader MemberPolicyReader, ackReader NodeAckReader) *Engine {
	e := &Engine{
		db:           db,
		usageReader:  usageReader,
		policyReader: policyReader,
		ackReader:    ackReader,
		memStates:    make(map[string]*EvaluationResult),
		memMaxGen:    make(map[uuid.UUID]decstr.U64),
		memOverrides: make(map[uuid.UUID][]*MemberOverride),
	}
	if db != nil {
		if e.usageReader == nil {
			e.usageReader = &pgUsageReader{db: db}
		}
		if e.policyReader == nil {
			e.policyReader = &pgMemberPolicyReader{db: db}
		}
		if e.ackReader == nil {
			e.ackReader = &pgNodeAckReader{db: db}
		}
	}
	return e
}

// SetModeOverride sets an optional runtime override for the Fair-Use mode ("shadow" or "enforce")
// from central configuration (`FVPN_FUP_MODE`, §11.2). Passing "" clears the override so the active
// policy revision's `mode` field governs.
func (e *Engine) SetModeOverride(mode string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch strings.TrimSpace(mode) {
	case string(api.FairUsePayloadModeShadow):
		m := api.FairUsePayloadModeShadow
		e.modeOverride = &m
	case string(api.FairUsePayloadModeEnforce):
		m := api.FairUsePayloadModeEnforce
		e.modeOverride = &m
	default:
		e.modeOverride = nil
	}
}

func stateKey(memberID uuid.UUID, periodID string) string {
	return memberID.String() + ":" + periodID
}

// Compute evaluates a member's fair-use state for an accounting period (§11.1, §11.2, §11.5).
// Identical tier-selection, stricter-only clamping, and override resolution code runs in
// both `shadow` and `enforce` modes; only the output state and generation publication differ.
func (e *Engine) Compute(ctx context.Context, in ComputeInput) (*EvaluationResult, error) {
	if in.MemberID == uuid.Nil {
		return nil, errors.New("fup: member_id is required")
	}
	now := in.Now.UTC()
	if in.Now.IsZero() {
		now = time.Now().UTC()
	}
	currentPeriodID := strings.TrimSpace(in.CurrentPeriodID)
	if currentPeriodID == "" {
		currentPeriodID = CurrentUTCMonthPeriodID(now)
	}
	periodID := strings.TrimSpace(in.PeriodID)
	if periodID == "" {
		periodID = currentPeriodID
	}
	actor := strings.TrimSpace(in.Actor)
	if actor == "" {
		actor = fupSystemActorUUID
	}

	if e.usageReader == nil || e.policyReader == nil {
		return nil, errors.New("fup: usageReader and policyReader are required")
	}

	usageSnap, err := e.usageReader.GetMemberPeriodUsage(ctx, in.MemberID, periodID)
	if err != nil {
		return nil, fmt.Errorf("fup: read usage snapshot: %w", err)
	}
	if usageSnap.PeriodStart.IsZero() || usageSnap.PeriodEnd.IsZero() {
		pStart, pEnd, err := PeriodBoundsForUTCMonth(periodID)
		if err != nil {
			return nil, err
		}
		usageSnap.PeriodStart = pStart
		usageSnap.PeriodEnd = pEnd
	}
	if usageSnap.AggregateAsOf.IsZero() {
		usageSnap.AggregateAsOf = now
	}

	polCtx, err := e.policyReader.GetMemberFairUsePolicy(ctx, in.MemberID, now)
	if err != nil {
		return nil, fmt.Errorf("fup: read member fair_use policy: %w", err)
	}
	policy := polCtx.FairUsePolicy

	// 1. Select candidate tier from total_bytes using threshold <= U < next (§11.1).
	candidateIdx, _, err := SelectTier(policy.Tiers, usageSnap.TotalBytes)
	if err != nil {
		return nil, err
	}

	// 2. Load previous state for (member_id, period_id) and max generation across all periods.
	prevState, maxGen, err := e.loadPreviousState(ctx, in.MemberID, periodID)
	if err != nil {
		return nil, err
	}

	// 3. Enforce stricter-only movement within the same accounting period (§11.1).
	// Normal accumulation never lowers the tier index; only an audited negative ledger
	// adjustment (`AllowRelaxation` / `HasNegativeAdjustment`) allows tier relaxation.
	allowRelax := in.AllowRelaxation || usageSnap.HasNegativeAdjustment
	relaxReason := in.RelaxationReason
	if relaxReason == "" {
		relaxReason = usageSnap.AdjustmentReason
	}
	relaxed := false
	if prevState != nil && candidateIdx < prevState.WouldApplyTierIndex {
		if allowRelax {
			relaxed = true
			if relaxReason == "" {
				relaxReason = "ledger_negative_adjustment"
			}
		} else {
			candidateIdx = prevState.WouldApplyTierIndex
		}
	}

	selectedTier := policy.Tiers[candidateIdx]

	// 4. Resolve active time-limited admin override (`valid_from <= now < valid_until` and not revoked, FUP04).
	activeOverride, err := e.resolveActiveOverride(ctx, in.MemberID, policy.Overrides, now)
	if err != nil {
		return nil, err
	}

	// 5. Compute would-apply outcome (shared by shadow and enforce modes).
	wouldApplyTierIdx := candidateIdx
	wouldApplyTierName := selectedTier.DisplayName
	wouldApplyDownBps := selectedTier.DownBps
	wouldApplyUpBps := selectedTier.UpBps
	wouldApplyPriority := selectedTier.Priority
	wouldApplyReason := api.MemberRatePolicyReasonNormal
	wouldApplyState := EnforcedStateNormal
	if candidateIdx > 0 {
		wouldApplyReason = api.MemberRatePolicyReasonFairUseTier
		wouldApplyState = EnforcedStateAdjusted
	}

	var overrideID *uuid.UUID
	var overrideValidUntil *time.Time
	if activeOverride != nil {
		wouldApplyDownBps = activeOverride.DownBps
		wouldApplyUpBps = activeOverride.UpBps
		wouldApplyPriority = activeOverride.Priority
		wouldApplyReason = api.MemberRatePolicyReasonAdminOverride
		wouldApplyState = EnforcedStateOverride
		if activeOverride.ID != uuid.Nil {
			idCopy := activeOverride.ID
			overrideID = &idCopy
		}
		untilCopy := activeOverride.ValidUntil.UTC()
		overrideValidUntil = &untilCopy
	}

	// Determine effective mode: if policy.Mode is "shadow" or now < policy.EffectiveAt, run in shadow mode (§11.1, §11.2).
	effectiveMode := policy.Mode
	e.mu.RLock()
	if e.modeOverride != nil {
		effectiveMode = *e.modeOverride
	}
	e.mu.RUnlock()
	if now.Before(policy.EffectiveAt) {
		effectiveMode = api.FairUsePayloadModeShadow
	}

	isLatePreviousPeriod := periodID < currentPeriodID

	impactSummary := map[string]any{
		"period_id":              periodID,
		"total_bytes":            usageSnap.TotalBytes.String(),
		"aggregate_as_of":        usageSnap.AggregateAsOf.UTC().Format(time.RFC3339Nano),
		"metering_version":       string(policy.AccountingBasis.MeteringVersion),
		"policy_mode":            string(effectiveMode),
		"would_apply_tier_index": wouldApplyTierIdx,
		"would_apply_down_bps":   wouldApplyDownBps,
		"would_apply_up_bps":     wouldApplyUpBps,
		"would_apply_priority":   wouldApplyPriority,
		"would_apply_reason":     string(wouldApplyReason),
		"relaxed":                relaxed,
		"late_period_event":      isLatePreviousPeriod,
	}
	if usageSnap.Watermark != nil {
		impactSummary["ledger_watermark"] = usageSnap.Watermark
	}
	if overrideValidUntil != nil {
		impactSummary["override_valid_until"] = overrideValidUntil.Format(time.RFC3339Nano)
	}

	// 6. Branch on output/publication for late previous-period events, shadow mode, and enforce mode (§11.2, §11.5).
	var (
		enforcedState    EnforcedState
		enforcedTierIdx  int
		enforcedTierName api.LocalizedText
		enforcedDownBps  *int64
		enforcedUpBps    *int64
		enforcedPriority *int
		enforcedReason   *api.MemberRatePolicyReason
		generation       decstr.U64
		genIncremented   bool
		validUntil       = usageSnap.PeriodEnd.UTC()
	)

	if overrideValidUntil != nil && overrideValidUntil.Before(validUntil) {
		validUntil = *overrideValidUntil
	}

	if isLatePreviousPeriod {
		// Late event for a previous period (§11.5, AT17, AT19):
		// Update only the previous period's evaluation record; never alter its enforced generation
		// or the current period's state/generation.
		if prevState != nil {
			enforcedState = prevState.EnforcedState
			enforcedTierIdx = prevState.EnforcedTierIndex
			enforcedTierName = prevState.EnforcedTierName
			enforcedDownBps = copyInt64Ptr(prevState.EnforcedDownBps)
			enforcedUpBps = copyInt64Ptr(prevState.EnforcedUpBps)
			enforcedPriority = copyIntPtr(prevState.EnforcedPriority)
			enforcedReason = copyReasonPtr(prevState.EnforcedReason)
			generation = prevState.Generation
		} else {
			enforcedState = EnforcedStateNormal
			enforcedTierIdx = 0
			enforcedTierName = policy.Tiers[0].DisplayName
			generation = 0
		}
		genIncremented = false
	} else if effectiveMode == api.FairUsePayloadModeShadow {
		// Shadow mode (§11.2): record would_apply_tier and impact_summary, never enforce or emit a generation.
		enforcedState = EnforcedStateNormal
		enforcedTierIdx = 0
		enforcedTierName = policy.Tiers[0].DisplayName
		enforcedDownBps = nil
		enforcedUpBps = nil
		enforcedPriority = nil
		enforcedReason = nil
		if prevState != nil {
			generation = prevState.Generation
		} else {
			generation = 0
		}
		genIncremented = false
	} else {
		// Enforce mode (§11.2): produce a new member policy generation iff the enforced parameters changed.
		enforcedState = wouldApplyState
		enforcedTierIdx = wouldApplyTierIdx
		enforcedTierName = wouldApplyTierName
		enforcedDownBps = copyInt64Ptr(&wouldApplyDownBps)
		enforcedUpBps = copyInt64Ptr(&wouldApplyUpBps)
		enforcedPriority = copyIntPtr(&wouldApplyPriority)
		enforcedReason = copyReasonPtr(&wouldApplyReason)

		changed := enforcedOutputChanged(
			prevState,
			effectiveMode,
			enforcedState,
			enforcedTierIdx,
			enforcedDownBps,
			enforcedUpBps,
			enforcedPriority,
			enforcedReason,
			overrideValidUntil,
		)
		if changed {
			generation = maxGen + 1
			genIncremented = true
		} else {
			generation = prevState.Generation
			genIncremented = false
		}
	}

	res := &EvaluationResult{
		StateID:               uuid.New(),
		EvaluationID:          uuid.New(),
		MemberID:              in.MemberID,
		PeriodID:              periodID,
		ValidFrom:             usageSnap.PeriodStart.UTC(),
		ValidUntil:            validUntil,
		PolicyRevisionID:      polCtx.FairUseRevisionID,
		Mode:                  effectiveMode,
		WouldApplyTierIndex:   wouldApplyTierIdx,
		WouldApplyTierName:    wouldApplyTierName,
		WouldApplyDownBps:     wouldApplyDownBps,
		WouldApplyUpBps:       wouldApplyUpBps,
		WouldApplyPriority:    wouldApplyPriority,
		WouldApplyReason:      wouldApplyReason,
		EnforcedState:         enforcedState,
		EnforcedTierIndex:     enforcedTierIdx,
		EnforcedTierName:      enforcedTierName,
		EnforcedDownBps:       enforcedDownBps,
		EnforcedUpBps:         enforcedUpBps,
		EnforcedPriority:      enforcedPriority,
		EnforcedReason:        enforcedReason,
		Generation:            generation,
		GenerationIncremented: genIncremented,
		Relaxed:               relaxed,
		RelaxationReason:      relaxReason,
		LatePeriodEvent:       isLatePreviousPeriod,
		TotalBytes:            usageSnap.TotalBytes,
		AggregateAsOf:         usageSnap.AggregateAsOf.UTC(),
		OverrideID:            overrideID,
		OverrideValidUntil:    overrideValidUntil,
		Watermark:             usageSnap.Watermark,
		ImpactSummary:         impactSummary,
		ComputedAt:            now,
	}
	if prevState != nil {
		res.StateID = prevState.StateID
	}

	if err := e.persistEvaluation(ctx, res, actor); err != nil {
		return nil, err
	}
	return res, nil
}

// SetOverride records a time-limited, audited admin override for a member and immediately
// recomputes the member's fair-use state (§11.1, FUP04).
func (e *Engine) SetOverride(ctx context.Context, in SetOverrideInput) (*MemberOverride, *EvaluationResult, error) {
	if in.MemberID == uuid.Nil {
		return nil, nil, fmt.Errorf("%w: member_id is required", ErrInvalidOverride)
	}
	now := in.Now.UTC()
	if in.Now.IsZero() {
		now = time.Now().UTC()
	}
	validFrom := in.ValidFrom.UTC()
	if in.ValidFrom.IsZero() {
		validFrom = now
	}
	validUntil := in.ValidUntil.UTC()
	if !validUntil.After(validFrom) {
		return nil, nil, fmt.Errorf("%w: valid_until must be after valid_from", ErrInvalidOverride)
	}
	if in.DownBps < 0 || in.UpBps < 0 {
		return nil, nil, fmt.Errorf("%w: down_bps and up_bps must be non-negative", ErrInvalidOverride)
	}
	if in.Priority < 0 || in.Priority > 7 {
		return nil, nil, fmt.Errorf("%w: priority must be between 0 and 7", ErrInvalidOverride)
	}
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 500 {
		return nil, nil, fmt.Errorf("%w: reason is required, nonblank, and must be <= 500 characters", ErrInvalidOverride)
	}
	if in.ActorID == uuid.Nil {
		return nil, nil, fmt.Errorf("%w: actor_id is required", ErrInvalidOverride)
	}

	ovID, err := uuid.NewV7()
	if err != nil {
		ovID = uuid.New()
	}
	ov := &MemberOverride{
		ID:         ovID,
		MemberID:   in.MemberID,
		DownBps:    in.DownBps,
		UpBps:      in.UpBps,
		Priority:   in.Priority,
		ValidFrom:  validFrom,
		ValidUntil: validUntil,
		Reason:     in.Reason,
		ActorID:    in.ActorID,
		CreatedAt:  now,
	}

	if e.db != nil {
		afterDiff, err := json.Marshal(map[string]any{
			"override_id": ov.ID.String(),
			"member_id":   ov.MemberID.String(),
			"down_bps":    ov.DownBps,
			"up_bps":      ov.UpBps,
			"priority":    ov.Priority,
			"valid_from":  ov.ValidFrom.Format(time.RFC3339Nano),
			"valid_until": ov.ValidUntil.Format(time.RFC3339Nano),
			"reason":      ov.Reason,
		})
		if err != nil {
			return nil, nil, err
		}

		err = db.InTx(ctx, e.db, db.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `
				INSERT INTO fup_member_overrides (
					id, member_id, down_bps, up_bps, priority,
					valid_from, valid_until, reason, actor_id, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			`, ov.ID, ov.MemberID, ov.DownBps, ov.UpBps, ov.Priority,
				ov.ValidFrom, ov.ValidUntil, ov.Reason, ov.ActorID, ov.CreatedAt); err != nil {
				return fmt.Errorf("fup: insert override: %w", err)
			}
			if err := db.AppendAudit(ctx, tx, db.AuditEvent{
				Actor:         in.ActorID.String(),
				ActorType:     string(api.AuditEventActorTypeAdmin),
				Action:        "fup.override.set",
				Resource:      string(api.ResourceTypeMember) + ":" + in.MemberID.String(),
				Result:        string(api.AuditEventResultSucceeded),
				SchemaVersion: 1,
				Before:        []byte(`{}`),
				After:         afterDiff,
			}); err != nil {
				return fmt.Errorf("fup: append override audit event: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	} else {
		e.mu.Lock()
		e.memOverrides[in.MemberID] = append(e.memOverrides[in.MemberID], ov)
		e.mu.Unlock()
	}

	eval, err := e.Compute(ctx, ComputeInput{
		MemberID: in.MemberID,
		Actor:    in.ActorID.String(),
		Now:      now,
	})
	if err != nil {
		return ov, nil, err
	}
	return ov, eval, nil
}

// RevokeOverride revokes an active admin override early and recomputes the member's fair-use state.
func (e *Engine) RevokeOverride(ctx context.Context, in RevokeOverrideInput) (*EvaluationResult, error) {
	now := in.Now.UTC()
	if in.Now.IsZero() {
		now = time.Now().UTC()
	}
	if in.OverrideID == uuid.Nil || in.MemberID == uuid.Nil || in.ActorID == uuid.Nil || strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 500 {
		return nil, fmt.Errorf("%w: override_id, member_id, actor_id and nonblank reason are required to revoke an override", ErrInvalidOverride)
	}

	if e.db != nil {
		afterDiff, err := json.Marshal(map[string]any{
			"override_id":       in.OverrideID.String(),
			"member_id":         in.MemberID.String(),
			"revoked_at":        now.Format(time.RFC3339Nano),
			"revoked_by":        in.ActorID.String(),
			"revocation_reason": in.Reason,
		})
		if err != nil {
			return nil, err
		}

		err = db.InTx(ctx, e.db, db.TxOptions{}, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `
				UPDATE fup_member_overrides
				SET revoked_at = $1, revoked_by = $2, revocation_reason = $3
				WHERE id = $4 AND member_id = $5 AND revoked_at IS NULL
			`, now, in.ActorID, in.Reason, in.OverrideID, in.MemberID)
			if err != nil {
				return fmt.Errorf("fup: update override revocation: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return ErrMemberStateNotFound
			}
			if err := db.AppendAudit(ctx, tx, db.AuditEvent{
				Actor:         in.ActorID.String(),
				ActorType:     string(api.AuditEventActorTypeAdmin),
				Action:        "fup.override.revoked",
				Resource:      string(api.ResourceTypeMember) + ":" + in.MemberID.String(),
				Result:        string(api.AuditEventResultSucceeded),
				SchemaVersion: 1,
				Before:        []byte(`{}`),
				After:         afterDiff,
			}); err != nil {
				return fmt.Errorf("fup: append revoke override audit: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		e.mu.Lock()
		list := e.memOverrides[in.MemberID]
		found := false
		for _, ov := range list {
			if ov.ID == in.OverrideID && ov.RevokedAt == nil {
				tCopy := now
				actorCopy := in.ActorID
				reasonCopy := in.Reason
				ov.RevokedAt = &tCopy
				ov.RevokedBy = &actorCopy
				ov.RevocationReason = &reasonCopy
				found = true
				break
			}
		}
		e.mu.Unlock()
		if !found {
			return nil, ErrMemberStateNotFound
		}
	}

	return e.Compute(ctx, ComputeInput{
		MemberID: in.MemberID,
		Actor:    in.ActorID.String(),
		Now:      now,
	})
}

// GetMemberStatus returns the member-visible `api.FairUseStatus` for `(memberID, periodID)` (§11.2, §11.6, FUP03).
// In shadow mode it always reports `state: "normal"` with no rate or tier throttling.
// In enforce mode `Applying` remains `true` until all serving nodes have reported
// `applied_fup_generation >= generation` (never claiming applied on a 202).
func (e *Engine) GetMemberStatus(ctx context.Context, memberID uuid.UUID, periodID string) (api.FairUseStatus, error) {
	adminStatus, err := e.GetAdminMemberStatus(ctx, memberID, periodID)
	if err != nil {
		return api.FairUseStatus{}, err
	}
	return adminStatus.MemberStatus, nil
}

// GetAdminMemberStatus returns the admin-visible fair-use status including shadow evaluation
// details and per-node acknowledgment progress (`"pending on N nodes"` / `"applied on N nodes"`, §11.2, §11.6, FUP03).
func (e *Engine) GetAdminMemberStatus(ctx context.Context, memberID uuid.UUID, periodID string) (AdminMemberFairUseStatus, error) {
	if periodID == "" {
		periodID = CurrentUTCMonthPeriodID(time.Now().UTC())
	}
	st, _, err := e.loadPreviousState(ctx, memberID, periodID)
	if err != nil {
		return AdminMemberFairUseStatus{}, err
	}
	if st == nil {
		return AdminMemberFairUseStatus{}, ErrMemberStateNotFound
	}

	var ackStatus NodeAckStatus
	if st.Mode == api.FairUsePayloadModeEnforce && st.Generation > 0 && e.ackReader != nil {
		ackStatus, err = e.ackReader.GetMemberNodeAckStatus(ctx, memberID, st.Generation)
		if err != nil {
			return AdminMemberFairUseStatus{}, fmt.Errorf("fup: query node ack status: %w", err)
		}
	}

	asOf := st.AggregateAsOf.UTC()
	resetsAt := st.ValidUntil.UTC()

	var memberStatus api.FairUseStatus
	var nodeSyncSummary string

	if st.Mode == api.FairUsePayloadModeShadow {
		// Shadow mode (§11.2): never claim throttling on the member view.
		memberStatus = api.FairUseStatus{
			State:     api.FairUseStatusStateNormal,
			Applying:  false,
			AsOf:      presence.NewNullableWithValue(asOf),
			ResetsAt:  presence.NewNullableWithValue(resetsAt),
			Rate:      presence.NewNullNullable[api.RatePair](),
			Reason:    presence.NewNullNullable[api.FairUseStatusReason](),
			TierIndex: presence.NewNullNullable[int](),
			TierName:  presence.NewNullNullable[api.LocalizedText](),
		}
		nodeSyncSummary = "shadow (no enforcement)"
	} else {
		// Enforce mode (§11.2, FUP03): Applying is true until all serving nodes ACK the generation.
		applying := false
		if st.Generation > 0 {
			if ackStatus.TotalServingNodes == 0 || ackStatus.PendingNodes > 0 {
				applying = true
			}
		}
		if ackStatus.TotalServingNodes == 0 {
			nodeSyncSummary = "pending desired-state compilation"
		} else if ackStatus.PendingNodes > 0 {
			nodeSyncSummary = fmt.Sprintf("pending on %d nodes", ackStatus.PendingNodes)
		} else {
			nodeSyncSummary = fmt.Sprintf("applied on %d nodes", ackStatus.AckedNodes)
		}

		var ratePair *api.RatePair
		if st.EnforcedDownBps != nil && st.EnforcedUpBps != nil {
			ratePair = &api.RatePair{
				DownBps: *st.EnforcedDownBps,
				UpBps:   *st.EnforcedUpBps,
			}
		}
		tierIdxCopy := st.EnforcedTierIndex
		tierNameCopy := st.EnforcedTierName

		memberState := api.FairUseStatusStateNormal
		if st.EnforcedState != EnforcedStateNormal {
			memberState = api.FairUseStatusStateAdjusted
		}

		var memberReason *api.FairUseStatusReason
		if st.EnforcedReason != nil {
			switch *st.EnforcedReason {
			case api.MemberRatePolicyReasonFairUseTier:
				r := api.FairUseStatusReasonFairUseTier
				memberReason = &r
			case api.MemberRatePolicyReasonAdminOverride:
				r := api.FairUseStatusReasonAdminOverride
				memberReason = &r
			}
		}

		memberStatus = api.FairUseStatus{
			State:     memberState,
			Applying:  applying,
			AsOf:      presence.NewNullableWithValue(asOf),
			ResetsAt:  presence.NewNullableWithValue(resetsAt),
			Rate:      presence.FromPointer(ratePair),
			Reason:    presence.FromPointer(memberReason),
			TierIndex: presence.NewNullableWithValue(tierIdxCopy),
			TierName:  presence.NewNullableWithValue(tierNameCopy),
		}
	}

	return AdminMemberFairUseStatus{
		MemberStatus:        memberStatus,
		Mode:                st.Mode,
		EnforcedState:       st.EnforcedState,
		Generation:          st.Generation,
		TotalBytes:          st.TotalBytes,
		WouldApplyTierIndex: st.WouldApplyTierIndex,
		WouldApplyTierName:  st.WouldApplyTierName,
		WouldApplyRate: api.RatePair{
			DownBps: st.WouldApplyDownBps,
			UpBps:   st.WouldApplyUpBps,
		},
		WouldApplyPriority: st.WouldApplyPriority,
		WouldApplyReason:   st.WouldApplyReason,
		OverrideID:         st.OverrideID,
		OverrideValidUntil: st.OverrideValidUntil,
		TotalServingNodes:  ackStatus.TotalServingNodes,
		AckedNodes:         ackStatus.AckedNodes,
		PendingNodes:       ackStatus.PendingNodes,
		NodeSyncSummary:    nodeSyncSummary,
		LastComputedAt:     st.ComputedAt,
	}, nil
}

func enforcedOutputChanged(
	prev *EvaluationResult,
	mode api.FairUsePayloadMode,
	state EnforcedState,
	tierIdx int,
	downBps, upBps *int64,
	priority *int,
	reason *api.MemberRatePolicyReason,
	overrideValidUntil *time.Time,
) bool {
	if prev == nil {
		return true
	}
	if prev.Mode != mode || prev.Generation == 0 {
		return true
	}
	if prev.EnforcedState != state || prev.EnforcedTierIndex != tierIdx {
		return true
	}
	if !equalInt64Ptr(prev.EnforcedDownBps, downBps) || !equalInt64Ptr(prev.EnforcedUpBps, upBps) {
		return true
	}
	if !equalIntPtr(prev.EnforcedPriority, priority) || !equalReasonPtr(prev.EnforcedReason, reason) {
		return true
	}
	if !equalTimePtr(prev.OverrideValidUntil, overrideValidUntil) {
		return true
	}
	return false
}

func (e *Engine) resolveActiveOverride(
	ctx context.Context,
	memberID uuid.UUID,
	policyOverrides []api.FairUseOverride,
	now time.Time,
) (*MemberOverride, error) {
	if e.db != nil {
		var ov MemberOverride
		err := e.db.QueryRow(ctx, `
			SELECT id, member_id, down_bps, up_bps, priority,
			       valid_from, valid_until, reason, actor_id, created_at
			FROM fup_member_overrides
			WHERE member_id = $1
			  AND revoked_at IS NULL
			  AND valid_from <= $2
			  AND valid_until > $2
			ORDER BY created_at DESC
			LIMIT 1
		`, memberID, now).Scan(
			&ov.ID, &ov.MemberID, &ov.DownBps, &ov.UpBps, &ov.Priority,
			&ov.ValidFrom, &ov.ValidUntil, &ov.Reason, &ov.ActorID, &ov.CreatedAt,
		)
		if err == nil {
			return &ov, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("fup: query active override: %w", err)
		}
	} else {
		e.mu.RLock()
		list := e.memOverrides[memberID]
		var latest *MemberOverride
		for _, candidate := range list {
			if candidate.RevokedAt != nil {
				continue
			}
			if !now.Before(candidate.ValidFrom) && now.Before(candidate.ValidUntil) {
				if latest == nil || candidate.CreatedAt.After(latest.CreatedAt) {
					cp := *candidate
					latest = &cp
				}
			}
		}
		e.mu.RUnlock()
		if latest != nil {
			return latest, nil
		}
	}

	// Also check policy-embedded overrides if present and still valid (`valid_until > now`).
	for i := len(policyOverrides) - 1; i >= 0; i-- {
		po := policyOverrides[i]
		if po.MemberId == memberID && now.Before(po.ValidUntil) {
			return &MemberOverride{
				ID:         uuid.Nil,
				MemberID:   po.MemberId,
				DownBps:    po.DownBps,
				UpBps:      po.UpBps,
				Priority:   po.Priority,
				ValidFrom:  now,
				ValidUntil: po.ValidUntil.UTC(),
				Reason:     po.Reason,
				ActorID:    po.ActorId,
			}, nil
		}
	}
	return nil, nil
}

func (e *Engine) loadPreviousState(ctx context.Context, memberID uuid.UUID, periodID string) (*EvaluationResult, decstr.U64, error) {
	if e.db == nil {
		e.mu.RLock()
		defer e.mu.RUnlock()
		maxGen := e.memMaxGen[memberID]
		prev, ok := e.memStates[stateKey(memberID, periodID)]
		if !ok {
			return nil, maxGen, nil
		}
		cp := *prev
		return &cp, maxGen, nil
	}

	var maxGenStr string
	err := e.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(generation), 0)::text
		FROM fup_member_states
		WHERE member_id = $1
	`, memberID).Scan(&maxGenStr)
	if err != nil {
		return nil, 0, fmt.Errorf("fup: query max member generation: %w", err)
	}
	maxGen, err := decstr.Parse(maxGenStr)
	if err != nil {
		return nil, 0, fmt.Errorf("fup: parse max member generation: %w", err)
	}

	var (
		res               EvaluationResult
		modeStr           string
		wouldNameJSON     []byte
		wouldReasonStr    string
		enforcedStateStr  string
		enforcedNameJSON  []byte
		enforcedReasonStr *string
		genStr            string
		totalBytesStr     string
		impactJSON        []byte
	)
	err = e.db.QueryRow(ctx, `
		SELECT id, member_id, period_id, period_start, period_end,
		       policy_revision_id, mode,
		       would_apply_tier_index, would_apply_tier_name,
		       would_apply_down_bps, would_apply_up_bps, would_apply_priority, would_apply_reason,
		       enforced_state, enforced_tier_index, enforced_tier_name,
		       enforced_down_bps, enforced_up_bps, enforced_priority, enforced_reason,
		       generation::text, total_bytes::text, aggregate_as_of,
		       override_id, override_valid_until, impact_summary, last_computed_at
		FROM fup_member_states
		WHERE member_id = $1 AND period_id = $2
	`, memberID, periodID).Scan(
		&res.StateID, &res.MemberID, &res.PeriodID, &res.ValidFrom, &res.ValidUntil,
		&res.PolicyRevisionID, &modeStr,
		&res.WouldApplyTierIndex, &wouldNameJSON,
		&res.WouldApplyDownBps, &res.WouldApplyUpBps, &res.WouldApplyPriority, &wouldReasonStr,
		&enforcedStateStr, &res.EnforcedTierIndex, &enforcedNameJSON,
		&res.EnforcedDownBps, &res.EnforcedUpBps, &res.EnforcedPriority, &enforcedReasonStr,
		&genStr, &totalBytesStr, &res.AggregateAsOf,
		&res.OverrideID, &res.OverrideValidUntil, &impactJSON, &res.ComputedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, maxGen, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("fup: load previous fup_member_states: %w", err)
	}

	res.Mode = api.FairUsePayloadMode(modeStr)
	_ = json.Unmarshal(wouldNameJSON, &res.WouldApplyTierName)
	res.WouldApplyReason = api.MemberRatePolicyReason(wouldReasonStr)
	res.EnforcedState = EnforcedState(enforcedStateStr)
	_ = json.Unmarshal(enforcedNameJSON, &res.EnforcedTierName)
	if enforcedReasonStr != nil {
		r := api.MemberRatePolicyReason(*enforcedReasonStr)
		res.EnforcedReason = &r
	}
	res.Generation, err = decstr.Parse(genStr)
	if err != nil {
		return nil, 0, err
	}
	res.TotalBytes, err = decstr.Parse(totalBytesStr)
	if err != nil {
		return nil, 0, err
	}
	_ = json.Unmarshal(impactJSON, &res.ImpactSummary)

	return &res, maxGen, nil
}

func (e *Engine) persistEvaluation(ctx context.Context, res *EvaluationResult, actor string) error {
	if e.db == nil {
		e.mu.Lock()
		defer e.mu.Unlock()
		cp := *res
		e.memStates[stateKey(res.MemberID, res.PeriodID)] = &cp
		e.memEvaluations = append(e.memEvaluations, &cp)
		if res.Generation > e.memMaxGen[res.MemberID] {
			e.memMaxGen[res.MemberID] = res.Generation
		}
		return nil
	}

	wouldNameBytes, err := json.Marshal(res.WouldApplyTierName)
	if err != nil {
		return fmt.Errorf("fup: marshal would_apply_tier_name: %w", err)
	}
	enforcedNameBytes, err := json.Marshal(res.EnforcedTierName)
	if err != nil {
		return fmt.Errorf("fup: marshal enforced_tier_name: %w", err)
	}
	impactBytes, err := json.Marshal(res.ImpactSummary)
	if err != nil {
		return fmt.Errorf("fup: marshal impact_summary: %w", err)
	}
	var enforcedReasonStr *string
	if res.EnforcedReason != nil {
		s := string(*res.EnforcedReason)
		enforcedReasonStr = &s
	}
	var relaxReasonPtr *string
	if res.RelaxationReason != "" {
		s := res.RelaxationReason
		relaxReasonPtr = &s
	}

	return db.InTx(ctx, e.db, db.TxOptions{}, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO fup_member_states (
				id, member_id, period_id, period_start, period_end,
				policy_revision_id, mode,
				would_apply_tier_index, would_apply_tier_name,
				would_apply_down_bps, would_apply_up_bps, would_apply_priority, would_apply_reason,
				enforced_state, enforced_tier_index, enforced_tier_name,
				enforced_down_bps, enforced_up_bps, enforced_priority, enforced_reason,
				generation, total_bytes, aggregate_as_of,
				override_id, override_valid_until, impact_summary, last_computed_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7,
				$8, $9::jsonb,
				$10, $11, $12, $13,
				$14, $15, $16::jsonb,
				$17, $18, $19, $20,
				$21::numeric, $22::numeric, $23,
				$24, $25, $26::jsonb, $27
			)
			ON CONFLICT (member_id, period_id) DO UPDATE SET
				policy_revision_id = EXCLUDED.policy_revision_id,
				mode = EXCLUDED.mode,
				would_apply_tier_index = EXCLUDED.would_apply_tier_index,
				would_apply_tier_name = EXCLUDED.would_apply_tier_name,
				would_apply_down_bps = EXCLUDED.would_apply_down_bps,
				would_apply_up_bps = EXCLUDED.would_apply_up_bps,
				would_apply_priority = EXCLUDED.would_apply_priority,
				would_apply_reason = EXCLUDED.would_apply_reason,
				enforced_state = EXCLUDED.enforced_state,
				enforced_tier_index = EXCLUDED.enforced_tier_index,
				enforced_tier_name = EXCLUDED.enforced_tier_name,
				enforced_down_bps = EXCLUDED.enforced_down_bps,
				enforced_up_bps = EXCLUDED.enforced_up_bps,
				enforced_priority = EXCLUDED.enforced_priority,
				enforced_reason = EXCLUDED.enforced_reason,
				generation = EXCLUDED.generation,
				total_bytes = EXCLUDED.total_bytes,
				aggregate_as_of = EXCLUDED.aggregate_as_of,
				override_id = EXCLUDED.override_id,
				override_valid_until = EXCLUDED.override_valid_until,
				impact_summary = EXCLUDED.impact_summary,
				last_computed_at = EXCLUDED.last_computed_at
			RETURNING id
		`,
			res.StateID, res.MemberID, res.PeriodID, res.ValidFrom, res.ValidUntil,
			res.PolicyRevisionID, string(res.Mode),
			res.WouldApplyTierIndex, string(wouldNameBytes),
			res.WouldApplyDownBps, res.WouldApplyUpBps, res.WouldApplyPriority, string(res.WouldApplyReason),
			string(res.EnforcedState), res.EnforcedTierIndex, string(enforcedNameBytes),
			res.EnforcedDownBps, res.EnforcedUpBps, res.EnforcedPriority, enforcedReasonStr,
			res.Generation.String(), res.TotalBytes.String(), res.AggregateAsOf,
			res.OverrideID, res.OverrideValidUntil, string(impactBytes), res.ComputedAt,
		).Scan(&res.StateID)
		if err != nil {
			return fmt.Errorf("fup: upsert fup_member_states: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO fup_evaluations (
				id, member_id, period_id, policy_revision_id, mode,
				would_apply_tier_index, would_apply_down_bps, would_apply_up_bps, would_apply_priority, would_apply_reason,
				enforced_state, enforced_tier_index, enforced_down_bps, enforced_up_bps, enforced_priority, enforced_reason,
				generation, generation_incremented, relaxed, relaxation_reason,
				total_bytes, aggregate_as_of, impact_summary, computed_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16,
				$17::numeric, $18, $19, $20,
				$21::numeric, $22, $23::jsonb, $24
			)
		`,
			res.EvaluationID, res.MemberID, res.PeriodID, res.PolicyRevisionID, string(res.Mode),
			res.WouldApplyTierIndex, res.WouldApplyDownBps, res.WouldApplyUpBps, res.WouldApplyPriority, string(res.WouldApplyReason),
			string(res.EnforcedState), res.EnforcedTierIndex, res.EnforcedDownBps, res.EnforcedUpBps, res.EnforcedPriority, enforcedReasonStr,
			res.Generation.String(), res.GenerationIncremented, res.Relaxed, relaxReasonPtr,
			res.TotalBytes.String(), res.AggregateAsOf, string(impactBytes), res.ComputedAt,
		)
		if err != nil {
			return fmt.Errorf("fup: insert fup_evaluations: %w", err)
		}

		actID, actType := fupAuditActor(actor)
		if res.Relaxed {
			relaxAudit, err := json.Marshal(map[string]any{
				"member_id":         res.MemberID.String(),
				"period_id":         res.PeriodID,
				"tier_index":        res.WouldApplyTierIndex,
				"total_bytes":       res.TotalBytes.String(),
				"relaxation_reason": res.RelaxationReason,
			})
			if err != nil {
				return err
			}
			if err := db.AppendAudit(ctx, tx, db.AuditEvent{
				Actor:         actID,
				ActorType:     actType,
				Action:        "fup.tier.relaxed",
				Resource:      string(api.ResourceTypeMember) + ":" + res.MemberID.String(),
				Result:        string(api.AuditEventResultSucceeded),
				SchemaVersion: 1,
				Before:        []byte(`{}`),
				After:         relaxAudit,
			}); err != nil {
				return fmt.Errorf("fup: append relaxation audit event: %w", err)
			}
		}

		if res.GenerationIncremented {
			outboxPayload, err := json.Marshal(map[string]any{
				"member_id":          res.MemberID.String(),
				"period_id":          res.PeriodID,
				"policy_revision_id": res.PolicyRevisionID.String(),
				"generation":         res.Generation.String(),
				"enforced_state":     string(res.EnforcedState),
				"tier_index":         res.EnforcedTierIndex,
				"down_bps":           res.EnforcedDownBps,
				"up_bps":             res.EnforcedUpBps,
				"priority":           res.EnforcedPriority,
				"reason":             enforcedReasonStr,
				"valid_from":         res.ValidFrom.Format(time.RFC3339Nano),
				"valid_until":        res.ValidUntil.Format(time.RFC3339Nano),
			})
			if err != nil {
				return err
			}
			if err := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{
				AggregateID:      res.StateID,
				AggregateType:    "fup_member_state",
				AggregateVersion: res.Generation.String(),
				EventType:        "fup.member_generation_changed",
				SchemaVersion:    1,
				Payload:          outboxPayload,
				AvailableAt:      res.ComputedAt,
			}); err != nil {
				return fmt.Errorf("fup: enqueue outbox event: %w", err)
			}
			if err := db.AppendAudit(ctx, tx, db.AuditEvent{
				Actor:         actID,
				ActorType:     actType,
				Action:        "fup.generation.issued",
				Resource:      string(api.ResourceTypeMember) + ":" + res.MemberID.String(),
				Result:        string(api.AuditEventResultSucceeded),
				SchemaVersion: 1,
				Before:        []byte(`{}`),
				After:         outboxPayload,
			}); err != nil {
				return fmt.Errorf("fup: append generation audit event: %w", err)
			}
		}
		return nil
	})
}

// Default PostgreSQL implementations of UsageReader, MemberPolicyReader, and NodeAckReader.

const fupSystemActorUUID = "00000000-0000-0000-0000-000000000003"

func fupAuditActor(actor string) (string, string) {
	trimmed := strings.TrimSpace(actor)
	if id, err := uuid.Parse(trimmed); err == nil {
		if id == uuid.Nil || id.String() == fupSystemActorUUID {
			return fupSystemActorUUID, string(api.AuditEventActorTypeSystem)
		}
		return id.String(), string(api.AuditEventActorTypeAdmin)
	}
	return fupSystemActorUUID, string(api.AuditEventActorTypeSystem)
}

type pgUsageReader struct {
	db *pgxpool.Pool
}

func (r *pgUsageReader) GetMemberPeriodUsage(ctx context.Context, memberID uuid.UUID, periodID string) (UsageSnapshot, error) {
	pStart, pEnd, err := PeriodBoundsForUTCMonth(periodID)
	if err != nil {
		return UsageSnapshot{}, err
	}
	snap := UsageSnapshot{
		MemberID:        memberID,
		PeriodID:        periodID,
		PeriodStart:     pStart,
		PeriodEnd:       pEnd,
		TotalBytes:      0,
		AggregateAsOf:   time.Now().UTC(),
		MeteringVersion: string(api.MeteringVersionForwardAdmissionV1),
	}

	var totalStr, genStr string
	var epoch uuid.UUID
	var aggAsOf time.Time
	var meteringVer string
	err = r.db.QueryRow(ctx, `
		SELECT r.total_bytes::text,
		       r.aggregate_as_of,
		       r.metering_version,
		       r.period_start,
		       r.period_end,
		       COALESCE((SELECT current_epoch FROM usage_ledger_state LIMIT 1), '00000000-0000-0000-0000-000000000000'::uuid),
		       r.projection_generation::text
		FROM usage_rollups r
		JOIN usage_projection_generations g
		  ON g.projection_name = 'member_usage'
		 AND g.state = 'active'
		 AND g.generation = r.projection_generation
		WHERE r.member_id = $1 AND r.period = $2 AND r.granularity = 'month'
		LIMIT 1
	`, memberID, periodID).Scan(&totalStr, &aggAsOf, &meteringVer, &snap.PeriodStart, &snap.PeriodEnd, &epoch, &genStr)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return UsageSnapshot{}, fmt.Errorf("fup: query usage_rollups: %w", err)
	}
	if err == nil {
		u, parseErr := decstr.Parse(totalStr)
		if parseErr != nil {
			return UsageSnapshot{}, fmt.Errorf("fup: parse usage_rollups total_bytes=%q: %w", totalStr, parseErr)
		}
		gen, parseGenErr := decstr.Parse(genStr)
		if parseGenErr != nil {
			return UsageSnapshot{}, fmt.Errorf("fup: parse projection_generation=%q: %w", genStr, parseGenErr)
		}
		snap.TotalBytes = u
		snap.AggregateAsOf = aggAsOf.UTC()
		snap.MeteringVersion = meteringVer
		snap.Watermark = &LedgerWatermark{
			LedgerEpoch:          epoch,
			ProjectionGeneration: gen,
			AggregateAsOf:        aggAsOf.UTC(),
		}
	}

	// Check if there is any negative adjustment in `usage_adjustments` for this member and period (§11.1).
	var negReason string
	err = r.db.QueryRow(ctx, `
		SELECT reason
		FROM usage_adjustments
		WHERE member_id = $1 AND period = $2 AND (upload_bytes + download_bytes) < 0
		ORDER BY created_at DESC
		LIMIT 1
	`, memberID, periodID).Scan(&negReason)
	if err == nil {
		snap.HasNegativeAdjustment = true
		snap.AdjustmentReason = negReason
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return UsageSnapshot{}, fmt.Errorf("fup: query usage_adjustments: %w", err)
	}

	return snap, nil
}

type pgMemberPolicyReader struct {
	db *pgxpool.Pool
}

func (r *pgMemberPolicyReader) GetMemberFairUsePolicy(ctx context.Context, memberID uuid.UUID, _ time.Time) (MemberPolicyContext, error) {
	var revID uuid.UUID
	var rawPayload []byte
	err := r.db.QueryRow(ctx, `
		SELECT pr.id, pr.payload
		FROM policy_definitions pd
		JOIN policy_revisions pr ON pr.id = pd.active_revision_id
		WHERE pd.type = 'fair_use'
		ORDER BY pd.updated_at DESC
		LIMIT 1
	`).Scan(&revID, &rawPayload)
	if err != nil {
		return MemberPolicyContext{}, fmt.Errorf("fup: query active fair_use policy revision: %w", err)
	}
	var payload api.FairUsePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return MemberPolicyContext{}, fmt.Errorf("fup: decode fair_use payload: %w", err)
	}
	return MemberPolicyContext{
		MemberID:          memberID,
		FairUseRevisionID: revID,
		FairUsePolicy:     payload,
	}, nil
}

type pgNodeAckReader struct {
	db *pgxpool.Pool
}

func (r *pgNodeAckReader) GetMemberNodeAckStatus(ctx context.Context, _ uuid.UUID, generation decstr.U64) (NodeAckStatus, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(
		           (SELECT no.checks->'applied_policy_refs'->>'fup_generation'
		            FROM node_observations no
		            WHERE no.node_id = n.id
		              AND no.safety_status = 'passed'
		              AND no.control_status = 'online'
		            ORDER BY no.received_at DESC
		            LIMIT 1),
		           ''
		       ) AS obs_fup_gen
		FROM nodes n
		WHERE n.lifecycle_state = 'registered'
	`)
	if err != nil {
		return NodeAckStatus{}, fmt.Errorf("fup: query registered nodes ack status: %w", err)
	}
	defer rows.Close()

	var status NodeAckStatus
	for rows.Next() {
		var obsFupGenStr string
		if err := rows.Scan(&obsFupGenStr); err != nil {
			return NodeAckStatus{}, err
		}
		status.TotalServingNodes++
		acked := false
		// Fail-closed: only an online node with safety_status=passed and an explicit
		// applied_policy_refs.fup_generation >= generation counts as ACKed (§11.2, FUP03).
		if obsFupGenStr != "" {
			if gen, err := decstr.Parse(obsFupGenStr); err == nil && gen >= generation {
				acked = true
			}
		}
		if acked {
			status.AckedNodes++
		}
	}
	if err := rows.Err(); err != nil {
		return NodeAckStatus{}, err
	}
	status.PendingNodes = status.TotalServingNodes - status.AckedNodes
	return status, nil
}

func copyInt64Ptr(p *int64) *int64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyReasonPtr(p *api.MemberRatePolicyReason) *api.MemberRatePolicyReason {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func equalIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func equalReasonPtr(a, b *api.MemberRatePolicyReason) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

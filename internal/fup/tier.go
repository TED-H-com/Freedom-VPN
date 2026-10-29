// Package fup implements the Freedom VPN Fair-Use Policy (FUP) evaluation engine,
// effective rate calculation, shadow/enforce state machine, admin overrides,
// and coalesced background worker (§4 FUP01–FUP04, §11.1–§11.3, §11.5, AT18, AT19).
package fup

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// ErrInvalidTiers indicates that the fair-use tier table is empty or malformed.
var ErrInvalidTiers = errors.New("fup: invalid or empty fair_use tiers")

// RateScopeDirection identifies the traffic direction in a rate_scope tuple (§11.3).
type RateScopeDirection string

const (
	// DirectionDown is the download (exit-to-member) direction.
	DirectionDown RateScopeDirection = "down"
	// DirectionUp is the upload (member-to-exit) direction.
	DirectionUp RateScopeDirection = "up"
)

// RateScope represents the per-node per-member directional rate scope tuple
// rate_scope = (member_id, node_id, direction) defined in §11.3.
// All authorized devices of a member using the same exit node share a single
// HTB/fq_codel class per direction on that node. Two different exit nodes have
// distinct RateScope values and each provide their own rate R.
type RateScope struct {
	MemberID  uuid.UUID
	NodeID    uuid.UUID
	Direction RateScopeDirection
}

// String formats the canonical rate_scope tuple `(member_id,node_id,direction)` (§11.3).
func (s RateScope) String() string {
	return fmt.Sprintf("(%s,%s,%s)", s.MemberID.String(), s.NodeID.String(), string(s.Direction))
}

// RateScopeKey returns the canonical rate_scope key for `(member_id, node_id, direction)` (§11.3).
func RateScopeKey(memberID, nodeID uuid.UUID, direction RateScopeDirection) string {
	return RateScope{
		MemberID:  memberID,
		NodeID:    nodeID,
		Direction: direction,
	}.String()
}

// SelectTier selects the fair-use tier for a member's period usage total U (`totalBytes`)
// using the half-open interval rule `threshold <= U < next_threshold` (§11.1).
// Equality (`U == threshold`) enters the new tier immediately.
// All comparisons operate on uint64 (`decstr.U64`), preserving exact integer precision
// above 2^53 (e.g., 2^53 + 1 = 9007199254740993) up to 2^64 - 1.
func SelectTier(tiers []api.FairUseTier, totalBytes decstr.U64) (int, api.FairUseTier, error) {
	if len(tiers) == 0 {
		return 0, api.FairUseTier{}, ErrInvalidTiers
	}
	if uint64(tiers[0].ThresholdBytes) != 0 {
		return 0, api.FairUseTier{}, fmt.Errorf("%w: tier[0].threshold_bytes must be 0, got %s", ErrInvalidTiers, tiers[0].ThresholdBytes.String())
	}

	u := uint64(totalBytes)
	selectedIdx := 0
	for i := 0; i < len(tiers); i++ {
		thresh := uint64(tiers[i].ThresholdBytes)
		if i > 0 && thresh <= uint64(tiers[i-1].ThresholdBytes) {
			return 0, api.FairUseTier{}, fmt.Errorf("%w: tier[%d].threshold_bytes (%d) must be > tier[%d].threshold_bytes (%d)",
				ErrInvalidTiers, i, thresh, i-1, uint64(tiers[i-1].ThresholdBytes))
		}
		if u >= thresh {
			selectedIdx = i
		} else {
			break
		}
	}
	return selectedIdx, tiers[selectedIdx], nil
}

// EffectiveRateBps computes `effective_rate = min(plan_rate, fair_use_tier_rate, node_member_rate_cap)`
// for a single direction in bits per second (§11.3, AT18).
// A nil pointer means that component imposes no rate limit; if all three are nil, nil is returned.
func EffectiveRateBps(planRateBps, tierRateBps, nodeCapBps *int64) *int64 {
	var minVal *int64
	for _, candidate := range []*int64{planRateBps, tierRateBps, nodeCapBps} {
		if candidate == nil {
			continue
		}
		v := *candidate
		if minVal == nil || v < *minVal {
			copied := v
			minVal = &copied
		}
	}
	return minVal
}

// EffectiveRatePair computes directional effective rates `min(plan_rate, fair_use_tier_rate, node_member_rate_cap)`
// for both download and upload directions (§11.3, AT18).
func EffectiveRatePair(planRate, tierRate, nodeCap *api.RatePair) *api.RatePair {
	var planDown, planUp *int64
	if planRate != nil {
		d, u := planRate.DownBps, planRate.UpBps
		planDown, planUp = &d, &u
	}
	var tierDown, tierUp *int64
	if tierRate != nil {
		d, u := tierRate.DownBps, tierRate.UpBps
		tierDown, tierUp = &d, &u
	}
	var capDown, capUp *int64
	if nodeCap != nil {
		d, u := nodeCap.DownBps, nodeCap.UpBps
		capDown, capUp = &d, &u
	}

	down := EffectiveRateBps(planDown, tierDown, capDown)
	up := EffectiveRateBps(planUp, tierUp, capUp)
	if down == nil && up == nil {
		return nil
	}
	var res api.RatePair
	if down != nil {
		res.DownBps = *down
	}
	if up != nil {
		res.UpBps = *up
	}
	return &res
}

// BuildNodeMemberRatePolicy constructs the per-node `api.MemberRatePolicy` for a specific exit node (§11.2, §11.3).
// Because `rate_scope = (member_id, node_id, direction)`, each exit node computes its own directional
// `DownBps` and `UpBps` using `min(plan_rate, fair_use_tier_rate, node_member_rate_cap)` and applies
// that rate R independently on that node.
func BuildNodeMemberRatePolicy(eval *EvaluationResult, planRate, nodeCap *api.RatePair) api.MemberRatePolicy {
	var planDown, planUp *int64
	if planRate != nil {
		d, u := planRate.DownBps, planRate.UpBps
		planDown, planUp = &d, &u
	}
	var fupDown, fupUp *int64
	if eval.Mode == api.FairUsePayloadModeEnforce {
		fupDown = eval.EnforcedDownBps
		fupUp = eval.EnforcedUpBps
	}
	var capDown, capUp *int64
	if nodeCap != nil {
		d, u := nodeCap.DownBps, nodeCap.UpBps
		capDown, capUp = &d, &u
	}

	downBps := EffectiveRateBps(planDown, fupDown, capDown)
	upBps := EffectiveRateBps(planUp, fupUp, capUp)

	mode := api.MemberRatePolicyModeShadow
	if eval.Mode == api.FairUsePayloadModeEnforce {
		mode = api.MemberRatePolicyModeEnforce
	}

	reason := api.MemberRatePolicyReasonNormal
	if eval.EnforcedReason != nil {
		reason = *eval.EnforcedReason
	}
	priority := 0
	if eval.EnforcedPriority != nil {
		priority = *eval.EnforcedPriority
	}

	return api.MemberRatePolicy{
		MemberId:   eval.MemberID,
		PeriodId:   eval.PeriodID,
		Mode:       mode,
		Generation: eval.Generation,
		TierIndex:  eval.EnforcedTierIndex,
		Reason:     reason,
		Priority:   priority,
		DownBps:    presence.FromPointer(downBps),
		UpBps:      presence.FromPointer(upBps),
		ValidFrom:  eval.ValidFrom,
		ValidUntil: eval.ValidUntil,
	}
}

// PeriodBoundsForUTCMonth returns the half-open UTC interval `[start, end)` for a `YYYY-MM` period ID (§10.4).
func PeriodBoundsForUTCMonth(periodID string) (time.Time, time.Time, error) {
	t, err := time.Parse("2006-01", periodID)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("fup: invalid period_id %q: %w", periodID, err)
	}
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start, end, nil
}

// CurrentUTCMonthPeriodID formats `now` in UTC as a `YYYY-MM` period ID (§10.4).
func CurrentUTCMonthPeriodID(now time.Time) string {
	return now.UTC().Format("2006-01")
}

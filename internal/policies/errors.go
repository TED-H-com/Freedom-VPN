// Package policies implements the versioned policy store, JSON Schema 2020-12
// validation, type-specific semantic validators, fixed isolation baseline
// guards, and atomic policy activation (§11.1, §11.5, §12.4, §12.5, §13.2,
// §14.2, §14.5, §15.3, AT18, AT19, AT27).
package policies

import (
	"errors"
	"fmt"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// RuleReason identifies a specific semantic or baseline validation failure so
// tests and callers can assert exact failure causes while HTTP handlers render
// standard apierr envelopes via errors.As(err, &apiErr).
type RuleReason string

const (
	ReasonSchemaInvalid              RuleReason = "schema_invalid"
	ReasonCanonicalJSONInvalid       RuleReason = "canonical_json_invalid"
	ReasonUnknownPolicyType          RuleReason = "unknown_policy_type"
	ReasonSchemaVersionInvalid       RuleReason = "schema_version_invalid"
	ReasonIsolationBaselineViolation RuleReason = "isolation_baseline_violation"

	// Fair-use rules (§11.1)
	ReasonFairUseModeInvalid            RuleReason = "fair_use_mode_invalid"
	ReasonFairUseEffectiveAtInvalid     RuleReason = "fair_use_effective_at_invalid"
	ReasonFairUseEffectivePeriodInvalid RuleReason = "fair_use_effective_period_invalid"
	ReasonFairUseAccountingBasisInvalid RuleReason = "fair_use_accounting_basis_invalid"
	ReasonFairUseEmptyTiers             RuleReason = "fair_use_empty_tiers"
	ReasonFairUseTierZeroThreshold      RuleReason = "fair_use_tier_zero_threshold_not_zero"
	ReasonFairUseThresholdOrder         RuleReason = "fair_use_threshold_not_strictly_increasing"
	ReasonFairUseNegativeRate           RuleReason = "fair_use_negative_rate"
	ReasonFairUseMissingDirection       RuleReason = "fair_use_missing_direction"
	ReasonFairUseRateOrder              RuleReason = "fair_use_rate_increasing_across_tiers"
	ReasonFairUseBelowMinimumRate       RuleReason = "fair_use_rate_below_minimum_rate"
	ReasonFairUsePriorityBounds         RuleReason = "fair_use_priority_out_of_bounds"
	ReasonFairUsePriorityOrder          RuleReason = "fair_use_priority_decreasing_across_tiers"
	ReasonFairUseDisplayNameEmpty       RuleReason = "fair_use_display_name_empty"
	ReasonFairUseDuplicateOverride      RuleReason = "fair_use_duplicate_override_member"
	ReasonFairUseExpiredOverride        RuleReason = "fair_use_expired_override"
	ReasonFairUseInvalidOverride        RuleReason = "fair_use_invalid_override"

	// Plan rules (§14.2, §15.2)
	ReasonPlanDeviceLimitInvalid RuleReason = "plan_device_limit_invalid"
	ReasonPlanExitGroupsEmpty    RuleReason = "plan_exit_groups_empty"
	ReasonPlanExitGroupDuplicate RuleReason = "plan_exit_group_duplicate"
	ReasonPlanUnknownExitGroup   RuleReason = "plan_unknown_exit_group"
	ReasonPlanPriceInvalid       RuleReason = "plan_price_invalid"
	ReasonPlanRateLimitInvalid   RuleReason = "plan_rate_limit_invalid"
	ReasonPlanMissingFairUseRef  RuleReason = "plan_missing_fair_use_revision"
	ReasonPlanMissingDNSProfile  RuleReason = "plan_missing_dns_profile_revision"
	ReasonPlanDisplayNameEmpty   RuleReason = "plan_display_name_empty"

	// DNS / Domain rule set / Threat feed rules (§12.3, §12.4, §12.5, §14.2)
	ReasonDNSDuplicateRuleID         RuleReason = "dns_duplicate_rule_id"
	ReasonDNSSameLayerConflict       RuleReason = "dns_same_layer_conflict"
	ReasonDNSDomainNotCanonical      RuleReason = "dns_domain_not_canonical"
	ReasonDNSInvalidRevision         RuleReason = "dns_invalid_revision"
	ReasonDNSProfileMismatch         RuleReason = "dns_rule_profile_mismatch"
	ReasonDNSUnknownRuleSetRevision  RuleReason = "dns_unknown_rule_set_revision"
	ReasonDNSUnknownThreatFeedRev    RuleReason = "dns_unknown_threat_feed_revision"
	ReasonDNSInvalidProfile          RuleReason = "dns_invalid_profile"
	ReasonThreatFeedInvalidURL       RuleReason = "threat_feed_invalid_url"
	ReasonThreatFeedUnconfirmedBasis RuleReason = "threat_feed_unconfirmed_commercial_basis"
	ReasonThreatFeedInvalidValidity  RuleReason = "threat_feed_invalid_validity_window"
	ReasonThreatFeedInvalidDigest    RuleReason = "threat_feed_invalid_digest"
	ReasonThreatFeedInvalidSize      RuleReason = "threat_feed_invalid_max_size"

	// Release support rules (§13.2, AT27)
	ReasonReleaseServingEmpty         RuleReason = "release_serving_empty"
	ReasonReleaseApprovedNotInServing RuleReason = "release_approved_not_subset_of_serving"
	ReasonReleaseRevokedOverlap       RuleReason = "release_revoked_overlaps_active_set"
	ReasonReleaseDuplicateID          RuleReason = "release_duplicate_id"
	ReasonReleaseInvalidID            RuleReason = "release_invalid_id"
	ReasonReleaseUnknownID            RuleReason = "release_unknown_id"
)

// ValidationError wraps an *apierr.Error with structured field and reason metadata.
type ValidationError struct {
	Field   string
	Reason  RuleReason
	Message string
	APIErr  *apierr.Error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s (%s at %s): %s", e.APIErr.Code, e.Reason, e.Field, e.Message)
	}
	return fmt.Sprintf("%s (%s): %s", e.APIErr.Code, e.Reason, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.APIErr
}

func newValidationError(field string, reason RuleReason, message string) error {
	fe := apierr.FieldError{
		Field:  field,
		Reason: string(reason),
	}
	ae := apierr.Validation(message, fe).WithDetail("reason", string(reason))
	return &ValidationError{
		Field:   field,
		Reason:  reason,
		Message: message,
		APIErr:  ae,
	}
}

func newConflictError(field string, reason RuleReason, message string) error {
	fe := apierr.FieldError{
		Field:  field,
		Reason: string(reason),
	}
	ae := apierr.New(apierr.StateConflict, message).
		WithDetail("reason", string(reason)).
		WithDetail("fields", []apierr.FieldError{fe})
	return &ValidationError{
		Field:   field,
		Reason:  reason,
		Message: message,
		APIErr:  ae,
	}
}

// HasReason reports whether err is a ValidationError with the given RuleReason.
func HasReason(err error, reason RuleReason) bool {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Reason == reason
	}
	return false
}

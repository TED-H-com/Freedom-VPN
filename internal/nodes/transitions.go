// Package nodes implements the central node registry, state transitions,
// observation ingestion, staleness detection, and node aggregate views (§5, §14.2, §14.4, §15.3).
package nodes

import (
	"fmt"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// allowedLifecycleTransitions defines the valid state transitions for a node's
// lifecycle_state (§5.1). retired is a terminal tombstone state.
var allowedLifecycleTransitions = map[api.LifecycleState]map[api.LifecycleState]bool{
	api.LifecycleStateEnrolling: {
		api.LifecycleStateRegistered: true,
		api.LifecycleStateRevoked:    true,
		api.LifecycleStateRetired:    true,
	},
	api.LifecycleStateRegistered: {
		api.LifecycleStateRevoked: true,
		api.LifecycleStateRetired: true,
	},
	api.LifecycleStateRevoked: {
		api.LifecycleStateRegistered: true,
		api.LifecycleStateRetired:    true,
	},
	api.LifecycleStateRetired: {},
}

// allowedProvisioningTransitions defines the valid state transitions for a node's
// provisioning_state (§5.1).
var allowedProvisioningTransitions = map[api.ProvisioningState]map[api.ProvisioningState]bool{
	api.ProvisioningStatePending: {
		api.ProvisioningStatePending:  true,
		api.ProvisioningStateApplying: true,
		api.ProvisioningStateReady:    true,
		api.ProvisioningStateFailed:   true,
	},
	api.ProvisioningStateApplying: {
		api.ProvisioningStatePending:  true,
		api.ProvisioningStateApplying: true,
		api.ProvisioningStateReady:    true,
		api.ProvisioningStateFailed:   true,
	},
	api.ProvisioningStateReady: {
		api.ProvisioningStatePending:  true,
		api.ProvisioningStateApplying: true,
		api.ProvisioningStateReady:    true,
		api.ProvisioningStateFailed:   true,
	},
	api.ProvisioningStateFailed: {
		api.ProvisioningStatePending:  true,
		api.ProvisioningStateApplying: true,
		api.ProvisioningStateReady:    true,
		api.ProvisioningStateFailed:   true,
	},
}

// CanTransitionLifecycle reports whether transitioning from -> to is permitted (§5.1).
func CanTransitionLifecycle(from, to api.LifecycleState) bool {
	targets, ok := allowedLifecycleTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

// TransitionLifecycle validates and returns the new LifecycleState, or a
// 409 STATE_CONFLICT API error if the transition is forbidden (§5.1).
func TransitionLifecycle(from, to api.LifecycleState) (api.LifecycleState, error) {
	if !CanTransitionLifecycle(from, to) {
		return from, apierr.Newf(apierr.StateConflict, "invalid node lifecycle transition from %q to %q", from, to).
			WithDetail("from", string(from)).
			WithDetail("to", string(to))
	}
	return to, nil
}

// CanTransitionProvisioning reports whether transitioning from -> to is permitted (§5.1).
func CanTransitionProvisioning(from, to api.ProvisioningState) bool {
	targets, ok := allowedProvisioningTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

// TransitionProvisioning validates and returns the new ProvisioningState, or a
// 409 STATE_CONFLICT API error if the transition is forbidden (§5.1).
func TransitionProvisioning(from, to api.ProvisioningState) (api.ProvisioningState, error) {
	if !CanTransitionProvisioning(from, to) {
		return from, apierr.Newf(apierr.StateConflict, "invalid node provisioning transition from %q to %q", from, to).
			WithDetail("from", string(from)).
			WithDetail("to", string(to))
	}
	return to, nil
}

// ServingReadiness describes all conditions required for a node to be published
// or serve member traffic (§5.1 "可發布服務（Serving）的複合條件").
type ServingReadiness struct {
	LifecycleState            api.LifecycleState
	ProvisioningState         api.ProvisioningState
	PublicationIntent         api.PublicationIntent
	ControlStatus             api.ControlStatus
	SafetyStatus              api.SafetyStatus
	DataHealth                api.DataHealth
	DesiredConfigGeneration   uint64
	AppliedConfigGeneration   uint64
	CurrentReleaseID          *uuid.UUID
	ServingReleaseIDs         map[uuid.UUID]bool
	RequireOnlineAndACKForRun bool
}

// IsServing reports whether a node satisfies the composite serving condition (§5.1):
//   - lifecycle_state == registered
//   - provisioning_state == ready
//   - publication_intent == published
//   - control_status == online
//   - safety_status == passed
//   - data_health in (healthy, degraded)
//   - applied_config_generation >= desired_config_generation (> 0)
func (r ServingReadiness) IsServing() bool {
	if r.PublicationIntent != api.PublicationIntentPublished {
		return false
	}
	if r.AppliedConfigGeneration == 0 || r.AppliedConfigGeneration < r.DesiredConfigGeneration {
		return false
	}
	r.RequireOnlineAndACKForRun = true
	return CheckPublishable(r) == nil
}

// CheckPublishable verifies whether a node meets the baseline requirements to
// accept publication_intent=published (§5.1, §14.2 PUT /admin/nodes/{id}/publication).
// Returns a 422 NODE_NOT_SERVICEABLE error with failed_checks details if not serviceable.
func CheckPublishable(r ServingReadiness) error {
	var failed []string
	if r.LifecycleState != api.LifecycleStateRegistered {
		failed = append(failed, fmt.Sprintf("lifecycle_state=%s (want registered)", r.LifecycleState))
	}
	if r.ProvisioningState != api.ProvisioningStateReady {
		failed = append(failed, fmt.Sprintf("provisioning_state=%s (want ready)", r.ProvisioningState))
	}
	if r.SafetyStatus == api.SafetyStatusFailed || r.SafetyStatus == api.SafetyStatusExpired {
		failed = append(failed, fmt.Sprintf("safety_status=%s", r.SafetyStatus))
	}
	if r.DataHealth == api.DataHealthFailed {
		failed = append(failed, fmt.Sprintf("data_health=%s", r.DataHealth))
	}
	if len(r.ServingReleaseIDs) > 0 {
		if r.CurrentReleaseID == nil || !r.ServingReleaseIDs[*r.CurrentReleaseID] {
			failed = append(failed, "current_release_id not in serving_release_ids")
		}
	}
	if r.RequireOnlineAndACKForRun {
		if r.ControlStatus != api.ControlStatusOnline {
			failed = append(failed, fmt.Sprintf("control_status=%s (want online)", r.ControlStatus))
		}
		if r.SafetyStatus != api.SafetyStatusPassed {
			failed = append(failed, fmt.Sprintf("safety_status=%s (want passed)", r.SafetyStatus))
		}
		if r.AppliedConfigGeneration < r.DesiredConfigGeneration {
			failed = append(failed, fmt.Sprintf("applied_config_generation=%d < desired_config_generation=%d", r.AppliedConfigGeneration, r.DesiredConfigGeneration))
		}
	}
	if len(failed) > 0 {
		return apierr.New(apierr.NodeNotServiceable, "node does not satisfy serving baseline requirements").
			WithDetail("failed_checks", failed)
	}
	return nil
}

package nodes

import (
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestLifecycleTransitions(t *testing.T) {
	allowed := [][2]api.LifecycleState{
		{api.LifecycleStateEnrolling, api.LifecycleStateRegistered},
		{api.LifecycleStateEnrolling, api.LifecycleStateRevoked},
		{api.LifecycleStateEnrolling, api.LifecycleStateRetired},
		{api.LifecycleStateRegistered, api.LifecycleStateRevoked},
		{api.LifecycleStateRegistered, api.LifecycleStateRetired},
		{api.LifecycleStateRevoked, api.LifecycleStateRegistered},
		{api.LifecycleStateRevoked, api.LifecycleStateRetired},
	}

	for _, pair := range allowed {
		if !CanTransitionLifecycle(pair[0], pair[1]) {
			t.Errorf("expected lifecycle transition %s -> %s to be allowed", pair[0], pair[1])
		}
		got, err := TransitionLifecycle(pair[0], pair[1])
		if err != nil || got != pair[1] {
			t.Errorf("TransitionLifecycle(%s, %s) = (%s, %v), want (%s, nil)", pair[0], pair[1], got, err, pair[1])
		}
	}

	forbidden := [][2]api.LifecycleState{
		{api.LifecycleStateRegistered, api.LifecycleStateEnrolling},
		{api.LifecycleStateRevoked, api.LifecycleStateEnrolling},
		{api.LifecycleStateRetired, api.LifecycleStateEnrolling},
		{api.LifecycleStateRetired, api.LifecycleStateRegistered},
		{api.LifecycleStateRetired, api.LifecycleStateRevoked},
	}

	for _, pair := range forbidden {
		if CanTransitionLifecycle(pair[0], pair[1]) {
			t.Errorf("expected lifecycle transition %s -> %s to be forbidden", pair[0], pair[1])
		}
		_, err := TransitionLifecycle(pair[0], pair[1])
		if err == nil {
			t.Errorf("TransitionLifecycle(%s, %s) succeeded, want error", pair[0], pair[1])
		}
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.Code != apierr.StateConflict {
			t.Errorf("TransitionLifecycle(%s, %s) error = %v, want STATE_CONFLICT", pair[0], pair[1], err)
		}
	}
}

func TestProvisioningTransitions(t *testing.T) {
	allowed := [][2]api.ProvisioningState{
		{api.ProvisioningStatePending, api.ProvisioningStateApplying},
		{api.ProvisioningStateApplying, api.ProvisioningStateReady},
		{api.ProvisioningStateReady, api.ProvisioningStateFailed},
		{api.ProvisioningStateFailed, api.ProvisioningStateReady},
		{api.ProvisioningStateReady, api.ProvisioningStateApplying},
	}
	for _, pair := range allowed {
		if !CanTransitionProvisioning(pair[0], pair[1]) {
			t.Errorf("expected provisioning transition %s -> %s to be allowed", pair[0], pair[1])
		}
		if _, err := TransitionProvisioning(pair[0], pair[1]); err != nil {
			t.Errorf("TransitionProvisioning(%s, %s) unexpected error: %v", pair[0], pair[1], err)
		}
	}

	if CanTransitionProvisioning(api.ProvisioningState("invalid"), api.ProvisioningStateReady) {
		t.Errorf("expected invalid -> ready to be forbidden")
	}
}

func TestServingReadinessAndCheckPublishable(t *testing.T) {
	relID := uuid.MustParse("01900000-0000-7000-8000-000000000010")
	valid := ServingReadiness{
		LifecycleState:            api.LifecycleStateRegistered,
		ProvisioningState:         api.ProvisioningStateReady,
		PublicationIntent:         api.PublicationIntentPublished,
		ControlStatus:             api.ControlStatusOnline,
		SafetyStatus:              api.SafetyStatusPassed,
		DataHealth:                api.DataHealthHealthy,
		DesiredConfigGeneration:   3,
		AppliedConfigGeneration:   3,
		CurrentReleaseID:          &relID,
		ServingReleaseIDs:         map[uuid.UUID]bool{relID: true},
		RequireOnlineAndACKForRun: true,
	}
	if !valid.IsServing() {
		t.Fatalf("expected valid readiness to be serving")
	}
	if err := CheckPublishable(valid); err != nil {
		t.Fatalf("CheckPublishable(valid) = %v, want nil", err)
	}

	// Degraded data_health is still publishable/serving as long as not failed.
	degradedHealth := valid
	degradedHealth.DataHealth = api.DataHealthDegraded
	if !degradedHealth.IsServing() {
		t.Fatalf("expected degraded data_health to remain serving")
	}

	cases := []struct {
		name   string
		mutate func(*ServingReadiness)
	}{
		{"unpublished intent", func(r *ServingReadiness) { r.PublicationIntent = api.PublicationIntentUnpublished }},
		{"disabled intent", func(r *ServingReadiness) { r.PublicationIntent = api.PublicationIntentDisabled }},
		{"revoked lifecycle", func(r *ServingReadiness) { r.LifecycleState = api.LifecycleStateRevoked }},
		{"applying provisioning", func(r *ServingReadiness) { r.ProvisioningState = api.ProvisioningStateApplying }},
		{"stale control", func(r *ServingReadiness) { r.ControlStatus = api.ControlStatusStale }},
		{"failed safety", func(r *ServingReadiness) { r.SafetyStatus = api.SafetyStatusFailed }},
		{"failed data_health", func(r *ServingReadiness) { r.DataHealth = api.DataHealthFailed }},
		{"zero applied generation", func(r *ServingReadiness) { r.AppliedConfigGeneration = 0 }},
		{"lagging applied generation", func(r *ServingReadiness) { r.AppliedConfigGeneration = 2 }},
		{"nil release when serving list enforced", func(r *ServingReadiness) { r.CurrentReleaseID = nil }},
		{"unapproved release", func(r *ServingReadiness) {
			other := uuid.MustParse("01900000-0000-7000-8000-000000000099")
			r.CurrentReleaseID = &other
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cp := valid
			tc.mutate(&cp)
			if cp.IsServing() {
				t.Errorf("expected IsServing() = false for %s", tc.name)
			}
			if cp.PublicationIntent == api.PublicationIntentPublished && cp.AppliedConfigGeneration > 0 {
				if err := CheckPublishable(cp); err == nil {
					t.Errorf("expected CheckPublishable to fail for %s", tc.name)
				}
			}
		})
	}
}

func TestParseIfMatchAndFormatETag(t *testing.T) {
	if _, err := ParseIfMatch(""); err == nil {
		t.Fatalf("expected empty If-Match to fail")
	} else {
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.Code != apierr.PreconditionRequired || ae.HTTPStatus() != 428 {
			t.Fatalf("empty If-Match error = %v, want 428 PRECONDITION_REQUIRED", err)
		}
	}

	for _, bad := range []string{"*", `"*"`, `"abc"`, `"0"`, `"-5"`} {
		if _, err := ParseIfMatch(bad); err == nil {
			t.Errorf("ParseIfMatch(%q) succeeded, want error", bad)
		} else {
			var ae *apierr.Error
			if !errors.As(err, &ae) || ae.Code != apierr.ResourceVersionConflict || ae.HTTPStatus() != 412 {
				t.Errorf("ParseIfMatch(%q) error = %v, want 412 RESOURCE_VERSION_CONFLICT", bad, err)
			}
		}
	}

	v, err := ParseIfMatch(`"42"`)
	if err != nil || v != "42" {
		t.Fatalf("ParseIfMatch(\"42\") = (%q, %v), want (\"42\", nil)", v, err)
	}
	if got := FormatETag("42"); got != `"42"` {
		t.Fatalf("FormatETag(\"42\") = %q, want \"42\"", got)
	}
}

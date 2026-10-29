package entitlement

import (
	"context"
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestAPI01_RejectUnknownOperationAndResourceEnums(t *testing.T) {
	actor := Actor{ID: uuid.New(), Role: string(api.AuditEventActorTypeAdmin), RequestID: uuid.New()}
	for _, tc := range []struct {
		name     string
		kind     api.OperationKind
		resource api.ResourceType
	}{
		{"unknown kind", api.OperationKind("unknown"), api.ResourceTypeMember},
		{"unknown resource", api.OperationKindMemberGrantRevoke, api.ResourceType("unknown")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Reject before touching a transaction, including an invalid kind that
			// otherwise has a valid resource and authenticated actor (§14.1).
			if _, err := Operation(context.Background(), nil, actor, tc.kind, tc.resource, uuid.New(), 1); !errors.Is(err, ErrInvalid) {
				t.Fatal(err)
			}
		})
	}
}

func TestAPI01_AggregateResourcesFailClosed(t *testing.T) {
	for _, tc := range []struct {
		aggregate string
		resource  api.ResourceType
	}{
		{string(api.ResourceTypeMember), api.ResourceTypeMember},
		{string(api.ResourceTypeMemberDevice), api.ResourceTypeMemberDevice},
		{"device", api.ResourceTypeMemberDevice},
		{"grant", api.ResourceTypeEntitlementGrant},
		{"billing_session", api.ResourceTypeCheckoutSession},
	} {
		got, err := ResourceType(tc.aggregate)
		if err != nil || got != tc.resource || !got.Valid() {
			t.Fatalf("%q: got %q, %v", tc.aggregate, got, err)
		}
	}
	for _, aggregate := range []string{"", "unknown", "member:" + uuid.NewString()} {
		if _, err := ResourceType(aggregate); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%q: %v", aggregate, err)
		}
	}
}

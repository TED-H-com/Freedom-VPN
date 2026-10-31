// Package httpadapter exposes the member/billing services through the strict API.
package httpadapter

import (
	"context"
	"errors"
	"net/http"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Services is immutable after construction. Missing dependencies fail closed.
// Domain construction supplies services directly to the handler set.
type Services struct {
	Members *members.Service
	Devices *devices.Service
	Billing *billing.Service
}
type contextKey struct{}
type requestContext struct {
	signature string
}

// RequestContext retains only the provider signature, never the Authorization header.
// The strict framework still captures/validates RawBody and enforces every route.
func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature := ""
		if len(r.Header.Values("Stripe-Signature")) == 1 && len(r.Header.Get("Stripe-Signature")) <= 8192 {
			signature = r.Header.Get("Stripe-Signature")
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, requestContext{signature: signature})))
	})
}

type Handlers struct {
	Pool     *pgxpool.Pool
	Services *Services
}

func (h Handlers) services(_ context.Context) (*Services, error) {
	s := h.Services
	if s == nil || s.Members == nil || s.Devices == nil || s.Billing == nil {
		return nil, apierr.New(apierr.DependencyUnavailable, "Member services are not configured.")
	}
	return s, nil
}
func actor(ctx context.Context) entitlement.Actor {
	p, _ := auth.FromContext(ctx)
	requestID, _ := apierr.RequestIDFromContext(ctx)
	id, _ := uuid.Parse(requestID)
	return entitlement.Actor{ID: p.ID, Role: string(p.Role), RequestID: id}
}
func command(ctx context.Context, key uuid.UUID) (entitlement.Command, error) {
	tx, err := httpapi.Transaction(ctx)
	return entitlement.Command{Actor: actor(ctx), Key: key, Tx: tx}, err
}
func domainError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, db.ErrNotFound):
		return apierr.New(apierr.ResourceNotFound, "Resource not found.")
	case errors.Is(err, entitlement.ErrForbidden), errors.Is(err, devices.ErrEntitlement), errors.Is(err, devices.ErrUnmapped):
		return apierr.New(apierr.Forbidden, "Access is unavailable.")
	case errors.Is(err, devices.ErrLimit):
		return apierr.New(apierr.DeviceLimitReached, "Device limit reached.")
	case errors.Is(err, billing.ErrSignature):
		return apierr.New(apierr.WebhookSignatureBad, "Webhook signature is invalid.")
	case errors.Is(err, db.ErrIdempotencyConflict):
		return apierr.New(apierr.IdempotencyConflict, "Idempotency key has a different payload.")
	case errors.Is(err, entitlement.ErrInvalid), errors.Is(err, billing.ErrReturnURL), errors.Is(err, billing.ErrPayload):
		return apierr.New(apierr.ValidationFailed, "Invalid member or billing request.")
	case errors.Is(err, entitlement.ErrConflict):
		return apierr.New(apierr.StateConflict, "Resource state conflicts with this request.")
	case errors.Is(err, billing.ErrBusy), errors.Is(err, billing.ErrProvider), errors.Is(err, billing.ErrAnomaly):
		return apierr.New(apierr.DependencyUnavailable, "Billing is temporarily unavailable.")
	default:
		return err
	}
}
func (h Handlers) accepted(ctx context.Context, id uuid.UUID) (gen.OperationAcceptedJSONResponse, error) {
	p, _ := auth.FromContext(ctx)
	op, err := (operations.Store{Pool: h.Pool}).Get(ctx, id, p)
	return operations.Accepted(op, p.Role), err
}
func result(ctx context.Context, resource api.ResourceType, id uuid.UUID, op *uuid.UUID, status int) error {
	if !resource.Valid() {
		return entitlement.ErrInvalid
	}
	return httpapi.SetResult(ctx, db.IdempotencyResult{ResourceType: string(resource), ResourceID: &id, OperationID: op, ResponseStatus: status})
}

// Deferred visitors execute only after Commands has committed. Provider I/O and
// resource rereads therefore cannot run inside a request transaction (§14.5).
type deferred struct {
	run func(http.ResponseWriter) error
}

func (d deferred) VisitUpdateMemberResponse(w http.ResponseWriter) error          { return d.run(w) }
func (d deferred) VisitCreateMemberGrantResponse(w http.ResponseWriter) error     { return d.run(w) }
func (d deferred) VisitRevokeMemberGrantResponse(w http.ResponseWriter) error     { return d.run(w) }
func (d deferred) VisitRevokeMemberDeviceResponse(w http.ResponseWriter) error    { return d.run(w) }
func (d deferred) VisitAuthorizeMemberDeviceResponse(w http.ResponseWriter) error { return d.run(w) }
func (d deferred) VisitCreateCheckoutSessionResponse(w http.ResponseWriter) error { return d.run(w) }
func (d deferred) VisitCreateBillingPortalSessionResponse(w http.ResponseWriter) error {
	return d.run(w)
}

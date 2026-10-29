package httpadapter

import (
	"context"
	"errors"
	"net/http"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func (h Handlers) UpdateMember(ctx context.Context, r gen.UpdateMemberRequestObject) (gen.UpdateMemberResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	c, err := command(ctx, r.Params.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	var version decstr.U64
	if err = c.Tx.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1 FOR UPDATE`, r.MemberId).Scan(&version); err != nil {
		return nil, domainError(db.MapError(err))
	}
	if err = httpapi.CheckIfMatch(r.Params.IfMatch, version); err != nil {
		return nil, err
	}
	change, err := s.Members.Patch(ctx, c, r.MemberId, &version, *r.Body)
	if err != nil {
		return nil, domainError(err)
	}
	status := 200
	if change.OperationID != nil {
		status = 202
	}
	if err = result(ctx, api.ResourceTypeMember, r.MemberId, change.OperationID, status); err != nil {
		return nil, err
	}
	return h.updateResponse(ctx, r.MemberId, change.OperationID), nil
}
func (h Handlers) updateResponse(ctx context.Context, id uuid.UUID, op *uuid.UUID) deferred {
	return deferred{func(w http.ResponseWriter) error {
		if op != nil {
			a, e := h.accepted(ctx, *op)
			if e != nil {
				return e
			}
			return (gen.UpdateMember202JSONResponse{OperationAcceptedJSONResponse: a}).VisitUpdateMemberResponse(w)
		}
		s, e := h.services(ctx)
		if e != nil {
			return e
		}
		m, e := s.Members.Get(ctx, actor(ctx), id)
		if e != nil {
			return domainError(e)
		}
		body, e := h.detail(ctx, m)
		if e != nil {
			return e
		}
		return (gen.UpdateMember200JSONResponse{Body: body, Headers: gen.UpdateMember200ResponseHeaders{ETag: pointer(httpapi.ETag(m.ResourceVersion))}}).VisitUpdateMemberResponse(w)
	}}
}
func (h Handlers) CreateMemberGrant(ctx context.Context, r gen.CreateMemberGrantRequestObject) (gen.CreateMemberGrantResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	c, err := command(ctx, r.Params.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	change, err := s.Members.Grant(ctx, c, r.MemberId, *r.Body)
	if err != nil {
		return nil, domainError(err)
	}
	if err = result(ctx, api.ResourceTypeEntitlementGrant, change.ResourceID, change.OperationID, 202); err != nil {
		return nil, err
	}
	return h.grantResponse(ctx, *change.OperationID), nil
}
func (h Handlers) grantResponse(ctx context.Context, id uuid.UUID) deferred {
	return deferred{func(w http.ResponseWriter) error {
		a, e := h.accepted(ctx, id)
		if e != nil {
			return e
		}
		return (gen.CreateMemberGrant202JSONResponse{OperationAcceptedJSONResponse: a}).VisitCreateMemberGrantResponse(w)
	}}
}
func (h Handlers) RevokeMemberDevice(ctx context.Context, r gen.RevokeMemberDeviceRequestObject) (gen.RevokeMemberDeviceResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	c, err := command(ctx, r.Params.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	change, err := s.Devices.RevokeDevice(ctx, c, r.DeviceId)
	if err != nil {
		return nil, domainError(err)
	}
	if change.OperationID == nil {
		return nil, apierr.New(apierr.StateConflict, "Device has no revocation operation.")
	}
	if err = result(ctx, api.ResourceTypeMemberDevice, r.DeviceId, change.OperationID, 202); err != nil {
		return nil, err
	}
	return h.revokeResponse(ctx, *change.OperationID), nil
}
func (h Handlers) revokeResponse(ctx context.Context, id uuid.UUID) deferred {
	return deferred{func(w http.ResponseWriter) error {
		a, e := h.accepted(ctx, id)
		if e != nil {
			return e
		}
		return (gen.RevokeMemberDevice202JSONResponse{OperationAcceptedJSONResponse: a}).VisitRevokeMemberDeviceResponse(w)
	}}
}
func (h Handlers) CreateCheckoutSession(ctx context.Context, r gen.CreateCheckoutSessionRequestObject) (gen.CreateCheckoutSessionResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	c, err := command(ctx, r.Params.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	intent, err := s.Billing.ReserveCheckout(ctx, c, r.Body.PlanId, r.Body.ReturnUrl)
	if err != nil {
		return nil, domainError(err)
	}
	if err = result(ctx, api.ResourceTypeCheckoutSession, intent.ID, &intent.OperationID, 201); err != nil {
		return nil, err
	}
	return h.checkoutResponse(ctx, intent.ID), nil
}
func (h Handlers) checkoutResponse(ctx context.Context, id uuid.UUID) deferred {
	return deferred{func(w http.ResponseWriter) error {
		s, e := h.services(ctx)
		if e != nil {
			return e
		}
		hosted, e := s.Billing.CompleteSession(ctx, actor(ctx), id)
		if e != nil {
			return domainError(e)
		}
		w.Header().Set("Cache-Control", "no-store")
		return (gen.CreateCheckoutSession201JSONResponse{Body: api.CheckoutSession{CheckoutSessionId: id, CheckoutUrl: hosted.URL, ExpiresAt: hosted.ExpiresAt, Provider: s.Billing.Provider.Name()}}).VisitCreateCheckoutSessionResponse(w)
	}}
}
func (h Handlers) CreateBillingPortalSession(ctx context.Context, r gen.CreateBillingPortalSessionRequestObject) (gen.CreateBillingPortalSessionResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	c, err := command(ctx, r.Params.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	intent, err := s.Billing.Portal(ctx, c, r.Body.ReturnUrl)
	if err != nil {
		return nil, domainError(err)
	}
	if err = result(ctx, api.ResourceTypeCheckoutSession, intent.ID, nil, 201); err != nil {
		return nil, err
	}
	return h.portalResponse(ctx, intent.ID), nil
}
func (h Handlers) portalResponse(ctx context.Context, id uuid.UUID) deferred {
	return deferred{func(w http.ResponseWriter) error {
		s, e := h.services(ctx)
		if e != nil {
			return e
		}
		// Recheck ownership before any external I/O, including replay (§16.1).
		_, e = s.Billing.SessionResult(ctx, actor(ctx), id)
		if e != nil && !errors.Is(e, billing.ErrBusy) {
			return domainError(e)
		}
		if e != nil {
			if e = s.Billing.FulfillSession(ctx, id); e != nil {
				return domainError(e)
			}
		}
		hosted, e := s.Billing.SessionResult(ctx, actor(ctx), id)
		if e != nil {
			return domainError(e)
		}
		return (gen.CreateBillingPortalSession201JSONResponse{Body: api.BillingPortalSession{PortalUrl: hosted.URL, ExpiresAt: hosted.ExpiresAt}}).VisitCreateBillingPortalSessionResponse(w)
	}}
}
func (h Handlers) Replay(ctx context.Context, name string, r db.IdempotencyRecord) (any, error) {
	if r.ResourceID == nil {
		return nil, apierr.New(apierr.InternalError, "Missing command reference.")
	}
	switch name {
	case "UpdateMember":
		return h.updateResponse(ctx, *r.ResourceID, r.OperationID), nil
	case "CreateBillingPortalSession":
		return h.portalResponse(ctx, *r.ResourceID), nil
	}
	if r.OperationID == nil {
		return nil, apierr.New(apierr.InternalError, "Missing operation reference.")
	}
	if _, err := h.accepted(ctx, *r.OperationID); err != nil {
		return nil, err
	}
	switch name {
	case "CreateMemberGrant":
		return h.grantResponse(ctx, *r.OperationID), nil
	case "RevokeMemberDevice":
		return h.revokeResponse(ctx, *r.OperationID), nil
	case "CreateCheckoutSession":
		return h.checkoutResponse(ctx, *r.ResourceID), nil
	}
	return nil, apierr.New(apierr.InternalError, "Unknown member command.")
}
func (h Handlers) ReceiveProviderWebhook(ctx context.Context, r gen.ReceiveProviderWebhookRequestObject) (gen.ReceiveProviderWebhookResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	if r.Provider != s.Billing.Provider.Name() {
		return nil, apierr.New(apierr.ResourceNotFound, "Unknown payment provider.")
	}
	transport, _ := ctx.Value(contextKey{}).(requestContext)
	ingested, err := s.Billing.Ingest(ctx, httpapi.RawBody(ctx), transport.signature)
	if err != nil {
		return nil, domainError(err)
	}
	return gen.ReceiveProviderWebhook202JSONResponse{Body: api.WebhookAck{Received: true, Duplicate: ingested.Duplicate}}, nil
}

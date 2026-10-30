package httpwire

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type Handlers struct {
	Runtime *Runtime
}

func (h *Handlers) ready(context.Context) error {
	if h.Runtime == nil {
		return errors.New("node runtime was not constructed")
	}
	return nil
}

type tokenKey struct{}

// RequestContext retains enrollment credentials in request-local memory only.
// The domain returns it as Built.Middleware; logging never consumes this value.
func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/agent/enroll" {
			value := r.Header.Get("Authorization")
			if strings.HasPrefix(value, "Enrollment ") {
				r = r.WithContext(context.WithValue(r.Context(), tokenKey{}, strings.TrimPrefix(value, "Enrollment ")))
			}
		}
		next.ServeHTTP(w, r)
	})
}
func ptr[T any](v T) *T { return &v }
func value[T any](v *T) (z T) {
	if v != nil {
		return *v
	}
	return
}
func principal(ctx context.Context) auth.Principal { p, _ := auth.FromContext(ctx); return p }
func adminCommand(ctx context.Context, key uuid.UUID) enrollment.Command {
	return enrollment.Command{Principal: principal(ctx).Scope(), Key: key}
}
func releaseCommand(ctx context.Context, key uuid.UUID) releases.Command {
	return releases.Command{ActorID: principal(ctx).ID, IdempotencyKey: key}
}
func identity(ctx context.Context) fleetupdate.Identity {
	p := principal(ctx)
	return fleetupdate.Identity{NodeID: p.NodeID, IncarnationID: p.IncarnationID}
}
func result(ctx context.Context, kind string, id uuid.UUID, status int, op *uuid.UUID) error {
	if _, e := httpapi.Transaction(ctx); e != nil {
		return nil
	}
	return httpapi.SetResult(ctx, db.IdempotencyResult{ResourceType: kind, ResourceID: &id, ResponseStatus: status, OperationID: op})
}
func (h *Handlers) operation(ctx context.Context, id uuid.UUID) (gen.OperationAcceptedJSONResponse, error) {
	var op api.Operation
	var e error
	if tx, ok := commandtx.Transaction(ctx); ok {
		op, e = nodes.GetOperationTx(ctx, tx, id)
	} else {
		op, e = (operations.Store{Pool: h.Runtime.Pool}).Get(ctx, id, principal(ctx))
	}
	if e != nil {
		return gen.OperationAcceptedJSONResponse{}, e
	}
	return operations.Accepted(op, principal(ctx).Role), nil
}
func (h *Handlers) CreateEnrollment(ctx context.Context, r gen.CreateEnrollmentRequestObject) (gen.CreateEnrollmentResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Enrollment.CreateEnrollment(ctx, adminCommand(ctx, r.Params.IdempotencyKey), *r.Body)
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "enrollment", out.Enrollment.EnrollmentId, 201, nil); e != nil {
		return nil, e
	}
	return gen.CreateEnrollment201JSONResponse{Body: out, Headers: gen.CreateEnrollment201ResponseHeaders{CacheControl: ptr("no-store"), Location: ptr("/api/v1/admin/enrollments/" + out.Enrollment.EnrollmentId.String())}}, nil
}
func (h *Handlers) GetEnrollment(ctx context.Context, r gen.GetEnrollmentRequestObject) (gen.GetEnrollmentResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Enrollment.GetEnrollment(ctx, r.EnrollmentId)
	return gen.GetEnrollment200JSONResponse{Body: out}, e
}
func (h *Handlers) RevokeEnrollment(ctx context.Context, r gen.RevokeEnrollmentRequestObject) (gen.RevokeEnrollmentResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Enrollment.RevokeEnrollment(ctx, adminCommand(ctx, r.Params.IdempotencyKey), r.EnrollmentId, r.Body.Reason)
	if e != nil {
		return nil, e
	}
	e = result(ctx, "enrollment", r.EnrollmentId, 200, nil)
	return gen.RevokeEnrollment200JSONResponse{Body: out}, e
}
func (h *Handlers) EnrollNode(ctx context.Context, r gen.EnrollNodeRequestObject) (gen.EnrollNodeResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	token, _ := ctx.Value(tokenKey{}).(string)
	out, e := h.Runtime.Enrollment.Enroll(ctx, token, *r.Body)
	if e != nil {
		return nil, e
	}
	status := 201
	if out.Replay {
		status = 200
	}
	if e = result(ctx, "node", out.Response.NodeId, status, &out.OperationID); e != nil {
		return nil, e
	}
	if out.Replay {
		return gen.EnrollNode200JSONResponse{Body: out.Response}, nil
	}
	return gen.EnrollNode201JSONResponse{Body: out.Response}, nil
}
func (h *Handlers) ConsumeAgentBootstrap(ctx context.Context, r gen.ConsumeAgentBootstrapRequestObject) (gen.ConsumeAgentBootstrapResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	cert, e := h.Runtime.PKI.Certificate(ctx)
	if e != nil {
		return nil, e
	}
	out, e := h.Runtime.Enrollment.TakeBootstrap(ctx, cert)
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "node", principal(ctx).NodeID, 200, nil); e != nil {
		return nil, e
	}
	return gen.ConsumeAgentBootstrap200JSONResponse{Body: api.AgentBootstrap{ExternalKeyId: out.ExternalKeyID, SetupKey: out.SetupKey, ExpiresAt: out.ExpiresAt}}, nil
}
func (h *Handlers) RenewAgentCertificate(ctx context.Context, r gen.RenewAgentCertificateRequestObject) (gen.RenewAgentCertificateResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	cert, e := h.Runtime.PKI.Certificate(ctx)
	if e != nil {
		return nil, e
	}
	out, e := h.Runtime.PKI.Renew(ctx, cert, *r.Body)
	if e != nil {
		return nil, e
	}
	e = result(ctx, "node", principal(ctx).NodeID, 201, nil)
	return gen.RenewAgentCertificate201JSONResponse{Body: out}, e
}
func (h *Handlers) ListNodes(ctx context.Context, r gen.ListNodesRequestObject) (gen.ListNodesResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	p := r.Params
	out, e := h.Runtime.Nodes.ListNodes(ctx, nodes.ListNodesParams{Cursor: value(p.Cursor), Limit: value(p.Limit), LifecycleState: p.LifecycleState, PublicationIntent: p.PublicationIntent, ControlStatus: p.ControlStatus, Region: p.Region})
	return gen.ListNodes200JSONResponse{Body: out}, e
}
func (h *Handlers) GetNode(ctx context.Context, r gen.GetNodeRequestObject) (gen.GetNodeResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, tag, e := h.Runtime.Nodes.GetNodeWithDNSEvidence(ctx, r.NodeId)
	if e != nil {
		return nil, e
	}
	return gen.GetNode200JSONResponse{Body: out, Headers: gen.GetNode200ResponseHeaders{ETag: &tag}}, nil
}
func (h *Handlers) ListNodeIncarnations(ctx context.Context, r gen.ListNodeIncarnationsRequestObject) (gen.ListNodeIncarnationsResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Nodes.ListNodeIncarnations(ctx, r.NodeId, r.Params)
	if e != nil {
		return nil, e
	}
	return gen.ListNodeIncarnations200JSONResponse{Body: out}, nil
}
func (h *Handlers) ListMemberExits(ctx context.Context, _ gen.ListMemberExitsRequestObject) (gen.ListMemberExitsResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Nodes.ListMemberExits(ctx, principal(ctx).ID)
	return gen.ListMemberExits200JSONResponse{Body: out}, e
}
func requestID(ctx context.Context) *uuid.UUID {
	s, _ := apierr.RequestIDFromContext(ctx)
	id, e := uuid.Parse(s)
	if e != nil {
		return nil
	}
	return &id
}
func (h *Handlers) UpdateNode(ctx context.Context, r gen.UpdateNodeRequestObject) (gen.UpdateNodeResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Nodes.UpdateNode(ctx, nodes.UpdateNodeParams{NodeID: r.NodeId, Principal: principal(ctx).Scope(), IdempotencyKey: r.Params.IdempotencyKey, IfMatch: r.Params.IfMatch, RequestID: requestID(ctx), Patch: *r.Body})
	if e != nil {
		return nil, e
	}
	var op *uuid.UUID
	if out.Operation != nil {
		op = &out.Operation.OperationId
	}
	if e = result(ctx, "node", r.NodeId, out.Status, op); e != nil {
		return nil, e
	}
	if op != nil {
		return gen.UpdateNode202JSONResponse{OperationAcceptedJSONResponse: operations.Accepted(*out.Operation, auth.Admin)}, nil
	}
	return gen.UpdateNode200JSONResponse{Body: *out.Node, Headers: gen.UpdateNode200ResponseHeaders{ETag: &out.ETag}}, nil
}
func (h *Handlers) RevokeNode(ctx context.Context, r gen.RevokeNodeRequestObject) (gen.RevokeNodeResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	v, e := nodes.ParseIfMatch(r.Params.IfMatch)
	if e != nil {
		return nil, e
	}
	version, e := decstr.Parse(v)
	if e != nil {
		return nil, e
	}
	out, e := h.Runtime.Enrollment.RevokeNode(ctx, adminCommand(ctx, r.Params.IdempotencyKey), r.NodeId, &version, r.Body.Reason)
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "node", r.NodeId, 202, &out.OperationID); e != nil {
		return nil, e
	}
	op, e := h.operation(ctx, out.OperationID)
	return gen.RevokeNode202JSONResponse{OperationAcceptedJSONResponse: op}, e
}
func (h *Handlers) RetireNode(ctx context.Context, r gen.RetireNodeRequestObject) (gen.RetireNodeResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	v, e := nodes.ParseIfMatch(r.Params.IfMatch)
	if e != nil {
		return nil, e
	}
	version, e := decstr.Parse(v)
	if e != nil {
		return nil, e
	}
	out, e := h.Runtime.Enrollment.RetireNode(ctx, adminCommand(ctx, r.Params.IdempotencyKey), r.NodeId, &version, "Operator retired node")
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "node", r.NodeId, 202, &out.OperationID); e != nil {
		return nil, e
	}
	op, e := h.operation(ctx, out.OperationID)
	return gen.RetireNode202JSONResponse{OperationAcceptedJSONResponse: op}, e
}
func (h *Handlers) SubmitObservation(ctx context.Context, r gen.SubmitObservationRequestObject) (gen.SubmitObservationResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	i := identity(ctx)
	var out api.ObservationAccepted
	var e error
	if tx, err := httpapi.Transaction(ctx); err == nil {
		out, e = h.Runtime.Nodes.SubmitObservationTx(ctx, tx, i.NodeID, i.IncarnationID, *r.Body)
	} else {
		out, e = h.Runtime.Nodes.SubmitObservation(ctx, i.NodeID, i.IncarnationID, *r.Body)
	}
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "node", i.NodeID, 202, nil); e != nil {
		return nil, e
	}
	return gen.SubmitObservation202JSONResponse{Body: out}, nil
}
func (h *Handlers) ListReleases(ctx context.Context, r gen.ListReleasesRequestObject) (gen.ListReleasesResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Releases.List(ctx, value(r.Params.Cursor), value(r.Params.Limit))
	return gen.ListReleases200JSONResponse{Body: out}, e
}
func (h *Handlers) RegisterRelease(ctx context.Context, r gen.RegisterReleaseRequestObject) (gen.RegisterReleaseResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Releases.Register(ctx, releaseCommand(ctx, r.Params.IdempotencyKey), httpapi.RawBody(ctx))
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "release", out.ReleaseID, out.Status, out.OperationID); e != nil {
		return nil, e
	}
	if out.Status == 202 {
		op, e := h.operation(ctx, *out.OperationID)
		return gen.RegisterRelease202JSONResponse{OperationAcceptedJSONResponse: op}, e
	}
	release, e := h.Runtime.Releases.Get(ctx, out.ReleaseID)
	return gen.RegisterRelease201JSONResponse{Body: release, Headers: gen.RegisterRelease201ResponseHeaders{Location: ptr("/api/v1/admin/releases")}}, e
}
func (h *Handlers) CreateUpdateBatch(ctx context.Context, r gen.CreateUpdateBatchRequestObject) (gen.CreateUpdateBatchResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Fleet.CreateBatch(ctx, releaseCommand(ctx, r.Params.IdempotencyKey), *r.Body)
	if e != nil {
		return nil, e
	}
	e = result(ctx, "update_batch", out.BatchId, 202, &out.OperationId)
	return gen.CreateUpdateBatch202JSONResponse{Body: out, Headers: gen.CreateUpdateBatch202ResponseHeaders{Location: &out.StatusUrl}}, e
}
func (h *Handlers) GetUpdateBatch(ctx context.Context, r gen.GetUpdateBatchRequestObject) (gen.GetUpdateBatchResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Fleet.GetBatch(ctx, r.BatchId, value(r.Params.Cursor), value(r.Params.Limit))
	return gen.GetUpdateBatch200JSONResponse{Body: out}, e
}
func (h *Handlers) RetryUpdateBatch(ctx context.Context, r gen.RetryUpdateBatchRequestObject) (gen.RetryUpdateBatchResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	out, e := h.Runtime.Fleet.Retry(ctx, releaseCommand(ctx, r.Params.IdempotencyKey), r.BatchId, *r.Body)
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "update_batch", r.BatchId, 202, &out.OperationID); e != nil {
		return nil, e
	}
	op, e := h.operation(ctx, out.OperationID)
	return gen.RetryUpdateBatch202JSONResponse{OperationAcceptedJSONResponse: op}, e
}
func (h *Handlers) SubmitUpdateResult(ctx context.Context, r gen.SubmitUpdateResultRequestObject) (gen.SubmitUpdateResultResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	var out api.UpdateResultAck
	var e error
	if tx, err := httpapi.Transaction(ctx); err == nil {
		out, e = h.Runtime.Fleet.SubmitResultTx(ctx, tx, identity(ctx), httpapi.RawBody(ctx))
	} else {
		out, e = h.Runtime.Fleet.SubmitResult(ctx, identity(ctx), httpapi.RawBody(ctx))
	}
	if e != nil {
		return nil, e
	}
	e = result(ctx, "node", principal(ctx).NodeID, 202, nil)
	return gen.SubmitUpdateResult202JSONResponse{Body: out}, e
}

// Compile-time checks catch accidental non-matching generated method signatures.
var _ httpapi.ReplayHandler = (*Handlers)(nil)

func (h *Handlers) Replay(ctx context.Context, name string, ref db.IdempotencyRecord) (any, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	switch name {
	case "UpdateFleetConfig":
		return h.replayFleetConfig(ctx, ref)
	case "SubmitObservation", "SubmitUpdateResult", "RenewServingLease":
		return h.replayAgentReceipt(ctx, name, ref)
	case "CreateAdminGroup":
		out, e := h.Runtime.Nodes.CreatedGroup(ctx, *ref.ResourceID)
		return gen.CreateAdminGroup201JSONResponse{Body: out}, e
	case "CreateEnrollment":
		out, e := h.Runtime.Enrollment.GetEnrollment(ctx, *ref.ResourceID)
		return gen.CreateEnrollment201JSONResponse{Body: api.EnrollmentCreated{Enrollment: out, SecretAvailable: false, Token: presence.NewNullNullable[string]()}}, e
	case "RevokeEnrollment":
		out, e := h.Runtime.Enrollment.GetEnrollment(ctx, *ref.ResourceID)
		return gen.RevokeEnrollment200JSONResponse{Body: out}, e
	case "ConsumeAgentBootstrap":
		return nil, apierr.New(apierr.BootstrapConsumed, "Bootstrap was already delivered.")
	case "EnrollNode":
		var body api.EnrollRequest
		if e := json.Unmarshal(httpapi.RawBody(ctx), &body); e != nil {
			return nil, e
		}
		return h.EnrollNode(ctx, gen.EnrollNodeRequestObject{Body: &body})
	case "RenewAgentCertificate":
		var body api.CertificateRenewRequest
		if e := json.Unmarshal(httpapi.RawBody(ctx), &body); e != nil {
			return nil, e
		}
		return h.RenewAgentCertificate(ctx, gen.RenewAgentCertificateRequestObject{Body: &body})
	case "CreateUpdateBatch":
		out, e := h.Runtime.Fleet.Accepted(ctx, *ref.ResourceID)
		return gen.CreateUpdateBatch202JSONResponse{Body: out, Headers: gen.CreateUpdateBatch202ResponseHeaders{Location: &out.StatusUrl}}, e
	case "UpdateNode":
		if ref.OperationID == nil {
			out, tag, e := h.Runtime.Nodes.GetNode(ctx, *ref.ResourceID)
			return gen.UpdateNode200JSONResponse{Body: out, Headers: gen.UpdateNode200ResponseHeaders{ETag: &tag}}, e
		}
	case "RegisterRelease":
		if ref.ResponseStatus != nil && *ref.ResponseStatus == 201 {
			out, e := h.Runtime.Releases.Get(ctx, *ref.ResourceID)
			return gen.RegisterRelease201JSONResponse{Body: out}, e
		}
	}
	if ref.OperationID == nil {
		return nil, errors.New("missing replay reference")
	}
	op, e := h.operation(ctx, *ref.OperationID)
	if e != nil {
		return nil, e
	}
	switch name {
	case "UpdateNode":
		return gen.UpdateNode202JSONResponse{OperationAcceptedJSONResponse: op}, nil
	case "RevokeNode":
		return gen.RevokeNode202JSONResponse{OperationAcceptedJSONResponse: op}, nil
	case "RetireNode":
		return gen.RetireNode202JSONResponse{OperationAcceptedJSONResponse: op}, nil
	case "RegisterRelease":
		return gen.RegisterRelease202JSONResponse{OperationAcceptedJSONResponse: op}, nil
	case "SetNodePublication":
		return gen.SetNodePublication202JSONResponse{OperationAcceptedJSONResponse: op}, nil
	case "RetryUpdateBatch":
		return gen.RetryUpdateBatch202JSONResponse{OperationAcceptedJSONResponse: op}, nil
	}
	return nil, errors.New("unsupported node command replay")
}

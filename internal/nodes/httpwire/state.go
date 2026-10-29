package httpwire

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

type purposeSigner struct {
	provider signing.Provider
	purpose  envelope.Purpose
}

func (s purposeSigner) Sign(ctx context.Context, p envelope.Purpose, v any) (envelope.Envelope, error) {
	if p != s.purpose {
		return envelope.Envelope{}, signing.ErrUnavailable
	}
	return s.provider.SignEnvelope(ctx, signing.Purpose(p), v)
}

type policySnapshot struct{ value desiredstate.PolicySnapshot }

func (s policySnapshot) ReadPolicySnapshot(context.Context, uuid.UUID, time.Time) (desiredstate.PolicySnapshot, error) {
	return s.value, nil
}

// CompileControlState supports the control-only staging/disabled path. Published
// member forwarding requires the policy owner's BuildNodeMemberRatePolicy hook;
// this checkout lacks that export. Never substitute the legacy rate calculation.
func (r *Runtime) CompileControlState(ctx context.Context, node, incarnation uuid.UUID) (out desiredstate.CompiledDesiredState, err error) {
	err = commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		ctx = commandtx.WithTransaction(ctx, tx)
		var publication api.PublicationIntent
		var current uuid.UUID
		var lifecycle string
		var generationText string
		e := tx.QueryRow(ctx, `SELECT n.lifecycle_state,n.current_incarnation_id,n.desired_config_generation::text,e.publication_intent FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1 FOR UPDATE OF n`, node).Scan(&lifecycle, &current, &generationText, &publication)
		if e != nil {
			return e
		}
		if current != incarnation {
			return apierr.New(apierr.IncarnationMismatch, "Incarnation changed.")
		}
		if lifecycle != "registered" {
			return apierr.New(apierr.IdentityRevoked, "Node identity inactive.")
		}
		if publication == api.PublicationIntentPublished {
			return apierr.New(apierr.DependencyUnavailable, "Member policy compiler is unavailable.")
		}
		work, e := r.Fleet.NodeUpdateIntent(ctx, fleetupdate.Identity{NodeID: node, IncarnationID: incarnation})
		if e != nil {
			return e
		}
		snap, e := desiredstate.NewDBPolicyReader(tx).ReadPolicySnapshot(ctx, node, time.Now().UTC())
		if e != nil {
			return e
		}
		if len(r.Config.DNSResolverIPv4) == 0 {
			return apierr.New(apierr.DependencyUnavailable, "Node DNS configuration required.")
		}
		snap.DNSResolverIPv4 = append([]string(nil), r.Config.DNSResolverIPv4...)
		if e = tx.QueryRow(ctx, `SELECT active_revision_id FROM policy_definitions WHERE type='release_support' AND active_revision_id IS NOT NULL ORDER BY id LIMIT 1`).Scan(&snap.SupportedReleasePolicyID); e != nil {
			return apierr.New(apierr.DependencyUnavailable, "Release support policy required.")
		}
		// Never infer serving authorization from a desired release fallback.
		if e = tx.QueryRow(ctx, `SELECT coalesce(array_agg(id),'{}'::uuid[]) FROM unnest((SELECT supported_release_ids FROM fleet_config)) id WHERE NOT id=ANY((SELECT revoked_release_ids FROM fleet_config)::uuid[])`).Scan(&snap.ServingReleaseIDs); e != nil {
			return e
		}
		if work != nil {
			snap.UpdateIntent = &work.Intent
		}
		var raw []byte
		var tag string
		e = tx.QueryRow(ctx, `SELECT envelope,etag FROM desired_state_envelopes WHERE node_id=$1 AND incarnation_id=$2`, node, incarnation).Scan(&raw, &tag)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if e == nil && work == nil {
			var cached api.DesiredStateEnvelope
			if json.Unmarshal(raw, &cached) == nil && cached.Signed.ExpiresAt.After(time.Now().Add(time.Duration(r.Config.DesiredStateSeconds)*time.Second/2)) && cached.Signed.ConfigGeneration.String() == generationText && cached.Signed.PublicationIntent == publication && cached.Signed.Update.IsNull() {
				a, _ := canonjson.Marshal(cached.Signed.ServingReleaseIds)
				b, _ := canonjson.Marshal(snap.ServingReleaseIDs)
				if string(a) == string(b) {
					out = desiredstate.CompiledDesiredState{Envelope: cached, ETag: tag, Generation: uint64(cached.Signed.ConfigGeneration)}
					return nil
				}
			}
		}
		compiler := desiredstate.NewCompiler(r.Pool, purposeSigner{r.Provider, envelope.PurposeDesiredState}, desiredstate.WithEnvelopeTTL(time.Duration(r.Config.DesiredStateSeconds)*time.Second), desiredstate.WithPolicyReader(policySnapshot{snap}))
		out, e = compiler.CompileTx(ctx, tx, node)
		return e
	})
	return
}
func (h *Handlers) GetDesiredState(ctx context.Context, r gen.GetDesiredStateRequestObject) (gen.GetDesiredStateResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	p := principal(ctx)
	out, e := h.Runtime.CompileControlState(ctx, p.NodeID, p.IncarnationID)
	if e != nil {
		return nil, e
	}
	for _, tag := range strings.Split(value(r.Params.IfNoneMatch), ",") {
		if strings.TrimSpace(tag) == "*" || strings.TrimPrefix(strings.TrimSpace(tag), "W/") == out.ETag {
			return stateNotModified(out.ETag), nil
		}
	}
	return gen.GetDesiredState200JSONResponse{Body: out.Envelope, Headers: gen.GetDesiredState200ResponseHeaders{ETag: &out.ETag, CacheControl: ptr("no-store")}}, nil
}

type stateNotModified string

func (s stateNotModified) VisitGetDesiredStateResponse(w http.ResponseWriter) error {
	w.Header().Set("ETag", string(s))
	w.WriteHeader(http.StatusNotModified)
	return nil
}
func (h *Handlers) RenewServingLease(ctx context.Context, r gen.RenewServingLeaseRequestObject) (gen.RenewServingLeaseResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	if _, ok := commandtx.Transaction(ctx); ok {
		return nil, apierr.New(apierr.DependencyUnavailable, "Transactional lease adapter is unavailable.")
	}
	p := principal(ctx)
	svc := leases.NewService(h.Runtime.Pool, purposeSigner{h.Runtime.Provider, envelope.PurposeLease})
	out, e := svc.Renew(ctx, p.NodeID, p.IncarnationID, *r.Body)
	return gen.RenewServingLease200JSONResponse{Body: out}, e
}

func (h *Handlers) SetNodePublication(ctx context.Context, r gen.SetNodePublicationRequestObject) (gen.SetNodePublicationResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	if r.Body.PublicationIntent == api.PublicationIntentPublished {
		return nil, apierr.New(apierr.DependencyUnavailable, "Member policy compiler is unavailable.")
	}
	tx, e := httpapi.Transaction(ctx)
	if e != nil {
		return nil, e
	}
	var op uuid.UUID
	e = commandtx.InTx(ctx, h.Runtime.Pool, db.TxOptions{}, func(_ pgx.Tx) error {
		var rvText, genText string
		var exit uuid.UUID
		var old api.PublicationIntent
		e := tx.QueryRow(ctx, `SELECT n.resource_version::text,n.desired_config_generation::text,e.id,e.publication_intent FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1 FOR UPDATE OF n,e`, r.NodeId).Scan(&rvText, &genText, &exit, &old)
		if e != nil {
			return e
		}
		rv, e := decstr.Parse(rvText)
		if e != nil {
			return e
		}
		if e = httpapi.CheckIfMatch(r.Params.IfMatch, rv); e != nil {
			return e
		}
		gen, e := decstr.Parse(genText)
		if e != nil {
			return e
		}
		gen, e = gen.Add(1)
		if e != nil {
			return e
		}
		rv, e = rv.Add(1)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE nodes SET desired_config_generation=$2,resource_version=$3 WHERE id=$1`, r.NodeId, gen.String(), rv.String()); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE exits SET publication_intent=$2,resource_version=resource_version+1 WHERE id=$1`, exit, r.Body.PublicationIntent); e != nil {
			return e
		}
		kind := api.OperationKindNodeUnpublish
		if r.Body.PublicationIntent == api.PublicationIntentDisabled {
			kind = api.OperationKindNodeDisable
		}
		rid := requestID(ctx)
		if rid == nil {
			return errors.New("missing request ID")
		}
		op, e = operations.Create(ctx, tx, operations.CreateRequest{Kind: kind, Principal: principal(ctx), Resource: api.ResourceRef{Type: "node", Id: r.NodeId}, RequestID: *rid, ResourceVersion: &rv, DesiredGeneration: &gen, Steps: map[string]operations.Step{"agent_policy": {Required: true}, "route_withdrawal": {Required: true}}})
		if e != nil {
			return e
		}
		payload, e := canonjson.Marshal(map[string]any{"schema_version": 1, "node_id": r.NodeId, "operation_id": op})
		if e != nil {
			return e
		}
		if e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: r.NodeId, AggregateType: "node", AggregateVersion: rv.String(), EventType: "node.routes.withdraw", SchemaVersion: 1, Payload: payload}); e != nil {
			return e
		}
		after, _ := canonjson.Marshal(map[string]any{"publication_intent": r.Body.PublicationIntent, "reason": r.Body.Reason})
		return db.AppendAudit(ctx, tx, db.AuditEvent{Actor: principal(ctx).ID.String(), ActorType: "admin", Action: string(kind), Resource: "node:" + r.NodeId.String(), OperationID: &op, RequestID: rid, SchemaVersion: 1, Before: []byte(`{}`), After: after, Result: "accepted"})
	})
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "node", r.NodeId, 202, &op); e != nil {
		return nil, e
	}
	accepted, e := h.operation(ctx, op)
	return gen.SetNodePublication202JSONResponse{OperationAcceptedJSONResponse: accepted}, e
}

package httpwire

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// DesiredStateCompiler builds the runtime compiler for background execution.
func (r *Runtime) DesiredStateCompiler() *desiredstate.Compiler {
	return r.desiredStateCompiler(purposeSigner{r.Provider, envelope.PurposeDesiredState})
}

func (r *Runtime) desiredStateCompiler(signer desiredstate.Signer, opts ...desiredstate.Option) *desiredstate.Compiler {
	base := []desiredstate.Option{
		desiredstate.WithDocumentInputs(r.ResolveDocumentInputs, nodes.RecordIssuedResolverDocumentTx),
		desiredstate.WithEnvelopeTTL(time.Duration(r.Config.DesiredStateSeconds) * time.Second),
		desiredstate.WithDNSResolvers(r.Config.DNSResolverIPv4, r.Config.DNSResolverIPv6),
	}
	return desiredstate.NewCompiler(r.Pool, signer, append(base, opts...)...)
}

// CompileControlState compiles current DB policy, entitlement and update inputs
// in one transaction. A valid unchanged envelope retains its generation and ETag.
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
		work, e := r.Fleet.NodeUpdateIntent(ctx, fleetupdate.Identity{NodeID: node, IncarnationID: incarnation})
		if e != nil {
			return e
		}
		if len(r.Config.DNSResolverIPv4)+len(r.Config.DNSResolverIPv6) == 0 {
			return apierr.New(apierr.DependencyUnavailable, "Node DNS configuration required.")
		}
		now := time.Now().UTC()
		snap, e := desiredstate.NewDBPolicyReader(tx).ReadPolicySnapshot(ctx, node, now)
		if e != nil {
			return e
		}
		var hasReleasePolicy bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM policy_definitions WHERE type='release_support' AND active_revision_id IS NOT NULL)`).Scan(&hasReleasePolicy); e != nil {
			return e
		}
		if !hasReleasePolicy {
			return apierr.New(apierr.DependencyUnavailable, "Release support policy required.")
		}

		// Only an explicit serving allowlist authorizes a release (§13.2).
		if e = tx.QueryRow(ctx, `SELECT coalesce(array_agg(id),'{}'::uuid[]) FROM unnest((SELECT supported_release_ids FROM fleet_config WHERE supported_release_policy='allowlist')) id WHERE NOT id=ANY((SELECT revoked_release_ids FROM fleet_config)::uuid[])`).Scan(&snap.ServingReleaseIDs); e != nil {
			return e
		}

		if work != nil {
			snap.UpdateIntent = &work.Intent
		}
		var raw []byte
		var tag string
		var refs []uuid.UUID
		e = tx.QueryRow(ctx, `SELECT e.envelope,e.etag,s.policy_refs FROM desired_state_envelopes e JOIN desired_states s ON s.id=e.desired_state_id WHERE e.node_id=$1 AND e.incarnation_id=$2`, node, incarnation).Scan(&raw, &tag, &refs)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var cached api.DesiredStateEnvelope
		var cachedResult *desiredstate.CompiledDesiredState
		// Concrete revision IDs matter even if their maximum generation did not
		// change. Member/FUP/binding inputs are compared after DB compilation below.
		if e == nil && json.Unmarshal(raw, &cached) == nil && cached.Signed.ConfigGeneration.String() == generationText && cached.Signed.ExpiresAt.After(now.Add(time.Duration(r.Config.DesiredStateSeconds)*time.Second/2)) && slices.Equal(refs, snap.PolicyRefs) {
			trust, e := r.Provider.TrustBundle(ctx)
			if e != nil {
				return e
			}
			if _, _, e = envelope.Verify(raw, trust, envelope.PurposeDesiredState, now); e == nil {
				cachedResult = &desiredstate.CompiledDesiredState{Envelope: cached, ETag: tag, Generation: uint64(cached.Signed.ConfigGeneration)}
			}
		}
		signer := cachedStateSigner{signer: purposeSigner{r.Provider, envelope.PurposeDesiredState}, cached: cachedResult}
		compiler := r.desiredStateCompiler(signer,
			desiredstate.WithClock(func() time.Time { return now }),
			desiredstate.WithEntitlementReader(desiredstate.NewDBEntitlementReader(tx)),
			desiredstate.WithPolicyReader(policySnapshot{snap}))
		out, e = compiler.CompileTx(ctx, tx, node)
		if errors.Is(e, errStateUnchanged) {
			out = *cachedResult
			return nil
		}
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
	if e := nodes.CheckPublicationDelivery(ctx, h.Runtime.Pool, out.Envelope.Signed); e != nil {
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
	p := principal(ctx)
	svc := leases.NewService(h.Runtime.Pool, purposeSigner{h.Runtime.Provider, envelope.PurposeLease})
	var out api.ServingLeaseEnvelope
	var e error
	if tx, err := httpapi.Transaction(ctx); err == nil {
		if e = h.observeAgentEndpointTx(ctx, tx); e != nil {
			return nil, e
		}
		out, e = svc.RenewTx(ctx, tx, p.NodeID, p.IncarnationID, *r.Body)
	} else {
		e = db.InTx(ctx, h.Runtime.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if err := h.observeAgentEndpointTx(ctx, tx); err != nil {
				return err
			}
			var err error
			out, err = svc.RenewTx(ctx, tx, p.NodeID, p.IncarnationID, *r.Body)
			return err
		})
	}
	if e != nil {
		return nil, e
	}
	e = result(ctx, "node", p.NodeID, 200, nil)
	return gen.RenewServingLease200JSONResponse{Body: out}, e
}

func (h *Handlers) SetNodePublication(ctx context.Context, r gen.SetNodePublicationRequestObject) (gen.SetNodePublicationResponseObject, error) {
	if e := h.ready(ctx); e != nil {
		return nil, e
	}
	tx, e := httpapi.Transaction(ctx)
	if e != nil {
		return nil, e
	}
	// §5.2/§14.5: the caller owns idempotency and the commit. The Tx path
	// uses only the transaction and clock; compilation and NetBird run later.
	svc := exits.NewService(h.Runtime.Pool, nil, nil)
	var op api.Operation
	e = commandtx.InTx(ctx, h.Runtime.Pool, db.TxOptions{}, func(_ pgx.Tx) error {
		var err error
		op, _, err = svc.SetPublicationTx(ctx, tx, exits.SetPublicationParams{
			NodeID: r.NodeId, Principal: principal(ctx).Scope(), IfMatch: r.Params.IfMatch,
			RequestID: requestID(ctx), Request: *r.Body,
		})
		return err
	})
	if e != nil {
		return nil, e
	}
	if e = result(ctx, "node", r.NodeId, 202, &op.OperationId); e != nil {
		return nil, e
	}
	accepted, e := h.operation(ctx, op.OperationId)
	return gen.SetNodePublication202JSONResponse{OperationAcceptedJSONResponse: accepted}, e
}

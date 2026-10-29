package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	policystore "github.com/TED-H-com/Freedom-VPN/internal/policies"
	usagepkg "github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

func policies() Domain {
	return NewPoliciesDomain(nil)
}

// NewPoliciesDomain constructs the control-plane Domain for the policy catalog
// and Fair-Use evaluation (§11, §12.4, §14.5, D-045). When cfg is nil, settings
// are loaded from Dependencies.Config.PoliciesConfigFile via config.LoadPolicies.
func NewPoliciesDomain(cfg *config.PoliciesConfig) Domain {
	return Domain{
		Name: "policies",
		Build: func(_ context.Context, d Dependencies) (Built, error) {
			if err := config.ValidatePoliciesCentralBounds(d.Config); err != nil {
				return Built{}, err
			}
			var pCfg config.PoliciesConfig
			if cfg != nil {
				if err := cfg.Validate(); err != nil {
					return Built{}, err
				}
				pCfg = *cfg
			} else {
				var err error
				pCfg, err = config.LoadPolicies(d.Config.PoliciesConfigFile)
				if err != nil {
					return Built{}, err
				}
			}
			return Built{
				Handlers: []any{NewPoliciesHandler(d.Pool, d.Logger, pCfg)},
			}, nil
		},
	}
}

// PoliciesHandler serves the central policy catalog (GET /admin/policies,
// POST /admin/policies, POST /admin/policies/{definition_id}/activate) for all
// six policy types (plan, fair_use, dns_profile, domain_rule_set, threat_feed,
// release_support) plus Fair-Use evaluation, status, and admin override operations
// (§11, §12.4, §14.5, §15.4).
type PoliciesHandler struct {
	pool       *pgxpool.Pool
	logger     *slog.Logger
	store      *policystore.Store
	fupEngine  *fup.Engine
	fupWorker  *fup.Worker
	operations operations.Store
}

// NewPoliciesHandler constructs a PoliciesHandler from explicit configuration.
func NewPoliciesHandler(pool *pgxpool.Pool, logger *slog.Logger, pCfg config.PoliciesConfig) *PoliciesHandler {
	store := policystore.NewStore(pool)

	var usageReader fup.UsageReader
	if pool != nil {
		if uSvc, err := usagepkg.NewService(usagepkg.Config{Pool: pool}); err == nil {
			usageReader = fup.NewUsageServiceReader(uSvc, pool)
		}
	}
	engine := fup.NewEngine(pool, usageReader, nil, nil)
	if pCfg.FUPMode != "" {
		engine.SetModeOverride(pCfg.FUPMode)
	}
	fupWorker := fup.NewWorker(engine)

	return &PoliciesHandler{
		pool:       pool,
		logger:     logger,
		store:      store,
		fupEngine:  engine,
		fupWorker:  fupWorker,
		operations: operations.Store{Pool: pool},
	}
}

// Store returns the underlying policies.Store.
func (h *PoliciesHandler) Store() *policystore.Store { return h.store }

// FUPEngine returns the Fair-Use policy evaluation engine.
func (h *PoliciesHandler) FUPEngine() *fup.Engine { return h.fupEngine }

// FUPWorker returns the coalesced Fair-Use evaluation worker.
func (h *PoliciesHandler) FUPWorker() *fup.Worker { return h.fupWorker }

// ListPolicies handles GET /api/v1/admin/policies (§14.2, §14.5).
func (h *PoliciesHandler) ListPolicies(ctx context.Context, r gen.ListPoliciesRequestObject) (gen.ListPoliciesResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Admin {
		return nil, apierr.New(apierr.Forbidden, "Administrator required.")
	}
	limit := 0
	if r.Params.Limit != nil {
		limit = *r.Params.Limit
	}
	cursor := ""
	if r.Params.Cursor != nil {
		cursor = *r.Params.Cursor
	}
	list, err := h.store.ListDefinitions(ctx, policystore.ListDefinitionsFilter{
		Type:   r.Params.Type,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}
	return gen.ListPolicies200JSONResponse{Body: *list}, nil
}

// CreatePolicyRevision handles POST /api/v1/admin/policies (§14.5).
func (h *PoliciesHandler) CreatePolicyRevision(ctx context.Context, r gen.CreatePolicyRevisionRequestObject) (gen.CreatePolicyRevisionResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Admin || !p.MFAVerified {
		return nil, apierr.New(apierr.Forbidden, "Administrator MFA required.")
	}
	if r.Body == nil {
		return nil, apierr.New(apierr.MalformedRequest, "Request body is required.")
	}

	rawBody := httpapi.RawBody(ctx)
	if len(rawBody) == 0 {
		var err error
		rawBody, err = json.Marshal(r.Body)
		if err != nil {
			return nil, apierr.New(apierr.MalformedRequest, "Invalid policy revision payload.")
		}
	}

	var env struct {
		DefinitionID  *uuid.UUID      `json:"definition_id"`
		Code          *string         `json:"code"`
		Type          api.PolicyType  `json:"type"`
		SchemaVersion int             `json:"schema_version"`
		Payload       json.RawMessage `json:"payload"`
		Reason        string          `json:"reason"`
	}
	if err := json.Unmarshal(rawBody, &env); err != nil {
		return nil, apierr.New(apierr.MalformedRequest, "Invalid policy revision JSON.")
	}

	rec, err := h.store.CreateRevision(ctx, policystore.CreateRevisionInput{
		DefinitionID:  env.DefinitionID,
		Code:          env.Code,
		Type:          env.Type,
		SchemaVersion: env.SchemaVersion,
		Payload:       env.Payload,
		Reason:        env.Reason,
		ActorID:       p.ID,
		Principal:     p.Scope(),
	})
	if err != nil {
		return nil, err
	}

	if _, txErr := httpapi.Transaction(ctx); txErr == nil {
		if err := httpapi.SetResult(ctx, db.IdempotencyResult{
			ResourceType:   "policy_revision",
			ResourceID:     &rec.RevisionID,
			ResponseStatus: 201,
		}); err != nil {
			return nil, err
		}
	}

	apiRev, err := rec.ToAPI()
	if err != nil {
		return nil, err
	}
	location := "/api/v1/admin/policies/" + rec.DefinitionID.String()
	return gen.CreatePolicyRevision201JSONResponse{
		Body: apiRev,
		Headers: gen.CreatePolicyRevision201ResponseHeaders{
			Location: &location,
		},
	}, nil
}

// ActivatePolicyRevision handles POST /api/v1/admin/policies/{definition_id}/activate (§14.5).
func (h *PoliciesHandler) ActivatePolicyRevision(ctx context.Context, r gen.ActivatePolicyRevisionRequestObject) (gen.ActivatePolicyRevisionResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Admin || !p.MFAVerified {
		return nil, apierr.New(apierr.Forbidden, "Administrator MFA required.")
	}
	if r.Body == nil {
		return nil, apierr.New(apierr.MalformedRequest, "Request body is required.")
	}

	defID := uuid.UUID(r.DefinitionId)
	revID := uuid.UUID(r.Body.RevisionId)

	res, err := h.store.ActivateRevision(ctx, policystore.ActivateRevisionInput{
		DefinitionID:    defID,
		RevisionID:      revID,
		ExpectedVersion: r.Params.IfMatch,
		EffectiveAt:     r.Body.EffectiveAt,
		Reason:          r.Body.Reason,
		ActorID:         p.ID,
		Principal:       p.Scope(),
	})
	if err != nil {
		return nil, err
	}

	opID := uuid.UUID(res.Operation.OperationId)
	if _, txErr := httpapi.Transaction(ctx); txErr == nil {
		if err := httpapi.SetResult(ctx, db.IdempotencyResult{
			ResourceType:   "policy_definition",
			ResourceID:     &res.DefinitionID,
			OperationID:    &opID,
			ResponseStatus: 202,
		}); err != nil {
			return nil, err
		}
	}

	return gen.ActivatePolicyRevision202JSONResponse{
		OperationAcceptedJSONResponse: operations.Accepted(res.Operation, p.Role),
	}, nil
}

// Replay re-reads persisted policy revisions or activation operations for idempotent retries (§14.5).
func (h *PoliciesHandler) Replay(ctx context.Context, operationID string, record db.IdempotencyRecord) (any, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Admin || !p.MFAVerified {
		return nil, apierr.New(apierr.Forbidden, "Administrator MFA required.")
	}

	switch operationID {
	case "CreatePolicyRevision":
		if record.ResourceID == nil {
			return nil, errors.New("missing policy_revision resource ID on idempotency record")
		}
		rec, err := h.store.GetRevision(ctx, *record.ResourceID)
		if err != nil {
			return nil, err
		}
		apiRev, err := rec.ToAPI()
		if err != nil {
			return nil, err
		}
		location := "/api/v1/admin/policies/" + rec.DefinitionID.String()
		return gen.CreatePolicyRevision201JSONResponse{
			Body: apiRev,
			Headers: gen.CreatePolicyRevision201ResponseHeaders{
				Location: &location,
			},
		}, nil

	case "ActivatePolicyRevision":
		if record.OperationID == nil {
			return nil, errors.New("missing operation ID on policy activation idempotency record")
		}
		op, err := h.operations.Get(ctx, *record.OperationID, p)
		if err != nil {
			return nil, err
		}
		return gen.ActivatePolicyRevision202JSONResponse{
			OperationAcceptedJSONResponse: operations.Accepted(op, p.Role),
		}, nil

	default:
		return nil, fmt.Errorf("policies: unsupported replay operation %q", operationID)
	}
}

// GetMemberFairUseStatus returns the member-visible Fair-Use status for a member and accounting period (§11.6).
func (h *PoliciesHandler) GetMemberFairUseStatus(ctx context.Context, memberID uuid.UUID, periodID string) (api.FairUseStatus, error) {
	return h.fupEngine.GetMemberStatus(ctx, memberID, periodID)
}

// GetAdminMemberFairUseStatus returns the admin Fair-Use diagnostics and per-node ACK status (§11.2, §11.6, FUP03).
func (h *PoliciesHandler) GetAdminMemberFairUseStatus(ctx context.Context, memberID uuid.UUID, periodID string) (fup.AdminMemberFairUseStatus, error) {
	return h.fupEngine.GetAdminMemberStatus(ctx, memberID, periodID)
}

// SetMemberFairUseOverride creates a time-limited admin rate override and recomputes the member's FUP state (§11.1, FUP04).
func (h *PoliciesHandler) SetMemberFairUseOverride(ctx context.Context, in fup.SetOverrideInput) (*fup.MemberOverride, *fup.EvaluationResult, error) {
	return h.fupEngine.SetOverride(ctx, in)
}

// RevokeMemberFairUseOverride revokes an active admin rate override early and recomputes the member's FUP state (§11.1, FUP04).
func (h *PoliciesHandler) RevokeMemberFairUseOverride(ctx context.Context, in fup.RevokeOverrideInput) (*fup.EvaluationResult, error) {
	return h.fupEngine.RevokeOverride(ctx, in)
}

// EvaluateAndBuildMemberRatePolicy evaluates the member's Fair-Use state for periodID
// and compiles the node-facing api.MemberRatePolicy using fup.BuildNodeMemberRatePolicy (§11.3, AT18).
func (h *PoliciesHandler) EvaluateAndBuildMemberRatePolicy(
	ctx context.Context,
	memberID uuid.UUID,
	periodID string,
	planRate, nodeCap *api.RatePair,
) (api.MemberRatePolicy, *fup.EvaluationResult, error) {
	eval, err := h.fupEngine.Compute(ctx, fup.ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
	})
	if err != nil {
		return api.MemberRatePolicy{}, nil, err
	}
	return fup.BuildNodeMemberRatePolicy(eval, planRate, nodeCap), eval, nil
}

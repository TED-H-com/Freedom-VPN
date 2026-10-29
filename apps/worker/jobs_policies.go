package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	usagepkg "github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// FUPCompiler exposes Fair-Use evaluation and node rate-policy compilation to
// the desired-state domain via the worker composition context (§11.2, §11.3, D-026).
type FUPCompiler interface {
	EvaluateAndBuildMemberRatePolicy(ctx context.Context, memberID uuid.UUID, periodID string, planRate, nodeCap *api.RatePair) (api.MemberRatePolicy, *fup.EvaluationResult, error)
	Compute(ctx context.Context, in fup.ComputeInput) (*fup.EvaluationResult, error)
	Flush(ctx context.Context, now time.Time) ([]*fup.EvaluationResult, error)
}

// PoliciesServices holds explicitly configured worker dependencies for the
// policies and Fair-Use domain.
type PoliciesServices struct {
	Pool      *pgxpool.Pool
	Config    config.PoliciesConfig
	FUPEngine *fup.Engine
	FUPWorker *fup.Worker
	Compiler  membersruntime.Compiler
}

// EvaluateAndBuildMemberRatePolicy implements FUPCompiler.
func (s *PoliciesServices) EvaluateAndBuildMemberRatePolicy(
	ctx context.Context,
	memberID uuid.UUID,
	periodID string,
	planRate, nodeCap *api.RatePair,
) (api.MemberRatePolicy, *fup.EvaluationResult, error) {
	eval, err := s.FUPEngine.Compute(ctx, fup.ComputeInput{
		MemberID: memberID,
		PeriodID: periodID,
	})
	if err != nil {
		return api.MemberRatePolicy{}, nil, err
	}
	return fup.BuildNodeMemberRatePolicy(eval, planRate, nodeCap), eval, nil
}

// Compute implements FUPCompiler.
func (s *PoliciesServices) Compute(ctx context.Context, in fup.ComputeInput) (*fup.EvaluationResult, error) {
	return s.FUPEngine.Compute(ctx, in)
}

// Flush implements FUPCompiler.
func (s *PoliciesServices) Flush(ctx context.Context, now time.Time) ([]*fup.EvaluationResult, error) {
	return s.FUPWorker.Flush(ctx, now)
}

type policiesContextKey struct{}
type fupCompilerContextKey struct{}

// WithPolicies supplies immutable policies worker dependencies via Built.Context.
func WithPolicies(ctx context.Context, s *PoliciesServices) context.Context {
	return context.WithValue(ctx, policiesContextKey{}, s)
}

// PoliciesFromContext retrieves the PoliciesServices attached to the worker context.
func PoliciesFromContext(ctx context.Context) *PoliciesServices {
	s, _ := ctx.Value(policiesContextKey{}).( *PoliciesServices)
	return s
}

// WithFUPCompiler attaches a FUPCompiler to the worker composition context so
// the desired-state domain can evaluate member Fair-Use policies without
// ordering domain builds.
func WithFUPCompiler(ctx context.Context, c FUPCompiler) context.Context {
	return context.WithValue(ctx, fupCompilerContextKey{}, c)
}

// FUPCompilerFromContext returns the FUPCompiler from the worker context.
func FUPCompilerFromContext(ctx context.Context) FUPCompiler {
	c, _ := ctx.Value(fupCompilerContextKey{}).(FUPCompiler)
	return c
}

// NewPoliciesServices constructs the policies and Fair-Use worker services from
// explicit configuration and database pool.
func NewPoliciesServices(pool *pgxpool.Pool, pCfg config.PoliciesConfig, compiler membersruntime.Compiler) (*PoliciesServices, error) {
	if err := pCfg.Validate(); err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, errors.New("policies worker database required")
	}
	uSvc, err := usagepkg.NewService(usagepkg.Config{Pool: pool})
	if err != nil {
		return nil, err
	}
	usageReader := fup.NewUsageServiceReader(uSvc, pool)
	engine := fup.NewEngine(pool, usageReader, nil, nil)
	if pCfg.FUPMode != "" {
		engine.SetModeOverride(pCfg.FUPMode)
	}
	fupW := fup.NewWorker(engine)
	if compiler == nil {
		compiler = desiredstate.NewCompiler(pool, nil)
	}
	return &PoliciesServices{
		Pool:      pool,
		Config:    pCfg,
		FUPEngine: engine,
		FUPWorker: fupW,
		Compiler:  compiler,
	}, nil
}

func policies() Domain {
	return NewPoliciesDomain(nil, nil)
}

// NewPoliciesDomain constructs the worker Domain for policy activation and
// Fair-Use evaluation (§11, §14.5). When cfg is nil, settings are validated
// from Dependencies.Config.
func NewPoliciesDomain(cfg *config.PoliciesConfig, compiler membersruntime.Compiler) Domain {
	return Domain{
		Name: "policies",
		Build: func(_ context.Context, d Dependencies) (Built, error) {
			var pCfg config.PoliciesConfig
			if cfg != nil {
				if err := cfg.Validate(); err != nil {
					return Built{}, err
				}
				pCfg = *cfg
			} else {
				var err error
				pCfg, err = config.PoliciesFromConfig(d.Config)
				if err != nil {
					return Built{}, err
				}
			}
			if d.Pool == nil {
				return Built{}, &config.SettingError{Name: "database_url_file", Err: errors.New("policies worker database required")}
			}
			s, err := NewPoliciesServices(d.Pool, pCfg, compiler)
			if err != nil {
				return Built{}, err
			}
			return Built{
				Jobs: PolicyJobs(),
				Context: func(ctx context.Context) context.Context {
					ctx = WithPolicies(ctx, s)
					ctx = WithFUPCompiler(ctx, s)
					if s.Compiler != nil && membersruntime.CompilerFromContext(ctx) == nil {
						ctx = membersruntime.WithCompiler(ctx, s.Compiler)
					}
					return ctx
				},
			}, nil
		},
	}
}

// PolicyJobs returns a fresh registry of policy and Fair-Use outbox handlers.
func PolicyJobs() Registry {
	r := Registry{}
	for _, name := range []string{
		"policy.activated",
		"fup.evaluate",
		"fup.member_generation_changed",
		"fleet.release_support_changed",
	} {
		r.Register(name, policyJob)
	}
	return r
}

func policyJob(ctx context.Context, event db.ClaimedEvent) (string, error) {
	s := PoliciesFromContext(ctx)
	if s == nil || s.Pool == nil || s.FUPWorker == nil {
		return "", errors.New("policies worker dependencies required")
	}
	if event.SchemaVersion != 1 {
		return "", errors.New("unsupported policies event schema")
	}
	switch event.EventType {
	case "policy.activated":
		return handlePolicyActivated(ctx, s, event)
	case "fup.evaluate":
		return handleFUPEvaluate(ctx, s, event)
	case "fup.member_generation_changed":
		return handleFUPMemberGenerationChanged(ctx, s, event)
	case "fleet.release_support_changed":
		var fleetID uuid.UUID
		if err := s.Pool.QueryRow(ctx, `SELECT id FROM fleet_config WHERE id = $1`, event.AggregateID).Scan(&fleetID); err != nil {
			return "", fmt.Errorf("policies worker: read back fleet_config %s: %w", event.AggregateID, err)
		}
		return "fleet_config:" + fleetID.String(), nil
	default:
		return "", errors.New("unowned policy event")
	}
}

func handlePolicyActivated(ctx context.Context, s *PoliciesServices, event db.ClaimedEvent) (string, error) {
	pool := s.Pool
	var payload struct {
		DefinitionID string `json:"definition_id"`
		Type         string `json:"type"`
		RevisionID   string `json:"revision_id"`
		Generation   string `json:"generation"`
		OperationID  string `json:"operation_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "", fmt.Errorf("policies worker: decode policy.activated payload: %w", err)
	}
	defID, err := uuid.Parse(payload.DefinitionID)
	if err != nil {
		return "", fmt.Errorf("policies worker: invalid definition_id: %w", err)
	}
	if _, err := uuid.Parse(payload.RevisionID); err != nil {
		return "", fmt.Errorf("policies worker: invalid revision_id: %w", err)
	}

	// Read back committed policy_definitions state to verify activation evidence.
	var activeRevID *uuid.UUID
	var curGenStr string
	if err := pool.QueryRow(ctx, `
		SELECT active_revision_id, generation::text
		FROM policy_definitions
		WHERE id = $1
	`, defID).Scan(&activeRevID, &curGenStr); err != nil {
		return "", fmt.Errorf("policies worker: read back policy_definitions: %w", err)
	}
	if activeRevID == nil {
		return "", errors.New("policies worker: policy definition has no active revision")
	}

	// When a fair_use or plan policy is activated, run a coalesced FUP sweep across active members.
	if payload.Type == string(api.PolicyTypeFairUse) || payload.Type == string(api.PolicyTypePlan) {
		batchLimit := s.Config.FUPBatchSize
		if batchLimit <= 0 {
			batchLimit = 64
		}
		rows, err := pool.Query(ctx, `SELECT id FROM members WHERE status = 'active' ORDER BY id LIMIT $1`, batchLimit)
		if err != nil {
			return "", fmt.Errorf("policies worker: query active members for FUP sweep: %w", err)
		}
		var memberIDs []uuid.UUID
		for rows.Next() {
			var mid uuid.UUID
			if err := rows.Scan(&mid); err != nil {
				rows.Close()
				return "", err
			}
			memberIDs = append(memberIDs, mid)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return "", err
		}
		if len(memberIDs) > 0 {
			periodID := fup.CurrentUTCMonthPeriodID(time.Now().UTC())
			s.FUPWorker.ScheduleMemberSweep(memberIDs, periodID)
			if _, err := s.FUPWorker.Flush(ctx, time.Now().UTC()); err != nil {
				return "", fmt.Errorf("policies worker: FUP flush on policy activation: %w", err)
			}
		}
	}

	evidenceRef := "policy_definition:" + defID.String()

	// Transition the policy.activate operation's desired_state_compile step to succeeded with read-back evidence.
	if payload.OperationID != "" {
		opID, err := uuid.Parse(payload.OperationID)
		if err == nil && opID != uuid.Nil {
			var genPtr *decstr.U64
			if gen, err := decstr.Parse(payload.Generation); err == nil {
				genPtr = &gen
			}
			err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
				var opStatus string
				if err := tx.QueryRow(ctx, `SELECT status FROM operations WHERE id = $1 FOR UPDATE`, opID).Scan(&opStatus); err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return nil
					}
					return err
				}
				if opStatus == string(api.OperationStatusSucceeded) || opStatus == string(api.OperationStatusSuperseded) {
					return nil
				}
				return operations.Transition(
					ctx,
					tx,
					opID,
					"desired_state_compile",
					api.OperationStepStatusSucceeded,
					&operations.Evidence{
						Reference:  evidenceRef,
						ObservedAt: time.Now().UTC(),
					},
					nil,
					genPtr,
				)
			})
			if err != nil {
				return "", fmt.Errorf("policies worker: transition operation %s: %w", opID, err)
			}
		}
	}

	return evidenceRef, nil
}

func handleFUPEvaluate(ctx context.Context, s *PoliciesServices, event db.ClaimedEvent) (string, error) {
	var payload struct {
		MemberID         string `json:"member_id"`
		PeriodID         string `json:"period_id"`
		AllowRelaxation  bool   `json:"allow_relaxation"`
		RelaxationReason string `json:"relaxation_reason"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "", fmt.Errorf("fup worker: decode fup.evaluate payload: %w", err)
	}
	memberID, err := uuid.Parse(payload.MemberID)
	if err != nil {
		return "", fmt.Errorf("fup worker: invalid member_id: %w", err)
	}
	periodID := payload.PeriodID
	if periodID == "" {
		periodID = fup.CurrentUTCMonthPeriodID(time.Now().UTC())
	}

	if payload.AllowRelaxation {
		s.FUPWorker.NotifyLedgerCorrection(memberID, periodID, payload.RelaxationReason)
	} else {
		s.FUPWorker.NotifyLedgerChange(memberID, periodID)
	}
	results, err := s.FUPWorker.Flush(ctx, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("fup worker: flush evaluation: %w", err)
	}
	if len(results) == 0 {
		return "member:" + memberID.String(), nil
	}
	return "fup_evaluation:" + results[0].EvaluationID.String(), nil
}

func handleFUPMemberGenerationChanged(ctx context.Context, s *PoliciesServices, event db.ClaimedEvent) (string, error) {
	var stateID uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		SELECT id FROM fup_member_states WHERE id = $1
	`, event.AggregateID).Scan(&stateID)
	if err != nil {
		return "", fmt.Errorf("fup worker: read back fup_member_states %s: %w", event.AggregateID, err)
	}
	return "fup_member_state:" + stateID.String(), nil
}

package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type workerPoolContextKey struct{}

// WithPool attaches a *pgxpool.Pool to ctx for worker job handlers.
func WithPool(ctx context.Context, pool *pgxpool.Pool) context.Context {
	if pool == nil {
		return ctx
	}
	return context.WithValue(ctx, workerPoolContextKey{}, pool)
}

func resolveWorkerPool(ctx context.Context, explicit *pgxpool.Pool) (*pgxpool.Pool, func(), error) {
	if explicit != nil {
		return explicit, func() {}, nil
	}
	if p, ok := ctx.Value(workerPoolContextKey{}).(*pgxpool.Pool); ok && p != nil {
		return p, func() {}, nil
	}
	dsn := os.Getenv("FVPN_DATABASE_URL")
	if dsn == "" {
		if dsnFile := os.Getenv("FVPN_DATABASE_URL_FILE"); dsnFile != "" {
			if s, err := config.ReadSecret(dsnFile); err == nil {
				dsn = s
			}
		}
	}
	if dsn == "" {
		return nil, nil, errors.New("worker database pool unavailable")
	}
	pool, err := db.Open(ctx, dsn, db.OpenOptions{})
	if err != nil {
		return nil, nil, err
	}
	return pool, pool.Close, nil
}

// NewFUPWorker constructs a coalesced Fair-Use evaluation worker wired to
// usage.Service.GetMemberPeriodTotal and configured from FVPN_FUP_MODE (§11.1, §11.2).
func NewFUPWorker(pool *pgxpool.Pool) (*fup.Worker, *fup.Engine) {
	pCfg, _ := config.LoadPoliciesFromEnv()
	var usageReader fup.UsageReader
	if pool != nil {
		if uSvc, err := usage.NewService(usage.Config{Pool: pool}); err == nil {
			usageReader = fup.NewUsageServiceReader(uSvc, pool)
		}
	}
	engine := fup.NewEngine(pool, usageReader, nil, nil)
	if pCfg.FUPMode != "" {
		engine.SetModeOverride(pCfg.FUPMode)
	}
	return fup.NewWorker(engine), engine
}

func registerPoliciesJobs(r Registry) {
	RegisterPoliciesJobsWithPool(r, nil)
}

// RegisterPoliciesJobsWithPool registers policy activation and Fair-Use outbox handlers
// bound to an optional explicit database pool.
func RegisterPoliciesJobsWithPool(r Registry, pool *pgxpool.Pool) {
	r.Register("policy.activated", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		p, cleanup, err := resolveWorkerPool(ctx, pool)
		if err != nil {
			return "", err
		}
		defer cleanup()
		return handlePolicyActivated(ctx, p, event)
	})
	r.Register("fup.evaluate", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		p, cleanup, err := resolveWorkerPool(ctx, pool)
		if err != nil {
			return "", err
		}
		defer cleanup()
		return handleFUPEvaluate(ctx, p, event)
	})
	r.Register("fup.member_generation_changed", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		p, cleanup, err := resolveWorkerPool(ctx, pool)
		if err != nil {
			return "", err
		}
		defer cleanup()
		return handleFUPMemberGenerationChanged(ctx, p, event)
	})
	r.Register("fleet.release_support_changed", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		p, cleanup, err := resolveWorkerPool(ctx, pool)
		if err != nil {
			return "", err
		}
		defer cleanup()
		var fleetID uuid.UUID
		if err := p.QueryRow(ctx, `SELECT id FROM fleet_config WHERE id = $1`, event.AggregateID).Scan(&fleetID); err != nil {
			return "", fmt.Errorf("policies worker: read back fleet_config %s: %w", event.AggregateID, err)
		}
		return "fleet_config:" + fleetID.String(), nil
	})
}

func handlePolicyActivated(ctx context.Context, pool *pgxpool.Pool, event db.ClaimedEvent) (string, error) {
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
	revID, err := uuid.Parse(payload.RevisionID)
	if err != nil {
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
	_ = revID

	// When a fair_use or plan policy is activated, run a coalesced FUP sweep across active members.
	if payload.Type == string(api.PolicyTypeFairUse) || payload.Type == string(api.PolicyTypePlan) {
		fupW, _ := NewFUPWorker(pool)
		rows, err := pool.Query(ctx, `SELECT id FROM members WHERE status = 'active' ORDER BY id LIMIT 500`)
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
			fupW.ScheduleMemberSweep(memberIDs, periodID)
			if _, err := fupW.Flush(ctx, time.Now().UTC()); err != nil {
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

func handleFUPEvaluate(ctx context.Context, pool *pgxpool.Pool, event db.ClaimedEvent) (string, error) {
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

	fupW, _ := NewFUPWorker(pool)
	if payload.AllowRelaxation {
		fupW.NotifyLedgerCorrection(memberID, periodID, payload.RelaxationReason)
	} else {
		fupW.NotifyLedgerChange(memberID, periodID)
	}
	results, err := fupW.Flush(ctx, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("fup worker: flush evaluation: %w", err)
	}
	if len(results) == 0 {
		return "member:" + memberID.String(), nil
	}
	return "fup_evaluation:" + results[0].EvaluationID.String(), nil
}

func handleFUPMemberGenerationChanged(ctx context.Context, pool *pgxpool.Pool, event db.ClaimedEvent) (string, error) {
	var stateID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT id FROM fup_member_states WHERE id = $1
	`, event.AggregateID).Scan(&stateID)
	if err != nil {
		return "", fmt.Errorf("fup worker: read back fup_member_states %s: %w", event.AggregateID, err)
	}
	return "fup_member_state:" + stateID.String(), nil
}

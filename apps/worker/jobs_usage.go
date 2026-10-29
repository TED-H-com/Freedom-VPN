package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
)

func registerUsageJobs(r Registry) {
	RegisterUsageJobsWithPool(r, nil)
}

// RegisterUsageJobsWithPool registers usage projection rebuild and retention purge
// outbox handlers bound to an optional explicit database pool (§10.9, §15.4).
func RegisterUsageJobsWithPool(r Registry, pool *pgxpool.Pool) {
	r.Register("usage.projection_rebuild", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		p, cleanup, err := resolveWorkerPool(ctx, pool)
		if err != nil {
			return "", err
		}
		defer cleanup()
		return handleUsageProjectionRebuild(ctx, p, event)
	})
	r.Register("usage.retention_purge", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		p, cleanup, err := resolveWorkerPool(ctx, pool)
		if err != nil {
			return "", err
		}
		defer cleanup()
		return handleUsageRetentionPurge(ctx, p, event)
	})
}

func handleUsageProjectionRebuild(ctx context.Context, pool *pgxpool.Pool, _ db.ClaimedEvent) (string, error) {
	svc, err := usage.NewService(usage.Config{Pool: pool})
	if err != nil {
		return "", err
	}
	if _, err := svc.RebuildProjection(ctx); err != nil {
		return "", fmt.Errorf("usage worker: rebuild projection: %w", err)
	}
	st, err := svc.GetLedgerState(ctx)
	if err != nil {
		return "", fmt.Errorf("usage worker: read back ledger state: %w", err)
	}
	return "usage_projection:" + st.LedgerEpoch.String(), nil
}

func handleUsageRetentionPurge(ctx context.Context, pool *pgxpool.Pool, event db.ClaimedEvent) (string, error) {
	uCfg, err := config.LoadUsageFromEnv()
	if err != nil {
		return "", err
	}
	var payload struct {
		Cutoff *time.Time `json:"cutoff,omitempty"`
		Reason string     `json:"reason,omitempty"`
	}
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -uCfg.RawUsageRetentionDays)
	if payload.Cutoff != nil && !payload.Cutoff.IsZero() {
		cutoff = payload.Cutoff.UTC()
	}
	reason := payload.Reason
	if reason == "" {
		reason = "scheduled usage_event_payloads retention purge"
	}

	svc, err := usage.NewService(usage.Config{Pool: pool})
	if err != nil {
		return "", err
	}
	if _, err := svc.PurgeRetainedPayloads(ctx, cutoff, reason); err != nil {
		return "", fmt.Errorf("usage worker: purge retained payloads: %w", err)
	}
	return "fleet_config:00000000-0000-0000-0000-000000000002", nil
}

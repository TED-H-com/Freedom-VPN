package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	usagepkg "github.com/TED-H-com/Freedom-VPN/internal/usage"
)

// UsageServices holds explicitly configured worker dependencies for the usage
// ledger projection and retention domain.
type UsageServices struct {
	Pool    *pgxpool.Pool
	Config  config.UsageConfig
	Service *usagepkg.Service
}

type usageContextKey struct{}

// WithUsage supplies immutable usage worker dependencies via Built.Context.
func WithUsage(ctx context.Context, s *UsageServices) context.Context {
	return context.WithValue(ctx, usageContextKey{}, s)
}

// UsageFromContext retrieves the UsageServices attached to the worker context.
func UsageFromContext(ctx context.Context) *UsageServices {
	s, _ := ctx.Value(usageContextKey{}).(*UsageServices)
	return s
}

func usage() Domain {
	return NewUsageDomain(nil)
}

// NewUsageDomain constructs the worker Domain for usage projection rebuild and
// retention purge jobs (§10.9, §15.4). When cfg is nil, settings are validated
// from Dependencies.Config.
func NewUsageDomain(cfg *config.UsageConfig) Domain {
	return Domain{
		Name: "usage",
		Build: func(_ context.Context, d Dependencies) (Built, error) {
			var uCfg config.UsageConfig
			if cfg != nil {
				if err := cfg.Validate(); err != nil {
					return Built{}, err
				}
				uCfg = *cfg
			} else {
				var err error
				uCfg, err = config.UsageFromConfig(d.Config)
				if err != nil {
					return Built{}, err
				}
			}
			if d.Pool == nil {
				return Built{}, &config.SettingError{Name: "database_url_file", Err: errors.New("usage worker database required")}
			}
			svc, err := usagepkg.NewService(usagepkg.Config{Pool: d.Pool})
			if err != nil {
				return Built{}, err
			}
			s := &UsageServices{
				Pool:    d.Pool,
				Config:  uCfg,
				Service: svc,
			}
			return Built{
				Jobs:    UsageJobs(),
				Context: func(ctx context.Context) context.Context { return WithUsage(ctx, s) },
			}, nil
		},
	}
}

// UsageJobs returns a fresh registry of usage projection and retention outbox handlers.
func UsageJobs() Registry {
	r := Registry{}
	for _, name := range []string{"usage.projection_rebuild", "usage.retention_purge"} {
		r.Register(name, usageJob)
	}
	return r
}

func usageJob(ctx context.Context, event db.ClaimedEvent) (string, error) {
	s := UsageFromContext(ctx)
	if s == nil || s.Pool == nil || s.Service == nil {
		return "", errors.New("usage worker dependencies required")
	}
	if event.SchemaVersion != 1 {
		return "", errors.New("unsupported usage event schema")
	}
	switch event.EventType {
	case "usage.projection_rebuild":
		return handleUsageProjectionRebuild(ctx, s, event)
	case "usage.retention_purge":
		return handleUsageRetentionPurge(ctx, s, event)
	default:
		return "", errors.New("unowned usage event")
	}
}

func handleUsageProjectionRebuild(ctx context.Context, s *UsageServices, _ db.ClaimedEvent) (string, error) {
	if _, err := s.Service.RebuildProjection(ctx); err != nil {
		return "", fmt.Errorf("usage worker: rebuild projection: %w", err)
	}
	st, err := s.Service.GetLedgerState(ctx)
	if err != nil {
		return "", fmt.Errorf("usage worker: read back ledger state: %w", err)
	}
	return "usage_projection:" + st.LedgerEpoch.String(), nil
}

func handleUsageRetentionPurge(ctx context.Context, s *UsageServices, event db.ClaimedEvent) (string, error) {
	var payload struct {
		Cutoff *time.Time `json:"cutoff,omitempty"`
		Reason string     `json:"reason,omitempty"`
	}
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -s.Config.RawUsageRetentionDays)
	if payload.Cutoff != nil && !payload.Cutoff.IsZero() {
		cutoff = payload.Cutoff.UTC()
	}
	reason := payload.Reason
	if reason == "" {
		reason = "scheduled usage_event_payloads retention purge"
	}

	if _, err := s.Service.PurgeRetainedPayloads(ctx, cutoff, reason); err != nil {
		return "", fmt.Errorf("usage worker: purge retained payloads: %w", err)
	}
	return "fleet_config:00000000-0000-0000-0000-000000000002", nil
}

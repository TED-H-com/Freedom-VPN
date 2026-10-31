package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	usagepkg "github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func usage() Domain {
	return NewUsageDomain(nil, nil, nil)
}

// NewUsageDomain constructs the control-plane Domain for usage ingestion,
// checkpoints, and member usage queries (§10.9, §14.2, §14.3, §15.4, D-045).
// When uCfg or pCfg is nil, settings are loaded from Dependencies.Config.UsageConfigFile
// and Dependencies.Config.PoliciesConfigFile.
func NewUsageDomain(uCfg *config.UsageConfig, pCfg *config.PoliciesConfig, signer usagepkg.Signer) Domain {
	return Domain{
		Name: "usage",
		Build: func(_ context.Context, d Dependencies) (Built, error) {
			if err := config.ValidateUsageCentralBounds(d.Config); err != nil {
				return Built{}, err
			}
			if err := config.ValidatePoliciesCentralBounds(d.Config); err != nil {
				return Built{}, err
			}
			var uc config.UsageConfig
			if uCfg != nil {
				if err := uCfg.Validate(); err != nil {
					return Built{}, err
				}
				uc = *uCfg
			} else {
				var err error
				uc, err = config.LoadUsage(d.Config.UsageConfigFile)
				if err != nil {
					return Built{}, err
				}
			}
			var pc config.PoliciesConfig
			if pCfg != nil {
				if err := pCfg.Validate(); err != nil {
					return Built{}, err
				}
				pc = *pCfg
			} else {
				var err error
				pc, err = config.LoadPolicies(d.Config.PoliciesConfigFile)
				if err != nil {
					return Built{}, err
				}
			}
			h, err := NewUsageHandler(d.Pool, d.Logger, uc, pc, signer)
			if err != nil {
				return Built{}, err
			}
			return Built{
				Handlers: []any{h},
				Runners:  []func(context.Context) error{h.RunCleanup},
			}, nil
		},
	}
}

// UsageHandler serves node usage batch ingestion (POST /agent/usage-batches),
// signed checkpoint retrieval (GET /agent/usage-checkpoint), member usage
// queries (GET /member/usage), plus admin drill-down, adjustments, and
// freeze/recovery operations (§10.9, §14.2, §14.3, §15.4, §16.3, §17.3).
type UsageHandler struct {
	pool      *pgxpool.Pool
	logger    *slog.Logger
	service   *usagepkg.Service
	fupEngine *fup.Engine
	fupWorker *fup.Worker
}

// NewUsageHandler constructs a UsageHandler from explicit configuration.
// When signer is nil, uCfg.SigningConfigFile is opened with signing.OpenFileProvider;
// there is no fallback or compiled-in signing key (D-045).
func NewUsageHandler(
	pool *pgxpool.Pool,
	logger *slog.Logger,
	uCfg config.UsageConfig,
	pCfg config.PoliciesConfig,
	signer usagepkg.Signer,
) (*UsageHandler, error) {
	if err := uCfg.Validate(); err != nil {
		return nil, err
	}
	seconds := int(usagepkg.SamplingIntervalBound / time.Second)
	if uCfg.SamplingIntervalBoundSeconds != nil {
		seconds = *uCfg.SamplingIntervalBoundSeconds
	}
	var epochSigner usagepkg.Signer
	if signer == nil {
		if strings.TrimSpace(uCfg.SigningConfigFile) == "" {
			return nil, &config.SettingError{Name: "signing_config_file", Err: errors.New("signing_config_file required")}
		}
		prov, err := signing.OpenFileProvider(uCfg.SigningConfigFile)
		if err != nil {
			return nil, &config.SettingError{Name: "signing_config_file", Err: err}
		}
		ps := usagepkg.NewProviderSigner(prov)
		signer = ps
		epochSigner = ps
	} else {
		epochSigner = signer
	}
	var svc *usagepkg.Service
	if pool != nil {
		var err error
		svc, err = usagepkg.NewService(usagepkg.Config{
			Pool:                  pool,
			Signer:                signer,
			LedgerEpochSigner:     epochSigner,
			SamplingIntervalBound: time.Duration(seconds) * time.Second,
		})
		if err != nil {
			return nil, err
		}
	}

	var usageReader fup.UsageReader
	if pool != nil && svc != nil {
		usageReader = fup.NewUsageServiceReader(svc, pool)
	}
	engine := fup.NewEngine(pool, usageReader, nil, nil)
	if pCfg.FUPMode != "" {
		engine.SetModeOverride(pCfg.FUPMode)
	}
	fupWorker := fup.NewWorker(engine)

	return &UsageHandler{
		pool:      pool,
		logger:    logger,
		service:   svc,
		fupEngine: engine,
		fupWorker: fupWorker,
	}, nil
}

// Service returns the underlying usage.Service.
func (h *UsageHandler) Service() *usagepkg.Service { return h.service }

// SubmitUsageBatch handles POST /api/v1/agent/usage-batches (§10.6, §14.3, §14.5).
// Node identity comes strictly from the verified mTLS principal in context.
// Data is committed to PostgreSQL before ACK; frozen ledger returns 503.
func (h *UsageHandler) SubmitUsageBatch(ctx context.Context, r gen.SubmitUsageBatchRequestObject) (gen.SubmitUsageBatchResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Agent || p.NodeID == uuid.Nil || p.IncarnationID == uuid.Nil {
		return nil, apierr.New(apierr.Unauthenticated, "Authenticated node mTLS identity required.")
	}
	if r.Body == nil {
		return nil, apierr.New(apierr.MalformedRequest, "Request body is required.")
	}

	tx, hasTx := httpapi.Transaction(ctx)
	record, hasRecord := httpapi.CommandRecord(ctx)
	if hasTx == nil && !hasRecord {
		return nil, errors.New("usage: command record unavailable for replay persistence")
	}

	res, err := h.service.IngestBatch(ctx, usagepkg.NodeIdentity{
		NodeID:            p.NodeID,
		NodeIncarnationID: p.IncarnationID,
	}, *r.Body)
	if err != nil {
		return nil, err
	}

	// Notify coalesced FUP evaluation for members with newly accepted usage samples.
	h.notifyFairUseOnAcceptedEvents(ctx, r.Body.Events, res.Results)

	if hasTx == nil {
		rawBody, err := canonjson.Marshal(res.UsageBatchResult)
		if err != nil {
			return nil, fmt.Errorf("usage: marshal canonical replay body: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO usage_batch_replays (id, node_id, incarnation_id, response_body, expiry)
			VALUES ($1, $2, $3, $4, $5)
		`, record.ID, p.NodeID, p.IncarnationID, rawBody, record.Expiry); err != nil {
			return nil, fmt.Errorf("usage: persist batch replay %s: %w", record.ID, err)
		}
		nodeID := p.NodeID
		if err := httpapi.SetResult(ctx, db.IdempotencyResult{
			ResourceType:   "node",
			ResourceID:     &nodeID,
			ResponseStatus: 200,
		}); err != nil {
			return nil, err
		}
	}

	return gen.SubmitUsageBatch200JSONResponse{
		Body: res.UsageBatchResult,
	}, nil
}

// GetUsageCheckpoint handles GET /api/v1/agent/usage-checkpoint (§14.3, §16.3, §17.3).
func (h *UsageHandler) GetUsageCheckpoint(ctx context.Context, _ gen.GetUsageCheckpointRequestObject) (gen.GetUsageCheckpointResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Agent || p.NodeID == uuid.Nil || p.IncarnationID == uuid.Nil {
		return nil, apierr.New(apierr.Unauthenticated, "Authenticated node mTLS identity required.")
	}
	env, err := h.service.GetCheckpoint(ctx, usagepkg.NodeIdentity{
		NodeID:            p.NodeID,
		NodeIncarnationID: p.IncarnationID,
	})
	if err != nil {
		return nil, err
	}
	return gen.GetUsageCheckpoint200JSONResponse{
		Body: env,
	}, nil
}

// GetMemberUsage handles GET /api/v1/member/usage (§10.9, §11.6, §14.2).
func (h *UsageHandler) GetMemberUsage(ctx context.Context, r gen.GetMemberUsageRequestObject) (gen.GetMemberUsageResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Member || p.ID == uuid.Nil {
		return nil, apierr.New(apierr.Forbidden, "Member session required.")
	}

	period := ""
	if r.Params.PeriodId != nil {
		period = *r.Params.PeriodId
	}
	gran := usagepkg.GranularityMonth
	if r.Params.Granularity != nil {
		gran = usagepkg.Granularity(*r.Params.Granularity)
	}

	var fairUsePtr *api.FairUseStatus
	if h.fupEngine != nil {
		targetPeriod := period
		if targetPeriod == "" {
			targetPeriod = usagepkg.MonthOf(time.Now().UTC()).ID
		}
		if st, err := h.fupEngine.GetMemberStatus(ctx, p.ID, targetPeriod); err == nil {
			fairUsePtr = &st
		}
	}

	mu, err := h.service.GetMemberUsage(ctx, usagepkg.MemberUsageQuery{
		MemberID:    p.ID,
		Period:      period,
		Granularity: gran,
		FairUse:     fairUsePtr,
	})
	if err != nil {
		return nil, err
	}
	return gen.GetMemberUsage200JSONResponse{
		Body: mu,
	}, nil
}

// Replay handles idempotent retries for SubmitUsageBatch when an optional Idempotency-Key header was supplied (§14.5, §17.3).
func (h *UsageHandler) Replay(ctx context.Context, operationID string, ref db.IdempotencyRecord) (any, error) {
	if operationID != "SubmitUsageBatch" {
		return nil, fmt.Errorf("usage: unsupported replay operation %q", operationID)
	}
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Agent || p.NodeID == uuid.Nil || p.IncarnationID == uuid.Nil {
		return nil, apierr.New(apierr.Unauthenticated, "Authenticated node mTLS identity required.")
	}
	if ref.ID == uuid.Nil {
		return nil, fmt.Errorf("usage: missing idempotency record ID for replay")
	}
	if h.pool == nil {
		return nil, apierr.New(apierr.DependencyUnavailable, "usage ledger database unavailable")
	}

	var (
		storedNodeID uuid.UUID
		storedIncID  uuid.UUID
		rawBody      []byte
	)
	err := h.pool.QueryRow(ctx, `
		SELECT node_id, incarnation_id, response_body
		FROM usage_batch_replays
		WHERE id = $1 AND expiry > clock_timestamp()
	`, ref.ID).Scan(&storedNodeID, &storedIncID, &rawBody)
	if err != nil {
		return nil, fmt.Errorf("usage: load batch replay %s: %w", ref.ID, err)
	}
	if storedNodeID != p.NodeID || storedIncID != p.IncarnationID {
		return nil, apierr.New(apierr.Forbidden, "Node mTLS identity does not match idempotency record owner.")
	}

	var body api.UsageBatchResult
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return nil, fmt.Errorf("usage: decode batch replay %s: %w", ref.ID, err)
	}
	return gen.SubmitUsageBatch200JSONResponse{
		Body: body,
	}, nil
}

// CleanupReplays deletes expired usage_batch_replays rows and any rows whose
// idempotency_requests record has been removed (§14.5).
func (h *UsageHandler) CleanupReplays(ctx context.Context) error {
	if h == nil || h.pool == nil {
		return nil
	}
	_, err := h.pool.Exec(ctx, `
		DELETE FROM usage_batch_replays
		WHERE expiry <= clock_timestamp()
		   OR NOT EXISTS (SELECT 1 FROM idempotency_requests r WHERE r.id = usage_batch_replays.id)
	`)
	return err
}

// RunCleanup periodically purges expired usage_batch_replays rows until ctx is canceled.
func (h *UsageHandler) RunCleanup(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			probe, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := h.CleanupReplays(probe)
			cancel()
			if ctx.Err() != nil {
				return nil
			}
			if err != nil && h.logger != nil {
				h.logger.Warn("usage batch replay cleanup failed", "error_code", "CLEANUP_FAILED")
			}
		}
	}
}

// GetMemberUsageSummary returns the compact api.MemberUsageSummary for member detail/list views.
func (h *UsageHandler) GetMemberUsageSummary(ctx context.Context, memberID uuid.UUID, period string) (api.MemberUsageSummary, error) {
	var tierIdx *int
	if h.fupEngine != nil {
		targetPeriod := period
		if targetPeriod == "" {
			targetPeriod = usagepkg.MonthOf(time.Now().UTC()).ID
		}
		if st, err := h.fupEngine.GetMemberStatus(ctx, memberID, targetPeriod); err == nil {
			if idx, err := st.TierIndex.Get(); err == nil {
				tierIdx = &idx
			}
		}
	}
	return h.service.GetMemberUsageSummary(ctx, memberID, period, tierIdx)
}

// GetMemberDrillDown returns the admin usage drill-down by node and device for a member and UTC month.
func (h *UsageHandler) GetMemberDrillDown(ctx context.Context, memberID uuid.UUID, period string) (usagepkg.MemberDrillDown, error) {
	return h.service.GetMemberDrillDown(ctx, memberID, period)
}

// CreateAdjustment records an audited append-only usage adjustment and triggers FUP evaluation.
func (h *UsageHandler) CreateAdjustment(ctx context.Context, req usagepkg.AdjustmentCreate) (usagepkg.Adjustment, error) {
	adj, err := h.service.CreateAdjustment(ctx, req)
	if err != nil {
		return usagepkg.Adjustment{}, err
	}
	if h.fupWorker != nil {
		if adj.UploadBytes+adj.DownloadBytes < 0 {
			h.fupWorker.NotifyLedgerCorrection(adj.MemberID, adj.Period, adj.Reason)
		} else {
			h.fupWorker.NotifyLedgerChange(adj.MemberID, adj.Period)
		}
		_, _ = h.fupWorker.Flush(ctx, time.Now().UTC())
	}
	return adj, nil
}

// FreezeIngest freezes central usage ingestion and writes an audit row (§17.3).
func (h *UsageHandler) FreezeIngest(ctx context.Context, actor, reason string) error {
	return h.service.FreezeIngest(ctx, actor, reason)
}

// BeginRecovery rotates ledger_epoch with a signed ledger-epoch recovery record and writes an audit row (§17.3).
func (h *UsageHandler) BeginRecovery(ctx context.Context, req usagepkg.RecoveryRequest) (usagepkg.RecoveryState, error) {
	return h.service.BeginRecovery(ctx, req)
}

// CompleteRecovery exits reconciling mode and writes an audit row (§17.3).
func (h *UsageHandler) CompleteRecovery(ctx context.Context, actor string) error {
	return h.service.CompleteRecovery(ctx, actor)
}

func (h *UsageHandler) notifyFairUseOnAcceptedEvents(ctx context.Context, events []api.UsageEvent, results []api.UsageEventResult) {
	if h.pool == nil || h.fupWorker == nil {
		return
	}
	var hasActiveFUP bool
	if err := h.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM policy_definitions
			WHERE type = 'fair_use' AND active_revision_id IS NOT NULL
		)
	`).Scan(&hasActiveFUP); err != nil || !hasActiveFUP {
		return
	}

	type memberPeriod struct {
		memberID uuid.UUID
		periodID string
	}
	seen := make(map[memberPeriod]struct{})
	for i, ev := range events {
		if i >= len(results) || results[i].Status != api.UsageEventResultStatusAccepted {
			continue
		}
		for _, rec := range ev.Records {
			mid := uuid.UUID(rec.MemberId)
			segs, err := usagepkg.Split(ev.IntervalStart, ev.IntervalEnd, uint64(rec.UploadBytes), uint64(rec.DownloadBytes), usagepkg.GranularityMonth)
			if err != nil || len(segs) == 0 {
				mp := memberPeriod{memberID: mid, periodID: rec.PeriodId}
				seen[mp] = struct{}{}
				continue
			}
			for _, seg := range segs {
				mp := memberPeriod{memberID: mid, periodID: seg.ID}
				seen[mp] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return
	}
	for mp := range seen {
		h.fupWorker.NotifyLedgerChange(mp.memberID, mp.periodID)
		payload, _ := json.Marshal(map[string]any{
			"member_id": mp.memberID.String(),
			"period_id": mp.periodID,
		})
		_ = db.InTx(ctx, h.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{
				AggregateID:      mp.memberID,
				AggregateType:    "member",
				AggregateVersion: "1",
				EventType:        "fup.evaluate",
				SchemaVersion:    1,
				Payload:          payload,
			})
		})
	}
	_, _ = h.fupWorker.Flush(ctx, time.Now().UTC())
}

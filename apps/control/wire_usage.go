package control

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
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
)

// DefaultRecoveryKeyID is the key identifier used by the fallback TEST ONLY
// recovery signer when no signing config file is configured in tests.
const DefaultRecoveryKeyID = "test-only-central-recovery-key"

// DefaultLedgerEpochKeyID is the key identifier used by the fallback TEST ONLY
// ledger-epoch signer when no signing config file is configured in tests.
const DefaultLedgerEpochKeyID = "test-only-central-recovery-key-ledger-epoch"

// DefaultTestRecoveryPublicKey returns the Ed25519 public key for DefaultRecoveryKeyID
// (TEST ONLY - NOT A SECRET).
func DefaultTestRecoveryPublicKey() ed25519.PublicKey {
	return defaultTestRecoverySigner().PrivateKey.Public().(ed25519.PublicKey)
}

// DefaultTestLedgerEpochPublicKey returns the Ed25519 public key for DefaultLedgerEpochKeyID
// (TEST ONLY - NOT A SECRET).
func DefaultTestLedgerEpochPublicKey() ed25519.PublicKey {
	return defaultTestLedgerEpochSigner().PrivateKey.Public().(ed25519.PublicKey)
}

func defaultTestRecoverySigner() *usagepkg.Ed25519Signer {
	// TEST ONLY - NOT A SECRET: deterministic seed used only when SigningConfigFile is unset.
	seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET: fvpn-control-default-recovery-signer-v1"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	return usagepkg.NewRecoverySigner(DefaultRecoveryKeyID, priv)
}

func defaultTestLedgerEpochSigner() *usagepkg.Ed25519Signer {
	// TEST ONLY - NOT A SECRET: deterministic seed used only when SigningConfigFile is unset.
	seed := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET: fvpn-control-default-ledger-epoch-signer-v1"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	return usagepkg.NewLedgerEpochSigner(DefaultLedgerEpochKeyID, priv)
}

func usage() Domain {
	return NewUsageDomain(nil, nil, nil)
}

// NewUsageDomain constructs the control-plane Domain for usage ingestion,
// checkpoints, and member usage queries (§10.9, §14.2, §14.3, §15.4).
// When uCfg or pCfg is nil, settings are validated from Dependencies.Config.
func NewUsageDomain(uCfg *config.UsageConfig, pCfg *config.PoliciesConfig, signer usagepkg.Signer) Domain {
	return Domain{
		Name: "usage",
		Build: func(_ context.Context, d Dependencies) (Built, error) {
			var uc config.UsageConfig
			if uCfg != nil {
				if err := uCfg.Validate(); err != nil {
					return Built{}, err
				}
				uc = *uCfg
			} else {
				var err error
				uc, err = config.UsageFromConfig(d.Config)
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
				pc, err = config.PoliciesFromConfig(d.Config)
				if err != nil {
					return Built{}, err
				}
			}
			if d.Pool == nil {
				return Built{}, &config.SettingError{Name: "database_url_file", Err: errors.New("usage database required")}
			}
			h, err := NewUsageHandler(d.Pool, d.Logger, uc, pc, signer)
			if err != nil {
				return Built{}, err
			}
			return Built{Handlers: []any{h}}, nil
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

	mu            sync.RWMutex
	replayResults map[string][]api.UsageEventResult
}

// NewUsageHandler constructs a UsageHandler from explicit configuration.
func NewUsageHandler(
	pool *pgxpool.Pool,
	logger *slog.Logger,
	uCfg config.UsageConfig,
	pCfg config.PoliciesConfig,
	signer usagepkg.Signer,
) (*UsageHandler, error) {
	var epochSigner usagepkg.Signer
	if signer == nil && uCfg.SigningConfigFile != "" {
		prov, err := signing.OpenFileProvider(uCfg.SigningConfigFile)
		if err != nil {
			return nil, &config.SettingError{Name: "signing_config_file", Err: err}
		}
		ps := usagepkg.NewProviderSigner(prov)
		signer = ps
		epochSigner = ps
	}
	if signer == nil {
		signer = defaultTestRecoverySigner()
		epochSigner = defaultTestLedgerEpochSigner()
	} else if epochSigner == nil {
		epochSigner = signer
	}

	svc, err := usagepkg.NewService(usagepkg.Config{
		Pool:              pool,
		Signer:            signer,
		LedgerEpochSigner: epochSigner,
	})
	if err != nil {
		return nil, err
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
		pool:          pool,
		logger:        logger,
		service:       svc,
		fupEngine:     engine,
		fupWorker:     fupWorker,
		replayResults: make(map[string][]api.UsageEventResult),
	}, nil
}

// Service returns the underlying usage.Service.
func (h *UsageHandler) Service() *usagepkg.Service { return h.service }

// SubmitUsageBatch handles POST /api/v1/agent/usage-batches (§10.6, §14.3).
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

	res, err := h.service.IngestBatch(ctx, usagepkg.NodeIdentity{
		NodeID:            p.NodeID,
		NodeIncarnationID: p.IncarnationID,
	}, *r.Body)
	if err != nil {
		return nil, err
	}

	// Notify coalesced FUP evaluation for members with newly accepted usage samples.
	h.notifyFairUseOnAcceptedEvents(ctx, r.Body.Events, res.Results)

	if _, txErr := httpapi.Transaction(ctx); txErr == nil {
		nodeID := p.NodeID
		if err := httpapi.SetResult(ctx, db.IdempotencyResult{
			ResourceType:   "node",
			ResourceID:     &nodeID,
			ResponseStatus: 200,
		}); err != nil {
			return nil, err
		}
		if r.Params.IdempotencyKey != nil {
			key := uuid.UUID(*r.Params.IdempotencyKey).String()
			h.mu.Lock()
			h.replayResults[p.Scope()+":"+key] = res.UsageBatchResult.Results
			h.mu.Unlock()
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

// Replay handles idempotent retries for SubmitUsageBatch when an optional Idempotency-Key header was supplied (§14.5).
func (h *UsageHandler) Replay(ctx context.Context, operationID string, _ db.IdempotencyRecord) (any, error) {
	if operationID != "SubmitUsageBatch" {
		return nil, fmt.Errorf("usage: unsupported replay operation %q", operationID)
	}
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Agent || p.NodeID == uuid.Nil || p.IncarnationID == uuid.Nil {
		return nil, apierr.New(apierr.Unauthenticated, "Authenticated node mTLS identity required.")
	}

	wm, err := h.service.GetStreamWatermark(ctx, usagepkg.NodeIdentity{
		NodeID:            p.NodeID,
		NodeIncarnationID: p.IncarnationID,
	})
	if err != nil {
		return nil, err
	}
	return gen.SubmitUsageBatch200JSONResponse{
		Body: wm,
	}, nil
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

// Package httpwire composes the node domains at the real HTTP boundary.
package httpwire

import (
	"context"
	"encoding/pem"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Runtime struct {
	ResolverDocuments nodes.ResolverDocumentSource
	Pool              *pgxpool.Pool
	PKI               *pki.Service
	Enrollment        *enrollment.Service
	Nodes             *nodes.Service
	Releases          *releases.Service
	Fleet             *fleetupdate.Service
	Provider          signing.Provider
	Config            config.Nodes
}

// Open builds services once from process-owned dependencies. It neither reads
// environment nor opens a pool, and does not start background work (D-026).
func Open(ctx context.Context, pool *pgxpool.Pool, c config.Nodes, log *slog.Logger) (*Runtime, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	// Validate the configured credential at startup; the adapter reloads it for
	// rotation and never retains/logs this validation result.
	if _, e := config.ReadSecret(c.NetBirdTokenFile); e != nil {
		return nil, e
	}
	provider, e := signing.OpenFileProvider(c.ProviderDirectory)
	if e != nil {
		return nil, e
	}
	raw, e := os.ReadFile(c.CACertificateFile)
	if e != nil {
		return nil, e
	}
	block, rest := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" || len(rest) != 0 {
		return nil, errors.New("invalid node CA PEM")
	}
	ca, e := pki.NewCA(ctx, provider, c.CAKeyID, block.Bytes, time.Duration(c.CertificateSeconds)*time.Second)
	if e != nil {
		return nil, e
	}
	identity, e := pki.NewService(pool, ca, provider, time.Duration(c.RenewalOverlapSeconds)*time.Second)
	if e != nil {
		return nil, e
	}
	enroll, e := enrollment.NewService(pool, ca, identity, provider, provider, enrollment.Config{InstallerValidity: time.Duration(c.InstallerSeconds) * time.Second, PreinstalledValidity: time.Duration(c.PreinstalledSeconds) * time.Second, BootstrapValidity: time.Duration(c.BootstrapSeconds) * time.Second, Logger: log})
	if e != nil {
		return nil, e
	}
	trust, e := provider.TrustBundle(ctx)
	if e != nil {
		return nil, e
	}
	validator, e := releases.NewValidator(trust, c.ReleaseOrigins)
	if e != nil {
		return nil, e
	}
	validator = validator.WithCurrentTrust(func() (*envelope.TrustStore, error) { return provider.TrustBundle(context.Background()) })
	rel := &releases.Service{Pool: pool, Validator: validator, VerifyArtifacts: true}
	fleet, e := fleetupdate.New(pool, rel, fleetupdate.Config{AuthorizationTTL: time.Duration(c.AuthorizationSeconds) * time.Second, OnlineWindow: time.Duration(c.OnlineSeconds) * time.Second, StuckWindow: time.Duration(c.StuckSeconds) * time.Second})
	if e != nil {
		return nil, e
	}
	r := &Runtime{ResolverDocuments: NewResolverDocumentSource(pool), Pool: pool, PKI: identity, Enrollment: enroll, Releases: rel, Fleet: fleet, Provider: provider, Config: c}
	r.Nodes = nodes.NewService(pool, nodes.WithVersionObserver(func(ctx context.Context, tx pgx.Tx, report api.ObservationReport) error {
		if e := fleet.ObserveReportTx(ctx, tx, fleetupdate.Identity{NodeID: report.NodeId, IncarnationID: report.NodeIncarnationId}, report); e != nil {
			return e
		}
		// Baseline policy acknowledgements also progress without a management
		// peer ID; the worker leaves peer_readback pending until one is confirmed.
		payload, e := canonjson.Marshal(map[string]any{"schema_version": 1, "report_id": report.ReportId})
		if e != nil {
			return e
		}
		return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: report.NodeIncarnationId, AggregateType: "node-incarnation", AggregateVersion: "1", EventType: "node.peer.observe", SchemaVersion: 1, Payload: payload})
	}))
	return r, nil
}

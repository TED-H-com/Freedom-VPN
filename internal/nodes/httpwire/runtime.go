// Package httpwire composes the node domains at the real HTTP boundary.
package httpwire

import (
	"context"
	"encoding/pem"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"time"
)

type Runtime struct {
	Pool       *pgxpool.Pool
	PKI        *pki.Service
	Enrollment *enrollment.Service
	Nodes      *nodes.Service
	Releases   *releases.Service
	Fleet      *fleetupdate.Service
	Provider   signing.Provider
	Config     config.Nodes
}

func Open(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) (*Runtime, error) {
	c, e := config.LoadNodes()
	if e != nil {
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
	rel := &releases.Service{Pool: pool, Validator: validator, VerifyArtifacts: true}
	fleet, e := fleetupdate.New(pool, rel, fleetupdate.Config{AuthorizationTTL: time.Duration(c.AuthorizationSeconds) * time.Second, OnlineWindow: time.Duration(c.OnlineSeconds) * time.Second, StuckWindow: time.Duration(c.StuckSeconds) * time.Second})
	if e != nil {
		return nil, e
	}
	r := &Runtime{Pool: pool, PKI: identity, Enrollment: enroll, Releases: rel, Fleet: fleet, Provider: provider, Config: c}
	r.Nodes = nodes.NewService(pool, nodes.WithVersionObserver(func(ctx context.Context, tx pgx.Tx, report api.ObservationReport) error {
		return fleet.ObserveReportTx(ctx, tx, fleetupdate.Identity{NodeID: report.NodeId, IncarnationID: report.NodeIncarnationId}, report)
	}))
	return r, nil
}

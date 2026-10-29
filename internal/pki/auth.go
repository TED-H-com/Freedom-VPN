package pki

import (
	"context"
	"crypto/sha256"
	"crypto/x509"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// AuthenticateAgent adapts verified TLS identity to the HTTP framework (§16.1).
// ID identifies the authenticated serial, never a caller-supplied node header.
func (s *Service) AuthenticateAgent(ctx context.Context, cert *x509.Certificate) (auth.Principal, error) {
	id, e := s.Authenticate(ctx, cert)
	if e != nil {
		return auth.Principal{}, e
	}
	p := auth.Principal{Role: auth.Agent, NodeID: id.NodeID, IncarnationID: id.IncarnationID}
	fp := sha256.Sum256(cert.Raw)
	e = s.pool.QueryRow(ctx, `SELECT id FROM node_certificates WHERE incarnation_id=$1 AND fingerprint=$2`, id.IncarnationID, fp[:]).Scan(&p.ID)
	return p, e
}

// Certificate recovers only the public certificate selected by authentication.
func (s *Service) Certificate(ctx context.Context) (*x509.Certificate, error) {
	p, ok := auth.FromContext(ctx)
	if !ok || p.Role != auth.Agent {
		return nil, apierr.New(apierr.Unauthenticated, "Machine authentication required.")
	}
	var raw []byte
	e := s.pool.QueryRow(ctx, `SELECT public_certificate FROM node_certificates WHERE id=$1 AND incarnation_id=$2`, p.ID, p.IncarnationID).Scan(&raw)
	if e != nil {
		return nil, apierr.New(apierr.Unauthenticated, "Certificate not registered.")
	}
	cert, e := x509.ParseCertificate(raw)
	if e != nil {
		return nil, e
	}
	id, e := s.Authenticate(ctx, cert)
	if e != nil {
		return nil, e
	}
	if id.NodeID != p.NodeID {
		return nil, apierr.New(apierr.IncarnationMismatch, "Certificate identity differs.")
	}
	return cert, nil
}

var _ auth.AgentAuthenticator = (*Service)(nil)

package pki

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool    *pgxpool.Pool
	ca      *CA
	fence   Revocations
	overlap time.Duration
}

func NewService(pool *pgxpool.Pool, ca *CA, fence Revocations, overlap time.Duration) (*Service, error) {
	if pool == nil || ca == nil || fence == nil || overlap < 0 || overlap >= ca.validity {
		return nil, errors.New("pki: database, CA, revocation fence and bounded overlap required")
	}
	return &Service{pool, ca, fence, overlap}, nil
}
func (s *Service) Authenticate(ctx context.Context, cert *x509.Certificate) (NodeIdentity, error) {
	id, err := s.ca.Verify(cert)
	if err != nil {
		return NodeIdentity{}, err
	}
	if err = s.CheckFence(ctx, id); err != nil {
		return NodeIdentity{}, err
	}
	if err = s.check(ctx, s.pool, id, cert); err != nil {
		return NodeIdentity{}, err
	}
	return id, nil
}
func (s *Service) CheckFence(ctx context.Context, id NodeIdentity) error {
	for _, v := range []uuid.UUID{id.NodeID, id.IncarnationID} {
		denied, err := s.fence.IdentityDenied(ctx, v.String())
		if err != nil {
			return apierr.New(apierr.DependencyUnavailable, "Identity authority unavailable.")
		}
		if denied {
			return apierr.New(apierr.IdentityRevoked, "Node identity revoked.")
		}
	}
	return nil
}

type rowReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Service) check(ctx context.Context, q rowReader, id NodeIdentity, cert *x509.Certificate) error {
	fp := sha256.Sum256(cert.Raw)
	var allowed bool
	err := q.QueryRow(ctx, `SELECT coalesce(n.lifecycle_state IN ('enrolling','registered') AND n.current_incarnation_id=i.id
 AND i.revoked_at IS NULL AND c.revoked_at IS NULL AND c.not_before<=clock_timestamp()
 AND least(c.not_after,coalesce(c.usable_until,c.not_after))>clock_timestamp(),false)
 FROM nodes n JOIN node_incarnations i ON i.node_id=n.id JOIN node_certificates c ON c.incarnation_id=i.id
 WHERE n.id=$1 AND i.id=$2 AND c.issuer=$3 AND c.fingerprint=$4`, id.NodeID, id.IncarnationID, s.ca.Issuer(), fp[:]).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(apierr.Unauthenticated, "Certificate is not registered.")
	}
	if err != nil {
		return apierr.New(apierr.DependencyUnavailable, "Identity database unavailable.")
	}
	if !allowed {
		return apierr.New(apierr.IdentityRevoked, "Node identity is inactive or revoked.")
	}
	return nil
}

// LockAuthenticated serializes state changes against node revocation. A caller
// must Authenticate before starting the transaction, then call this to close
// the check/use race. Certificate DER, never body IDs, selects the lock.
func (s *Service) LockAuthenticated(ctx context.Context, tx pgx.Tx, cert *x509.Certificate) (NodeIdentity, error) {
	id, err := s.ca.Verify(cert)
	if err != nil {
		return NodeIdentity{}, err
	}
	var node uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, id.NodeID).Scan(&node); err != nil {
		return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Unknown identity.")
	}
	if err = s.CheckFence(ctx, id); err != nil {
		return NodeIdentity{}, err
	}
	if err = s.check(ctx, tx, id, cert); err != nil {
		return NodeIdentity{}, err
	}
	return id, nil
}

// StoreCertificate is public-certificate-only storage, inside the enrollment or
// renewal transaction. Issue happened before taking a database row lock.
func StoreCertificate(ctx context.Context, tx pgx.Tx, incarnation uuid.UUID, c Issued) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	fp := sha256.Sum256(c.Certificate.Raw)
	_, err = tx.Exec(ctx, `INSERT INTO node_certificates(id,incarnation_id,issuer,serial,fingerprint,public_certificate,not_before,not_after) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, id, incarnation, c.Issuer, c.Certificate.SerialNumber.Text(16), fp[:], c.Certificate.Raw, c.Certificate.NotBefore, c.Certificate.NotAfter)
	return id, err
}
func (s *Service) Renew(ctx context.Context, cert *x509.Certificate, r api.CertificateRenewRequest) (out api.CertificateRenewResponse, err error) {
	id, err := s.Authenticate(ctx, cert)
	if err != nil {
		return out, err
	}
	if r.NodeId != id.NodeID || r.NodeIncarnationId != id.IncarnationID {
		return out, apierr.New(apierr.IncarnationMismatch, "Certificate identity does not match request.")
	}
	if r.RenewalRequestId == uuid.Nil {
		return out, invalidCSR()
	}
	if _, err = ParseCSR(r.CsrPem); err != nil {
		return out, err
	}
	raw, err := canonjson.Marshal(r)
	if err != nil {
		return out, invalidCSR()
	}
	hash := sha256.Sum256(raw)
	issued, err := s.ca.Issue(ctx, id, r.CsrPem)
	if err != nil {
		return out, err
	}
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := s.LockAuthenticated(ctx, tx, cert); e != nil {
			return e
		}
		var der, storedHash []byte
		e := tx.QueryRow(ctx, `SELECT c.public_certificate,r.request_hash FROM node_certificate_renewals r JOIN node_certificates c ON c.id=r.certificate_id WHERE r.incarnation_id=$1 AND r.request_id=$2`, id.IncarnationID, r.RenewalRequestId).Scan(&der, &storedHash)
		if e == nil {
			if !equalHash(storedHash, hash[:]) {
				return apierr.New(apierr.IdempotencyConflict, "Renewal request changed.")
			}
			c, e := x509.ParseCertificate(der)
			if e != nil {
				return e
			}
			out = renewResponse(c, s.ca.ChainPEM())
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		// Cap every older serial; repeated renewal never extends old overlap.
		if _, e = tx.Exec(ctx, `UPDATE node_certificates SET usable_until=least(coalesce(usable_until,not_after),clock_timestamp()+$2::bigint*interval '1 millisecond') WHERE incarnation_id=$1`, id.IncarnationID, s.overlap.Milliseconds()); e != nil {
			return e
		}
		certID, e := StoreCertificate(ctx, tx, id.IncarnationID, issued)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO node_certificate_renewals(incarnation_id,request_id,request_hash,certificate_id) VALUES($1,$2,$3,$4)`, id.IncarnationID, r.RenewalRequestId, hash[:], certID)
		if e != nil {
			return e
		}
		out = renewResponse(issued.Certificate, issued.CAChainPEM)
		return nil
	})
	if err != nil {
		return api.CertificateRenewResponse{}, safeError(err)
	}
	return out, nil
}
func equalHash(a, b []byte) bool { return string(a) == string(b) }
func renewResponse(c *x509.Certificate, chain string) api.CertificateRenewResponse {
	return api.CertificateRenewResponse{CaChainPem: chain, CertificatePem: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})), CertificateSerial: c.SerialNumber.Text(16), NotBefore: c.NotBefore.UTC(), NotAfter: c.NotAfter.UTC()}
}
func safeError(err error) error {
	var e *apierr.Error
	if errors.As(err, &e) {
		return e
	}
	return apierr.New(apierr.DependencyUnavailable, "Certificate operation unavailable.")
}

var _ AgentAuthenticator = (*Service)(nil)

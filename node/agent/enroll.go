package agent

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// Hooks are crash points for enrollment and bootstrap tests (§6.2, AT06).
// They run after the corresponding durable write, or, for the response hook,
// after the HTTP success and before the certificate files are stored.
type Hooks struct {
	AfterEnrollmentPersist  func() error
	AfterEnrollmentResponse func() error
	AfterBootstrapPersist   func() error
}

const enrollAttempts = 5

// Enroll creates one identity and exchanges it for a certificate (§6.1, §6.2).
// A second call does not mint another key or incarnation (AT06, AT07).
// token is used only as the Authorization header value and is not stored.
func (e *Engine) Enroll(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("agent: enrollment token is empty")
	}
	if e.enrolled {
		return nil
	}
	done, err := e.alreadyEnrolled(ctx)
	if err != nil || done {
		return err
	}
	key, err := loadOrCreateKey(e.cfg.keyPath())
	if err != nil {
		return err
	}
	row, err := e.ensurePending(ctx)
	if err != nil {
		return err
	}
	if row.CSRPEM == "" {
		csr, err := newCSR(key)
		if err != nil {
			return err
		}
		if err := e.store.SetEnrollmentCSR(ctx, csr); err != nil {
			return err
		}
		if err := writeAtomic(e.cfg.csrPath(), []byte(csr), 0o600); err != nil {
			return err
		}
		row.CSRPEM = csr
	} else if err := writeAtomic(e.cfg.csrPath(), []byte(row.CSRPEM), 0o600); err != nil {
		return err
	}
	if err := writeEnrollmentFile(e.cfg.enrollPath(), row.RequestID, row.IncarnationID); err != nil {
		return err
	}
	if e.hooks.AfterEnrollmentPersist != nil {
		if err := e.hooks.AfterEnrollmentPersist(); err != nil {
			return err
		}
	}
	resp, err := e.enrollRetry(ctx, token, key, row)
	if err != nil {
		return err
	}
	if e.hooks.AfterEnrollmentResponse != nil {
		if err := e.hooks.AfterEnrollmentResponse(); err != nil {
			return err
		}
	}
	return e.finishEnroll(ctx, key, row, resp)
}

func (e *Engine) alreadyEnrolled(ctx context.Context) (bool, error) {
	id, ok, err := e.store.Identity(ctx)
	if err != nil {
		return false, err
	}
	if ok {
		e.nodeID = id.NodeID
		e.incID = id.IncarnationID
		e.enrolled = true
		_ = e.loadCert()
		return true, nil
	}
	row, ok, err := e.store.Enrollment(ctx)
	if err != nil || !ok || row.Status != agentstore.StatusPending {
		return false, err
	}
	if !fileExists(e.cfg.certPath()) || !fileExists(e.cfg.nodePath()) {
		return false, nil
	}
	nf, err := loadNodeFile(e.cfg.nodePath())
	if err != nil {
		return false, err
	}
	node, err := uuid.Parse(nf.NodeID)
	if err != nil {
		return false, err
	}
	inc, err := uuid.Parse(nf.NodeIncarnationID)
	if err != nil {
		return false, err
	}
	if inc != row.IncarnationID || node == uuid.Nil {
		return false, errors.New("agent: enrollment files do not match the pending request")
	}
	certPEM, err := os.ReadFile(e.cfg.certPath())
	if err != nil {
		return false, err
	}
	cert, err := certFromPEM(certPEM)
	if err != nil {
		return false, err
	}
	key, err := loadOrCreateKey(e.cfg.keyPath())
	if err != nil {
		return false, err
	}
	if !keyMatches(cert, key) {
		return false, errors.New("agent: certificate does not match the identity key")
	}
	sum, err := publicSHA256(key)
	if err != nil {
		return false, err
	}
	if err := e.store.CompleteEnrollment(ctx, agentstore.Identity{
		NodeID: node, IncarnationID: inc, Serial: nf.Serial, NotAfter: nf.NotAfter, PublicSHA256: sum,
	}, e.clock.Wall()); err != nil {
		return false, err
	}
	e.nodeID, e.incID, e.enrolled = node, inc, true
	return true, e.loadCert()
}

func (e *Engine) ensurePending(ctx context.Context) (agentstore.Enrollment, error) {
	row, ok, err := e.store.Enrollment(ctx)
	if err != nil {
		return agentstore.Enrollment{}, err
	}
	if ok {
		if row.Status == agentstore.StatusEnrolled {
			return agentstore.Enrollment{}, errors.New("agent: enrollment already finished")
		}
		return row, nil
	}
	req, err := uuid.NewV7()
	if err != nil {
		return agentstore.Enrollment{}, err
	}
	inc, err := uuid.NewV7()
	if err != nil {
		return agentstore.Enrollment{}, err
	}
	if err := e.store.PutPendingEnrollment(ctx, req, inc, e.clock.Wall()); err != nil {
		return agentstore.Enrollment{}, err
	}
	row, ok, err = e.store.Enrollment(ctx)
	if err != nil {
		return agentstore.Enrollment{}, err
	}
	if !ok {
		return agentstore.Enrollment{}, errors.New("agent: enrollment was not stored")
	}
	return row, nil
}

func (e *Engine) enrollRetry(ctx context.Context, token string, key *ecdsa.PrivateKey, row agentstore.Enrollment) (api.EnrollResponse, error) {
	var last error
	for attempt := 0; attempt < enrollAttempts; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, enrollBackoff(attempt)); err != nil {
				return api.EnrollResponse{}, err
			}
		}
		resp, status, err := e.enrollOnce(ctx, token, key, row)
		if err == nil && (status == 200 || status == 201) {
			return resp, nil
		}
		if err == nil {
			err = errors.New("agent: enrollment was not accepted")
		}
		last = err
		if !retryable(err) {
			return api.EnrollResponse{}, err
		}
	}
	if last == nil {
		last = errors.New("agent: enrollment failed")
	}
	return api.EnrollResponse{}, last
}

func (e *Engine) enrollOnce(ctx context.Context, token string, key *ecdsa.PrivateKey, row agentstore.Enrollment) (api.EnrollResponse, int, error) {
	platform, err := localPlatform()
	if err != nil {
		return api.EnrollResponse{}, 0, err
	}
	req := api.EnrollRequest{
		AgentRelease:        e.cfg.AgentRelease,
		CsrPem:              row.CSRPEM,
		EnrollmentRequestId: row.RequestID,
		NodeIncarnationId:   row.IncarnationID,
		Platform:            platform,
	}
	proof, err := signProof(key, req)
	if err != nil {
		return api.EnrollResponse{}, 0, err
	}
	req.Proof = proof
	resp, status, err := e.client.Enroll(ctx, token, row.RequestID, req)
	if err != nil {
		var se *controlclient.StatusError
		if errors.As(err, &se) {
			return api.EnrollResponse{}, se.Status, err
		}
		return api.EnrollResponse{}, 0, err
	}
	return resp, status, nil
}

func (e *Engine) finishEnroll(ctx context.Context, key *ecdsa.PrivateKey, row agentstore.Enrollment, resp api.EnrollResponse) error {
	if resp.NodeIncarnationId != row.IncarnationID || resp.NodeId == uuid.Nil || resp.CertificatePem == "" || resp.CertificateSerial == "" {
		return errors.New("agent: enrollment response does not match the pending request")
	}
	cert, err := certFromPEM([]byte(resp.CertificatePem))
	if err != nil {
		return err
	}
	if !keyMatches(cert, key) {
		return errors.New("agent: certificate does not match the identity key")
	}
	if err := writeAtomic(e.cfg.certPath(), []byte(resp.CertificatePem), 0o644); err != nil {
		return err
	}
	if err := writeAtomic(e.cfg.caPath(), []byte(resp.CaChainPem), 0o644); err != nil {
		return err
	}
	notAfter := resp.CertificateNotAfter.UTC()
	if notAfter.IsZero() {
		notAfter = cert.NotAfter.UTC()
	}
	if err := writeNodeFile(e.cfg.nodePath(), nodeFile{
		NodeID: resp.NodeId.String(), NodeIncarnationID: resp.NodeIncarnationId.String(),
		Serial: resp.CertificateSerial, NotAfter: notAfter,
	}); err != nil {
		return err
	}
	sum, err := publicSHA256(key)
	if err != nil {
		return err
	}
	if err := e.store.CompleteEnrollment(ctx, agentstore.Identity{
		NodeID: resp.NodeId, IncarnationID: resp.NodeIncarnationId,
		Serial: resp.CertificateSerial, NotAfter: notAfter, PublicSHA256: sum,
	}, e.clock.Wall()); err != nil {
		return err
	}
	e.nodeID = resp.NodeId
	e.incID = resp.NodeIncarnationId
	e.enrolled = true
	return e.loadCert()
}

func enrollBackoff(attempt int) time.Duration {
	d := 200 * time.Millisecond
	for i := 1; i < attempt; i++ {
		d *= 2
		if d > 5*time.Second {
			d = 5 * time.Second
			break
		}
	}
	return d
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func retryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var se *controlclient.StatusError
	if errors.As(err, &se) {
		switch se.Status {
		case 429, 500, 502, 503, 504:
			return true
		default:
			return false
		}
	}
	return true
}

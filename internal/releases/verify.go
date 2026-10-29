package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Fetcher implementations must enforce approved origins on redirects and pin
// public DNS answers when dialing. The supplied HTTPSFetcher does both (§16.2).
type Fetcher interface {
	Open(context.Context, string) (io.ReadCloser, error)
}
type HTTPSFetcher struct {
	validator *Validator
	client    *http.Client
}

// NewHTTPSFetcher requires an explicit measured timeout; it never uses host
// proxy settings, follows redirects, or connects to private/special addresses.
func NewHTTPSFetcher(v *Validator, timeout time.Duration) (*HTTPSFetcher, error) {
	if timeout <= 0 {
		return nil, errors.New("artifact timeout must be configured")
	}
	tr := &http.Transport{TLSHandshakeTimeout: timeout, ResponseHeaderTimeout: timeout, DisableKeepAlives: true}
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if e != nil || len(ips) == 0 {
			return nil, errors.New("storage DNS lookup failed")
		}
		for _, ip := range ips {
			if !publicAddress(ip) {
				return nil, errors.New("storage DNS resolved to a nonpublic address")
			}
		}
		return (&net.Dialer{Timeout: timeout}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
	return &HTTPSFetcher{validator: v, client: &http.Client{Transport: tr, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("storage redirects are forbidden") }}}, nil
}
func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, p := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "2001::/32", "2002::/16", "64:ff9b::/96"} {
		if netip.MustParsePrefix(p).Contains(ip) {
			return false
		}
	}
	return true
}
func (f *HTTPSFetcher) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	if !f.validator.AllowURL(url) {
		return nil, apierr.New(apierr.ValidationFailed, "Storage origin is not allowed")
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if e != nil {
		return nil, e
	}
	resp, e := f.client.Do(req)
	if e != nil {
		return nil, apierr.New(apierr.StoreUnavailable, "Artifact fetch failed")
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, apierr.New(apierr.StoreUnavailable, "Artifact store did not return content")
	}
	return resp.Body, nil
}

// VerifyRelease is an idempotent worker step. All fetching occurs outside a
// transaction. Concurrent workers can verify immutable bytes but only the first
// pending->terminal transition commits evidence. A failed release stays ineligible.
func (s *Service) VerifyRelease(ctx context.Context, id uuid.UUID, fetch Fetcher) error {
	if fetch == nil {
		return errors.New("allowlisted fetcher required")
	}
	var raw []byte
	var state string
	e := s.Pool.QueryRow(ctx, `SELECT r.manifest,v.state FROM releases r JOIN release_verifications v ON v.release_id=r.id WHERE r.id=$1`, id).Scan(&raw, &state)
	if e != nil {
		return e
	}
	if state != "pending" {
		return nil
	}
	m, _, verifyErr := s.Validator.ValidateManifest(raw, time.Now().UTC())
	if verifyErr == nil {
		for _, a := range m.Signed.Artifacts {
			if verifyErr = s.verifyArtifact(ctx, fetch, a); verifyErr != nil {
				break
			}
		}
	}
	return commandtx.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var op, requestID *uuid.UUID
		if e = tx.QueryRow(ctx, `SELECT v.state,v.operation_id,(SELECT request_id FROM operations WHERE id=v.operation_id) FROM release_verifications v WHERE v.release_id=$1 FOR UPDATE`, id).Scan(&state, &op, &requestID); e != nil {
			return e
		}
		if state != "pending" {
			return nil
		}
		state = "verified"
		var code *string
		if verifyErr != nil {
			state = "failed"
			c := string(apierr.StoreUnavailable)
			var ae *apierr.Error
			if errors.As(verifyErr, &ae) {
				c = string(ae.Code)
			}
			code = &c
		}
		_, e = tx.Exec(ctx, `UPDATE release_verifications SET state=$2,error_code=$3,verified_at=clock_timestamp() WHERE release_id=$1`, id, state, code)
		if e != nil {
			return e
		}
		result := api.AuditEventResultSucceeded
		if verifyErr != nil {
			result = api.AuditEventResultFailed
		}
		if op != nil {
			status := api.OperationStepStatusSucceeded
			var safeCode *api.ErrorCode
			var evidence *operations.Evidence
			if verifyErr != nil {
				status = api.OperationStepStatusFailed
				c := api.ErrorCode(*code)
				safeCode = &c
			} else {
				evidence = &operations.Evidence{Reference: "release:" + id.String(), ObservedAt: time.Now().UTC()}
			}
			if e = operations.Transition(ctx, tx, *op, "verify", status, evidence, safeCode, nil); e != nil {
				return e
			}
		}
		if requestID == nil {
			id := commandtx.RequestID(ctx)
			requestID = &id
		}
		event, e := audit.NewEvent(audit.Entry{
			Actor: audit.Actor{Type: api.AuditEventActorTypeSystem, ID: uuid.MustParse(SystemActorID)}, Action: "release.verify",
			Resource: api.ResourceRef{Type: api.ResourceTypeRelease, Id: id}, Result: result,
			OperationID: op, RequestID: requestID,
			After: map[string]any{"verification_state": state, "error_code": code}, Fields: []string{"verification_state", "error_code"},
		})
		if e != nil {
			return e
		}
		return audit.AppendEvent(ctx, tx, event)
	})
}
func (s *Service) verifyArtifact(ctx context.Context, f Fetcher, a api.ManifestArtifact) error {
	if !s.Validator.AllowURL(a.Url) {
		return apierr.New(apierr.ValidationFailed, "Artifact storage origin is not allowed")
	}
	if a.SizeBytes >= math.MaxInt64 {
		return apierr.New(apierr.SizeMismatch, "Artifact exceeds verifier size limit")
	}
	r, e := f.Open(ctx, a.Url)
	if e != nil {
		return e
	}
	defer r.Close()
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(r, int64(a.SizeBytes)+1))
	if e != nil {
		return apierr.New(apierr.StoreUnavailable, "Artifact read failed")
	}
	if n != int64(a.SizeBytes) {
		return apierr.New(apierr.SizeMismatch, "Artifact length differs from manifest")
	}
	if hex.EncodeToString(h.Sum(nil)) != a.Sha256 {
		return apierr.New(apierr.HashMismatch, "Artifact digest differs from manifest")
	}
	return nil
}

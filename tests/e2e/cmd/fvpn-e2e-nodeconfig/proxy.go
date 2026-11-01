package main

// TEST ONLY transport fault. This loopback listener uses the enrolled node's
// real mTLS identity upstream. No policy, signature, cache or database is edited.
import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold"
	"github.com/google/uuid"
)

const faultMarker = "/var/lib/vpn-node/e2e/corrupt-resolver"
const faultEvidence = "/var/lib/vpn-node/e2e/resolver-fault.json"
const usageHoldMarker = "/var/lib/vpn-node/e2e/hold-usage"

// usageWithholdBody is the agent-visible 503. Control-side FreezeIngest is
// the product fault, but releasing it mints a new ledger epoch. This marker
// is the lab transport boundary the agent actually dials (127.0.0.1:9444).
const usageWithholdBody = `{"error":{"code":"DEPENDENCY_UNAVAILABLE","message":"usage acknowledgements withheld","request_id":"00000000-0000-0000-0000-000000000000","retryable":true,"details":{}}}`

func withholdUsage(w http.ResponseWriter, r *http.Request, marker string) bool {
	if r.Method != http.MethodPost || r.URL.Path != "/api/v1/agent/usage-batches" {
		return false
	}
	if _, err := os.Stat(marker); err != nil {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = io.WriteString(w, usageWithholdBody)
	return true
}

type resolverFault struct {
	marker, evidence string
	mu               sync.Mutex
	count            int
}

func (f *resolverFault) modify(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK || resp.Request.Method != http.MethodGet || !strings.HasPrefix(resp.Request.URL.Path, "/api/v1/agent/resolver-documents/") {
		return nil
	}
	marker, err := os.ReadFile(f.marker)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.New("fault marker unavailable")
	}
	id := strings.TrimPrefix(resp.Request.URL.Path, "/api/v1/agent/resolver-documents/")
	if strings.TrimSpace(string(marker)) != "*" && strings.TrimSpace(string(marker)) != id {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	resp.Body.Close()
	if err != nil || len(body) == 0 || len(body) > 1<<20 {
		return errors.New("fault response bound")
	}
	original := sha256.Sum256(body)
	body[0] ^= 1 // Exact length preserved; the authentic signed digest must fail.
	changed := sha256.Sum256(body)
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.Header.Set("Cache-Control", "no-store")
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	raw, err := json.Marshal(map[string]any{"fault_injected": true, "responses": f.count, "document_id": id, "http_status": resp.StatusCode, "size_bytes": len(body), "original_sha256": hex.EncodeToString(original[:]), "corrupted_sha256": hex.EncodeToString(changed[:])})
	if err != nil {
		return err
	}
	if err = os.WriteFile(f.evidence+".tmp", raw, 0600); err != nil {
		return err
	}
	return os.Rename(f.evidence+".tmp", f.evidence)
}

func resolverProxy() error {
	pool := func(path string) (*x509.CertPool, error) {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		p := x509.NewCertPool()
		if !p.AppendCertsFromPEM(raw) {
			return nil, errors.New("proxy CA")
		}
		return p, nil
	}
	serverRoots, err := pool("/etc/vpn-node/ca.pem")
	if err != nil {
		return err
	}
	clientRoots, err := pool("/etc/vpn-node/agent-ca.pem")
	if err != nil {
		return err
	}
	identity, err := tls.LoadX509KeyPair("/var/lib/vpn-node/identity/cert.pem", "/var/lib/vpn-node/identity/key.pem")
	if err != nil {
		return err
	}
	upstream, _ := url.Parse("https://agent.example.invalid:9443")
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy.Transport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: serverRoots, Certificates: []tls.Certificate{identity}}, ResponseHeaderTimeout: 15 * time.Second}
	fault := &resolverFault{marker: faultMarker, evidence: faultEvidence}
	hold := &matchingLeaseHold{directory: leasehold.Directory, now: time.Now}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if err := hold.modify(resp); err != nil {
			return err
		}
		return fault.modify(resp)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "lab transport unavailable", http.StatusBadGateway)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if withholdUsage(w, r, usageHoldMarker) {
			return
		}
		proxy.ServeHTTP(w, r)
	})
	srv := &http.Server{Addr: "127.0.0.1:9444", Handler: newFirstOfferObserver().wrap(handler), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ClientCAs: clientRoots, ClientAuth: tls.RequireAndVerifyClientCert}}
	return srv.ListenAndServeTLS("/etc/vpn-node/fault-proxy.crt", "/etc/vpn-node/fault-proxy.key")
}

// matchingLeaseHold is TEST ONLY. Successful desired responses select the next
// generation; only that generation's renewal responses become prompt 503s.
// Sleeping in the synchronous renewal would suppress Steps and alarm reports.
// No envelope is saved or replayed. Release passes fresh upstream bytes exactly.
type matchingLeaseHold struct {
	mu        sync.Mutex
	directory string
	now       func() time.Time
	evidence  *leasehold.Evidence
	until     time.Time // monotonic while this proxy process lives
}

const leaseWithholdBody = `{"error":{"code":"DEPENDENCY_UNAVAILABLE","message":"TEST ONLY matching lease withheld","request_id":"00000000-0000-0000-0000-000000000000","retryable":true,"details":{}}}`

func (f *matchingLeaseHold) save() error {
	raw, err := json.Marshal(f.evidence)
	if err != nil {
		return err
	}
	path := filepath.Join(f.directory, f.evidence.Arm.ID+".json")
	if err = os.WriteFile(path+".tmp", raw, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

func (f *matchingLeaseHold) release(now time.Time, reason string) error {
	f.evidence.ReleasedAt = &now
	f.evidence.Events = append(f.evidence.Events, leasehold.Event{Kind: "released", At: now, Reason: reason})
	if err := f.save(); err != nil {
		return err
	}
	// Only this arm's TEST ONLY marker directory; receipts live outside it.
	return os.RemoveAll(filepath.Join(f.directory, "active"))
}

func (f *matchingLeaseHold) modify(resp *http.Response) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	active := filepath.Join(f.directory, "active")
	if f.evidence == nil || f.evidence.ReleasedAt != nil {
		raw, err := os.ReadFile(filepath.Join(active, "arm.json"))
		if errors.Is(err, os.ErrNotExist) {
			if f.evidence == nil {
				// A proxy restart releases a consumed fixture; an agent-only stop
				// leaves this proxy and its monotonic hold intact.
				used, e := os.ReadFile(filepath.Join(active, "consumed.json"))
				if errors.Is(e, os.ErrNotExist) {
					return nil
				}
				var arm leasehold.Arm
				if e != nil || json.Unmarshal(used, &arm) != nil {
					return errors.New("lease hold restart receipt unavailable")
				}
				if _, e = uuid.Parse(arm.ID); e != nil {
					return errors.New("lease hold restart ID invalid")
				}
				prior, e := os.ReadFile(filepath.Join(f.directory, arm.ID+".json"))
				if e != nil || json.Unmarshal(prior, &f.evidence) != nil || f.evidence == nil {
					return errors.New("lease hold restart evidence unavailable")
				}
				return f.release(now, "proxy_restarted")
			}
			if n := len(f.evidence.Events); n > 0 && f.evidence.Events[n-1].Kind == "matching_lease_forwarded" {
				return nil
			}
		} else {
			if err != nil {
				return errors.New("lease hold arm unavailable")
			}
			var arm leasehold.Arm
			if json.Unmarshal(raw, &arm) != nil {
				return errors.New("lease hold arm invalid")
			}
			id, ie := uuid.Parse(arm.ID)
			base, ge := decstr.Parse(arm.BaseGeneration)
			if ie != nil || id == uuid.Nil || ge != nil || base == 0 || arm.HoldSeconds < 1 || arm.HoldSeconds > 120 || arm.ArmedAt.IsZero() || arm.ExpiresAt.After(arm.ArmedAt.Add(3*time.Minute)) || !arm.ExpiresAt.After(arm.ArmedAt) {
				return errors.New("lease hold arm invalid")
			}
			// Atomic consume: restarting this proxy cannot re-arm the same file.
			if err = os.Rename(filepath.Join(active, "arm.json"), filepath.Join(active, "consumed.json")); err != nil {
				return err
			}
			f.evidence = &leasehold.Evidence{Arm: arm, Events: []leasehold.Event{{Kind: "armed", At: arm.ArmedAt}, {Kind: "arm_consumed", At: now}}}
			if err = f.save(); err != nil {
				return err
			}
			f.until = time.Time{}
		}
	}
	ev := f.evidence
	if ev.ReleasedAt == nil && !now.Before(ev.Arm.ExpiresAt) {
		return f.release(now, "arm_deadline")
	}
	reason := ""
	if !f.until.IsZero() && !now.Before(f.until) {
		reason = "hold_deadline"
	}
	if raw, err := os.ReadFile(filepath.Join(active, "release")); err == nil && string(raw) == ev.Arm.ID {
		reason = "explicit_release"
	}
	if reason != "" && ev.ReleasedAt == nil {
		if err := f.release(now, reason); err != nil {
			return err
		}
	}
	desired := resp.Request.Method == http.MethodGet && resp.Request.URL.Path == "/api/v1/agent/desired-state"
	lease := resp.Request.Method == http.MethodPost && resp.Request.URL.Path == "/api/v1/agent/lease-renewals"
	if resp.StatusCode != http.StatusOK || (!desired && !lease) {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (4<<20)+1))
	_ = resp.Body.Close()
	if err != nil || len(body) > 4<<20 {
		return errors.New("lease hold response bound")
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	var doc struct {
		Signed struct {
			Generation string `json:"config_generation"`
			LeaseID    string `json:"lease_id"`
		} `json:"signed"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return errors.New("lease hold envelope invalid")
	}
	generation, err := decstr.Parse(doc.Signed.Generation)
	if err != nil {
		return errors.New("lease hold generation invalid")
	}
	base, _ := decstr.Parse(ev.Arm.BaseGeneration)
	if desired && generation > base && ev.TargetGeneration == "" && ev.ReleasedAt == nil {
		ev.TargetGeneration = doc.Signed.Generation
		ev.Events = append(ev.Events, leasehold.Event{Kind: "desired_forwarded", At: now, Generation: doc.Signed.Generation, SHA256: digestBytes(body), Bytes: len(body), UpstreamStatus: 200, DownstreamStatus: 200})
		return f.save()
	}
	if !lease || ev.TargetGeneration == "" {
		return nil
	}
	if doc.Signed.Generation != ev.TargetGeneration {
		target, _ := decstr.Parse(ev.TargetGeneration)
		if generation > target && ev.ReleasedAt == nil {
			return f.release(now, "generation_superseded")
		}
		return nil
	}
	id, err := uuid.Parse(doc.Signed.LeaseID)
	if err != nil || id == uuid.Nil {
		return errors.New("lease hold lease ID invalid")
	}
	event := leasehold.Event{Kind: "matching_lease_forwarded", At: now, Generation: doc.Signed.Generation, LeaseID: id.String(), SHA256: digestBytes(body), Bytes: len(body), UpstreamStatus: 200, DownstreamStatus: 200}
	if ev.ReleasedAt == nil {
		if ev.HoldStartedAt == nil {
			f.until = now.Add(time.Duration(ev.Arm.HoldSeconds) * time.Second)
			if f.until.After(ev.Arm.ExpiresAt) {
				f.until = ev.Arm.ExpiresAt
			}
			until := f.until.UTC()
			ev.HoldStartedAt, ev.HoldUntil = &now, &until
			ev.Events = append(ev.Events, leasehold.Event{Kind: "hold_started", At: now, Generation: doc.Signed.Generation})
		}
		event.Kind, event.DownstreamStatus = "withheld", 503
		resp.StatusCode, resp.Status = 503, "503 Service Unavailable"
		resp.Header = http.Header{"Content-Type": {"application/json"}, "Cache-Control": {"no-store"}, "Content-Length": {strconv.Itoa(len(leaseWithholdBody))}}
		resp.ContentLength = int64(len(leaseWithholdBody))
		resp.Body = io.NopCloser(strings.NewReader(leaseWithholdBody))
	}
	ev.Events = append(ev.Events, event)
	return f.save()
}

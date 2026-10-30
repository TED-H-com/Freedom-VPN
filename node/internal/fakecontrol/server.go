package fakecontrol

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

var errInvalid = errors.New("fakecontrol: invalid enrollment")

// Fixture is the desired state the server signs for whichever node enrolled.
// TEST ONLY: rates and lifetimes here are not product plan values.
type Fixture struct {
	ReleaseID   uuid.UUID
	Publication api.PublicationIntent
	Generation  uint64
	TTL         time.Duration
	// Capacity nil is an explicit JSON null (the node then cannot shape).
	Capacity  *api.NodeCapacity
	Bindings  []api.DeviceBinding
	Policies  []api.MemberRatePolicy
	Resolvers []string
	// Resolver is the signed resolver-document reference. Nil omits it.
	Resolver *api.ResolverDocumentRef
	// Egress is the signed egress probe. Nil omits it and keeps the local target.
	Egress *api.EgressProbe
}

type enrolled struct {
	fp      string
	request uuid.UUID
	inc     uuid.UUID
	node    uuid.UUID
	resp    api.EnrollResponse
}

// Server is the fake control plane. It listens on 127.0.0.1 with a runtime CA.
type Server struct {
	URL string

	mu        sync.Mutex
	ln        net.Listener
	httpSrv   *http.Server
	caCert    *x509.Certificate
	caKey     *ecdsa.PrivateKey
	caPEM     []byte
	trust     []byte
	desPriv   ed25519.PrivateKey
	leasePriv ed25519.PrivateKey
	recPriv   ed25519.PrivateKey
	desID     string
	leaseID   string
	recID     string

	tokens        map[string]*enrolled
	revoked       map[uuid.UUID]bool
	fixture       Fixture
	genDelta      int64
	forceGen      uint64
	corrupt       bool
	wrongNode     bool
	wrongInc      bool
	expire        bool
	delay         time.Duration
	failLeft      int
	leaseFail     int
	leaseRetryN   int
	leaseRetryFor time.Duration
	// leasePosts counts POST /lease-renewals, including a forced failure.
	leasePosts int
	// lastLeaseIssued keeps a new lease strictly after the previous one
	// when two renewals land in the same wall-clock tick.
	lastLeaseIssued time.Time
	offset          time.Duration
	nowFn           func() time.Time
	hold            chan struct{}
	// bootMode is ready, notready or expired. consumed is remembered after 200.
	bootMode     string
	bootKey      string
	bootExt      string
	bootExp      time.Time
	bootConsumed bool
	obs          map[string]api.ObservationAccepted
	lastObs      api.ObservationReport
	hasObs       bool
	enrollCount  int
	lastHash     string
	serviceFor   time.Duration
	maxOffline   int
	certTTL      time.Duration
	bump         uint64
	serial       int64
	renewed      map[string]api.CertificateRenewResponse
	sticky       bool
	stickyRaw    []byte
	stickyETag   string

	// Resolver documents. TEST ONLY. A missing id is 404 NOT_FOUND.
	docs       map[uuid.UUID][]byte
	docFetches int

	// Usage ledger. TEST ONLY knobs let the lab drive ACK, 503, 429 and a
	// signed checkpoint. usageEpoch is the live central generation.
	usageEpoch    uuid.UUID
	usageNodes    map[string]*usageLedger
	usageUnavail  int
	usageRetryN   int
	usageRetryFor time.Duration
	usageConflict int
	usageForce    *usageForce
}

// Start listens on 127.0.0.1 and returns a server with TEST ONLY keys.
func Start() (*Server, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fvpn-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(48 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	srvKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	srvTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "control.example.invalid"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(48 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost", "control.example.invalid"},
	}
	srvDER, err := x509.CreateCertificate(rand.Reader, srvTmpl, caCert, &srvKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	srvTLS, err := certPair(srvDER, srvKey)
	if err != nil {
		return nil, err
	}

	desPub, desPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	leasePub, leasePriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	recPub, recPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	nb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	na := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)
	ts, err := envelope.NewNodeTrustStore(
		envelope.TrustedKey{KeyID: "test-desired", PublicKey: desPub, Purposes: []envelope.Purpose{envelope.PurposeDesiredState}, NotBefore: nb, NotAfter: na},
		envelope.TrustedKey{KeyID: "test-lease", PublicKey: leasePub, Purposes: []envelope.Purpose{envelope.PurposeLease}, NotBefore: nb, NotAfter: na},
		envelope.TrustedKey{KeyID: "test-recovery", PublicKey: recPub, Purposes: []envelope.Purpose{envelope.PurposeRecovery}, NotBefore: nb, NotAfter: na},
	)
	if err != nil {
		return nil, err
	}
	trust, err := json.Marshal(ts)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	s := &Server{
		caCert: caCert, caKey: caKey, caPEM: caPEM, trust: trust,
		desPriv: desPriv, leasePriv: leasePriv, recPriv: recPriv,
		desID: "test-desired", leaseID: "test-lease", recID: "test-recovery",
		usageEpoch: uuid.Must(uuid.NewV7()), usageNodes: map[string]*usageLedger{},
		tokens: map[string]*enrolled{}, revoked: map[uuid.UUID]bool{},
		obs: map[string]api.ObservationAccepted{}, renewed: map[string]api.CertificateRenewResponse{},
		bootMode: "ready", bootKey: "test-only-setup-key", bootExt: "test-ext",
		bootExp:    time.Now().Add(time.Hour).UTC(),
		serviceFor: time.Hour, maxOffline: 3600, certTTL: 24 * time.Hour, serial: 10,
		fixture: Fixture{
			ReleaseID:   uuid.MustParse("019a0000-0000-7000-8000-0000000000c1"),
			Publication: api.PublicationIntentPublished,
			Generation:  1,
			TTL:         time.Hour,
			Capacity:    &api.NodeCapacity{UpBps: 200_000_000, DownBps: 200_000_000},
		},
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{srvTLS},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return nil, err
	}
	s.ln = ln
	s.URL = "https://" + ln.Addr().String()
	s.httpSrv = &http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = s.httpSrv.Serve(ln) }()
	return s, nil
}

// Close stops the listener.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

// CAPool is the pool the agent must pin.
func (s *Server) CAPool() *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(s.caCert)
	return p
}

// CACertPEM is the PEM CA certificate.
func (s *Server) CACertPEM() []byte { return append([]byte(nil), s.caPEM...) }

// TrustJSON is the node trust bundle (desired-state, lease and recovery).
func (s *Server) TrustJSON() []byte { return append([]byte(nil), s.trust...) }

// AddToken registers an enrollment token. Only its hash is kept.
func (s *Server) AddToken(token string) {
	sum := sha256.Sum256([]byte(token))
	s.mu.Lock()
	s.tokens[hex.EncodeToString(sum[:])] = nil
	s.mu.Unlock()
}

// Observations is how many distinct reports were accepted with 202.
func (s *Server) Observations() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.obs)
}

// EnrollCount is how many enrollment HTTP calls were attempted.
func (s *Server) EnrollCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enrollCount
}

// SetFixture replaces the desired-state template.
func (s *Server) SetFixture(f Fixture) {
	s.mu.Lock()
	s.fixture = f
	s.mu.Unlock()
}

// SetDelay sleeps before every request.
func (s *Server) SetDelay(d time.Duration) {
	s.mu.Lock()
	s.delay = d
	s.mu.Unlock()
}

// FailNext makes the next n requests return 503.
func (s *Server) FailNext(n int) {
	s.mu.Lock()
	s.failLeft = n
	s.mu.Unlock()
}

// FailLeases makes the next n lease renewals return 503. Desired state is
// not affected. TEST ONLY.
func (s *Server) FailLeases(n int) {
	s.mu.Lock()
	s.leaseFail = n
	s.mu.Unlock()
}

// SetLeaseRetryAfter makes the next n renewals return 429. TEST ONLY.
func (s *Server) SetLeaseRetryAfter(n int, seconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaseRetryN = n
	s.leaseRetryFor = time.Duration(seconds) * time.Second
}

// LeaseRenewals is how many lease POSTs were attempted, including failures.
func (s *Server) LeaseRenewals() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.leasePosts
}

// CorruptSignatures makes the next signed documents fail verification.
func (s *Server) CorruptSignatures(on bool) {
	s.mu.Lock()
	s.corrupt = on
	s.mu.Unlock()
}

// SetGenerationDelta adds n to the signed config generation (negative is older).
func (s *Server) SetGenerationDelta(n int64) {
	s.mu.Lock()
	s.genDelta = n
	s.mu.Unlock()
}

// SetWrongIdentity puts a different node or incarnation in the signed document.
func (s *Server) SetWrongIdentity(node, inc bool) {
	s.mu.Lock()
	s.wrongNode, s.wrongInc = node, inc
	s.mu.Unlock()
}

// ExpireDesired signs a document that is already outside its window.
func (s *Server) ExpireDesired(on bool) {
	s.mu.Lock()
	s.expire = on
	s.mu.Unlock()
}

// SetClockOffset skews the signed server time.
func (s *Server) SetClockOffset(d time.Duration) {
	s.mu.Lock()
	s.offset = d
	s.mu.Unlock()
}

// SetClock supplies signed server time from the test's clock. Nil restores
// wall time. SetClockOffset still applies, for explicit skew scenarios.
func (s *Server) SetClock(now func() time.Time) {
	s.mu.Lock()
	s.nowFn = now
	s.mu.Unlock()
}

// Revoke makes later mTLS calls return 403 IDENTITY_REVOKED.
func (s *Server) Revoke() {
	s.mu.Lock()
	s.revoked = map[uuid.UUID]bool{}
	for _, rec := range s.tokens {
		if rec != nil {
			s.revoked[rec.node] = true
		}
	}
	s.mu.Unlock()
}

// HoldLeases makes lease renewal block until ReleaseLeases.
func (s *Server) HoldLeases() {
	s.mu.Lock()
	if s.hold == nil {
		s.hold = make(chan struct{})
	}
	s.mu.Unlock()
}

// ReleaseLeases unblocks lease renewal.
func (s *Server) ReleaseLeases() {
	s.mu.Lock()
	if s.hold != nil {
		close(s.hold)
		s.hold = nil
	}
	s.mu.Unlock()
}

// SetBootstrap selects ready, notready or expired. A 200 moves the server to consumed.
func (s *Server) SetBootstrap(mode string) {
	s.mu.Lock()
	s.bootMode = mode
	s.bootConsumed = false
	s.mu.Unlock()
}

// SetServiceFor sets how far effective_service_end sits past issued_at.
// A cached sticky lease is dropped so the next renewal uses the new window.
func (s *Server) SetServiceFor(d time.Duration) {
	s.mu.Lock()
	s.serviceFor = d
	s.stickyRaw = nil
	s.stickyETag = ""
	s.mu.Unlock()
}

// StickyLease replays the first signed lease body. A replay keeps lease_id
// and issued_at, so the agent must not restart the monotonic deadline (§16.3).
func (s *Server) StickyLease(on bool) {
	s.mu.Lock()
	s.sticky = on
	if !on {
		s.stickyRaw = nil
		s.stickyETag = ""
	}
	s.mu.Unlock()
}

// BumpGeneration forces the next desired state to a new generation and ETag.
func (s *Server) BumpGeneration() {
	s.mu.Lock()
	s.bump++
	s.mu.Unlock()
}

// SetResolverRef sets the signed resolver-document reference and bumps the
// generation. Nil omits the reference. TEST ONLY.
func (s *Server) SetResolverRef(ref *api.ResolverDocumentRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ref == nil {
		s.fixture.Resolver = nil
	} else {
		cp := *ref
		s.fixture.Resolver = &cp
	}
	s.bump++
}

// SetEgressProbe sets the signed egress probe and bumps the generation.
// Nil omits it. TEST ONLY.
func (s *Server) SetEgressProbe(p *api.EgressProbe) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p == nil {
		s.fixture.Egress = nil
	} else {
		cp := *p
		cp.Targets = append([]api.EgressProbeTarget(nil), p.Targets...)
		s.fixture.Egress = &cp
	}
	s.bump++
}

// PutResolverBody serves body for GET /agent/resolver-documents/{id}.
// The bytes are returned unchanged. TEST ONLY.
func (s *Server) PutResolverBody(id uuid.UUID, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.docs == nil {
		s.docs = map[uuid.UUID][]byte{}
	}
	s.docs[id] = append([]byte(nil), body...)
}

// ResolverFetches is how many resolver-document GETs were attempted,
// including 404s. TEST ONLY.
func (s *Server) ResolverFetches() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.docFetches
}

// SetPublication changes the signed publication intent and bumps the
// generation so a cached ETag cannot hide it. TEST ONLY.
func (s *Server) SetPublication(intent api.PublicationIntent) {
	s.mu.Lock()
	s.fixture.Publication = intent
	s.bump++
	s.mu.Unlock()
}

// LastObservation is the newest accepted report. TEST ONLY.
func (s *Server) LastObservation() (api.ObservationReport, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastObs, s.hasObs
}

func (s *Server) now() time.Time {
	s.mu.Lock()
	fn, off := s.nowFn, s.offset
	s.mu.Unlock()
	t := time.Now()
	if fn != nil {
		t = fn()
	}
	return t.Add(off).UTC()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	s.mu.Lock()
	delay, fail := s.delay, s.failLeft
	if s.failLeft > 0 {
		s.failLeft--
	}
	s.mu.Unlock()
	if delay > 0 {
		t := time.NewTimer(delay)
		select {
		case <-t.C:
		case <-r.Context().Done():
			return
		}
	}
	if fail > 0 {
		apierr.Write(w, r, apierr.New(apierr.DependencyUnavailable, "Test control is unavailable."))
		return
	}
	if s.serveResolver(w, r) {
		return
	}
	switch r.URL.Path {
	case "/api/v1/agent/enroll":
		s.enroll(w, r)
	case "/api/v1/agent/desired-state":
		s.desired(w, r)
	case "/api/v1/agent/observations":
		s.observe(w, r)
	case "/api/v1/agent/lease-renewals":
		s.lease(w, r)
	case "/api/v1/agent/certificates/renew":
		s.renew(w, r)
	case "/api/v1/agent/bootstrap":
		s.bootstrap(w, r)
	case "/api/v1/agent/usage-batches":
		s.usage(w, r)
	case "/api/v1/agent/usage-checkpoint":
		s.usageCheckpoint(w, r)
	default:
		apierr.Write(w, r, apierr.New(apierr.ResourceNotFound, "Unknown agent path."))
	}
}

func (s *Server) enroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	token, ok := bearerEnrollment(r)
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	s.mu.Lock()
	s.enrollCount++
	s.lastHash = hash
	rec, known := s.tokens[hash]
	s.mu.Unlock()
	if !ok || !known {
		apierr.Write(w, r, apierr.New(apierr.EnrollmentTokenInvalid, "Enrollment token is not valid."))
		return
	}
	var req api.EnrollRequest
	if err := decodeJSON(r, &req); err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid enrollment request."))
		return
	}
	csr, err := verifyEnroll(req)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid node CSR or proof."))
		return
	}
	fp := fingerprint(csr)
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec != nil {
		if rec.fp != fp || rec.request != req.EnrollmentRequestId || rec.inc != req.NodeIncarnationId {
			apierr.Write(w, r, apierr.New(apierr.EnrollmentKeyMismatch, "Enrollment key does not match."))
			return
		}
		writeJSON(w, http.StatusOK, rec.resp)
		return
	}
	pub := csr.PublicKey.(*ecdsa.PublicKey)
	node := uuid.Must(uuid.NewV7())
	resp, err := s.issue(pub, node, req.NodeIncarnationId)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.InternalError, "Internal error."))
		return
	}
	s.tokens[hash] = &enrolled{fp: fp, request: req.EnrollmentRequestId, inc: req.NodeIncarnationId, node: node, resp: resp}
	writeJSON(w, http.StatusCreated, resp)
}

func bearerEnrollment(r *http.Request) (string, bool) {
	const p = "Enrollment "
	h := r.Header.Get("Authorization")
	if len(h) <= len(p) || h[:len(p)] != p {
		return "", false
	}
	tok := h[len(p):]
	if tok == "" {
		return "", false
	}
	return tok, true
}

func (s *Server) issue(pub *ecdsa.PublicKey, node, inc uuid.UUID) (api.EnrollResponse, error) {
	s.serial++
	serial := big.NewInt(s.serial)
	nb := time.Now().Add(-time.Minute)
	na := time.Now().Add(s.certTTL)
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		NotBefore:    nb,
		NotAfter:     na,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		URIs: []*url.URL{
			mustURI("urn:fvpn:node:" + node.String()),
			mustURI("urn:fvpn:incarnation:" + inc.String()),
		},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, s.caCert, pub, s.caKey)
	if err != nil {
		return api.EnrollResponse{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return api.EnrollResponse{
		CaChainPem:          string(s.caPEM),
		CertificateNotAfter: na.UTC(),
		CertificatePem:      string(certPEM),
		CertificateSerial:   hex.EncodeToString(serial.Bytes()),
		LifecycleState:      api.LifecycleStateRegistered,
		Next:                api.EnrollResponseNextApiv1agentdesiredState,
		NodeId:              node,
		NodeIncarnationId:   inc,
		PublicationIntent:   api.PublicationIntentPublished,
	}, nil
}

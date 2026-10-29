package fakecontrol

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func mustURI(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func certPair(der []byte, key *ecdsa.PrivateKey) (tls.Certificate, error) {
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
}

func decodeJSON(r *http.Request, dest any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, canonjson.MaxBytes+1))
	if err != nil {
		return err
	}
	if len(body) > canonjson.MaxBytes {
		return errInvalid
	}
	dec := json.NewDecoder(bytesReader(body))
	dec.DisallowUnknownFields()
	return dec.Decode(dest)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) identity(r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return uuid.Nil, uuid.Nil, false
	}
	var node, inc uuid.UUID
	for _, u := range r.TLS.PeerCertificates[0].URIs {
		switch {
		case len(u.String()) > len("urn:fvpn:node:") && u.String()[:len("urn:fvpn:node:")] == "urn:fvpn:node:":
			node, _ = uuid.Parse(u.String()[len("urn:fvpn:node:"):])
		case len(u.String()) > len("urn:fvpn:incarnation:") && u.String()[:len("urn:fvpn:incarnation:")] == "urn:fvpn:incarnation:":
			inc, _ = uuid.Parse(u.String()[len("urn:fvpn:incarnation:"):])
		}
	}
	if node == uuid.Nil || inc == uuid.Nil {
		return uuid.Nil, uuid.Nil, false
	}
	s.mu.Lock()
	rev := s.revoked[node]
	s.mu.Unlock()
	if rev {
		return node, inc, false
	}
	return node, inc, true
}

func (s *Server) requireIdentity(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	node, inc, ok := s.identity(r)
	if ok {
		return node, inc, true
	}
	s.mu.Lock()
	rev := node != uuid.Nil && s.revoked[node]
	s.mu.Unlock()
	if rev {
		apierr.Write(w, r, apierr.New(apierr.IdentityRevoked, "Identity is revoked."))
		return uuid.Nil, uuid.Nil, false
	}
	apierr.Write(w, r, apierr.New(apierr.Unauthenticated, "Client certificate is required."))
	return uuid.Nil, uuid.Nil, false
}

func (s *Server) desired(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	node, inc, ok := s.requireIdentity(w, r)
	if !ok {
		return
	}
	raw, etag, err := s.signDesired(node, inc)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.InternalError, "Internal error."))
		return
	}
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) signDesired(node, inc uuid.UUID) ([]byte, string, error) {
	s.mu.Lock()
	f := s.fixture
	delta := s.genDelta
	force := s.forceGen
	bump := s.bump
	corrupt := s.corrupt
	wrongNode := s.wrongNode
	wrongInc := s.wrongInc
	expire := s.expire
	s.mu.Unlock()
	now := s.now()
	gen := f.Generation
	if gen == 0 {
		gen = 1
	}
	gen += bump
	if delta != 0 {
		ng := int64(gen) + delta
		if ng < 1 {
			ng = 1
		}
		gen = uint64(ng)
	}
	if force > 0 {
		gen = force
	}
	ttl := f.TTL
	if ttl <= 0 {
		ttl = time.Hour
	}
	issued, expires := now.Add(-time.Minute), now.Add(ttl)
	if expire {
		expires = now.Add(-time.Minute)
		issued = expires.Add(-time.Hour)
	}
	if wrongNode {
		node = uuid.Must(uuid.NewV7())
	}
	if wrongInc {
		inc = uuid.Must(uuid.NewV7())
	}
	pub := f.Publication
	if pub == "" {
		pub = api.PublicationIntentPublished
	}
	rel := f.ReleaseID
	if rel == uuid.Nil {
		rel = uuid.MustParse("019a0000-0000-7000-8000-0000000000c1")
	}
	resolvers := f.Resolvers
	if resolvers == nil {
		resolvers = []string{}
	}
	bindings := f.Bindings
	if bindings == nil {
		bindings = []api.DeviceBinding{}
	}
	policies := f.Policies
	if policies == nil {
		policies = []api.MemberRatePolicy{}
	}
	var cap presence.Nullable[api.NodeCapacity]
	if f.Capacity == nil {
		cap = presence.NewNullNullable[api.NodeCapacity]()
	} else {
		copied := *f.Capacity
		if !copied.MemberDownBpsCap.IsSpecified() {
			copied.MemberDownBpsCap = presence.NewNullNullable[int64]()
		}
		if !copied.MemberUpBpsCap.IsSpecified() {
			copied.MemberUpBpsCap = presence.NewNullNullable[int64]()
		}
		cap = presence.NewNullableWithValue(copied)
	}
	ds := api.DesiredState{
		AccountingPeriod: api.AccountingPeriod{
			PeriodId:    "2026-10",
			PeriodStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		},
		BindingRevision:          decstr.U64(1),
		ConfigGeneration:         decstr.U64(gen),
		DesiredReleaseId:         presence.NewNullNullable[uuid.UUID](),
		DeviceBindings:           bindings,
		Dns:                      api.DnsServiceConfig{ResolverIpv4: resolvers, ResolverIpv6: []string{}},
		ExpiresAt:                expires,
		IssuedAt:                 issued,
		LeaseRenewalEndpoint:     api.DesiredStateLeaseRenewalEndpointApiv1agentleaseRenewals,
		MemberRatePolicies:       policies,
		NodeCapacity:             cap,
		NodeId:                   node,
		NodeIncarnationId:        inc,
		PolicyRefs:               api.PolicyRefs{AuthorizationGeneration: 1, DnsGeneration: 1, FupGeneration: 1},
		PublicationIntent:        pub,
		SafetyProfile:            "internet-only-v1",
		SchemaVersion:            api.DesiredStateSchemaVersionN1,
		ServingReleaseIds:        []uuid.UUID{rel},
		SupportedReleasePolicyId: uuid.MustParse("019a0000-0000-7000-8000-0000000000d1"),
		Update:                   presence.NewNullNullable[api.UpdateIntent](),
	}
	return s.sign(ds, s.desPriv, s.desID, corrupt)
}

func (s *Server) sign(doc any, priv ed25519.PrivateKey, keyID string, corrupt bool) ([]byte, string, error) {
	env, err := envelope.Sign(doc, priv, keyID)
	if err != nil {
		return nil, "", err
	}
	if corrupt && len(env.Signatures) > 0 {
		env.Signatures[0].SignatureB64 = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	}
	raw, err := env.MarshalCanonical()
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(raw)
	etag := `"` + hexEncode(sum[:]) + `"`
	return raw, etag, nil
}

func (s *Server) lease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	node, inc, ok := s.requireIdentity(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	hold := s.hold
	s.mu.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-r.Context().Done():
			return
		}
	}
	var req api.LeaseRenewalRequest
	if err := decodeJSON(r, &req); err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid lease renewal."))
		return
	}
	if req.NodeId != node || req.NodeIncarnationId != inc {
		apierr.Write(w, r, apierr.New(apierr.IdentityRevoked, "Identity does not match the certificate."))
		return
	}
	raw, _, err := s.signLease(node, inc)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.InternalError, "Internal error."))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) signLease(node, inc uuid.UUID) ([]byte, string, error) {
	s.mu.Lock()
	if s.sticky && s.stickyRaw != nil && !s.corrupt && !s.wrongNode && !s.wrongInc {
		raw := append([]byte(nil), s.stickyRaw...)
		etag := s.stickyETag
		s.mu.Unlock()
		return raw, etag, nil
	}
	f := s.fixture
	serviceFor := s.serviceFor
	maxOff := s.maxOffline
	corrupt := s.corrupt
	wrongNode := s.wrongNode
	wrongInc := s.wrongInc
	sticky := s.sticky
	s.mu.Unlock()
	if maxOff < 60 {
		maxOff = 60
	}
	if maxOff > 86400 {
		maxOff = 86400
	}
	if serviceFor <= 0 {
		serviceFor = time.Hour
	}
	now := s.now()
	issued := now
	nodeEnd := issued.Add(time.Duration(maxOff) * time.Second)
	if wrongNode {
		node = uuid.Must(uuid.NewV7())
	}
	if wrongInc {
		inc = uuid.Must(uuid.NewV7())
	}
	seen := map[uuid.UUID]bool{}
	var members []api.MemberLease
	for _, b := range f.Bindings {
		if seen[b.MemberId] {
			continue
		}
		seen[b.MemberId] = true
		svc := issued.Add(serviceFor)
		leaseEnd := svc
		if nodeEnd.Before(leaseEnd) {
			leaseEnd = nodeEnd
		}
		offEnd := issued.Add(time.Duration(maxOff) * time.Second)
		if offEnd.Before(leaseEnd) {
			leaseEnd = offEnd
		}
		if !svc.After(issued) {
			continue
		}
		members = append(members, api.MemberLease{
			EffectiveServiceEnd: svc,
			EntitlementRevision: 1,
			LeaseEnd:            leaseEnd,
			MemberId:            b.MemberId,
		})
	}
	if members == nil {
		members = []api.MemberLease{}
	}
	gen := f.Generation
	if gen == 0 {
		gen = 1
	}
	lease := api.ServingLease{
		AuthorizationGeneration: 1,
		ConfigGeneration:        decstr.U64(gen),
		ExpiresAt:               nodeEnd,
		IssuedAt:                issued,
		LeaseId:                 uuid.Must(uuid.NewV7()),
		MaxOfflineSeconds:       maxOff,
		Members:                 members,
		NodeId:                  node,
		NodeIncarnationId:       inc,
		SchemaVersion:           api.ServingLeaseSchemaVersionN1,
	}
	raw, etag, err := s.sign(lease, s.leasePriv, s.leaseID, corrupt)
	if err != nil {
		return nil, "", err
	}
	if sticky && !corrupt {
		s.mu.Lock()
		if s.stickyRaw == nil {
			s.stickyRaw = append([]byte(nil), raw...)
			s.stickyETag = etag
		}
		s.mu.Unlock()
	}
	return raw, etag, nil
}

func (s *Server) observe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	node, inc, ok := s.requireIdentity(w, r)
	if !ok {
		return
	}
	var rep api.ObservationReport
	if err := decodeJSON(r, &rep); err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid observation."))
		return
	}
	if rep.NodeId != node || rep.NodeIncarnationId != inc || rep.ReportId == uuid.Nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Observation identity does not match."))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.obs[rep.ReportId.String()]; ok {
		writeJSON(w, http.StatusAccepted, prev)
		return
	}
	acc := api.ObservationAccepted{
		Accepted:   true,
		ReceivedAt: time.Now().UTC(),
		ReportId:   rep.ReportId,
	}
	s.obs[rep.ReportId.String()] = acc
	writeJSON(w, http.StatusAccepted, acc)
}

func (s *Server) renew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	node, inc, ok := s.requireIdentity(w, r)
	if !ok {
		return
	}
	var req api.CertificateRenewRequest
	if err := decodeJSON(r, &req); err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid renewal."))
		return
	}
	if req.NodeId != node || req.NodeIncarnationId != inc {
		apierr.Write(w, r, apierr.New(apierr.IdentityRevoked, "Identity does not match the certificate."))
		return
	}
	csr, err := parseCSR(req.CsrPem)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid node CSR or proof."))
		return
	}
	key := req.RenewalRequestId.String()
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.renewed[key]; ok {
		writeJSON(w, http.StatusCreated, prev)
		return
	}
	pub := csr.PublicKey.(*ecdsa.PublicKey)
	enr, err := s.issue(pub, node, inc)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.InternalError, "Internal error."))
		return
	}
	resp := api.CertificateRenewResponse{
		CaChainPem:        enr.CaChainPem,
		CertificatePem:    enr.CertificatePem,
		CertificateSerial: enr.CertificateSerial,
		NotAfter:          enr.CertificateNotAfter,
		NotBefore:         time.Now().Add(-time.Minute).UTC(),
	}
	s.renewed[key] = resp
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	if _, _, ok := s.requireIdentity(w, r); !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bootConsumed || s.bootMode == "consumed" {
		apierr.Write(w, r, apierr.New(apierr.BootstrapConsumed, "Bootstrap credential was already delivered."))
		return
	}
	switch s.bootMode {
	case "notready":
		apierr.Write(w, r, apierr.New(apierr.BootstrapNotReady, "Bootstrap credential is not ready."))
		return
	case "expired":
		apierr.Write(w, r, apierr.New(apierr.BootstrapExpired, "Bootstrap credential has expired."))
		return
	}
	body := api.AgentBootstrap{ExpiresAt: s.bootExp, ExternalKeyId: s.bootExt, SetupKey: s.bootKey}
	s.bootConsumed = true
	writeJSON(w, http.StatusOK, body)
}

func hexEncode(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexdigits[c>>4]
		out[i*2+1] = hexdigits[c&0x0f]
	}
	return string(out)
}

// bytesReader avoids importing bytes in the hot path name collision. It is
// io.Reader over a slice.
type sliceReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) io.Reader { return &sliceReader{b: b} }

func (s *sliceReader) Read(p []byte) (int, error) {
	if s.i >= len(s.b) {
		return 0, io.EOF
	}
	n := copy(p, s.b[s.i:])
	s.i += n
	return n, nil
}

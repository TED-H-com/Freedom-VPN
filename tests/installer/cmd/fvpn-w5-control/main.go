//go:build installerfixture

// Command fvpn-w5-control is the installer lab's enrollment endpoint.
// TEST ONLY. control-build.py builds this source in a temporary harness module
// with a node/ import path so it can consume the unchanged internal fixture.
// Enrollment-only mode retains the socket/marker suite's original boundary.
package main

import (
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
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func main() {
	dir := flag.String("dir", "", "directory for CA, trust and the listen URL")
	tokenFile := flag.String("token-file", "", "file containing one enrollment token")
	extra := flag.String("extra-token-file", "", "file containing a second enrollment token")
	health := flag.Bool("upgrade-health", false, "TEST ONLY signed unpublished health fixture with a real TCP listener")
	flag.Parse()
	// serve blocks on success. A return is a setup failure. The message
	// stays fixed so a token path is not copied into the log.
	if *health {
		_ = serveUpgrade(*dir, *tokenFile, *extra)
	} else {
		_ = serve(*dir, *tokenFile, *extra)
	}
	fmt.Fprintln(os.Stderr, "fvpn-w5-control: failed")
	os.Exit(1)
}

func serveUpgrade(dir, tokenFile, extraFile string) error {
	if dir == "" || tokenFile == "" {
		return fmt.Errorf("dir and token file are required")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	srv, err := fakecontrol.Start()
	if err != nil {
		return err
	}
	defer srv.Close()
	for _, path := range []string{tokenFile, extraFile} {
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		token := strings.TrimSpace(string(raw))
		if token == "" || strings.ContainsAny(token, "\r\n") {
			return fmt.Errorf("token file is empty")
		}
		srv.AddToken(token)
	}
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer probe.Close()
	go func() {
		for {
			conn, err := probe.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	srv.SetFixture(fakecontrol.UpgradeHealthFixture(
		uuid.MustParse("019a0000-0000-7000-8000-0000000000c1"),
		api.EgressProbeTarget{Address: "127.0.0.1", Port: probe.Addr().(*net.TCPAddr).Port, Protocol: api.EgressProbeTargetProtocolTcp},
	))
	// Local test control can stop only the probe listener. Signed control stays
	// available so a stopped listener must fail the candidate's actual TCP check.
	mux := http.NewServeMux()
	mux.HandleFunc("POST /stop-probe", func(w http.ResponseWriter, _ *http.Request) {
		_ = probe.Close()
		w.WriteHeader(http.StatusNoContent)
	})
	admin, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer admin.Close()
	httpSrv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = httpSrv.Serve(admin) }()
	for name, body := range map[string][]byte{
		"ca.pem": srv.CACertPEM(), "trust.json": srv.TrustJSON(),
		"url": []byte(srv.URL + "\n"), "probe": []byte(probe.Addr().String() + "\n"),
		"admin": []byte("http://" + admin.Addr().String() + "\n"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "ready"), []byte("ok\n"), 0644); err != nil {
		return err
	}
	select {}
}

type issued struct {
	fp   string
	resp api.EnrollResponse
}

type server struct {
	mu     sync.Mutex
	caPEM  []byte
	caCert *x509.Certificate
	caKey  *ecdsa.PrivateKey
	serial int64
	tokens map[string]*issued
}

func serve(dir, tokenFile, extraFile string) error {
	if dir == "" || tokenFile == "" {
		return fmt.Errorf("dir and token file are required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tokens := map[string]*issued{}
	if err := addTokenFile(tokens, tokenFile); err != nil {
		return err
	}
	if extraFile != "" {
		if err := addTokenFile(tokens, extraFile); err != nil {
			return err
		}
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fvpn-w5-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(48 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	srvKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
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
		return err
	}
	leaf, err := x509.ParseCertificate(srvDER)
	if err != nil {
		return err
	}
	srvTLS := tls.Certificate{
		Certificate: [][]byte{srvDER},
		PrivateKey:  srvKey,
		Leaf:        leaf,
	}
	trust, err := nodeTrust()
	if err != nil {
		return err
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{srvTLS},
		ClientAuth:   tls.RequestClientCert,
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return err
	}
	s := &server{caPEM: caPEM, caCert: caCert, caKey: caKey, serial: 10, tokens: tokens}
	httpSrv := &http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = httpSrv.Serve(ln) }()
	url := "https://" + ln.Addr().String()
	if err := os.WriteFile(dir+"/ca.pem", caPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(dir+"/trust.json", trust, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(dir+"/url", []byte(url+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(dir+"/ready", []byte("ok\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "fvpn-w5-control ready")
	select {}
}

func addTokenFile(dst map[string]*issued, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return fmt.Errorf("token file is empty")
	}
	sum := sha256.Sum256([]byte(token))
	dst[hex.EncodeToString(sum[:])] = nil
	return nil
}

func nodeTrust() ([]byte, error) {
	nb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	na := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)
	key := func() ed25519.PublicKey {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil
		}
		return pub
	}
	des, lease, rec := key(), key(), key()
	if des == nil || lease == nil || rec == nil {
		return nil, fmt.Errorf("trust key")
	}
	ts, err := envelope.NewNodeTrustStore(
		envelope.TrustedKey{KeyID: "test-desired", PublicKey: des, Purposes: []envelope.Purpose{envelope.PurposeDesiredState}, NotBefore: nb, NotAfter: na},
		envelope.TrustedKey{KeyID: "test-lease", PublicKey: lease, Purposes: []envelope.Purpose{envelope.PurposeLease}, NotBefore: nb, NotAfter: na},
		envelope.TrustedKey{KeyID: "test-recovery", PublicKey: rec, Purposes: []envelope.Purpose{envelope.PurposeRecovery}, NotBefore: nb, NotAfter: na},
	)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(ts)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/agent/enroll":
		s.enroll(w, r)
	case "/api/v1/agent/bootstrap":
		s.bootstrap(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *server) enroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	token, ok := bearer(r)
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	s.mu.Lock()
	rec, known := s.tokens[hash]
	s.mu.Unlock()
	if !ok || !known {
		http.Error(w, "token", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "body", http.StatusBadRequest)
		return
	}
	var req api.EnrollRequest
	if json.Unmarshal(body, &req) != nil {
		http.Error(w, "body", http.StatusBadRequest)
		return
	}
	csr, err := parseCSR(req.CsrPem)
	if err != nil {
		http.Error(w, "csr", http.StatusBadRequest)
		return
	}
	pub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		http.Error(w, "csr", http.StatusBadRequest)
		return
	}
	fp := fingerprint(pub)
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec != nil {
		if rec.fp != fp {
			http.Error(w, "mismatch", http.StatusConflict)
			return
		}
		writeJSON(w, http.StatusOK, rec.resp)
		return
	}
	resp, err := s.issue(pub, req.NodeIncarnationId)
	if err != nil {
		http.Error(w, "issue", http.StatusInternalServerError)
		return
	}
	s.tokens[hash] = &issued{fp: fp, resp: resp}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *server) bootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	// TEST ONLY credential. The same value is what the agent lab returns.
	// It is not a NetBird setup key and is not logged.
	writeJSON(w, http.StatusOK, api.AgentBootstrap{
		ExpiresAt:     time.Now().Add(time.Hour).UTC(),
		ExternalKeyId: "test-ext",
		SetupKey:      "test-only-setup-key",
	})
}

func (s *server) issue(pub *ecdsa.PublicKey, inc uuid.UUID) (api.EnrollResponse, error) {
	s.serial++
	serial := big.NewInt(s.serial)
	node := uuid.Must(uuid.NewV7())
	nb := time.Now().Add(-time.Minute)
	na := time.Now().Add(24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		NotBefore:    nb,
		NotAfter:     na,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
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

func parseCSR(text string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(text))
	if block == nil {
		return nil, fmt.Errorf("csr")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, err
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, err
	}
	return csr, nil
}

func fingerprint(pub *ecdsa.PublicKey) string {
	raw, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func bearer(r *http.Request) (string, bool) {
	const p = "Enrollment "
	h := r.Header.Get("Authorization")
	if len(h) <= len(p) || !strings.HasPrefix(h, p) {
		return "", false
	}
	tok := h[len(p):]
	if tok == "" {
		return "", false
	}
	return tok, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

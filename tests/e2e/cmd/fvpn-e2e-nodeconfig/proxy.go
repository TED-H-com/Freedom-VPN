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
	"strings"
	"sync"
	"time"
)

const faultMarker = "/var/lib/vpn-node/e2e/corrupt-resolver"
const faultEvidence = "/var/lib/vpn-node/e2e/resolver-fault.json"

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
	proxy.ModifyResponse = fault.modify
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "lab transport unavailable", http.StatusBadGateway)
	}
	srv := &http.Server{Addr: "127.0.0.1:9444", Handler: proxy, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ClientCAs: clientRoots, ClientAuth: tls.RequireAndVerifyClientCert}}
	return srv.ListenAndServeTLS("/etc/vpn-node/fault-proxy.crt", "/etc/vpn-node/fault-proxy.key")
}

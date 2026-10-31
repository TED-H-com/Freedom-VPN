package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

// installerProxy adapts the installer's single HTTPS control URL to this lab's
// split public-enrollment and direct-mTLS listeners. Only enrollment is public;
// all other calls need the current node's verified client certificate. The
// upstream presents that same identity, reloaded after controlled reenrollment.
func installerProxy() error {
	pool := func(path string) (*x509.CertPool, error) {
		b, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		p := x509.NewCertPool()
		if !p.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("CA")
		}
		return p, nil
	}
	roots, err := pool("/var/tmp/e2e-rel/ca.pem")
	if err != nil {
		return err
	}
	clients, err := pool("/var/tmp/e2e-rel/agent-ca.pem")
	if err != nil {
		return err
	}
	proxy := func(target string, config *tls.Config) *httputil.ReverseProxy {
		u, _ := url.Parse(target)
		p := httputil.NewSingleHostReverseProxy(u)
		p.Transport = &http.Transport{TLSClientConfig: config, DisableKeepAlives: true, ResponseHeaderTimeout: 15 * time.Second}
		p.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "lab upstream unavailable", http.StatusBadGateway)
		}
		return p
	}
	public := proxy("https://portal.example.invalid:8443", &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots})
	identity := func() (tls.Certificate, error) {
		return tls.LoadX509KeyPair("/var/lib/vpn-node/identity/cert.pem", "/var/lib/vpn-node/identity/key.pem")
	}
	agent := proxy("https://agent.example.invalid:9443", &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots, GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) { c, e := identity(); return &c, e }})
	handler := installerProxyHandler(public, agent, identity, usageHoldMarker, "/var/lib/vpn-node/e2e/hold-first-poll")
	srv := &http.Server{Addr: "127.0.0.1:9444", Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13, ClientAuth: tls.VerifyClientCertIfGiven, ClientCAs: clients}}
	return srv.ListenAndServeTLS("/var/tmp/e2e-rel/fault-proxy.crt", "/var/tmp/e2e-rel/fault-proxy.key")
}

func installerProxyHandler(public, agent http.Handler, identity func() (tls.Certificate, error), holdMarker, firstPollMarker string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/agent/enroll" {
			public.ServeHTTP(w, r)
			return
		}
		current, err := identity()
		if err != nil || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 || len(current.Certificate) == 0 || !bytes.Equal(r.TLS.PeerCertificates[0].Raw, current.Certificate[0]) {
			http.Error(w, "node identity required", http.StatusForbidden)
			return
		}
		if withholdUsage(w, r, holdMarker) {
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/agent/desired-state" {
			if _, err := os.Stat(firstPollMarker); err == nil {
				http.Error(w, "TEST ONLY first desired-state poll held", http.StatusServiceUnavailable)
				return
			}
		}
		agent.ServeHTTP(w, r)
	})
}

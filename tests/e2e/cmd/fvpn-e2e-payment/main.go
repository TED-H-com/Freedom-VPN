// Command fvpn-e2e-payment answers the Stripe calls the control plane makes
// against https://api.stripe.com (§9). The hostname override is a compose
// alias. Authenticated TEST ONLY hooks supply linked provider truth.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", ":443", "listen address")
	cert := flag.String("cert", "", "TLS certificate")
	keyFile := flag.String("key", "", "TLS private key")
	hookFile := flag.String("hook-secret-file", "", "TEST ONLY fixture hook secret")
	webhookFile := flag.String("webhook-secret-file", "", "TEST ONLY webhook signing secret")
	apiKeyFile := flag.String("api-key-file", "", "expected bearer token")
	flag.Parse()
	if *cert == "" || *keyFile == "" || *apiKeyFile == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-payment: cert, key and api key are required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*apiKeyFile)
	if err != nil || strings.TrimSpace(string(raw)) == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-payment: api key")
		os.Exit(1)
	}
	hook, hookErr := os.ReadFile(*hookFile)
	webhook, webhookErr := os.ReadFile(*webhookFile)
	if hookErr != nil || webhookErr != nil || len(strings.TrimSpace(string(hook))) == 0 || len(strings.TrimSpace(string(webhook))) == 0 {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-payment: hook secrets required")
		os.Exit(2)
	}
	s := &server{apiKey: strings.TrimSpace(string(raw)), hookKey: strings.TrimSpace(string(hook)), webhookKey: strings.TrimSpace(string(webhook)), sessions: map[string]session{}, truth: map[string]snapshot{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/", s.route)
	srv := &http.Server{
		Addr: *addr, Handler: mux,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second,
	}
	if err := srv.ListenAndServeTLS(*cert, *keyFile); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-payment: stopped")
		os.Exit(1)
	}
}

type session struct {
	ID, URL                         string
	Expires                         int64
	Created                         int64
	Checkout, Price, Customer, Form string
}

type server struct {
	mu                  sync.Mutex
	apiKey              string
	hookKey, webhookKey string
	truth               map[string]snapshot
	refunds             map[string][]refund
	sessions            map[string]session
}

func (s *server) route(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/e2e/") {
		s.hook(w, r)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || s.apiKey == "" || subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), []byte(s.apiKey)) != 1 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}
	path := r.URL.Path
	switch {
	case r.Method == http.MethodPost && path == "/v1/checkout/sessions":
		s.createCheckout(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/v1/checkout/sessions/"):
		s.getCheckout(w, r, strings.TrimPrefix(path, "/v1/checkout/sessions/"))
	case r.Method == http.MethodPost && path == "/v1/billing_portal/sessions":
		s.createPortal(w, r)
	case r.Method == http.MethodGet:
		s.readTruth(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"message": "unknown"}})
	}
}

func (s *server) createCheckout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.PostForm.Get("metadata[fvpn_checkout_id]") == "" || r.PostForm.Get("line_items[0][price]") == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "form"}})
		return
	}
	id, err := newID("cs_e2e_")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"message": "id"}})
		return
	}
	expires := time.Now().Add(time.Hour).Unix()
	if raw := r.PostForm.Get("expires_at"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > time.Now().Unix() {
			expires = n
		}
	}
	customer := r.PostForm.Get("customer")
	if customer == "" {
		customer = "cus_" + id
	}
	item := session{ID: id, URL: "https://payments.example.invalid/checkout", Expires: expires, Created: time.Now().Unix(), Checkout: r.PostForm.Get("metadata[fvpn_checkout_id]"), Price: r.PostForm.Get("line_items[0][price]"), Customer: customer, Form: r.PostForm.Encode()}
	key := r.Header.Get("Idempotency-Key")
	s.mu.Lock()
	if key != "" {
		if old, ok := s.sessions["idem:"+key]; ok {
			if old.Form != item.Form {
				s.mu.Unlock()
				writeJSON(w, 409, map[string]string{"error": "idempotency conflict"})
				return
			}
			item = old
		} else {
			s.sessions[id] = item
			s.sessions["idem:"+key] = item
		}
	} else {
		s.sessions[id] = item
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"id": item.ID, "url": item.URL, "expires_at": item.Expires})
}

func (s *server) getCheckout(w http.ResponseWriter, _ *http.Request, id string) {
	id, _ = url.PathUnescape(id)
	s.mu.Lock()
	item, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"message": "missing"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": item.ID, "url": item.URL, "expires_at": item.Expires})
}

func (s *server) createPortal(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.PostForm.Get("customer") == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "form"}})
		return
	}
	id, err := newID("bps_e2e_")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"message": "id"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "url": "https://payments.example.invalid/portal", "created": time.Now().Unix()})
}

func newID(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

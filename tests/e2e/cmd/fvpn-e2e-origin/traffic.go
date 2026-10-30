package main

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// TEST ONLY: one authenticated, bounded synthetic probe slot, no URL or payload
// logging. Normal health checks and release downloads are never collected.
type trafficProbe struct {
	mu     sync.Mutex
	id     string
	count  int
	source string
}

func (p *trafficProbe) register(mux *http.ServeMux, secret string) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-E2E-Secret")), []byte(secret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		switch r.Method {
		case "PUT":
			var body struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body) != nil {
				http.Error(w, "bad probe", http.StatusBadRequest)
				return
			}
			if _, err := uuid.Parse(body.ID); err != nil {
				http.Error(w, "bad probe", http.StatusBadRequest)
				return
			}
			p.id, p.count, p.source = body.ID, 0, ""
			w.WriteHeader(http.StatusNoContent)
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"probe_id": p.id, "requests": p.count, "source_address": p.source})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
	mux.HandleFunc("GET /e2e/traffic", handler)
	mux.HandleFunc("PUT /e2e/traffic", handler)
}
func (p *trafficProbe) observe(r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.id != "" && r.Header.Get("X-E2E-Probe") == p.id {
		p.count++
		p.source, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
}

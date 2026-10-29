// Command fvpn-labcontrol is the in-container fake control plane for the
// agent lab (TEST ONLY). The lab container has no network, so this process
// must run inside it. Signing keys are generated at startup and are not
// product keys. The setup key is never written or logged.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func main() {
	dir := flag.String("dir", "/scratch/w2", "directory for CA, trust and the token file")
	admin := flag.String("admin", "127.0.0.1:9091", "admin listen address inside the container")
	flag.Parse()
	if err := serve(*dir, *admin); err != nil {
		fmt.Fprintln(os.Stderr, "labcontrol: failed")
		os.Exit(1)
	}
}

func serve(dir, admin string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	srv, err := fakecontrol.Start()
	if err != nil {
		return err
	}
	defer srv.Close()
	now := time.Now().UTC()
	cap := &api.NodeCapacity{
		UpBps:            200_000_000,
		DownBps:          200_000_000,
		MemberUpBpsCap:   presence.NewNullNullable[int64](),
		MemberDownBpsCap: presence.NewNullNullable[int64](),
	}
	srv.SetFixture(fakecontrol.Fixture{
		ReleaseID:   uuid.MustParse("019a0000-0000-7000-8000-0000000000c1"),
		Publication: api.PublicationIntentPublished,
		Generation:  1,
		TTL:         time.Hour,
		Capacity:    cap,
		Resolvers:   []string{"198.51.100.53"},
		Bindings:    labBindings(now),
	})
	token := "test-only-enrollment-token"
	srv.AddToken(token)
	if err := os.WriteFile(filepath.Join(dir, "ca.pem"), srv.CACertPEM(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "trust.json"), srv.TrustJSON(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "token"), []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "url"), []byte(srv.URL+"\n"), 0o644); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "observations": srv.Observations()})
	})
	mux.HandleFunc("POST /hold", func(w http.ResponseWriter, _ *http.Request) {
		srv.HoldLeases()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /release", func(w http.ResponseWriter, _ *http.Request) {
		srv.ReleaseLeases()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /revoke", func(w http.ResponseWriter, _ *http.Request) {
		srv.Revoke()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /bump", func(w http.ResponseWriter, _ *http.Request) {
		srv.BumpGeneration()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /service", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Seconds int `json:"seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Seconds <= 0 {
			http.Error(w, "seconds", http.StatusBadRequest)
			return
		}
		srv.SetServiceFor(time.Duration(body.Seconds) * time.Second)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /usage/unavailable", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			N int `json:"n"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.N < 0 {
			http.Error(w, "n", http.StatusBadRequest)
			return
		}
		srv.SetUsageUnavailable(body.N)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /usage/retry", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			N       int `json:"n"`
			Seconds int `json:"seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.N < 0 || body.Seconds < 0 {
			http.Error(w, "n", http.StatusBadRequest)
			return
		}
		srv.SetUsageRetryAfter(body.N, body.Seconds)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /usage/checkpoint", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Mode    string `json:"mode"`
			Highest uint64 `json:"highest"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "mode", http.StatusBadRequest)
			return
		}
		epoch, err := srv.SetUsageCheckpoint(body.Mode, body.Highest)
		if err != nil {
			http.Error(w, "mode", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"epoch": epoch.String()})
	})
	mux.HandleFunc("GET /usage", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(srv.UsageReport())
	})
	ln, err := net.Listen("tcp", admin)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "labcontrol admin stopped")
		}
	}()
	if err := os.WriteFile(filepath.Join(dir, "ready"), []byte("ok\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "labcontrol ready")
	select {}
}

func labBindings(now time.Time) []api.DeviceBinding {
	from := now.Add(-time.Minute)
	until := now.Add(24 * time.Hour)
	one := func(member, device, peer, ip string) api.DeviceBinding {
		return api.DeviceBinding{
			AllowedIpv4:     []string{ip + "/32"},
			AllowedIpv6:     []string{},
			BindingRevision: 1,
			DeviceId:        uuid.MustParse(device),
			MemberId:        uuid.MustParse(member),
			PeerId:          peer,
			ValidFrom:       from,
			ValidUntil:      until,
		}
	}
	return []api.DeviceBinding{
		one("019a0000-0000-7000-8000-00000000000a", "019a0000-0000-7000-8000-0000000000a1", "peer-a1", "100.64.1.1"),
		one("019a0000-0000-7000-8000-00000000000a", "019a0000-0000-7000-8000-0000000000a2", "peer-a2", "100.64.1.2"),
		one("019a0000-0000-7000-8000-00000000000b", "019a0000-0000-7000-8000-0000000000b1", "peer-b1", "100.64.2.1"),
	}
}

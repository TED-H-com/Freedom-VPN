// Command fvpn-e2e-origin serves release artifacts over HTTPS (§13.4).
// The address 192.0.3.10 is a routable unicast that the release fetcher
// accepts: RFC1918, loopback, documentation and CGNAT ranges are refused.
package main

import (
	"crypto/subtle"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	dnsTraceRole := flag.String("dns-trace-role", "", "TEST ONLY primary, secondary or authority metadata trace")
	dnsUpstream := flag.String("dns-upstream", "", "TEST ONLY DNS-only relay to the filtering resolver")
	udpAddr := flag.String("probe-udp-addr", "", "TEST ONLY authenticated UDP probe listener")
	memberTCP := flag.String("member-tcp-addr", "", "TEST ONLY persistent member TCP echo")
	memberUDP := flag.String("member-udp-addr", "", "TEST ONLY authenticated member UDP echo")
	addr := flag.String("addr", ":443", "listen address")
	cert := flag.String("cert", "", "TLS certificate")
	key := flag.String("key", "", "TLS private key")
	dir := flag.String("dir", "/var/lib/releases", "artifact root")
	secretFile := flag.String("secret-file", "", "upload secret")
	flag.Parse()
	if *dnsUpstream != "" {
		var err error
		if *dnsTraceRole != "" {
			if *dnsTraceRole != "primary" && *dnsTraceRole != "secondary" && *dnsTraceRole != "authority" {
				os.Exit(2)
			}
			if os.MkdirAll("/var/lib/fvpn-dns", 0700) != nil {
				os.Exit(1)
			}
			f, e := os.OpenFile("/var/lib/fvpn-dns/events.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if e != nil {
				os.Exit(1)
			}
			_ = f.Close()
			err = dnsrelay.ServeObserved(*addr, *dnsUpstream, "", &dnsrelay.Trace{Role: *dnsTraceRole, Path: "/var/lib/fvpn-dns/events.jsonl"})
		} else {
			err = dnsrelay.Serve(*addr, *dnsUpstream, "")
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "DNS relay failed")
			os.Exit(1)
		}
		return
	}
	if *cert == "" || *key == "" || *secretFile == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: cert, key and secret are required")
		os.Exit(2)
	}
	secret, err := os.ReadFile(*secretFile)
	if err != nil || strings.TrimSpace(string(secret)) == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: secret")
		os.Exit(1)
	}
	token := strings.TrimSpace(string(secret))
	root, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: directory")
		os.Exit(1)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: directory")
		os.Exit(1)
	}
	mux := http.NewServeMux()
	probe := &trafficProbe{}
	probe.register(mux, token)
	if *udpAddr != "" {
		conn, err := net.ListenPacket("udp4", *udpAddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: probe listener")
			os.Exit(1)
		}
		defer conn.Close()
		go probe.serveUDP(conn, []byte(token))
	}
	// Member echo is a different listener from the updater probe. UDP replies
	// require a device the TCP hello registered.
	echo := newMemberEcho()
	if *memberTCP != "" {
		ln, err := net.Listen("tcp4", *memberTCP)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: member tcp")
			os.Exit(1)
		}
		defer ln.Close()
		go echo.serveTCP(ln)
	}
	if *memberUDP != "" {
		conn, err := net.ListenPacket("udp4", *memberUDP)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: member udp")
			os.Exit(1)
		}
		defer conn.Close()
		go echo.serveUDP(conn, []byte(token))
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		probe.observe(r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("PUT /e2e/object", func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-E2E-Secret")), []byte(token)) != 1 {
			http.Error(w, "unauthorized\n", http.StatusUnauthorized)
			return
		}
		name := r.URL.Query().Get("name")
		path, ok := safePath(root, name)
		if !ok {
			http.Error(w, "bad path\n", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20+1))
		if err != nil || len(body) == 0 || len(body) > 64<<20 {
			http.Error(w, "bad body\n", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			http.Error(w, "unavailable\n", http.StatusInternalServerError)
			return
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, body, 0o644); err != nil {
			http.Error(w, "unavailable\n", http.StatusInternalServerError)
			return
		}
		if err := os.Rename(tmp, path); err != nil {
			http.Error(w, "unavailable\n", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("GET /", http.FileServer(noListing{http.Dir(root)}))
	srv := &http.Server{
		Addr: *addr, Handler: mux,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second,
	}
	if err := srv.ListenAndServeTLS(*cert, *key); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-origin: stopped")
		os.Exit(1)
	}
}

type noListing struct{ http.FileSystem }

func (n noListing) Open(name string) (http.File, error) {
	f, err := n.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if stat.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

func safePath(root, name string) (string, bool) {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return "", false
	}
	clean := filepath.Clean(name)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return "", false
	}
	full := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return full, true
}

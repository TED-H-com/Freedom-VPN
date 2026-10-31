//go:build installerfixture

package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// TEST ONLY central for the generated-config delivery boundary. The node issuer
// and server trust are deliberately different. Enrollment and signed state use
// the unchanged shared fixture. This is not the production fleet-update store.
func reportingControl(dir string, control *fakecontrol.Server) (string, []byte, func(), error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", nil, nil, err
	}
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		return "", nil, nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(control.CACertPEM()) {
		return "", nil, nil, fmt.Errorf("client CA")
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
		ClientAuth: tls.VerifyClientCertIfGiven, ClientCAs: pool,
	})
	if err != nil {
		return "", nil, nil, err
	}
	receipts := &resultReceiver{dir: filepath.Join(dir, "results"), control: control}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/agent/update-results", receipts.accept)
	mux.Handle("/", control)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(ln) }()
	return "https://" + ln.Addr().String(), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), func() { _ = server.Close() }, nil
}

type resultReceiver struct {
	mu      sync.Mutex
	dir     string
	control *fakecontrol.Server
}

func (s *resultReceiver) accept(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
		http.Error(w, "client certificate required", http.StatusUnauthorized)
		return
	}
	id, err := pki.Identity(r.TLS.PeerCertificates[0])
	if err != nil {
		http.Error(w, "identity", http.StatusUnauthorized)
		return
	}
	// Use the fixture's current enrollment/revocation fence as well as TLS proof.
	probe := r.Clone(r.Context())
	probe.Method = http.MethodGet
	probe.URL.Path = "/api/v1/agent/desired-state"
	recorder := httptest.NewRecorder()
	s.control.ServeHTTP(recorder, probe)
	if recorder.Code != http.StatusOK {
		http.Error(w, "identity fence", http.StatusForbidden)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	var result api.UpdateResult
	if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &result) != nil {
		http.Error(w, "body", http.StatusBadRequest)
		return
	}
	if result.NodeId != id.NodeID || result.IncarnationId != id.IncarnationID {
		http.Error(w, "identity mismatch", http.StatusForbidden)
		return
	}
	hash, err := canonjson.HashHex(raw, "payload_hash")
	if err != nil || hash != result.PayloadHash || result.Source == nil || *result.Source != api.UpdateResultSourceInstaller || result.JobId != installerjob.ID(result.RequestedReleaseId, result.ManifestSha256, result.IncarnationId) || r.Header.Get("Idempotency-Key") != result.ResultId.String() {
		http.Error(w, "receipt integrity", http.StatusConflict)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		http.Error(w, "store", 500)
		return
	}
	path := filepath.Join(s.dir, result.ResultId.String())
	ack := api.UpdateResultAck{Accepted: true, AppliedToCurrentObservation: true, ObservedUpdateGeneration: result.Generation, ReceivedAt: time.Now().UTC(), ResultId: result.ResultId}
	if stored, err := os.ReadFile(path + ".json"); err == nil {
		if !bytes.Equal(stored, raw) {
			http.Error(w, "conflict", http.StatusConflict)
			return
		}
		saved, err := os.ReadFile(path + ".ack.json")
		if err != nil || json.Unmarshal(saved, &ack) != nil {
			http.Error(w, "store", 500)
			return
		}
	} else if !os.IsNotExist(err) {
		http.Error(w, "store", 500)
		return
	} else {
		ackRaw, err := json.Marshal(ack)
		if err != nil || fsutil.WriteFileAtomic(path+".json", raw, 0600) != nil || fsutil.WriteFileAtomic(path+".ack.json", ackRaw, 0600) != nil {
			http.Error(w, "store", 500)
			return
		}
	}
	writeJSON(w, http.StatusAccepted, ack)
}

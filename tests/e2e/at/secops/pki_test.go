//go:build e2e && integration

package secops

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type identityProbe struct {
	mu    sync.Mutex
	seen  auth.Principal
	calls int
}

func (p *identityProbe) ConsumeAgentBootstrap(ctx context.Context, _ gen.ConsumeAgentBootstrapRequestObject) (gen.ConsumeAgentBootstrapResponseObject, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen, _ = auth.FromContext(ctx)
	p.calls++
	return nil, apierr.New(apierr.BootstrapNotReady, "TEST ONLY identity probe; no bootstrap secret issued.")
}

// Real TLS handshakes and real PKI/database revocation checks, with a deliberately
// inert bootstrap endpoint. This does not claim a production domain response.
func TestAT28_RealMTLSTwoNodesExpiredRevokedAndForgedIDs(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()
	provider, err := signing.InitializeFileProvider(filepath.Join(t.TempDir(), "vault"))
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := provider.Rotate(ctx, signing.NodeCA, 24*time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	der, err := pki.CreateDevCA(ctx, provider, keyID, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := pki.NewCA(ctx, provider, keyID, der, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	service, err := pki.NewService(pool, ca, provider, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	caSigner, err := provider.CertificateSigner(ctx, keyID)
	if err != nil {
		t.Fatal(err)
	}
	type identity struct {
		id     pki.NodeIdentity
		issued pki.Issued
		key    *ecdsa.PrivateKey
	}
	ids := make([]identity, 4)
	for i := range ids {
		id := &ids[i]
		id.id = pki.NodeIdentity{NodeID: uuid.New(), IncarnationID: uuid.New()}
		key, csr := pkitest.CSR(t)
		id.key = key
		id.issued, err = ca.Issue(ctx, id.id, csr)
		if err != nil {
			t.Fatal(err)
		}
		if i == 2 {
			// Sign an actually expired leaf. Never alter the host clock.
			template := *id.issued.Certificate
			template.NotBefore, template.NotAfter = time.Now().Add(-time.Hour), time.Now().Add(-time.Minute)
			expiredDER, e := x509.CreateCertificate(rand.Reader, &template, root, key.Public(), caSigner)
			if e != nil {
				t.Fatal(e)
			}
			id.issued.Certificate, err = x509.ParseCertificate(expiredDER)
			if err != nil {
				t.Fatal(err)
			}
		}
		err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if _, e := tx.Exec(ctx, `INSERT INTO nodes(id,lifecycle_state,provisioning_state) VALUES($1,'registered','ready')`, id.id.NodeID); e != nil {
				return e
			}
			parsed, e := pki.ParseCSR(csr)
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch) VALUES($1,$2,$3,'ubuntu','amd64')`, id.id.IncarnationID, id.id.NodeID, pki.Fingerprint(parsed)); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE nodes SET current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, id.id.NodeID, id.id.IncarnationID); e != nil {
				return e
			}
			_, e = pki.StoreCertificate(ctx, tx, id.id.IncarnationID, id.issued)
			return e
		})
		if err != nil {
			t.Fatal("isolated certificate fixture failed")
		}
	}
	if _, err = pool.Exec(ctx, `UPDATE node_certificates SET revoked_at=clock_timestamp() WHERE incarnation_id=$1`, ids[3].id.IncarnationID); err != nil {
		t.Fatal("certificate revocation failed")
	}
	probe := &identityProbe{}
	composed, err := httpapi.Compose(probe)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := httpapi.New(composed, httpapi.Options{Config: config.Defaults(), AgentListener: true, Agents: service})
	if err != nil {
		t.Fatal(err)
	}
	var access bytes.Buffer
	server := httptest.NewUnstartedServer(httpapi.Transport(slog.New(slog.NewJSONHandler(&access, nil)), handler))
	server.Config.ErrorLog = log.New(io.Discard, "", 0) // TLS errors can include certificate subjects.
	roots := x509.NewCertPool()
	roots.AddCert(root)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: roots}
	server.StartTLS()
	defer server.Close()
	serverRoots := x509.NewCertPool()
	serverRoots.AddCert(server.Certificate())
	var secretNeedles []string
	for i, id := range ids {
		t.Run([]string{"node_a", "node_b", "expired", "revoked"}[i], func(t *testing.T) {
			keyDER, e := x509.MarshalECPrivateKey(id.key)
			if e != nil {
				t.Fatal(e)
			}
			keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
			secretNeedles = append(secretNeedles, string(keyPEM))
			tr := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: serverRoots, Certificates: []tls.Certificate{{Certificate: [][]byte{id.issued.Certificate.Raw}, PrivateKey: id.key}}}}
			defer tr.CloseIdleConnections()
			client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
			req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/api/v1/agent/bootstrap", nil)
			req.Header.Set("X-Node-ID", ids[(i+1)%2].id.NodeID.String())
			resp, e := client.Do(req)
			if i == 2 {
				if e == nil {
					resp.Body.Close()
					t.Fatal("expired certificate passed TLS")
				}
				return
			}
			if e != nil {
				t.Fatal("unexpected TLS request failure")
			}
			body, e := io.ReadAll(resp.Body)
			resp.Body.Close()
			if e != nil {
				t.Fatal("read response")
			}
			if i == 3 {
				if resp.StatusCode != 403 || !bytes.Contains(body, []byte("IDENTITY_REVOKED")) {
					t.Fatal("revoked certificate reached domain")
				}
				return
			}
			if resp.StatusCode != 409 || !bytes.Contains(body, []byte("BOOTSTRAP_NOT_READY")) {
				t.Fatalf("identity probe status=%d", resp.StatusCode)
			}
			probe.mu.Lock()
			seen := probe.seen
			probe.mu.Unlock()
			if seen.NodeID != id.id.NodeID || seen.IncarnationID != id.id.IncarnationID {
				t.Fatal("header overrode certificate identity")
			}
		})
	}
	probe.mu.Lock()
	calls := probe.calls
	probe.mu.Unlock()
	if calls != 2 {
		t.Fatalf("domain calls=%d want=2", calls)
	}
	if err := provider.DenyIdentity(ctx, ids[0].id.NodeID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateAgent(ctx, ids[0].issued.Certificate); err == nil {
		t.Fatal("external revocation fence ignored")
	}
	for _, needle := range secretNeedles {
		if strings.Contains(access.String(), needle) {
			t.Fatal("private key reached access log")
		}
	}
	t.Log("two TLS identities preserved despite forged headers; expired TLS leaf refused; revoked serial and external identity fence refused; private keys generated at runtime")
}

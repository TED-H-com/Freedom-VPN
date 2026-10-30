package main

// TEST ONLY wire client for harness-owned enrollment inputs. No central or
// agent database is edited; all private files stay in the selected container.
import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

const harnessRoot = "/var/lib/vpn-node/e2e/harness/"

func harnessPath(p string) bool {
	return strings.HasPrefix(p, harnessRoot) && filepath.Clean(p) == p && len(p) < 256
}
func enrollmentGenerate() error {
	var in struct {
		Directory string `json:"directory"`
	}
	if json.NewDecoder(os.Stdin).Decode(&in) != nil || !harnessPath(in.Directory) {
		return fmt.Errorf("invalid private directory")
	}
	if err := os.Mkdir(in.Directory, 0700); err != nil {
		return err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	if err = os.WriteFile(in.Directory+"/key.pem", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); err != nil {
		return err
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{SignatureAlgorithm: x509.ECDSAWithSHA256}, key)
	if err != nil {
		return err
	}
	req := api.EnrollRequest{EnrollmentRequestId: uuid.New(), NodeIncarnationId: uuid.New(), CsrPem: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})), Platform: api.NodePlatform{Arch: api.Architecture(runtime.GOARCH), Os: "ubuntu", OsVersion: "24.04"}}
	raw, err := os.ReadFile("/etc/vpn-node/agent-enroll.json")
	if err != nil {
		return err
	}
	var cfg struct {
		Release string `json:"agent_release"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return fmt.Errorf("agent config")
	}
	req.AgentRelease = cfg.Release
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	var fields map[string]any
	_ = json.Unmarshal(b, &fields)
	delete(fields, "proof")
	b, err = canonjson.Marshal(fields)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	sig, err := ecdsa.SignASN1(rand.Reader, key, sum[:])
	if err != nil {
		return err
	}
	req.Proof = base64.StdEncoding.EncodeToString(sig)
	b, err = json.Marshal(req)
	if err != nil {
		return err
	}
	return os.WriteFile(in.Directory+"/request.json", b, 0600)
}
func enrollmentJoin() error {
	var in struct {
		KeyFile, RequestFile, TokenFile string
		DiscardResponse, Join           bool
	}
	if json.NewDecoder(os.Stdin).Decode(&in) != nil || !harnessPath(in.KeyFile) || !harnessPath(in.RequestFile) || !harnessPath(in.TokenFile) {
		return fmt.Errorf("invalid private input paths")
	}
	raw, err := os.ReadFile(in.RequestFile)
	if err != nil || len(raw) > 65536 {
		return fmt.Errorf("request unavailable")
	}
	var req api.EnrollRequest
	if json.Unmarshal(raw, &req) != nil {
		return fmt.Errorf("request invalid")
	}
	// Validate key/CSR matching before any secret-bearing network request.
	keyRaw, err := os.ReadFile(in.KeyFile)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(keyRaw)
	if block == nil {
		return fmt.Errorf("key invalid")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	key, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("P256 key required")
	}
	block, _ = pem.Decode([]byte(req.CsrPem))
	if block == nil {
		return fmt.Errorf("CSR invalid")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil || csr.CheckSignature() != nil {
		return fmt.Errorf("CSR invalid")
	}
	pub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok || !pub.Equal(&key.PublicKey) {
		return fmt.Errorf("CSR key mismatch")
	}
	token, err := os.ReadFile(in.TokenFile)
	if err != nil {
		return err
	}
	roots := x509.NewCertPool()
	ca, err := os.ReadFile("/etc/vpn-node/ca.pem")
	if err != nil || !roots.AppendCertsFromPEM(ca) {
		return fmt.Errorf("CA unavailable")
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, "POST", "https://portal.example.invalid:8443/api/v1/agent/enroll", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Enrollment "+strings.TrimSpace(string(token)))
	request.Header.Set("Idempotency-Key", req.EnrollmentRequestId.String())
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("enrollment transport failed")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	out := map[string]any{"enroll_http": resp.StatusCode, "request_sha256": hex.EncodeToString(sum[:]), "response_discarded": in.DiscardResponse, "joined": false}
	emit := func() error { return json.NewEncoder(os.Stdout).Encode(out) }
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		var fail api.ErrorEnvelope
		if json.Unmarshal(body, &fail) == nil {
			out["error_code"] = fail.Error.Code
		}
		return emit()
	}
	// Intentional crash point: consume the wire response but retain no certificate
	// or bootstrap; retry uses exactly the original persisted request bytes.
	if in.DiscardResponse {
		return emit()
	}
	var enrolled api.EnrollResponse
	if json.Unmarshal(body, &enrolled) != nil || enrolled.NodeIncarnationId != req.NodeIncarnationId {
		return fmt.Errorf("enrollment response mismatch")
	}
	out["node_id"] = enrolled.NodeId
	out["incarnation_id"] = enrolled.NodeIncarnationId
	dir := filepath.Dir(in.KeyFile)
	if err = os.WriteFile(dir+"/cert.pem", []byte(enrolled.CertificatePem), 0600); err != nil {
		return err
	}
	pair, err := tls.X509KeyPair([]byte(enrolled.CertificatePem), keyRaw)
	if err != nil {
		return err
	}
	if !in.Join {
		return emit()
	}
	mtls := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: []tls.Certificate{pair}}}
	defer mtls.CloseIdleConnections()
	client = &http.Client{Transport: mtls, Timeout: 15 * time.Second}
	var boot api.AgentBootstrap
	for ctx.Err() == nil {
		request, _ = http.NewRequestWithContext(ctx, "POST", "https://agent.example.invalid:9443/api/v1/agent/bootstrap", nil)
		request.Header.Set("Idempotency-Key", uuid.NewString())
		resp, err = client.Do(request)
		if err != nil {
			return fmt.Errorf("bootstrap transport failed")
		}
		body, err = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			return err
		}
		out["bootstrap_http"] = resp.StatusCode
		if resp.StatusCode == 200 {
			if json.Unmarshal(body, &boot) != nil || boot.SetupKey == "" {
				return fmt.Errorf("bootstrap invalid")
			}
			break
		}
		var fail api.ErrorEnvelope
		_ = json.Unmarshal(body, &fail)
		if string(fail.Error.Code) != "BOOTSTRAP_NOT_READY" {
			out["error_code"] = fail.Error.Code
			return emit()
		}
		time.Sleep(time.Second)
	}
	if ctx.Err() != nil {
		return fmt.Errorf("bootstrap deadline")
	}
	path := dir + "/setup.key"
	if err = os.WriteFile(path, []byte(boot.SetupKey), 0600); err != nil {
		return err
	}
	defer os.Remove(path)
	// A harness identity uses the waiting slot's real native daemon. It deliberately
	// does not import fabricated local agent state or start a synthetic agent.
	live := exec.CommandContext(ctx, "netbird", "status", "--check", "live").Run()
	if live != nil {
		log, err := os.OpenFile(dir+"/netbird.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		cmd := exec.Command("netbird", "service", "run", "--log-level", "warn", "--log-file", "console")
		cmd.Stdout = log
		cmd.Stderr = log
		err = cmd.Start()
		log.Close()
		if err != nil {
			return fmt.Errorf("daemon start failed")
		}
		_ = cmd.Process.Release()
		for i := 0; i < 30; i++ {
			if exec.CommandContext(ctx, "netbird", "status", "--check", "live").Run() == nil {
				break
			}
			time.Sleep(time.Second)
		}
	}
	err = exec.CommandContext(ctx, "netbird", "up", "--setup-key-file", path, "--management-url", "https://netbird.example.invalid:443", "--interface-name", "fvmb0", "--disable-dns", "--disable-ipv6", "--disable-firewall", "--log-level", "warn").Run()
	out["joined"] = err == nil
	if err != nil {
		out["error_code"] = "NATIVE_JOIN_FAILED"
	}
	return emit()
}

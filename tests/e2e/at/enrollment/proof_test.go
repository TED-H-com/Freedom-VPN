package enrollment

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestAT06_CSRProofAndIndependentKeys(t *testing.T) {
	a, err := newHolder()
	if err != nil {
		t.Fatal(err)
	}
	b, err := newHolder()
	if err != nil {
		t.Fatal(err)
	}
	if a.fingerprint == b.fingerprint || a.incarnation == b.incarnation {
		t.Fatal("holders share identity")
	}
	var doc map[string]any
	if err = json.Unmarshal(a.body, &doc); err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode([]byte(doc["csr_pem"].(string)))
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil || csr.CheckSignature() != nil {
		t.Fatal("invalid CSR self-signature")
	}
	proof, err := base64.StdEncoding.DecodeString(doc["proof"].(string))
	if err != nil {
		t.Fatal(err)
	}
	delete(doc, "proof")
	canonical, err := canonjson.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	pub := csr.PublicKey.(*ecdsa.PublicKey)
	if !ecdsa.VerifyASN1(pub, digest[:], proof) {
		t.Fatal("proof does not bind full request")
	}
	doc["node_incarnation_id"] = b.incarnation.String()
	canonical, err = canonjson.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	digest = sha256.Sum256(canonical)
	if ecdsa.VerifyASN1(pub, digest[:], proof) {
		t.Fatal("proof accepts another incarnation")
	}
}

func TestAT06_ResponseActuallyLostBeforeCaller(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "identical request" {
			t.Error("request changed")
		}
		requests++
		status := http.StatusOK
		if requests == 1 {
			status = http.StatusCreated
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"node_id":"00000000-0000-4000-8000-000000000001","node_incarnation_id":"00000000-0000-4000-8000-000000000002","certificate_pem":"TEST ONLY public certificate"}`)
	}))
	defer server.Close()
	fault := &dropSuccess{base: server.Client().Transport}
	client := &http.Client{Transport: fault}
	resp, err := client.Post(server.URL, "application/json", strings.NewReader("identical request"))
	if resp != nil || !errors.Is(err, errDropped) || !fault.dropped || fault.status != 201 {
		t.Fatal("success reached caller or was not observed")
	}
	resp, err = client.Post(server.URL, "application/json", strings.NewReader("identical request"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || requests != 2 || fault.certificateHash != hash([]byte("TEST ONLY public certificate")) {
		t.Fatal("retry failed or observer lost original proof")
	}
}

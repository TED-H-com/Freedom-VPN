package enrollment_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestAT06_ProofVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/proof-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Label       string            `json:"label"`
		Request     api.EnrollRequest `json:"request"`
		Canonical   string            `json:"canonical"`
		SHA256      string            `json:"sha256"`
		Fingerprint string            `json:"public_key_sha256"`
	}
	if err = json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if v.Label != "TEST ONLY - NOT A SECRET" {
		t.Fatal("test vector label missing")
	}
	_, hash, err := enrollment.VerifyProof(v.Request)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(hash) != v.SHA256 {
		t.Fatal("wrong proof hash")
	}
	canonical, err := enrollment.ProofInput(v.Request)
	if err != nil || string(canonical) != v.Canonical {
		t.Fatal("canonical bytes differ", err)
	}
	csr, _, err := enrollment.VerifyProof(v.Request)
	if err != nil {
		t.Fatal(err)
	}
	fp := sha256.Sum256(csr.RawSubjectPublicKeyInfo)
	if hex.EncodeToString(fp[:]) != v.Fingerprint {
		t.Fatal("fingerprint differs")
	}
	mutated := v.Request
	mutated.Platform.Arch = "arm64"
	if _, _, err = enrollment.VerifyProof(mutated); err == nil {
		t.Fatal("modified signed field accepted")
	}
	mutated = v.Request
	mutated.Proof += "\n"
	if _, _, err = enrollment.VerifyProof(mutated); err == nil {
		t.Fatal("noncanonical base64 accepted")
	}
	other, _ := pkitest.Request(t)
	mutated = v.Request
	mutated.CsrPem = other.CsrPem
	if _, _, err = enrollment.VerifyProof(mutated); err == nil {
		t.Fatal("proof accepted for other public key")
	}
}

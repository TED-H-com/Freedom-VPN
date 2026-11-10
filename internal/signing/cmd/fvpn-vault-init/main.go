// fvpn-vault-init initializes a fresh local FileProvider ceremony directory.
// FileProvider's development and whole-directory rollback limits still apply.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type window struct {
	Validity string `json:"validity"`
	Overlap  string `json:"overlap"`
}
type caConfig struct {
	CommonName string `json:"common_name"`
	Validity   string `json:"validity"`
}
type config struct {
	Provider         string                     `json:"provider"`
	ReleaseTrustFile string                     `json:"release_trust_file"`
	ReleaseKeyID     string                     `json:"release_key_id"`
	CA               caConfig                   `json:"ca"`
	Purposes         map[signing.Purpose]window `json:"purposes"`
}
type durations struct{ validity, overlap time.Duration }
type prepared struct {
	config     config
	start      time.Time
	caValidity time.Duration
	windows    map[signing.Purpose]durations
	release    envelope.TrustedKey
}
type keySummary struct {
	ID           string          `json:"key_id"`
	Purpose      signing.Purpose `json:"purpose"`
	NotBefore    time.Time       `json:"not_before"`
	NotAfter     time.Time       `json:"not_after"`
	PublicSHA256 string          `json:"public_sha256,omitempty"`
}
type receipt struct {
	Kind              string       `json:"kind"`
	Provider          string       `json:"provider"`
	CompletedAt       time.Time    `json:"completed_at"`
	Vault             string       `json:"vault"`
	CACertificate     string       `json:"ca_certificate"`
	CAKeyID           string       `json:"ca_key_id"`
	CADERSHA256       string       `json:"ca_der_sha256"`
	CANotBefore       time.Time    `json:"ca_not_before"`
	CANotAfter        time.Time    `json:"ca_not_after"`
	TrustBundle       string       `json:"trust_bundle"`
	TrustBundleSHA256 string       `json:"trust_bundle_sha256"`
	LedgerRevision    decstr.U64   `json:"ledger_revision"`
	Keys              []keySummary `json:"keys"`
	Checks            []string     `json:"checks"`
}

func purposes() []signing.Purpose {
	return []signing.Purpose{signing.NodeCA, signing.BootstrapSecret, signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch}
}
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fvpn-vault-init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("config", "", "explicit ceremony JSON configuration")
	out := fs.String("out", "", "new output directory; parent must already exist")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || *path == "" || *out == "" {
		fmt.Fprintln(stderr, "usage: fvpn-vault-init -config CONFIG.json -out NEW_DIRECTORY")
		return 1
	}
	raw, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(stderr, "fvpn-vault-init: cannot read configuration")
		return 1
	}
	p, err := prepare(raw)
	if err == nil {
		err = initialize(context.Background(), p, *out, writeExclusive)
	}
	if err != nil {
		fmt.Fprintln(stderr, "fvpn-vault-init:", err)
		return 1
	}
	if _, err := fmt.Fprintln(stdout, "FileProvider ceremony complete; inspect receipt.json before deployment."); err != nil {
		fmt.Fprintln(stderr, "fvpn-vault-init: output failed; completed receipt remains in destination")
		return 1
	}
	return 0
}

func duration(raw string, zero bool) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 || (!zero && d < time.Second) || d%time.Second != 0 {
		return 0, errors.New("durations must be explicit whole seconds, valid and in range")
	}
	return d, nil
}
func prepare(raw []byte) (prepared, error) {
	var p prepared
	if _, err := canonjson.Canonicalize(raw); err != nil {
		return p, errors.New("invalid ceremony JSON")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&p.config); err != nil {
		return p, errors.New("invalid ceremony configuration")
	}
	c := p.config
	if c.Provider != "file" || !filepath.IsAbs(c.ReleaseTrustFile) || c.ReleaseKeyID == "" || len(c.ReleaseKeyID) > 128 || len(c.Purposes) != len(purposes()) {
		return p, errors.New("explicit file provider, absolute public trust path, release key ID and six purposes required")
	}
	if !utf8.ValidString(c.CA.CommonName) || strings.TrimSpace(c.CA.CommonName) != c.CA.CommonName || c.CA.CommonName == "" || len(c.CA.CommonName) > 64 || strings.ContainsFunc(c.CA.CommonName, unicode.IsControl) {
		return p, errors.New("explicit CA common name required (1 to 64 bytes, no control characters)")
	}
	p.windows = make(map[signing.Purpose]durations)
	for _, purpose := range purposes() {
		w, ok := c.Purposes[purpose]
		if !ok {
			return p, errors.New("exactly the six central purposes are required")
		}
		validity, err := duration(w.Validity, false)
		if err != nil {
			return p, err
		}
		overlap, err := duration(w.Overlap, true)
		if err != nil {
			return p, err
		}
		if overlap >= validity {
			return p, errors.New("purpose overlap must be shorter than validity")
		}
		p.windows[purpose] = durations{validity, overlap}
	}
	var err error
	p.caValidity, err = duration(c.CA.Validity, false)
	if err != nil {
		return p, err
	}
	if p.caValidity > p.windows[signing.NodeCA].validity {
		return p, errors.New("CA certificate validity exceeds node-ca key validity")
	}
	trustRaw, err := os.ReadFile(c.ReleaseTrustFile)
	if err != nil {
		return p, errors.New("cannot read release public trust")
	}
	ts, err := envelope.ParseNodeTrustStoreJSON(trustRaw)
	if err != nil || ts.Len() != 1 {
		return p, errors.New("exactly one valid public release root is required")
	}
	k, ok := ts.Lookup(c.ReleaseKeyID)
	now := time.Now().UTC()
	p.start = now.Truncate(time.Second)
	if !ok || len(k.Purposes) != 1 || k.Purposes[0] != envelope.PurposeRelease || k.Revoked || now.Before(k.NotBefore) || !now.Before(k.NotAfter) {
		return p, errors.New("selected release root must be current, nonrevoked and release-only")
	}
	p.release = k
	return p, nil
}

type outputWriter func(string, []byte, os.FileMode) error

func initialize(ctx context.Context, p prepared, destination string, write outputWriter) error {
	if ctx.Err() != nil {
		return errors.New("ceremony canceled")
	}
	// Recheck time immediately before the first mutation, including queued calls.
	now := time.Now().UTC()
	if now.Before(p.release.NotBefore) || !now.Before(p.release.NotAfter) || !now.Before(p.start.Add(p.caValidity)) {
		return errors.New("ceremony or release validity window elapsed")
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return errors.New("destination must be new and its parent must exist and be writable")
	}
	fail := func(stage string) error {
		return fmt.Errorf("%s failed; partial destination retained, inspect it before any further action", stage)
	}
	vaultDir := filepath.Join(destination, "vault")
	vault, err := signing.InitializeFileProvider(vaultDir)
	if err != nil {
		return fail("vault initialization")
	}
	ids := make(map[signing.Purpose]string)
	for _, purpose := range purposes() {
		w := p.windows[purpose]
		ids[purpose], err = vault.Rotate(ctx, purpose, w.validity, w.overlap)
		if err != nil {
			return fail("purpose creation")
		}
	}
	der, err := createCA(ctx, vault, ids[signing.NodeCA], p)
	if err != nil {
		return fail("CA creation")
	}
	if _, err = pki.NewCA(ctx, vault, ids[signing.NodeCA], der, time.Second); err != nil {
		return fail("CA validation")
	}
	if err = vault.PinReleaseRoot(ctx, p.release); err != nil {
		return fail("release root pinning")
	}
	bundle, err := vault.TrustBundle(ctx)
	if err != nil || bundle.Len() != 4 {
		return fail("node trust export")
	}
	trustRaw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fail("node trust serialization")
	}
	trustRaw = append(trustRaw, '\n')
	var state struct {
		Revision decstr.U64          `json:"revision"`
		Keys     []signing.KeyRecord `json:"keys"`
	}
	ledgerRaw, err := os.ReadFile(filepath.Join(vaultDir, "state.json"))
	if err != nil {
		return fail("public ledger read")
	}
	if err = json.Unmarshal(ledgerRaw, &state); err != nil || len(state.Keys) != 7 {
		return fail("public ledger validation")
	}
	if err = checkKeys(ctx, vault, state.Keys); err != nil {
		return fail("key-use verification")
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil || cert.CheckSignatureFrom(cert) != nil {
		return fail("CA signature verification")
	}
	caRaw := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	for _, output := range []struct {
		name string
		raw  []byte
	}{{"agent-ca.pem", caRaw}, {"trust.json", trustRaw}} {
		path := filepath.Join(destination, output.name)
		if err = write(path, output.raw, 0600); err != nil {
			return fail("public output write")
		}
		got, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(got, output.raw) {
			return fail("public output read-back")
		}
	}
	if ctx.Err() != nil {
		return fail("ceremony cancellation")
	}
	completedAt := time.Now().UTC()
	if err = validateWindows(completedAt, cert, state.Keys, ids[signing.NodeCA]); err != nil {
		return fail("completion validity check")
	}
	r := receipt{Kind: "fvpn-file-vault-init-v1", Provider: "file", CompletedAt: completedAt, Vault: "vault", CACertificate: "agent-ca.pem", CAKeyID: ids[signing.NodeCA], CADERSHA256: digest(der), CANotBefore: cert.NotBefore, CANotAfter: cert.NotAfter, TrustBundle: "trust.json", TrustBundleSHA256: digest(trustRaw), LedgerRevision: state.Revision, Checks: []string{"central-signatures", "bootstrap-roundtrip", "ca-signature-and-provider-match", "public-output-read-back"}}
	for _, k := range state.Keys {
		v := keySummary{ID: k.ID, Purpose: k.Purpose, NotBefore: k.NotBefore, NotAfter: k.NotAfter}
		if len(k.Public) > 0 {
			v.PublicSHA256 = digest(k.Public)
		}
		r.Keys = append(r.Keys, v)
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fail("receipt serialization")
	}
	if err = write(filepath.Join(destination, "receipt.json"), append(raw, '\n'), 0600); err != nil {
		return fail("completion receipt write")
	}
	return nil
}
func validateWindows(now time.Time, cert *x509.Certificate, keys []signing.KeyRecord, caKeyID string) error {
	if now.Before(cert.NotBefore) || !now.Before(cert.NotAfter) {
		return errors.New("CA outside validity window")
	}
	foundCA := false
	for _, k := range keys {
		if k.State != "active" || now.Before(k.NotBefore) || !now.Before(k.NotAfter) {
			return errors.New("key outside active window")
		}
		if k.Purpose == signing.NodeCA && k.ID == caKeyID {
			foundCA = true
			if cert.NotAfter.After(k.NotAfter) {
				return errors.New("CA outlives actual provider key")
			}
		}
	}
	if !foundCA {
		return errors.New("CA key reference missing")
	}
	return nil
}

func createCA(ctx context.Context, vault *signing.FileProvider, keyID string, p prepared) ([]byte, error) {
	signer, err := vault.CertificateSigner(ctx, keyID)
	if err != nil {
		return nil, err
	}
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, new(big.Int).Sub(limit, big.NewInt(1)))
	if err != nil {
		return nil, err
	}
	serial.Add(serial, big.NewInt(1))
	t := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: p.config.CA.CommonName}, NotBefore: p.start, NotAfter: p.start.Add(p.caValidity), IsCA: true, BasicConstraintsValid: true, MaxPathLenZero: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, SignatureAlgorithm: x509.ECDSAWithSHA256}
	return x509.CreateCertificate(rand.Reader, t, t, signer.Public(), signer)
}
func checkKeys(ctx context.Context, vault *signing.FileProvider, keys []signing.KeyRecord) error {
	var trusted []envelope.TrustedKey
	for _, k := range keys {
		purpose := envelope.Purpose(k.Purpose)
		if purpose.Valid() {
			trusted = append(trusted, envelope.TrustedKey{KeyID: k.ID, PublicKey: k.Public, Purposes: []envelope.Purpose{purpose}, NotBefore: k.NotBefore, NotAfter: k.NotAfter})
		}
	}
	ts, err := envelope.NewTrustStore(trusted...)
	if err != nil {
		return err
	}
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
		e, err := vault.SignEnvelope(ctx, purpose, map[string]any{"kind": "fvpn-vault-init-check-v1"})
		if err != nil {
			return err
		}
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, _, err = envelope.Verify(raw, ts, envelope.Purpose(purpose), time.Now()); err != nil {
			return err
		}
	}
	probe := []byte("fvpn-vault-init-check-v1")
	sealed, err := vault.Seal(ctx, probe, probe)
	if err != nil {
		return err
	}
	opened, err := vault.Open(ctx, sealed, probe)
	if err != nil || !bytes.Equal(opened, probe) {
		return errors.New("bootstrap roundtrip failed")
	}
	return nil
}
func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

// Publish only a fully written and closed file. Link refuses an existing name.
// This is not a transaction across outputs or a crash-durability guarantee.
func writeExclusive(path string, raw []byte, mode os.FileMode) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".vault-init-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = f.Chmod(mode); err != nil {
		return err
	}
	if _, err = f.Write(raw); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Link(f.Name(), path)
}

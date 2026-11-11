package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type keySummary struct {
	ID           string          `json:"key_id"`
	Purpose      signing.Purpose `json:"purpose"`
	State        string          `json:"state"`
	NotBefore    time.Time       `json:"not_before"`
	NotAfter     time.Time       `json:"not_after"`
	RevokedAt    *time.Time      `json:"revoked_at,omitempty"`
	PublicSHA256 string          `json:"public_sha256,omitempty"`
	WindowStatus string          `json:"window_status"`
	KeyUseStatus string          `json:"key_use_status"`
}
type purposeSummary struct {
	Purpose        signing.Purpose `json:"purpose"`
	MetadataStatus string          `json:"metadata_status"`
	KeyUseStatus   string          `json:"key_use_status"`
	Result         string          `json:"result"`
}
type inspection struct {
	Kind                string           `json:"kind"`
	Provider            string           `json:"provider"`
	InspectedAt         time.Time        `json:"inspected_at"`
	LedgerRevision      decstr.U64       `json:"ledger_revision"`
	ReleaseRevision     decstr.U64       `json:"release_revision"`
	DeniedIdentityCount int              `json:"denied_identity_count"`
	KeyUseChecked       bool             `json:"key_use_checked"`
	Keys                []keySummary     `json:"keys"`
	Purposes            []purposeSummary `json:"purposes"`
	Result              string           `json:"result"`
}

func purposes() []signing.Purpose {
	return []signing.Purpose{signing.NodeCA, signing.BootstrapSecret, signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch, signing.Release}
}
func sortedKeys(keys []signing.KeyRecord) []signing.KeyRecord {
	out := append([]signing.KeyRecord(nil), keys...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func windowStatus(k signing.KeyRecord, now time.Time) string {
	if now.Before(k.NotBefore) {
		return "not_yet_valid"
	}
	if !now.Before(k.NotAfter) {
		return "expired"
	}
	return "current"
}
func inspect(ctx context.Context, p signing.Provider, snapshot signing.FileSnapshot, now time.Time, check bool) inspection {
	r := inspection{Kind: "fvpn-file-vault-inspect-v1", Provider: "file", InspectedAt: now,
		LedgerRevision: snapshot.Revision, ReleaseRevision: snapshot.ReleaseRevision,
		DeniedIdentityCount: snapshot.DeniedIdentityCount, KeyUseChecked: check,
		Keys: []keySummary{}, Purposes: []purposeSummary{}, Result: "pass"}
	for _, k := range sortedKeys(snapshot.Keys) {
		v := keySummary{ID: k.ID, Purpose: k.Purpose, State: k.State, NotBefore: k.NotBefore.UTC(), NotAfter: k.NotAfter.UTC(), WindowStatus: windowStatus(k, now), KeyUseStatus: "not_checked"}
		if k.RevokedAt != nil {
			at := k.RevokedAt.UTC()
			v.RevokedAt = &at
		}
		if len(k.Public) != 0 {
			v.PublicSHA256 = digest(k.Public)
		}
		if check {
			v.KeyUseStatus = "not_applicable"
			if k.Purpose != signing.Release && k.State == "active" && v.WindowStatus == "current" {
				v.KeyUseStatus = "passed"
				if checkKey(ctx, p, k, now) != nil {
					v.KeyUseStatus = "failed"
				}
			}
		}
		r.Keys = append(r.Keys, v)
	}
	for _, purpose := range purposes() {
		s := purposeSummary{Purpose: purpose, MetadataStatus: metadataStatus(r.Keys, purpose), KeyUseStatus: "not_checked", Result: "pass"}
		if check {
			s.KeyUseStatus = "not_applicable"
			for _, k := range r.Keys {
				if k.Purpose == purpose && k.State == "active" && k.WindowStatus == "current" && purpose != signing.Release {
					s.KeyUseStatus = k.KeyUseStatus
				}
			}
		}
		if s.MetadataStatus != "pass" || (check && purpose != signing.Release && s.KeyUseStatus != "passed") {
			s.Result, r.Result = "fail", "fail"
		}
		r.Purposes = append(r.Purposes, s)
	}
	return r
}
func metadataStatus(keys []keySummary, purpose signing.Purpose) string {
	revoked, expired, future := false, false, false
	for _, k := range keys {
		if k.Purpose != purpose {
			continue
		}
		if k.State == "revoked" {
			revoked = true
			continue
		}
		if purpose != signing.Release && k.State != "active" {
			continue
		}
		switch k.WindowStatus {
		case "current":
			return "pass"
		case "not_yet_valid":
			future = true
		case "expired":
			expired = true
		}
	}
	if future {
		return "not_yet_valid"
	}
	if expired {
		return "expired"
	}
	if revoked {
		return "revoked"
	}
	if purpose == signing.Release {
		return "no_key"
	}
	return "no_active_key"
}

// Probes are domain separated and deliberately lack every consumer's required
// payload/schema fields (§16.2). They never leave memory; no certificate is made.
func checkKey(ctx context.Context, p signing.Provider, k signing.KeyRecord, now time.Time) error {
	const domain = "fvpn-file-vault-inspect-probe-v1"
	switch k.Purpose {
	case signing.BootstrapSecret:
		plain, aad := []byte(domain+":plaintext"), []byte(domain+":aad:"+k.ID)
		sealed, err := p.Seal(ctx, plain, aad)
		if err != nil {
			return err
		}
		if sealed.KeyID != k.ID {
			return errors.New("probe key mismatch")
		}
		opened, err := p.Open(ctx, sealed, aad)
		if err != nil {
			return err
		}
		if !bytes.Equal(opened, plain) {
			return errors.New("probe mismatch")
		}
		return nil
	case signing.NodeCA:
		signer, err := p.CertificateSigner(ctx, k.ID)
		if err != nil {
			return err
		}
		public, err := x509.ParsePKIXPublicKey(k.Public)
		if err != nil {
			return err
		}
		pub, ok := public.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("probe public key invalid")
		}
		got, err := x509.MarshalPKIXPublicKey(signer.Public())
		if err != nil || !bytes.Equal(got, k.Public) {
			return errors.New("probe key mismatch")
		}
		hash := sha256.Sum256([]byte(domain + ":ca-digest:" + k.ID))
		sig, err := signer.Sign(rand.Reader, hash[:], crypto.SHA256)
		if err != nil {
			return err
		}
		if !ecdsa.VerifyASN1(pub, hash[:], sig) {
			return errors.New("probe signature invalid")
		}
		return nil
	default:
		purpose := envelope.Purpose(k.Purpose)
		ts, err := envelope.NewTrustStore(envelope.TrustedKey{KeyID: k.ID, PublicKey: k.Public, Purposes: []envelope.Purpose{purpose}, NotBefore: k.NotBefore, NotAfter: k.NotAfter})
		if err != nil {
			return err
		}
		e, err := p.SignEnvelope(ctx, k.Purpose, map[string]string{"kind": domain, "purpose": string(k.Purpose), "key_id": k.ID})
		if err != nil {
			return err
		}
		if len(e.Signatures) != 1 || e.Signatures[0].KeyID != k.ID {
			return errors.New("probe key mismatch")
		}
		raw, err := e.MarshalCanonical()
		if err != nil {
			return err
		}
		_, id, err := envelope.Verify(raw, ts, purpose, now)
		if err != nil {
			return err
		}
		if id != k.ID {
			return errors.New("probe key mismatch")
		}
		return nil
	}
}
func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func document(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

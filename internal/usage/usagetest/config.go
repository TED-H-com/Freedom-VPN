// Package usagetest provisions TEST ONLY usage-domain settings and an ephemeral
// file signing provider for composition and binary tests. It has no dependency on
// either application's composition root.
package usagetest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type Fixture struct {
	Usage            config.UsageConfig
	Path             string
	Provider         *signing.FileProvider
	CentralTrust     *envelope.TrustStore
	RecoveryKeyID    string
	LedgerEpochKeyID string
}

// New creates an ephemeral signing provider under t.TempDir, rotates the
// Recovery and LedgerEpoch purposes, and writes a TEST ONLY usage configuration
// file pointing to that provider. It never changes process environment or
// supplies a production default.
func New(t testing.TB) Fixture {
	t.Helper()
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "usage-vault")
	vault, err := signing.InitializeFileProvider(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	recKeyID, err := vault.Rotate(t.Context(), signing.Recovery, 48*time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	epochKeyID, err := vault.Rotate(t.Context(), signing.LedgerEpoch, 48*time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	stateRaw, err := os.ReadFile(filepath.Join(vaultPath, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Keys []signing.KeyRecord `json:"keys"`
	}
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		t.Fatal(err)
	}
	var trusted []envelope.TrustedKey
	for _, k := range state.Keys {
		trusted = append(trusted, envelope.TrustedKey{
			KeyID:     k.ID,
			PublicKey: k.Public,
			Purposes:  []envelope.Purpose{envelope.Purpose(k.Purpose)},
			NotBefore: k.NotBefore,
			NotAfter:  k.NotAfter,
			Revoked:   k.State == "revoked",
		})
	}
	centralTrust, err := envelope.NewTrustStore(trusted...)
	if err != nil {
		t.Fatal(err)
	}
	u := config.UsageConfig{
		SigningConfigFile:     vaultPath,
		RawUsageRetentionDays: 90,
		RollupRetentionMonths: 24,
		AuditRetentionDays:    365,
		ProjectionBatchSize:   256,
	}
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "usage.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return Fixture{
		Usage:            u,
		Path:             path,
		Provider:         vault,
		CentralTrust:     centralTrust,
		RecoveryKeyID:    recKeyID,
		LedgerEpochKeyID: epochKeyID,
	}
}

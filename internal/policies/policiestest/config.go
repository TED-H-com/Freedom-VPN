// Package policiestest provisions TEST ONLY policies-domain settings for
// composition and binary tests. It has no dependency on either application's
// composition root.
package policiestest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

type Fixture struct {
	Policies config.PoliciesConfig
	Path     string
}

// New writes an explicit TEST ONLY policies configuration file under t.TempDir.
// It never changes process environment or supplies a production default.
func New(t testing.TB) Fixture {
	t.Helper()
	dir := t.TempDir()
	p := config.PoliciesConfig{
		FUPMode:            "",
		FUPSweepSeconds:    60,
		FUPBatchSize:       64,
		DNSUpstreamProfile: "TEST ONLY",
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "policies.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return Fixture{Policies: p, Path: path}
}

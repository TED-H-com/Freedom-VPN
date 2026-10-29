package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSEC01_ConfigValidationAndRedaction(t *testing.T) {
	c := Defaults()
	if c.Validate() == nil {
		t.Fatal("missing database accepted")
	}
	c.DatabaseURLFile = "secrets/database"
	c.OIDCSecretFile = "secrets/oidc"
	c.TLSKeyFile = "secrets/key"
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(c.Effective())
	if strings.Contains(string(b), "secrets/") {
		t.Fatal("secret paths printed")
	}
	c.WorkerJobSeconds = c.WorkerLeaseSeconds
	if c.Validate() == nil {
		t.Fatal("lease permits overlap")
	}
}
func TestAPI01_ConfigFileAndEnv(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	if e := os.WriteFile(p, []byte(`{"database_url_file":"database","session_idle_seconds":10}`), 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("FVPN_SESSION_IDLE_SECONDS", "20")
	c, e := Load(p)
	if e != nil || c.SessionIdleSeconds != 20 || c.IdempotencySeconds != 604800 {
		t.Fatal(c, e)
	}
	if e = os.WriteFile(p, []byte(`{"unknown":true}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Load(p); e == nil {
		t.Fatal("unknown setting accepted")
	}
}

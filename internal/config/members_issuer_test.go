package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestD052_F09MemberLoaderRequiresCanonicalCentralIssuer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "members.json")
	raw, err := json.Marshal(validMemberSettings())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadMembersForIssuer(path, "https://issuer.example.invalid"); err != nil {
		t.Fatal(err)
	}
	for _, issuer := range []string{"https://ISSUER.EXAMPLE.INVALID", "https://issuer.example.invalid/", "https://issuer.example.invalid:443", "https://issuer.example.invalid?private"} {
		_, err = LoadMembersForIssuer(path, issuer)
		var setting *SettingError
		if !errors.As(err, &setting) || setting.Name != "oidc_issuer" {
			t.Fatal("noncanonical issuer accepted", err)
		}
	}
}

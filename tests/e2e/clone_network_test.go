//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRewriteLabHostReplacesAgentOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lab-hosts")
	original := "172.28.30.30 agent.example.invalid\n172.28.30.21 netbird.example.invalid\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := rewriteLabHost(path, "agent.example.invalid", "172.29.30.30"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "172.29.30.30 agent.example.invalid\n172.28.30.21 netbird.example.invalid\n"
	if string(got) != want {
		t.Fatalf("lab-hosts:\n%s", got)
	}
}

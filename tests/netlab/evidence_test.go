package netlab

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Committed evidence is public. Do not inspect ignored daemon logs or scratch.
func TestCommittedEvidenceFiles_ValidAndHygieneClean(t *testing.T) {
	root := filepath.Join("..", "..")
	cmd := exec.Command("git", "ls-files", "-z", "--", "docs/poc/G2-*")
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range strings.Split(string(raw), "\x00") {
		if path == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		// This original G2 artifact is a Go struct dump despite its suffix.
		// Preserve that historical evidence; still scan its complete text.
		legacyText := path == "docs/poc/G2-evidence/TestAT12_ISO01_ISO04_AllowDeny/counters.json" && strings.HasPrefix(string(b), "{Counters:map[")
		if filepath.Ext(path) == ".json" && !legacyText && !json.Valid(b) {
			t.Errorf("invalid JSON: %s", path)
		}
		for _, bad := range []string{"/ho" + "me/", "BEGIN " + "EC PRIVATE KEY", "BEGIN " + "RSA PRIVATE KEY", "BEGIN " + "OPENSSH PRIVATE KEY", "sk_" + "live_", "gh" + "p_", "gh" + "o_"} {
			if strings.Contains(string(b), bad) {
				t.Errorf("public evidence contains forbidden text: %s", path)
			}
		}
	}
}

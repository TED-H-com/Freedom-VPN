package netlab

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestD098_Followup18ArchivePins(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "poc", "G2-followup18-evidence")
	raw, err := os.ReadFile(filepath.Join(root, "archive.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Source string `json:"source_commit"`
		Files  []struct {
			Path   string `json:"path"`
			Bytes  int    `json:"bytes"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Source != "dd5c8897e65c1b1dd4399bd8ee7d64f841fa20ec" || len(manifest.Files) == 0 {
		t.Fatal("missing archive provenance")
	}
	seen := map[string]bool{}
	for _, f := range manifest.Files {
		if !filepath.IsLocal(f.Path) || seen[f.Path] {
			t.Fatalf("invalid archive path %q", f.Path)
		}
		seen[f.Path] = true
		b, err := os.ReadFile(filepath.Join(root, f.Path))
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(b)
		if len(b) != f.Bytes || fmt.Sprintf("%x", hash) != f.SHA256 {
			t.Errorf("archive changed: %s", f.Path)
		}
	}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !d.IsDir() && rel != "archive.json" && !seen[rel] {
			t.Errorf("unpinned archive file: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("D-098: %d byte/SHA-256 pins match", len(manifest.Files))
}

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

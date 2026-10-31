package netlab

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
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
		if strings.HasSuffix(path, "-pins.json") || filepath.Base(path) == "archive.json" {
			if err := verifyEvidencePins(os.DirFS(root), path, b); err != nil {
				t.Errorf("%s: %v", path, err)
			}
		}
		for _, bad := range []string{"/ho" + "me/", "BEGIN " + "EC PRIVATE KEY", "BEGIN " + "RSA PRIVATE KEY", "BEGIN " + "OPENSSH PRIVATE KEY", "sk_" + "live_", "gh" + "p_", "gh" + "o_"} {
			if strings.Contains(string(b), bad) {
				t.Errorf("public evidence contains forbidden text: %s", path)
			}
		}
	}
}

// Source pins are repository-relative; retention/archive pins are relative to
// their manifest. raw_sha256 records private pre-normalization provenance, not
// the retained tree bytes: only sha256 (and bytes, when present) names those.
func verifyEvidencePins(tree fs.FS, manifestPath string, raw []byte) error {
	var manifest struct {
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Bytes  *int   `json:"bytes"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if len(manifest.Files) == 0 {
		return errors.New("empty pin manifest")
	}
	var failures []error
	seen := map[string]bool{}
	for _, pin := range manifest.Files {
		if !fs.ValidPath(pin.Path) || pin.Path == "." || strings.Contains(pin.Path, "\\") {
			failures = append(failures, fmt.Errorf("invalid pin path %q", pin.Path))
			continue
		}
		target := pin.Path
		if path.Base(manifestPath) != "source-pins.json" {
			target = path.Join(path.Dir(manifestPath), target)
		}
		if seen[target] {
			failures = append(failures, fmt.Errorf("duplicate pin: %s", target))
		}
		seen[target] = true
		if !immutableEvidencePath(target) {
			failures = append(failures, fmt.Errorf("D-098: pin outside docs/poc/ must name an immutable archive: %s", target))
		}
		b, err := fs.ReadFile(tree, target)
		if err != nil {
			failures = append(failures, fmt.Errorf("read pin %s: %w", target, err))
			continue
		}
		if actual := fmt.Sprintf("%x", sha256.Sum256(b)); actual != pin.SHA256 {
			failures = append(failures, fmt.Errorf("SHA-256 mismatch: %s: got %s, want %s", target, actual, pin.SHA256))
		}
		if pin.Bytes != nil && len(b) != *pin.Bytes {
			failures = append(failures, fmt.Errorf("byte length mismatch: %s: got %d, want %d", target, len(b), *pin.Bytes))
		}
	}
	return errors.Join(failures...)
}

func immutableEvidencePath(target string) bool {
	if strings.HasPrefix(target, "docs/poc/") {
		return true
	}
	// D-098 accepts named history snapshots and the devstack's dated run-ID
	// directories. A date or a follow-up directory alone is not a run archive.
	return regexp.MustCompile(`(?:^|/)history/[^/]+/.+`).MatchString(target) ||
		regexp.MustCompile(`^docs/e2e/runs/[0-9]{4}-[0-9]{2}-[0-9]{2}/(?:[^/]+/)*[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}/.+`).MatchString(target)
}

func TestD098_EvidencePinValidation(t *testing.T) {
	const archived = "docs/e2e/at/members/history/follow-up-8/AT09.json"
	const mutable = "docs/e2e/at/members/AT09.json"
	const run = "docs/e2e/runs/2026-10-31/followup18/2454c833-ae93-48e0-96b0-00dee7cc3117/row.json"
	const local = "docs/poc/G2-example-evidence/row.json"
	const source = "docs/poc/G2-example-evidence/source-pins.json"
	const retention = "docs/poc/G2-example-evidence/retention-pins.json"
	content := []byte("TEST ONLY evidence\n")
	digest := fmt.Sprintf("%x", sha256.Sum256(content))
	pin := func(target, hash string) string {
		return fmt.Sprintf(`{"files":[{"path":%q,"sha256":%q}]}`, target, hash)
	}
	tree := fstest.MapFS{}
	for _, target := range []string{archived, mutable, run, local, "docs/e2e/runs/2026-10-31/followup18/row.json", "docs/e2e/at/members/not-history/follow-up-8/AT09.json", "docs/e2e/at/members/history/AT09.json"} {
		tree[target] = &fstest.MapFile{Data: content}
	}
	for _, tc := range []struct {
		name, manifest, raw, wantError string
	}{
		{"history snapshot", source, pin(archived, digest), ""},
		{"run-ID snapshot", source, pin(run, digest), ""},
		{"owned source", source, pin(local, digest), ""},
		{"relative retention", retention, pin("row.json", digest), ""},
		{"relative archive", "docs/poc/G2-example-evidence/archive.json", pin("row.json", digest), ""},
		{"mutable matching bytes", source, pin(mutable, digest), "immutable archive"},
		{"dated directory alone", source, pin("docs/e2e/runs/2026-10-31/followup18/row.json", digest), "immutable archive"},
		{"history lookalike", source, pin("docs/e2e/at/members/not-history/follow-up-8/AT09.json", digest), "immutable archive"},
		{"history lacks snapshot", source, pin("docs/e2e/at/members/history/AT09.json", digest), "immutable archive"},
		{"changed source", source, pin(archived, strings.Repeat("0", 64)), "SHA-256 mismatch"},
		{"changed retained bytes", retention, pin("row.json", strings.Repeat("0", 64)), "SHA-256 mismatch"},
		{"missing target", source, pin("docs/e2e/at/members/history/follow-up-8/missing.json", digest), "read pin"},
		{"traversal", source, pin("docs/poc/../../members/AT09.json", digest), "invalid pin path"},
		{"relative traversal", retention, pin("../row.json", digest), "invalid pin path"},
		{"absolute path", source, pin("/docs/poc/row.json", digest), "invalid pin path"},
		{"empty manifest", source, `{"files":[]}`, "empty pin manifest"},
		{"malformed JSON", source, `{`, "unexpected end"},
		{"wrong byte length", retention, fmt.Sprintf(`{"files":[{"path":"row.json","sha256":%q,"bytes":0}]}`, digest), "byte length mismatch"},
		{"duplicate target", source, fmt.Sprintf(`{"files":[{"path":%q,"sha256":%q},{"path":%q,"sha256":%q}]}`, archived, digest, archived, digest), "duplicate pin"},
		{"checks later entries", source, fmt.Sprintf(`{"files":[{"path":%q,"sha256":%q},{"path":%q,"sha256":%q}]}`, archived, digest, run, strings.Repeat("0", 64)), "SHA-256 mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyEvidencePins(tree, tc.manifest, []byte(tc.raw))
			if tc.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

package members

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommittedEvidenceHygiene(t *testing.T) {
	root := filepath.Join("../../../..", "docs/e2e/at/members")
	records := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, bad := range []string{"/ho" + "me/", "BEGIN " + "EC PRIVATE KEY", "BEGIN " + "RSA PRIVATE KEY", "BEGIN " + "OPENSSH PRIVATE KEY", "sk_" + "live_", "gh" + "p_", "gh" + "o_", "user_code=", "\"password\":", "\"setup_key\":"} {
			if strings.Contains(string(raw), bad) {
				t.Errorf("unsafe evidence %s", path)
			}
		}
		if filepath.Ext(path) == ".json" && len(filepath.Base(path)) == len("AT01.json") && strings.HasPrefix(filepath.Base(path), "AT") {
			var rec struct {
				RunID  string `json:"run_id"`
				Source string `json:"source_commit"`
				Status string `json:"status"`
			}
			strict := filepath.Dir(path) == root || filepath.Dir(path) == filepath.Join(root, "history/follow-up-7") || filepath.Dir(path) == filepath.Join(root, "history/follow-up-8") || filepath.Dir(path) == filepath.Join(root, "history/follow-up-9") || filepath.Dir(path) == filepath.Join(root, "history/follow-up-10") || filepath.Dir(path) == filepath.Join(root, "history/follow-up-11")
			if json.Unmarshal(raw, &rec) != nil || (strict && (rec.RunID == "" || len(rec.Source) != 40)) {
				t.Errorf("missing provenance %s", path)
			}
			if rec.Status != "pass" && rec.Status != "blocked" && rec.Status != "fail" {
				t.Errorf("invalid status %s", path)
			}
			records++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if records < 10 {
		t.Fatal("missing case records")
	}
	t.Logf("scanned %d acceptance JSON records plus documentation", records)
}
func TestD098_Followup7Archive(t *testing.T) {
	verifyArchive(t, "follow-up-7", "297faa31-2f90-4aab-8cf0-6818629f46c8", 31)
}

func TestD098_Followup8Archive(t *testing.T) {
	verifyArchive(t, "follow-up-8", "69f92f49-73dc-4779-8101-dfb6a802a96b", 32)
}

func TestD098_Followup9Archive(t *testing.T) {
	verifyArchive(t, "follow-up-9", "11e305fe-6360-4dbd-95b1-67274aa4988c", 33)
}

func TestD098_Followup10Archive(t *testing.T) {
	verifyArchive(t, "follow-up-10", "2be2d32f-bfd9-46f5-bf29-afaa6617ae94", 35)
}

func TestD098_Followup11Archive(t *testing.T) {
	verifyArchive(t, "follow-up-11", "7edb263f-1f6e-4f94-9e8e-26099f96a699", 43)
}

func TestD098_Followup12Archive(t *testing.T) {
	verifyArchive(t, "follow-up-12", "b98b78ee-9162-419e-ae2f-d1d9a618a3e8", 52)
}

func verifyArchive(t *testing.T, followup, run string, count int) {
	t.Helper()
	root := filepath.Join("../../../..", "docs/e2e/at/members/history", followup)
	sums, err := os.ReadFile(filepath.Join(root, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(sums)), "\n")
	if len(lines) != count {
		t.Fatalf("archive inventory: got %d, want %d", len(lines), count)
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatal("invalid checksum manifest")
		}
		raw, err := os.ReadFile(filepath.Join(root, fields[1]))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != fields[0] {
			t.Errorf("archive changed: %s", fields[1])
		}
		if strings.HasPrefix(fields[1], "AT") && filepath.Ext(fields[1]) == ".json" {
			var rec struct {
				RunID string `json:"run_id"`
			}
			wantRun := run
			// Follow-up 12 deliberately retained AT01/AT09 from follow-up 11.
			if followup == "follow-up-12" && (fields[1] == "AT01.json" || strings.HasPrefix(fields[1], "AT09")) {
				wantRun = "7edb263f-1f6e-4f94-9e8e-26099f96a699"
			}
			if json.Unmarshal(raw, &rec) != nil || rec.RunID != wantRun {
				t.Errorf("wrong archive run: %s", fields[1])
			}
		}
	}
}

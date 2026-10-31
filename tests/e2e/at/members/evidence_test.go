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
			strict := filepath.Dir(path) == root || filepath.Dir(path) == filepath.Join(root, "history/follow-up-7")
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
	root := filepath.Join("../../../..", "docs/e2e/at/members/history/follow-up-7")
	sums, err := os.ReadFile(filepath.Join(root, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(sums)), "\n") {
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
			if json.Unmarshal(raw, &rec) != nil || rec.RunID != "297faa31-2f90-4aab-8cf0-6818629f46c8" {
				t.Errorf("wrong archive run: %s", fields[1])
			}
		}
	}
}

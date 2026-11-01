package isolation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestISO_F6_03_ProducerTimingCorrelation(t *testing.T) {
	out, err := exec.Command("python3", "-B", "-m", "unittest", "-v", "producer_timing_test.py").CombinedOutput()
	if err != nil {
		t.Fatalf("producer correlation: %v\n%s", err, out)
	}
	t.Log(string(out))
}

func TestD098_Followup8ArchivePins(t *testing.T) {
	root := "../../../../docs/e2e/at/isolation/history/follow-up-8"
	raw, err := os.ReadFile(filepath.Join(root, "archive.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		RunID string `json:"run_id"`
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.RunID != "7779d83c-2f78-4699-ac88-8fcc59e8cf87" || len(m.Files) != 80 {
		t.Fatal("wrong archive snapshot")
	}
	pins := map[string]string{
		"deadline-verdict.json":    "94d777e21b93689e899442f5e3b7d62463527099c8ce511de56d87b6b7253ae1",
		"recovery-verdict.json":    "b1a40773309c350664e502d87e8699fe9388c1211b94c2c241f5968bfaa452ce",
		"continuity-analysis.json": "9e5d77f88c69c8b30fe3d78abe185c37bb9ccb610ea464f532b4a75d7ada5b5c",
	}
	for _, f := range m.Files {
		body, e := os.ReadFile(filepath.Join(root, f.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(body)
		got := hex.EncodeToString(sum[:])
		if got != f.SHA256 {
			t.Errorf("archive hash mismatch: %s", f.Path)
		}
		if want, ok := pins[f.Path]; ok {
			if got != want {
				t.Errorf("netlab pin mismatch: %s", f.Path)
			}
			delete(pins, f.Path)
		}
	}
	if len(pins) != 0 {
		t.Fatal("netlab pins missing from manifest")
	}
}

func TestD098_Followup9ArchivePins(t *testing.T) {
	root := "../../../../docs/e2e/at/isolation/history/follow-up-9"
	raw, err := os.ReadFile(filepath.Join(root, "archive.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RunID string `json:"run_id"`
		Files []struct{ Path, SHA256 string }
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RunID != "2be2797b-7a28-4e9f-a1fe-5fffa4c1b047" || len(manifest.Files) != 86 {
		t.Fatal("wrong archived run")
	}
	for _, f := range manifest.Files {
		body, e := os.ReadFile(filepath.Join(root, f.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			t.Errorf("archive changed: %s", f.Path)
		}
	}
}

func TestD098_Followup10ArchivePins(t *testing.T) {
	root := "../../../../docs/e2e/at/isolation/history/follow-up-10"
	raw, err := os.ReadFile(filepath.Join(root, "archive.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RunID string `json:"run_id"`
		Files []struct{ Path, SHA256 string }
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RunID != "bfcc3880-dc02-485d-990c-13811d6e8835" || len(manifest.Files) != 100 {
		t.Fatal("wrong archived run")
	}
	for _, f := range manifest.Files {
		body, e := os.ReadFile(filepath.Join(root, f.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			t.Errorf("archive changed: %s", f.Path)
		}
	}
}

func TestD098_Followup11ArchivePins(t *testing.T) {
	root := "../../../../docs/e2e/at/isolation/history/follow-up-11"
	raw, err := os.ReadFile(filepath.Join(root, "archive.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RunID string `json:"run_id"`
		Files []struct{ Path, SHA256 string }
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RunID != "edd986c4-16d2-4aab-b92f-baa19f8bd829" || len(manifest.Files) != 103 {
		t.Fatal("wrong archived run")
	}
	for _, f := range manifest.Files {
		body, e := os.ReadFile(filepath.Join(root, f.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			t.Errorf("archive changed: %s", f.Path)
		}
	}
}

func TestD087_OriginalWireAndAcceptedTransition(t *testing.T) {
	out, err := exec.Command("python3", "-B", "-m", "unittest", "-v", "grant_evidence_test.py").CombinedOutput()
	if err != nil {
		t.Fatalf("grant transition: %v\n%s", err, out)
	}
	t.Log(string(out))
}

func TestD098_Followup12ArchivePins(t *testing.T) {
	root := "../../../../docs/e2e/at/isolation/history/follow-up-12"
	raw, err := os.ReadFile(filepath.Join(root, "archive.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RunID string `json:"run_id"`
		Files []struct{ Path, SHA256 string }
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RunID != "1c25ee44-1872-4cce-a9f5-d52491d87095" || len(manifest.Files) != 108 {
		t.Fatal("wrong archived run")
	}
	for _, f := range manifest.Files {
		body, e := os.ReadFile(filepath.Join(root, f.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			t.Errorf("archive changed: %s", f.Path)
		}
	}
}

func TestAT29_M3CentralHoldEvidence(t *testing.T) {
	out, err := exec.Command("python3", "-B", "-m", "unittest", "-v", "hold_evidence_test.py").CombinedOutput()
	if err != nil {
		t.Fatalf("central hold joins: %v\n%s", err, out)
	}
	t.Log(string(out))
}

func TestD098_Followup13ArchivePins(t *testing.T) {
	root := "../../../../docs/e2e/at/isolation/history/follow-up-13"
	raw, err := os.ReadFile(filepath.Join(root, "archive.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RunID string `json:"run_id"`
		Files []struct{ Path, SHA256 string }
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RunID != "38074149-1137-4782-80fd-8669c36c6fcb" || len(manifest.Files) != 112 {
		t.Fatal("wrong archived run")
	}
	for _, f := range manifest.Files {
		body, e := os.ReadFile(filepath.Join(root, f.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			t.Errorf("archive changed: %s", f.Path)
		}
	}
}

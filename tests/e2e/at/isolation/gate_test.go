//go:build e2e

package isolation

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Exercise the real test exit status, including the early blocked path. A
// failure must not suppress later AT records or leave an earlier pass in place.
func TestAT10_AT13_GateWritesBeforeFailure(t *testing.T) {
	if status := os.Getenv("FVPN_ISOLATION_GATE_CHILD"); status != "" {
		dir := os.Getenv("FVPN_AT_EVIDENCE_DIR")
		if status == StatusBlocked {
			blockProduct(t, dir, Record{}, "serving baseline unavailable")
		} else {
			writeProduct(t, dir, Record{}, productObs{}, exitProbe{fail: "protected destination delivered"})
		}
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{StatusBlocked, StatusFail} {
		t.Run(status, func(t *testing.T) {
			dir := t.TempDir()
			for _, id := range []string{"AT10", "AT11", "AT12", "AT13"} {
				if err := Write(dir, Record{ID: id, Status: StatusPass}); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(exe, "-test.run=^TestAT10_AT13_GateWritesBeforeFailure$")
			cmd.Env = append(os.Environ(), "FVPN_ISOLATION_GATE_CHILD="+status, "FVPN_AT_EVIDENCE_DIR="+dir)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("non-pass gate exited successfully: %s", out)
			}
			for _, id := range []string{"AT10", "AT11", "AT12", "AT13"} {
				raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
				if err != nil {
					t.Fatal(err)
				}
				var rec Record
				if err := json.Unmarshal(raw, &rec); err != nil {
					t.Fatal(err)
				}
				want := StatusBlocked
				if id == "AT10" {
					want = status
				}
				if rec.Status != want {
					t.Fatalf("%s status %q, want %q; child output: %s", id, rec.Status, want, out)
				}
				if _, err := os.Stat(filepath.Join(dir, id+".md")); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

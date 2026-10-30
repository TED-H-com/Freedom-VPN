package updates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactStripsSecrets(t *testing.T) {
	// TEST ONLY - NOT A SECRET: synthetic redaction inputs.
	in := `"subject": "private-subject" path /home/someone/key postgres://fvpn:pw@127.0.0.1/fvpn bearer abc "token": "one-time" -----BEGIN EC PRIVATE KEY-----
secret
-----END EC PRIVATE KEY-----`
	out := Redact(in)
	for _, part := range []string{"private-subject", "/home/someone", "postgres://", "bearer abc", "one-time", "BEGIN EC"} {
		if strings.Contains(out, part) {
			t.Fatalf("redaction left %q in %s", part, out)
		}
	}
}

func TestClassifyFirstInstallRollback(t *testing.T) {
	health := `{"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"}`
	lab := ClassifyFirstInstallRollback(RollbackFacts{
		HealthJSON: health, ErrorCode: "ROLLBACK_FAILED", ErrorDetail: "ROLLBACK_FAILED",
		Events: "event stage=checking detail=\nevent stage=rolling_back detail=\nevent stage=recovery_required detail=",
	})
	if lab != FindingNoLastGood {
		t.Fatalf("lab: %s", lab)
	}
	if got := ClassifyFirstInstallRollback(RollbackFacts{ErrorCode: "HASH_MISMATCH"}); got != FindingNotReproduced {
		t.Fatalf("hash: %s", got)
	}
	defect := ClassifyFirstInstallRollback(RollbackFacts{
		HealthJSON: health, ErrorCode: "ROLLBACK_FAILED", LastGood: "rel",
		Events: "stage=rollback detail=symlink",
	})
	if defect != FindingInconclusive {
		t.Fatalf("defect: %s", defect)
	}
}

// TestAT23_AT27_RecordsBeforeGate verifies acceptance cannot report green
// with incomplete AT evidence, including an early return with missing records.
func TestAT23_AT27_RecordsBeforeGate(t *testing.T) {
	for _, status := range []Status{StatusPass, StatusBlocked, StatusFail, "failed", StatusNeedsOperatorRun, ""} {
		for _, assigned := range []string{"AT23", "AT24", "AT25", "AT26", "AT27"} {
			t.Run(assigned+"/"+string(status), func(t *testing.T) {
				dir := t.TempDir()
				records := map[string]*Record{}
				for _, id := range []string{"AT23", "AT24", "AT25", "AT26", "AT27"} {
					records[id] = &Record{ID: id, Status: StatusPass}
				}
				records[assigned].Status = status
				err := WriteAcceptance(dir, records)
				if (err != nil) != (status != StatusPass) {
					t.Fatalf("status %q: gate error %v", status, err)
				}
				for id, want := range records {
					b, err := os.ReadFile(filepath.Join(dir, id+".json"))
					if err != nil {
						t.Fatal(err)
					}
					var got Record
					if err := json.Unmarshal(b, &got); err != nil || got.Status != want.Status {
						t.Fatalf("record %s: %s (%v)", id, b, err)
					}
					if _, err := os.Stat(filepath.Join(dir, id+".md")); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
	if err := WriteAcceptance(t.TempDir(), map[string]*Record{}); err == nil {
		t.Fatal("missing assigned AT records passed")
	}
}

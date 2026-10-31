package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedact_StripsSecrets(t *testing.T) {
	home := "/" + "home/someone"
	live := "sk_" + "live_abc"
	gh := "gh" + "p_abc"
	key := "-----BEGIN " + "OPENSSH PRIVATE KEY-----\nsecret\n-----END " + "OPENSSH PRIVATE KEY-----"
	code := "user_" + "code=AAAA-BBBB"
	in := "path=" + home + "/key postgres://fvpn:s3cret@db/fvpn Bearer tok \"setup_key\":\"abc\" " + live + " " + gh + " " + code + "\n" + key
	out := Redact(in)
	for _, banned := range []string{home, "s3cret", "Bearer tok", "abc", "sk_" + "live_", "gh" + "p_", "BEGIN " + "OPENSSH", "AAAA-BBBB"} {
		if strings.Contains(out, banned) {
			t.Fatalf("redaction left %q in %s", banned, out)
		}
	}
}

func TestWrite_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	rec := Record{
		ID: "at-sample", Title: "sample", Spec: []string{"§19 AT06"}, Status: StatusPass,
		Observations: []Observation{{Name: "token", Detail: "length 43 path=" + "/" + "home/someone/token"}},
	}
	if err := Write(dir, rec); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "at-sample.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "/"+"home/") || !strings.Contains(string(b), "Status: pass") {
		t.Fatalf("summary: %s", b)
	}
}

func TestSEC02_WriteRedactsIdentitySubjects(t *testing.T) {
	dir := t.TempDir()
	rec := Record{ID: "identity", Status: StatusBlocked, RunID: "test-run", SourceCommit: "test-commit"}
	for _, name := range []string{"portal_sub", "netbird_user_id", "peer_user_id", "external_user_id"} {
		rec.Observations = append(rec.Observations, Observation{Name: name, Detail: "synthetic-private-subject"})
	}
	rec.Observations = append(rec.Observations, Observation{Name: "same_subject", Detail: "true"})
	if err := Write(dir, rec); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".json", ".md"} {
		raw, err := os.ReadFile(filepath.Join(dir, rec.ID+ext))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "synthetic-private-subject") || !strings.Contains(string(raw), "same_subject") || !strings.Contains(string(raw), "test-run") {
			t.Fatalf("unsafe or incomplete evidence: %s", raw)
		}
	}
}

// The follow-up campaign's archived failures are part of the public artifact.
func TestCommittedFixtureEvidence_ValidAndHygieneClean(t *testing.T) {
	root := "../../docs/e2e/runs/2026-10-31"
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("no committed follow-up fixture records")
	}
	records := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && filepath.Dir(path) == root && d.Name() != "followup16" && d.Name() != "followup17" && d.Name() != "followup18" {
				return filepath.SkipDir
			}
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".json" {
			var rec Record
			if json.Unmarshal(raw, &rec) != nil || rec.ID == "" || rec.RunID == "" || len(rec.SourceCommit) != 40 {
				t.Errorf("invalid fixture record: %s", path)
			}
			switch rec.Status {
			case StatusPass, StatusBlocked, StatusFail:
			default:
				t.Errorf("invalid claim: %s", path)
			}
			records++
		}
		for _, bad := range []string{"/ho" + "me/", "BEGIN " + "EC PRIVATE KEY", "BEGIN " + "RSA PRIVATE KEY", "BEGIN " + "OPENSSH PRIVATE KEY", "sk_" + "live_", "gh" + "p_", "gh" + "o_", "user_code=", "\"password\":", "\"setup_key\":"} {
			if strings.Contains(string(raw), bad) {
				t.Errorf("unsafe committed evidence: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if records == 0 {
		t.Fatal("no fixture records scanned")
	}
}

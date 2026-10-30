package usage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedact_RemovesSecretsAndHomePaths(t *testing.T) {
	homePrefix := "/ho" + "me/"
	pemHeader := "BEGIN " + "EC " + "PRIVATE " + "KEY"
	pemFooter := "END " + "EC " + "PRIVATE " + "KEY"
	bearerTok := "jwt_" + strings.Repeat("a", 24)
	setupTok := "setup_" + strings.Repeat("b", 24)
	enrollKey := "enrollment_" + "token"
	enrollTok := "tok_" + strings.Repeat("c", 24)
	skPrefix := "sk_" + "live_"
	skTok := skPrefix + strings.Repeat("x", 24)
	ghpPrefix := "gh" + "p_"
	ghpTok := ghpPrefix + strings.Repeat("y", 24)
	customTok := "custom_" + strings.Repeat("z", 24)

	input := strings.Join([]string{
		"Authorization: Bearer " + bearerTok,
		`{"setup_key":"` + setupTok + `","` + enrollKey + `":"` + enrollTok + `","upload_bytes":"9007199254740993"}`,
		"-----" + pemHeader + "-----\n" + strings.Repeat("M", 32) + "\n-----" + pemFooter + "-----",
		"path=" + homePrefix + "alice/projects/freedom-vpn/wt/w4-at-usage/docs",
		"stripe=" + skTok + " pat=" + ghpTok + " custom=" + customTok,
	}, "\n")

	got := Redact(input, customTok)
	for _, forbidden := range []string{
		bearerTok,
		setupTok,
		enrollTok,
		pemHeader,
		homePrefix + "alice",
		skPrefix,
		ghpPrefix,
		customTok,
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Redact leaked %q in output:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, `"upload_bytes":"9007199254740993"`) {
		t.Fatalf("Redact altered non-secret decimal string: %s", got)
	}
}

func TestWriteEvidence_ValidatesAndWritesJSONAndMarkdown(t *testing.T) {
	tmp := t.TempDir()
	homePrefix := "/ho" + "me/"
	setupTok := "setup_" + strings.Repeat("s", 24)
	rec := Record{
		ID:              "AT15",
		Title:           "Usage Event Idempotency, Contiguous ACK and Ledger Recovery",
		Spec:            "§10.4, §10.5, §10.6, §10.8, §17.3, §19 AT15",
		Requirements:    []string{"MTR03", "MTR04", "API01", "DATA01"},
		Status:          StatusBlocked,
		At:              "2026-10-29T20:00:00Z",
		Build:           BuildMetadata{GitCommit: "350e57c"},
		ImageDigests:    map[string]string{"control": "sha256:1111"},
		PolicyRevision:  "plan:rev1,fair_use:rev1",
		NodeIncarnation: "00000000-0000-7000-8000-000000000001/00000000-0000-7000-8000-000000000002",
		Expected:        "highest_contiguous_seq advances only when missing sequences are filled.",
		Actual:          "Live mTLS usage-batches and usage-checkpoint verified; member traffic precondition blocked by 422 NODE_NOT_SERVICEABLE.",
		Blockers:        []string{"PUT /api/v1/admin/nodes/{id}/publication returned 422 NODE_NOT_SERVICEABLE"},
		Commands: []CommandRecord{
			{Command: "POST /api/v1/agent/usage-batches", ExitCode: 0, Output: `{"highest_contiguous_seq":"1"}`},
		},
		Observations: map[string]any{
			"secret_test": `{"setup_key":"` + setupTok + `"}`,
			"path_test":   homePrefix + "tester/worktree",
		},
		MechanismChecks: []MechanismCheck{
			{Name: "OutOfOrderSeq1Seq3ThenSeq2", SpecRef: "§10.6", Status: StatusPass, Summary: "Contiguous ACK stayed at 1 until seq 2 arrived, then advanced to 3."},
		},
	}

	if err := WriteEvidence(tmp, rec, setupTok); err != nil {
		t.Fatalf("WriteEvidence failed: %v", err)
	}

	for _, ext := range []string{".json", ".md"} {
		raw, err := os.ReadFile(filepath.Join(tmp, "AT15"+ext))
		if err != nil {
			t.Fatalf("read AT15%s: %v", ext, err)
		}
		text := string(raw)
		if strings.Contains(text, setupTok) || strings.Contains(text, homePrefix+"tester") {
			t.Fatalf("AT15%s leaked secret or home path:\n%s", ext, text)
		}
	}
}

func TestCommittedEvidenceFiles_ValidAndHygieneClean(t *testing.T) {
	evidenceDir := filepath.Clean("../../../../docs/e2e/at/usage")
	if _, err := os.Stat(evidenceDir); os.IsNotExist(err) {
		t.Skip("docs/e2e/at/usage not found")
	}

	cases := []string{"AT14", "AT15", "AT16", "AT17", "AT18", "AT19"}
	for _, id := range cases {
		jsonPath := filepath.Join(evidenceDir, id+".json")
		mdPath := filepath.Join(evidenceDir, id+".md")
		jsonBytes, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("read %s: %v", jsonPath, err)
		}
		mdBytes, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("read %s: %v", mdPath, err)
		}
		var rec Record
		if err := json.Unmarshal(jsonBytes, &rec); err != nil {
			t.Fatalf("unmarshal %s: %v", jsonPath, err)
		}
		if rec.ID != id {
			t.Fatalf("%s ID mismatch: got %q", jsonPath, rec.ID)
		}
		if err := ValidateRecord(rec); err != nil {
			t.Fatalf("ValidateRecord(%s): %v", jsonPath, err)
		}
		for _, text := range []string{string(jsonBytes), string(mdBytes)} {
			for _, bad := range []string{
				"/ho" + "me/",
				"BEGIN " + "EC " + "PRIVATE " + "KEY",
				"BEGIN " + "RSA " + "PRIVATE " + "KEY",
				"BEGIN " + "OPENSSH " + "PRIVATE " + "KEY",
				"sk_" + "live_",
				"gh" + "p_",
				"gh" + "o_",
			} {
				if strings.Contains(text, bad) {
					t.Fatalf("%s leaked %q", id, bad)
				}
			}
		}
	}
	for _, doc := range []string{"README.md", "operator-run.md", "requests.md"} {
		p := filepath.Join(evidenceDir, doc)
		raw, err := os.ReadFile(p)
		if err != nil || len(raw) == 0 {
			t.Fatalf("missing or empty %s: %v", p, err)
		}
	}
}

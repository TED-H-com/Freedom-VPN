package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSummaryFixture(t *testing.T, dir string, rows []SummaryRow) string {
	t.Helper()
	if err := WriteSummary(
		dir,
		rows,
		"019a0000-0000-7000-8000-000000000001",
		"0123456789abcdef0123456789abcdef01234567",
	); err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}
	return filepath.Join(dir, SummaryFile)
}

func TestD144_CommittedExpectedBlockedListAndFollowup29FullSummary(t *testing.T) {
	list, err := DefaultNightlyExpectedBlockedList()
	if err != nil {
		t.Fatalf("DefaultNightlyExpectedBlockedList: %v", err)
	}
	if len(list.Entries) != 2 {
		t.Fatalf("expected 2 entries in committed list, got %d", len(list.Entries))
	}
	wantIDs := map[string]string{
		"member-ipv6":  "w3-devstack-e2e / platform operators",
		"lease-expiry": "w2-agent-core",
	}
	for _, entry := range list.Entries {
		wantOwner, ok := wantIDs[entry.ID]
		if !ok {
			t.Fatalf("unexpected entry id %q in committed list", entry.ID)
		}
		if entry.Owner != wantOwner {
			t.Fatalf("entry %s owner = %q, want %q", entry.ID, entry.Owner, wantOwner)
		}
		if entry.Reason == "" || entry.EstablishedFrom == "" {
			t.Fatalf("entry %s missing reason or established_from", entry.ID)
		}
	}

	summaryPath := filepath.Join("..", "..", "docs", "e2e", "runs", "2026-11-03", "followup29", "after-full", "acceptance-summary.md")
	rep, err := EvaluateNightlyGate(summaryPath, list, NightlyGateOptions{
		TestExitCode: 1,
		TestLog:      "    scenarios_test.go:164: acceptance summary: pass=31 blocked=2 fail=0 run=91eb0372-e42b-4698-a137-7cce53b263fa source=f52fc9381405d4aafd064dc0ef055977de97f418\n    scenarios_test.go:171: one or more scenarios are blocked or failed; see docs/e2e/runs\n--- FAIL: TestE2E_Acceptance (480.00s)\nFAIL\n",
	})
	if err != nil || !rep.OK {
		t.Fatalf("expected followup29/after-full to pass D-144 gate, got ok=%v err=%v", rep.OK, err)
	}
	if rep.PassCount != 31 || rep.BlockedCount != 2 || rep.FailCount != 0 || rep.ErrorCount != 0 {
		t.Fatalf("unexpected counts: %+v", rep)
	}
	if len(rep.ExpectedBlocked) != 2 || len(rep.UnexpectedBlocked) != 0 || len(rep.Unblocked) != 0 {
		t.Fatalf("unexpected dispositions: %+v", rep)
	}
}

func TestD144_ListedBlockedRowKeepsJobGreenAndPreservesBlockedStatus(t *testing.T) {
	list, err := DefaultNightlyExpectedBlockedList()
	if err != nil {
		t.Fatalf("DefaultNightlyExpectedBlockedList: %v", err)
	}
	runDir := t.TempDir()
	summaryPath := writeSummaryFixture(t, runDir, []SummaryRow{
		{ID: "admin-login", Status: StatusPass},
		{ID: "member-ipv6", Status: StatusBlocked},
		{ID: "lease-expiry", Status: StatusBlocked},
	})
	origSummary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile summary: %v", err)
	}

	rep, err := EvaluateNightlyGate(summaryPath, list, NightlyGateOptions{
		TestExitCode: 1,
		TestLog:      "    scenarios_test.go:164: acceptance summary: pass=1 blocked=2 fail=0 run=019a0000-0000-7000-8000-000000000001 source=0123456789abcdef0123456789abcdef01234567\n    scenarios_test.go:171: one or more scenarios are blocked or failed; see docs/e2e/runs\n--- FAIL: TestE2E_Acceptance (12.00s)\n",
	})
	if err != nil || !rep.OK || rep.Verdict != "pass" {
		t.Fatalf("expected listed blocked rows to pass gate, got ok=%v verdict=%q err=%v", rep.OK, rep.Verdict, err)
	}
	if len(rep.ExpectedBlocked) != 2 {
		t.Fatalf("expected 2 expected-blocked rows, got %d", len(rep.ExpectedBlocked))
	}

	stepSummaryPath := filepath.Join(t.TempDir(), "step-summary.md")
	if err := WriteNightlyGateArtifacts(runDir, rep, stepSummaryPath); err != nil {
		t.Fatalf("WriteNightlyGateArtifacts: %v", err)
	}

	// Invariant: acceptance-summary.md stays untouched with Status=blocked.
	afterSummary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile summary after: %v", err)
	}
	if string(afterSummary) != string(origSummary) {
		t.Fatalf("acceptance-summary.md was modified by nightly gate")
	}
	if !strings.Contains(string(afterSummary), "| member-ipv6 | blocked |") {
		t.Fatalf("acceptance-summary.md lost blocked status: %s", afterSummary)
	}

	// Uploaded records and step summary name the listed blocked rows with reason and owner.
	gateMD, err := os.ReadFile(filepath.Join(runDir, "nightly-gate.md"))
	if err != nil {
		t.Fatalf("ReadFile nightly-gate.md: %v", err)
	}
	stepMD, err := os.ReadFile(stepSummaryPath)
	if err != nil {
		t.Fatalf("ReadFile step-summary.md: %v", err)
	}
	for _, doc := range []string{string(gateMD), string(stepMD)} {
		for _, needle := range []string{
			"`member-ipv6`",
			"w3-devstack-e2e / platform operators",
			"DisableIPv6",
			"`lease-expiry`",
			"w2-agent-core",
			"UPDATE_BUSY",
		} {
			if !strings.Contains(doc, needle) {
				t.Fatalf("output missing %q:\n%s", needle, doc)
			}
		}
	}
}

func TestD144_ListedRowThatPassesIsReportedUnblockedAndSucceeds(t *testing.T) {
	list, err := DefaultNightlyExpectedBlockedList()
	if err != nil {
		t.Fatalf("DefaultNightlyExpectedBlockedList: %v", err)
	}
	runDir := t.TempDir()
	summaryPath := writeSummaryFixture(t, runDir, []SummaryRow{
		{ID: "admin-login", Status: StatusPass},
		{ID: "member-ipv6", Status: StatusBlocked},
		{ID: "lease-expiry", Status: StatusPass},
	})

	rep, err := EvaluateNightlyGate(summaryPath, list, NightlyGateOptions{
		TestExitCode: 1,
		TestLog:      "    scenarios_test.go:164: acceptance summary: pass=2 blocked=1 fail=0 run=019a0000-0000-7000-8000-000000000001 source=0123456789abcdef0123456789abcdef01234567\n    scenarios_test.go:171: one or more scenarios are blocked or failed; see docs/e2e/runs\n--- FAIL: TestE2E_Acceptance (12.00s)\n",
	})
	if err != nil || !rep.OK || rep.Verdict != "pass" {
		t.Fatalf("expected listed passing row to succeed, got ok=%v verdict=%q err=%v", rep.OK, rep.Verdict, err)
	}
	if len(rep.Unblocked) != 1 || rep.Unblocked[0].ID != "lease-expiry" {
		t.Fatalf("expected lease-expiry in Unblocked, got %+v", rep.Unblocked)
	}
	if len(rep.ExpectedBlocked) != 1 || rep.ExpectedBlocked[0].ID != "member-ipv6" {
		t.Fatalf("expected member-ipv6 in ExpectedBlocked, got %+v", rep.ExpectedBlocked)
	}
	md := rep.Markdown()
	if !strings.Contains(md, "### Unblocked Rows (passed; remove from expected-blocked list)") || !strings.Contains(md, "`lease-expiry`") {
		t.Fatalf("markdown did not report unblocked row:\n%s", md)
	}
}

func TestD144_UnlistedBlockedRowFails(t *testing.T) {
	list, err := DefaultNightlyExpectedBlockedList()
	if err != nil {
		t.Fatalf("DefaultNightlyExpectedBlockedList: %v", err)
	}
	runDir := t.TempDir()
	summaryPath := writeSummaryFixture(t, runDir, []SummaryRow{
		{ID: "admin-login", Status: StatusPass},
		{ID: "member-ipv6", Status: StatusBlocked},
		{ID: "portal-billing-return", Status: StatusBlocked},
	})

	rep, err := EvaluateNightlyGate(summaryPath, list, NightlyGateOptions{})
	if err == nil || rep.OK || rep.Verdict != "fail" {
		t.Fatalf("expected unlisted blocked row to fail gate, got ok=%v verdict=%q err=%v", rep.OK, rep.Verdict, err)
	}
	if len(rep.UnexpectedBlocked) != 1 || rep.UnexpectedBlocked[0].ID != "portal-billing-return" {
		t.Fatalf("expected portal-billing-return in UnexpectedBlocked, got %+v", rep.UnexpectedBlocked)
	}
}

func TestD144_FailRowFails(t *testing.T) {
	list, err := DefaultNightlyExpectedBlockedList()
	if err != nil {
		t.Fatalf("DefaultNightlyExpectedBlockedList: %v", err)
	}
	runDir := t.TempDir()
	summaryPath := writeSummaryFixture(t, runDir, []SummaryRow{
		{ID: "admin-login", Status: StatusFail},
		{ID: "member-ipv6", Status: StatusBlocked},
	})

	rep, err := EvaluateNightlyGate(summaryPath, list, NightlyGateOptions{})
	if err == nil || rep.OK || rep.Verdict != "fail" {
		t.Fatalf("expected fail row to fail gate, got ok=%v verdict=%q err=%v", rep.OK, rep.Verdict, err)
	}
	if rep.FailCount != 1 || len(rep.Failed) != 1 || rep.Failed[0].ID != "admin-login" {
		t.Fatalf("expected admin-login in Failed, got %+v", rep.Failed)
	}
}

func TestD144_ErrorRowAndRunnerErrorFail(t *testing.T) {
	list, err := DefaultNightlyExpectedBlockedList()
	if err != nil {
		t.Fatalf("DefaultNightlyExpectedBlockedList: %v", err)
	}

	// Subcase A: summary contains an explicit error status row.
	runDirA := t.TempDir()
	summaryPathA := filepath.Join(runDirA, SummaryFile)
	rawErrorSummary := strings.Join([]string{
		"# acceptance-summary",
		"",
		"acceptance summary: pass=1 blocked=0 fail=0 run=run-1 source=abc",
		"",
		"A blocked scenario is still non-green; see each record for its owner.",
		"",
		"| Scenario | Status |",
		"|---|---|",
		"| admin-login | pass |",
		"| node-restart | error |",
		"",
	}, "\n")
	if err := os.WriteFile(summaryPathA, []byte(rawErrorSummary), 0o644); err != nil {
		t.Fatalf("WriteFile summaryA: %v", err)
	}
	repA, errA := EvaluateNightlyGate(summaryPathA, list, NightlyGateOptions{})
	if errA == nil || repA.OK || repA.ErrorCount != 1 {
		t.Fatalf("expected error status row to fail gate, got ok=%v error_count=%d err=%v", repA.OK, repA.ErrorCount, errA)
	}

	// Subcase B: summary rows are all pass/expected-blocked, but another unit test failed in go test.
	runDirB := t.TempDir()
	summaryPathB := writeSummaryFixture(t, runDirB, []SummaryRow{
		{ID: "admin-login", Status: StatusPass},
		{ID: "member-ipv6", Status: StatusBlocked},
	})
	repB, errB := EvaluateNightlyGate(summaryPathB, list, NightlyGateOptions{
		TestExitCode: 1,
		TestLog:      "--- FAIL: TestPrefixE2EDoesNotMatchE2EExports (0.01s)\n    scenarios_test.go:171: one or more scenarios are blocked or failed; see docs/e2e/runs\n--- FAIL: TestE2E_Acceptance (10.00s)\n",
	})
	if errB == nil || repB.OK || repB.ErrorCount == 0 {
		t.Fatalf("expected non-acceptance test failure to fail gate as error, got ok=%v error_count=%d err=%v", repB.OK, repB.ErrorCount, errB)
	}
}

func TestD144_MissingOrUnreadableSummaryFails(t *testing.T) {
	list, err := DefaultNightlyExpectedBlockedList()
	if err != nil {
		t.Fatalf("DefaultNightlyExpectedBlockedList: %v", err)
	}

	// Missing summary file.
	missingPath := filepath.Join(t.TempDir(), "does-not-exist", "acceptance-summary.md")
	repMissing, errMissing := EvaluateNightlyGate(missingPath, list, NightlyGateOptions{})
	if errMissing == nil || repMissing.OK || repMissing.ErrorCount == 0 {
		t.Fatalf("expected missing summary to fail gate, got ok=%v err=%v", repMissing.OK, errMissing)
	}

	// Unreadable summary (path is a directory).
	dirAsFile := t.TempDir()
	repDir, errDir := EvaluateNightlyGate(dirAsFile, list, NightlyGateOptions{})
	if errDir == nil || repDir.OK || repDir.ErrorCount == 0 {
		t.Fatalf("expected directory summary path to fail gate, got ok=%v err=%v", repDir.OK, errDir)
	}

	// Malformed summary (no rows / totals mismatch).
	badSummary := filepath.Join(t.TempDir(), "acceptance-summary.md")
	if err := os.WriteFile(badSummary, []byte("# End-to-end Acceptance Summary\n- Totals: pass=2, blocked=0, fail=0\n| Scenario | Spec | Status | Detail |\n|---|---|---|---|\n| [admin-oidc](admin-oidc.md) | §5.2 | **pass** | ok |\n"), 0o644); err != nil {
		t.Fatalf("WriteFile badSummary: %v", err)
	}
	repBad, errBad := EvaluateNightlyGate(badSummary, list, NightlyGateOptions{})
	if errBad == nil || repBad.OK || repBad.ErrorCount == 0 {
		t.Fatalf("expected malformed summary with totals mismatch to fail gate, got ok=%v err=%v", repBad.OK, errBad)
	}
}

func TestD144_MissingUnreadableOrMalformedExpectedBlockedListFails(t *testing.T) {
	runDir := t.TempDir()
	summaryPath := writeSummaryFixture(t, runDir, []SummaryRow{
		{ID: "admin-login", Status: StatusPass},
	})

	// Missing expected-blocked list file.
	missingList := filepath.Join(t.TempDir(), "does-not-exist.json")
	repMissing, errMissing := EvaluateNightlyGateWithListFile(summaryPath, missingList, NightlyGateOptions{})
	if errMissing == nil || repMissing.OK || repMissing.Verdict != "fail" || repMissing.ErrorCount == 0 {
		t.Fatalf("expected missing expected-blocked list to fail gate, got ok=%v verdict=%q err=%v", repMissing.OK, repMissing.Verdict, errMissing)
	}

	// Unreadable expected-blocked list (path is a directory).
	dirAsList := t.TempDir()
	repDir, errDir := EvaluateNightlyGateWithListFile(summaryPath, dirAsList, NightlyGateOptions{})
	if errDir == nil || repDir.OK || repDir.Verdict != "fail" || repDir.ErrorCount == 0 {
		t.Fatalf("expected directory expected-blocked list path to fail gate, got ok=%v verdict=%q err=%v", repDir.OK, repDir.Verdict, errDir)
	}

	// Malformed expected-blocked list (wrong schema / missing required fields / duplicate IDs / invalid JSON).
	for name, body := range map[string]string{
		"invalid_json":    `{not-json`,
		"wrong_schema":    `{"schema":"wrong","entries":[]}`,
		"missing_owner":   `{"schema":"fvpn.e2e.nightly-expected-blocked.v1","entries":[{"id":"member-ipv6","owner":"","reason":"x","established_from":"y"}]}`,
		"missing_reason":  `{"schema":"fvpn.e2e.nightly-expected-blocked.v1","entries":[{"id":"member-ipv6","owner":"o","reason":"","established_from":"y"}]}`,
		"duplicate_entry": `{"schema":"fvpn.e2e.nightly-expected-blocked.v1","entries":[{"id":"member-ipv6","owner":"o","reason":"x","established_from":"y"},{"id":"member-ipv6","owner":"o","reason":"x","established_from":"y"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			badList := filepath.Join(t.TempDir(), "bad-list.json")
			if err := os.WriteFile(badList, []byte(body), 0o644); err != nil {
				t.Fatalf("WriteFile %s: %v", name, err)
			}
			repBad, errBad := EvaluateNightlyGateWithListFile(summaryPath, badList, NightlyGateOptions{})
			if errBad == nil || repBad.OK || repBad.Verdict != "fail" || repBad.ErrorCount == 0 {
				t.Fatalf("expected %s expected-blocked list to fail gate, got ok=%v verdict=%q err=%v", name, repBad.OK, repBad.Verdict, errBad)
			}
		})
	}
}

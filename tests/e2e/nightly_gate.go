package e2e

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const NightlyExpectedBlockedSchema = "fvpn.e2e.nightly-expected-blocked.v1"

type NightlyExpectedBlockedEntry struct {
	ID              string `json:"id"`
	Owner           string `json:"owner"`
	Reason          string `json:"reason"`
	EstablishedFrom string `json:"established_from"`
}

type NightlyExpectedBlockedList struct {
	Schema  string                        `json:"schema"`
	Entries []NightlyExpectedBlockedEntry `json:"entries"`
}

type AcceptanceSummaryRow struct {
	ID     string `json:"id"`
	Spec   string `json:"spec"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
}

type ParsedAcceptanceSummary struct {
	SourceCommit string                 `json:"source_commit"`
	RunID        string                 `json:"run_id"`
	CompletedAt  string                 `json:"completed_at"`
	HeaderPass   int                    `json:"header_pass"`
	HeaderBlock  int                    `json:"header_blocked"`
	HeaderFail   int                    `json:"header_fail"`
	Rows         []AcceptanceSummaryRow `json:"rows"`
}

type NightlyGateOptions struct {
	TestExitCode int
	TestLog      string
	RunnerError  string
}

type NightlyBlockedDisposition struct {
	ID              string `json:"id"`
	Spec            string `json:"spec,omitempty"`
	Status          Status `json:"status"`
	Detail          string `json:"detail,omitempty"`
	Owner           string `json:"owner,omitempty"`
	Reason          string `json:"reason,omitempty"`
	EstablishedFrom string `json:"established_from,omitempty"`
}

type NightlyGateReport struct {
	Schema            string                      `json:"schema"`
	EvaluatedAt       string                      `json:"evaluated_at"`
	SummaryPath       string                      `json:"summary_path"`
	SourceCommit      string                      `json:"source_commit,omitempty"`
	RunID             string                      `json:"run_id,omitempty"`
	OK                bool                        `json:"ok"`
	Verdict           string                      `json:"verdict"`
	PassCount         int                         `json:"pass_count"`
	BlockedCount      int                         `json:"blocked_count"`
	FailCount         int                         `json:"fail_count"`
	ErrorCount        int                         `json:"error_count"`
	ExpectedBlocked   []NightlyBlockedDisposition `json:"expected_blocked"`
	Unblocked         []NightlyBlockedDisposition `json:"unblocked"`
	UnexpectedBlocked []NightlyBlockedDisposition `json:"unexpected_blocked"`
	Failed            []NightlyBlockedDisposition `json:"failed"`
	Errors            []string                    `json:"errors"`
}

func DefaultNightlyExpectedBlockedListPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd for expected-blocked list: %w", err)
	}
	dir := wd
	for {
		if info, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil && !info.IsDir() {
			return filepath.Join(dir, "tests", "e2e", "nightly_expected_blocked.json"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("tests", "e2e", "nightly_expected_blocked.json"), nil
}

func DefaultNightlyExpectedBlockedList() (NightlyExpectedBlockedList, error) {
	path, err := DefaultNightlyExpectedBlockedListPath()
	if err != nil {
		return NightlyExpectedBlockedList{}, err
	}
	return LoadNightlyExpectedBlockedList(path)
}

func LoadNightlyExpectedBlockedList(path string) (NightlyExpectedBlockedList, error) {
	info, err := os.Stat(path)
	if err != nil {
		return NightlyExpectedBlockedList{}, fmt.Errorf("stat expected-blocked list: %w", err)
	}
	if info.IsDir() {
		return NightlyExpectedBlockedList{}, fmt.Errorf("expected-blocked list path %q is a directory", path)
	}
	// Read at run time instead of go:embed because isolation's lint copies only tests/e2e/*.go into a temp dir.
	raw, err := os.ReadFile(path)
	if err != nil {
		return NightlyExpectedBlockedList{}, fmt.Errorf("read expected-blocked list: %w", err)
	}
	return ParseNightlyExpectedBlockedList(raw)
}

func ParseNightlyExpectedBlockedList(raw []byte) (NightlyExpectedBlockedList, error) {
	var list NightlyExpectedBlockedList
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&list); err != nil {
		return NightlyExpectedBlockedList{}, fmt.Errorf("decode expected-blocked list: %w", err)
	}
	if list.Schema != NightlyExpectedBlockedSchema {
		return NightlyExpectedBlockedList{}, fmt.Errorf("unexpected expected-blocked schema %q", list.Schema)
	}
	seen := make(map[string]struct{}, len(list.Entries))
	for _, entry := range list.Entries {
		if !nightlySafeLabel.MatchString(entry.ID) {
			return NightlyExpectedBlockedList{}, fmt.Errorf("invalid expected-blocked scenario id %q", entry.ID)
		}
		if strings.TrimSpace(entry.Owner) == "" {
			return NightlyExpectedBlockedList{}, fmt.Errorf("expected-blocked scenario %q missing owner", entry.ID)
		}
		if strings.TrimSpace(entry.Reason) == "" {
			return NightlyExpectedBlockedList{}, fmt.Errorf("expected-blocked scenario %q missing reason", entry.ID)
		}
		if strings.TrimSpace(entry.EstablishedFrom) == "" {
			return NightlyExpectedBlockedList{}, fmt.Errorf("expected-blocked scenario %q missing established_from", entry.ID)
		}
		if _, dup := seen[entry.ID]; dup {
			return NightlyExpectedBlockedList{}, fmt.Errorf("duplicate expected-blocked scenario id %q", entry.ID)
		}
		seen[entry.ID] = struct{}{}
	}
	return list, nil
}

var (
	nightlySafeLabel     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	summaryTotalsPattern = regexp.MustCompile(`^acceptance summary: pass=([0-9]+) blocked=([0-9]+) fail=([0-9]+) run=([^[:space:]]+) source=([^[:space:]]+)$`)
	markdownLinkPattern  = regexp.MustCompile(`^\[([^\]]+)\]\(([^)]+)\)$`)
)

func nightlyOneLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.TrimSpace(s)
}

func ParseAcceptanceSummaryFile(path string) (ParsedAcceptanceSummary, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ParsedAcceptanceSummary{}, fmt.Errorf("stat acceptance summary: %w", err)
	}
	if info.IsDir() {
		return ParsedAcceptanceSummary{}, fmt.Errorf("acceptance summary path %q is a directory", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ParsedAcceptanceSummary{}, fmt.Errorf("read acceptance summary: %w", err)
	}
	return ParseAcceptanceSummary(raw)
}

func ParseAcceptanceSummary(raw []byte) (ParsedAcceptanceSummary, error) {
	var parsed ParsedAcceptanceSummary
	hasHeading := false
	hasTotals := false
	inTable := false
	seen := map[string]struct{}{}

	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if line == "# acceptance-summary" || line == "# End-to-end Acceptance Summary" {
			hasHeading = true
			continue
		}
		if m := summaryTotalsPattern.FindStringSubmatch(line); m != nil {
			p, errP := strconv.Atoi(m[1])
			b, errB := strconv.Atoi(m[2])
			f, errF := strconv.Atoi(m[3])
			if errP != nil || errB != nil || errF != nil {
				return ParsedAcceptanceSummary{}, fmt.Errorf("invalid summary totals line %q", line)
			}
			parsed.HeaderPass = p
			parsed.HeaderBlock = b
			parsed.HeaderFail = f
			parsed.RunID = m[4]
			parsed.SourceCommit = m[5]
			hasTotals = true
			continue
		}
		if line == "| Scenario | Status |" || line == "| Scenario | Spec | Status | Detail |" {
			inTable = true
			continue
		}
		if inTable && strings.HasPrefix(line, "|---") {
			continue
		}
		if inTable && strings.HasPrefix(line, "|") {
			row, err := parseSummaryTableRow(line)
			if err != nil {
				return ParsedAcceptanceSummary{}, err
			}
			if _, dup := seen[row.ID]; dup {
				return ParsedAcceptanceSummary{}, fmt.Errorf("duplicate scenario row %q in acceptance summary", row.ID)
			}
			seen[row.ID] = struct{}{}
			parsed.Rows = append(parsed.Rows, row)
		}
	}
	if err := sc.Err(); err != nil {
		return ParsedAcceptanceSummary{}, fmt.Errorf("scan acceptance summary: %w", err)
	}
	if !hasHeading {
		return ParsedAcceptanceSummary{}, errors.New("acceptance summary missing heading")
	}
	if !hasTotals {
		return ParsedAcceptanceSummary{}, errors.New("acceptance summary missing totals line")
	}
	if len(parsed.Rows) == 0 {
		return ParsedAcceptanceSummary{}, errors.New("acceptance summary contains no scenario rows")
	}
	passCount, blockedCount, failCount, otherCount := 0, 0, 0, 0
	for _, r := range parsed.Rows {
		switch r.Status {
		case StatusPass:
			passCount++
		case StatusBlocked:
			blockedCount++
		case StatusFail:
			failCount++
		default:
			otherCount++
		}
	}
	if otherCount == 0 && (passCount != parsed.HeaderPass || blockedCount != parsed.HeaderBlock || failCount != parsed.HeaderFail) {
		return ParsedAcceptanceSummary{}, fmt.Errorf(
			"acceptance summary totals mismatch: header pass=%d blocked=%d fail=%d vs rows pass=%d blocked=%d fail=%d",
			parsed.HeaderPass, parsed.HeaderBlock, parsed.HeaderFail, passCount, blockedCount, failCount,
		)
	}
	return parsed, nil
}

func parseSummaryTableRow(line string) (AcceptanceSummaryRow, error) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return AcceptanceSummaryRow{}, fmt.Errorf("malformed summary table row %q", line)
	}
	parts := strings.Split(trimmed[1:len(trimmed)-1], "|")
	var rawScenario, spec, rawStatus, detail string
	switch {
	case len(parts) == 2:
		rawScenario = strings.TrimSpace(parts[0])
		rawStatus = strings.TrimSpace(parts[1])
	case len(parts) >= 4:
		rawScenario = strings.TrimSpace(parts[0])
		spec = strings.TrimSpace(parts[1])
		rawStatus = strings.TrimSpace(parts[2])
		detail = strings.TrimSpace(strings.Join(parts[3:], "|"))
	default:
		return AcceptanceSummaryRow{}, fmt.Errorf("summary table row has %d columns, want 2 or 4: %q", len(parts), line)
	}

	id := rawScenario
	if m := markdownLinkPattern.FindStringSubmatch(rawScenario); m != nil {
		id = strings.TrimSpace(m[1])
	}
	if !nightlySafeLabel.MatchString(id) {
		return AcceptanceSummaryRow{}, fmt.Errorf("invalid scenario id %q in summary row %q", id, line)
	}
	status := strings.Trim(rawStatus, "* ")
	if status == "" {
		return AcceptanceSummaryRow{}, fmt.Errorf("empty status for scenario %q", id)
	}
	return AcceptanceSummaryRow{
		ID:     id,
		Spec:   spec,
		Status: Status(status),
		Detail: detail,
	}, nil
}

func EvaluateNightlyGateWithListFile(summaryPath, listPath string, opts NightlyGateOptions) (NightlyGateReport, error) {
	var list NightlyExpectedBlockedList
	var err error
	if strings.TrimSpace(listPath) != "" {
		list, err = LoadNightlyExpectedBlockedList(listPath)
	} else {
		list, err = DefaultNightlyExpectedBlockedList()
	}
	if err != nil {
		rep := NightlyGateReport{
			Schema:            "fvpn.e2e.nightly-gate-report.v1",
			EvaluatedAt:       time.Now().UTC().Format(time.RFC3339),
			SummaryPath:       summaryPath,
			OK:                false,
			Verdict:           "fail",
			ExpectedBlocked:   []NightlyBlockedDisposition{},
			Unblocked:         []NightlyBlockedDisposition{},
			UnexpectedBlocked: []NightlyBlockedDisposition{},
			Failed:            []NightlyBlockedDisposition{},
			Errors:            []string{fmt.Sprintf("expected-blocked list unreadable or invalid: %v", err)},
			ErrorCount:        1,
		}
		if strings.TrimSpace(opts.RunnerError) != "" {
			rep.Errors = append(rep.Errors, strings.TrimSpace(opts.RunnerError))
			rep.ErrorCount++
		}
		return rep, err
	}
	return EvaluateNightlyGate(summaryPath, list, opts)
}

func EvaluateNightlyGate(summaryPath string, list NightlyExpectedBlockedList, opts NightlyGateOptions) (NightlyGateReport, error) {
	rep := NightlyGateReport{
		Schema:            "fvpn.e2e.nightly-gate-report.v1",
		EvaluatedAt:       time.Now().UTC().Format(time.RFC3339),
		SummaryPath:       summaryPath,
		OK:                true,
		Verdict:           "pass",
		ExpectedBlocked:   []NightlyBlockedDisposition{},
		Unblocked:         []NightlyBlockedDisposition{},
		UnexpectedBlocked: []NightlyBlockedDisposition{},
		Failed:            []NightlyBlockedDisposition{},
		Errors:            []string{},
	}
	if strings.TrimSpace(opts.RunnerError) != "" {
		rep.Errors = append(rep.Errors, strings.TrimSpace(opts.RunnerError))
		rep.ErrorCount++
		rep.OK = false
	}

	parsed, err := ParseAcceptanceSummaryFile(summaryPath)
	if err != nil {
		msg := fmt.Sprintf("acceptance summary unreadable or invalid: %v", err)
		rep.Errors = append(rep.Errors, msg)
		rep.ErrorCount++
		rep.OK = false
		rep.Verdict = "fail"
		return rep, err
	}
	rep.SourceCommit = parsed.SourceCommit
	rep.RunID = parsed.RunID

	expectedByID := make(map[string]NightlyExpectedBlockedEntry, len(list.Entries))
	for _, entry := range list.Entries {
		expectedByID[entry.ID] = entry
	}

	for _, row := range parsed.Rows {
		entry, listed := expectedByID[row.ID]
		switch row.Status {
		case StatusPass:
			rep.PassCount++
			if listed {
				rep.Unblocked = append(rep.Unblocked, NightlyBlockedDisposition{
					ID:              row.ID,
					Spec:            row.Spec,
					Status:          row.Status,
					Detail:          row.Detail,
					Owner:           entry.Owner,
					Reason:          entry.Reason,
					EstablishedFrom: entry.EstablishedFrom,
				})
			}
		case StatusBlocked:
			rep.BlockedCount++
			if listed {
				rep.ExpectedBlocked = append(rep.ExpectedBlocked, NightlyBlockedDisposition{
					ID:              row.ID,
					Spec:            row.Spec,
					Status:          row.Status,
					Detail:          row.Detail,
					Owner:           entry.Owner,
					Reason:          entry.Reason,
					EstablishedFrom: entry.EstablishedFrom,
				})
			} else {
				rep.UnexpectedBlocked = append(rep.UnexpectedBlocked, NightlyBlockedDisposition{
					ID:     row.ID,
					Spec:   row.Spec,
					Status: row.Status,
					Detail: row.Detail,
				})
				rep.OK = false
			}
		case StatusFail:
			rep.FailCount++
			rep.Failed = append(rep.Failed, NightlyBlockedDisposition{
				ID:     row.ID,
				Spec:   row.Spec,
				Status: row.Status,
				Detail: row.Detail,
			})
			rep.OK = false
		default:
			rep.ErrorCount++
			rep.Errors = append(rep.Errors, fmt.Sprintf("scenario %s reported error status %q: %s", row.ID, row.Status, row.Detail))
			rep.OK = false
		}
	}

	if opts.TestExitCode != 0 {
		if errMsg := classifyNonZeroTestExit(opts.TestExitCode, opts.TestLog, rep.BlockedCount, rep.FailCount); errMsg != "" {
			rep.ErrorCount++
			rep.Errors = append(rep.Errors, errMsg)
			rep.OK = false
		}
	}

	if !rep.OK {
		rep.Verdict = "fail"
		return rep, fmt.Errorf(
			"nightly gate failed (pass=%d expected_blocked=%d unexpected_blocked=%d fail=%d error=%d)",
			rep.PassCount, len(rep.ExpectedBlocked), len(rep.UnexpectedBlocked), rep.FailCount, rep.ErrorCount,
		)
	}
	return rep, nil
}

func classifyNonZeroTestExit(exitCode int, testLog string, blockedCount, failCount int) string {
	if exitCode == 0 {
		return ""
	}
	if blockedCount == 0 && failCount == 0 {
		return fmt.Sprintf("e2e-test exited with code %d despite 0 blocked and 0 failed summary rows", exitCode)
	}
	if strings.TrimSpace(testLog) == "" {
		return ""
	}
	sc := bufio.NewScanner(strings.NewReader(testLog))
	sawAcceptanceCompleted := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.Contains(line, "one or more scenarios are blocked or failed") || strings.Contains(line, "acceptance summary: pass=") {
			sawAcceptanceCompleted = true
		}
		if strings.HasPrefix(line, "panic:") {
			return fmt.Sprintf("e2e-test log contains panic: %s", line)
		}
		if strings.HasPrefix(line, "--- FAIL: ") {
			rest := strings.TrimPrefix(line, "--- FAIL: ")
			fields := strings.Fields(rest)
			if len(fields) > 0 && fields[0] != "TestE2E_Acceptance" {
				return fmt.Sprintf("non-acceptance test failed: %s", fields[0])
			}
		}
	}
	if !sawAcceptanceCompleted {
		return fmt.Sprintf("e2e-test exited with code %d without TestE2E_Acceptance summary completion line", exitCode)
	}
	return ""
}

func (r NightlyGateReport) Markdown() string {
	var b strings.Builder
	b.WriteString("## D-144 Nightly Stack Gate\n\n")
	fmt.Fprintf(&b, "- Verdict: **%s**\n", r.Verdict)
	if r.SourceCommit != "" {
		fmt.Fprintf(&b, "- Source commit: `%s`\n", r.SourceCommit)
	}
	if r.RunID != "" {
		fmt.Fprintf(&b, "- Run ID: `%s`\n", r.RunID)
	}
	fmt.Fprintf(&b, "- Evaluated at: `%s`\n", r.EvaluatedAt)
	fmt.Fprintf(&b, "- Counts: pass=%d, expected_blocked=%d, unblocked=%d, unexpected_blocked=%d, fail=%d, error=%d\n\n",
		r.PassCount, len(r.ExpectedBlocked), len(r.Unblocked), len(r.UnexpectedBlocked), r.FailCount, r.ErrorCount)

	if len(r.ExpectedBlocked) > 0 {
		b.WriteString("### Expected-Blocked Rows (acceptance status remains `blocked`)\n\n")
		b.WriteString("| Scenario | Acceptance Status | Owner | Reason | Detail |\n")
		b.WriteString("|---|---|---|---|---|\n")
		for _, item := range r.ExpectedBlocked {
			fmt.Fprintf(&b, "| `%s` | **%s** | %s | %s | %s |\n",
				item.ID, item.Status, nightlyOneLine(item.Owner), nightlyOneLine(item.Reason), nightlyOneLine(item.Detail))
		}
		b.WriteString("\n")
	}

	if len(r.Unblocked) > 0 {
		b.WriteString("### Unblocked Rows (passed; remove from expected-blocked list)\n\n")
		b.WriteString("| Scenario | Acceptance Status | Previous Owner | Previous Reason | Detail |\n")
		b.WriteString("|---|---|---|---|---|\n")
		for _, item := range r.Unblocked {
			fmt.Fprintf(&b, "| `%s` | **%s** | %s | %s | %s |\n",
				item.ID, item.Status, nightlyOneLine(item.Owner), nightlyOneLine(item.Reason), nightlyOneLine(item.Detail))
		}
		b.WriteString("\n")
	}

	if len(r.UnexpectedBlocked) > 0 {
		b.WriteString("### Unexpected Blocked Rows (not on expected-blocked list; fails nightly gate)\n\n")
		b.WriteString("| Scenario | Acceptance Status | Detail |\n")
		b.WriteString("|---|---|---|\n")
		for _, item := range r.UnexpectedBlocked {
			fmt.Fprintf(&b, "| `%s` | **%s** | %s |\n", item.ID, item.Status, nightlyOneLine(item.Detail))
		}
		b.WriteString("\n")
	}

	if len(r.Failed) > 0 {
		b.WriteString("### Failed Rows (fails nightly gate)\n\n")
		b.WriteString("| Scenario | Acceptance Status | Detail |\n")
		b.WriteString("|---|---|---|\n")
		for _, item := range r.Failed {
			fmt.Fprintf(&b, "| `%s` | **%s** | %s |\n", item.ID, item.Status, nightlyOneLine(item.Detail))
		}
		b.WriteString("\n")
	}

	if len(r.Errors) > 0 {
		b.WriteString("### Errors (fails nightly gate)\n\n")
		for _, msg := range r.Errors {
			fmt.Fprintf(&b, "- %s\n", nightlyOneLine(msg))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func WriteNightlyGateArtifacts(runDir string, rep NightlyGateReport, stepSummaryPath string) error {
	md := rep.Markdown()
	if strings.TrimSpace(runDir) != "" {
		if info, err := os.Stat(runDir); err == nil && info.IsDir() {
			if err := os.WriteFile(filepath.Join(runDir, "nightly-gate.md"), []byte(md), 0o644); err != nil {
				return fmt.Errorf("write nightly-gate.md: %w", err)
			}
		}
	}
	if strings.TrimSpace(stepSummaryPath) != "" {
		f, err := os.OpenFile(stepSummaryPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open step summary %q: %w", stepSummaryPath, err)
		}
		defer f.Close()
		if rep.SummaryPath != "" {
			if rawSummary, err := os.ReadFile(rep.SummaryPath); err == nil && len(rawSummary) > 0 {
				if _, err := f.Write(rawSummary); err != nil {
					return fmt.Errorf("write acceptance summary to step summary: %w", err)
				}
				if !strings.HasSuffix(string(rawSummary), "\n") {
					_, _ = f.WriteString("\n")
				}
				_, _ = f.WriteString("\n")
			}
		}
		if _, err := f.WriteString(md); err != nil {
			return fmt.Errorf("write nightly gate markdown to step summary: %w", err)
		}
	}
	return nil
}

func ResolveNightlySummaryPath(runsRoot, label string) (string, error) {
	if !runLabelPattern.MatchString(label) {
		return "", fmt.Errorf("invalid run label %q", label)
	}
	pattern := filepath.Join(runsRoot, "*", label, "acceptance-summary.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob acceptance summary: %w", err)
	}
	if len(matches) == 0 {
		return filepath.Join(runsRoot, time.Now().UTC().Format("2006-01-02"), label, "acceptance-summary.md"), nil
	}
	best := matches[0]
	var bestMod time.Time
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && info.ModTime().After(bestMod) {
			best = m
			bestMod = info.ModTime()
		}
	}
	return best, nil
}

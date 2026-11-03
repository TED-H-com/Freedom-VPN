package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// RunLabelEnv optionally names the evidence directory below
// docs/e2e/runs/<UTC date>/. The nightly stack job sets it so the records of
// that run are separable from committed history and can be uploaded alone.
const RunLabelEnv = "FVPN_E2E_RUN_LABEL"

// SummaryFile is written next to the scenario records by runners that
// produce an acceptance summary.
const SummaryFile = "acceptance-summary.md"

var runLabelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(/[a-z0-9][a-z0-9-]*)*$`)

// RunDir returns docs/e2e/runs/<UTC date>/<label>. An empty label selects
// fallback. Labels are lowercase slash-separated segments, so a label can
// never leave docs/e2e/runs/<date>.
func RunDir(root, label, fallback string, now time.Time) (string, error) {
	if label == "" {
		label = fallback
	}
	if len(label) > 128 || !runLabelPattern.MatchString(label) {
		return "", fmt.Errorf("%s must be lowercase [a-z0-9-] path segments, got %q", RunLabelEnv, label)
	}
	return filepath.Join(root, "docs", "e2e", "runs", now.UTC().Format("2006-01-02"), filepath.FromSlash(label)), nil
}

// SummaryRow is one recorded scenario outcome, in record order.
type SummaryRow struct {
	ID     string
	Status Status
}

// SummaryLine is the acceptance summary line shared by the test log and the
// summary file.
func SummaryLine(rows []SummaryRow, runID, source string) string {
	counts := map[Status]int{}
	for _, row := range rows {
		counts[row.Status]++
	}
	return fmt.Sprintf("acceptance summary: pass=%d blocked=%d fail=%d run=%s source=%s", counts[StatusPass], counts[StatusBlocked], counts[StatusFail], runID, source)
}

// WriteSummary stores the acceptance summary and every scenario status. It
// holds scenario IDs and statuses only; observations stay in the records.
func WriteSummary(dir string, rows []SummaryRow, runID, source string) error {
	var b strings.Builder
	b.WriteString("# acceptance-summary\n\n")
	b.WriteString(SummaryLine(rows, runID, source) + "\n\n")
	b.WriteString("A blocked scenario is still non-green; see each record for its owner.\n\n")
	b.WriteString("| Scenario | Status |\n|---|---|\n")
	for _, row := range rows {
		b.WriteString("| " + Redact(row.ID) + " | " + string(row.Status) + " |\n")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, SummaryFile), []byte(b.String()), 0o644)
}

// maxProbeDetail bounds one browser-probe diagnostic line.
const maxProbeDetail = 400

// ProbeFailureDetail returns the first non-empty line of a failed browser
// probe's stderr, bounded. Every secret (the session cookie values given to
// the probe) is removed first, then the worktree and home directory become
// <worktree> and <home>, then Redact runs. The record then names the missing
// piece (for example a browser executable) without leaking a cookie or path.
func ProbeFailureDetail(stderr []byte, secrets []string, root, home string) string {
	text := string(stderr)
	for _, secret := range secrets {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[redacted]")
		}
	}
	if root != "" && root != "/" {
		text = strings.ReplaceAll(text, root, "<worktree>")
	}
	if home != "" && home != "/" {
		text = strings.ReplaceAll(text, home, "<home>")
	}
	line := ""
	for _, candidate := range strings.Split(text, "\n") {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			line = candidate
			break
		}
	}
	if len(line) > maxProbeDetail {
		cut := maxProbeDetail
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut--
		}
		line = line[:cut] + "…"
	}
	return Redact(line)
}

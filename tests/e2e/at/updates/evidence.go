package updates

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// writeAcceptance saves every record before rejecting incomplete acceptance.
// Extra diagnostic records do not substitute for any assigned AT (§19).
func writeAcceptance(dir string, records map[string]*Record) error {
	var failures []error
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if records[id] == nil {
			failures = append(failures, fmt.Errorf("%s: missing record", id))
			continue
		}
		if err := Write(dir, *records[id]); err != nil {
			failures = append(failures, fmt.Errorf("write %s: %w", id, err))
		}
	}
	for _, id := range []string{"at23", "at24", "at25", "at26", "at27"} {
		rec := records[id]
		if rec == nil {
			failures = append(failures, fmt.Errorf("%s: missing assigned AT", id))
		} else if rec.Status != StatusPass {
			failures = append(failures, fmt.Errorf("%s: %s; see docs/e2e/at/updates/%s.json", id, rec.Status, id))
		}
	}
	return errors.Join(failures...)
}

// Status is the claim a case is allowed to make. pass is something the run
// observed. blocked is a product or lab precondition the run reached.
// needs_operator_run is an exact manual script for hardware this host does
// not have. fail is an unexpected harness break.
type Status string

const (
	StatusPass             Status = "pass"
	StatusBlocked          Status = "blocked"
	StatusNeedsOperatorRun Status = "needs_operator_run"
	StatusFail             Status = "fail"
)

// Observation is one fact safe to publish. Detail must not carry secrets.
type Observation struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// Record is one acceptance case (§19). Paths, tokens and DSNs are redacted
// before the file is written.
type Record struct {
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Spec            []string      `json:"spec"`
	Status          Status        `json:"status"`
	At              string        `json:"at"`
	Expected        string        `json:"expected"`
	Actual          string        `json:"actual"`
	Git             string        `json:"git,omitempty"`
	Images          string        `json:"images,omitempty"`
	PolicyRevision  string        `json:"policy_revision,omitempty"`
	NodeIncarnation string        `json:"node_incarnation,omitempty"`
	Commands        []string      `json:"commands,omitempty"`
	Observations    []Observation `json:"observations"`
}

func mustCompile(parts ...string) *regexp.Regexp {
	return regexp.MustCompile(strings.Join(parts, ""))
}

// Patterns are assembled so the source does not contain secret-shaped literals.
var redactPatterns = []*regexp.Regexp{
	mustCompile("/", "home/", `[^[:space:]"']+`),
	regexp.MustCompile(`(?i)postgres://[^[:space:]"']+`),
	regexp.MustCompile(`(?i)bearer[[:space:]]+[^[:space:]"']+`),
	mustCompile("sk_", "live_", `[A-Za-z0-9]+`),
	mustCompile("gh", "p_", `[A-Za-z0-9]+`),
	mustCompile("gh", "o_", `[A-Za-z0-9]+`),
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`(?i)"(token|setup_key|password|secret|authorization|seed_b64|api_key)"[[:space:]]*:[[:space:]]*"[^"]*"`),
}

// Redact removes secret-shaped substrings from evidence text.
func Redact(s string) string {
	jsonSecret := redactPatterns[len(redactPatterns)-1]
	s = jsonSecret.ReplaceAllString(s, `"$1": "[redacted]"`)
	for _, p := range redactPatterns[:len(redactPatterns)-1] {
		s = p.ReplaceAllString(s, "[redacted]")
	}
	return s
}

// Write stores a JSON record and a short Markdown page.
func Write(dir string, rec Record) error {
	if rec.At == "" {
		rec.At = time.Now().UTC().Format(time.RFC3339)
	}
	rec.Title = Redact(rec.Title)
	rec.Expected = Redact(rec.Expected)
	rec.Actual = Redact(rec.Actual)
	rec.Git = Redact(rec.Git)
	rec.Images = Redact(rec.Images)
	for i := range rec.Commands {
		rec.Commands[i] = Redact(rec.Commands[i])
	}
	for i := range rec.Observations {
		rec.Observations[i].Name = Redact(rec.Observations[i].Name)
		rec.Observations[i].Detail = Redact(rec.Observations[i].Detail)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	base := filepath.Join(dir, rec.ID)
	if err := os.WriteFile(base+".json", raw, 0o644); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# " + rec.ID + "\n\n")
	b.WriteString(rec.Title + "\n\n")
	b.WriteString("Status: " + string(rec.Status) + "\n\n")
	if len(rec.Spec) > 0 {
		b.WriteString("Spec: " + strings.Join(rec.Spec, ", ") + "\n\n")
	}
	b.WriteString("At: " + rec.At + "\n\n")
	if rec.Expected != "" {
		b.WriteString("Expected: " + rec.Expected + "\n\n")
	}
	if rec.Actual != "" {
		b.WriteString("Actual: " + rec.Actual + "\n\n")
	}
	if rec.Git != "" {
		b.WriteString("Git: " + rec.Git + "\n\n")
	}
	if rec.Images != "" {
		b.WriteString("Images: " + rec.Images + "\n\n")
	}
	if rec.PolicyRevision != "" {
		b.WriteString("Policy revision: " + rec.PolicyRevision + "\n\n")
	}
	if rec.NodeIncarnation != "" {
		b.WriteString("Node incarnation: " + rec.NodeIncarnation + "\n\n")
	}
	if len(rec.Commands) > 0 {
		b.WriteString("Commands:\n\n")
		for _, c := range rec.Commands {
			b.WriteString("- " + c + "\n")
		}
		b.WriteString("\n")
	}
	for _, ob := range rec.Observations {
		b.WriteString(strings.TrimRight("- "+ob.Name+": "+ob.Detail, " \t") + "\n")
	}
	b.WriteString("\n")
	return os.WriteFile(base+".md", []byte(b.String()), 0o644)
}

// RollbackFacts is the journal evidence used to explain a first-install
// ROLLBACK_FAILED (§13.9, §13.11).
type RollbackFacts struct {
	HealthJSON      string
	ErrorCode       string
	ErrorDetail     string
	LastGood        string
	RollbackRelease string
	Events          string
}

const (
	FindingNotReproduced = "not_reproduced"
	FindingNoLastGood    = "health_failed_without_last_good"
	FindingInconclusive  = "inconclusive"
)

// ClassifyFirstInstallRollback reports why a first install ended in
// recovery. This classifies journal facts, not product correctness: a missing
// probe adapter needs code/spec analysis, and an existing last-good release
// can legitimately be rejected by rollback safety checks (§13.9).
func ClassifyFirstInstallRollback(in RollbackFacts) string {
	if in.ErrorCode != "ROLLBACK_FAILED" {
		return FindingNotReproduced
	}
	events := strings.ToLower(in.Events + " " + in.ErrorDetail)
	restored := in.LastGood != "" || in.RollbackRelease != "" || strings.Contains(events, "symlink") || strings.Contains(events, "synctree")
	if restored {
		return FindingInconclusive
	}
	health := strings.ReplaceAll(strings.ToLower(in.HealthJSON), " ", "")
	failedClosed := strings.Contains(health, `"dns_policy":"failed"`) &&
		strings.Contains(health, `"internet_only":"failed"`) &&
		strings.Contains(health, `"tunnel_probe":"failed"`)
	if failedClosed {
		return FindingNoLastGood
	}
	return FindingInconclusive
}

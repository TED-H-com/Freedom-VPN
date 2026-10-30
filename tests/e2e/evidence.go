package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Status is the claim a scenario is allowed to make. pass is something the
// run observed. blocked is a dependency or measurement gap the run reached and recorded.
// fail is an unexpected break in the harness or a step that should already work.
type Status string

const (
	StatusPass    Status = "pass"
	StatusBlocked Status = "blocked"
	StatusFail    Status = "fail"
)

// Observation is one fact safe to publish. Detail must not carry secrets.
type Observation struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// Record is one scenario's evidence (§19).
type Record struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Spec         []string      `json:"spec"`
	Status       Status        `json:"status"`
	Observations []Observation `json:"observations"`
	At           string        `json:"at"`
	RunID        string        `json:"run_id,omitempty"`
	SourceCommit string        `json:"source_commit,omitempty"`
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
	regexp.MustCompile(`(?i)user_code=[^[:space:]&"']+`),
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

// Write stores a JSON record and a short Markdown summary. The directory is
// docs/e2e/runs/<date>/ and the files contain no secrets.
func Write(dir string, rec Record) error {
	if rec.At == "" {
		rec.At = time.Now().UTC().Format(time.RFC3339)
	}
	for i := range rec.Observations {
		switch rec.Observations[i].Name {
		case "portal_sub", "netbird_user_id", "peer_user_id", "external_user_id":
			rec.Observations[i].Detail = "[redacted]"
		}
		rec.Observations[i].Detail = Redact(rec.Observations[i].Detail)
		rec.Observations[i].Name = Redact(rec.Observations[i].Name)
	}
	rec.Title = Redact(rec.Title)
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
	if rec.RunID != "" {
		b.WriteString("Run: " + rec.RunID + "\n\nSource: " + rec.SourceCommit + "\n\n")
	}
	if len(rec.Spec) > 0 {
		b.WriteString("Spec: " + strings.Join(rec.Spec, ", ") + "\n\n")
	}
	for _, ob := range rec.Observations {
		b.WriteString("- " + ob.Name + ": " + ob.Detail + "\n")
	}
	b.WriteString("\n")
	lines := strings.Split(b.String(), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return os.WriteFile(base+".md", []byte(strings.TrimRight(strings.Join(lines, "\n"), "\n")+"\n"), 0o644)
}

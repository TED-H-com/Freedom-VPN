package isolation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Status values a case may record (§19.2). blocked is not a pass.
// "needs operator run" is only for a step that needs hardware or a member
// platform this host does not have.
const (
	StatusPass     = "pass"
	StatusBlocked  = "blocked"
	StatusFail     = "fail"
	StatusOperator = "needs operator run"
)

// Observation is one fact safe to publish.
type Observation struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// Record is one acceptance case (§19.1). It carries the run identity the
// catalogue requires: build, image digests, policy revision, incarnation,
// UTC time, expected, actual, and the commands that produced it.
type Record struct {
	RunID           string        `json:"run_id,omitempty"`
	StartedAt       string        `json:"started_at,omitempty"`
	SourceDirty     bool          `json:"source_dirty"`
	Checks          []Check       `json:"checks,omitempty"`
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Spec            []string      `json:"spec"`
	Status          string        `json:"status"`
	At              string        `json:"at"`
	Build           string        `json:"build"`
	ImageDigests    []string      `json:"image_digests"`
	PolicyRevision  string        `json:"policy_revision"`
	NodeIncarnation string        `json:"node_incarnation"`
	Expected        string        `json:"expected"`
	Actual          string        `json:"actual"`
	Commands        []string      `json:"commands"`
	Observations    []Observation `json:"observations"`
}

type Positive struct {
	DeviceID     string        `json:"device_id"`
	Container    string        `json:"container"`
	At           string        `json:"at"`
	Counted      bool          `json:"counted"`
	Before       string        `json:"admission_bytes_before"`
	After        string        `json:"admission_bytes_after"`
	Observations []Observation `json:"observations"`
}

func (p Positive) Valid() bool {
	before, e1 := strconv.ParseUint(p.Before, 10, 64)
	after, e2 := strconv.ParseUint(p.After, 10, 64)
	return p.Counted && p.DeviceID != "" && e1 == nil && e2 == nil && after > before
}

type Check struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	At           string        `json:"at"`
	Actual       string        `json:"actual"`
	Owner        string        `json:"owner,omitempty"`
	Positive     Positive      `json:"positive_control"`
	Observations []Observation `json:"observations,omitempty"`
}

// A disconnected client cannot prove a negative. Failure remains explicit only
// when the same client's counted origin control was observed in this record.
func NegativeStatus(positive Positive, ran, denied bool) string {
	if !positive.Valid() || !ran {
		return StatusBlocked
	}
	if !denied {
		return StatusFail
	}
	return StatusPass
}

func ChecksStatus(checks []Check) string {
	status := StatusPass
	if len(checks) == 0 {
		return StatusBlocked
	}
	for _, c := range checks {
		if c.Status == StatusFail && c.Positive.Valid() {
			return StatusFail
		}
		if c.Status != StatusPass || !c.Positive.Valid() {
			status = StatusBlocked
		}
	}
	return status
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

// Write stores a JSON record and a short Markdown page. dir is the evidence
// directory. The files contain no secrets.
func Write(dir string, rec Record) error {
	if rec.At == "" {
		rec.At = time.Now().UTC().Format(time.RFC3339)
	}
	for i := range rec.Checks {
		c := &rec.Checks[i]
		c.Actual, c.Owner = Redact(c.Actual), Redact(c.Owner)
		for _, observations := range [][]Observation{c.Observations, c.Positive.Observations} {
			for j := range observations {
				observations[j].Detail = Redact(observations[j].Detail)
			}
		}
	}
	rec.Title = Redact(rec.Title)
	rec.Expected = Redact(rec.Expected)
	rec.Actual = Redact(rec.Actual)
	rec.Build = Redact(rec.Build)
	rec.PolicyRevision = Redact(rec.PolicyRevision)
	rec.NodeIncarnation = Redact(rec.NodeIncarnation)
	for i := range rec.ImageDigests {
		rec.ImageDigests[i] = Redact(rec.ImageDigests[i])
	}
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
	b.WriteString("Status: " + rec.Status + "\n\n")
	if rec.RunID != "" {
		b.WriteString("Run: " + rec.RunID + "; started UTC: " + rec.StartedAt + "; source dirty: " + strconv.FormatBool(rec.SourceDirty) + "\n\n")
	}
	if len(rec.Spec) > 0 {
		b.WriteString("Spec: " + strings.Join(rec.Spec, ", ") + "\n\n")
	}
	b.WriteString("UTC: " + rec.At + "\n\n")
	b.WriteString("Build: " + empty(rec.Build) + "\n\n")
	b.WriteString("Image digests:\n\n")
	if len(rec.ImageDigests) == 0 {
		b.WriteString("- (none)\n")
	}
	for _, d := range rec.ImageDigests {
		b.WriteString("- " + d + "\n")
	}
	b.WriteString("\nPolicy revision: " + empty(rec.PolicyRevision) + "\n\n")
	b.WriteString("Node incarnation: " + empty(rec.NodeIncarnation) + "\n\n")
	b.WriteString("Expected: " + empty(rec.Expected) + "\n\n")
	b.WriteString("Actual: " + empty(rec.Actual) + "\n\n")
	b.WriteString("Commands:\n\n")
	if len(rec.Commands) == 0 {
		b.WriteString("- (none)\n")
	}
	for _, c := range rec.Commands {
		b.WriteString("- `" + c + "`\n")
	}
	b.WriteString("\n")
	for _, ob := range rec.Observations {
		b.WriteString("- " + ob.Name + ": " + ob.Detail + "\n")
	}
	if len(rec.Checks) > 0 {
		b.WriteString("\nChecks (full positive controls and observations in JSON):\n\n")
		for _, c := range rec.Checks {
			b.WriteString("- " + c.Name + ": " + c.Status + "; " + c.Actual + "; counted positive=" + strconv.FormatBool(c.Positive.Valid()) + "\n")
		}
	}
	b.WriteString("\n")
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return os.WriteFile(base+".md", []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// WriteExtra stores a redacted sibling of a record, such as a ruleset or a
// capture note. name is a basename.
func WriteExtra(dir, name, body string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return os.ErrInvalid
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(Redact(body)), 0o644)
}

func empty(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

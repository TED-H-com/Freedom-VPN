// Package usage implements the Wave 4 acceptance suite and evidence writer for
// AT14–AT19 (§10 Usage Metering and Ledger, §11 Fair-Use Policy and Rate Shaping,
// §17.3 Ledger Recovery, §19 Acceptance Cases AT14–AT19).
package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Status values allowed by §19.1 and docs/e2e/README.md.
const (
	StatusPass           = "pass"
	StatusFail           = "fail"
	StatusBlocked        = "blocked"
	StatusManualRequired = "manual_required"
)

// BuildMetadata records the tested commit, Go toolchain, and compiled binary hashes (§19.1).
type BuildMetadata struct {
	GitCommit string            `json:"git_commit"`
	GitDirty  bool              `json:"git_dirty"`
	GoVersion string            `json:"go_version"`
	GOOS      string            `json:"goos"`
	GOARCH    string            `json:"goarch"`
	Binaries  map[string]string `json:"binaries,omitempty"`
}

// CommandRecord captures a single command or API call executed during an AT case (§19.1).
type CommandRecord struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output,omitempty"`
	At       string `json:"at"`
}

// MechanismCheck captures a deterministic component or integration verification step
// proving an AT deliverable against the real production packages (internal/usage,
// internal/fup, node/internal/meter, node/internal/usagestore, node/internal/usageupload,
// node/enforce). Supporting mechanism checks never turn a blocked product AT green.
type MechanismCheck struct {
	Name    string         `json:"name"`
	SpecRef string         `json:"spec_ref"`
	Status  string         `json:"status"`
	Summary string         `json:"summary"`
	Details map[string]any `json:"details,omitempty"`
}

// Record is the §19.1 acceptance evidence structure written to docs/e2e/at/usage/AT<NN>.json.
type Record struct {
	ID              string            `json:"id"`
	RunID           string            `json:"run_id,omitempty"`
	Title           string            `json:"title"`
	Spec            string            `json:"spec"`
	Requirements    []string          `json:"requirements"`
	Status          string            `json:"status"`
	At              string            `json:"at"`
	Build           BuildMetadata     `json:"build"`
	ImageDigests    map[string]string `json:"image_digests"`
	PolicyRevision  string            `json:"policy_revision"`
	NodeIncarnation string            `json:"node_incarnation"`
	Expected        string            `json:"expected"`
	Actual          string            `json:"actual"`
	Blockers        []string          `json:"blockers,omitempty"`
	Commands        []CommandRecord   `json:"commands"`
	Observations    map[string]any    `json:"observations"`
	MechanismChecks []MechanismCheck  `json:"mechanism_checks,omitempty"`
}

var (
	rePEMBlock     = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE ` + `KEY-----.*?-----END [A-Z0-9 ]*PRIVATE ` + `KEY-----`)
	reAuthHeader   = regexp.MustCompile(`(?i)(Authorization:\s*(?:Bearer|Token)\s+)[^\s"']+`)
	reJSONSecret   = regexp.MustCompile(`(?i)("(?:setup_key|enrollment_` + `token|webhook_secret|client_secret|password|private_key|secret)"\s*:\s*")[^"]+(")`)
	reQueryToken   = regexp.MustCompile(`(?i)((?:token|secret|password|setup_key)=)[^&\s"']+`)
	reHomePath     = regexp.MustCompile(`/ho` + `me/[^\s"':]+`)
	reSkLive       = regexp.MustCompile(`sk_` + `live_[A-Za-z0-9_]+`)
	reGitHubPat    = regexp.MustCompile(`gh[po]_[A-Za-z0-9_]+`)
	reLabOriginIP  = regexp.MustCompile(`\b192\.0\.[3-9]\.`)
	reDockerBridge = regexp.MustCompile(`\b172\.(?:1[6-9]|2[0-9]|3[0-1])\.[0-9]+\.`)
)

// Redact removes private keys, bearer/setup/enrollment tokens, webhook secrets,
// passwords, host home paths, and non-RFC5737 lab bridge IPs before writing evidence (§4, §8, SEC02).
func Redact(s string, extraSecrets ...string) string {
	out := s
	for _, sec := range extraSecrets {
		trimmed := strings.TrimSpace(sec)
		if len(trimmed) >= 6 {
			out = strings.ReplaceAll(out, trimmed, "[REDACTED]")
		}
	}
	out = rePEMBlock.ReplaceAllString(out, "[REDACTED PEM KEY]")
	out = reAuthHeader.ReplaceAllString(out, "${1}[REDACTED]")
	out = reJSONSecret.ReplaceAllString(out, "${1}[REDACTED]${2}")
	out = reQueryToken.ReplaceAllString(out, "${1}[REDACTED]")
	out = reHomePath.ReplaceAllString(out, "$REPO")
	out = reSkLive.ReplaceAllString(out, "[REDACTED]")
	out = reGitHubPat.ReplaceAllString(out, "[REDACTED]")
	out = reLabOriginIP.ReplaceAllString(out, "198.51.100.")
	out = reDockerBridge.ReplaceAllString(out, "192.0.2.")
	return out
}

func redactAny(v any, extraSecrets ...string) any {
	switch val := v.(type) {
	case string:
		return Redact(val, extraSecrets...)
	case []string:
		res := make([]string, len(val))
		for i, item := range val {
			res[i] = Redact(item, extraSecrets...)
		}
		return res
	case []any:
		res := make([]any, len(val))
		for i, item := range val {
			res[i] = redactAny(item, extraSecrets...)
		}
		return res
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[k] = redactAny(item, extraSecrets...)
		}
		return res
	case map[string]string:
		res := make(map[string]string, len(val))
		for k, item := range val {
			res[k] = Redact(item, extraSecrets...)
		}
		return res
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return v
		}
		redacted := Redact(string(raw), extraSecrets...)
		var decoded any
		if err := json.Unmarshal([]byte(redacted), &decoded); err != nil {
			return redacted
		}
		return decoded
	}
}

// ValidateRecord checks that all mandatory §19.1 fields are populated and valid.
func ValidateRecord(r Record) error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("record id is required")
	}
	if strings.TrimSpace(r.Title) == "" {
		return fmt.Errorf("%s: title is required", r.ID)
	}
	if strings.TrimSpace(r.Spec) == "" {
		return fmt.Errorf("%s: spec is required", r.ID)
	}
	switch r.Status {
	case StatusPass, StatusFail, StatusBlocked, StatusManualRequired:
	default:
		return fmt.Errorf("%s: invalid status %q", r.ID, r.Status)
	}
	if strings.TrimSpace(r.At) == "" {
		return fmt.Errorf("%s: timestamp 'at' is required", r.ID)
	}
	if strings.TrimSpace(r.Build.GitCommit) == "" {
		return fmt.Errorf("%s: build.git_commit is required", r.ID)
	}
	if strings.TrimSpace(r.Expected) == "" || strings.TrimSpace(r.Actual) == "" {
		return fmt.Errorf("%s: expected and actual are required", r.ID)
	}
	if len(r.Commands) == 0 {
		return fmt.Errorf("%s: at least one command record is required", r.ID)
	}
	if len(r.Observations) == 0 {
		return fmt.Errorf("%s: observations must not be empty", r.ID)
	}
	return nil
}

// WriteEvidence redacts and writes docs/e2e/at/usage/<ID>.json and <ID>.md.
func WriteEvidence(dir string, r Record, extraSecrets ...string) error {
	if r.At == "" {
		r.At = time.Now().UTC().Format(time.RFC3339)
	}
	if r.Build.GoVersion == "" {
		r.Build.GoVersion = runtime.Version()
	}
	if r.Build.GOOS == "" {
		r.Build.GOOS = runtime.GOOS
	}
	if r.Build.GOARCH == "" {
		r.Build.GOARCH = runtime.GOARCH
	}
	if r.ImageDigests == nil {
		r.ImageDigests = map[string]string{}
	}
	if r.Observations == nil {
		r.Observations = map[string]any{}
	}

	r.Title = Redact(r.Title, extraSecrets...)
	r.Spec = Redact(r.Spec, extraSecrets...)
	r.PolicyRevision = Redact(r.PolicyRevision, extraSecrets...)
	r.NodeIncarnation = Redact(r.NodeIncarnation, extraSecrets...)
	r.Expected = Redact(r.Expected, extraSecrets...)
	r.Actual = Redact(r.Actual, extraSecrets...)
	for i := range r.Blockers {
		r.Blockers[i] = Redact(r.Blockers[i], extraSecrets...)
	}
	for i := range r.Commands {
		r.Commands[i].Command = Redact(r.Commands[i].Command, extraSecrets...)
		r.Commands[i].Output = Redact(r.Commands[i].Output, extraSecrets...)
		if r.Commands[i].At == "" {
			r.Commands[i].At = r.At
		}
	}
	if obs, ok := redactAny(r.Observations, extraSecrets...).(map[string]any); ok {
		r.Observations = obs
	}
	for i := range r.MechanismChecks {
		r.MechanismChecks[i].Summary = Redact(r.MechanismChecks[i].Summary, extraSecrets...)
		if det, ok := redactAny(r.MechanismChecks[i].Details, extraSecrets...).(map[string]any); ok {
			r.MechanismChecks[i].Details = det
		}
	}

	if err := ValidateRecord(r); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	jsonBytes = append(jsonBytes, '\n')
	if err := os.WriteFile(filepath.Join(dir, r.ID+".json"), jsonBytes, 0o644); err != nil {
		return err
	}

	md := renderMarkdown(r)
	lines := strings.Split(md, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	md = strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	return os.WriteFile(filepath.Join(dir, r.ID+".md"), []byte(md), 0o644)
}

func renderMarkdown(r Record) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s: %s\n\n", r.ID, r.Title)
	fmt.Fprintf(&b, "| Field | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| **Case ID** | `%s` |\n", r.ID)
	if r.RunID != "" {
		fmt.Fprintf(&b, "| **Run ID** | `%s` |\n", r.RunID)
	}
	fmt.Fprintf(&b, "| **Spec Reference** | `%s` |\n", r.Spec)
	fmt.Fprintf(&b, "| **Requirements** | `%s` |\n", strings.Join(r.Requirements, ", "))
	fmt.Fprintf(&b, "| **Status** | `%s` |\n", r.Status)
	fmt.Fprintf(&b, "| **Recorded At** | `%s` |\n", r.At)
	fmt.Fprintf(&b, "| **Git Commit** | `%s` (dirty: `%t`) |\n", r.Build.GitCommit, r.Build.GitDirty)
	fmt.Fprintf(&b, "| **Go Toolchain** | `%s` (`%s/%s`) |\n", r.Build.GoVersion, r.Build.GOOS, r.Build.GOARCH)
	fmt.Fprintf(&b, "| **Active Policy Revisions** | `%s` |\n", r.PolicyRevision)
	fmt.Fprintf(&b, "| **Node Incarnation** | `%s` |\n\n", r.NodeIncarnation)

	fmt.Fprintf(&b, "## Expected (§19)\n\n%s\n\n", r.Expected)
	fmt.Fprintf(&b, "## Actual\n\n%s\n\n", r.Actual)

	if len(r.Blockers) > 0 {
		fmt.Fprintf(&b, "## Product Precondition Blockers\n\n")
		for _, blk := range r.Blockers {
			fmt.Fprintf(&b, "- %s\n", blk)
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(r.MechanismChecks) > 0 {
		fmt.Fprintf(&b, "## Mechanism & Component Verification\n\n")
		fmt.Fprintf(&b, "> Supporting mechanism checks execute the production packages (`internal/usage`, `internal/fup`, `node/internal/meter`, `node/internal/usagestore`, `node/internal/usageupload`, `node/enforce`) and in-container netns data-plane probes (`fv_at14`, `fv_at18`). They do not override the assembled product status `%s`.\n\n", r.Status)
		fmt.Fprintf(&b, "| Subcase | Spec | Status | Summary |\n|---|---|---|---|\n")
		for _, m := range r.MechanismChecks {
			fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %s |\n", m.Name, m.SpecRef, m.Status, strings.ReplaceAll(m.Summary, "|", "\\|"))
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(r.ImageDigests) > 0 {
		fmt.Fprintf(&b, "## Assigned Stack Image Digests (`w4-usage` / port base `48000`)\n\n")
		keys := make([]string, 0, len(r.ImageDigests))
		for k := range r.ImageDigests {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&b, "| Service | Image Digest |\n|---|---|\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "| `%s` | `%s` |\n", k, r.ImageDigests[k])
		}
		fmt.Fprintf(&b, "\n")
	}

	obsJSON, _ := json.MarshalIndent(r.Observations, "", "  ")
	fmt.Fprintf(&b, "## Structured Observations\n\n```json\n%s\n```\n\n", string(obsJSON))

	fmt.Fprintf(&b, "## Command & API Transcript\n\n")
	for i, cmd := range r.Commands {
		fmt.Fprintf(&b, "### %d. `%s` (exit `%d`, at `%s`)\n\n", i+1, cmd.Command, cmd.ExitCode, cmd.At)
		if strings.TrimSpace(cmd.Output) != "" {
			fmt.Fprintf(&b, "```text\n%s\n```\n\n", strings.TrimSpace(cmd.Output))
		}
	}
	return b.String()
}

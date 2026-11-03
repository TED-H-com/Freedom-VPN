package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunDir_LabelSelectsConfinedDirectory(t *testing.T) {
	now := time.Date(2026, 11, 3, 4, 30, 0, 0, time.UTC)
	root := filepath.FromSlash("/srv/fvpn")
	cases := []struct {
		label, want string
	}{
		{"", "/srv/fvpn/docs/e2e/runs/2026-11-03/followup15"},
		{"nightly", "/srv/fvpn/docs/e2e/runs/2026-11-03/nightly"},
		{"followup29/after-full", "/srv/fvpn/docs/e2e/runs/2026-11-03/followup29/after-full"},
	}
	for _, c := range cases {
		got, err := RunDir(root, c.label, "followup15", now)
		if err != nil || got != filepath.FromSlash(c.want) {
			t.Errorf("RunDir(%q) = %q, %v; want %q", c.label, got, err, c.want)
		}
	}
	// The UTC date, not the local one, names the directory.
	if got, _ := RunDir(root, "x", "", time.Date(2026, 11, 2, 23, 0, 0, 0, time.FixedZone("EST", -5*3600))); !strings.Contains(got, "2026-11-03") {
		t.Errorf("RunDir local date leak: %q", got)
	}
	for _, bad := range []string{"../x", "a/../../b", "/abs", "Nightly", "a//b", "a/", "-a", "a b", strings.Repeat("a", 129)} {
		if got, err := RunDir(root, bad, "followup15", now); err == nil {
			t.Errorf("RunDir accepted %q as %q", bad, got)
		}
	}
}

func TestWriteSummary_CountsEveryStatus(t *testing.T) {
	dir := t.TempDir()
	rows := []SummaryRow{{"stack-health", StatusPass}, {"portal-billing-return", StatusFail}, {"billing", StatusFail}, {"member-ipv6", StatusBlocked}}
	if err := WriteSummary(dir, rows, "run-1", "0123456789012345678901234567890123456789"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, SummaryFile))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"acceptance summary: pass=1 blocked=1 fail=2 run=run-1 source=0123456789012345678901234567890123456789",
		"| portal-billing-return | fail |",
		"| member-ipv6 | blocked |",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("summary lacks %q:\n%s", want, text)
		}
	}
}

// SEC02 / AGENTS §4: a failed probe's diagnostic must not carry the session
// cookie values it was given, nor a host path.
func TestSEC02_ProbeFailureDetailRedactsCookiesAndPaths(t *testing.T) {
	root := filepath.FromSlash("/srv/users/lab/wt")
	home := filepath.FromSlash("/srv/users/lab")
	cookie := "s3ss10n-c00k1e-TEST-ONLY"
	stderr := []byte("\n\nbrowserType.launch: Executable doesn't exist at " + home + "/.cache/ms-playwright/chromium_headless_shell-1248/chrome-linux/headless_shell cookie=" + cookie + " probe=" + root + "/deploy/devstack/probe-portal.mjs\nsecond line " + cookie + "\n")
	got := ProbeFailureDetail(stderr, []string{"", cookie}, root, home)
	want := "browserType.launch: Executable doesn't exist at <home>/.cache/ms-playwright/chromium_headless_shell-1248/chrome-linux/headless_shell cookie=[redacted] probe=<worktree>/deploy/devstack/probe-portal.mjs"
	if got != want {
		t.Fatalf("detail\n got %q\nwant %q", got, want)
	}
	long := ProbeFailureDetail([]byte(strings.Repeat("é", 500)), nil, "", "")
	if len(long) > maxProbeDetail+len("…") || !strings.HasSuffix(long, "…") || !strings.HasPrefix(long, "é") {
		t.Fatalf("unbounded or split detail: %d bytes", len(long))
	}
	if ProbeFailureDetail(nil, []string{cookie}, root, home) != "" {
		t.Fatal("empty stderr must give an empty detail")
	}
}

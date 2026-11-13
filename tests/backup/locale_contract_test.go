package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Fixed to the reviewed 191473a0 content, not extracted from current scripts.
// Includes comments: every new locale mention needs an explicit review.
func TestAT30_BackupLocaleMentions(t *testing.T) {
	const comment = "# Enumerate ASCII characters: regex ranges depend on the caller's locale."
	allowed := map[string]map[string]string{
		"deploy/backup/backup.sh":       {comment: "Explains literal ASCII membership; changes no locale."},
		"deploy/backup/restore-pitr.sh": {comment: "Explains literal ASCII membership; changes no locale."},
		"deploy/backup/verify.sh":       {comment: "Explains literal ASCII membership; changes no locale."},
		"deploy/backup/postgres/fvpn-postgres-entrypoint.sh": {
			"\t\t" + comment:                       "Explains literal ASCII membership; changes no locale.",
			"\tif ! LC_ALL=C awk -v quote=\"'\" '": "Existing C locale for the recovery-file sanitizer's syntax.",
			"\t\t-c external_pid_file= -c logging_collector=off -c log_destination=stderr -c lc_messages=C": "Existing English diagnostics for the isolated recovery resource checks.",
			"\tcontrol=\"$(LC_ALL=C pg_controldata \"$data_dir\")\" || return":                              "Existing stable pg_controldata field names for recovery resource extraction.",
		},
	}
	mentions := regexp.MustCompile(`LANG|LC_[[:alnum:]_]*|lc_[[:alnum:]_]*|\benv[ \t]+-i\b|(?i:locale)`)
	for path, lines := range allowed {
		content, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		seen := make(map[string]int)
		for i, line := range strings.Split(string(content), "\n") {
			if !mentions.MatchString(line) {
				continue
			}
			if _, ok := lines[line]; !ok {
				t.Errorf("%s:%d: unapproved locale mention: %s", path, i+1, line)
			} else {
				seen[line]++
				if seen[line] > 1 {
					t.Errorf("%s:%d: duplicated locale allowlist entry: %s", path, i+1, line)
				}
			}
		}
		for line, reason := range lines {
			if seen[line] == 0 {
				t.Errorf("%s: missing fixed locale allowlist entry %q (%s)", path, line, reason)
			}
		}
	}
}

func TestAT30_LocaleSnapshot(t *testing.T) {
	// Include a future/unknown LC_* name and distinguish empty from unset.
	t.Setenv("LANGUAGE", "C")
	t.Setenv("LC_TIME", "C")
	t.Setenv("LC_FVPN_TEST", "TEST ONLY")
	cmd := exec.Command("bash", "-c", `source testdata/locale-snapshot.sh; snapshot_locale; unset LANGUAGE LC_FVPN_TEST; LC_FVPN_EMPTY=; snapshot_locale`)
	out, err := cmd.CombinedOutput()
	if err != nil || localeWarning(out) {
		t.Fatalf("snapshot: %v %s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `LANGUAGE=C;`) || !strings.Contains(lines[0], `LC_TIME=C;`) || !strings.Contains(lines[0], `LC_FVPN_TEST=TEST\ ONLY;`) ||
		strings.Contains(lines[1], "LANGUAGE=") || strings.Contains(lines[1], "LC_FVPN_TEST=") || !strings.Contains(lines[1], "LC_FVPN_EMPTY='';") {
		t.Fatalf("snapshot omitted/changed locale variables: %s", out)
	}
}

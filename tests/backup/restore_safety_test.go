package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAT30_RestoreRefusesResolvedPrimaryDescendants(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	primary := filepath.Join(root, "primary")
	if err := os.Mkdir(primary, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(primary, alias); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, target, guard string }{
		{"child", filepath.Join(primary, "child"), "--primary-dir"},
		{"symlink", filepath.Join(alias, "child"), "--primary-dir"},
		{"pgdata", filepath.Join(alias, "child"), "PGDATA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"../../deploy/backup/restore-pitr.sh", "--latest", "--target-dir", tc.target}
			if tc.guard != "PGDATA" {
				args = append(args, tc.guard, primary)
			}
			cmd := exec.Command("bash", args...)
			cmd.Env = append(os.Environ(), "PGDATA="+primary)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "refusing") {
				t.Fatalf("guard failed: %v %s", err, out)
			}
			if _, err := os.Stat(tc.target); !os.IsNotExist(err) {
				t.Fatalf("guard created child: %v", err)
			}
		})
	}
}

func TestAT30_VerifyRefusesUnsafePrefixAndMissingPrimary(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"--verify-prefix", "unowned", "--primary-container", "unused", "--repo-volume", "unused"},
		{"--repo-volume", "unused"},
	} {
		cmd := exec.Command("bash", append([]string{"../../deploy/backup/verify.sh"}, args...)...)
		cmd.Env = append(os.Environ(), "FVPN_PRIMARY_CONTAINER=")
		out, err := cmd.CombinedOutput()
		if err == nil || strings.Contains(string(out), "status=PASS") {
			t.Fatalf("unsafe verification: %v %s", err, out)
		}
		if !strings.Contains(string(out), "invalid --verify-prefix") && !strings.Contains(string(out), "archive state unknown") {
			t.Fatalf("unexpected refusal: %s", out)
		}
	}
}

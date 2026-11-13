package backup

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// The image-backed drill requires all three locales. Unit tests need only C,
// which is available without installing a locale on the developer/CI host.
func TestAT30_BackupGuardsC(t *testing.T) {
	scripts, err := filepath.Abs("../../deploy/backup")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "testdata/locale-guards.sh", scripts, t.TempDir(), "C", "lang")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C guard matrix: %v\n%s", err, out)
	}
	t.Logf("%s", out)
}

//go:build e2e

package e2e

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestISO01CompleteCaptureExport(t *testing.T) {
	input := strings.Repeat("TEST ONLY capture row\n", 200000)
	dest := filepath.Join(t.TempDir(), "capture.jsonl")
	cmd := exec.Command("cat")
	cmd.Stdin = strings.NewReader(input)
	out, err := exportCommand(cmd, dest)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dest)
	if err != nil || string(raw) != input || out.Bytes != fmt.Sprint(len(input)) || out.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("large capture differs")
	}
	cmd = exec.Command("sh", "-c", "printf partial; exit 1")
	if _, err := exportCommand(cmd, dest); err == nil {
		t.Fatal("failed export accepted")
	}
	raw, _ = os.ReadFile(dest)
	if string(raw) != input {
		t.Fatal("failed command replaced complete capture")
	}
	var diagnostic diagnosticBuffer
	cmd = exec.Command("cat")
	cmd.Stdin, cmd.Stdout = strings.NewReader(input), &diagnostic
	if err = cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !diagnostic.overflow || diagnostic.Len() != 1<<20 {
		t.Fatal("diagnostic overflow not explicit")
	}
}

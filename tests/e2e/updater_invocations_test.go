package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestD158_TwoUpdaterInvocationOutputsSurvive(t *testing.T) {
	root := t.TempDir()
	script := "../../deploy/devstack/updater-invocation.sh"
	if _, err := os.Stat(script); os.IsNotExist(err) {
		t.Skip("launcher fixture absent in isolated lint copy")
	}
	outputs := []string{"first stdout\n{\"msg\":\"update lock wait\",\"acquired\":true}\nfirst stderr\n", "second stdout\nsecond stderr without newline"}
	for i, output := range outputs {
		// Exercise the recorder with both a success and a failure, and mixed streams.
		code := "0"
		if i == 1 {
			code = "3"
		}
		cmd := exec.Command("sh", script, root, "sh", "-c", `printf '%s' "$1"; printf '%s' "$2" >&2; exit "$3"`, "test", output[:7], output[7:], code)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("recorder: %v %s", err, b)
		}
	}
	archive, err := exec.Command("tar", "-C", root, "-cf", "-", "updater-invocations").Output()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ReadUpdaterInvocations(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want both invocations, got %d", len(rows))
	}
	for i, row := range rows {
		want := 0
		if i == 1 {
			want = 3
		}
		start, e1 := time.Parse(time.RFC3339Nano, row.StartedAt)
		end, e2 := time.Parse(time.RFC3339Nano, row.EndedAt)
		if row.Output != outputs[i] || row.ExitCode == nil || *row.ExitCode != want || row.PID <= 0 || e1 != nil || e2 != nil || end.Before(start) {
			t.Fatalf("receipt %d: %+v", i, row)
		}
	}
	latest, err := os.ReadFile(filepath.Join(root, "updater.out"))
	if err != nil || string(latest) != outputs[1] {
		t.Fatal("latest compatibility output changed")
	}
	t.Log("two invocations retained byte-for-byte; exits 0/3 and start/end/PID receipts verified")
}

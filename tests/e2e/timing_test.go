package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestD160_AdmissionPrecedesSectionAndNeverRunsInside(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "deploy/devstack/admission.py")); os.IsNotExist(err) {
		t.Skip("admission fixture absent in isolated lint copy")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "events.jsonl")
	hold := filepath.Join(dir, "hold.sh")
	if err := os.WriteFile(hold, []byte("echo free\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FVPN_HOLD_SH", hold)
	t.Setenv("FVPN_E2E_ADMISSION_LOG", log)
	t.Setenv("FVPN_E2E_TIMED_SECTION", "")
	s := TimedSections{Root: root, Log: log}
	end, err := s.Begin(context.Background(), "lease-expiry")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hold, []byte("exit 99\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Admit(context.Background(), "illegal"); err == nil {
		t.Fatal("allowed admission inside section")
	}
	// The same wrapper entry point used by Docker returns the exact child output,
	// even after the holder changes. It must neither run the script nor wait.
	cmd := exec.Command("python3", "-B", "-c", `import importlib.util,sys
spec=importlib.util.spec_from_file_location('admission',sys.argv[1]);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
sys.argv=['docker','exact tool stream'];m.tool('docker',{'docker':'/bin/echo'})`, filepath.Join(root, "deploy/devstack/admission.py"))
	output, err := cmd.CombinedOutput()
	if err != nil || string(output) != "exact tool stream\n" {
		t.Fatalf("tool stream: %v %q", err, output)
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("FVPN_E2E_TIMED_SECTION") != "" {
		t.Fatal("section leaked")
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"admission", "section_start", "command_in_section", "section_end"}
	if len(lines) != len(want) {
		t.Fatalf("events: %s", data)
	}
	for i, line := range lines {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		if row["event"] != want[i] || row["at"] == nil {
			t.Fatalf("event %d: %s", i, line)
		}
	}
	t.Log("admission precedes start; held host inside section does not check/wait or contaminate tool output; end recorded")
}

func TestD160_HeldBeforeSectionWaitsBeforeStart(t *testing.T) {
	script := "../../deploy/devstack/admission.py"
	if _, err := os.Stat(script); os.IsNotExist(err) {
		t.Skip("admission fixture absent")
	}
	t.Setenv("FVPN_E2E_ADMISSION_LOG", filepath.Join(t.TempDir(), "events.jsonl"))
	cmd := exec.Command("python3", "-B", "-c", `import importlib.util,subprocess,sys
s=importlib.util.spec_from_file_location('admission',sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m)
answers=iter([subprocess.CompletedProcess([],1,'held',''),subprocess.CompletedProcess([],0,'free','')]);sleeps=[]
m.admit('lease-expiry',check=lambda:next(answers),sleep=sleeps.append)
assert sleeps==[60],sleeps`, script)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("admission wait: %v %s", err, b)
	}
	t.Log("held/free answers retained; exactly one 60-second recheck interval requested before section")
}

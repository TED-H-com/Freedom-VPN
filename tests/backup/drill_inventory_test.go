package backup

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAT30_DrillRequiredInventory(t *testing.T) {
	// Independently specify the contract so changing the checker's required list
	// cannot silently make omission of one of these executions pass this test.
	required := []string{
		"TestAT30_BackupPITRDrill", "TestAT30_IsolatedHostConfig",
		"TestAT30_IsolatedHostConfig/time", "TestAT30_IsolatedHostConfig/lsn", "TestAT30_IsolatedHostConfig/latest",
		"TestAT30_IsolatedHostConfig/direct_failure_cleanup", "TestAT30_IsolatedHostConfig/preexisting_volume",
		"TestAT30_BackupLocaleGuards", "TestAT30_BackupLocaleGuards/C/all", "TestAT30_BackupLocaleGuards/C.UTF-8/all",
		"TestAT30_BackupLocaleGuards/en_US.UTF-8/all", "TestAT30_BackupLocaleGuards/en_US.UTF-8/lang",
		"TestAT30_BackupLocaleGuards/en_US.UTF-8/collate", "TestAT30_BackupLocaleGuards/en_US.UTF-8/ctype",
	}
	type event struct{ Action, Package, Test string }
	for _, missing := range append([]string{""}, required...) {
		name := missing
		if name == "" {
			name = "complete"
		}
		t.Run(name, func(t *testing.T) {
			for _, scenario := range []string{"omitted", "skip", "fail", "no_run", "process_failed", "invalid_json", "build_failed"} {
				t.Run(scenario, func(t *testing.T) {
					var data strings.Builder
					write := func(action, test string) {
						line, err := json.Marshal(event{action, "github.com/TED-H-com/Freedom-VPN/tests/backup", test})
						if err != nil {
							t.Fatal(err)
						}
						data.Write(line)
						data.WriteByte('\n')
					}
					for _, test := range required {
						if test == missing && scenario == "omitted" {
							continue
						}
						if test != missing || scenario != "no_run" {
							write("run", test)
						}
						action := "pass"
						if test == missing && (scenario == "skip" || scenario == "fail") {
							action = scenario
						}
						write(action, test)
					}
					// Actual package events omit Test rather than supplying an empty name.
					data.WriteString("{\"Action\":\"pass\",\"Package\":\"github.com/TED-H-com/Freedom-VPN/tests/backup\"}\n")
					if scenario == "invalid_json" {
						data.WriteString("not json\n")
					}
					if scenario == "build_failed" {
						// No test is omitted and the producer exits zero: the action
						// itself must fail, independently of process/package status.
						data.WriteString("{\"Action\":\"build-fail\",\"ImportPath\":\"example.invalid/build\"}\n")
					}
					file := filepath.Join(t.TempDir(), "events")
					if err := os.WriteFile(file, []byte(data.String()), 0o600); err != nil {
						t.Fatal(err)
					}
					status := "0"
					if scenario == "process_failed" {
						status = "1"
					}
					cmd := exec.Command("python3", "check_drill.py", "--", "bash", "-c", `cat "$1"; exit "$2"`, "_", file, status)
					out, err := cmd.CombinedOutput()
					streamFailure := scenario == "process_failed" || scenario == "invalid_json" || scenario == "build_failed"
					wantFailure := missing != "" || streamFailure
					if (err != nil) != wantFailure {
						t.Fatalf("inventory failure=%v want=%v: %s", err, wantFailure, out)
					}
					if scenario == "build_failed" {
						if !strings.Contains(string(out), "backup-drill: build-fail event") {
							t.Fatalf("inventory failure did not name build-fail: %s", out)
						}
						t.Logf("complete inventory and producer exit 0 rejected: %s", out)
					}
					if wantFailure && !streamFailure && !strings.Contains(string(out), "required test did not run and pass: "+missing) {
						t.Fatalf("inventory failure did not name %s: %s", missing, out)
					}
				})
			}
		})
	}
}

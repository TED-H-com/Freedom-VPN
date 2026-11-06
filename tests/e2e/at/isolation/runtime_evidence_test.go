package isolation

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TEST ONLY source read; all process values are synthetic and never executed.
func readinessRead(t *testing.T, change func(map[string]any)) map[string]any {
	t.Helper()
	proc := map[string]any{"status": "available", "role": "fvpn-agent", "pid": 42, "start_ticks": "100"}
	snapshot := map[string]any{
		"read_start": runtimeNow(), "read_end": runtimeNow(), "process_coherent": true,
		"process_before": proc, "process_after": proc,
		"agent_pid":        RuntimeRead{Status: "available", Text: "42\n"},
		"agent_process_id": RuntimeRead{Status: "available", Text: "00000000-0000-4000-8000-000000000001\n"},
	}
	if change != nil {
		change(snapshot)
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{"status": "available", "phase": "after-clone-creation", "role": "source", "snapshot": json.RawMessage(raw)}
}

func TestENR02_RuntimeReadinessRetainsExactSuccessfulRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reads := []map[string]any{
			{"status": "unavailable", "error": "container read unavailable", "read_start": runtimeNow()},
			readinessRead(t, func(s map[string]any) { s["process_coherent"] = false }),
			readinessRead(t, nil),
		}
		calls := 0
		got, ready := WaitRuntimeProcess(t.Context(), func(ctx context.Context) map[string]any {
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("sampler has no deadline")
			}
			if calls == len(reads) {
				t.Fatal("refetched the ready snapshot")
			}
			row := reads[calls]
			calls++
			return row
		})
		if !ready || calls != 3 || !reflect.DeepEqual(got["snapshot"], reads[2]["snapshot"]) {
			t.Fatalf("did not select the actual ready read: %v calls=%d", got, calls)
		}
		receipt := got["readiness"].(runtimeReadiness)
		if receipt.Status != "ready" || receipt.BudgetSeconds != 20 || !reflect.DeepEqual(receipt.Attempts, reads[:2]) {
			t.Fatalf("original failures not retained: %+v", receipt)
		}
		for _, row := range reads {
			if _, ok := row["readiness"]; ok {
				t.Fatal("mutated an original sample")
			}
		}
	})
}

func TestENR02_RuntimeReadinessRequiresLiveLaunchIdentity(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"wrong-role": func(s map[string]any) { s["process_before"].(map[string]any)["role"] = "fvpn-watchdog" },
		"changed-process": func(s map[string]any) {
			s["process_after"] = map[string]any{"status": "available", "role": "fvpn-agent", "pid": 43, "start_ticks": "101"}
		},
		"unavailable-launch": func(s map[string]any) { s["agent_process_id"] = RuntimeRead{Status: "unavailable"} },
		"malformed-launch": func(s map[string]any) {
			s["agent_process_id"] = RuntimeRead{Status: "available", Text: "TEST ONLY malformed"}
		},
		"missing-launch":      func(s map[string]any) { delete(s, "agent_process_id") },
		"wrong-pid-file":      func(s map[string]any) { s["agent_pid"] = RuntimeRead{Status: "available", Text: "43"} },
		"missing-start-ticks": func(s map[string]any) { delete(s["process_before"].(map[string]any), "start_ticks") },
		"missing-read-bound":  func(s map[string]any) { delete(s, "read_start") },
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				first := readinessRead(t, change)
				calls := 0
				got, ready := WaitRuntimeProcess(t.Context(), func(context.Context) map[string]any {
					calls++
					if calls == 1 {
						return first
					}
					return readinessRead(t, nil)
				})
				if !ready || calls != 2 || !reflect.DeepEqual(got["readiness"].(runtimeReadiness).Attempts, []map[string]any{first}) {
					t.Fatalf("invalid process/launch accepted or lost: %v calls=%d", got, calls)
				}
			})
		})
	}
}

func TestENR02_RuntimeReadinessDeadlineAndCancellation(t *testing.T) {
	for _, mode := range []string{"deadline-polling", "deadline-blocked", "cancel-blocked", "already-canceled"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if mode == "already-canceled" {
					cancel()
				}
				if mode == "cancel-blocked" {
					time.AfterFunc(time.Second, cancel)
				}
				started := time.Now()
				var last map[string]any
				calls := 0
				got, ready := WaitRuntimeProcess(ctx, func(wait context.Context) map[string]any {
					calls++
					if mode != "deadline-polling" {
						<-wait.Done()
						// A blocking sampler can finish with apparently coherent data
						// after its deadline. It must stay diagnostic, not qualifying.
						last = readinessRead(t, nil)
					} else {
						last = readinessRead(t, func(s map[string]any) { s["process_coherent"] = false })
					}
					return last
				})
				want := 20 * time.Second
				if mode == "already-canceled" {
					want = 0
				} else if mode == "cancel-blocked" {
					want = time.Second
				}
				receipt := got["readiness"].(runtimeReadiness)
				if ready || got["status"] != "unavailable" || receipt.Status == "ready" || time.Since(started) != want || len(receipt.Attempts) != calls || !reflect.DeepEqual(receipt.Attempts[calls-1], last) {
					t.Fatalf("deadline/cancel lost evidence or accepted late read: %v elapsed=%s", got, time.Since(started))
				}
				if last["status"] != "available" {
					t.Fatal("failed envelope replaced the original observation")
				}
			})
		})
	}
}

func TestD141_RuntimeProjectionAndBoundedLogs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	if got := runtimeRead(path); got.Status != "unavailable" {
		t.Fatal(got)
	}
	if err := os.WriteFile(path, []byte("before shutdown\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before := runtimeRead(path)
	if err := os.WriteFile(path, []byte("new launch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	after := runtimeRead(path)
	if before.Status != "available" || before.Text != "before shutdown\n" || after.Text != "new launch\n" || before.SHA256 == after.SHA256 {
		t.Fatal("independent snapshots lost pre-stop bytes")
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", runtimeReadLimit+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if got := runtimeRead(path); got.Status != "truncated" || got.Bytes != runtimeReadLimit {
		t.Fatal(got.Status, got.Bytes)
	}
	cfg := RuntimeRead{Status: "available", Text: `{"control_url":"https://127.0.0.1:9444","observation_seconds":1,"heartbeat_seconds":1,"poll_seconds":1,"datapath":{"member_ingress":"fvmb0","uplink":"fvup0","snat":"198.51.100.23"},"private":"TEST ONLY excluded"}`}
	projected := projectRuntimeConfig("/etc/vpn-node/agent-run.json", cfg)
	raw, _ := json.Marshal(projected)
	if strings.Contains(string(raw), "excluded") || projected["status"] != "available" {
		t.Fatal(string(raw))
	}
	cfg.Text = `{"datapath":`
	if got := projectRuntimeConfig("/etc/vpn-node/agent-run.json", cfg); got["status"] != "parse error" {
		t.Fatal(got)
	}
	home := "/" + "home/" + "alice/secret"
	unapproved := projectRuntimeConfig(home+"/agent-run.json", cfg)
	if unapproved["status"] != "unapproved path" || strings.Contains(unapproved["path"].(string), "alice") {
		t.Fatal(unapproved)
	}
	if err := os.WriteFile(path, []byte("path="+home+" setup_key=nbsecret \"token\":\"jwtsecret\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	redacted := runtimeRead(path)
	if strings.Contains(redacted.Text, "alice") || strings.Contains(redacted.Text, "nbsecret") || strings.Contains(redacted.Text, "jwtsecret") {
		t.Fatal(redacted.Text)
	}
}

func TestD141_RuntimeArgumentsAreAllowlisted(t *testing.T) {
	for _, tc := range []struct{ args, role string }{
		{"/usr/local/bin/fvpn-agent\x00run\x00-config\x00/etc/vpn-node/agent-run.json\x00", "fvpn-agent"},
		{"/usr/local/bin/fvpn-watchdog\x00run\x00-config\x00/etc/vpn-node/agent-run.json\x00", "fvpn-watchdog"},
		{"/usr/local/bin/fvpn-e2e-nodeconfig\x00resolver-proxy\x00", "resolver-proxy"},
		{"/usr/local/bin/fvpn-agent\x00enroll\x00-token\x00TEST ONLY excluded\x00", ""},
		{"/usr/local/bin/fvpn-agent\x00run\x00-config\x00/unapproved\x00", ""},
		{"/usr/local/bin/fvpn-agent\x00run\x00-config\x00/etc/vpn-node/agent-run.json\x00-extra\x00", ""},
	} {
		role, _, _ := approvedProcess([]byte(tc.args))
		if role != tc.role {
			t.Fatalf("role %q expected %q", role, tc.role)
		}
	}
}

func TestD141_CloneTransportAndStopReview(t *testing.T) {
	cmd := exec.Command("python3", "-B", "-m", "unittest", "-v", "clone_evidence_test.py")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatal(err)
	}
}

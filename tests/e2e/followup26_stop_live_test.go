//go:build e2e

package e2e

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait"
	"github.com/google/uuid"
)

// docker cp works after PID1 exits; docker exec cannot recover that receipt.
func stoppedFile26(ctx context.Context, container, path string) ([]byte, error) {
	raw, err := exec.CommandContext(ctx, "docker", "cp", container+":"+path, "-").Output()
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	h, err := tr.Next()
	if err != nil {
		return nil, err
	}
	if h.Size > 65536 {
		return nil, fmt.Errorf("receipt too large")
	}
	return io.ReadAll(tr)
}

func TestE2E_ContainerStop26(t *testing.T) {
	if os.Getenv("FVPN_E2E_STOP26") != "1" {
		t.Skip("set FVPN_E2E_STOP26=1 on the enrolled owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	head, err := exec.CommandContext(t.Context(), "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs/2026-11-01/followup26", id)}
	defer r.finish()
	ctx := t.Context()
	s := &Stack{lab: l}
	for _, mode := range []string{"graceful", "forced"} {
		rec := Record{ID: "container-stop-" + mode, Title: "TEST ONLY PID1 forwarding and 15 second agent stop budget", Status: StatusFail}
		func() {
			defer func() { r.record(rec) }()
			if exec.CommandContext(ctx, "docker", "start", l.node).Run() != nil {
				return
			}
			if s.Service(ctx, "start", "/usr/local") != nil {
				return
			}
			processID, e := l.docker(l.node, "cat", "/var/lib/vpn-node/e2e/agent-process-id")
			if e != nil {
				return
			}
			pid, e := l.docker(l.node, "cat", "/var/lib/vpn-node/e2e/agent.pid")
			if e != nil {
				return
			}
			proc, e := l.docker(l.node, "cat", "/proc/"+strings.TrimSpace(string(pid))+"/stat")
			if e != nil {
				return
			}
			fields := strings.Fields(string(proc)[strings.LastIndexByte(string(proc), ')')+1:])
			if len(fields) < 20 {
				return
			}
			startTicks := fields[19]
			if mode == "forced" {
				if _, e = l.docker(l.node, "kill", "-STOP", strings.TrimSpace(string(pid))); e != nil {
					return
				}
			}
			start := time.Now().UTC()
			noteJSON(&rec, "stop_started_at", start)
			// The configured Docker grace is intentionally used, with no -t override.
			e = exec.CommandContext(ctx, "docker", "stop", l.node).Run()
			elapsed := time.Since(start)
			noteJSON(&rec, "stop_duration_ms", elapsed.Milliseconds())
			state, se := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{json .State}}`, l.node).Output()
			var st struct {
				Running    bool
				ExitCode   int
				OOMKilled  bool
				StartedAt  string
				FinishedAt string
			}
			if se != nil || json.Unmarshal(state, &st) != nil {
				return
			}
			noteJSON(&rec, "container", st)
			receipt, re := stoppedFile26(ctx, l.node, "/var/lib/vpn-node/e2e/agent-exits/"+strings.TrimSpace(string(processID))+".json")
			var result agentwait.Result
			if re != nil || json.Unmarshal(receipt, &result) != nil {
				return
			}
			noteJSON(&rec, "agent_wait", result)
			wantPID, _ := strconv.Atoi(strings.TrimSpace(string(pid)))
			ok := e == nil && !st.Running && st.ExitCode == 0 && !st.OOMKilled && result.ID == strings.TrimSpace(string(processID)) && result.PID == wantPID && result.StartTicks == startTicks && !result.WaitObservedAt.Before(start)
			if mode == "graceful" {
				ok = ok && result.ExitCode != nil && *result.ExitCode == 0 && result.Signal == nil
			} else {
				raw, ke := stoppedFile26(ctx, l.node, "/var/lib/vpn-node/e2e/agent-exits/"+result.ID+".sigkill.json")
				var kill struct {
					PID       int       `json:"pid"`
					Signal    int       `json:"signal"`
					Accepted  bool      `json:"kill_accepted"`
					Waited    int64     `json:"waited_ms"`
					Requested time.Time `json:"requested_at"`
					Term      time.Time `json:"term_requested_at"`
				}
				if ke != nil || json.Unmarshal(raw, &kill) != nil {
					return
				}
				noteJSON(&rec, "supervisor_sigkill", kill)
				ok = ok && result.ExitCode == nil && result.Signal != nil && *result.Signal == 9 && kill.Accepted && kill.PID == result.PID && kill.Waited >= 15000 && kill.Waited < 20000 && elapsed < 25*time.Second
			}
			if ok {
				rec.Status = StatusPass
			}
		}()
	}
	// Leave the final stopped container for the normal e2e-down inventory cleanup.
}

//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait"
	"github.com/google/uuid"
)

func TestE2E_Followup25Native(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP25") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP25=1 with a fresh owned stack")
	}
	family := os.Getenv("FVPN_E2E_FOLLOWUP25_FAMILY")
	if family != "ipv4" && family != "ipv6" {
		t.Fatal("explicit ipv4 or ipv6 required")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	arm := os.Getenv("FVPN_E2E_FOLLOWUP25_ARM")
	if arm != "kernel" && arm != "userspace" {
		t.Fatal("explicit kernel or userspace arm required")
	}
	if arm == "userspace" {
		t.Cleanup(func() {
			clean, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := exec.CommandContext(clean, "docker", "exec", l.node, "rm", "-f", "/var/lib/vpn-node/e2e/test-only-userspace-wireguard").Run(); err != nil {
				t.Error("remove diagnostic marker:", err)
			}
		})
		// Entry point has not enrolled or launched its daemon on this fresh stack.
		if err := exec.CommandContext(t.Context(), "docker", "exec", l.node, "touch", "/var/lib/vpn-node/e2e/test-only-userspace-wireguard").Run(); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FVPN_E2E_FOLLOWUP24", "1")
	t.Setenv("FVPN_E2E_FOLLOWUP24_FAMILY", family)
	TestE2E_Followup24Native(t)
}

func TestE2E_AgentExitReadback(t *testing.T) {
	if os.Getenv("FVPN_E2E_EXIT25") != "1" {
		t.Skip("set FVPN_E2E_EXIT25=1 after an enrolled first-selection attempt")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx := t.Context()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup25", id)}
	defer r.finish()
	s := &Stack{lab: l}
	defer func() {
		if err := s.Service(context.WithoutCancel(ctx), "start", "/usr/local"); err != nil {
			t.Error(err)
		}
	}()
	for _, mode := range []string{"graceful", "sigkill"} {
		rec := Record{ID: "agent-exit-" + mode, Title: "TEST ONLY persistent parent wait status", Spec: []string{"AT23", "UPD05"}, Status: StatusFail}
		func() {
			defer func() { r.record(rec) }()
			if err := s.Service(ctx, "start", "/usr/local"); err != nil {
				t.Error(err)
				return
			}
			processID, err := l.docker(l.node, "cat", "/var/lib/vpn-node/e2e/agent-process-id")
			if err != nil {
				t.Error(err)
				return
			}
			pid, err := l.docker(l.node, "cat", "/var/lib/vpn-node/e2e/agent.pid")
			if err != nil {
				t.Error(err)
				return
			}
			if mode == "sigkill" {
				if _, err = l.docker(l.node, "sh", "-c", `touch /var/lib/vpn-node/e2e/pause-agent; kill -KILL "$1"`, "kill-agent", strings.TrimSpace(string(pid))); err != nil {
					t.Error(err)
					return
				}
			}
			start := time.Now()
			if err := s.Service(ctx, "stop", ""); err != nil {
				t.Error(err)
				return
			}
			noteJSON(&rec, "stop_duration_ms", time.Since(start).Milliseconds())
			path := "/var/lib/vpn-node/e2e/agent-exits/" + strings.TrimSpace(string(processID)) + ".json"
			raw, err := l.docker(l.node, "cat", path)
			if err != nil {
				t.Error(err)
				return
			}
			var result agentwait.Result
			if json.Unmarshal(raw, &result) != nil {
				t.Error("malformed wait receipt")
				return
			}
			noteJSON(&rec, "wait", result)
			if result.ID != strings.TrimSpace(string(processID)) || result.StartTicks == "" || result.WaitObservedAt.IsZero() {
				t.Error("receipt identity/time mismatch")
				return
			}
			if mode == "graceful" && (result.ExitCode == nil || *result.ExitCode != 0 || result.Signal != nil) {
				t.Error("agent did not exit zero")
				return
			}
			if mode == "sigkill" && (result.ExitCode != nil || result.Signal == nil || *result.Signal != 9) {
				t.Error("agent did not die by SIGKILL")
				return
			}
			if _, err = l.docker(l.node, "test", "-e", "/proc/"+strings.TrimSpace(string(pid))); err == nil {
				t.Error("agent not reaped")
				return
			}
			if err := s.Service(ctx, "stop", ""); err != nil {
				t.Error(err)
				return
			}
			again, err := l.docker(l.node, "cat", path)
			if err != nil || string(again) != string(raw) {
				t.Error("repeat stop replaced receipt")
				return
			}
			rec.Status = StatusPass
		}()
	}
}

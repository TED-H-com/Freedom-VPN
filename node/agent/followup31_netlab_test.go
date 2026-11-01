//go:build netlab

package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// TEST ONLY: real Engine/TLS control/nft/tc, forty ordinary desired-first
// changes. No injected delay/failure, forced renewal or deadline adjustment.
// Member 2 stays admitted; member 1 alternates revoked/admitted. Packet loss
// is evidence to retain, never a tolerance or a completed-guard exemption.
func followup31Generations(t *testing.T, r *rig) {
	t.Helper()
	dir := os.Getenv("FVPN_AGENT_EVIDENCE_DIR")
	run := func(args ...string) {
		t.Helper()
		if raw, err := exec.CommandContext(t.Context(), args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("lab command: %v %s", err, raw)
		}
	}
	write := func(name string, value any) {
		t.Helper()
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("ip", "netns", "exec", "fv-inet", "ip", "addr", "add", "203.0.113.10/32", "dev", "fv-ia")
	run("ip", "route", "add", "203.0.113.10/32", "via", "192.0.2.38", "dev", "fv-ih")
	run("sysctl", "-qw", "net.ipv4.ip_forward=1")
	for _, dev := range []string{"all", "default", "fv-fh", "fv-fa", "fv-ih"} {
		run("sysctl", "-qw", "net.ipv4.conf."+dev+".rp_filter=0")
	}
	r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24"}
	r.srv.SetServiceFor(time.Hour)
	r.srv.SetMaxOfflineSeconds(3600)
	r.eng.cfg.RenewLeadSeconds = 1
	bindings := threeBindings(r.eng.clock.Wall())[:2]
	r.srv.SetFixture(fixtureWithBindings(12, bindings))
	stepOK(t, r)
	if !r.eng.serving || len(r.eng.appliedModel.Bindings) != 2 {
		t.Fatal("two-source setup failed", r.eng.decision.Errors)
	}
	// Every source is sent once per 10 ms. This is a header-only synthetic
	// lab stream; timestamps are userspace observations, not kernel verdicts.
	script := strings.ReplaceAll(followup29Continuous, "target=now+20000000", "target=now+5000000")
	captureCtx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 150*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(captureCtx, "python3", "-u", "-c", script, filepath.Join(dir, "capture-stop.json"))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	reader := bufio.NewReader(stdout)
	if line, err := reader.ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatal("capture startup", line, err)
	}
	t.Cleanup(func() {
		write("capture-stop.json", true)
		var trace json.RawMessage
		if err := json.NewDecoder(reader).Decode(&trace); err != nil {
			t.Error("capture incomplete", err, stderr.String())
			return
		}
		write("continuous.json", trace)
		if err := cmd.Wait(); err != nil {
			t.Error(err)
		}
	})
	type transition struct {
		Generation          decstr.U64    `json:"generation"`
		OldLeaseGeneration  decstr.U64    `json:"old_lease_generation"`
		Start               time.Duration `json:"start_ns,string"`
		End                 time.Duration `json:"end_ns,string"`
		SequenceBefore      decstr.U64    `json:"sequence_before"`
		SequenceAfter       decstr.U64    `json:"sequence_after"`
		Serving             bool          `json:"serving"`
		FirstMemberAdmitted bool          `json:"first_member_admitted"`
	}
	var transitions []transition
	t.Cleanup(func() { write("transitions.json", transitions) })
	time.Sleep(200 * time.Millisecond)
	for i := range 40 {
		gen := uint64(13 + i)
		members := bindings
		if i%2 == 0 {
			members = bindings[1:]
		}
		row := transition{Generation: decstr.U64(gen), OldLeaseGeneration: r.eng.lease.ConfigGeneration,
			Start: r.eng.clock.Mono(), SequenceBefore: decstr.U64(r.eng.measurementSequence), FirstMemberAdmitted: i%2 != 0}
		r.srv.SetFixture(fixtureWithBindings(gen, members))
		stepOK(t, r)
		row.End, row.SequenceAfter, row.Serving = r.eng.clock.Mono(), decstr.U64(r.eng.measurementSequence), r.eng.serving
		transitions = append(transitions, row)
		if !r.eng.serving || r.eng.appliedGen != gen || uint64(r.eng.lease.ConfigGeneration) != gen {
			t.Fatal("generation did not recover", gen, r.eng.decision.Errors)
		}
		// Keep a distinct observed forwarding interval between transitions.
		time.Sleep(100 * time.Millisecond)
	}
	t.Log("D136: forty desired-first generation changes completed; all packet losses retained")
}

//go:build netlab

package agent

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/tests/netlab"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/google/uuid"
)

type measuredSystemPlane struct {
	SystemPlane
	reads   []time.Duration
	paths   []time.Duration
	applies []enforce.ApplyResult
}

func (p *measuredSystemPlane) Apply(ctx context.Context, model enforce.Model, counters map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.SystemPlane.Apply(ctx, model, counters)
	p.applies = append(p.applies, res)
	return res, err
}

func (p *measuredSystemPlane) VerifyServing(ctx context.Context, model enforce.Model, assignments map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	start := time.Now()
	ev, err := p.SystemPlane.VerifyServing(ctx, model, assignments)
	p.reads = append(p.reads, time.Since(start))
	return ev, err
}

func (p *measuredSystemPlane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	start := time.Now()
	path, err := p.SystemPlane.MemberPath(ctx, m, expected)
	p.paths = append(p.paths, time.Since(start))
	return path, err
}

// Invoked only by the foreground container launcher in tests/netlab/agent.
// The engine and TLS fake control run in this process; nft/tc are real. No
// traffic generator or external endpoint is used for the latency measurement.
func TestD085_LiveReadNetlab(t *testing.T) {
	if os.Getenv("FVPN_AGENT_LIVE_NETLAB") != "1" {
		t.Skip("requires tests/netlab/agent/run-live.sh foreground privileged container")
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		t.Fatal("refusing to change networking outside the lab container")
	}
	run := func(name string, args ...string) {
		t.Helper()
		if out, err := exec.CommandContext(t.Context(), name, args...).CombinedOutput(); err != nil {
			t.Fatalf("lab %s: %v %s", name, err, out)
		}
	}
	for _, args := range [][]string{{"uname", "-r"}, {"nft", "--version"}, {"tc", "-V"}} {
		out, err := exec.CommandContext(t.Context(), args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("D104 platform %s: %s", args[0], strings.TrimSpace(string(out)))
	}
	for _, pair := range [][2]string{{"fv-fh", "fv-fa"}, {"fv-ih", "fv-ia"}} {
		run("ip", "link", "add", pair[0], "type", "veth", "peer", "name", pair[1])
		for _, iface := range pair {
			run("ip", "link", "set", iface, "up")
		}
	}
	// The lab topology owns these addresses and routes, exactly as the
	// netlab D-090 recovery helper does (documentation IPv6 prefixes here).
	run("ip", "addr", "add", "192.0.2.33/30", "dev", "fv-fh")
	run("ip", "addr", "add", "192.0.2.34/30", "dev", "fv-fa")
	run("ip", "addr", "add", "192.0.2.37/30", "dev", "fv-ih")
	run("ip", "netns", "add", "fv-inet")
	run("ip", "link", "set", "fv-ia", "netns", "fv-inet")
	run("ip", "netns", "exec", "fv-inet", "ip", "addr", "add", "192.0.2.38/30", "dev", "fv-ia")
	run("ip", "netns", "exec", "fv-inet", "ip", "link", "set", "fv-ia", "up")
	reconcile := func() {
		run("ip", "route", "replace", "100.64.0.0/10", "via", "192.0.2.34", "dev", "fv-fh")
		run("ip", "-6", "addr", "replace", "2001:db8:ff::33/64", "dev", "fv-fh", "nodad")
		run("ip", "-6", "route", "replace", "2001:db8:a1::/64", "via", "2001:db8:ff::34", "dev", "fv-fh")
	}
	reconcile()
	control := exec.CommandContext(t.Context(), "ip", "netns", "exec", "fv-inet", "python3", "-u", "-c", `import socket
s=socket.socket(); s.bind(('192.0.2.38',40080)); s.listen(); print('ready',flush=True)
while True:
 c,_=s.accept(); c.sendall(b'control-ok'); c.close()
`)
	ready, err := control.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := control.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = control.Process.Kill(); _ = control.Wait() })
	if line, err := bufio.NewReader(ready).ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatal("control listener", line, err)
	}
	r := newRig(t)
	r.eng.cfg.BootIDFile = "/proc/sys/kernel/random/boot_id"
	bootRaw, err := os.ReadFile(r.eng.cfg.BootIDFile)
	if err != nil {
		t.Fatal(err)
	}
	boot, err := uuid.Parse(strings.TrimSpace(string(bootRaw)))
	if err != nil {
		t.Fatal(err)
	}
	r.eng.bootID = boot
	r.eng.meterSource = staticSource{boot: boot.String()}
	r.eng.clock = timeanchor.System{}
	r.srv.SetClock(r.eng.clock.Wall)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	p := &measuredSystemPlane{}
	p.Executor = enforce.System{}
	r.eng.cfg.TestMode = true
	evidenceDir := os.Getenv("FVPN_AGENT_EVIDENCE_DIR")
	if evidenceDir == "" {
		evidenceDir = t.TempDir()
	}
	r.eng.cfg.TestTimingPath = filepath.Join(evidenceDir, "operations.jsonl")
	r.eng.plane = p
	r.eng.memberPath = nil // the initial fake plane has no kernel topology
	r.eng.cfg.HeartbeatSeconds, r.eng.cfg.PollSeconds = 1, 1
	r.eng.withdraw(t.Context())
	stepOK(t, r)
	if !r.eng.serving {
		t.Fatalf("real Apply failed: %+v %s", r.eng.decision.Errors, r.log.String())
	}
	p.reads = nil
	p.paths = nil
	const steps = 30
	started := time.Now()
	for range steps {
		stepOK(t, r)
		if !mustBeat(t, r).Serving {
			t.Fatalf("live read failed: %+v", r.eng.decision.Errors)
		}
	}
	stepMean := time.Since(started) / steps
	if len(p.reads) != 3*steps {
		t.Fatalf("live reads=%d want=%d", len(p.reads), 3*steps)
	}
	var total time.Duration
	for _, d := range p.reads {
		total += d
	}
	slices.Sort(p.reads)
	t.Logf("real nft/tc: steps=%d reads=%d mean_read=%s p50=%s p95=%s max=%s added_live_reads_per_step=%s total_step_mean=%s timeout=%s",
		steps, len(p.reads), total/time.Duration(len(p.reads)), p.reads[len(p.reads)/2], p.reads[len(p.reads)*95/100], p.reads[len(p.reads)-1], total/steps, stepMean, r.eng.servingReadTimeout())

	var pathTotal time.Duration
	for _, d := range p.paths {
		pathTotal += d
	}
	t.Logf("D096 topology reads: count=%d mean=%s added_per_step=%s; follow-up 21 live-read reference=52.6ms; combined_verification_per_step=%s", len(p.paths), pathTotal/time.Duration(len(p.paths)), pathTotal/steps, (pathTotal+total)/steps)

	run("tc", "qdisc", "add", "dev", "fv-fh", "clsact")
	run("tc", "filter", "add", "dev", "fv-fh", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
	before := liveBypassPackets(t)
	if before != 1 {
		t.Fatalf("injection did not exercise bypass: packets=%d", before)
	}
	started = time.Now()
	stepOK(t, r)
	t.Logf("D090 injected next-step withdrawal cost=%s", time.Since(started))
	assertLiveWithdrawn(t, r, 2, "TC_FOREIGN_FILTER")
	assertTCWire(t, r)
	q := r.eng.containment.Quarantine
	if q == nil || !q.Quarantined || !q.Recorded || !q.Baseline.Installed || !q.Baseline.Verified || len(q.Links) != 1 || q.Links[0].After.AdminUp {
		t.Fatalf("quarantine evidence: %+v", q)
	}
	after := liveBypassPackets(t)
	if after != 0 {
		t.Fatalf("quarantine bypass packets=%d", after)
	}
	c, err := net.DialTimeout("tcp", "192.0.2.38:40080", time.Second)
	if err != nil {
		t.Fatal("uplink control", err)
	}
	_ = c.SetReadDeadline(time.Now().Add(time.Second))
	b := make([]byte, 32)
	n, err := c.Read(b)
	_ = c.Close()
	if err != nil || string(b[:n]) != "control-ok" {
		t.Fatal("control response", err)
	}
	t.Logf("D090 deciding observation: pre-quarantine uplink_packets=%d nft_delta=0; post-quarantine uplink_packets=%d nft_delta=0; node uplink control=pass TLS observation=accepted; evidence=%s", before, after, r.eng.containmentJSON())
	// A daemon may raise the link between steps. The next step contains it.
	run("ip", "link", "set", "fv-fh", "up")
	stepOK(t, r)
	if mustBeat(t, r).Serving || !r.eng.containment.Quarantine.Quarantined {
		t.Fatal("repeat containment failed")
	}
	// Only this harness removes its foreign state. Apply raises the member
	// link; D-096 must reject the missing routes/IPv6 address before serving.
	run("tc", "filter", "del", "dev", "fv-fh", "ingress", "pref", "1")
	run("tc", "qdisc", "del", "dev", "fv-fh", "clsact")
	stepOK(t, r)
	if mustBeat(t, r).Serving || mustBeat(t, r).Reason != "member_path_unconfigured" {
		t.Fatalf("premature recovery: %+v", mustBeat(t, r))
	}
	reconcile()
	stepOK(t, r)
	if !mustBeat(t, r).Serving || r.eng.applyError != nil {
		t.Fatalf("real recovery failed: %+v", r.eng.decision.Errors)
	}
	t.Log("D096 clean Apply raised link but stayed non-serving until harness restored two routes and one IPv6 address; next full Apply + topology check restored serving")
	followup25Restart(t, r, p, reconcile)
	followup23LiveProof(t, r)
}

// Same ingress injector as the netlab deciding observation. tcpdump readiness
// is explicit; the capture is a supervised child and all traffic is in-lab.
func liveBypassPackets(t *testing.T) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tcpdump", "-l", "-n", "-i", "fv-ih", "-c", "1", "udp port 39997")
	var out bytes.Buffer
	cmd.Stdout = &out
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewScanner(stderr)
	ready := false
	for reader.Scan() {
		if strings.Contains(reader.Text(), "listening on") {
			ready = true
			break
		}
	}
	if !ready {
		_ = cmd.Wait()
		t.Fatal("capture not ready")
	}
	obs, injectErr := netlab.InjectMemberIngress(t.Context(), netlab.IngressSpec{MemberIface: "fv-fh", Source: netip.MustParseAddr("100.64.9.9"), Dest: netip.MustParseAddr("203.0.113.10"), DestPort: 39997})
	err = cmd.Wait()
	if injectErr != nil {
		t.Fatal(injectErr)
	}
	if obs.After["ctunku"].Packets != obs.Before["ctunku"].Packets {
		t.Fatalf("injection reached nft: %+v", obs)
	}
	if out.Len() == 0 {
		if ctx.Err() == nil {
			t.Fatalf("capture failed: %v", err)
		}
		return 0
	}
	if err != nil {
		t.Fatal(fmt.Errorf("capture: %w", err))
	}
	return 1
}

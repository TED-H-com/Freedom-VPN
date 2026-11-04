//go:build e2e && integration

package secops

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

type nodeLease16 struct {
	Signed     json.RawMessage `json:"signed"`
	Sent       int64           `json:"request_monotonic_ns,string"`
	Deadline   int64           `json:"deadline_monotonic_ns,string"`
	ID         string          `json:"lease_id"`
	Generation string          `json:"generation"`
	Issued     time.Time       `json:"issued_at"`
	Expires    time.Time       `json:"expires_at"`
	Max        int             `json:"max_offline_seconds"`
	Members    []struct {
		ID  string    `json:"member_id"`
		End time.Time `json:"lease_end"`
	} `json:"members"`
}

func mono16() int64 {
	var ts unix.Timespec
	if unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts) != nil {
		return 0
	}
	return ts.Nano()
}

func (p *liveProbe) nodeLease16() (nodeLease16, error) {
	var out nodeLease16
	// Read inside the owned node: its private store is root-owned. A single
	// read-only SQLite query observes a consistent DB/WAL snapshot without
	// separately copying a moving WAL or changing the source's permissions.
	raw, err := p.docker("node", "/var/lib/vpn-node/e2e/secops-leaseprobe")
	if err != nil {
		return out, err
	}
	var env struct {
		Signed json.RawMessage `json:"signed"`
		Sent   int64           `json:"request_monotonic_ns,string"`
	}
	if err = json.Unmarshal(raw, &env); err != nil {
		return out, err
	}
	out.Sent = env.Sent
	var doc struct {
		ID         string    `json:"lease_id"`
		Generation string    `json:"config_generation"`
		Issued     time.Time `json:"issued_at"`
		Expires    time.Time `json:"expires_at"`
		Max        int       `json:"max_offline_seconds"`
		Members    []struct {
			ID  string    `json:"member_id"`
			End time.Time `json:"lease_end"`
		} `json:"members"`
	}
	if err = json.Unmarshal(env.Signed, &doc); err != nil {
		return out, err
	}
	out.Signed = env.Signed
	out.ID = doc.ID
	out.Generation = doc.Generation
	out.Issued = doc.Issued
	out.Expires = doc.Expires
	out.Max = doc.Max
	out.Members = doc.Members
	out.Deadline = out.Sent + int64(doc.Expires.Sub(doc.Issued))
	return out, nil
}

func (p *liveProbe) continuitySample16(transport *labstack.Stack, device uuid.UUID, start time.Time) continuitySample16 {
	s := continuitySample16{ReadStart: time.Since(start).Seconds()}
	s.Transport = sampleTransport11(p.t.Context(), transport, device, start)
	// Fields only: no command line, environment, credential or other process.
	raw, err := p.docker("node", "sh", "-c", `cat /run/vpn-node/heartbeat; p=$(cat /var/lib/vpn-node/e2e/agent.pid) && awk '{print "agent_identity " $1 " " $22; print "agent_state " $3}' /proc/"$p"/stat`)
	s.Heartbeat = map[string]string{}
	state := ""
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) == 3 && f[0] == "agent_identity" {
			s.AgentIdentity = f[1] + " " + f[2]
		}
		if len(f) == 2 && f[0] == "agent_state" {
			state = f[1]
		}
		if len(f) == 2 && f[0] != "boot_id" && f[0] != "agent_state" {
			s.Heartbeat[f[0]] = f[1]
		}
	}
	s.AgentAlive = err == nil && (state == "S" || state == "R" || state == "D")
	s.MonoNS = mono16()
	s.ReadEnd = time.Since(start).Seconds()
	return s
}

func timeoutLog16(raw []byte, loss time.Time, budget time.Duration) ([]string, bool) {
	lines := []string{}
	proved := false
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.Contains(line, `err="control request ended"`) || (!strings.Contains(line, `msg="desired state"`) && !strings.Contains(line, `msg="lease renewal"`)) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, strings.TrimPrefix(fields[0], "time="))
		if err != nil || at.Before(loss) {
			continue
		}
		// This is safeErr's fixed redacted timeout/cancellation line. With the
		// original agent alive and no cancellation/signals, it denotes deadline
		// exhaustion, not connection refusal (which logs "control request failed").
		lines = append(lines, line)
		proved = proved || !at.Before(loss.Add(budget))
	}
	return lines, proved
}

func (p *liveProbe) liveAgentExpiry16(stack *labstack.Stack, node uuid.UUID, row map[string]any) {
	t := p.t
	row["status"] = "failed"
	row["owner"] = "w4-at-security-ops"
	row["bound_sources"] = map[string]any{
		"D":               "node's accepted SQLite lease mono_sent_ns + (expires_at-issued_at); require heartbeat policy_expires_mono_ns equal to D",
		"closure_reserve": "node/enforce/model.go DeadlineInstallBound=100ms + DeadlineGranularity=1ms; D-140/D-142 isolation lower closure boundary",
		"continuity_end":  "D - 101ms - 1s sampler resolution; formula declared before fault, instantiated from final accepted lease, never fitted to observed closure",
		"minimum_window":  "task requires at least three Step intervals; min(heartbeat_seconds,poll_seconds) from actual node config and node/agent/serving.go stepInterval",
		"control_budget":  "node/agent/serving.go controlPassTimeout: min(5s,watchdog_stale_seconds/3)",
		"cutoff":          "D+3s, then ten seconds quiet; existing followup11_transports_test.go lease_expiry observation bounds",
		"sampling":        "existing one-second transport cadence; record read start/end; original TCP/9446 and authenticated UDP/9445 independent senders",
		"setup_recovery":  "existing transport harness 90s setup and 120s recovery; TEST ONLY 30-minute finite standing, 60-second fleet cap",
	}
	adminSession := p.login("e2e-admin", "admin")
	admin := labstack.BrowserSession{Client: adminSession.client, CSRF: adminSession.csrf}
	m := p.login("secops-deadline-live-agent", "member")
	end := time.Now().UTC().Add(30 * time.Minute).Truncate(time.Second)
	body := api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY AT29 live agent continuity", PlanRevisionId: &p.planRevision, ValidUntil: end}
	var peer flowPeer
	controlStopped, workerPaused, fleetChanged := false, false, false
	restored := false
	var originalFleet int
	restore := func() {
		if restored {
			return
		}
		restored = true
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		fail := func(message string) { row["status"] = "failed"; t.Error(message) }
		if controlStopped {
			ok := exec.CommandContext(ctx, "docker", "start", livePrefix+"control").Run() == nil
			row["control_restart_ok"] = ok
			if !ok {
				fail("control restoration failed")
			}
		}
		if workerPaused {
			ok := exec.CommandContext(ctx, "docker", "unpause", livePrefix+"worker").Run() == nil
			row["worker_unpause_ok"] = ok
			if !ok {
				fail("worker restoration failed")
			}
		}
		var response labstack.Response
		for ctx.Err() == nil {
			response, _ = stack.Call(ctx, admin, "GET", "/api/v1/admin/fleet-config", nil, nil)
			if response.Status == 200 {
				break
			}
			if !pauseStanding15(ctx) {
				break
			}
		}
		if fleetChanged {
			changed, err := stack.Call(ctx, admin, "PUT", "/api/v1/admin/fleet-config", map[string]any{"max_offline_lease_seconds": originalFleet, "reason": "TEST ONLY restore original fleet cap"}, map[string]string{"If-Match": response.Header.Get("ETag"), "Idempotency-Key": uuid.NewString()})
			row["fleet_restoration_http_status"] = changed.Status
			after, readErr := stack.Call(ctx, admin, "GET", "/api/v1/admin/fleet-config", nil, nil)
			var config struct {
				Maximum int `json:"max_offline_lease_seconds"`
			}
			ok := err == nil && readErr == nil && changed.Status >= 200 && changed.Status < 300 && after.Status == 200 && json.Unmarshal(after.Body, &config) == nil && config.Maximum == originalFleet
			row["fleet_restoration_ok"] = ok
			if !ok {
				fail("fleet restoration read-back failed")
			}
		}
		standing := map[string]any{"status": "failed", "original_valid_until": end}
		row["standing_restoration"] = standing
		if err := restoreStanding15(ctx, stack, admin, m.member, peer.Device, body, standing); err != nil {
			fail("member restoration: " + err.Error())
		}
	}
	// Register before any public mutation or fault, including failed requests.
	t.Cleanup(restore)
	initial := map[string]any{}
	row["initial_grant"] = initial
	if _, err := grantCommand15(t.Context(), stack, admin, "/api/v1/admin/members/"+m.member+"/grants", body, initial); err != nil {
		t.Error(err)
		return
	}
	var ok bool
	peer, ok = p.joinFlowPeer(stack, m, "secops-deadline-live-agent", "f16-live-agent-expiry")
	if !ok {
		t.Error("live-agent member admission")
		return
	}
	row["before_counted_flow"] = p.transportFlow11(peer, 90*time.Second)
	if row["before_counted_flow"].(map[string]any)["counted"] != true {
		return
	}
	transport, err := stack.MemberTransportStack(livePrefix+peer.Service, "node")
	if err != nil {
		t.Error(err)
		return
	}
	t.Cleanup(func() {
		ok := transport.StopEstablishedTransports(context.Background()) == nil
		row["transport_cleanup_ok"] = ok
		if !ok {
			row["status"] = "failed"
			t.Error("transport cleanup")
		}
	})
	start := time.Now()
	monoStart := mono16()
	// Anchor uncertainty is the short interval between the two monotonic reads.
	row["clock_anchor"] = map[string]any{"clock": "CLOCK_MONOTONIC shared by this Linux host and its containers; no clock changes", "mono_ns": strconv.FormatInt(monoStart, 10), "elapsed_after_seconds": time.Since(start).Seconds()}
	session, err := transport.StartEstablishedTransports(t.Context(), peer.Device)
	row["session"] = session
	if err != nil {
		t.Error(err)
		return
	}
	a := p.continuitySample16(transport, peer.Device, start)
	time.Sleep(4 * time.Second)
	b := p.continuitySample16(transport, peer.Device, start)
	row["positive_samples"] = []continuitySample16{a, b}
	if !activeTransport11(a.Transport, b.Transport) {
		t.Error("original TCP/UDP positive prerequisite")
		return
	}
	raw, err := os.ReadFile(filepath.Join(p.state, "node/etc/agent-run.json"))
	var cfg struct {
		Heartbeat int `json:"heartbeat_seconds"`
		Poll      int `json:"poll_seconds"`
		Stale     int `json:"watchdog_stale_seconds"`
	}
	if err != nil || json.Unmarshal(raw, &cfg) != nil || min(cfg.Heartbeat, cfg.Poll, cfg.Stale) <= 0 {
		t.Error("actual agent cadence unavailable")
		return
	}
	row["agent_cadence"] = cfg
	fleet, err := stack.Call(t.Context(), admin, "GET", "/api/v1/admin/fleet-config", nil, nil)
	var config struct {
		Maximum int `json:"max_offline_lease_seconds"`
	}
	if err != nil || fleet.Status != 200 || json.Unmarshal(fleet.Body, &config) != nil || config.Maximum <= 0 {
		t.Error("original fleet cap unavailable")
		return
	}
	originalFleet = config.Maximum
	row["original_fleet_maximum"] = originalFleet
	fleetChanged = true
	row["fleet_set_60"] = p.fleetMaximum(60)
	binary := filepath.Join(p.root, ".agent/tmp/secops-leaseprobe")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./tests/e2e/at/secops/leaseprobe")
	build.Dir = p.root
	if build.Run() != nil {
		t.Error("read-only lease probe build")
		return
	}
	t.Cleanup(func() { _ = os.Remove(binary) })
	if exec.CommandContext(t.Context(), "docker", "cp", binary, livePrefix+"node:/var/lib/vpn-node/e2e/secops-leaseprobe").Run() != nil {
		t.Error("read-only lease probe install")
		return
	}
	var lease nodeLease16
	for until := time.Now().Add(90 * time.Second); time.Now().Before(until); {
		lease, err = p.nodeLease16()
		member := false
		for _, v := range lease.Members {
			member = member || (v.ID == m.member && !v.End.Before(lease.Expires))
		}
		hb := p.heartbeatView()
		if err == nil && lease.Max == 60 && member && lease.Deadline-mono16() > int64(4*time.Duration(min(cfg.Heartbeat, cfg.Poll))*time.Second) && hb["serving"] == "1" && hb["policy_expires_mono_ns"] == strconv.FormatInt(lease.Deadline, 10) {
			break
		}
		time.Sleep(time.Second)
	}
	row["lease_before_fault"] = lease
	if err != nil || lease.Max != 60 {
		t.Error("no locally accepted short lease")
		return
	}
	row["before_kernel_timeout"] = p.timeoutView()
	samples := []continuitySample16{p.continuitySample16(transport, peer.Device, start)}
	loss := time.Now()
	row["control_loss_started_at"] = loss.UTC()
	row["control_loss_monotonic_seconds"] = time.Since(start).Seconds()
	workerPaused = true
	if exec.CommandContext(t.Context(), "docker", "pause", livePrefix+"worker").Run() != nil {
		t.Error("worker pause")
		return
	}
	controlStopped = true
	if exec.CommandContext(t.Context(), "docker", "stop", "--time", "2", livePrefix+"control").Run() != nil {
		t.Error("control stop")
		return
	}
	row["control_stop_completed_at"] = time.Now().UTC()
	row["control_stop_completed_monotonic_seconds"] = time.Since(start).Seconds()
	// The node may have accepted one last in-flight renewal. Read the actual
	// retained lease after disconnection; no later renewal is possible now.
	lease, err = p.nodeLease16()
	row["signed_lease"] = lease
	row["service_end"] = end
	if err != nil {
		t.Error("disconnected local lease unreadable")
		return
	}
	deadline := float64(lease.Deadline-monoStart) / 1e9
	bounds := continuityBounds16{Loss: loss.Sub(start).Seconds(), Deadline: deadline, End: continuityEnd16(deadline), Step: float64(min(cfg.Heartbeat, cfg.Poll)), Budget: min(5, float64(cfg.Stale)/3), Stale: float64(cfg.Stale)}
	row["bounds"] = bounds
	row["earliest_deadline"] = lease.Expires
	first := p.continuitySample16(transport, peer.Device, start)
	samples = append(samples, first)
	member := false
	for _, v := range lease.Members {
		member = member || (v.ID == m.member && !v.End.Before(lease.Expires))
	}
	if !member || lease.Max != 60 || first.Heartbeat["policy_expires_mono_ns"] != strconv.FormatInt(lease.Deadline, 10) || bounds.End-bounds.Loss < 3*bounds.Step {
		row["samples"] = samples
		t.Error("lease/heartbeat deadline or minimum continuity window not qualified")
		return
	}
	// Schedule the final continuity read at End. The preceding observation is
	// at least one cadence earlier so independent sender counts can advance.
	next := time.Since(start).Seconds() + continuityCadence16.Seconds()
	for next < bounds.Deadline+(expirySlack16+expiryQuiet16).Seconds()+continuityCadence16.Seconds() {
		if next < bounds.End && next+continuityCadence16.Seconds() > bounds.End {
			next = bounds.End
		}
		if wait := time.Duration(next*1e9) - time.Since(start); wait > 0 {
			time.Sleep(wait)
		}
		samples = append(samples, p.continuitySample16(transport, peer.Device, start))
		next += continuityCadence16.Seconds()
	}
	row["samples"] = samples
	log, logErr := p.docker("node", "cat", "/var/lib/vpn-node/e2e/agent.log")
	lines, timeoutProved := timeoutLog16(log, loss, time.Duration(bounds.Budget*1e9))
	row["agent_timeout_log"] = lines
	row["agent_log_read_ok"] = logErr == nil
	row["timeout_evidence_source"] = "node/agent/sync.go safeErr: control request ended means deadline/cancellation; original PID/start ticks remains alive without any pause/signal; connection refusal logs control request failed"
	verdict := evaluateContinuity16(samples, bounds, logErr == nil && timeoutProved)
	row["verdict"] = verdict
	row["after_kernel_timeout"] = p.timeoutView()
	row["after_heartbeat"] = p.heartbeatView()
	if !verdict.Passed {
		t.Error("live-agent continuity/cutoff failed:", verdict.Errors)
	}
	restore()
	row["recovery_flow"] = p.transportFlow11(peer, 120*time.Second)
	row["recovery_ack"] = p.publicationObservation(node)
	final := p.continuitySample16(transport, peer.Device, start)
	row["agent_after_recovery"] = final
	if !final.AgentAlive || final.AgentIdentity != samples[0].AgentIdentity {
		t.Error("original agent did not survive recovery")
	}
	if !t.Failed() {
		row["status"] = "passed"
	}
	t.Logf("AT29 control_lost_agent_alive: %s", row["status"])
}

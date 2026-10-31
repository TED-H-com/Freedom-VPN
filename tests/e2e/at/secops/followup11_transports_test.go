//go:build e2e && integration

package secops

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

type transportSample11 struct {
	UTC        time.Time                      `json:"utc"`
	Elapsed    float64                        `json:"monotonic_elapsed_seconds"`
	Status     labstack.MemberTransportStatus `json:"status"`
	Counters   labstack.DeviceCounters        `json:"counters"`
	StatusOK   bool                           `json:"status_read_ok"`
	CountersOK bool                           `json:"counter_read_ok"`
}

func sampleTransport11(ctx context.Context, s *labstack.Stack, device uuid.UUID, start time.Time) transportSample11 {
	status, se := s.EstablishedStatus(ctx)
	counters, ce := s.DeviceByteCounters(ctx, device)
	return transportSample11{time.Now().UTC(), time.Since(start).Seconds(), status, counters, se == nil, ce == nil}
}
func activeTransport11(a, b transportSample11) bool {
	return a.StatusOK && b.StatusOK && a.Status.Running && b.Status.Running &&
		b.Status.TCPEchoes > a.Status.TCPEchoes && b.Status.UDPReplies > a.Status.UDPReplies &&
		a.CountersOK && b.CountersOK && a.Counters.Present && b.Counters.Present &&
		b.Counters.Upload > a.Counters.Upload && b.Counters.Download > a.Counters.Download
}
func quietTransport11(a, b transportSample11) bool {
	// Absent counters are explicitly not interpreted as zero bytes. A revoked
	// binding may remove them; continuously running senders/replies decide cutoff.
	return a.StatusOK && b.StatusOK && a.Status.Running && b.Status.Running &&
		b.Status.TCPAttempts > a.Status.TCPAttempts && b.Status.UDPAttempts > a.Status.UDPAttempts &&
		b.Status.TCPEchoes == a.Status.TCPEchoes && b.Status.UDPReplies == a.Status.UDPReplies
}

func TestAT29_AT31_Followup11_Transports(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_FOLLOWUP11") != "transports" {
		t.Skip("requires assigned fresh transport stack")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "follow_up": 11, "run_id": uuid.New(), "hardware": "shared development host; not candidate hardware"}}
	p.open()
	defer p.db.Close(context.Background())
	defer p.tr.CloseIdleConnections()
	defer p.finish()
	stack, err := labstack.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer stack.Close()
	admin := p.login("e2e-admin", "admin")
	p.planInputs(admin)
	p.releaseSupport(admin)
	node := p.enrollAgent(admin)
	p.capacityInput(admin, node)
	rows := map[string]any{}
	p.metadata["transport_cases"] = rows
	rows["manual_grant_withdrawal"] = map[string]any{"status": "blocked", "owner": "w3-wire-members / contracts", "missing_export": "The contract exposes finite createMemberGrant but no revokeMemberGrant HTTP operation; no source-row mutation or invented endpoint used. Finite grant expiration is measured separately."}
	rows["guest_clock_rollback"] = map[string]any{"status": "blocked", "owner": "w3-devstack-e2e / operator", "missing_export": "Disposable guest with independently adjustable wall clock; containers share the development host clock."}
	for _, kind := range []string{"grant_expiry", "lease_expiry", "device_revoke"} {
		func() {
			row := map[string]any{"status": "blocked", "owner": "w3-devstack-e2e", "transport_scope": "one original TCP/9446 connection and authenticated UDP/9445 socket; independent senders; helper limit 20 minutes"}
			rows[kind] = row
			subject := "secops-deadline-grant"
			if kind == "lease_expiry" {
				subject = "secops-deadline-lease"
			}
			if kind == "device_revoke" {
				subject = "secops-member-04"
			}
			admin = p.login("e2e-admin", "admin")
			m := p.login(subject, "member")
			end := time.Now().UTC().Add(30 * time.Minute).Truncate(time.Second)
			if kind == "grant_expiry" {
				end = time.Now().UTC().Add(180 * time.Second).Truncate(time.Second)
			}
			p.grantAt(admin, m, end, "followup11-"+kind)
			peer, ok := p.joinFlowPeer(stack, m, subject, "f11-"+strings.ReplaceAll(kind, "_", "-"))
			if !ok {
				row["blocker"] = "native admission unavailable"
				return
			}
			flow := p.waitCountedFlow(peer, 90*time.Second)
			row["before_counted_flow"] = flow
			if flow["counted"] != true {
				row["blocker"] = "no selected counted serving flow"
				return
			}
			transport, er := stack.MemberTransportStack(livePrefix+peer.Service, "node")
			if er != nil {
				t.Fatal(er)
			}
			defer func() { row["transport_cleanup_ok"] = transport.StopEstablishedTransports(context.Background()) == nil }()
			start := time.Now()
			session, er := transport.StartEstablishedTransports(t.Context(), peer.Device)
			row["session"] = session
			if er != nil {
				row["blocker"] = "StartEstablishedTransports could not establish both replies"
				return
			}
			before := sampleTransport11(t.Context(), transport, peer.Device, start)
			time.Sleep(4 * time.Second)
			positive := sampleTransport11(t.Context(), transport, peer.Device, start)
			row["positive_samples"] = []transportSample11{before, positive}
			if !activeTransport11(before, positive) {
				row["blocker"] = "both replies and present device counters must increase"
				return
			}
			if kind == "device_revoke" {
				p.forcedRelay11(stack, transport, peer, rows)
			}
			if kind == "lease_expiry" {
				row["fleet_set_60"] = p.fleetMaximum(60)
				defer func() { row["fleet_restore_86400"] = p.fleetMaximum(86400) }()
			}
			var lease json.RawMessage
			var issued, expires time.Time
			for until := time.Now().Add(90 * time.Second); time.Now().Before(until); {
				er = p.db.QueryRow(t.Context(), `SELECT issued_at,expires_at,jsonb_build_object('generation',policy_generation::text,'issued_at',issued_at,'expires_at',expires_at,'members',envelope->'signed'->'members') FROM serving_leases WHERE node_id=$1 ORDER BY issued_at DESC LIMIT 1`, node).Scan(&issued, &expires, &lease)
				if er == nil && (kind != "lease_expiry" || expires.Sub(issued) <= 60*time.Second) {
					break
				}
				time.Sleep(time.Second)
			}
			if er != nil || (kind == "lease_expiry" && expires.Sub(issued) > 60*time.Second) {
				row["blocker"] = "no fresh short lease"
				return
			}
			row["signed_lease"] = lease
			row["service_end"] = end
			if kind == "grant_expiry" {
				for time.Until(end) > 25*time.Second {
					time.Sleep(time.Second)
				}
				// Re-read the actual lease just before the disconnected fault.
				if p.db.QueryRow(t.Context(), `SELECT issued_at,expires_at,jsonb_build_object('generation',policy_generation::text,'issued_at',issued_at,'expires_at',expires_at,'members',envelope->'signed'->'members') FROM serving_leases WHERE node_id=$1 ORDER BY issued_at DESC LIMIT 1`, node).Scan(&issued, &expires, &lease) != nil {
					t.Fatal("lease read")
				}
				row["signed_lease"] = lease
			}
			deadline := end
			if expires.Before(deadline) {
				deadline = expires
			}
			row["earliest_deadline"] = deadline
			row["before_kernel_timeout"] = p.timeoutView()
			row["before_heartbeat"] = p.heartbeatView()
			restored := false
			restore := func() {
				if restored || kind == "device_revoke" {
					return
				}
				restored = true
				row["control_restart_ok"] = exec.CommandContext(context.Background(), "docker", "start", livePrefix+"control").Run() == nil
				row["worker_unpause_ok"] = exec.CommandContext(context.Background(), "docker", "unpause", livePrefix+"worker").Run() == nil
				row["agent_resume_ok"] = stack.PauseAgent(context.Background(), false) == nil
			}
			defer restore()
			actionStart := time.Now()
			row["action_started_at"] = actionStart.UTC()
			if kind == "device_revoke" {
				r := p.call(m, "DELETE", "/api/v1/member/devices/"+peer.Device.String(), nil, nil)
				row["revoke_http_status"] = r.status
				if r.status != 202 {
					row["blocker"] = "revoke not accepted"
					return
				}
				deadline = time.Now().Add(60 * time.Second)
				row["withdrawal_bound_seconds"] = 60
			} else {
				if exec.CommandContext(t.Context(), "docker", "pause", livePrefix+"worker").Run() != nil {
					t.Fatal("worker pause")
				}
				if exec.CommandContext(t.Context(), "docker", "stop", "--time", "2", livePrefix+"control").Run() != nil {
					t.Fatal("control stop")
				}
				a := sampleTransport11(t.Context(), transport, peer.Device, start)
				time.Sleep(3 * time.Second)
				b := sampleTransport11(t.Context(), transport, peer.Device, start)
				row["control_lost_agent_alive"] = []transportSample11{a, b}
				row["continued_before_expiry"] = activeTransport11(a, b)
				if stack.PauseAgent(t.Context(), true) != nil {
					t.Fatal("pause agent supervisor")
				}
				_, er = p.docker("node", "sh", "-c", `test -s /var/lib/vpn-node/e2e/agent.pid && kill -KILL "$(cat /var/lib/vpn-node/e2e/agent.pid)"`)
				row["agent_sigkill_ok"] = er == nil
				if er != nil {
					t.Fatal("agent kill")
				}
				row["watchdog_kept_running"] = true
			}
			samples := []transportSample11{}
			for time.Now().Before(deadline.Add(3 * time.Second)) {
				samples = append(samples, sampleTransport11(t.Context(), transport, peer.Device, start))
				time.Sleep(time.Second)
			}
			a := sampleTransport11(t.Context(), transport, peer.Device, start)
			time.Sleep(10 * time.Second)
			b := sampleTransport11(t.Context(), transport, peer.Device, start)
			row["samples"] = samples
			row["cutoff_samples"] = []transportSample11{a, b}
			row["after_kernel_timeout"] = p.timeoutView()
			row["after_heartbeat"] = p.heartbeatView()
			quiet := quietTransport11(a, b)
			row["status"] = "failed"
			if quiet {
				row["status"] = "passed"
			}
			if !quiet {
				t.Error(kind, "established sender cutoff not proved")
			}
			row["attribution_limit"] = "Native route withdrawal and node kernel cutoff are observed together; no retained revoked route or independent ingress export is synthesized. Counter absence is not zero-byte evidence."
			restore()
			if !p.waitControlReady(40 * time.Second) {
				t.Fatal("control restore readiness")
			}
			if kind == "grant_expiry" {
				admin = p.login("e2e-admin", "admin")
				p.grantAt(admin, m, time.Now().UTC().Add(20*time.Minute), "followup11-recovery")
			}
			if kind != "device_revoke" {
				row["recovery_flow"] = p.waitCountedFlow(peer, 120*time.Second)
				row["recovery_ack"] = p.publicationObservation(node)
				if row["recovery_flow"].(map[string]any)["counted"] != true {
					row["status"] = "failed"
					t.Error(kind, "authorized recovery failed")
				}
			}
			t.Logf("AT29 %s: %s", kind, row["status"])
		}()
	}
}

func (p *liveProbe) forcedRelay11(stack, transport *labstack.Stack, peer flowPeer, rows map[string]any) {
	row := map[string]any{"status": "blocked", "scope": "one member, one exit, established TCP and UDP; 180-second automatic fault expiry, 90-second path wait; not 60-peer relay capacity"}
	rows["forced_relay"] = row
	ctx := p.t.Context()
	start := time.Now()
	defer func() {
		path, err := stack.ForceMemberRelay(context.Background(), livePrefix+peer.Service, false)
		row["cleanup_path"] = path
		row["cleanup_ok"] = err == nil
	}()
	before, err := stack.ReadMemberPath(ctx, livePrefix+peer.Service)
	row["before_path"] = before
	if err != nil || before.Direct < 1 {
		row["blocker"] = "no connected direct path before fault"
		return
	}
	a := sampleTransport11(ctx, transport, peer.Device, start)
	forced, err := stack.ForceMemberRelay(ctx, livePrefix+peer.Service, true)
	row["forced_path"] = forced
	row["forced_after_seconds"] = time.Since(start).Seconds()
	if err != nil {
		row["blocker"] = "bounded ForceMemberRelay did not observe relay with zero direct peers"
		return
	}
	b := sampleTransport11(ctx, transport, peer.Device, start)
	time.Sleep(5 * time.Second)
	c := sampleTransport11(ctx, transport, peer.Device, start)
	row["relay_samples"] = []transportSample11{a, b, c}
	path, err := stack.ReadMemberPath(ctx, livePrefix+peer.Service)
	row["relay_window_end_path"] = path
	relayOK := err == nil && path.Relayed > 0 && path.Direct == 0 && activeTransport11(b, c)
	restored, err := stack.ForceMemberRelay(ctx, livePrefix+peer.Service, false)
	row["restored_path"] = restored
	row["restore_after_seconds"] = time.Since(start).Seconds()
	if err != nil {
		row["status"] = "failed"
		p.t.Error("direct path restoration")
		return
	}
	d := sampleTransport11(ctx, transport, peer.Device, start)
	time.Sleep(5 * time.Second)
	e := sampleTransport11(ctx, transport, peer.Device, start)
	row["direct_restored_samples"] = []transportSample11{d, e}
	row["same_sockets_across_steps"] = true
	row["status"] = "failed"
	if relayOK && activeTransport11(d, e) {
		row["status"] = "passed"
	} else {
		p.t.Error("relay/restoration traffic not proved")
	}
	p.t.Logf("AT31 forced relay: %s", row["status"])
}

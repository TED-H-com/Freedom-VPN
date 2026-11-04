//go:build e2e && integration

package secops

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func (p *liveProbe) manualGrantWithdrawal15(stack *labstack.Stack, node uuid.UUID, row map[string]any) {
	t := p.t
	row["status"] = "failed"
	row["owner"] = "w4-at-security-ops"
	row["bounds"] = map[string]any{
		"online_withdrawal_seconds": 60, "online_withdrawal_source": "§16.5 online withdrawal recommended starting target; measured conservatively from request start before central commit",
		"offline_deadline_source": "§9.4, §16.3 and §19 AT29: minimum of actual signed lease and service end; recorded separately, not used to extend online withdrawal",
		"harness_setup_seconds":   90, "harness_recovery_seconds": 120, "harness_quiet_window_seconds": 10, "harness_sample_seconds": 1,
		"harness_source":       "existing followup11_transports_test.go setup, recovery, cutoff and polling windows; test observation limits, not product deadlines",
		"grant_window_minutes": 30, "grant_window_source": "existing transport fixture, TEST ONLY finite grant; restore the same end, never extend it",
		"recovery_ack_seconds": 60, "recovery_ack_source": "§16.5 online policy application target",
		"positive_window_seconds": 4, "positive_window_source": "existing followup11_transports_test.go positive transport prerequisite",
	}
	adminSession := p.login("e2e-admin", "admin")
	admin := labstack.BrowserSession{Client: adminSession.client, CSRF: adminSession.csrf}
	m := p.login("secops-deadline-withdrawal", "member")
	end := time.Now().UTC().Add(30 * time.Minute).Truncate(time.Second)
	path := "/api/v1/admin/members/" + m.member + "/grants"
	body := api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY AT29 manual withdrawal", PlanRevisionId: &p.planRevision, ValidUntil: end}
	var peer flowPeer
	restored := false
	// Register before the first grant request, including a lost-response path.
	restore := func() {
		if restored {
			return
		}
		restored = true
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		recovery := map[string]any{"status": "failed", "original_valid_until": end}
		row["standing_restoration"] = recovery
		if err := restoreStanding15(ctx, stack, admin, m.member, peer.Device, body, recovery); err != nil {
			t.Error("AT29 standing restoration:", err)
		}
	}
	t.Cleanup(restore)
	initial := map[string]any{}
	row["initial_grant"] = initial
	grant, err := grantCommand15(t.Context(), stack, admin, path, body, initial)
	if err != nil {
		t.Error(err)
		return
	}
	for until := time.Now().Add(90 * time.Second); ; {
		_, ok := memberStanding15(t.Context(), stack, admin, m.member, end)
		if ok {
			break
		}
		if time.Now().After(until) {
			t.Error("initial grant projection timeout")
			return
		}
		time.Sleep(time.Second)
	}
	var ok bool
	peer, ok = p.joinFlowPeer(stack, m, "secops-deadline-withdrawal", "f15-manual-grant-withdrawal")
	if !ok {
		row["blocker"] = "native admission unavailable"
		t.Error("manual withdrawal admission")
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
			t.Error("transport cleanup failed")
		}
	})
	start := time.Now()
	session, err := transport.StartEstablishedTransports(t.Context(), peer.Device)
	row["session"] = session
	if err != nil {
		t.Error("established withdrawal transports:", err)
		return
	}
	a := sampleTransport11(t.Context(), transport, peer.Device, start)
	time.Sleep(4 * time.Second)
	b := sampleTransport11(t.Context(), transport, peer.Device, start)
	row["positive_samples"] = []transportSample11{a, b}
	if !activeTransport11(a, b) {
		t.Error("both original transports must be active")
		return
	}
	row["before_kernel_timeout"], row["before_heartbeat"] = p.timeoutView(), p.heartbeatView()
	row["service_end"] = end
	// Retain actual signed lease deadlines without signatures or private material.
	var lease json.RawMessage
	var offlineEnd time.Time
	leaseReadStart := time.Now()
	for until := leaseReadStart.Add(90 * time.Second); time.Now().Before(until); {
		// The central observation can lag local forwarding by its own report
		// cadence. Wait for the exact observed lease containing this member;
		// never substitute the latest issued (possibly unapplied) lease.
		readErr := p.db.QueryRow(t.Context(), `SELECT jsonb_build_object('generation',s.policy_generation::text,'issued_at',s.issued_at,'expires_at',s.expires_at,'members',s.envelope->'signed'->'members')
FROM nodes n JOIN node_observations o ON o.node_id=n.id AND o.incarnation_id=n.current_incarnation_id AND o.observed_at=n.last_observed_at
JOIN serving_leases s ON s.id=(o.report_payload->'lease'->>'lease_id')::uuid WHERE n.id=$1 ORDER BY o.received_at DESC LIMIT 1`, node).Scan(&lease)
		var window struct {
			Expires time.Time `json:"expires_at"`
			Members []struct {
				Member string    `json:"member_id"`
				End    time.Time `json:"lease_end"`
			} `json:"members"`
		}
		if readErr == nil && json.Unmarshal(lease, &window) == nil && !window.Expires.IsZero() {
			for _, member := range window.Members {
				if member.Member == m.member && !member.End.IsZero() {
					offlineEnd = end
					if window.Expires.Before(offlineEnd) {
						offlineEnd = window.Expires
					}
					if member.End.Before(offlineEnd) {
						offlineEnd = member.End
					}
				}
			}
		}
		if offlineEnd.After(time.Now()) {
			break
		}
		time.Sleep(time.Second)
	}
	row["lease_observation_wait_seconds"] = time.Since(leaseReadStart).Seconds()
	row["signed_lease"] = lease
	if !offlineEnd.After(time.Now()) {
		t.Error("observed lease does not authorize withdrawal subject")
		return
	}
	row["offline_earliest_deadline"] = offlineEnd
	row["lease_source"] = "lease ID named by current immutable agent observation"
	action := time.Now()
	deadline := action.Add(60 * time.Second)
	if offlineEnd.Before(deadline) {
		deadline = action.Add(offlineEnd.Sub(action))
	}
	row["action_started_at"], row["earliest_deadline"] = action.UTC(), deadline.UTC()
	row["action_monotonic_seconds"], row["deadline_monotonic_seconds"] = action.Sub(start).Seconds(), deadline.Sub(start).Seconds()
	withdrawal := map[string]any{}
	row["withdrawal"] = withdrawal
	op, err := grantCommand15(t.Context(), stack, admin, path+"/"+grant.Resource.Id.String()+"/revoke", api.ReasonRequest{Reason: "TEST ONLY AT29 manual withdrawal"}, withdrawal)
	if err != nil {
		t.Error(err)
		return
	}
	if op.Kind != api.OperationKindMemberGrantRevoke || op.Resource.Id != grant.Resource.Id {
		t.Error("wrong withdrawal receipt")
		return
	}
	samples := []transportSample11{b}
	completed := false
	for time.Now().Before(deadline.Add(10 * time.Second)) {
		samples = append(samples, sampleTransport11(t.Context(), transport, peer.Device, start))
		actual, valid := readOperation15(t.Context(), stack, admin, op.OperationId)
		withdrawal["last_operation"] = actual
		if valid && actual.Status == api.OperationStatusSucceeded && !completed {
			completed = true
			withdrawal["completed_at"], withdrawal["completion_monotonic_seconds"] = time.Now().UTC(), time.Since(start).Seconds()
		}
		if _, exists := row["pending_device"]; !exists {
			_, d := p.deviceRead10(adminSession, m.member, peer.Device, true)
			if d.Status == "pending" {
				row["pending_device"], row["pending_after_seconds"] = d, time.Since(action).Seconds()
			}
		}
		time.Sleep(time.Second)
	}
	row["samples"] = samples
	row["after_kernel_timeout"], row["after_heartbeat"] = p.timeoutView(), p.heartbeatView()
	stops, cutoff := transportStops15(samples, deadline.Sub(start).Seconds())
	row["transport_stop_points"] = stops
	row["new_flow_after_withdrawal"] = p.rawFlow(peer)
	row["attribution_limit"] = "Native route withdrawal and node kernel policy removal occur together; timeout set and unchanged original sockets are retained, without claiming separate route-versus-conntrack causality."
	if !completed || !cutoff || row["new_flow_after_withdrawal"].(map[string]any)["http_200"] != false {
		t.Error("manual withdrawal completion/cutoff not proved")
	}
	// Hold the withdrawn state with the Agent stopped. The dedicated watchdog
	// and kernel remain independent; no process restart can authorize a member.
	if err := stack.PauseAgent(t.Context(), true); err != nil {
		t.Error(err)
		return
	}
	resume := func() { row["agent_resume_ok"] = stack.PauseAgent(context.Background(), false) == nil }
	defer resume()
	_, err = p.docker("node", "sh", "-c", `test -s /var/lib/vpn-node/e2e/agent.pid && kill -KILL "$(cat /var/lib/vpn-node/e2e/agent.pid)"`)
	row["agent_sigkill_ok"] = err == nil
	_, watchdogErr := p.docker("node", "pgrep", "-f", "^fvpn-watchdog run")
	row["watchdog_kept_running"] = watchdogErr == nil
	a = sampleTransport11(t.Context(), transport, peer.Device, start)
	time.Sleep(10 * time.Second)
	b = sampleTransport11(t.Context(), transport, peer.Device, start)
	row["independent_cutoff_samples"] = []transportSample11{a, b}
	if err != nil || watchdogErr != nil || !quietTransport11(a, b) {
		t.Error("withdrawal independent cutoff not proved")
	}
	resume()
	restore()
	row["recovery_flow"] = p.transportFlow11(peer, 120*time.Second)
	ackOK := false
	for until := time.Now().Add(60 * time.Second); time.Now().Before(until); {
		ack := p.publicationObservation(node)
		generations := p.observedLeaseGeneration(node)
		row["recovery_ack"], row["recovery_generations"] = ack, generations
		before := row["before_heartbeat"].(map[string]string)["generation"]
		ackOK = generations["read_ok"] == true && generations["matches_applied"] == true && generations["desired_generation"] == generations["applied_generation"] && generations["applied_generation"] != before && ack["member_forwarding_enabled"] == true
		if ackOK {
			break
		}
		time.Sleep(time.Second)
	}
	if !ackOK {
		t.Error("withdrawal recovery ACK missing")
	}
	// The cutoff evidence above uses only the original sockets. Recovery opens
	// fresh ones after the new grant and its applied ACK, on the same ports.
	recoveredSession, recoveryErr := transport.StartEstablishedTransports(t.Context(), peer.Device)
	row["recovery_session"] = recoveredSession
	if recoveryErr != nil {
		t.Error("withdrawal recovery transports:", recoveryErr)
	} else {
		a = sampleTransport11(t.Context(), transport, peer.Device, start)
		time.Sleep(4 * time.Second)
		b = sampleTransport11(t.Context(), transport, peer.Device, start)
		row["recovery_transport_samples"] = []transportSample11{a, b}
		if !activeTransport11(a, b) {
			t.Error("withdrawal recovery TCP/UDP not both counted")
		}
	}
	if !t.Failed() {
		row["status"] = "passed"
	}
	t.Logf("AT29 manual_grant_withdrawal: %s", row["status"])
}

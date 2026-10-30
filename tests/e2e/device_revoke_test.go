//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

func (r *runner) deviceRevoke(ctx context.Context) {
	rec := Record{ID: "device-revoke", Title: "Revoke one native device while its sibling keeps forwarding", Spec: []string{"§8", "§10", "§19 AT02", "F3-02"}, Status: StatusBlocked}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	note("owner", "w3-devstack-e2e: controlled stale-client transport and post-revoke ingress measurement")
	if !r.flowCounted {
		note("gate", "positive member flow prerequisite absent")
		return
	}
	s := &Stack{lab: r.lab}
	member, err := s.Login(ctx, "e2e-member", "member")
	if err != nil {
		note("gate", "member login failed")
		return
	}
	container, err := s.StartNativePeer(ctx, "revoke")
	if err != nil {
		note("gate", err.Error())
		return
	}
	peer, err := s.JoinNativeMember(ctx, member, "e2e-member", container, container)
	if err != nil {
		note("gate", err.Error())
		return
	}
	second := *r
	secondLab := *r.lab
	second.lab = &secondLab
	second.lab.member = container
	second.usageDevice = peer.DeviceID
	label := func(prefix string) func(string, string) { return func(k, v string) { note(prefix+"_"+k, v) } }
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	bound := waitUntil(wait, time.Second, func() bool { _, ok := readProjectedBinding(r.desiredRaw(), peer.DeviceID); return ok })
	cancel()
	if !bound || !second.countedFlow(ctx, label("revoked_before")) || !r.countedFlow(ctx, label("sibling_before")) {
		note("gate", "two counted flows not established")
		return
	}
	if err := s.PauseNativeControl(ctx, container, true); err != nil {
		note("gate", "cannot pause native NetBird control service")
		return
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := s.PauseNativeControl(cleanup, container, false)
		note("client_control_resumed", boolText(err == nil))
		if err != nil {
			rec.Status = StatusFail
		}
	}()
	note("client_control_paused", "SIGSTOP of native NetBird service only; kernel WireGuard/routes remain; exit agent continues")
	pausedFlow := second.tracedFetch(ctx, label("paused_before"))
	note("paused_client_counted", boolText(pausedFlow))
	if !pausedFlow {
		note("gate", "paused native service cannot retain positive transport; missing stale-client kernel path, owner w3-devstack-e2e")
		return
	}
	pid, _ := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/agent.pid")
	beforeGen := r.lab.forwardChain().Generation
	ct, _ := r.lab.docker(r.lab.node, "sh", "-c", "command -v conntrack")
	note("conntrack_executable", strings.TrimSpace(string(ct)))
	// Keep native requests in flight across DELETE on the deliberately stale
	// kernel transport. Never restore a revoked route or alter node enforcement.
	transitionCtx, stopTransition := context.WithTimeout(ctx, 30*time.Second)
	defer stopTransition()
	transition := exec.CommandContext(transitionCtx, "docker", "exec", container, "timeout", "25", "sh", "-c", `i=0; while [ "$i" -lt 100 ]; do wget --timeout=1 --tries=1 --ca-certificate=/etc/vpn-node/ca.pem -qO /dev/null https://releases.example.invalid/healthz || true; i=$((i+1)); sleep 0.1; done`)
	transitionStarted := transition.Start() == nil
	defer func() {
		stopTransition()
		if transitionStarted {
			_ = transition.Wait()
		}
	}()
	transitionRaw, _ := r.lab.docker(r.lab.node, "nft", "-j", "list", "table", "inet", "fvpn")
	transitionState, _ := enforce.ParseState(transitionRaw)
	start := time.Now().UTC()
	resp, err := s.Call(ctx, member, "DELETE", "/api/v1/member/devices/"+peer.DeviceID.String(), nil, map[string]string{"Idempotency-Key": uuid.NewString()})
	note("delete", httpDetail(resp, err))
	note("delete_path", "/api/v1/member/devices/"+peer.DeviceID.String())
	note("delete_started_at", start.Format(time.RFC3339Nano))
	if err != nil || resp.Status != 202 {
		return
	}
	wait, cancel = context.WithTimeout(ctx, 90*time.Second)
	applied := waitUntil(wait, time.Second, func() bool {
		_, bound := readProjectedBinding(r.desiredRaw(), peer.DeviceID)
		return !bound && r.lab.forwardChain().Generation > beforeGen && r.lab.forwardChain().MemberAccept > 0
	})
	cancel()
	note("new_generation_applied", boolText(applied))
	transitionRaw, _ = r.lab.docker(r.lab.node, "nft", "-j", "list", "table", "inet", "fvpn")
	transitionEnd, _ := enforce.ParseState(transitionRaw)
	note("transition_ctunku_before", fmt.Sprint(transitionState.Counters["ctunku"].Packets))
	note("transition_ctunku_after", fmt.Sprint(transitionEnd.Counters["ctunku"].Packets))
	routeAfter, _ := r.lab.docker(container, "ip", "route", "get", r.originIP())
	networksAfter := []byte("not queried: native userspace is deliberately paused")
	note("revoked_route_after", strings.TrimSpace(string(routeAfter)))
	note("revoked_networks_after", strings.TrimSpace(string(networksAfter)))
	if !applied {
		note("gate", "revoked binding still present or new serving generation absent after 90s")
		return
	}
	conn, e := r.lab.db(ctx)
	if e != nil {
		note("gate", "accepted observation projection unavailable")
		return
	}
	if e == nil {
		defer conn.Close(ctx)
		var row []byte
		wait, cancel = context.WithTimeout(ctx, 30*time.Second)
		observed := waitUntil(wait, time.Second, func() bool {
			return conn.QueryRow(wait, `SELECT jsonb_build_object('generation',applied_config_generation::text,'provisioning_state',provisioning_state,'safety_status',safety_status) FROM nodes WHERE id=$1 AND applied_config_generation>$2`, r.nodeID, beforeGen).Scan(&row) == nil
		})
		cancel()
		note("new_observation_projected", boolText(observed))
		note("stored_node", string(row))
		var observation []byte
		if conn.QueryRow(ctx, `SELECT jsonb_build_object('generation',report_payload->'applied_config_generation','checks',report_payload->'checks','errors',report_payload->'errors','member_forwarding_enabled',report_payload->'member_forwarding_enabled') FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC LIMIT 1`, r.nodeID).Scan(&observation) == nil {
			note("accepted_observation_after_delete", string(observation))
		}
		if !observed {
			return
		}
	}
	addr, _ := r.lab.docker(container, "ip", "-o", "-4", "addr", "show", "fvnb0")
	match := regexp.MustCompile(`inet (100\.[0-9.]+)/`).FindStringSubmatch(string(addr))
	if len(match) != 2 {
		return
	}
	source := match[1]
	frozen, ok := second.admissionBytes()
	if !ok {
		return
	}
	raw, _ := r.lab.docker(r.lab.node, "nft", "-j", "list", "table", "inet", "fvpn")
	st, _ := enforce.ParseState(raw)
	captureCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	cap := exec.CommandContext(captureCtx, "docker", "exec", r.lab.node, "timeout", "-s", "INT", "8", "tcpdump", "-tt", "-nn", "-q", "-s", "96", "-i", "fvmb0", "src", source, "and", "dst", r.originIP(), "and", "tcp", "port", "443")
	var output bytes.Buffer
	cap.Stdout = &output
	cap.Stderr = &output
	if cap.Start() != nil {
		return
	}
	time.Sleep(time.Second)
	id := uuid.NewString()
	if _, err = r.originProbe(ctx, "PUT", id); err != nil {
		_ = cap.Wait()
		return
	}
	fetch := exec.CommandContext(captureCtx, "docker", "exec", container, "wget", "--timeout=3", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "--header=X-E2E-Probe: "+id, "-qO-", r.lab.releases+"/healthz")
	body, fetchErr := fetch.Output()
	_ = cap.Wait()
	note("revoked_capture_node_ingress", output.String())
	origin, _ := r.originProbe(ctx, "GET", id)
	note("revoked_origin_receipt", origin)
	var arrival struct {
		Requests int `json:"requests"`
	}
	decoded := json.Unmarshal([]byte(origin), &arrival) == nil
	after, ok := second.admissionBytes()
	raw, _ = r.lab.docker(r.lab.node, "nft", "-j", "list", "table", "inet", "fvpn")
	end, _ := enforce.ParseState(raw)
	note("revoked_bytes_before", fmt.Sprint(frozen))
	note("revoked_bytes_after", fmt.Sprint(after))
	note("ctunku_packets_before", fmt.Sprint(st.Counters["ctunku"].Packets))
	note("ctunku_packets_after", fmt.Sprint(end.Counters["ctunku"].Packets))
	ct, ctErr := r.lab.docker(r.lab.node, "conntrack", "-L", "-s", source, "-d", r.originIP(), "-o", "extended")
	note("revoked_conntrack_after", strings.TrimSpace(string(ct)))
	packets := regexp.MustCompile(`(?m)^([0-9]+\.[0-9]+) IP `).FindStringSubmatch(output.String())
	if len(packets) == 2 {
		seconds, _ := strconv.ParseFloat(packets[1], 64)
		note("delete_to_first_observed_drop_ms", fmt.Sprintf("%.3f", seconds*1000-float64(start.UnixNano())/1e6))
		note("timing_scope", "first captured post-apply drop, not an upper-bound stop SLA or pass criterion")
	}
	siblingOK := r.countedFlow(ctx, label("sibling_after"))
	note("member_accept_after", itoa(r.lab.forwardChain().MemberAccept))
	ctEmpty := ctErr == nil && !strings.Contains(string(ct), "src="+source)
	blocked := fetchErr != nil && len(body) == 0 && decoded && arrival.Requests == 0 && len(packets) == 2 && end.Counters["ctunku"].Packets > st.Counters["ctunku"].Packets && ok && after == frozen && ctEmpty
	note("revoked_flow_dropped", boolText(blocked))
	note("sibling_counted", boolText(siblingOK))
	// Reuse the freed slot through another real native login, never projection
	// repair. This also exercises serving apply after conntrack withdrawal.
	replacement, replaceErr := s.StartNativePeer(ctx, "replacement")
	replacementOK := false
	if replaceErr == nil {
		p, e := s.JoinNativeMember(ctx, member, "e2e-member", replacement, replacement)
		if e == nil {
			replaced := *r
			otherLab := *r.lab
			replaced.lab = &otherLab
			replaced.lab.member = replacement
			replaced.usageDevice = p.DeviceID
			replacementOK = replaced.countedFlow(ctx, label("replacement"))
		} else {
			note("replacement_error", e.Error())
		}
	} else {
		note("replacement_error", replaceErr.Error())
	}
	note("replacement_counted", boolText(replacementOK))
	pidAfter, _ := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/agent.pid")
	same := bytes.Equal(pid, pidAfter)
	note("same_agent_pid", boolText(same))
	note("member_accept_after_replacement", itoa(r.lab.forwardChain().MemberAccept))
	if !blocked && !strings.Contains(string(routeAfter), "dev fvnb0") {
		note("gate", "native exit route withdrawn before post-delete request: zero node ingress/drop increments; counter freeze, no origin receipt and sibling/replacement success are partial evidence only")
		note("delete_to_first_observed_drop_ms", "not observed; route withdrawal stopped traffic before node ingress")
	}
	if blocked && siblingOK && same && replacementOK {
		rec.Status = StatusPass
	}
}

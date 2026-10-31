//go:build e2e

package enrollment

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Business setup uses only APIs and the shared native identity fixture. A
// catalogue or a 202 is insufficient: the positive control below requires a
// populated signed serving lease, a fresh serving heartbeat and live grants.
func (p *probe) prepareFreezeSource(node uuid.UUID) bool {
	detail := map[string]any{"slot": "node", "installer": false, "node_id": node}
	ok := false
	defer func() { p.expect("AT07", "serving_freeze_source", ok, detail) }()
	policy := func(kind string, payload any) uuid.UUID {
		id, _, err := p.stack.ActivatePolicy(p.ctx, p.admin, kind, "enrollment-freeze-"+kind, payload)
		if err != nil {
			detail["failed_policy"] = kind
			return uuid.Nil
		}
		return id
	}
	resp, err := p.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "enrollment-freeze", Name: "TEST ONLY enrollment freeze"})
	var group api.MemberGroup
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &group) != nil {
		detail["step"] = "group"
		return false
	}
	raw, err := os.ReadFile("contracts/examples/policies/fair-use.json")
	if err != nil {
		return false
	}
	var fair map[string]any
	if json.Unmarshal(raw, &fair) != nil {
		return false
	}
	fair["overrides"] = []any{}
	now := time.Now().UTC()
	fair["effective_at"] = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	fup := policy("fair_use", fair)
	rules := policy("domain_rule_set", map[string]any{"rules": []any{map[string]any{"rule_id": uuid.New(), "domain": "feed-01.example.invalid", "include_subdomains": false, "action": "allow", "profile_id": nil, "reason": "admin_rule", "revision": "1"}}})
	profile := policy("dns_profile", map[string]any{"profile_code": "enrollment-freeze", "display_name": map[string]string{"zh-TW": "TEST ONLY"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{rules}, "threat_feed_revision_ids": []string{}})
	plan := policy("plan", map[string]any{"display_name": map[string]string{"zh-TW": "TEST ONLY"}, "device_limit": 3, "exit_group_ids": []uuid.UUID{group.GroupId}, "price": map[string]any{"amount_minor": 1000, "currency": "USD", "billing_interval": "month", "provider": "stripe", "provider_price_ref": "e2e-price"}, "rate_limit": map[string]any{"down_bps": 100000000, "up_bps": 100000000}, "fair_use_revision_id": fup, "dns_profile_revision_id": profile})
	if fup == uuid.Nil || rules == uuid.Nil || profile == uuid.Nil || plan == uuid.Nil {
		return false
	}
	path := "/api/v1/admin/nodes/" + node.String()
	mutate := func(method, suffix string, body any) bool {
		get, e := p.call("GET", path, nil)
		if e != nil || get.Status != 200 {
			return false
		}
		r, e := p.callHeaders(method, path+suffix, body, map[string]string{"If-Match": get.Header.Get("ETag")})
		detail[method+suffix] = safeResponse(r, e)
		return e == nil && (r.Status == 200 || r.Status == 202)
	}
	if !mutate("PATCH", "", map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}) {
		return false
	}
	// Ready is a real accepted agent projection, never seeded by this harness.
	until := time.Now().Add(90 * time.Second)
	for time.Now().Before(until) && p.ctx.Err() == nil {
		v := p.identityProjection(node)
		if v["provisioning_state"] == "ready" && v["safety_status"] == "passed" {
			break
		}
		if !waitRetry(p.ctx, "2") {
			return false
		}
	}
	if !mutate("PUT", "/publication", api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, GroupIds: &[]uuid.UUID{group.GroupId}, Reason: "TEST ONLY serving freeze"}) {
		return false
	}
	member, err := p.stack.Login(p.ctx, "e2e-member", "member")
	if err != nil {
		detail["step"] = "member_login"
		return false
	}
	r, err := p.call("POST", "/api/v1/admin/members/"+member.MemberID+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY serving freeze", PlanRevisionId: &plan, ValidUntil: time.Now().UTC().Add(time.Hour)})
	detail["finite_grant"] = safeResponse(r, err)
	if err != nil || r.Status != 202 {
		return false
	}
	peer, err := p.stack.JoinNativeMember(p.ctx, member, "e2e-member", "fvpn-w4-enroll-member", "fvpn-member-e2e")
	detail["native_identity_matched"], detail["native_authorized"], detail["native_log_scan"] = peer.IdentityMatched, peer.Authorized, peer.Stream
	if err != nil || !peer.IdentityMatched || !peer.Authorized {
		detail["step"] = "native_join"
		return false
	}
	until = time.Now().Add(120 * time.Second)
	for time.Now().Before(until) && p.ctx.Err() == nil {
		snap := p.freezeSnapshot(node)
		detail["positive_control"] = snap
		if snap.serving() {
			ok = true
			break
		}
		if !waitRetry(p.ctx, "2") {
			break
		}
	}
	return ok
}

type freezeSample struct {
	Start            time.Time  `json:"read_start"`
	End              time.Time  `json:"read_end"`
	NodeRead         bool       `json:"node_read_ok"`
	CentralRead      bool       `json:"central_read_ok"`
	HeartbeatWall    time.Time  `json:"heartbeat_wall"`
	HeartbeatServing bool       `json:"heartbeat_serving"`
	Reason           string     `json:"heartbeat_reason"`
	ForwardDrop      bool       `json:"forward_policy_drop"`
	MemberAccept     int        `json:"member_accept_rules"`
	Frozen           bool       `json:"mapping_frozen"`
	ConflictAt       *time.Time `json:"conflict_at"`
	LeaseID          string     `json:"lease_id"`
	LeaseMembers     int        `json:"lease_members"`
	LeaseExpires     time.Time  `json:"lease_expires"`
	ReportAt         time.Time  `json:"report_received_at"`
	ReportForwarding bool       `json:"report_forwarding"`
	ReportFrozen     bool       `json:"report_identity_frozen"`
}

func (s freezeSample) serving() bool {
	return s.NodeRead && s.CentralRead && !s.Frozen && s.HeartbeatServing && s.ForwardDrop && s.MemberAccept > 0 && s.LeaseMembers > 0 && s.LeaseExpires.After(s.End) && s.End.Sub(s.HeartbeatWall) < 10*time.Second
}
func (s freezeSample) withdrawn() bool {
	return s.NodeRead && s.CentralRead && s.Frozen && !s.HeartbeatServing && s.Reason == "identity_frozen" && s.ForwardDrop && s.MemberAccept == 0
}

// Query only this disposable prefix's PostgreSQL. No shared DSN or mutation.
func (p *probe) freezeSQL(query string, out any) bool {
	raw, err := exec.CommandContext(p.ctx, "docker", "exec", "fvpn-w4-enroll-postgres", "psql", "-X", "-U", "fvpn", "-d", "fvpn", "-At", "-c", query).Output()
	return err == nil && json.Unmarshal(bytes.TrimSpace(raw), out) == nil
}
func (p *probe) freezeSnapshot(node uuid.UUID) freezeSample {
	s := freezeSample{Start: time.Now().UTC()}
	// Only allowlisted heartbeat fields and rule counts leave the private read.
	raw, err := exec.CommandContext(p.ctx, "docker", "exec", "fvpn-w4-enroll-node", "sh", "-c", "cat /run/vpn-node/heartbeat && nft list chain inet fvpn forward").Output()
	s.NodeRead = err == nil
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "wall_unix_ns":
			n, e := strconv.ParseInt(value, 10, 64)
			if e == nil {
				s.HeartbeatWall = time.Unix(0, n).UTC()
			}
		case "serving":
			s.HeartbeatServing = value == "1"
		case "reason":
			s.Reason = value
		}
		if strings.Contains(line, "hook forward") && strings.Contains(line, "policy drop") {
			s.ForwardDrop = true
		}
		if strings.Contains(line, `"fvmb0"`) && strings.Contains(line, "accept") && !strings.Contains(line, "drop") && !strings.Contains(line, "policy ") {
			s.MemberAccept++
		}
	}
	query := fmt.Sprintf(`SELECT jsonb_build_object(
      'mapping_frozen', EXISTS(SELECT 1 FROM node_identity_endpoints WHERE incarnation_id=n.current_incarnation_id AND mapping_frozen),
      'conflict_at',(SELECT min(observed_at) FROM node_identity_conflicts WHERE incarnation_id=n.current_incarnation_id),
      'lease_id',l.id,'lease_members',jsonb_array_length(COALESCE(l.envelope->'signed'->'members','[]')),
      'lease_expires',l.expires_at,'report_received_at',o.received_at,
      'report_forwarding',o.report_payload->'member_forwarding_enabled',
      'report_identity_frozen',COALESCE(o.report_payload->'errors','[]') @> '[{"message":"identity_frozen"}]')
      FROM nodes n LEFT JOIN LATERAL (SELECT * FROM serving_leases WHERE node_id=n.id ORDER BY issued_at DESC LIMIT 1) l ON true
      LEFT JOIN LATERAL (SELECT * FROM node_observations WHERE node_id=n.id ORDER BY received_at DESC LIMIT 1) o ON true
      WHERE n.id='%s'`, node)
	s.CentralRead = p.freezeSQL(query, &s)
	s.End = time.Now().UTC()
	return s
}

type freezeMonitor struct {
	stop      context.CancelFunc
	done      chan struct{}
	samples   []freezeSample
	baseline  freezeSample
	heartbeat time.Duration
}

func (p *probe) startFreezeMonitor(node uuid.UUID) *freezeMonitor {
	raw := dockerText(p.ctx, "exec", "fvpn-w4-enroll-node", "cat", "/etc/vpn-node/agent-run.json")
	var cfg struct {
		Heartbeat int `json:"heartbeat_seconds"`
	}
	if json.Unmarshal([]byte(raw), &cfg) != nil || cfg.Heartbeat <= 0 {
		return nil
	}
	m := &freezeMonitor{baseline: p.freezeSnapshot(node), heartbeat: time.Duration(cfg.Heartbeat) * time.Second, done: make(chan struct{})}
	if !m.baseline.serving() {
		return nil
	}
	ctx, cancel := context.WithCancel(p.ctx)
	m.stop = cancel
	go func() {
		defer close(m.done)
		for {
			m.samples = append(m.samples, p.freezeSnapshot(node))
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	}()
	return m
}
func (m *freezeMonitor) finish() { m.stop(); <-m.done }

func (p *probe) checkServingFreeze(node uuid.UUID, m *freezeMonitor) {
	if m == nil {
		for _, name := range []string{"serving_freeze_withdrawal", "serving_freeze_restart"} {
			p.note("AT07", name, "blocked", map[string]any{"gap": "Serving primary source prerequisite not observed; clone projection alone cannot prove live-grant withdrawal"})
		}
		return
	}
	m.finish()
	detail := map[string]any{"source_slot": "node", "source_installer": false, "baseline": m.baseline, "samples": m.samples, "heartbeat_ms": m.heartbeat.Milliseconds(),
		"timing_bound": "from central conflict transaction (before any identity_frozen=true receipt) to completed live-rule read; conservative upper bound, no inferred receipt timestamp"}
	var first freezeSample
	for _, sample := range m.samples {
		if sample.withdrawn() {
			first = sample
			break
		}
	}
	bounded := false
	if first.ConflictAt != nil {
		elapsed := first.End.Sub(*first.ConflictAt)
		detail["withdrawal_upper_bound_ms"] = elapsed.Milliseconds()
		bounded = elapsed >= 0 && elapsed <= m.heartbeat
	}
	// Replay the actual latest product report; no fabricated observation. The
	// server annotates stored receipts with the current identity_frozen flag.
	var report json.RawMessage
	reportOK := p.freezeSQL(fmt.Sprintf("SELECT report_payload FROM node_observations WHERE node_id='%s' ORDER BY received_at DESC LIMIT 1", node), &report)
	receiptOK := false
	if reportOK {
		cmd := exec.CommandContext(p.ctx, "docker", "exec", "-i", "fvpn-w4-enroll-node", "wget", "-q", "-O", "-", "--content-on-error", "--timeout=10", "--ca-certificate=/etc/vpn-node/ca.pem", "--certificate=/var/lib/vpn-node/identity/cert.pem", "--private-key=/var/lib/vpn-node/identity/key.pem", "--header=Content-Type: application/json", "--header=Idempotency-Key: "+uuid.NewString(), "--post-data="+string(report), "https://127.0.0.1:9444/api/v1/agent/observations")
		raw, err := cmd.Output()
		var receipt api.ObservationAccepted
		receiptOK = err == nil && json.Unmarshal(raw, &receipt) == nil && receipt.IdentityFrozen != nil && *receipt.IdentityFrozen
		detail["receipt_identity_frozen_true"] = receiptOK
		detail["receipt_report_id"], detail["receipt_received_at"] = receipt.ReportId, receipt.ReceivedAt
	}
	p.expect("AT07", "serving_freeze_withdrawal", bounded && receiptOK, detail)
	// Stop through the supervisor before copying the SQLite/WAL pair, then
	// inspect the durable latch read-only on disk. Never change identity state.
	stopped := p.stack.Service(p.ctx, "stop", "") == nil
	latched := false
	dir, err := os.MkdirTemp(os.Getenv("TMPDIR"), "freeze-latch-")
	if err == nil {
		defer os.RemoveAll(dir)
		for _, name := range []string{"agent.db", "agent.db-wal", "agent.db-shm"} {
			_ = exec.CommandContext(p.ctx, "docker", "cp", "fvpn-w4-enroll-node:/var/lib/vpn-node/state/"+name, filepath.Join(dir, name)).Run()
		}
		db, e := sql.Open("sqlite", "file:"+filepath.Join(dir, "agent.db")+"?mode=ro")
		if e == nil {
			var value string
			e = db.QueryRow("SELECT value FROM meta WHERE key='identity_frozen'").Scan(&value)
			latched = e == nil && value == "1"
			_ = db.Close()
		}
	}
	restartAt := time.Now().UTC()
	started := p.stack.Service(p.ctx, "start", "/usr/local") == nil
	var after freezeSample
	until := time.Now().Add(30 * time.Second)
	for time.Now().Before(until) {
		after = p.freezeSnapshot(node)
		if after.withdrawn() && after.HeartbeatWall.After(restartAt) && after.ReportAt.After(restartAt) && after.ReportFrozen && after.LeaseMembers == 0 {
			break
		}
		if !waitRetry(p.ctx, "1") {
			break
		}
	}
	p.expect("AT07", "serving_freeze_restart", stopped && latched && started && after.withdrawn() && after.HeartbeatWall.After(restartAt) && after.ReportAt.After(restartAt) && after.ReportFrozen && after.LeaseMembers == 0,
		map[string]any{"stop_ok": stopped, "durable_identity_frozen": latched, "start_ok": started, "restart_at": restartAt, "after": after, "scope": "agent process restart; fresh heartbeat and accepted observation, persisted SQLite latch, zero live member grants"})
}

//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/google/uuid"
)

func TestE2E_Followup18Native(t *testing.T) {
	if os.Getenv("FVPN_E2E_NATIVE18") != "1" {
		t.Skip("set FVPN_E2E_NATIVE18=1 with a fresh owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	head, _ := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), installerCampaignDirectory(), id)}
	defer r.finish()
	s := &Stack{lab: l}
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	must(s.PrepareNativeUnderlay(ctx))
	must(s.InstallFixtureProbe(ctx))
	must(s.StartFixtureTargets(ctx))
	rev, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "native18-rules", map[string]any{"rules": []any{map[string]any{"rule_id": uuid.New(), "domain": "feed-01.example.invalid", "include_subdomains": true, "action": "block", "profile_id": nil, "reason": "admin_rule", "revision": "1"}}})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "native18-profile", map[string]any{"profile_code": "native18", "display_name": map[string]string{"zh-TW": "TEST ONLY native18"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{rev}, "threat_feed_revision_ids": []string{}})
	must(err)
	r.catalogue(ctx, admin)
	_, err = PrepareServingBaseline(ctx, admin.http, admin.csrf, false)
	must(err)
	if r.enroll(ctx, admin) == uuid.Nil {
		return
	}
	r.desired(ctx)
	r.publish(ctx, admin, r.nodeID)
	r.fixtureServingMember(ctx, admin)
	if r.flowCounted {
		if os.Getenv("FVPN_E2E_FOLLOWUP19") == "1" {
			r.nativeMTU19(ctx, s)
		} else {
			r.nativeMTU18(ctx, s)
		}
	} else {
		r.record(Record{ID: "native-mtu-comparison", Title: "MTU comparison requires counted path", Status: StatusBlocked})
	}
	if os.Getenv("FVPN_E2E_FOLLOWUP19") == "1" {
		return
	}
	r.firstOffer18(ctx, s, admin)
	// Reproduce members F8-01 on a second exit without the primary's
	// deliberately added private-target dummy interface.
	must(s.StartSecondExit(ctx))
	second, err := s.EnrollAt(ctx, b, "TEST ONLY F8-01 second exit", "node2")
	must(err)
	selected, err := s.MemberTransportStack(l.member, "node2")
	must(err)
	r.lab, r.nodeID, r.incID = selected.lab, second.NodeID, second.IncarnationID
	originalDir := r.dir
	r.dir = filepath.Join(originalDir, "node2")
	r.publish(ctx, admin, second.NodeID)
	r.dir = originalDir
	pre := Record{ID: "second-exit-counted", Title: "Serving second exit before namespace fault", Spec: []string{"AT09", "F8-01"}, Status: StatusFail}
	if r.selectCurrentExit18(ctx, recPathNote(&pre)) && r.tracedFetch(ctx, recPathNote(&pre)) {
		pre.Status = StatusPass
	}
	r.record(pre)
	rec := Record{ID: "namespace-restart-90", Title: "F8-01 unchanged-generation namespace restart, full first 90 seconds", Spec: []string{"AT08", "AT09", "F8-01"}, Status: StatusFail}
	got, err := s.RestartNodeNamespace90(ctx, "node2", r.nodeID)
	noteJSON(&rec, "restart", got)
	if err != nil {
		recPathNote(&rec)("error", err.Error())
	}
	if err == nil && got.GenerationBefore == got.GenerationAfter && got.ServingAt != nil {
		rec.Status = StatusPass
	}
	recPathNote(&rec)("owner", "agent-core follow-up 25; second-exit namespace reproduction; no generation change or repair")
	r.record(rec)
}

// This supplement uses the retained enrolled primary from either campaign. It does
// not enroll another identity or replace its store to manufacture acceptance.
func TestE2E_Followup18FirstOffer(t *testing.T) {
	if os.Getenv("FVPN_E2E_FIRST_OFFER18") != "1" {
		t.Skip("set FVPN_E2E_FIRST_OFFER18=1 with a retained enrolled primary")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	head, _ := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), installerCampaignDirectory(), id)}
	defer r.finish()
	raw, err := l.docker(l.node, "cat", "/var/lib/vpn-node/identity/node.json")
	var identity struct {
		NodeID uuid.UUID `json:"node_id"`
	}
	if err != nil || json.Unmarshal(raw, &identity) != nil || identity.NodeID == uuid.Nil {
		t.Fatal("retained primary identity unavailable")
	}
	r.nodeID = identity.NodeID
	if admin := r.login(ctx, "e2e-admin", "admin"); admin != nil {
		r.firstOffer18(ctx, &Stack{lab: l}, admin)
	}
}

func (r *runner) firstOffer18(ctx context.Context, s *Stack, admin *session) {
	rec := Record{ID: "first-desired-offer", Title: "SECOPS-21 actual response, local accepted digest and applied receipt", Spec: []string{"SECOPS-21", "§14.7"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	selected, err := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/agent-selected")
	if err != nil {
		recPathNote(&rec)("selected_release", err.Error())
		return
	}
	release := strings.TrimSpace(string(selected))
	// Quiet the agent before arming so no pre-hold GET can be mistaken for first.
	err = s.Service(ctx, "stop", "")
	if err != nil {
		recPathNote(&rec)("stop", err.Error())
		return
	}
	defer func() {
		_ = s.ReleaseDesiredOffer(context.WithoutCancel(ctx), "node")
		_ = s.Service(context.WithoutCancel(ctx), "start", release)
	}()
	id, err := s.ArmFirstDesiredOffer(ctx, "node")
	noteJSON(&rec, "capture_id", id)
	if err != nil {
		recPathNote(&rec)("arm_error", err.Error())
		return
	}
	// Advance desired state through the real command path while the retained
	// agent is stopped and held. An unchanged ETag correctly returns only 304.
	db, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer db.Close(ctx)
	var retained string
	if db.QueryRow(ctx, `SELECT generation::text FROM desired_state_envelopes WHERE node_id=$1`, r.nodeID).Scan(&retained) != nil {
		return
	}
	detail, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
	if err != nil || detail.Status != 200 {
		return
	}
	var current struct {
		Capacity struct {
			Down uint64 `json:"down_bps"`
		} `json:"capacity"`
	}
	if json.Unmarshal(detail.Body, &current) != nil {
		return
	}
	parentRate := uint64(210000000)
	if current.Capacity.Down == parentRate {
		parentRate = 220000000
	}
	capacity := map[string]any{"capacity": map[string]any{"down_bps": parentRate, "up_bps": parentRate, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}
	patched, err := admin.call(ctx, "PATCH", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), capacity, map[string]string{"If-Match": detail.Header.Get("ETag"), "Idempotency-Key": uuid.NewString()})
	recPathNote(&rec)("configuration_command", httpDetail(patched, err))
	recPathNote(&rec)("retained_generation", retained)
	if err != nil || (patched.Status != 200 && patched.Status != 202) {
		return
	}
	changed, stopChanged := context.WithTimeout(ctx, 30*time.Second)
	ready := waitUntil(changed, time.Second, func() bool {
		// Generation advancement also happens on the agent GET. Read back the
		// requested capacity here, then prove delivery/application separately.
		got, e := admin.call(changed, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
		return e == nil && got.Status == 200 && json.Unmarshal(got.Body, &current) == nil && current.Capacity.Down == parentRate
	})
	stopChanged()
	if !ready {
		recPathNote(&rec)("command", "requested capacity not read back")
		return
	}
	if err = s.Service(ctx, "start", release); err != nil {
		recPathNote(&rec)("resume_error", err.Error())
		return
	}
	if err = s.ReleaseDesiredOffer(ctx, "node"); err != nil {
		recPathNote(&rec)("release_error", err.Error())
		return
	}
	wait, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	var got FirstOfferEvidence
	ok := waitUntil(wait, time.Second, func() bool {
		got, err = s.FirstDesiredOffer(wait, "node")
		return err == nil && got.CaptureID == id && got.First != nil && got.First.HTTPStatus == 200 && got.First.WriteComplete && got.Local != nil && got.ACK != nil
	})
	noteJSON(&rec, "capture", got)
	recPathNote(&rec)("scope", "live hold/release of retained state; full PostgreSQL restore and greater-than-retained assertion belong to secops")
	if ok {
		db, e := r.lab.db(ctx)
		if e == nil {
			defer db.Close(ctx)
			var raw []byte
			e = db.QueryRow(ctx, `SELECT jsonb_build_object('report_id',report_id,'received_at',received_at,'applied_generation',applied_generation::text) FROM node_observations WHERE node_id=$1 AND report_id=$2 AND applied_generation::text=$3`, r.nodeID, got.ACK.ReportID, got.First.Generation).Scan(&raw)
			if e == nil {
				noteJSON(&rec, "central_receipt", json.RawMessage(raw))
				rec.Status = StatusPass
			}
		}
	}
}

// Select the published route belonging to this incarnation when both exits are
// available. The provider and central reads only identify the real API route.
func (r *runner) selectCurrentExit18(ctx context.Context, note func(string, string)) bool {
	db, err := r.lab.db(ctx)
	if err != nil {
		return false
	}
	defer db.Close(ctx)
	var peer string
	if db.QueryRow(ctx, `SELECT peer_id FROM node_incarnations WHERE id=$1`, r.incID).Scan(&peer) != nil {
		return false
	}
	f := &probeFixture{lab: r.lab}
	wait, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	ok := waitUntil(wait, time.Second, func() bool {
		var routes []netbird.Route
		if f.netbirdAPI(wait, "GET", "/routes", nil, &routes) != nil {
			return false
		}
		advertised, _ := r.lab.docker(r.lab.member, "netbird", "routes", "list")
		for _, route := range routes {
			if route.Peer == peer && route.Network == "0.0.0.0/0" && route.Enabled && strings.Contains(string(advertised), route.NetworkID) {
				out, e := r.lab.docker(r.lab.member, "netbird", "routes", "select", route.NetworkID)
				note("selected_exit", strings.TrimSpace(string(out)))
				return e == nil
			}
		}
		return false
	})
	time.Sleep(3 * time.Second)
	return ok
}

func (r *runner) nativeMTU18(ctx context.Context, s *Stack) {
	rec := Record{ID: "native-mtu-comparison", Title: "F8-02 standalone versus reconnect matrix with PMTU and MSS capture", Spec: []string{"AT09", "AT11", "F8-02"}, Status: StatusPass}
	defer func() { r.record(rec) }()
	defer func() {
		_, _ = s.ApplyNativeInputs(context.WithoutCancel(ctx), NativeFixtureInputs{Underlay: "dual", MTU: 1500, Protocol: "tcp", Platform: "linux"})
	}()
	// Original standalone ordering, followed by the consumer's exact matrix.
	rows := []struct {
		name, family string
		mtu          int
		reconnect    bool
	}{{"standalone1500", "dual", 1500, false}, {"standalone1280", "dual", 1280, false}, {"matrix4", "ipv4", 1500, true}, {"matrix6", "ipv6", 1500, true}, {"matrixDual", "dual", 1500, true}, {"matrix1280", "dual", 1280, true}, {"ipv6_1280", "ipv6", 1280, true}, {"ipv4_1280", "ipv4", 1280, true}}
	for _, row := range rows {
		selected, err := s.ApplyNativeInputs(ctx, NativeFixtureInputs{Underlay: row.family, MTU: row.mtu, Protocol: "tcp", Platform: "linux"})
		noteJSON(&rec, row.name+"_input", selected)
		if err != nil {
			rec.Status = StatusFail
			recPathNote(&rec)(row.name+"_error", err.Error())
			continue
		}
		if row.reconnect {
			_, err = r.lab.docker(r.lab.member, "netbird", "down")
			if err == nil {
				_, err = r.lab.docker(r.lab.member, "netbird", "up")
			}
			if err != nil {
				rec.Status = StatusFail
				continue
			}
			wait, cancel := context.WithTimeout(ctx, 60*time.Second)
			_ = waitUntil(wait, time.Second, func() bool {
				p, e := s.ObserveNativeUnderlay(wait)
				chosen, ce := s.ObserveSelectedNativeExit(wait)
				return e == nil && ce == nil && chosen.Connected && p.Path.Direct+p.Path.Relayed > 0
			})
			cancel()
			time.Sleep(3 * time.Second)
		}
		paths := []string{r.lab.member, r.lab.node}
		for _, c := range paths {
			script := `echo $$ > /var/lib/vpn-node/e2e/mtu-capture.pid; exec tcpdump --immediate-mode -U -nn -s 128 -i any -w /var/lib/vpn-node/e2e/mtu.pcap 'icmp or icmp6 or tcp port 9446 or udp' 2>/var/lib/vpn-node/e2e/mtu-capture.log`
			e := exec.CommandContext(ctx, "docker", "exec", "-d", c, "sh", "-c", script).Run()
			if e != nil {
				rec.Status = StatusFail
			}
		}
		time.Sleep(time.Second)
		before, _ := s.DeviceByteCounters(ctx, r.usageDevice)
		path, _ := s.ObserveNativeUnderlay(ctx)
		chosenBefore, cbErr := s.ObserveSelectedNativeExit(ctx)
		got, e := s.fixtureProbe(ctx, r.lab.member, "-mode", "bulk", "-addr", r.lab.originIP+":9446", "-device", r.usageDevice.String())
		after, ae := s.DeviceByteCounters(ctx, r.usageDevice)
		end, _ := s.ObserveNativeUnderlay(ctx)
		chosenAfter, caErr := s.ObserveSelectedNativeExit(ctx)
		noteJSON(&rec, row.name+"_transfer", map[string]any{"before": before, "after": after, "path_before": path, "path_after": end, "selected_before": chosenBefore, "selected_after": chosenAfter, "bulk": got})
		if e != nil || ae != nil || cbErr != nil || caErr != nil || !chosenBefore.Connected || !chosenAfter.Connected || !got.Available || got.Bytes != 64064 || !after.Present || after.Upload <= before.Upload || after.Download <= before.Download {
			rec.Status = StatusFail
		}
		for i, c := range paths {
			_, _ = r.lab.docker(c, "sh", "-c", `kill -INT "$(cat /var/lib/vpn-node/e2e/mtu-capture.pid)"`)
			flush, stop := context.WithTimeout(ctx, 3*time.Second)
			_ = waitUntil(flush, 100*time.Millisecond, func() bool {
				_, e := r.lab.docker(c, "sh", "-c", `kill -0 "$(cat /var/lib/vpn-node/e2e/mtu-capture.pid)"`)
				return e != nil
			})
			stop()
			stats, _ := r.lab.docker(c, "cat", "/var/lib/vpn-node/e2e/mtu-capture.log")
			recPathNote(&rec)(fmt.Sprintf("%s_capture_%d_stats", row.name, i), string(stats))
			// TCP SYN metadata contains MSS; ICMP summaries contain advertised MTU.
			metadata, e := r.lab.docker(c, "tcpdump", "-nn", "-tt", "-v", "-r", "/var/lib/vpn-node/e2e/mtu.pcap", "icmp or icmp6 or (tcp port 9446 and tcp[tcpflags] & tcp-syn != 0)")
			recPathNote(&rec)(fmt.Sprintf("%s_capture_%d_headers", row.name, i), string(metadata))
			if e != nil {
				rec.Status = StatusFail
			}
			readback, _ := r.lab.docker(c, "sh", "-c", `ip -j link; ip -j route; ip -j -6 route; ss -tin`)
			recPathNote(&rec)(fmt.Sprintf("%s_path_%d", row.name, i), string(readback))
			slot := "node"
			if i == 0 {
				slot = "member"
			}
			receipt, exportErr := s.ExportContainerFile(ctx, slot, "/var/lib/vpn-node/e2e/mtu.pcap", filepath.Join(r.lab.root, ".agent/tmp", r.runID+"-"+row.name+"-"+slot+".pcap"))
			noteJSON(&rec, fmt.Sprintf("%s_capture_%d_receipt", row.name, i), receipt)
			if exportErr != nil {
				rec.Status = StatusFail
			}
		}
	}
}

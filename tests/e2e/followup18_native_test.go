//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup18", id)}
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
		r.nativeMTU18(ctx, s)
	} else {
		r.record(Record{ID: "native-mtu-comparison", Title: "MTU comparison requires counted path", Status: StatusBlocked})
	}
	r.firstOffer18(ctx, s)
	rec := Record{ID: "namespace-restart-90", Title: "F8-01 unchanged-generation namespace restart, full first 90 seconds", Spec: []string{"AT08", "AT09", "F8-01"}, Status: StatusFail}
	got, err := s.RestartNodeNamespace90(ctx, "node", r.nodeID)
	noteJSON(&rec, "restart", got)
	if err != nil {
		recPathNote(&rec)("error", err.Error())
	}
	if err == nil && got.GenerationBefore == got.GenerationAfter && got.ServingAt != nil {
		rec.Status = StatusPass
	}
	recPathNote(&rec)("owner", "agent-core follow-up 25; primary namespace reproduction, export also supports node2")
	r.record(rec)
}

func (r *runner) firstOffer18(ctx context.Context, s *Stack) {
	rec := Record{ID: "first-desired-offer", Title: "SECOPS-21 actual response, local accepted digest and applied receipt", Spec: []string{"SECOPS-21", "§14.7"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	// Quiet the agent before arming so no pre-hold GET can be mistaken for first.
	_, err := r.lab.docker(r.lab.node, "/usr/local/bin/agent-service.sh", "stop")
	if err != nil {
		recPathNote(&rec)("stop", err.Error())
		return
	}
	id, err := s.ArmFirstDesiredOffer(ctx, "node")
	noteJSON(&rec, "capture_id", id)
	_, resume := r.lab.docker(r.lab.node, "/usr/local/bin/agent-service.sh", "start")
	if err != nil || resume != nil {
		return
	}
	if err = s.ReleaseDesiredOffer(ctx, "node"); err != nil {
		return
	}
	wait, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	var got FirstOfferEvidence
	ok := waitUntil(wait, time.Second, func() bool {
		got, err = s.FirstDesiredOffer(wait, "node")
		return err == nil && got.CaptureID == id && got.First != nil && got.First.WriteComplete && got.Local != nil && got.ACK != nil
	})
	noteJSON(&rec, "capture", got)
	recPathNote(&rec)("scope", "live hold/release of retained state; full PostgreSQL restore and greater-than-retained assertion belong to secops")
	if ok {
		rec.Status = StatusPass
	}
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
				return e == nil && p.Path.Direct+p.Path.Relayed > 0
			})
			cancel()
			time.Sleep(3 * time.Second)
		}
		paths := []string{r.lab.member, r.lab.node}
		for _, c := range paths {
			script := `echo $$ > /var/lib/vpn-node/e2e/mtu-capture.pid; exec tcpdump -U -nn -s 128 -i any -w /var/lib/vpn-node/e2e/mtu.pcap 'icmp or icmp6 or tcp port 9446 or udp'`
			e := exec.CommandContext(ctx, "docker", "exec", "-d", c, "sh", "-c", script).Run()
			if e != nil {
				rec.Status = StatusFail
			}
		}
		time.Sleep(time.Second)
		before, _ := s.DeviceByteCounters(ctx, r.usageDevice)
		path, _ := s.ObserveNativeUnderlay(ctx)
		got, e := s.fixtureProbe(ctx, r.lab.member, "-mode", "bulk", "-addr", r.lab.originIP+":9446", "-device", r.usageDevice.String())
		after, ae := s.DeviceByteCounters(ctx, r.usageDevice)
		end, _ := s.ObserveNativeUnderlay(ctx)
		noteJSON(&rec, row.name+"_transfer", map[string]any{"before": before, "after": after, "path_before": path, "path_after": end, "bulk": got})
		if e != nil || ae != nil || !got.Available || got.Bytes != 64064 || !after.Present || after.Upload <= before.Upload || after.Download <= before.Download {
			rec.Status = StatusFail
		}
		for i, c := range paths {
			_, _ = r.lab.docker(c, "sh", "-c", `kill -INT "$(cat /var/lib/vpn-node/e2e/mtu-capture.pid)"`)
			time.Sleep(200 * time.Millisecond)
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

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

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// One assembled fixture campaign, deliberately excluding the ENR-06 wrappers.
func TestE2E_Followup16Fixtures(t *testing.T) {
	if os.Getenv("FVPN_E2E_FIXTURES16") != "1" {
		t.Skip("set FVPN_E2E_FIXTURES16=1 with a fresh two-resolver owned stack")
	}
	runFixtureCampaign(t, false)
}

func runFixtureCampaign(t *testing.T, dnsOnly bool) {
	t.Helper()
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 35*time.Minute)
	defer cancel()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	runID := uuid.NewString()
	r := &runner{t: t, lab: l, runID: runID, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), fixtureCampaignDirectory(), runID)}
	defer r.finish()
	s := &Stack{lab: l}
	if err = s.SetDNSFixturePhase("initial"); err != nil {
		t.Fatal(err)
	}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	if err = s.PrepareNativeUnderlay(ctx); err != nil {
		r.record(Record{ID: "fixture-setup", Title: "Pre-enrollment underlay setup", Spec: []string{"AT11"}, Status: StatusFail, Observations: []Observation{{Name: "error", Detail: err.Error()}}})
		return
	}
	if err = s.InstallFixtureProbe(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.StartFixtureTargets(ctx); err != nil {
		t.Fatal(err)
	}
	rules := []any{}
	for _, in := range []struct{ name, action string }{{"feed-01.example.invalid", "block"}, {"blocked-cname.example.invalid", "block"}, {"rebind.example.invalid", "allow"}} {
		rules = append(rules, map[string]any{"rule_id": uuid.New(), "domain": in.name, "include_subdomains": true, "action": in.action, "profile_id": nil, "reason": "admin_rule", "revision": "1"})
	}
	rev, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "fixture-negative-rules", map[string]any{"rules": rules})
	if err != nil {
		t.Fatal(err)
	}
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "fixture-negative-profile", map[string]any{"profile_code": "fixture-negative", "display_name": map[string]string{"zh-TW": "TEST ONLY DNS matrix"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{rev}, "threat_feed_revision_ids": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	r.catalogue(ctx, admin)
	baseline, err := PrepareServingBaseline(ctx, admin.http, admin.csrf, false)
	if err != nil {
		t.Fatal(err)
	}
	rec := Record{ID: "fixture-baseline", Title: "API-created signed serving baseline", Spec: []string{"§13.2"}, Status: StatusPass}
	noteJSON(&rec, "baseline", baseline)
	r.record(rec)
	retained, retainErr := s.RetainEnrollingThirdNode(ctx, b)
	rec = Record{ID: "fleet-retained", Title: "Third pre-existing enrolling identity retained before bootstrap", Spec: []string{"§6.2", "AT24", "AT26", "D-089"}, Status: StatusPass}
	noteJSON(&rec, "snapshot", retained)
	if retainErr != nil {
		rec.Status = StatusFail
		recPathNote(&rec)("error", retainErr.Error())
	}
	r.record(rec)
	node := r.enroll(ctx, admin)
	if node == uuid.Nil {
		return
	}
	r.desired(ctx)
	r.publish(ctx, admin, node)
	r.fixtureServingMember(ctx, admin)
	var second EnrolledNode
	if err = s.StartSecondExit(ctx); err == nil {
		second, err = s.EnrollAt(ctx, b, "TEST ONLY second filtering resolver", "node2")
	}
	rec = Record{ID: "second-resolver", Title: "Independent second real node/filter", Spec: []string{"§12.2", "AT21"}, Status: StatusPass}
	noteJSON(&rec, "identity", second)
	if err != nil {
		rec.Status = StatusFail
		recPathNote(&rec)("error", err.Error())
	} else {
		wait, cancel := context.WithTimeout(ctx, 90*time.Second)
		if !waitUntil(wait, time.Second, func() bool {
			resp, e := s.Call(wait, b, "GET", "/api/v1/admin/nodes/"+second.NodeID.String(), nil, nil)
			var d api.NodeDetail
			if e != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &d) != nil {
				return false
			}
			ev, e := d.DnsPolicyEvidence.Get()
			return e == nil && ev.Outcome == api.DnsPolicyEvidenceOutcomeApplied
		}) {
			rec.Status = StatusFail
			recPathNote(&rec)("error", "second resolver policy application deadline")
		}
		cancel()
	}
	r.record(rec)
	if !dnsOnly {
		r.fixtureUnderlay(ctx, s)
		r.fixtureInputs(ctx, s)
	}
	config, dnsErr := s.ConfigureNativeDNS(ctx, b, r.usageDevice, []uuid.UUID{node, second.NodeID})
	rec = Record{ID: "native-dns-config", Title: "Native client uses the two issued controlled resolvers", Spec: []string{"§12.2", "AT21"}, Status: StatusPass}
	noteJSON(&rec, "configuration", config)
	if dnsErr != nil {
		rec.Status = StatusFail
		recPathNote(&rec)("error", dnsErr.Error())
	}
	r.record(rec)
	if dnsErr == nil {
		if os.Getenv("FVPN_E2E_FOLLOWUP23") == "1" {
			r.nativeDNS23(ctx, s)
			return
		}
		r.fixtureFallback(ctx, s)
		r.fixtureDNSNegative(ctx, s)
		if dnsOnly {
			r.verifyDNSHistory21()
		}
	} else {
		for _, id := range []string{"native-dns-fallback", "dns-negative"} {
			r.record(Record{ID: id, Title: id, Spec: []string{"AT20", "AT21"}, Status: StatusBlocked, Observations: []Observation{{Name: "prerequisite", Detail: "native DNS configuration failed"}}})
		}
	}
	if dnsOnly {
		return
	}
	r.fixtureFiniteGrant(ctx, s, b)
	rec = Record{ID: "fleet-midrun-boot", Title: "Retained third identity resumes and becomes eligible mid-run", Spec: []string{"§6.2", "AT24", "AT26", "D-089"}, Status: StatusBlocked}
	if retainErr == nil {
		booted, e := s.BootEnrollingThirdNode(ctx, b)
		noteJSON(&rec, "snapshot", booted)
		if e != nil {
			rec.Status = StatusFail
			recPathNote(&rec)("error", e.Error())
		} else {
			rec.Status = StatusPass
		}
	}
	r.record(rec)
}
func noteJSON(rec *Record, key string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		recPathNote(rec)(key, "marshal failed")
		return
	}
	recPathNote(rec)(key, string(raw))
}

// Await a real counted path after native exit selection. A single HTTPS timeout
// during ICE negotiation is retained as an attempt, not treated as readiness.
func (r *runner) fixtureServingMember(ctx context.Context, admin *session) {
	rec := Record{ID: "fixture-member-admission", Title: "Native identity, finite grant, selected exit and counted forwarding", Spec: []string{"§8", "AT02", "AT12"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	member, err := r.lab.login(ctx, "e2e-member", "member")
	if err != nil {
		note("error", "member login")
		return
	}
	rev := r.planRevID
	grant, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/members/"+member.memberID+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY native fixtures", ValidUntil: time.Now().UTC().Add(time.Hour), PlanRevisionId: &rev}, map[string]string{"Idempotency-Key": uuid.NewString()})
	note("grant", httpDetail(grant, err))
	if err != nil || grant.Status != 202 {
		return
	}
	sub := r.notePortalSubject(ctx, member.memberID, note)
	var contact *NativeCapture
	if os.Getenv("FVPN_E2E_FOLLOWUP27") == "native" {
		contact, err = (&Stack{lab: r.lab}).StartNativeFirstContactCapture(ctx)
		if err != nil {
			note("first_contact_capture", "unavailable")
			return
		}
		defer func() {
			capture := Record{ID: "native-session", Title: "First contact through sustained stream passive capture", Status: StatusPass}
			r.finishNativeCapture22(ctx, &capture, contact)
			r.record(capture)
		}()
	}
	r.memberNetBird(ctx, member, sub, note)
	if !r.usageReady {
		return
	}
	if os.Getenv("FVPN_E2E_FOLLOWUP27") == "hold" {
		if r.servingGate26(ctx, "admission-serving") {
			rec.Status = StatusPass
		}
		return
	}
	if contact != nil {
		if err = contact.BindPeers(ctx); err != nil {
			return
		}
		if !r.servingGate26(ctx, "admission-serving", contact) {
			return
		}
		r.native27(ctx, &Stack{lab: r.lab}, contact)
		// Admission is independent of the measured first request.
		rec.Status = StatusPass
		return
	}
	if os.Getenv("FVPN_E2E_FOLLOWUP22") == "1" || os.Getenv("FVPN_E2E_FOLLOWUP23") == "1" || os.Getenv("FVPN_E2E_FOLLOWUP24") == "1" {
		if !r.servingGate26(ctx, "admission-serving") {
			note("measurement", "admission failed; no first-selection request launched")
			return
		}
		r.nativeInitial22(ctx, &Stack{lab: r.lab})
	}
	if os.Getenv("FVPN_E2E_FOLLOWUP24") == "1" {
		if r.flowCounted {
			rec.Status = StatusPass
		}
		return
	}
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	attempt := 0
	r.flowCounted = waitUntil(wait, 3*time.Second, func() bool { attempt++; note("counted_attempt", fmt.Sprint(attempt)); return r.countedFlow(wait, note) })
	if r.flowCounted {
		rec.Status = StatusPass
	}
}

func (r *runner) fixtureUnderlay(ctx context.Context, s *Stack) {
	rec := Record{ID: "native-underlay", Title: "Real member/node IPv4, IPv6 and dual underlay selectors", Spec: []string{"§8.3", "AT09", "AT11"}, Status: StatusPass}
	defer func() { r.record(rec) }()
	if !r.flowCounted {
		rec.Status = StatusBlocked
		recPathNote(&rec)("prerequisite", "counted member flow unavailable")
		return
	}
	// Controlled TCP reachability independently proves the new IPv6 link.
	slot := strings.TrimPrefix(s.lab.labPrefix, "172.28.")
	for _, addr := range []string{s.lab.labPrefix + ".40:9447", "[2001:db8:" + slot + "::40]:9447"} {
		got, e := s.fixtureProbe(ctx, s.lab.member, "-mode", "connect", "-addr", addr)
		key := "v4_reachable"
		if strings.Contains(addr, "[") {
			key = "v6_reachable"
		}
		noteJSON(&rec, key, got)
		if e != nil || !got.Connected {
			rec.Status = StatusFail
		}
	}
	for _, family := range []string{"ipv4", "ipv6", "dual"} {
		in := NativeFixtureInputs{Underlay: family, MTU: 1500, Protocol: "tcp", Platform: "linux"}
		selected, e := s.ApplyNativeInputs(ctx, in)
		noteJSON(&rec, family+"_input", selected)
		if e != nil {
			rec.Status = StatusFail
			recPathNote(&rec)(family+"_error", e.Error())
			continue
		}
		_ = exec.CommandContext(ctx, "docker", "exec", s.lab.member, "netbird", "down").Run()
		if exec.CommandContext(ctx, "docker", "exec", s.lab.member, "netbird", "up").Run() != nil {
			rec.Status = StatusFail
			continue
		}
		wait, cancel := context.WithTimeout(ctx, 60*time.Second)
		var path NativeUnderlayObservation
		seen := waitUntil(wait, time.Second, func() bool {
			path, e = s.ObserveNativeUnderlay(wait)
			return e == nil && path.Path.Direct+path.Path.Relayed > 0
		})
		cancel()
		time.Sleep(3 * time.Second)
		path, _ = s.ObserveNativeUnderlay(ctx)
		noteJSON(&rec, family+"_transport", path)
		got, e := s.fixtureProbe(ctx, s.lab.member, "-mode", "bulk", "-addr", s.lab.originIP+":9446", "-device", r.usageDevice.String())
		noteJSON(&rec, family+"_member_bulk", got)
		after, _ := s.ObserveNativeUnderlay(ctx)
		noteJSON(&rec, family+"_transport_after_bulk", after)
		if !seen || e != nil || !got.Available {
			rec.Status = StatusFail
		}
	}
	recPathNote(&rec)("scope", "peer UDP family availability is selected; TCP control retained; NetBird's chosen path is observed, not prescribed; IPv6 overlay and platform leak prevention are unclaimed")
}
func (r *runner) fixtureInputs(ctx context.Context, s *Stack) {
	if os.Getenv("FVPN_E2E_FOLLOWUP20") == "1" {
		r.captureMTU19(ctx, s, true)
		return
	}
	rec := Record{ID: "native-matrix-inputs", Title: "Declared protocol, MTU and real-platform inputs", Spec: []string{"§8.3", "AT09", "AT11"}, Status: StatusPass}
	defer func() { r.record(rec) }()
	for _, platform := range []string{"linux", "android", "ios", "windows", "macos"} {
		for _, proto := range []string{"tcp", "udp", "quic"} {
			selection, e := SelectNativeFixture(NativeFixtureInputs{Underlay: "dual", MTU: 1280, Protocol: proto, Platform: platform})
			noteJSON(&rec, platform+"_"+proto, selection)
			if e != nil {
				rec.Status = StatusFail
			}
		}
	}
	if !r.flowCounted {
		rec.Status = StatusBlocked
		return
	}
	for _, mtu := range []int{1500, 1280} {
		_, e := s.ApplyNativeInputs(ctx, NativeFixtureInputs{Underlay: "dual", MTU: mtu, Protocol: "tcp", Platform: "linux"})
		if e != nil {
			rec.Status = StatusFail
			continue
		}
		before, e := s.DeviceByteCounters(ctx, r.usageDevice)
		if e != nil {
			rec.Status = StatusFail
			continue
		}
		got, e := s.fixtureProbe(ctx, s.lab.member, "-mode", "bulk", "-addr", s.lab.originIP+":9446", "-device", r.usageDevice.String())
		noteJSON(&rec, fmt.Sprintf("mtu_%d_bulk", mtu), got)
		after, _ := s.DeviceByteCounters(ctx, r.usageDevice)
		noteJSON(&rec, fmt.Sprintf("mtu_%d_counters_before", mtu), before)
		noteJSON(&rec, fmt.Sprintf("mtu_%d_counters_after", mtu), after)
		if e != nil || !got.Available || !after.Present || after.Upload <= before.Upload || after.Download <= before.Download {
			rec.Status = StatusFail
		}
	}
	_, _ = s.ApplyNativeInputs(ctx, NativeFixtureInputs{Underlay: "dual", MTU: 1500, Protocol: "tcp", Platform: "linux"})
	recPathNote(&rec)("scope", "selector contract plus bounded TCP transfer at two MTUs; QUIC and non-Linux platforms are explicitly unavailable, not acceptance passes; ICMP black-hole and physical second NAT remain operator inputs")
}
func (r *runner) fixtureFallback(ctx context.Context, s *Stack) {
	rec := Record{ID: "native-dns-fallback", Title: "System resolver fallback, both-down failure and recovery", Spec: []string{"§12.2", "AT21"}, Status: StatusPass}
	defer func() { r.record(rec) }()
	// Stop the origin transport rather than the node filter: DNS policy health
	// remains real, and only the chosen client's path to that resolver is down.
	primary := s.lab.prefix + "-origin-dns"
	secondary := s.lab.prefix + "-origin-dns-secondary"
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = exec.CommandContext(c, "docker", "start", primary, secondary).Run()
	}()
	probe := func(key, name string, want bool) {
		got, e := s.fixtureProbe(ctx, s.lab.member, "-mode", "lookup", "-name", name, "-expect", s.lab.originIP)
		noteJSON(&rec, key, got)
		if e != nil || got.Available != want || want && !got.ExpectedAnswer {
			rec.Status = StatusFail
		}
	}
	probe("healthy", "rr.example.invalid", true)
	down := time.Now().UTC()
	if exec.CommandContext(ctx, "docker", "stop", "-t", "2", primary).Run() != nil {
		rec.Status = StatusFail
		return
	}
	noteJSON(&rec, "primary_down_at", down)
	probe("automatic_secondary", "notfeed-01.example.invalid", true)
	blocked, e := s.ProbeNativeDNS(ctx, "", "feed-01.example.invalid", 1, false, "")
	noteJSON(&rec, "secondary_block_policy", blocked)
	if e != nil || blocked.RCode != 3 {
		rec.Status = StatusFail
	}
	if exec.CommandContext(ctx, "docker", "stop", "-t", "2", secondary).Run() != nil {
		rec.Status = StatusFail
		return
	}
	probe("both_down_fail_closed", "cache.example.invalid", false)
	if exec.CommandContext(ctx, "docker", "start", primary, secondary).Run() != nil {
		rec.Status = StatusFail
		return
	}
	// NetBird backs off dead upstreams. Bound recovery independently and export it.
	start := time.Now()
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var recovered FixtureProbeResult
	ok := waitUntil(wait, 2*time.Second, func() bool {
		recovered, e = s.fixtureProbe(wait, s.lab.member, "-mode", "lookup", "-name", "alias.example.invalid", "-expect", s.lab.originIP)
		return e == nil && recovered.Available && recovered.ExpectedAnswer
	})
	noteJSON(&rec, "recovery", recovered)
	recPathNote(&rec)("recovery_ms", fmt.Sprint(time.Since(start).Milliseconds()))
	if !ok {
		rec.Status = StatusFail
	}
	recPathNote(&rec)("expected", "primary unavailable -> same-policy secondary answers; both unavailable -> resolution fails; recovery answers without public fallback")
}
func (r *runner) fixtureDNSNegative(ctx context.Context, s *Stack) {
	rec := Record{ID: "dns-negative", Title: "Controlled RR types, label boundaries, cache expiry and allowlisted rebinding", Spec: []string{"§12.5", "§12.8", "AT20", "ISO01", "DNS02"}, Status: StatusPass}
	defer func() { r.record(rec) }()
	for _, phase := range []string{"before", "after"} {
		read := func() {
			for _, slot := range []string{s.lab.node, s.lab.prefix + "-node2"} {
				raw, err := s.lab.docker(slot, "/usr/local/bin/fvpn-e2e-nodeconfig", "dns-readback")
				var info map[string]any
				if err != nil || json.Unmarshal(raw, &info) != nil {
					rec.Status = StatusFail
					recPathNote(&rec)("resolver_settings_"+phase, "read failed")
					continue
				}
				noteJSON(&rec, "resolver_settings_"+phase+"_"+strings.TrimPrefix(slot, s.lab.prefix+"-"), info)
			}
		}
		if phase == "before" {
			read()
		} else {
			defer read()
		}
	}
	for _, proto := range []bool{false, true} {
		for _, tc := range []struct {
			name   string
			typ    uint16
			rcode  int
			answer bool
		}{
			{"rr.example.invalid", 1, 0, true}, {"rr.example.invalid", 28, 0, true}, {"alias.example.invalid", 5, 0, true}, {"service.example.invalid", 64, 0, true}, {"service.example.invalid", 65, 0, true},
			{"feed-01.example.invalid", 1, 3, false}, {"feed-01.example.invalid", 28, 3, false}, {"blocked-cname.example.invalid", 5, 0, false}, {"feed-01.example.invalid", 64, 0, false}, {"feed-01.example.invalid", 65, 3, false},
			{"blocked-alias.example.invalid", 1, 3, false}, {"sub.feed-01.example.invalid", 1, 3, false}, {"notfeed-01.example.invalid", 1, 0, true}, {"absent.example.invalid", 1, 3, false},
		} {
			// Every blocked fixture except the intentionally absent name exists at
			// the owned authority. An absent upstream answer cannot pass a rule.
			if !tc.answer && tc.name != "absent.example.invalid" {
				control, e := s.fixtureProbe(ctx, s.lab.node, "-mode", "dns", "-addr", s.lab.labPrefix+".54:53", "-name", tc.name, "-type", fmt.Sprint(tc.typ))
				noteJSON(&rec, fmt.Sprintf("authority_%d_%t", len(rec.Observations), proto), control)
				if e != nil || control.RCode != 0 || len(control.Types) == 0 {
					rec.Status = StatusFail
				}
			}
			synthetic := !tc.answer && tc.name != "absent.example.invalid"
			// F8-TTL5 is the origin TTL. Let prior negative cache entries age
			// out naturally; do not flush caches or equate NODATA with NXDOMAIN.
			if synthetic {
				time.Sleep(6 * time.Second)
			}
			started := time.Now()
			got, e := s.ProbeNativeDNS(ctx, "", tc.name, tc.typ, proto, "")
			elapsed := time.Since(started)
			backend := map[uint16]string{1: "F8-A", 28: "F8-AAAA", 5: "F8-CNAME", 64: "F8-SVCB", 65: "F8-HTTPS"}[tc.typ]
			if tc.name == "blocked-alias.example.invalid" {
				backend = "F8-A scope C"
			}
			if tc.name == "sub.feed-01.example.invalid" {
				backend = "F8-A scope W"
			}
			if !synthetic {
				backend = "upstream authority control; separate from synthetic filtering"
			}
			// The resolver profile starts each synthetic SOA at 5. Whole-second
			// cache aging during the measured lookup can decrease that value.
			ageBudget := uint32(elapsed/time.Second) + 1
			minimum := uint32(0)
			if ageBudget < 5 {
				minimum = 5 - ageBudget
			}
			ttlOK := !synthetic || got.HasSOA && got.AuthorityTTL >= minimum && got.AuthorityTTL <= 5 && elapsed < 5*time.Second
			key := fmt.Sprintf("case_%d_%t", len(rec.Observations), proto)
			noteJSON(&rec, key, map[string]any{"case": strings.Split(tc.name, ".")[0], "rr_type": tc.typ, "tcp": proto, "expected_rcodes": []int{tc.rcode}, "expected_answer": tc.answer, "backend_row": backend, "ttl_source": "F8-TTL5: synthetic origin SOA TTL 5; 6s prior-cache expiry", "lookup_milliseconds": elapsed.Milliseconds(), "cache_aged_minimum_ttl": minimum, "actual": got})
			if e != nil || !got.Available || got.RCode != tc.rcode || tc.answer && len(got.Types) == 0 || !tc.answer && len(got.Types) != 0 || !ttlOK {
				rec.Status = StatusFail
			}

		}
	}
	first, e := s.ProbeNativeDNS(ctx, "", "cache.example.invalid", 1, false, s.lab.originIP)
	noteJSON(&rec, "cache_initial", first)
	rebind, e2 := s.ProbeNativeDNS(ctx, "", "rebind.example.invalid", 1, false, s.lab.originIP)
	noteJSON(&rec, "rebind_initial", rebind)
	if e != nil || e2 != nil || !first.ExpectedAnswer || !rebind.ExpectedAnswer {
		rec.Status = StatusFail
	}
	if e = s.SetDNSFixturePhase("private"); e != nil {
		rec.Status = StatusFail
		return
	}
	wait, cancel := context.WithTimeout(ctx, 4*time.Second)
	var authority FixtureProbeResult
	changedAuthority := waitUntil(wait, 200*time.Millisecond, func() bool {
		authority, e = s.fixtureProbe(wait, s.lab.node, "-mode", "dns", "-addr", s.lab.labPrefix+".54:53", "-name", "cache.example.invalid", "-expect", "10.203.0.30")
		return e == nil && authority.ExpectedAnswer
	})
	cancel()
	noteJSON(&rec, "changed_authority", authority)
	if !changedAuthority {
		rec.Status = StatusFail
	}
	negative, e := s.ProbeNativeDNS(ctx, "", "absent.example.invalid", 1, false, "")
	noteJSON(&rec, "negative_cache_before_ttl", negative)
	if e != nil || negative.RCode != 3 {
		rec.Status = StatusFail
	}
	cached, e := s.ProbeNativeDNS(ctx, "", "cache.example.invalid", 1, false, s.lab.originIP)
	noteJSON(&rec, "cache_before_ttl", cached)
	if e != nil || !cached.ExpectedAnswer {
		rec.Status = StatusFail
	}
	time.Sleep(12 * time.Second)
	negative, e = s.ProbeNativeDNS(ctx, "", "absent.example.invalid", 1, false, s.lab.originIP)
	noteJSON(&rec, "negative_cache_after_ttl", negative)
	if e != nil || !negative.ExpectedAnswer {
		rec.Status = StatusFail
	}
	changed, e := s.ProbeNativeDNS(ctx, "", "cache.example.invalid", 1, false, "10.203.0.30")
	noteJSON(&rec, "cache_after_ttl", changed)
	if e != nil || !changed.ExpectedAnswer {
		rec.Status = StatusFail
	}
	for _, tc := range []struct {
		typ      uint16
		expected string
	}{{1, "10.203.0.30"}, {28, "fd00::30"}} {
		got, e := s.ProbeNativeDNS(ctx, "", "rebind.example.invalid", tc.typ, false, tc.expected)
		noteJSON(&rec, fmt.Sprintf("rebind_rr_%d", tc.typ), got)
		if e != nil || !got.ExpectedAnswer {
			rec.Status = StatusFail
		}
	}
	// Destination control listener is positive from the node; the native member
	// must fail using the returned private answer. DNS allowlisting grants no IP access.
	control, e := s.fixtureProbe(ctx, s.lab.node, "-mode", "connect", "-addr", "10.203.0.30:9447")
	noteJSON(&rec, "private_target_positive_control", control)
	denied, e2 := s.fixtureProbe(ctx, s.lab.member, "-mode", "lookup", "-name", "rebind.example.invalid", "-addr", "9447", "-expect", "10.203.0.30")
	noteJSON(&rec, "rebind_private_connect", denied)
	if e != nil || e2 != nil || !control.Connected || !denied.ExpectedAnswer || denied.Connected {
		rec.Status = StatusFail
	}
	v6Control, e := s.fixtureProbe(ctx, s.lab.node, "-mode", "connect", "-addr", "[fd00::30]:9447")
	v6Denied, e2 := s.fixtureProbe(ctx, s.lab.member, "-mode", "connect", "-addr", "[fd00::30]:9447")
	noteJSON(&rec, "private_ipv6_positive_control", v6Control)
	noteJSON(&rec, "private_ipv6_denied", v6Denied)
	if e != nil || e2 != nil || !v6Control.Connected || v6Denied.Connected {
		rec.Status = StatusFail
	}
	recPathNote(&rec)("expected", "blocked A/AAAA/HTTPS and CNAME-chain A yield NXDOMAIN; direct CNAME/SVCB yield RCODE 0 empty NODATA with synthetic SOA TTL 5 (cache aging measured); allowed label boundary resolves; old cache survives until TTL then changes; allowlisted private A/AAAA never authorize member forwarding")
}
func (r *runner) fixtureFiniteGrant(ctx context.Context, s *Stack, admin BrowserSession) {
	rec := Record{ID: "finite-grant-downgrade", Title: "Finite plan downgrade via production revoke, admin create and worker readback", Spec: []string{"§9.2", "AT02", "D-050", "D-081"}, Status: StatusPass}
	defer func() { r.record(rec) }()
	// Separate grant-only cohort avoids the primary member's payment checkout.
	counts, err := s.ScaleNativeMembers(ctx, admin, r.planRevID, "finite", 1, 3)
	noteJSON(&rec, "native_cohort", counts)
	if err != nil {
		rec.Status = StatusFail
		recPathNote(&rec)("error", err.Error())
		return
	}
	member, err := s.Login(ctx, "finite-u01", "member")
	if err != nil {
		rec.Status = StatusFail
		return
	}
	conn, err := r.lab.db(ctx)
	if err != nil {
		rec.Status = StatusFail
		return
	}
	defer conn.Close(ctx)
	var raw []byte
	if conn.QueryRow(ctx, `SELECT payload FROM plan_revisions WHERE id=$1`, r.planRevID).Scan(&raw) != nil {
		rec.Status = StatusFail
		return
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil {
		rec.Status = StatusFail
		return
	}
	payload["device_limit"] = 2
	lower, _, err := s.ActivatePolicy(ctx, admin, "plan", "finite-lower", payload)
	if err != nil {
		rec.Status = StatusFail
		return
	}
	transition, err := s.ReplaceFiniteGrant(ctx, admin, uuid.MustParse(member.MemberID), lower, time.Now().UTC().Add(time.Hour))
	noteJSON(&rec, "downgrade", transition)
	if err != nil {
		rec.Status = StatusFail
		recPathNote(&rec)("error", err.Error())
		return
	}
	if !transition.Projected || transition.Authorized != 2 || transition.Pending != 1 {
		rec.Status = StatusFail
	}
	invalid, e := s.ReplaceFiniteGrant(ctx, admin, uuid.MustParse(member.MemberID), lower, time.Now().UTC().Add(-time.Second))
	noteJSON(&rec, "expired_input_rejected", e != nil && !invalid.Accepted && len(invalid.Revoked) == 0)
	if e == nil {
		rec.Status = StatusFail
	}
	restored, err := s.ReplaceFiniteGrant(ctx, admin, uuid.MustParse(member.MemberID), r.planRevID, time.Now().UTC().Add(time.Hour))
	noteJSON(&rec, "restored", restored)
	if err != nil || !restored.Projected || restored.Authorized != 3 || restored.Pending != 0 {
		rec.Status = StatusFail
		if err != nil {
			recPathNote(&rec)("restore_error", err.Error())
		}
	}
}

func fixtureCampaignDirectory() string {
	if os.Getenv("FVPN_E2E_FOLLOWUP23") == "1" {
		return "followup23"
	}
	if os.Getenv("FVPN_E2E_FOLLOWUP21") == "1" {
		return "followup21"
	}
	if os.Getenv("FVPN_E2E_FOLLOWUP20") == "1" {
		return "followup20"
	}
	if os.Getenv("FVPN_E2E_FOLLOWUP19") == "1" {
		return "followup19"
	}
	return "followup16"
}

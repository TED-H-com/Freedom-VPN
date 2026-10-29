package dns

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"golang.org/x/net/dns/dnsmessage"
)

func Test12_5_EvidenceIsTheDNSOKSignal(t *testing.T) {
	ok := Evidence{PolicyGeneration: "3", DocumentHash: strings.Repeat("ab", 32), Verified: true}
	if !ok.DNSOK() || ok.DNSPolicyStatus() != api.HealthCheckStatusPassed || DNSPolicyCheckName != api.NodeCheckNameDnsPolicy {
		t.Fatalf("verified evidence does not advertise dns_ok: %+v", ok)
	}
	bad := Evidence{Verified: false}
	if bad.DNSOK() || bad.DNSPolicyStatus() != api.HealthCheckStatusFailed {
		t.Fatal("unverified evidence would advertise dns_ok")
	}
}

func Test12_5_ApplyReadsBackAndProbes(t *testing.T) {
	f, ap := newRig(t, "normal")
	gen, doc, hash := testDoc(t, false, "192.0.2.54")
	ev, err := ap.Apply(context.Background(), gen, doc)
	if err != nil {
		t.Fatal(err)
	}
	if !ev.Verified || !ev.DNSOK() || ev.DocumentHash != hash || ev.PolicyGeneration != gen || ev.AppliedAt.IsZero() {
		t.Fatalf("evidence %+v", ev)
	}
	if ev.DNSPolicyStatus() != api.HealthCheckStatusPassed {
		t.Fatal(ev.DNSPolicyStatus())
	}
	if _, err := ap.Apply(context.Background(), gen, doc); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.filters != 0 || f.queryLog || f.stats || f.safeEnabled {
		t.Fatal("privacy or filters left on")
	}
	if !strings.Contains(strings.Join(f.rules, "\n"), hash) {
		t.Fatal("applied rules do not carry the document hash")
	}
	if f.setRules != 1 {
		t.Fatalf("reapply wrote rules %d times", f.setRules)
	}
}

func Test12_5_SafeSearchSettingIsApplied(t *testing.T) {
	f, ap := newRig(t, "normal")
	gen, doc, _ := testDoc(t, true, "192.0.2.54")
	if _, err := ap.Apply(context.Background(), gen, doc); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.safeEnabled || !f.safe.Google || !f.safe.YouTube {
		t.Fatalf("safe search was not enabled: %+v", f.safe)
	}
}

func Test12_5_InvalidDocumentDoesNotWrite(t *testing.T) {
	f, ap := newRig(t, "normal")
	_, doc, _ := testDoc(t, false, "192.0.2.54")
	doc.SchemaVersion = 0
	ev, err := ap.Apply(context.Background(), "3", doc)
	if err == nil || ev.Verified {
		t.Fatal("invalid document was applied")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setRules != 0 || len(f.dnsPosts) != 0 {
		t.Fatalf("invalid document wrote resolver config: rules %d dns %d", f.setRules, len(f.dnsPosts))
	}
	if strings.Contains(err.Error(), "tracker.example.com") {
		t.Fatal("error recorded a query name")
	}
}

func Test12_5_SetRulesFailureRestoresUpstreams(t *testing.T) {
	f, ap := newRig(t, "normal")
	f.mu.Lock()
	f.failRules = 1
	f.mu.Unlock()
	gen, doc, hash := testDoc(t, false, "192.0.2.54")
	ev, err := ap.Apply(context.Background(), gen, doc)
	if err == nil || ev.Verified {
		t.Fatal("partial apply was reported as verified")
	}
	if !errors.Is(err, errRolledBack) {
		t.Fatalf("err %v", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if strings.Join(f.upstreams, ",") != "192.0.2.53" {
		t.Fatalf("upstreams %v", f.upstreams)
	}
	if strings.Contains(strings.Join(f.rules, "\n"), hash) {
		t.Fatal("failed rules were left installed")
	}
	if strings.Contains(err.Error(), "tracker") {
		t.Fatal("rollback error recorded a query name")
	}
}

func Test12_7_PrivacyReadBackFailureRollsBack(t *testing.T) {
	f, ap := newRig(t, "normal")
	f.mu.Lock()
	f.stickQueryLog = true
	f.mu.Unlock()
	gen, doc, hash := testDoc(t, false, "192.0.2.54")
	ev, err := ap.Apply(context.Background(), gen, doc)
	if err == nil || ev.DNSOK() {
		t.Fatal("query log left the policy looking verified")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.sawQueryLogDisable {
		t.Fatal("applier did not try to disable the query log")
	}
	if strings.Contains(strings.Join(f.rules, "\n"), hash) {
		t.Fatal("rules stayed installed after the privacy check failed")
	}
}

func Test12_5_ProbeMismatchRollsBack(t *testing.T) {
	f, ap := newRig(t, "always-answer")
	gen, doc, hash := testDoc(t, false, "192.0.2.54")
	ev, err := ap.Apply(context.Background(), gen, doc)
	if err == nil || ev.Verified || !errors.Is(err, errBlockedProbe) && !errors.Is(err, errRolledBack) {
		t.Fatalf("evidence %+v err %v", ev, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if strings.Contains(strings.Join(f.rules, "\n"), hash) || strings.Join(f.upstreams, ",") != "192.0.2.53" {
		t.Fatalf("probe failure left rules %v upstreams %v", f.rules, f.upstreams)
	}
}

func Test17_1_WithdrawIsFailClosed(t *testing.T) {
	_, ap := newRig(t, "withdraw-servfail")
	gen, doc, hash := testDoc(t, false, "192.0.2.54")
	if _, err := ap.Apply(context.Background(), gen, doc); err != nil {
		t.Fatal(err)
	}
	ev, err := ap.Withdraw(context.Background(), gen, hash)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Verified || ev.DNSOK() || ev.DocumentHash != hash || ev.DNSPolicyStatus() != api.HealthCheckStatusFailed {
		t.Fatalf("withdraw evidence %+v", ev)
	}
}

func Test17_1_WithdrawAnswerRollsBack(t *testing.T) {
	f, ap := newRig(t, "withdraw-answers")
	gen, doc, hash := testDoc(t, false, "192.0.2.54")
	applied, err := ap.Apply(context.Background(), gen, doc)
	if err != nil || !applied.Verified {
		t.Fatal(err)
	}
	ev, err := ap.Withdraw(context.Background(), gen, hash)
	if err == nil || ev.DNSOK() != ev.Verified {
		t.Fatalf("evidence %+v err %v", ev, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if strings.Contains(strings.Join(f.upstreams, ","), dnspublish.WithdrawUpstream) {
		t.Fatalf("fail-closed upstream left in place: %v", f.upstreams)
	}
	if !strings.Contains(strings.Join(f.rules, "\n"), hash) {
		t.Fatal("previous policy was not restored")
	}
	if !ev.Verified || ev.DocumentHash != hash {
		t.Fatalf("previous policy was not reported: %+v", ev)
	}
}

func Test12_5_WithdrawOlderHashKeepsCurrent(t *testing.T) {
	_, ap := newRig(t, "normal")
	gen, doc, hash := testDoc(t, false, "192.0.2.54")
	if _, err := ap.Apply(context.Background(), gen, doc); err != nil {
		t.Fatal(err)
	}
	older := strings.Repeat("ab", 32)
	ev, err := ap.Withdraw(context.Background(), "1", older)
	if err != nil || !ev.Verified || ev.DocumentHash != hash {
		t.Fatalf("superseded withdraw changed the resolver: %+v err %v", ev, err)
	}
}

func testDoc(t *testing.T, safe bool, upstream string) (string, dnspublish.Document, string) {
	t.Helper()
	compiled, err := dnspublish.Compile(dnspublish.Input{
		ProfileCode: "standard", BlockResponse: "nxdomain", BlockedTTLSeconds: 60, SafeSearch: safe,
		Upstreams: []string{upstream},
		Rules: []dnspublish.Rule{
			{RuleID: "01900000-0000-7000-8000-0000000000b1", Domain: "tracker.example.com", IncludeSubdomains: true, Action: "block", Reason: "admin_rule"},
			{RuleID: "01900000-0000-7000-8000-0000000000a1", Domain: "cdn.example.com", IncludeSubdomains: false, Action: "allow", Reason: "false_positive"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return "3", compiled.Document, compiled.Hash
}

type rig struct {
	mu                 sync.Mutex
	rules              []string
	upstreams          []string
	bootstrap          []string
	fallback           []string
	blocking           string
	ttl                uint32
	protection         bool
	filtering          bool
	queryLog           bool
	stats              bool
	safeEnabled        bool
	safe               engines
	filters            int
	setRules           int
	failRules          int
	stickQueryLog      bool
	sawQueryLogDisable bool
	dnsPosts           [][]string
	answerMode         string
}

func newRig(t *testing.T, mode string) (*rig, *Applier) {
	t.Helper()
	f := &rig{
		upstreams:  []string{"192.0.2.53"},
		bootstrap:  []string{"192.0.2.53"},
		fallback:   []string{},
		blocking:   "nxdomain",
		ttl:        60,
		protection: true,
		filtering:  true,
		filters:    1,
		answerMode: mode,
	}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	addr := startDNS(t, f.answer)
	ap, err := New([]Target{{ID: "r1", AdminURL: srv.URL, DNSAddr: addr}})
	if err != nil {
		t.Fatal(err)
	}
	return f, ap
}

func (f *rig) answer(name string) (dnsmessage.RCode, netip.Addr, bool) {
	f.mu.Lock()
	up := append([]string(nil), f.upstreams...)
	mode := f.answerMode
	f.mu.Unlock()
	if contains(up, dnspublish.WithdrawUpstream) {
		if mode == "withdraw-answers" {
			return dnsmessage.RCodeSuccess, netip.MustParseAddr("192.0.2.10"), true
		}
		return dnsmessage.RCodeServerFailure, netip.Addr{}, false
	}
	if mode == "always-answer" || name != "tracker.example.com" {
		return dnsmessage.RCodeSuccess, netip.MustParseAddr("192.0.2.15"), true
	}
	return dnsmessage.RCodeNameError, netip.Addr{}, false
}

func (f *rig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/control/dns_info":
		writeJSON(w, map[string]any{
			"upstream_dns": f.upstreams, "bootstrap_dns": f.bootstrap, "fallback_dns": f.fallback,
			"blocking_mode": f.blocking, "blocked_response_ttl": f.ttl, "protection_enabled": f.protection,
		})
	case "/control/dns_config":
		var in struct {
			UpstreamDNS        []string `json:"upstream_dns"`
			BootstrapDNS       []string `json:"bootstrap_dns"`
			FallbackDNS        []string `json:"fallback_dns"`
			BlockingMode       string   `json:"blocking_mode"`
			BlockedResponseTTL uint32   `json:"blocked_response_ttl"`
			ProtectionEnabled  bool     `json:"protection_enabled"`
		}
		_ = json.Unmarshal(body, &in)
		f.upstreams, f.bootstrap, f.fallback = in.UpstreamDNS, in.BootstrapDNS, in.FallbackDNS
		f.blocking, f.ttl, f.protection = in.BlockingMode, in.BlockedResponseTTL, in.ProtectionEnabled
		f.dnsPosts = append(f.dnsPosts, append([]string(nil), in.UpstreamDNS...))
		w.WriteHeader(http.StatusOK)
	case "/control/filtering/status":
		filters := []any{}
		if f.filters > 0 {
			filters = []any{map[string]any{"id": 1, "url": "https://filters.example/list.txt", "enabled": true}}
		}
		writeJSON(w, map[string]any{"enabled": f.filtering, "filters": filters, "user_rules": f.rules})
	case "/control/filtering/config":
		var in struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.Unmarshal(body, &in)
		f.filtering = in.Enabled
		w.WriteHeader(http.StatusOK)
	case "/control/filtering/remove_url":
		f.filters = 0
		w.WriteHeader(http.StatusOK)
	case "/control/filtering/set_rules":
		f.setRules++
		if f.failRules > 0 {
			f.failRules--
			http.Error(w, "no", http.StatusInternalServerError)
			return
		}
		var in struct {
			Rules []string `json:"rules"`
		}
		_ = json.Unmarshal(body, &in)
		f.rules = append([]string(nil), in.Rules...)
		w.WriteHeader(http.StatusOK)
	case "/control/querylog/config":
		enabled := f.queryLog
		if f.stickQueryLog {
			enabled = true
		}
		file := false
		writeJSON(w, map[string]any{"enabled": enabled, "file_enabled": file})
	case "/control/querylog/config/update":
		var in struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.Unmarshal(body, &in)
		if !in.Enabled {
			f.sawQueryLogDisable = true
		}
		if !f.stickQueryLog {
			f.queryLog = in.Enabled
		}
		w.WriteHeader(http.StatusOK)
	case "/control/stats/config":
		writeJSON(w, map[string]any{"enabled": f.stats})
	case "/control/stats/config/update":
		var in struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.Unmarshal(body, &in)
		f.stats = in.Enabled
		w.WriteHeader(http.StatusOK)
	case "/control/safesearch/status":
		writeJSON(w, map[string]any{
			"enabled": f.safeEnabled, "bing": f.safe.Bing, "duckduckgo": f.safe.DuckDuckGo,
			"ecosia": f.safe.Ecosia, "google": f.safe.Google, "pixabay": f.safe.Pixabay,
			"yandex": f.safe.Yandex, "youtube": f.safe.YouTube,
		})
	case "/control/safesearch/settings":
		var in safeStatus
		_ = json.Unmarshal(body, &in)
		f.safeEnabled, f.safe = in.Enabled, in.engines
		w.WriteHeader(http.StatusOK)
	case "/control/cache_clear":
		w.WriteHeader(http.StatusOK)
	default:
		http.NotFound(w, r)
	}
}

func startDNS(t *testing.T, answer func(name string) (dnsmessage.RCode, netip.Addr, bool)) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			var msg dnsmessage.Message
			if err := msg.Unpack(buf[:n]); err != nil || len(msg.Questions) == 0 {
				continue
			}
			qname := strings.ToLower(strings.TrimSuffix(msg.Questions[0].Name.String(), "."))
			code, ip, ok := answer(qname)
			msg.Response = true
			msg.RecursionAvailable = true
			msg.RCode = code
			msg.Answers = nil
			if ok && ip.Is4() {
				msg.Answers = []dnsmessage.Resource{{
					Header: dnsmessage.ResourceHeader{Name: msg.Questions[0].Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 30},
					Body:   &dnsmessage.AResource{A: ip.As4()},
				}}
			}
			out, err := msg.Pack()
			if err != nil {
				continue
			}
			_, _ = pc.WriteTo(out, addr)
		}
	}()
	return pc.LocalAddr().String()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func contains(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}

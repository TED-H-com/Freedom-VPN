//go:build e2e

package members

import (
	"fmt"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func (l *live) configureNativeDNS(r *record) bool {
	if !l.secondExit(r) {
		return false
	}
	// Both nodes must first report the same current product-applied bytes.
	l.resolverEvidence(r)
	restore := l.useSecond()
	l.resolverEvidence(r)
	restore()
	cfg, err := l.hook.ConfigureNativeDNS(l.ctx, labstack.BrowserSession{Client: l.admin.client, CSRF: l.admin.csrf}, l.awaitDevice(l.nativePeer), []uuid.UUID{l.node, l.second.NodeID})
	l.nativeDNS = err == nil && len(cfg.Resolvers) == 2 && cfg.NativeResolversMatch && cfg.NativeDNSEnabled && cfg.SystemUsesNativeStub && cfg.SameAppliedDocument
	r.jsonCheck("native two-resolver configuration", "exact issued pair, native enabled, one OS stub, identical applied documents", map[string]any{"configuration": cfg, "error": safeError(err)}, "ConfigureNativeDNS; TEST ONLY peer-restricted distribution of production-issued addresses", l.nativeDNS)
	if l.nativeDNS {
		start := time.Now()
		var attempts []labstack.FixtureProbeResult
		ready := l.wait(30*time.Second, func() bool {
			got, e := l.hook.ProbeNativeDNS(l.ctx, "", "rr.example.invalid", 1, false, l.originIP())
			attempts = append(attempts, got)
			return e == nil && got.Available && got.ExpectedAnswer
		})
		r.jsonCheck("native DNS forwarding readiness", "bounded positive lookup after native down/up; preserve startup attempts", map[string]any{"attempts": attempts, "elapsed_ms": time.Since(start).Milliseconds()}, "ProbeNativeDNS startup readiness before one-shot RR matrix", ready)
		l.nativeDNS = ready
	}
	if !l.nativeDNS {
		r.block("F4-05: native DNS setup did not converge; no fallback or matrix inferred")
	}
	return l.nativeDNS
}
func (l *live) nativeDNSMatrix(r *record) {
	if !l.configureNativeDNS(r) {
		return
	}
	// AT08 restarted the node process; restore the independent private listener.
	err := l.hook.StartFixtureTargets(l.ctx)
	r.check("private fixture listeners", "ready", safeError(err), "StartFixtureTargets in owned node namespace", err == nil)
	for _, tcp := range []bool{false, true} {
		for index, tc := range []struct {
			name   string
			typ    uint16
			code   int
			answer bool
		}{
			{"rr.example.invalid", 1, 0, true}, {"rr.example.invalid", 28, 0, true}, {"alias.example.invalid", 5, 0, true}, {"service.example.invalid", 64, 0, true}, {"service.example.invalid", 65, 0, true},
			{"feed-01.example.invalid", 1, 3, false}, {"feed-01.example.invalid", 28, 3, false}, {"blocked-cname.example.invalid", 5, 0, false}, {"feed-01.example.invalid", 64, 0, false}, {"feed-01.example.invalid", 65, 3, false},
			{"blocked-alias.example.invalid", 1, 3, false}, {"sub.feed-01.example.invalid", 1, 3, false}, {"notfeed-01.example.invalid", 1, 0, true}, {"absent.example.invalid", 1, 3, false},
		} {
			label := fmt.Sprintf("RR case %02d TCP=%t", index+1, tcp)
			if !tc.answer && tc.name != "absent.example.invalid" {
				control, e := l.fixture("node", "-mode", "dns", "-addr", l.stack.LabPrefix+".54:53", "-name", tc.name, "-type", fmt.Sprint(tc.typ))
				r.jsonCheck(label+" authority control", "existing upstream RR", control, "direct owned authority positive control; no query contents exported", e == nil && control.Available && control.RCode == 0 && len(control.Types) > 0)
			}
			got, e := l.hook.ProbeNativeDNS(l.ctx, "", tc.name, tc.typ, tcp, "")
			// dns-publish follow-up 8: synthetic filtering is type-specific.
			// Absent upstream names and positive cache controls are separate.
			tableRow, scope := "upstream control (not synthetic filtering)", ""
			shapeOK := got.Available && got.RCode == tc.code && (len(got.Types) > 0) == tc.answer
			if !tc.answer && tc.name != "absent.example.invalid" {
				tableRow = map[uint16]string{1: "F8-A", 28: "F8-AAAA", 5: "F8-CNAME", 64: "F8-SVCB", 65: "F8-HTTPS"}[tc.typ]
				scope = "D"
				if tc.name == "blocked-alias.example.invalid" {
					scope = "C"
				} else if strings.HasPrefix(tc.name, "sub.") {
					scope = "W"
				}
				shapeOK = shapeOK && tableRow != "" && got.HasSOA && got.AuthorityTTL == 5
				tableRow += " / F8-TTL5"
			}
			r.jsonCheck(label, "exact RCODE/answers; filtered rows require SOA TTL 5 per docs/dns/BACKEND.md §2.1.1–§2.1.4", map[string]any{"case": strings.Split(tc.name, ".")[0], "rr_type": tc.typ, "tcp": tcp, "expected_rcode": tc.code, "expected_answer": tc.answer, "backend_table_row": tableRow, "scope": scope, "actual": got}, "ProbeNativeDNS through native OS stub; retain observed SOA/TTL/types", e == nil && shapeOK)
		}
	}
	for _, name := range []string{"FEED-01.example.invalid", "feed-01.example.invalid."} {
		got, e := l.hook.ProbeNativeDNS(l.ctx, "", name, 1, false, "")
		r.jsonCheck("native case/dot normalization", "F8-A / F8-TTL5: RCODE 3, zero answers, SOA TTL 5", got, "ProbeNativeDNS mixed case/trailing dot synthetic blocked name; docs/dns/BACKEND.md §2.1", e == nil && blockedDNS5(got))
	}
	l.nativeDNSCache(r)
	r.block("F4-07: IDNA wire-positive fixture is still absent (canonical A-label API input is tested). Owner: w3-devstack-e2e. DNS wording retired 2026-10-31 by docs/dns/BACKEND.md §2.1.1–§2.1.4; distribution/platform gaps remain separate.")
}
func (l *live) nativeDNSCache(r *record) {
	// Refresh the negative within its five-second window immediately before change.
	first, e := l.hook.ProbeNativeDNS(l.ctx, "", "cache.example.invalid", 1, false, l.originIP())
	negative, ne := l.hook.ProbeNativeDNS(l.ctx, "", "absent.example.invalid", 1, false, "")
	r.jsonCheck("cache initial", "positive TTL <=10, negative SOA TTL <=5", map[string]any{"positive": first, "negative": negative}, "native DNS before authority-only change", e == nil && ne == nil && first.ExpectedAnswer && len(first.TTLs) > 0 && first.TTLs[0] <= 10 && negative.RCode == 3 && negative.HasSOA && negative.AuthorityTTL <= 5)
	changedAt := time.Now().UTC()
	err := l.hook.SetDNSFixturePhase("private")
	defer func() { _ = l.hook.SetDNSFixturePhase("initial") }()
	var authority labstack.FixtureProbeResult
	changed := l.wait(4*time.Second, func() bool {
		authority, e = l.fixture("node", "-mode", "dns", "-addr", l.stack.LabPrefix+".54:53", "-name", "cache.example.invalid", "-expect", "10.203.0.30")
		return e == nil && authority.ExpectedAnswer
	})
	r.jsonCheck("changed authority", "new answer while caches untouched", map[string]any{"authority": authority, "changed_at": changedAt}, "SetDNSFixturePhase(private); query authority until new serial visible", err == nil && changed)
	negative, ne = l.hook.ProbeNativeDNS(l.ctx, "", "absent.example.invalid", 1, false, "")
	cached, e := l.hook.ProbeNativeDNS(l.ctx, "", "cache.example.invalid", 1, false, l.originIP())
	r.jsonCheck("positive/negative cache before TTL", "old positive and NXDOMAIN persist with remaining TTL", map[string]any{"positive": cached, "negative": negative, "observed_at": time.Now().UTC()}, "ProbeNativeDNS without flushing either cache", e == nil && ne == nil && cached.ExpectedAnswer && len(cached.TTLs) > 0 && cached.TTLs[0] > 0 && cached.TTLs[0] <= 10 && negative.RCode == 3 && negative.HasSOA && negative.AuthorityTTL <= 5)
	time.Sleep(12 * time.Second)
	negative, ne = l.hook.ProbeNativeDNS(l.ctx, "", "absent.example.invalid", 1, false, l.originIP())
	cached, e = l.hook.ProbeNativeDNS(l.ctx, "", "cache.example.invalid", 1, false, "10.203.0.30")
	r.jsonCheck("positive/negative cache after TTL", "both new authority answers visible", map[string]any{"positive": cached, "formerly_negative": negative, "observed_at": time.Now().UTC()}, "ProbeNativeDNS after real TTL elapses", e == nil && ne == nil && cached.ExpectedAnswer && negative.ExpectedAnswer)
	for _, tc := range []struct {
		typ      uint16
		ip, addr string
	}{{1, "10.203.0.30", "10.203.0.30:9447"}, {28, "fd00::30", "[fd00::30]:9447"}} {
		answer, e := l.hook.ProbeNativeDNS(l.ctx, "", "rebind.example.invalid", tc.typ, false, tc.ip)
		control, ce := l.fixture("node", "-mode", "connect", "-addr", tc.addr)
		denied, de := l.fixture("member", "-mode", "connect", "-addr", tc.addr)
		if tc.typ == 1 {
			denied, de = l.fixture("member", "-mode", "lookup", "-name", "rebind.example.invalid", "-addr", "9447", "-expect", tc.ip)
		}
		r.jsonCheck(fmt.Sprintf("allowlisted private rebinding type %d", tc.typ), "private DNS answer; node listener works; member connection fails", map[string]any{"answer": answer, "node_control": control, "member": denied}, "native DNS private phase; independent node listener and member connection", e == nil && ce == nil && de == nil && answer.ExpectedAnswer && control.Connected && !denied.Connected && (tc.typ != 1 || denied.ExpectedAnswer))
	}
}
func (l *live) nativeDNSFallback(r *record) {
	if !l.nativeDNS {
		r.block("F4-05: native two-resolver configuration prerequisite failed")
		return
	}
	primary, secondary := l.name("origin-dns"), l.name("origin-dns-secondary")
	defer func() { l.cleanupDocker("start", primary, secondary) }()
	probe := func(label, name string, want bool) {
		got, e := l.fixture("member", "-mode", "lookup", "-name", name, "-expect", l.originIP())
		r.jsonCheck(label, "normal system resolution matches availability; bounded at 13 seconds", got, "fixture lookup uses OS stub, no explicit resolver", e == nil && got.Available == want && (!want || got.ExpectedAnswer) && got.Milliseconds <= 13000)
	}
	probe("native DNS healthy", "rr.example.invalid", true)
	at := time.Now().UTC()
	_, err := l.docker("stop", "-t", "2", primary)
	r.jsonCheck("primary resolver transport down", "stopped", map[string]any{"at": at, "stopped": err == nil}, "docker stop owned origin-dns transport; node policy health stays up", err == nil)
	probe("automatic secondary fallback", "notfeed-01.example.invalid", true)
	blocked, e := l.hook.ProbeNativeDNS(l.ctx, "", "feed-01.example.invalid", 1, false, "")
	r.jsonCheck("fallback block policy", "F8-A / F8-TTL5: RCODE 3, zero answers, SOA TTL 5 from remaining resolver", blocked, "ProbeNativeDNS with primary transport down; docs/dns/BACKEND.md §2.1.4", e == nil && blockedDNS5(blocked))
	_, err = l.docker("stop", "-t", "2", secondary)
	r.check("both resolver transports down", "stopped", safeError(err), "docker stop owned secondary transport", err == nil)
	// Unique uncached name avoids the positive/negative matrix's cache entries.
	probe("both down bounded failure", "both-down-"+l.run+".example.invalid", false)
	_, err = l.docker("start", primary, secondary)
	start := time.Now().UTC()
	var recovered labstack.FixtureProbeResult
	ok := l.wait(90*time.Second, func() bool {
		recovered, e = l.fixture("member", "-mode", "lookup", "-name", "alias.example.invalid", "-expect", l.originIP())
		return e == nil && recovered.Available && recovered.ExpectedAnswer
	})
	r.jsonCheck("native DNS recovery", "system resolution resumes within bounded recovery", map[string]any{"result": recovered, "recovery_ms": time.Since(start).Milliseconds()}, "restart only owned resolver transports; native system lookup", err == nil && ok)
	r.block("F4-05 production automatic nameserver distribution is unclaimed: ConfigureNativeDNS uses a TEST ONLY peer group with the exact product-issued list. Android Private DNS/browser DoH remain operator/native-client integration work.")
}

// The fixture applies a five-second blocked-response profile. This checks the
// exact fresh synthetic response, not a cached upstream NXDOMAIN convention.
func blockedDNS5(got labstack.FixtureProbeResult) bool {
	return got.Available && got.RCode == 3 && len(got.Types) == 0 && got.HasSOA && got.AuthorityTTL == 5
}

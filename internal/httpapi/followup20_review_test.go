package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/google/uuid"
)

func reviewAgentPOST(h http.Handler, leaf *x509.Certificate) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/lease-renewals", strings.NewReader(`{}`))
	r.RemoteAddr = "198.51.100.40:40000"
	r.Header.Set("Content-Type", "application/json")
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{leaf}}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func reviewLeaf(raw []byte) *x509.Certificate {
	return &x509.Certificate{Raw: raw}
}

// reviewNodeLeaf is two still-usable serials of one node: different DER,
// the same urn:fvpn:node SAN that CA.Issue writes.
func reviewNodeLeaf(raw []byte, node, inc uuid.UUID) *x509.Certificate {
	return &x509.Certificate{
		Raw: raw,
		URIs: []*url.URL{
			{Scheme: "urn", Opaque: "fvpn:node:" + node.String()},
			{Scheme: "urn", Opaque: "fvpn:incarnation:" + inc.String()},
		},
	}
}

func reviewAgentHandler(t *testing.T, c config.Config, agents auth.AgentAuthenticator) http.Handler {
	t.Helper()
	c.PublicOrigin = "https://portal.example.invalid"
	sets, err := Compose()
	if err != nil {
		t.Fatal(err)
	}
	h, err := New(sets, Options{Config: c, AgentListener: true, Agents: agents})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// TestReviewR14_OneHundredCertStormFitsTheAgentListener is 100 nodes each
// sending one lease renewal at the 10s first wait. Each leaf is its own
// ingress principal (burst 60). The class gate is 128 and the table holds
// 10000 principals, so this storm is inside the follow-up 19 budget.
func TestReviewR14_OneHundredCertStormFitsTheAgentListener(t *testing.T) {
	c := config.Defaults()
	if c.IngressRate.Burst < 60 || c.MaxConcurrent < 128 || c.MaxRatePrincipals < 10000 {
		t.Fatalf("production budget changed: ingress=%+v concurrent=%d principals=%d", c.IngressRate, c.MaxConcurrent, c.MaxRatePrincipals)
	}
	h := reviewAgentHandler(t, c, auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
		return auth.Principal{Role: auth.Agent, ID: uuid.New(), NodeID: uuid.New(), IncarnationID: uuid.New()}, nil
	}))
	denied := 0
	for i := 0; i < 100; i++ {
		raw := []byte(fmt.Sprintf("leaf-%03d", i))
		if reviewAgentPOST(h, reviewLeaf(raw)).Code == http.StatusTooManyRequests {
			denied++
		}
	}
	if denied != 0 {
		t.Fatalf("100 distinct leaves at one renewal each: %d denied, budget ingress burst %d concurrent %d principals %d", denied, c.IngressRate.Burst, c.MaxConcurrent, c.MaxRatePrincipals)
	}
}

// TestReviewR14_DistinctLeavesMultiplyOneNodeBudget is the renewal-overlap
// case. Two leaves carry the same urn:fvpn:node SAN and different DER.
// After leaf A spends the ingress burst, leaf B is refused. A third leaf
// of another node is not.
func TestReviewR14_DistinctLeavesMultiplyOneNodeBudget(t *testing.T) {
	c := config.Defaults()
	c.IngressRate = config.Bucket{PerSecond: 0.0001, Burst: 2}
	c.MaxConcurrent = 8
	node := uuid.Must(uuid.NewV7())
	other := uuid.Must(uuid.NewV7())
	inc := uuid.Must(uuid.NewV7())
	same := auth.Principal{Role: auth.Agent, ID: node, NodeID: node, IncarnationID: inc}
	var seen []string
	h := reviewAgentHandler(t, c, auth.AgentAuthenticatorFunc(func(_ context.Context, cert *x509.Certificate) (auth.Principal, error) {
		seen = append(seen, certRateKey(cert))
		return same, nil
	}))
	leafA := reviewNodeLeaf([]byte("renewal-leaf-a"), node, inc)
	leafB := reviewNodeLeaf([]byte("renewal-leaf-b"), node, inc)
	if certRateKey(leafA) == certRateKey(leafB) {
		t.Fatal("leaves share a fingerprint; a 429 would not show the node key")
	}
	if agentIngressKey(leafA) != "node:"+node.String() || agentIngressKey(leafB) != agentIngressKey(leafA) {
		t.Fatalf("ingress keys %s %s, want one node bucket", agentIngressKey(leafA), agentIngressKey(leafB))
	}
	deniedA := 0
	for range 3 {
		if reviewAgentPOST(h, leafA).Code == http.StatusTooManyRequests {
			deniedA++
		}
	}
	if deniedA != 1 {
		t.Fatalf("leaf A burst: 429s=%d, want 1", deniedA)
	}
	if got := reviewAgentPOST(h, leafB); got.Code != http.StatusTooManyRequests {
		t.Fatalf("leaf B of node %s was admitted after leaf A spent the ingress burst (status %d, keys %v)", node, got.Code, seen)
	}
	leafC := reviewNodeLeaf([]byte("renewal-leaf-c"), other, inc)
	if got := reviewAgentPOST(h, leafC); got.Code == http.StatusTooManyRequests {
		t.Fatalf("node %s was refused on node %s's burst: %d", other, node, got.Code)
	}
}

func TestReviewR14_CertFingerprintIsLeafRawOnly(t *testing.T) {
	raw := []byte("TEST ONLY leaf der")
	sum := sha256.Sum256(raw)
	want := hex.EncodeToString(sum[:])
	leaf := &x509.Certificate{Raw: raw, RawTBSCertificate: make([]byte, 1<<20)}
	other := &x509.Certificate{Raw: append([]byte(nil), raw...), RawTBSCertificate: []byte{1}}
	if certRateKey(leaf) != want || certRateKey(other) != want {
		t.Fatalf("fingerprint is not sha256(cert.Raw): %s %s want %s", certRateKey(leaf), certRateKey(other), want)
	}
	if certRateKey(reviewLeaf([]byte{1})) == certRateKey(reviewLeaf([]byte{2})) {
		t.Fatal("distinct leaves collided")
	}
}

// TestReviewR14_EvictIdleWindowOfEight keeps a full idle bucket reachable
// behind a depleted tail. The walk continues past eight depleted keys until
// it finds that bucket or exhausts the table.
func TestReviewR14_EvictIdleWindowOfEight(t *testing.T) {
	now := time.Date(2026, 10, 30, 15, 24, 0, 0, time.UTC)
	setLimiterClock(t, func() time.Time { return now })
	fast := config.Bucket{PerSecond: 1000, Burst: 5}
	dead := config.Bucket{PerSecond: 0, Burst: 1}

	fill := func(t *testing.T, max, depleted int) *limiter {
		t.Helper()
		l := newLimiter(max)
		if l.allow("F", fast) != 0 {
			t.Fatal("F")
		}
		for i := 0; i < depleted; i++ {
			if l.allow(fmt.Sprintf("D%d", i), dead) != 0 {
				t.Fatalf("D%d", i)
			}
		}
		if l.allow("F", fast) != 0 {
			t.Fatal("touch F")
		}
		return l
	}

	t.Run("eight_depleted_hide_a_full_bucket", func(t *testing.T) {
		l := fill(t, 9, 8)
		now = now.Add(time.Second)
		if len(l.entries) != l.max {
			t.Fatalf("table %d/%d", len(l.entries), l.max)
		}
		if d := l.allow("newcomer", fast); d != 0 {
			t.Fatalf("newcomer refused while a full idle bucket sat past 8 depleted keys (wait %s, entries %d)", d, len(l.entries))
		}
	})

	t.Run("seven_depleted_reaches_the_full_bucket", func(t *testing.T) {
		l := fill(t, 8, 7)
		now = now.Add(time.Second)
		if d := l.allow("newcomer", fast); d != 0 {
			t.Fatalf("full idle bucket behind 7 depleted keys was not evicted: wait %s", d)
		}
	})
}

// TestReviewR14_Anonymous48HalfBudget is the hosting-/48 cap after D-083.
// At max 10 the cap is min(10/2, 64) = 5. The next /64 in that /48 is
// refused without an insert, and another /48 still enters.
func TestReviewR14_Anonymous48HalfBudget(t *testing.T) {
	const maxPrincipals = 10
	if anonymousGroupCap(maxPrincipals) != 5 {
		t.Fatalf("cap(%d) = %d, want 5", maxPrincipals, anonymousGroupCap(maxPrincipals))
	}
	now := time.Date(2026, 10, 30, 22, 26, 0, 0, time.UTC)
	setLimiterClock(t, func() time.Time { return now })
	l := newLimiter(maxPrincipals)
	b := config.Bucket{PerSecond: 1, Burst: 1}
	admitted := 0
	var group string
	for i := 0; i < 8; i++ {
		ip := net.ParseIP(fmt.Sprintf("2001:db8:0:%x::", i))
		key, ok := rateKeyFromIP(ip)
		if !ok {
			t.Fatalf("rate key %v", ip)
		}
		group = anonymousGroup(key)
		if group == "" {
			t.Fatalf("no /48 group for %s", key)
		}
		if l.allowGrouped(key, group, b) == 0 {
			admitted++
		}
	}
	if admitted != 5 || len(l.entries) != 5 {
		t.Fatalf("/48 %s admitted %d of the first 8 keys (%d entries); cap is 5", group, admitted, len(l.entries))
	}
	past, ok := rateKeyFromIP(net.ParseIP("2001:db8:0:a::"))
	if !ok {
		t.Fatal("past key")
	}
	if d := l.allowGrouped(past, anonymousGroup(past), b); d == 0 || len(l.entries) != 5 {
		t.Fatalf("hosting /48 passed the cap: wait %s entries %d", d, len(l.entries))
	}
	other, ok := rateKeyFromIP(net.ParseIP("2001:db8:1::"))
	if !ok {
		t.Fatal("other /48")
	}
	if anonymousGroup(other) == group {
		t.Fatal("other prefix collapsed into the hosting /48")
	}
	if d := l.allowGrouped(other, anonymousGroup(other), b); d != 0 {
		t.Fatalf("other /48 refused: %s", d)
	}
}

func TestReviewR14_ForwardedAddressParseAgreesWithNormalForm(t *testing.T) {
	type sample struct {
		in   string
		note string
	}
	var inputs []sample
	add := func(in, note string) { inputs = append(inputs, sample{in, note}) }

	hosts := []string{
		"192.0.2.8",
		"2001:db8::1",
		"2001:db8:1:2:3:4:5:6",
		"2001:db8:1::2:3:4:5:6",
		"::1",
		"fe80::1",
		"::ffff:192.0.2.8",
		"2001:db8::",
		"::",
	}
	for _, h := range hosts {
		add(h, "bare")
		add("["+h+"]", "brackets")
		add("["+h+"]:443", "brackets-port")
		add(strings.ToUpper(h), "upper")
		add(h+"%eth0", "zone")
		add("["+h+"%eth0]", "brackets-zone")
		add("["+h+"%eth0]:443", "brackets-zone-port")
	}
	add("192.0.2.8:443", "v4-port")
	add("2001:db8:1:2:3:4:5:6:443", "eight-fields-plus-port")
	add("2001:db8:1:2:3:4:5:6::", "eight-fields-plus-trailing-compression")
	add("::1:2:3:4:5:6:7", "compression-at-start-seven")
	add("1:2:3:4:5:6:7::", "compression-at-end-seven")
	add("2001:0db8::1", "leading-zero")
	add("2001:db8::1%eth0", "zone-eth0")
	add("2001:db8::1%eth1", "zone-eth1")
	add("[fe80::1%eth0]", "link-local-zone")
	add("[fe80::1%eth1]", "link-local-other-zone")
	add("::ffff:192.0.2.8", "v4-mapped")
	add("192.0.2.8", "v4-of-mapped")
	add("", "empty")
	add("[", "broken-bracket")
	add("2001:db8::1::2", "two-compressions")
	add("999.1.1.1", "bad-v4")
	add("[2001:db8::1]:99999", "port-out-of-range")

	var diverged []string
	keys := map[string]map[string]string{}
	remember := func(host, spelling, key string) {
		if keys[host] == nil {
			keys[host] = map[string]string{}
		}
		keys[host][spelling] = key
	}
	for _, s := range inputs {
		fwd := parseForwardedIP(s.in)
		rem := parseRemoteIP(s.in)
		for _, got := range []struct {
			kind string
			ip   net.IP
		}{{"forwarded", fwd}, {"remote", rem}} {
			if got.ip == nil {
				continue
			}
			oracle, ok := independentHost(s.in, got.kind == "forwarded")
			collapsed, collapsedOK := zeroWidthHost(s.in)
			switch {
			case collapsedOK && got.ip.Equal(collapsed):
				// Documented eight-field zero-width compression. A later
				// port split of the same text is a different host.
			case ok && !got.ip.Equal(oracle):
				diverged = append(diverged, fmt.Sprintf("%s %q (%s) parsed %s, independent host %s", got.kind, s.in, s.note, got.ip, oracle))
			case !ok && sameBracketHost(s.in, got.ip):
				// SplitHostPort does not range-check the port. The host is
				// still the bracketed address, which the rule allows.
			case !ok:
				diverged = append(diverged, fmt.Sprintf("%s %q (%s) parsed %s with no bracketed or RFC-5952 host", got.kind, s.in, s.note, got.ip))
			}
			if key, ok := rateKeyFromIP(got.ip); ok {
				remember(canonicalHost(got.ip), s.in+"/"+got.kind, key)
			}
		}
	}
	if len(diverged) != 0 {
		t.Errorf("parse diverged from bracketed or RFC-5952 form:\n%s", strings.Join(diverged, "\n"))
	}

	// Designed shared buckets. These are listed, not failed.
	sameKey(t, "192.0.2.8", "::ffff:192.0.2.8")
	sameKey(t, "2001:db8::1", "2001:DB8::1")
	sameKey(t, "2001:db8::1", "2001:0db8::1")
	sameKey(t, "2001:db8::1", "2001:db8::1%eth0")
	sameKey(t, "fe80::1%eth0", "fe80::1%eth1")
	sameKey(t, "2001:db8:1:2:3:4:5:6", "2001:db8:1::2:3:4:5:6")
	sameKey(t, "2001:db8:1:2:3:4:5:6", "2001:db8:1:2:3:4:5:6:443")

	full, ok := rateKeyFromIP(parseForwardedIP("2001:db8:1:2:3:4:5:6"))
	stripped, ok2 := rateKeyFromIP(parseForwardedIP("2001:db8:1::2:3:4:5:6"))
	if !ok || !ok2 || full != stripped {
		t.Fatalf("r13 H-2 vector moved the /64: %s %s", full, stripped)
	}
	if got := parseForwardedIP("2001:db8:1:2:3:4:5:6::"); got != nil {
		t.Fatalf("eight fields plus a trailing :: parsed as %s", got)
	}

	mappedNet, err := config.ParseTrustedProxies([]string{"::ffff:192.0.2.8/128"})
	if err == nil || mappedNet != nil {
		t.Fatalf("IPv4-mapped trusted proxy was accepted: %v %v", mappedNet, err)
	}
	v4Net, err := config.ParseTrustedProxies([]string{"192.0.2.8/32"})
	if err != nil {
		t.Fatal(err)
	}
	if !ipInNets(net.ParseIP("192.0.2.8"), v4Net) || !ipInNets(net.ParseIP("::ffff:192.0.2.8"), v4Net) {
		t.Fatal("192.0.2.8/32 did not match the v4 peer and the IPv4-mapped spelling")
	}
	v6Net, err := config.ParseTrustedProxies([]string{"2001:db8::1/128", "2001:0db8:0:0:0:0:0:2/128"})
	if err != nil {
		t.Fatal(err)
	}
	if !ipInNets(net.ParseIP("2001:0db8::1"), v6Net) || !ipInNets(net.ParseIP("2001:db8::2"), v6Net) {
		t.Fatal("trusted-proxy CIDR did not normalise leading zeros")
	}
	// rightmostForwarded keeps the rightmost hop that is not itself a trusted
	// proxy. The client has to be that hop.
	peerKey, ok := requestRateKey("192.0.2.8:443", []string{"203.0.113.9"}, v4Net, true)
	if !ok || peerKey != "203.0.113.9" {
		t.Fatalf("trusted v4 peer did not select the client hop: %s %v", peerKey, ok)
	}
	mappedKey, ok := requestRateKey("[::ffff:192.0.2.8]:443", []string{"203.0.113.9"}, v4Net, true)
	if !ok || mappedKey != "203.0.113.9" {
		t.Fatalf("IPv4-mapped trusted peer spelling did not select the client hop: %s %v", mappedKey, ok)
	}
}

func sameBracketHost(s string, got net.IP) bool {
	if len(s) < 2 || s[0] != '[' || got == nil {
		return false
	}
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		return false
	}
	ip, ok := parseIndependent(host)
	return ok && got.Equal(ip)
}

func sameKey(t *testing.T, a, b string) {
	t.Helper()
	ka, oka := rateKeyFromIP(parseForwardedIP(a))
	kb, okb := rateKeyFromIP(parseForwardedIP(b))
	if !oka || !okb || ka != kb {
		t.Errorf("%q key %s (%v) and %q key %s (%v)", a, ka, oka, b, kb, okb)
	}
}

func canonicalHost(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}

// independentHost is net.ParseIP / net.SplitHostPort without the product's
// zero-width eight-field collapse. bareBracketOK selects X-Forwarded-For.
func independentHost(s string, bareBracketOK bool) (net.IP, bool) {
	if len(s) >= 2 && s[0] == '[' {
		if s[len(s)-1] == ']' {
			if !bareBracketOK {
				return nil, false
			}
			return parseIndependent(s[1 : len(s)-1])
		}
		host, port, err := net.SplitHostPort(s)
		if err != nil {
			return nil, false
		}
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, false
		}
		return parseIndependent(host)
	}
	if ip, ok := parseIndependent(s); ok {
		return ip, true
	}
	if strings.Count(s, ":") <= 1 {
		host, port, err := net.SplitHostPort(s)
		if err != nil {
			return nil, false
		}
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, false
		}
		return parseIndependent(host)
	}
	colon := strings.LastIndexByte(s, ':')
	if colon <= 0 || colon == len(s)-1 {
		return nil, false
	}
	port := s[colon+1:]
	if _, err := strconv.ParseUint(port, 10, 16); err != nil || strings.IndexByte(s[:colon], ':') < 0 {
		return nil, false
	}
	return parseIndependent(s[:colon])
}

func parseIndependent(host string) (net.IP, bool) {
	if i := strings.IndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	ip := net.ParseIP(host)
	return ip, ip != nil
}

func zeroWidthHost(s string) (net.IP, bool) {
	host := s
	if len(s) >= 2 && s[0] == '[' {
		if s[len(s)-1] == ']' {
			host = s[1 : len(s)-1]
		} else if h, _, err := net.SplitHostPort(s); err == nil {
			host = h
		}
	}
	if i := strings.IndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	if strings.Count(host, "::") != 1 {
		return nil, false
	}
	n := 0
	for _, p := range strings.Split(host, ":") {
		if p != "" {
			n++
		}
	}
	if n != 8 {
		return nil, false
	}
	ip := net.ParseIP(strings.Replace(host, "::", ":", 1))
	return ip, ip != nil
}

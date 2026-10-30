package dns

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"golang.org/x/net/dns/dnsmessage"
)

const maxProbes = 4

var (
	errBlockedProbe   = errors.New("blocked probe mismatch")
	errAllowedProbe   = errors.New("allowed probe mismatch")
	errWithdrawProbe  = errors.New("withdrawn resolver still answered")
	errRolledBack     = errors.New("resolver update failed and the previous configuration was restored")
	errRollbackFailed = errors.New("resolver update failed and rollback did not restore the previous configuration")
)

// Target is one AdGuard Home resolver. DNSAddr is host:port of its DNS listener.
type Target struct {
	ID       string
	AdminURL string
	Username string
	Password string
	DNSAddr  string
}

// Applier converges every target to the same document or withdraws them together.
// A failure restores the previous configuration. It never leaves a resolver
// half-updated, and it never enables the query log or statistics (§12.5, §12.7, §17.1).
type Applier struct {
	mu      sync.Mutex
	targets []Target
	http    *http.Client
	dns     *dnspolicy.DNSClient
	clock   func() time.Time
}

// New builds an applier. Every target needs an admin URL and a DNS address.
func New(targets []Target) (*Applier, error) {
	if len(targets) == 0 {
		return nil, errors.New("dns applier needs a resolver")
	}
	cp := append([]Target(nil), targets...)
	for i, t := range cp {
		if strings.TrimSpace(t.AdminURL) == "" || strings.TrimSpace(t.DNSAddr) == "" {
			return nil, fmt.Errorf("resolver %s needs an admin URL and a dns address", t.ID)
		}
		if strings.TrimSpace(t.ID) == "" {
			cp[i].ID = fmt.Sprintf("resolver-%d", i+1)
		}
	}
	return &Applier{
		targets: cp,
		http:    newHTTPClient(),
		dns:     dnspolicy.NewDNSClient(3 * time.Second),
	}, nil
}

// Apply converges every resolver to doc and verifies read-back plus a synthetic
// query. generation is the decimal policy_generation stored beside the document.
func (a *Applier) Apply(ctx context.Context, generation string, doc dnspublish.Document) (Evidence, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	gen, err := decstr.Parse(strings.TrimSpace(generation))
	if err != nil {
		return a.unverified(), fmt.Errorf("policy_generation: %w", err)
	}
	hash, _, err := dnspublish.Hash(doc)
	if err != nil {
		return a.unverified(), err
	}
	if err := dnspublish.Verify(doc); err != nil {
		ev, _ := a.verifiedCurrent(ctx, "")
		if ev.Verified {
			return ev, err
		}
		return a.unverified(), err
	}
	rules, err := dnspublish.RenderRules(doc, gen.String(), hash)
	if err != nil {
		return a.unverified(), err
	}
	snaps, err := a.snapshot(ctx)
	if err != nil {
		return a.unverified(), err
	}
	if a.already(snaps, doc, rules, gen.String(), hash) {
		if err := a.probeDocument(ctx, doc); err != nil {
			ev := evidenceFrom(snaps[0], a.now())
			ev.Verified = false
			return ev, err
		}
		ev := evidenceFrom(documentState(doc, rules), a.now())
		ev.Verified = true
		return ev, nil
	}
	desired := documentState(doc, rules)
	if err := a.writeAll(ctx, desired); err != nil {
		return a.rollback(ctx, snaps, err)
	}
	if err := a.readBack(ctx, desired, hash, gen.String()); err != nil {
		return a.rollback(ctx, snaps, err)
	}
	if err := a.probeDocument(ctx, doc); err != nil {
		return a.rollback(ctx, snaps, err)
	}
	ev := evidenceFrom(documentState(doc, rules), a.now())
	ev.Verified = true
	return ev, nil
}

// Withdraw removes a policy. An empty hash withdraws whatever is installed.
// A different hash that still verifies is left in place. The returned evidence
// has Verified false when the resolver is fail-closed (§17.1).
func (a *Applier) Withdraw(ctx context.Context, generation, hash string) (Evidence, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	hash = strings.ToLower(strings.TrimSpace(hash))
	if hash != "" && !validHash(hash) {
		return a.unverified(), errors.New("document_hash must be 64 hex characters")
	}
	genStr := "0"
	if strings.TrimSpace(generation) != "" {
		gen, err := decstr.Parse(strings.TrimSpace(generation))
		if err != nil {
			return a.unverified(), fmt.Errorf("policy_generation: %w", err)
		}
		genStr = gen.String()
	}
	snaps, err := a.snapshot(ctx)
	if err != nil {
		return a.unverified(), err
	}
	if hash != "" {
		if ev, ok := a.verifiedCurrent(ctx, hash); ok && ev.Verified {
			return ev, nil
		}
	}
	if hash == "" {
		if g := headerValue(snaps[0].Rules, dnspublish.HeaderPolicyGenerationPrefix); generation == "" && g != "" {
			genStr = g
		}
		if h := headerValue(snaps[0].Rules, dnspolicy.HeaderBundleHashPrefix); validHash(h) {
			hash = strings.ToLower(h)
		}
	}
	names := probeNames(snaps[0].Rules)
	closed := closedState(genStr, hash)
	if err := a.writeAll(ctx, closed); err != nil {
		return a.rollback(ctx, snaps, err)
	}
	if err := a.readBack(ctx, closed, hash, genStr); err != nil {
		return a.rollback(ctx, snaps, err)
	}
	if err := a.probeNoAddress(ctx, names); err != nil {
		return a.rollback(ctx, snaps, err)
	}
	return Evidence{PolicyGeneration: genStr, DocumentHash: hash, AppliedAt: a.now(), Verified: false}, nil
}

func (a *Applier) snapshot(ctx context.Context) ([]resolverState, error) {
	out := make([]resolverState, len(a.targets))
	for i, t := range a.targets {
		st, err := a.readState(ctx, t)
		if err != nil {
			return nil, fmt.Errorf("%s: read configuration: %w", t.ID, err)
		}
		out[i] = st
	}
	return out, nil
}

func (a *Applier) writeAll(ctx context.Context, st resolverState) error {
	for _, t := range a.targets {
		if err := a.writeState(ctx, t, st); err != nil {
			return err
		}
	}
	return nil
}

func (a *Applier) readBack(ctx context.Context, want resolverState, hash, generation string) error {
	for _, t := range a.targets {
		got, err := a.readState(ctx, t)
		if err != nil {
			return fmt.Errorf("%s: read back: %w", t.ID, err)
		}
		if !sameState(got, want) {
			return fmt.Errorf("%s: read-back mismatch", t.ID)
		}
		if hash != "" && !strings.EqualFold(headerValue(got.Rules, dnspolicy.HeaderBundleHashPrefix), hash) {
			return fmt.Errorf("%s: document hash was not applied", t.ID)
		}
		if generation != "" && headerValue(got.Rules, dnspublish.HeaderPolicyGenerationPrefix) != generation && !withdrawn(got) {
			return fmt.Errorf("%s: policy generation was not applied", t.ID)
		}
		if !privacyOff(got) {
			return fmt.Errorf("%s: query log or statistics is enabled", t.ID)
		}
	}
	return nil
}

func (a *Applier) already(snaps []resolverState, doc dnspublish.Document, rules []string, generation, hash string) bool {
	want := documentState(doc, rules)
	for _, st := range snaps {
		if !sameState(st, want) || !strings.EqualFold(headerValue(st.Rules, dnspolicy.HeaderBundleHashPrefix), hash) {
			return false
		}
		if headerValue(st.Rules, dnspublish.HeaderPolicyGenerationPrefix) != generation {
			return false
		}
	}
	return true
}

func (a *Applier) rollback(ctx context.Context, snaps []resolverState, cause error) (Evidence, error) {
	for i, t := range a.targets {
		if err := a.writeState(ctx, t, snaps[i]); err != nil {
			return a.unverified(), errors.Join(errRollbackFailed, cause, err)
		}
		got, err := a.readState(ctx, t)
		if err != nil || !sameState(got, snaps[i]) {
			return a.unverified(), errors.Join(errRollbackFailed, cause)
		}
	}
	ev := evidenceFrom(snaps[0], a.now())
	ev.Verified = a.snapshotsVerify(ctx, snaps)
	return ev, errors.Join(errRolledBack, cause)
}

func (a *Applier) snapshotsVerify(ctx context.Context, snaps []resolverState) bool {
	if len(snaps) == 0 {
		return false
	}
	hash := headerValue(snaps[0].Rules, dnspolicy.HeaderBundleHashPrefix)
	for i, st := range snaps {
		if !strings.EqualFold(headerValue(st.Rules, dnspolicy.HeaderBundleHashPrefix), hash) {
			return false
		}
		if !a.oneVerifies(ctx, st, a.targets[i].DNSAddr) {
			return false
		}
	}
	return validHash(hash)
}

func (a *Applier) verifiedCurrent(ctx context.Context, exceptHash string) (Evidence, bool) {
	snaps, err := a.snapshot(ctx)
	if err != nil || len(snaps) == 0 {
		return Evidence{}, false
	}
	hash := strings.ToLower(headerValue(snaps[0].Rules, dnspolicy.HeaderBundleHashPrefix))
	if !validHash(hash) || (exceptHash != "" && strings.EqualFold(hash, exceptHash)) {
		return Evidence{}, false
	}
	if !a.snapshotsVerify(ctx, snaps) {
		return Evidence{}, false
	}
	ev := evidenceFrom(snaps[0], a.now())
	ev.Verified = true
	return ev, true
}

func (a *Applier) oneVerifies(ctx context.Context, st resolverState, dnsAddr string) bool {
	if !privacyOff(st) || len(st.Fallback) != 0 || !st.Filtering || !st.Protection || st.Filters != 0 {
		return false
	}
	if withdrawn(st) || sameStrings(st.Upstreams, []string{dnspublish.WithdrawUpstream}) {
		return false
	}
	if !validHash(headerValue(st.Rules, dnspolicy.HeaderBundleHashPrefix)) {
		return false
	}
	if headerValue(st.Rules, dnspublish.HeaderPolicyGenerationPrefix) == "" {
		return false
	}
	blockedNames, allowedNames := domains(st.Rules)
	return a.probe(ctx, []string{dnsAddr}, blockedNames, allowedNames, st.BlockingMode) == nil
}

func (a *Applier) probeDocument(ctx context.Context, doc dnspublish.Document) error {
	rules, err := dnspublish.RenderRules(doc, "0", strings.Repeat("0", 64))
	if err != nil {
		return err
	}
	blockedNames, allowedNames := domains(rules)
	addrs := make([]string, len(a.targets))
	for i, t := range a.targets {
		addrs[i] = t.DNSAddr
	}
	return a.probe(ctx, addrs, blockedNames, allowedNames, aghMode(doc.BlockResponse))
}

func (a *Applier) probe(ctx context.Context, addrs, blockedNames, allowedNames []string, mode string) error {
	if len(blockedNames) == 0 && len(allowedNames) == 0 {
		blockedNames = nil
		allowedNames = []string{"example.com"}
	}
	for _, addr := range addrs {
		for _, name := range capNames(blockedNames) {
			resp, err := a.dns.Query(ctx, addr, name, dnsmessage.TypeA)
			if err != nil || !responseBlocked(resp, mode) {
				return errBlockedProbe
			}
		}
		for _, name := range capNames(allowedNames) {
			resp, err := a.dns.Query(ctx, addr, name, dnsmessage.TypeA)
			if err != nil || responseBlocked(resp, mode) || !answersAddress(resp) {
				return errAllowedProbe
			}
			if mode == "null_ip" && nullAddr(resp) {
				return errAllowedProbe
			}
		}
	}
	return nil
}

func (a *Applier) probeNoAddress(ctx context.Context, names []string) error {
	if len(names) == 0 {
		names = []string{"example.com"}
	}
	for _, t := range a.targets {
		for _, name := range capNames(names) {
			resp, err := a.dns.Query(ctx, t.DNSAddr, name, dnsmessage.TypeA)
			if err != nil {
				continue
			}
			if answersAddress(resp) {
				return fmt.Errorf("%s: %w", t.ID, errWithdrawProbe)
			}
		}
	}
	return nil
}

func (a *Applier) unverified() Evidence {
	return Evidence{AppliedAt: a.now(), Verified: false}
}

func (a *Applier) now() time.Time {
	if a.clock != nil {
		return a.clock().UTC()
	}
	return time.Now().UTC()
}

func documentState(doc dnspublish.Document, rules []string) resolverState {
	on := doc.SafeSearch
	fileOff := false
	return resolverState{
		Rules:        append([]string(nil), rules...),
		Upstreams:    append([]string(nil), doc.Upstreams...),
		Bootstrap:    append([]string(nil), doc.BootstrapDNS...),
		Fallback:     []string{},
		BlockingMode: aghMode(doc.BlockResponse),
		BlockedTTL:   uint32(doc.BlockedTTLSeconds),
		Protection:   true,
		Filtering:    true,
		QueryLogFile: &fileOff,
		SafeEnabled:  on,
		Safe:         engines{Bing: on, DuckDuckGo: on, Ecosia: on, Google: on, Pixabay: on, Yandex: on, YouTube: on},
	}
}

func closedState(generation, hash string) resolverState {
	fileOff := false
	return resolverState{
		Rules:        withdrawnRules(generation, hash),
		Upstreams:    []string{dnspublish.WithdrawUpstream},
		Bootstrap:    []string{dnspublish.WithdrawUpstream},
		Fallback:     []string{},
		BlockingMode: "nxdomain",
		BlockedTTL:   uint32(dnspublish.DefaultBlockedTTLSeconds),
		Protection:   true,
		Filtering:    true,
		QueryLogFile: &fileOff,
	}
}

func withdrawnRules(generation, hash string) []string {
	lines := []string{
		"! fvpn-bundle-version: 1",
		dnspublish.HeaderWithdrawnPrefix + "true",
		dnspublish.HeaderPolicyGenerationPrefix + generation,
	}
	if hash != "" {
		lines = append(lines, dnspolicy.HeaderBundleHashPrefix+hash)
	}
	return lines
}

func aghMode(block string) string {
	switch block {
	case "refused":
		return "refused"
	case "null_address":
		return "null_ip"
	default:
		return "nxdomain"
	}
}

func privacyOff(st resolverState) bool {
	if st.QueryLog || st.Statistics {
		return false
	}
	return st.QueryLogFile == nil || !*st.QueryLogFile
}

func withdrawn(st resolverState) bool {
	return strings.EqualFold(headerValue(st.Rules, dnspublish.HeaderWithdrawnPrefix), "true")
}

func sameState(got, want resolverState) bool {
	if !sameStrings(got.Rules, want.Rules) || !sameStrings(got.Upstreams, want.Upstreams) || !sameStrings(got.Bootstrap, want.Bootstrap) || !sameStrings(got.Fallback, want.Fallback) {
		return false
	}
	if got.BlockingMode != want.BlockingMode || got.BlockedTTL != want.BlockedTTL || got.Protection != want.Protection || got.Filtering != want.Filtering {
		return false
	}
	if got.QueryLog != want.QueryLog || got.Statistics != want.Statistics || flag(got.QueryLogFile) != flag(want.QueryLogFile) {
		return false
	}
	// External lists are removed and not put back. A snapshot that still had
	// one only fails when the desired state itself requires zero filters.
	if want.Filters == 0 && got.Filters != 0 {
		return false
	}
	if got.SafeEnabled != want.SafeEnabled || got.Safe != want.Safe {
		return false
	}
	return true
}

func flag(p *bool) bool { return p != nil && *p }

func sameStrings(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func headerValue(rules []string, prefix string) string {
	for _, line := range rules {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func evidenceFrom(st resolverState, now time.Time) Evidence {
	var feeds []FeedEvidence
	for _, line := range st.Rules {
		if !strings.HasPrefix(line, "! fvpn-feed: ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "! fvpn-feed: "))
		if len(fields) != 3 || !validHash(fields[1]) {
			continue
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil || count < 1 {
			continue
		}
		feeds = append(feeds, FeedEvidence{FeedCode: fields[0], ContentSHA256: fields[1], RuleCount: count})
	}
	return Evidence{
		Feeds:            feeds,
		PolicyGeneration: headerValue(st.Rules, dnspublish.HeaderPolicyGenerationPrefix),
		DocumentHash:     strings.ToLower(headerValue(st.Rules, dnspolicy.HeaderBundleHashPrefix)),
		AppliedAt:        now,
		Verified:         false,
	}
}

func validHash(h string) bool {
	if len(h) != 64 {
		return false
	}
	for _, r := range h {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

// Classify probes by effective precedence, including overlapping suffixes.
// An allow that overrides a feed must not also be probed as blocked.
func domains(rules []string) (blockedNames, allowedNames []string) {
	type entry struct {
		domain   string
		sub      bool
		priority int
	}
	var entries []entry
	for _, line := range rules {
		if line == "" || strings.HasPrefix(line, "!") {
			continue
		}
		priority := 3
		if strings.HasPrefix(line, "@@") {
			priority = 2
		}
		if strings.HasSuffix(line, "$important") {
			priority = 1
		}
		raw := strings.TrimPrefix(line, "@@")
		sub := strings.HasPrefix(raw, "||")
		raw = strings.TrimLeft(raw, "|")
		if i := strings.IndexByte(raw, '^'); i >= 0 {
			raw = raw[:i]
		} else {
			continue
		}
		entries = append(entries, entry{raw, sub, priority})
	}
	seen := map[string]bool{}
	for _, candidate := range entries {
		if seen[candidate.domain] {
			continue
		}
		seen[candidate.domain] = true
		priority := 4
		for _, rule := range entries {
			if rule.priority < priority && dnspolicy.MatchesDomain(rule.domain, rule.sub, candidate.domain) {
				priority = rule.priority
			}
		}
		if priority == 2 {
			allowedNames = append(allowedNames, candidate.domain)
		} else {
			blockedNames = append(blockedNames, candidate.domain)
		}
	}
	return blockedNames, allowedNames
}

func probeNames(rules []string) []string {
	blockedNames, allowedNames := domains(rules)
	out := append(append([]string{}, blockedNames...), allowedNames...)
	if len(out) == 0 {
		return []string{"example.com"}
	}
	return capNames(out)
}

func capNames(in []string) []string {
	if len(in) > maxProbes {
		return in[:maxProbes]
	}
	return in
}

func responseBlocked(resp *dnspolicy.DNSResponse, mode string) bool {
	if resp == nil {
		return false
	}
	switch mode {
	case "refused":
		return resp.RCode == dnsmessage.RCodeRefused
	case "null_ip":
		return resp.IsSuccess() && nullAddr(resp)
	default:
		return resp.IsNXDOMAIN()
	}
}

func nullAddr(resp *dnspolicy.DNSResponse) bool {
	for _, ans := range resp.Answers {
		if ans.Type == dnsmessage.TypeA && ans.IP.Is4() && ans.IP.String() == "0.0.0.0" {
			return true
		}
	}
	return false
}

func answersAddress(resp *dnspolicy.DNSResponse) bool {
	if resp == nil || !resp.IsSuccess() {
		return false
	}
	for _, ans := range resp.Answers {
		if ans.Type == dnsmessage.TypeA || ans.Type == dnsmessage.TypeAAAA {
			return ans.IP.IsValid()
		}
	}
	return false
}

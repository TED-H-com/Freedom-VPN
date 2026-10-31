package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

type revocationPlane struct {
	*memPlane
	ex                  renderedExecutor
	models              []enforce.Model
	results             []enforce.ApplyResult
	errors              []error
	baselinesAfterApply int
	unverified          bool
}

func (p *revocationPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	p.ex.model = m
	p.models = append(p.models, m)
	p.baselinesAfterApply = 0
	res, err := enforce.Apply(ctx, &p.ex, m, prev)
	if p.unverified {
		res.Verified = false
	}
	p.results = append(p.results, res)
	p.errors = append(p.errors, err)
	return res, err
}

func (p *revocationPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	p.baselinesAfterApply++
	return p.memPlane.Baseline(ctx, dp)
}

func TestISO01_PendingConntrackDeleteKeepsServingAndRetries(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	if !r.eng.serving || !p.results[0].Verified || p.errors[0] != nil {
		t.Fatalf("fixture did not install verified ruleset: %v", p.errors)
	}
	previousCounters := cloneCounters(r.eng.counters)
	binding := oneBinding(r.clock.Wall())
	binding.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, binding))
	p.ex.failDelete = true
	stepOK(t, r)
	wantRevoke := []netip.Addr{netip.MustParseAddr("100.64.1.1")}
	assertPending := func() {
		t.Helper()
		last := len(p.results) - 1
		if !p.results[last].Verified || !enforce.IsConntrackDeleteFailure(p.errors[last]) {
			t.Fatalf("not a verified conntrack failure: %+v %v", p.results[last], p.errors[last])
		}
		if !reflect.DeepEqual(p.models[last].Revoke, wantRevoke) || p.baselinesAfterApply != 0 {
			t.Fatalf("pending revoke lost or baseline restored: revoke=%v baselines=%d", p.models[last].Revoke, p.baselinesAfterApply)
		}
		if !reflect.DeepEqual(r.eng.counters, previousCounters) {
			t.Fatal("partial apply result discarded prior counters")
		}
		// Force the normal outbox delivery so the fake control sees this tick.
		r.eng.lastObs = time.Time{}
		if err := r.eng.observe(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := r.eng.flush(context.Background()); err != nil {
			t.Fatal(err)
		}
		rep, ok := r.srv.LastObservation()
		gen, err := rep.AppliedConfigGeneration.Get()
		if !ok || err != nil || gen != 3 || r.eng.appliedGen != 3 || !rep.MemberForwardingEnabled || !r.eng.decision.Serve || len(rep.ForwardingBlockReasons) != 0 || rep.SafetyStatus != api.SafetyStatusPassed {
			t.Fatalf("pending delete withdrew verified generation: serving=%v gen=%d reasons=%v errors=%v", rep.MemberForwardingEnabled, gen, rep.ForwardingBlockReasons, rep.Errors)
		}
		for _, check := range rep.Checks {
			if check.Status != api.HealthCheckStatusPassed {
				t.Fatalf("pending delete failed check: %+v", check)
			}
		}
		if len(rep.Errors) != 1 || rep.Errors[0].Code != api.ErrorCode(enforce.ReasonIsolation) || rep.Errors[0].Stage != "apply" || rep.Errors[0].Message != "conntrack delete pending; serving ruleset verified" {
			t.Fatalf("pending entry: %+v", rep.Errors)
		}
		if beat := mustBeat(t, r); !beat.Serving || beat.Reason != "serving" || beat.Generation != 3 {
			t.Fatalf("pending heartbeat: %+v", beat)
		}
	}
	assertPending()
	stored, ok, err := r.eng.store.Desired(context.Background())
	if err != nil || !ok || !stored.HasApplied || stored.Applied != 3 {
		t.Fatalf("generation not durably acknowledged: %+v %v", stored, err)
	}
	// Reload durable runtime state as a replacement process would. The new
	// binding addresses alone no longer identify the pending old address.
	r.eng.pendingRevoke = nil
	if err := r.eng.loadRuntime(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(p.ex.deletes) != 1 {
		t.Fatalf("delete retried within tick: %v", p.ex.deletes)
	}
	for range 2 {
		applies, deletes := len(p.models), len(p.ex.deletes)
		stepOK(t, r)
		assertPending()
		if len(p.models) != applies+1 || len(p.ex.deletes) != deletes+1 {
			t.Fatal("retry did not wait for next apply tick")
		}
	}
	// Renewal remains possible while the pending entry is present.
	renewals := r.srv.LeaseRenewals()
	r.eng.cfg.RenewLeadSeconds = 3600
	advance := r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt) * 3 / 4
	r.clock.Advance(advance)
	r.srv.SetClockOffset(advance)
	stepOK(t, r)
	assertPending()
	if r.srv.LeaseRenewals() != renewals+1 {
		t.Fatal("pending delete suppressed renewal")
	}
	p.ex.failDelete = false
	stepOK(t, r)
	if !reflect.DeepEqual(p.models[len(p.models)-1].Revoke, wantRevoke) {
		t.Fatal("successful retry lost revoke list")
	}
	r.eng.lastObs = time.Time{}
	if err := r.eng.observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	rep, _ := r.srv.LastObservation()
	if len(rep.Errors) != 0 || !rep.MemberForwardingEnabled || !r.eng.decision.Serve {
		t.Fatalf("successful delete did not clear pending entry: %+v", rep.Errors)
	}
	if err := r.eng.loadRuntime(context.Background()); err != nil {
		t.Fatal(err)
	}
	stepOK(t, r)
	if len(p.models[len(p.models)-1].Revoke) != 0 {
		t.Fatal("successful deletes remained queued")
	}
}

func TestISO01_UnverifiedConntrackFailureStillRestoresBaseline(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane, unverified: true, ex: renderedExecutor{revocationExecutor: revocationExecutor{failDelete: true}}}
	r.eng.plane = p
	binding := oneBinding(r.clock.Wall())
	binding.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, binding))
	stepOK(t, r)
	// Apply verified the rendered ruleset, then the delete failed, so the
	// error is *enforce.ConntrackDeleteError. This plane clears Verified.
	// The agent restores the baseline because an unverified result outranks
	// that pending delete.
	if !enforce.IsConntrackDeleteFailure(p.errors[0]) || p.results[0].Verified || r.eng.serving || r.eng.decision.Serve || p.baselinesAfterApply == 0 || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonSafetyCheckFailed) {
		t.Fatalf("unverified typed delete failure did not fail closed: verified=%v err=%v serving=%v baselines=%d reasons=%v", p.results[0].Verified, p.errors[0], r.eng.serving, p.baselinesAfterApply, r.eng.decision.Reasons)
	}
}

func TestISO01_VerifyFailureStillRestoresBaseline(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane, ex: renderedExecutor{revocationExecutor: revocationExecutor{failVerify: true}}}
	r.eng.plane = p
	stepOK(t, r)
	if p.results[0].Verified || p.errors[0] == nil || enforce.IsConntrackDeleteFailure(p.errors[0]) || r.eng.serving || r.eng.decision.Serve || p.baselinesAfterApply == 0 {
		t.Fatalf("verify failure did not fail closed: result=%+v err=%v", p.results[0], p.errors[0])
	}
	if len(p.ex.scripts) < 2 || !strings.Contains(p.ex.scripts[len(p.ex.scripts)-1], "fvpn_baseline=1") || len(p.ex.deletes) != 0 {
		t.Fatal("enforce did not restore baseline before any delete")
	}
}

func TestLEASE01_RetryAfterPreservesFreshLeaseUntilDeadline(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(map[bool]string{false: "renew", true: "expire"}[expire], func(t *testing.T) {
			r := newRig(t)
			serveBoundGeneration(t, r)
			r.eng.cfg.RenewLeadSeconds = 3600
			advance := r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt) * 3 / 4
			r.clock.Advance(advance)
			r.srv.SetClockOffset(advance)
			wait := 30 * time.Second
			if expire {
				wait = r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt) + time.Minute
			}
			r.srv.SetLeaseRetryAfter(1, int(wait/time.Second))
			count := r.srv.LeaseRenewals()
			assertServing := func() {
				t.Helper()
				lease, _ := checkBy(r.eng.decision.Checks, api.NodeCheckNameLease)
				if !r.eng.serving || !r.eng.decision.Serve || lease.Status != api.HealthCheckStatusPassed || lease.Detail == nil || !strings.Contains(*lease.Detail, "Retry-After") || mustBeat(t, r).Reason != "serving" {
					t.Fatalf("backoff invalidated fresh lease: serving=%v reasons=%v check=%+v", r.eng.serving, r.eng.decision.Reasons, lease)
				}
				if r.srv.LeaseRenewals() != count+1 {
					t.Fatal("renewed before Retry-After")
				}
			}
			stepOK(t, r)
			assertServing()
			stepOK(t, r)
			assertServing()
			if expire {
				deadline := r.eng.leaseMono + r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt)
				r.clock.Advance(deadline - r.clock.Mono())
				stepOK(t, r)
				lease, _ := checkBy(r.eng.decision.Checks, api.NodeCheckNameLease)
				if r.eng.serving || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonLeaseExpired) || lease.Status != api.HealthCheckStatusFailed || codeOf(t, lease) != api.ErrorCodeLEASEEXPIRED || r.srv.LeaseRenewals() != count+1 {
					t.Fatal("backoff extended expired lease or retried early")
				}
				return
			}
			r.clock.Advance(wait - time.Second)
			stepOK(t, r)
			assertServing()
			r.clock.Advance(time.Second)
			stepOK(t, r)
			if r.srv.LeaseRenewals() != count+2 || !r.eng.serving || r.eng.leaseRetryAt != 0 {
				t.Fatal("renewal did not recover at Retry-After")
			}
		})
	}
}

// renderedExecutor answers nft -j list with the table the installed scripts
// leave behind. Each script is the one enforce.Render just handed to NftBatch,
// so a later verifier change that still accepts Render keeps these tests green.
// failVerify still returns an empty listing. Conntrack and tc qdisc/class
// listings stay on the embedded executor; filters and policy rules are explicit.
type renderedExecutor struct {
	revocationExecutor
	folded    int
	table     listedTable
	foreignTC bool
}

func (ex *renderedExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if (name == "tc" && len(args) > 1 && args[0] == "-j" && args[1] == "filter") || name == "ip" {
		if ex.foreignTC && name == "tc" && strings.Join(args, " ") == "-j filter show dev "+ex.model.Datapath.MemberIngress+" ingress" {
			return []byte(`[{"kind":"matchall","options":{"handle":1,"actions":[{"kind":"mirred"}]}}]`), nil
		}
		return []byte(`[]`), nil
	}
	if name == "nft" && ex.installed && !ex.failVerify {
		if err := ex.fold(); err != nil {
			return nil, err
		}
		return ex.table.listingJSON()
	}
	return ex.revocationExecutor.Output(ctx, name, args...)
}

func TestReviewR15_TCForeignFilterWithdrawsServingOnApply(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	if !r.eng.serving || !p.results[0].Verified || p.errors[0] != nil {
		t.Fatalf("clean serving control failed: %v", p.errors)
	}
	p.ex.foreignTC = true
	stepOK(t, r)
	last := len(p.results) - 1
	if r.eng.serving || p.results[last].Verified || p.errors[last] == nil || !strings.Contains(p.errors[last].Error(), "TC_FOREIGN_FILTER") || p.baselinesAfterApply == 0 {
		t.Fatalf("foreign tc did not withdraw serving: serving=%v result=%+v errors=%v baselines=%d", r.eng.serving, p.results[last], p.errors, p.baselinesAfterApply)
	}
	// The contract does not yet contain the new reason enum. Preserve the
	// specific reason in the message through the existing fallback code.
	if r.eng.applyError == nil || r.eng.applyError.Code != api.ErrorCodeHEALTHCHECKFAILED || r.eng.applyError.Message != "TC_FOREIGN_FILTER" {
		t.Fatalf("lost tc failure: %+v", r.eng.applyError)
	}
}

func (ex *renderedExecutor) fold() error {
	ex.table.init()
	for ex.folded < len(ex.scripts) {
		if err := ex.table.applyScript(ex.scripts[ex.folded]); err != nil {
			return err
		}
		ex.folded++
	}
	return nil
}

type listedCounter struct {
	packets uint64
	bytes   uint64
}

type listedChain struct {
	typ    string
	hook   string
	policy string
	prio   int64
	rules  []listedRule
}

type listedRule struct {
	comment string
	expr    []any
}

type listedSet struct {
	flags []string
	elems []string
}

type listedTable struct {
	counters     map[string]listedCounter
	counterOrder []string
	sets         map[string]*listedSet
	setOrder     []string
	chains       map[string]*listedChain
	chainOrder   []string
}

func (t *listedTable) init() {
	if t.counters != nil {
		return
	}
	t.counters = map[string]listedCounter{}
	t.sets = map[string]*listedSet{}
	t.chains = map[string]*listedChain{}
}

func (t *listedTable) applyScript(script string) error {
	for _, line := range strings.Split(script, "\n") {
		if err := t.applyLine(line); err != nil {
			return err
		}
	}
	return nil
}

func (t *listedTable) applyLine(line string) error {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	tok := tokenizeNFT(line)
	if len(tok) < 2 {
		return fmt.Errorf("nft listing: short line %q", line)
	}
	op := tok[0] + " " + tok[1]
	if op == "add table" {
		return nil
	}
	if len(tok) < 5 || tok[2] != "inet" || tok[3] != "fvpn" {
		return fmt.Errorf("nft listing: unrecognized line %q", line)
	}
	name := tok[4]
	switch op {
	case "add counter":
		if _, ok := t.counters[name]; ok {
			return nil
		}
		// Stable readings so a later tick's verified delete failure does not
		// look like a meter reset or a discarded snapshot.
		t.counters[name] = listedCounter{packets: 1, bytes: 100}
		t.counterOrder = append(t.counterOrder, name)
		return nil
	case "add set":
		body, err := braceBody(line)
		if err != nil {
			return err
		}
		t.sets[name] = &listedSet{flags: setFlags(body)}
		t.setOrder = append(t.setOrder, name)
		return nil
	case "add element":
		set := t.sets[name]
		if set == nil {
			return fmt.Errorf("nft listing: element for missing set %s", name)
		}
		body, err := braceBody(line)
		if err != nil {
			return err
		}
		set.elems = append(set.elems, splitElems(body)...)
		return nil
	case "flush set":
		set := t.sets[name]
		if set == nil {
			return fmt.Errorf("nft listing: flush of missing set %s", name)
		}
		set.elems = nil
		return nil
	case "delete set":
		delete(t.sets, name)
		t.setOrder = dropName(t.setOrder, name)
		return nil
	case "delete chain":
		delete(t.chains, name)
		t.chainOrder = dropName(t.chainOrder, name)
		return nil
	case "add chain":
		body, err := braceBody(line)
		if err != nil {
			return err
		}
		ch, err := parseChainSpec(body)
		if err != nil {
			return err
		}
		t.chains[name] = ch
		t.chainOrder = append(t.chainOrder, name)
		return nil
	case "add rule":
		ch := t.chains[name]
		if ch == nil {
			return fmt.Errorf("nft listing: rule in missing chain %s", name)
		}
		comment, expr, err := parseRule(tok[5:])
		if err != nil {
			return fmt.Errorf("nft listing: %s: %w", line, err)
		}
		ch.rules = append(ch.rules, listedRule{comment: comment, expr: expr})
		return nil
	default:
		return fmt.Errorf("nft listing: unrecognized line %q", line)
	}
}

func (t *listedTable) listingJSON() ([]byte, error) {
	doc := make([]any, 0, len(t.chainOrder)+len(t.counterOrder)+len(t.setOrder))
	for _, name := range t.chainOrder {
		ch := t.chains[name]
		doc = append(doc, map[string]any{"chain": map[string]any{
			"family": "inet", "table": "fvpn", "name": name,
			"type": ch.typ, "hook": ch.hook, "prio": ch.prio, "policy": ch.policy,
		}})
		for _, rule := range ch.rules {
			obj := map[string]any{
				"family": "inet", "table": "fvpn", "chain": name, "expr": rule.expr,
			}
			if rule.comment != "" {
				obj["comment"] = rule.comment
			}
			doc = append(doc, map[string]any{"rule": obj})
		}
	}
	for _, name := range t.counterOrder {
		c := t.counters[name]
		doc = append(doc, map[string]any{"counter": map[string]any{
			"family": "inet", "table": "fvpn", "name": name,
			"packets": c.packets, "bytes": c.bytes,
		}})
	}
	for _, name := range t.setOrder {
		set := t.sets[name]
		obj := map[string]any{"family": "inet", "table": "fvpn", "name": name}
		if len(set.flags) > 0 {
			obj["flags"] = set.flags
		}
		if len(set.elems) > 0 {
			obj["elem"] = set.elems
		}
		doc = append(doc, map[string]any{"set": obj})
	}
	return json.Marshal(map[string]any{"nftables": doc})
}

func tokenizeNFT(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote && (c == ' ' || c == '\t') {
			if b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
			}
			continue
		}
		b.WriteByte(c)
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

func braceBody(line string) (string, error) {
	i := strings.IndexByte(line, '{')
	j := strings.LastIndexByte(line, '}')
	if i < 0 || j < i {
		return "", fmt.Errorf("nft listing: missing braces in %q", line)
	}
	return strings.TrimSpace(line[i+1 : j]), nil
}

func setFlags(spec string) []string {
	var flags []string
	if strings.Contains(spec, "flags timeout") {
		flags = append(flags, "timeout")
	}
	if strings.Contains(spec, "flags interval") {
		flags = append(flags, "interval")
	}
	return flags
}

func splitElems(body string) []string {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	parts := strings.Split(body, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func dropName(names []string, name string) []string {
	out := names[:0]
	for _, n := range names {
		if n != name {
			out = append(out, n)
		}
	}
	return out
}

func parseChainSpec(spec string) (*listedChain, error) {
	f := strings.Fields(strings.ReplaceAll(spec, ";", " "))
	ch := &listedChain{}
	seenPrio := false
	for i := 0; i < len(f); i++ {
		if i+1 >= len(f) {
			return nil, fmt.Errorf("nft listing: chain spec %q", spec)
		}
		switch f[i] {
		case "type":
			ch.typ = f[i+1]
		case "hook":
			ch.hook = f[i+1]
		case "policy":
			ch.policy = f[i+1]
		case "priority":
			n, err := strconv.ParseInt(f[i+1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("nft listing: chain priority %q", f[i+1])
			}
			ch.prio = n
			seenPrio = true
		default:
			return nil, fmt.Errorf("nft listing: chain spec %q", spec)
		}
		i++
	}
	if ch.typ == "" || ch.hook == "" || ch.policy == "" || !seenPrio {
		return nil, fmt.Errorf("nft listing: chain spec %q", spec)
	}
	return ch, nil
}

func parseRule(tok []string) (string, []any, error) {
	var comment string
	var expr []any
	for i := 0; i < len(tok); i++ {
		switch tok[i] {
		case "iifname", "oifname":
			if i+1 >= len(tok) {
				return "", nil, fmt.Errorf("missing %s", tok[i])
			}
			expr = append(expr, matchMeta("==", tok[i], tok[i+1]))
			i++
		case "meta":
			if i+2 < len(tok) && tok[i+1] == "nfproto" {
				expr = append(expr, matchMeta("==", "nfproto", tok[i+2]))
				i += 2
				continue
			}
			if i+3 < len(tok) && tok[i+1] == "priority" && tok[i+2] == "set" {
				expr = append(expr, map[string]any{"mangle": map[string]any{
					"key":   map[string]any{"meta": map[string]any{"key": "priority"}},
					"value": tok[i+3],
				}})
				i += 3
				continue
			}
			return "", nil, fmt.Errorf("meta %q", strings.Join(tok[i:], " "))
		case "ct":
			if i+2 >= len(tok) || tok[i+1] != "state" {
				return "", nil, fmt.Errorf("ct")
			}
			rest := i + 2
			op := "=="
			if tok[rest] == "!=" {
				op = "!="
				rest++
			}
			right, last, err := ctRight(tok, rest)
			if err != nil {
				return "", nil, err
			}
			expr = append(expr, map[string]any{"match": map[string]any{
				"op":    op,
				"left":  map[string]any{"ct": map[string]any{"key": "state"}},
				"right": right,
			}})
			i = last
		case "ip":
			if i+2 >= len(tok) {
				return "", nil, fmt.Errorf("ip")
			}
			field := tok[i+1]
			rest := i + 2
			if tok[rest] == "." {
				for rest < len(tok) && !strings.HasPrefix(tok[rest], "@") {
					rest++
				}
				if rest >= len(tok) {
					return "", nil, fmt.Errorf("concat set")
				}
				expr = append(expr, map[string]any{"match": map[string]any{
					"op":    "==",
					"left":  map[string]any{"concat": []any{field, "l4proto", "dport"}},
					"right": tok[rest],
				}})
				i = rest
				continue
			}
			op := "=="
			if tok[rest] == "!=" {
				op = "!="
				rest++
				if rest >= len(tok) {
					return "", nil, fmt.Errorf("negated ip match")
				}
			}
			expr = append(expr, map[string]any{"match": map[string]any{
				"op":    op,
				"left":  map[string]any{"payload": map[string]any{"protocol": "ip", "field": field}},
				"right": tok[rest],
			}})
			i = rest
		case "tcp", "udp":
			if i+2 >= len(tok) {
				return "", nil, fmt.Errorf("%s port", tok[i])
			}
			port, err := strconv.Atoi(tok[i+2])
			if err != nil {
				return "", nil, fmt.Errorf("%s port %q", tok[i], tok[i+2])
			}
			expr = append(expr, map[string]any{"match": map[string]any{
				"op":    "==",
				"left":  map[string]any{"payload": map[string]any{"protocol": tok[i], "field": tok[i+1]}},
				"right": port,
			}})
			i += 2
		case "counter":
			if i+2 >= len(tok) || tok[i+1] != "name" {
				return "", nil, fmt.Errorf("counter")
			}
			expr = append(expr, map[string]any{"counter": map[string]any{"name": tok[i+2]}})
			i += 2
		case "accept":
			expr = append(expr, map[string]any{"accept": nil})
		case "drop":
			expr = append(expr, map[string]any{"drop": nil})
		case "snat":
			if i+3 >= len(tok) || tok[i+1] != "ip" || tok[i+2] != "to" {
				return "", nil, fmt.Errorf("snat")
			}
			expr = append(expr, map[string]any{"snat": map[string]any{"addr": tok[i+3]}})
			i += 3
		case "comment":
			if i+1 >= len(tok) {
				return "", nil, fmt.Errorf("comment")
			}
			comment = tok[i+1]
			i++
		default:
			return "", nil, fmt.Errorf("token %q", tok[i])
		}
	}
	return comment, expr, nil
}

func ctRight(tok []string, i int) (any, int, error) {
	if i >= len(tok) {
		return nil, i, fmt.Errorf("ct state")
	}
	if tok[i] != "{" {
		raw := strings.Trim(tok[i], ",")
		if !strings.Contains(raw, ",") {
			return raw, i, nil
		}
		var states []any
		for _, p := range strings.Split(raw, ",") {
			if p != "" {
				states = append(states, p)
			}
		}
		return states, i, nil
	}
	var states []any
	i++
	for i < len(tok) && tok[i] != "}" {
		name := strings.Trim(tok[i], ",")
		if name != "" {
			states = append(states, name)
		}
		i++
	}
	if i >= len(tok) || tok[i] != "}" {
		return nil, i, fmt.Errorf("unclosed ct state")
	}
	return states, i, nil
}

func matchMeta(op, key, right string) map[string]any {
	return map[string]any{"match": map[string]any{
		"op":    op,
		"left":  map[string]any{"meta": map[string]any{"key": key}},
		"right": right,
	}}
}

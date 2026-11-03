package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/netip"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
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
	p.models = append(p.models, m)
	m = kernelClockModel(m)
	p.ex.model = m
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
// leave behind, committing only complete successful batches. TC readback comes
// from commands applied to the embedded executor, never from the desired model.
// failVerify still returns an empty listing; filters and policy rules are explicit.
type renderedExecutor struct {
	revocationExecutor
	folded             int
	table              listedTable
	foreignTC          bool
	linkDown           bool
	allowedInstalledAt time.Time
}

// The kernel decrements remaining element lifetime after installation. Keep
// the original timeout for verification, but report a decaying expires value.
// Sampling at NftBatch (not lazy fold/read) also models slow consumers.
func (ex *renderedExecutor) NftBatch(ctx context.Context, script string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ex.fold(); err != nil {
		return err
	}
	at := time.Now()
	if err := ex.table.applyScript(script); err != nil {
		return err
	}
	if err := ex.revocationExecutor.NftBatch(ctx, script); err != nil {
		return err
	}
	ex.folded = len(ex.scripts)
	if strings.Contains(script, "add element inet fvpn allowed_v4") || strings.Contains(script, "flush set inet fvpn allowed_v4") {
		ex.allowedInstalledAt = at
	}
	return nil
}

func (ex *renderedExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "ip" && len(args) == 5 && args[0] == "link" && args[1] == "set" {
		ex.linkDown = args[4] == "down"
		return nil, nil
	}
	if name == "ip" && len(args) == 5 && args[0] == "-j" && args[1] == "link" {
		flags := []string{"UP"}
		if ex.linkDown {
			flags = []string{}
		}
		return json.Marshal([]any{map[string]any{"ifname": args[4], "flags": flags, "operstate": "UNKNOWN"}})
	}
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
		if strings.Join(args, " ") == "list set inet fvpn allowed_v4" {
			set := ex.table.sets["allowed_v4"]
			if set == nil {
				return nil, fmt.Errorf("missing allowed_v4")
			}
			body := ""
			var elements []string
			for _, element := range set.elems {
				fields := strings.Fields(element)
				if len(fields) != 3 || fields[1] != "timeout" || ex.allowedInstalledAt.IsZero() {
					return nil, fmt.Errorf("invalid timed fixture element")
				}
				timeout, err := time.ParseDuration(fields[2])
				if err != nil {
					return nil, err
				}
				remaining := timeout - time.Since(ex.allowedInstalledAt)
				if remaining <= 0 {
					continue
				}
				elements = append(elements, fmt.Sprintf("%s expires %dms", element, remaining/time.Millisecond))
			}
			if len(elements) > 0 {
				body = " elements = { " + strings.Join(elements, ", ") + " }"
			}
			return []byte("table inet fvpn { set allowed_v4 { type ipv4_addr flags timeout" + body + " } }"), nil
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
	requireApplyChangeForTest(r)
	stepOK(t, r)
	last := len(p.results) - 1
	if r.eng.serving || p.results[last].Verified || p.errors[last] == nil || !strings.Contains(p.errors[last].Error(), "TC_FOREIGN_FILTER") || p.baselinesAfterApply == 0 {
		t.Fatalf("foreign tc did not withdraw serving: serving=%v result=%+v errors=%v baselines=%d", r.eng.serving, p.results[last], p.errors, p.baselinesAfterApply)
	}
	// The contract does not yet contain the new reason enum. Preserve the
	// specific reason in the message through the existing fallback code.
	if r.eng.applyError == nil || r.eng.applyError.Code != api.ErrorCodeTCFOREIGNFILTER || r.eng.applyError.Message != "TC_FOREIGN_FILTER" {
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
	exists       bool
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
	// Rules' parsed expressions are immutable. Clone every map, object and
	// mutable slice so a late error cannot expose an earlier statement.
	next := *t
	next.counters = maps.Clone(t.counters)
	next.counterOrder = slices.Clone(t.counterOrder)
	next.setOrder = slices.Clone(t.setOrder)
	next.chainOrder = slices.Clone(t.chainOrder)
	next.sets = make(map[string]*listedSet, len(t.sets))
	for name, set := range t.sets {
		next.sets[name] = &listedSet{flags: slices.Clone(set.flags), elems: slices.Clone(set.elems)}
	}
	next.chains = make(map[string]*listedChain, len(t.chains))
	for name, chain := range t.chains {
		copy := *chain
		copy.rules = slices.Clone(chain.rules)
		next.chains[name] = &copy
	}
	next.init()
	for _, line := range strings.Split(script, "\n") {
		if err := next.applyLine(line); err != nil {
			return err
		}
	}
	*t = next
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
		if len(tok) != 4 || tok[2] != "inet" || tok[3] != "fvpn" {
			return fmt.Errorf("nft listing: unrecognized table %q", line)
		}
		t.exists = true
		return nil
	}
	if !t.exists {
		return fmt.Errorf("nft listing: missing table")
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
		if t.sets[name] != nil {
			return nil
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
		if t.sets[name] == nil {
			return fmt.Errorf("nft listing: delete missing set %s", name)
		}
		delete(t.sets, name)
		t.setOrder = dropName(t.setOrder, name)
		return nil
	case "flush chain":
		ch := t.chains[name]
		if ch == nil {
			return fmt.Errorf("nft listing: flush missing chain %s", name)
		}
		ch.rules = nil
		return nil
	case "delete chain":
		if t.chains[name] == nil {
			return fmt.Errorf("nft listing: delete missing chain %s", name)
		}
		delete(t.chains, name)
		t.chainOrder = dropName(t.chainOrder, name)
		return nil
	case "add chain":
		var body string
		if len(tok) != 5 {
			var err error
			body, err = braceBody(line)
			if err != nil {
				return err
			}
		}
		ch, err := parseChainSpec(body)
		if err != nil {
			return err
		}
		if t.chains[name] != nil {
			return nil
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
	if t.exists {
		doc = append(doc, map[string]any{"table": map[string]any{"family": "inet", "name": "fvpn"}})
	}
	for _, name := range t.chainOrder {
		ch := t.chains[name]
		obj := map[string]any{
			"family": "inet", "table": "fvpn", "name": name,
		}
		if ch.hook != "" {
			obj["type"], obj["hook"], obj["prio"], obj["policy"] = ch.typ, ch.hook, ch.prio, ch.policy
		}
		doc = append(doc, map[string]any{"chain": obj})
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
		if name == "denied_v4" || name == "host_v4" {
			obj["type"] = "ipv4_addr"
			// Match nft JSON's prefix/address shapes. Keep the original
			// decomposition; verification compares coverage after merging.
			elems := make([]any, 0, len(set.elems))
			for _, text := range set.elems {
				p, err := netip.ParsePrefix(text)
				if err != nil {
					return nil, fmt.Errorf("invalid isolation fixture prefix: %w", err)
				}
				if p.Bits() == 32 {
					elems = append(elems, p.Addr().String())
				} else {
					elems = append(elems, map[string]any{"prefix": map[string]any{"addr": p.Masked().Addr().String(), "len": p.Bits()}})
				}
			}
			obj["elem"] = elems
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
	if len(f) == 0 {
		return ch, nil
	}
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
				// The renderer writes a hexadecimal integer; nft read-back
				// uses a JSON integer or major:minor, not that hex string.
				value, err := strconv.ParseUint(tok[i+3], 0, 32)
				if err != nil {
					return "", nil, fmt.Errorf("meta priority %q", tok[i+3])
				}
				expr = append(expr, map[string]any{"mangle": map[string]any{
					"key":   map[string]any{"meta": map[string]any{"key": "priority"}},
					"value": value,
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
			// nft 1.0.9 lists the implicit singleton RELATED test as "in".
			// Preserve the existing negated established/related guard.
			if state, ok := right.(string); ok && state == "related" && op == "==" {
				op = "in"
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
		case "icmp":
			if i+2 >= len(tok) || (tok[i+1] != "type" && tok[i+1] != "code") {
				return "", nil, fmt.Errorf("icmp type/code match")
			}
			expr = append(expr, map[string]any{"match": map[string]any{
				"op":    "==",
				"left":  map[string]any{"payload": map[string]any{"protocol": "icmp", "field": tok[i+1]}},
				"right": tok[i+2],
			}})
			i += 2
		case "tcp", "udp":
			if tok[i] == "tcp" && i+1 < len(tok) {
				switch tok[i+1] {
				case "flags":
					if i+5 >= len(tok) || tok[i+2] != "&" || tok[i+3] != "syn" || tok[i+4] != "==" || tok[i+5] != "syn" {
						return "", nil, fmt.Errorf("tcp SYN flag test")
					}
					expr = append(expr, map[string]any{"match": map[string]any{
						"op":    "==",
						"left":  map[string]any{"&": []any{map[string]any{"payload": map[string]any{"protocol": "tcp", "field": "flags"}}, "syn"}},
						"right": "syn",
					}})
					i += 5
					continue
				case "option":
					if i+5 >= len(tok) || tok[i+2] != "maxseg" || tok[i+3] != "size" || (tok[i+4] != ">" && tok[i+4] != "set") {
						return "", nil, fmt.Errorf("tcp maxseg comparison/mangle")
					}
					value, err := strconv.ParseUint(tok[i+5], 10, 16)
					if err != nil {
						return "", nil, fmt.Errorf("tcp maxseg size %q", tok[i+5])
					}
					key := map[string]any{"tcp option": map[string]any{"name": "maxseg", "field": "size"}}
					if tok[i+4] == "set" {
						expr = append(expr, map[string]any{"mangle": map[string]any{"key": key, "value": value}})
					} else {
						expr = append(expr, map[string]any{"match": map[string]any{"op": ">", "left": key, "right": value}})
					}
					i += 5
					continue
				}
			}
			if i+2 >= len(tok) || (tok[i+1] != "sport" && tok[i+1] != "dport") {
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

// TestD096_RealNetBird is launched by tests/netlab/scripts/gates.sh.
// TLS control is the existing rig; NetBird, nft, tc, routes and packets are real.
func TestD096_RealNetBird(t *testing.T) {
	if os.Getenv("FVPN_REAL_NETBIRD") != "1" {
		t.Skip("requires isolated real-NetBird netlab")
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		t.Fatal("requires container")
	}
	raw, err := os.ReadFile("/scratch/real-netbird/peers.json")
	if err != nil {
		t.Fatal(err)
	}
	var peers map[string]string
	if err := json.Unmarshal(raw, &peers); err != nil {
		t.Fatal(err)
	}
	run := func(name string, args ...string) string {
		t.Helper()
		b, err := exec.CommandContext(t.Context(), name, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("lab command %s: %v %s", name, err, b)
		}
		return string(b)
	}
	r := newRig(t)
	r.eng.clock = timeanchor.System{}
	r.srv.SetClock(r.eng.clock.Wall)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.cfg.Datapath.MemberIngress = "fvnb0"
	r.eng.prober = &verProbe{version: "0.82.0"}
	binding := oneBinding(r.clock.Wall())
	binding.AllowedIpv4 = []string{peers["fv-member"] + "/32"}
	r.srv.SetFixture(generationFixture(3, binding))
	p := &followup14Plane{}
	r.eng.plane = p
	r.eng.memberPath = nil
	r.eng.cfg.HeartbeatSeconds, r.eng.cfg.PollSeconds = 1, 1
	r.eng.withdraw(t.Context())
	stepOK(t, r)
	if !r.eng.serving {
		t.Fatalf("real Apply failed: %+v; summary=%s", r.eng.decision.Errors, p.result.Evidence.ReportDetail())
	}
	t.Logf("initial generation=3 topology_addresses=%d topology_routes=%d tc=%s", len(r.eng.memberPath.Addresses), len(r.eng.memberPath.Routes), p.result.Evidence.ReportDetail())
	// Reproduce the isolation interface-change sequence without planted tc state.
	for cycle := range 10 {
		run("ip", "link", "add", "fvisotest", "type", "dummy")
		run("ip", "addr", "add", "198.51.100.10/32", "dev", "fvisotest")
		run("ip", "link", "set", "fvisotest", "up")
		stepOK(t, r)
		run("ip", "link", "del", "fvisotest")
		stepOK(t, r)
		if !r.eng.serving {
			t.Fatalf("ISO-F6-01 cycle=%d summary=%s errors=%+v", cycle, p.result.Evidence.ReportDetail(), r.eng.decision.Errors)
		}
	}
	t.Log("ISO-F6-01: 10 real-NetBird interface add/remove + Apply cycles passed, no foreign object planted")
	// Capture headers only on the isolated uplink. Traffic is from the real peer
	// to this node's overlay address; the planted redirect is the positive control.
	capture := exec.CommandContext(t.Context(), "tcpdump", "-l", "-nn", "-q", "-e", "-tt", "-i", "fv-ih", "ether[0] == 0x45 and ether[9] == 17 and ether[22:2] == 39997")
	var packets followup14Buffer
	capture.Stdout = &packets
	pipe, err := capture.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := capture.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = capture.Process.Kill(); _ = capture.Wait() })
	scan := bufio.NewScanner(pipe)
	ready := false
	for scan.Scan() {
		if strings.Contains(scan.Text(), "listening on") {
			ready = true
			break
		}
	}
	if !ready {
		t.Fatal("capture unavailable")
	}
	var captureStats strings.Builder
	captureDone := make(chan struct{})
	go func() {
		defer close(captureDone)
		for scan.Scan() {
			captureStats.WriteString(scan.Text() + "\n")
		}
	}()
	sender := exec.CommandContext(t.Context(), "ip", "netns", "exec", "fv-member", "python3", "-u", "-c", `import socket,time,sys
s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM)
s.bind(('',40000))
while True:
 s.sendto(b'TEST ONLY D096', (sys.argv[1],39997));time.sleep(.01)
`, peers["fv-node"])
	if err := sender.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sender.Process.Kill(); _ = sender.Wait() })
	var latencies []time.Duration
	for cycle := range 10 {
		beforePositive := packets.Count()
		run("tc", "qdisc", "add", "dev", "fvnb0", "clsact")
		run("tc", "filter", "add", "dev", "fvnb0", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
		time.Sleep(250 * time.Millisecond)
		positive := packets.Count() - beforePositive
		if positive == 0 {
			t.Fatal("real peer redirect positive control produced no packet")
		}
		stepOK(t, r)
		if mustBeat(t, r).Serving || r.eng.containment == nil || !r.eng.containment.Quarantine.Quarantined {
			t.Fatal("quarantine failed")
		}
		time.Sleep(100 * time.Millisecond) // drain capture stdout before the down interval
		downStart := packets.Count()
		var downAddresses []struct {
			Addresses []json.RawMessage `json:"addr_info"`
		}
		if err := json.Unmarshal([]byte(run("ip", "-j", "addr", "show", "dev", "fvnb0")), &downAddresses); err != nil {
			t.Fatal(err)
		}
		var downRoutes []struct {
			Flags []string `json:"flags"`
		}
		if err := json.Unmarshal([]byte(run("ip", "-j", "-4", "route", "show", "table", "all", "dev", "fvnb0")), &downRoutes); err != nil {
			t.Fatal(err)
		}
		downFlags := 0
		for _, route := range downRoutes {
			if slices.Contains(route.Flags, "linkdown") {
				downFlags++
			}
		}
		t.Logf("D096 cycle=%d while_down_addresses=%d routes4=%d linkdown_routes=%d", cycle+1, len(downAddresses[0].Addresses), len(downRoutes), downFlags)
		raises := 0
		started := time.Now()
		for time.Since(started) < 5*time.Second {
			b := run("ip", "-j", "link", "show", "dev", "fvnb0")
			var links []struct {
				Flags []string `json:"flags"`
			}
			if err := json.Unmarshal([]byte(b), &links); err != nil {
				t.Fatal(err)
			}
			if len(links) == 1 && slices.Contains(links[0].Flags, "UP") {
				raises++
			}
			time.Sleep(20 * time.Millisecond)
		}
		downPackets := packets.Count() - downStart
		_, pathErr := p.MemberPath(t.Context(), *r.eng.appliedModel, r.eng.memberPath)
		// Read the engine's actual evidence before the recovery removes it.
		t.Logf("D096 cycle=%d positive_redirect_frames=%d down_ms=%d link_up_samples=%d down_uplink_packets=%d member_path_check=%v heartbeat_serving=%t containment=%s", cycle+1, positive, time.Since(started).Milliseconds(), raises, downPackets, pathErr, mustBeat(t, r).Serving, r.eng.containmentJSON())
		if raises != 0 || downPackets != 0 {
			t.Fatalf("unexpected natural link raise or bypass; packets=%s", packets.String())
		}
		run("tc", "filter", "del", "dev", "fvnb0", "ingress", "pref", "1")
		run("tc", "qdisc", "del", "dev", "fvnb0", "clsact")
		p.raised = time.Time{}
		p.raiseStarted = time.Time{}
		stepOK(t, r)
		if !mustBeat(t, r).Serving {
			t.Fatalf("recovery stayed blocked: %+v", r.eng.decision.Errors)
		}
		if p.raiseStarted.IsZero() || p.pathOK.Before(p.raised) {
			t.Fatal("missing link-up/topology timing")
		}
		latencies = append(latencies, p.pathOK.Sub(p.raiseStarted))
		t.Logf("D096 cycle=%d raise_command_to_topology_verified_us=%d addresses=%d routes=%d", cycle+1, latencies[len(latencies)-1].Microseconds(), len(r.eng.memberPath.Addresses), len(r.eng.memberPath.Routes))
	}
	slices.Sort(latencies)
	t.Logf("D096 cycles=10 natural_raise_cycles=0 quarantine_observation_seconds=50 poll_ms=20 raise_command_to_path_median_us=%d worst_us=%d (upper bounds including verification; IPv4 retained; no harness route/address repair)", (latencies[4]+latencies[5]).Microseconds()/2, latencies[9].Microseconds())
	// A deliberate real NetBird reconnect recreates its own link and removes
	// the foreign attachment. Cancel the deciding Step's control context:
	// deferred containment must still run, without authorizing a clean Apply.
	var raceWindows []time.Duration
	for cycle := range 10 {
		run("tc", "qdisc", "add", "dev", "fvnb0", "clsact")
		run("tc", "filter", "add", "dev", "fvnb0", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
		stepOK(t, r)
		readInput := func() uint64 {
			b := run("nft", "-j", "list", "ruleset")
			rs, err := enforce.ParseRuleset([]byte(b))
			if err != nil {
				t.Fatal(err)
			}
			return rs.Counters["ctin"].Packets
		}
		time.Sleep(100 * time.Millisecond)
		packetStart := packets.Count()
		inputBefore := readInput()
		run("netbird", "--daemon-addr", "unix:///scratch/real-netbird/node.sock", "down")
		lastDown := time.Now()
		reconnectStart := lastDown
		up := exec.CommandContext(t.Context(), "netbird", "--daemon-addr", "unix:///scratch/real-netbird/node.sock", "up")
		if err := up.Start(); err != nil {
			t.Fatal(err)
		}
		var firstUp time.Time
		for time.Since(reconnectStart) < 15*time.Second {
			sampledAt := time.Now()
			raw, _ := exec.CommandContext(t.Context(), "ip", "-j", "link", "show", "dev", "fvnb0").Output()
			var links []struct {
				Flags []string `json:"flags"`
			}
			if json.Unmarshal(raw, &links) == nil && len(links) == 1 && slices.Contains(links[0].Flags, "UP") {
				firstUp = time.Now()
				break
			}
			lastDown = sampledAt
			time.Sleep(20 * time.Millisecond)
		}
		if err := up.Wait(); err != nil {
			t.Fatal("NetBird reconnect failed")
		}
		if firstUp.IsZero() {
			t.Fatal("NetBird did not raise link")
		}
		// Interface creation precedes peer handshake readiness. Keep capture
		// running throughout and require real ingress before evaluating this
		// window; silence during reconnect cannot prove baseline containment.
		trafficDeadline := time.Now().Add(10 * time.Second)
		for readInput() <= inputBefore && time.Now().Before(trafficDeadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if readInput() <= inputBefore {
			t.Fatal("real peer did not reconnect within traffic readiness bound")
		}
		trafficReady := time.Now()
		_, pathErr := p.MemberPath(t.Context(), *r.eng.appliedModel, r.eng.memberPath)
		// Read current VerifyServing too: a topology pass is not an admission pass.
		_, serveErr := p.VerifyServing(t.Context(), *r.eng.appliedModel, r.eng.appliedAssignments)
		p.lowered = time.Time{}
		time.Sleep(time.Second)
		canceled, cancel := context.WithCancel(t.Context())
		cancel()
		if err := r.eng.Step(canceled); err != context.Canceled {
			t.Fatalf("canceled Step: %v", err)
		}
		end := p.lowered
		if end.IsZero() || mustBeat(t, r).Serving || !r.eng.containment.Quarantine.Quarantined {
			t.Fatal("reconnect was not re-contained")
		}
		time.Sleep(100 * time.Millisecond)
		frames := packets.Count() - packetStart
		inputDrops := readInput() - inputBefore
		raceWindows = append(raceWindows, p.lowerStarted.Sub(firstUp))
		t.Logf("D096 reconnect_cycle=%d link_raise_window_lower_us=%d upper_us=%d link_up_to_traffic_ready_us=%d d096_during_window=%v serving_verify=%v unmetered_frames=%d input_drop_packets=%d heartbeat_serving=%t quarantine_before_up=%t quarantine_after_up=%t", cycle+1, p.lowerStarted.Sub(firstUp).Microseconds(), end.Sub(lastDown).Microseconds(), trafficReady.Sub(firstUp).Microseconds(), pathErr, serveErr, frames, inputDrops, mustBeat(t, r).Serving, r.eng.containment.Quarantine.Links[0].Before.AdminUp, r.eng.containment.Quarantine.Links[0].After.AdminUp)
		if inputDrops == 0 {
			t.Fatal("no real peer ingress during reconnect window")
		}
		if frames != 0 {
			t.Fatalf("reconnect baseline leaked: %s", packets.String())
		}
		stepOK(t, r)
		if !mustBeat(t, r).Serving {
			t.Fatalf("post-reconnect recovery failed: errors=%+v reasons=%+v tc=%s containment=%s", r.eng.decision.Errors, r.eng.decision.Reasons, p.result.Evidence.ReportDetail(), r.eng.containmentJSON())
		}
	}
	slices.Sort(raceWindows)
	t.Logf("D096 deliberate_reconnects=10 raise_cycles=10 window_median_us=%d worst_us=%d cadence=1s step_context=canceled foreign_attachment_removed_by_NetBird=true", (raceWindows[4]+raceWindows[5]).Microseconds()/2, raceWindows[9].Microseconds())
	_ = sender.Process.Kill()
	_ = sender.Wait()
	if err := capture.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case <-captureDone:
	case <-time.After(3 * time.Second):
		t.Fatal("capture did not flush")
	}
	if err := capture.Wait(); err != nil {
		t.Fatal(err)
	}
	captured := regexp.MustCompile(`(?m)^(\d+) packets captured$`).FindStringSubmatch(captureStats.String())
	dropped := regexp.MustCompile(`(?m)^(\d+) packets dropped by kernel$`).FindStringSubmatch(captureStats.String())
	if len(captured) != 2 || len(dropped) != 2 || dropped[1] != "0" {
		t.Fatal("capture completeness unproved")
	}
	n, err := strconv.Atoi(captured[1])
	if err != nil || n != packets.Count() {
		t.Fatal("capture header count mismatch")
	}
	t.Logf("D096 capture_complete=true captured_redirect_frames=%d kernel_capture_drops=0 packet=TEST_ONLY_IPv4_UDP_40000_to_39997_length42 raw_IPv4_on_Ethernet=true", n)

}

type followup14Plane struct {
	SystemPlane
	result       enforce.ApplyResult
	raised       time.Time
	raiseStarted time.Time
	lowerStarted time.Time
	lowered      time.Time
	pathOK       time.Time
}

func (p *followup14Plane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	p.SystemPlane.Executor = &followup14Executor{plane: p}
	res, err := p.SystemPlane.Apply(ctx, m, prev)
	p.result = res
	return res, err
}
func (p *followup14Plane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	path, err := p.SystemPlane.MemberPath(ctx, m, expected)
	if err == nil {
		p.pathOK = time.Now()
	}
	return path, err
}

type followup14Executor struct {
	enforce.System
	plane *followup14Plane
}

func (ex *followup14Executor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	started := time.Now()
	b, err := ex.System.Output(ctx, name, args...)
	if name == "ip" && strings.Join(args, " ") == "link set dev fvnb0 up" && err == nil {
		ex.plane.raised = time.Now()
		ex.plane.raiseStarted = started
	}
	if name == "ip" && strings.Join(args, " ") == "link set dev fvnb0 down" && err == nil && ex.plane.lowered.IsZero() {
		ex.plane.lowered = time.Now()
		ex.plane.lowerStarted = started
	}

	return b, err
}

type followup14Buffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (b *followup14Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}
func (b *followup14Buffer) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	count := 0
	for _, line := range strings.Split(b.b.String(), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			if _, err := strconv.ParseFloat(fields[0], 64); err == nil {
				count++
			}
		}
	}
	return count
}
func (b *followup14Buffer) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.b.String() }

// Regression for the reproduced agent-core 23 recovery latch.
func TestISO_F6_RetainedFailureBlocksNewGeneration(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	p.ex.foreignTC = true
	requireApplyChangeForTest(r)
	stepOK(t, r)
	if r.eng.applyError == nil || r.eng.applyError.Code != api.ErrorCodeTCFOREIGNFILTER {
		t.Fatal("fixture did not reject tc")
	}
	p.ex.foreignTC = false
	// The reported capacity update was after the tc rejection. Model that order.
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	applies, renewals := len(p.results), r.srv.LeaseRenewals()
	for range 5 {
		r.clock.Advance(time.Second)
		stepOK(t, r)
	}
	if !r.eng.serving || r.eng.appliedGen != 3 || r.eng.lease.ConfigGeneration != 3 || len(p.results) <= applies || r.srv.LeaseRenewals() <= renewals {
		t.Logf("errors=%v decision=%+v", p.errors, r.eng.decision)
		t.Fatalf("recovery failed: serving=%t applied=%d desired=%s lease=%s apply_calls=%d renewals=%d", r.eng.serving, r.eng.appliedGen, r.eng.desired.ConfigGeneration, r.eng.lease.ConfigGeneration, len(p.results)-applies, r.srv.LeaseRenewals()-renewals)
	}
	t.Logf("ISO-F6-01 recovered: Apply delta=%d renewal delta=%d lease=%s serving=%t", len(p.results)-applies, r.srv.LeaseRenewals()-renewals, r.eng.lease.ConfigGeneration, r.eng.serving)
}

// The crash boundary is represented by a successful kernel Apply with no
// retained_applied write. The retained record and rolled-back config agree;
// only a live verifier can discover the unrecorded installed set contents.
func TestAT13_F27_UnrecordedApplyRefusesRestartAdoption(t *testing.T) {
	for _, field := range []string{"denied", "host"} {
		t.Run(field, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			r.eng.cfg.TestMode = true
			r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24", "198.51.100.0/24"}
			r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
			r.eng.cfg.HostV4 = []string{"198.51.100.1/32"}
			mono, err := enforce.MonotonicNow()
			if err != nil {
				t.Fatal(err)
			}
			p := &f35RestartPlane{memPlane: r.plane, offset: mono - r.clock.Mono()}
			r.eng.plane = p
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			retained, ok, err := r.eng.store.Meta(t.Context(), "retained_applied")
			if err != nil || !ok {
				t.Fatal("retained model missing", err)
			}
			var stored retainedApplied
			if err := json.Unmarshal([]byte(retained), &stored); err != nil {
				t.Fatal(err)
			}
			// Quiesce the old userspace fixture, preserving its same-boot heartbeat.
			// Then install the state a process dying before SetMeta would leave.
			if err := r.eng.Close(); err != nil {
				t.Fatal(err)
			}
			changed := stored.Model
			if field == "denied" {
				changed.ExtraDeniedV4 = []netip.Prefix{netip.MustParsePrefix("203.0.113.9/32")}
			} else {
				changed.HostV4 = []netip.Prefix{netip.MustParsePrefix("198.51.100.9/32")}
			}
			res, err := p.Apply(t.Context(), changed, nil)
			if err != nil || !res.Verified {
				t.Fatalf("unrecorded Apply: %v", err)
			}
			next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: p, MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = next.Close() })
			r.eng = next
			raw, _, err := next.store.Meta(t.Context(), "retained_applied")
			if err != nil || raw != retained {
				t.Fatal("fixture changed durable retained model", err)
			}
			if err := next.loadRuntime(t.Context()); err != nil {
				t.Fatal(err)
			}
			candidate, valid := next.baselineProbeModel(r.clock.Wall(), nil)
			if !valid || !sameRestartSets(stored.Model, candidate) {
				t.Fatal("retained/candidate sets differ; not the H-2 window")
			}
			applies := len(p.models)
			if err := next.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			if next.serving || mustBeat(t, r).Serving || !next.baselineOK || len(p.models) != applies {
				t.Fatalf("kernel mismatch adopted: serving=%v baseline=%v applies=%d/%d", next.serving, next.baselineOK, len(p.models), applies)
			}
			if len(p.ex.table.sets["allowed_v4"].elems) != 0 {
				t.Fatal("refused adoption did not empty member admission")
			}
		})
	}
}

func TestAT13_F27_PeriodicLiveSetDriftWithdrawsServing(t *testing.T) {
	for _, field := range []string{"denied_v4", "host_v4"} {
		for _, change := range []string{"add", "remove", "replace"} {
			t.Run(field+"/"+change, func(t *testing.T) {
				r := newRig(t)
				alignControlClock(r)
				r.eng.cfg.TestMode = true
				r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24", "198.51.100.0/24"}
				r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
				r.eng.cfg.HostV4 = []string{"198.51.100.1/32"}
				mono, err := enforce.MonotonicNow()
				if err != nil {
					t.Fatal(err)
				}
				p := &f35RestartPlane{memPlane: r.plane, offset: mono - r.clock.Mono()}
				r.eng.plane = p
				serveBoundGeneration(t, r)
				r.manualUsage(t)
				if !r.eng.verifyCurrentServing(t.Context()) {
					t.Fatal("positive live control")
				}
				if err := p.ex.fold(); err != nil {
					t.Fatal(err)
				}
				set := p.ex.table.sets[field]
				switch change {
				case "add":
					set.elems = append(set.elems, "203.0.113.99/32")
				case "remove":
					set.elems = set.elems[1:]
				case "replace":
					set.elems[0] = "203.0.113.99/32"
				}
				applies := len(p.models)
				if r.eng.verifyCurrentServing(t.Context()) || r.eng.serving || !r.eng.baselineOK || len(p.models) != applies {
					t.Fatal("live drift did not withdraw without a new Apply")
				}
				if len(p.ex.table.sets["allowed_v4"].elems) != 0 {
					t.Fatal("withdrawal left member admission")
				}
			})
		}
	}
}

// These checks exercise the simulator itself. The matching kernel behavior is
// retained in follow-up 32's single live atomic-baseline check.
func TestAT13_F32_SimulatorNFTTransactions(t *testing.T) {
	const initial = `add table inet fvpn
add chain inet fvpn forward { type filter hook forward priority -150; policy drop; }
add rule inet fvpn forward iifname "fv-fh" accept
add set inet fvpn allowed_v4 { type ipv4_addr; flags timeout; }
add element inet fvpn allowed_v4 { 100.64.1.1 timeout 30s }
add counter inet fvpn ctbase
`
	ex := &renderedExecutor{}
	if err := ex.NftBatch(t.Context(), initial); err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		t.Helper()
		raw, err := ex.Output(t.Context(), "nft", "-j", "list", "table", "inet", "fvpn")
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	before, installedAt := snapshot(), ex.allowedInstalledAt
	// Re-adding a table, set or chain preserves contents and chain attributes.
	// A bare add of the existing base chain must not turn it into a regular one.
	if err := ex.NftBatch(t.Context(), `add table inet fvpn
add chain inet fvpn forward
add chain inet fvpn forward { type filter hook forward priority -150; policy drop; }
add set inet fvpn allowed_v4 { type ipv4_addr; flags timeout; }
add counter inet fvpn ctbase
`); err != nil {
		t.Fatal(err)
	}
	if snapshot() != before || ex.allowedInstalledAt != installedAt {
		t.Fatal("idempotent add replaced contents, attributes, counters or expiry anchor")
	}
	for _, suffix := range []string{
		"add rule inet fvpn fv_missing drop\n",
		"delete chain inet fvpn fv_missing\n",
		"delete set inet fvpn fv_missing\n",
		"unsupported statement\n",
	} {
		// Abort after mutations of every mutable object and its ordering slice.
		attempt := `flush set inet fvpn allowed_v4
add element inet fvpn allowed_v4 { 100.64.1.2 timeout 10s }
delete chain inet fvpn forward
add chain inet fvpn forward
add rule inet fvpn forward drop
add counter inet fvpn ctnew
add set inet fvpn fv_new { type ipv4_addr; }
` + suffix
		count := len(ex.scripts)
		if err := ex.NftBatch(t.Context(), attempt); err == nil {
			t.Fatal("invalid trailing statement accepted")
		}
		if snapshot() != before || ex.allowedInstalledAt != installedAt || len(ex.scripts) != count {
			t.Fatal("failed batch partly committed state, expiry anchor or successful history")
		}
	}
	cold := &renderedExecutor{}
	if err := cold.NftBatch(t.Context(), initial+"add rule inet fvpn fv_missing drop\n"); err == nil || cold.installed || cold.table.exists {
		t.Fatal("failed cold batch created a table")
	}
	if err := ex.NftBatch(t.Context(), `add chain inet fvpn fv_regular
add rule inet fvpn fv_regular drop
add chain inet fvpn fv_regular
`); err != nil {
		t.Fatal(err)
	}
	if ch := ex.table.chains["fv_regular"]; ch.hook != "" || len(ch.rules) != 1 {
		t.Fatal("bare regular-chain add lost a rule or gained a hook")
	}
	if err := ex.NftBatch(t.Context(), `delete chain inet fvpn fv_regular
add chain inet fvpn fv_regular { }
`); err != nil || len(ex.table.chains["fv_regular"].rules) != 0 {
		t.Fatal("regular-chain delete/recreate failed", err)
	}
	if err := ex.NftBatch(t.Context(), "delete chain inet fvpn fv_regular\n"); err != nil {
		t.Fatal(err)
	}
	base := enforce.Model{Datapath: enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: netip.MustParseAddr("192.0.2.37")}}
	batch, err := enforce.RenderBaseline(base, enforce.State{})
	if err != nil {
		t.Fatal(err)
	}
	if err := cold.NftBatch(t.Context(), batch); err != nil {
		t.Fatal("cold baseline", err)
	}
	raw, err := cold.Output(t.Context(), "nft", "-j", "list", "ruleset")
	if err != nil {
		t.Fatal(err)
	}
	if err := enforce.VerifyBaseline(raw, base); err != nil {
		t.Fatal("cold baseline readback", err)
	}
	for range 2 {
		if err := ex.NftBatch(t.Context(), batch); err != nil {
			t.Fatal(err)
		}
		if err := enforce.VerifyBaseline([]byte(snapshot()), base); err != nil {
			t.Fatal("self-contained baseline", err)
		}
	}
	t.Log("add preserves objects and expiry; bare chains supported; failed warm/cold batches commit nothing; repeated baseline verified")
}

// Hooks observe the actual production Apply order. Panicking only at the
// committed cut models process death without executing Apply's error cleanup.
type f32OrderExecutor struct {
	*renderedExecutor
	after func(string)
}

func (ex *f32OrderExecutor) TCBatch(ctx context.Context, script string) error {
	if err := ex.renderedExecutor.TCBatch(ctx, script); err != nil {
		return err
	}
	phase := "prepare"
	if strings.Contains(script, "class del ") {
		phase = "retire"
	}
	ex.after(phase)
	return nil
}

func (ex *f32OrderExecutor) NftBatch(ctx context.Context, script string) error {
	if err := ex.renderedExecutor.NftBatch(ctx, script); err != nil {
		return err
	}
	if strings.Contains(script, "# inet fvpn generation ") {
		ex.after("commit")
	} else if strings.TrimSpace(script) == "delete chain inet fvpn fv_deadline_guard" {
		ex.after("release")
	}
	return nil
}

func TestAT13_F32_SimulatorTCCommitAndCrashCut(t *testing.T) {
	for _, mode := range []string{"complete", "restart_apply", "baseline"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t)
			serveTwoBindings(t, r)
			old := kernelClockModel(r.plane.last)
			ex := &renderedExecutor{revocationExecutor: revocationExecutor{model: old}}
			initial, err := enforce.Apply(t.Context(), ex, old, nil)
			if err != nil || !initial.Verified || len(old.Bindings) != 2 {
				t.Fatal("two-member serving setup", err)
			}
			candidate := old
			candidate.Generation++
			removed := old.Bindings[1].MemberID
			candidate.Bindings = slices.Clone(old.Bindings[:1])
			candidate.Leases = slices.DeleteFunc(slices.Clone(old.Leases), func(l enforce.MemberLease) bool { return l.MemberID == removed })
			candidate.QoS.Members = slices.DeleteFunc(slices.Clone(old.QoS.Members), func(q enforce.MemberQoS) bool { return q.MemberID == removed })
			candidate.QoS.Members[0].Up.RateBPS /= 2
			candidate.QoS.Members[0].Down.RateBPS /= 2
			candidate.QoS.Assignments = initial.Plan.Assignments
			candidateAssignments := maps.Clone(initial.Plan.Assignments)
			delete(candidateAssignments, removed)
			ex.model = candidate
			minor := initial.Plan.Assignments[removed]
			handle := fmt.Sprintf("1:%x", minor)
			readTC := func(wantOrphan bool) map[string][2][]byte {
				t.Helper()
				out := map[string][2][]byte{}
				for _, dev := range []string{old.Datapath.MemberIngress, old.Datapath.Uplink} {
					qdisc, err := ex.Output(t.Context(), "tc", "-j", "qdisc", "show", "dev", dev)
					if err != nil {
						t.Fatal(err)
					}
					class, err := ex.Output(t.Context(), "tc", "-j", "class", "show", "dev", dev)
					if err != nil {
						t.Fatal(err)
					}
					view, err := enforce.ParseQdiscJSON(qdisc)
					if err != nil {
						t.Fatal(err)
					}
					view, err = enforce.ParseClassJSON(class, view)
					if err != nil || view.Classes[handle].Found != wantOrphan || view.Leaves[minor] != wantOrphan {
						t.Fatalf("%s orphan presence want=%v: %+v %v", dev, wantOrphan, view, err)
					}
					out[dev] = [2][]byte{qdisc, class}
				}
				return out
			}
			const crash = "TEST ONLY: process died after nft commit"
			var events []string
			wrapped := &f32OrderExecutor{renderedExecutor: ex, after: func(phase string) {
				events = append(events, phase)
				raw, err := ex.table.listingJSON()
				if err != nil {
					t.Fatal(err)
				}
				state, err := enforce.ParseState(raw)
				if err != nil {
					t.Fatal(err)
				}
				switch phase {
				case "prepare":
					readTC(true)
					if state.Generation != old.Generation || state.Classes[removed] != minor {
						t.Fatal("preparation replaced old nft references")
					}
				case "commit":
					liveTC := readTC(true)
					if state.Generation != candidate.Generation || state.Classes[removed] != 0 || strings.Contains(string(raw), fmt.Sprintf("0x%x", enforce.ClassID(minor))) {
						t.Fatal("commit retained a reference to the orphan")
					}
					if !state.Chains["fv_deadline_guard"] {
						t.Fatal("commit omitted guard")
					}
					if err := enforce.VerifyQoS(liveTC, candidate, candidateAssignments); err == nil {
						t.Fatal("serving verification accepted transitional orphan")
					}
					if mode != "complete" {
						panic(crash)
					}
				case "retire", "release":
					readTC(false)
				}
			}}
			func() {
				defer func() {
					if p := recover(); p != nil && (mode == "complete" || p != crash) {
						panic(p)
					}
				}()
				res, err := enforce.Apply(t.Context(), wrapped, candidate, nil)
				if mode != "complete" || err != nil || !res.Verified {
					t.Fatalf("apply returned instead of expected serving/crash: verified=%v err=%v", res.Verified, err)
				}
			}()
			if mode == "complete" {
				if !slices.Equal(events, []string{"prepare", "commit", "retire", "release"}) {
					t.Fatal("wrong order", events)
				}
			} else {
				if !slices.Equal(events, []string{"prepare", "commit"}) {
					t.Fatal("crash cut not observed", events)
				}
				readTC(true)
				if mode == "restart_apply" {
					res, err := enforce.Apply(t.Context(), ex, candidate, nil)
					if err != nil || !res.Verified {
						t.Fatal("recovery did not reclaim orphan and verify", err)
					}
				} else if err := enforce.ApplyBaseline(t.Context(), ex, candidate.Datapath); err != nil {
					t.Fatal("crash baseline failed", err)
				}
				readTC(false)
			}
			t.Logf("%s: observed %v; old classes survive preparation/commit; referenced classes intact; orphan reclaimed", mode, events)
		})
	}
}

func TestAT13_F32_SimulatorTCPartialFailureAndRootCleanup(t *testing.T) {
	ex := &revocationExecutor{}
	// tc -batch is not atomic: retain successful commands before an error.
	err := ex.TCBatch(t.Context(), `qdisc add dev fv-ih root handle 1: htb default fffe
class replace dev fv-ih parent 1: classid 1:1 htb rate 20mbit ceil 20mbit
class replace dev fv-ih parent 1:1 classid 1:10 htb rate 10mbit ceil 20mbit
qdisc add dev fv-ih parent 1:10 handle 10: fq_codel
class del dev fv-ih classid 1:999
`)
	if err == nil {
		t.Fatal("missing-class deletion accepted")
	}
	dev := ex.tcDevices["fv-ih"]
	if dev == nil || dev.classes["1:10"] == nil || dev.qdiscs["1:10"] == nil {
		t.Fatal("tc failure incorrectly undid earlier mutations")
	}
	if _, err := ex.Output(t.Context(), "tc", "qdisc", "del", "dev", "fv-ih", "root"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"qdisc", "class"} {
		raw, err := ex.Output(t.Context(), "tc", "-j", kind, "show", "dev", "fv-ih")
		if err != nil || string(raw) != "[]" {
			t.Fatalf("root deletion left %s: %s %v", kind, raw, err)
		}
	}
}

// D-146/D-147: compare the simulator's statements with independent kernel
// JSON shapes. Values are translated, not substituted from the desired model.
func TestAT09_AT13_F34_SimulatorTCPAndICMPExpressions(t *testing.T) {
	for _, tc := range []struct{ name, text, want string }{
		{"mss", "tcp flags & syn == syn tcp option maxseg size > 1160 tcp option maxseg size set 1160", `[
            {"match":{"op":"==","left":{"&":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},
            {"match":{"op":">","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},
            {"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}
        ]`},
		{"literal-values", "tcp option maxseg size > 1200 tcp option maxseg size set 1000", `[
            {"match":{"op":">","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1200}},
            {"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1000}}
        ]`},
		{"pmtu", "ct state related icmp type destination-unreachable icmp code frag-needed meta priority set 0x0001002f", `[
            {"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},
            {"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},
            {"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},
            {"mangle":{"key":{"meta":{"key":"priority"}},"value":65583}}
        ]`},
		{"not-related", "ct state != related", `[
            {"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":"related"}}
        ]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, expr, err := parseRule(tokenizeNFT(tc.text))
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(expr)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("wrong nft expressions: got %s; want %s", raw, tc.want)
			}
		})
	}
}

func TestAT13_F34_SimulatorRejectsUnknownStatementAtomically(t *testing.T) {
	ex := &renderedExecutor{}
	const initial = `add table inet fvpn
add chain inet fvpn forward { type filter hook forward priority -150; policy drop; }
`
	if err := ex.NftBatch(t.Context(), initial); err != nil {
		t.Fatal(err)
	}
	before, err := ex.table.listingJSON()
	if err != nil {
		t.Fatal(err)
	}
	installedAt, history := ex.allowedInstalledAt, len(ex.scripts)
	const clamp = `add rule inet fvpn forward iifname "fv-fh" oifname "fv-ih" ip saddr 100.64.1.1 tcp flags & syn == syn tcp option maxseg size > 1160 tcp option maxseg size set 1160
`
	for _, unsupported := range []string{
		"meta nftrace set 1",
		"tcp option timestamp size set 1160",
		"tcp option maxseg size set",
		"tcp option maxseg size set 65536",
		"tcp flags | syn == syn",
		"tcp mystery 1",
		"icmp mystery frag-needed",
	} {
		t.Run(unsupported, func(t *testing.T) {
			if err := ex.NftBatch(t.Context(), clamp+"add rule inet fvpn forward "+unsupported+"\n"); err == nil {
				t.Fatal("unknown or malformed trailing statement accepted")
			}
			after, err := ex.table.listingJSON()
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) || ex.allowedInstalledAt != installedAt || len(ex.scripts) != history {
				t.Fatal("failed batch changed rules, expiry anchor or successful history")
			}
		})
	}
}

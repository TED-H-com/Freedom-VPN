package enforce

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestReviewR15_TCParser(t *testing.T) {
	for _, kind := range []string{"matchall", "flower", "u32", "bpf"} {
		for _, action := range []string{"mirred", "pedit", "nat", "skbedit", "bpf"} {
			t.Run(kind+"_"+action, func(t *testing.T) {
				raw := fmt.Sprintf(`[{"kind":%q,"parent":"ffff:fff2","options":{"handle":1,"actions":[{"kind":%q,"to_dev":"fv-ih"}]}}]`, kind, action)
				objects, err := ParseTCObjects([]byte(raw))
				if err != nil || len(objects) != 1 || objects[0].Kind != kind || objects[0].Handle != "1" || objects[0].Parent != "ffff:fff2" || len(objects[0].Actions) != 1 || objects[0].Actions[0] != action {
					t.Fatalf("objects=%+v err=%v", objects, err)
				}
			})
		}
	}
	for _, raw := range []string{"", "null", "{}", "[] []", "[null]", "[{}]", `[{"kind":7}]`, `[{"kind":"clsact","handle":{}}]`, `[{"kind":"matchall","options":{"actions":[{}]}}]`} {
		if _, err := ParseTCObjects([]byte(raw)); err == nil {
			t.Errorf("accepted malformed evidence %q", raw)
		}
	}
	if objects, err := ParseTCObjects([]byte(`[]`)); err != nil || len(objects) != 0 {
		t.Fatalf("empty array: %+v %v", objects, err)
	}
}

func TestReviewR15_TCQdiscOwnership(t *testing.T) {
	m := smallModel()
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	minor := plan.Assignments[m.QoS.Members[0].MemberID]
	for _, kind := range []string{"noqueue", "pfifo_fast", "fq_codel", "fq", "mq", "cake"} {
		t.Run(kind, func(t *testing.T) {
			obj := TCObject{Kind: kind, Handle: "0:", Root: true}
			if err := checkTCFixture(obj, "qdisc", m, plan.Assignments); err != nil {
				t.Fatal(err)
			}
			// Even a default root does not authorize a filter.
			if err := checkTCFixture(TCObject{Kind: "matchall"}, "root", m, plan.Assignments); err == nil {
				t.Fatal("accepted default-root filter")
			}
		})
	}
	for _, obj := range []TCObject{
		{Kind: "htb", Handle: "1:", Root: true},
		{Kind: "fq_codel", Handle: fmt.Sprintf("%x:", minor), Parent: classHandle(minor)},
	} {
		if err := checkTCFixture(obj, "qdisc", m, plan.Assignments); err != nil {
			t.Fatal(err)
		}
	}
	for _, obj := range []TCObject{
		{Kind: "clsact", Handle: "ffff:", Parent: "ffff:fff1"},
		{Kind: "ingress", Handle: "ffff:", Parent: "ffff:fff1"},
		{Kind: "netem", Handle: "10:", Root: true},
		{Kind: "htb", Handle: "1:2", Root: true},
		{Kind: "fq_codel", Handle: "20:", Parent: "1:9999"},
		{Kind: "cake", Handle: "10:", Root: true},
		{Kind: "fq_codel", Handle: fmt.Sprintf("%x:", minor), Parent: "2:1"},
		{Kind: "htb", Handle: "1:", Root: true, Actions: []string{"mirred"}},
	} {
		if err := checkTCFixture(obj, "qdisc", m, plan.Assignments); err == nil {
			t.Errorf("accepted foreign qdisc %+v", obj)
		}
	}
}

func checkTCFixture(obj TCObject, attachment string, m Model, assignments map[uuid.UUID]uint16) error {
	return verifyTCObjects([]TCReadEvidence{{Interface: m.Datapath.MemberIngress, Attachment: attachment, Objects: []TCObject{obj}}}, m, assignments)
}

func TestReviewR15_TCServingLiveReads(t *testing.T) {
	m := smallModel()
	m.Revoke = nil
	plan, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	minor := plan.Assignments[m.QoS.Members[0].MemberID]
	for _, iface := range []string{m.Datapath.MemberIngress, m.Datapath.Uplink} {
		for _, attachment := range []string{"root", "ingress", "egress", "1:1", "1:fffe", classHandle(minor), fmt.Sprintf("%x:", minor)} {
			t.Run(iface+"_"+attachment, func(t *testing.T) {
				f := servingExec(m)
				inner := f.handler
				foreign := false
				f.handler = func(name string, args []string) ([]byte, error) {
					if name == "nft" && strings.Join(args, " ") == "-j list ruleset" {
						f.phase = "qos"
						return fixtureServing(m, plan.Assignments), nil
					}
					if name == "tc" && strings.Join(args, " ") == filterCommand(iface, attachment) && foreign {
						return []byte(`[{"kind":"flower","options":{"handle":1,"actions":[{"kind":"mirred"}]}}]`), nil
					}
					return inner(name, args)
				}
				for _, bad := range []bool{false, true, false} {
					foreign = bad
					ev, err := VerifyServing(context.Background(), f, m, plan.Assignments)
					if bad {
						requireCheck(t, err, ReasonTCForeignFilter)
						if !strings.Contains(err.Error(), iface) || !strings.Contains(err.Error(), attachment) || !strings.Contains(err.Error(), "kind flower") {
							t.Fatal(err)
						}
					} else if err != nil {
						t.Fatal(err)
					}
					if len(ev.TC) < 8 || len(ev.PolicyRules) != 2 {
						t.Fatalf("incomplete evidence: %+v", ev)
					}
				}
			})
		}
	}
}

func filterCommand(iface, attachment string) string {
	key := "-j filter show dev " + iface
	switch attachment {
	case "root":
		return key
	case "ingress", "egress":
		return key + " " + attachment
	default:
		return key + " parent " + attachment
	}
}

func TestReviewR15_TCApplyFailsClosed(t *testing.T) {
	for _, fixture := range []string{"clsact", "filter", "read_error", "invalid_json"} {
		t.Run(fixture, func(t *testing.T) {
			m := smallModel()
			m.Revoke = nil
			f := servingExec(m)
			inner := f.handler
			f.handler = func(name string, args []string) ([]byte, error) {
				key := strings.Join(args, " ")
				if name == "tc" && fixture == "clsact" && f.phase == "qos" && key == "-j qdisc show dev "+m.Datapath.MemberIngress {
					raw, err := inner(name, args)
					return []byte(strings.TrimSuffix(string(raw), "]") + `,{"kind":"clsact","handle":"ffff:","parent":"ffff:fff1"}]`), err
				}
				if name == "tc" && key == filterCommand(m.Datapath.MemberIngress, "ingress") {
					switch fixture {
					case "filter":
						return []byte(`[{"kind":"u32","options":{"actions":[{"kind":"skbedit"}]}}]`), nil
					case "read_error":
						return nil, fmt.Errorf("TEST ONLY: command unavailable")
					case "invalid_json":
						return []byte(`null`), nil
					}
				}
				return inner(name, args)
			}
			res, err := Apply(context.Background(), f, m, nil)
			requireCheck(t, err, ReasonTCForeignFilter)
			if res.Verified || len(res.Evidence.TC) == 0 || !strings.Contains(err.Error(), m.Datapath.MemberIngress) {
				t.Fatalf("result=%+v err=%v", res, err)
			}
			if len(f.nft) < 2 || !strings.Contains(f.nft[len(f.nft)-1], "fvpn_baseline=1") {
				t.Fatal("did not install baseline")
			}
			for _, script := range f.tc {
				if strings.Contains(script, "clsact") || strings.Contains(script, "filter del") {
					t.Fatal("mutated foreign tc state")
				}
			}
		})
	}
}

func TestReviewR15_PolicyRulesEvidenceOnly(t *testing.T) {
	for _, failed := range []bool{false, true} {
		m := smallModel()
		m.Revoke = nil
		f := servingExec(m)
		inner := f.handler
		f.handler = func(name string, args []string) ([]byte, error) {
			if name == "ip" {
				if failed {
					return nil, fmt.Errorf("TEST ONLY: rule scan unavailable")
				}
				return []byte(`[{"priority":100,"src":"100.64.9.9","table":100}]`), nil
			}
			return inner(name, args)
		}
		res, err := Apply(context.Background(), f, m, nil)
		if err != nil || !res.Verified || len(res.Evidence.PolicyRules) != 2 {
			t.Fatalf("result=%+v err=%v", res, err)
		}
		for _, rule := range res.Evidence.PolicyRules {
			if (!failed && (rule.Count != 1 || rule.Error != "")) || (failed && rule.Error != "read_failed") {
				t.Fatalf("rule evidence %+v", rule)
			}
		}
		raw, _ := json.Marshal(res.Evidence)
		if strings.Contains(string(raw), "100.64.9.9") {
			t.Fatal("evidence retained rule address")
		}
	}
}

func TestReviewR15_TCDefaultsFailQoSNotForeign(t *testing.T) {
	for _, fixture := range []struct {
		name, qdisc, classes string
	}{
		{"mq_text", `[{"kind":"mq","handle":"0:","root":true},{"kind":"fq_codel","handle":"0:","parent":":1"},{"kind":"fq_codel","handle":"0:","parent":":2"}]`, "class mq :1 root\nclass mq :2 root\n"},
		{"mq_json", `[{"kind":"mq","handle":"0:","root":true},{"kind":"fq_codel","handle":"0:","parent":":1"}]`, `[{"kind":"mq","handle":":1","root":true}]`},
		{"fq_codel_flows", `[{"kind":"fq_codel","handle":"0:","root":true}]`, "class fq_codel :1 root\n"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			m := smallModel()
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				t.Fatal(err)
			}
			f := servingExec(m)
			inner := f.handler
			f.handler = func(name string, args []string) ([]byte, error) {
				if name == "nft" {
					return fixtureServing(m, plan.Assignments), nil
				}
				if name == "tc" && len(args) > 1 {
					switch args[1] {
					case "qdisc":
						return []byte(fixture.qdisc), nil
					case "class":
						return []byte(fixture.classes), nil
					}
				}
				return inner(name, args)
			}
			_, err = VerifyServing(context.Background(), f, m, plan.Assignments)
			requireCheck(t, err, ReasonQoS)
		})
	}
}

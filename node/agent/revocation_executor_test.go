package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// revocationExecutor supplies nft/tc read-back fixtures to the real enforce.Apply.
// The serving JSON follows node/enforce's verification fixture; no host command runs.
type revocationExecutor struct {
	model      enforce.Model
	installed  bool
	failDelete bool
	failVerify bool
	deletes    []string
	scripts    []string
}

func (ex *revocationExecutor) LookPath(name string) (string, error) { return "/test-only/" + name, nil }
func (ex *revocationExecutor) NftBatch(_ context.Context, script string) error {
	ex.scripts = append(ex.scripts, script)
	ex.installed = true
	return nil
}
func (ex *revocationExecutor) TCBatch(context.Context, string) error { return nil }
func (ex *revocationExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if name == "conntrack" {
		ex.deletes = append(ex.deletes, strings.Join(args, " "))
		if ex.failDelete {
			return nil, fmt.Errorf("TEST ONLY: delete failed")
		}
		return nil, nil
	}
	if name == "nft" && !ex.installed {
		return nil, &enforce.CmdError{Out: "No such file", Err: fmt.Errorf("TEST ONLY: empty ruleset")}
	}
	plan, err := enforce.Render(ex.model, enforce.State{}, enforce.TCState{})
	if err != nil {
		return nil, err
	}
	if name == "nft" {
		if ex.failVerify {
			return []byte(`{"nftables":[]}`), nil
		}
		return revocationRuleset(ex.model, plan.Assignments), nil
	}
	if name == "tc" && len(args) > 1 {
		if args[0] != "-j" {
			return nil, nil
		}
		if !ex.installed {
			return []byte(`[]`), nil
		}
		minor := plan.Assignments[ex.model.Bindings[0].MemberID]
		handle := fmt.Sprintf("1:%x", minor)
		if args[1] == "qdisc" {
			return fmt.Appendf(nil, `[{"kind":"htb","handle":"1:","root":true,"options":{"default":"fffe"}},{"kind":"fq_codel","handle":"%x:","parent":"%s"}]`, minor, handle), nil
		}
		q := ex.model.QoS
		parent, host, member := q.Up, q.HostUp, q.Members[0].Up
		if args[len(args)-1] == ex.model.Datapath.MemberIngress {
			parent, host, member = q.Down, q.HostDown, q.Members[0].Down
		}
		return fmt.Appendf(nil, `[{"kind":"htb","handle":"1:1","options":{"rate":"%dbit","ceil":"%dbit"}},{"kind":"htb","handle":"1:fffe","options":{"rate":"%dbit","ceil":"%dbit"}},{"kind":"htb","handle":"%s","options":{"rate":"%dbit","ceil":"%dbit"}}]`, parent.RateBPS, parent.CeilBPS, host.RateBPS, host.CeilBPS, handle, member.RateBPS, member.CeilBPS), nil
	}
	return nil, fmt.Errorf("unexpected fixture command %s", name)
}

func revocationRuleset(m enforce.Model, assignments map[uuid.UUID]uint16) []byte {
	minor := assignments[m.Bindings[0].MemberID]
	classid := enforce.ClassID(minor)
	up := "u_" + strings.ReplaceAll(m.Bindings[0].DeviceID.String(), "-", "")
	dn := "d_" + strings.ReplaceAll(m.Bindings[0].DeviceID.String(), "-", "")
	doc := map[string]any{
		"nftables": []any{
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "forward", "type": "filter", "hook": "forward", "prio": -150, "policy": "drop"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "comment": fmt.Sprintf("fvpn_generation=%d fvpn_metering=%s", m.Generation, enforce.MeteringVersion), "expr": []any{map[string]any{"match": map[string]any{"left": "never", "right": "never"}}}}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "expr": []any{
				map[string]any{"match": map[string]any{"op": "!=", "left": map[string]any{"payload": map[string]any{"protocol": "ip", "field": "saddr"}}, "right": "@allowed_v4"}},
				map[string]any{"counter": map[string]any{"name": "ctunku"}},
				map[string]any{"drop": nil},
			}}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.Uplink}},
				map[string]any{"counter": map[string]any{"name": up}},
				map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": classid}},
				map[string]any{"accept": nil},
			}}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "forward", "expr": []any{
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.Uplink}},
				map[string]any{"match": map[string]any{"op": "==", "left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.MemberIngress}},
				map[string]any{"counter": map[string]any{"name": dn}},
				map[string]any{"mangle": map[string]any{"key": map[string]any{"meta": map[string]any{"key": "priority"}}, "value": classid}},
				map[string]any{"accept": nil},
			}}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "input", "type": "filter", "hook": "input", "prio": -150, "policy": "accept"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "input", "expr": []any{
				map[string]any{"match": map[string]any{"left": map[string]any{"meta": map[string]any{"key": "iifname"}}, "right": m.Datapath.MemberIngress}},
				map[string]any{"drop": nil},
			}}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "output", "type": "filter", "hook": "output", "prio": -150, "policy": "accept"}},
			map[string]any{"chain": map[string]any{"family": "inet", "table": "fvpn", "name": "postrouting", "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}},
			map[string]any{"rule": map[string]any{"family": "inet", "table": "fvpn", "chain": "postrouting", "expr": []any{
				map[string]any{"match": map[string]any{"left": map[string]any{"meta": map[string]any{"key": "oifname"}}, "right": m.Datapath.Uplink}},
				map[string]any{"snat": map[string]any{"addr": m.Datapath.SNAT.String()}},
			}}},
			map[string]any{"counter": map[string]any{"family": "inet", "table": "fvpn", "name": up, "packets": 1, "bytes": 100}},
			map[string]any{"counter": map[string]any{"family": "inet", "table": "fvpn", "name": dn, "packets": 1, "bytes": 100}},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return raw
}

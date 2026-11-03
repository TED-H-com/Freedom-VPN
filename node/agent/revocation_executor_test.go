package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
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
	tcDevices  map[string]*listedTCDevice
}

func (ex *revocationExecutor) LookPath(name string) (string, error) { return "/test-only/" + name, nil }
func (ex *revocationExecutor) NftBatch(_ context.Context, script string) error {
	ex.scripts = append(ex.scripts, script)
	ex.installed = true
	return nil
}

// TC batches are sequential commands, not transactions. In particular, a
// prepare batch preserves classes absent from the desired model until a later
// retire command deletes them. A failed command retains earlier mutations.
func (ex *revocationExecutor) TCBatch(ctx context.Context, script string) error {
	for _, line := range strings.Split(script, "\n") {
		if err := ctx.Err(); err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := ex.tcCommand(strings.Fields(line)); err != nil {
			return err
		}
	}
	return nil
}

type listedTCDevice struct {
	qdiscs  map[string]map[string]any // root, or the parent class handle
	classes map[string]map[string]any // class handle
}

func tcField(fields []string, key string) string {
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == key {
			return fields[i+1]
		}
	}
	return ""
}

// Model the commands emitted by enforce, rejecting unsupported or impossible
// operations instead of synthesizing the desired state in readback.
func (ex *revocationExecutor) tcCommand(f []string) error {
	bad := func() error { return fmt.Errorf("tc fixture: unsupported or missing object: %s", strings.Join(f, " ")) }
	if len(f) < 5 || f[2] != "dev" || f[3] == "" {
		return bad()
	}
	if ex.tcDevices == nil {
		ex.tcDevices = map[string]*listedTCDevice{}
	}
	dev := ex.tcDevices[f[3]]
	if dev == nil {
		dev = &listedTCDevice{qdiscs: map[string]map[string]any{}, classes: map[string]map[string]any{}}
	}
	parent, handle := tcField(f, "parent"), tcField(f, "handle")
	switch f[0] + " " + f[1] {
	case "qdisc add":
		if slices.Contains(f, "root") {
			if dev.qdiscs["root"] != nil || handle == "" || !slices.Contains(f, "htb") || tcField(f, "default") == "" {
				return bad()
			}
			dev.qdiscs["root"] = map[string]any{"kind": "htb", "handle": handle, "root": true, "options": map[string]any{"default": tcField(f, "default")}}
		} else {
			if parent == "" || handle == "" || dev.classes[parent] == nil || dev.qdiscs[parent] != nil || !slices.Contains(f, "fq_codel") {
				return bad()
			}
			dev.qdiscs[parent] = map[string]any{"kind": "fq_codel", "handle": handle, "parent": parent}
		}
	case "qdisc del":
		if slices.Contains(f, "root") {
			if dev.qdiscs["root"] == nil {
				return &enforce.CmdError{Out: "No such file", Err: bad()}
			}
			delete(ex.tcDevices, f[3])
			return nil
		}
		if parent == "" || dev.qdiscs[parent] == nil {
			return bad()
		}
		delete(dev.qdiscs, parent)
	case "class replace":
		id, rate, ceil := tcField(f, "classid"), tcField(f, "rate"), tcField(f, "ceil")
		root := dev.qdiscs["root"]
		if root == nil || id == "" || parent == "" || rate == "" || ceil == "" || !slices.Contains(f, "htb") {
			return bad()
		}
		if parent != root["handle"] && dev.classes[parent] == nil {
			return bad()
		}
		dev.classes[id] = map[string]any{"kind": "htb", "handle": id, "parent": parent, "options": map[string]any{"rate": rate, "ceil": ceil}}
	case "class del":
		id := tcField(f, "classid")
		if id == "" || dev.classes[id] == nil {
			return bad()
		}
		for _, class := range dev.classes {
			if class["parent"] == id {
				return bad()
			}
		}
		delete(dev.classes, id)
		delete(dev.qdiscs, id)
	default:
		return bad()
	}
	ex.tcDevices[f[3]] = dev
	return nil
}

func (ex *revocationExecutor) tcOutput(args []string) ([]byte, error) {
	if len(args) >= 2 && args[0] == "-j" {
		if len(args) < 5 || args[2] != "show" || args[3] != "dev" || !slices.Contains([]string{"qdisc", "class", "filter"}, args[1]) {
			return nil, fmt.Errorf("tc fixture: unsupported listing %v", args)
		}
		var objects map[string]map[string]any
		dev := ex.tcDevices[tcField(args, "dev")]
		if dev != nil {
			switch args[1] {
			case "qdisc":
				objects = dev.qdiscs
			case "class":
				objects = dev.classes
			case "filter":
			}
		}
		keys := make([]string, 0, len(objects))
		for key := range objects {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		out := make([]any, 0, len(keys))
		for _, key := range keys {
			out = append(out, objects[key])
		}
		return json.Marshal(out)
	}
	return nil, ex.tcCommand(args)
}

func (ex *revocationExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if name == "conntrack" {
		ex.deletes = append(ex.deletes, strings.Join(args, " "))
		if ex.failDelete {
			return nil, fmt.Errorf("TEST ONLY: delete failed")
		}
		return nil, nil
	}
	if name == "tc" {
		return ex.tcOutput(args)
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

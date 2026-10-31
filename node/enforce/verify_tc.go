package enforce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// TCObject retains attachment identity and action kinds, never packet matches
// or action parameters (which can contain destinations).
type TCObject struct {
	Kind    string   `json:"kind"`
	Handle  string   `json:"handle,omitempty"`
	Parent  string   `json:"parent,omitempty"`
	Root    bool     `json:"root,omitempty"`
	Actions []string `json:"actions,omitempty"`
}

// ParseTCObjects reads a tc -j qdisc/filter array. Empty output, null, malformed
// objects and trailing documents are not evidence of an empty attachment.
func ParseTCObjects(raw []byte) ([]TCObject, error) {
	var arr []struct {
		Kind    string          `json:"kind"`
		Handle  json.RawMessage `json:"handle"`
		Parent  string          `json:"parent"`
		Root    bool            `json:"root"`
		Options struct {
			Handle  json.RawMessage `json:"handle"`
			Actions []struct {
				Kind string `json:"kind"`
			} `json:"actions"`
		} `json:"options"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil || arr == nil {
		return nil, fmt.Errorf("tc listing is not an object array")
	}
	out := make([]TCObject, 0, len(arr))
	for _, obj := range arr {
		if obj.Kind == "" {
			return nil, fmt.Errorf("tc object has no kind")
		}
		handle := obj.Handle
		if len(handle) == 0 {
			handle = obj.Options.Handle
		}
		h, err := tcObjectHandle(handle)
		if err != nil {
			return nil, err
		}
		v := TCObject{Kind: obj.Kind, Handle: h, Parent: obj.Parent, Root: obj.Root}
		for _, action := range obj.Options.Actions {
			if action.Kind == "" {
				return nil, fmt.Errorf("tc action has no kind")
			}
			v.Actions = append(v.Actions, action.Kind)
		}
		out = append(out, v)
	}
	return out, nil
}

func tcObjectHandle(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil // Filter protocol headers need not carry a handle.
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return s, nil
	}
	// Some classifiers put a numeric handle in options, without a hex prefix.
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		if _, err := n.Int64(); err == nil {
			return n.String(), nil
		}
	}
	return "", fmt.Errorf("tc object has an invalid handle")
}

// TCReadEvidence is one live attachment read. Error is a fixed status, not
// command output; tc output can contain packet match addresses.
type TCReadEvidence struct {
	Interface  string     `json:"interface"`
	Attachment string     `json:"attachment"`
	Objects    []TCObject `json:"objects"`
	Error      string     `json:"error,omitempty"`
}

// PolicyRuleEvidence records the scan without retaining address selectors or
// destinations. Policy routing is informational, not a serving rejection.
type PolicyRuleEvidence struct {
	Family string `json:"family"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

// ServingEvidence is the live read-back retained even when verification fails.
// Counters are for Apply's samples and are not duplicated in JSON evidence.
type ServingEvidence struct {
	Quarantined      bool                    `json:"quarantined"`
	QuarantinedLinks []string                `json:"quarantined_links,omitempty"`
	TC               []TCReadEvidence        `json:"tc"`
	PolicyRules      []PolicyRuleEvidence    `json:"policy_rules"`
	Counters         map[string]CounterValue `json:"-"`
}

// VerifyServing reads live nft, tc and policy-routing state on every call.
// A caller cannot establish serving from a saved nft listing alone. It does
// not mutate networking; the caller must Quarantine on foreign tc failure.
// Quarantined reports a retained recovery marker, not current containment;
// use Quarantine evidence for the link-down and baseline read-back.
func VerifyServing(ctx context.Context, ex Executor, m Model, assignments map[uuid.UUID]uint16) (ServingEvidence, error) {
	return verifyServing(ctx, ex, m, assignments, false)
}

// Apply alone may verify the staged serving rules while the quarantine marker
// is retained. Public verification always rejects that marker.
func verifyServing(ctx context.Context, ex Executor, m Model, assignments map[uuid.UUID]uint16, recovering bool) (ev ServingEvidence, err error) {
	defer func() { ev.PolicyRules = readPolicyRules(ctx, ex) }()
	raw, nftErr := ex.Output(ctx, "nft", "-j", "list", "ruleset")
	if nftErr == nil {
		rs, parseErr := ParseRuleset(raw)
		if parseErr == nil {
			ev.Counters = rs.Counters
			ev.QuarantinedLinks = quarantinedLinks(rs.Sets)
			ev.Quarantined = len(ev.QuarantinedLinks) > 0
		}
		nftErr = verifyServingRuleset(raw, m, assignments)
	}
	// Scan tc even with the link down or nft baseline installed: neither may
	// hide the retained redirect or change its specific failure reason.
	qos, reads, err := readServingTC(ctx, ex, m)
	ev.TC = reads
	if err != nil {
		return ev, err
	}
	if err := verifyTCObjects(reads, m, assignments); err != nil {
		return ev, err
	}
	if nftErr != nil {
		return ev, nftErr
	}
	if ev.Quarantined && !recovering {
		return ev, checkf(ReasonIsolation, "member path quarantined; verified Apply required")
	}
	return ev, VerifyQoS(qos, m, assignments)
}

func readServingTC(ctx context.Context, ex Executor, m Model) (map[string][2][]byte, []TCReadEvidence, error) {
	qos := map[string][2][]byte{}
	var reads []TCReadEvidence
	read := func(iface, attachment string, args ...string) ([]TCObject, []byte, error) {
		r := TCReadEvidence{Interface: iface, Attachment: attachment}
		raw, err := ex.Output(ctx, "tc", args...)
		if err != nil {
			r.Error = "read_failed"
		} else {
			r.Objects, err = ParseTCObjects(raw)
			if err != nil {
				r.Error = "invalid_json"
			}
		}
		reads = append(reads, r)
		if err != nil {
			return nil, nil, checkf(ReasonTCForeignFilter, "interface %s parent %s kind unreadable: %s", iface, attachment, r.Error)
		}
		return r.Objects, raw, nil
	}
	for _, iface := range []string{m.Datapath.MemberIngress, m.Datapath.Uplink} {
		objects, q, err := read(iface, "qdisc", "-j", "qdisc", "show", "dev", iface)
		if err != nil {
			return qos, reads, err
		}
		// Each command dumps all chains of one block. The bare command reads
		// the root block only; HTB classes and fq_codel have separate blocks.
		for _, hook := range []string{"root", "ingress", "egress"} {
			args := []string{"-j", "filter", "show", "dev", iface}
			if hook != "root" {
				args = append(args, hook)
			}
			if _, _, err := read(iface, hook, args...); err != nil {
				return qos, reads, err
			}
		}
		c, err := ex.Output(ctx, "tc", "-j", "class", "show", "dev", iface)
		if err != nil {
			return qos, reads, checkf(ReasonTCForeignFilter, "interface %s parent class kind unreadable: read_failed", iface)
		}
		parents := map[string]bool{}
		htb := false
		for _, obj := range objects {
			htb = htb || obj.Kind == "htb"
			if !obj.Root && obj.Kind != "clsact" && obj.Kind != "ingress" && obj.Handle != "" {
				parents[obj.Handle] = true
			}
		}
		// Only HTB classes have independent classifier blocks. Default mq
		// classes and fq_codel flow pseudo-classes are not HTB evidence and
		// must reach the separate QoS check instead of being called foreign.
		if htb {
			view, err := ParseClassJSON(c, QdiscView{})
			if err != nil {
				return qos, reads, checkf(ReasonTCForeignFilter, "interface %s parent class kind unreadable: invalid_listing", iface)
			}
			for handle := range view.Classes {
				parents[handle] = true
			}
		}
		ordered := make([]string, 0, len(parents))
		for parent := range parents {
			ordered = append(ordered, parent)
		}
		sort.Strings(ordered)
		for _, parent := range ordered {
			if _, _, err := read(iface, parent, "-j", "filter", "show", "dev", iface, "parent", parent); err != nil {
				return qos, reads, err
			}
		}
		qos[iface] = [2][]byte{q, c}
	}
	return qos, reads, nil
}

func verifyTCObjects(reads []TCReadEvidence, m Model, assignments map[uuid.UUID]uint16) error {
	leaves := map[string]string{}
	for _, member := range m.QoS.Members {
		if minor, ok := assignments[member.MemberID]; ok {
			leaves[classHandle(minor)] = fmt.Sprintf("%x:", minor)
		}
	}
	for _, read := range reads {
		objects := read.Objects
		defaultMQ := false
		for _, obj := range objects {
			defaultMQ = defaultMQ || (obj.Root && obj.Kind == "mq" && obj.Handle == "0:" && obj.Parent == "")
		}
		if read.Attachment != "qdisc" {
			// tc emits protocol headers followed by actual filters. Prefer an
			// actual handle in the error; a header alone still fails closed.
			for _, obj := range objects {
				if obj.Handle != "" {
					objects = []TCObject{obj}
					break
				}
			}
		}
		for _, obj := range objects {
			if read.Attachment == "qdisc" && len(obj.Actions) == 0 {
				if obj.Root && obj.Parent == "" {
					if obj.Kind == "htb" && obj.Handle == "1:" {
						continue
					}
					if obj.Handle == "0:" && defaultQdisc(obj.Kind) {
						continue
					}
				}
				if !obj.Root && obj.Kind == "fq_codel" && leaves[obj.Parent] == obj.Handle && obj.Handle != "" {
					continue
				}
				if defaultMQ && !obj.Root && obj.Handle == "0:" && defaultQdisc(obj.Kind) && mqQueueParent(obj.Parent) {
					continue
				}
			}
			return checkf(ReasonTCForeignFilter, "interface %s parent %s handle %s kind %s", read.Interface, tcParent(read.Attachment, obj), obj.Handle, obj.Kind)
		}
	}
	return nil
}

func mqQueueParent(parent string) bool {
	minor, ok := strings.CutPrefix(parent, ":")
	if !ok {
		return false
	}
	n, err := strconv.ParseUint(minor, 16, 16)
	return err == nil && n > 0
}

func tcParent(attachment string, obj TCObject) string {
	if obj.Parent != "" {
		return obj.Parent
	}
	if obj.Root {
		return "root"
	}
	return attachment
}

func defaultQdisc(kind string) bool {
	switch kind {
	case "noqueue", "pfifo_fast", "fq_codel", "fq", "mq", "cake":
		return true
	default:
		return false
	}
}

func readPolicyRules(ctx context.Context, ex Executor) []PolicyRuleEvidence {
	out := make([]PolicyRuleEvidence, 0, 2)
	for _, family := range []string{"-4", "-6"} {
		r := PolicyRuleEvidence{Family: strings.TrimPrefix(family, "-")}
		raw, err := ex.Output(ctx, "ip", "-j", family, "rule", "show")
		var rules []map[string]json.RawMessage
		if err != nil {
			r.Error = "read_failed"
		} else if json.Unmarshal(bytes.TrimSpace(raw), &rules) != nil || rules == nil {
			r.Error = "invalid_json"
		} else {
			r.Count = len(rules)
		}
		out = append(out, r)
	}
	return out
}

package enforce

import (
	"context"
	"net/netip"
	"strings"

	"github.com/google/uuid"
)

// ApplyResult is what the agent keeps after a successful render.
type ApplyResult struct {
	Plan     RenderPlan
	Resets   []Reset
	Counters map[string]CounterValue
	Samples  []Sample
}

// Apply installs the model. A verification mismatch installs the boot
// baseline and returns the check error, so member forwarding does not stay
// on a path we cannot account (§10.3, §8.2). Resets are reported and never
// cleared by recreating counter objects (§10.7).
func Apply(ctx context.Context, ex Executor, m Model, previous map[string]CounterValue) (ApplyResult, error) {
	state, err := loadState(ctx, ex)
	if err != nil {
		return ApplyResult{}, err
	}
	tcState, err := loadTC(ctx, ex, []string{m.Datapath.Uplink, m.Datapath.MemberIngress})
	if err != nil {
		return ApplyResult{}, err
	}
	resets := DetectMeterResets(previous, state)
	if len(m.QoS.Assignments) == 0 && len(state.Classes) > 0 {
		// The kernel comment is the memory of class minors. Recomputing from
		// scratch could move a member when another member is added or removed
		// (§11.4).
		copied := make(map[uuid.UUID]uint16, len(state.Classes))
		for id, minor := range state.Classes {
			copied[id] = minor
		}
		m.QoS.Assignments = copied
	}
	plan, err := Render(m, state, tcState)
	if err != nil {
		return ApplyResult{}, err
	}
	// Shape first, then open forwarding. A failed nft install still rolls
	// back to the baseline, which removes the qdisc (§8.2, §11.5).
	if err := ex.TCBatch(ctx, plan.TC); err != nil {
		_ = installBaseline(ctx, ex, m.Datapath, state)
		return ApplyResult{}, err
	}
	if err := ex.NftBatch(ctx, plan.NFT); err != nil {
		_ = installBaseline(ctx, ex, m.Datapath, state)
		return ApplyResult{}, err
	}
	raw, err := ex.Output(ctx, "nft", "-j", "list", "table", TableFamily, TableName)
	if err != nil {
		_ = installBaseline(ctx, ex, m.Datapath, state)
		return ApplyResult{}, err
	}
	if err := VerifyServing(raw, m, plan.Assignments); err != nil {
		_ = installBaseline(ctx, ex, m.Datapath, state)
		return ApplyResult{Plan: plan, Resets: resets}, err
	}
	qosJSON, err := readQoS(ctx, ex, m)
	if err != nil {
		_ = installBaseline(ctx, ex, m.Datapath, state)
		return ApplyResult{}, err
	}
	if err := VerifyQoS(qosJSON, m, plan.Assignments); err != nil {
		_ = installBaseline(ctx, ex, m.Datapath, state)
		return ApplyResult{Plan: plan, Resets: resets}, err
	}
	for _, addr := range m.Revoke {
		if err := deleteConntrack(ctx, ex, addr); err != nil {
			return ApplyResult{Plan: plan, Resets: resets}, err
		}
	}
	rs, err := ParseRuleset(raw)
	if err != nil {
		return ApplyResult{}, err
	}
	samples, err := Samples(m, rs.Counters)
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Plan: plan, Resets: resets, Counters: rs.Counters, Samples: samples}, nil
}

// ApplyBaseline installs the fail-closed ruleset and removes our HTB roots.
// The probe exception is not included. ApplyBaselineModel carries one.
func ApplyBaseline(ctx context.Context, ex Executor, dp Datapath) error {
	return ApplyBaselineModel(ctx, ex, Model{Datapath: dp})
}

// ApplyBaselineModel installs the fail-closed ruleset. A non-nil Probe is the
// only forward accept. A nil Probe removes the sets and the rules (§13.9).
func ApplyBaselineModel(ctx context.Context, ex Executor, m Model) error {
	state, err := loadState(ctx, ex)
	if err != nil {
		return err
	}
	if err := installBaselineModel(ctx, ex, m, state); err != nil {
		return err
	}
	raw, err := ex.Output(ctx, "nft", "-j", "list", "table", TableFamily, TableName)
	if err != nil {
		return err
	}
	return VerifyBaseline(raw, m)
}

func installBaseline(ctx context.Context, ex Executor, dp Datapath, state State) error {
	return installBaselineModel(ctx, ex, Model{Datapath: dp}, state)
}

func installBaselineModel(ctx context.Context, ex Executor, m Model, state State) error {
	if fresh, err := loadState(ctx, ex); err == nil {
		state = fresh
	}
	script, err := RenderBaseline(m, state)
	if err != nil {
		return err
	}
	if err := ex.NftBatch(ctx, script); err != nil {
		return err
	}
	for _, ifc := range []string{m.Datapath.Uplink, m.Datapath.MemberIngress} {
		_, err := ex.Output(ctx, "tc", "qdisc", "del", "dev", ifc, "root")
		if err != nil && !missingQdisc(err) {
			return err
		}
	}
	return nil
}

func loadState(ctx context.Context, ex Executor) (State, error) {
	raw, err := ex.Output(ctx, "nft", "-j", "list", "table", TableFamily, TableName)
	if err != nil {
		if isNoTable(err) {
			return State{Chains: map[string]bool{}, Sets: map[string]bool{}, Counters: map[string]CounterValue{}}, nil
		}
		return State{}, err
	}
	return ParseState(raw)
}

func loadTC(ctx context.Context, ex Executor, ifaces []string) (TCState, error) {
	st := TCState{Devices: map[string]TCDev{}}
	for _, ifc := range ifaces {
		raw, err := ex.Output(ctx, "tc", "-j", "qdisc", "show", "dev", ifc)
		if err != nil {
			if missingQdisc(err) {
				st.Devices[ifc] = TCDev{}
				continue
			}
			return st, err
		}
		view, err := ParseQdiscJSON(raw)
		if err != nil {
			return st, err
		}
		classRaw, err := ex.Output(ctx, "tc", "-j", "class", "show", "dev", ifc)
		if err != nil {
			if !missingQdisc(err) {
				return st, err
			}
			classRaw = nil
		}
		if len(strings.TrimSpace(string(classRaw))) > 0 {
			view, err = ParseClassJSON(classRaw, view)
			if err != nil {
				return st, err
			}
		}
		st.Devices[ifc] = TCDev{RootHTB: view.RootHTB, Leaves: view.Leaves, Classes: childClassMinors(view)}
	}
	return st, nil
}

// DetectMeterResets is DetectResets, except when the table, the forward chain, or
// the generation marker was cleared. Flush keeps named counter values, so the
// whole previous set is one meter epoch (§10.7). Call it when the caller holds
// a State. DetectResets is the value comparison alone.
func DetectMeterResets(previous map[string]CounterValue, state State) []Reset {
	if len(previous) == 0 {
		return nil
	}
	if !state.Table || !state.hasChain("forward") || state.Generation == 0 {
		return externalMeterReset(previous, state.Counters)
	}
	return DetectResets(previous, state.Counters)
}

func readQoS(ctx context.Context, ex Executor, m Model) (map[string][2][]byte, error) {
	out := map[string][2][]byte{}
	for _, ifc := range []string{m.Datapath.Uplink, m.Datapath.MemberIngress} {
		q, err := ex.Output(ctx, "tc", "-j", "qdisc", "show", "dev", ifc)
		if err != nil {
			return nil, err
		}
		c, err := ex.Output(ctx, "tc", "-j", "class", "show", "dev", ifc)
		if err != nil {
			return nil, err
		}
		out[ifc] = [2][]byte{q, c}
	}
	return out, nil
}

func deleteConntrack(ctx context.Context, ex Executor, addr netip.Addr) error {
	if !addr.IsValid() {
		return checkf(ReasonIsolation, "revoke address is invalid")
	}
	for _, flag := range []string{"-s", "-d"} {
		_, err := ex.Output(ctx, "conntrack", "-D", flag, addr.String())
		if err == nil || conntrackEmpty(err) {
			continue
		}
		return err
	}
	return nil
}

func conntrackEmpty(err error) bool {
	if err == nil {
		return true
	}
	return strings.Contains(err.Error(), "flow entries")
}

func missingQdisc(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "No such file") ||
		strings.Contains(msg, "Cannot delete") ||
		strings.Contains(msg, "Cannot find device") ||
		strings.Contains(msg, "Invalid handle")
}

// Samples joins absolute counter readings to the bindings that explain them
// (§10.5). DNS counters stay in dns_service_v1.
func Samples(m Model, counters map[string]CounterValue) ([]Sample, error) {
	byID := map[string]Binding{}
	for _, b := range m.Bindings {
		byID[b.DeviceID.String()] = b
	}
	var out []Sample
	for name, val := range counters {
		if diagnosticNamespace(name) {
			// Drops and baseline hits stay in diagnostic_v1. They are not
			// device samples and must not enter the usage ledger (§10.1).
			continue
		}
		id, dir, ns, ok := parseDeviceCounter(name)
		if !ok {
			continue
		}
		b := byID[id.String()]
		out = append(out, Sample{
			DeviceID:        id,
			MemberID:        b.MemberID,
			PeerID:          b.PeerID,
			BindingRevision: b.BindingRevision,
			Direction:       dir,
			Packets:         val.Packets,
			Bytes:           val.Bytes,
			Namespace:       ns,
		})
	}
	sortSamples(out)
	return out, nil
}

func sortSamples(out []Sample) {
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && sampleLess(out[j], out[j-1]) {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
}

func sampleLess(a, b Sample) bool {
	if a.DeviceID.String() != b.DeviceID.String() {
		return a.DeviceID.String() < b.DeviceID.String()
	}
	return a.Direction < b.Direction
}

package enforce

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"github.com/google/uuid"
)

// conntrackDeleteDetail is the fixed CheckError text for a conntrack -D
// failure after the serving ruleset was verified. It contains no address.
const conntrackDeleteDetail = "conntrack delete failed after the serving ruleset was verified"

// ApplyResult is what the agent keeps after a render.
type ApplyResult struct {
	Plan     RenderPlan
	Resets   []Reset
	Counters map[string]CounterValue
	Samples  []Sample
	// Verified is true only when the serving ruleset and QoS were verified
	// and left installed. It is false when the baseline was installed or
	// nothing was installed.
	Verified   bool
	Evidence   ServingEvidence
	Quarantine *QuarantineEvidence
}

// ConntrackDeleteError means the serving ruleset was left installed and one
// or more revoke addresses were not deleted from conntrack. Failed lists
// each m.Revoke address whose conntrack -D did not succeed, in that order.
// Pending is len(Failed). The error string is the fixed CheckError text
// and carries no address. Unwrap is the *CheckError the agent already matches.
type ConntrackDeleteError struct {
	Failed  []netip.Addr
	Pending int
	err     error
}

func (e *ConntrackDeleteError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *ConntrackDeleteError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// IsConntrackDeleteFailure reports a conntrack delete that left the verified
// serving ruleset in place. A verification failure and RequireConntrack are
// not this error.
func IsConntrackDeleteFailure(err error) bool {
	var target *ConntrackDeleteError
	return errors.As(err, &target)
}

// Apply installs the model. A verification mismatch installs the boot
// baseline and returns the check error, so member forwarding does not stay
// on a path we cannot account (§10.3, §8.2). Resets are reported and never
// cleared by recreating counter objects (§10.7).
func Apply(ctx context.Context, ex Executor, m Model, previous map[string]CounterValue) (ApplyResult, error) {
	if err := validateDatapath(m.Datapath); err != nil {
		return ApplyResult{}, err
	}
	state, err := loadState(ctx, ex)
	if err != nil {
		return ApplyResult{}, err
	}
	quarantined := quarantinedLinks(state.Sets)
	if len(quarantined) > 0 {
		// A changed desired model cannot authorize raising a retired link.
		if len(quarantined) != 1 || quarantined[0] != m.Datapath.MemberIngress {
			return ApplyResult{}, checkf(ReasonIsolation, "quarantined member path differs from applied model")
		}
		if _, err := lowerMemberLink(ctx, ex, m.Datapath.MemberIngress); err != nil {
			q, qerr := Quarantine(ctx, ex, m)
			return ApplyResult{Quarantine: &q}, errors.Join(err, qerr)
		}
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
	// Check before any class/qdisc mutation so a stale class deletion cannot
	// delete a foreign attached filter. Existing classes may belong to the
	// previous generation; admit their handles for this ownership-only scan.
	_, reads, tcErr := readServingTC(ctx, ex, m)
	owned := make(map[uuid.UUID]uint16)
	ownershipModel := m
	ownershipModel.QoS.Members = nil
	for _, dev := range tcState.Devices {
		for minor := range dev.Classes {
			id := kernelClassPlaceholder(minor)
			owned[id] = minor
			ownershipModel.QoS.Members = append(ownershipModel.QoS.Members, MemberQoS{MemberID: id})
		}
	}
	if tcErr == nil {
		tcErr = verifyTCObjects(reads, ownershipModel, owned)
	}
	if tcErr != nil {
		q, qerr := Quarantine(ctx, ex, m)
		return ApplyResult{Plan: plan, Resets: resets, Evidence: ServingEvidence{TC: reads, Quarantined: q.Recorded, QuarantinedLinks: quarantinedLinksFor(m, q)}, Quarantine: &q}, errors.Join(tcErr, qerr)
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
	// VerifyServing reads live nft and tc state together. A foreign filter is
	// the same serving-gate failure as a flowtable bypass.
	evidence, err := verifyServing(ctx, ex, m, plan.Assignments, len(quarantined) > 0)
	if err != nil {
		var check *CheckError
		if (errors.As(err, &check) && check.Reason == ReasonTCForeignFilter) || len(quarantined) > 0 {
			q, qerr := Quarantine(ctx, ex, m)
			evidence.Quarantined, evidence.QuarantinedLinks = q.Recorded, quarantinedLinksFor(m, q)
			return ApplyResult{Plan: plan, Resets: resets, Evidence: evidence, Quarantine: &q}, errors.Join(err, qerr)
		}
		_ = installBaseline(ctx, ex, m.Datapath, state)
		return ApplyResult{Plan: plan, Resets: resets, Evidence: evidence}, err
	}
	if len(quarantined) > 0 {
		evidence, err = releaseQuarantine(ctx, ex, m, plan.Assignments)
		if err != nil {
			q, qerr := Quarantine(ctx, ex, m)
			return ApplyResult{Plan: plan, Resets: resets, Evidence: evidence, Quarantine: &q}, errors.Join(err, qerr)
		}
	}
	// Admission already drops an address that left allowed_v4, before any
	// established or related accept. A conntrack failure here must not
	// replace the verified ruleset with the baseline. Counters and samples
	// are taken from the verified listing before the deletes, so a failure
	// still reports the generation that stayed installed. Every revoke
	// address is attempted. Failed lists the addresses whose conntrack -D
	// did not succeed. The caller retries those addresses on the next
	// Apply (§8.2).
	verified := ApplyResult{Plan: plan, Resets: resets, Verified: true, Evidence: evidence}
	samples, err := Samples(m, evidence.Counters)
	if err != nil {
		return ApplyResult{Verified: true}, err
	}
	verified.Counters = evidence.Counters
	verified.Samples = samples
	var cmdErrs []error
	var other error
	var failed []netip.Addr
	for _, addr := range m.Revoke {
		err := deleteConntrack(ctx, ex, addr)
		if err == nil {
			continue
		}
		if conntrackCommandFailed(err) {
			failed = append(failed, addr)
			cmdErrs = append(cmdErrs, err)
			continue
		}
		if other == nil {
			other = err
		}
	}
	if len(failed) > 0 {
		return verified, &ConntrackDeleteError{Failed: failed, Pending: len(failed), err: errors.Join(cmdErrs...)}
	}
	if other != nil {
		return verified, other
	}
	return verified, nil
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
	raw, err := ex.Output(ctx, "nft", "-j", "list", "ruleset")
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
	// A baseline refresh during quarantine must retain foreign filters even
	// when they are attached to our root/classes. Quarantine owns no tc state.
	if len(quarantinedLinks(state.Sets)) > 0 {
		return nil
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

func deleteConntrack(ctx context.Context, ex Executor, addr netip.Addr) error {
	if !addr.IsValid() {
		return checkf(ReasonIsolation, "revoke address is invalid")
	}
	for _, flag := range []string{"-s", "-d"} {
		_, err := ex.Output(ctx, "conntrack", "-D", flag, addr.String())
		if err == nil || conntrackEmpty(err) {
			continue
		}
		// The command text names the address. The reported error must not.
		return checkf(ReasonIsolation, conntrackDeleteDetail)
	}
	return nil
}

func conntrackCommandFailed(err error) bool {
	var ce *CheckError
	return errors.As(err, &ce) && ce.Reason == ReasonIsolation && ce.Detail == conntrackDeleteDetail
}

// RequireConntrack reports whether the conntrack binary is installed.
// The agent calls it at startup. A missing binary is ReasonIsolation and
// the detail does not include a path.
func RequireConntrack(ex Executor) error {
	if ex == nil {
		return checkf(ReasonIsolation, "conntrack is not installed")
	}
	path, err := ex.LookPath("conntrack")
	if err != nil || strings.TrimSpace(path) == "" {
		return checkf(ReasonIsolation, "conntrack is not installed")
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
		b, ok := byID[id.String()]
		if !ok || b.MemberID == uuid.Nil || b.BindingRevision == 0 {
			// A removed device's last reading is already in plan.Finals.
			// The named counter stays in the kernel (§10.7).
			continue
		}
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

package agent

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

const (
	// msgExternalReset is the observation text when the ruleset itself was
	// cleared. It names no counter and no device (§13.11, max 500).
	msgExternalReset = "ruleset was reset outside the agent"
	msgCounterGap    = "counter discontinuity"
	nodeErrorLimit   = 500
)

func clipMsg(s string) string {
	if len(s) > nodeErrorLimit {
		return s[:nodeErrorLimit]
	}
	return s
}

func cloneCounters(in map[string]enforce.CounterValue) map[string]enforce.CounterValue {
	out := make(map[string]enforce.CounterValue, len(in))
	for name, v := range in {
		out[name] = v
	}
	return out
}

// usageName is a forward_admission_v1 counter (u_<32 hex> or d_<32 hex>).
// DNS counters (nu_ / nd_) are not usage.
func usageName(name string) bool {
	var rest string
	switch {
	case strings.HasPrefix(name, "u_"):
		rest = name[len("u_"):]
	case strings.HasPrefix(name, "d_"):
		rest = name[len("d_"):]
	default:
		return false
	}
	if len(rest) != 32 {
		return false
	}
	for _, r := range rest {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func usageOnly(in map[string]enforce.CounterValue) map[string]enforce.CounterValue {
	out := make(map[string]enforce.CounterValue, len(in))
	for name, v := range in {
		if usageName(name) {
			out[name] = v
		}
	}
	return out
}

func liveUsage(prev map[string]enforce.CounterValue) bool {
	for name, v := range prev {
		if usageName(name) && (v.Packets > 0 || v.Bytes > 0) {
			return true
		}
	}
	return false
}

func brokenRuleset(st enforce.State) bool {
	return !st.Table || !st.Chains["forward"] || st.Generation == 0
}

func externalDetail(resets []enforce.Reset) bool {
	for _, r := range resets {
		if strings.Contains(r.Detail, "ruleset was reset outside the agent") {
			return true
		}
	}
	return false
}

// classifyExternal is the sample-path reading of DetectMeterResets.
// expectServing is false during the agent's own baseline, so a planned
// final sample is not an external reset. An idle usage counter (zero in
// both snapshots) restores the ruleset and does not open an epoch.
// A value drop is left for the meter's own switch. A structural reset the
// value comparison cannot see asks for one nudged sample.
func classifyExternal(prev map[string]enforce.CounterValue, st enforce.State, expectServing bool) (restore, forceSwitch, report bool) {
	if !expectServing || len(prev) == 0 {
		return false, false, false
	}
	if len(enforce.DetectMeterResets(prev, st)) == 0 {
		return false, false, false
	}
	usageDrop := len(enforce.DetectResets(usageOnly(prev), usageOnly(st.Counters))) > 0
	if !brokenRuleset(st) {
		if usageDrop {
			return false, false, true
		}
		return false, false, false
	}
	if !liveUsage(prev) {
		return true, false, false
	}
	if usageDrop {
		return true, false, true
	}
	return true, true, true
}

// applyNeedsForce reads the resets Apply already computed on the pre-apply
// state. The serving ruleset is back by the time this runs. wasServing is
// false for the first apply after the boot baseline, which is not an
// external clear. Idle usage counters never open an epoch.
func applyNeedsForce(prev map[string]enforce.CounterValue, resets []enforce.Reset, post map[string]enforce.CounterValue, wasServing bool) (force, report bool) {
	if len(resets) == 0 || !liveUsage(prev) {
		return false, false
	}
	usageDrop := len(enforce.DetectResets(usageOnly(prev), usageOnly(post))) > 0
	ext := externalDetail(resets)
	if ext && !wasServing {
		if usageDrop {
			return false, true
		}
		return false, false
	}
	if ext && !usageDrop {
		return true, true
	}
	if usageDrop {
		return false, true
	}
	return false, false
}

// nudgeOne decrements one live usage counter by a single packet, or by one
// byte when that counter has no packets. The meter's switch treats a drop
// as counter_discontinuity and does not bill the backwards counter, so the
// checkpoint keeps the real byte total.
func nudgeOne(in map[string]enforce.CounterValue) map[string]enforce.CounterValue {
	out := cloneCounters(in)
	names := make([]string, 0, len(out))
	for name, v := range out {
		if usageName(name) && (v.Packets > 0 || v.Bytes > 0) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return out
	}
	c := out[names[0]]
	if c.Packets > 0 {
		c.Packets--
	} else {
		c.Bytes--
	}
	out[names[0]] = c
	return out
}

func (k *kernelSource) load(ctx context.Context) (enforce.State, error) {
	if k.read != nil {
		return k.read(ctx)
	}
	return readKernelState(ctx)
}

func readKernelState(ctx context.Context) (enforce.State, error) {
	raw, err := enforce.System{}.Output(ctx, "nft", "-j", "list", "table", enforce.TableFamily, enforce.TableName)
	if err != nil {
		if nftMissing(err) {
			return enforce.State{Counters: map[string]enforce.CounterValue{}}, nil
		}
		return enforce.State{}, err
	}
	st, err := enforce.ParseState(raw)
	if err != nil {
		return enforce.State{}, err
	}
	if st.Counters == nil {
		st.Counters = map[string]enforce.CounterValue{}
	}
	return st, nil
}

// capture stores the ruleset the agent currently holds. Apply does not
// return a State, so the agent reads it back after a successful install.
func (k *kernelSource) capture(ctx context.Context) error {
	st, err := k.load(ctx)
	if err != nil {
		return err
	}
	k.mu.Lock()
	k.state = st
	k.mu.Unlock()
	return nil
}

func (k *kernelSource) noteApply(prev map[string]enforce.CounterValue, resets []enforce.Reset, post map[string]enforce.CounterValue, wasServing bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(post) > 0 || len(k.prev) == 0 {
		k.prev = cloneCounters(post)
	}
	force, report := applyNeedsForce(prev, resets, post, wasServing)
	if !force && !report {
		return
	}
	if force && !k.switched {
		k.nudge = true
		k.switched = true
		k.pending = true
		k.external = true
		return
	}
	if report && !k.reported {
		k.pending = true
		if externalDetail(resets) {
			k.external = true
		}
	}
}

func (k *kernelSource) nudgePending() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.nudge
}

// A live generation rejection is external-reset evidence even if the sampler
// has not run yet. FinalSample consumes the existing discontinuity mechanism
// after withdrawal. Idle counters still do not invent a gap or a bill.
func (k *kernelSource) rejectedGeneration() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !liveUsage(k.prev) || k.switched {
		return
	}
	k.nudge, k.switched, k.pending, k.external = true, true, true, true
}

// rearm asks the next sample to switch again after a sample read the nudge
// and then failed to commit it.
func (k *kernelSource) rearm() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.nudge = true
	k.switched = true
	k.pending = true
	k.external = true
}

func (k *kernelSource) takeReport() (string, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.pending || k.reported {
		return "", false
	}
	k.pending = false
	k.reported = true
	if k.external {
		return msgExternalReset, true
	}
	return msgCounterGap, true
}

func (e *Engine) noteResetsWithoutKernel(prev map[string]enforce.CounterValue, resets []enforce.Reset, post map[string]enforce.CounterValue, wasServing bool) {
	_, report := applyNeedsForce(prev, resets, post, wasServing)
	if !report {
		return
	}
	msg := msgCounterGap
	if externalDetail(resets) {
		msg = msgExternalReset
	}
	e.decision.Errors = append(e.decision.Errors, api.NodeError{
		Code: api.ErrorCodeMETEREPOCHCHANGE, Message: clipMsg(msg),
		ObservedAt: e.clock.Wall().UTC(), Stage: "meter",
	})
}

func (e *Engine) publishMeterEpoch() {
	if e.kernel == nil {
		return
	}
	msg, ok := e.kernel.takeReport()
	if !ok {
		return
	}
	e.decision.Errors = append(e.decision.Errors, api.NodeError{
		Code: api.ErrorCodeMETEREPOCHCHANGE, Message: clipMsg(msg),
		ObservedAt: e.clock.Wall().UTC(), Stage: "meter",
	})
}

// switchEpoch opens one meter epoch through Meter.Sample when the counter
// values alone would not. The one-packet nudge is consumed by whichever
// sample reads first.
func (e *Engine) switchEpoch(ctx context.Context) {
	if e.kernel == nil || e.meter == nil || !e.kernel.nudgePending() {
		return
	}
	err := e.meter.Sample(ctx)
	if errors.Is(err, meter.ErrBusy) {
		time.Sleep(20 * time.Millisecond)
		err = e.meter.Sample(ctx)
	}
	if err == nil || ctx.Err() != nil {
		return
	}
	if errors.Is(err, meter.ErrBusy) {
		return
	}
	if !e.kernel.nudgePending() {
		e.kernel.rearm()
	}
	if !errors.Is(err, meter.ErrHold) && !errors.Is(err, meter.ErrPaused) {
		e.log.Warn("meter epoch", "err", err.Error())
	}
}

package meter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Directions of forward_admission_v1 counters (§10.1). DNS directions are
// not listed: those counters are a different namespace (D-019).
const (
	DirUpload   = "upload"
	DirDownload = "download"
)

// ErrHold means the wall clock stepped backwards by less than one sample
// interval. The sample is skipped so the interval cannot go negative (§10.9).
var ErrHold = fmt.Errorf("meter: wall clock jitter, sample held")

// Observation is one read of the kernel counters and the binding snapshot
// that explains them (§10.2, §10.5).
type Observation struct {
	Counters map[string]enforce.CounterValue
	Bindings []enforce.Binding
	// BootID is the host boot identity. A change means the kernel counters
	// were rebuilt even when the numbers happen to look continuous (§10.7).
	BootID string
}

// CounterName is the nftables counter name enforce uses for one device
// direction. DNS counters use a different prefix and are not usage.
func CounterName(device uuid.UUID, direction string) string {
	hexID := strings.ReplaceAll(device.String(), "-", "")
	switch direction {
	case DirUpload:
		return "u_" + hexID
	case DirDownload:
		return "d_" + hexID
	default:
		return ""
	}
}

func parseUsageCounter(name string) (uuid.UUID, string, bool) {
	var rest, dir string
	switch {
	case strings.HasPrefix(name, "u_"):
		rest, dir = name[len("u_"):], DirUpload
	case strings.HasPrefix(name, "d_"):
		rest, dir = name[len("d_"):], DirDownload
	default:
		return uuid.Nil, "", false
	}
	// Reject the DNS prefixes nu_ and nd_. "nu_" does not start with "u_",
	// but a short or dashed name is still not a device id.
	if len(rest) != 32 {
		return uuid.Nil, "", false
	}
	canon := rest[0:8] + "-" + rest[8:12] + "-" + rest[12:16] + "-" + rest[16:20] + "-" + rest[20:32]
	id, err := uuid.Parse(canon)
	if err != nil {
		return uuid.Nil, "", false
	}
	if strings.ReplaceAll(id.String(), "-", "") != rest {
		return uuid.Nil, "", false
	}
	return id, dir, true
}

type indexed struct {
	byDevice map[uuid.UUID]enforce.Binding
	conflict map[uuid.UUID]bool
}

func indexBindings(bindings []enforce.Binding) indexed {
	out := indexed{
		byDevice: make(map[uuid.UUID]enforce.Binding, len(bindings)),
		conflict: map[uuid.UUID]bool{},
	}
	for _, b := range bindings {
		prev, ok := out.byDevice[b.DeviceID]
		if ok && (prev.MemberID != b.MemberID || prev.BindingRevision != b.BindingRevision) {
			out.conflict[b.DeviceID] = true
		}
		out.byDevice[b.DeviceID] = b
	}
	return out
}

// usageCounters keeps only forward_admission_v1 device counters (D-019).
func usageCounters(obs Observation) []usagestore.Counter {
	idx := indexBindings(obs.Bindings)
	names := make([]string, 0, len(obs.Counters))
	for name := range obs.Counters {
		if _, _, ok := parseUsageCounter(name); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	out := make([]usagestore.Counter, 0, len(names))
	for _, name := range names {
		dev, dir, _ := parseUsageCounter(name)
		v := obs.Counters[name]
		b := idx.byDevice[dev]
		// Two bindings for one device disagree. The bytes cannot be
		// attributed, so the revision stays zero and the sample becomes
		// a gap (§10.2, §10.7).
		if idx.conflict[dev] {
			b = enforce.Binding{}
		}
		out = append(out, usagestore.Counter{
			Name: name, DeviceID: dev, Direction: dir,
			Packets: v.Packets, Bytes: v.Bytes,
			MemberID: b.MemberID, BindingRevision: b.BindingRevision,
		})
	}
	return out
}

func counterMap(list []usagestore.Counter) map[string]enforce.CounterValue {
	out := make(map[string]enforce.CounterValue, len(list))
	for _, c := range list {
		out[c.Name] = enforce.CounterValue{Packets: c.Packets, Bytes: c.Bytes}
	}
	return out
}

func bootChanged(stored, current string) bool {
	if stored == "" && current == "" {
		return false
	}
	return stored != current
}

type plan struct {
	commit usagestore.Commit
	hold   bool
	noop   bool
}

// planContinuity decides whether a process start may keep the open meter
// epoch (§10.7). A short restart with the same boot id and counters that
// have not gone backwards continues the epoch. Anything else opens a new
// epoch and, when bytes may have been lost, a gap event.
func planContinuity(head usagestore.Head, obs Observation, now time.Time) (plan, error) {
	now = now.UTC()
	cur := usageCounters(obs)
	if head.OpenEpoch == uuid.Nil {
		return planBaseline(head, cur, now, obs.BootID), nil
	}
	if !bootChanged(head.BootID, obs.BootID) && len(enforce.DetectResets(counterMap(head.Counters), counterMap(cur))) == 0 {
		return plan{noop: true}, nil
	}
	return planSwitch(head, obs, cur, now, true, DefaultSampleInterval)
}

func planBaseline(head usagestore.Head, cur []usagestore.Counter, now time.Time, bootID string) plan {
	id := uuid.Must(uuid.NewV7())
	reason := "baseline"
	if head.HasHistory {
		if head.LastCloseReason == "final_sample" {
			reason = "after_final_sample"
		} else {
			reason = "continuity_unknown"
		}
	}
	c := usagestore.Commit{
		Counters:  cur,
		Open:      &usagestore.NewEpoch{ID: id, Reason: reason, BootID: bootID},
		BootID:    bootID,
		SampledAt: now,
	}
	// A clean final sample already checkpointed the previous epoch. The new
	// baseline is the counters as they are, not a claim that they are zero
	// (§13.6 step 11). Unknown bytes exist only when this is the first
	// ledger and the counters are already non-zero, or the previous epoch
	// was not closed cleanly.
	needGap := false
	gapReason := ""
	switch {
	case !head.HasHistory && anyBytes(cur):
		needGap = true
		gapReason = string(api.UsageGapReasonUnknown)
	case head.HasHistory && head.LastCloseReason != "final_sample":
		needGap = true
		gapReason = string(api.UsageGapReasonUnknown)
	}
	if needGap {
		start := now.Add(-time.Nanosecond)
		c.Event = &usagestore.Draft{
			EpochID:   id,
			Start:     start,
			End:       now,
			Quality:   string(api.UsageQualityGap),
			GapReason: gapReason,
			Detail:    "unknown counter continuity",
		}
	}
	return plan{commit: c}
}

func anyBytes(list []usagestore.Counter) bool {
	for _, c := range list {
		if c.Bytes > 0 || c.Packets > 0 {
			return true
		}
	}
	return false
}

// planSample builds the next interval commit. final closes the epoch after
// the sample (§13.6 step 8).
func planSample(head usagestore.Head, obs Observation, now time.Time, interval time.Duration, final bool) (plan, error) {
	now = now.UTC()
	if head.OpenEpoch == uuid.Nil {
		if final {
			return plan{}, ErrNoEpoch
		}
		p, err := planContinuity(head, obs, now)
		return p, err
	}
	cur := usageCounters(obs)
	resets := enforce.DetectResets(counterMap(head.Counters), counterMap(cur))
	if bootChanged(head.BootID, obs.BootID) || len(resets) > 0 {
		// The switch commit closes the old epoch. When this is a final
		// sample the caller closes the fresh epoch afterwards so the next
		// process opens its own (§13.6 step 11).
		return planSwitch(head, obs, cur, now, false, interval)
	}
	if !head.LastSample.IsZero() && !now.After(head.LastSample) {
		if head.LastSample.Sub(now) <= interval {
			// A retry of FinalSample after the interval was already
			// checkpointed still closes. Changed counters wait for a
			// forward clock so those bytes are not abandoned (§10.9).
			if final && sameReadings(head.Counters, cur) {
				return plan{commit: usagestore.Commit{
					Epoch:       head.OpenEpoch,
					Counters:    head.Counters,
					BootID:      obs.BootID,
					CloseEpoch:  true,
					CloseReason: "final_sample",
					SampledAt:   head.LastSample,
				}}, nil
			}
			return plan{hold: true}, nil
		}
		return planClockGap(head, obs, now)
	}
	end, estimated := stampEnd(head.LastSample, now, interval)
	if head.LastSample.IsZero() {
		end = now
		estimated = false
	}
	if !end.After(head.LastSample) && !head.LastSample.IsZero() {
		return plan{hold: true}, nil
	}
	recs, mapping := attribute(head, cur, end)
	c := usagestore.Commit{
		Epoch:     head.OpenEpoch,
		Counters:  cur,
		BootID:    obs.BootID,
		SampledAt: end,
	}
	if len(recs) == 0 && !mapping && !estimated {
		c.Counters = cur
		if final {
			c.CloseEpoch = true
			c.CloseReason = "final_sample"
		}
		return plan{commit: c}, nil
	}
	d := usagestore.Draft{
		EpochID: head.OpenEpoch,
		Start:   head.LastSample,
		End:     end,
		Records: recs,
		Quality: string(api.UsageQualityComplete),
	}
	if head.LastSample.IsZero() {
		d.Start = end.Add(-time.Nanosecond)
	}
	switch {
	case mapping:
		d.Quality = string(api.UsageQualityGap)
		d.GapReason = string(api.UsageGapReasonUnknown)
		d.Detail = "mapping conflict"
		if !head.LastSample.IsZero() {
			t := head.LastSample
			d.LastCheckpoint = &t
		}
	case estimated:
		d.Quality = string(api.UsageQualityBoundaryEstimated)
	}
	c.Event = &d
	if final {
		c.CloseEpoch = true
		c.CloseReason = "final_sample"
	}
	return plan{commit: c}, nil
}

func planClockGap(head usagestore.Head, obs Observation, now time.Time) (plan, error) {
	cur := usageCounters(obs)
	start := head.LastSample
	end := start.Add(time.Nanosecond)
	t := head.LastSample
	c := usagestore.Commit{
		Epoch:     head.OpenEpoch,
		Counters:  cur,
		BootID:    obs.BootID,
		SampledAt: end,
		Event: &usagestore.Draft{
			EpochID:        head.OpenEpoch,
			Start:          start,
			End:            end,
			Quality:        string(api.UsageQualityGap),
			GapReason:      string(api.UsageGapReasonClockAnomaly),
			LastCheckpoint: &t,
			Detail:         "wall clock moved backwards",
		},
	}
	return plan{commit: c}, nil
}

func sameReadings(a, b []usagestore.Counter) bool {
	if len(a) != len(b) {
		return false
	}
	by := make(map[string]usagestore.Counter, len(b))
	for _, c := range b {
		by[c.Name] = c
	}
	for _, c := range a {
		d, ok := by[c.Name]
		if !ok || d.Packets != c.Packets || d.Bytes != c.Bytes {
			return false
		}
	}
	return true
}

func planSwitch(head usagestore.Head, obs Observation, cur []usagestore.Counter, now time.Time, opening bool, interval time.Duration) (plan, error) {
	reason := string(api.UsageGapReasonCounterDiscontinuity)
	detail := "counter discontinuity"
	if bootChanged(head.BootID, obs.BootID) {
		reason = string(api.UsageGapReasonPowerLoss)
		detail = "host boot changed"
	}
	// A boot change may have wrapped the counters back up. Apparent
	// increases are not trustworthy (§10.7). A single counter reset still
	// reports the other devices' non-negative deltas.
	var recs []usagestore.DraftRecord
	end := now
	estimated := false
	if !opening && !head.LastSample.IsZero() {
		end, estimated = stampEnd(head.LastSample, now, interval)
		if !bootChanged(head.BootID, obs.BootID) {
			recs, _ = attribute(head, cur, end)
		}
	}
	start := head.LastSample
	if !end.After(start) {
		start = end.Add(-time.Nanosecond)
	}
	var last *time.Time
	if !head.LastSample.IsZero() {
		t := head.LastSample
		last = &t
	}
	id := uuid.Must(uuid.NewV7())
	d := usagestore.Draft{
		EpochID:        head.OpenEpoch,
		Start:          start,
		End:            end,
		Quality:        string(api.UsageQualityGap),
		GapReason:      reason,
		LastCheckpoint: last,
		Records:        recs,
		Detail:         detail,
	}
	if estimated {
		// Gap wins over boundary_estimated: missing bytes are the fact
		// the ledger has to keep (§10.7, §10.9).
		d.Detail += "; boundary sample missed"
	}
	boot := obs.BootID
	if boot == "" {
		boot = head.BootID
	}
	return plan{commit: usagestore.Commit{
		Event:       &d,
		Counters:    cur,
		CloseEpoch:  true,
		CloseReason: reason,
		Open:        &usagestore.NewEpoch{ID: id, Reason: reason, BootID: boot},
		BootID:      boot,
		SampledAt:   end,
	}}, nil
}

func attribute(head usagestore.Head, cur []usagestore.Counter, end time.Time) ([]usagestore.DraftRecord, bool) {
	prev := make(map[string]usagestore.Counter, len(head.Counters))
	for _, c := range head.Counters {
		prev[c.Name] = c
	}
	type acc struct {
		up, down uint64
		member   uuid.UUID
		rev      uint64
		oldMem   uuid.UUID
		oldRev   uint64
		hadOld   bool
	}
	by := map[uuid.UUID]*acc{}
	order := []uuid.UUID{}
	touch := func(id uuid.UUID) *acc {
		a, ok := by[id]
		if ok {
			return a
		}
		a = &acc{}
		by[id] = a
		order = append(order, id)
		return a
	}
	for _, c := range cur {
		p, ok := prev[c.Name]
		var delta uint64
		if !ok {
			delta = c.Bytes
		} else if c.Bytes >= p.Bytes && c.Packets >= p.Packets {
			delta = c.Bytes - p.Bytes
		} else {
			continue
		}
		a := touch(c.DeviceID)
		if c.Direction == DirUpload {
			a.up += delta
		} else {
			a.down += delta
		}
		if c.MemberID != uuid.Nil {
			a.member = c.MemberID
			a.rev = c.BindingRevision
		}
		if ok && (p.MemberID != uuid.Nil || p.BindingRevision != 0) {
			a.hadOld = true
			a.oldMem = p.MemberID
			a.oldRev = p.BindingRevision
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i].String() < order[j].String() })
	period := usagestore.PeriodID(head.LastSample)
	if head.LastSample.IsZero() {
		period = usagestore.PeriodID(end)
	}
	var recs []usagestore.DraftRecord
	mapping := false
	for _, id := range order {
		a := by[id]
		if a.up == 0 && a.down == 0 {
			continue
		}
		if a.member == uuid.Nil || a.rev == 0 || (a.hadOld && (a.oldMem != a.member || a.oldRev != a.rev)) {
			mapping = true
			continue
		}
		recs = append(recs, usagestore.DraftRecord{
			MemberID: a.member, DeviceID: id, BindingRevision: a.rev,
			PeriodID: period, Upload: a.up, Download: a.down,
		})
	}
	return recs, mapping
}

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

const usageIntentKey = "usage_apply_intent"

// usageIntent is written before admission, after any outgoing identity has
// been settled under Baseline. These identities can count in Epoch and its
// uninterrupted same-boot reset successors. A final sample ends that interval.
// This is accounting evidence, never authority to forward.
type usageIntent struct {
	Boot       uuid.UUID       `json:"boot_id"`
	Epoch      uuid.UUID       `json:"epoch_id"`
	Generation decstr.U64      `json:"generation"`
	Bindings   []usageIdentity `json:"bindings"`
}

type usageIdentity struct {
	Device   uuid.UUID  `json:"device_id"`
	Member   uuid.UUID  `json:"member_id"`
	Revision decstr.U64 `json:"binding_revision"`
}

func (p usageIntent) bindings() []enforce.Binding {
	result := make([]enforce.Binding, 0, len(p.Bindings))
	for _, b := range p.Bindings {
		result = append(result, enforce.Binding{DeviceID: b.Device, MemberID: b.Member, BindingRevision: uint64(b.Revision)})
	}
	return result
}

func (e *Engine) recordUsageIntent(ctx context.Context, model enforce.Model) error {
	if e.usage == nil || e.meter == nil {
		return fmt.Errorf("usage: no owned meter for admission")
	}
	h, err := e.usage.Head(ctx)
	if err != nil {
		return err
	}
	if h.OpenEpoch == uuid.Nil || h.BootID != e.bootID.String() {
		return fmt.Errorf("usage: no current epoch for admission")
	}
	if old := e.restartIntent; old != nil && old.Boot == e.bootID && old.Epoch == h.OpenEpoch && uint64(old.Generation) == model.Generation && sameUsageBindings(old.bindings(), model.Bindings) {
		return nil // a same-generation renewal needs no second durable intent
	}
	intent := usageIntent{Boot: e.bootID, Epoch: h.OpenEpoch, Generation: decstr.U64(model.Generation)}
	for _, b := range model.Bindings {
		intent.Bindings = append(intent.Bindings, usageIdentity{Device: b.DeviceID, Member: b.MemberID, Revision: decstr.U64(b.BindingRevision)})
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	if err := e.store.SetMeta(ctx, usageIntentKey, string(raw)); err != nil {
		return err
	}
	e.restartIntent = &intent
	return nil
}

// captureRestartBindings runs before adoption or Baseline can erase the live
// generation. The marker and counting rules share one atomic nft transaction.
// The live retained generation excludes a newer, merely fetched document.
// Generation zero cannot exclude an Apply preceding a crashed withdrawal.
func (e *Engine) captureRestartBindings(ctx context.Context) {
	e.restartBindings, e.restartIntent = nil, nil
	// A failed nft read is not evidence that the durable intent disappeared.
	// The ledger checks its interval below; adoption still verifies live rules.
	if raw, exists, err := e.store.Meta(ctx, usageIntentKey); err == nil && exists {
		var intent usageIntent
		if json.Unmarshal([]byte(raw), &intent) == nil && intent.Boot == e.bootID && intent.Epoch != uuid.Nil && intent.Generation != 0 {
			e.restartIntent = &intent
		}
	}
	reader, ok := e.plane.(interface {
		RestartGeneration(context.Context) (uint64, error)
	})
	if !ok {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, e.servingReadTimeout())
	defer cancel()
	gen, err := reader.RestartGeneration(readCtx)
	if err != nil || readCtx.Err() != nil {
		return
	}
	retained, desired := e.appliedModel, e.restartDesired
	if p := e.restartIntent; p != nil && gen != 0 && gen != uint64(p.Generation) &&
		!(retained != nil && gen == retained.Generation && sameUsageBindings(retained.Bindings, p.bindings())) {
		e.restartIntent = nil // a contradictory live marker defeats the intent
	}
	// Legacy recovery: unique identities among the generations that could
	// have counted. A later checkpoint check rejects unknown older bindings.
	switch {
	case retained != nil && gen == retained.Generation:
		e.restartBindings = retained.Bindings
	case gen == 0 || (desired != nil && gen == uint64(desired.ConfigGeneration)):
		var old []enforce.Binding
		if retained != nil {
			old = retained.Bindings
		}
		if desired != nil && desiredMayCount(*desired) && (retained == nil || uint64(desired.ConfigGeneration) >= retained.Generation) {
			e.restartBindings = uniqueUsageBindings(old, desired.DeviceBindings)
		} else if gen == 0 {
			e.restartBindings = old
		}
	}
}

// Documents that can never render member admission are not candidates. Expiry
// at this restart is not evidence about whether they counted before the crash.
func desiredMayCount(d api.DesiredState) bool {
	if d.PublicationIntent != api.PublicationIntentPublished {
		return false
	}
	_, err := d.NodeCapacity.Get()
	return err == nil
}

func (e *Engine) restartHeadBindings(ctx context.Context, st *usagestore.Store, h usagestore.Head) ([]enforce.Binding, error) {
	if h.OpenEpoch == uuid.Nil || h.BootID != e.bootID.String() {
		return nil, nil
	}
	if p := e.restartIntent; p != nil {
		if p.Epoch == h.OpenEpoch {
			return p.bindings(), nil
		}
		continuous, err := st.SameCountingInterval(ctx, p.Epoch, h.OpenEpoch, e.bootID.String())
		if err != nil {
			return nil, err
		}
		if continuous {
			return p.bindings(), nil
		}
	}
	// Without a scoped intent, an absent/nil-bound checkpoint could conceal
	// an intermediate add/remove/rebind (r22 H-1). Do not charge that entire
	// legacy delta to the latest owner. Bound unchanged devices remain exact;
	// conflicting or unknown devices produce a gap alongside their records.
	known := map[uuid.UUID]enforce.Binding{}
	for _, c := range h.Counters {
		if c.MemberID != uuid.Nil && c.BindingRevision != 0 {
			known[c.DeviceID] = enforce.Binding{DeviceID: c.DeviceID, MemberID: c.MemberID, BindingRevision: c.BindingRevision}
		}
	}
	var safe []enforce.Binding
	for _, b := range e.restartBindings {
		if old, ok := known[b.DeviceID]; ok && old.MemberID == b.MemberID && old.BindingRevision == b.BindingRevision {
			safe = append(safe, b)
		}
	}
	return safe, nil
}

// Identity ignores admission addresses, timeouts and QoS. Adding or removing
// a device also changes the set: its preserved counter may be reused later.
func sameUsageBindings(a, b []enforce.Binding) bool {
	type identity struct {
		member   uuid.UUID
		revision uint64
	}
	index := func(bindings []enforce.Binding) map[uuid.UUID]identity {
		out := make(map[uuid.UUID]identity, len(bindings))
		for _, v := range bindings {
			out[v.DeviceID] = identity{v.MemberID, v.BindingRevision}
		}
		return out
	}
	return maps.Equal(index(a), index(b))
}

// A generation omitting a device has no rule incrementing its usage counter.
// Keep identities present in only one candidate; omit identities that disagree.
func uniqueUsageBindings(retained []enforce.Binding, desired []api.DeviceBinding) []enforce.Binding {
	by := map[uuid.UUID]enforce.Binding{}
	conflict := map[uuid.UUID]bool{}
	add := func(b enforce.Binding) {
		if old, ok := by[b.DeviceID]; ok && (old.MemberID != b.MemberID || old.BindingRevision != b.BindingRevision) {
			conflict[b.DeviceID] = true
		}
		by[b.DeviceID] = b
	}
	for _, b := range retained {
		add(b)
	}
	for _, b := range desired {
		add(enforce.Binding{DeviceID: b.DeviceId, MemberID: b.MemberId, BindingRevision: uint64(b.BindingRevision)})
	}
	var out []enforce.Binding
	for id, b := range by {
		if !conflict[id] {
			out = append(out, b)
		}
	}
	return out
}

// RestartGeneration reads the installed marker before Boot replaces it. It
// uses the plane's executor, so tests and network labs retain their namespace.
func (p SystemPlane) RestartGeneration(ctx context.Context) (uint64, error) {
	raw, err := p.executor().Output(ctx, "nft", "-j", "list", "table", enforce.TableFamily, enforce.TableName)
	if err != nil {
		if nftMissing(err) {
			return 0, nil
		}
		return 0, err
	}
	st, err := enforce.ParseState(raw)
	return st.Generation, err
}

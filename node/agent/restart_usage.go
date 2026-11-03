package agent

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// captureRestartBindings runs before adoption can withdraw rules and before
// any desired refresh. retained_applied can lag an Apply that already admitted
// packets. The persisted desired document bounds that newer candidate; a live
// generation read rejects installations that neither document can explain.
// Generation zero can be a prior crashed Boot's baseline, so it does not erase
// uncertainty about the still-pending interval. This also survives two crashes.
func (e *Engine) captureRestartBindings(ctx context.Context) {
	e.restartBindings = nil
	if e.appliedModel == nil || e.restartDesired == nil {
		return
	}
	retained := e.appliedModel
	desired := e.restartDesired
	if uint64(desired.ConfigGeneration) < retained.Generation {
		return
	}
	reader, ok := e.plane.(interface {
		RestartGeneration(context.Context) (uint64, error)
	})
	if !ok {
		return
	} // no live provenance: a gap is safer than an invented owner
	readCtx, cancel := context.WithTimeout(ctx, e.servingReadTimeout())
	defer cancel()
	gen, err := reader.RestartGeneration(readCtx)
	if err != nil || readCtx.Err() != nil {
		return
	}
	if gen != 0 && gen != retained.Generation && gen != uint64(desired.ConfigGeneration) {
		return
	}
	e.restartBindings = commonUsageBindings(retained.Bindings, desired.DeviceBindings)
}

// Attribution identity is device, member and binding revision. Other model
// fields govern admission, not who owns a counter's bytes. Missing, conflicting
// or rebound identities are omitted; the meter emits their delta as a gap while
// retaining records for the devices common to every plausible generation.
func commonUsageBindings(retained []enforce.Binding, desired []api.DeviceBinding) []enforce.Binding {
	by := map[uuid.UUID]api.DeviceBinding{}
	conflict := map[uuid.UUID]bool{}
	for _, b := range desired {
		if old, ok := by[b.DeviceId]; ok && (old.MemberId != b.MemberId || old.BindingRevision != b.BindingRevision) {
			conflict[b.DeviceId] = true
		}
		by[b.DeviceId] = b
	}
	var common []enforce.Binding
	for _, b := range retained {
		newer, ok := by[b.DeviceID]
		if ok && !conflict[b.DeviceID] && newer.MemberId == b.MemberID && uint64(newer.BindingRevision) == b.BindingRevision {
			common = append(common, b)
		}
	}
	return common
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

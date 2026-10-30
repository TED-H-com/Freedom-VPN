package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
)

const pendingRevocationsKey = "pending_revocations"

func mergeRevocations(pending, dropped []netip.Addr) []netip.Addr {
	all := append(slices.Clone(pending), dropped...)
	slices.SortFunc(all, netip.Addr.Compare)
	return slices.Compact(all)
}

// Retain the entire delete batch until one Apply completes it. Deleting an
// address again is harmless, including after partial progress or a restart.
func (e *Engine) saveRevocations(ctx context.Context, pending []netip.Addr) {
	if slices.Equal(e.pendingRevoke, pending) {
		return
	}
	e.pendingRevoke = slices.Clone(pending)
	raw, err := json.Marshal(pending)
	if err == nil {
		err = e.store.SetMeta(ctx, pendingRevocationsKey, string(raw))
	}
	if err != nil {
		e.storageOK = false
	}
}

func (e *Engine) loadRevocations(ctx context.Context) error {
	raw, ok, err := e.store.Meta(ctx, pendingRevocationsKey)
	if err != nil || !ok {
		return err
	}
	if err := json.Unmarshal([]byte(raw), &e.pendingRevoke); err != nil {
		return errors.New("agent: invalid pending revocations")
	}
	for _, addr := range e.pendingRevoke {
		if !addr.Is4() || addr.IsUnspecified() {
			return errors.New("agent: invalid pending revocations")
		}
	}
	return nil
}

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strconv"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

func compareUUID(a, b uuid.UUID) int {
	return bytes.Compare(a[:], b[:])
}

const (
	pendingRevocationsKey = "pending_revocations"
	conntrackAttemptsKey  = "conntrack_delete_attempts"
	conntrackBlockedKey   = "conntrack_blocked"
)

// maxConntrackDeleteAttempts bounds reserved serving Apply delete batches.
// A crash or pre-delete failure consumes the reservation conservatively.
// This is an attempt bound, not a wall-clock guarantee. Baseline-only cleanup
// may retry after exhaustion without authorizing member forwarding.
// A missing binary does not use this count. LookPath fails closed at once.
const maxConntrackDeleteAttempts = 12

// addrID is one overlay host the last serving apply installed, or one
// host whose conntrack delete is still outstanding. The address alone is
// not an identity: the same IPv4 can move to another device or member.
type addrID struct {
	Addr     netip.Addr `json:"addr"`
	DeviceID uuid.UUID  `json:"device_id"`
	MemberID uuid.UUID  `json:"member_id"`
}

func (a addrID) known() bool {
	return a.DeviceID != uuid.Nil || a.MemberID != uuid.Nil
}

// planRevokes is the address list passed to Apply. A previous or pending
// identity that is not the current (addr, device, member) is deleted.
// The same triple admitted again is not deleted. A migrated row with a
// nil device and member is not that triple: a binding of the same address
// is a reassignment, so the address is queued until conntrack -D succeeds.
func planRevokes(pending, prev []addrID, bindings []enforce.Binding) []netip.Addr {
	current := map[addrID]bool{}
	for _, b := range bindings {
		for _, p := range b.AllowedV4 {
			id := addrID{Addr: p.Addr(), DeviceID: b.DeviceID, MemberID: b.MemberID}
			current[id] = true
		}
	}
	var out []netip.Addr
	seen := map[netip.Addr]bool{}
	consider := func(id addrID) {
		if !id.Addr.Is4() || seen[id.Addr] || current[id] {
			return
		}
		seen[id.Addr] = true
		out = append(out, id.Addr)
	}
	for _, id := range pending {
		consider(id)
	}
	for _, id := range prev {
		consider(id)
	}
	slices.SortFunc(out, netip.Addr.Compare)
	return out
}

// pendingAfterFailure keeps the queued identities whose conntrack -D failed.
// failed is ConntrackDeleteError.Failed. A re-admitted (addr, device, member)
// is not kept. Every other address of that revoke set is retired.
func pendingAfterFailure(pending, prev []addrID, bindings []enforce.Binding, failed []netip.Addr) []addrID {
	if len(failed) == 0 {
		return nil
	}
	failSet := map[netip.Addr]struct{}{}
	for _, addr := range failed {
		failSet[addr] = struct{}{}
	}
	revoke := map[netip.Addr]bool{}
	for _, addr := range planRevokes(pending, prev, bindings) {
		revoke[addr] = true
	}
	var out []addrID
	seen := map[addrID]bool{}
	keep := func(id addrID) {
		if _, ok := failSet[id.Addr]; !ok || !revoke[id.Addr] || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range pending {
		keep(id)
	}
	for _, id := range prev {
		keep(id)
	}
	slices.SortFunc(out, func(a, b addrID) int {
		if c := a.Addr.Compare(b.Addr); c != 0 {
			return c
		}
		if c := compareUUID(a.DeviceID, b.DeviceID); c != 0 {
			return c
		}
		return compareUUID(a.MemberID, b.MemberID)
	})
	return out
}

func bindingIDs(m enforce.Model) []addrID {
	var out []addrID
	seen := map[addrID]bool{}
	for _, b := range m.Bindings {
		for _, p := range b.AllowedV4 {
			id := addrID{Addr: p.Addr(), DeviceID: b.DeviceID, MemberID: b.MemberID}
			if !id.Addr.Is4() || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
		}
	}
	slices.SortFunc(out, func(a, b addrID) int {
		if c := a.Addr.Compare(b.Addr); c != 0 {
			return c
		}
		if c := compareUUID(a.DeviceID, b.DeviceID); c != 0 {
			return c
		}
		return compareUUID(a.MemberID, b.MemberID)
	})
	return out
}

func sameAddrs(a, b []addrID) bool {
	return slices.Equal(a, b)
}

// Retain the identities whose deletes have not succeeded. Deleting an
// address again is harmless, including after partial progress or a restart.
func (e *Engine) saveRevocations(ctx context.Context, pending []addrID) {
	if sameAddrs(e.pendingRevoke, pending) {
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
	parsed, err := parsePendingRevocations(raw)
	if err != nil {
		return err
	}
	e.pendingRevoke = parsed
	return nil
}

func parsePendingRevocations(raw string) ([]addrID, error) {
	text := raw
	if text == "" || text == "null" {
		return nil, nil
	}
	var keyed []addrID
	if err := json.Unmarshal([]byte(text), &keyed); err == nil {
		for _, id := range keyed {
			if !id.Addr.Is4() || id.Addr.IsUnspecified() {
				return nil, errors.New("agent: invalid pending revocations")
			}
		}
		return keyed, nil
	}
	var flat []netip.Addr
	if err := json.Unmarshal([]byte(text), &flat); err != nil {
		return nil, errors.New("agent: invalid pending revocations")
	}
	out := make([]addrID, 0, len(flat))
	for _, addr := range flat {
		if !addr.Is4() || addr.IsUnspecified() {
			return nil, errors.New("agent: invalid pending revocations")
		}
		out = append(out, addrID{Addr: addr})
	}
	return out, nil
}

func (e *Engine) loadConntrackState(ctx context.Context) error {
	raw, ok, err := e.store.Meta(ctx, conntrackAttemptsKey)
	if err != nil {
		return err
	}
	if ok && raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return errors.New("agent: invalid conntrack delete attempts")
		}
		e.conntrackAttempts = n
	}
	blocked, ok, err := e.store.Meta(ctx, conntrackBlockedKey)
	if err != nil {
		return err
	}
	e.conntrackBlocked = ok && blocked == "1"
	if e.conntrackAttempts >= maxConntrackDeleteAttempts {
		e.conntrackBlocked = true
	}
	if e.conntrackBlocked && e.conntrackDetail == "" {
		e.conntrackDetail = conntrackMissingDetail
		if e.conntrackAttempts >= maxConntrackDeleteAttempts {
			e.conntrackDetail = conntrackBoundDetail
		}
	}
	return nil
}

func (e *Engine) persistConntrack(ctx context.Context) {
	blocked := "0"
	if e.conntrackBlocked {
		blocked = "1"
	}
	if err := e.store.SetMeta(ctx, conntrackAttemptsKey, strconv.Itoa(e.conntrackAttempts)); err != nil {
		e.storageOK = false
	}
	if err := e.store.SetMeta(ctx, conntrackBlockedKey, blocked); err != nil {
		e.storageOK = false
	}
}

func (e *Engine) reserveConntrack(ctx context.Context, model enforce.Model) error {
	pending := pendingAfterFailure(e.pendingRevoke, e.addresses, model.Bindings, model.Revoke)
	raw, err := json.Marshal(pending)
	if err == nil {
		err = e.store.SetMetaBatch(ctx, map[string]string{
			pendingRevocationsKey: string(raw),
			conntrackAttemptsKey:  strconv.Itoa(e.conntrackAttempts + 1),
		})
	}
	if err != nil {
		e.storageOK = false
		return err
	}
	e.pendingRevoke = pending
	e.conntrackAttempts++
	return nil
}

func (e *Engine) conntrackMissing() bool {
	looker, ok := e.plane.(interface {
		ConntrackLookPath() error
	})
	if !ok {
		return false
	}
	return looker.ConntrackLookPath() != nil
}

func (e *Engine) markConntrackBlocked(ctx context.Context, detail string) {
	e.conntrackBlocked = true
	e.conntrackDetail = detail
	e.persistConntrack(ctx)
}

func (e *Engine) clearConntrackBlock(ctx context.Context) {
	e.conntrackAttempts = 0
	e.conntrackBlocked = false
	e.conntrackDetail = ""
	e.persistConntrack(ctx)
}

// flushPendingConntrack retries deletes without installing a serving
// ruleset. Planes that do not expose a flusher stay failed closed. The
// failed subset is what stays pending, as on the serving Apply path.
func (e *Engine) flushPendingConntrack(ctx context.Context) bool {
	if len(e.pendingRevoke) == 0 {
		return true
	}
	flusher, ok := e.plane.(interface {
		FlushConntrack(context.Context, []netip.Addr) ([]netip.Addr, error)
	})
	if !ok {
		return false
	}
	addrs := make([]netip.Addr, 0, len(e.pendingRevoke))
	seen := map[netip.Addr]bool{}
	for _, id := range e.pendingRevoke {
		if seen[id.Addr] {
			continue
		}
		seen[id.Addr] = true
		addrs = append(addrs, id.Addr)
	}
	failed, err := flusher.FlushConntrack(ctx, addrs)
	if err != nil {
		return false
	}
	if len(failed) == 0 {
		e.saveRevocations(ctx, nil)
		return true
	}
	failSet := map[netip.Addr]bool{}
	for _, addr := range failed {
		failSet[addr] = true
	}
	var keep []addrID
	for _, id := range e.pendingRevoke {
		if failSet[id.Addr] {
			keep = append(keep, id)
		}
	}
	e.saveRevocations(ctx, keep)
	return false
}

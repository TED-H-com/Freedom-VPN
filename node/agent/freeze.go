package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// A freeze is an incarnation latch, never cleared by an absent/false response
// or by a later signed envelope. Resolution is revoke and re-enroll.
func (e *Engine) freezeIdentity(ctx context.Context) {
	if e.identityFrozen {
		e.persistFreeze(ctx)
		return
	}
	e.identityFrozen = true
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	// Stop the member path before persistence or control I/O can block.
	e.withdraw(cleanup)
	e.persistFreeze(cleanup)
	e.decision = Decide(e.input())
	e.lastObs = time.Time{}
	e.beat()
	// D-088 overrides both a renewal already attempted this step and D-087's
	// scheduled slot. Later renewals use the ordinary grant-less schedule.
	e.leaseAttempted = false
	e.leaseRetryAt = 0
	e.leaseFresh = false
	_ = e.syncLease(cleanup)
	e.decision = Decide(e.input())
}

// The sidecar survives a locked SQLite writer. An in-flight observation is
// independently journaled before HTTP, so a failed sidecar write (disk full)
// still requires replay of the uncertain response before any new forwarding.
func (e *Engine) persistFreeze(ctx context.Context) {
	if e.freezeDurable {
		return
	}
	_ = writeAtomic(filepath.Join(e.cfg.StateDir, "identity-freeze.intent"), []byte(e.incID.String()+"\n"), 0o600)
	if err := e.store.SetMeta(ctx, "identity_frozen", "1"); err != nil {
		e.storageOK = false
		return
	}
	e.freezeDurable = true
}

func (e *Engine) loadFreeze(ctx context.Context) error {
	v, _, err := e.store.Meta(ctx, "identity_frozen")
	if err != nil {
		return err
	}
	e.identityFrozen, e.freezeDurable = e.identityFrozen || v == "1", v == "1"
	if raw, err := os.ReadFile(filepath.Join(e.cfg.StateDir, "identity-freeze.intent")); err == nil {
		id, ok, readErr := e.store.Identity(ctx)
		if readErr != nil {
			return readErr
		}
		inc := uuid.Nil
		if ok {
			inc = id.IncarnationID
		}
		if strings.TrimSpace(string(raw)) == inc.String() {
			e.identityFrozen = true
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if e.identityFrozen {
		e.persistFreeze(ctx)
	}
	items, err := e.store.Pending(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Attempts > 0 {
			e.freezeUncertain = true
		}
	}
	return nil
}

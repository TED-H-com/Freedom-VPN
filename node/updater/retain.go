package updater

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// Retain keeps current, last_good, the active transaction's releases, open
// jobs and any release named by a result that is not yet acknowledged.
// It never deletes identity, state or the usage outbox (§13.7, §13.8).
func (e *Engine) Retain(ctx context.Context) error {
	if e.Journal == nil {
		return nil
	}
	return e.withLock(ctx, func() error { return e.retainLocked(ctx) })
}

func (e *Engine) retainLocked(ctx context.Context) error {
	keep := map[string]bool{}
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return err
	}
	keep[p.CurrentRelease] = p.CurrentRelease != ""
	keep[p.LastGood] = p.LastGood != ""
	if txn, err := e.Journal.ActiveTxn(ctx); err == nil {
		keep[txn.ReleaseID] = txn.ReleaseID != ""
		keep[txn.PreviousReleaseID] = txn.PreviousReleaseID != ""
		keep[txn.LastGoodReleaseID] = txn.LastGoodReleaseID != ""
		keep[txn.RollbackReleaseID] = txn.RollbackReleaseID != ""
	}
	open, err := e.Journal.OpenJobs(ctx)
	if err != nil {
		return err
	}
	for _, job := range open {
		keep[job.ReleaseID] = job.ReleaseID != ""
	}
	box, err := e.Journal.Outbox(ctx)
	if err != nil {
		return err
	}
	for _, ent := range box {
		var res api.UpdateResult
		if json.Unmarshal(ent.Payload, &res) != nil {
			continue
		}
		if id, err := res.CurrentReleaseId.Get(); err == nil {
			keep[id.String()] = true
		}
		if id, err := res.PreviousReleaseId.Get(); err == nil {
			keep[id.String()] = true
		}
		if res.RequestedReleaseId != uuid.Nil {
			keep[res.RequestedReleaseId.String()] = true
		}
	}
	ids, err := e.Journal.ListReleases(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if keep[id] {
			continue
		}
		dir := e.Layout.releaseDir(id)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := e.Journal.DeleteRelease(ctx, id); err != nil {
			return err
		}
	}
	arts, err := e.Journal.ListArtifacts(ctx)
	if err != nil {
		return err
	}
	for _, a := range arts {
		if keep[a.ReleaseID] {
			continue
		}
		if a.Path != "" {
			_ = os.Remove(a.Path)
		}
		if err := e.Journal.DeleteArtifact(ctx, a.SHA256); err != nil {
			return err
		}
	}
	// Drop partial downloads. Verified artifact files were handled above.
	entries, err := os.ReadDir(e.Layout.Staging)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, ent := range entries {
		name := ent.Name()
		if len(name) > 8 && (name[len(name)-8:] == ".partial" || filepath.Ext(name) == ".meta") {
			_ = os.Remove(filepath.Join(e.Layout.Staging, name))
		}
	}
	return nil
}

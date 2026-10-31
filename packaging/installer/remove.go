package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// Remove stops and disables the product units and deletes only the nft
// table, netns and interfaces this installer created (§6.3). Identity, the
// usage store and the update journal stay unless purge is set.
func Remove(ctx context.Context, env *Env, purge bool) (*Result, error) {
	if env == nil {
		env = &Env{}
	}
	if err := requireSystemd(env); err != nil {
		return nil, err
	}
	l := env.layout()
	lk, err := tryLock(ctx, l)
	if err != nil {
		if errors.Is(err, errBusy) {
			return nil, errf(ExitBusy, "busy", "update lock is held")
		}
		lk = nil
	}
	if lk != nil {
		defer lk.Release()
	}
	for _, unit := range stopOrder {
		_ = env.services().Stop(ctx, unit)
		_ = env.services().Disable(ctx, unit)
		_ = os.Remove(filepath.Join(l.SystemdDir, unit))
	}
	_ = env.services().DaemonReload(ctx)
	removeOwnedNetwork(ctx, env)
	msg := "removed: units and owned network objects; identity, usage store and update journal kept"
	if purge {
		// Read owned before the install directory goes away.
		if lk != nil {
			_ = lk.Release()
			lk = nil
		}
		for _, p := range []string{
			l.IdentityDir, l.RetiredDir, l.StateDir, l.UpdateJournal,
			l.StagingDir, l.InstallDir, l.OptDir, l.EtcDir, l.LockPath,
		} {
			if err := os.RemoveAll(p); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
		}
		_ = os.RemoveAll(l.VarDir)
		msg = "purge: removed identity, usage store and update journal"
	}
	return &Result{Status: statusRemoved, Message: msg}, nil
}

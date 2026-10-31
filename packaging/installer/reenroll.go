package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

// identityNames are the agent identity files (§6.1). reenroll moves them.
// It does not delete them (§6.2, AT07).
var identityNames = []string{
	"key.pem", "cert.pem", "csr.pem", "ca.pem", "setup.key", "setup.meta",
	"node.json", "enrollment.json", "renew.csr",
}

// stateNamesBound are incarnation-bound files. The usage store is not in
// this list. usagestore.Open returns ErrIdentity when the node or
// incarnation differs and does not upload that file. usageupload resends
// stored canonical bytes and has no re-home path. Central rejects a batch
// whose incarnation disagrees with the mTLS identity (403). The old
// incarnation's unacknowledged events stay on disk, unuploaded.
var stateNamesBound = []string{"agent.db", "agent.db-wal", "agent.db-shm", "desired-state.json"}

// Reenroll stops serving, retires the local identity, and enrolls a new
// incarnation with fvpn-agent (§6.2, AT07). The usage store is not moved.
func Reenroll(ctx context.Context, env *Env, tokenFile string) (*Result, error) {
	if env == nil {
		env = &Env{}
	}
	if err := checkTokenFile(tokenFile); err != nil {
		return nil, err
	}
	if err := requireSystemd(env); err != nil {
		return nil, err
	}
	l := env.layout()
	if !identityPresent(l) {
		return nil, errf(ExitFailed, "failed", "no enrolled identity to retire")
	}
	lk, err := tryLock(ctx, l)
	if err != nil {
		if errors.Is(err, errBusy) {
			return nil, errf(ExitBusy, "busy", "update lock is held")
		}
		return nil, err
	}
	released := false
	release := func() {
		if !released && lk != nil {
			_ = lk.Release()
			released = true
		}
	}
	defer release()
	for _, unit := range stopOrder {
		_ = env.services().Stop(ctx, unit)
	}
	stamp := env.now().UTC().Format("20060102T150405Z")
	dest := filepath.Join(l.RetiredDir, stamp)
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dest, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(l.RetiredDir, 0o700); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	moved := [][2]string{}
	move := func(src, dst string) error {
		if _, err := os.Lstat(src); err != nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		moved = append(moved, [2]string{src, dst})
		return nil
	}
	for _, name := range identityNames {
		if err := move(filepath.Join(l.IdentityDir, name), filepath.Join(dest, name)); err != nil {
			restore(moved)
			return nil, err
		}
	}
	for _, name := range stateNamesBound {
		if err := move(filepath.Join(l.StateDir, name), filepath.Join(dest, "state-"+name)); err != nil {
			restore(moved)
			return nil, err
		}
	}
	if err := moveTree(l.NetbirdLib, filepath.Join(dest, "netbird-lib"), &moved); err != nil {
		restore(moved)
		return nil, err
	}
	if err := moveTree(l.NetbirdEtc, filepath.Join(dest, "netbird-etc"), &moved); err != nil {
		restore(moved)
		return nil, err
	}
	if err := env.run(ctx, l.agentBin(), "enroll", "-config", l.agentConfig(), "-token-file", tokenFile); err != nil {
		restore(moved)
		return nil, errf(ExitFailed, "failed", "enrollment failed")
	}
	_ = shredFile(tokenFile)
	if err := refreshUpdaterIdentity(l); err != nil {
		return nil, err
	}
	release()
	for _, unit := range startOrder {
		if err := env.services().Start(ctx, unit); err != nil {
			return nil, errf(ExitFailed, "failed", "starting %s failed", unit)
		}
	}
	baseline, err := seedBaseline(ctx, env)
	if err != nil {
		return nil, err
	}
	node, err := readNode(l)
	if err != nil {
		return nil, err
	}
	return &Result{
		Status: statusReenrolled, IncarnationID: node.NodeIncarnationID, Baseline: baseline,
		Message: "identity retired under identity.retired; usage store kept",
	}, nil
}

func restore(moved [][2]string) {
	for i := len(moved) - 1; i >= 0; i-- {
		_ = os.Rename(moved[i][1], moved[i][0])
	}
}

func moveTree(src, dst string, moved *[][2]string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	for _, e := range entries {
		from := filepath.Join(src, e.Name())
		to := filepath.Join(dst, e.Name())
		if err := os.Rename(from, to); err != nil {
			return err
		}
		*moved = append(*moved, [2]string{from, to})
	}
	return nil
}

func refreshUpdaterIdentity(l Layout) error {
	_, err := updater.LoadConfig(l.updaterConfig())
	if err != nil {
		return errf(ExitFailed, "failed", "updater config cannot be read")
	}
	node, err := readNode(l)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(node.NodeID)
	if err != nil {
		return errf(ExitFailed, "failed", "identity node_id is not a UUID")
	}
	inc, err := uuid.Parse(node.NodeIncarnationID)
	if err != nil {
		return errf(ExitFailed, "failed", "identity incarnation is not a UUID")
	}
	// Preserve the configured fallback (including its absence). LoadConfig
	// resolves the kernel boot id for runtime use, not for writing to disk.
	original, err := os.ReadFile(l.updaterConfig())
	if err != nil {
		return err
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(original, &cfg); err != nil {
		return err
	}
	cfg["node_id"], _ = json.Marshal(id)
	cfg["incarnation_id"], _ = json.Marshal(inc)
	raw, err := marshalIndent(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(l.updaterConfig(), raw, 0o644)
}

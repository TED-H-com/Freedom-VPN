package installer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ImagePrepare removes generated secrets so a disk image can be cloned
// (§6.3, ENR03). It refuses while an enrolled identity is present unless
// the caller passes the explicit image-build flag. Binaries, units and the
// first-boot hook stay.
func ImagePrepare(ctx context.Context, env *Env, force bool) (*Result, error) {
	if env == nil {
		env = &Env{}
	}
	l := env.layout()
	if identityPresent(l) || fileExists(filepath.Join(l.IdentityDir, "key.pem")) {
		if !force {
			return nil, errf(ExitFailed, "identity_present", "refusing to strip an enrolled identity without --i-am-building-an-image")
		}
	}
	if err := requireSystemd(env); err != nil {
		return nil, err
	}
	for _, unit := range stopOrder {
		_ = env.services().Stop(ctx, unit)
	}
	for _, name := range identityNames {
		_ = os.Remove(filepath.Join(l.IdentityDir, name))
	}
	for _, name := range append(append([]string{}, stateNamesBound...), "usage.db", "usage.db-wal", "usage.db-shm") {
		_ = os.Remove(filepath.Join(l.StateDir, name))
	}
	_ = removeDirContents(l.NetbirdLib)
	_ = removeDirContents(l.NetbirdEtc)
	if err := removeSSHHostKeys(l.SSHDir); err != nil {
		return nil, err
	}
	if err := resetMachineID(l.MachineID); err != nil {
		return nil, err
	}
	if err := resetMachineIDIfRegular(l.DBusMachineID); err != nil {
		return nil, err
	}
	raw, err := marshalIndent(firstbootConfig{TokenFile: l.firstbootToken()})
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(l.EtcDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(l.firstbootFile(), raw, 0o644); err != nil {
		return nil, err
	}
	// A clone must not start the agent or netbird-up before enrollment.
	// netbird-up waits for setup.key and would hold the boot transaction
	// (§6.3). firstboot enables the member units after enroll.
	for _, unit := range requiredUnits {
		if unit == UnitFirstboot {
			if err := env.services().Enable(ctx, unit); err != nil {
				return nil, err
			}
			continue
		}
		_ = env.services().Disable(ctx, unit)
	}
	return &Result{
		Status:  statusPrepared,
		Message: "image prepared; place a 0600 token at the first-boot path before the first boot",
	}, nil
}

// FirstBoot enrolls a prepared image when the token file is present.
// An image that already has identity exits 0 without minting another key.
func FirstBoot(ctx context.Context, env *Env) (*Result, error) {
	if env == nil {
		env = &Env{}
	}
	l := env.layout()
	if identityPresent(l) {
		node, _ := readNode(l)
		return &Result{Status: statusAlreadyEnrolled, IncarnationID: node.NodeIncarnationID}, nil
	}
	if err := ensureMachineID(l); err != nil {
		return nil, err
	}
	token := l.firstbootToken()
	if raw, err := os.ReadFile(l.firstbootFile()); err == nil {
		var cfg firstbootConfig
		if json.Unmarshal(raw, &cfg) == nil && cfg.TokenFile != "" {
			token = cfg.TokenFile
		}
	}
	if err := checkTokenFile(token); err != nil {
		return nil, err
	}
	lk, err := tryLock(ctx, l)
	if err != nil {
		if errors.Is(err, errBusy) {
			return nil, errf(ExitBusy, "busy", "update lock is held")
		}
		return nil, err
	}
	defer lk.Release()
	if err := env.run(ctx, l.agentBin(), "enroll", "-config", l.agentConfig(), "-token-file", token); err != nil {
		return nil, errf(ExitFailed, "failed", "enrollment failed")
	}
	_ = shredFile(token)
	if err := refreshUpdaterIdentity(l); err != nil {
		return nil, err
	}
	_ = lk.Release()
	for _, unit := range requiredUnits {
		if unit == UnitFirstboot {
			continue
		}
		if err := env.services().Enable(ctx, unit); err != nil {
			return nil, errf(ExitFailed, "failed", "enabling %s failed", unit)
		}
	}
	for _, unit := range startOrder {
		if err := env.services().Start(ctx, unit); err != nil {
			return nil, errf(ExitFailed, "failed", "starting %s failed: %s", unit, oneLine(err.Error()))
		}
	}
	node, err := readNode(l)
	if err != nil {
		return nil, err
	}
	return &Result{Status: statusEnrolled, IncarnationID: node.NodeIncarnationID}, nil
}

// NetbirdUp waits for the setup key file and execs `netbird up` with that
// path. The key is not passed as an argument value (§18.2).
func NetbirdUp(ctx context.Context, env *Env) error {
	if env == nil {
		env = &Env{}
	}
	l := env.layout()
	raw, err := os.ReadFile(l.netbirdFile())
	if err != nil {
		return errf(ExitFailed, "failed", "netbird config is missing")
	}
	var cfg netbirdConfig
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.ManagementURL == "" {
		return errf(ExitFailed, "failed", "netbird config is missing the management URL")
	}
	key := filepath.Join(l.IdentityDir, "setup.key")
	deadline := time.Now().Add(60 * time.Second)
	for {
		if st, err := os.Lstat(key); err == nil && st.Mode().IsRegular() {
			break
		}
		if time.Now().After(deadline) {
			return errf(ExitFailed, "failed", "setup key file was not written")
		}
		select {
		case <-ctx.Done():
			return errf(ExitFailed, "failed", "setup key file was not written")
		case <-time.After(200 * time.Millisecond):
		}
	}
	return env.run(ctx, "netbird", "up",
		"--setup-key-file", key,
		"--management-url", cfg.ManagementURL,
		"--interface-name", cfg.InterfaceName,
		"--disable-dns", "--disable-ipv6", "--disable-firewall",
		"--log-level", "warn", "--log-file", "console")
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func removeDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func removeSSHHostKeys(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "ssh_host_") {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func resetMachineID(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.WriteFile(path, nil, 0o644)
}

func resetMachineIDIfRegular(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	return os.WriteFile(path, nil, 0o644)
}

func ensureMachineID(l Layout) error {
	if err := writeMachineIDIfEmpty(l.MachineID); err != nil {
		return err
	}
	st, err := os.Lstat(l.DBusMachineID)
	if err != nil || st.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	return writeMachineIDIfEmpty(l.DBusMachineID)
}

func writeMachineIDIfEmpty(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := strings.TrimSpace(string(raw))
	if text != "" && text != "uninitialized" {
		return nil
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	id := hex.EncodeToString(buf) + "\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(id), 0o644)
}

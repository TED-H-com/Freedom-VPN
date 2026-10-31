package installer

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// netbirdKeyURL and the apt line are the official repository published by
// the NetBird install script for v0.82.0. This installer does not run that
// script: the script installs an unpinned netbird and may install a UI.
const (
	netbirdKeyURL = "https://pkgs.netbird.io/debian/public.key"
	netbirdSuite  = "stable"
)

func netbirdListLine(keyring string) string {
	return fmt.Sprintf("deb [signed-by=%s] https://pkgs.netbird.io/debian %s main\n", keyring, netbirdSuite)
}

// ensureNetbird installs the pinned client only when the binary is missing.
// A different installed version is a dependency failure, not an apt change.
func ensureNetbird(ctx context.Context, env *Env, j *Journal) (bool, error) {
	path, err := env.look("netbird")
	if err == nil && path != "" {
		out, verr := env.output(ctx, "netbird", "version")
		if verr != nil {
			return false, errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "netbird version could not be read")
		}
		if !netbirdVersionOK(string(out)) {
			return false, errf(ExitDependency, "NETBIRD_VERSION", "installed netbird is not %s", NetBirdPin)
		}
		return false, nil
	}
	if _, err := env.look("curl"); err != nil {
		return false, errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "missing tool curl")
	}
	if _, err := env.look("gpg"); err != nil {
		return false, errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "missing tool gpg")
	}
	l := env.layout()
	if err := j.MkdirAll(filepathDir(l.Keyring), 0o755); err != nil {
		return false, err
	}
	if err := j.MkdirAll(filepathDir(l.AptList), 0o755); err != nil {
		return false, err
	}
	// Record the keyring path before curl writes it. Rollback removes it
	// when it was absent. The bytes are a public repository key.
	if err := j.record(Record{Action: "write", Path: l.Keyring, Prev: "absent", Mode: "0644"}); err != nil {
		return false, err
	}
	asc := l.Keyring + ".asc"
	if err := env.run(ctx, "curl", "-fsSL", "-o", asc, netbirdKeyURL); err != nil {
		return false, errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "cannot fetch the NetBird repository key")
	}
	if err := env.run(ctx, "gpg", "--batch", "--yes", "--dearmor", "-o", l.Keyring, asc); err != nil {
		_ = os.Remove(asc)
		return false, errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "cannot install the NetBird repository key")
	}
	_ = os.Remove(asc)
	line := []byte(netbirdListLine(l.Keyring))
	if err := j.WriteFile(l.AptList, line, 0o644); err != nil {
		return false, err
	}
	if err := env.run(ctx, "apt-get", "update"); err != nil {
		return false, errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "apt-get update failed")
	}
	if err := j.Note("package", "netbird", "netbird", "installed "+NetBirdPin+" from pkgs.netbird.io"); err != nil {
		return false, err
	}
	if err := env.run(ctx, "apt-get", "install", "-y", "--no-install-recommends", "netbird="+NetBirdPin); err != nil {
		return false, errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "apt-get install netbird failed")
	}
	out, err := env.output(ctx, "netbird", "version")
	if err != nil || !netbirdVersionOK(string(out)) {
		return false, errf(ExitDependency, "NETBIRD_VERSION", "installed netbird is not %s", NetBirdPin)
	}
	return true, nil
}

func netbirdVersionOK(out string) bool {
	for _, f := range strings.Fields(out) {
		f = strings.TrimPrefix(f, "v")
		f = strings.Trim(f, ",;")
		if f == NetBirdPin || strings.HasPrefix(f, NetBirdPin+"-") || strings.HasPrefix(f, NetBirdPin+"+") {
			return true
		}
	}
	return false
}

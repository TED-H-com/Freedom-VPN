package installer

import (
	"context"
	"os"
	"strings"
)

// requiredTools are the host programs §6.3 and the agent executor need.
// A missing tool is a refusal before any directory or journal is created.
var requiredTools = []string{"nft", "tc", "conntrack", "ip", "systemctl"}

func checkTools(env *Env) error {
	for _, name := range requiredTools {
		if _, err := env.look(name); err != nil {
			return errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "missing tool %s", name)
		}
	}
	return nil
}

// probeWireGuard creates and deletes an fv-prefixed interface (§4).
// It does not modprobe and does not change the default route. The kernel
// module has to already be available; the minimum kernel version is 待實測
// and this check does not invent one.
func probeWireGuard(ctx context.Context, env *Env) error {
	const name = "fvwprobe"
	if err := env.run(ctx, "ip", "link", "add", name, "type", "wireguard"); err != nil {
		_ = env.run(ctx, "ip", "link", "delete", name)
		return errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "wireguard is not available")
	}
	if err := env.run(ctx, "ip", "link", "delete", name); err != nil {
		return errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "wireguard probe interface could not be removed")
	}
	return nil
}

func hasSystemd(env *Env) bool {
	if env != nil && (env.AssumeSystemd || env.Services != nil) {
		return true
	}
	if os.Getenv("FVPN_INSTALL_SERVICE_MANAGER") == "file" {
		return true
	}
	if env == nil {
		return false
	}
	if _, err := env.look("systemctl"); err != nil {
		return false
	}
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	raw, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(raw)) == "systemd"
}

func requireSystemd(env *Env) error {
	if hasSystemd(env) {
		return nil
	}
	return errf(ExitDependency, "DEPENDENCY_UNAVAILABLE", "systemd is required")
}

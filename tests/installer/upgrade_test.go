//go:build installer

package installertest

import (
	"strings"
	"testing"
)

func TestF905_CLIDowngradeHasNoStagingEffects(t *testing.T) {
	const c = "fvpn-w5-node-installer-downgrade"
	startSystemd(t, c)
	stage(t, c)
	url, ca, bundle := startControl(t, c)
	writeStdin(t, c, "/run/w5-token", token1+"\n", "0600")
	_, out, code := doInstall(t, c, relIn("2", rel2.ReleaseID), url, ca, bundle, "/run/w5-token")
	if code != 0 {
		t.Fatalf("install: %d %s", code, out)
	}
	assertCLIDowngrade(t, c)
}

func assertCLIDowngrade(t *testing.T, c string) {
	t.Helper()
	snapshot := func() string {
		return mustExec(t, c, "sh", "-c", `find /opt/vpn-node/releases /var/lib/vpn-node/update-staging -printf '%p %m %s %T@\n' | sort`) +
			mustExec(t, c, "sh", "-c", `find /opt/vpn-node/releases /var/lib/vpn-node/update-staging -type f -exec sha256sum {} + | sort`) +
			mustExec(t, c, "readlink", "/opt/vpn-node/current") +
			mustExec(t, c, "nft", "-j", "list", "table", "inet", "fvpn_install")
	}
	before := snapshot()
	lower := relIn("1", rel1.ReleaseID)
	out, code := dexec(t, c, "/usr/local/bin/fvpn-node-install", "upgrade", "--json",
		"--manifest", lower.Manifest, "--artifact", lower.Artifact, "--trust", lower.Trust)
	if code != 1 || !strings.Contains(out, "DOWNGRADE") || strings.Contains(out, `"staged"`) {
		t.Fatalf("downgrade: exit=%d %s", code, out)
	}
	if snapshot() != before {
		t.Fatal("downgrade changed release/staging bytes, modes, mtimes, current or marker")
	}
	t.Log("F9-05: CLI DOWNGRADE exit 1; release/staging bytes, modes, mtimes, pending record, current and marker unchanged")
}

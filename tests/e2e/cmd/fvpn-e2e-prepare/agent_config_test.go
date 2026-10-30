package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
)

func TestD061_OnlyRuntimeConfigHasHostLocalResolver(t *testing.T) {
	state := t.TempDir()
	if err := writeAgentConfigs(state, "192.0.3"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"agent-enroll.json", "agent-run.json"} {
		cfg, err := agent.LoadConfig(filepath.Join(state, "node", "etc", name))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.UpdaterConfig != "/etc/vpn-node/updater-recover.json" {
			t.Fatal("recovery depends on one-shot trigger")
		}
		if name == "agent-enroll.json" && cfg.DNS != nil {
			t.Fatal("enrollment configured resolver")
		}
		if name == "agent-run.json" && (cfg.DNS == nil || len(cfg.DNS.Resolvers) != 1 || cfg.DNS.Resolvers[0].AdminURL != "http://127.0.0.1:8080" || cfg.DNS.Resolvers[0].DNSAddr != "127.0.0.1:53") {
			t.Fatal("runtime resolver wiring missing")
		}
	}
	if err := writeAdGuard(state, "172.28.30.54"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(state, "node", "etc", "resolver-credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatal("credential permissions")
	}
}

// Command fvpn-e2e-nodeconfig writes TEST ONLY boot recovery configuration
// after real enrollment. Identity keys remain inside the node.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

func main() {
	run := prepare
	if len(os.Args) >= 2 && os.Args[1] == "probe-client" {
		run = probeClient
	}
	if len(os.Args) == 2 && os.Args[1] == "probe-watch" {
		run = probeWatch
	}
	if len(os.Args) == 2 && os.Args[1] == "dns-probe" {
		run = dnsProbe
	}
	if len(os.Args) == 2 && os.Args[1] == "resolver-proxy" {
		run = resolverProxy
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "lab boot recovery configuration failed")
		os.Exit(1)
	}
}

func prepare() error {
	raw, err := os.ReadFile("/var/lib/vpn-node/identity/node.json")
	if err != nil {
		return err
	}
	var identity struct {
		Node        uuid.UUID `json:"node_id"`
		Incarnation uuid.UUID `json:"node_incarnation_id"`
	}
	if err = json.Unmarshal(raw, &identity); err != nil {
		return err
	}
	boot, err := os.ReadFile("/etc/vpn-node/boot-id")
	if err != nil {
		return err
	}
	bootID, err := uuid.Parse(strings.TrimSpace(string(boot)))
	if err != nil {
		return err
	}
	cfg := updater.Config{
		NodeID: identity.Node, IncarnationID: identity.Incarnation, BootID: bootID,
		Platform:     updater.Platform{OSID: "ubuntu", OSVersion: "24.04", Architecture: runtime.GOARCH, UpdaterVersion: "1.0.0", ControlAPIMajor: 1, PolicySchema: 1, UsageSchema: 1, StateSchema: 1},
		ReleaseTrust: "/etc/vpn-node/trust.json", DesiredTrust: "/etc/vpn-node/trust.json", DesiredState: "/var/lib/vpn-node/state/desired-state.json",
		MeterCheckpoint: "/var/lib/vpn-node/e2e/meter.json",
		StopCommand:     []string{"/usr/local/bin/agent-service.sh", "stop"}, StartCommand: []string{"/usr/local/bin/agent-service.sh", "start"},
		ResultURL: "https://agent.example.invalid:9443/api/v1/agent/update-results", AllowedArtifactOrigins: []string{"https://agent.example.invalid:9443"},
		MTLSCert: "/var/lib/vpn-node/identity/cert.pem", MTLSKey: "/var/lib/vpn-node/identity/key.pem", MTLSCA: "/etc/vpn-node/ca.pem",
	}
	raw, err = json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := "/etc/vpn-node/updater-recover.json"
	if err = os.WriteFile(path+".tmp", raw, 0600); err != nil {
		return err
	}
	if err = os.Rename(path+".tmp", path); err != nil {
		return err
	}
	_, err = updater.LoadConfig(path)
	return err
}

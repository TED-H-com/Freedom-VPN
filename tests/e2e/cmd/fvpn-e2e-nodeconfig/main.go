// Command fvpn-e2e-nodeconfig writes TEST ONLY boot recovery configuration
// after real enrollment. Identity keys remain inside the node.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

func main() {
	run := prepare
	if len(os.Args) == 2 && os.Args[1] == "upgrade-observe" {
		run = upgradeObserve
	}
	if len(os.Args) == 2 && os.Args[1] == "usage-copy" {
		run = usageCopy
	}
	if len(os.Args) == 2 && os.Args[1] == "journal-copy" {
		run = journalCopy
	}
	if len(os.Args) == 2 && os.Args[1] == "dns-relay" {
		run = nodeDNSRelay
	}
	if len(os.Args) == 2 && os.Args[1] == "installer-proxy" {
		run = installerProxy
	}
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
	if len(os.Args) == 2 && os.Args[1] == "telemetry" {
		run = telemetry
	}
	if len(os.Args) == 2 && os.Args[1] == "http-burst" {
		run = httpBurst
	}
	if len(os.Args) == 2 && os.Args[1] == "enrollment-generate" {
		run = enrollmentGenerate
	}
	if len(os.Args) == 2 && os.Args[1] == "enrollment-join" {
		run = enrollmentJoin
	}
	if len(os.Args) == 2 && os.Args[1] == "identity-fingerprints" {
		run = identityFingerprints
	}
	if len(os.Args) == 2 && os.Args[1] == "probe-issued" {
		run = probeIssued
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
		AgentConfig:     "/etc/vpn-node/agent-run.json",
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

// The origin-network relay must reach AdGuard through loopback because the
// signed resolver document permits loopback/overlay clients, not origin IPs.
func nodeDNSRelay() error {
	raw, err := os.ReadFile("/etc/vpn-node/agent-run.json")
	if err != nil {
		return err
	}
	var cfg struct {
		Datapath struct {
			SNAT string `json:"snat"`
		} `json:"datapath"`
	}
	if err = json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	ip := net.ParseIP(cfg.Datapath.SNAT).To4()
	if ip == nil {
		return fmt.Errorf("DNS relay address")
	}
	sourceOctet := byte(53)
	if ip[3] == 22 {
		sourceOctet = 54
	}
	ip[3] = sourceOctet
	return dnsrelay.Serve(cfg.Datapath.SNAT+":5353", "127.0.0.1:53", ip.String())
}

//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StartSecondExit creates one more real node and its independent resolver,
// identity/state directories and durable network namespace on the existing lab
// bridges. Call after PrepareServingBaseline, then EnrollAt(..., "node2").
// It does not publish the exit or infer serviceability. e2e-down removes it.
func (s *Stack) StartSecondExit(ctx context.Context) error {
	base := filepath.Join(s.lab.state, "node2")
	if err := os.Mkdir(base, 0700); err != nil {
		return fmt.Errorf("second node requires a fresh slot: %w", err)
	}
	for _, dir := range []string{"etc", "var/e2e", "adguard/conf", "adguard/work"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0700); err != nil {
			return err
		}
	}
	run := func(args ...string) error {
		if err := exec.CommandContext(ctx, "docker", args...).Run(); err != nil {
			return fmt.Errorf("second node Docker step failed: %w", err)
		}
		return nil
	}
	if err := run("cp", s.lab.node+":/etc/vpn-node/.", filepath.Join(base, "etc")); err != nil {
		return err
	}
	// Clone configuration and public trust, never the node's durable identity.
	// The entrypoint generates a fresh boot id/recovery config after enrollment.
	for _, file := range []string{"boot-id", "updater-recover.json"} {
		_ = os.Remove(filepath.Join(base, "etc", file))
	}
	prefix := s.lab.originIP[:strings.LastIndex(s.lab.originIP, ".")]
	for _, file := range []string{"agent-enroll.json", "agent-run.json"} {
		path := filepath.Join(base, "etc", file)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var cfg map[string]any
		if err = json.Unmarshal(raw, &cfg); err != nil {
			return err
		}
		dp, ok := cfg["datapath"].(map[string]any)
		if !ok {
			return fmt.Errorf("datapath missing")
		}
		dp["snat"] = prefix + ".22"
		raw, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		if err = os.WriteFile(path, raw, 0600); err != nil {
			return err
		}
	}
	if err := run("cp", s.lab.prefix+"-adguard:/opt/adguardhome/conf/AdGuardHome.yaml", filepath.Join(base, "adguard/conf/AdGuardHome.yaml")); err != nil {
		return err
	}
	image, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Image}}", s.lab.node).Output()
	if err != nil {
		return err
	}
	dnsImage, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Image}}", s.lab.prefix+"-adguard").Output()
	if err != nil {
		return err
	}
	name := s.lab.prefix + "-node2"
	ns := name + "-netns"
	if err = run("run", "-d", "--name", ns, "--hostname", name, "--network", s.lab.prefix, "--entrypoint", "sleep", strings.TrimSpace(string(image)), "infinity"); err != nil {
		return err
	}
	if err = run("network", "connect", "--ip", prefix+".22", s.lab.prefix+"-origin", ns); err != nil {
		return err
	}
	if err = run("run", "-d", "--name", name+"-adguard", "--network", "container:"+ns, "--mount", "type=bind,src="+filepath.Join(base, "adguard/conf")+",dst=/opt/adguardhome/conf", "--mount", "type=bind,src="+filepath.Join(base, "adguard/work")+",dst=/opt/adguardhome/work", strings.TrimSpace(string(dnsImage)), "--no-check-update", "--no-permcheck", "-c", "/opt/adguardhome/conf/AdGuardHome.yaml", "-w", "/opt/adguardhome/work"); err != nil {
		return err
	}
	return run("run", "-d", "--name", name, "--privileged", "--network", "container:"+ns, "--mount", "type=bind,src="+filepath.Join(base, "etc")+",dst=/etc/vpn-node", "--mount", "type=bind,src="+filepath.Join(base, "var")+",dst=/var/lib/vpn-node", strings.TrimSpace(string(image)))
}

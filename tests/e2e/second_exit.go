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
	"time"
)

// StartSecondExit creates one more real node and its independent resolver,
// identity/state directories and durable network namespace on the existing lab
// bridges. Call after PrepareServingBaseline, then EnrollAt(..., "node2").
// It does not publish the exit or infer serviceability. e2e-down removes it.
func (s *Stack) StartSecondExit(ctx context.Context) error { return s.startSecondExit(ctx, false) }
func (s *Stack) startSecondExit(ctx context.Context, clone bool) error {
	slot := "node2"
	lastOctet := ".22"
	if clone {
		slot = "node-clone"
		lastOctet = ".23"
	}
	base := filepath.Join(s.lab.state, slot)
	if err := os.Mkdir(base, 0700); err != nil {
		return fmt.Errorf("second node requires a fresh slot: %w", err)
	}
	for _, dir := range []string{"etc", "var/e2e", "adguard/conf", "adguard/work", "netbird", "opt"} {
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
	if clone {
		if err := run("stop", "-t", "20", s.lab.node); err != nil {
			return err
		}
		defer func() {
			resume, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()
			_ = exec.CommandContext(resume, "docker", "start", s.lab.node).Run()
		}()
	}
	if err := run("cp", s.lab.node+":/etc/vpn-node/.", filepath.Join(base, "etc")); err != nil {
		return err
	}
	// Clean second exits copy configuration/public trust; the explicit fault
	// fixture below also snapshots durable identity while the source is stopped.
	// The entrypoint generates a fresh boot id/recovery config after enrollment.
	for _, file := range []string{"boot-id", "updater-recover.json"} {
		_ = os.Remove(filepath.Join(base, "etc", file))
	}
	if !clone {
		_ = os.Remove(filepath.Join(base, "etc", "machine-id"))
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
		dp["snat"] = prefix + lastOctet
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
	name := s.lab.prefix + "-" + slot
	ns := name + "-netns"
	if err = run("run", "-d", "--name", ns, "--hostname", name, "--network", s.lab.prefix, "--entrypoint", "sleep", strings.TrimSpace(string(image)), "infinity"); err != nil {
		return err
	}
	if err = run("network", "connect", "--ip", prefix+lastOctet, s.lab.prefix+"-origin", ns); err != nil {
		return err
	}
	if err = run("run", "-d", "--name", name+"-adguard", "--network", "container:"+ns, "--mount", "type=bind,src="+filepath.Join(base, "adguard/conf")+",dst=/opt/adguardhome/conf", "--mount", "type=bind,src="+filepath.Join(base, "adguard/work")+",dst=/opt/adguardhome/work", strings.TrimSpace(string(dnsImage)), "--no-check-update", "--no-permcheck", "-c", "/opt/adguardhome/conf/AdGuardHome.yaml", "-w", "/opt/adguardhome/work"); err != nil {
		return err
	}
	if clone {
		// Source is stopped and target has not booted. Pipe archives directly into
		// the private target mounts; never print/store an archive in evidence.
		// The client stores its X25519 identity in /var/lib/netbird, not /etc/netbird.
		for _, copy := range []struct{ source, target string }{{"/var/lib/vpn-node/.", "var"}, {"/var/lib/netbird/.", "netbird"}, {"/opt/vpn-node/.", "opt"}} {
			read := exec.CommandContext(ctx, "docker", "cp", s.lab.node+":"+copy.source, "-")
			write := exec.CommandContext(ctx, "docker", "run", "--rm", "-i", "--name", name+"-copy", "--network", "none", "--mount", "type=bind,src="+filepath.Join(base, copy.target)+",dst=/copy", "--entrypoint", "tar", strings.TrimSpace(string(image)), "-xf", "-", "-C", "/copy")
			pipe, err := read.StdoutPipe()
			if err != nil {
				return err
			}
			write.Stdin = pipe
			if err = read.Start(); err != nil {
				return err
			}
			we := write.Run()
			_ = pipe.Close()
			re := read.Wait()
			if we != nil || re != nil {
				return fmt.Errorf("private clone copy failed")
			}
		}
		// Runtime trigger/PID files are not cloned actions; identity and ledgers are.
		for _, file := range []string{"updater.json", "pause-agent", "agent.pid"} {
			_ = os.Remove(filepath.Join(base, "var/e2e", file))
		}
	}
	// A clean second exit keeps an empty directory off the client's state path.
	// The clone bind-mounts the copied profile where the client reads it.
	// --network container: shares interfaces, not /etc/hosts, and Docker
	// rejects --add-host in that mode. The entrypoint appends lab-hosts.
	// The copied file already names the lab /24. The clone rewrites
	// agent.example.invalid to the presentation control address first.
	netbirdDest := "/etc/netbird"
	args := []string{"run", "-d", "--name", name, "--privileged", "--network", "container:" + ns}
	if clone {
		netbirdDest = "/var/lib/netbird"
		pres, attachErr := s.AttachClonePresentationNetwork(ctx, ns)
		if attachErr != nil {
			return attachErr
		}
		if err := rewriteLabHost(filepath.Join(base, "etc", "lab-hosts"), "agent.example.invalid", pres.Control); err != nil {
			return err
		}
	}
	args = append(args,
		"--mount", "type=bind,src="+filepath.Join(base, "etc")+",dst=/etc/vpn-node",
		"--mount", "type=bind,src="+filepath.Join(base, "var")+",dst=/var/lib/vpn-node",
		"--mount", "type=bind,src="+filepath.Join(base, "netbird")+",dst="+netbirdDest,
		"--mount", "type=bind,src="+filepath.Join(base, "opt")+",dst=/opt/vpn-node",
		strings.TrimSpace(string(image)),
	)
	return run(args...)
}

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

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// BuildInstallerRelease shares the lab's release signer and builds real Linux
// amd64/arm64 binaries at the installer layout, with unchanged packaged units.
// RegisterRelease performs public registration and worker verification readback.
func (s *Stack) BuildInstallerRelease(ctx context.Context, dir, version, sequence, build string, options ReleaseOptions) (BuiltRelease, error) {
	options.InstallerLayout = true
	b, err := s.BuildRelease(ctx, dir, version, sequence, build, options)
	if err == nil {
		err = os.WriteFile(filepath.Join(dir, "manifest.json"), b.Manifest, 0600)
	}
	return b, err
}

// RegisterInstallerReleases registers PrepareInstallerNode's older/newer real
// bundles. Activate a release_support policy including their IDs separately;
// registration alone is not permission to serve or a healthy update outcome.
func (s *Stack) RegisterInstallerReleases(ctx context.Context, admin BrowserSession) ([]BuiltRelease, error) {
	var result []BuiltRelease
	for _, which := range []string{"older", "newer"} {
		dir := filepath.Join(s.lab.root, ".agent", "tmp", "e2e-installer-rel", which)
		raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
		if err != nil {
			return result, err
		}
		var envelope struct{ Signed api.ReleaseManifest }
		if err = json.Unmarshal(raw, &envelope); err != nil {
			return result, err
		}
		hash, err := canonjson.HashHex(raw)
		if err != nil {
			return result, err
		}
		b := BuiltRelease{ID: envelope.Signed.ReleaseId, Directory: dir, Manifest: raw, ManifestSHA256: hash}
		if err = s.RegisterRelease(ctx, admin, b); err != nil {
			return result, err
		}
		result = append(result, b)
	}
	return result, nil
}

// ConfigureInstallerServing supplies the real AdGuard backend missing from the
// installer's minimal configuration. The generated agent.json is preserved;
// a TEST ONLY runtime drop-in selects agent-lab.json, copied from it with DNS
// and short observation intervals. Call again after reenrollment/release change.
// This does not publish an exit, grant a member, or write observations.
func (s *Stack) ConfigureInstallerServing(ctx context.Context) error {
	c := s.installerContainer()
	base := filepath.Join(s.lab.state, "installer-node", "adguard")
	for _, dir := range []string{"conf", "work"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0700); err != nil {
			return err
		}
	}
	name := c + "-adguard"
	if exec.CommandContext(ctx, "docker", "inspect", name).Run() != nil {
		if err := dockerCopy(ctx, s.lab.prefix+"-adguard:/opt/adguardhome/conf/AdGuardHome.yaml", filepath.Join(base, "conf", "AdGuardHome.yaml")); err != nil {
			return err
		}
		img, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Image}}", s.lab.prefix+"-adguard").Output()
		if err != nil {
			return err
		}
		if err = exec.CommandContext(ctx, "docker", "run", "-d", "--name", name, "--network", "container:"+c,
			"--mount", "type=bind,src="+filepath.Join(base, "conf")+",dst=/opt/adguardhome/conf",
			"--mount", "type=bind,src="+filepath.Join(base, "work")+",dst=/opt/adguardhome/work",
			strings.TrimSpace(string(img)), "--no-check-update", "--no-permcheck", "-c", "/opt/adguardhome/conf/AdGuardHome.yaml", "-w", "/opt/adguardhome/work").Run(); err != nil {
			return err
		}
	}
	if err := dockerCopy(ctx, filepath.Join(s.lab.state, "node", "etc", "resolver-credentials.json"), c+":/etc/vpn-node/resolver-credentials.json"); err != nil {
		return err
	}
	if err := exec.CommandContext(ctx, "docker", "exec", c, "sh", "-c", "chown 0:0 /etc/vpn-node/resolver-credentials.json && chmod 600 /etc/vpn-node/resolver-credentials.json").Run(); err != nil {
		return err
	}
	raw, err := exec.CommandContext(ctx, "docker", "exec", c, "cat", "/etc/vpn-node/agent.json").Output()
	if err != nil {
		return err
	}
	var cfg map[string]any
	if err = json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	cfg["dns"] = map[string]any{"resolvers": []any{map[string]string{"id": "lab", "admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}}, "resolver_credentials_file": "/etc/vpn-node/resolver-credentials.json"}
	cfg["test_mode"] = true
	cfg["heartbeat_seconds"], cfg["poll_seconds"], cfg["observation_seconds"] = 5, 5, 5
	raw, err = json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err = writeContainerFile(ctx, c, "/etc/vpn-node/agent-lab.json", "644", raw); err != nil {
		return err
	}
	for _, unit := range []string{"agent", "watchdog"} {
		body := fmt.Sprintf("[Service]\nExecStart=\nExecStart=/opt/vpn-node/current/fvpn-%s run -config /etc/vpn-node/agent-lab.json\n", unit)
		if err = writeContainerFile(ctx, c, "/run/systemd/system/fvpn-"+unit+".service.d/e2e-serving.conf", "644", []byte(body)); err != nil {
			return err
		}
	}
	return exec.CommandContext(ctx, "docker", "exec", c, "systemctl", "daemon-reload").Run()
}

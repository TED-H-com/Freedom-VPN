//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ClonePresentation is the second IPv4 /24 a cloned node uses when it presents
// the same certificate. It is not an assigned port-slot lab. The legitimate
// node stays on the stack's lab /24. Compatibility domains are /24, so a
// second address on the same lab network cannot conflict (D-072).
type ClonePresentation struct {
	Network string `json:"network"`
	Subnet  string `json:"subnet"`
	Control string `json:"control_address"`
	Clone   string `json:"clone_address"`
}

// AttachClonePresentationNetwork ensures the clone presentation bridge exists
// and attaches control at the control address. When container is non-empty it
// must be the clone network-namespace container: that container is attached at
// the clone address, and agent.example.invalid in that container's /etc/hosts
// is pinned to the presentation control address.
//
// A process container started with --network container:<netns> shares the
// interfaces and not the filesystem. Docker also rejects --add-host in that
// mode. StartClonedIdentityFixture rewrites the clone's lab-hosts so
// agent.example.invalid is the presentation control address; the entrypoint
// appends that file. An empty container only prepares the network and control.
func (s *Stack) AttachClonePresentationNetwork(ctx context.Context, container string) (ClonePresentation, error) {
	pres, err := s.clonePresentation()
	if err != nil {
		return ClonePresentation{}, err
	}
	if err = s.ensureCloneNetwork(ctx, pres); err != nil {
		return pres, err
	}
	if err = s.connectPresentation(ctx, s.lab.control, pres.Network, pres.Control); err != nil {
		return pres, err
	}
	if container == "" {
		return pres, nil
	}
	if err = s.connectPresentation(ctx, container, pres.Network, pres.Clone); err != nil {
		return pres, err
	}
	if err = s.pinPresentationHost(ctx, container, pres); err != nil {
		return pres, err
	}
	return pres, nil
}

func (s *Stack) labPrefix() (string, error) {
	raw, err := os.ReadFile(filepath.Join(s.lab.state, "stack.json"))
	if err != nil {
		return "", err
	}
	var stack struct {
		LabPrefix string `json:"lab_prefix"`
	}
	if json.Unmarshal(raw, &stack) != nil || stack.LabPrefix == "" {
		return "", fmt.Errorf("lab prefix unavailable")
	}
	if len(strings.Split(stack.LabPrefix, ".")) != 3 {
		return "", fmt.Errorf("lab prefix unavailable")
	}
	return stack.LabPrefix, nil
}

// rewriteLabHost sets one hostname in the process container's lab-hosts file.
// Docker rejects --add-host when the network mode is container:<id>. The
// entrypoint appends names from this file that are not already in /etc/hosts.
func rewriteLabHost(path, name, ip string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("lab hosts unavailable: %w", err)
	}
	lines := strings.Split(string(raw), "\n")
	found := false
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == name {
			lines[i] = ip + " " + name
			found = true
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines[len(lines)-1] = ip + " " + name
			lines = append(lines, "")
		} else {
			lines = append(lines, ip+" "+name)
		}
	}
	body := strings.Join(lines, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0o600)
}

func (s *Stack) clonePresentation() (ClonePresentation, error) {
	prefix, err := s.labPrefix()
	if err != nil {
		return ClonePresentation{}, err
	}
	parts := strings.Split(prefix, ".")
	second, err := strconv.Atoi(parts[1])
	if err != nil {
		return ClonePresentation{}, err
	}
	base := fmt.Sprintf("%s.%d.%s", parts[0], second+1, parts[2])
	return ClonePresentation{
		Network: s.lab.prefix + "-node-clone-net",
		Subnet:  base + ".0/24",
		Control: base + ".30",
		Clone:   base + ".23",
	}, nil
}

func (s *Stack) ensureCloneNetwork(ctx context.Context, pres ClonePresentation) error {
	if exec.CommandContext(ctx, "docker", "network", "inspect", pres.Network).Run() == nil {
		return nil
	}
	gateway := strings.TrimSuffix(pres.Subnet, ".0/24") + ".1"
	return exec.CommandContext(ctx, "docker", "network", "create",
		"--driver", "bridge",
		"--subnet", pres.Subnet,
		"--gateway", gateway,
		"-o", "com.docker.network.bridge.enable_ip_masquerade=false",
		pres.Network).Run()
}

func (s *Stack) connectPresentation(ctx context.Context, container, network, address string) error {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", `{{range $name, $net := .NetworkSettings.Networks}}{{$name}} {{$net.IPAddress}}{{"\n"}}{{end}}`, container).Output()
	if err != nil {
		return fmt.Errorf("presentation inspect failed")
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == network {
			if fields[1] != address {
				return fmt.Errorf("presentation address already differs")
			}
			return nil
		}
	}
	if err = exec.CommandContext(ctx, "docker", "network", "connect", "--ip", address, network, container).Run(); err != nil {
		return fmt.Errorf("presentation connect failed")
	}
	return nil
}

func (s *Stack) pinPresentationHost(ctx context.Context, container string, pres ClonePresentation) error {
	line := pres.Control + " agent.example.invalid"
	script := `grep -F -- "$1" /etc/hosts >/dev/null 2>&1 || printf '%s\n' "$1" >> /etc/hosts`
	return exec.CommandContext(ctx, "docker", "exec", container, "sh", "-c", script, "sh", line).Run()
}

//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// InstallerServiceState retains active/inactive/failed/not-found states. The
// updater oneshot normally finishes inactive; restoring it must not apply a job.
type InstallerServiceState map[string]string

var installerConsumerUnits = []string{"fvpn-agent.service", "fvpn-watchdog.service", "fvpn-updater.service", "fvpn-updater.timer"}

func (s *Stack) InstallerServices(ctx context.Context, slot string) (InstallerServiceState, error) {
	if err := s.requireInstallerSlot(slot); err != nil {
		return nil, err
	}
	out := InstallerServiceState{}
	for _, unit := range installerConsumerUnits {
		raw, err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "systemctl", "show", unit, "-p", "ActiveState", "--value").Output()
		if err != nil {
			return nil, fmt.Errorf("service state %s: %w", unit, err)
		}
		out[unit] = strings.TrimSpace(string(raw))
	}
	return out, nil
}

// RestoreInstallerServices restarts only services active before sampling (or
// started by the installer). It never enables units, removes capture barriers,
// or starts an inactive updater oneshot. Use outcome.Services even on failure.
func (s *Stack) RestoreInstallerServices(ctx context.Context, slot string, state InstallerServiceState) error {
	if err := s.requireInstallerSlot(slot); err != nil {
		return err
	}
	if len(state) != len(installerConsumerUnits) {
		return fmt.Errorf("complete installer service snapshot required")
	}
	for _, unit := range installerConsumerUnits {
		v, ok := state[unit]
		if !ok || (v != "active" && v != "inactive" && v != "failed" && v != "activating") {
			return fmt.Errorf("invalid installer service snapshot for %s", unit)
		}
	}
	for _, unit := range installerConsumerUnits {
		if state[unit] != "active" && state[unit] != "activating" {
			continue
		}
		if err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "systemctl", "start", unit).Run(); err != nil {
			return fmt.Errorf("restore %s: %w", unit, err)
		}
		// A oneshot may already have completed; its command exit is the evidence.
		if unit != "fvpn-updater.service" {
			if err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "systemctl", "is-active", "--quiet", unit).Run(); err != nil {
				return fmt.Errorf("restore %s is not active", unit)
			}
		}
	}
	return nil
}

func (s *Stack) retainInstallerServices(ctx context.Context, out *InstallerOutcome, before InstallerServiceState) error {
	after, err := s.InstallerServices(ctx, "installer-node")
	out.Services = before
	if err != nil {
		return err
	}
	for unit, state := range after {
		if state == "active" || state == "activating" {
			out.Services[unit] = state
		}
	}
	return nil
}

// HoldInstallerFirstPoll and HoldInstallerApply are TEST ONLY runtime systemd
// conditions. Arm before installation/staging. Release removes only our drop-in
// and reloads units; it does not start anything. Packaged ExecStart is unchanged.
func (s *Stack) HoldInstallerFirstPoll(ctx context.Context, hold bool) error {
	return s.installerBarrier(ctx, "fvpn-agent.service", hold)
}
func (s *Stack) HoldInstallerApply(ctx context.Context, hold bool) error {
	return s.installerBarrier(ctx, "fvpn-updater.service", hold)
}
func (s *Stack) installerBarrier(ctx context.Context, unit string, hold bool) error {
	path := "/run/systemd/system/" + unit + ".d/e2e-capture.conf"
	marker := "/var/tmp/e2e-rel/hold-" + unit
	if hold {
		if err := writeContainerFile(ctx, s.installerContainer(), marker, "600", nil); err != nil {
			return err
		}
		if err := writeContainerFile(ctx, s.installerContainer(), path, "644", []byte("[Unit]\nConditionPathExists=!"+marker+"\n")); err != nil {
			return err
		}
	} else if err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "rm", "-f", path, marker).Run(); err != nil {
		return err
	}
	return exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "systemctl", "daemon-reload").Run()
}

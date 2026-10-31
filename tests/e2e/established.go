//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread"
	"github.com/google/uuid"
)

// DeviceCounters is one device's admission byte counters. Present is false
// when the named counters are not in the current ruleset.
type DeviceCounters = counterread.Result

// EstablishedTransports is one persistent member TCP session and its
// authenticated UDP echo. Stop closes both and removes the key file.
type EstablishedTransports struct {
	Device uuid.UUID
	TCP    string
	UDP    string
}

const (
	memberSessionBin    = "/var/lib/vpn-node/e2e/fvpn-e2e-member-session"
	memberSessionPID    = "/var/lib/vpn-node/e2e/member-session.pid"
	memberSessionStatus = "/var/lib/vpn-node/e2e/member-session.status"
	memberSessionKey    = "/var/lib/vpn-node/e2e/member-echo.key"
)

// DeviceByteCounters reads u_<device> and d_<device> from inet fvpn.
func (s *Stack) DeviceByteCounters(ctx context.Context, device uuid.UUID) (DeviceCounters, error) {
	out := DeviceCounters{}
	started := time.Now().UTC()
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.node, "/usr/local/bin/fvpn-e2e-nodeconfig", "device-counters", device.String()).Output()
	if err != nil {
		out.Diagnostic.Started, out.Diagnostic.Completed = started, time.Now().UTC()
		out.Diagnostic.Stage, out.Diagnostic.Category = "container_exec", "container_exec"
		if ctx.Err() != nil {
			out.Diagnostic.Category = "timeout_or_cancel"
		}
		return out, fmt.Errorf("counter read: %s", out.Diagnostic.Category)
	}
	if json.Unmarshal(raw, &out) != nil {
		out.Diagnostic.Stage = "helper_decode"
		out.Diagnostic.Category = "parse"
		return out, fmt.Errorf("counter helper parse")
	}
	if out.Diagnostic.Category != "none" {
		return out, fmt.Errorf("counter %s: %s", out.Diagnostic.Stage, out.Diagnostic.Category)
	}
	return out, nil
}

// CountedMemberTraffic is the counter read the member-traffic scenario uses
// after a device has been admitted. It does not open a new session.
func (s *Stack) CountedMemberTraffic(ctx context.Context, device uuid.UUID) (DeviceCounters, error) {
	return s.DeviceByteCounters(ctx, device)
}

// StartEstablishedTransports copies the echo helper into the member and
// keeps TCP :9446 and authenticated UDP :9445 open until Stop. The UDP key
// is a mode-0600 file. It is not an argument and it is not the updater probe.
func (s *Stack) StartEstablishedTransports(ctx context.Context, device uuid.UUID) (EstablishedTransports, error) {
	if device == uuid.Nil {
		return EstablishedTransports{}, fmt.Errorf("device required")
	}
	if err := s.StopEstablishedTransports(ctx); err != nil {
		return EstablishedTransports{}, err
	}
	bin, err := s.memberSessionBinary(ctx)
	if err != nil {
		return EstablishedTransports{}, err
	}
	if err = dockerCopy(ctx, bin, s.lab.member+":"+memberSessionBin); err != nil {
		return EstablishedTransports{}, err
	}
	if out, err := exec.CommandContext(ctx, "docker", "exec", s.lab.member, "chmod", "0755", memberSessionBin).CombinedOutput(); err != nil {
		return EstablishedTransports{}, fmt.Errorf("member session mode: %s", clipText(string(out), 200))
	}
	secret, err := s.lab.secret("origin.secret")
	if err != nil {
		return EstablishedTransports{}, err
	}
	if err = writeContainerFile(ctx, s.lab.member, memberSessionKey, "600", []byte(secret+"\n")); err != nil {
		return EstablishedTransports{}, err
	}
	secret = ""
	out := EstablishedTransports{Device: device, TCP: s.lab.originIP + ":9446", UDP: s.lab.originIP + ":9445"}
	cmd := exec.CommandContext(ctx, "docker", "exec", "-d", s.lab.member, memberSessionBin,
		"-tcp", out.TCP, "-udp", out.UDP, "-device", device.String(),
		"-key-file", memberSessionKey, "-pid-file", memberSessionPID, "-status-file", memberSessionStatus)
	if raw, err := cmd.CombinedOutput(); err != nil {
		_ = s.StopEstablishedTransports(ctx)
		return EstablishedTransports{}, fmt.Errorf("member session start: %s", clipText(string(raw), 200))
	}
	wait, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	alive := waitUntil(wait, 200*time.Millisecond, func() bool {
		status, err := s.EstablishedStatus(wait)
		return err == nil && status.Running && status.TCPEchoes > 0 && status.UDPReplies > 0
	})
	if !alive {
		status, _ := exec.CommandContext(ctx, "docker", "exec", s.lab.member, "cat", memberSessionStatus).CombinedOutput()
		_ = s.StopEstablishedTransports(ctx)
		return EstablishedTransports{}, fmt.Errorf("member session exited: %s", clipText(string(status), 200))
	}
	return out, nil
}

// StopEstablishedTransports ends the helper and deletes the key file.
func (s *Stack) StopEstablishedTransports(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "exec", s.lab.member, "sh", "-c", `
if [ -s "$1" ]; then
  pid=$(cat "$1")
  case "$pid" in ''|*[!0-9]*) exit 1;; esac
  if [ -r "/proc/$pid/cmdline" ] && tr '\0' ' ' < "/proc/$pid/cmdline" | grep -q fvpn-e2e-member-session; then
    kill "$pid"
    i=0
    while kill -0 "$pid" 2>/dev/null && [ "$i" -lt 50 ]; do
      test "$(awk '{print $3}' /proc/$pid/stat 2>/dev/null)" = Z && break
      sleep 0.1; i=$((i+1))
    done
    test "$i" -lt 50 || exit 1
  fi
fi
rm -f "$1" "$2" "$2.tmp" "$3"
`, "sh", memberSessionPID, memberSessionStatus, memberSessionKey)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stop member session: %w", err)
	}
	return nil
}

// MemberSessionStatus is the helper's tcp_echoes and udp_replies counts.
func (s *Stack) MemberSessionStatus(ctx context.Context) (tcpEchoes, udpReplies int, problem string, err error) {
	status, err := s.EstablishedStatus(ctx)
	return status.TCPEchoes, status.UDPReplies, status.Error, err
}

// MemberTransportStatus distinguishes an active sender from successful replies.
type MemberTransportStatus struct {
	ReadDiagnostic struct {
		Started   time.Time `json:"started"`
		Completed time.Time `json:"completed"`
		Stage     string    `json:"stage"`
		Category  string    `json:"category"`
	} `json:"read_diagnostic"`
	TCPEchoes   int    `json:"tcp_echoes"`
	UDPReplies  int    `json:"udp_replies"`
	TCPAttempts int    `json:"tcp_attempts"`
	UDPAttempts int    `json:"udp_attempts"`
	Running     bool   `json:"running"`
	Error       string `json:"error"`
}

func (s *Stack) EstablishedStatus(ctx context.Context) (MemberTransportStatus, error) {
	var status MemberTransportStatus
	started := time.Now().UTC()
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.member, "cat", memberSessionStatus).Output()
	if err == nil {
		err = json.Unmarshal(raw, &status)
		if err != nil {
			status.ReadDiagnostic.Category = "parse"
		}
	} else {
		status.ReadDiagnostic.Category = "container_read"
		if ctx.Err() != nil {
			status.ReadDiagnostic.Category = "timeout_or_cancel"
		}
	}
	status.ReadDiagnostic.Started = started
	status.ReadDiagnostic.Completed = time.Now().UTC()
	status.ReadDiagnostic.Stage = "member_status_read"
	if err == nil {
		status.ReadDiagnostic.Category = "none"
	}
	return status, err
}

func (s *Stack) memberSessionBinary(ctx context.Context) (string, error) {
	dest := filepath.Join(s.lab.root, ".agent", "tmp", "fvpn-e2e-member-session")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", dest, "./tests/e2e/cmd/fvpn-e2e-member-session")
	cmd.Dir = s.lab.root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("member session build: %s", clipText(string(out), 400))
	}
	return dest, nil
}

func dockerCopy(ctx context.Context, src, dest string) error {
	out, err := exec.CommandContext(ctx, "docker", "cp", src, dest).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker cp: %s", clipText(string(out), 200))
	}
	return nil
}

// MemberTransportStack selects an owned native member and exit for the same
// transport/counter methods, including a caller's independently admitted cohort.
func (s *Stack) MemberTransportStack(member, nodeSlot string) (*Stack, error) {
	name, err := s.memberContainer(member)
	if err != nil {
		return nil, err
	}
	if nodeSlot == "" {
		nodeSlot = "node"
	}
	node, _, err := s.usageSlot(nodeSlot)
	if err != nil {
		return nil, err
	}
	copyLab := *s.lab
	copyLab.member = name
	copyLab.node = node
	return &Stack{lab: &copyLab}, nil
}

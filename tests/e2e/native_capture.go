//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture"
	"github.com/google/uuid"
)

type NativePathSample struct {
	Label  string                            `json:"label"`
	Points map[string]nativecapture.Snapshot `json:"points"`
}
type NativeCapturePoint struct {
	Start          nativecapture.Clock `json:"launch_before"`
	Ready          nativecapture.Clock `json:"ready_after"`
	Stop           nativecapture.Clock `json:"stop_before"`
	Stopped        nativecapture.Clock `json:"stopped_after"`
	ProcessStopped bool                `json:"process_stopped"`
	Captured       int                 `json:"captured"`
	Received       int                 `json:"received_by_filter"`
	Dropped        int                 `json:"kernel_drops"`
	StatisticsOK   bool                `json:"statistics_ok"`
	Receipt        ContainerExport     `json:"private_pcap_receipt"`
}
type NativeCaptureEvidence struct {
	ID      string                         `json:"id"`
	Points  map[string]*NativeCapturePoint `json:"points"`
	Samples []NativePathSample             `json:"samples"`
}

// NativeCapture owns three namespace-wide captures. Each SLL2 record identifies
// its interface and direction, retaining both tunnel and selected underlay hops.
// Use serial Sample/Stop calls; traffic and readbacks may run concurrently.
type NativeCapture struct {
	stack      *Stack
	targets    map[string]string
	containers map[string]string
	directory  string
	evidence   NativeCaptureEvidence
}

const nativeObserve = "/var/lib/vpn-node/e2e/native-observe"

func (s *Stack) nativeAddresses(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	for role, c := range map[string]string{"member": s.lab.member, "node": s.lab.node} {
		raw, e := exec.CommandContext(ctx, "docker", "exec", c, "netbird", "status", "--json").Output()
		var status nativecapture.Status
		if e != nil || json.Unmarshal(raw, &status) != nil || status.IP == "" {
			return nil, fmt.Errorf("%s native address unavailable", role)
		}
		out[role] = strings.Split(status.IP, "/")[0]
	}
	return out, nil
}

// StartNativeCapture must precede route selection. It does not select, reconnect,
// repair or generate traffic. Raw pcaps remain under the caller's ignored disk
// directory; use decode_capture.py before publishing. On error, started captures
// are stopped and exported too. It requires already joined member/node peers.
func (s *Stack) StartNativeCapture(ctx context.Context) (_ *NativeCapture, err error) {
	return s.startNativeCapture(ctx, nil)
}

func (s *Stack) startNativeCapture(ctx context.Context, extra map[string]string) (_ *NativeCapture, err error) {
	addresses, err := s.nativeAddresses(ctx)
	if err != nil {
		return nil, err
	}
	c := &NativeCapture{stack: s, targets: map[string]string{"member": addresses["node"], "node": addresses["member"]}, containers: map[string]string{"member": s.lab.member, "node": s.lab.node, "origin": s.lab.prefix + "-origin"}, evidence: NativeCaptureEvidence{ID: uuid.NewString(), Points: map[string]*NativeCapturePoint{}}}
	for role, container := range extra {
		c.containers[role] = container
	}
	c.directory = "/var/lib/vpn-node/e2e/native-" + c.evidence.ID
	build := filepath.Join(s.lab.root, ".agent/tmp/native-observe")
	if err = os.MkdirAll(filepath.Dir(build), 0700); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", build, "./tests/e2e/cmd/fvpn-e2e-native-observe")
	cmd.Dir = s.lab.root
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("build native observer: %w", err)
	}
	defer func() {
		if err != nil {
			_, _ = c.Stop(context.WithoutCancel(ctx), filepath.Join(s.lab.root, ".agent/tmp/native-incomplete", c.evidence.ID))
		}
	}()
	for _, role := range c.roles() {
		container := c.containers[role]
		if err = exec.CommandContext(ctx, "docker", "exec", container, "mkdir", "-p", c.directory).Run(); err != nil {
			return nil, err
		}
		if err = exec.CommandContext(ctx, "docker", "cp", build, container+":"+nativeObserve).Run(); err != nil {
			return nil, err
		}
		p := &NativeCapturePoint{Captured: -1, Received: -1, Dropped: -1}
		c.evidence.Points[role] = p
		p.Start, err = c.clock(ctx, role)
		if err != nil {
			return nil, err
		}
		// Fixed filter excludes application contents from public output, but raw
		// pcaps are private. Full snaplen permits opaque outer-datagram hash joins.
		script := `umask 077; echo $$ > "$1/pid"; exec tcpdump --immediate-mode -U -B 32768 -nn -s 65535 -i any -w "$1/raw.pcap" 'icmp or icmp6 or udp or tcp port 443 or tcp port 9446 or tcp port 53' 2>"$1/stats"`
		err = exec.CommandContext(ctx, "docker", "exec", "-d", container, "sh", "-c", script, "capture", c.directory).Run()
		if err != nil {
			return nil, err
		}
		wait, cancel := context.WithTimeout(ctx, 5*time.Second)
		ready := waitUntil(wait, 50*time.Millisecond, func() bool {
			b, e := exec.CommandContext(wait, "docker", "exec", container, "cat", c.directory+"/stats").Output()
			return e == nil && strings.Contains(string(b), "listening on any")
		})
		cancel()
		if !ready {
			return nil, fmt.Errorf("%s capture not ready", role)
		}
		p.Ready, err = c.clock(ctx, role)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}
func (c *NativeCapture) clock(ctx context.Context, role string) (out nativecapture.Clock, err error) {
	b, err := exec.CommandContext(ctx, "docker", "exec", c.containers[role], nativeObserve, "-clock").Output()
	if err == nil {
		err = json.Unmarshal(b, &out)
	}
	return
}

// Sample retains independent command/clock bounds at every hop. No simultaneous
// sampling or exact kernel route-install instant is claimed.
func (c *NativeCapture) Sample(ctx context.Context, label string) error {
	sample := NativePathSample{Label: label, Points: map[string]nativecapture.Snapshot{}}
	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error
	for role, container := range c.containers {
		wg.Go(func() {
			raw, e := exec.CommandContext(ctx, "docker", "exec", container, nativeObserve, "-peer", c.targets[role], "-origin", c.stack.lab.originIP).Output()
			var out nativecapture.Snapshot
			if e == nil {
				e = json.Unmarshal(raw, &out)
			}
			mu.Lock()
			defer mu.Unlock()
			if e != nil {
				errs = append(errs, fmt.Errorf("%s path snapshot unavailable", role))
			}
			sample.Points[role] = out
		})
	}
	wg.Wait()
	c.evidence.Samples = append(c.evidence.Samples, sample)
	return errors.Join(errs...)
}

// Stop flushes all captures before exporting exact bytes. Directory must be an
// ignored private disk path. Statistics are explicit; unknown is never zero.
func (c *NativeCapture) Stop(ctx context.Context, directory string) (NativeCaptureEvidence, error) {
	var errs []error
	for _, role := range c.roles() {
		p := c.evidence.Points[role]
		if p == nil {
			continue
		}
		container := c.containers[role]
		p.Stop, _ = c.clock(ctx, role)
		script := `read pid < "$1/pid"; kill -INT "$pid"`
		if e := exec.CommandContext(ctx, "docker", "exec", container, "sh", "-c", script, "capture", c.directory).Run(); e != nil {
			errs = append(errs, fmt.Errorf("%s capture stop: %w", role, e))
		}
		wait, cancel := context.WithTimeout(ctx, 5*time.Second)
		p.ProcessStopped = waitUntil(wait, 50*time.Millisecond, func() bool {
			return exec.CommandContext(wait, "docker", "exec", container, "sh", "-c", `read pid < "$1/pid"; kill -0 "$pid"`, "capture", c.directory).Run() != nil
		})
		cancel()
		p.Stopped, _ = c.clock(ctx, role)
		raw, e := exec.CommandContext(ctx, "docker", "exec", container, "cat", c.directory+"/stats").Output()
		p.Captured, p.Received, p.Dropped, p.StatisticsOK = parseNativeCaptureStats(string(raw))
		if e != nil || !p.ProcessStopped || !p.StatisticsOK || p.Dropped != 0 {
			errs = append(errs, fmt.Errorf("%s capture incomplete or dropped packets", role))
		}
		cmd := exec.CommandContext(ctx, "docker", "exec", container, "cat", c.directory+"/raw.pcap")
		p.Receipt, e = exportCommand(cmd, filepath.Join(directory, role+".pcap"))
		if e != nil {
			errs = append(errs, e)
		}
	}
	return c.evidence, errors.Join(errs...)
}
func parseNativeCaptureStats(raw string) (int, int, int, bool) {
	counts := []int{-1, -1, -1}
	for i, label := range []string{"packets captured", "packets received by filter", "packets dropped by kernel"} {
		match := regexp.MustCompile(`(?m)^(\d+) ` + strings.Replace(label, "packets", "packets?", 1) + `$`).FindStringSubmatch(raw)
		if len(match) != 2 {
			return counts[0], counts[1], counts[2], false
		}
		counts[i], _ = strconv.Atoi(match[1])
	}
	return counts[0], counts[1], counts[2], true
}

func (c *NativeCapture) roles() []string {
	roles := []string{}
	for role := range c.containers {
		roles = append(roles, role)
	}
	slices.Sort(roles)
	return roles
}

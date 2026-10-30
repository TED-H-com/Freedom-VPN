//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IngressCounter is one absolute nft named-counter reading from
// `nft -j list table inet fvpn`. The inject hook only reads that table.
type IngressCounter struct {
	Packets uint64 `json:"packets"`
	Bytes   uint64 `json:"bytes"`
}

// MemberIngressResult is the typed result of one `fvpn-netlab inject` inside
// the node network namespace. Mechanism is empty on the command line; fvmb0
// (link type 65534) selects tun-mirred. Before and After are absolute counters.
// Deltas lists only counters whose packet or byte reading changed, and only
// when the after reading did not go backwards.
type MemberIngressResult struct {
	Mechanism   string                    `json:"mechanism"`
	LinkType    int                       `json:"link_type"`
	MemberIface string                    `json:"member_iface"`
	Source      string                    `json:"source"`
	Dest        string                    `json:"dest"`
	SourcePort  uint16                    `json:"source_port"`
	DestPort    uint16                    `json:"dest_port"`
	Before      map[string]IngressCounter `json:"before"`
	After       map[string]IngressCounter `json:"after"`
	Deltas      map[string]IngressCounter `json:"deltas,omitempty"`
}

// MemberIngressProof is one injected UDP datagram plus the uplink and origin
// tcpdump text taken around it. The origin capture runs in the origin
// container's network namespace. Neither capture writes product state.
type MemberIngressProof struct {
	MemberIngressResult
	UplinkCapture string `json:"uplink_capture"`
	OriginCapture string `json:"origin_capture"`
}

var (
	netlabOnce sync.Once
	netlabBin  string
	netlabErr  error
)

// InjectMemberIngress copies the lab `fvpn-netlab` binary into this stack's
// node container and runs `inject` there. -mechanism stays empty. The hook
// does not write table inet fvpn, call wg, or change the member address,
// qdisc, or routes. Callers compare Deltas: a non-admitted source moves
// ctunku by one packet, and an admitted source moves that device's u_<hex>
// counter (device UUID with dashes removed).
func (s *Stack) InjectMemberIngress(ctx context.Context, iface, source, dest string, destPort uint16) (MemberIngressResult, error) {
	var zero MemberIngressResult
	if iface == "" || source == "" || dest == "" {
		return zero, fmt.Errorf("member ingress requires iface, source and dest")
	}
	if destPort == 0 {
		destPort = 9
	}
	bin, err := s.netlabBinary(ctx)
	if err != nil {
		return zero, err
	}
	cp := exec.CommandContext(ctx, "docker", "cp", bin, s.lab.node+":/usr/local/bin/fvpn-netlab")
	if out, err := cp.CombinedOutput(); err != nil {
		return zero, fmt.Errorf("copy fvpn-netlab: %w: %s", err, strings.TrimSpace(string(out)))
	}
	chmod := exec.CommandContext(ctx, "docker", "exec", s.lab.node, "chmod", "755", "/usr/local/bin/fvpn-netlab")
	if out, err := chmod.CombinedOutput(); err != nil {
		return zero, fmt.Errorf("chmod fvpn-netlab: %w: %s", err, strings.TrimSpace(string(out)))
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", s.lab.node,
		"fvpn-netlab", "inject",
		"-iface", iface,
		"-src", source,
		"-dst", dest,
		"-port", strconv.FormatUint(uint64(destPort), 10),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return zero, fmt.Errorf("inject: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var result MemberIngressResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return zero, fmt.Errorf("inject json: %w", err)
	}
	result.Deltas = ingressDeltas(result.Before, result.After)
	return result, nil
}

// ObserveMemberIngress injects one UDP/9 datagram from source to dest on
// fvmb0 and returns the counter result with the fvup0 and origin captures.
// Dest is the origin IPv4 beyond the node. The command leaves -mechanism empty.
func (s *Stack) ObserveMemberIngress(ctx context.Context, source, dest string) (MemberIngressProof, error) {
	var proof MemberIngressProof
	capCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	var uplink, origin bytes.Buffer
	up := exec.CommandContext(capCtx, "docker", "exec", s.lab.node, "timeout", "-s", "INT", "18", "tcpdump", "-l", "-nn", "-q", "-s", "96", "-i", "fvup0", "udp", "and", "port", "9")
	up.Stdout = &uplink
	up.Stderr = &uplink
	image, err := exec.CommandContext(capCtx, "docker", "inspect", "--format={{.Image}}", s.lab.node).Output()
	if err != nil {
		return proof, fmt.Errorf("node image: %w", err)
	}
	capName := s.lab.prefix + "-origin-cap"
	_ = exec.CommandContext(capCtx, "docker", "rm", "-f", capName).Run()
	side := exec.CommandContext(capCtx, "docker", "run", "--rm", "--name", capName,
		"--network", "container:"+s.lab.prefix+"-origin",
		"--cap-add=NET_RAW", "--cap-add=NET_ADMIN",
		"--entrypoint", "sh", strings.TrimSpace(string(image)),
		"-c", "exec timeout -s INT 18 tcpdump -l -nn -q -s 96 -i any udp and port 9")
	side.Stdout = &origin
	side.Stderr = &origin
	if err := up.Start(); err != nil {
		return proof, fmt.Errorf("uplink capture: %w", err)
	}
	if err := side.Start(); err != nil {
		_ = up.Wait()
		proof.UplinkCapture = uplink.String()
		return proof, fmt.Errorf("origin capture: %w", err)
	}
	defer func() { _ = exec.Command("docker", "rm", "-f", capName).Run() }()
	listen, stopListen := context.WithTimeout(capCtx, 8*time.Second)
	ready := waitUntil(listen, 200*time.Millisecond, func() bool {
		return strings.Contains(uplink.String(), "listening") && strings.Contains(origin.String(), "listening")
	})
	stopListen()
	if !ready {
		_ = up.Wait()
		_ = side.Wait()
		proof.UplinkCapture = uplink.String()
		proof.OriginCapture = origin.String()
		return proof, fmt.Errorf("capture did not start")
	}
	result, err := s.InjectMemberIngress(capCtx, "fvmb0", source, dest, 9)
	_ = up.Wait()
	_ = side.Wait()
	proof.MemberIngressResult = result
	proof.UplinkCapture = uplink.String()
	proof.OriginCapture = origin.String()
	if err != nil {
		return proof, err
	}
	return proof, nil
}

func (s *Stack) netlabBinary(ctx context.Context) (string, error) {
	root := s.lab.root
	netlabOnce.Do(func() {
		dir := filepath.Join(root, ".agent", "e2e-build")
		tmp := filepath.Join(root, ".agent", "tmp")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			netlabErr = err
			return
		}
		if err := os.MkdirAll(tmp, 0o755); err != nil {
			netlabErr = err
			return
		}
		bin := filepath.Join(dir, "fvpn-netlab")
		cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", bin, "./tests/netlab/cmd/fvpn-netlab")
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=linux",
			"GOARCH=amd64",
			"GOTOOLCHAIN=local",
			"TMPDIR="+tmp,
			"GOTMPDIR="+tmp,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			netlabErr = fmt.Errorf("build fvpn-netlab: %w: %s", err, strings.TrimSpace(string(out)))
			return
		}
		netlabBin = bin
	})
	return netlabBin, netlabErr
}

func ingressDeltas(before, after map[string]IngressCounter) map[string]IngressCounter {
	names := map[string]struct{}{}
	for name := range before {
		names[name] = struct{}{}
	}
	for name := range after {
		names[name] = struct{}{}
	}
	out := map[string]IngressCounter{}
	for name := range names {
		delta, ok := ingressCounterDelta(before, after, name)
		if !ok || (delta.Packets == 0 && delta.Bytes == 0) {
			continue
		}
		out[name] = delta
	}
	return out
}

func ingressCounterDelta(before, after map[string]IngressCounter, name string) (IngressCounter, bool) {
	b, a := before[name], after[name]
	if a.Packets < b.Packets || a.Bytes < b.Bytes {
		return IngressCounter{}, false
	}
	return IngressCounter{Packets: a.Packets - b.Packets, Bytes: a.Bytes - b.Bytes}, true
}

func deviceCountersHeld(before, after map[string]IngressCounter, allow ...string) bool {
	allowed := map[string]bool{}
	for _, name := range allow {
		if name != "" {
			allowed[name] = true
		}
	}
	names := map[string]struct{}{}
	for name := range before {
		names[name] = struct{}{}
	}
	for name := range after {
		names[name] = struct{}{}
	}
	for name := range names {
		if allowed[name] || !deviceCounterName(name) {
			continue
		}
		if before[name] != after[name] {
			return false
		}
	}
	return true
}

func deviceCounterName(name string) bool {
	return strings.HasPrefix(name, "u_") || strings.HasPrefix(name, "d_") || strings.HasPrefix(name, "nu_") || strings.HasPrefix(name, "nd_")
}

func captureSawDatagram(text string) bool {
	return strings.Contains(text, "UDP") && strings.Contains(text, ".40000")
}

func captureEmpty(text string) bool {
	if strings.Contains(text, "UDP") || strings.Contains(text, ".40000 >") {
		return false
	}
	// timeout -s INT prints the summary. A capture stopped before that still
	// counts as empty when it reached "listening" and recorded no datagram.
	if strings.Contains(text, "0 packets captured") {
		return true
	}
	return strings.Contains(text, "listening on") && !strings.Contains(text, " IP ")
}

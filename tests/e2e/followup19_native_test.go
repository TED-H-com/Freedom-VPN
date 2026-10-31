//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2E_Followup19Native(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP19") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP19=1 with fresh stack")
	}
	t.Setenv("FVPN_E2E_NATIVE18", "1")
	TestE2E_Followup18Native(t)
}
func (r *runner) nativeMTU19(ctx context.Context, s *Stack) {
	rec := Record{ID: "native-mtu-comparison", Title: "F8-02 exact reconnect ordering with endpoint and complete header evidence", Spec: []string{"AT09", "AT11", "F8-02"}, Status: StatusPass}
	defer func() { r.record(rec) }()
	defer func() {
		_, _ = s.ApplyNativeInputs(context.WithoutCancel(ctx), NativeFixtureInputs{Underlay: "dual", MTU: 1500, Protocol: "tcp", Platform: "linux"})
	}()
	// Same four-step reconnect sequence as members' original 11,011-byte row.
	// Underlay family is independent of the IPv4 overlay payload (D-123).
	_, _ = r.lab.docker(r.lab.prefix+"-origin", "mkdir", "-p", "/var/lib/fvpn-evidence", "/var/lib/vpn-node/e2e")
	_, _ = r.lab.docker(r.lab.prefix+"-origin", "touch", "/var/lib/fvpn-evidence/enable")
	rows := []struct {
		name, family string
		mtu          int
		reconnect    bool
	}{{"matrix4", "ipv4", 1500, true}, {"matrix6", "ipv6", 1500, true}, {"matrixDual", "dual", 1500, true}, {"matrix1280", "dual", 1280, true}}
	for _, row := range rows {
		selected, err := s.ApplyNativeInputs(ctx, NativeFixtureInputs{Underlay: row.family, MTU: row.mtu, Protocol: "tcp", Platform: "linux"})
		noteJSON(&rec, row.name+"_input", selected)
		if err != nil {
			rec.Status = StatusFail
			recPathNote(&rec)(row.name+"_error", err.Error())
			continue
		}
		if row.reconnect {
			_, err = r.lab.docker(r.lab.member, "netbird", "down")
			if err == nil {
				_, err = r.lab.docker(r.lab.member, "netbird", "up")
			}
			if err != nil {
				rec.Status = StatusFail
				continue
			}
			wait, cancel := context.WithTimeout(ctx, 60*time.Second)
			_ = waitUntil(wait, time.Second, func() bool {
				p, e := s.ObserveNativeUnderlay(wait)
				chosen, ce := s.ObserveSelectedNativeExit(wait)
				return e == nil && ce == nil && chosen.Connected && p.Path.Direct+p.Path.Relayed > 0
			})
			cancel()
			time.Sleep(3 * time.Second)
		}
		var ipv6Control FixtureProbeResult
		if row.family == "dual" {
			ipv6Control, _ = s.fixtureProbe(ctx, r.lab.node, "-mode", "connect", "-addr", "[fd00::30]:9447")
		}
		paths := []string{r.lab.member, r.lab.node, r.lab.prefix + "-origin"}
		captureStarted := make([]time.Time, len(paths))
		for i, c := range paths {
			captureStarted[i] = time.Now().UTC()
			script := `echo $$ > /var/lib/vpn-node/e2e/mtu-capture.pid; exec tcpdump --immediate-mode -U -nn -s 128 -i any -w /var/lib/vpn-node/e2e/mtu.pcap 'icmp or icmp6 or tcp port 9446 or tcp port 9447 or udp' 2>/var/lib/vpn-node/e2e/mtu-capture.log`
			e := exec.CommandContext(ctx, "docker", "exec", "-d", c, "sh", "-c", script).Run()
			if e != nil {
				rec.Status = StatusFail
			}
		}
		time.Sleep(time.Second)
		before, _ := s.DeviceByteCounters(ctx, r.usageDevice)
		path, _ := s.ObserveNativeUnderlay(ctx)
		chosenBefore, cbErr := s.ObserveSelectedNativeExit(ctx)
		got, e := s.fixtureProbe(ctx, r.lab.member, "-mode", "bulk", "-addr", r.lab.originIP+":9446", "-device", r.usageDevice.String())
		after, ae := s.DeviceByteCounters(ctx, r.usageDevice)
		end, _ := s.ObserveNativeUnderlay(ctx)
		chosenAfter, caErr := s.ObserveSelectedNativeExit(ctx)
		noteJSON(&rec, row.name+"_transfer", map[string]any{"before": before, "after": after, "path_before": path, "path_after": end, "selected_before": chosenBefore, "selected_after": chosenAfter, "bulk": got})
		if e != nil || ae != nil || cbErr != nil || caErr != nil || !chosenBefore.Connected || !chosenAfter.Connected || !got.Available || got.Bytes != 64064 || !after.Present || after.Upload <= before.Upload || after.Download <= before.Download {
			rec.Status = StatusFail
		}
		if row.family == "dual" {
			denied, denyErr := s.fixtureProbe(ctx, r.lab.member, "-mode", "connect", "-addr", "[fd00::30]:9447")
			noteJSON(&rec, row.name+"_ipv6_block", map[string]any{"positive_control": ipv6Control, "native_member": denied, "scope": "D-123: IPv6 connection must fail; IPv4 bulk must complete. Native refusal may precede tunnel ingress; this does not claim ctv6 was hit. Kernel ctv6 proof is the separate netlab row."})
			if !ipv6Control.Connected || denyErr != nil || denied.Connected {
				rec.Status = StatusFail
			}
		}
		for i, c := range paths {
			stopRequested := time.Now().UTC()
			_, _ = r.lab.docker(c, "sh", "-c", `kill -INT "$(cat /var/lib/vpn-node/e2e/mtu-capture.pid)"`)
			flush, stop := context.WithTimeout(ctx, 3*time.Second)
			stopped := waitUntil(flush, 100*time.Millisecond, func() bool {
				_, e := r.lab.docker(c, "sh", "-c", `kill -0 "$(cat /var/lib/vpn-node/e2e/mtu-capture.pid)"`)
				return e != nil
			})
			stop()
			noteJSON(&rec, fmt.Sprintf("%s_capture_%d_window", row.name, i), map[string]any{"launch_requested_at": captureStarted[i], "stop_requested_at": stopRequested, "stop_observed_at": time.Now().UTC(), "process_stopped": stopped, "scope": "process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode"})
			if !stopped {
				rec.Status = StatusFail
			}
			if row.family == "dual" && i == 1 {
				index, indexErr := r.lab.docker(c, "cat", "/sys/class/net/fvup0/ifindex")
				filter := "ip6 and tcp port 9447 and outbound and ifindex " + strings.TrimSpace(string(index))
				escaped, leakErr := exec.CommandContext(ctx, "docker", "exec", c, "tcpdump", "-nn", "-r", "/var/lib/vpn-node/e2e/mtu.pcap", filter).Output()
				noLeak := indexErr == nil && leakErr == nil && len(strings.TrimSpace(string(escaped))) == 0
				recPathNote(&rec)(row.name+"_ipv6_uplink_no_leak", boolText(noLeak))
				if !noLeak {
					rec.Status = StatusFail
				}
			}
			stats, _ := r.lab.docker(c, "cat", "/var/lib/vpn-node/e2e/mtu-capture.log")
			recPathNote(&rec)(fmt.Sprintf("%s_capture_%d_stats", row.name, i), string(stats))
			// Full TCP headers only; all outer headers are decoded from the pinned private pcap.
			metadata, e := r.lab.docker(c, "tcpdump", "-nn", "-S", "-tt", "-v", "-r", "/var/lib/vpn-node/e2e/mtu.pcap", "tcp port 9446")
			recPathNote(&rec)(fmt.Sprintf("%s_capture_%d_headers", row.name, i), string(metadata))
			if e != nil {
				rec.Status = StatusFail
			}
			readback, _ := r.lab.docker(c, "sh", "-c", `for d in /sys/class/net/*; do basename "$d"; cat "$d/ifindex" "$d/mtu"; done; ip -j link; ip -j route; ip -j -6 route; ss -tin`)
			recPathNote(&rec)(fmt.Sprintf("%s_path_%d", row.name, i), string(readback))
			slot := "node"
			if i == 0 {
				slot = "member"
			}
			if i == 2 {
				slot = "origin"
			}
			receipt, exportErr := s.ExportContainerFile(ctx, slot, "/var/lib/vpn-node/e2e/mtu.pcap", filepath.Join(r.lab.root, ".agent/tmp/followup19", r.runID+"-"+row.name+"-"+slot+".pcap"))
			noteJSON(&rec, fmt.Sprintf("%s_capture_%d_receipt", row.name, i), receipt)
			if exportErr != nil {
				rec.Status = StatusFail
			}
		}
		endpoint, e := r.lab.docker(r.lab.prefix+"-origin", "cat", "/var/lib/fvpn-evidence/echo.jsonl")
		recPathNote(&rec)(row.name+"_origin_endpoint_samples", string(endpoint))
		if e != nil {
			rec.Status = StatusFail
		}
	}
	recPathNote(&rec)("topology", "The measured bottleneck is the member's eth0/eth1 MTU. The any-interface pcaps include both underlays and the tunnel, node tunnel/uplink, and controlled origin. No separate bottleneck router exists; no host networking touched.")
	recPathNote(&rec)("D123", "IPv4 overlay over the observed native underlay. IPv6Block no-leak is independently proven in docs/poc/G2-f802-mtu1280-2026-10-31.md; no IPv6 overlay success is claimed or required.")
}

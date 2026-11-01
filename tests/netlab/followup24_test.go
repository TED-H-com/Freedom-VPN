//go:build netlab

package netlab

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// ISO02/AT13: the probe return hole must obey a new deny even with conntrack
// intact. Exercise both callers of writeProbeDown, retaining resolver DNS.
func TestAT13_ProbeDownloadDestinationDeny(t *testing.T) {
	skipLab(t)
	for _, baseline := range []bool{false, true} {
		t.Run(fmt.Sprintf("baseline=%t", baseline), func(t *testing.T) {
			flushConntrack(t)
			const log = "/scratch/stream-f24-probe.log"
			_, _, _ = dockerExec("sh", "-c", "rm -f "+log+" "+log+".pid")
			t.Cleanup(func() {
				killPidFile(log + ".pid")
				stopCaptures(t)
				applyBaseline(t)
			})
			now := time.Now().UTC()
			m := functionalModel(now, 4240)
			m.Probe = &enforce.ProbeException{
				PeerID: "peer-probe", SourceV4: mustPrefix("100.64.8.1/32"),
				ResolverDNS: true, ExpiresAt: now.Add(120 * time.Second),
				Destinations: []enforce.ProbeDestination{
					{AddrV4: mustPrefix("203.0.113.10/32"), Proto: "udp", Port: 9000},
					{AddrV4: mustPrefix("203.0.113.10/32"), Proto: "tcp", Port: 9001},
				},
			}
			apply := func() {
				if baseline {
					applyBaselineModel(t, m)
				} else {
					applyAt(t, m)
				}
			}
			apply()
			if err := dockerDetach("ip", "netns", "exec", "fv-probe", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", log); err != nil {
				t.Fatal(err)
			}
			waitStreamReady(t, log, 8*time.Second)
			ct := mustExec(t, "conntrack", "-L", "-p", "udp", "-s", "100.64.8.1")
			// Reuse the tuple parser with only its fixed original member changed.
			sport, reply, ok := udpReplyPorts(strings.ReplaceAll(ct, "100.64.8.1", "100.64.1.1"))
			if !ok {
				t.Fatalf("probe stream has no conntrack tuple: %s", ct)
			}
			m.Generation++
			m.Now = time.Now().UTC()
			m.ExtraDeniedV4 = append(m.ExtraDeniedV4, mustPrefix("203.0.113.10/32"))
			// A newly forbidden tuple cannot stay in a valid probe policy. Keep
			// an allowed sibling tuple so the same probe return hole remains live.
			m.Probe.Destinations = []enforce.ProbeDestination{{AddrV4: mustPrefix("203.0.113.20/32"), Proto: "tcp", Port: 80}}
			apply()
			ct = mustExec(t, "conntrack", "-L", "-p", "udp", "-s", "100.64.8.1")
			if !strings.Contains(ct, "dport=9000") {
				t.Fatal("proof requires conntrack to remain")
			}
			before := readCounters(t, m)
			startCapture(t, "root", "fv-fh", fmt.Sprintf("udp and dst host 100.64.8.1 and dst port %d", sport), "/scratch/cap-f24-probe.txt")
			time.Sleep(300 * time.Millisecond)
			injectReply(t, reply, 4)
			time.Sleep(400 * time.Millisecond)
			stopCaptures(t)
			after := readCounters(t, m)
			if n := packetLines(captureBody(t, "/scratch/cap-f24-probe.txt")); n != 0 {
				t.Fatalf("denied probe return forwarded: %d", n)
			}
			if counterPackets(after.Counters, "ctdenyd")-counterPackets(before.Counters, "ctdenyd") < 4 {
				t.Fatal("probe return source drops not counted")
			}
			controls := runProbe(t, []probeCase{
				{"probe-sibling", "fv-probe", "tcp", "203.0.113.20", 80, "100.64.8.1", "allow", 800, "", "", "", "", 0},
				{"probe-dns-udp", "fv-probe", "dns", "198.51.100.53", 53, "100.64.8.1", "allow", 800, "", "", "", "", 0},
				{"probe-dns-tcp", "fv-probe", "tcp", "198.51.100.53", 53, "100.64.8.1", "allow", 800, "", "", "", "", 0},
			})
			printProbe(t, "probe download controls", controls)
			requireProbe(t, controls)
		})
	}
}

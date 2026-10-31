//go:build netlab

package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
)

// TEST ONLY controlled-load diagnosis, opt-in through the live lab launcher.
// The host harness owns its CPU workers. These files synchronize their actual
// start/stop after two sources have independently forwarded. No injected nft
// error, artificial command delay, deadline override or packet-loss allowance.
func followup29Sibling(t *testing.T, r *rig, p *measuredSystemPlane) {
	dir := os.Getenv("FVPN_AGENT_EVIDENCE_DIR")
	run := func(args ...string) []byte {
		t.Helper()
		out, err := exec.CommandContext(t.Context(), args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("lab command: %v %s", err, out)
		}
		return out
	}
	write := func(name string, value any) {
		t.Helper()
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	wait := func(name string) {
		t.Helper()
		for start := time.Now(); time.Since(start) < 90*time.Second; time.Sleep(20 * time.Millisecond) {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return
			}
		}
		t.Fatal("host load handshake missing", name)
	}
	run("ip", "netns", "exec", "fv-inet", "ip", "addr", "add", "203.0.113.10/32", "dev", "fv-ia")
	run("ip", "route", "add", "203.0.113.10/32", "via", "192.0.2.38", "dev", "fv-ih")
	run("sysctl", "-qw", "net.ipv4.ip_forward=1")
	for _, dev := range []string{"all", "default", "fv-fh", "fv-fa", "fv-ih"} {
		run("sysctl", "-qw", "net.ipv4.conf."+dev+".rp_filter=0")
	}
	r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24"}
	r.srv.StickyLease(false)
	r.srv.SetServiceFor(time.Hour)
	bindings := threeBindings(r.eng.clock.Wall())[:2]
	r.srv.SetFixture(fixtureWithBindings(12, bindings))
	stepOK(t, r)
	if !r.eng.serving || len(r.eng.appliedModel.Bindings) != 2 {
		t.Fatal("two-source setup failed", r.eng.decision.Errors)
	}
	type probe struct {
		Source  string `json:"source"`
		Sent    string `json:"sent_ns"`
		Ingress string `json:"ingress_ns"`
		Egress  string `json:"egress_ns"`
		End     string `json:"end_ns"`
	}
	probes := func(name string) []probe {
		t.Helper()
		before, _ := timeanchor.Mono()
		nftBefore := run("nft", "-j", "list", "table", "inet", "fvpn")
		raw := run("python3", "-u", "-c", followup29Packets)
		nftAfter := run("nft", "-j", "list", "table", "inet", "fvpn")
		after, _ := timeanchor.Mono()
		var rows []probe
		if err := json.Unmarshal(raw, &rows); err != nil {
			t.Fatal(err)
		}
		write(name+".json", map[string]any{"start_ns": strconv.FormatInt(int64(before), 10), "end_ns": strconv.FormatInt(int64(after), 10), "packets": json.RawMessage(raw), "nft_before": json.RawMessage(nftBefore), "nft_after": json.RawMessage(nftAfter)})
		return rows
	}
	for _, row := range probes("before-load") {
		if row.Ingress == "0" || row.Egress == "0" {
			t.Fatal("missing two-source positive control", row)
		}
	}
	write("ready.json", map[string]bool{"two_sources_forwarded": true})
	wait("load-ready.json")
	r.srv.SetFixture(fixtureWithBindings(13, bindings[:1]))
	found := false
	for range 40 {
		first := len(p.applies)
		stepOK(t, r)
		for _, result := range p.applies[first:] {
			d := result.Deadline
			if d.InstalledBy-d.RenderStart > d.InstallBound && d.InstallStart > 0 && !d.GuardReleased {
				found = true
				write("failed-apply.json", result.Evidence.TimingEvidence())
			}
		}
		if found {
			break
		}
	}
	rows := probes("after-loaded-attempts")
	write("load-done.json", map[string]bool{"install_overrun_found": found})
	wait("load-stopped.json")
	if !found {
		t.Fatal("controlled CPU load did not reproduce install overrun")
	}
	if r.eng.applyError == nil || r.eng.serving {
		t.Fatal("failed apply did not withdraw", r.eng.decision.Errors)
	}
	for _, row := range rows {
		if row.Ingress == "0" || row.Egress != "0" {
			t.Fatal("baseline did not block both sources", row)
		}
	}
	for range 5 {
		stepOK(t, r)
		if r.eng.serving {
			break
		}
	}
	if !r.eng.serving {
		t.Fatal("quiet recovery failed", r.eng.decision.Errors)
	}
	rows = probes("after-recovery")
	if rows[0].Egress == "0" || rows[1].Ingress == "0" || rows[1].Egress != "0" {
		t.Fatal("sibling/revoked recovery controls", rows)
	}
	assertTCWire(t, r)
	write("outcome.json", map[string]bool{"failed_install": found, "both_blocked_at_baseline": true, "sibling_recovered": true, "revoked_still_blocked": true})
}

const followup29Packets = `import socket,struct,time,select,json
def mac(dev): return bytes.fromhex(open('/sys/class/net/'+dev+'/address').read().strip().replace(':',''))
def checksum(b):
 s=sum(struct.unpack('!%dH'%(len(b)//2),b));s=(s>>16)+(s&65535);s+=(s>>16);return (~s)&65535
send=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(3));send.bind(('fv-fa',0))
reads=[]
for dev in ['fv-fh','fv-ih']:
 s=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(3));s.bind((dev,0));s.setblocking(False);reads.append(s)
rows=[]
for seq,source in enumerate(['100.64.1.1','100.64.1.2']):
 payload=struct.pack('!I',seq)+b'TEST ONLY';udp=struct.pack('!HHHH',40000,9,len(payload)+8,0)+payload
 ip=struct.pack('!BBHHHBBH4s4s',69,0,len(udp)+20,seq,0,64,17,0,socket.inet_aton(source),socket.inet_aton('203.0.113.10'));ip=ip[:10]+struct.pack('!H',checksum(ip))+ip[12:]
 row=dict(source=source,sent_ns=time.monotonic_ns(),ingress_ns=0,egress_ns=0)
 send.send(mac('fv-fh')+mac('fv-fa')+b'\x08\x00'+ip+udp);end=time.monotonic_ns()+500000000
 while time.monotonic_ns()<end:
  ready,_,_=select.select(reads,[],[],max(0,(end-time.monotonic_ns())/1e9))
  for s in ready:
   packet=s.recv(65535);at=time.monotonic_ns()
   if len(packet)>=46 and packet[12:14]==b'\x08\x00' and packet[23]==17 and packet[36:38]==struct.pack('!H',9) and packet[42:46]==struct.pack('!I',seq): row['ingress_ns' if s==reads[0] else 'egress_ns']=at
 row['end_ns']=time.monotonic_ns();rows.append({k:str(v) for k,v in row.items()})
print(json.dumps(rows))
`

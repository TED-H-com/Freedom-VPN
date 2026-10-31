//go:build netlab

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// Runs after D-096 in the same exclusively owned container and namespace.
func followup23LiveProof(t *testing.T, r *rig) {
	t.Helper()
	run := func(name string, args ...string) {
		t.Helper()
		if out, err := exec.CommandContext(t.Context(), name, args...).CombinedOutput(); err != nil {
			t.Fatalf("lab command: %v %s", err, out)
		}
	}
	run("ip", "-6", "addr", "del", "2001:db8:ff::33/64", "dev", "fv-fh")
	run("ip", "-6", "addr", "add", "2001:db8:ff::35/64", "dev", "fv-fh", "nodad")
	for range 2 {
		stepOK(t, r)
	}
	if mustBeat(t, r).Serving || mustBeat(t, r).Reason != "member_path_unconfigured" {
		t.Fatal("out-of-band address replacement did not hold", mustBeat(t, r))
	}
	binding := oneBinding(r.clock.Wall())
	binding.ValidUntil = r.clock.Wall().Add(3 * time.Hour)
	r.srv.SetServiceFor(2 * time.Hour)
	r.srv.SetFixture(generationFixture(3, binding))

	stepOK(t, r)
	if !mustBeat(t, r).Serving || r.eng.memberPath.Generation != 3 {
		t.Fatalf("newer Apply did not recapture: %+v", r.eng.decision.Errors)
	}
	t.Log("D100 real kernel: IPv6 address replaced out of band -> same generation held twice/member_path_unconfigured -> published generation 3, clean verified Apply -> hashes recaptured, serving")

	// A counted IPv4 frame uses the member veth's peer, with no tc injection.
	// The documentation destination is explicitly exempt only in this TEST ONLY lab.
	run("ip", "netns", "exec", "fv-inet", "ip", "addr", "add", "203.0.113.10/32", "dev", "fv-ia")
	run("ip", "route", "add", "203.0.113.10/32", "via", "192.0.2.38", "dev", "fv-ih")
	run("sysctl", "-qw", "net.ipv4.ip_forward=1")
	for _, dev := range []string{"all", "default", "fv-fh", "fv-fa", "fv-ih"} {
		run("sysctl", "-qw", "net.ipv4.conf."+dev+".rp_filter=0")
	}
	r.eng.cfg.TestMode = true
	r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24"}
	// A new signed generation/lease uses a 2.9s member window. The engine,
	// library and packet collector now all use the actual CLOCK_MONOTONIC.
	stepOK(t, r) // install the explicit TEST ONLY destination exemption before measuring
	followup25Renewals(t, r)
	r.srv.SetServiceFor(2900 * time.Millisecond)
	r.srv.SetFixture(generationFixture(5, binding))
	stepOK(t, r)
	if !r.eng.serving || len(r.eng.appliedModel.Bindings) != 1 {
		t.Fatalf("expiry setup failed: %+v", r.eng.decision.Errors)
	}
	r.srv.FailLeases(100)
	packetDeadline := r.eng.policyExpires
	if d := r.eng.appliedModel.Leases[0].DeadlineMono; d == nil || *d != packetDeadline {
		t.Fatal("absolute deadline missing")
	}
	beforeNFT, _ := exec.CommandContext(t.Context(), "nft", "list", "table", "inet", "fvpn").CombinedOutput()
	cmd := exec.CommandContext(t.Context(), "python3", "-u", "-c", expiryPacketScript, fmt.Sprint(int64(packetDeadline)))
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("packet trace: %v %s", err, raw)
	}
	if dir := os.Getenv("FVPN_AGENT_EVIDENCE_DIR"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, "d104-expiry.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var trace struct {
		Deadline int64 `json:"deadline_ns,string"`
		Packets  []struct {
			Sent    int64 `json:"sent_ns,string"`
			Ingress int64 `json:"ingress_ns,string"`
			Egress  int64 `json:"egress_ns,string"`
			End     int64 `json:"end_ns,string"`
		} `json:"packets"`
	}
	if err := json.Unmarshal(raw, &trace); err != nil {
		t.Fatal(err, string(raw))
	}
	var lastPositive, firstDrop int64
	positive, drops := 0, 0
	dropped := false
	for _, p := range trace.Packets {
		if p.Ingress == 0 {
			t.Fatal("packet did not reach member ingress", string(raw))
		}
		if p.Egress > 0 {
			if dropped {
				t.Fatal("positive after expiry drop", string(raw))
			}
			positive++
			lastPositive = p.Egress
			if p.Egress >= trace.Deadline {
				t.Fatal("post-deadline forwarding", string(raw))
			}
		} else {
			drops++
			dropped = true
			if firstDrop == 0 {
				firstDrop = p.Sent
			}
		}
	}
	if positive < 3 || drops < 3 || firstDrop == 0 || firstDrop > trace.Deadline+int64(100*time.Millisecond) || lastPositive < trace.Deadline-int64(enforce.DeadlineInstallBound+enforce.DeadlineGranularity) {
		afterNFT, _ := exec.CommandContext(t.Context(), "nft", "list", "table", "inet", "fvpn").CombinedOutput()
		t.Logf("before nft:\n%s\nafter nft:\n%s", beforeNFT, afterNFT)
		t.Fatal("D-104 continuity bound failed", string(raw))
	}
	t.Logf("D104 agent deadline continuity: deadline_mono_ns=%d last_positive_ns=%d offset=%s first_dropped_send_ns=%d offset=%s positives=%d drops=%d cadence=100ms", trace.Deadline, lastPositive, time.Duration(lastPositive-trace.Deadline), firstDrop, time.Duration(firstDrop-trace.Deadline), positive, drops)
	t.Logf("D104 packet trace (one CLOCK_MONOTONIC): %s", raw)
	r.eng.finishServingStep(t.Context())
	if mustBeat(t, r).Serving {
		t.Fatal("post-expiry heartbeat served")
	}
}

const expiryPacketScript = `import socket,struct,time,select,json,sys
D=int(sys.argv[1])
def mac(dev):
 return bytes.fromhex(open('/sys/class/net/'+dev+'/address').read().strip().replace(':',''))
def checksum(b):
 s=sum(struct.unpack('!%dH'%(len(b)//2),b));s=(s>>16)+(s&65535);s+=(s>>16);return (~s)&65535
src=socket.inet_aton('100.64.1.1');dst=socket.inet_aton('203.0.113.10')
send=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(3));send.bind(('fv-fa',0))
reads=[]
for dev in ['fv-fh','fv-ih']:
 s=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(3));s.bind((dev,0));s.setblocking(False);reads.append(s)
rows=[];now=time.monotonic_ns();phase=D-101000000;start=phase-((phase-now)//100000000)*100000000;seq=0
while time.monotonic_ns()<D+600000000:
 target=start+seq*100000000;time.sleep(max(0,(target-time.monotonic_ns())/1e9))
 payload=struct.pack('!I',seq)+b'TEST ONLY'
 udp=struct.pack('!HHHH',40098,40099,len(payload)+8,0)+payload
 ip=struct.pack('!BBHHHBBH4s4s',69,0,len(udp)+20,seq,0,64,17,0,src,dst)
 ip=ip[:10]+struct.pack('!H',checksum(ip))+ip[12:]
 frame=mac('fv-fh')+mac('fv-fa')+b'\x08\x00'+ip+udp
 row={'sent_ns':time.monotonic_ns(),'ingress_ns':0,'egress_ns':0};send.send(frame)
 end=row['sent_ns']+80000000
 while time.monotonic_ns()<end:
  ready,_,_=select.select(reads,[],[],max(0,(end-time.monotonic_ns())/1e9))
  for s in ready:
   packet=s.recv(65535);at=time.monotonic_ns()
   if len(packet)>=46 and packet[12:14]==b'\x08\x00' and packet[23]==17 and packet[36:38]==struct.pack('!H',40099) and packet[42:46]==struct.pack('!I',seq):
    row['ingress_ns' if s==reads[0] else 'egress_ns']=at
 row['end_ns']=time.monotonic_ns();rows.append(row);seq+=1
print(json.dumps({'deadline_ns':str(D),'packets':[{k:str(v) for k,v in row.items()} for row in rows]}))
`

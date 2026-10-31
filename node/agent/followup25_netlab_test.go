//go:build netlab

package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
)

// Real process replacement with the same durable generation and a transient
// boot-time foreign attachment. The lab owns veth address/route restoration.
func followup25Restart(t *testing.T, r *rig, p *measuredSystemPlane, restore func()) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.CommandContext(t.Context(), args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	r.srv.StickyLease(false)
	r.srv.SetMemberGrantsReady(false)
	gen := r.eng.appliedGen
	cfg := r.eng.cfg
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	run("tc", "qdisc", "add", "dev", "fv-fh", "clsact")
	run("tc", "filter", "add", "dev", "fv-fh", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
	next, err := Start(Options{Config: cfg, Clock: timeanchor.System{}, Plane: p, MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	r.eng = next
	t.Cleanup(func() { _ = next.Close() })
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		stepOK(t, r)
		if mustBeat(t, r).Serving {
			t.Fatal("dirty restart served")
		}
	}
	if next.applyError == nil || next.containment == nil {
		t.Fatal("restart failed to retain rejection")
	}
	rejectedAt := next.containment.ReadAt
	run("tc", "filter", "del", "dev", "fv-fh", "ingress", "pref", "1")
	stepOK(t, r)
	if mustBeat(t, r).Serving {
		t.Fatal("foreign clsact accepted")
	}
	run("tc", "qdisc", "del", "dev", "fv-fh", "clsact")
	start := time.Now()
	applies := len(p.applies)
	stepOK(t, r)
	if mustBeat(t, r).Serving {
		t.Fatal("served before topology owner restored path")
	}
	restore()
	stepOK(t, r)
	if next.applyError != nil || next.serving || next.appliedGen != gen || len(next.appliedModel.Bindings) != 0 {
		t.Fatalf("grant-less restart did not verify deny-all recovery: %+v", next.decision)
	}
	r.srv.SetMemberGrantsReady(true)
	for range 5 {
		time.Sleep(time.Second)
		stepOK(t, r)
		if next.serving {
			break
		}
	}
	if !mustBeat(t, r).Serving || next.applyError != nil || next.appliedGen != gen || uint64(next.lease.ConfigGeneration) != gen || len(p.applies) <= applies {
		t.Fatalf("same-generation restart did not recover: %+v", next.decision)
	}
	rep := mustReport(t, r, t.Context())
	tc, err := rep.TcEvidence.Get()
	if err != nil || !tc.Verified || uint64(tc.CandidateGeneration) != gen || !tc.ReadAt.After(rejectedAt) {
		t.Fatal("recovery lacks fresh D106", tc, err)
	}
	assertTCContract(t, tc)
	if next.containment.ReadAt != rejectedAt {
		t.Fatal("recovery erased historical read")
	}
	r.manualUsage(t)
	t.Logf("F8-01 same-generation restart: generation=%d clean Apply delta=%d removal-to-serving=%s; transient boot tc rejected; clsact-only negative; topology owner restored veth; history retained", gen, len(p.applies)-applies, time.Since(start))
}

func followup25Renewals(t *testing.T, r *rig) {
	t.Helper()
	// One-second Steps across two ordinary time-driven renewals: TEST ONLY
	// 60s node lease, 30s renewal lead (with existing jitter). No forced renewal
	// flag or generation change occurs during the measured interval.
	r.srv.StickyLease(false)
	r.srv.SetServiceFor(time.Hour)
	r.srv.SetMaxOfflineSeconds(60)
	r.eng.cfg.RenewLeadSeconds = 30
	r.srv.SetFixture(generationFixture(4, oneBinding(r.eng.clock.Wall())))
	stepOK(t, r)
	cmd := exec.CommandContext(t.Context(), "python3", "-u", "-c", renewalPackets)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(stdout)
	if line, err := reader.ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatal("capture startup", line, err)
	}
	start := time.Now()
	before := r.srv.LeaseRenewals()
	p := r.eng.plane.(*measuredSystemPlane)
	first := len(p.applies)
	for i := range 75 {
		time.Sleep(max(0, time.Until(start.Add(time.Duration(i+1)*time.Second))))
		stepOK(t, r)
		if !r.eng.serving {
			t.Fatal("renewal failed", r.eng.decision.Errors)
		}
	}
	var trace struct {
		Packets    []struct{ Sent, Ingress, Egress int64 }
		Start, End int64
	}
	if err := json.NewDecoder(reader).Decode(&trace); err != nil {
		t.Fatal(err, stderr.String())
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err, stderr.String())
	}
	positive, drop := 0, 0
	for _, packet := range trace.Packets {
		if packet.Ingress == 0 {
			t.Fatal("probe missed ingress")
		}
		if packet.Egress > 0 {
			positive++
		} else {
			drop++
		}
	}
	if dir := os.Getenv("FVPN_AGENT_EVIDENCE_DIR"); dir != "" {
		timings := make([]*enforce.TimingEvidence, 0, len(p.applies)-first)
		for _, res := range p.applies[first:] {
			timings = append(timings, res.Evidence.TimingEvidence())
		}
		raw, err := json.MarshalIndent(map[string]any{"trace": trace, "timings": timings, "operations_file": "operations.jsonl"}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "d104-renewals.json"), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("D104 all Apply/live/blocked snapshots and lease transactions retained in operations.jsonl with run/boot anchors")
	renewals := r.srv.LeaseRenewals() - before
	if renewals < 2 || positive < 50000 {
		t.Fatalf("missing renewal/traffic: renewals=%d positive=%d", renewals, positive)
	}
	for _, res := range p.applies[first:] {
		if !res.Deadline.GuardReleased || res.Deadline.InstalledBy-res.Deadline.RenderStart > res.Deadline.InstallBound {
			t.Fatal("unguarded Apply", res.Deadline)
		}
		tc := res.Evidence.TCReadbackEvidence()
		if tc == nil || !tc.Verified {
			t.Fatal("missing Apply evidence")
		}
		assertTCContract(t, *tc)
		t.Logf("D104 renewal Apply timing: %+v", res.Deadline)
	}
	t.Logf("D104 renewal availability: step_cadence=1s lease=60s lead=30s renewals=%d guarded_applies=%d packets=%d delivered=%d dropped=%d availability=%.4f%% packet_cadence=1ms duration=%s (includes every guard interval)", renewals, len(p.applies)-first, len(trace.Packets), positive, drop, 100*float64(positive)/float64(len(trace.Packets)), time.Duration(trace.End-trace.Start))
}

const renewalPackets = `import socket,struct,time,select,json

def mac(dev): return bytes.fromhex(open('/sys/class/net/'+dev+'/address').read().strip().replace(':',''))
def checksum(b):
 s=sum(struct.unpack('!%dH'%(len(b)//2),b));s=(s>>16)+(s&65535);s+=(s>>16);return (~s)&65535
src=socket.inet_aton('100.64.1.1');dst=socket.inet_aton('203.0.113.10')
send=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(3));send.bind(('fv-fa',0))
reads=[]
for dev in ['fv-fh','fv-ih']:
 s=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(3));s.bind((dev,0));s.setblocking(False);reads.append(s)
header=mac('fv-fh')+mac('fv-fa')+b'\x08\x00'
rows=[];start=time.monotonic_ns();end=start+76000000000;target=start
print('ready',flush=True)
while time.monotonic_ns()<end+80000000:
 now=time.monotonic_ns()
 if now>=target and now<end:
  seq=len(rows);payload=struct.pack('!I',seq)+b'TEST ONLY'
  udp=struct.pack('!HHHH',40098,40099,len(payload)+8,0)+payload
  ip=struct.pack('!BBHHHBBH4s4s',69,0,len(udp)+20,seq%65536,0,64,17,0,src,dst);ip=ip[:10]+struct.pack('!H',checksum(ip))+ip[12:]
  rows.append({'Sent':time.monotonic_ns(),'Ingress':0,'Egress':0});send.send(header+ip+udp);target+=1000000
 ready,_,_=select.select(reads,[],[],max(0,min(0.001,(target-time.monotonic_ns())/1e9)) if now<end else 0.001)
 for s in ready:
  while True:
   try: packet=s.recv(65535)
   except BlockingIOError: break
   at=time.monotonic_ns()
   if len(packet)>=46 and packet[12:14]==b'\x08\x00' and packet[23]==17 and packet[36:38]==struct.pack('!H',40099):
    seq=struct.unpack('!I',packet[42:46])[0]
    if seq<len(rows): rows[seq]['Ingress' if s==reads[0] else 'Egress']=at
print(json.dumps({'Packets':rows,'Start':start,'End':time.monotonic_ns()}))
`

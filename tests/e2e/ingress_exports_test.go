//go:build e2e

package e2e

import "testing"

func TestIngressCounterDelta(t *testing.T) {
	before := map[string]IngressCounter{
		"ctunku": {Packets: 3, Bytes: 100},
		"u_abc":  {Packets: 1, Bytes: 40},
		"d_abc":  {Packets: 2, Bytes: 80},
	}
	after := map[string]IngressCounter{
		"ctunku": {Packets: 4, Bytes: 129},
		"u_abc":  {Packets: 1, Bytes: 40},
		"d_abc":  {Packets: 2, Bytes: 80},
	}
	delta, ok := ingressCounterDelta(before, after, "ctunku")
	if !ok || delta.Packets != 1 || delta.Bytes != 29 {
		t.Fatalf("ctunku delta %+v ok=%t", delta, ok)
	}
	if !deviceCountersHeld(before, after, "") {
		t.Fatal("device counters moved on a ctunku-only change")
	}
	changed := ingressDeltas(before, after)
	if len(changed) != 1 || changed["ctunku"].Packets != 1 {
		t.Fatalf("deltas %+v", changed)
	}
	admitted := map[string]IngressCounter{
		"ctunku": {Packets: 4, Bytes: 129},
		"u_abc":  {Packets: 2, Bytes: 69},
		"d_abc":  {Packets: 2, Bytes: 80},
	}
	if deviceCountersHeld(after, admitted, "") {
		t.Fatal("upload movement was treated as held")
	}
	if !deviceCountersHeld(after, admitted, "u_abc") {
		t.Fatal("allowed upload counter was not ignored")
	}
	withDownload := map[string]IngressCounter{
		"ctunku": {Packets: 4, Bytes: 129},
		"u_abc":  {Packets: 2, Bytes: 69},
		"d_abc":  {Packets: 3, Bytes: 137},
	}
	if deviceCountersHeld(after, withDownload, "u_abc") {
		t.Fatal("same-device download movement was ignored")
	}
	if !deviceCountersHeld(after, withDownload, "u_abc", "d_abc") {
		t.Fatal("same-device upload and download were not both allowed")
	}
	if _, ok := ingressCounterDelta(admitted, after, "u_abc"); ok {
		t.Fatal("a backwards counter was accepted")
	}
	if !captureEmpty("0 packets captured\n") || captureEmpty("0 packets dropped\nIP 1.2.3.4.40000 > 5.6.7.8.9: UDP, length 1\n") {
		t.Fatal("empty capture classification")
	}
	if !captureEmpty("listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 96 bytes\n") {
		t.Fatal("listening capture with no datagram was not empty")
	}
	hit := "IP 100.64.0.2.40000 > 192.0.3.10.9: UDP, length 1"
	if !captureSawDatagram(hit) || captureSawDatagram("0 packets captured") {
		t.Fatal("datagram classification")
	}
}

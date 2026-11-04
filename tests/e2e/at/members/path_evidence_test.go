package members

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestAT01_PathParsersDropKeysAndRawAddresses(t *testing.T) {
	routes := parsePathRoutes([]byte(`[{"dst":"2001:db8::2","dev":"eth2","gateway":"2001:db8::1","src":"2001:db8::3","metric":1024,"cache":[],"publicKey":"TEST ONLY KEY"}]`))
	neigh := parsePathNeighbours([]byte(`[{"dst":"2001:db8::2","dev":"eth2","lladdr":"00:00:5e:00:53:01","state":["STALE"]},{"dst":"192.0.2.2","dev":"eth0","state":["FAILED"]}]`))
	wg := parseWireGuardDump([]byte("fvnb0 TEST_ONLY_PRIVATE TEST_ONLY_PUBLIC 51820 off\nfvnb0 TEST_ONLY_PEER TEST_ONLY_PSK [2001:db8::2]:51820 100.64.0.2/32 1793790000 296 480 0\n"), "TEST_ONLY_PEER", "100.64.0.2/32")
	if len(routes) != 1 || routes[0].Source == "" || routes[0].Family != "ipv6" || routes[0].Interface != "eth2" || len(neigh) != 2 || neigh[0].States[0] != "STALE" || neigh[1].States[0] != "FAILED" || !wg.Found || wg.Received != "296" || wg.Sent != "480" || wg.Handshake != "1793790000" {
		t.Fatalf("parsing: %+v %+v %+v", routes, neigh, wg)
	}
	raw, _ := json.Marshal([]any{routes, neigh, wg})
	for _, bad := range []string{"TEST_ONLY", "2001:db8", "192.0.2", "100.64.0", "00:00:5e", "publicKey"} {
		if strings.Contains(string(raw), bad) {
			t.Fatalf("unsafe output: %s", bad)
		}
	}
	if parseWireGuardDump([]byte("fvnb0 TEST_ONLY_PRIVATE TEST_ONLY_PUBLIC 51820 off"), "", "100.64.0.2").Found {
		t.Fatal("interface row used as peer")
	}
	if parseWireGuardDump([]byte("fvnb0 other (none) 192.0.2.2:51820 0.0.0.0/0 0 0 0 0"), "target", "100.64.0.2").Found {
		t.Fatal("wrong peer")
	}
	if len(parsePathRoutes([]byte("error: private.example.invalid"))) != 0 || len(parsePathNeighbours([]byte("invalid"))) != 0 {
		t.Fatal("invalid tool output accepted")
	}
}

func TestD146_ControlTCPAllowlist(t *testing.T) {
	raw := []byte("ESTAB 0 0 192.0.2.10:45786 192.0.2.20:443 users:((\"fvpn-agent\",pid=42,fd=8))\n\t cubic wscale:7,7 rto:204 rtt:0.29/0.15 mss:1448 pmtu:1500 retrans:0/2\nESTAB 0 0 192.0.2.10:45787 192.0.2.20:443 users:((\"netbird\",pid=43,fd=8))\n\t rtt:1/1 mss:100\nESTAB 0 0 [2001:db8::10]:45786 [2001:db8::20]:443 users:((\"fvpn-agent\",pid=42,fd=9))\n\t rtt:0.31/0.2 mss:1428 retrans:0/0\n")
	v := parseControlTCP(raw, "192.0.2.20", "443")
	if len(v) != 1 || v[0].RTT != "0.29/0.15" || v[0].Retransmissions != "0/2" || v[0].MSS != "1448" || v[0].Family != "ipv4" {
		t.Fatalf("TCP: %+v", v)
	}
	v6 := parseControlTCP(raw, "2001:db8::20", "443")
	if len(v6) != 1 || v6[0].Family != "ipv6" {
		t.Fatalf("TCP6: %+v", v6)
	}
	b, _ := json.Marshal(v)
	if strings.Contains(string(b), "192.0.2") || strings.Contains(string(b), "pid") {
		t.Fatal("identity escaped")
	}
}

func TestD146_ObservationWindowHalfOpenAndLogReplacement(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2026-11-04T09:06:20Z")
	end := start.Add(20 * time.Second)
	rows := []byte(`[{"received_at":"2026-11-04T09:06:19Z","report_id":"before"},{"received_at":"2026-11-04T09:06:20Z","report_id":"start","generation":"75","incarnation":"test-incarnation","payload":{"secret":"discard"}},{"received_at":"2026-11-04T09:06:39Z","report_id":"inside","generation":null},{"received_at":"2026-11-04T09:06:40Z","report_id":"end"}]`)
	v := observationsInWindow(rows, start, end)
	b, _ := json.Marshal(v)
	if len(v) != 2 || v[0].ReportID != "start" || v[1].ReportID != "inside" || strings.Contains(string(b), "payload") || strings.Contains(string(b), "discard") {
		t.Fatalf("window: %s", b)
	}
	before := []byte("time=2026-11-04T09:06:10Z level=INFO msg=started\n")
	after := append(append([]byte{}, before...), []byte("time=2026-11-04T09:06:30Z level=INFO msg=sample\n")...)
	retained := observationLogWindow(before, after, start, end)
	if retained["overwrite_or_replacement_detected"] != false || retained["timestamped_lines_in_window"] != 1 || retained["submission_events_available"] != false {
		t.Fatal(retained)
	}
	replaced := observationLogWindow(before, []byte("time=2026-11-04T09:06:35Z level=INFO msg=restarted\n"), start, end)
	if replaced["overwrite_or_replacement_detected"] != true || replaced["first_retained_log_time"] == nil {
		t.Fatal(replaced)
	}
}

func TestAT01_UnderlayCaptureParser(t *testing.T) {
	if b, e := exec.Command("python3", "-B", "underlay_evidence_test.py").CombinedOutput(); e != nil {
		t.Fatalf("capture parser: %v\n%s", e, b)
	}
}

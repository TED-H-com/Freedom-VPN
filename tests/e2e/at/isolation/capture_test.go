package isolation

import "testing"

func TestAT13_CaptureSummarySingularAndLoss(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		count int
		valid bool
	}{
		{"listening on fvmb0\n1 packet captured\n1 packet received by filter\n0 packets dropped by kernel\n", 1, true},
		{"listening on fvup0\n0 packets captured\n0 packets received by filter\n0 packets dropped by kernel\n", 0, true},
		{"listening on fvmb0\n5 packets captured\n5 packets received by filter\n0 packets dropped by kernel\n", 5, true},
		{"listening on fvmb0\n0 packets captured\n2 packets dropped by kernel\n", 0, false},
		{"listening on fvmb0\n0 packets captured\n10 packets dropped by kernel\n", 0, false},
		{"listening on fvmb0\ntcpdump: pcap_loop: The interface disappeared\n0 packets captured\n0 packets dropped by kernel\n", 0, false},
		{"tcpdump: fvmb0: No such device exists\n", 0, false},
		{"listening on fvmb0\n", 0, false},
	} {
		n, ok := CaptureCount(tc.raw)
		if n != tc.count || ok != tc.valid {
			t.Fatalf("capture %q: (%d,%t), want (%d,%t)", tc.raw, n, ok, tc.count, tc.valid)
		}
	}
}

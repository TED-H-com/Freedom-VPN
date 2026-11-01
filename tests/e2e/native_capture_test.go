//go:build e2e

package e2e

import "testing"

func TestAT01_CaptureMissingStatsNeverMeansZeroDrops(t *testing.T) {
	a, b, c, ok := parseNativeCaptureStats("10 packets captured\n12 packets received by filter\n2 packets dropped by kernel\n")
	if !ok || a != 10 || b != 12 || c != 2 {
		t.Fatalf("%d %d %d %t", a, b, c, ok)
	}
	_, _, c, ok = parseNativeCaptureStats("10 packets captured\n12 packets received by filter\n")
	if ok || c != -1 {
		t.Fatal("missing stats treated as no loss")
	}
}

func TestAT01_CaptureSingularStatistics(t *testing.T) {
	a, b, c, ok := parseNativeCaptureStats("1 packet captured\n1 packet received by filter\n0 packets dropped by kernel\n")
	if !ok || a != 1 || b != 1 || c != 0 {
		t.Fatal("singular tcpdump statistics rejected")
	}
}

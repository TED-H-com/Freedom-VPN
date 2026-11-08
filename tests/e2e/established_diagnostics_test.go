//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSEC02_MemberStartupDiagnosticBoundedProjection(t *testing.T) {
	raw := []byte(`{"error":"tcp","running":false,"tcp_echoes":0,"udp_replies":0,"startup_stage":"tcp_hello_read","startup_class":"timeout","startup_started_at":"2026-01-01T00:00:00.123456789Z","startup_failed_at":"2026-01-01T00:00:05.123456789Z","private":"TEST ONLY private key"}`)
	clean := memberStartupDiagnostic(raw)
	if len(clean) <= 200 {
		t.Fatal("fixture does not exercise previous clipping")
	}
	var got map[string]any
	if json.Unmarshal([]byte(clean), &got) != nil || got["startup_stage"] != "tcp_hello_read" || got["startup_class"] != "timeout" || got["startup_failed_at"] != "2026-01-01T00:00:05.123456789Z" || got["tcp_echoes"] != float64(0) || got["running"] != false || strings.Contains(clean, "private") {
		t.Fatalf("lost or leaked status: %s", clean)
	}
	for _, bad := range []string{"null", "{}", `{"running":null,"tcp_echoes":0,"udp_replies":0}`, `{"running":false,"tcp_echoes":null,"udp_replies":0}`, `{"running":false,"tcp_echoes":0}`, "not JSON", strings.Repeat("x", 4097), `{"tcp_echoes":-1}`, `{"running":"false"}`, `{"tcp_echoes":1.5}`} {
		if got := memberStartupDiagnostic([]byte(bad)); got != "status unavailable" {
			t.Fatalf("accepted bad status: %s", got)
		}
	}
	clean = memberStartupDiagnostic([]byte(`{"running":false,"udp_replies":0,"error":"TEST ONLY private","startup_stage":"TEST ONLY private","startup_class":"TEST ONLY private","startup_failed_at":"TEST ONLY private","tcp_echoes":18446744073709551615}`))
	if strings.Contains(clean, "private") || strings.Contains(clean, "startup_") || !strings.Contains(clean, "18446744073709551615") {
		t.Fatalf("bad enum/numeric projection: %s", clean)
	}
	legacy := memberStartupDiagnostic([]byte(`{"error":"tcp","running":false,"tcp_echoes":0,"udp_replies":0}`))
	if strings.Contains(legacy, "startup_") || !strings.Contains(legacy, `"error":"tcp"`) {
		t.Fatalf("legacy status changed: %s", legacy)
	}
}

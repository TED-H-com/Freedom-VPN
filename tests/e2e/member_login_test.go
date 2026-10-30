//go:build e2e

package e2e

import (
	"net/url"
	"testing"
	"time"
)

func TestParseLoginPageFillsLocalUser(t *testing.T) {
	page := `<a href="/oauth2/auth/local?x=1">Local</a>
<form method="post" action="/oauth2/auth/local">
<input type="text" name="login" value="">
<input type="password" name="password" value="">
</form>`
	got := parseLoginPage(page)
	if got.Action != "/oauth2/auth/local" || got.Fields["login"] != "" || len(got.Links) != 1 {
		t.Fatalf("form: %+v", got)
	}
	confirm := `<form action="/oauth2/device/auth"><input type="hidden" name="user_code" value="kept"><input name="approval" value="deny"></form>`
	got = parseLoginPage(confirm)
	if got.Fields["user_code"] != "kept" || got.Fields["approval"] != "deny" {
		t.Fatalf("confirm: %+v", got)
	}
}

func TestResolveLoginRefKeepsState(t *testing.T) {
	base, err := url.Parse("https://netbird.example.invalid/oauth2/auth/local/login?back=%2F&state=page")
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolveLoginRef(base, "/oauth2/auth/local/login?back=%2F&amp;state=kept")
	if err != nil {
		t.Fatal(err)
	}
	if got.Query().Get("state") != "kept" || got.Path != "/oauth2/auth/local/login" {
		t.Fatalf("resolved: %s", got.Redacted())
	}
}

func TestDeviceFlowURLTakesTheLastMatch(t *testing.T) {
	log := "noise\nUse this URL to log in:\n\nhttps://netbird.example.invalid/oauth2/device/auth?user_code=AAAA-BBBB.\n"
	if got := deviceFlowURL(log); got != "https://netbird.example.invalid/oauth2/device/auth?user_code=AAAA-BBBB" {
		t.Fatalf("url: %q", got)
	}
	if scrubDeviceURL(log) == log || deviceFlowURL("no url") != "" {
		t.Fatal("scrub")
	}
}

func TestDeviceFlowPathDropsTheUserCode(t *testing.T) {
	raw := "https://netbird.example.invalid/oauth2/device/auth?user_code=AAAA-BBBB"
	if got := deviceFlowPath(raw); got != "/oauth2/device/auth" {
		t.Fatalf("path: %q", got)
	}
	if deviceFlowPath("https://example.invalid/oauth2/device/auth?user_code=AAAA") != "" {
		t.Fatal("foreign host")
	}
}

func TestCoveredIntervalStaysInsideTheWindow(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	start, end, ok := coveredInterval(now.Add(-time.Hour), now.Add(time.Hour), now)
	if !ok || !start.Equal(now.Add(-30*time.Second)) || !end.Equal(now) {
		t.Fatalf("inside: %s %s %v", start, end, ok)
	}
	start, end, ok = coveredInterval(now.Add(-10*time.Second), now.Add(time.Hour), now)
	if !ok || !start.Equal(now.Add(-10*time.Second)) || !end.Equal(now) {
		t.Fatalf("clamp start: %s %s %v", start, end, ok)
	}
	start, end, ok = coveredInterval(now.Add(-time.Hour), now.Add(-time.Minute), now)
	if !ok || !end.Equal(now.Add(-time.Minute)) || start.Before(now.Add(-time.Hour)) || !end.After(start) {
		t.Fatalf("past window: %s %s %v", start, end, ok)
	}
	if _, _, ok = coveredInterval(now, now, now); ok {
		t.Fatal("empty window")
	}
}

func TestForwardGeneration(t *testing.T) {
	if n, ok := forwardGeneration(`counter comment "fvpn_generation=0 fvpn_baseline=1"`); !ok || n != 0 {
		t.Fatalf("baseline generation: %d %v", n, ok)
	}
	if n, ok := forwardGeneration(`comment "fvpn_generation=2 fvpn_metering=forward_admission_v1"`); !ok || n != 2 {
		t.Fatalf("serving generation: %d %v", n, ok)
	}
	text := `
	chain forward {
		type filter hook forward priority filter; policy drop;
		counter comment "fvpn_generation=1 fvpn_metering=forward_admission_v1"
		iifname "fvmb0" oifname "fvup0" ip saddr 100.64.0.2 accept
	}
`
	got := parseForwardChain(text)
	if got.Generation != 1 || got.MemberAccept != 1 || got.Policy != "drop" {
		t.Fatalf("chain: %+v", got)
	}
}

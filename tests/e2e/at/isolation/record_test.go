package isolation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedact_StripsSecrets(t *testing.T) {
	home := "/" + "home/" + "someone/key"
	live := "sk_" + "live_" + "abc"
	in := "path=" + home + " postgres://fvpn:s3cret@db/fvpn Bearer tok \"setup_key\":\"abc\" " + live
	out := Redact(in)
	if strings.Contains(out, "someone") || strings.Contains(out, "s3cret") || strings.Contains(out, "abc") {
		t.Fatalf("redact left a secret: %s", out)
	}
	if !strings.Contains(out, "[redacted]") {
		t.Fatalf("redact produced %s", out)
	}
}

func TestWrite_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	rec := Record{
		ID: "AT10", Title: "host isolation", Spec: []string{"§19 AT10", "ISO01"},
		Status: StatusBlocked, Build: "abc", ImageDigests: []string{"sha256:dead"},
		PolicyRevision: "0", NodeIncarnation: "none",
		Expected: "protected destinations refused", Actual: "generation 0",
		Commands:     []string{"make -f mk/at-isolation.mk at-isolation"},
		Observations: []Observation{{Name: "note", Detail: "token length only"}},
	}
	if err := Write(dir, rec); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "AT10.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "/"+"home/") {
		t.Fatal("json contains a home path")
	}
	md, err := os.ReadFile(filepath.Join(dir, "AT10.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(md)
	for _, want := range []string{"Status: blocked", "Build: abc", "Policy revision: 0", "generation 0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("markdown missing %q\n%s", want, text)
		}
	}
}

func TestParseForwardChainIgnoresNetBirdAccepts(t *testing.T) {
	text := `
	chain forward {
		type filter hook forward priority mangle; policy drop;
		oifname "fvnb0" ct state established,related counter packets 0 bytes 0 accept
		iifname "fvnb0" counter packets 0 bytes 0 accept
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1"
		iifname "fvmb0" counter name "ctbase" drop
		oifname "fvmb0" counter name "ctbase" drop
	}
`
	view := ParseForwardChain(text)
	if !view.Present || view.Policy != "drop" || !view.Baseline || view.Generation != 0 || view.MemberAccept != 0 {
		t.Fatalf("view %+v", view)
	}
	serving := `
	chain forward {
		type filter hook forward priority mangle; policy drop;
		iifname "fvmb0" ip saddr 100.64.0.5 accept comment "fvpn_generation=4"
	}
`
	view = ParseForwardChain(serving)
	if view.Generation != 4 || view.MemberAccept != 1 {
		t.Fatalf("serving view %+v", view)
	}
}

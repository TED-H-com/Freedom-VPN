package isolation

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
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
	if view.Generation != 4 || view.MemberAccept != 1 || view.MSSClamp != 0 || view.PMTUAccept != 0 {
		t.Fatalf("serving view %+v", view)
	}
}

func TestParseForwardChain_D146AndD147Shapes(t *testing.T) {
	probeBaseline := `
	chain forward {
		type filter hook forward priority -150; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1"
		iifname "fvmb0" oifname "fvup0" ip saddr @probe_v4 ip daddr @resolver_v4 udp dport 53 counter name "probe_up" accept
		iifname "fvmb0" oifname "fvup0" ip saddr @probe_v4 ip daddr . meta l4proto . th dport @probe_dst_v4 counter name "probe_up" accept
		iifname "fvup0" oifname "fvmb0" ip daddr @probe_v4 ct state established,related counter name "probe_down" accept
		iifname "fvmb0" counter name "ctbase" drop
		oifname "fvmb0" counter name "ctbase" drop
	}
`
	if got := ParseForwardChain(probeBaseline); !got.Present || !got.Baseline || got.Generation != 0 || got.MemberAccept != 3 || got.LabMemberAccept != 0 || got.MSSClamp != 0 || got.PMTUAccept != 0 {
		t.Fatalf("probe baseline view %+v", got)
	}

	fullServing := `
	chain forward {
		type filter hook forward priority -150; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=7 fvpn_metering=forward_admission_v1"
		iifname "fv-fh" oifname "fv-ih" ip saddr 100.64.1.1 ip daddr @resolver_v4 tcp dport 53 counter name "du_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-fh" oifname "fv-ih" ip saddr 100.64.1.1 ip daddr @resolver_v4 udp dport 53 counter name "du_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-fh" oifname "fv-ih" ip daddr @resolver_v4 counter name "ctdns" drop
		iifname "fv-fh" oifname "fv-ih" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fv-fh" oifname "fv-ih" ip daddr @host_v4 counter name "cthost" drop
		iifname "fv-fh" oifname "fv-ih" ip saddr 100.64.1.1 tcp flags & (syn) == syn tcp option maxseg size > 1160 tcp option maxseg size set 1160
		iifname "fv-fh" oifname "fv-ih" ip saddr 100.64.1.1 counter name "u_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-ih" oifname "fv-fh" ip saddr @resolver_v4 ip daddr 100.64.1.1 tcp sport 53 counter name "dd_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-ih" oifname "fv-fh" ip saddr @resolver_v4 ip daddr 100.64.1.1 udp sport 53 counter name "dd_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-ih" oifname "fv-fh" ip saddr @resolver_v4 ip daddr 100.64.1.1 ct state related icmp type destination-unreachable icmp code frag-needed counter name "d_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-ih" oifname "fv-fh" ip saddr @denied_v4 ip daddr 100.64.1.1 ct state related icmp type destination-unreachable icmp code frag-needed counter name "d_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-ih" oifname "fv-fh" ip saddr @host_v4 ip daddr 100.64.1.1 ct state related icmp type destination-unreachable icmp code frag-needed counter name "d_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
		iifname "fv-ih" oifname "fv-fh" ip saddr @resolver_v4 counter name "ctdenyd" drop
		iifname "fv-ih" oifname "fv-fh" ip saddr @denied_v4 counter name "ctdenyd" drop
		iifname "fv-ih" oifname "fv-fh" ip saddr @host_v4 counter name "ctdenyd" drop
		iifname "fv-ih" oifname "fv-fh" ip daddr 100.64.1.1 tcp flags & syn == syn tcp option maxseg size > 1160 tcp option maxseg size set 1160
		iifname "fv-ih" oifname "fv-fh" ip daddr 100.64.1.1 counter name "d_019a00000000700080000000000000a1" meta priority set 0x0001002f accept
	}
`
	got := ParseForwardChain(fullServing)
	if !got.Present || got.Baseline || got.Policy != "drop" || got.Generation != 7 || got.MSSClamp != 2 || got.PMTUAccept != 3 || got.LabMemberAccept != 9 {
		t.Fatalf("full serving view %+v", got)
	}
}

// Scan all committed evidence, including archives, without reading ignored state.
func TestCommittedEvidenceFiles_ValidAndHygieneClean(t *testing.T) {
	root := "../../../../docs/e2e/at/isolation"
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		logicalPath := path
		if strings.HasSuffix(path, ".gz") {
			reader, err := gzip.NewReader(bytes.NewReader(raw))
			if err != nil {
				return err
			}
			raw, err = io.ReadAll(reader)
			_ = reader.Close()
			if err != nil {
				return err
			}
			logicalPath = strings.TrimSuffix(path, ".gz")
		}
		if filepath.Ext(logicalPath) == ".json" && !json.Valid(raw) {
			t.Errorf("invalid JSON: %s", path)
		}
		if filepath.Ext(logicalPath) == ".jsonl" {
			for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
				if !json.Valid(line) {
					t.Errorf("invalid JSONL: %s", path)
					break
				}
			}
		}
		for _, bad := range []string{"/ho" + "me/", "BEGIN " + "EC PRIVATE KEY", "BEGIN " + "RSA PRIVATE KEY", "BEGIN " + "OPENSSH PRIVATE KEY", "sk_" + "live_", "gh" + "p_", "gh" + "o_"} {
			if strings.Contains(string(raw), bad) {
				t.Errorf("%s contains forbidden evidence text", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

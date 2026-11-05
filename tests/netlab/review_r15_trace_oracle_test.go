package netlab

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

var r15TraceLine = regexp.MustCompile(`^trace id ([0-9a-f]+) (\S+) (\S+) (\S+) (.+)$`)
var r15DropRule = regexp.MustCompile(`counter name "?ctunku"? drop \(verdict drop\)$`)

// The shared counter is diagnostic, not a packet identity. Only the exact
// injected tuple entering fvpn and reaching its ctunku drop proves admission.
func validateR15PolicyDrop(obs IngressObservation, trace, traceErr, uplink, captureErr string) error {
	before, beforeOK := obs.Before["ctunku"]
	after, afterOK := obs.After["ctunku"]
	if !beforeOK || !afterOK || after.Packets <= before.Packets || after.Bytes <= before.Bytes {
		return fmt.Errorf("ctunku missing, reset, or not advanced: before=%+v after=%+v", before, after)
	}
	if obs.MemberIface != "fv-fh" || obs.Source != "100.64.9.9" || obs.Dest != "203.0.113.10" || obs.SourcePort != IngressSourcePort || obs.DestPort != 39996 {
		return fmt.Errorf("unexpected injection identity: %+v", obs)
	}
	if strings.TrimSpace(traceErr) != "" {
		return fmt.Errorf("nft trace error: %s", traceErr)
	}
	if strings.TrimSpace(uplink) != "" || !strings.Contains(captureErr, "listening on fv-ih") ||
		!regexp.MustCompile(`(?m)^0 packets captured$`).MatchString(captureErr) ||
		!regexp.MustCompile(`(?m)^0 packets dropped by kernel$`).MatchString(captureErr) {
		return fmt.Errorf("uplink forwarded traffic or capture incomplete: body=%q stderr=%q", uplink, captureErr)
	}
	packetID, packets := "", 0
	drops := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(trace), "\n") {
		m := r15TraceLine.FindStringSubmatch(line)
		if m == nil {
			return fmt.Errorf("invalid nft trace line: %q", line)
		}
		if m[2] != "inet" || m[3] != "fvpn" || m[4] != "forward" {
			continue
		}
		if strings.HasPrefix(m[5], "packet:") {
			for _, field := range []string{`iif "fv-fh"`, `oif "fv-ih"`, "ip saddr 100.64.9.9", "ip daddr 203.0.113.10", "udp sport 40000", "udp dport 39996"} {
				if !strings.Contains(" "+m[5]+" ", " "+field+" ") {
					return fmt.Errorf("trace packet has wrong tuple/interface: %q", line)
				}
			}
			packetID, packets = m[1], packets+1
		}
		if strings.HasPrefix(m[5], "rule ") && r15DropRule.MatchString(m[5]) {
			drops[m[1]]++
		}
	}
	if packets != 1 || len(drops) != 1 || drops[packetID] != 1 {
		return fmt.Errorf("need one attributed fvpn ctunku drop: packets=%d packet_id=%q drops=%v", packets, packetID, drops)
	}
	return nil
}

func TestR15PolicyDropAttribution(t *testing.T) {
	// Synthetic nft text: fixtures exercise attribution, never claim a live run.
	const packet = `trace id a1 inet fvpn forward packet: iif "fv-fh" oif "fv-ih" ip saddr 100.64.9.9 ip daddr 203.0.113.10 udp sport 40000 udp dport 39996 udp length 9`
	const drop = `trace id a1 inet fvpn forward rule iifname "fv-fh" oifname "fv-ih" ip saddr != @allowed_v4 counter name "ctunku" drop (verdict drop)`
	const capOK = "tcpdump: listening on fv-ih, link-type EN10MB\n0 packets captured\n0 packets received by filter\n0 packets dropped by kernel\n"
	obs := IngressObservation{MemberIface: "fv-fh", Source: "100.64.9.9", Dest: "203.0.113.10", SourcePort: 40000, DestPort: 39996,
		Before: map[string]enforce.CounterValue{"ctunku": {Packets: 5, Bytes: 145}}, After: map[string]enforce.CounterValue{"ctunku": {Packets: 7, Bytes: 203}}}
	for _, tc := range []struct{ name, trace, traceErr, uplink, capErr string }{
		{"missing injection", drop, "", "", capOK},
		{"unrelated growth without drop", packet, "", "", capOK},
		{"wrong tuple", strings.Replace(packet, "100.64.9.9", "100.64.9.90", 1) + "\n" + drop, "", "", capOK},
		{"wrong trace identity", packet + "\n" + strings.Replace(drop, "a1", "b2", 1), "", "", capOK},
		{"wrong counter", packet + "\n" + strings.Replace(drop, "ctunku", "ctdeny", 1), "", "", capOK},
		{"duplicate injection", packet + "\n" + packet + "\n" + drop, "", "", capOK},
		{"duplicate drop", packet + "\n" + drop + "\n" + drop, "", "", capOK},
		{"forwarding", packet + "\n" + drop, "", "IP 100.64.9.9.40000 > 203.0.113.10.39996", capOK},
		{"malformed trace", packet + "\ntruncated", "", "", capOK},
		{"monitor error", packet + "\n" + drop, "No buffer space available", "", capOK},
		{"capture incomplete", packet + "\n" + drop, "", "", "listening on fv-ih"},
		{"capture dropped packets", packet + "\n" + drop, "", "", strings.Replace(capOK, "0 packets dropped", "1 packets dropped", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateR15PolicyDrop(obs, tc.trace, tc.traceErr, tc.uplink, tc.capErr); err == nil {
				t.Fatal("invalid evidence accepted")
			}
		})
	}
	if err := validateR15PolicyDrop(obs, packet+"\n"+drop, "", "\n", capOK); err != nil {
		t.Fatalf("attributed drop with extra aggregate growth: %v", err)
	}
	for _, after := range []enforce.CounterValue{{Packets: 4, Bytes: 203}, {Packets: 7, Bytes: 100}, {Packets: 5, Bytes: 145}} {
		obs.After["ctunku"] = after
		if err := validateR15PolicyDrop(obs, packet+"\n"+drop, "", "", capOK); err == nil {
			t.Fatalf("counter regression/no growth accepted: %+v", after)
		}
	}
	delete(obs.After, "ctunku")
	if err := validateR15PolicyDrop(obs, packet+"\n"+drop, "", "", capOK); err == nil {
		t.Fatal("missing counter accepted")
	}
}

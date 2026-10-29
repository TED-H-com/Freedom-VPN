//go:build netlab

package netlab

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNetBird_InterferenceFailClosed(t *testing.T) {
	skipLab(t)
	flushConntrack(t)
	t.Cleanup(func() {
		_, _, _ = dockerExec("bash", "/opt/netlab/scripts/nb.sh", "off")
		_, _, _ = dockerExec("nft", "delete", "table", "inet", "fvprobe")
	})
	mustExec(t, "bash", "/opt/netlab/scripts/nb.sh", "on")
	now := time.Now().UTC()
	m := functionalModel(now, 40)
	applyAt(t, m)

	open := runProbe(t, []probeCase{
		{"nb-allow", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0},
		{"nb-member", "fv-a1", "tcp", "100.64.1.2", 8080, "", "deny", 800, "", "", "", "", 0},
		{"nb-lan", "fv-a1", "tcp", "10.1.2.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"nb-rogue", "fv-rogue", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"nb-meta", "fv-a1", "tcp", "169.254.169.254", 80, "", "deny", 800, "", "", "", "", 0},
	})
	table := printProbe(t, "netbird accept-all inner", open)
	requireProbe(t, open)

	mustExec(t, "bash", "/opt/netlab/scripts/nb.sh", "masq")
	flushConntrack(t)
	before := readCounters(t, m)
	hidden := runProbe(t, []probeCase{
		{"masq-a1", "fv-a1", "tcp", "203.0.113.10", 80, "", "deny", 800, "", "", "", "", 0},
		{"masq-b1", "fv-b1", "icmp", "203.0.113.10", 0, "", "deny", 800, "", "", "", "", 0},
	})
	table += printProbe(t, "netbird masquerade", hidden)
	requireProbe(t, hidden)
	after := readCounters(t, m)
	if counterPackets(after.Counters, "ctunku") <= counterPackets(before.Counters, "ctunku") {
		t.Fatal("masqueraded source was not dropped as unknown")
	}
	if sampleBytes(after.Samples, devA1, "upload") != sampleBytes(before.Samples, devA1, "upload") {
		t.Fatal("masqueraded traffic was attributed to a device")
	}
	raw, _ := jsonMarshalModel(m)
	writeScratch(t, "/scratch/meter.json", raw)
	_, errOut, err := dockerExec("fvpn-netlab", "meter-path", "-f", "/scratch/meter.json", "192.0.2.34")
	if err == nil || !strings.Contains(errOut, "METER_PATH_INVALID") {
		t.Fatalf("meter path did not fail closed: %v %s", err, errOut)
	}

	mustExec(t, "bash", "/opt/netlab/scripts/nb.sh", "off")
	flushConntrack(t)
	back := runProbe(t, []probeCase{{"restored", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0}})
	table += printProbe(t, "netbird off", back)
	requireProbe(t, back)

	mark := probeMarks(t)
	table += mark + "\n"
	save(t, "netbird.txt", table)
	fmt.Print(table)
	noteSummary("NetBird inner_accept_blocked=1 masquerade_fail_closed=1 marks=" + oneLine(mark))
}

func probeMarks(t *testing.T) string {
	t.Helper()
	script := `
nft delete table inet fvprobe >/dev/null 2>&1 || true
nft -f - <<'EOF'
table inet fvprobe {
	chain pre {
		type filter hook prerouting priority -180; policy accept;
		iifname "fv-fh" meta mark set 0x1bd11 ct mark set 0x1bd22 counter
	}
	chain post {
		type filter hook postrouting priority 90; policy accept;
		oifname "fv-ih" meta mark 0x1bd11 ct mark 0x1bd22 counter
	}
}
EOF
`
	out, errOut, err := dockerExec("bash", "-c", script)
	if err != nil {
		return "mark_probe=failed " + oneLine(out+" "+errOut)
	}
	_ = runProbe(t, []probeCase{{"mark", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 800, "", "", "", "", 0}})
	list := mustExec(t, "nft", "list", "table", "inet", "fvprobe")
	_, _, _ = dockerExec("nft", "delete", "table", "inet", "fvprobe")
	save(t, "fvprobe.nft", list)
	post := 0
	pre := 0
	sections := strings.Split(list, "chain ")
	for _, sec := range sections {
		pkts := counterPacketsIn(sec)
		if strings.Contains(sec, "hook postrouting") {
			post = pkts
		}
		if strings.Contains(sec, "hook prerouting") {
			pre = pkts
		}
	}
	if pre == 0 || post == 0 {
		t.Fatalf("fwmark was not preserved (pre=%d post=%d)\n%s", pre, post, list)
	}
	return fmt.Sprintf("mark_pre=%d mark_post=%d preserved=1", pre, post)
}

func counterPacketsIn(section string) int {
	// "counter packets 3 bytes 180"
	idx := strings.Index(section, "counter packets ")
	if idx < 0 {
		return 0
	}
	rest := section[idx+len("counter packets "):]
	n := 0
	fmt.Sscan(rest, &n)
	return n
}

func jsonMarshalModel(m any) ([]byte, error) {
	return mustJSON(m), nil
}

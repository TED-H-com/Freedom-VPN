//go:build netlab

package agentlab

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Real agent process, kernel and TLS fake control. The harness owns this veth
// topology; its repair is not evidence of NetBird repair (D-108).
func TestLabISO_F6_RecoveryAndEvidence(t *testing.T) {
	skipLab(t)
	stopAgent()
	stopWatch()
	flushConntrack(t)
	admin(t, "/release", "")
	admin(t, "/service", `{"seconds":3600}`)
	admin(t, "/bump", "") // Start at generation 2, matching the original latch proof.
	// Keep the ordinary lab topology owner stopped: this case observes the
	// retained-path hold and performs the deciding repair explicitly below.
	startProc(t, "fvpn-agent", labDir+"/agent.pid", labDir+"/agent.log")
	defer stopAgent()
	if !waitProbe(t, "allow", 25*time.Second) {
		t.Fatal("initial counted control failed", debugPlane(t))
	}
	observation := func() api.ObservationReport {
		t.Helper()
		raw, ok := adminTryGet(t, "/observation")
		var rep api.ObservationReport
		if !ok || json.Unmarshal([]byte(raw), &rep) != nil {
			t.Fatal("observation unavailable")
		}
		return rep
	}
	clean := observation()
	checkCleanEvidence(t, clean)
	gen, err := clean.AppliedConfigGeneration.Get()
	if err != nil {
		t.Fatal(err)
	}
	// Save only this lab's member-link configuration before quarantine.
	var addresses []struct {
		Info []struct {
			Family, Local, Scope string
			PrefixLen            int `json:"prefixlen"`
		} `json:"addr_info"`
	}
	if err := json.Unmarshal([]byte(mustExec(t, "ip", "-j", "addr", "show", "dev", "fv-fh")), &addresses); err != nil {
		t.Fatal(err)
	}
	type route struct{ Dst, Gateway string }
	routes := map[string][]route{}
	for _, family := range []string{"-4", "-6"} {
		var rows []route
		if err := json.Unmarshal([]byte(mustExec(t, "ip", "-j", family, "route", "show", "dev", "fv-fh")), &rows); err != nil {
			t.Fatal(err)
		}
		routes[family] = rows
	}
	restore := func() {
		for _, link := range addresses {
			for _, a := range link.Info {
				if a.Family == "inet6" && a.Scope == "global" {
					mustExec(t, "ip", "-6", "addr", "replace", fmt.Sprintf("%s/%d", a.Local, a.PrefixLen), "dev", "fv-fh", "nodad")
				}
			}
		}
		for family, rows := range routes {
			for _, route := range rows {
				if route.Gateway != "" {
					mustExec(t, "ip", family, "route", "replace", route.Dst, "via", route.Gateway, "dev", "fv-fh")
				}
			}
		}
	}
	// A failed assertion must not leave this shared test topology broken for
	// later cases. This is harness-owned veth state, never an agent repair.
	t.Cleanup(func() {
		stopAgent()
		mustExec(t, "ip", "link", "set", "fv-fh", "up")
		restore()
	})
	start := time.Now()
	mustExec(t, "tc", "qdisc", "add", "dev", "fv-fh", "clsact")
	mustExec(t, "tc", "filter", "add", "dev", "fv-fh", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
	defer func() { _, _, _ = dockerExec("tc", "qdisc", "del", "dev", "fv-fh", "clsact") }()
	deadline := time.Now().Add(8 * time.Second)
	var rejected api.ObservationReport
	for time.Now().Before(deadline) {
		rejected = observation()
		if !rejected.MemberForwardingEnabled && hasTCRejection(rejected) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if rejected.MemberForwardingEnabled || !hasTCRejection(rejected) {
		t.Fatal("foreign tc did not withdraw", debugPlane(t))
	}
	link := mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh")
	if strings.Contains(link, `"UP"`) {
		t.Fatal("member link not contained", link)
	}
	contained := time.Since(start)
	admin(t, "/bump", "")
	time.Sleep(2200 * time.Millisecond)
	blocked := observation()
	got, _ := blocked.AppliedConfigGeneration.Get()
	if blocked.MemberForwardingEnabled || got > gen {
		t.Fatal("foreign filter served or ACKed new generation")
	}
	mustExec(t, "tc", "filter", "del", "dev", "fv-fh", "ingress", "pref", "1")
	mustExec(t, "tc", "qdisc", "del", "dev", "fv-fh", "clsact")
	removed := time.Now()
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh"), `"UP"`) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh"), `"UP"`) {
		t.Fatal("clean Apply never raised member link", debugPlane(t))
	}
	if observation().MemberForwardingEnabled {
		t.Fatal("served before expected IPv6 path was restored")
	}
	raised := time.Since(removed)
	restore()
	if !waitProbe(t, "allow", 15*time.Second) {
		t.Logf("recovery observation: %+v", observation())
		t.Log("member addresses", mustExec(t, "ip", "-j", "addr", "show", "dev", "fv-fh"))
		t.Log("member routes", mustExec(t, "ip", "-j", "-6", "route", "show", "table", "all", "dev", "fv-fh"))
		t.Fatal("counted forwarding did not recover", debugPlane(t))
	}
	deadline = time.Now().Add(5 * time.Second)
	var recovered api.ObservationReport
	for time.Now().Before(deadline) {
		recovered = observation()
		if recovered.MemberForwardingEnabled {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	checkCleanEvidence(t, recovered)
	got, err = recovered.AppliedConfigGeneration.Get()
	if err != nil || got <= gen || !recovered.MemberForwardingEnabled || !hasTCRejection(recovered) {
		t.Fatal("verified recovery or historical rejection missing")
	}
	leaseGen := strings.TrimSpace(mustExec(t, "python3", "-c", `import sqlite3,json
c=sqlite3.connect('file:/scratch/w2/state/agent.db?mode=ro',uri=True)
print(json.loads(c.execute('select envelope from lease').fetchone()[0])['signed']['config_generation'])`))
	if leaseGen != got.String() {
		t.Fatal("recovered lease generation", leaseGen, got)
	}
	t.Logf("ISO-F6-01/02: injection-to-contained-read=%s; removal-to-clean-link-up=%s; removal-to-counted-serving-observation=%s; applied=%s lease=%s; original rejection rows retained; first counted observation qos=%s", contained, raised, time.Since(removed), got, leaseGen, qosDetail(clean))
}

func hasTCRejection(rep api.ObservationReport) bool {
	for _, row := range rep.Errors {
		if row.Stage == "tc_evidence" && row.Code == api.ErrorCodeTCFOREIGNFILTER {
			return true
		}
	}
	return false
}
func qosDetail(rep api.ObservationReport) string {
	for _, c := range rep.Checks {
		if c.Name == api.NodeCheckNameQos && c.Detail != nil {
			return *c.Detail
		}
	}
	return ""
}
func checkCleanEvidence(t *testing.T, rep api.ObservationReport) {
	t.Helper()
	for _, c := range rep.Checks {
		if c.Name == api.NodeCheckNameQos && c.Status == api.HealthCheckStatusPassed && c.Detail != nil && strings.Contains(*c.Detail, "source=live") && strings.Contains(*c.Detail, "verified=true") && strings.Contains(*c.Detail, "sha256") && len(*c.Detail) <= 500 {
			return
		}
	}
	t.Fatalf("clean observation missing live qos evidence: %+v", rep.Checks)
}

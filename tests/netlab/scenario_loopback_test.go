//go:build netlab

package netlab

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TestDockerDNS_LoopbackScopedServes runs the lab node container on a
// user-defined Docker network, where embedded DNS NAT is in the root
// netns. Serving publication succeeds, and a non-admitted inject is still
// dropped by inet fvpn.
func TestDockerDNS_LoopbackScopedServes(t *testing.T) {
	skipLab(t)
	before := mustExec(t, "nft", "-j", "list", "ruleset")
	if !strings.Contains(before, "127.0.0.11") || !strings.Contains(before, "DOCKER_POSTROUTING") || !strings.Contains(before, `"SNAT"`) {
		t.Fatalf("embedded DNS NAT is absent:\n%s", before)
	}

	now := time.Now().UTC()
	m := functionalModel(now, 82)
	applyAt(t, m)
	if _, stderr, code := runLab(t, m, "verify"); code != 0 {
		t.Fatalf("verify with embedded DNS exit %d\n%s", code, stderr)
	}

	raw := mustExec(t, "nft", "-j", "list", "ruleset")
	rs, err := enforce.ParseRuleset([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	ev := enforce.LoopbackEvidence(rs)
	saddrN := 0
	daddrN := 0
	for _, item := range ev {
		if item.Class != enforce.LoopbackScopedClass || item.Family != "ip" || item.Table != "nat" || item.Hook != "postrouting" || !item.HasHandle {
			continue
		}
		if item.Chain == "DOCKER_POSTROUTING" && item.Match == "ip saddr 127.0.0.11" {
			saddrN++
		}
		if item.Chain == "POSTROUTING" && item.Match == "ip daddr 127.0.0.11" {
			daddrN++
		}
	}
	if saddrN < 1 || daddrN < 1 {
		t.Fatalf("postrouting loopback evidence saddr=%d daddr=%d\n%+v", saddrN, daddrN, ev)
	}

	flushConntrack(t)
	var obs IngressObservation
	uplink := captureDuring(t, "root", "fv-ih", "udp port 39997", func() {
		obs = injectPort(t, "fv-fh", "100.64.9.9", "203.0.113.10", "39997")
	})
	ctunku := counterDelta(obs, "ctunku").Packets
	uplinkN := 0
	if strings.TrimSpace(uplink) != "" {
		uplinkN = 1
	}
	if ctunku < 1 || uplinkN != 0 {
		t.Fatalf("embedded DNS changed the fvpn drop: ctunku=%d uplink=%q", ctunku, uplink)
	}

	line := "LOOPBACK dns=1 serving=1 class=" + enforce.LoopbackScopedClass +
		" postrouting_saddr=" + strconv.Itoa(saddrN) +
		" postrouting_daddr=" + strconv.Itoa(daddrN) +
		" ctunku=" + strconv.FormatUint(ctunku, 10) +
		" uplink=0"
	t.Log("SUMMARY " + line)
	noteSummary(line)
}

//go:build e2e

package members

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func (l *live) fixture(service string, args ...string) (out labstack.FixtureProbeResult, err error) {
	raw, err := l.inside(service, append([]string{"/usr/local/bin/fvpn-e2e-fixture-probe"}, args...)...)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(raw, &out)
	return out, err
}
func (l *live) nativeMatrix(r *record) {
	id := l.awaitDevice(l.nativePeer)
	if id == uuid.Nil {
		r.block("AT09 native matrix requires an admitted device")
		return
	}
	defer func() {
		_, _ = l.hook.ApplyNativeInputs(l.ctx, labstack.NativeFixtureInputs{Underlay: "dual", MTU: 1500, Protocol: "tcp", Platform: "linux"})
	}()
	for _, row := range []struct {
		family string
		mtu    int
	}{{"ipv4", 1500}, {"ipv6", 1500}, {"dual", 1500}, {"dual", 1280}, {"ipv6", 1280}} {
		label := fmt.Sprintf("native %s MTU %d", row.family, row.mtu)
		capture := l.startNativeCapture(r, "member", fmt.Sprintf("matrix-%s-%d", row.family, row.mtu))
		capture.sample("before-input")
		selection, e := l.hook.ApplyNativeInputs(l.ctx, labstack.NativeFixtureInputs{Underlay: row.family, MTU: row.mtu, Protocol: "tcp", Platform: "linux"})
		r.jsonCheck(label+" input", "available", selection, "ApplyNativeInputs isolated member interfaces", e == nil && selection.Available)
		if e != nil {
			capture.stop()
			continue
		}
		capture.sample("before-reconnect")
		_, down := l.inside("member", "netbird", "down")
		_, up := l.inside("member", "netbird", "up")
		ready := l.wait(60*time.Second, func() bool {
			path, e := l.hook.ObserveNativeUnderlay(l.ctx)
			chosen, ce := l.selectedNativePath()
			return ce == nil && chosen.Connected && e == nil && path.Path.Direct+path.Path.Relayed > 0
		})
		time.Sleep(3 * time.Second)
		beforePath, pe := l.hook.ObserveNativeUnderlay(l.ctx)
		chosenBefore, cbErr := l.selectedNativePath()
		familyOK := func(p selectedPath) bool {
			return (p.Family == "ipv4" || p.Family == "ipv6") && (row.family == "dual" || p.Family == row.family)
		}
		before, be := l.hook.DeviceByteCounters(l.ctx, id)
		capture.sample("before-bulk")
		got, ge := l.fixture("member", "-mode", "bulk", "-addr", l.originIP()+":9446", "-device", id.String())
		capture.sample("after-bulk")
		capEvidence := capture.stop()
		after, ae := l.hook.DeviceByteCounters(l.ctx, id)
		afterPath, ape := l.hook.ObserveNativeUnderlay(l.ctx)
		chosenAfter, caErr := l.selectedNativePath()
		r.jsonCheck(label+" transfer", "IPv4 overlay: 64064 TCP bytes and upload/download counters increase; selected underlay family recorded; D-123 IPv6Block, no IPv6 end-to-end MTU/PMTU claim", map[string]any{"decision": "under provisional D-137", "native_capture": capEvidence, "scope": map[bool]string{true: "receive-only 1280 without PTB; PMTU black-hole limitation, never a supported-path PASS", false: "1500 baseline"}[row.mtu == 1280], "overlay_family": "ipv4", "ipv6_mode": "block", "ipv6_no_leak_evidence": "docs/poc/G2-f802-mtu1280-2026-10-31.md (independent netlab measurement; ctv6 drops in both directions, no member class dequeue)", "selection": selection, "before_path": beforePath, "after_path": afterPath, "selected_exit_before": chosenBefore, "selected_exit_after": chosenAfter, "before_counters": before, "after_counters": after, "transfer": got}, "native down/up; ObserveNativeUnderlay; fixture bulk; DeviceByteCounters", familyOK(chosenBefore) && familyOK(chosenAfter) && cbErr == nil && caErr == nil && chosenBefore.Connected && chosenAfter.Connected && down == nil && up == nil && ready && pe == nil && be == nil && ge == nil && ae == nil && ape == nil && got.Available && got.Bytes == 64064 && before.Present && after.Present && after.Upload > before.Upload && after.Download > before.Download)
		if row.mtu == 1280 {
			// Retain the actual byte assertion on failure. Even a complete transfer
			// through this asymmetric fixture cannot certify supported PMTU.
			if r.Steps[len(r.Steps)-1].Status == "pass" {
				r.Steps[len(r.Steps)-1].Status = "blocked"
			}
			r.block("D-137 provisional limitation: receive-only 1280 without PTB is a PMTU black hole; measured bytes retained, never a supported-path PASS")
		}
		if row.family == "ipv6" && row.mtu == 1280 {
			l.knownPMTU1280(r, id)
		}
	}
	l.routedPMTU1280(r, id)
	for _, row := range []struct{ platform, protocol string }{{"linux", "quic"}, {"android", "tcp"}, {"ios", "tcp"}, {"windows", "tcp"}, {"macos", "tcp"}} {
		selection, e := labstack.SelectNativeFixture(labstack.NativeFixtureInputs{Underlay: "dual", MTU: 1280, Protocol: row.protocol, Platform: row.platform})
		if e != nil || selection.Available || selection.Reason == "" {
			r.jsonCheck("unavailable selector contract", "explicit unavailable", selection, "SelectNativeFixture", false)
			continue
		}
		raw, _ := json.Marshal(selection)
		r.Steps = append(r.Steps, step{Name: "native " + row.platform + " " + row.protocol, Status: "blocked", Expected: "real protocol/platform execution", Actual: string(raw), Command: "SelectNativeFixture; declaration only", At: time.Now().UTC().Format(time.RFC3339)})
		r.block("F4-03 " + row.platform + "/" + row.protocol + ": " + selection.Reason)
	}
	// Follow-up 15 supplies a real relay fault and member UDP, separately from QUIC.
	// Its bounded hold controls eth0. Select IPv4 so eth1 cannot remain a direct IPv6 path.
	relayInput, relayErr := l.hook.ApplyNativeInputs(l.ctx, labstack.NativeFixtureInputs{Underlay: "ipv4", MTU: 1500, Protocol: "tcp", Platform: "linux"})
	r.jsonCheck("relay IPv4 precondition", "available; IPv6 member UDP held", relayInput, "ApplyNativeInputs(ipv4) before eth0 relay hold", relayErr == nil && relayInput.Available)
	if relayErr != nil {
		return
	}
	before, _ := l.hook.ReadMemberPath(l.ctx, "")
	forced, e := l.hook.ForceMemberRelay(l.ctx, "", true)
	r.jsonCheck("forced native relay", "observed relay", map[string]any{"before": before, "forced": forced, "error": safeError(e)}, "ForceMemberRelay", e == nil && forced.Relayed > 0)
	if e == nil {
		flow := l.startPersistent(r, "member", id)
		restored, re := l.hook.ForceMemberRelay(l.ctx, "", false)
		r.jsonCheck("restore direct transport", "direct observed after fault removed", restored, "ForceMemberRelay(false)", re == nil && restored.Direct > 0)
		if flow != nil {
			time.Sleep(2 * time.Second)
			flow.cancel()
			<-flow.done
			_ = flow.stack.StopEstablishedTransports(l.ctx)
			r.jsonCheck("relay to direct same sockets", "TCP and UDP continue", flow.ticks, "EstablishedStatus; persistent sockets never restarted", len(flow.ticks) > 1 && flow.ticks[len(flow.ticks)-1].Status.TCPEchoes > flow.ticks[0].Status.TCPEchoes && flow.ticks[len(flow.ticks)-1].Status.UDPReplies > flow.ticks[0].Status.UDPReplies)
		}
	} else {
		_, _ = l.hook.ForceMemberRelay(l.ctx, "", false)
	}
	r.block("D-123: v1.0 IPv6Block. IPv6 no-leak evidence is the independent rendered-policy netlab follow-up 18 measurement (docs/poc/G2-f802-mtu1280-2026-10-31.md); this native fixture measures IPv4 payload over the selected underlay, not end-to-end IPv6 MTU/PMTU. D-137 is provisional; historical F8-02 stays failed. Known route MTU is separate from reactive PTB.")
	r.block("F4-03: physical second NAT/roaming and ICMP-black-hole PMTU remain platform/operator work; finite TCP transfers at 1500/1280 are not a full PMTU campaign.")
}

func (l *live) selectedNativePath() (selectedPath, error) {
	var peerID string
	if err := l.pool.QueryRow(l.ctx, `SELECT peer_id FROM node_incarnations WHERE id=$1`, l.inc).Scan(&peerID); err != nil {
		return selectedPath{}, fmt.Errorf("selected exit peer unavailable")
	}
	got := l.nb("GET", "/peers/"+url.PathEscape(peerID), nil)
	var peer struct {
		IP string `json:"ip"`
	}
	if got.status != 200 || json.Unmarshal(got.body, &peer) != nil || peer.IP == "" {
		return selectedPath{}, fmt.Errorf("selected exit address unavailable")
	}
	raw, err := l.inside("member", "netbird", "status", "--json")
	if err != nil {
		return selectedPath{}, err
	}
	return selectNativePath(raw, peer.IP)
}

// D-137 supported scope: prior knowledge at the selected sender, without a
// reconnect, global MTU/MSS change, or an invented reactive-PTB observation.
func (l *live) knownPMTU1280(r *record, id uuid.UUID) {
	capture := l.startNativeCapture(r, "member", "known-pmtu1280")
	defer capture.stop()
	capture.sample("before-route-constraint")
	cleanup, err := l.hook.ConstrainSelectedNativePeer1280(l.ctx)
	r.check("D-137 selected-peer 1280 constraint", "isolated selected IPv6 peer route installed", safeError(err), "ConstrainSelectedNativePeer1280; TEST ONLY known PMTU", err == nil)
	if err != nil {
		return
	}
	cleaned := false
	defer func() {
		if !cleaned {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if e := cleanup(ctx); e != nil {
				r.block("D-137 selected route cleanup failed")
			}
		}
	}()
	capture.sample("before-bulk")
	before, be := l.hook.DeviceByteCounters(l.ctx, id)
	got, ge := l.fixture("member", "-mode", "bulk", "-addr", l.originIP()+":9446", "-device", id.String())
	after, ae := l.hook.DeviceByteCounters(l.ctx, id)
	capture.sample("after-bulk")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = cleanup(ctx)
	cancel()
	cleaned = err == nil
	capture.sample("after-route-removal")
	evidence := capture.stop()
	r.jsonCheck("D-137 supported known-PMTU 1280 transfer", "64064 TCP bytes with both counters increasing; selected route metric 1280 and exact removal; no reactive PTB claim", map[string]any{"decision": "under provisional D-137", "scope": "sender-learnable path, known PMTU", "transfer": got, "before_counters": before, "after_counters": after, "native_capture": evidence, "route_removed": err == nil, "route_metric_readbacks_match": knownPMTURouteReadback(evidence)}, "ConstrainSelectedNativePeer1280; original bounded bulk; route and endpoint readbacks; exact cleanup", knownPMTURouteReadback(evidence) && be == nil && ge == nil && ae == nil && err == nil && got.Available && got.Bytes == 64064 && before.Present && after.Present && after.Upload > before.Upload && after.Download > before.Download)
}

//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestE2E_Followup23Native(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP23") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP23=1 with a fresh owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	for _, c := range []string{l.member, l.node} {
		if err = exec.CommandContext(t.Context(), "docker", "exec", c, "netbird", "debug", "log", "level", "debug").Run(); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FVPN_E2E_NATIVE18", "1")
	TestE2E_Followup18Native(t)
}

func (r *runner) nativeRouter23(ctx context.Context, s *Stack) {
	// Retain the consumer's original four-row order and the unchanged limitation
	// and known-PMTU controls before adding this separate supported-scope fixture.
	r.nativeMTU22(ctx, s)
	h, err := s.PrepareNativeRouter1280(ctx)
	if err != nil {
		r.record(Record{ID: "router-setup", Title: "Reactive PMTU hop", Status: StatusFail, Observations: []Observation{{Name: "error", Detail: err.Error()}}})
		return
	}
	for _, family := range []string{"ipv6", "ipv4"} {
		rec := Record{ID: "router-" + family, Title: "D-137 routed 1280 native bulk", Spec: []string{"AT09", "D-137"}, Status: StatusFail}
		func() {
			if err = h.Select(ctx, family); err != nil {
				recPathNote(&rec)("error", err.Error())
				return
			}
			c, e := s.startNativeCapture(ctx, map[string]string{"router": h.Container})
			if e != nil {
				recPathNote(&rec)("error", e.Error())
				return
			}
			defer r.finishNativeCapture22(ctx, &rec, c)
			if c.Sample(ctx, "before-reconnect") != nil {
				return
			}
			for _, action := range []string{"down", "up"} {
				if _, e = r.lab.docker(r.lab.member, "netbird", action); e != nil {
					return
				}
			}
			wait, cancel := context.WithTimeout(ctx, 60*time.Second)
			ready := waitUntil(wait, time.Second, func() bool {
				p, e := s.ObserveSelectedNativeExit(wait)
				return e == nil && p.Connected && p.Type == "P2P" && p.Family == family
			})
			cancel()
			if !ready {
				recPathNote(&rec)("error", "selected family not ready")
				return
			}
			time.Sleep(3 * time.Second)
			if c.Sample(ctx, "before-bulk") != nil {
				return
			}
			before, be := s.DeviceByteCounters(ctx, r.usageDevice)
			chosen, ce := s.ObserveSelectedNativeExit(ctx)
			got, ge := s.fixtureProbe(ctx, r.lab.member, "-mode", "bulk", "-addr", r.lab.originIP+":9446", "-device", r.usageDevice.String())
			after, ae := s.DeviceByteCounters(ctx, r.usageDevice)
			end, ee := s.ObserveSelectedNativeExit(ctx)
			noteJSON(&rec, "transfer", map[string]any{"before": before, "after": after, "selected_before": chosen, "selected_after": end, "bulk": got})
			if c.Sample(ctx, "after-bulk") != nil {
				return
			}
			if be == nil && ce == nil && ge == nil && ae == nil && ee == nil && chosen.Family == family && end.Family == family && got.Available && got.Bytes == 64064 && after.Upload > before.Upload && after.Download > before.Download {
				rec.Status = StatusPass
			}
		}()
		r.record(rec)
	}
	// Raw native logs are private. A separate allowlist decoder publishes events.
	for _, role := range []string{"member", "node"} {
		_, e := s.ExportContainerFile(ctx, role, "/var/lib/vpn-node/e2e/netbird-service.log", filepath.Join(r.lab.root, ".agent/tmp/followup23", r.runID, role+"-debug.log"))
		if e != nil {
			r.t.Log(e)
		}
	}
}

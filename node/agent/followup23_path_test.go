package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

type topologyPlane struct {
	*livePlane
	topology   *memberPathExecutor
	afterApply func()
}

func (p *topologyPlane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	return (SystemPlane{Executor: p.topology}).MemberPath(ctx, m, expected)
}
func (p *topologyPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.livePlane.Apply(ctx, m, prev)
	if p.afterApply != nil {
		p.afterApply()
	}
	return res, err
}
func topologyFixture() *memberPathExecutor {
	return &memberPathExecutor{
		address: `[{"ifname":"fv-fh","flags":["UP"],"addr_info":[{"family":"inet","local":"192.0.2.33","prefixlen":30}]}]`,
		routes4: `[{"dst":"100.64.0.0/10","gateway":"192.0.2.34","dev":"fv-fh"}]`,
		routes6: `[]`, routeGet: `[{"dst":"100.64.1.1","dev":"fv-fh"}]`,
	}
}
func TestD100_TopologyRefreshRequiresCleanNewerPublishedApply(t *testing.T) {
	for _, fault := range []string{"none", "verify", "path_after_apply", "apply", "legacy"} {
		t.Run(fault, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			p := &topologyPlane{livePlane: &livePlane{memPlane: r.plane}, topology: topologyFixture()}
			r.eng.plane = p
			r.eng.memberPath = nil
			stepOK(t, r)
			original := *r.eng.memberPath
			if original.Generation != 2 {
				t.Fatal("missing capture generation", original)
			}
			p.topology.address = `[{"ifname":"fv-fh","flags":["UP"],"addr_info":[{"family":"inet","local":"192.0.2.35","prefixlen":30}]}]`
			p.topology.routes4 = `[{"dst":"100.64.0.0/10","gateway":"192.0.2.36","dev":"fv-fh"}]`
			for range 2 {
				stepOK(t, r)
			}
			if mustBeat(t, r).Serving || mustBeat(t, r).Reason != "member_path_unconfigured" || !reflect.DeepEqual(original, *r.eng.memberPath) {
				t.Fatal("same generation recaptured topology", mustBeat(t, r))
			}
			if fault == "legacy" {
				r.eng.memberPath.Generation = 0
				original.Generation = 0
				stepOK(t, r)
				if r.eng.serving {
					t.Fatal("legacy same-generation refresh")
				}
			}
			r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
			advanceToRenewal(r)
			switch fault {
			case "verify":
				p.injectApply = true
			case "apply":
				p.foreign = true
			case "path_after_apply":
				p.afterApply = func() { p.topology.routeGet = `[{"dev":"fv-ih"}]` }
			}
			stepOK(t, r)
			if fault != "none" && fault != "legacy" {
				if r.eng.serving || !reflect.DeepEqual(original, *r.eng.memberPath) {
					t.Fatal("failed apply refreshed topology")
				}
				p.injectApply = false
				p.foreign = false
				p.afterApply = nil
				p.topology.routeGet = `[{"dev":"fv-fh"}]`
				stepOK(t, r)
			}
			if !mustBeat(t, r).Serving || r.eng.memberPath.Generation != 3 || reflect.DeepEqual(original.Addresses, r.eng.memberPath.Addresses) || reflect.DeepEqual(original.Routes, r.eng.memberPath.Routes) {
				t.Fatalf("new generation did not recapture: beat=%+v path=%+v errors=%+v", mustBeat(t, r), r.eng.memberPath, r.eng.decision.Errors)
			}
			saved, ok, err := r.eng.store.Meta(t.Context(), "member_path")
			if err != nil || !ok || saved == "" {
				t.Fatal("capture not durable", err)
			}
			restartEngine(t, r, p)
			if r.eng.memberPath == nil || r.eng.memberPath.Generation != 3 {
				t.Fatal("restart lost capture generation")
			}
			p.topology.address = topologyFixture().address
			stepOK(t, r)
			if r.eng.serving || r.eng.memberPath.Generation != 3 {
				t.Fatal("same generation refreshed after restart")
			}
		})
	}
}

func TestD100_LegacyTopologyGenerationPinsBeforeBaselineACK(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	path := MemberPath{Interface: "fv-fh", Addresses: []string{"TEST ONLY topology hash"}}
	raw, err := json.Marshal(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.store.SetMeta(t.Context(), "member_path", string(raw)); err != nil {
		t.Fatal(err)
	}
	restartEngine(t, r, r.plane)
	if r.eng.memberPath.Generation != 2 {
		t.Fatal("legacy capture generation not pinned")
	}
	// A newer baseline ACK is not a topology capture and cannot move the pin.
	r.eng.appliedGen = 3
	if err := r.eng.migrateMemberPath(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.eng.memberPath.Generation != 2 {
		t.Fatal("baseline ACK changed capture provenance")
	}
}

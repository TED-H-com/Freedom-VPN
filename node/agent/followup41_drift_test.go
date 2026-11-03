package agent

import (
	"testing"
	"time"
)

// TEST ONLY real enforcement/verifier over the existing emulated nft/tc
// executor. No host networking, no performance or timing claim.
func TestISO01_F41_SkippedTicksStillDetectLiveDrift(t *testing.T) {
	for _, kind := range []string{"deleted_element", "deleted_chain", "foreign_tc"} {
		t.Run(kind, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			p := &replayPlane{memPlane: r.plane}
			r.eng.plane = p
			r.clock.Advance(time.Second)
			r.eng.leaseRefreshPending = true
			stepOK(t, r)
			if !r.eng.serving {
				t.Fatalf("fixture failed: %+v", r.eng.applyError)
			}
			before := r.plane.applies
			scripts := len(p.x.scripts)
			r.clock.Advance(10 * time.Second)
			stepOK(t, r)
			if r.plane.applies != before || len(p.x.scripts) != scripts {
				t.Fatal("healthy tick rewrote kernel")
			}
			if err := p.x.fold(); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "deleted_element":
				p.x.table.sets["allowed_v4"].elems = nil
			case "deleted_chain":
				delete(p.x.table.chains, "forward")
			case "foreign_tc":
				p.x.foreignTC = true
			}
			r.clock.Advance(10 * time.Second)
			stepOK(t, r)
			if r.eng.serving || mustBeat(t, r).Serving || r.eng.applyError == nil {
				t.Fatalf("drift not withdrawn: %v %+v", r.eng.serving, r.eng.applyError)
			}
			if kind == "foreign_tc" {
				if !r.eng.quarantinePending || !p.x.foreignTC {
					t.Fatal("foreign attachment was not retained under quarantine")
				}
				p.x.foreignTC = false // only its owner removes the injected filter
			}
			r.clock.Advance(10 * time.Second)
			stepOK(t, r)
			if !r.eng.serving || r.plane.applies <= before || !mustBeat(t, r).Serving {
				t.Fatalf("repair did not recover: %v %+v", r.eng.serving, r.eng.applyError)
			}
		})
	}
}

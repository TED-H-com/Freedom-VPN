package agent

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func reviewForeignEvidence() enforce.ServingEvidence {
	ev := enforce.ServingEvidence{TC: make([]enforce.TCReadEvidence, 20)}
	for i := range ev.TC {
		ev.TC[i].Interface = "bad iface " + strconv.Itoa(i)
		ev.TC[i].Objects = make([]enforce.TCObject, 6)
		for j := range ev.TC[i].Objects {
			ev.TC[i].Objects[j] = enforce.TCObject{
				Kind:    "qdisc",
				Handle:  "1:" + strconv.Itoa(j),
				Actions: []string{"mirred", "pedit", "nat", "skbedit", "bpf"},
			}
		}
	}
	return ev
}

// TestReviewR16_ContainmentEvidenceIsCapped bounds retained tc evidence and
// replaces it on the next read. Initial quarantine is kept once.
func TestReviewR16_ContainmentEvidenceIsCapped(t *testing.T) {
	e := &Engine{}
	e.recordForeign(reviewForeignEvidence())
	if e.containment == nil || e.containment.Code != api.ErrorCodeTCFOREIGNFILTER || len(e.containment.TC) != 16 {
		t.Fatalf("cap: %+v", e.containment)
	}
	for _, read := range e.containment.TC {
		if read.Interface != "redacted" || len(read.Objects) != 4 {
			t.Fatalf("read not bounded: %+v", read)
		}
		for _, obj := range read.Objects {
			if len(obj.Actions) != 4 {
				t.Fatalf("actions not bounded: %+v", obj)
			}
		}
	}
	first := e.containmentJSON()
	e.recordForeign(reviewForeignEvidence())
	if e.containmentJSON() != first || len(e.containment.TC) != 16 {
		t.Fatal("second foreign read grew containment evidence")
	}

	e.planeLocked = true
	e.plane = &memPlane{}
	e.appliedModel = &enforce.Model{Datapath: enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih"}}
	e.quarantine(t.Context())
	if e.containment.InitialQuarantine == nil || e.containment.Quarantine == nil {
		t.Fatal("quarantine evidence missing")
	}
	raw, _ := json.Marshal(e.containment.InitialQuarantine)
	e.quarantine(t.Context())
	again, _ := json.Marshal(e.containment.InitialQuarantine)
	if string(raw) != string(again) {
		t.Fatalf("initial quarantine was replaced: %s -> %s", raw, again)
	}
	if e.containmentJSON() == "" {
		t.Fatal("evidence dropped")
	}
}

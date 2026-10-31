package enforce

import (
	"context"
	"strings"
	"testing"
)

// TestReviewR16_QuarantineNeverLowersUplink lowers only the member ingress.
// A model that names one interface as both member ingress and uplink is
// rejected before any link command.
func TestReviewR16_QuarantineNeverLowersUplink(t *testing.T) {
	f := newQuarantineExec(t)
	if _, err := Quarantine(context.Background(), f, f.m); err != nil {
		t.Fatal(err)
	}
	sawMember := false
	for _, op := range f.ops {
		if strings.Contains(op, "link set dev "+f.m.Datapath.Uplink) {
			t.Fatalf("TODO(review): quarantine touched the uplink: %s", op)
		}
		if op == "ip link set dev fv-fh down" {
			sawMember = true
		}
	}
	if !sawMember {
		t.Fatalf("member link was not lowered: %v", f.ops)
	}
	f.m.Datapath.MemberIngress = f.m.Datapath.Uplink
	f.ops = nil
	if err := validateDatapath(f.m.Datapath); err == nil {
		t.Fatal("shared member/uplink interface was accepted")
	}
	if _, err := Quarantine(context.Background(), f, f.m); err == nil || len(f.ops) != 0 {
		t.Fatalf("TODO(review): shared interface was mutated: %v %v", err, f.ops)
	}
}

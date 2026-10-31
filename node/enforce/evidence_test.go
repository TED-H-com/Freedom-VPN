package enforce

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestISO_F6_BoundedRejectionBeforeRollback(t *testing.T) {
	m := smallModel()
	m.Revoke = nil
	m.Generation = 18446744073709551615
	f := servingExec(m)
	inner := f.handler
	f.handler = func(name string, args []string) ([]byte, error) {
		if name == "tc" && strings.Join(args, " ") == filterCommand(m.Datapath.MemberIngress, "ingress") {
			return []byte(`[{"kind":"flower","options":{"handle":42,"keys":{"dst_ip":"203.0.113.19"},"actions":[{"kind":"mirred","to_dev":"private-destination"}]}}]`), nil
		}
		return inner(name, args)
	}
	res, err := Apply(context.Background(), f, m, nil)
	requireCheck(t, err, ReasonTCForeignFilter)
	s := res.Evidence.TCSummary
	if res.Verified || s.Generation.String() != "18446744073709551615" || s.Phase != "pre_apply" || s.Rejected == nil || len(s.Rejected.Objects) != 1 || s.Rejected.Objects[0].Handle != "42" {
		t.Fatalf("rejected snapshot lost: %+v", s)
	}
	if res.Quarantine == nil || !res.Quarantine.Baseline.Installed {
		t.Fatal("not contained")
	}
	body, _ := json.Marshal(res.Evidence)
	for _, secret := range []string{"203.0.113.19", "private-destination", "dst_ip", "to_dev"} {
		if strings.Contains(string(body), secret) {
			t.Fatal("selector or action parameter escaped")
		}
	}
	if len(res.Evidence.ReportDetail()) > 500 {
		t.Fatal("summary exceeds product detail bound")
	}
}

func TestISO_F6_SummaryDeterminismAndBounds(t *testing.T) {
	var reads []TCReadEvidence
	for i := range 100 {
		r := TCReadEvidence{Interface: "fv-member", Attachment: fmt.Sprintf("1:%x", i), Objects: []TCObject{}}
		for j := range 20 {
			r.Objects = append(r.Objects, TCObject{Kind: "flower", Handle: fmt.Sprint(j), Actions: []string{"mirred", "bpf", "skbedit", "pedit", "nat"}})
		}
		reads = append(reads, r)
	}
	rejected := reads[99]
	obj := rejected.Objects[19]
	err := tcCheck(rejected, &obj)
	a := ServingEvidence{TC: reads}
	a.retainTC(14, "verify", err)
	for i, j := 0, len(reads)-1; i < j; i, j = i+1, j-1 {
		reads[i], reads[j] = reads[j], reads[i]
	}
	b := ServingEvidence{TC: reads}
	b.retainTC(14, "verify", err)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("dump ordering changes evidence")
	}
	if len(a.TC) != 16 || !a.TCSummary.Truncated || a.TCSummary.Reads != 100 || a.TCSummary.Objects != 2000 || a.TCSummary.Rejected.Attachment != "1:63" || a.TCSummary.Rejected.Objects[0].Handle != "19" {
		t.Fatalf("bounds lost deciding object: %+v", a.TCSummary)
	}
	for _, r := range a.TC {
		if len(r.Objects) > 4 {
			t.Fatal("object bound")
		}
		for _, o := range r.Objects {
			if len(o.Actions) > 4 {
				t.Fatal("action bound")
			}
		}
	}
	for _, s := range []string{"invalid with space", "https://example.invalid/private", strings.Repeat("x", 33)} {
		if cleanTCToken(s) != "redacted" {
			t.Fatal("token not sanitized")
		}
	}
	// Maximal valid token lengths exercise the plain-text fallback without
	// clipping an object into invalid JSON or dropping the deciding identity.
	token := strings.Repeat("x", 32)
	a.TCSummary.Rejected = &TCReadEvidence{Interface: token, Attachment: token, Error: token, Objects: []TCObject{{Kind: token, Handle: token, Parent: token, Actions: []string{token, token, token, token}}}}
	if n := len(a.ReportDetail()); n > 500 {
		t.Fatalf("detail bytes=%d", n)
	}
}

func TestISO_F6_LoadFailureRetainsAttachment(t *testing.T) {
	m := smallModel()
	m.Revoke = nil
	f := servingExec(m)
	inner := f.handler
	f.handler = func(name string, args []string) ([]byte, error) {
		if name == "tc" && strings.Join(args, " ") == "-j class show dev "+m.Datapath.Uplink {
			return nil, fmt.Errorf("TEST ONLY: private command output")
		}
		return inner(name, args)
	}
	res, err := Apply(context.Background(), f, m, nil)
	requireCheck(t, err, ReasonTCForeignFilter)
	s := res.Evidence.TCSummary
	if s.Phase != "load" || s.Rejected == nil || s.Rejected.Interface != m.Datapath.Uplink || s.Rejected.Attachment != "class" || s.Rejected.Error != "read_failed" {
		t.Fatalf("lost load error: %+v", s)
	}
	if strings.Contains(res.Evidence.ReportDetail(), "private") {
		t.Fatal("raw error leaked")
	}
}

func TestISO_F6_PopulatedAttachmentsPrecedeEmptyBlocks(t *testing.T) {
	ev := ServingEvidence{}
	for i := range 20 {
		ev.TC = append(ev.TC, TCReadEvidence{Interface: "fv-a", Attachment: fmt.Sprintf("1:%x", i)})
	}
	rejected := TCReadEvidence{Interface: "fv-z", Attachment: "qdisc", Objects: []TCObject{{Kind: "clsact", Handle: "ffff:", Parent: "ffff:fff1"}}}
	ev.TC = append(ev.TC, rejected, TCReadEvidence{Interface: "fv-z", Attachment: "egress", Objects: []TCObject{{Kind: "matchall", Actions: []string{"skbedit"}}}})
	ev.retainTC(14, "verify", tcCheck(rejected, &rejected.Objects[0]))
	if ev.TC[0].Attachment != "qdisc" || ev.TC[1].Attachment != "egress" || len(ev.TC) != 16 || !ev.TCSummary.Truncated {
		t.Fatalf("lost populated attachment behind empty blocks: %+v", ev.TC)
	}
}

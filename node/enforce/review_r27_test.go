package enforce

import (
	"context"
	"strings"
	"testing"
)

// TestR27_Held_LoadTCAndVerifyServingClassifyReadFailedAsTCForeignFilter
// confirms the orchestrator's thread on base 7dd898b9:
//   - loadTC (node/enforce/apply.go:395-418) and readServingTC
//     (node/enforce/verify_tc.go:183-186) record Error: "read_failed" when
//     `tc -j ... show dev <iface>` fails (including when <iface> does not
//     exist yet at boot: `Cannot find device "<iface>"`).
//   - tcCheck(r, nil) (node/enforce/evidence.go:183-189) turns any non-empty
//     r.Error into ReasonTCForeignFilter ("<iface> <attachment>: kind unreadable: read_failed").
//   - When Quarantine (node/enforce/quarantine.go:53-116) runs while <iface>
//     cannot be read/lowered, it still installs the quarantine_<iface> nft set
//     marker and the verified no-probe baseline, and reports
//     Quarantined: false, Recorded: true, Links[0].Error: "down_verify_failed",
//     and ErrQuarantineFailed.
//   - Once <iface> exists and no foreign tc object is present, Apply verifies
//     the staged ruleset, raises <iface>, removes quarantine_<iface>, and
//     restores Verified: true, Quarantined: false.
func TestR27_Held_LoadTCAndVerifyServingClassifyReadFailedAsTCForeignFilter(t *testing.T) {
	ctx := context.Background()
	f := newQuarantineExec(t)
	f.foreign = false
	f.fault = "tc_read"

	// 1. VerifyServing on an unreadable/missing interface returns ReasonTCForeignFilter
	// with Rejected.Error == "read_failed".
	ev, err := VerifyServing(ctx, f, f.m, f.plan.Assignments)
	if ReasonCode(err) != ReasonTCForeignFilter {
		t.Fatalf("VerifyServing reason = %q (%v), want %q", ReasonCode(err), err, ReasonTCForeignFilter)
	}
	if !strings.Contains(err.Error(), "kind unreadable: read_failed") {
		t.Fatalf("VerifyServing error = %q, want kind unreadable: read_failed", err.Error())
	}
	tcEv := ev.TCReadbackEvidence()
	if tcEv == nil || tcEv.Verified {
		t.Fatalf("VerifyServing TCReadbackEvidence = %+v, want unverified snapshot", tcEv)
	}
	rej, rErr := tcEv.Rejected.Get()
	if rErr != nil || rej.Interface != f.m.Datapath.MemberIngress || rej.Attachment != "qdisc" || rej.Error == nil || *rej.Error != "read_failed" {
		t.Fatalf("VerifyServing Rejected = %+v (err=%v), want interface=%s attachment=qdisc error=read_failed", rej, rErr, f.m.Datapath.MemberIngress)
	}

	// 2. Apply -> loadTC on an unreadable/missing interface also returns ReasonTCForeignFilter
	// with Phase == "load" and Rejected.Error == "read_failed".
	res, err := Apply(ctx, f, f.m, nil)
	if ReasonCode(err) != ReasonTCForeignFilter || res.Verified {
		t.Fatalf("Apply on unreadable tc: verified=%v reason=%q err=%v", res.Verified, ReasonCode(err), err)
	}
	loadEv := res.Evidence.TCReadbackEvidence()
	if loadEv == nil || loadEv.Phase != "load" {
		t.Fatalf("Apply TCReadbackEvidence phase = %+v, want load", loadEv)
	}

	// 3. Quarantine when the interface does not exist yet ("link_read" fails)
	// records the quarantine_<iface> set and verified baseline, and reports down_verify_failed.
	f.fault = "link_read"
	q, qErr := Quarantine(ctx, f, f.m)
	if qErr == nil || q.Quarantined || !q.Recorded || !q.Baseline.Installed || !q.Baseline.Verified {
		t.Fatalf("Quarantine with missing interface: q=%+v err=%v", q, qErr)
	}
	if len(q.Links) != 1 || q.Links[0].Interface != f.m.Datapath.MemberIngress || q.Links[0].Error != "down_verify_failed" {
		t.Fatalf("Quarantine link evidence = %+v, want down_verify_failed", q.Links)
	}

	// 4. Once the interface exists and has no foreign tc objects, Apply releases
	// quarantine_<iface>, raises the link, and verifies serving.
	f.fault = ""
	f.foreign = false
	clean, err := Apply(ctx, f, f.m, nil)
	if err != nil || !clean.Verified || clean.Evidence.Quarantined || !f.up || f.marker {
		t.Fatalf("Apply recovery after interface appears: clean=%+v up=%v marker=%v err=%v", clean, f.up, f.marker, err)
	}
}

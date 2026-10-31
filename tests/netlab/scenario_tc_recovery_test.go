//go:build netlab

package netlab

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// The library must not carry a historical TC verdict into a newer Apply.
// This does not bypass the separate agent lease/decision recovery gate.
func TestISO_F6_01_ForeignRemovalNewGeneration(t *testing.T) {
	skipLab(t)
	applyBaseline(t)
	m := functionalModel(time.Now().UTC(), 31)
	t.Cleanup(func() {
		_, _, _ = dockerExec("tc", "qdisc", "del", "dev", "fv-fh", "clsact")
		applyAt(t, m)
		reconcileLabMemberPath(t) // lab owns the veth's static routes/IPv6 address
		applyBaseline(t)
	})
	type row struct {
		Stage    string                  `json:"stage"`
		Started  time.Time               `json:"started_at"`
		Finished time.Time               `json:"finished_at"`
		Exit     int                     `json:"exit"`
		Evidence *api.TCReadbackEvidence `json:"tc_evidence"`
	}
	trace := struct {
		Scope       string `json:"scope"`
		Rows        []row  `json:"rows"`
		RouteOwner  string `json:"route_owner"`
		MemberProbe bool   `json:"member_probe_after_route_reconciliation"`
	}{Scope: "TEST ONLY: direct node/enforce on lab veth; no agent decision or lease bypass claim", RouteOwner: "lab veth topology; explicit reconciliation after verified Apply"}
	persist := func() {
		raw, err := json.MarshalIndent(trace, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		save(t, "trace.json", string(raw)+"\n")
	}
	check := func(stage, cmd string, wantExit int, want api.ErrorCode, verified bool) *api.TCReadbackEvidence {
		t.Helper()
		started := time.Now().UTC()
		out, stderr, code := runLab(t, m, cmd)
		finished := time.Now().UTC()
		var doc struct {
			Verified   bool                    `json:"verified"`
			Evidence   enforce.ServingEvidence `json:"evidence"`
			TCEvidence *api.TCReadbackEvidence `json:"tc_evidence"`
		}
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		ev := doc.TCEvidence
		trace.Rows = append(trace.Rows, row{stage, started, finished, code, ev})
		persist()
		if ev == nil || code != wantExit || ev.Verified != verified || ev.CandidateGeneration != decstr.U64(m.Generation) || ev.ReadAt.Before(started) || ev.ReadAt.After(finished) {
			t.Fatalf("%s: code=%d stderr=%s evidence=%+v", stage, code, stderr, ev)
		}
		got, _ := ev.ErrorCode.Get()
		if got != want || (want == "" && !ev.ErrorCode.IsNull()) {
			t.Fatalf("%s error=%s want=%s", stage, got, want)
		}
		if verified && (!ev.Rejected.IsNull() || (cmd == "apply" && (!doc.Verified || doc.Evidence.Quarantined)) || strings.Contains(out+stderr, "TC_FOREIGN_FILTER")) {
			t.Fatal("clean Apply retained foreign verdict or quarantine")
		}
		if !verified && want != api.ErrorCodeTCFOREIGNFILTER && !ev.Rejected.IsNull() {
			t.Fatal("clean tc retained deciding object")
		}
		projection := struct {
			Reads     int                    `json:"reads"`
			Objects   int                    `json:"objects"`
			Truncated bool                   `json:"truncated"`
			TC        []api.TCReadAttachment `json:"tc"`
		}{ev.ReadsTotal, ev.ObjectsTotal, ev.Truncated, ev.Attachments}
		raw, err := json.Marshal(projection)
		if err != nil {
			t.Fatal(err)
		}
		hash, err := canonjson.HashHex(raw)
		if err != nil || hash != ev.Sha256 {
			t.Fatalf("wire digest: %s %v", hash, err)
		}
		return ev
	}
	check("initial generation 31", "apply", 0, "", true)
	mustExec(t, "tc", "qdisc", "add", "dev", "fv-fh", "clsact")
	mustExec(t, "tc", "filter", "add", "dev", "fv-fh", "ingress", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
	rejected := check("foreign filter injected", "verify", 3, api.ErrorCodeTCFOREIGNFILTER, false)
	if rejected.Rejected.IsNull() {
		t.Fatal("lost deciding object")
	}
	check("dirty Apply quarantines", "apply", 3, api.ErrorCodeTCFOREIGNFILTER, false)
	assertLabLink(t, "fv-fh", false)
	// Removing only the filter does not authorize a foreign clsact qdisc.
	mustExec(t, "tc", "filter", "del", "dev", "fv-fh", "ingress")
	check("filter removed but foreign clsact remains", "verify", 3, api.ErrorCodeTCFOREIGNFILTER, false)
	mustExec(t, "tc", "qdisc", "del", "dev", "fv-fh", "clsact")
	check("all foreign objects removed; baseline still installed", "verify", 3, api.ErrorCodeGENERATIONMISMATCH, false)
	m.Generation = 32
	m.QoS.Down.CeilBPS += 1_000_000
	clean := check("clean newer generation 32 Apply", "apply", 0, "", true)
	assertLabLink(t, "fv-fh", true)
	assertLabLink(t, "fv-ih", true)
	check("fresh public verify generation 32", "verify", 0, "", true)
	if !clean.ReadAt.After(rejected.ReadAt) {
		t.Fatal("stale read time")
	}
	reconcileLabMemberPath(t)
	probes := runProbe(t, []probeCase{{"recovered-member", "fv-a1", "tcp", "203.0.113.10", 80, "", "allow", 2000, "", "", "", "", 0}})
	requireProbe(t, probes)
	trace.MemberProbe = true
	persist()
	t.Logf("31 rejected %s -> foreign removed: GENERATION_MISMATCH -> Apply 32 verified %s; TC_FOREIGN_FILTER absent, marker cleared, member probe passed after lab route reconciliation", rejected.ReadAt.Format(time.RFC3339Nano), clean.ReadAt.Format(time.RFC3339Nano))
}

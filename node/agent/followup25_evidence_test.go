package agent

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func assertTCContract(t *testing.T, tc api.TCReadbackEvidence) {
	t.Helper()
	if tc.ReadAt.IsZero() || tc.ReadAt.Location() != time.UTC || len(tc.Attachments) > 16 {
		t.Fatalf("invalid read: %+v", tc)
	}
	token := func(s string) {
		t.Helper()
		if s == "" || evidenceToken(s) != s {
			t.Fatalf("unredacted token %q", s)
		}
	}
	row := func(iface, attachment string, objects []api.TCReadObject, limit int) {
		token(iface)
		token(attachment)
		if len(objects) > limit {
			t.Fatal("object bound")
		}
		for _, o := range objects {
			token(o.Kind)
			if o.Handle != nil {
				token(*o.Handle)
			}
			if o.Parent != nil {
				token(*o.Parent)
			}
			if o.Actions != nil {
				if len(*o.Actions) > 4 {
					t.Fatal("action bound")
				}
				for _, a := range *o.Actions {
					token(a)
				}
			}
		}
	}
	for _, a := range tc.Attachments {
		row(a.Interface, a.Attachment, a.Objects, 4)
	}
	if r, err := tc.Rejected.Get(); err == nil {
		row(r.Interface, r.Attachment, r.Objects, 1)
	}
	raw, err := json.Marshal(map[string]any{"reads": tc.ReadsTotal, "objects": tc.ObjectsTotal, "truncated": tc.Truncated, "tc": tc.Attachments})
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(canonical)); got != tc.Sha256 {
		t.Fatalf("digest changed: %s != %s", got, tc.Sha256)
	}
}

func TestD106_ContractExamplesRoundTrip(t *testing.T) {
	for _, name := range []string{"verified", "rejected", "truncated", "null"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "contracts", "examples", "valid", "ObservationReport.tc-"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var rep api.ObservationReport
			if err := json.Unmarshal(raw, &rep); err != nil {
				t.Fatal(err)
			}
			requireObservationSchema(t, rep)
			tc, err := rep.TcEvidence.Get()
			if name == "null" {
				if err == nil || !rep.TcEvidence.IsNull() {
					t.Fatal("null became a read")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				assertTCContract(t, tc)
				if tc.CandidateGeneration.String() != "18446744073709551615" {
					t.Fatal("generation lost precision")
				}
			}
			encoded, err := json.Marshal(rep)
			if err != nil {
				t.Fatal(err)
			}
			a, _ := canonjson.Canonicalize(raw)
			b, _ := canonjson.Canonicalize(encoded)
			if !bytes.Equal(a, b) {
				t.Fatalf("generated round trip changed example: %s", encoded)
			}
		})
	}
}

func TestD106_ProducerRetainsAuthoritativeSnapshot(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	if rep := mustReport(t, r, t.Context()); !rep.TcEvidence.IsNull() {
		t.Fatal("no read must be null")
	}
	m := *r.eng.appliedModel
	// This executor emulates a kernel; translate the manual fixture's sample to
	// the real CLOCK_MONOTONIC domain used by enforce's installation/verifier.
	now, err := enforce.MonotonicNow()
	if err != nil {
		t.Fatal(err)
	}
	for i := range m.Leases {
		d := now + *m.Leases[i].DeadlineMono - m.NowMono
		m.Leases[i].DeadlineMono = &d
	}
	m.NowMono = now
	x := &renderedExecutor{}
	x.model = m
	plane := SystemPlane{Executor: x}
	start := time.Now().UTC()
	res, err := plane.Apply(t.Context(), m, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := res.Evidence.TCReadbackEvidence()
	if want == nil || !want.Verified || want.ReadAt.Before(start) {
		t.Fatal("missing Apply read", want)
	}
	r.eng.retainTCReadback(res.Evidence)
	rep := mustReport(t, r, t.Context())
	got, err := rep.TcEvidence.Get()
	if err != nil {
		t.Fatal(err)
	}
	assertTCContract(t, got)
	if !got.ReadAt.Equal(want.ReadAt) || got.CandidateGeneration != want.CandidateGeneration {
		t.Fatal("read relabeled")
	}
	// A complete clean verifier is not permission to ignore an outer failure.
	p := &evidencePlane{memPlane: r.plane, ev: res.Evidence, err: os.ErrPermission}
	r.eng.plane = p
	if r.eng.verifyServingModel(t.Context(), m, res.Plan.Assignments) || r.eng.serving {
		t.Fatal("outer failure served")
	}
	rep = mustReport(t, r, t.Context())
	got, _ = rep.TcEvidence.Get()
	if !got.Verified || !got.ReadAt.Equal(want.ReadAt) || rep.MemberForwardingEnabled {
		t.Fatal("verifier/engine outcomes conflated")
	}
	x.foreignTC = true
	ev, err := plane.VerifyServing(t.Context(), m, res.Plan.Assignments)
	if err == nil {
		t.Fatal("foreign filter accepted")
	}
	p.ev, p.err = ev, err
	r.eng.verifyServingModel(t.Context(), m, res.Plan.Assignments)
	failed := mustReport(t, r, t.Context())
	rejected, _ := failed.TcEvidence.Get()
	assertTCContract(t, rejected)
	if rejected.Verified || rejected.Rejected.IsNull() {
		t.Fatal("rejection lost")
	}
	history := r.eng.containment.ReadAt
	x.foreignTC = false
	ev, err = plane.VerifyServing(t.Context(), m, res.Plan.Assignments)
	if err != nil {
		t.Fatal(err)
	}
	p.ev, p.err = ev, nil
	r.eng.verifyServingModel(t.Context(), m, res.Plan.Assignments)
	clean := mustReport(t, r, t.Context())
	got, _ = clean.TcEvidence.Get()
	assertTCContract(t, got)
	if !got.Verified || !got.ErrorCode.IsNull() || !got.Rejected.IsNull() || !got.ReadAt.After(rejected.ReadAt) {
		t.Fatal("clean live read missing")
	}
	found := false
	for _, row := range clean.Errors {
		if row.Stage == "tc_evidence" {
			found = true
			if !row.ObservedAt.Equal(history) {
				t.Fatal("historical time changed")
			}
		}
	}
	if !found {
		t.Fatal("historical rejection discarded")
	}
	r.eng.retainTCReadback(enforce.ServingEvidence{})
	if !r.eng.tcReadback.ReadAt.Equal(got.ReadAt) {
		t.Fatal("no-read attempt erased actual read")
	}
	requireObservationSchema(t, clean)
}

package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func omissionRig(t *testing.T) *rig {
	t.Helper()
	r := newRig(t)
	r.srv.SetClock(r.clock.Wall)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	return r
}

func omittedChallenge(t *testing.T, r *rig, reason string) healthattest.Challenge {
	t.Helper()
	return issueChallenge(t, r, func(in *healthattest.Input) {
		in.ProbeOmitted = &healthattest.ProbeOmission{PeerID: "private-probe-peer-100.64.8.9", Reason: reason}
	})
}

func reportedTunnel(t *testing.T, r *rig) api.NodeCheck {
	t.Helper()
	report, err := r.eng.report(context.Background(), r.clock.Wall())
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the observation JSON boundary, not just the in-memory decision.
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	check, ok := checkBy(report.Checks, api.NodeCheckNameTunnelProbe)
	if !ok {
		t.Fatal("missing tunnel check")
	}
	return check
}

func TestUPD05_ProbeOmissionDetailAndAttestation(t *testing.T) {
	for _, reason := range []string{"malformed", "denied_destination", "node_source", "node_destination", "control_destination", "resolver_destination", "agent_config_unavailable", "node_addresses_unavailable", "control_addresses_unavailable", "resolver_addresses_unavailable"} {
		t.Run(reason, func(t *testing.T) {
			r := omissionRig(t)
			ch := omittedChallenge(t, r, reason)
			stepOK(t, r)
			check := reportedTunnel(t, r)
			att, _, err := healthattest.ReadAttestation(r.eng.cfg.JournalDir)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("detail=%v status=%s attestation=%s/%q", detailValue(check), check.Status, att.Checks.TunnelProbe.Status, att.Checks.TunnelProbe.Reason)
			if check.Detail == nil || *check.Detail != "probe_omitted: "+reason || len(*check.Detail) > maxCheckDetail {
				t.Errorf("omission detail = %q", detailValue(check))
			}
			if check.Status != api.HealthCheckStatusNotRun || !check.ErrorCode.IsNull() || r.eng.input().ProbeRan {
				t.Errorf("omitted probe counted as run: %+v", check)
			}
			if r.eng.decision.DataHealth != api.DataHealthUnknown || r.eng.serving {
				t.Error("omission claimed healthy/serving")
			}
			if att.Checks.TunnelProbe.Status != api.HealthCheckStatusNotRun || att.Checks.TunnelProbe.Reason != "probe refused" {
				t.Errorf("omission attestation = %+v", att.Checks.TunnelProbe)
			}
			if got := healthattest.Validate(ch, att, r.eng.trustedNow(), ch.Components); got != healthattest.ReasonNotRun {
				t.Errorf("omission binding validation = %q", got)
			}
			if strings.Contains(r.log.String(), ch.ProbeOmitted.PeerID) || strings.Contains(detailValue(check), "100.64.") {
				t.Error("omission leaked peer/address")
			}
			for _, call := range r.plane.calls {
				if call.model.Probe != nil {
					t.Fatal("omitted probe installed an exception")
				}
			}
			in := r.eng.input()
			in.BaselineOK = false
			failed, _ := checkBy(Decide(in).Checks, api.NodeCheckNameTunnelProbe)
			if failed.Status != api.HealthCheckStatusFailed || codeOf(t, failed) != api.ErrorCodeISOLATIONBASELINEMISSING || detailValue(failed) != "probe_omitted: "+reason {
				t.Fatalf("omission hid baseline failure: %+v", failed)
			}
		})
	}
}

type omittedNoRunProbe struct{}

func (omittedNoRunProbe) Probe(context.Context, ProbeRequest) ProbeResult {
	return ProbeResult{Check: api.HealthCheckStatusNotRun, Health: api.DataHealthUnknown, NetbirdVersion: "0.76.2"}
}

func TestUPD05_ProbeOmissionAuthority(t *testing.T) {
	for _, mode := range []string{"answer_only", "boot", "not_run", "future", "expired", "untrusted"} {
		t.Run(mode, func(t *testing.T) {
			r := omissionRig(t)
			ch := omittedChallenge(t, r, "node_source")
			switch mode {
			case "not_run":
				r.eng.prober = omittedNoRunProbe{}
			case "future":
				ch.IssuedAt = r.clock.Wall().Add(time.Hour)
				ch.ExpiresAt = ch.IssuedAt.Add(45 * time.Second)
			case "expired":
				ch.IssuedAt = r.clock.Wall().Add(-time.Minute)
				ch.ExpiresAt = r.clock.Wall().Add(-time.Second)
			case "untrusted":
				r.eng.anchor.Trusted = false
			}
			if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, ch); err != nil {
				t.Fatal(err)
			}
			if mode == "boot" {
				if err := r.eng.Boot(context.Background()); err != nil {
					t.Fatal(err)
				}
			} else {
				// A cached overlay result after issued_at is insufficient: this
				// new omission must be read and bound before answering.
				r.eng.answerHealth(context.Background())
			}
			att, _, err := healthattest.ReadAttestation(r.eng.cfg.JournalDir)
			if mode == "future" || mode == "expired" || mode == "untrusted" {
				if !os.IsNotExist(err) || r.eng.input().ProbeOmitted != "" {
					t.Fatalf("inactive omission recorded or answered: %v", err)
				}
				return
			}
			if err != nil || att.Checks.TunnelProbe.Status != api.HealthCheckStatusNotRun || att.Checks.TunnelProbe.Reason != "probe refused" {
				t.Fatalf("omission answer: %+v err=%v", att.Checks.TunnelProbe, err)
			}
			if got := healthattest.Validate(ch, att, r.clock.Wall(), ch.Components); got != healthattest.ReasonNotRun {
				t.Fatalf("validation: %s", got)
			}
		})
	}
}

func detailValue(c api.NodeCheck) string {
	if c.Detail == nil {
		return ""
	}
	return *c.Detail
}

func TestUPD05_ProbeOmissionSurvivesConsumption(t *testing.T) {
	for _, mark := range []bool{false, true} {
		t.Run(map[bool]string{false: "remove", true: "mark"}[mark], func(t *testing.T) {
			r := omissionRig(t)
			ch := omittedChallenge(t, r, "denied_destination")
			stepOK(t, r)
			first := reportedTunnel(t, r)
			if detailValue(first) != "probe_omitted: denied_destination" {
				t.Fatalf("initial detail %q", detailValue(first))
			}
			var err error
			if mark {
				err = healthattest.MarkConsumed(r.eng.cfg.JournalDir, ch)
			} else {
				err = healthattest.Consume(r.eng.cfg.JournalDir)
			}
			if err != nil {
				t.Fatal(err)
			}
			r.clock.Advance(5 * time.Second)
			stepOK(t, r)
			retained := reportedTunnel(t, r)
			if detailValue(retained) != detailValue(first) || !retained.ObservedAt.Equal(first.ObservedAt) || retained.Status != first.Status {
				t.Fatalf("consumption lost evidence: first=%+v next=%+v", first, retained)
			}
			// An ordinary overlay-only challenge is not newer synthetic evidence.
			issueChallenge(t, r, nil)
			stepOK(t, r)
			if got := reportedTunnel(t, r); detailValue(got) != detailValue(first) || !got.ObservedAt.Equal(first.ObservedAt) {
				t.Fatal("ordinary overlay check replaced omission")
			}
			omittedChallenge(t, r, "node_addresses_unavailable")
			stepOK(t, r)
			second := reportedTunnel(t, r)
			if detailValue(second) != "probe_omitted: node_addresses_unavailable" || !second.ObservedAt.After(first.ObservedAt) {
				t.Fatalf("later omission did not replace evidence: %+v", second)
			}
			// Replaying the older omission cannot regress the retained evidence.
			if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, ch); err != nil {
				t.Fatal(err)
			}
			stepOK(t, r)
			if got := reportedTunnel(t, r); detailValue(got) != detailValue(second) || !got.ObservedAt.Equal(second.ObservedAt) {
				t.Fatal("older omission replaced newer evidence")
			}
			r.clock.Advance(5 * time.Second)
			issueChallenge(t, r, func(in *healthattest.Input) { in.Probe = labProbe() })
			stepOK(t, r)
			got := reportedTunnel(t, r)
			if strings.HasPrefix(detailValue(got), "probe_omitted:") || !strings.Contains(detailValue(got), "Egress probe") || got.Status != api.HealthCheckStatusPassed {
				t.Fatalf("later probe did not replace omission: %+v", got)
			}
		})
	}
}

func TestUPD05_MatchingChallengeOmissionReader(t *testing.T) {
	for _, tc := range []string{"valid", "both", "unknown", "invalid_reason", "foreign"} {
		t.Run(tc, func(t *testing.T) {
			r := omissionRig(t)
			ch := omittedChallenge(t, r, "node_source")
			raw, err := json.Marshal(ch)
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatal(err)
			}
			switch tc {
			case "both":
				body["probe"] = labProbe()
			case "unknown":
				body["unrelated_unknown_member"] = true
			case "invalid_reason":
				body["probe_omitted"].(map[string]any)["reason"] = "private-probe-peer-100.64.8.9"
			case "foreign":
				body["release_id"] = "019a0000-0000-7000-8000-0000000000ff"
			}
			raw, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(r.eng.cfg.JournalDir, healthattest.ChallengeFile), raw, 0o600); err != nil {
				t.Fatal(err)
			}
			_, matches := r.eng.matchingChallenge()
			if matches != (tc == "valid") {
				t.Fatalf("matches=%v", matches)
			}
			stepOK(t, r)
			att, _, err := healthattest.ReadAttestation(r.eng.cfg.JournalDir)
			if tc == "valid" {
				if err != nil || att.Nonce != ch.Nonce {
					t.Fatalf("valid omission not answered: %v", err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("rejected challenge answered: %v", err)
			}
		})
	}
}

func TestUPD05_ProbeOmissionRewriteDoesNotAttach(t *testing.T) {
	r := omissionRig(t)
	ch := omittedChallenge(t, r, "node_source")
	snap := &snapProbe{verProbe: verProbe{version: "0.76.2"}}
	r.eng.prober = snap
	snap.during = func() {
		edited := ch
		edited.ProbeOmitted = &healthattest.ProbeOmission{PeerID: ch.ProbeOmitted.PeerID, Reason: "node_destination"}
		if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, edited); err != nil {
			t.Fatal(err)
		}
	}
	stepOK(t, r)
	if _, err := os.Lstat(attestPath(r)); !os.IsNotExist(err) {
		t.Fatalf("rewritten omission was answered: %v", err)
	}
	if strings.HasPrefix(detailValue(reportedTunnel(t, r)), "probe_omitted:") {
		t.Fatal("rewritten omission attached to old probe")
	}
}

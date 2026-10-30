package agent

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/dns"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestReviewD064_ProbeDoesNotTakeUpdateLock(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	enrollBoot(t, r)
	lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	ch := probeChallenge(t, r, labProbe(), 0)
	stepOK(t, r)
	requireAttest(t, r, ch)
	if countKind(r.plane.calls, "baseline_model") != 2 || r.plane.last.Probe != nil {
		t.Fatalf("locked updater prevented install/clear: %+v", r.plane.calls)
	}
	if r.eng.serving || r.plane.applies != 0 || !hasCode(r.eng.decision.Errors, api.ErrorCodeUPDATEBUSY) {
		t.Fatalf("serving apply ignored update lock: serving=%v applies=%d errors=%+v", r.eng.serving, r.plane.applies, r.eng.decision.Errors)
	}
	if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonUpdateInProgress) {
		t.Fatalf("missing update_in_progress: %v", r.eng.decision.Reasons)
	}
	if other, err := nodelock.TryAcquire(r.eng.cfg.LockPath); !errors.Is(err, nodelock.ErrBusy) {
		if other != nil {
			other.Release()
		}
		t.Fatalf("probe released updater lock: %v", err)
	}
}

func TestReviewD063_HealthRechecksBaseline(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		clear, baseline, boot bool
	}{
		{"clear_only", true, false, false},
		{"enforce_only", false, true, false},
		{"boot_clear", true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			p := &failAfterProbePlane{}
			r.eng.plane = p
			r.eng.prober = &verProbe{version: "0.76.2"}
			r.srv.SetPublication(api.PublicationIntentUnpublished)
			enrollBoot(t, r)
			stepOK(t, r)
			holdUpdateLock(t, r)
			p.failClear, p.failBaseline = tc.clear, tc.baseline
			ch := probeChallenge(t, r, labProbe(), 0)
			if tc.boot {
				r.eng.answerHealthOnBoot(context.Background())
			} else {
				stepOK(t, r)
			}
			att := requireAttest(t, r, ch)
			if att.Checks.TunnelProbe.Status != api.HealthCheckStatusFailed || r.eng.serving {
				t.Fatalf("unsafe probe passed: %+v serving=%v", att.Checks, r.eng.serving)
			}
			if tc.clear && r.eng.lastProbe.Detail != "probe_clear_failed" {
				t.Fatalf("lost cleanup failure: %+v", r.eng.lastProbe)
			}
			if !r.eng.baselineOK && att.Checks.InternetOnly.Status != api.HealthCheckStatusFailed {
				t.Fatalf("baseline passed: %+v", att.Checks)
			}
			if tc.baseline {
				for _, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameTunnelProbe} {
					if got, _ := decisionStatus(r.eng.decision, name); got != api.HealthCheckStatusFailed {
						t.Fatalf("step kept stale %s=%s", name, got)
					}
				}
			}
		})
	}
}

func TestReviewD063_ProbeChallengeInstalledOnce(t *testing.T) {
	for _, writeFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "answered", true: "answer_write_failed"}[writeFails], func(t *testing.T) {
			r := newRig(t)
			r.eng.prober = &verProbe{version: "0.76.2"}
			r.srv.SetPublication(api.PublicationIntentUnpublished)
			enrollBoot(t, r)
			stepOK(t, r)
			holdUpdateLock(t, r)
			ch := probeChallenge(t, r, labProbe(), 0)
			if writeFails {
				if err := os.Mkdir(attestPath(r), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			stepOK(t, r)
			if !writeFails {
				requireAttest(t, r, ch)
			}
			n := countKind(r.plane.calls, "baseline_model")
			if n != 2 {
				t.Fatalf("expected one install and clear, got %d", n)
			}
			for range 3 {
				stepOK(t, r)
			}
			if countKind(r.plane.calls, "baseline_model") != n {
				t.Fatal("challenge was reinstalled on a later step")
			}
			if writeFails {
				if err := os.Remove(attestPath(r)); err != nil {
					t.Fatal(err)
				}
				stepOK(t, r)
				requireAttest(t, r, ch)
			}
			if r.plane.last.Probe != nil {
				t.Fatal("probe remained after answer")
			}
		})
	}
}

func TestReviewD063_ProbeIssuanceBounds(t *testing.T) {
	r := newRig(t)
	enrollBoot(t, r)
	stepOK(t, r)
	now := r.eng.trustedNow()
	if maxProbeTargets != enforce.ProbeMaxDestinations {
		t.Fatal("probe destination limits drifted")
	}
	for _, tc := range []struct {
		name            string
		issued, expires time.Time
		allowed         bool
	}{
		{"future", now.Add(time.Second), now.Add(45 * time.Second), false},
		{"at_issue", now, now.Add(45 * time.Second), false},
		{"at_expiry", now.Add(-45 * time.Second), now, false},
		{"over_max", now.Add(-time.Second), now.Add(300 * time.Second), false},
		{"max_from_issuance", now.Add(-100 * time.Second), now.Add(200 * time.Second), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch := healthattest.Challenge{IssuedAt: tc.issued, ExpiresAt: tc.expires, Probe: labProbe()}
			exc, ok := r.eng.challengeProbe(ch, now)
			if ok != tc.allowed {
				t.Fatalf("allowed=%v", ok)
			}
			if ok && (exc.ExpiresAt != tc.expires || exc.TimeoutCeilSeconds != 200) {
				t.Fatalf("window extended: %+v", exc)
			}
		})
	}
	ch := healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: labProbe()}
	for range maxProbeTargets {
		ch.Probe.Destinations = append(ch.Probe.Destinations, ch.Probe.Destinations[0])
	}
	if _, ok := r.eng.challengeProbe(ch, now); ok {
		t.Fatal("excess destinations allowed")
	}
	ch.Probe = labProbe()
	// Rendering only: no traffic is sent to these test socket values.
	r.eng.desired = &api.DesiredState{DeviceBindings: []api.DeviceBinding{{PeerId: "member", AllowedIpv4: []string{"1.2.3.4"}}}}
	if _, ok := r.eng.challengeProbe(ch, now); ok {
		t.Fatal("destination overlapped member binding")
	}
	if !r.eng.memberCollision("probe", netip.MustParseAddr("1.2.3.4")) {
		t.Fatal("member collision missed bare host")
	}
}

func TestReviewD064_ProbeRequiresSameLiveChallenge(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	holdUpdateLock(t, r)
	ch := probeChallenge(t, r, labProbe(), 0)
	if err := os.Remove(filepath.Join(r.eng.cfg.JournalDir, healthattest.ChallengeFile)); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.eng.installProbeException(context.Background(), ch); ok {
		t.Fatal("installed without live authority")
	}
	replacement := probeChallenge(t, r, labProbe(), 0)
	if _, ok := r.eng.installProbeException(context.Background(), ch); ok {
		t.Fatal("installed replaced challenge")
	}
	// Replace the challenge while answering: the old probe cannot attest the new issuance.
	r.eng.probeAt = time.Time{}
	r.eng.prober = &snapProbe{verProbe: verProbe{version: "0.76.2"}, during: func() {
		ch.Probe.SourceV4 = "100.64.8.2/32"
		if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, ch); err != nil {
			t.Fatal(err)
		}
	}}
	r.eng.answerHealth(context.Background())
	if _, err := os.Stat(attestPath(r)); !os.IsNotExist(err) {
		t.Fatalf("answered replaced challenge %s: %v", replacement.Nonce, err)
	}
	if r.plane.last.Probe != nil {
		t.Fatal("replacement left exception installed")
	}
}

func TestReviewD063_DNSApplierEvidenceMustMatch(t *testing.T) {
	for _, tc := range []struct {
		name, hash, gen string
		pass            bool
	}{
		{"digest_mismatch", strings.Repeat("de", 32), "", false},
		{"generation_mismatch", "", "1", false},
		{"normalized_match", "MATCH", "MATCH", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			r.eng.prober = &verProbe{version: "0.76.2"}
			body := []byte(exampleResolverBody)
			ref := exampleRef(body)
			hash, gen := tc.hash, tc.gen
			if hash == "MATCH" {
				hash = "  " + strings.ToUpper(ref.ContentSha256) + "  "
			}
			if gen == "MATCH" {
				gen = "  " + ref.Generation.String() + "  "
			}
			r.eng.dnsApply = &scriptDNS{ret: dns.Evidence{DocumentHash: hash, PolicyGeneration: gen}}
			r.srv.PutResolverBody(ref.DocumentId, body)
			r.srv.SetResolverRef(&ref)
			enrollBoot(t, r)
			ch := issueChallenge(t, r, nil)
			stepOK(t, r)
			att := requireAttest(t, r, ch)
			if (att.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed) != tc.pass {
				t.Fatalf("health %+v", att.Checks.DNSPolicy)
			}
			if !tc.pass {
				if r.eng.appliedDNS.ok || r.eng.dnsEv.Outcome != api.DnsPolicyEvidenceOutcomeFailed {
					t.Fatalf("mismatch was applied: %+v", r.eng.dnsEv)
				}
				if code, err := r.eng.dnsEv.ErrorCode.Get(); err != nil || code != api.ErrorCodeVALIDATIONFAILED {
					t.Fatalf("code=%s err=%v", code, err)
				}
			} else if r.eng.appliedDNS.hash != ref.ContentSha256 {
				t.Fatalf("stored unverified hash: %+v", r.eng.appliedDNS)
			}
		})
	}
}

func TestReviewD061_CredentialOwnershipAndInode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(path, []byte("TEST ONLY"), 0o600); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	st := *fi.Sys().(*syscall.Stat_t)
	st.Uid, st.Gid, st.Nlink = 0, 0, 1
	base := &statOverrideFileInfo{FileInfo: fi, st: &st}
	if !credentialMode(base) {
		t.Fatal("root:root single-link file rejected")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*syscall.Stat_t)
	}{
		{"group", func(s *syscall.Stat_t) { s.Gid = 1 }},
		{"owner", func(s *syscall.Stat_t) { s.Uid = 1 }},
		{"hardlink", func(s *syscall.Stat_t) { s.Nlink = 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := st
			tc.mutate(&other)
			if credentialMode(&statOverrideFileInfo{FileInfo: fi, st: &other}) {
				t.Fatal("unsafe credentials accepted")
			}
		})
	}
	other := st
	other.Ino++
	if sameStat(base, &statOverrideFileInfo{FileInfo: fi, st: &other}) {
		t.Fatal("replaced inode accepted")
	}
	other = st
	other.Dev++
	if sameStat(base, &statOverrideFileInfo{FileInfo: fi, st: &other}) {
		t.Fatal("same inode on another device accepted")
	}
}

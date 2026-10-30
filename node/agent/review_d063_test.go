package agent

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/dns"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type failAfterProbePlane struct {
	memPlane
	failClear    bool
	failBaseline bool
}

func (p *failAfterProbePlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	if p.failBaseline {
		p.calls = append(p.calls, planeCall{kind: "baseline_failed"})
		return &enforce.CheckError{
			Reason: enforce.ReasonIsolationBaselineMissing,
			Detail: "baseline altered during probe window",
		}
	}
	return p.memPlane.Baseline(ctx, dp)
}

func (p *failAfterProbePlane) BaselineModel(ctx context.Context, model enforce.Model) error {
	if model.Probe == nil && p.failClear {
		p.calls = append(p.calls, planeCall{kind: "baseline_model_clear_failed", model: model})
		return &enforce.CheckError{
			Reason: enforce.ReasonProbeUnbounded,
			Detail: "VerifyBaseline failed after probe",
		}
	}
	return p.memPlane.BaselineModel(ctx, model)
}

func TestReviewD063_Agent_BaselineFailureAfterProbeStillAttestsPassed(t *testing.T) {
	// H-1: cleanup and subsequent baseline verification failures must fail both health checks.
	r := newRig(t)
	fp := &failAfterProbePlane{}
	r.eng.plane = fp
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)

	// Simulate baseline tampering / VerifyBaseline failure right after the probe runs.
	fp.failClear = true
	fp.failBaseline = true
	holdUpdateLock(t, r)
	ch := probeChallenge(t, r, labProbe(), 0)
	stepOK(t, r)

	if r.eng.baselineOK {
		t.Fatal("expected e.baselineOK to be false after baseline verification failed")
	}
	att := requireAttest(t, r, ch)
	if att.Checks.InternetOnly.Status != api.HealthCheckStatusFailed || att.Checks.TunnelProbe.Status != api.HealthCheckStatusFailed {
		t.Fatalf("health attestation passed after VerifyBaseline failed post-probe: internet_only=%+v tunnel_probe=%+v (baselineOK=%v)",
			att.Checks.InternetOnly, att.Checks.TunnelProbe, r.eng.baselineOK)
	}
}

func TestReviewD063_Agent_ProbeUsesStaleServingDecisionWhenLatchBlocks(t *testing.T) {
	// H-2: a new forwarding latch withdraws members before the challenge probe runs.
	r := newRig(t)
	devID := uuid.Must(uuid.NewV7())
	memID := uuid.Must(uuid.NewV7())
	r.srv.SetFixture(fakecontrol.Fixture{
		ReleaseID:   uuid.MustParse("019a0000-0000-7000-8000-0000000000c1"),
		Publication: api.PublicationIntentPublished,
		Generation:  1,
		TTL:         time.Hour,
		Capacity:    &api.NodeCapacity{UpBps: 200_000_000, DownBps: 200_000_000},
		Bindings: []api.DeviceBinding{{
			AllowedIpv4:     []string{"100.64.1.9/32"},
			AllowedIpv6:     []string{},
			BindingRevision: decstr.U64(1),
			DeviceId:        devID,
			MemberId:        memID,
			PeerId:          "member-peer",
			ValidFrom:       time.Now().Add(-time.Hour).UTC(),
			ValidUntil:      time.Now().Add(time.Hour).UTC(),
		}},
	})
	r.srv.BumpGeneration()
	var probeRanWithServingBindings bool
	r.eng.prober = &snapProbe{
		verProbe: verProbe{version: "0.76.2"},
		during: func() {
			last := r.plane.calls[len(r.plane.calls)-1]
			if last.kind == "apply" && len(last.model.Bindings) > 0 {
				probeRanWithServingBindings = true
			}
		},
	}
	enrollBoot(t, r)
	stepOK(t, r)
	if !r.eng.serving {
		t.Fatalf("expected step 1 to serve, got reasons=%v", r.eng.decision.Reasons)
	}

	// Updater engages the §13.9 forwarding.blocked latch and writes a health challenge with a probe.
	if err := os.MkdirAll(r.eng.cfg.JournalDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked"), []byte("switch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	holdUpdateLock(t, r)
	probeChallenge(t, r, labProbe(), 0)
	stepOK(t, r)

	if probeRanWithServingBindings {
		t.Fatalf("probe exception called Apply with stale serving model (Bindings=%d) while forwarding.blocked latch was active", len(r.plane.last.Bindings))
	}
}

func TestReviewD063_Agent_ProbeInstalledOutsideWindowAndRepeatedEveryStep(t *testing.T) {
	// M-1: a future issuance must not authorize an exception. Repeated-step coverage is in TestReviewD063_ProbeChallengeInstalledOnce.
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)

	now := r.clock.Wall().UTC()
	futureCh := issueChallenge(t, r, func(in *healthattest.Input) {
		in.Now = now.Add(2 * time.Minute)
		in.TTL = 45 * time.Second
	})
	futureCh.Probe = labProbe()
	if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, futureCh); err != nil {
		t.Fatal(err)
	}
	before := len(r.plane.calls)
	stepOK(t, r)
	for _, c := range r.plane.calls[before:] {
		if c.model.Probe != nil {
			t.Fatalf("installed probe exception at %s before challenge IssuedAt %s (ExpiresAt=%s)",
				now.Format(time.RFC3339), futureCh.IssuedAt.Format(time.RFC3339), c.model.Probe.ExpiresAt.Format(time.RFC3339))
		}
	}
}

type resetOncePlane struct {
	memPlane
	pendingResets []enforce.Reset
	counters      map[string]enforce.CounterValue
}

func (p *resetOncePlane) Apply(ctx context.Context, model enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.memPlane.Apply(ctx, model, prev)
	if err != nil {
		return res, err
	}
	res.Counters = cloneCounters(p.counters)
	if len(p.pendingResets) > 0 {
		// Like enforce.Apply when state.Generation == 0 after an external nft reset:
		// only the FIRST Apply sees state.Generation == 0 before NftBatch restores
		// the forward chain and fvpn_generation comment in the kernel.
		res.Resets = p.pendingResets
		p.pendingResets = nil
	}
	return res, nil
}

func TestReviewD063_Agent_ServingProbeSwallowsExternalMeterReset(t *testing.T) {
	// M-4: the first apply sees the external reset and restores the generation marker. That reset must open exactly one epoch and an unbilled gap.
	r := newRig(t)
	rp := &resetOncePlane{}
	r.eng.plane = rp
	ks := &kernelSource{
		boot: r.eng.bootID.String(), bindings: r.eng.meterBindings,
		expect: func() bool { return r.eng.expectServing.Load() },
		read: func(context.Context) (enforce.State, error) {
			return enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Generation: 1, Counters: cloneCounters(rp.counters)}, nil
		},
	}
	r.eng.kernel = ks
	r.eng.meterSource = ks
	r.eng.prober = &verProbe{version: "0.76.2"}
	devID := uuid.Must(uuid.NewV7())
	memID := uuid.Must(uuid.NewV7())
	r.srv.SetFixture(fakecontrol.Fixture{
		ReleaseID:   uuid.MustParse("019a0000-0000-7000-8000-0000000000c1"),
		Publication: api.PublicationIntentPublished,
		Generation:  1,
		TTL:         time.Hour,
		Capacity:    &api.NodeCapacity{UpBps: 200_000_000, DownBps: 200_000_000},
		Bindings: []api.DeviceBinding{{
			AllowedIpv4:     []string{"100.64.1.9/32"},
			AllowedIpv6:     []string{},
			BindingRevision: decstr.U64(1),
			DeviceId:        devID,
			MemberId:        memID,
			PeerId:          "member-peer",
			ValidFrom:       time.Now().Add(-time.Hour).UTC(),
			ValidUntil:      time.Now().Add(time.Hour).UTC(),
		}},
	})
	r.srv.BumpGeneration()
	enrollBoot(t, r)
	stepOK(t, r)
	r.manualUsage(t)

	upCounter := "u_" + strings.ReplaceAll(devID.String(), "-", "")
	r.eng.counters = map[string]enforce.CounterValue{
		upCounter: {Packets: 50, Bytes: 5000},
	}
	rp.counters = cloneCounters(r.eng.counters)
	if err := r.eng.meter.Sample(context.Background()); err != nil {
		t.Fatal(err)
	}
	before, err := r.eng.usage.Head(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rp.pendingResets = []enforce.Reset{{
		Name:     upCounter,
		Previous: 5000,
		Current:  5000,
		Reason:   enforce.ReasonMeterEpoch,
		Detail:   "ruleset was reset outside the agent (table, forward chain, or generation)",
	}}

	probeChallenge(t, r, labProbe(), 0)
	stepOK(t, r)

	after, err := r.eng.usage.Head(context.Background())
	if err != nil || after.OpenEpoch == before.OpenEpoch || after.OpenEpoch == uuid.Nil {
		t.Fatalf("serving probe swallowed reset: before=%s after=%s err=%v", before.OpenEpoch, after.OpenEpoch, err)
	}
	if !hasCode(r.eng.decision.Errors, api.ErrorCodeMETEREPOCHCHANGE) {
		t.Fatalf("missing reset evidence: %+v", r.eng.decision.Errors)
	}
	gap, err := r.eng.usage.Event(context.Background(), before.NextSeq)
	if err != nil {
		t.Fatal(err)
	}
	var ev api.UsageEvent
	if err := json.Unmarshal(gap.Payload, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Quality != api.UsageQualityGap || ev.Gap == nil || ev.Gap.Reason != api.UsageGapReasonCounterDiscontinuity || billed(ev) != 0 {
		t.Fatalf("reset did not record an unbilled discontinuity: %+v", ev)
	}
	stepOK(t, r)
	quiet, err := r.eng.usage.Head(context.Background())
	if err != nil || quiet.OpenEpoch != after.OpenEpoch {
		t.Fatalf("reset opened another epoch: %+v %v", quiet, err)
	}

}

func TestReviewD063_Agent_DNSPolicyPassesOnApplierDigestMismatch(t *testing.T) {
	// H-4: verified evidence for another digest or generation cannot attest DNS success.
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	body := []byte(exampleResolverBody)
	ref := exampleRef(body)
	wrongDigest := strings.Repeat("de", 32)
	if wrongDigest == strings.ToLower(ref.ContentSha256) {
		t.Fatal("test digest collided with ref digest")
	}
	r.eng.dnsApply = &scriptDNS{
		ret: dns.Evidence{
			DocumentHash:     wrongDigest,
			PolicyGeneration: "1",
		},
	}
	r.srv.PutResolverBody(ref.DocumentId, body)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	ch := issueChallenge(t, r, nil)
	stepOK(t, r)

	att := requireAttest(t, r, ch)
	if att.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed {
		t.Fatalf("dns_policy attested passed when applier returned digest %s != expected %s (generation %q != %q)",
			wrongDigest, ref.ContentSha256, "1", ref.Generation)
	}
}

func TestReviewD063_Agent_JournalSnapshotOverwritesReleaseIDWithUnverifiedOrNullPointer(t *testing.T) {
	// M-5: a claimed job cannot erase or replace the running release via journal pointers.
	t.Run("blank_journal_current_release_wipes_configured_release_to_null", func(t *testing.T) {
		r := newRig(t)
		jobID := uuid.MustParse("019a0000-0000-7000-8000-0000000000a9")
		targetRel := uuid.MustParse("019a0000-0000-7000-8000-0000000000b9")
		path := withJournal(t, filepath.Join(t.TempDir(), "claimed-no-pointers"), func(j *updatejournal.Journal) {
			seedUpdate(t, j, jobID, decstr.U64(7), targetRel.String())
		})
		r.eng.cfg.UpdaterJournal = path
		rep := mustReport(t, r, context.Background())
		got, err := rep.CurrentReleaseId.Get()
		if err != nil || got != r.eng.cfg.ReleaseUUID() {
			t.Fatalf("claimed job with unset journal pointer wiped CurrentReleaseId: got=%v err=%v, want %s",
				got, err, r.eng.cfg.ReleaseUUID())
		}
	})

	t.Run("journal_current_release_claims_release_node_never_ran", func(t *testing.T) {
		r := newRig(t)
		jobID := uuid.MustParse("019a0000-0000-7000-8000-0000000000aa")
		targetRel := uuid.MustParse("019a0000-0000-7000-8000-0000000000bb")
		neverRanRel := uuid.MustParse("019a0000-0000-7000-8000-0000000000ff")
		path := withJournal(t, filepath.Join(t.TempDir(), "spoofed-pointer"), func(j *updatejournal.Journal) {
			if err := j.SetPointers(context.Background(), updatejournal.Pointers{
				CurrentRelease: neverRanRel.String(),
				LastGood:       neverRanRel.String(),
			}); err != nil {
				t.Fatal(err)
			}
			seedUpdate(t, j, jobID, decstr.U64(8), targetRel.String())
		})
		r.eng.cfg.UpdaterJournal = path
		rep := mustReport(t, r, context.Background())
		got, err := rep.CurrentReleaseId.Get()
		if err == nil && got == neverRanRel {
			t.Fatalf("observation claimed CurrentReleaseId=%s from journal meta that equals neither running binary (%s) nor job target (%s)",
				got, r.eng.cfg.ReleaseUUID(), targetRel)
		}
	})
}

func TestReviewD061_Agent_HardLinkedCredentialFileNotRejected(t *testing.T) {
	// L-1: mode 0600 and root ownership do not make a multiply linked credential safe.
	dir := t.TempDir()
	primary := filepath.Join(dir, "creds.json")
	worldDir := filepath.Join(dir, "world")
	if err := os.MkdirAll(worldDir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(primary, []byte(`{"username":"e2e","password":"pw-hardlink-secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	hardlink := filepath.Join(worldDir, "linked-creds.json")
	if err := os.Link(primary, hardlink); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(primary)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		t.Fatal("missing Stat_t")
	}
	if st.Nlink < 2 {
		t.Fatalf("expected Nlink >= 2, got %d", st.Nlink)
	}
	// Verify credentialMode with uid normalized to 0 (since tests run unprivileged):
	// if credentialMode checked st.Nlink == 1, a cloned Stat_t with Uid=0 and Nlink=2
	// would be rejected.
	if credentialModeAllowingTestUID(fi) {
		t.Fatalf("credentialMode accepted a hard-linked 0600 file with Nlink=%d", st.Nlink)
	}
}

// credentialModeAllowingTestUID substitutes root ownership so link-count
// behavior can be tested without root.
func credentialModeAllowingTestUID(fi os.FileInfo) bool {
	fake := *fi.Sys().(*syscall.Stat_t)
	fake.Uid = 0
	fake.Gid = 0
	return credentialMode(&statOverrideFileInfo{FileInfo: fi, st: &fake})
}

type statOverrideFileInfo struct {
	os.FileInfo
	st *syscall.Stat_t
}

func (s *statOverrideFileInfo) Sys() any { return s.st }

func TestReviewD063_Held_AgentProbeCrashAndUsageCounters(t *testing.T) {
	// Crash mid-probe: if the process crashes while the probe exception is installed
	// on the data plane (before defer clear() runs), the rendered ruleset bounds
	// every set element by timeout <= ProbeMaxSeconds, and a fresh agent Boot/Step
	// immediately restores the fail-closed baseline without the probe sets.
	r := newRig(t)
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	var crashedMidProbe bool
	r.eng.prober = &snapProbe{
		verProbe: verProbe{version: "0.76.2"},
		during: func() {
			crashedMidProbe = true
		},
	}
	enrollBoot(t, r)
	stepOK(t, r)
	holdUpdateLock(t, r)
	ch := probeChallenge(t, r, labProbe(), 45*time.Second)
	clearFn, installed := r.eng.installProbeException(context.Background(), ch)
	if !installed || clearFn == nil {
		t.Fatal("probe exception was not installed")
	}
	if r.plane.last.Probe == nil {
		t.Fatal("plane did not receive probe model")
	}
	script, err := enforce.RenderBaseline(r.plane.last, enforce.State{})
	if err != nil {
		t.Fatal(err)
	}
	seconds := r.plane.last.Probe.TimeoutCeilSeconds
	want := "timeout " + itoaU64(seconds) + "s"
	if seconds == 0 || seconds > 43 || !strings.Contains(script, "100.64.8.1 "+want) || !strings.Contains(script, "1.2.3.4 . tcp . 80 "+want) {
		t.Fatalf("mid-probe ruleset missing kernel element timeouts:\n%s", script)
	}
	_ = r.eng.prober.Probe(context.Background(), ProbeRequest{})
	// Simulate crash before clearFn() runs: next Boot() and Step() re-apply clean baseline.
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	stepOK(t, r)
	if !crashedMidProbe {
		t.Fatal("probe did not run")
	}
	if r.plane.last.Probe != nil {
		t.Fatalf("probe remained after recovery step: %+v", r.plane.last.Probe)
	}
}

func TestReviewD063_Held_HealthBeforeDNSApplyAndStaleETag(t *testing.T) {
	body1 := []byte(exampleResolverBody)
	ref1 := exampleRef(body1)

	compiled2, err := dnspublishCompileForTest("strict")
	if err != nil {
		t.Fatal(err)
	}
	ref2 := exampleRef(compiled2)
	ref2.DocumentId = uuid.MustParse("01900000-0000-7000-8000-000000000099")
	ref2.Generation = decstr.U64(9007199254740994)

	t.Run("downloaded_in_cache_but_not_yet_applied", func(t *testing.T) {
		r := newRig(t)
		r.eng.prober = &verProbe{version: "0.76.2"}
		r.srv.PutResolverBody(ref1.DocumentId, body1)
		r.srv.SetResolverRef(&ref1)
		enrollBoot(t, r)
		// Pre-populate the downloaded document in the local resolver cache before Apply runs.
		if err := r.eng.store.PutResolverCache(context.Background(), ref1.DocumentId.String(), ref1.ContentSha256, body1); err != nil {
			t.Fatal(err)
		}
		if err := r.eng.syncDesired(context.Background()); err != nil {
			t.Fatal(err)
		}
		ch := issueChallenge(t, r, nil)
		// Establish trusted time, then answer before applyResolver has executed.
		if err := r.eng.syncLease(context.Background()); err != nil {
			t.Fatal(err)
		}
		r.eng.answerHealth(context.Background())
		att := requireAttest(t, r, ch)
		if att.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed {
			t.Fatalf("dns_policy passed while document was only downloaded in cache: %+v", att.Checks.DNSPolicy)
		}
	})

	t.Run("applied_for_different_document_id", func(t *testing.T) {
		r := newRig(t)
		r.eng.prober = &verProbe{version: "0.76.2"}
		r.eng.dnsApply = &scriptDNS{ret: dns.Evidence{DocumentHash: ref1.ContentSha256}}
		r.srv.PutResolverBody(ref1.DocumentId, body1)
		r.srv.SetResolverRef(&ref1)
		enrollBoot(t, r)
		stepOK(t, r)
		if mustDNS(t, r.eng.decision.Checks).Status != api.HealthCheckStatusPassed {
			t.Fatal("first document did not apply")
		}
		// Control switches desired state to ref2, whose body is not yet available/applied.
		r.srv.SetResolverRef(&ref2)
		ch := issueChallenge(t, r, nil)
		stepOK(t, r)
		att := requireAttest(t, r, ch)
		if att.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed {
			t.Fatalf("dns_policy passed when only previous document_id was applied: %+v", att.Checks.DNSPolicy)
		}
	})

	t.Run("downloaded_with_sha256_mismatch", func(t *testing.T) {
		r := newRig(t)
		r.eng.prober = &verProbe{version: "0.76.2"}
		r.eng.dnsApply = &scriptDNS{}
		badRef := ref1
		badRef.ContentSha256 = strings.Repeat("00", 32)
		r.srv.PutResolverBody(badRef.DocumentId, body1)
		r.srv.SetResolverRef(&badRef)
		enrollBoot(t, r)
		ch := issueChallenge(t, r, nil)
		stepOK(t, r)
		att := requireAttest(t, r, ch)
		if att.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed {
			t.Fatalf("dns_policy passed on SHA-256 mismatch: %+v", att.Checks.DNSPolicy)
		}
	})
}

func dnspublishCompileForTest(profile string) ([]byte, error) {
	body := strings.Replace(exampleResolverBody, `"standard"`, `"`+profile+`"`, 1)
	return []byte(body), nil
}

func TestReviewD063_Held_JournalSnapshotHardening(t *testing.T) {
	const marker = "JOURNAL-SECRET-BYTES-DO-NOT-LEAK"

	cases := []struct {
		name  string
		setup func(t *testing.T, dir string) string
	}{
		{
			name: "huge_file_16mb",
			setup: func(t *testing.T, dir string) string {
				p := filepath.Join(dir, "huge-journal.db")
				buf := bytes.Repeat([]byte(marker), (16<<20)/len(marker))
				if err := os.WriteFile(p, buf, 0o600); err != nil {
					t.Fatal(err)
				}
				return p
			},
		},
		{
			name: "truncated_sqlite_file",
			setup: func(t *testing.T, dir string) string {
				p := filepath.Join(dir, "truncated-journal.db")
				if err := os.WriteFile(p, []byte("SQLite format 3\x00"+marker), 0o600); err != nil {
					t.Fatal(err)
				}
				return p
			},
		},
		{
			name: "fifo_named_pipe",
			setup: func(t *testing.T, dir string) string {
				p := filepath.Join(dir, "fifo-journal.db")
				if err := syscall.Mkfifo(p, 0o600); err != nil {
					t.Fatal(err)
				}
				return p
			},
		},
		{
			name: "symlink_to_device",
			setup: func(t *testing.T, dir string) string {
				p := filepath.Join(dir, "symlink-dev-null.db")
				if err := os.Symlink("/dev/null", p); err != nil {
					t.Fatal(err)
				}
				return p
			},
		},
		{
			name: "wrong_user_version",
			setup: func(t *testing.T, dir string) string {
				p := filepath.Join(dir, "wrong-version.db")
				db, err := sql.Open("sqlite", "file:"+p)
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				if _, err := db.Exec(`PRAGMA user_version = 99; CREATE TABLE leak(v TEXT); INSERT INTO leak VALUES ('` + marker + `');`); err != nil {
					t.Fatal(err)
				}
				return p
			},
		},
		{
			name: "locked_by_another_connection",
			setup: func(t *testing.T, dir string) string {
				p := withJournal(t, dir, func(*updatejournal.Journal) {})
				db, err := sql.Open("sqlite", "file:"+p+"?_pragma=journal_mode(WAL)")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = db.Close() })
				conn, err := db.Conn(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = conn.Close() })
				if _, err := conn.ExecContext(context.Background(), `BEGIN EXCLUSIVE; INSERT INTO meta(key, value) VALUES ('leak', '`+marker+`');`); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _, _ = conn.ExecContext(context.Background(), `ROLLBACK`) })
				return p
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			r.log.Reset()
			path := tc.setup(t, t.TempDir())
			r.eng.cfg.UpdaterJournal = path

			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			rep := mustReport(t, r, ctx)
			elapsed := time.Since(start)
			if elapsed > 1500*time.Millisecond {
				t.Fatalf("observation took %s (> deadline)", elapsed)
			}
			assertNullUpdate(t, rep)
			if got, err := rep.CurrentReleaseId.Get(); err != nil || got != r.eng.cfg.ReleaseUUID() {
				t.Fatalf("CurrentReleaseId=%v err=%v, want %s", got, err, r.eng.cfg.ReleaseUUID())
			}
			rawObs, err := json.Marshal(rep)
			if err != nil {
				t.Fatal(err)
			}
			for _, surface := range []struct{ name, text string }{
				{"log", r.log.String()},
				{"observation", string(rawObs)},
			} {
				if strings.Contains(surface.text, path) || strings.Contains(surface.text, marker) {
					t.Fatalf("%s leaked journal path or bytes: %s", surface.name, surface.text)
				}
			}
		})
	}
}

func TestReviewD061_Held_ResolverAdminURLsAndSIGHUPHalfWrittenFile(t *testing.T) {
	badURLs := []string{
		"http://127.0.0.1@evil.example/",
		"http://127.0.0.1@evil.example:8080",
		"http://[::ffff:127.0.0.1]:3000",
		"http://localhost.",
		"http://localhost.:3000",
		"http://127.1",
		"http://127.1:3000",
	}
	for _, u := range badURLs {
		cfg := &DNSConfig{
			Resolvers:               []DNSResolver{{ID: "lab", AdminURL: u, DNSAddr: "127.0.0.1:53"}},
			ResolverCredentialsFile: "/etc/vpn-node/resolver-credentials.json",
		}
		if err := (Config{DNS: cfg}).validateDNS(); err == nil {
			t.Fatalf("validateDNS accepted non-host-local admin URL %q", u)
		}
	}

	// Symlink in a world-writable directory and SIGHUP with a half-written credential file.
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	const secret = "pw-sighup-half-written-do-not-leak"

	worldDir := filepath.Join(t.TempDir(), "world-writable")
	if err := os.MkdirAll(worldDir, 0o777); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(t.TempDir(), "real-creds.json")
	if err := os.WriteFile(targetFile, []byte(`{"username":"e2e","password":"`+secret+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(worldDir, "symlink-creds.json")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readResolverCredentials(symlinkPath); err == nil {
		t.Fatal("symlink in world-writable dir was not refused")
	}

	halfPath := filepath.Join(t.TempDir(), "half-creds.json")
	if err := os.WriteFile(halfPath, []byte(`{"username":"e2e","password":"`+secret), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := parseResolverCredentials([]byte(`{"username":"e2e","password":"` + secret)); err == nil {
		t.Fatal("half-written JSON was not refused")
	}

	r.eng.cfg.DNS = &DNSConfig{
		Resolvers:               []DNSResolver{{ID: "lab", AdminURL: "http://127.0.0.1:8080", DNSAddr: "127.0.0.1:53"}},
		ResolverCredentialsFile: halfPath,
	}
	// Simulate SIGHUP reload with half-written file and adversarial admin URL reload.
	r.eng.reloadResolvers()
	if r.eng.dnsApply != nil {
		t.Fatal("reloadResolvers constructed an applier from a half-written credential file")
	}
	r.eng.cfg.DNS.Resolvers[0].AdminURL = "http://[::ffff:127.0.0.1]:3000"
	r.eng.reloadResolvers()
	if r.eng.dnsApply != nil {
		t.Fatal("reloadResolvers constructed an applier with an IPv4-mapped IPv6 admin URL")
	}

	body := []byte(exampleResolverBody)
	ref := exampleRef(body)
	r.srv.PutResolverBody(ref.DocumentId, body)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	ch := issueChallenge(t, r, nil)
	stepOK(t, r)
	att := requireAttest(t, r, ch)
	if att.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed {
		t.Fatalf("dns_policy passed after failed SIGHUP reload: %+v", att.Checks.DNSPolicy)
	}
	rawAtt, err := os.ReadFile(attestPath(r))
	if err != nil {
		t.Fatal(err)
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	rawObs, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []struct{ name, text string }{
		{"log", r.log.String()},
		{"observation", string(rawObs)},
		{"attestation", string(rawAtt)},
	} {
		if strings.Contains(surface.text, secret) || strings.Contains(surface.text, halfPath) || strings.Contains(surface.text, symlinkPath) {
			t.Fatalf("%s leaked credential secret or file path: %s", surface.name, surface.text)
		}
	}
}

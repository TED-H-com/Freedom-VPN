package members

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// TestR27_M2_AT09_RestartProvisionalPredicateAndTimelineScope proves the
// predicate and timeline defects in AT09 indices 25 and 26
// (tests/e2e/at/members/second_exit_test.go:220, 263, 267-313 and
// tests/e2e/at/members/provisional_test.go:48-114) against the retained
// run 2 evidence (docs/e2e/at/members/F18-AT09-restart.json and AT09.json):
//  1. Because RestartNodeNamespace90 restarts an already-ready node on an
//     unchanged generation (66 -> 66) and Step 1 renews the lease at
//     21:49:53.374125Z before sending its first post-restart observation
//     (21:49:53.427228Z), central issues a populated lease (member_grants: 7).
//  2. Both post-restart non-serving observations therefore have
//     member_grants == 7 (no awaiting_member_grants), so
//     verifyProvisionalObservations sets provisionalCount = 0 and
//     allValid = false, failing index 25 and cascading via provisionalOK = false
//     (second_exit_test.go:263) to fail index 26 even though all 8 other
//     conjuncts of index 26 are true.
//  3. Even when a post-restart 0-grant provisional observation does occur,
//     verifyProvisionalObservations requires allValid across every
//     member_forwarding_enabled=false observation after resumedAt, which
//     rejects any subsequent safety_check_failed observation under the
//     populated lease.
func TestR27_M2_AT09_RestartProvisionalPredicateAndTimelineScope(t *testing.T) {
	// Construct a valid D-067 / D-126 post-restart provisional observation
	// (0-grant lease while tunnel_probe is failing), followed by a second
	// non-serving observation after the populated lease (member_grants > 0)
	// arrives while tunnel_probe is still failing for one more tick.
	now := time.Date(2026, 11, 3, 21, 49, 53, 427228000, time.UTC)
	provisionalObs := api.ObservationReport{
		MemberForwardingEnabled: false,
		SafetyStatus:            api.SafetyStatusFailed,
		ForwardingBlockReasons: []api.ForwardingBlockReason{
			api.ForwardingBlockReasonAwaitingMemberGrants,
			api.ForwardingBlockReasonSafetyCheckFailed,
		},
		Checks: []api.NodeCheck{
			{
				Name:       api.NodeCheckNameLease,
				Status:     api.HealthCheckStatusFailed,
				Detail:     new(awaitingGrantsDetail),
				ObservedAt: now,
			},
			{
				Name:       api.NodeCheckNameQos,
				Status:     api.HealthCheckStatusFailed,
				ObservedAt: now,
			},
			{
				Name:       api.NodeCheckNameSourceBinding,
				Status:     api.HealthCheckStatusFailed,
				ObservedAt: now,
			},
			{
				Name:       api.NodeCheckNameTunnelProbe,
				Status:     api.HealthCheckStatusFailed,
				ObservedAt: now,
			},
		},
		Errors: []api.NodeError{
			{
				Code:       api.ErrorCodeAWAITINGMEMBERGRANTS,
				Stage:      "lease",
				Message:    awaitingGrantsDetail,
				ObservedAt: now,
			},
			{
				Code:       api.ErrorCodeHEALTHCHECKFAILED,
				Stage:      "probe",
				Message:    "tunnel_probe",
				ObservedAt: now,
			},
		},
	}
	provisionalObs.Checks[0].ErrorCode.Set(api.ErrorCodeAWAITINGMEMBERGRANTS)
	provisionalObs.Checks[1].ErrorCode.Set(api.ErrorCodeAWAITINGMEMBERGRANTS)
	provisionalObs.Checks[2].ErrorCode.Set(api.ErrorCodeAWAITINGMEMBERGRANTS)
	provisionalObs.Checks[3].ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)

	populatedSafetyObs := api.ObservationReport{
		MemberForwardingEnabled: false,
		SafetyStatus:            api.SafetyStatusFailed,
		ForwardingBlockReasons: []api.ForwardingBlockReason{
			api.ForwardingBlockReasonSafetyCheckFailed,
		},
		Checks: []api.NodeCheck{
			{Name: api.NodeCheckNameLease, Status: api.HealthCheckStatusPassed, ObservedAt: now},
			{Name: api.NodeCheckNameQos, Status: api.HealthCheckStatusPassed, ObservedAt: now},
			{Name: api.NodeCheckNameSourceBinding, Status: api.HealthCheckStatusPassed, ObservedAt: now},
			{Name: api.NodeCheckNameTunnelProbe, Status: api.HealthCheckStatusFailed, ObservedAt: now},
		},
		Errors: []api.NodeError{
			{
				Code:       api.ErrorCodeHEALTHCHECKFAILED,
				Stage:      "probe",
				Message:    "tunnel_probe",
				ObservedAt: now,
			},
		},
	}
	populatedSafetyObs.Checks[3].ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)

	// Call the live predicate, with the same two reports and explicit held
	// leases. M-2's old copied loop could never exercise the repaired code.
	resumedAt := now.Add(-time.Second)
	leases := []restartLease{
		{ID: uuid.New(), IssuedAt: resumedAt, ExpiresAt: now.Add(time.Hour), Generation: "66", MemberGrants: 0},
		{ID: uuid.New(), IssuedAt: now.Add(time.Second), ExpiresAt: now.Add(time.Hour), Generation: "66", MemberGrants: 7},
	}
	provisionalObs.ObservedAt = now
	provisionalObs.Lease.LeaseId.Set(leases[0].ID)
	populatedSafetyObs.ObservedAt = now.Add(2 * time.Second)
	populatedSafetyObs.Lease.LeaseId.Set(leases[1].ID)
	got := assessRestart(resumedAt, "66", leases, []restartObservation{{now, provisionalObs}, {now.Add(2 * time.Second), populatedSafetyObs}})
	if !got.SequenceOK || got.D067Status != "pass" {
		t.Fatalf("M-2: rejected grant-less then populated safety sequence: %+v", got)
	}
}

// TestR27_M3_AT02_SiblingRecoveryBracketAndQuarantinePredicates proves the
// four failing clauses in siblingRecovery (tests/e2e/at/members/apply_attribution_test.go:159-313)
// and counter_diagnostics_test.go:33-48:
//  1. apply_attribution_test.go:159 only links failed Applies to baselines with
//     Cause == "apply_rollback", ignoring Cause == "apply_quarantine"
//     (node/agent/engine.go:1316), so failed_apply_bracketed == 0 and
//     measurement_valid becomes false at line 226.
//  2. apply_attribution_test.go:171 closes open brackets at any verified Apply,
//     including a D-136 closedRecovery 0-grant deny-all Apply (seq=584,
//     lease.Members == 0, heartbeat.Serving == false), rather than waiting for
//     the member-serving Apply (seq=593, lease.Members == 1,
//     heartbeat.Serving == true), causing b.Recovered == false and classifying
//     packets dropped during the 0-grant deny-all hold as
//     losses_outside_baseline_or_completed_guard.
//  3. apply_attribution_test.go:310 requires missingIngress == 0 even when
//     Quarantine administratively lowers fvmb0 per D-090.
//  4. apply_attribution_test.go:313 sets valid = measurementValid && attempts == len(egress),
//     failing AT02 indices 190 and 191 on any fail-closed packet drop.
func TestR27_M3_AT02_SiblingRecoveryBracketAndQuarantinePredicates(t *testing.T) {
	boot := uuid.New()
	run := uuid.New()

	// Model the exact AT02 foreign-TC quarantine and D-136 closedRecovery sequence:
	// seq 1: failed Apply (before_install, TC_FOREIGN_FILTER)
	// seq 2: baseline with cause "apply_quarantine" (as emitted by engine.go:1316)
	// seq 3: lease renewal returning 0 member grants (provisional)
	// seq 4: D-136 closedRecovery Apply (Succeeded: true, Verified: true, 0 grants)
	// seq 5: heartbeat with Serving: false, Reason: "safety_check_failed"
	// seq 6: lease renewal returning 1 member grant
	// seq 7: serving Apply (Succeeded: true, Verified: true, 1 grant)
	// seq 8: heartbeat with Serving: true
	records := []agent.Measurement{
		{
			RunID:    run,
			BootID:   boot,
			Sequence: 1,
			Timing:   &enforce.TimingEvidence{Operation: "apply", CandidateGeneration: 12, Start: 100, End: 150, Succeeded: false},
		},
		{
			RunID:    run,
			BootID:   boot,
			Sequence: 2,
			Baseline: &agent.MeasurementBaseline{Cause: "apply_quarantine", Start: 151, End: 180, Succeeded: true},
		},
		{
			RunID:    run,
			BootID:   boot,
			Sequence: 3,
			Lease:    &agent.MeasurementLease{Generation: 13, Members: 0, RequestMono: 200},
		},
		{
			RunID:    run,
			BootID:   boot,
			Sequence: 4,
			Timing:   &enforce.TimingEvidence{Operation: "apply", CandidateGeneration: 13, Start: 220, End: 260, Succeeded: true, Verified: true},
		},
		{
			RunID:     run,
			BootID:    boot,
			Sequence:  5,
			Heartbeat: &agent.MeasurementHeartbeat{Generation: 13, Serving: false, BaselineOK: true, Reason: "safety_check_failed", Mono: 270},
		},
		{
			RunID:    run,
			BootID:   boot,
			Sequence: 6,
			Lease:    &agent.MeasurementLease{Generation: 13, Members: 1, RequestMono: 400},
		},
		{
			RunID:    run,
			BootID:   boot,
			Sequence: 7,
			Timing:   &enforce.TimingEvidence{Operation: "apply", CandidateGeneration: 13, Start: 420, End: 460, Succeeded: true, Verified: true},
		},
		{
			RunID:     run,
			BootID:    boot,
			Sequence:  8,
			Heartbeat: &agent.MeasurementHeartbeat{Generation: 13, Serving: true, Written: true, BaselineOK: true, Mono: 470},
		},
	}

	var trace []json.RawMessage
	for _, m := range records {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		trace = append(trace, b)
	}

	// Packet 1: forwarded before fault (t=80..81).
	// Packet 2: dropped at node ingress while closedRecovery 0-grant deny-all
	//           Apply (seq=4) is in effect before serving Apply (seq=7) (t=300).
	// Packet 3: forwarded after serving Apply (seq=7) (t=480..481).
	cap := siblingObservation{
		Boot:           boot.String(),
		Start:          50,
		End:            600,
		StatsAvailable: true,
		Packets: []siblingPacket{
			{1, 80, "fvmb0"},
			{1, 81, "fvup0"},
			{2, 300, "fvmb0"},
			{3, 480, "fvmb0"},
			{3, 481, "fvup0"},
		},
	}
	rawCap, err := json.Marshal(cap)
	if err != nil {
		t.Fatal(err)
	}

	out, recoveryOK, observed := siblingRecovery(trace, rawCap)
	if !observed {
		t.Fatal("expected failed Apply in capture to be observed")
	}

	// TODO(review r27): M-3 — siblingRecovery fails to bracket a failed Apply
	// followed by baseline cause "apply_quarantine" (failed_apply_bracketed=0),
	// closes the bracket at the non-serving 0-grant closedRecovery Apply (seq=4)
	// instead of the serving Apply (seq=7) (causing losses_outside_baseline_or_completed_guard=1
	// and b.Recovered=false), and requires 0 total packet losses for recoveryOK.
	if !recoveryOK || out["measurement_valid"] != true || out["failed_apply_bracketed"] != 1 || out["losses_outside_baseline_or_completed_guard"] != 0 {
		t.Fatalf("TODO(review r27): M-3: siblingRecovery rejected valid quarantine + D-136 closedRecovery sequence: recoveryOK=%v measurement_valid=%v failed_apply_bracketed=%v losses_outside=%v",
			recoveryOK, out["measurement_valid"], out["failed_apply_bracketed"], out["losses_outside_baseline_or_completed_guard"])
	}
}

// TestR27_Held_AT09_RestartRecoveryConjunctsAndTimelineTrace verifies the exact
// recorded values in docs/e2e/at/members/F18-AT09-restart.json and
// docs/e2e/at/members/AT09.json (steps[25] and steps[26]):
//   - index 26 had sameGeneration=true (66==66), fwdAfter=true,
//     postRestartPopulated=true (lease at 21:49:53.374125Z has member_grants=7),
//     counted=true, firstByteSeconds=12.548980167 (within (0, 90]), queryOK=true,
//     and failed solely because provisionalOK (index 25) was false;
//   - observations[0] in index 26 (21:48:44.725565Z, carrying STALE_GENERATION
//     "applied=66 lease=65" and candidate_generation="21") preceded
//     capture.started_at (21:49:52.571423298Z) by 67.846s.
func TestR27_Held_AT09_RestartRecoveryConjunctsAndTimelineTrace(t *testing.T) {
	raw, err := os.ReadFile("../../../../docs/e2e/at/members/F18-AT09-restart.json")
	if err != nil {
		t.Fatal(err)
	}
	var wrapper struct {
		OriginalSteps []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Actual string `json:"actual"`
		} `json:"original_steps"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		t.Fatal(err)
	}
	if len(wrapper.OriginalSteps) < 7 {
		t.Fatalf("expected 7 original_steps in F18-AT09-restart.json, got %d", len(wrapper.OriginalSteps))
	}

	var doc struct {
		Capture struct {
			StartedAt        time.Time `json:"started_at"`
			GenerationBefore string    `json:"generation_before"`
			GenerationAfter  string    `json:"generation_after"`
		}
		Provisional struct {
			AllMatchRevision11      bool             `json:"all_match_revision_11"`
			ProvisionalObservations int              `json:"provisional_observations"`
			Observations            []map[string]any `json:"observations"`
		}
		Timeline struct {
			DesiredBefore           string    `json:"desired_before"`
			DesiredAfter            string    `json:"desired_after"`
			AppliedBefore           string    `json:"applied_before"`
			AppliedAfter            string    `json:"applied_after"`
			FirstCountedByteAt      time.Time `json:"first_counted_byte_at"`
			FirstCountedByteSeconds float64   `json:"first_counted_byte_seconds"`
			Leases                  []struct {
				IssuedAt     time.Time `json:"issued_at"`
				ExpiresAt    time.Time `json:"expires_at"`
				Generation   string    `json:"generation"`
				MemberGrants int       `json:"member_grants"`
			} `json:"leases"`
			Observations []struct {
				FirstAt    time.Time `json:"first_at"`
				LastAt     time.Time `json:"last_at"`
				Generation string    `json:"generation"`
				Forwarding bool      `json:"forwarding"`
				Reasons    []string  `json:"reasons"`
				Errors     []struct {
					Code    string `json:"code"`
					Stage   string `json:"stage"`
					Message string `json:"message"`
				} `json:"errors"`
			} `json:"observations"`
		}
	}
	var capStep struct {
		Capture struct {
			StartedAt        time.Time `json:"started_at"`
			GenerationBefore string    `json:"generation_before"`
			GenerationAfter  string    `json:"generation_after"`
		} `json:"capture"`
	}
	if err := json.Unmarshal([]byte(wrapper.OriginalSteps[4].Actual), &capStep); err != nil {
		t.Fatal(err)
	}
	doc.Capture = capStep.Capture
	if err := json.Unmarshal([]byte(wrapper.OriginalSteps[5].Actual), &doc.Provisional); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(wrapper.OriginalSteps[6].Actual), &doc.Timeline); err != nil {
		t.Fatal(err)
	}

	resumedAt := doc.Capture.StartedAt
	sameGeneration := doc.Timeline.DesiredBefore != "0" &&
		doc.Timeline.DesiredBefore == doc.Timeline.AppliedBefore &&
		doc.Timeline.DesiredAfter == doc.Timeline.DesiredBefore &&
		doc.Timeline.AppliedAfter == doc.Timeline.DesiredBefore
	fwdAfter := doc.Timeline.Observations[len(doc.Timeline.Observations)-1].Forwarding
	postRestartPopulated := false
	for _, l := range doc.Timeline.Leases {
		if !l.IssuedAt.Before(resumedAt) && l.Generation == doc.Timeline.DesiredAfter && l.MemberGrants > 0 {
			postRestartPopulated = true
		}
	}
	firstByteOK := !doc.Timeline.FirstCountedByteAt.IsZero() &&
		doc.Timeline.FirstCountedByteSeconds > 0 &&
		doc.Timeline.FirstCountedByteSeconds <= 90
	provisionalOK := doc.Provisional.ProvisionalObservations > 0 && doc.Provisional.AllMatchRevision11

	if !sameGeneration || !fwdAfter || !postRestartPopulated || !firstByteOK {
		t.Fatalf("unexpected false non-provisional conjunct: sameGeneration=%v fwdAfter=%v postRestartPopulated=%v firstByteOK=%v",
			sameGeneration, fwdAfter, postRestartPopulated, firstByteOK)
	}
	if provisionalOK || doc.Provisional.AllMatchRevision11 || doc.Provisional.ProvisionalObservations != 2 || len(doc.Provisional.Observations) != 2 {
		t.Fatalf("expected provisionalOK=false with all_match_revision_11=false across 2 non-serving observations, got provisionalOK=%v count=%d len=%d",
			provisionalOK, doc.Provisional.ProvisionalObservations, len(doc.Provisional.Observations))
	}

	// Verify that observations[0] (carrying STALE_GENERATION) occurred 67.85s BEFORE resumedAt.
	obs0 := doc.Timeline.Observations[0]
	if !obs0.FirstAt.Before(resumedAt) || resumedAt.Sub(obs0.FirstAt) < 60*time.Second {
		t.Fatalf("expected observations[0] (%s) to precede resumedAt (%s) by >60s", obs0.FirstAt, resumedAt)
	}
	foundStale := false
	for _, e := range obs0.Errors {
		if e.Stage == "lease" && e.Code == "STALE_GENERATION" && strings.Contains(e.Message, "applied=66 lease=65") {
			foundStale = true
		}
	}
	if !foundStale {
		t.Fatalf("expected pre-restart STALE_GENERATION applied=66 lease=65 in observations[0], got %+v", obs0.Errors)
	}
}

// TestR27_Held_AT02_CapturedRecoveryPacketAndBracketAccounting verifies the
// exact packet and bracket accounting in docs/e2e/at/members/F18-AT02-recovery.json.gz:
//   - 6,250 sender attempts, 5,267 ingress packets, 5,008 egress packets,
//     1,242 lost admitted packets (983 missing ingress + 259 ingress without egress);
//   - all 983 missing-ingress packets are contiguous seq 4984..5966 sent while
//     fvmb0 was administratively DOWN during the foreign-TC quarantine window;
//   - the 259 ingress-without-egress losses decompose into:
//   - 8 during policy_missing bracket 25 (base_seq=404 -> Apply 408): 1 in guard + 7 outside guard;
//   - 8 during policy_missing bracket 28 (base_seq=416 -> Apply 420): 1 in guard + 7 outside guard;
//   - 4 during closedRecovery Apply 584 (brackets 31..49, outside guard);
//   - 233 between closedRecovery Apply 584's end (63882732277516) and
//     awaiting_member_grants baseline 589's start (63887519837810), which
//     account for 100% of losses_outside_baseline_or_completed_guard == 233;
//   - 6 during awaiting_member_grants bracket 50 (base_seq=589 -> Apply 593): 1 in guard + 5 outside guard.
func TestR27_Held_AT02_CapturedRecoveryPacketAndBracketAccounting(t *testing.T) {
	f, err := os.Open("../../../../docs/e2e/at/members/F18-AT02-recovery.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()

	var doc struct {
		Observation siblingObservation `json:"observation"`
		Sender      struct {
			Sends []struct {
				Sequence decstr.U64 `json:"sequence"`
			} `json:"sends"`
		} `json:"sender"`
		MeasurementValid                      bool              `json:"measurement_valid"`
		SenderAttempts                        int               `json:"sender_attempts"`
		IngressPackets                        int               `json:"ingress_packets"`
		EgressPackets                         int               `json:"egress_packets"`
		LostAdmittedPackets                   int               `json:"lost_admitted_packets"`
		NotObservedAtNodeIngress              int               `json:"not_observed_at_node_ingress"`
		NodeIngressWithoutEgress              int               `json:"node_ingress_without_egress"`
		LossesInCompletedGuards               int               `json:"losses_in_completed_guards"`
		LossesOutsideCompletedGuards          int               `json:"losses_outside_completed_guards"`
		LossesOutsideBaselineOrCompletedGuard int               `json:"losses_outside_baseline_or_completed_guard"`
		FailedApplyInCapture                  int               `json:"failed_apply_in_capture"`
		FailedApplyBracketed                  int               `json:"failed_apply_bracketed"`
		Brackets                              []recoveryBracket `json:"brackets"`
	}
	if err := json.NewDecoder(gz).Decode(&doc); err != nil {
		t.Fatal(err)
	}

	if doc.SenderAttempts != 6250 || doc.IngressPackets != 5267 || doc.EgressPackets != 5008 ||
		doc.LostAdmittedPackets != 1242 || doc.NotObservedAtNodeIngress != 983 ||
		doc.NodeIngressWithoutEgress != 259 || doc.LossesInCompletedGuards != 3 ||
		doc.LossesOutsideCompletedGuards != 256 || doc.LossesOutsideBaselineOrCompletedGuard != 233 ||
		doc.FailedApplyInCapture != 3 || doc.FailedApplyBracketed != 0 || doc.MeasurementValid {
		t.Fatalf("unexpected recovery summary: %+v", doc)
	}

	ingress := map[int]int64{}
	egress := map[int]int64{}
	for _, p := range doc.Observation.Packets {
		seq := int(p.Sequence)
		switch p.Interface {
		case "fvmb0":
			if _, exists := ingress[seq]; !exists {
				ingress[seq] = int64(p.At)
			}
		case "fvup0":
			if _, exists := egress[seq]; !exists {
				egress[seq] = int64(p.At)
			}
		}
	}

	// 1. All 983 missing-ingress packets are contiguous seq 4984..5966 during fvmb0 quarantine.
	missingCount := 0
	minMissingSeq, maxMissingSeq := 1<<30, -1
	for _, s := range doc.Sender.Sends {
		seq := int(s.Sequence)
		if _, ok := ingress[seq]; !ok {
			missingCount++
			if seq < minMissingSeq {
				minMissingSeq = seq
			}
			if seq > maxMissingSeq {
				maxMissingSeq = seq
			}
		}
	}
	if missingCount != 983 || minMissingSeq != 4984 || maxMissingSeq != 5966 {
		t.Fatalf("missing ingress packets: count=%d range=%d..%d, want 983 in 4984..5966",
			missingCount, minMissingSeq, maxMissingSeq)
	}

	// 2. All 233 unbracketed ingress-without-egress losses fall strictly between
	// Apply 584's recovery_end_ns (63882732277516) and baseline 589's
	// baseline_start_ns (63887519837810), i.e. while Apply 584's closedRecovery
	// 0-grant deny-all ruleset was active.
	const apply584End = int64(63882732277516)
	const base589Start = int64(63887519837810)
	between584And589 := 0
	for seq, inMono := range ingress {
		if _, ok := egress[seq]; ok {
			continue
		}
		if inMono > apply584End && inMono < base589Start {
			between584And589++
		}
	}
	if between584And589 != 233 {
		t.Fatalf("losses between Apply 584 end and baseline 589 start = %d, want 233", between584And589)
	}
}

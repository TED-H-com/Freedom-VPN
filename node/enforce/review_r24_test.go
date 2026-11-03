package enforce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen proves
// Finding H-4 against main:
//  1. When Apply starts from a cold table (state == State{}) and fails in
//     verifyServing after releaseDeadlineGuard has already opened the candidate,
//     a transient loadState failure inside installBaselineModel causes
//     installBaselineModel to silently ignore the error and fall back to the
//     stale pre-install State{}. RenderBaseline then omits `delete chain inet
//     fvpn forward` and `flush set inet fvpn allowed_v4`, appending baseline
//     drop rules *after* the live candidate's accept rules and leaving
//     @allowed_v4 populated.
//  2. When Apply's ctx expires during verifyServing (after releaseDeadlineGuard
//     has already removed fv_deadline_guard), Apply calls installBaseline with
//     the expired ctx instead of an independent cleanup context, so the
//     rollback NftBatch fails before executing.
func TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen(t *testing.T) {
	t.Run("cold_start_rollback_with_failed_loadState_omits_delete_chain_and_flush_set", func(t *testing.T) {
		m := smallModel()
		base := servingExec(m)
		var batches []string
		tableListing := 0
		ex := &r24HookExec{
			fakeExec: base,
			onBatch: func(_ context.Context, script string) error {
				batches = append(batches, script)
				return nil
			},
			onOutput: func(ctx context.Context, name string, args ...string) ([]byte, error) {
				key := name + " " + strings.Join(args, " ")
				switch key {
				case "nft -j list table inet fvpn":
					tableListing++
					if tableListing == 1 {
						// Cold start before Apply: table does not exist yet.
						return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit 1"), Out: "Error: No such file or directory"}
					}
					// During post-release rollback, loadState fails transiently.
					return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit 1"), Out: "netlink busy"}
				case "nft -j list ruleset":
					// Post-release verification read fails after guard release.
					return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit 1"), Out: "verifier read failed"}
				default:
					return base.Output(ctx, name, args...)
				}
			},
		}
		res, err := Apply(context.Background(), ex, m, nil)
		if err == nil || res.Verified {
			t.Fatalf("expected post-release verification failure, got res=%+v err=%v", res, err)
		}
		if len(batches) != 3 {
			t.Fatalf("expected 3 nft batches (install, release, rollback), got %d", len(batches))
		}
		rollback := batches[2]
		if !strings.Contains(rollback, "delete chain inet fvpn forward\n") || !strings.Contains(rollback, "flush set inet fvpn allowed_v4\n") {
			// TODO(review r24): H-4 — installBaselineModel ignored loadState's error and fell back to the pre-install cold State{}, so RenderBaseline omitted `delete chain inet fvpn forward` and `flush set inet fvpn allowed_v4`, appending baseline drops after the candidate's live accept rules.
			t.Fatalf("TODO(review r24): H-4: post-release rollback rendered against stale cold State{} omitted `delete chain inet fvpn forward` and `flush set inet fvpn allowed_v4`:\n%s", rollback)
		}
	})

	t.Run("expired_ctx_during_verify_aborts_rollback_after_guard_release", func(t *testing.T) {
		m := smallModel()
		base := servingExec(m)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var batches []string
		ex := &r24HookExec{
			fakeExec: base,
			onBatch: func(batchCtx context.Context, script string) error {
				if err := batchCtx.Err(); err != nil {
					return err
				}
				batches = append(batches, script)
				return nil
			},
			onOutput: func(outCtx context.Context, name string, args ...string) ([]byte, error) {
				key := name + " " + strings.Join(args, " ")
				if key == "nft -j list ruleset" {
					// Context expires during post-release verification.
					cancel()
					return nil, outCtx.Err()
				}
				return base.Output(outCtx, name, args...)
			},
		}
		res, err := Apply(ctx, ex, m, nil)
		if err == nil || res.Verified {
			t.Fatalf("expected verification failure on canceled ctx, got res=%+v err=%v", res, err)
		}
		hasRollback := false
		for _, b := range batches {
			if strings.Contains(b, "fvpn_baseline=1") {
				hasRollback = true
			}
		}
		if !hasRollback {
			// TODO(review r24): H-4 — Apply passes its expired ctx directly into installBaseline after releaseDeadlineGuard has already opened the candidate, so rollback fails with context.Canceled without executing an nft baseline batch.
			t.Fatalf("TODO(review r24): H-4: guard was released (batches=%d, GuardReleased=%v) and verification failed on canceled ctx, but installBaseline reused the canceled ctx and executed no baseline rollback batch",
				len(batches), res.Deadline.GuardReleased)
		}
	})
}

// TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch proves Finding H-2's
// datapath ordering defect on main (and carried into C1.1): when a member is
// removed in generation N, Apply runs TCBatch (`qdisc del` + `class del` for
// the removed member's HTB class 1:<minor>) at apply.go:151 *before* NftBatch
// replaces `forward` at apply.go:183. Between TCBatch and NftBatch, packets of
// the removed member still match the old `forward` rule `meta priority set
// 1:<minor> accept`; because class 1:<minor> was already deleted, HTB routes
// those packets into the root qdisc's `default fffe` class (HostClass, the
// host control-plane reserve, §8.3).
func TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch(t *testing.T) {
	m1 := goldenModel()
	firstPlan, err := Render(m1, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	removedMinor := firstPlan.Assignments[memberB]
	removedHandle := classHandle(removedMinor)

	// Generation 8 removes memberB while memberA's devA1 remains.
	m2 := smallModel()
	m2.Generation = 8
	m2.QoS.Assignments = firstPlan.Assignments

	base := servingExec(m2)
	var events []string
	ex := &r24HookExec{
		fakeExec: base,
		onTC: func(_ context.Context, script string) error {
			if strings.Contains(script, "class del dev fv-ih classid "+removedHandle) {
				events = append(events, "tc_delete_member_class:"+removedHandle)
			}
			return nil
		},
		onBatch: func(_ context.Context, script string) error {
			if strings.Contains(script, "add chain inet fvpn "+deadlineGuard) {
				events = append(events, "nft_guarded_install")
			}
			return nil
		},
		onOutput: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			key := name + " " + strings.Join(args, " ")
			// Before Apply's TCBatch runs, report both memberA and memberB leaves present in live TC.
			if len(events) == 0 && (key == "tc -j class show dev fv-ih" || key == "tc -j class show dev fv-fh") {
				minorA := firstPlan.Assignments[memberA]
				return []byte(fmt.Sprintf(
					`[{"kind":"htb","handle":"1:1","options":{"rate":"40mbit","ceil":"40mbit"}},{"kind":"htb","handle":"1:fffe","options":{"rate":"1mbit","ceil":"40mbit"}},{"kind":"htb","handle":%q,"options":{"rate":"20mbit","ceil":"20mbit"}},{"kind":"htb","handle":%q,"options":{"rate":"10mbit","ceil":"20mbit"}}]`,
					classHandle(minorA), removedHandle,
				)), nil
			}
			if len(events) == 0 && (key == "tc -j qdisc show dev fv-ih" || key == "tc -j qdisc show dev fv-fh") {
				minorA := firstPlan.Assignments[memberA]
				return []byte(fmt.Sprintf(
					`[{"kind":"htb","handle":"1:","root":true,"options":{"default":"fffe"}},{"kind":"fq_codel","handle":"%x:","parent":%q},{"kind":"fq_codel","handle":"%x:","parent":%q}]`,
					minorA, classHandle(minorA), removedMinor, removedHandle,
				)), nil
			}
			return base.Output(ctx, name, args...)
		},
	}
	if _, err := Apply(context.Background(), ex, m2, nil); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(events) >= 2 && events[0] == "tc_delete_member_class:"+removedHandle && events[1] == "nft_guarded_install" {
		// TODO(review r24): H-2 — Apply deletes orphan member HTB classes in TCBatch before NftBatch removes their `meta priority set` rules from `forward`, directing packets of the removed member into HTB root `default fffe` (HTBDefaultMinor) until NftBatch commits.
		t.Fatalf("TODO(review r24): H-2: orphan member HTB class %s was deleted in TCBatch before NftBatch replaced forward (order=%v), routing packets with meta priority 0x%x into HTB default class 1:%x (HTBDefaultMinor)",
			removedHandle, events, ClassID(removedMinor), HTBDefaultMinor)
	}
}

// TestR24_M2_PostShapeClockCheckSkipsBaselineInApply proves Finding M-2
// against main: Apply checks `timing.RenderStart < anchor` (`m.NowMono`) at
// apply.go:164 *after* executing `ex.TCBatch` at apply.go:151, and returns
// `ReasonIsolation` ("monotonic clock must not move backwards") without
// calling `installBaseline` (unlike the adjacent error paths at lines 152 and
// 174).
func TestR24_M2_PostShapeClockCheckSkipsBaselineInApply(t *testing.T) {
	m := smallModel()
	nowMono, err := MonotonicNow()
	if err != nil {
		t.Fatal(err)
	}
	// Caller anchor is ahead of MonotonicNow(), triggering apply.go:164 after TCBatch.
	m.NowMono = nowMono + time.Hour
	deadline := m.NowMono + 30*time.Minute
	m.Leases[0].DeadlineMono = &deadline

	base := servingExec(m)
	var tcBatches, nftBatches []string
	ex := &r24HookExec{
		fakeExec: base,
		onTC: func(_ context.Context, script string) error {
			tcBatches = append(tcBatches, script)
			return nil
		},
		onBatch: func(_ context.Context, script string) error {
			nftBatches = append(nftBatches, script)
			return nil
		},
	}
	_, err = Apply(context.Background(), ex, m, nil)
	if err == nil || ReasonCode(err) != ReasonIsolation {
		t.Fatalf("expected ReasonIsolation from post-shape clock check, got %v", err)
	}
	if len(tcBatches) != 1 {
		t.Fatalf("expected TCBatch to have executed before the clock check, got %d", len(tcBatches))
	}
	hasBaseline := false
	for _, b := range nftBatches {
		if strings.Contains(b, "fvpn_baseline=1") {
			hasBaseline = true
		}
	}
	if !hasBaseline {
		// TODO(review r24): M-2 — Apply checked `timing.RenderStart < anchor` only after mutating live TC in `TCBatch` (apply.go:151 vs 164) and returned an error without calling `installBaseline`.
		t.Fatalf("TODO(review r24): M-2: Apply mutated live TC (tcBatches=%d) then failed at apply.go:164 (%v) without calling installBaseline (nftBatches=%d)",
			len(tcBatches), err, len(nftBatches))
	}
}

// TestR24_M3_NetlabTraceAndGuardLogRejectC1Scripts proves Finding M-3:
// the netlab trace recognizer (`traceExecutor.NftBatch` in
// `tests/netlab/cmd/fvpn-netlab/trace.go:72-89`) and the NFLOG group-29
// decorator (`guardLogScript` in `trace.go:157-177`) both fail on the exact
// C1.2 install and C1.4 release scripts specified in
// `docs/poc/G2-followup31-design/candidates.md`.
func TestR24_M3_NetlabTraceAndGuardLogRejectC1Scripts(t *testing.T) {
	c1Install := strings.Join([]string{
		"add table inet fvpn",
		"delete set inet fvpn fv_allow_g0",
		"create set inet fvpn fv_allow_g1 { type ipv4_addr; flags timeout; }",
		"add element inet fvpn fv_allow_g1 { 100.64.1.1 timeout 3599900ms }",
		"create set inet fvpn fv_eligible_g1 { type ipv4_addr; }",
		"add element inet fvpn fv_eligible_g1 { 100.64.1.1 }",
		"create set inet fvpn fv_token_g1 { type ipv4_addr; }",
		"add chain inet fvpn fv_deadline_guard { type filter hook forward priority -151; policy accept; }",
		"flush chain inet fvpn fv_deadline_guard",
		`add rule inet fvpn fv_deadline_guard iifname "fv-fh" oifname "fv-ih" meta nfproto ipv4 ip saddr @fv_eligible_g1 ip saddr @fv_allow_g0 return`,
		`add rule inet fvpn fv_deadline_guard iifname "fv-ih" oifname "fv-fh" meta nfproto ipv4 ip daddr @fv_eligible_g1 ip daddr @fv_allow_g0 return`,
		`add rule inet fvpn fv_deadline_guard iifname "fv-fh" drop`,
		`add rule inet fvpn fv_deadline_guard oifname "fv-fh" drop`,
	}, "\n")

	c1Release := strings.Join([]string{
		"delete set inet fvpn fv_token_g1",
		"delete chain inet fvpn fv_deadline_guard",
		"delete set inet fvpn fv_eligible_g1",
	}, "\n")

	// Exact recognizer logic from tests/netlab/cmd/fvpn-netlab/trace.go:73-81.
	classifyTraceBatch := func(script string) string {
		kind := "other"
		if strings.Contains(script, "add chain inet fvpn fv_deadline_guard ") {
			kind = "guard_install"
		}
		if strings.TrimSpace(script) == "delete chain inet fvpn fv_deadline_guard" {
			kind = "guard_release"
		}
		if strings.Contains(script, "fvpn_generation=0") {
			kind = "baseline"
		}
		return kind
	}

	// Exact NFLOG decorator logic from tests/netlab/cmd/fvpn-netlab/trace.go:157-177.
	guardLogScript := func(script string, generation uint64) (string, error) {
		lines := strings.Split(script, "\n")
		seen := map[string]bool{}
		for i, line := range lines {
			for _, direction := range []string{"iifname", "oifname"} {
				prefix := "add rule inet fvpn fv_deadline_guard " + direction + " "
				if !strings.HasPrefix(line, prefix) {
					continue
				}
				if seen[direction] || !strings.HasSuffix(line, " drop") {
					return "", errors.New("TEST ONLY unexpected guard drop shape")
				}
				seen[direction] = true
				lines[i] = strings.TrimSuffix(line, " drop") + fmt.Sprintf(" log prefix \"f29g%d-%s\" group 29 snaplen 128 queue-threshold 1 drop", generation, direction)
			}
		}
		if len(seen) != 2 {
			return "", errors.New("TEST ONLY expected two guard drops")
		}
		return strings.Join(lines, "\n"), nil
	}

	if got := classifyTraceBatch(c1Install); got != "guard_install" {
		t.Fatalf("expected C1.2 install to classify as guard_install, got %q", got)
	}
	// Proves M-3 part 1: C1.4 multi-line fenced release is misclassified as "other",
	// which causes reload_analysis.py:55 (`incomplete guard brackets`) and
	// requireProducerTiming (`reload_trace_test.go:125`) to fail every C1 run.
	if got := classifyTraceBatch(c1Release); got != "other" {
		t.Fatalf("expected current trace.go recognizer to misclassify C1.4 release as \"other\", got %q", got)
	}
	// Proves M-3 part 2: guardLogScript rejects C1.2's 4-rule guard because the
	// first two rules start with `add rule inet fvpn fv_deadline_guard iifname `
	// and end with ` return` rather than ` drop`.
	if _, err := guardLogScript(c1Install, 1001); err == nil || err.Error() != "TEST ONLY unexpected guard drop shape" {
		t.Fatalf("expected current guardLogScript to fail on C1.2 return rules, got err=%v", err)
	}
}

// TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity proves
// What Held on main for Question 1 (G1, G2, G3, G5, G6):
//   - guardedNFT wraps the candidate ruleset with `fv_deadline_guard` at
//     priority -151 dropping both iifname and oifname on MemberIngress;
//   - Apply's second Render pass sets timeoutReserve = DeadlineInstallBound
//     (100ms) and refuses to release the guard when InstalledBy - RenderStart
//     exceeds 100ms;
//   - verifyAllowedLifetimes enforces elem.timeout <= expected and
//     elem.expires <= max(expected - elapsed, 0) + 250ms;
//   - Render preserves existing device counters without emitting `delete counter`.
func TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity(t *testing.T) {
	m := smallModel()
	m.NowMono = 10 * time.Second
	deadline := m.NowMono + 30*time.Second
	m.Leases[0].DeadlineMono = &deadline

	// 1. First Render uses zero reserve; reserved Render subtracts 100ms.
	unreserved, err := Render(m, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	mReserved := m
	mReserved.timeoutReserve = DeadlineInstallBound
	reserved, err := Render(mReserved, State{}, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	devID := m.Bindings[0].DeviceID.String()
	if got, want := unreserved.ElementTimeouts[devID], 30*time.Second; got != want {
		t.Fatalf("unreserved timeout=%s want=%s", got, want)
	}
	if got, want := reserved.ElementTimeouts[devID], 30*time.Second-DeadlineInstallBound; got != want {
		t.Fatalf("reserved timeout=%s want=%s", got, want)
	}

	// 2. guardedNFT atomically wraps the batch in fv_deadline_guard at priority -151.
	guarded := guardedNFT(reserved.NFT, mReserved)
	for _, want := range []string{
		"add chain inet fvpn fv_deadline_guard { type filter hook forward priority -151; policy accept; }",
		"flush chain inet fvpn fv_deadline_guard",
		`add rule inet fvpn fv_deadline_guard iifname "fv-fh" drop`,
		`add rule inet fvpn fv_deadline_guard oifname "fv-fh" drop`,
	} {
		if !strings.Contains(guarded, want) {
			t.Fatalf("guarded batch missing %q:\n%s", want, guarded)
		}
	}

	// 3. verifyAllowedLifetimes enforces exact millisecond ceilings (el.timeout <= original, el.expires <= remaining).
	validListing := []byte("table inet fvpn { set allowed_v4 { type ipv4_addr flags timeout elements = { 100.64.1.1 timeout 29s900ms expires 29s } } }")
	if _, err := verifyAllowedLifetimes(validListing, m, m.NowMono+time.Second); err != nil {
		t.Fatalf("valid lifetime rejected: %v", err)
	}
	overCeilingListing := []byte("table inet fvpn { set allowed_v4 { type ipv4_addr flags timeout elements = { 100.64.1.1 timeout 29s900ms expires 29s1ms } } }")
	if _, err := verifyAllowedLifetimes(overCeilingListing, m, m.NowMono+time.Second); err == nil {
		t.Fatal("verifyAllowedLifetimes accepted kernel expires > remaining deadline")
	}

	// 4. Counter continuity: existing counters in State are never recreated or deleted.
	up := usageCounter(dirUpload, m.Bindings[0].DeviceID)
	dn := usageCounter(dirDownload, m.Bindings[0].DeviceID)
	withCounters := State{
		Table: true,
		Counters: map[string]CounterValue{
			up: {Packets: 10, Bytes: 1000},
			dn: {Packets: 20, Bytes: 2000},
		},
	}
	secondPass, err := Render(mReserved, withCounters, TCState{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(secondPass.NFT, "delete counter") || strings.Contains(secondPass.NFT, "add counter inet fvpn "+up) || strings.Contains(secondPass.NFT, "add counter inet fvpn "+dn) {
		t.Fatalf("second pass recreated or deleted existing device counters:\n%s", secondPass.NFT)
	}
}

type r24HookExec struct {
	*fakeExec
	onBatch  func(ctx context.Context, script string) error
	onTC     func(ctx context.Context, script string) error
	onOutput func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func (h *r24HookExec) NftBatch(ctx context.Context, script string) error {
	if h.onBatch != nil {
		if err := h.onBatch(ctx, script); err != nil {
			return err
		}
	}
	return h.fakeExec.NftBatch(ctx, script)
}

func (h *r24HookExec) TCBatch(ctx context.Context, script string) error {
	if h.onTC != nil {
		if err := h.onTC(ctx, script); err != nil {
			return err
		}
	}
	return h.fakeExec.TCBatch(ctx, script)
}

func (h *r24HookExec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if h.onOutput != nil {
		return h.onOutput(ctx, name, args...)
	}
	return h.fakeExec.Output(ctx, name, args...)
}

package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestUPD_D053_Review_RevokedCurrentRollbackIssuesChallengeAndReportsTruth(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	w := newWorld(t)
	w.releaseSeq = "1"
	older := w.makeRelease("older", 1, []int{1})
	w.releaseSeq = "2"
	cur := w.makeRelease("cur", 1, []int{1})
	w.keep(older)
	w.seed(cur)
	touchPersistent(t, w.layout)

	// Production NodeChecks health adapter with a healthy agent answering any challenge.
	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		return goodAttest(ch, api.PublicationIntentPublished), true
	})

	work := w.work(cur, "7", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{older.id, cur.id}
	work.PolicyGeneration = decstr.MustParse("18")
	work.PolicyKnown = true
	revoked := []uuid.UUID{cur.id}
	work.SignedRevoked = &revoked
	w.setWork(work)

	err := w.run()
	// When cur is revoked and older is healthy, restoreRelease should succeed
	// and Run should return RELEASE_NOT_APPROVED (not ErrMembersBlocked / ROLLBACK_FAILED).
	if asOp(err) == nil || asOp(err).Code != apierr.ReleaseNotApproved {
		p, _ := w.eng.Journal.Pointers(context.Background())
		res := w.oneResult()
		prevID := requiredValue(t, res.PreviousReleaseId)
		curID := requiredValue(t, res.CurrentReleaseId)
		t.Fatalf("revocation rollback under NodeChecks failed: err=%v; journal pointers=%+v; result state=%s code=%v recovery=%v current_release_id=%s previous_release_id=%s (want previous=%s)",
			err, p, res.State, requiredValue(t, res.ErrorCode), requiredValue(t, res.RecoveryRequired), curID, prevID, cur.id)
	}
}

func TestUPD_D053_Review_PollBreaksOnTransientFailedCheckBeforeWaitExpires(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	w := newWorld(t)
	next := w.makeRelease("new", 1, []int{1})
	touchPersistent(t, w.layout)

	w.eng.Sleep = nil
	w.eng.Attest = AttestWindow{Wait: 80 * time.Millisecond, Poll: 10 * time.Millisecond, TTL: 2 * time.Second}
	nc := &NodeChecks{Journal: w.layout.Journal, Node: w.node, rt: &checkRT{}}
	w.eng.Health = nc

	ch, err := healthattest.Issue(healthattest.Input{
		TxnID:         uuid.Must(uuid.NewV7()).String(),
		ReleaseID:     next.id.String(),
		NodeID:        w.node.ID.String(),
		IncarnationID: w.node.Incarnation.String(),
		BootID:        w.node.BootID.String(),
		Now:           w.now,
		TTL:           2 * time.Second,
		Components: healthattest.Components{
			NodeAgent:   "1.0.0",
			Netbird:     "0.28.0",
			NodeUpdater: "1.0.0",
			Enforcement: "1.0.0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	prior, err := healthattest.ObserveAttestation(w.layout.Journal)
	if err != nil {
		t.Fatal(err)
	}
	if err := healthattest.WriteChallenge(w.layout.Journal, ch); err != nil {
		t.Fatal(err)
	}
	nc.Now = func() time.Time { return w.now.Add(10 * time.Millisecond) }
	nc.Wait = 80 * time.Millisecond
	nc.Poll = 10 * time.Millisecond

	// First attestation written by candidate agent during warm-up has tunnel_probe=failed.
	warming := goodAttest(ch, api.PublicationIntentUnpublished)
	warming.Checks.TunnelProbe.Status = api.HealthCheckStatusFailed
	warming.Checks.TunnelProbe.Reason = "overlay_down"
	if err := healthattest.WriteAttestation(w.layout.Journal, warming); err != nil {
		t.Fatal(err)
	}

	// On the first poll sleep (10ms into an 80ms wait window), the candidate's
	// tunnel comes up and it rewrites health-attest.json with all checks passed.
	nc.Sleep = func(_ context.Context, _ time.Duration) error {
		ready := goodAttest(ch, api.PublicationIntentUnpublished)
		ready.GeneratedAt = ch.IssuedAt.Add(5 * time.Millisecond)
		return healthattest.WriteAttestation(w.layout.Journal, ready)
	}
	nc.Bind(ch, prior, "candidate")

	view := w.eng.viewFor(context.Background(), next.id.String(), &next.manifest)
	st, err := nc.TunnelProbe(context.Background(), view)
	if err != nil {
		t.Fatal(err)
	}
	if st != api.HealthCheckStatusPassed {
		t.Fatalf("NodeChecks.poll aborted on initial warm-up attestation instead of polling within Wait window: got tunnel_probe=%q, want %q", st, api.HealthCheckStatusPassed)
	}
}

func TestUPD_D053_Review_PollDeadlineIgnoresInjectedClock(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	node := Node{
		ID:          uuid.Must(uuid.NewV7()),
		Incarnation: uuid.Must(uuid.NewV7()),
		BootID:      uuid.Must(uuid.NewV7()),
	}
	curNow := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	ch, err := healthattest.Issue(healthattest.Input{
		TxnID:         uuid.Must(uuid.NewV7()).String(),
		ReleaseID:     uuid.Must(uuid.NewV7()).String(),
		NodeID:        node.ID.String(),
		IncarnationID: node.Incarnation.String(),
		BootID:        node.BootID.String(),
		Now:           curNow,
		TTL:           45 * time.Second,
		Components: healthattest.Components{
			NodeAgent:   "1.0.0",
			Netbird:     "0.28.0",
			NodeUpdater: "1.0.0",
			Enforcement: "1.0.0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := healthattest.WriteChallenge(dir, ch); err != nil {
		t.Fatal(err)
	}

	sleeps := 0
	nc := &NodeChecks{
		Journal: dir,
		Node:    node,
		Wait:    30 * time.Second,
		Poll:    10 * time.Second,
		Now:     func() time.Time { return curNow },
		Sleep: func(_ context.Context, d time.Duration) error {
			sleeps++
			curNow = curNow.Add(d)
			if sleeps > 5 {
				return context.DeadlineExceeded
			}
			return nil
		},
		rt: &checkRT{},
	}
	nc.Bind(ch, healthattest.Observed{}, "candidate")
	_, _ = nc.TunnelProbe(context.Background(), ReleaseView{ReleaseID: ch.ReleaseID})
	// With Wait=30s and Poll=10s, advancing curNow by 10s per sleep should exit
	// after 3 sleeps (30s of simulated clock time), not spin past Wait until
	// Sleep force-aborts at sleeps=6.
	if sleeps > 3 {
		t.Fatalf("NodeChecks.poll ignored injected Now() clock for wait deadline: slept %d times (simulated %s > Wait 30s)", sleeps, time.Duration(sleeps)*10*time.Second)
	}
}

func TestUPD_D053_Review_CrashInStepRollbackAfterSetLastGoodLiesRecoveryRequired(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	touchPersistent(t, w.layout)

	rollbackChecks := 0
	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		if ch.ReleaseID == next.id.String() {
			// Candidate fails health check.
			at := goodAttest(ch, api.PublicationIntentPublished)
			at.Checks.TunnelProbe.Status = api.HealthCheckStatusFailed
			at.Checks.TunnelProbe.Reason = "overlay_down"
			return at, true
		}
		rollbackChecks++
		at := goodAttest(ch, api.PublicationIntentPublished)
		return at, true
	})

	work := w.work(next, "11", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)

	// Stop right when the transaction enters StageRollingBack.
	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage == string(updatejournal.StageRollingBack) {
			return errors.New("stop at rolling_back")
		}
		return nil
	}
	if err := w.run(); err == nil {
		t.Fatal("expected hook stop at rolling_back")
	}
	w.eng.Hooks.AfterStage = nil

	txn, err := w.eng.Journal.ActiveTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Simulate 2 prior interrupted rollback attempts so the next attempt is #3 (maxRollbackAttempts).
	counterPath := filepath.Join(w.layout.Journal, "rollback-"+txn.TxnID)
	if err := os.WriteFile(counterPath, []byte("2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Exercise the actual third attempt. LastGood and StageRolledBack now
	// commit together, so the first crash boundary after LastGood changes is
	// after stepRollback returns, before seal. Do not reconstruct the removed
	// intermediate state by calling SetPointers directly.
	c, err := w.eng.claimedFor(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.eng.stepRollback(context.Background(), txn, c); err != nil {
		t.Fatal(err)
	}
	if rollbackChecks != 1 {
		t.Fatalf("rollback checks = %d, want 1", rollbackChecks)
	}
	if err := w.eng.Close(); err != nil {
		t.Fatal(err)
	}
	w.eng.Journal = nil

	// Now the updater restarts and runs Recover(ctx).
	recErr := w.eng.Recover(context.Background())
	last, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p, _ := w.eng.Journal.Pointers(context.Background())
	res := w.oneResult()
	if last.Stage != updatejournal.StageRolledBack || res.State != api.UpdateResultStateRolledBack || requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("crash after rollback health passed & LastGood updated produced a lie on recovery: recErr=%v, txn.Stage=%s, pointers=%+v, result.State=%s, result.ErrorCode=%v, result.RecoveryRequired=%v",
			recErr, last.Stage, p, res.State, requiredValue(t, res.ErrorCode), requiredValue(t, res.RecoveryRequired))
	}
}

func TestUPD_D053_Review_CrashInStepCheckAfterSetStateSchemaBreaksRollback(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 2, []int{1, 2})
	w.seed(old)
	touchPersistent(t, w.layout)

	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		return goodAttest(ch, api.PublicationIntentPublished), true
	})

	work := w.work(next, "12", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)

	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage == string(updatejournal.StageChecking) {
			return errors.New("stop at checking")
		}
		return nil
	}
	if err := w.run(); err == nil {
		t.Fatal("expected hook stop at checking")
	}
	w.eng.Hooks.AfterStage = nil

	txn, err := w.eng.Journal.ActiveTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Run the actual eighth check. The schema and LastGood can only become
	// visible with StageLocallySucceeded. Crash before seal and reopen SQLite.
	checksPath := filepath.Join(w.layout.Journal, "checks-"+txn.TxnID)
	if err := os.WriteFile(checksPath, []byte("7\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := w.eng.stepCheck(context.Background(), txn); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Close(); err != nil {
		t.Fatal(err)
	}
	w.eng.Journal = nil

	recErr := w.eng.Recover(context.Background())
	last, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p, _ := w.eng.Journal.Pointers(context.Background())
	if last.Stage == updatejournal.StageRecoveryRequired {
		t.Fatalf("premature LastGood/StateSchema mutation in stepCheck before advance(StageLocallySucceeded) broke rollback: recErr=%v, stage=%s, code=%s, pointers=%+v",
			recErr, last.Stage, last.ErrorCode, p)
	}
	if last.Stage != updatejournal.StageLocallySucceeded || p.LastGood != next.id.String() {
		t.Fatalf("committed check lost on restart: stage=%s pointers=%+v", last.Stage, p)
	}
	schema, err := w.eng.Journal.StateSchema(context.Background())
	if err != nil || schema != 2 {
		t.Fatalf("schema=%d err=%v", schema, err)
	}

}

func TestUPD_D053_Review_Held_VersionTokenAndTTLClampCannotHideMismatch(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	touchPersistent(t, w.layout)

	// Force TTL <= Wait so attestWindow() clamps TTL = Wait + Poll.
	w.eng.Sleep = nil
	w.eng.Attest = AttestWindow{Wait: 30 * time.Millisecond, Poll: 5 * time.Millisecond, TTL: 10 * time.Millisecond}
	w.eng.Health = &NodeChecks{Journal: w.layout.Journal, Node: w.node, rt: &checkRT{}}
	w.eng.Services = &scriptedAgent{
		inner:   w.svc,
		journal: w.layout.Journal,
		answer: func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			at := goodAttest(ch, api.PublicationIntentPublished)
			if ch.ReleaseID == next.id.String() {
				// Candidate reports a slightly different NetBird daemonVersion than manifest.
				at.NetbirdVersion = ch.Components.Netbird + "-mismatch"
			}
			return at, true
		},
	}

	work := w.work(next, "20", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}

	// Candidate must be rejected with netbird_version_mismatch and rolled back to old.
	w.mustCurrent(old.id)
	p, err := w.eng.Journal.Pointers(context.Background())
	if err != nil || p.LastGood != old.id.String() || p.CurrentRelease != old.id.String() {
		t.Fatalf("pointers %+v err=%v", p, err)
	}
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if txn.Stage != updatejournal.StageRolledBack || txn.ErrorCode != string(apierr.HealthCheckFailed) {
		t.Fatalf("stage=%s code=%s", txn.Stage, txn.ErrorCode)
	}
	if !detailsContain(t, txn.TxnID, w, string(healthattest.ReasonNetbirdVersion)) {
		t.Fatalf("expected %s in journal events", healthattest.ReasonNetbirdVersion)
	}
	res := w.oneResult()
	if res.State != api.UpdateResultStateRolledBack || requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("result state=%s recovery=%v", res.State, requiredValue(t, res.RecoveryRequired))
	}
}

func TestUPD_D053_Review_Held_PartialWriteAndChallengeWriteFailureRollBack(t *testing.T) {
	t.Run("partial JSON write rolls back with malformed", func(t *testing.T) {
		w := newWorld(t)
		old := w.makeRelease("old", 1, []int{1})
		next := w.makeRelease("new", 1, []int{1})
		w.seed(old)
		touchPersistent(t, w.layout)

		useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			if ch.ReleaseID == next.id.String() {
				// Simulate a truncated/partial attestation file on disk.
				path := filepath.Join(w.layout.Journal, healthattest.AttestationFile)
				_ = os.WriteFile(path, []byte(`{"txn_id":"`+ch.TxnID), 0o600)
				return healthattest.Attestation{}, false
			}
			return goodAttest(ch, api.PublicationIntentPublished), true
		})

		work := w.work(next, "21", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{old.id, next.id}
		w.setWork(work)
		if err := w.run(); err != nil {
			t.Fatal(err)
		}
		w.mustCurrent(old.id)
		txn, err := w.eng.Journal.LastTxn(context.Background())
		if err != nil || txn.Stage != updatejournal.StageRolledBack || txn.ErrorCode != string(apierr.HealthCheckFailed) {
			t.Fatalf("txn=%+v err=%v", txn, err)
		}
		if !detailsContain(t, txn.TxnID, w, string(healthattest.ReasonUnreadable)) {
			t.Fatalf("expected %s in journal events", healthattest.ReasonUnreadable)
		}
	})

	t.Run("challenge write failure fails closed and rolls back", func(t *testing.T) {
		w := newWorld(t)
		old := w.makeRelease("old", 1, []int{1})
		next := w.makeRelease("new", 1, []int{1})
		w.seed(old)
		touchPersistent(t, w.layout)

		// Obstruct health-challenge.json with a non-empty directory so os.Rename in WriteChallenge fails,
		// then remove the obstruction when old release is restored during stepRollback.
		chPath := filepath.Join(w.layout.Journal, healthattest.ChallengeFile)
		if err := os.MkdirAll(filepath.Join(chPath, "block"), 0o700); err != nil {
			t.Fatal(err)
		}

		w.eng.Sleep = nil
		w.eng.Attest = AttestWindow{Wait: 20 * time.Millisecond, Poll: 5 * time.Millisecond, TTL: 2 * time.Second}
		w.eng.Health = &NodeChecks{Journal: w.layout.Journal, Node: w.node, rt: &checkRT{}}
		w.eng.Services = &scriptedAgent{
			inner:   w.svc,
			journal: w.layout.Journal,
			answer: func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
				return goodAttest(ch, api.PublicationIntentPublished), true
			},
		}
		w.eng.Hooks.AfterStage = func(stage string) error {
			if stage == string(updatejournal.StageRollingBack) {
				_ = os.RemoveAll(chPath)
			}
			return nil
		}

		work := w.work(next, "22", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{old.id, next.id}
		w.setWork(work)
		if err := w.run(); err != nil {
			t.Fatal(err)
		}
		w.mustCurrent(old.id)
		txn, err := w.eng.Journal.LastTxn(context.Background())
		if err != nil || txn.Stage != updatejournal.StageRolledBack || txn.ErrorCode != string(apierr.HealthCheckFailed) {
			t.Fatalf("txn=%+v err=%v", txn, err)
		}
		if !detailsContain(t, txn.TxnID, w, string(healthattest.ReasonUnreadable)) {
			t.Fatalf("expected %s in journal events", healthattest.ReasonUnreadable)
		}
	})
}

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
	// TODO(review): When the currently running release `cur` is revoked via
	// SignedRevoked, rollbackIfRevoked calls restoreRelease(ctx, target)
	// (revoke.go:97, 238-272). Unlike stepRollback (apply.go:287),
	// restoreRelease switches the `current` symlink to `target`, updates
	// Pointers.CurrentRelease to `target`, stops `cur`, and starts `target`,
	// but NEVER calls e.issueChallenge (or e.consumeChallenge) before
	// e.runHealth (revoke.go:264).
	//
	// Under the production NodeChecks health adapter (host.go:151):
	// 1. NodeChecks.evaluate (attest.go:131) sees bound == nil and immediately
	//    fails runHealth with ReasonMissing ("attestation_missing"), so
	//    rolling back from a revoked release to a healthy retained release
	//    ALWAYS fails with ROLLBACK_FAILED.
	// 2. Worse (Rollback Truth lie): when restoreRelease returns
	//    ROLLBACK_FAILED, rollbackIfRevoked calls blockRevoked (revoke.go:98,
	//    108-130) without stopping `target` or restoring Pointers. On disk
	//    `current` points to `target` and `target` is left running, in the
	//    journal CurrentRelease is `target` while LastGood is still the
	//    revoked `cur`, and blockRevoked enqueues an UpdateResult reporting
	//    State=failed, Stage=rollback, ErrorCode=ROLLBACK_FAILED,
	//    RecoveryRequired=true, CurrentReleaseId=target, and
	//    PreviousReleaseId=target (lying that PreviousReleaseId was `target`
	//    instead of `cur`).
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
	// TODO(review): NodeChecks.poll (attest.go:142-158) breaks out of the
	// wait loop as soon as readOnce returns `got != nil` (line 148), and
	// readOnce (attest.go:195-199) returns `got = &at` whenever the challenge
	// binding is valid (`reason == ""` or `reason == ReasonNotRun`), even if
	// one of the agent's checks is still `failed` or `not_run` while the
	// candidate is warming up.
	//
	// If the candidate agent writes an initial attestation at t=2ms where
	// tunnel_probe is still `failed` (or `not_run`) and then rewrites
	// health-attest.json at t=10ms with all checks `passed` (well inside the
	// Wait window), poll aborts immediately on the first read instead of
	// polling until all checks pass or the Wait window expires.
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
	// TODO(review): NodeChecks.poll (attest.go:143, 151) computes
	// `deadline := time.Now().Add(n.wait())` and checks
	// `!time.Now().Before(deadline)` using the host wall clock `time.Now()`,
	// while readOnce validates challenge expiry against the injected clock
	// `n.now()` (attest.go:195). When `n.Sleep` advances the injected clock
	// `n.Now` by `Wait`, `poll` ignores `n.now()` and keeps spinning until
	// wall-clock `time.Now()` reaches `deadline` (or ctx is canceled).
	dir := t.TempDir()
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
	// TODO(review): In stepRollback (apply.go:240-322), on the 3rd rollback
	// attempt (`maxRollbackAttempts == 3`), the updater switches `current` to
	// `target` (`old`), starts `target`, runs `runHealth` (which passes), and
	// updates `Pointers{LastGood: target}` (apply.go:276-315) BEFORE advancing
	// the journal stage to `StageRolledBack` (apply.go:320).
	//
	// If the process crashes or errors at `snapshotPointers` (line 317) or
	// before `advance(StageRolledBack)` (line 320), the transaction stage in
	// SQLite remains `StageRollingBack`, while on disk `current` points to
	// `old`, `old` is running and passed health attestation, and `Pointers`
	// has `CurrentRelease == old` and `LastGood == old`.
	//
	// On recovery (`Recover`), `stepRollback` increments `rollback-<txn>` to 4
	// (`> maxRollbackAttempts`), immediately transitions to
	// `StageRecoveryRequired` (`ROLLBACK_FAILED`), keeps member forwarding
	// blocked, and enqueues an upstream `UpdateResult` with `State=failed`,
	// `ErrorCode=ROLLBACK_FAILED`, `RecoveryRequired=true`, and
	// `CurrentReleaseId=old` — lying that rollback failed when `old` is
	// actually restored, healthy, and running.
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

	// Execute all of stepRollback attempt #3 up through runHealth passing and
	// SetPointers(LastGood: old), then simulate a crash right before
	// advance(StageRolledBack) by making immutable.snap uncreatable ONLY on the
	// post-health snapshotPointers call (after health-challenge.json has been
	// consumed by consumeChallenge at line 307).
	// Wait: snapshotPointers runs twice in stepRollback: once at line 300
	// (while health-challenge.json exists) and once at line 317 (after
	// consumeChallenge removed health-challenge.json and SetPointers(LastGood)
	// succeeded!).
	rel, err := w.layout.relativeRelease(old.id.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(w.layout.Current); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, w.layout.Current); err != nil {
		t.Fatal(err)
	}
	// Run stepRollback attempt #3 up to SetPointers(LastGood: old), crashing before advance(StageRolledBack).
	if _, err := bumpCounter(counterPath); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Journal.SetPointers(context.Background(), updatejournal.Pointers{CurrentRelease: old.id.String()}); err != nil {
		t.Fatal(err)
	}
	_ = w.eng.Services.Stop(context.Background())
	if err := w.eng.issueChallenge(context.Background(), txn, old.id.String(), "rollback"); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Services.Start(context.Background(), w.layout.releaseDir(old.id.String())); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.snapshotPointers(context.Background()); err != nil {
		t.Fatal(err)
	}
	view := w.eng.viewFor(context.Background(), old.id.String(), w.eng.manifestOf(context.Background(), old.id.String()))
	h, ok := w.eng.runHealth(context.Background(), view)
	w.eng.recordHealth(context.Background(), txn.TxnID, h)
	w.eng.consumeChallenge()
	if !ok {
		t.Fatalf("expected rollback health check on old release to pass, got %+v", h)
	}
	if err := w.eng.Journal.SetPointers(context.Background(), updatejournal.Pointers{LastGood: old.id.String()}); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.snapshotPointers(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Process crashes here, right before line 320 `e.advance(ctx, txn, StageRolledBack, ...)`.

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
	// TODO(review): In stepCheck (apply.go:227-237), the updater mutates
	// `Pointers{LastGood: txn.ReleaseID}` (line 227) and
	// `SetStateSchema(schema)` (line 234) BEFORE committing
	// `advance(StageLocallySucceeded)` (line 237).
	//
	// If the process crashes after lines 227-235 and before line 237 on check
	// attempt 8 (`maxCheckAttempts`), the durable stage in SQLite is still
	// `StageChecking`, but `LastGood` has already been overwritten with the
	// candidate `txn.ReleaseID` and `StateSchema` has already been updated to
	// `txn.StateSchemaTo` (e.g. schema 2).
	//
	// On `Recover`, `stepCheck` increments `checks-<txn>` to 9
	// (`> maxCheckAttempts`) and enters `StageRollingBack`. When `pickRollback`
	// evaluates `old` (which has `StateSchemaTo: 1`, while candidate had
	// `StateSchemaTo: 2` and `StateSchemaSupported: [1, 2]`), `evalLastGood`
	// (health.go:154: `cur == rel.StateSchemaTo`) sees `cur == 2 != 1` and
	// rejects `old` as `SCHEMA_INCOMPATIBLE`, forcing `StageRecoveryRequired`
	// while leaving the unsealed candidate `next` as `CurrentRelease` and
	// `LastGood`!
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
	// Simulate crash on attempt 8 after SetPointers(LastGood: next) and
	// SetStateSchema(2), right before advance(StageLocallySucceeded).
	checksPath := filepath.Join(w.layout.Journal, "checks-"+txn.TxnID)
	if err := os.WriteFile(checksPath, []byte("8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Journal.SetPointers(context.Background(), updatejournal.Pointers{LastGood: next.id.String()}); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Journal.SetStateSchema(context.Background(), 2); err != nil {
		t.Fatal(err)
	}

	// Recover now runs stepCheck (attempt 9 > maxCheckAttempts) -> beginRollback -> stepRollback.
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

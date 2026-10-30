package updater

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestUPD_D053_CommitFailureDoesNotPromoteLastGoodOrSchema(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		stage, terminal := updatejournal.StageChecking, updatejournal.StageLocallySucceeded
		if rollback {
			stage, terminal = updatejournal.StageRollingBack, updatejournal.StageRolledBack
		}
		t.Run(string(terminal), func(t *testing.T) {
			w := newWorld(t)
			w.releaseSeq = "1"
			old := w.makeRelease("old", 1, []int{1})
			w.releaseSeq = "2"
			retained := w.makeRelease("retained", 1, []int{1})
			w.releaseSeq = "3"
			next := w.makeRelease("new", 2, []int{1, 2})
			w.keep(retained)
			w.seed(old)
			touchPersistent(t, w.layout)
			useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
				at := goodAttest(ch, api.PublicationIntentPublished)
				if rollback && ch.ReleaseID == next.id.String() {
					at.Checks.TunnelProbe.Status = api.HealthCheckStatusFailed
				}
				return at, true
			})
			work := w.work(next, "10", api.PublicationIntentPublished)
			work.Serving = []uuid.UUID{old.id, retained.id, next.id}
			w.setWork(work)
			w.eng.Hooks.AfterStage = func(s string) error {
				if s == string(stage) {
					return errors.New("stop before health")
				}
				return nil
			}
			if err := w.run(); err == nil {
				t.Fatal("expected stop before health")
			}
			w.eng.Hooks.AfterStage = nil
			txn, err := w.eng.Journal.ActiveTxn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			schemaBefore, err := w.eng.Journal.StateSchema(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite", filepath.Join(w.layout.Journal, updatejournal.FileName))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(`CREATE TRIGGER fail_terminal BEFORE UPDATE OF stage ON txns
				WHEN NEW.stage IN ('locally_succeeded', 'rolled_back') BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
				t.Fatal(err)
			}
			if _, err := w.eng.perform(context.Background(), txn); err == nil {
				t.Fatal("expected terminal commit failure")
			}
			p, err := w.eng.Journal.Pointers(context.Background())
			if err != nil || p.LastGood != old.id.String() {
				t.Fatalf("uncommitted health changed LastGood: %+v %v", p, err)
			}
			schema, err := w.eng.Journal.StateSchema(context.Background())
			if err != nil || schema != schemaBefore {
				t.Fatalf("uncommitted health changed schema: %d %v", schema, err)
			}
			active, err := w.eng.Journal.ActiveTxn(context.Background())
			if err != nil || active.Stage != stage {
				t.Fatalf("stage=%s err=%v", active.Stage, err)
			}
			if _, err := db.Exec(`DROP TRIGGER fail_terminal`); err != nil {
				t.Fatal(err)
			}
			if err := w.eng.Recover(context.Background()); err != nil {
				t.Fatal(err)
			}
			last, err := w.eng.Journal.LastTxn(context.Background())
			if err != nil || last.Stage != terminal {
				t.Fatalf("retry stage=%s err=%v", last.Stage, err)
			}
		})
	}
}

type recoveryServices struct {
	*scriptedAgent
	t       *testing.T
	layout  Layout
	prior   string
	running bool
	stops   int
}

func (s *recoveryServices) Stop(ctx context.Context) error {
	if s.stops == 0 {
		id, err := s.layout.currentID()
		if err != nil || id != s.prior {
			s.t.Fatalf("current changed before revoked service stopped: %s %v", id, err)
		}
	}
	s.stops++
	s.running = false
	return s.scriptedAgent.Stop(ctx)
}

func (s *recoveryServices) Start(ctx context.Context, dir string) error {
	if s.stops == 0 {
		s.t.Fatal("restored release started before revoked service stopped")
	}
	if _, _, err := healthattest.ReadChallenge(s.layout.Journal); err != nil {
		s.t.Fatal("restored release started without challenge", err)
	}
	s.running = true
	return s.scriptedAgent.Start(ctx, dir)
}

func TestUPD_D053_RevocationFailureStopsTargetAndReportsPrevious(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	cur := w.makeRelease("current", 1, []int{1})
	w.keep(old)
	w.seed(cur)
	touchPersistent(t, w.layout)
	var challenges []healthattest.Challenge
	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		challenges = append(challenges, ch)
		at := goodAttest(ch, api.PublicationIntentPublished)
		at.Checks.TunnelProbe.Status = api.HealthCheckStatusFailed
		at.Checks.TunnelProbe.Reason = "overlay_down"
		return at, true
	})
	svc := &recoveryServices{scriptedAgent: w.eng.Services.(*scriptedAgent), t: t, layout: w.layout, prior: cur.id.String(), running: true}
	w.eng.Services = svc
	work := w.work(cur, "8", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, cur.id}
	revoked := []uuid.UUID{cur.id}
	work.SignedRevoked = &revoked
	w.setWork(work)
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	if svc.running || svc.stops != 2 {
		t.Fatalf("unverified target running=%v stops=%d", svc.running, svc.stops)
	}
	if len(challenges) != 1 || challenges[0].ReleaseID != old.id.String() || challenges[0].TxnID == work.Intent.JobId.String() {
		t.Fatalf("recovery challenge=%+v", challenges)
	}
	w.mustCurrent(old.id)
	p, err := w.eng.Journal.Pointers(context.Background())
	if err != nil || p.CurrentRelease != old.id.String() || p.LastGood != cur.id.String() {
		t.Fatalf("pointers=%+v err=%v", p, err)
	}
	res := w.oneResult()
	if requiredValue(t, res.PreviousReleaseId) != cur.id || requiredValue(t, res.CurrentReleaseId) != old.id ||
		requiredValue(t, res.ErrorCode) != api.ErrorCode(apierr.RollbackFailed) || !requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("untruthful recovery result: %+v", res)
	}
	if _, _, err := healthattest.ReadChallenge(w.layout.Journal); !os.IsNotExist(err) {
		t.Fatal("recovery challenge not consumed", err)
	}
	if blocked, _, _ := w.gate.snapshot(); !blocked {
		t.Fatal("failed recovery unblocked members")
	}
}

func TestUPD_D053_LayoutRejectsUnsafeJournal(t *testing.T) {
	for _, kind := range []string{"symlink", "file", "shared"} {
		t.Run(kind, func(t *testing.T) {
			w := newWorld(t)
			// Use a fresh path; the existing journal belongs to newWorld.
			w.layout.Journal = filepath.Join(t.TempDir(), "journal")
			want := healthattest.ReasonBadMode
			switch kind {
			case "symlink":
				want = healthattest.ReasonSymlink
				if err := os.Symlink(t.TempDir(), w.layout.Journal); err != nil {
					t.Fatal(err)
				}
			case "file":
				want = healthattest.ReasonMalformed
				if err := os.WriteFile(w.layout.Journal, nil, 0o600); err != nil {
					t.Fatal(err)
				}
			case "shared":
				if err := os.Mkdir(w.layout.Journal, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(w.layout.Journal, 0o750); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.layout.Ensure(); healthattest.ReasonOf(err) != want {
				t.Fatalf("Ensure=%v, want %s", err, want)
			}
		})
	}
}

func TestUPD_D053_PollRetainsLastValidFailure(t *testing.T) {
	for _, status := range []api.HealthCheckStatus{api.HealthCheckStatusFailed, api.HealthCheckStatusNotRun} {
		t.Run(string(status), func(t *testing.T) {
			w := newWorld(t)
			next := w.makeRelease("next", 1, []int{1})
			touchPersistent(t, w.layout)
			useAttest(w, nil)
			txnID := uuid.Must(uuid.NewV7()).String()
			ch, err := healthattest.Issue(healthattest.Input{
				TxnID: txnID, ReleaseID: next.id.String(), NodeID: w.node.ID.String(),
				IncarnationID: w.node.Incarnation.String(), BootID: w.node.BootID.String(),
				Now: w.now, TTL: time.Minute, Components: healthattest.Components{
					NodeAgent: next.manifest.Components.NodeAgent, Netbird: next.manifest.Components.Netbird,
					NodeUpdater: next.manifest.Components.NodeUpdater, Enforcement: next.manifest.Components.Enforcement,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := healthattest.WriteChallenge(w.layout.Journal, ch); err != nil {
				t.Fatal(err)
			}
			at := goodAttest(ch, api.PublicationIntentUnpublished)
			at.Checks.DNSPolicy.Status = status
			if err := healthattest.WriteAttestation(w.layout.Journal, at); err != nil {
				t.Fatal(err)
			}
			now := w.now.Add(time.Second)
			sleeps := 0
			nc := w.eng.Health.(*NodeChecks)
			nc.Now = func() time.Time { return now }
			nc.Wait, nc.Poll = 30*time.Second, 10*time.Second
			nc.Sleep = func(_ context.Context, d time.Duration) error {
				sleeps++
				now = now.Add(d)
				// A partial replacement must not erase the last valid failure.
				return os.WriteFile(filepath.Join(w.layout.Journal, healthattest.AttestationFile), []byte("{"), 0o600)
			}
			snap := nc.poll(context.Background(), ReleaseView{ReleaseID: ch.ReleaseID}, ch, healthattest.Observed{}, "candidate")
			if sleeps != 3 || snap.detail.AgentDNS != string(status) || snap.health.DnsPolicy != api.HealthCheckStatusFailed {
				t.Fatalf("sleeps=%d detail=%+v", sleeps, snap.detail)
			}
		})
	}
}

func TestUPD_D053_UntrustedReasonNeverReachesJournal(t *testing.T) {
	w := newWorld(t)
	next := w.makeRelease("next", 1, []int{1})
	touchPersistent(t, w.layout)
	const untrusted = "https://member.example.com/private"
	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		at := goodAttest(ch, api.PublicationIntentUnpublished)
		at.Checks.DNSPolicy.Reason = untrusted
		return at, true
	})
	w.setWork(w.work(next, "9", api.PublicationIntentUnpublished))
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !detailsContain(t, txn.TxnID, w, string(healthattest.ReasonBadReason)) {
		t.Fatal("missing fixed rejection reason")
	}
	events, err := w.eng.Journal.TxnEvents(context.Background(), txn.TxnID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if strings.Contains(event.Detail, untrusted) {
			t.Fatal("untrusted check reason leaked into journal")
		}
	}
}

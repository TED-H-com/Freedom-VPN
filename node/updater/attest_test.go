package updater

import (
	"context"
	"encoding/json"
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

// scriptedAgent is the candidate the updater starts. It answers the live
// challenge when answer says so, and it does not answer a Start that runs
// before the challenge exists.
type scriptedAgent struct {
	inner   ServiceManager
	journal string
	answer  func(healthattest.Challenge) (healthattest.Attestation, bool)
}

func (s *scriptedAgent) Stop(ctx context.Context) error {
	if s.inner == nil {
		return nil
	}
	return s.inner.Stop(ctx)
}

func (s *scriptedAgent) Start(ctx context.Context, dir string) error {
	if s.inner != nil {
		if err := s.inner.Start(ctx, dir); err != nil {
			return err
		}
	}
	ch, _, err := healthattest.ReadChallenge(s.journal)
	if err != nil {
		return nil
	}
	if s.answer == nil {
		return nil
	}
	at, ok := s.answer(ch)
	if !ok {
		return nil
	}
	return healthattest.WriteAttestation(s.journal, at)
}

func goodAttest(ch healthattest.Challenge, published api.PublicationIntent) healthattest.Attestation {
	if published == "" {
		published = api.PublicationIntentUnpublished
	}
	return healthattest.Attestation{
		TxnID: ch.TxnID, Nonce: ch.Nonce, ReleaseID: ch.ReleaseID, NodeID: ch.NodeID,
		AgentVersion: ch.Components.NodeAgent, NetbirdVersion: ch.Components.Netbird,
		BootID: ch.BootID, IncarnationID: ch.IncarnationID,
		GeneratedAt: ch.IssuedAt.Add(time.Millisecond), Published: published,
		Checks: healthattest.Checks{
			InternetOnly: healthattest.Check{Status: api.HealthCheckStatusPassed, Reason: "baseline held"},
			DNSPolicy:    healthattest.Check{Status: api.HealthCheckStatusPassed, Reason: "policy loaded"},
			TunnelProbe:  healthattest.Check{Status: api.HealthCheckStatusPassed, Reason: "probe reached the lab endpoint"},
		},
	}
}

func useAttest(w *world, answer func(healthattest.Challenge) (healthattest.Attestation, bool)) {
	w.eng.Sleep = nil
	w.eng.Attest = AttestWindow{Wait: 40 * time.Millisecond, Poll: 5 * time.Millisecond, TTL: 2 * time.Second}
	w.eng.Health = &NodeChecks{Journal: w.layout.Journal, Node: w.node, rt: &checkRT{}}
	w.eng.Services = &scriptedAgent{inner: w.svc, journal: w.layout.Journal, answer: answer}
}

func TestUPD_D053_UnpublishedInstallSetsLastGood(t *testing.T) {
	w := newWorld(t)
	// The process running this updater is not the candidate's updater binary.
	w.eng.Platform.UpdaterVersion = "1.3.0"
	next := w.makeRelease("new", 1, []int{1})
	touchPersistent(t, w.layout)
	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		// The attestation claims the node is published. That must not open the gate.
		return goodAttest(ch, api.PublicationIntentPublished), true
	})
	work := w.work(next, "1", api.PublicationIntentUnpublished)
	work.Serving = []uuid.UUID{next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(next.id)
	p, err := w.eng.Journal.Pointers(context.Background())
	if err != nil || p.LastGood != next.id.String() {
		t.Fatalf("last_good %s %v", p.LastGood, err)
	}
	res := w.oneResult()
	if res.State != api.UpdateResultStateSucceeded || res.MemberForwardingEnabled || res.PublicationIntent != api.PublicationIntentUnpublished {
		t.Fatalf("result state %s forwarding %v intent %s", res.State, res.MemberForwardingEnabled, res.PublicationIntent)
	}
	blocked, _, unblocks := w.gate.snapshot()
	if !blocked || unblocks != 0 {
		t.Fatalf("gate blocked %v unblocks %d", blocked, unblocks)
	}
	if on, why := w.eng.Journal.BlocksForwarding(context.Background()); !on {
		t.Fatal("journal released an unpublished node", why)
	}
	if _, _, err := healthattest.ReadChallenge(w.layout.Journal); !os.IsNotExist(err) {
		t.Fatal("challenge still live", err)
	}
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil || !strings.Contains(string(txn.Health), `"tunnel_probe":"passed"`) {
		t.Fatalf("health %s %v", txn.Health, err)
	}
	if !detailsContain(t, txn.TxnID, w, `"updater_version_check":"tree_hash_only"`) {
		t.Fatal("updater version check was not recorded")
	}
}

func TestUPD_D053_PublishedIntentFollowsServingRules(t *testing.T) {
	t.Run("in serving set", func(t *testing.T) {
		w := newWorld(t)
		next := w.makeRelease("new", 1, []int{1})
		touchPersistent(t, w.layout)
		useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			return goodAttest(ch, api.PublicationIntentUnpublished), true
		})
		work := w.work(next, "2", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{next.id}
		w.setWork(work)
		if err := w.run(); err != nil {
			t.Fatal(err)
		}
		p, err := w.eng.Journal.Pointers(context.Background())
		if err != nil || p.LastGood != next.id.String() {
			t.Fatalf("last_good %s %v", p.LastGood, err)
		}
		res := w.oneResult()
		if res.State != api.UpdateResultStateSucceeded || !res.MemberForwardingEnabled {
			t.Fatalf("state %s forwarding %v", res.State, res.MemberForwardingEnabled)
		}
		blocked, _, unblocks := w.gate.snapshot()
		if blocked || unblocks == 0 {
			t.Fatalf("gate blocked %v unblocks %d", blocked, unblocks)
		}
	})
	t.Run("outside serving set", func(t *testing.T) {
		w := newWorld(t)
		next := w.makeRelease("new", 1, []int{1})
		touchPersistent(t, w.layout)
		useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			return goodAttest(ch, api.PublicationIntentPublished), true
		})
		work := w.work(next, "3", api.PublicationIntentPublished)
		work.Serving = nil
		w.setWork(work)
		if err := w.run(); err != nil {
			t.Fatal(err)
		}
		p, err := w.eng.Journal.Pointers(context.Background())
		if err != nil || p.LastGood != next.id.String() || p.CurrentRelease != next.id.String() {
			t.Fatalf("pointers %+v %v", p, err)
		}
		res := w.oneResult()
		if res.State != api.UpdateResultStateSucceeded || res.MemberForwardingEnabled {
			t.Fatalf("state %s forwarding %v", res.State, res.MemberForwardingEnabled)
		}
		if on, _ := w.eng.Journal.BlocksForwarding(context.Background()); !on {
			t.Fatal("serving rules released the gate")
		}
	})
}

func TestUPD_D053_MissingAttestationRecovers(t *testing.T) {
	w := newWorld(t)
	next := w.makeRelease("new", 1, []int{1})
	touchPersistent(t, w.layout)
	useAttest(w, func(healthattest.Challenge) (healthattest.Attestation, bool) {
		return healthattest.Attestation{}, false
	})
	work := w.work(next, "4", api.PublicationIntentUnpublished)
	work.Serving = []uuid.UUID{next.id}
	w.setWork(work)
	err := w.run()
	if !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	txn := expectRecovery(t, w)
	if !strings.Contains(string(txn.Health), `"internet_only":"failed"`) ||
		!strings.Contains(string(txn.Health), `"dns_policy":"failed"`) ||
		!strings.Contains(string(txn.Health), `"tunnel_probe":"failed"`) ||
		!strings.Contains(string(txn.Health), `"processes":"failed"`) ||
		!strings.Contains(string(txn.Health), `"identity_and_state":"passed"`) {
		t.Fatalf("health %s", txn.Health)
	}
	if !detailsContain(t, txn.TxnID, w, string(healthattest.ReasonMissing)) {
		t.Fatal("missing attestation was not journaled")
	}
	ev, err := w.eng.Journal.TxnEvents(context.Background(), txn.TxnID)
	if err != nil {
		t.Fatal(err)
	}
	var detailBeforeTerminal bool
	seenDetail := false
	for _, e := range ev {
		if strings.Contains(e.Detail, string(healthattest.ReasonMissing)) {
			seenDetail = true
		}
		if seenDetail && e.Stage == updatejournal.StageRecoveryRequired {
			detailBeforeTerminal = true
		}
	}
	if !detailBeforeTerminal {
		t.Fatalf("diagnostic was not kept beside recovery: %+v", ev)
	}
	p, _ := w.eng.Journal.Pointers(context.Background())
	if p.LastGood != "" || txn.RollbackReleaseID != "" {
		t.Fatalf("retained last_good %s rollback %s", p.LastGood, txn.RollbackReleaseID)
	}
	if p.CurrentRelease != next.id.String() {
		t.Fatalf("current moved to %s", p.CurrentRelease)
	}
}

func TestUPD_D053_AttestationRejections(t *testing.T) {
	nextBoot := uuid.Must(uuid.NewV7()).String()
	cases := []struct {
		name   string
		reason string
		answer func(healthattest.Challenge) (healthattest.Attestation, bool)
		health func(string) bool
	}{
		{"wrong nonce", string(healthattest.ReasonNonceMismatch), func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			at := goodAttest(ch, api.PublicationIntentUnpublished)
			at.Nonce = strings.Repeat("ab", 32)
			return at, true
		}, allProbesFailed},
		{"wrong release", string(healthattest.ReasonReleaseMismatch), func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			at := goodAttest(ch, api.PublicationIntentUnpublished)
			at.ReleaseID = uuid.Must(uuid.NewV7()).String()
			return at, true
		}, allProbesFailed},
		{"previous boot", string(healthattest.ReasonBootMismatch), func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			at := goodAttest(ch, api.PublicationIntentUnpublished)
			at.BootID = nextBoot
			return at, true
		}, allProbesFailed},
		{"expired", string(healthattest.ReasonExpired), func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			at := goodAttest(ch, api.PublicationIntentUnpublished)
			at.GeneratedAt = ch.ExpiresAt
			return at, true
		}, allProbesFailed},
		{"version mismatch", string(healthattest.ReasonAgentVersion), func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			at := goodAttest(ch, api.PublicationIntentUnpublished)
			at.AgentVersion = "9.9.9"
			return at, true
		}, allProbesFailed},
		{"not_run", string(healthattest.ReasonNotRun), func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
			at := goodAttest(ch, api.PublicationIntentUnpublished)
			at.Checks.TunnelProbe.Status = api.HealthCheckStatusNotRun
			at.Checks.TunnelProbe.Reason = "probe skipped"
			return at, true
		}, func(h string) bool {
			return strings.Contains(h, `"tunnel_probe":"failed"`) &&
				strings.Contains(h, `"internet_only":"passed"`) &&
				strings.Contains(h, `"dns_policy":"passed"`) &&
				strings.Contains(h, `"processes":"passed"`)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t)
			next := w.makeRelease("new", 1, []int{1})
			touchPersistent(t, w.layout)
			useAttest(w, tc.answer)
			work := w.work(next, "5", api.PublicationIntentUnpublished)
			work.Serving = []uuid.UUID{next.id}
			w.setWork(work)
			if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
				t.Fatal(err)
			}
			txn := expectRecovery(t, w)
			if !tc.health(string(txn.Health)) {
				t.Fatalf("health %s", txn.Health)
			}
			if !detailsContain(t, txn.TxnID, w, tc.reason) {
				t.Fatal("reason not journaled", tc.reason)
			}
		})
	}
}

func TestUPD_D053_PreseededAttestationIsIgnored(t *testing.T) {
	w := newWorld(t)
	next := w.makeRelease("new", 1, []int{1})
	touchPersistent(t, w.layout)
	seeded := filepath.Join(w.layout.Journal, healthattest.AttestationFile)
	if err := os.WriteFile(seeded, []byte("{\"status\":\"passed\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(seeded, past, past); err != nil {
		t.Fatal(err)
	}
	useAttest(w, func(healthattest.Challenge) (healthattest.Attestation, bool) {
		return healthattest.Attestation{}, false
	})
	work := w.work(next, "6", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{next.id}
	w.setWork(work)
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	txn := expectRecovery(t, w)
	if !allProbesFailed(string(txn.Health)) {
		t.Fatalf("health %s", txn.Health)
	}
	if !detailsContain(t, txn.TxnID, w, string(healthattest.ReasonPreseeded)) {
		t.Fatal("pre-seeded file was not journaled")
	}
}

func TestUPD_D053_RollbackAnswersItsOwnChallenge(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	touchPersistent(t, w.layout)
	useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
		at := goodAttest(ch, api.PublicationIntentPublished)
		if ch.ReleaseID == next.id.String() {
			at.Checks.TunnelProbe.Status = api.HealthCheckStatusFailed
			at.Checks.TunnelProbe.Reason = "probe refused"
		}
		return at, true
	})
	work := w.work(next, "7", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(old.id)
	p, err := w.eng.Journal.Pointers(context.Background())
	if err != nil || p.LastGood != old.id.String() {
		t.Fatalf("last_good %s %v", p.LastGood, err)
	}
	res := w.oneResult()
	if res.State != api.UpdateResultStateRolledBack || requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("state %s recovery %v", res.State, requiredValue(t, res.RecoveryRequired))
	}
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if txn.Stage != updatejournal.StageRolledBack || txn.ErrorCode != string(apierr.HealthCheckFailed) {
		t.Fatalf("stage %s code %s", txn.Stage, txn.ErrorCode)
	}
	if !strings.Contains(string(txn.CandidateHealth), `"tunnel_probe":"failed"`) {
		t.Fatalf("candidate health was replaced: %s", txn.CandidateHealth)
	}
	if !detailsContain(t, txn.TxnID, w, `"phase":"rollback"`) || !detailsContain(t, txn.TxnID, w, `"phase":"candidate"`) {
		t.Fatal("rollback health was not recorded beside the candidate failure")
	}
	if !detailsContain(t, txn.TxnID, w, `"tunnel_probe":"passed"`) {
		t.Fatal("restored release did not pass its own probe")
	}
}

func TestUPD_D053_ConfigCannotPassChecks(t *testing.T) {
	t.Setenv("FVPN_HEALTH_CHECKS_PASSED", "1")
	root := t.TempDir()
	trustPath := filepath.Join(root, "trust.json")
	writeTrust(t, trustPath)
	cfg := Config{
		Layout: testLayout(root), NodeID: uuid.Must(uuid.NewV7()),
		IncarnationID: uuid.Must(uuid.NewV7()), BootID: uuid.Must(uuid.NewV7()),
		Platform: testPlatform(), ReleaseTrust: trustPath, DesiredTrust: trustPath,
		DesiredState: filepath.Join(root, "desired.json"), MeterCheckpoint: filepath.Join(root, "meter.json"),
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	bad := append(append([]byte{}, b[:len(b)-1]...), []byte(`,"health_checks":{"internet_only":"passed","dns_policy":"passed","tunnel_probe":"passed"}}`)...)
	path := filepath.Join(root, "updater.json")
	if err := os.WriteFile(path, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("config declared the checks passed")
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := loaded.Engine()
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	if err := os.MkdirAll(loaded.Layout.Journal, 0o700); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(loaded.Layout.Journal, "health-passed.json")
	if err := os.WriteFile(sidecar, []byte(`{"internet_only":"passed","dns_policy":"passed","tunnel_probe":"passed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	nc, ok := eng.Health.(*NodeChecks)
	if !ok {
		t.Fatalf("health %T", eng.Health)
	}
	for _, st := range []api.HealthCheckStatus{
		mustStatus(nc.Isolation(context.Background(), ReleaseView{})),
		mustStatus(nc.DNSPolicy(context.Background(), ReleaseView{})),
		mustStatus(nc.TunnelProbe(context.Background(), ReleaseView{})),
	} {
		if st != api.HealthCheckStatusFailed {
			t.Fatal(st)
		}
	}
}

func expectRecovery(t *testing.T, w *world) updatejournal.Txn {
	t.Helper()
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if txn.Stage != updatejournal.StageRecoveryRequired || txn.ErrorCode != string(apierr.RollbackFailed) {
		t.Fatalf("stage %s code %s detail %s", txn.Stage, txn.ErrorCode, txn.ErrorDetail)
	}
	if on, _ := w.eng.Journal.BlocksForwarding(context.Background()); !on {
		t.Fatal("recovery released the gate")
	}
	return txn
}

func allProbesFailed(h string) bool {
	return strings.Contains(h, `"internet_only":"failed"`) &&
		strings.Contains(h, `"dns_policy":"failed"`) &&
		strings.Contains(h, `"tunnel_probe":"failed"`) &&
		strings.Contains(h, `"processes":"failed"`)
}

func detailsContain(t *testing.T, txnID string, w *world, needle string) bool {
	t.Helper()
	ev, err := w.eng.Journal.TxnEvents(context.Background(), txnID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ev {
		if strings.Contains(e.Detail, needle) {
			return true
		}
	}
	return false
}

package agent

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/google/uuid"
)

type memPlane struct {
	baselines int
	applies   int
	last      enforce.Model
}

func (m *memPlane) Apply(_ context.Context, model enforce.Model, _ map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	plan, err := enforce.Render(model, enforce.State{}, enforce.TCState{})
	if err != nil {
		return enforce.ApplyResult{}, err
	}
	m.applies++
	m.last = model
	return enforce.ApplyResult{Plan: plan}, nil
}

func (m *memPlane) Baseline(context.Context, enforce.Datapath) error {
	m.baselines++
	return nil
}

// staticSource is a counter reader with no kernel. Boot id matches the
// process so a quiet open does not look like a reboot.
type staticSource struct{ boot string }

func (s staticSource) Read(context.Context) (meter.Observation, error) {
	return meter.Observation{Counters: map[string]enforce.CounterValue{}, BootID: s.boot}, nil
}

type rig struct {
	eng   *Engine
	srv   *fakecontrol.Server
	clock *timeanchor.Manual
	plane *memPlane
	log   *bytes.Buffer
	dir   string
}

func newRig(t *testing.T) *rig {
	t.Helper()
	srv, err := fakecontrol.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	srv.AddToken("token-test-only")
	srv.StickyLease(true)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ca.pem"), srv.CACertPEM(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trust.json"), srv.TrustJSON(), 0o644); err != nil {
		t.Fatal(err)
	}
	boot := uuid.Must(uuid.NewV7())
	if err := os.WriteFile(filepath.Join(dir, "boot_id"), []byte(boot.String()+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		StateDir: filepath.Join(dir, "state"), IdentityDir: filepath.Join(dir, "identity"),
		JournalDir: filepath.Join(dir, "journal"), LockPath: filepath.Join(dir, "update.lock"),
		ControlURL: srv.URL, ServerCAFile: filepath.Join(dir, "ca.pem"), TrustBundle: filepath.Join(dir, "trust.json"),
		DesiredStatePath: filepath.Join(dir, "state", "desired-state.json"),
		HeartbeatPath:    filepath.Join(dir, "heartbeat"),
		Datapath:         DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"},
		AgentRelease:     "0.0.0-test", ReleaseID: "019a0000-0000-7000-8000-0000000000c1",
		RenewLeadSeconds: 120, HeartbeatSeconds: 10, WatchdogStaleSeconds: 30,
		PollSeconds: 10, ObservationSeconds: 30, NetbirdBackend: "kernel",
		HostReserveBPS: 1_000_000, BootIDFile: filepath.Join(dir, "boot_id"),
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
	clock := timeanchor.NewManual(time.Now().UTC(), time.Hour)
	plane := &memPlane{}
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	eng, err := Start(Options{Config: cfg, Clock: clock, Plane: plane, Log: log, MeterSource: staticSource{boot: boot.String()}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = eng.Close() })
	return &rig{eng: eng, srv: srv, clock: clock, plane: plane, log: &buf, dir: dir}
}

func TestReadTokenIgnoresEnvironment(t *testing.T) {
	t.Setenv("FVPN_ENROLLMENT_TOKEN", "from-env")
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadToken(path)
	if err != nil || got != "from-file" {
		t.Fatal(got, err)
	}
	if err := NeutralizeToken(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestEnroll_CrashBeforeSendReusesRequest(t *testing.T) {
	r := newRig(t)
	r.eng.hooks.AfterEnrollmentPersist = func() error { return errors.New("crash before send") }
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err == nil {
		t.Fatal("enroll survived the crash hook")
	}
	if n := r.srv.EnrollCount(); n != 0 {
		t.Fatalf("requests sent before the crash hook: %d", n)
	}
	key, err := os.ReadFile(r.eng.cfg.keyPath())
	if err != nil {
		t.Fatal(err)
	}
	r.eng.hooks.AfterEnrollmentPersist = nil
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if n := r.srv.EnrollCount(); n != 1 {
		t.Fatalf("enroll count %d", n)
	}
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if n := r.srv.EnrollCount(); n != 1 {
		t.Fatalf("second identity: enroll count %d", n)
	}
	again, err := os.ReadFile(r.eng.cfg.keyPath())
	if err != nil || !bytes.Equal(key, again) {
		t.Fatal("identity key changed", err)
	}
	info, err := os.Stat(r.eng.cfg.keyPath())
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode %v", info.Mode())
	}
}

func TestEnroll_CrashAfterResponseKeepsCertificate(t *testing.T) {
	r := newRig(t)
	r.eng.hooks.AfterEnrollmentResponse = func() error { return errors.New("crash before certificate") }
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err == nil {
		t.Fatal("enroll survived the response hook")
	}
	if fileExists(r.eng.cfg.certPath()) {
		t.Fatal("certificate stored before the response hook returned")
	}
	r.eng.hooks.AfterEnrollmentResponse = nil
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if n := r.srv.EnrollCount(); n != 2 {
		t.Fatalf("enroll count %d, want the same request retried", n)
	}
	if !r.eng.enrolled || r.eng.nodeID == uuid.Nil {
		t.Fatal("not enrolled")
	}
}

func TestStep_LeaseGatesAnd304(t *testing.T) {
	r := newRig(t)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || r.plane.applies != 0 || r.plane.baselines < 1 {
		t.Fatalf("forwarded before a lease: serving %v applies %d baselines %d", r.eng.serving, r.plane.applies, r.plane.baselines)
	}
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving {
		t.Fatalf("not serving: %v %v\n%s", r.eng.decision.Reasons, r.eng.decision.Errors, r.log.String())
	}
	if r.srv.Observations() < 1 {
		t.Fatal("observation was not accepted")
	}
	if r.plane.applies != 1 || r.plane.last.Generation == 0 {
		t.Fatalf("apply %d gen %d", r.plane.applies, r.plane.last.Generation)
	}
	if _, err := os.Stat(r.eng.cfg.DesiredStatePath); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(r.eng.cfg.setupKeyPath())
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatal(info, err)
	}
	secret, err := os.ReadFile(r.eng.cfg.setupKeyPath())
	if err != nil || len(secret) == 0 {
		t.Fatal(err)
	}
	logs := r.log.String()
	if bytes.Contains(r.log.Bytes(), secret) || bytes.Contains(r.log.Bytes(), []byte("token-test-only")) {
		t.Fatalf("secret in log: %s", logs)
	}
	mono, id := r.eng.leaseMono, r.eng.lease.LeaseId
	exp := r.eng.lease.ExpiresAt
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.leaseMono != mono || r.eng.lease.LeaseId != id || !r.eng.lease.ExpiresAt.Equal(exp) {
		t.Fatal("304 or lease replay moved the lease")
	}
	if !r.eng.serving {
		t.Fatal(r.eng.decision.Reasons)
	}
}

func TestStep_RejectsBadDesiredState(t *testing.T) {
	r := newRig(t)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(context.Background()); err != nil || r.eng.Step(context.Background()) != nil || !r.eng.serving {
		t.Fatal(err, r.eng.decision.Reasons)
	}
	accepted := uint64(r.eng.desired.ConfigGeneration)
	r.srv.CorruptSignatures(true)
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving || uint64(r.eng.desired.ConfigGeneration) != accepted {
		t.Fatalf("corrupt signature replaced desired state: serving %v gen %d", r.eng.serving, r.eng.desired.ConfigGeneration)
	}
	r.srv.CorruptSignatures(false)
	r.srv.SetWrongIdentity(true, false)
	r.srv.BumpGeneration()
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if uint64(r.eng.desired.ConfigGeneration) != accepted {
		t.Fatalf("wrong node was accepted: gen %d", r.eng.desired.ConfigGeneration)
	}
	r.srv.SetWrongIdentity(false, false)
	r.srv.BumpGeneration()
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	newer := uint64(r.eng.desired.ConfigGeneration)
	if newer <= accepted {
		t.Fatalf("generation did not advance: %d", newer)
	}
	r.srv.SetGenerationDelta(-int64(newer))
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if uint64(r.eng.desired.ConfigGeneration) != newer || !r.eng.serving {
		t.Fatalf("older generation applied: %d serving %v", r.eng.desired.ConfigGeneration, r.eng.serving)
	}
}

func TestStep_WallJumpDoesNotExtendLease(t *testing.T) {
	r := newRig(t)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(context.Background()); err != nil || r.eng.Step(context.Background()) != nil || !r.eng.serving {
		t.Fatal(r.eng.decision.Reasons)
	}
	mono := r.eng.leaseMono
	r.clock.JumpWall(-time.Hour)
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving {
		t.Fatal("forwarding stayed up across a backward wall jump")
	}
	if r.eng.leaseMono != mono {
		t.Fatalf("mono_sent changed from %s to %s", mono, r.eng.leaseMono)
	}
	if r.plane.baselines < 2 {
		t.Fatal("fail closed did not install the baseline")
	}
}

func TestStep_RebootRequiresNewerLease(t *testing.T) {
	r := newRig(t)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(context.Background()); err != nil || r.eng.Step(context.Background()) != nil || !r.eng.serving {
		t.Fatal(r.eng.decision.Reasons)
	}
	mono := r.eng.leaseMono
	storePath := r.eng.store.Path()
	cfg := r.eng.cfg
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	next := uuid.Must(uuid.NewV7())
	if err := os.WriteFile(cfg.BootIDFile, []byte(next.String()+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := agentstore.Open(storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	plane := &memPlane{}
	eng, err := Start(Options{Config: cfg, Store: store, Clock: r.clock, Plane: plane, Log: slog.New(slog.DiscardHandler), MeterSource: staticSource{boot: next.String()}})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	if err := eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if eng.serving {
		t.Fatal("restart forwarded before a lease response")
	}
	if err := eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if eng.serving {
		t.Fatal("rebooted process forwarded on a replayed lease")
	}
	if eng.leaseMono != mono {
		t.Fatalf("reboot restarted mono_sent: %s -> %s", mono, eng.leaseMono)
	}
}

package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// Only service management is simulated: the test agent is already running.
// Recover, journal transitions, FileMeter, NodeChecks and attestations are real.
type f49Services struct{}

func (f49Services) Stop(context.Context) error          { return nil }
func (f49Services) Start(context.Context, string) error { return nil }

type f49InitialMeter struct{}

func (f49InitialMeter) FinalSample(context.Context) (updater.UsageCheckpoint, error) {
	return updater.UsageCheckpoint{ClosedMeterEpochID: uuid.NewString()}, nil
}
func (f49InitialMeter) OpenEpoch(context.Context) (string, error) { return uuid.NewString(), nil }

type f49RecoveryFixture struct {
	Layout updater.Layout
	Node   updater.Node
	Key    envelope.TrustedKey
	Now    time.Time
}

func (f f49RecoveryFixture) engine(t *testing.T) *updater.Engine {
	t.Helper()
	trust, err := envelope.NewTrustStore(f.Key)
	if err != nil {
		t.Fatal(err)
	}
	return &updater.Engine{
		Layout: f.Layout, Node: f.Node, ReleaseTrust: trust,
		Platform: updater.Platform{StateSchema: 1, UpdaterVersion: "0.1.0"},
		Gate:     updater.FileGate{Path: filepath.Join(f.Layout.Journal, "forwarding.blocked")},
		Meter:    updater.FileMeter{Path: filepath.Join(f.Layout.Journal, "meter.checkpoint")},
		Services: f49Services{}, Health: &updater.NodeChecks{},
		Now:    func() time.Time { return f.Now },
		Attest: updater.AttestWindow{Wait: 3 * time.Second, Poll: 10 * time.Millisecond, TTL: 30 * time.Second},
	}
}

// Executed as the child of the production recoverOnce command. Explicit
// argv marker keeps ordinary go test from invoking it; no environment seam.
func TestF49_RecoveryChild(t *testing.T) {
	if len(os.Args) < 4 || os.Args[len(os.Args)-4] != "f49-recover-child" {
		return
	}
	raw, err := os.ReadFile(os.Args[len(os.Args)-1])
	if err != nil {
		t.Fatal(err)
	}
	var f f49RecoveryFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	e := f.engine(t)
	err = e.Recover(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if cerr := e.Close(); cerr != nil {
		t.Fatal(cerr)
	}
	os.Exit(updater.ExitCode(err))
}

func f49PrepareTransaction(t *testing.T, r *rig, stage updatejournal.Stage) {
	t.Helper()
	cfg := r.eng.cfg
	f := f49RecoveryFixture{
		Layout: updater.Layout{Releases: filepath.Join(r.dir, "releases"), Current: filepath.Join(r.dir, "current"), Identity: cfg.IdentityDir, State: cfg.StateDir, Journal: cfg.JournalDir, Lock: cfg.LockPath, Staging: filepath.Join(r.dir, "staging"), Trust: filepath.Join(r.dir, "release-trust")},
		Node:   updater.Node{ID: r.eng.nodeID, Incarnation: r.eng.incID, BootID: r.eng.bootID}, Now: r.clock.Wall().Add(-time.Millisecond),
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	f.Key = envelope.TrustedKey{KeyID: "f49-test-only", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: f.Now.Add(-time.Hour), NotAfter: f.Now.Add(time.Hour)}
	e := f.engine(t)
	if err := e.Layout.Ensure(); err != nil {
		t.Fatal(err)
	}
	// Re-sign the schema-valid example with an ephemeral key and this candidate.
	raw, err := os.ReadFile("../../contracts/examples/valid/ReleaseManifestEnvelope.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Signed api.ReleaseManifest `json:"signed"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	m := fixture.Signed
	m.ReleaseId = cfg.ReleaseUUID()
	m.Components.NodeAgent = cfg.AgentRelease
	m.Components.NodeUpdater = "0.1.0"
	m.Components.Enforcement = "1"
	env, err := envelope.Sign(m, priv, f.Key.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = canonjson.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	sum := hex.EncodeToString(hash[:])
	if _, err := updater.VerifyManifest(raw, sum, e.ReleaseTrust, f.Now); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join(e.Layout.Releases, cfg.ReleaseID)
	if err := os.MkdirAll(rel, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rel, "TEST_ONLY"), []byte("test candidate\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tree, err := fsutil.TreeHash(rel)
	if err != nil {
		t.Fatal(err)
	}
	j, err := updatejournal.Open(e.Layout.Journal, updatejournal.Options{})
	if err != nil {
		t.Fatal(err)
	}
	e.Journal = j
	jobID := uuid.NewString()
	work := updater.Work{Intent: api.UpdateIntent{JobId: uuid.MustParse(jobID), Attempt: 1, Generation: 1, NodeId: f.Node.ID, IncarnationId: f.Node.Incarnation, ReleaseId: cfg.ReleaseUUID(), ManifestSha256: sum, ManifestUrl: "https://example.invalid/manifest", PublicationIntent: api.PublicationIntentPublished, IssuedAt: f.Now.Add(-time.Minute), AuthorizationValidUntil: f.Now.Add(time.Hour)}, Serving: []uuid.UUID{cfg.ReleaseUUID()}}
	claim, err := json.Marshal(struct {
		updater.Work
		BootID      uuid.UUID `json:"boot_id"`
		StateSchema int       `json:"state_schema"`
	}{work, f.Node.BootID, 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, _, err := j.ClaimJob(ctx, updatejournal.Job{JobID: jobID, Attempt: 1, Generation: 1, ReleaseID: cfg.ReleaseID, ManifestSHA256: sum, IntentJSON: claim}); err != nil {
		t.Fatal(err)
	}
	if err := j.PutRelease(ctx, updatejournal.Release{ReleaseID: cfg.ReleaseID, ManifestSHA256: sum, ManifestEnvelope: raw, TreeHash: tree, StateSchemaTo: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.BeginTxn(ctx, updatejournal.Txn{TxnID: uuid.NewString(), JobID: jobID, Attempt: 1, Generation: 1, ReleaseID: cfg.ReleaseID, ManifestSHA256: sum, StateSchemaTo: 1}); err != nil {
		t.Fatal(err)
	}
	// Run the actual transition graph up to the crash point. StageBlocked uses
	// FileMeter so its durable wait note and barrier are production artifacts.
	if stage != updatejournal.StageBlocked {
		e.Meter = f49InitialMeter{}
	}
	e.Hooks.AfterStage = func(s string) error {
		if s == string(stage) {
			return updater.ErrCrashed
		}
		return nil
	}
	if err := e.Recover(ctx); !errors.Is(err, updater.ErrCrashed) {
		t.Fatalf("prepare %s: %v", stage, err)
	}
	txn, err := j.ActiveTxn(ctx)
	if err != nil || txn.Stage != stage {
		t.Fatalf("crash stage: %+v %v", txn, err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(r.dir, "recover-fixture.json")
	if err := os.WriteFile(config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(r.dir, "recover-transaction")
	// Production passes recover -config PATH. Preserve those arguments after
	// the helper marker. Paths originate solely from disk-backed test temp dirs.
	body := fmt.Sprintf("#!/bin/sh\nexec %q -test.run '^TestF49_RecoveryChild$' -- f49-recover-child \"$@\" 2>%q\n", exe, filepath.Join(r.dir, "recover-child.log"))
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.UpdaterRecover, r.eng.cfg.UpdaterConfig = script, config
}

func TestD155_CrashedTransactionsKeepAgentRunning(t *testing.T) {
	for _, stage := range []updatejournal.Stage{updatejournal.StageBlocked, updatejournal.StageSwitched, updatejournal.StageChecking} {
		t.Run(string(stage), func(t *testing.T) {
			r := newRig(t)
			serveBoundGeneration(t, r)
			cfg := r.eng.cfg
			boot := r.eng.bootID
			if err := r.eng.Close(); err != nil {
				t.Fatal(err)
			}
			f49PrepareTransaction(t, r, stage)
			cfg.UpdaterRecover, cfg.UpdaterConfig = r.eng.cfg.UpdaterRecover, r.eng.cfg.UpdaterConfig
			cand, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, Log: r.eng.log, MeterSource: staticSource{boot: boot.String()}, Prober: &verProbe{version: "0.76.2"}})
			if err != nil {
				t.Fatal(err)
			}
			defer cand.Close()
			r.eng = cand
			if err := cand.bootWithBudget(context.Background(), 600*time.Millisecond); err != nil {
				t.Fatal(err)
			}
			if cand.serving || !cand.recoverBlocked || cand.recovered {
				t.Fatal("served before recovery result")
			}
			// No wall-clock advance in the agent fixture is needed to drive recovery.
			// At most one child; the agent alone writes its real meter/health evidence.
			deadline := time.Now().Add(5 * time.Second)
			for !cand.recovered && time.Now().Before(deadline) {
				if err := cand.Step(context.Background()); err != nil {
					t.Fatal(err)
				}
				if !cand.recovered && cand.serving {
					t.Fatal("served while child pending")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !cand.recovered || cand.recoverBlocked || cand.latchBlocked {
				log, _ := os.ReadFile(filepath.Join(r.dir, "recover-child.log"))
				t.Fatalf("%s recovery incomplete: recovered=%v blocked=%v latch=%v child=%s agent=%s", stage, cand.recovered, cand.recoverBlocked, cand.latchBlocked, log, r.log.String())
			}
			j, err := updatejournal.Open(cfg.JournalDir, updatejournal.Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			txn, err := j.LastTxn(context.Background())
			if err != nil || txn.Stage != updatejournal.StageLocallySucceeded {
				t.Fatalf("transaction did not finish: %+v %v", txn, err)
			}
			r.clock.Advance(provisionalRenewCap + time.Second)
			if err := cand.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !cand.serving {
				t.Fatalf("recovered candidate did not serve: %+v", cand.decision)
			}
		})
	}
}

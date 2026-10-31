package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

func TestReviewR17_ReleaseIdentityFollowsTheRunningBinaryAndJournal(t *testing.T) {
	r := newRig(t)
	before := r.eng.cfg.ReleaseID
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(exe)
	candidate := uuid.Must(uuid.NewV7())
	rel := filepath.Join(parent, "r17-review-"+candidate.String())
	if err := os.Mkdir(rel, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(rel) })
	if err := os.Link(exe, filepath.Join(rel, "fvpn-agent")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(r.eng.cfg.CurrentReleasePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, r.eng.cfg.CurrentReleasePath); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.loadActiveRelease(t.Context(), exe); !errors.Is(err, errActiveRelease) || r.eng.cfg.ReleaseID != before {
		t.Fatalf("candidate before the journal was adopted: %v release=%s", err, r.eng.cfg.ReleaseID)
	}

	old := filepath.Join(parent, "r17-review-old-"+uuid.Must(uuid.NewV7()).String())
	if err := os.Mkdir(old, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(old) })
	if err := os.WriteFile(filepath.Join(old, "fvpn-agent"), []byte("not-the-running-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(r.eng.cfg.CurrentReleasePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(old, r.eng.cfg.CurrentReleasePath); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.loadActiveRelease(t.Context(), exe); !errors.Is(err, errActiveRelease) || r.eng.cfg.ReleaseID != before {
		t.Fatalf("a different binary was attested: %v release=%s", err, r.eng.cfg.ReleaseID)
	}

	if err := os.MkdirAll(r.eng.cfg.JournalDir, 0o700); err != nil {
		t.Fatal(err)
	}
	nodeID, incID := uuid.New(), uuid.New()
	r.eng.enrolled = true
	r.eng.nodeID, r.eng.incID = nodeID, incID
	ch := healthattest.Challenge{
		TxnID: uuid.NewString(), Nonce: "aabbccdd", ReleaseID: candidate.String(),
		NodeID: nodeID.String(), IncarnationID: incID.String(), BootID: r.eng.bootID.String(),
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, ch); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.eng.matchingChallenge(); ok {
		t.Fatal("a challenge for another release matched the configured identity")
	}
	ch.ReleaseID = r.eng.cfg.ReleaseUUID().String()
	if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, ch); err != nil {
		t.Fatal(err)
	}
	got, ok := r.eng.matchingChallenge()
	if !ok || got.ReleaseID != before {
		t.Fatalf("matching challenge %+v ok=%v", got, ok)
	}
}

func TestReviewR17_OpenEpochDoesNotPublishCheckpoint(t *testing.T) {
	r := newRig(t)
	if _, err := os.Stat(r.eng.cfg.MeterCheckpoint); !os.IsNotExist(err) {
		t.Fatalf("checkpoint before a close: %v", err)
	}
	if r.eng.usage != nil {
		if _, err := r.eng.usage.ClosedCheckpoint(t.Context()); err == nil {
			t.Fatal("an open epoch already had a closed checkpoint")
		}
	}
	if _, err := (updater.FileMeter{Path: r.eng.cfg.MeterCheckpoint}).FinalSample(t.Context()); err == nil {
		t.Fatal("an open epoch published a meter checkpoint")
	}
}

package signing

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

type fileFact struct {
	Size     int64
	Mode     os.FileMode
	Modified time.Time
	Hash     [32]byte
}

func manifest(t *testing.T, dir string) map[string]fileFact {
	t.Helper()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]fileFact{
		".": {Size: info.Size(), Mode: info.Mode(), Modified: info.ModTime()},
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		var raw []byte
		if info.Mode().IsRegular() {
			raw, err = os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
		}
		out[e.Name()] = fileFact{info.Size(), info.Mode(), info.ModTime(), sha256.Sum256(raw)}
	}
	return out
}
func unchanged(t *testing.T, p *FileProvider, before map[string]fileFact) {
	t.Helper()
	if !reflect.DeepEqual(before, manifest(t, p.dir)) {
		t.Fatal("vault manifest changed")
	}
}
func snap(t *testing.T, p *FileProvider) FileSnapshot {
	t.Helper()
	s, err := p.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func checkChange(t *testing.T, r ChangeResult, err error, want ChangeOutcome) {
	t.Helper()
	if err != nil || r.Outcome != want {
		t.Fatalf("outcome=%v err=%v; want %v", r.Outcome, err, want)
	}
}
func seeded(t *testing.T, purpose Purpose) (*FileProvider, string) {
	t.Helper()
	p := vault(t)
	id, err := p.Rotate(context.Background(), purpose, time.Hour, 0)
	if err != nil {
		t.Fatal(err)
	}
	return p, id
}
func TestSEC01_ConditionalRotationPurposes(t *testing.T) {
	for _, purpose := range []Purpose{DesiredState, Lease, Recovery, LedgerEpoch} {
		t.Run(string(purpose), func(t *testing.T) {
			p := vault(t)
			ctx := context.Background()
			now := time.Now().UTC()
			p.now = func() time.Time { return now }
			first, err := p.RotateConditional(ctx, purpose, 0, "", time.Hour, time.Minute)
			checkChange(t, first, err, ChangeApplied)
			signer, err := ForPurpose(p, purpose)
			if err != nil {
				t.Fatal(err)
			}
			old, err := signer.Sign(ctx, map[string]bool{"test_only": true})
			if err != nil {
				t.Fatal(err)
			}
			oldRaw, err := old.MarshalCanonical()
			if err != nil {
				t.Fatal(err)
			}
			before := snap(t, p)
			now = now.Add(time.Minute)
			r, err := p.RotateConditional(ctx, purpose, 1, first.Key.ID, 2*time.Hour, 3*time.Minute)
			checkChange(t, r, err, ChangeApplied)
			if r.OldRevision != 1 || r.NewRevision != 2 || r.Key.State != "active" || r.Key.NotBefore != now || r.Key.NotAfter != now.Add(2*time.Hour) || r.Previous.ID != first.Key.ID || r.Previous.State != "retired" || r.Previous.NotAfter != now.Add(3*time.Minute) {
				t.Fatal("incorrect rotation facts")
			}
			got, err := signer.Sign(ctx, map[string]bool{"test_only": true})
			if err != nil || got.Signatures[0].KeyID != r.Key.ID {
				t.Fatal("cached signer failed to switch", err)
			}
			after := snap(t, p)
			pre, _ := json.Marshal(before.NodeTrust)
			post, _ := json.Marshal(after.NodeTrust)
			if purpose == LedgerEpoch {
				if !bytes.Equal(pre, post) {
					t.Fatal("ledger key changed node bundle")
				}
				return
			}
			ts, err := envelope.ParseNodeTrustStoreJSON(post)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := envelope.Verify(oldRaw, ts, envelope.Purpose(purpose), now); err != nil {
				t.Fatal("overlap rejected", err)
			}
			for _, at := range []time.Time{r.Previous.NotAfter, r.Previous.NotAfter.Add(time.Nanosecond)} {
				if _, _, err := envelope.Verify(oldRaw, ts, envelope.Purpose(purpose), at); err == nil {
					t.Fatal("expired key accepted")
				}
			}
			newRaw, err := got.MarshalCanonical()
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := envelope.Verify(newRaw, before.NodeTrust, envelope.Purpose(purpose), now); err == nil {
				t.Fatal("old bundle accepted unknown key")
			}
			if _, _, err := envelope.Verify(newRaw, ts, envelope.Purpose(purpose), now); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestSEC01_ConditionalWindowNeverGrows(t *testing.T) {
	for _, tc := range []struct {
		name             string
		elapsed, overlap time.Duration
	}{
		{"long overlap", time.Minute, 2 * time.Hour}, {"expired", 2 * time.Hour, time.Hour}, {"zero", time.Minute, 0}, {"same clock", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := vault(t)
			now := time.Now().UTC()
			p.now = func() time.Time { return now }
			ctx := context.Background()
			first, err := p.RotateConditional(ctx, Lease, 0, "", time.Hour, 0)
			checkChange(t, first, err, ChangeApplied)
			now = now.Add(tc.elapsed)
			r, err := p.RotateConditional(ctx, Lease, 1, first.Key.ID, time.Hour, tc.overlap)
			checkChange(t, r, err, ChangeApplied)
			want := now.Add(tc.overlap)
			if first.Key.NotAfter.Before(want) {
				want = first.Key.NotAfter
			}
			if !want.After(first.Key.NotBefore) {
				want = first.Key.NotBefore.Add(time.Nanosecond)
			}
			if !r.Previous.NotAfter.Equal(want) || r.Previous.NotAfter.After(first.Key.NotAfter) {
				t.Fatal("window grew or wrong cap")
			}
			if tc.name == "same clock" {
				r2, err := p.RotateConditional(ctx, Lease, 2, r.Key.ID, time.Nanosecond, 0)
				checkChange(t, r2, err, ChangeApplied)
				r3, err := p.RotateConditional(ctx, Lease, 3, r2.Key.ID, time.Hour, 0)
				checkChange(t, r3, err, ChangeApplied)
				if r3.Previous.NotAfter.Sub(r3.Previous.NotBefore) != time.Nanosecond || r3.Previous.NotAfter.After(r2.Key.NotAfter) {
					t.Fatal("one nanosecond window grew")
				}
			}
		})
	}
}
func TestSEC01_ConditionalRefusals(t *testing.T) {
	p, id := seeded(t, DesiredState)
	before := manifest(t, p.dir)
	for _, tc := range []struct {
		purpose           Purpose
		revision          decstr.U64
		active            string
		validity, overlap time.Duration
	}{
		{DesiredState, 0, id, time.Hour, 0}, {DesiredState, 1, uuid.NewString(), time.Hour, 0}, {DesiredState, 1, "", time.Hour, 0}, {Lease, 1, id, time.Hour, 0},
		{NodeCA, 1, "", time.Hour, 0}, {BootstrapSecret, 1, "", time.Hour, 0}, {Release, 1, "", time.Hour, 0}, {"unknown", 1, "", time.Hour, 0},
		{DesiredState, 1, id, 0, 0}, {DesiredState, 1, id, time.Hour, -1},
	} {
		r, err := p.RotateConditional(context.Background(), tc.purpose, tc.revision, tc.active, tc.validity, tc.overlap)
		checkChange(t, r, err, ChangeRefused)
		if !reflect.DeepEqual(r, ChangeResult{Outcome: ChangeRefused}) {
			t.Fatal("refusal disclosed vault contents")
		}
		unchanged(t, p, before)
	}
}
func TestSEC01_ConditionalRepetitionAndRevocation(t *testing.T) {
	ctx := context.Background()
	p, id := seeded(t, DesiredState)
	env, err := p.SignEnvelope(ctx, DesiredState, map[string]bool{"test_only": true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	before := manifest(t, p.dir)
	r, err := p.RevokeConditional(ctx, id, DesiredState, 1, false)
	checkChange(t, r, err, ChangeRefused)
	unchanged(t, p, before)
	for _, args := range []struct {
		id      string
		purpose Purpose
		rev     decstr.U64
	}{{uuid.NewString(), DesiredState, 1}, {id, Lease, 1}, {id, DesiredState, 0}} {
		r, err = p.RevokeConditional(ctx, args.id, args.purpose, args.rev, true)
		checkChange(t, r, err, ChangeRefused)
		unchanged(t, p, before)
	}
	r, err = p.RevokeConditional(ctx, id, DesiredState, 1, true)
	checkChange(t, r, err, ChangeApplied)
	if r.Previous.State != "active" || r.Key.State != "revoked" || r.Key.RevokedAt == nil || r.NewRevision != 2 {
		t.Fatal("wrong revocation facts")
	}
	at := *r.Key.RevokedAt
	before = manifest(t, p.dir)
	r, err = p.RevokeConditional(ctx, id, DesiredState, 0, false)
	checkChange(t, r, err, ChangeAlreadyRevoked)
	unchanged(t, p, before)
	if !r.Key.RevokedAt.Equal(at) {
		t.Fatal("revocation time changed")
	}
	if _, err := p.SignEnvelope(ctx, DesiredState, map[string]bool{"test_only": true}); err == nil {
		t.Fatal("revoked key signed")
	}
	ts := snap(t, p).NodeTrust
	if _, _, err := envelope.Verify(raw, ts, envelope.PurposeDesiredState, at); err == nil {
		t.Fatal("revoked old signature accepted")
	}
	next, err := p.RotateConditional(ctx, DesiredState, 2, "", time.Hour, 0)
	checkChange(t, next, err, ChangeApplied)
	if next.Previous != nil {
		t.Fatal("revoked key retired again")
	}
	before = manifest(t, p.dir)
	r, err = p.RotateConditional(ctx, DesiredState, 2, "", time.Hour, 0)
	checkChange(t, r, err, ChangeRefused)
	unchanged(t, p, before)
	got, err := p.SignEnvelope(ctx, DesiredState, map[string]bool{"test_only": true})
	if err != nil || got.Signatures[0].KeyID != next.Key.ID {
		t.Fatal("replacement not signing", err)
	}
	if k := snap(t, p).Keys[0]; k.State != "revoked" || !k.RevokedAt.Equal(at) {
		t.Fatal("revocation resurrected")
	}
	third, err := p.RotateConditional(ctx, DesiredState, 3, next.Key.ID, time.Hour, time.Minute)
	checkChange(t, third, err, ChangeApplied)
	r, err = p.RevokeConditional(ctx, next.Key.ID, DesiredState, 4, false)
	checkChange(t, r, err, ChangeApplied)
	if r.Previous.State != "retired" {
		t.Fatal("retired key not revoked")
	}
}
func TestSEC01_ConditionalForbiddenRevokeAndLegacy(t *testing.T) {
	ctx := context.Background()
	p := vault(t)
	for _, purpose := range []Purpose{NodeCA, BootstrapSecret} {
		id, err := p.Rotate(ctx, purpose, time.Hour, 0)
		if err != nil {
			t.Fatal(err)
		}
		before := manifest(t, p.dir)
		r, err := p.RevokeConditional(ctx, id, purpose, snap(t, p).Revision, true)
		checkChange(t, r, err, ChangeRefused)
		unchanged(t, p, before)
		if err := p.Revoke(ctx, id); err != nil {
			t.Fatal("legacy revoke refused", err)
		}
		before = manifest(t, p.dir)
		if err := p.Revoke(ctx, id); err != nil {
			t.Fatal(err)
		}
		unchanged(t, p, before)
		if _, err := p.Rotate(ctx, purpose, time.Hour, 0); err != nil {
			t.Fatal("legacy rotation refused", err)
		}
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	now := time.Now().UTC()
	if err := p.PinReleaseRoot(ctx, envelope.TrustedKey{KeyID: id, PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: now, NotAfter: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	before := manifest(t, p.dir)
	r, err := p.RevokeConditional(ctx, id, Release, snap(t, p).Revision, true)
	checkChange(t, r, err, ChangeRefused)
	unchanged(t, p, before)
}
func TestSEC01_ConditionalConcurrency(t *testing.T) {
	for _, raceRevoke := range []bool{false, true} {
		p, id := seeded(t, Lease)
		ctx := context.Background()
		start := make(chan struct{})
		var wg sync.WaitGroup
		results := make(chan ChangeOutcome, 8)
		errs := make(chan error, 8)
		for i := 0; i < 8; i++ {
			h, err := OpenFileProvider(p.dir)
			if err != nil {
				t.Fatal(err)
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				var r ChangeResult
				var err error
				if raceRevoke && i == 0 {
					r, err = h.RevokeConditional(ctx, id, Lease, 1, true)
				} else {
					r, err = h.RotateConditional(ctx, Lease, 1, id, time.Hour, 0)
				}
				results <- r.Outcome
				errs <- err
			}()
		}
		close(start)
		wg.Wait()
		close(results)
		close(errs)
		applied := 0
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		for r := range results {
			if r == ChangeApplied {
				applied++
			} else if r != ChangeRefused {
				t.Fatal(r)
			}
		}
		s := snap(t, p)
		if applied != 1 || s.Revision != 2 {
			t.Fatal("not exactly one change")
		}
		files, err := filepath.Glob(filepath.Join(p.dir, "*.key"))
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != len(s.Keys) || (!raceRevoke && len(files) != 2) {
			t.Fatal("extra key files")
		}
	}
}
func TestSEC01_ConditionalPartialFailure(t *testing.T) {
	for _, operation := range []string{"rotate", "revoke"} {
		for _, stage := range []string{"key", "before rename", "after rename"} {
			if operation == "revoke" && stage == "key" {
				continue
			}
			t.Run(operation+"/"+stage, func(t *testing.T) {
				p, id := seeded(t, Lease)
				before := manifest(t, p.dir)
				injected := errors.New("TEST ONLY injected failure")
				switch stage {
				case "key":
					p.conditionalHooks.beforeKey = func(id string) error { return os.Mkdir(filepath.Join(p.dir, id+".key"), 0700) }
				case "before rename":
					p.conditionalHooks.beforeRename = func() error { return injected }
				case "after rename":
					p.conditionalHooks.afterRename = func() error { return injected }
				}
				call := func() (ChangeResult, error) {
					if operation == "rotate" {
						return p.RotateConditional(context.Background(), Lease, 1, id, time.Hour, 0)
					}
					return p.RevokeConditional(context.Background(), id, Lease, 1, true)
				}
				r, err := call()
				want := ChangeNotApplied
				if stage == "after rename" {
					want = ChangeUncertain
				}
				checkChange(t, r, err, want)
				s := snap(t, p)
				if stage == "after rename" {
					if s.Revision != 2 || !reflect.DeepEqual(s.Keys[len(s.Keys)-1], *r.Key) {
						t.Fatal("replacement missing")
					}
				} else {
					if s.Revision != 1 || manifest(t, p.dir)["state.json"] != before["state.json"] {
						t.Fatal("ledger changed")
					}
					if operation == "rotate" && stage == "before rename" {
						if r.CreatedKeyID == "" {
							t.Fatal("orphan not reported")
						}
						if _, err := os.Stat(filepath.Join(p.dir, r.CreatedKeyID+".key")); err != nil {
							t.Fatal(err)
						}
					}
					if stage == "key" && r.CreatedKeyID != "" {
						t.Fatal("claimed exclusive creation")
					}
				}
				p.conditionalHooks = conditionalHooks{}
				again, err := call()
				want = ChangeApplied
				if stage == "after rename" {
					want = ChangeRefused
					if operation == "revoke" {
						want = ChangeAlreadyRevoked
					}
				}
				checkChange(t, again, err, want)
				active := 0
				for _, k := range snap(t, p).Keys {
					if k.State == "active" {
						active++
					}
				}
				if operation == "rotate" && active != 1 {
					t.Fatal("wrong active count")
				}
			})
		}
	}
}
func TestSEC01_ConditionalLockInUse(t *testing.T) {
	for _, operation := range []string{"rotate", "revoke"} {
		for _, mutation := range []string{"group", "symlink", "replace while held", "replace before"} {
			t.Run(operation+"/"+mutation, func(t *testing.T) {
				p, id := seeded(t, Lease)
				path := filepath.Join(p.dir, "lock")
				replace := func() {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, nil, 0600); err != nil {
						t.Fatal(err)
					}
				}
				switch mutation {
				case "group":
					if err := os.Chmod(path, 0640); err != nil {
						t.Fatal(err)
					}
				case "symlink":
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink("state.json", path); err != nil {
						t.Fatal(err)
					}
				case "replace while held":
					p.conditionalHooks.locked = replace
				case "replace before":
					replace()
				}
				before := manifest(t, p.dir)
				var r ChangeResult
				var err error
				if operation == "rotate" {
					r, err = p.RotateConditional(context.Background(), Lease, 1, id, time.Hour, 0)
				} else {
					r, err = p.RevokeConditional(context.Background(), id, Lease, 1, true)
				}
				if mutation == "replace before" {
					checkChange(t, r, err, ChangeApplied)
					return
				}
				var lockErr *LockValidationError
				if !errors.As(err, &lockErr) || r.Outcome != 0 {
					t.Fatalf("wrong lock result: %v %+v", err, r)
				}
				after := manifest(t, p.dir)
				if mutation == "replace while held" {
					// Only this fixture replaces an entry after taking the manifest.
					// Ignore that entry and the directory size/mtime it changes;
					// retain the directory mode and every ledger/key file fact.
					delete(before, "lock")
					delete(after, "lock")
					directory := after["."]
					directory.Size = before["."].Size
					directory.Modified = before["."].Modified
					after["."] = directory
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatal("ledger or keys changed")
				}
			})
		}
	}
}
func TestSEC01_ConditionalDeadline(t *testing.T) {
	for _, operation := range []string{"rotate", "revoke"} {
		for _, failSave := range []bool{false, true} {
			p, id := seeded(t, Lease)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			p.conditionalHooks.locked = func() { <-ctx.Done() }
			if failSave {
				p.conditionalHooks.afterRename = func() error { return errors.New("TEST ONLY after rename") }
			}
			var r ChangeResult
			var err error
			if operation == "rotate" {
				r, err = p.RotateConditional(ctx, Lease, 1, id, time.Hour, 0)
			} else {
				r, err = p.RevokeConditional(ctx, id, Lease, 1, true)
			}
			cancel()
			want := ChangeApplied
			if failSave {
				want = ChangeUncertain
			}
			checkChange(t, r, err, want)
			if snap(t, p).Revision != 2 {
				t.Fatal("deadline canceled locked change")
			}
		}
	}
	p, id := seeded(t, Lease)
	before := manifest(t, p.dir)
	f, err := os.OpenFile(filepath.Join(p.dir, "lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	r, err := p.RotateConditional(ctx, Lease, 1, id, time.Hour, 0)
	if !errors.Is(err, context.DeadlineExceeded) || r.Outcome != 0 {
		t.Fatal("missing deadline", err)
	}
	unchanged(t, p, before)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	r, err = p.RotateConditional(context.Background(), Lease, 1, id, time.Hour, 0)
	checkChange(t, r, err, ChangeApplied)
}
func TestSEC01_ConditionalPermissions(t *testing.T) {
	for _, mask := range []int{0, 0777} {
		p := vault(t)
		previous := syscall.Umask(mask)
		r, err := p.RotateConditional(context.Background(), Lease, 0, "", time.Hour, 0)
		syscall.Umask(previous)
		checkChange(t, r, err, ChangeApplied)
		info, err := os.Stat(filepath.Join(p.dir, r.Key.ID+".key"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatal("wrong private key permissions")
		}
	}
}

func TestSEC01_ProviderManifestDetectsDirectoryChange(t *testing.T) {
	p := vault(t)
	// A distinct prior timestamp avoids dependence on filesystem clock precision.
	prior := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(p.dir, prior, prior); err != nil {
		t.Fatal(err)
	}
	before := manifest(t, p.dir)
	path := filepath.Join(p.dir, "test-only-transient")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	after := manifest(t, p.dir)
	if reflect.DeepEqual(before, after) || before["."].Modified.Equal(after["."].Modified) {
		t.Fatal("manifest missed create/remove directory modification")
	}
	delete(before, ".")
	delete(after, ".")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("negative control changed a surviving file")
	}
}

func TestSEC01_ProviderLedgerOwnerPermissions(t *testing.T) {
	for _, operation := range []string{"conditional rotate", "conditional revoke", "legacy revoke"} {
		t.Run(operation, func(t *testing.T) {
			p, oldID := seeded(t, Lease)
			ctx := context.Background()
			wantSigner := oldID
			if operation != "conditional rotate" {
				var err error
				wantSigner, err = p.Rotate(ctx, Lease, time.Hour, time.Minute)
				if err != nil {
					t.Fatal(err)
				}
			}
			func() {
				previous := syscall.Umask(0777)
				defer syscall.Umask(previous)
				switch operation {
				case "conditional rotate":
					r, err := p.RotateConditional(ctx, Lease, 1, oldID, time.Hour, time.Minute)
					checkChange(t, r, err, ChangeApplied)
					wantSigner = r.Key.ID
				case "conditional revoke":
					r, err := p.RevokeConditional(ctx, oldID, Lease, 2, false)
					checkChange(t, r, err, ChangeApplied)
				case "legacy revoke":
					if err := p.Revoke(ctx, oldID); err != nil {
						t.Fatal(err)
					}
				}
			}()
			info, err := os.Stat(filepath.Join(p.dir, "state.json"))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("ledger mode = %04o; want 0600", info.Mode().Perm())
			}
			reopened, err := OpenFileProvider(p.dir)
			if err != nil {
				t.Fatal("cannot reopen saved vault", err)
			}
			s := snap(t, reopened)
			wantRevision := decstr.U64(3)
			if operation == "conditional rotate" {
				wantRevision = 2
			}
			if s.Revision != wantRevision {
				t.Fatal("wrong saved revision")
			}
			signed, err := reopened.SignEnvelope(ctx, Lease, map[string]bool{"test_only": true})
			if err != nil || signed.Signatures[0].KeyID != wantSigner {
				t.Fatal("expected signer unavailable after save", err)
			}
		})
	}
}

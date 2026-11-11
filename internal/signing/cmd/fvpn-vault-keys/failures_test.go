package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// Faults at the command/provider boundary complement the real save/locked
// section seams in signing/rotate_test.go. Successful underlying mutations
// still use the real provider; no exported provider test hook is needed.
type faultProvider struct {
	changeProvider
	mutationError                                error
	result                                       *signing.ChangeResult
	uncertain, readFail, different, waitDeadline bool
	reads, calls                                 int
	alterSnapshot                                func(*signing.FileSnapshot)
}

func (p *faultProvider) finish(ctx context.Context, r signing.ChangeResult, err error) (signing.ChangeResult, error) {
	p.calls++
	if p.waitDeadline {
		<-ctx.Done()
	}
	if err == nil && p.uncertain {
		r.Outcome = signing.ChangeUncertain
	}
	return r, err
}
func (p *faultProvider) RotateConditional(ctx context.Context, purpose signing.Purpose, revision decstr.U64, active string, validity, overlap time.Duration) (signing.ChangeResult, error) {
	if p.mutationError != nil {
		return signing.ChangeResult{}, p.mutationError
	}
	if p.result != nil {
		p.calls++
		return *p.result, nil
	}
	r, err := p.changeProvider.RotateConditional(ctx, purpose, revision, active, validity, overlap)
	return p.finish(ctx, r, err)
}
func (p *faultProvider) RevokeConditional(ctx context.Context, id string, purpose signing.Purpose, revision decstr.U64, active bool) (signing.ChangeResult, error) {
	if p.mutationError != nil {
		return signing.ChangeResult{}, p.mutationError
	}
	if p.result != nil {
		p.calls++
		return *p.result, nil
	}
	r, err := p.changeProvider.RevokeConditional(ctx, id, purpose, revision, active)
	return p.finish(ctx, r, err)
}
func (p *faultProvider) Snapshot(ctx context.Context) (signing.FileSnapshot, error) {
	p.reads++
	if p.readFail {
		return signing.FileSnapshot{}, signing.ErrUnavailable
	}
	s, err := p.changeProvider.Snapshot(ctx)
	if p.different {
		s.Revision++
	}
	if err == nil && p.alterSnapshot != nil {
		p.alterSnapshot(&s)
	}
	return s, err
}
func withProvider(p changeProvider) command {
	return command{open: func(context.Context, string) (changeProvider, error) { return p, nil }}
}

// TestSEC01_ConditionalPartialFailure injects the real save-boundary failures.
// This test exercises the command's handling of their public result classes.
func TestSEC01_CommandPartialFailureResultHandling(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		for _, stage := range []string{"no file", "orphan", "after rename", "read failure", "different snapshot"} {
			if op == "revoke" && stage == "orphan" {
				continue
			}
			t.Run(op+"/"+stage, func(t *testing.T) {
				f := setup(t, signing.Lease)
				p := &faultProvider{changeProvider: f.p}
				var orphan string
				want, outcome := exitNotApplied, "not_applied"
				switch stage {
				case "no file":
					p.result = &signing.ChangeResult{Outcome: signing.ChangeNotApplied}
				case "orphan":
					orphan = uuid.NewString()
					if err := os.WriteFile(filepath.Join(f.dir, orphan+".key"), []byte("TEST ONLY incomplete file"), 0600); err != nil {
						t.Fatal(err)
					}
					p.result = &signing.ChangeResult{Outcome: signing.ChangeNotApplied, CreatedKeyID: orphan}
				case "after rename":
					p.uncertain = true
					want, outcome = exitDurability, "durability_not_confirmed"
				case "read failure":
					p.uncertain, p.readFail = true, true
					want, outcome = exitUnknown, "unknown"
				case "different snapshot":
					p.uncertain, p.different = true, true
					want, outcome = exitUnknown, "unknown"
				}
				before := manifest(t, f.dir)
				d := invoke(t, withProvider(p), f.args(op), want, outcome)
				if d.UnreferencedKeyID != orphan {
					t.Fatal("wrong orphan id")
				}
				if p.calls != 1 {
					t.Fatal("retried mutation")
				}
				if p.result != nil {
					unchanged(t, f.dir, before)
					if p.reads != 0 {
						t.Fatal("unnecessary read-back")
					}
					success := "applied"
					if op == "revoke" {
						success = "revoked"
					}
					invoke(t, command{}, f.args(op), exitOK, success)
				} else {
					if p.reads != 1 {
						t.Fatal("read-back count")
					}
					if stage == "after rename" && d.ReadBack != "matched" {
						t.Fatal("missing confirmation")
					}
					if stage == "read failure" && d.ReadBack != "failed" {
						t.Fatal("wrong failure")
					}
					if stage == "different snapshot" && d.ReadBack != "different" {
						t.Fatal("inferred result")
					}
					before = manifest(t, f.dir)
					code, result := exitRefused, "precondition_refused"
					if op == "revoke" {
						code, result = exitAlreadyRevoked, "already_revoked"
					}
					invoke(t, command{}, f.args(op), code, result)
					unchanged(t, f.dir, before)
				}
				if getSnapshot(t, f.p).Revision != 2 {
					t.Fatal("wrong final revision")
				}
			})
		}
	}
}

// TestSEC01_ConditionalDeadline holds the actual provider lock past its deadline
// and injects the real post-rename failure. This command-boundary fixture delays
// a completed call's return and substitutes its result to test reporting and the
// fresh read-back deadline; it does not hold a lock or fail a filesystem save.
func TestSEC01_CommandExpiredContextResultHandling(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		for _, uncertain := range []bool{false, true} {
			f := setup(t, signing.Lease)
			p := &faultProvider{changeProvider: f.p, waitDeadline: true, uncertain: uncertain}
			code, result := exitOK, "applied"
			if op == "revoke" {
				result = "revoked"
			}
			if uncertain {
				code, result = exitDurability, "durability_not_confirmed"
			}
			invoke(t, withProvider(p), append(f.args(op), "-lock-timeout", "30ms"), code, result)
			if getSnapshot(t, f.p).Revision != 2 {
				t.Fatal("change canceled")
			}
			if uncertain && p.reads != 1 {
				t.Fatal("no fresh read-back")
			}
		}
	}
	// The first deadline is shared by open and mutation; a slow open does not
	// buy a new timeout for the conditional call.
	f := setup(t, signing.Lease)
	before := manifest(t, f.dir)
	c := command{open: func(ctx context.Context, dir string) (changeProvider, error) {
		p, err := signing.OpenFileProviderContext(ctx, dir)
		<-ctx.Done()
		return p, err
	}}
	invoke(t, c, append(f.args("rotate"), "-lock-timeout", "20ms"), exitTimeout, "")
	unchanged(t, f.dir, before)
}

type failingWriter struct{ short bool }

func (w failingWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) - 1, nil
	}
	return 0, errors.New("TEST ONLY private error text")
}
func TestSEC01_CommandOutputFailure(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		for _, short := range []bool{false, true} {
			f := setup(t, signing.Lease)
			var stderr bytes.Buffer
			code := run(f.args(op), failingWriter{short}, &stderr)
			outcome := "applied"
			if op == "revoke" {
				outcome = "revoked"
			}
			assertOutput(t, code, "", stderr.String(), exitOutput, "", "fvpn-vault-keys: result output failed: "+outcome+"; run fvpn-vault inspect; do not repeat blindly\n")
			if getSnapshot(t, f.p).Revision != 2 {
				t.Fatal("output failure undid change")
			}
			before := manifest(t, f.dir)
			code, outcome = exitRefused, "precondition_refused"
			if op == "revoke" {
				code, outcome = exitAlreadyRevoked, "already_revoked"
			}
			invoke(t, command{}, f.args(op), code, outcome)
			unchanged(t, f.dir, before)
		}
	}
	// Every result class, including refusal and uncertainty, names only its
	// fixed class when encoding/output fails.
	for _, outcome := range []string{"applied", "revoked", "already_revoked", "precondition_refused", "not_applied", "durability_not_confirmed", "unknown"} {
		var stderr bytes.Buffer
		code := writeResult(failingWriter{}, &stderr, document{Outcome: outcome}, 0)
		assertOutput(t, code, "", stderr.String(), exitOutput, "", "fvpn-vault-keys: result output failed: "+outcome+"; run fvpn-vault inspect; do not repeat blindly\n")
	}
}

func TestSEC01_CommandReadBackRequiresExactRecord(t *testing.T) {
	for _, operation := range []string{"rotate", "revoke"} {
		t.Run(operation, func(t *testing.T) {
			f := setup(t, signing.Lease)
			p := &faultProvider{changeProvider: f.p, uncertain: true}
			p.alterSnapshot = func(s *signing.FileSnapshot) {
				if s.Revision != 2 {
					t.Fatal("fixture must retain expected next revision")
				}
				if operation == "rotate" {
					// Keep an active key at the expected revision, but substitute its ID.
					for i := range s.Keys {
						if s.Keys[i].State == "active" {
							s.Keys[i].ID = uuid.NewString()
							return
						}
					}
				} else {
					// Same ID and revision, but a different recorded revocation time.
					for i := range s.Keys {
						if s.Keys[i].ID == f.id && s.Keys[i].RevokedAt != nil {
							at := s.Keys[i].RevokedAt.Add(time.Second)
							s.Keys[i].RevokedAt = &at
							return
						}
					}
				}
				t.Fatal("fixture did not alter the target record")
			}
			d := invoke(t, withProvider(p), f.args(operation), 8, "unknown")
			if d.ReadBack != "different" || d.Change != nil || p.calls != 1 || p.reads != 1 {
				t.Fatalf("wrong read-back handling: document=%+v calls=%d reads=%d", d, p.calls, p.reads)
			}
			if getSnapshot(t, f.p).Revision != 2 {
				t.Fatal("mutation retried")
			}
		})
	}
}

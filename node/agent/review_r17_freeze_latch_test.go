package agent

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestReviewR17_DirtyGenerationIsNotAcknowledged(t *testing.T) {
	r := newRig(t)
	now := time.Now().UTC()
	if err := r.eng.store.AcceptDesired(t.Context(), agentstore.Desired{
		Generation: 4, Envelope: []byte(`{}`), IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	r.eng.desired = &api.DesiredState{ConfigGeneration: decstr.U64(4)}
	r.eng.desiredOK = true
	r.eng.appliedGen = 3
	r.eng.noteApply(errors.New("foreign object still present"))
	if r.eng.applyError == nil {
		t.Fatal("noteApply cleared the failure")
	}
	r.eng.ackApplied(t.Context(), 4)
	if r.eng.appliedGen != 3 || r.eng.applyError == nil {
		t.Fatalf("dirty generation was acknowledged: applied=%d err=%v", r.eng.appliedGen, r.eng.applyError)
	}
}

func TestReviewR17_RetainedApplyErrorStillRenewsTheDesiredGeneration(t *testing.T) {
	r := newRig(t)
	now := time.Now().UTC()
	r.eng.lease = &api.ServingLease{
		LeaseId: uuid.New(), ConfigGeneration: 4, IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		Members: []api.MemberLease{{}},
	}
	r.eng.leaseFresh = true
	r.eng.anchorOK = true
	r.eng.leaseMono = r.clock.Mono()
	r.eng.leaseRetryAt = 0
	r.eng.desired = &api.DesiredState{ConfigGeneration: 5}
	r.eng.desiredOK = true
	r.eng.applyError = &api.NodeError{Code: api.ErrorCodeTCFOREIGNFILTER, Message: "dirty", ObservedAt: now}
	r.eng.appliedGen = 4
	if !r.eng.needRenew() {
		t.Fatal("a retained apply error suppressed renewal of a newer desired generation")
	}
	r.eng.desired.ConfigGeneration = 4
	if r.eng.needRenew() {
		t.Fatal("a fresh lease for the desired generation renewed only because applyError was set")
	}
}

func TestReviewR17_FreezeIntentAndMetaBothFailStaysLatched(t *testing.T) {
	r := newRig(t)
	// Production bumps the outbox attempt before it calls freezeIdentity.
	// Both durable writes can still fail. The attempted report has to keep
	// the next process from serving.
	reportID := uuid.New()
	if err := r.eng.store.Enqueue(t.Context(), reportID, []byte(`{}`), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.store.BumpAttempt(t.Context(), reportID, "in_flight"); err != nil {
		t.Fatal(err)
	}
	hold, err := sql.Open("sqlite", "file:"+r.eng.cfg.DBPath()+"?_pragma=busy_timeout(0)")
	if err != nil {
		t.Fatal(err)
	}
	hold.SetMaxOpenConns(1)
	conn, err := hold.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	released := false
	release := func() {
		if released {
			return
		}
		released = true
		_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		_ = conn.Close()
		_ = hold.Close()
	}
	defer release()
	if err = os.Chmod(r.eng.cfg.StateDir, 0o555); err != nil {
		t.Fatal(err)
	}
	restored := false
	restore := func() {
		if restored {
			return
		}
		restored = true
		_ = os.Chmod(r.eng.cfg.StateDir, 0o700)
	}
	defer restore()

	done := make(chan struct{})
	go func() {
		r.eng.freezeIdentity(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("freezeIdentity did not return")
	}
	restore()
	release()
	if !r.eng.identityFrozen {
		t.Fatal("in-process latch was not set")
	}
	restartEngine(t, r, r.plane)
	in := r.eng.input()
	if r.eng.serving || in.StorageOK && !r.eng.identityFrozen && !r.eng.freezeUncertain {
		t.Fatalf("TODO(review): restart lost a decided freeze when the intent file and identity_frozen meta both failed (frozen=%v uncertain=%v storageOK=%v serving=%v)", r.eng.identityFrozen, r.eng.freezeUncertain, in.StorageOK, r.eng.serving)
	}
}

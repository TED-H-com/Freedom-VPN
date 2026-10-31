package agent

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// TestReviewR16_FrozenIncarnationDoesNotAckOrKeepServing is the in-process
// latch: forwarding is withdrawn in the freeze call, and a generation ACK is
// refused while the memory latch is set.
func TestReviewR16_FrozenIncarnationDoesNotAckOrKeepServing(t *testing.T) {
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
	r.eng.freezeIdentity(t.Context())
	if r.eng.appliedGen != 3 {
		t.Fatalf("freeze ACKed generation %d", r.eng.appliedGen)
	}
	r.eng.ackApplied(t.Context(), 4)
	if r.eng.appliedGen != 3 || !r.eng.identityFrozen {
		t.Fatalf("ACK while frozen moved generation to %d frozen=%v", r.eng.appliedGen, r.eng.identityFrozen)
	}
	beat := mustBeat(t, r)
	if beat.Serving || beat.Reason != "identity_frozen" {
		t.Fatalf("heartbeat after freeze: serving=%v reason=%s", beat.Serving, beat.Reason)
	}
	if v, ok, err := r.eng.store.Meta(t.Context(), "identity_frozen"); err != nil || !ok || v != "1" {
		t.Fatal("durable latch", v, ok, err)
	}
	restartEngine(t, r, r.plane)
	if !r.eng.identityFrozen {
		t.Fatal("restart cleared a persisted latch")
	}
	r.eng.desired = &api.DesiredState{ConfigGeneration: decstr.U64(4)}
	r.eng.desiredOK = true
	before := r.eng.appliedGen
	r.eng.ackApplied(t.Context(), before+1)
	if r.eng.appliedGen != before {
		t.Fatalf("restarted frozen incarnation ACKed generation %d", r.eng.appliedGen)
	}
}

// TestReviewR16_FailedFreezePersistIsLostOnRestart holds the agent database
// write lock across freezeIdentity. SetMeta fails; the next process must
// recover the durable intent before it can enforce. The original proof
// asserted the vulnerable outcome; the final assertions now require recovery.
func TestReviewR16_FailedFreezePersistIsLostOnRestart(t *testing.T) {
	r := newRig(t)
	hold, err := sql.Open("sqlite", "file:"+r.eng.cfg.DBPath()+"?_pragma=busy_timeout(0)")
	if err != nil {
		t.Fatal(err)
	}
	hold.SetMaxOpenConns(1)
	conn, err := hold.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
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

	done := make(chan struct{})
	go func() {
		r.eng.freezeIdentity(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("freezeIdentity did not return while the database write lock was held")
	}
	if !r.eng.identityFrozen || r.eng.storageOK {
		t.Fatalf("in-process fail-closed missing: frozen=%v storageOK=%v", r.eng.identityFrozen, r.eng.storageOK)
	}
	// The memory latch makes a second call return before any write.
	r.eng.freezeIdentity(t.Context())
	release()
	v, ok, err := r.eng.store.Meta(t.Context(), "identity_frozen")
	if err != nil {
		t.Fatal(err)
	}
	if ok && v == "1" {
		restartEngine(t, r, r.plane)
		if !r.eng.identityFrozen {
			t.Fatal("write that landed during the lock was lost on restart")
		}
		t.Log("sqlite lock did not keep SetMeta from committing; durable restart still frozen")
		return
	}
	restartEngine(t, r, r.plane)
	if !r.eng.identityFrozen {
		t.Fatal("restart lost a decided freeze whose SQLite write failed")
	}
	if got, present, err := r.eng.store.Meta(t.Context(), "identity_frozen"); err != nil || !present || got != "1" {
		t.Fatalf("restart did not recover durable freeze intent: value=%q present=%v err=%v", got, present, err)
	}
	stepOK(t, r)
	if r.eng.serving || mustBeat(t, r).Serving {
		t.Fatal("restarted failed-persist freeze allowed forwarding")
	}
}

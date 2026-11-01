package agentstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestF34_AppliedSnapshotAndReservationAreAtomic(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC()
	if err := s.AcceptDesired(ctx, Desired{Generation: 3, Envelope: []byte(`{}`), IssuedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkAppliedSnapshot(ctx, 2, "old"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER fail_snapshot BEFORE INSERT ON meta WHEN NEW.key='applied_snapshot' BEGIN SELECT RAISE(FAIL,'TEST ONLY write failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkAppliedSnapshot(ctx, 3, "new"); err == nil {
		t.Fatal("snapshot failure hidden")
	}
	d, _, err := s.Desired(ctx)
	old, _, merr := s.Meta(ctx, AppliedSnapshotKey)
	if err != nil || merr != nil || d.Applied != 2 || old != "old" {
		t.Fatal("partial snapshot commit")
	}
	if _, err := s.db.Exec(`DROP TRIGGER fail_snapshot`); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkAppliedSnapshot(ctx, 3, "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER fail_reservation BEFORE INSERT ON meta WHEN NEW.key='conntrack_delete_attempts' BEGIN SELECT RAISE(FAIL,'TEST ONLY write failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMetaBatch(ctx, map[string]string{"pending_revocations": "TEST ONLY pending", "conntrack_delete_attempts": "1"}); err == nil {
		t.Fatal("reservation failure hidden")
	}
	for _, key := range []string{"pending_revocations", "conntrack_delete_attempts"} {
		if _, ok, err := s.Meta(ctx, key); err != nil || ok {
			t.Fatalf("partial reservation key=%s present=%v err=%v", key, ok, err)
		}
	}
	if _, err := s.db.Exec(`DROP TRIGGER fail_reservation`); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMetaBatch(ctx, map[string]string{"pending_revocations": "TEST ONLY pending", "conntrack_delete_attempts": "1"}); err != nil {
		t.Fatal(err)
	}
}

package agentstore

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAT13_F42_LeaseAndRenewalJitterCommitTogether(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC()
	first := Lease{ID: uuid.Must(uuid.NewV7()), BootID: uuid.Must(uuid.NewV7()), MonoSent: time.Hour, IssuedAt: now, HighestIssuedAt: now, ExpiresAt: now.Add(time.Hour), Envelope: []byte("TEST ONLY"), RenewalJitter: -12 * time.Second}
	if err := s.SaveLease(t.Context(), first); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER f42_jitter_fail BEFORE UPDATE ON meta WHEN NEW.key='lease_renew_jitter_ns' BEGIN SELECT RAISE(FAIL,'TEST ONLY jitter failure'); END`); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ID = uuid.Must(uuid.NewV7())
	second.RenewalJitter = 12 * time.Second
	if err := s.SaveLease(t.Context(), second); err == nil {
		t.Fatal("injected transaction succeeded")
	}
	stored, ok, err := s.Lease(t.Context())
	if err != nil || !ok || stored.ID != first.ID || stored.RenewalJitter != first.RenewalJitter {
		t.Fatal("partial lease commit", stored, err)
	}
	if _, err := s.db.Exec(`DELETE FROM meta WHERE key='lease_renew_jitter_ns'`); err != nil {
		t.Fatal(err)
	}
	stored, ok, err = s.Lease(t.Context())
	if err != nil || !ok || stored.RenewalJitter != 0 {
		t.Fatal("legacy lease did not use safe base", err)
	}
}

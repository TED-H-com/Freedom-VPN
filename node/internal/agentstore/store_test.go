package agentstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStoreRoundTripAndStaleGeneration(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	req, inc := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	if err := s.PutPendingEnrollment(ctx, req, inc, now); err != nil {
		t.Fatal(err)
	}
	if err := s.PutPendingEnrollment(ctx, uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), now); err == nil {
		t.Fatal("second enrollment was stored")
	}
	if err := s.SetEnrollmentCSR(ctx, "-----BEGIN CERTIFICATE REQUEST-----\nMII=\n-----END CERTIFICATE REQUEST-----\n"); err != nil {
		t.Fatal(err)
	}
	node := uuid.Must(uuid.NewV7())
	if err := s.CompleteEnrollment(ctx, Identity{
		NodeID: node, IncarnationID: inc, Serial: "0a", NotAfter: now.Add(time.Hour), PublicSHA256: "abc",
	}, now); err != nil {
		t.Fatal(err)
	}
	id, ok, err := s.Identity(ctx)
	if err != nil || !ok || id.NodeID != node || id.Serial != "0a" {
		t.Fatal(id, ok, err)
	}
	if err := s.UpdateCertificate(ctx, "0b", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	id, _, err = s.Identity(ctx)
	if err != nil || id.Serial != "0b" || id.NodeID != node {
		t.Fatal(id, err)
	}

	env := []byte(`{"signed":{},"signatures":[]}`)
	if err := s.AcceptDesired(ctx, Desired{
		Generation: 2, ETag: `"b"`, Envelope: env, IssuedAt: now, ExpiresAt: now.Add(time.Hour), Publication: "published",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptDesired(ctx, Desired{
		Generation: 1, ETag: `"a"`, Envelope: env, IssuedAt: now, ExpiresAt: now.Add(time.Hour), Publication: "published",
	}); !errors.Is(err, ErrStaleGeneration) {
		t.Fatal(err)
	}
	got, ok, err := s.Desired(ctx)
	if err != nil || !ok || got.Generation != 2 || got.ETag != `"b"` {
		t.Fatal(got, ok, err)
	}
	if err := s.MarkApplied(ctx, 2); err != nil {
		t.Fatal(err)
	}

	boot := uuid.Must(uuid.NewV7())
	leaseID := uuid.Must(uuid.NewV7())
	if err := s.SaveLease(ctx, Lease{
		ID: leaseID, IssuedAt: now, ExpiresAt: now.Add(time.Hour), HighestIssuedAt: now,
		Envelope: env, MonoSent: time.Hour, BootID: boot,
	}); err != nil {
		t.Fatal(err)
	}
	lease, ok, err := s.Lease(ctx)
	if err != nil || !ok || lease.ID != leaseID || lease.MonoSent != time.Hour || lease.BootID != boot {
		t.Fatal(lease, ok, err)
	}
	if err := s.SaveAnchor(ctx, Anchor{BootID: boot, Server: now, Mono: time.Hour, Wall: now, Trusted: true}); err != nil {
		t.Fatal(err)
	}
	anchor, ok, err := s.Anchor(ctx)
	if err != nil || !ok || !anchor.Trusted || anchor.Mono != time.Hour {
		t.Fatal(anchor, ok, err)
	}
	member := uuid.Must(uuid.NewV7())
	if err := s.SaveQoS(ctx, map[uuid.UUID]uint16{member: 7}); err != nil {
		t.Fatal(err)
	}
	qos, err := s.QoS(ctx)
	if err != nil || qos[member] != 7 {
		t.Fatal(qos, err)
	}
	wide := uint64(1) << 62
	if err := s.SaveCounters(ctx, map[string]Counter{"up": {Packets: wide, Bytes: wide}}); err != nil {
		t.Fatal(err)
	}
	counters, err := s.Counters(ctx)
	if err != nil || counters["up"].Bytes != wide {
		t.Fatal(counters, err)
	}
	report := uuid.Must(uuid.NewV7())
	if err := s.Enqueue(ctx, report, []byte(`{"ok":true}`), now); err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue(ctx, report, []byte(`{"ok":false}`), now); err != nil {
		t.Fatal(err)
	}
	pending, err := s.Pending(ctx)
	if err != nil || len(pending) != 1 || string(pending[0].Payload) != `{"ok":true}` {
		t.Fatal(pending, err)
	}
	if err := s.MarkAccepted(ctx, report, now); err != nil {
		t.Fatal(err)
	}
	pending, err = s.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatal(pending, err)
	}
}

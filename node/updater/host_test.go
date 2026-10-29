package updater

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestUPD_WorkFromEnvelope(t *testing.T) {
	node := Node{ID: uuid.Must(uuid.NewV7()), Incarnation: uuid.Must(uuid.NewV7())}
	now := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	rel := uuid.Must(uuid.NewV7())
	revoked := uuid.Must(uuid.NewV7())
	ds := sampleDesired(node, rel, now)
	raw, trust := signState(t, ds, envelope.PurposeDesiredState, now)
	work, err := WorkFromEnvelope(raw, trust, node, now, []uuid.UUID{revoked})
	if err != nil {
		t.Fatal(err)
	}
	if work.Intent.ReleaseId != rel || work.PolicyGeneration.String() != "9" || len(work.Revoked) != 1 || work.Revoked[0] != revoked {
		t.Fatalf("work %+v", work.Intent.ReleaseId)
	}
	if _, err := WorkFromEnvelope(raw, trust, node, ds.ExpiresAt, nil); asOp(err) == nil || asOp(err).Code != apierr.AuthorizationExpired {
		t.Fatalf("expiry %v", err)
	}
	other := node
	other.ID = uuid.Must(uuid.NewV7())
	if _, err := WorkFromEnvelope(raw, trust, other, now, nil); asOp(err) == nil || asOp(err).Code != apierr.IncarnationMismatch {
		t.Fatalf("node %v", err)
	}
	ds.Update = nil
	none, noneTrust := signState(t, ds, envelope.PurposeDesiredState, now)
	if _, err := WorkFromEnvelope(none, noneTrust, node, now, nil); !errors.Is(err, ErrNoWork) {
		t.Fatalf("no work %v", err)
	}
	bad, badTrust := signState(t, sampleDesired(node, rel, now), envelope.PurposeRelease, now)
	if _, err := WorkFromEnvelope(bad, badTrust, node, now, nil); asOp(err) == nil || asOp(err).Code != apierr.KeyUntrusted {
		t.Fatalf("purpose %v", err)
	}
	raw[len(raw)-2] ^= 0x01
	if _, err := WorkFromEnvelope(raw, trust, node, now, nil); err == nil {
		t.Fatal("tampered envelope was accepted")
	}
}

func TestUPD_FileGateAndMeter(t *testing.T) {
	dir := t.TempDir()
	g := FileGate{Path: filepath.Join(dir, "forwarding.blocked")}
	ctx := context.Background()
	if err := g.Block(ctx); err != nil || !g.Blocked() {
		t.Fatal(err)
	}
	if err := g.Block(ctx); err != nil || !g.Blocked() {
		t.Fatal("second block")
	}
	if err := g.Unblock(ctx); err != nil || g.Blocked() {
		t.Fatal(err)
	}
	if err := g.Unblock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := (CommandServices{}).Stop(ctx); err == nil {
		t.Fatal("empty stop command ran")
	}
	if err := (CommandServices{StopCmd: []string{"/bin/true"}}).Stop(ctx); err != nil {
		t.Fatal(err)
	}
	// Start must pass the release directory. The shell fails when that
	// argument is missing.
	if err := (CommandServices{StartCmd: []string{"/bin/sh", "-c", "test -n \"$1\"", "sh"}}).Start(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if err := (CommandServices{StartCmd: []string{"/bin/true"}}).Start(ctx, ""); err == nil {
		t.Fatal("empty release directory was started")
	}

	cp := filepath.Join(dir, "meter.json")
	epoch := uuid.Must(uuid.NewV7()).String()
	body := []byte(`{"closed_meter_epoch_id":"` + epoch + `","last_durable_event_seq":"4","central_acked_through":"3"}`)
	if err := os.WriteFile(cp, body, 0o600); err != nil {
		t.Fatal(err)
	}
	m := FileMeter{Path: cp}
	a, err := m.FinalSample(ctx)
	if err != nil || a.ClosedMeterEpochID != epoch || a.LastDurableEventSeq.String() != "4" {
		t.Fatal(err, a)
	}
	b, err := m.FinalSample(ctx)
	if err != nil || b != a {
		t.Fatal(b, err)
	}
	if _, err := (FileMeter{Path: filepath.Join(dir, "missing")}).FinalSample(ctx); err == nil {
		t.Fatal("missing checkpoint invented a sample")
	}
	e1, err := m.OpenEpoch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	e2, err := m.OpenEpoch(ctx)
	if err != nil || e1 != e2 {
		t.Fatalf("epoch %s %s %v", e1, e2, err)
	}
	left, _ := os.ReadFile(cp)
	if string(left) != string(body) {
		t.Fatal("meter file was rewritten")
	}
}

func TestUPD_HTTPSink(t *testing.T) {
	var gotKey string
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"accepted":true,"accepted_as_history":true,"applied_to_current_observation":false,"observed_update_generation":"1","received_at":"2026-10-28T15:00:00Z","result_id":"00000000-0000-0000-0000-000000000001"}`))
		case "/conflict":
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	srv.EnableHTTP2 = false
	srv.StartTLS()
	t.Cleanup(srv.Close)
	id := uuid.Must(uuid.NewV7())
	payload := []byte(`{"result_id":"` + id.String() + `"}`)
	sink := HTTPSink{URL: srv.URL + "/ok", Client: srv.Client(), AllowLoopback: true}
	ack, conflict, err := sink.Submit(context.Background(), payload)
	if err != nil || conflict || !ack.AcceptedAsHistory || gotKey != id.String() {
		t.Fatalf("ack %+v conflict %v key %s err %v", ack, conflict, gotKey, err)
	}
	sink.URL = srv.URL + "/conflict"
	_, conflict, err = sink.Submit(context.Background(), payload)
	if err != nil || !conflict {
		t.Fatalf("conflict %v %v", conflict, err)
	}
	sink.URL = srv.URL + "/down"
	if _, _, err := sink.Submit(context.Background(), payload); err == nil {
		t.Fatal("503 was treated as acceptance")
	}
	sink.URL = "https://metadata.google.internal/agent/update-results"
	if _, _, err := sink.Submit(context.Background(), payload); err == nil {
		t.Fatal("metadata host was accepted")
	}
}

func TestUPD_NodeChecksFailClosed(t *testing.T) {
	var c NodeChecks
	for _, st := range []api.HealthCheckStatus{
		mustStatus(c.Isolation(context.Background(), ReleaseView{})),
		mustStatus(c.DNSPolicy(context.Background(), ReleaseView{})),
		mustStatus(c.TunnelProbe(context.Background(), ReleaseView{})),
		mustStatus(c.Processes(context.Background(), ReleaseView{})),
	} {
		if st != api.HealthCheckStatusFailed {
			t.Fatal(st)
		}
	}
}

func TestUPD_LoadConfigAndRecover(t *testing.T) {
	root := t.TempDir()
	trustPath := filepath.Join(root, "trust.json")
	writeTrust(t, trustPath)
	cfg := Config{
		Layout:          testLayout(root),
		NodeID:          uuid.Must(uuid.NewV7()),
		IncarnationID:   uuid.Must(uuid.NewV7()),
		BootID:          uuid.Must(uuid.NewV7()),
		Platform:        testPlatform(),
		ReleaseTrust:    trustPath,
		DesiredTrust:    trustPath,
		DesiredState:    filepath.Join(root, "desired.json"),
		MeterCheckpoint: filepath.Join(root, "meter.json"),
	}
	path := filepath.Join(root, "config.json")
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil || loaded.NodeID != cfg.NodeID {
		t.Fatal(err)
	}
	bad := append(append([]byte{}, b[:len(b)-1]...), []byte(`,"nope":true}`)...)
	if err := os.WriteFile(path, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("unknown field accepted")
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err = LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := loaded.Engine()
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	if err := eng.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ExitCode(nil) != 0 || ExitCode(ErrMembersBlocked) != 2 || ExitCode(errors.New("x")) != 1 {
		t.Fatal("exit codes")
	}
}

func mustStatus(st api.HealthCheckStatus, _ error) api.HealthCheckStatus { return st }

func sampleDesired(node Node, rel uuid.UUID, now time.Time) api.DesiredState {
	job := uuid.Must(uuid.NewV7())
	return api.DesiredState{
		SchemaVersion:            api.DesiredStateSchemaVersionN1,
		NodeId:                   node.ID,
		NodeIncarnationId:        node.Incarnation,
		ConfigGeneration:         decstr.MustParse("9"),
		PublicationIntent:        api.PublicationIntentPublished,
		SafetyProfile:            "internet-only-v1",
		BindingRevision:          decstr.MustParse("1"),
		ServingReleaseIds:        []uuid.UUID{rel},
		SupportedReleasePolicyId: uuid.Must(uuid.NewV7()),
		DeviceBindings:           []api.DeviceBinding{},
		MemberRatePolicies:       []api.MemberRatePolicy{},
		Dns:                      api.DnsServiceConfig{ResolverIpv4: []string{}, ResolverIpv6: []string{}},
		IssuedAt:                 now.Add(-time.Minute),
		ExpiresAt:                now.Add(time.Hour),
		Update: &api.UpdateIntent{
			Attempt: 1, AuthorizationValidUntil: now.Add(time.Minute),
			Generation: decstr.MustParse("4"), IncarnationId: node.Incarnation,
			IssuedAt: now.Add(-time.Minute), JobId: job,
			ManifestSha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ManifestUrl:    "https://releases.example.invalid/m.json",
			NodeId:         node.ID, PublicationIntent: api.PublicationIntentPublished, ReleaseId: rel,
		},
	}
}

func signState(t *testing.T, ds api.DesiredState, purpose envelope.Purpose, now time.Time) ([]byte, *envelope.TrustStore) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID: "desired-test", PublicKey: pub, Purposes: []envelope.Purpose{purpose},
		NotBefore: now.Add(-24 * time.Hour), NotAfter: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	env, err := envelope.Sign(ds, priv, "desired-test")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	return raw, ts
}

func writeTrust(t *testing.T, path string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID: "release-test", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease},
		NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

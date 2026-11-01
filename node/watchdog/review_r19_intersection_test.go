package watchdog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

type reviewR19Enforcer struct{ baselines int }

func (e *reviewR19Enforcer) Baseline(context.Context, enforce.Datapath) error {
	e.baselines++
	return nil
}

func (e *reviewR19Enforcer) BaselinePresent(context.Context) (bool, error) { return true, nil }

func TestReviewR19_Q3_V1HeartbeatTreatsIntersectionAsFullService(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heartbeat")
	boot := uuid.Must(uuid.NewV7())
	now := time.Hour
	if err := WriteHeartbeat(path, Beat{
		BootID: boot, Mono: now - time.Second, Wall: time.Unix(1_700_000_000, 0).UTC(),
		PolicyExpires: now + time.Minute, BaselineOK: true, Serving: true, Generation: 4,
		Reason: "generation_intersection",
	}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, []byte("transition intersection\nsubset not-a-generation\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHeartbeat(path)
	if err != nil {
		t.Fatal(err)
	}
	enf := &reviewR19Enforcer{}
	w := &Watch{
		Enforce: enf,
		BootID:  boot,
		NowMono: func() (time.Duration, error) { return now, nil },
		Config: Config{
			HeartbeatPath: path, WatchdogStaleSeconds: 30,
			Datapath: DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"},
		},
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !got.Serving || got.Generation != 4 || got.Reason != "generation_intersection" || enf.baselines != 0 {
		t.Fatalf("v1 intersection heartbeat was not treated as serving generation 4: %+v baselines=%d", got, enf.baselines)
	}
	if err := os.WriteFile(path, []byte("v2\nboot_id "+boot.String()+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadHeartbeat(path); err == nil {
		t.Fatal("v2 heartbeat was readable")
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if enf.baselines != 1 {
		t.Fatalf("unreadable v2 heartbeat baselined %d times", enf.baselines)
	}
	t.Fatalf("TODO(review): ReadHeartbeat accepted a v1 file with reason generation_intersection, serving=true and generation=4, and Once left the ruleset alone. Extra transition lines were ignored. A v2 first line is heartbeat_unreadable and Once baselined")
}

func TestReviewR19_Q3_WriteHeartbeatStaysV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heartbeat")
	if err := WriteHeartbeat(path, Beat{
		BootID: uuid.Must(uuid.NewV7()), Mono: time.Second, Wall: time.Unix(0, 0).UTC(),
		BaselineOK: true, Generation: 4, Reason: "generation_intersection",
	}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 3 || string(body[:3]) != "v1\n" {
		t.Fatalf("heartbeat did not start with v1: %q", body)
	}
}

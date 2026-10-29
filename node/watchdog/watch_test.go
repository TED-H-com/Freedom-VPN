package watchdog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

type fakeEnforcer struct {
	present int
	calls   int
}

func (f *fakeEnforcer) Baseline(context.Context, enforce.Datapath) error {
	f.calls++
	return nil
}

func (f *fakeEnforcer) BaselinePresent(context.Context) (bool, error) {
	return f.present > 0, nil
}

func TestWatch_HealthyDoesNotTouchRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heartbeat")
	boot := uuid.Must(uuid.NewV7())
	now := 20 * time.Minute
	beat := Beat{
		BootID: boot, Mono: now - 2*time.Second, Wall: time.Unix(1_700_000_000, 0),
		PolicyExpires: now + time.Hour, BaselineOK: true, Serving: true, Generation: 4, Reason: "serving",
	}
	if err := WriteHeartbeat(path, beat); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHeartbeat(path)
	if err != nil || got.Generation != 4 || !got.Serving {
		t.Fatal(err, got)
	}
	enf := &fakeEnforcer{present: 1}
	w := &Watch{
		Config:  Config{HeartbeatPath: path, WatchdogStaleSeconds: 30, Datapath: DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"}},
		Enforce: enf, BootID: boot, NowMono: func() (time.Duration, error) { return now, nil },
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if enf.calls != 0 {
		t.Fatalf("healthy watchdog applied baseline %d times", enf.calls)
	}
}

func TestWatch_StaleAppliesBaselineOncePerCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heartbeat")
	boot := uuid.Must(uuid.NewV7())
	now := 20 * time.Minute
	beat := Beat{
		BootID: boot, Mono: now - time.Minute, Wall: time.Unix(1_700_000_000, 0),
		PolicyExpires: now + time.Hour, BaselineOK: true, Serving: true, Generation: 4, Reason: "serving",
	}
	if err := WriteHeartbeat(path, beat); err != nil {
		t.Fatal(err)
	}
	enf := &fakeEnforcer{present: 1}
	w := &Watch{
		Config:  Config{HeartbeatPath: path, WatchdogStaleSeconds: 30, Datapath: DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"}},
		Enforce: enf, BootID: boot, NowMono: func() (time.Duration, error) { return now, nil },
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if enf.calls != 1 {
		t.Fatalf("calls %d", enf.calls)
	}
}

func TestWatch_RestartDoesNotResetDeadline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heartbeat")
	boot := uuid.Must(uuid.NewV7())
	expires := 50 * time.Minute
	now := 11 * time.Minute
	beat := Beat{
		BootID: boot, Mono: now - 2*time.Second, Wall: time.Unix(1_700_000_000, 0),
		PolicyExpires: expires, BaselineOK: true, Serving: true, Generation: 2, Reason: "serving",
	}
	if err := WriteHeartbeat(path, beat); err != nil {
		t.Fatal(err)
	}
	enf := &fakeEnforcer{present: 1}
	// A new process reads the same file and the same monotonic clock.
	w := &Watch{
		Config:  Config{HeartbeatPath: path, WatchdogStaleSeconds: 30, Datapath: DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"}},
		Enforce: enf, BootID: boot, NowMono: func() (time.Duration, error) { return now, nil },
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHeartbeat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.PolicyExpires != expires {
		t.Fatalf("watchdog rewrote the deadline to %s", got.PolicyExpires)
	}
	if enf.calls != 0 {
		t.Fatal("restart applied rules")
	}
}

package watchdog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/google/uuid"
)

func watchRig(t *testing.T) (*Watch, *fakeEnforcer, Beat, *bytes.Buffer) {
	t.Helper()
	now := 42821844519578 * time.Nanosecond // captured verify 958 start
	b := Beat{BootID: uuid.New(), Mono: now - time.Millisecond, Wall: time.Now().UTC(),
		PolicyExpires: 42835508568707 * time.Nanosecond, Generation: 30, Serving: true, BaselineOK: true, Reason: "serving"}
	x := &fakeEnforcer{present: 1}
	log := &bytes.Buffer{}
	w := &Watch{Config: Config{HeartbeatPath: filepath.Join(t.TempDir(), "heartbeat"), WatchdogStaleSeconds: 30,
		Datapath: DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"}},
		Enforce: x, BootID: b.BootID, NowMono: func() (time.Duration, error) { return now, nil }, Log: slog.New(slog.NewJSONHandler(log, nil))}
	if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
		t.Fatal(err)
	}
	return w, x, b, log
}

func TestAT13_ISOF603_HeartbeatReadPrecedesClockSample(t *testing.T) {
	w, x, b, _ := watchRig(t)
	now, _ := w.mono()
	w.NowMono = func() (time.Duration, error) {
		// The writer runs just AFTER the watchdog samples now. Reading this
		// heartbeat against that earlier sample used to trigger a baseline.
		b.Mono = now + time.Nanosecond
		if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
			t.Fatal(err)
		}
		return now, nil
	}
	if err := w.Once(t.Context()); err != nil {
		t.Fatal(err)
	}
	if x.calls != 0 {
		t.Fatal("concurrent fresh heartbeat erased generation 30")
	}
}

func TestAT13_ISOF603_RefreshCancelsStaleDecision(t *testing.T) {
	w, x, b, log := watchRig(t)
	now, _ := w.mono()
	b.Mono = now - 31*time.Second
	if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
		t.Fatal(err)
	}
	w.NowMono = func() (time.Duration, error) {
		b.Mono = now
		if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
			t.Fatal(err)
		}
		return now, nil
	}
	if err := w.Once(t.Context()); err != nil {
		t.Fatal(err)
	}
	if x.calls != 0 || !strings.Contains(log.String(), "heartbeat_stale") || !strings.Contains(log.String(), "baseline canceled") {
		t.Fatal("stale decision survived fresh heartbeat", log.String())
	}
}

func TestAT13_ISOF603_WatchdogBaselineReasons(t *testing.T) {
	for _, reason := range []string{"clock_unavailable", "heartbeat_unreadable", "boot_changed", "heartbeat_in_future", "heartbeat_stale", "baseline_unverified", "policy_expired", "serving_without_deadline", "baseline_missing"} {
		t.Run(reason, func(t *testing.T) {
			w, x, b, log := watchRig(t)
			now, _ := w.mono()
			switch reason {
			case "clock_unavailable":
				w.NowMono = func() (time.Duration, error) { return 0, errors.New("TEST ONLY") }
			case "boot_changed":
				w.BootID = uuid.New()
			case "heartbeat_in_future":
				b.Mono = now + time.Second
			case "heartbeat_stale":
				b.Mono = now - 31*time.Second
			case "baseline_unverified":
				b.BaselineOK = false
			case "policy_expired":
				b.PolicyExpires = now
			case "serving_without_deadline":
				b.PolicyExpires = 0
			case "baseline_missing":
				x.present = 0
			}
			if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
				t.Fatal(err)
			}
			if reason == "heartbeat_unreadable" {
				w.Config.HeartbeatPath += "-missing"
			}
			if err := w.Once(t.Context()); err != nil {
				t.Fatal(err)
			}
			if x.calls != 1 || !strings.Contains(log.String(), `"cause":"`+reason+`"`) || !strings.Contains(log.String(), `"phase":"end"`) {
				t.Fatal("missing baseline provenance", log.String())
			}
		})
	}
}

func TestAT13_ISOF603_WatchdogCannotOverwriteRenewal(t *testing.T) {
	w, x, b, _ := watchRig(t)
	w.Config.LockPath = filepath.Join(t.TempDir(), "apply.lock")
	lock, err := nodelock.TryAcquire(w.Config.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	b.PolicyExpires = b.Mono - time.Second
	if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
		t.Fatal(err)
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if x.calls != 0 {
		t.Fatal("watchdog wrote during Apply")
	}
	b.PolicyExpires = b.Mono + time.Minute
	if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
		t.Fatal(err)
	}
	lock.Release()
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if x.calls != 0 {
		t.Fatal("watchdog reused pre-renewal decision")
	}
}

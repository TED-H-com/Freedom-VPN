package watchdog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"os/exec"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/google/uuid"
)

// Enforcer installs the fail-closed baseline. BaselinePresent reports whether
// the inet fvpn table is on the host. A healthy watchdog must not call Baseline.
type Enforcer interface {
	Baseline(ctx context.Context, dp enforce.Datapath) error
	BaselinePresent(ctx context.Context) (bool, error)
}

// SystemEnforcer uses node/enforce and the nft binary. It does not open a
// control connection.
type SystemEnforcer struct{}

// Baseline installs the boot ruleset.
func (SystemEnforcer) Baseline(ctx context.Context, dp enforce.Datapath) error {
	return enforce.ApplyBaseline(ctx, enforce.System{}, dp)
}

// BaselinePresent is true when `nft list table inet fvpn` succeeds.
func (SystemEnforcer) BaselinePresent(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "nft", "list", "table", enforce.TableFamily, enforce.TableName)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		// An exit status alone cannot distinguish a missing table from a
		// failed read. Preserve that uncertainty in the baseline reason.
		return false, errors.New("watchdog: baseline read failed")
	}
	return true, nil
}

// Config is the watchdog's view of agent.json. Unknown fields are ignored so
// the agent and the watchdog can share one file.
type Config struct {
	LockPath                string         `json:"lock_path"`
	HeartbeatPath           string         `json:"heartbeat_path"`
	WatchdogStaleSeconds    int            `json:"watchdog_stale_seconds"`
	WatchdogIntervalSeconds int            `json:"watchdog_interval_seconds"`
	Datapath                DatapathConfig `json:"datapath"`
	BootIDFile              string         `json:"boot_id_file"`
}

// DatapathConfig names the interfaces the baseline binds to.
type DatapathConfig struct {
	MemberIngress string `json:"member_ingress"`
	Uplink        string `json:"uplink"`
	SNAT          string `json:"snat"`
}

// LoadConfig reads a watchdog config. Missing durations use the defaults.
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("watchdog: config: %w", err)
	}
	if c.LockPath == "" {
		c.LockPath = nodelock.DefaultPath
	}
	if c.HeartbeatPath == "" {
		c.HeartbeatPath = "/run/vpn-node/heartbeat"
	}
	if c.WatchdogStaleSeconds <= 0 {
		c.WatchdogStaleSeconds = 30
	}
	if c.WatchdogIntervalSeconds <= 0 {
		c.WatchdogIntervalSeconds = c.WatchdogStaleSeconds / 3
		if c.WatchdogIntervalSeconds < 1 {
			c.WatchdogIntervalSeconds = 1
		}
	}
	if c.Datapath.MemberIngress == "" || c.Datapath.Uplink == "" || c.Datapath.SNAT == "" {
		return Config{}, errors.New("watchdog: datapath is incomplete")
	}
	return c, nil
}

func (c Config) datapath() (enforce.Datapath, error) {
	snat, err := netip.ParseAddr(c.Datapath.SNAT)
	if err != nil {
		return enforce.Datapath{}, fmt.Errorf("watchdog: snat: %w", err)
	}
	return enforce.Datapath{MemberIngress: c.Datapath.MemberIngress, Uplink: c.Datapath.Uplink, SNAT: snat}, nil
}

// Watch is one watchdog loop.
type Watch struct {
	Log     *slog.Logger
	Config  Config
	Enforce Enforcer
	BootID  uuid.UUID
	Clock   timeanchor.Clock
	// NowMono overrides Clock when set. Tests use it to avoid a linux clock.
	NowMono func() (time.Duration, error)
}

// Once checks liveness once. A healthy node does not touch the ruleset, so a
// restart of the watchdog cannot restart a kernel timeout (§16.3).
func (w *Watch) Once(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if w.Enforce == nil {
		return errors.New("watchdog: no enforcer")
	}
	dp, err := w.Config.datapath()
	if err != nil {
		return err
	}
	// Serialize with Apply. Re-read the heartbeat after acquiring the lock:
	// an old expiry/stale decision must not erase a just-installed renewal.
	if w.Config.LockPath != "" {
		lock, err := nodelock.TryAcquire(w.Config.LockPath)
		if errors.Is(err, nodelock.ErrBusy) {
			return nil
		}
		if err != nil {
			return err
		}
		defer lock.Release()
	}
	reason, beat, mono := w.baselineDecision(ctx)
	if reason == "" {
		return nil
	}
	log := w.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("baseline decision", "writer", "watchdog", "cause", reason,
		"boot_id", w.BootID, "mono_ns", int64(mono), "heartbeat_mono_ns", int64(beat.Mono),
		"policy_expires_mono_ns", int64(beat.PolicyExpires), "generation", beat.Generation,
		"serving", beat.Serving, "baseline_ok", beat.BaselineOK)
	// A slow presence read can overlap a fresh heartbeat even under the
	// apply lock. Re-evaluate immediately before the destructive operation.
	reason, beat, mono = w.baselineDecision(ctx)
	if reason == "" {
		log.Info("baseline canceled", "writer", "watchdog", "cause", "heartbeat_refreshed")
		return nil
	}
	log.Info("baseline write", "writer", "watchdog", "phase", "begin", "cause", reason,
		"mono_ns", int64(mono), "heartbeat_mono_ns", int64(beat.Mono),
		"policy_expires_mono_ns", int64(beat.PolicyExpires), "generation", beat.Generation)
	err = w.Enforce.Baseline(ctx, dp)
	end, _ := w.mono()
	log.Info("baseline write", "writer", "watchdog", "phase", "end", "cause", reason,
		"mono_ns", int64(end), "succeeded", err == nil)
	return err
}

func (w *Watch) baselineDecision(ctx context.Context) (string, Beat, time.Duration) {
	// Read the atomically replaced heartbeat BEFORE sampling now. Otherwise
	// a heartbeat written between the two reads looks like a clock rollback.
	beat, readErr := ReadHeartbeat(w.Config.HeartbeatPath)
	mono, err := w.mono()
	if err != nil || mono <= 0 {
		return "clock_unavailable", beat, mono
	}
	if readErr != nil {
		return "heartbeat_unreadable", beat, mono
	}
	if w.BootID != uuid.Nil && beat.BootID != w.BootID {
		return "boot_changed", beat, mono
	}
	if beat.Mono > mono {
		return "heartbeat_in_future", beat, mono
	}
	if mono-beat.Mono > time.Duration(w.Config.WatchdogStaleSeconds)*time.Second {
		return "heartbeat_stale", beat, mono
	}
	if !beat.BaselineOK {
		return "baseline_unverified", beat, mono
	}
	if beat.PolicyExpires > 0 && mono >= beat.PolicyExpires {
		return "policy_expired", beat, mono
	}
	if beat.Serving && beat.PolicyExpires <= 0 {
		return "serving_without_deadline", beat, mono
	}
	readCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	ok, err := w.Enforce.BaselinePresent(readCtx)
	if err != nil {
		return "baseline_unreadable", beat, mono
	}
	if !ok {
		return "baseline_missing", beat, mono
	}
	return "", beat, mono
}

func (w *Watch) mono() (time.Duration, error) {
	if w.NowMono != nil {
		return w.NowMono()
	}
	if w.Clock != nil {
		d := w.Clock.Mono()
		if d <= 0 {
			return 0, errors.New("watchdog: monotonic clock")
		}
		return d, nil
	}
	return timeanchor.Mono()
}

// Run calls Once until ctx ends. Interval defaults come from the config.
func (w *Watch) Run(ctx context.Context) error {
	interval := time.Duration(w.Config.WatchdogIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	for {
		if err := w.Once(ctx); err != nil && ctx.Err() == nil {
			// Keep looping. The next pass retries the baseline.
		}
		t := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}

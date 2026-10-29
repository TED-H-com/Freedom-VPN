package watchdog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
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
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return false, nil
		}
		return false, nil
	}
	return true, nil
}

// Config is the watchdog's view of agent.json. Unknown fields are ignored so
// the agent and the watchdog can share one file.
type Config struct {
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
	if w.Enforce == nil {
		return errors.New("watchdog: no enforcer")
	}
	dp, err := w.Config.datapath()
	if err != nil {
		return err
	}
	if w.needsBaseline(ctx) {
		if err := w.Enforce.Baseline(ctx, dp); err != nil {
			return err
		}
	}
	return nil
}

func (w *Watch) needsBaseline(ctx context.Context) bool {
	mono, err := w.mono()
	if err != nil || mono <= 0 {
		return true
	}
	beat, err := ReadHeartbeat(w.Config.HeartbeatPath)
	if err != nil {
		return true
	}
	if w.BootID != uuid.Nil && beat.BootID != w.BootID {
		return true
	}
	if beat.Mono > mono {
		return true
	}
	age := mono - beat.Mono
	if age > time.Duration(w.Config.WatchdogStaleSeconds)*time.Second {
		return true
	}
	if !beat.BaselineOK {
		return true
	}
	if beat.PolicyExpires > 0 && mono >= beat.PolicyExpires {
		return true
	}
	if beat.Serving && beat.PolicyExpires <= 0 {
		return true
	}
	ok, err := w.Enforce.BaselinePresent(ctx)
	if err != nil || !ok {
		return true
	}
	return false
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

package installer

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// Real seed, apply, SQLite journal and status CLI; TEST ONLY service, meter
// and network-health adapters. This proves check reconciliation, not product
// dataplane health. The privileged suite separately requires real health.
func TestD089_CheckAfterJournaledSwitch(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	env := baseEnv(t, root)
	old := mustRelease(t, root, uuid.NewString(), "1")
	if _, err := Install(ctx, env, mustReq(t, root, old)); err != nil {
		t.Fatal(err)
	}
	installedHash := hashFile(t, env.Layout.installedFile())
	cfg, err := updater.LoadConfig(env.Layout.updaterConfig())
	if err != nil {
		t.Fatal(err)
	}
	cfg.MTLSCert, cfg.MTLSKey, cfg.MTLSCA = "", "", ""
	engine, err := cfg.Engine()
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	engine.Services = &reconcileServices{}
	engine.Meter = reconcileMeter{}
	engine.Health = reconcileHealth{}

	cli := filepath.Join(root, "status-updater")
	build := exec.Command("go", "build", "-o", cli, "../../node/updater/cmd/fvpn-updater")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build updater: %v %s", err, out)
	}
	command := env.Command
	env.Command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == env.Layout.updaterBin() && len(args) > 0 {
			if args[0] == "status" || (args[0] == "seed" && len(args) == 2) {
				return exec.CommandContext(ctx, cli, args...).CombinedOutput()
			}
			if args[0] == "seed" {
				return nil, engine.Seed(ctx, args[4], args[6])
			}
		}
		return command(ctx, name, args...)
	}
	// Remove source archive: real seed must use the installer's retained cache.
	if err := os.Remove(old.Artifact); err != nil {
		t.Fatal(err)
	}
	if r, err := seedBaseline(ctx, env); err != nil || r.Status != "seeded" {
		t.Fatalf("offline seed: %+v %v", r, err)
	}
	if r, err := Check(ctx, env); err != nil || r.ReleaseSource != "updater_journal" {
		t.Fatalf("seeded check: %+v %v", r, err)
	}
	next := mustRelease(t, root, uuid.NewString(), "2")
	trust, err := os.ReadFile(next.Trust)
	if err != nil {
		t.Fatal(err)
	}
	engine.ReleaseTrust, err = envelope.ParseNodeTrustStoreJSON(trust)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Upgrade(ctx, env, next.Manifest, next.Artifact, next.Trust); err != nil {
		t.Fatal(err)
	}
	if err := engine.ApplyPending(ctx, env.Layout.pendingUpgrade()); err != nil {
		t.Fatal(err)
	}
	r, err := Check(ctx, env)
	if err != nil || r.ReleaseID != next.ReleaseID || r.ReleaseSource != "updater_journal" || r.Updater.State != "succeeded" || r.InstallRecord.ReleaseID != old.ReleaseID || r.ManifestSHA256 != "" {
		t.Fatalf("post-switch check: %+v %v", r, err)
	}
	if hashFile(t, env.Layout.installedFile()) != installedHash {
		t.Fatal("check rewrote install history")
	}
	// F9-05: install history still says sequence 1, but the active journaled
	// release is sequence 2. A new signed sequence-1 release cannot be staged.
	lower := mustRelease(t, root, uuid.NewString(), "1")
	before := upgradeTrees(t, env.Layout)
	if _, err := Upgrade(ctx, env, lower.Manifest, lower.Artifact, lower.Trust); err == nil || AsError(err).Code != "DOWNGRADE" {
		t.Fatalf("post-switch downgrade: %v", err)
	}
	if upgradeTrees(t, env.Layout) != before {
		t.Fatal("post-switch downgrade changed staging or releases")
	}
	// A stale current link remains drift even though installed.json matches it.
	if err := fsutil.SwitchSymlink(env.Layout.CurrentLink, "releases/"+old.ReleaseID); err != nil {
		t.Fatal(err)
	}
	r, err = Check(ctx, env)
	if err == nil || !strings.Contains(strings.Join(r.Drifts, ","), "current") {
		t.Fatalf("current drift hidden: %+v", r)
	}
	before = upgradeTrees(t, env.Layout)
	if _, err := Upgrade(ctx, env, next.Manifest, next.Artifact, next.Trust); err == nil || AsError(err).Code != "installed_state" || upgradeTrees(t, env.Layout) != before {
		t.Fatalf("upgrade ignored live/journal pointer mismatch: %v", err)
	}
	// An unreadable journal must not fall back to the now-matching install record.
	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.Layout.UpdateJournal, "journal.db"), []byte("TEST ONLY corrupt journal"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err = Check(ctx, env)
	if err == nil || !strings.Contains(strings.Join(r.Drifts, ","), "updater_journal") {
		t.Fatalf("journal drift hidden: %+v", r)
	}
	if _, err := Upgrade(ctx, env, next.Manifest, next.Artifact, next.Trust); err == nil || AsError(err).Code != "installed_state" || upgradeTrees(t, env.Layout) != before {
		t.Fatalf("upgrade ignored unreadable journal: %v", err)
	}
	t.Log("real offline seed and journaled switch; current reconciled with unchanged install record; stale pointer and unreadable journal refused")
}

type reconcileServices struct{}

func (*reconcileServices) Stop(context.Context) error          { return nil }
func (*reconcileServices) Start(context.Context, string) error { return nil }

type reconcileMeter struct{}

func (reconcileMeter) FinalSample(context.Context) (updater.UsageCheckpoint, error) {
	return updater.UsageCheckpoint{ClosedMeterEpochID: uuid.NewString()}, nil
}
func (reconcileMeter) OpenEpoch(context.Context) (string, error) { return uuid.NewString(), nil }

type reconcileHealth struct{ updater.NodeChecks }

func (reconcileHealth) Isolation(context.Context, updater.ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusPassed, nil
}
func (reconcileHealth) DNSPolicy(context.Context, updater.ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusPassed, nil
}
func (reconcileHealth) TunnelProbe(context.Context, updater.ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusPassed, nil
}

func TestD089_StartCommandConsumesReleaseArgument(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	fx := mustRelease(t, dir, uuid.NewString(), "1")
	if _, err := Install(context.Background(), env, mustReq(t, dir, fx)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(env.Layout.updaterConfig())
	if err != nil {
		t.Fatal(err)
	}
	var cfg updater.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "systemctl"), []byte("#!/bin/sh\n[ \"$#\" = 3 ] && [ \"$1\" = start ] && [ \"$2\" = fvpn-watchdog.service ] && [ \"$3\" = fvpn-agent.service ]\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if err := (updater.CommandServices{StartCmd: cfg.StartCommand}).Start(context.Background(), env.Layout.releaseDir(fx.ReleaseID)); err != nil {
		t.Fatal(err)
	}
}

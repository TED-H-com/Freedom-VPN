package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/google/uuid"
)

func TestD089_PendingEnvelopeAndInvocationIdentity(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	old := mustRelease(t, dir, uuid.NewString(), "1")
	if _, err := Install(context.Background(), env, mustReq(t, dir, old)); err != nil {
		t.Fatal(err)
	}
	next := mustRelease(t, dir, uuid.NewString(), "2")
	previous := ""
	for range 2 {
		res, err := Upgrade(context.Background(), env, next.Manifest, next.Artifact, next.Trust)
		if err != nil {
			t.Fatal(err)
		}
		rep, err := Check(context.Background(), env)
		if err != nil || rep.PendingRelease == nil {
			t.Fatalf("check: %+v %v", rep, err)
		}
		pending := rep.PendingRelease
		if id, err := uuid.Parse(pending.RequestID); err != nil || id == uuid.Nil || pending.RequestID == previous {
			t.Fatalf("request id %q (previous %q)", pending.RequestID, previous)
		}
		previous = pending.RequestID
		if pending.ManifestPath != res.ManifestPath || pending.RequestID != res.RequestID || filepath.Dir(pending.ManifestPath) != env.Layout.StagingDir {
			t.Fatalf("result/record mismatch: %+v %+v", res, pending)
		}
		raw, err := os.ReadFile(pending.ManifestPath)
		if err != nil {
			t.Fatal(err)
		}
		sum, err := release.EnvelopeSHA256(raw)
		if err != nil || sum != next.SHA || sum != pending.ManifestSHA256 {
			t.Fatalf("retained envelope: %s %v", sum, err)
		}
		if !strings.Contains(rep.Text(), pending.RequestID) || !strings.Contains(rep.Text(), pending.ManifestPath) {
			t.Fatal("text check lost hand-off fields")
		}
	}
}

func TestD089_InstallAndReenrollSeedThroughInstalledBinary(t *testing.T) {
	for _, outcome := range []string{"supported", "without seed", "seed failed", "probe failed"} {
		t.Run(outcome, func(t *testing.T) {
			dir := t.TempDir()
			env := baseEnv(t, dir)
			fx := mustRelease(t, dir, uuid.NewString(), "1")
			req := mustReq(t, dir, fx)
			command := env.Command
			probes, seeds := 0, 0
			env.Command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				if name != env.Layout.updaterBin() || len(args) == 0 || args[0] != "seed" {
					return command(ctx, name, args...)
				}
				lk, err := nodelock.TryAcquire(env.Layout.LockPath)
				if err != nil {
					t.Fatalf("installer held shared lock across seed: %v", err)
				}
				_ = lk.Release()
				if strings.Join(args, " ") == "seed --help" {
					probes++
					switch outcome {
					case "without seed":
						return []byte("fvpn-updater: unknown command \"seed\"\n"), exec.Command("sh", "-c", "exit 64").Run()
					case "probe failed":
						return []byte("cannot execute updater"), errors.New("probe failed")
					}
					return []byte("Usage of seed: -manifest -tree\n"), nil
				}
				seeds++
				want := "seed --config " + env.Layout.updaterConfig() + " --manifest " + env.Layout.releaseManifest(fx.ReleaseID) + " --tree " + env.Layout.releaseDir(fx.ReleaseID)
				if strings.Join(args, " ") != want {
					t.Fatalf("seed flags: %v", args)
				}
				if hashFile(t, env.Layout.releaseManifest(fx.ReleaseID)) != hashFile(t, fx.Manifest) {
					t.Fatal("seed did not receive verified envelope bytes")
				}
				if outcome == "seed failed" {
					return []byte("CHECKPOINT_FAILED"), exec.Command("sh", "-c", "exit 1").Run()
				}
				return nil, nil
			}
			_, err := Install(context.Background(), env, req)
			failed := strings.HasSuffix(outcome, "failed")
			if (err != nil) != failed {
				t.Fatalf("install error: %v", err)
			}
			rep, checkErr := Check(context.Background(), env)
			if (checkErr != nil) != failed || rep.Baseline == nil {
				t.Fatalf("check %+v %v", rep, checkErr)
			}
			if failed {
				if rep.Baseline.Status != "failed" {
					t.Fatalf("failure hidden: %+v", rep.Baseline)
				}
				return
			}
			want := "seeded"
			if outcome == "without seed" {
				want = "not seeded (updater without seed)"
				if rep.Baseline.Probe.ExitCode != 64 || rep.Baseline.Probe.Stderr == "" || seeds != 0 {
					t.Fatalf("probe evidence: %+v seeds=%d", rep.Baseline, seeds)
				}
			}
			if rep.Baseline.Status != want {
				t.Fatalf("baseline %+v", rep.Baseline)
			}
			raw, err := os.ReadFile(env.Layout.updaterConfig())
			if err != nil {
				t.Fatal(err)
			}
			var cfg map[string]json.RawMessage
			if err := json.Unmarshal(raw, &cfg); err != nil {
				t.Fatal(err)
			}
			if _, ok := cfg["boot_id"]; ok {
				t.Fatal("installer requires a configured boot id")
			}
			token := filepath.Join(dir, "reenroll.token")
			if err := os.WriteFile(token, []byte("TEST ONLY enrollment token\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			res, err := Reenroll(context.Background(), env, token)
			if err != nil || res.Baseline.Status != want || probes != 2 {
				t.Fatalf("reenroll %+v %v probes=%d", res, err, probes)
			}
		})
	}
}

func TestAT26_InstallerUsesSharedKernelBootLock(t *testing.T) {
	env := baseEnv(t, t.TempDir())
	if err := os.MkdirAll(env.Layout.EtcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.Layout.updaterConfig(), []byte(`{"boot_id":"11111111-1111-4111-8111-111111111111"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	lk, err := tryLock(context.Background(), env.Layout)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	if other, err := nodelock.TryAcquire(env.Layout.LockPath); !errors.Is(err, nodelock.ErrBusy) {
		if other != nil {
			_ = other.Release()
		}
		t.Fatalf("shared contention: %v", err)
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		t.Fatal(err)
	}
	if got := readLock(env.Layout).BootID; got != strings.TrimSpace(string(boot)) {
		t.Fatalf("holder boot %s, want kernel value", got)
	}
}

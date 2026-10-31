package installer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRejectTokenArgv(t *testing.T) {
	if err := RejectTokenArgv([]string{"install", "--token", "secret"}); err == nil || AsError(err).Exit != ExitUsage {
		t.Fatalf("token argv: %v", err)
	}
	if err := RejectTokenArgv([]string{"install", "--token=secret"}); err == nil {
		t.Fatal("token= was accepted")
	}
	if err := RejectTokenArgv([]string{"install", "--token-file", "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestTokenFileMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkTokenFile(path); err == nil || AsError(err).Exit != ExitUsage {
		t.Fatalf("mode: %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkTokenFile(path); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := checkTokenFile(link); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestPlatformRefuse(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	if err := os.WriteFile(env.Layout.OSRelease, []byte("ID=debian\nVERSION_ID=\"12\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Identify(env)
	if err == nil || AsError(err).Exit != ExitPlatform || AsError(err).Code != "UNSUPPORTED_PLATFORM" {
		t.Fatalf("debian: %v", err)
	}
	if err := os.WriteFile(env.Layout.OSRelease, []byte("ID=ubuntu\nVERSION_ID=\"22.04\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Identify(env); err == nil {
		t.Fatal("22.04 accepted")
	}
	if err := os.WriteFile(env.Layout.OSRelease, []byte("ID=ubuntu\nVERSION_ID=\"24.04\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Identify(env)
	if err != nil || p.Arch != "amd64" || p.OSVersion != "24.04" {
		t.Fatalf("ubuntu: %+v %v", p, err)
	}
}

func TestMissingToolBeforeJournal(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	env.LookPath = func(name string) (string, error) {
		if name == "nft" {
			return "", errors.New("missing")
		}
		return "/usr/bin/" + name, nil
	}
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	_, err := Install(context.Background(), env, mustReq(t, dir, fx))
	if err == nil || AsError(err).Exit != ExitDependency {
		t.Fatalf("missing nft: %v", err)
	}
	if _, statErr := os.Stat(env.Layout.journalFile()); !os.IsNotExist(statErr) {
		t.Fatalf("journal exists: %v", statErr)
	}
	if _, statErr := os.Stat(env.Layout.VarDir); !os.IsNotExist(statErr) {
		t.Fatalf("var dir exists: %v", statErr)
	}
}

func TestInstallIdempotentAndUpgradeKeepsIdentity(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	res, err := Install(context.Background(), env, mustReq(t, dir, fx))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != statusInstalled || res.ReleaseID != fx.ReleaseID || res.IncarnationID == "" {
		t.Fatalf("result %+v", res)
	}
	key1 := hashFile(t, filepath.Join(env.Layout.IdentityDir, "key.pem"))
	node1 := hashFile(t, env.Layout.nodeFile())
	usage := []byte("usage-marker-not-a-secret\n")
	if err := os.WriteFile(env.Layout.usageDB(), usage, 0o600); err != nil {
		t.Fatal(err)
	}
	usage1 := hashFile(t, env.Layout.usageDB())
	again, err := Install(context.Background(), env, mustReq(t, dir, fx))
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != statusAlreadyCurrent {
		t.Fatalf("second status %s", again.Status)
	}
	if hashFile(t, filepath.Join(env.Layout.IdentityDir, "key.pem")) != key1 || hashFile(t, env.Layout.nodeFile()) != node1 {
		t.Fatal("identity changed on re-run")
	}
	if hashFile(t, env.Layout.usageDB()) != usage1 {
		t.Fatal("usage store changed on re-run")
	}
	rep, err := Check(context.Background(), env)
	if err != nil {
		t.Fatalf("check: %v drifts %v", err, rep.Drifts)
	}
	if rep.Status != statusHealthy || !rep.Identity.Present || rep.UsageStore != true {
		t.Fatalf("report %+v", rep)
	}
	if err := os.Remove(filepath.Join(env.Layout.SystemdDir, UnitAgent)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(env.Layout.IdentityDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rep, err = Check(context.Background(), env)
	if err == nil || AsError(err).Exit != ExitDrifted {
		t.Fatalf("drift: %v %+v", err, rep)
	}
	fx2 := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c2", "2")
	up, err := Install(context.Background(), env, mustReq(t, dir, fx2))
	if err != nil {
		t.Fatal(err)
	}
	if up.ReleaseID != fx2.ReleaseID {
		t.Fatalf("upgrade result %+v", up)
	}
	if hashFile(t, filepath.Join(env.Layout.IdentityDir, "key.pem")) != key1 {
		t.Fatal("identity changed on newer release")
	}
	if hashFile(t, env.Layout.usageDB()) != usage1 {
		t.Fatal("usage store changed on newer release")
	}
	target, err := os.Readlink(env.Layout.CurrentLink)
	if err != nil || target != "releases/"+fx2.ReleaseID {
		t.Fatalf("current -> %s %v", target, err)
	}
	low := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c3", "2")
	// A different release at an equal sequence is refused, not installed beside the current one.
	_, err = Install(context.Background(), env, mustReq(t, dir, low))
	if err == nil || AsError(err).Code != "DOWNGRADE" {
		t.Fatalf("downgrade: %v", err)
	}
}

func TestFaultRollsBackOwnScope(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	foreign := filepath.Join(dir, "foreign.txt")
	if err := os.WriteFile(foreign, []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	req := mustReq(t, dir, fx)
	req.FaultAfter = "directories"
	_, err := Install(context.Background(), env, req)
	if err == nil || AsError(err).Code != "rolled_back" {
		t.Fatalf("fault: %v", err)
	}
	if _, statErr := os.Stat(env.Layout.IdentityDir); !os.IsNotExist(statErr) {
		t.Fatalf("identity remains: %v", statErr)
	}
	if _, statErr := os.Stat(env.Layout.OptDir); !os.IsNotExist(statErr) {
		t.Fatalf("opt tree remains: %v", statErr)
	}
	got, err := os.ReadFile(foreign)
	if err != nil || string(got) != "keep\n" {
		t.Fatalf("foreign file: %q %v", got, err)
	}
	raw, err := os.ReadFile(env.Layout.journalFile())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"phase": "rolled_back"`) {
		t.Fatalf("journal %s", raw)
	}
	calls := joinedCalls(env)
	if strings.Contains(calls, "flush ruleset") || strings.Contains(calls, "resolv.conf") {
		t.Fatalf("host safety violated: %s", calls)
	}
}

func TestReenrollKeepsUsage(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	if _, err := Install(context.Background(), env, mustReq(t, dir, fx)); err != nil {
		t.Fatal(err)
	}
	usage := []byte("usage-marker-not-a-secret\n")
	if err := os.WriteFile(env.Layout.usageDB(), usage, 0o600); err != nil {
		t.Fatal(err)
	}
	before := hashFile(t, filepath.Join(env.Layout.IdentityDir, "key.pem"))
	oldInc := readInc(t, env.Layout)
	token := filepath.Join(dir, "token2")
	writeToken(t, token)
	res, err := Reenroll(context.Background(), env, token)
	if err != nil {
		t.Fatal(err)
	}
	if res.IncarnationID == "" || res.IncarnationID == oldInc {
		t.Fatalf("incarnation %s -> %s", oldInc, res.IncarnationID)
	}
	if hashFile(t, filepath.Join(env.Layout.IdentityDir, "key.pem")) == before {
		t.Fatal("key was not replaced")
	}
	got, err := os.ReadFile(env.Layout.usageDB())
	if err != nil || string(got) != string(usage) {
		t.Fatalf("usage %q %v", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(env.Layout.RetiredDir, "*", "key.pem"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("retired keys %v %v", matches, err)
	}
	st, err := os.Stat(filepath.Dir(matches[0]))
	if err != nil || st.Mode().Perm() != 0o700 {
		t.Fatalf("retired mode %v", st)
	}
	if _, err := os.Stat(token); !os.IsNotExist(err) {
		t.Fatal("token file was not consumed")
	}
}

func TestUpgradeDoesNotMoveCurrent(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	if _, err := Install(context.Background(), env, mustReq(t, dir, fx)); err != nil {
		t.Fatal(err)
	}
	before, err := os.Readlink(env.Layout.CurrentLink)
	if err != nil {
		t.Fatal(err)
	}
	fx2 := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c2", "2")
	res, err := Upgrade(context.Background(), env, fx2.Manifest, fx2.Artifact, fx2.Trust)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != statusStaged || res.ReleaseID != fx2.ReleaseID {
		t.Fatalf("upgrade %+v", res)
	}
	after, err := os.Readlink(env.Layout.CurrentLink)
	if err != nil || after != before {
		t.Fatalf("current moved %s -> %s", before, after)
	}
	if _, err := os.Stat(filepath.Join(env.Layout.releaseDir(fx2.ReleaseID), "fvpn-agent")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(env.Layout.pendingUpgrade()); err != nil {
		t.Fatal(err)
	}
}

func TestBusyLock(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	if err := os.MkdirAll(filepath.Dir(env.Layout.LockPath), 0o700); err != nil {
		t.Fatal(err)
	}
	lk, err := tryLock(env.Layout.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	_, err = Install(context.Background(), env, mustReq(t, dir, fx))
	if err == nil || AsError(err).Exit != ExitBusy {
		t.Fatalf("busy: %v", err)
	}
}

func TestImagePrepare(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	if _, err := Install(context.Background(), env, mustReq(t, dir, fx)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(env.Layout.SSHDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.Layout.SSHDir, "ssh_host_ed25519_key"), []byte("key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(env.Layout.NetbirdLib, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.Layout.NetbirdLib, "private.key"), []byte("nb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.Layout.MachineID, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImagePrepare(context.Background(), env, false); err == nil || AsError(err).Code != "identity_present" {
		t.Fatalf("refuse: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.Layout.IdentityDir, "key.pem")); err != nil {
		t.Fatal("key removed without the flag")
	}
	if _, err := ImagePrepare(context.Background(), env, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(env.Layout.IdentityDir, "key.pem")); !os.IsNotExist(err) {
		t.Fatal("key remains")
	}
	if _, err := os.Stat(env.Layout.usageDB()); !os.IsNotExist(err) {
		t.Fatal("usage remains")
	}
	if _, err := os.Stat(filepath.Join(env.Layout.SSHDir, "ssh_host_ed25519_key")); !os.IsNotExist(err) {
		t.Fatal("ssh host key remains")
	}
	mid, err := os.ReadFile(env.Layout.MachineID)
	if err != nil || len(strings.TrimSpace(string(mid))) != 0 {
		t.Fatalf("machine-id %q", mid)
	}
	if _, err := os.Stat(env.Layout.agentBin()); err != nil {
		t.Fatal("binary removed")
	}
}

func TestNetbirdAptPin(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	env.LookPath = func(name string) (string, error) {
		if name == "netbird" {
			return "", errors.New("missing")
		}
		switch name {
		case "nft", "tc", "conntrack", "ip", "systemctl", "curl", "gpg", "apt-get":
			return "/usr/bin/" + name, nil
		default:
			return "", errors.New("missing")
		}
	}
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	if _, err := Install(context.Background(), env, mustReq(t, dir, fx)); err != nil {
		t.Fatal(err)
	}
	calls := joinedCalls(env)
	if !strings.Contains(calls, "apt-get install -y --no-install-recommends netbird="+NetBirdPin) {
		t.Fatalf("apt pin missing: %s", calls)
	}
	raw, err := os.ReadFile(env.Layout.AptList)
	if err != nil || !strings.Contains(string(raw), "pkgs.netbird.io") {
		t.Fatalf("apt list %s %v", raw, err)
	}
	st, ok := readInstalled(env.Layout)
	if !ok || !st.NetbirdPackageInstalled {
		t.Fatalf("installed %+v", st)
	}
}

func TestVerifyTamperAndLatest(t *testing.T) {
	dir := t.TempDir()
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	if _, err := VerifyArchive(fx.Manifest, fx.Artifact, fx.Trust, "amd64", now); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fx.Artifact, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyArchive(fx.Manifest, fx.Artifact, fx.Trust, "amd64", now); err == nil {
		t.Fatal("tamper accepted")
	}
	latest := filepath.Join(dir, "latest")
	if err := os.WriteFile(latest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyArchive(fx.Manifest, latest, fx.Trust, "amd64", now); err == nil {
		t.Fatal("latest accepted")
	}
}

func TestJournalRestoresAndLeavesSibling(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	j, err := openJournal(env)
	if err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(dir, "sibling.txt")
	if err := os.WriteFile(sibling, []byte("stay\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(env.Layout.EtcDir, "owned.txt")
	if err := j.WriteFile(target, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := j.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("owned file remains")
	}
	got, err := os.ReadFile(sibling)
	if err != nil || string(got) != "stay\n" {
		t.Fatalf("sibling %q %v", got, err)
	}
}

func baseEnv(t *testing.T, root string) *Env {
	t.Helper()
	l := DefaultLayout().WithRoot(root)
	if err := os.MkdirAll(filepath.Dir(l.OSRelease), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.OSRelease, []byte("ID=ubuntu\nVERSION_ID=\"24.04\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(l.ProcBootID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.ProcBootID, []byte("11111111-1111-4111-8111-111111111111\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := &[]string{}
	env := &Env{
		Layout:        l,
		Arch:          "amd64",
		Kernel:        "Linux 6.8.0-test",
		Now:           func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) },
		Services:      NewMemServices(),
		AssumeSystemd: true,
		LookPath: func(name string) (string, error) {
			switch name {
			case "nft", "tc", "conntrack", "ip", "systemctl", "netbird", "curl", "gpg", "apt-get":
				return "/usr/bin/" + name, nil
			default:
				return "", errors.New("missing")
			}
		},
	}
	env.Command = func(_ context.Context, name string, args ...string) ([]byte, error) {
		line := filepath.Base(name) + " " + strings.Join(args, " ")
		*calls = append(*calls, line)
		if strings.HasSuffix(name, "fvpn-agent") && len(args) > 0 && args[0] == "enroll" {
			if err := writeFakeIdentity(l); err != nil {
				return nil, err
			}
			return nil, nil
		}
		base := filepath.Base(name)
		switch base {
		case "nft":
			if len(args) > 0 && args[0] == "list" {
				return nil, errors.New("no such table")
			}
		case "ip":
			if len(args) >= 2 && args[0] == "link" && args[1] == "show" {
				return nil, errors.New("no such link")
			}
			if len(args) >= 1 && args[0] == "netns" && len(args) > 1 && args[1] == "list" {
				return []byte(""), nil
			}
		case "netbird":
			if len(args) > 0 && args[0] == "version" {
				return []byte("0.82.0\n"), nil
			}
			return []byte("Management: Connected\n"), nil
		}
		return nil, nil
	}
	env.Stdout = &strings.Builder{}
	t.Cleanup(func() {
		os.WriteFile(filepath.Join(root, "calls.txt"), []byte(strings.Join(*calls, "\n")), 0o600)
	})
	// Stash the call slice on the builder via a side file the assertions read.
	env.Stderr = callSink{calls}
	return env
}

type callSink struct{ calls *[]string }

func (c callSink) Write(p []byte) (int, error) { return len(p), nil }

func joinedCalls(env *Env) string {
	sink, ok := env.Stderr.(callSink)
	if !ok {
		return ""
	}
	return strings.Join(*sink.calls, "\n")
}

func writeFakeIdentity(l Layout) error {
	if err := os.MkdirAll(l.IdentityDir, 0o700); err != nil {
		return err
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	body := "TEST ONLY identity material " + hex.EncodeToString(buf) + "\n"
	if err := os.WriteFile(filepath.Join(l.IdentityDir, "key.pem"), []byte(body), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(l.IdentityDir, "cert.pem"), []byte("cert "+hex.EncodeToString(buf)+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(l.IdentityDir, "ca.pem"), []byte("ca\n"), 0o600); err != nil {
		return err
	}
	raw, err := json.Marshal(nodeInfo{NodeID: uuid.NewString(), NodeIncarnationID: uuid.NewString()})
	if err != nil {
		return err
	}
	return os.WriteFile(l.nodeFile(), append(raw, '\n'), 0o600)
}

func mustRelease(t *testing.T, dir, id, seq string) ReleaseFixture {
	t.Helper()
	script := []byte("#!/bin/sh\nexit 0\n")
	fx, err := BuildReleaseFixture(filepath.Join(dir, "rel-"+seq+"-"+id[:8]), unitDir(t), id, seq, map[string][]byte{
		"fvpn-agent": script, "fvpn-updater": script, "fvpn-watchdog": script, "fvpn-node-install": script,
	})
	if err != nil {
		t.Fatal(err)
	}
	return fx
}

func mustReq(t *testing.T, dir string, fx ReleaseFixture) InstallRequest {
	t.Helper()
	token := filepath.Join(dir, "token-"+fx.Sequence)
	writeToken(t, token)
	ca := filepath.Join(dir, "ca-"+fx.Sequence+".pem")
	if err := os.WriteFile(ca, []byte("-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return InstallRequest{
		ManifestPath: fx.Manifest, ArtifactPath: fx.Artifact, TrustPath: fx.Trust,
		TokenFile: token, ControlURL: "https://control.example.invalid",
		ServerCA: ca, TrustBundle: fx.Trust, HostReserveBPS: 1_000_000,
		SNAT: "192.0.2.1", NetbirdManagementURL: "https://netbird.example.invalid",
	}
}

func writeToken(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test-only-enrollment-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hashFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func readInc(t *testing.T, l Layout) string {
	t.Helper()
	n, err := readNode(l)
	if err != nil {
		t.Fatal(err)
	}
	return n.NodeIncarnationID
}

func unitDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(dir, "deploy", "node", "systemd")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("deploy/node/systemd not found")
		}
		dir = parent
	}
}

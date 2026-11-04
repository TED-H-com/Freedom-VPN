package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// writeR29RecoverScript writes an executable fvpn-updater stand-in that uses
// the real kernel flock on lockPath to reproduce update.lock contention.
// When lockPath is held by another open descriptor, it prints the updater's
// UPDATE_BUSY stderr line and exits 3 (Option B ExitBusy). When lockPath is
// free, it exits with the code written in outcomePath ("0" or "2").
func writeR29RecoverScript(t *testing.T, dir, lockPath, outcomePath, countPath string) (scriptPath, cfgPath string) {
	t.Helper()
	scriptPath = filepath.Join(dir, "fake-updater-recover.sh")
	cfgPath = filepath.Join(dir, "updater.json")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outcomePath, []byte("0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
echo 1 >> %q
exec 9>>%q
if ! flock -n 9; then
	echo "fvpn-updater: updater: UPDATE_BUSY: lock" >&2
	exit 3
fi
code="$(tr -d ' \t\r\n' < %q)"
exit "${code:-0}"
`, countPath, lockPath, outcomePath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return scriptPath, cfgPath
}

func TestR29_H1_CandidateBootUnderHeldUpdateLockDefersRecoverAndServesAfterCommit(t *testing.T) {
	// TODO(review r29): H-1
	// Assumes Option B: when update.lock is held by a live same-boot updater during
	// candidate Boot (or fvpn-updater recover exits 3 / UPDATE_BUSY), the candidate
	// keeps recoverBlocked=true while contended and retries recover on Step once
	// update.lock is released, clearing recoverBlocked and serving without a restart.
	r := newRig(t)
	serveBoundGeneration(t, r)

	// Simulate fvpn-updater stepBlock / stepFinalSample / stepStop before stepStart:
	// the updater writes forwarding.blocked and stops the old agent while holding update.lock.
	barrierID := uuid.Must(uuid.NewV7()).String()
	blockedPath := filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")
	if err := writeAtomic(blockedPath, []byte(barrierID+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := r.eng.cfg
	bootID := r.eng.bootID
	log := r.eng.log
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	lock, err := nodelock.Acquire(context.Background(), cfg.LockPath, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()

	dir := t.TempDir()
	outcomePath := filepath.Join(dir, "recover-outcome")
	countPath := filepath.Join(dir, "recover-count")
	cfg.UpdaterRecover, cfg.UpdaterConfig = writeR29RecoverScript(t, dir, cfg.LockPath, outcomePath, countPath)

	cand, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		Log:         log,
		MeterSource: staticSource{boot: bootID.String()},
		Prober:      &verProbe{version: "0.76.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cand.Close()
	r.eng = cand

	// 1. stepStart: candidate boots while parent fvpn-updater holds update.lock and forwarding.blocked is set.
	if err := cand.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cand.serving || !hasReason(cand.decision.Reasons, api.ForwardingBlockReasonRecoveryRequired) {
		t.Fatalf("candidate served during stepStart under held lock: serving=%v reasons=%v", cand.serving, cand.decision.Reasons)
	}

	// 2. stepCheck: updater issues a D-053 health challenge with a D-064 probe while still holding update.lock.
	beforeCalls := len(r.plane.calls)
	ch := probeChallenge(t, r, labProbe(), 0)
	if err := cand.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cand.serving || !hasReason(cand.decision.Reasons, api.ForwardingBlockReasonRecoveryRequired) {
		t.Fatalf("candidate served during stepCheck under held lock: serving=%v reasons=%v", cand.serving, cand.decision.Reasons)
	}
	var installedProbe, clearedProbe *enforce.Model
	for _, c := range r.plane.calls[beforeCalls:] {
		if c.kind != "baseline_model" {
			continue
		}
		m := c.model
		if m.Probe != nil && installedProbe == nil {
			installedProbe = &m
			continue
		}
		if m.Probe == nil && installedProbe != nil && clearedProbe == nil {
			clearedProbe = &m
		}
	}
	if installedProbe == nil || clearedProbe == nil {
		t.Fatalf("D-064 probe exception did not install and clear while recover was deferred: install=%v clear=%v", installedProbe, clearedProbe)
	}
	requireAttest(t, r, ch)

	// 3. seal: parent updater unblocks forwarding.blocked and releases update.lock.
	if err := os.Remove(blockedPath); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	// 4. Subsequent Step(s) on the SAME candidate process must run recover once the lock
	// is free, clear recoverBlocked, and resume member serving without a second restart.
	if err := cand.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cand.serving {
		// Allow one D-067 provisional lease renewal step if the lease was renewed while applying.
		r.clock.Advance(provisionalRenewCap + time.Second)
		if err := cand.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if cand.recoverBlocked || !cand.serving || cand.appliedGen == 0 || len(r.plane.last.Bindings) == 0 {
		t.Fatalf("candidate stayed blocked after updater released update.lock and unblocked gate: recoverBlocked=%v latchBlocked=%v serving=%v appliedGen=%d reasons=%v",
			cand.recoverBlocked, cand.latchBlocked, cand.serving, cand.appliedGen, cand.decision.Reasons)
	}
}

func TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery(t *testing.T) {
	// TODO(review r29): M-1
	// Assumes Option B: when update.lock is free after an updater crash in StageChecking
	// (or StageBlocked) and fvpn-updater recover waits for the running agent to write
	// health-attest.json (or meter.checkpoint), Boot/Step bounds its synchronous wait so
	// the agent can answer the challenge while recover finishes in the background.
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentPublished)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}

	cfg := r.eng.cfg
	bootID := r.eng.bootID
	nodeID := r.eng.nodeID
	incID := r.eng.incID
	log := r.eng.log
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	// Simulate an updater that crashed in StageSwitched/StageChecking, leaving
	// forwarding.blocked on disk and update.lock free. When fvpn-updater recover
	// runs, it takes update.lock, writes a fresh health-challenge.json, and polls
	// for health-attest.json from the candidate agent before removing forwarding.blocked.
	blockedPath := filepath.Join(cfg.JournalDir, "forwarding.blocked")
	if err := writeAtomic(blockedPath, []byte(uuid.Must(uuid.NewV7()).String()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	challengeBytes, err := IssueHealthChallenge(
		uuid.Must(uuid.NewV7()).String(),
		cfg.ReleaseUUID().String(),
		nodeID.String(),
		incID.String(),
		bootID.String(),
		r.clock.Wall().UTC().Add(-time.Second),
		30*time.Second,
		cfg.AgentRelease,
		"0.76.2",
		"0.1.0",
		"1",
	)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	chSource := filepath.Join(dir, "challenge.json")
	if err := os.WriteFile(chSource, challengeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(dir, "recover-wait-attest.sh")
	cfgPath := filepath.Join(dir, "updater.json")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	chDest := filepath.Join(cfg.JournalDir, healthattest.ChallengeFile)
	attDest := filepath.Join(cfg.JournalDir, healthattest.AttestationFile)
	script := fmt.Sprintf(`#!/bin/sh
set -eu
exec 9>>%q
flock -n 9
cp %q %q
chmod 0600 %q
i=0
while [ "$i" -lt 40 ]; do
	if [ -f %q ]; then
		rm -f %q
		exit 0
	fi
	sleep 0.05
	i=$((i + 1))
done
exit 1
`, cfg.LockPath, chSource, chDest, chDest, attDest, blockedPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg.UpdaterRecover = scriptPath
	cfg.UpdaterConfig = cfgPath

	cand, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		Log:         log,
		MeterSource: staticSource{boot: bootID.String()},
		Prober:      &verProbe{version: "0.76.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cand.Close()

	// Boot uses bootLocalBudget (10s < the script's 15s wait and production's
	// HealthAttestWait=30s): if recoverOnce blocks synchronously inside Boot
	// before answerHealthOnBoot/Step can run, Boot times out with ErrBoot and
	// kills the recovery process before health-attest.json is written.
	if err := cand.Boot(context.Background()); err != nil {
		t.Fatalf("Boot self-deadlocked waiting for synchronous recoverOnce to receive health-attest.json from this same agent: %v", err)
	}
	for i := 0; i < 200 && (cand.recoverBlocked || cand.latchBlocked); i++ {
		if err := cand.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
		time.Sleep(25 * time.Millisecond)
	}
	if cand.recoverBlocked || cand.latchBlocked {
		t.Fatalf("open transaction recovery did not complete after agent answered health challenge: recoverBlocked=%v latchBlocked=%v",
			cand.recoverBlocked, cand.latchBlocked)
	}
}

func TestR29_Held_D053AndD064PassWhileRecoverBlockedUnderHeldUpdateLock(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)

	blockedPath := filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")
	if err := writeAtomic(blockedPath, []byte(uuid.Must(uuid.NewV7()).String()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := r.eng.cfg
	bootID := r.eng.bootID
	log := r.eng.log
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	lock, err := nodelock.Acquire(context.Background(), cfg.LockPath, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()

	dir := t.TempDir()
	outcomePath := filepath.Join(dir, "recover-outcome")
	countPath := filepath.Join(dir, "recover-count")
	cfg.UpdaterRecover, cfg.UpdaterConfig = writeR29RecoverScript(t, dir, cfg.LockPath, outcomePath, countPath)

	cand, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		Log:         log,
		MeterSource: staticSource{boot: bootID.String()},
		Prober:      &verProbe{version: "0.76.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cand.Close()
	r.eng = cand

	if err := cand.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	appliesBefore := r.plane.applies
	beforeCalls := len(r.plane.calls)
	ch := probeChallenge(t, r, labProbe(), 0)
	if err := cand.Step(context.Background()); err != nil {
		t.Fatal(err)
	}

	if cand.serving || r.plane.applies != appliesBefore || !hasReason(cand.decision.Reasons, api.ForwardingBlockReasonRecoveryRequired) {
		t.Fatalf("candidate must keep forwarding off while update.lock is held: serving=%v applies=%d->%d reasons=%v",
			cand.serving, appliesBefore, r.plane.applies, cand.decision.Reasons)
	}
	if other, err := nodelock.TryAcquire(cfg.LockPath); !errors.Is(err, nodelock.ErrBusy) {
		if other != nil {
			_ = other.Release()
		}
		t.Fatalf("parent update.lock must remain held during stepCheck: %v", err)
	}
	var install, cleared *enforce.Model
	for _, c := range r.plane.calls[beforeCalls:] {
		if c.kind != "baseline_model" {
			continue
		}
		m := c.model
		if m.Probe != nil && install == nil {
			install = &m
			continue
		}
		if m.Probe == nil && install != nil && cleared == nil {
			cleared = &m
		}
	}
	if install == nil || cleared == nil || install.Generation != 0 || cleared.Generation != 0 || len(install.Bindings) != 0 || len(cleared.Bindings) != 0 {
		t.Fatalf("D-064 baseline probe exception must install and clear with zero member bindings: install=%+v clear=%+v", install, cleared)
	}
	requireAttest(t, r, ch)
}

func TestR29_Held_TerminalRecoveryRequiredExit2KeepsForwardingOffAfterLockRelease(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)

	blockedPath := filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")
	if err := writeAtomic(blockedPath, []byte(uuid.Must(uuid.NewV7()).String()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := r.eng.cfg
	bootID := r.eng.bootID
	log := r.eng.log
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	lock, err := nodelock.Acquire(context.Background(), cfg.LockPath, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()

	dir := t.TempDir()
	outcomePath := filepath.Join(dir, "recover-outcome")
	countPath := filepath.Join(dir, "recover-count")
	cfg.UpdaterRecover, cfg.UpdaterConfig = writeR29RecoverScript(t, dir, cfg.LockPath, outcomePath, countPath)
	if err := os.WriteFile(outcomePath, []byte("2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cand, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		Log:         log,
		MeterSource: staticSource{boot: bootID.String()},
		Prober:      &verProbe{version: "0.76.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cand.Close()
	r.eng = cand

	if err := cand.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Release update.lock (and even remove forwarding.blocked to isolate exit code 2):
	// when fvpn-updater recover exits 2 (recovery_required), Step must keep forwarding off.
	_ = os.Remove(blockedPath)
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if err := cand.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cand.recoverBlocked || cand.serving || !hasReason(cand.decision.Reasons, api.ForwardingBlockReasonRecoveryRequired) {
		t.Fatalf("recover exit 2 (recovery_required) after lock release must keep forwarding off: recoverBlocked=%v serving=%v reasons=%v",
			cand.recoverBlocked, cand.serving, cand.decision.Reasons)
	}
}

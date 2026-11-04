package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
)

// recoverRetryInterval is the minimum between child starts, including busy
// exits and interruptions. Lock-only deferrals do not spawn a process.
const recoverRetryInterval = time.Second

// Only the first pass waits for the fast path. Later Steps poll the result
// without waiting, so the agent can supply the child's checkpoint/attestation.
const recoverFastWait = 250 * time.Millisecond

type recoveryJob struct {
	cancel      context.CancelFunc
	done        chan struct{}
	err         error // published by closing done; only Boot/Step applies it
	interrupted bool
}

func (e *Engine) recoverOnce(ctx context.Context) {
	if e.recovered {
		return
	}
	if e.cfg.UpdaterRecover == "" {
		e.recovered = true
		return
	}
	e.recoverBlocked = true
	if e.runCtx != nil && e.runCtx.Err() != nil {
		return // a stop request is not recovery failure evidence
	}
	e.recoverMu.Lock()
	defer e.recoverMu.Unlock()
	if e.recoverClosed {
		return
	}
	if e.recoverJob != nil {
		select {
		case <-e.recoverJob.done:
			e.finishRecovery()
		default:
		}
		return
	}
	if e.cfg.UpdaterConfig == "" {
		e.recovered = true
		return
	}
	// This precheck also supports an updater whose live-busy exit predates 3.
	if ctx.Err() != nil || e.clock.Mono() < e.recoverRetryAt || e.updateLockHeld() {
		return
	}
	parent := e.runCtx
	if parent == nil {
		parent = context.Background()
	}
	childCtx, cancel := context.WithCancel(parent)
	job := &recoveryJob{cancel: cancel, done: make(chan struct{})}
	e.recoverJob = job
	e.recoverRetryAt = e.clock.Mono() + recoverRetryInterval
	command, config := e.cfg.UpdaterRecover, e.cfg.UpdaterConfig
	go func() {
		defer close(job.done)
		defer cancel()
		cmd := exec.CommandContext(childCtx, command, "recover", "-config", config)
		// An updater may be waiting on its own helper. Cancel the child group
		// as well as the updater, and reap the direct child with Run/Wait.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error {
			err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			if errors.Is(err, syscall.ESRCH) {
				return os.ErrProcessDone
			}
			return err
		}
		job.err = cmd.Run()
		job.interrupted = childCtx.Err() != nil
	}()
	timer := time.NewTimer(recoverFastWait)
	defer timer.Stop()
	select {
	case <-job.done:
		e.finishRecovery()
	case <-ctx.Done():
		// The local Boot/Step budget is not the child's lifetime. A real
		// stop cancels runCtx, or Close cancels and joins explicitly.
	case <-timer.C:
	}
}

// Called only by Boot/Step with recoverMu held. No background goroutine
// mutates engine decision fields or touches its stores.
func (e *Engine) finishRecovery() {
	job := e.recoverJob
	e.recoverJob = nil
	if job.interrupted {
		return
	} // interrupted recovery supplies no failure evidence
	var ee *exec.ExitError
	if errors.As(job.err, &ee) && ee.ExitCode() == updater.ExitBusy {
		return
	}
	e.recovered = true
	e.recoverBlocked = job.err != nil
	if job.err != nil && !(errors.As(job.err, &ee) && ee.ExitCode() == 2) {
		e.log.Error("updater recover", "err", "forwarding stays blocked")
	}
}

func (e *Engine) stopRecovery() {
	e.recoverMu.Lock()
	defer e.recoverMu.Unlock()
	e.recoverClosed = true
	if job := e.recoverJob; job != nil {
		job.cancel()
		<-job.done
	}
}

// Package agentwait is the TEST ONLY lab agent parent. It preserves the actual
// wait status, including signal termination, without polling for a zombie.
package agentwait

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type Result struct {
	ID             string    `json:"id"`
	PID            int       `json:"pid"`
	StartTicks     string    `json:"start_ticks"`
	StartedAt      time.Time `json:"started_at"`
	WaitObservedAt time.Time `json:"wait_observed_at"`
	WaitStatus     uint32    `json:"wait_status"`
	ExitCode       *int      `json:"exit_code"`
	Signal         *int      `json:"signal"`
	CoreDump       bool      `json:"core_dump"`
}

func atomic(path string, raw []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".agent-wait-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(raw)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// Run does not change child arguments, signals, shutdown deadlines or restart
// policy. The existing shell supervisor signals the child directly. Each launch
// has a unique receipt; neither a restart nor a repeated stop overwrites it.
// WaitObservedAt is the parent's observation time, not an exact kernel exit time.
func Run(base, binary string, args ...string) error {
	if err := os.MkdirAll(filepath.Join(base, "agent-exits"), 0700); err != nil {
		return err
	}
	cmd := exec.Command(binary, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	r := Result{ID: uuid.NewString(), StartedAt: time.Now().UTC()}
	if err := cmd.Start(); err != nil {
		return err
	}
	r.PID = cmd.Process.Pid
	// This parent has not waited yet: even an immediate exit remains readable.
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", r.PID))
	if err == nil {
		fields := strings.Fields(string(raw[strings.LastIndexByte(string(raw), ')')+1:]))
		if len(fields) < 20 {
			err = fmt.Errorf("short process stat")
		} else {
			r.StartTicks = fields[19]
		}
	}
	if err == nil {
		err = atomic(filepath.Join(base, "agent-process-id"), []byte(r.ID+"\n"))
	}
	if err == nil {
		err = atomic(filepath.Join(base, "agent.pid"), []byte(strconv.Itoa(r.PID)+"\n"))
	}
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	waitErr := cmd.Wait()
	r.WaitObservedAt = time.Now().UTC()
	var exitErr *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exitErr) {
		return waitErr
	}
	status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		return fmt.Errorf("wait status unavailable")
	}
	r.WaitStatus = uint32(status)
	r.CoreDump = status.CoreDump()
	if status.Exited() {
		v := status.ExitStatus()
		r.ExitCode = &v
	}
	if status.Signaled() {
		v := int(status.Signal())
		r.Signal = &v
	}
	raw, err = json.Marshal(r)
	if err != nil {
		return err
	}
	return atomic(filepath.Join(base, "agent-exits", r.ID+".json"), append(raw, '\n'))
}

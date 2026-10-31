package agent

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// MeasurementHost is TEST ONLY context, not evidence that CPU contention caused
// an error. Both reads are outside Apply (including its install/release guard).
// proc_stat is the kernel-visible aggregate, not this process's cgroup quota.
// CPU counters are USER_HZ ticks; idle excludes iowait. Consumers may compute
// 100*delta(idle)/delta(total) only for two OK, ordered, nondecreasing samples.
type MeasurementHost struct {
	Before HostSample `json:"before_apply"`
	After  HostSample `json:"after_apply"`
	// These classify the returned error, never its text or command output.
	Command  string `json:"failed_command,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Signal   int    `json:"signal,omitempty"`
}

type HostSample struct {
	Start   time.Duration `json:"start_ns,string"`
	End     time.Duration `json:"end_ns,string"`
	OK      bool          `json:"ok"`
	Total   decstr.U64    `json:"cpu_total_ticks"`
	Idle    decstr.U64    `json:"cpu_idle_ticks"`
	IOWait  decstr.U64    `json:"cpu_iowait_ticks"`
	Running decstr.U64    `json:"procs_running"`
	Blocked decstr.U64    `json:"procs_blocked"`
}

func (e *Engine) measureHostBeforeApply() *MeasurementHost {
	if !e.measurementEnabled() {
		return nil
	}
	return &MeasurementHost{Before: readHostSample()}
}

func (h *MeasurementHost) finish(err error) {
	if h == nil {
		return
	}
	h.After = readHostSample()
	var command *enforce.CmdError
	if errors.As(err, &command) {
		switch command.Cmd {
		case "nft", "tc":
			h.Command = command.Cmd
		default:
			h.Command = "other"
		}
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code := exit.ExitCode()
		h.ExitCode = &code
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			h.Signal = int(status.Signal())
		}
	}
}

func readHostSample() HostSample {
	var s HostSample
	var err error
	s.Start, err = timeanchor.Mono()
	if err != nil {
		return s
	}
	raw, readErr := os.ReadFile("/proc/stat")
	s.End, err = timeanchor.Mono()
	if readErr != nil || err != nil || s.End < s.Start {
		return s
	}
	return parseHostSample(raw, s)
}

func parseHostSample(raw []byte, s HostSample) HostSample {
	seen := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || (f[0] != "cpu" && f[0] != "procs_running" && f[0] != "procs_blocked") {
			continue
		}
		if seen[f[0]] || len(f) < 2 || (f[0] == "cpu" && len(f) < 9) {
			return s
		}
		seen[f[0]] = true
		n := 1
		if f[0] == "cpu" {
			n = 8 // guest/guest_nice are already included in user/nice
		}
		for i := 1; i <= n; i++ {
			v, err := strconv.ParseUint(f[i], 10, 64)
			if err != nil {
				return s
			}
			switch f[0] {
			case "cpu":
				if v > ^uint64(0)-uint64(s.Total) {
					return s
				}
				s.Total += decstr.U64(v)
				if i == 4 {
					s.Idle = decstr.U64(v)
				}
				if i == 5 {
					s.IOWait = decstr.U64(v)
				}
			case "procs_running":
				s.Running = decstr.U64(v)
			case "procs_blocked":
				s.Blocked = decstr.U64(v)
			}
		}
	}
	s.OK = len(seen) == 3
	return s
}

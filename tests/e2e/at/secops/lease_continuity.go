//go:build e2e

package secops

import (
	"fmt"
	"strconv"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

const continuityCadence16 = time.Second
const expirySlack16 = 3 * time.Second  // followup11_transports_test.go lease_expiry
const expiryQuiet16 = 10 * time.Second // same existing post-deadline observation

type continuityBounds16 struct {
	Loss     float64 `json:"control_loss_monotonic_seconds"`
	Deadline float64 `json:"deadline_monotonic_seconds"`
	End      float64 `json:"continuity_end_monotonic_seconds"`
	Step     float64 `json:"step_interval_seconds"`
	Budget   float64 `json:"control_budget_seconds"`
	Stale    float64 `json:"watchdog_stale_seconds"`
}

func continuityEnd16(deadline float64) float64 {
	return deadline - (enforce.DeadlineInstallBound + enforce.DeadlineGranularity + continuityCadence16).Seconds()
}

type continuitySample16 struct {
	Transport     transportSample11 `json:"transport"`
	ReadStart     float64           `json:"read_start_monotonic_seconds"`
	ReadEnd       float64           `json:"read_end_monotonic_seconds"`
	MonoNS        int64             `json:"clock_monotonic_ns,string"`
	Heartbeat     map[string]string `json:"heartbeat"`
	AgentIdentity string            `json:"agent_pid_start_ticks"`
	AgentAlive    bool              `json:"agent_alive"`
}

type continuityVerdict16 struct {
	Passed        bool                          `json:"passed"`
	Errors        []string                      `json:"errors"`
	Boundary      map[string]map[string]float64 `json:"boundary_observations"`
	WindowSeconds float64                       `json:"continuity_window_seconds"`
	Stops         map[string]transportStop15    `json:"transport_stop_points"`
}

// D-140/D-142 remain the isolation lane's packet-level qualification. This
// sampler proves continuity at its declared one-second resolution, including
// heartbeat/liveness, and uses AT29's existing D+3s cutoff observation bound.
// It never moves the window in response to an early closure or a slow read.
func evaluateContinuity16(samples []continuitySample16, b continuityBounds16, timeoutProved bool) continuityVerdict16 {
	v := continuityVerdict16{Errors: []string{}, WindowSeconds: b.End - b.Loss}
	reject := func(s string) { v.Errors = append(v.Errors, s) }
	if b.Step <= 0 || b.Budget <= 0 || b.Stale <= 0 || b.End != continuityEnd16(b.Deadline) || v.WindowSeconds < 3*b.Step {
		reject("continuity window must reach D-101ms-1s and span at least three Step intervals")
	}
	if !timeoutProved {
		reject("agent log does not prove a full control budget timed out")
	}
	if len(samples) < 3 {
		reject("missing samples")
		return v
	}
	identity := samples[0].AgentIdentity
	covered := false
	transports := make([]transportSample11, 0, len(samples))
	for i, s := range samples {
		transports = append(transports, s.Transport)
		if !s.AgentAlive || identity == "" || s.AgentIdentity != identity {
			reject(fmt.Sprintf("sample %d: original agent not alive", i))
		}
		if s.ReadEnd < s.ReadStart || s.Transport.Elapsed < s.ReadStart || s.Transport.Elapsed > s.ReadEnd {
			reject(fmt.Sprintf("sample %d: invalid monotonic read interval", i))
		}
		if i > 0 && s.ReadStart <= samples[i-1].ReadEnd {
			reject(fmt.Sprintf("sample %d: overlapping/out-of-order reads", i))
		}
		if i > 1 && s.ReadStart-samples[i-1].ReadStart > b.Step {
			reject("sampling gap spans a Step interval")
		}
		if s.ReadEnd-s.ReadStart > continuityCadence16.Seconds() {
			reject("sample read exceeds declared one-second resolution")
		}
		if s.ReadStart <= b.End {
			if s.ReadEnd > b.Deadline-(enforce.DeadlineInstallBound+enforce.DeadlineGranularity).Seconds() {
				reject("continuity read crossed earliest permitted closure")
			}
			if s.Heartbeat["serving"] != "1" || s.Heartbeat["reason"] != "serving" {
				reject(fmt.Sprintf("sample %d: heartbeat not serving", i))
			}
			mono, err := strconv.ParseInt(s.Heartbeat["mono_ns"], 10, 64)
			if err != nil || mono <= 0 || mono > s.MonoNS || float64(s.MonoNS-mono)/1e9 > b.Stale {
				reject(fmt.Sprintf("sample %d: missing/stale heartbeat", i))
			}
			if i > 0 && !activeTransport11(samples[i-1].Transport, s.Transport) {
				reject(fmt.Sprintf("sample %d: forwarding interrupted before window end", i))
			}
		}
		// The final continuity sample is explicitly scheduled at End. It must
		// start by End+one cadence and finish before the closure boundary; no
		// wall-time or measured-gap extension of the continuity target is allowed.
		if s.ReadStart >= b.End && !covered {
			covered = true
			if s.ReadEnd > b.End+continuityCadence16.Seconds() || i == 0 || !activeTransport11(samples[i-1].Transport, s.Transport) || s.Heartbeat["serving"] != "1" || s.Heartbeat["reason"] != "serving" {
				reject("final continuity sample missing positive transports/serving heartbeat before closure")
			}
		}
	}
	if samples[0].ReadStart > b.Loss || !covered {
		reject("samples do not cover control loss through continuity end")
	}
	var cutoff bool
	v.Stops, cutoff = continuityStops16(transports, b.Deadline+expirySlack16.Seconds())
	if !cutoff || samples[len(samples)-1].ReadEnd < b.Deadline+(expirySlack16+expiryQuiet16).Seconds() {
		reject("cutoff not proved by D+3s and through the ten-second quiet window")
	}
	v.Boundary = map[string]map[string]float64{}
	for _, protocol := range []string{"TCP", "UDP"} {
		point := map[string]float64{}
		for i := 1; i < len(transports); i++ {
			prev, cur := transports[i-1], transports[i]
			x, y := prev.Status.TCPEchoes, cur.Status.TCPEchoes
			if protocol == "UDP" {
				x, y = prev.Status.UDPReplies, cur.Status.UDPReplies
			}
			if y > x && cur.Elapsed < b.Deadline {
				point["last_reply_observed_before_D"] = cur.Elapsed
			}
			if y == x && cur.Elapsed >= b.Deadline {
				if _, ok := point["first_quiet_observed_after_D"]; !ok {
					point["first_quiet_observed_after_D"] = cur.Elapsed
				}
			}
		}
		v.Boundary[protocol] = point
	}
	v.Passed = len(v.Errors) == 0
	return v
}

// Independent TCP attempts can take one second after cutoff and status is
// published every 100ms (fvpn-e2e-member-session). An individual one-second
// sample may therefore repeat the attempt count. Require nondecreasing counts
// throughout and advancing attempts over the complete quiet observation window.
func continuityStops16(samples []transportSample11, deadline float64) (map[string]transportStop15, bool) {
	out := map[string]transportStop15{}
	all := len(samples) >= 3
	for _, protocol := range []string{"TCP", "UDP"} {
		stop := transportStop15{Deadline: deadline}
		valid := len(samples) >= 3
		lastChange := 0
		for i, s := range samples {
			valid = valid && s.StatusOK && s.Status.Running && s.Status.Error == ""
			if i == 0 {
				continue
			}
			p := samples[i-1]
			x, y := p.Status.TCPEchoes, s.Status.TCPEchoes
			a, b := p.Status.TCPAttempts, s.Status.TCPAttempts
			if protocol == "UDP" {
				x, y = p.Status.UDPReplies, s.Status.UDPReplies
				a, b = p.Status.UDPAttempts, s.Status.UDPAttempts
			}
			valid = valid && s.Elapsed > p.Elapsed && y >= x && b >= a
			if y != x {
				lastChange = i
			}
		}
		if lastChange+1 < len(samples) {
			first, last := samples[lastChange+1], samples[len(samples)-1]
			a, b := first.Status.TCPAttempts, last.Status.TCPAttempts
			if protocol == "UDP" {
				a, b = first.Status.UDPAttempts, last.Status.UDPAttempts
			}
			stop.LastReplyObserved = samples[lastChange].Elapsed
			stop.FirstQuiet = first.Elapsed
			stop.Passed = valid && b > a && first.Elapsed <= deadline && last.Elapsed >= deadline+expiryQuiet16.Seconds()
		}
		out[protocol] = stop
		all = all && stop.Passed
	}
	return out, all
}

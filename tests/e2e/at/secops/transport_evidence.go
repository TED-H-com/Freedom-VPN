//go:build e2e

package secops

import (
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"time"
)

type transportSample11 struct {
	UTC        time.Time                      `json:"utc"`
	Elapsed    float64                        `json:"monotonic_elapsed_seconds"`
	Status     labstack.MemberTransportStatus `json:"status"`
	Counters   labstack.DeviceCounters        `json:"counters"`
	StatusOK   bool                           `json:"status_read_ok"`
	CountersOK bool                           `json:"counter_read_ok"`
}

func activeTransport11(a, b transportSample11) bool {
	return a.StatusOK && b.StatusOK && a.Status.Running && b.Status.Running &&
		b.Status.TCPEchoes > a.Status.TCPEchoes && b.Status.UDPReplies > a.Status.UDPReplies &&
		a.CountersOK && b.CountersOK && a.Counters.Present && b.Counters.Present &&
		b.Counters.Upload > a.Counters.Upload && b.Counters.Download > a.Counters.Download
}
func quietTransport11(a, b transportSample11) bool {
	// Absent counters are explicitly not interpreted as zero bytes. A revoked
	// binding may remove them; continuously running senders/replies decide cutoff.
	return a.StatusOK && b.StatusOK && a.Status.Running && b.Status.Running &&
		b.Status.TCPAttempts > a.Status.TCPAttempts && b.Status.UDPAttempts > a.Status.UDPAttempts &&
		b.Status.TCPEchoes == a.Status.TCPEchoes && b.Status.UDPReplies == a.Status.UDPReplies
}

type transportStop15 struct {
	LastReplyObserved float64 `json:"last_reply_observed_monotonic_seconds"`
	FirstQuiet        float64 `json:"first_quiet_monotonic_seconds"`
	Deadline          float64 `json:"deadline_monotonic_seconds"`
	Passed            bool    `json:"passed"`
}

// Report an observation interval, never invent the instant of the last packet.
// Every sample must be readable with both senders alive. The final reply change
// must be followed by quiet observations before the target and through the end.
func transportStops15(samples []transportSample11, deadline float64) (map[string]transportStop15, bool) {
	out := map[string]transportStop15{}
	all := true
	for _, protocol := range []string{"TCP", "UDP"} {
		stop := transportStop15{Deadline: deadline}
		valid := len(samples) >= 3
		lastChange := 0
		for i, s := range samples {
			valid = valid && s.StatusOK && s.Status.Running && s.Status.Error == ""
			if i == 0 {
				continue
			}
			previous := samples[i-1]
			x, y := previous.Status.TCPEchoes, s.Status.TCPEchoes
			a, b := previous.Status.TCPAttempts, s.Status.TCPAttempts
			if protocol == "UDP" {
				x, y = previous.Status.UDPReplies, s.Status.UDPReplies
				a, b = previous.Status.UDPAttempts, s.Status.UDPAttempts
			}
			valid = valid && s.Elapsed > previous.Elapsed && b >= a && y >= x
			if i > 1 {
				valid = valid && b > a
			}
			if y != x {
				lastChange = i
			}
		}
		if lastChange+1 < len(samples) {
			stop.LastReplyObserved = samples[lastChange].Elapsed
			stop.FirstQuiet = samples[lastChange+1].Elapsed
			stop.Passed = valid && stop.FirstQuiet <= deadline && samples[len(samples)-1].Elapsed >= deadline
		}
		out[protocol] = stop
		all = all && stop.Passed
	}
	return out, all
}

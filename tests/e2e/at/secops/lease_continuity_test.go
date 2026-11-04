//go:build e2e

package secops

import (
	"fmt"
	"testing"
)

func TestAT29_LiveAgentContinuitySynthetic(t *testing.T) {
	b := continuityBounds16{Loss: 1, Deadline: 40, End: continuityEnd16(40), Step: 5, Budget: 5, Stale: 30}
	makeSamples := func(stop int) []continuitySample16 {
		var out []continuitySample16
		for i := 0; i <= 54; i++ {
			s := continuitySample16{ReadStart: float64(i), ReadEnd: float64(i) + .05, MonoNS: int64(i+100) * 1e9, AgentAlive: true, AgentIdentity: "100 200", Heartbeat: map[string]string{"serving": "1", "reason": "serving", "mono_ns": fmt.Sprint(int64(i+100) * 1e9)}}
			s.Transport = transportSample11{Elapsed: float64(i) + .01, StatusOK: true, CountersOK: true}
			s.Transport.Status.Running = true
			s.Transport.Status.TCPAttempts = i + 1
			s.Transport.Status.UDPAttempts = i + 1
			n := min(i, stop) + 1
			s.Transport.Status.TCPEchoes = n
			s.Transport.Status.UDPReplies = n
			s.Transport.Counters.Present = true
			s.Transport.Counters.Upload = uint64(n * 100)
			s.Transport.Counters.Download = uint64(n * 100)
			if float64(i) > b.End+1 {
				s.Heartbeat["serving"] = "0"
				s.Heartbeat["reason"] = "lease_expired"
			}
			out = append(out, s)
		}
		return out
	}
	for _, tc := range []struct {
		name          string
		stop          int
		mutate        func([]continuitySample16)
		timeout, want bool
	}{
		{"continuous until D then quiet", 39, nil, true, true},
		{"H1 withdrawal five seconds after control loss", 6, nil, true, false},
		{"forwarding after D bound", 45, nil, true, false},
		{"refused control is not timeout evidence", 39, nil, false, false},
		{"agent restart", 39, func(s []continuitySample16) { s[10].AgentIdentity = "101 201" }, true, false},
		{"early heartbeat withdrawal", 39, func(s []continuitySample16) {
			s[6].Heartbeat["serving"] = "0"
			s[6].Heartbeat["reason"] = "safety_check_failed"
		}, true, false},
		{"UDP stops early", 39, func(s []continuitySample16) {
			for i := 6; i < len(s); i++ {
				s[i].Transport.Status.UDPReplies = 6
			}
		}, true, false},
		{"dead sender after D", 39, func(s []continuitySample16) { s[50].Transport.Status.Running = false }, true, false},
		{"stalled TCP sender throughout quiet", 39, func(s []continuitySample16) {
			for i := 40; i < len(s); i++ {
				s[i].Transport.Status.TCPAttempts = 40
			}
		}, true, false},
		{"missing heartbeat", 39, func(s []continuitySample16) { delete(s[10].Heartbeat, "mono_ns") }, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := makeSamples(tc.stop)
			if tc.mutate != nil {
				tc.mutate(s)
			}
			v := evaluateContinuity16(s, b, tc.timeout)
			if v.Passed != tc.want {
				t.Fatalf("verdict=%+v want=%v", v, tc.want)
			}
		})
	}
	t.Run("less than three Steps", func(t *testing.T) {
		short := b
		short.Loss = 30
		if evaluateContinuity16(makeSamples(39), short, true).Passed {
			t.Fatal("short continuity accepted")
		}
	})
}

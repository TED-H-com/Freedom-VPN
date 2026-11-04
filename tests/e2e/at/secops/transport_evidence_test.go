//go:build e2e

package secops

import "testing"

func TestAT29_WithdrawalStopPoints(t *testing.T) {
	base := make([]transportSample11, 5)
	for i := range base {
		base[i].Elapsed = float64(i * 20)
		base[i].StatusOK = true
		base[i].Status.Running = true
		base[i].Status.TCPAttempts = i + 1
		base[i].Status.UDPAttempts = i + 1
		base[i].Status.TCPEchoes = 1
		base[i].Status.UDPReplies = 1
	}
	for _, tc := range []struct {
		name   string
		change func([]transportSample11)
		want   bool
	}{
		{"both stop with senders alive", func([]transportSample11) {}, true},
		{"late TCP reply", func(s []transportSample11) { s[4].Status.TCPEchoes++ }, false},
		{"late UDP reply", func(s []transportSample11) { s[4].Status.UDPReplies++ }, false},
		{"dead sender", func(s []transportSample11) { s[4].Status.Running = false }, false},
		{"stalled TCP attempts", func(s []transportSample11) { s[4].Status.TCPAttempts = s[3].Status.TCPAttempts }, false},
		{"missing status", func(s []transportSample11) { s[2].StatusOK = false }, false},
		{"counter reset", func(s []transportSample11) { s[4].Status.UDPReplies = 0 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := append([]transportSample11(nil), base...)
			tc.change(s)
			_, got := transportStops15(s, 60)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

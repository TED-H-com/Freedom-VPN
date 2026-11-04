//go:build e2e

package secops

import "testing"

func TestAT29_TransportActivityPredicates(t *testing.T) {
	before := transportSample11{StatusOK: true, CountersOK: true}
	before.Status.Running = true
	before.Status.TCPAttempts, before.Status.UDPAttempts = 10, 10
	before.Status.TCPEchoes, before.Status.UDPReplies = 8, 8
	before.Counters.Present = true
	before.Counters.Upload, before.Counters.Download = 100, 200

	// Quiet traffic still has two active senders, with unchanged replies/bytes.
	quiet := before
	quiet.Status.TCPAttempts++
	quiet.Status.UDPAttempts++
	active := quiet
	active.Status.TCPEchoes++
	active.Status.UDPReplies++
	active.Counters.Upload += 50
	active.Counters.Download += 50

	tcpStalled, udpStalled := active, active
	tcpStalled.Status.TCPEchoes = before.Status.TCPEchoes
	udpStalled.Status.UDPReplies = before.Status.UDPReplies
	// The quiet predicate must not mistake a stopped sender for enforced silence.
	tcpSenderStopped, udpSenderStopped := quiet, quiet
	tcpSenderStopped.Status.TCPAttempts = before.Status.TCPAttempts
	udpSenderStopped.Status.UDPAttempts = before.Status.UDPAttempts

	for _, tc := range []struct {
		name       string
		after      transportSample11
		wantActive bool
		wantQuiet  bool
	}{
		{"both transports and byte counters advanced", active, true, false},
		{"only sender attempts advanced", quiet, false, true},
		{"TCP replies stalled while UDP advanced", tcpStalled, false, false},
		{"UDP replies stalled while TCP advanced", udpStalled, false, false},
		{"quiet replies but TCP sender stopped", tcpSenderStopped, false, false},
		{"quiet replies but UDP sender stopped", udpSenderStopped, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := activeTransport11(before, tc.after); got != tc.wantActive {
				t.Errorf("activeTransport11 = %v, want %v", got, tc.wantActive)
			}
			if got := quietTransport11(before, tc.after); got != tc.wantQuiet {
				t.Errorf("quietTransport11 = %v, want %v", got, tc.wantQuiet)
			}
		})
	}
}

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

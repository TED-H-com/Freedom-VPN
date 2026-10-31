//go:build e2e

package e2e

import (
	"slices"
	"testing"
)

func TestENR02_InstallerFixtureLoopbackOptInScope(t *testing.T) {
	s := &Stack{}
	for _, tc := range []struct {
		origin    string
		opt, want bool
	}{
		{"https://127.0.0.1:9444", true, true},
		{"https://127.0.0.1:9444", false, false},
		{"https://control.example.invalid", true, false},
		{"https://127.0.0.1:9445", true, false},
	} {
		got := slices.Contains(s.installArgs(installerSpec{ControlURL: tc.origin, AllowLoopback: tc.opt}), "--allow-loopback")
		if got != tc.want {
			t.Errorf("origin %s opt=%t: flag=%t", tc.origin, tc.opt, got)
		}
	}
}

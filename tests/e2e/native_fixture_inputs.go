package e2e

import "fmt"

// NativeFixtureInputs are declarations, not claims about platform coverage.
// Underlay affects only peer transport. The product still serves IPv4 members.
type NativeFixtureInputs struct {
	Underlay string `json:"underlay"` // ipv4, ipv6, dual
	MTU      int    `json:"mtu"`      // 1280..1500, isolated container interfaces only
	Protocol string `json:"protocol"` // tcp, udp, quic
	Platform string `json:"platform"` // linux, android, ios, windows, macos
}

func (in NativeFixtureInputs) Validate() error {
	if in.Underlay != "ipv4" && in.Underlay != "ipv6" && in.Underlay != "dual" {
		return fmt.Errorf("underlay must be ipv4, ipv6 or dual")
	}
	if in.MTU < 1280 || in.MTU > 1500 {
		return fmt.Errorf("MTU must be 1280..1500")
	}
	if in.Protocol != "tcp" && in.Protocol != "udp" && in.Protocol != "quic" {
		return fmt.Errorf("protocol must be tcp, udp or quic")
	}
	switch in.Platform {
	case "linux", "android", "ios", "windows", "macos":
	default:
		return fmt.Errorf("unknown platform")
	}
	return nil
}

type NativeFixtureSelection struct {
	Input     NativeFixtureInputs `json:"input"`
	Available bool                `json:"available"`
	Reason    string              `json:"reason"`
}

// SelectNativeFixture fails closed on workloads this container cannot execute.
// A caller may record a QUIC/platform request without silently using UDP/Linux.
func SelectNativeFixture(in NativeFixtureInputs) (NativeFixtureSelection, error) {
	out := NativeFixtureSelection{Input: in}
	if err := in.Validate(); err != nil {
		return out, err
	}
	if in.Platform != "linux" {
		out.Reason = "requires the named real platform and operator client; no emulation by the Linux container"
		return out, nil
	}
	if in.Protocol == "quic" {
		out.Reason = "requires a pinned QUIC client/server at the controlled origin; UDP echo is not QUIC"
		return out, nil
	}
	out.Available = true
	return out, nil
}

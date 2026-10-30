package config

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestD058_NodeEgressProbeValidation(t *testing.T) {
	if err := validateNodeEgressProbe(nil); err != nil {
		t.Fatal(err)
	}
	valid := api.EgressProbeTarget{Address: "192.0.2.53", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}
	for _, tc := range []struct {
		name  string
		probe api.EgressProbe
		valid bool
	}{
		{"omitted timeout", api.EgressProbe{Targets: []api.EgressProbeTarget{valid}}, true},
		{"minimum timeout", api.EgressProbe{Targets: []api.EgressProbeTarget{valid}, TimeoutMs: new(100)}, true},
		{"maximum timeout", api.EgressProbe{Targets: []api.EgressProbeTarget{valid}, TimeoutMs: new(10000)}, true},
		{"empty", api.EgressProbe{}, false},
		{"duplicate", api.EgressProbe{Targets: []api.EgressProbeTarget{valid, valid}}, false},
		{"mapped duplicate", api.EgressProbe{Targets: []api.EgressProbeTarget{valid, {Address: "::ffff:192.0.2.53", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}}}, false},
		{"too many", api.EgressProbe{Targets: make([]api.EgressProbeTarget, 5)}, false},
		{"short timeout", api.EgressProbe{Targets: []api.EgressProbeTarget{valid}, TimeoutMs: new(99)}, false},
		{"long timeout", api.EgressProbe{Targets: []api.EgressProbeTarget{valid}, TimeoutMs: new(10001)}, false},
		{"hostname", api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "example.invalid", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}}}, false},
		{"zone", api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "fe80::1%fv0", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}}}, false},
		{"network", api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "192.0.2.0/24", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}}}, false},
		{"embedded port", api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "192.0.2.53:443", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}}}, false},
		{"udp", api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "192.0.2.53", Port: 443, Protocol: "udp"}}}, false},
		{"zero port", api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "192.0.2.53", Protocol: api.EgressProbeTargetProtocolTcp}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateNodeEgressProbe(&tc.probe); (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
		})
	}
}

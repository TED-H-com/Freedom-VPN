package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// Isolate the historical tracer statement with the unchanged real verifier.
// Empty tc listings are only this unit fixture; no live observation is forged.
type traceListing26 struct{ raw string }

func (traceListing26) NftBatch(context.Context, string) error {
	return fmt.Errorf("unexpected mutation")
}
func (traceListing26) TCBatch(context.Context, string) error {
	return fmt.Errorf("unexpected mutation")
}
func (traceListing26) LookPath(name string) (string, error) { return name, nil }
func (x traceListing26) Output(_ context.Context, name string, _ ...string) ([]byte, error) {
	if name == "nft" {
		return []byte(x.raw), nil
	}
	return []byte(`[]`), nil
}

func TestAT01_HistoricalNodeTraceRejectedByServingVerifier(t *testing.T) {
	raw := `{"nftables":[{"chain":{"family":"inet","table":"fvtrace_test","name":"prerouting","type":"filter","hook":"prerouting","prio":-500,"policy":"accept"}},{"rule":{"family":"inet","table":"fvtrace_test","chain":"prerouting","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip6","field":"nexthdr"}},"right":"udp"}},{"mangle":{"key":{"meta":{"key":"nftrace"}},"value":1}}]}}]}`
	_, err := enforce.VerifyServing(t.Context(), traceListing26{raw}, enforce.Model{Datapath: enforce.Datapath{MemberIngress: "fvmb0", Uplink: "fvup0"}}, nil)
	if err == nil || enforce.ReasonCode(err) != enforce.ReasonIsolation || !strings.Contains(err.Error(), "table fvtrace_test family inet hook prerouting priority -500 chain prerouting statement mangle") {
		t.Fatalf("historical tracer was not the deciding rejection: %v", err)
	}
}

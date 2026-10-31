package httpwire

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type reviewSigner struct{ called bool }

func (s *reviewSigner) Sign(context.Context, envelope.Purpose, any) (envelope.Envelope, error) {
	s.called = true
	return envelope.Envelope{}, nil
}

func reviewDesiredState(t *testing.T) api.DesiredState {
	t.Helper()
	raw, err := os.ReadFile("../../../contracts/vectors/desired-state-signing.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Envelope struct {
			Signed api.DesiredState `json:"signed"`
		} `json:"envelope"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, err := canonjson.Marshal(doc.Envelope.Signed); err != nil {
		t.Fatal(err)
	}
	return doc.Envelope.Signed
}

// TestReviewR16_StateCacheDropsGenerationOnlyAdvance is the D-099 mechanism.
// A candidate whose only change is the generation (and the freshness timestamps
// the cache itself overwrites) must still be signed. wire-nodes follow-up 29
// owns the fix; this proof stays red until that lands.
func TestReviewR16_StateCacheDropsGenerationOnlyAdvance(t *testing.T) {
	previous := reviewDesiredState(t)
	candidate := previous
	candidate.ConfigGeneration = nodes.ParseU64(uint64(previous.ConfigGeneration) + 6)
	inner := &reviewSigner{}
	signer := cachedStateSigner{signer: inner, cached: &desiredstate.CompiledDesiredState{
		Envelope:   api.DesiredStateEnvelope{Signed: previous},
		Generation: uint64(previous.ConfigGeneration),
	}}
	_, err := signer.Sign(context.Background(), envelope.PurposeDesiredState, candidate)
	if errors.Is(err, errStateUnchanged) || !inner.called {
		t.Fatalf("TODO(review): D-099 cachedStateSigner dropped a generation-only advance from %s to %s (err=%v signed=%v)", previous.ConfigGeneration, candidate.ConfigGeneration, err, inner.called)
	}
}

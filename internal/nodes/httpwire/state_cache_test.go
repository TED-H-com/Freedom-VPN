package httpwire

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type captureStateSigner struct{ signed any }

func (s *captureStateSigner) Sign(_ context.Context, _ envelope.Purpose, signed any) (envelope.Envelope, error) {
	s.signed = signed
	return envelope.Envelope{}, nil
}

func TestD099_StateCacheGenerationComparison(t *testing.T) {
	for _, tc := range []struct {
		name              string
		cached, candidate decstr.U64
		changeContent     bool
		unchanged         bool
	}{
		{"same_generation", 5, 5, false, true},
		{"ordinary_compile_proposal", 5, 6, false, true},
		{"higher_generation_fence", 2, 6, false, false},
		{"lower_generation", 5, 4, false, false},
		{"overflow_is_not_ordinary_proposal", decstr.U64(^uint64(0)), 0, false, false},
		{"changed_content", 5, 6, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC()
			previous := api.DesiredState{ConfigGeneration: tc.cached, IssuedAt: now, ExpiresAt: now.Add(time.Hour), PublicationIntent: api.PublicationIntentUnpublished}
			candidate := previous
			candidate.ConfigGeneration = tc.candidate
			candidate.IssuedAt, candidate.ExpiresAt = now.Add(time.Minute), now.Add(2*time.Hour)
			if tc.changeContent {
				candidate.PublicationIntent = api.PublicationIntentDisabled
			}
			delegate := &captureStateSigner{}
			cache := &desiredstate.CompiledDesiredState{Envelope: api.DesiredStateEnvelope{Signed: previous}, Generation: uint64(tc.cached)}
			signer := cachedStateSigner{signer: delegate, cached: cache}
			_, err := signer.Sign(t.Context(), envelope.PurposeDesiredState, candidate)
			if tc.unchanged {
				if !errors.Is(err, errStateUnchanged) || delegate.signed != nil {
					t.Fatalf("unchanged state signed: err=%v", err)
				}
			} else if err != nil || !reflect.DeepEqual(delegate.signed, candidate) {
				t.Fatalf("generation/freshness/content changed before signing: err=%v", err)
			}
			if !reflect.DeepEqual(cache.Envelope.Signed, previous) {
				t.Fatal("comparison mutated the cache")
			}
		})
	}
}

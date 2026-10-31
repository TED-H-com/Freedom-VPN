package httpwire

import (
	"bytes"
	"context"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

var errStateUnchanged = errors.New("desired state inputs unchanged")

type cachedStateSigner struct {
	signer desiredstate.Signer
	cached *desiredstate.CompiledDesiredState
}

func (s cachedStateSigner) Sign(ctx context.Context, purpose envelope.Purpose, signed any) (envelope.Envelope, error) {
	if desired, ok := signed.(api.DesiredState); ok && purpose == envelope.PurposeDesiredState && s.cached != nil {
		previous := s.cached.Envelope.Signed
		// CompileTx always proposes max(desired, applied, stored)+1, even for
		// unchanged content. Only its ordinary one-step proposal can be elided;
		// a different generation fence must reach the signer (D-099). Runtime
		// also excludes outstanding restore holds from cache eligibility.
		if desired.ConfigGeneration != previous.ConfigGeneration &&
			(desired.ConfigGeneration == 0 || desired.ConfigGeneration-1 != previous.ConfigGeneration) {
			return s.signer.Sign(ctx, purpose, signed)
		}
		// CompileTx has read all member eligibility, FUP and binding evidence but
		// has not written any rows at this point. Compare the entire document,
		// excluding freshness and the checked one-step proposal, instead of
		// duplicating compiler logic.
		desired.ConfigGeneration = previous.ConfigGeneration
		desired.IssuedAt, desired.ExpiresAt = previous.IssuedAt, previous.ExpiresAt
		a, err := canonjson.Marshal(desired)
		if err != nil {
			return envelope.Envelope{}, err
		}
		b, err := canonjson.Marshal(previous)
		if err != nil {
			return envelope.Envelope{}, err
		}
		if bytes.Equal(a, b) {
			return envelope.Envelope{}, errStateUnchanged
		}
	}
	return s.signer.Sign(ctx, purpose, signed)
}

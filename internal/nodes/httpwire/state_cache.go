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
		// CompileTx has read all member eligibility, FUP and binding evidence but
		// has not written any rows at this point. Compare the entire document,
		// excluding only freshness fields, instead of duplicating compiler logic.
		previous := s.cached.Envelope.Signed
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

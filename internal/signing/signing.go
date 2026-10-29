// Package signing separates central signing, node CA and bootstrap encryption
// keys from release verification trust (§13.4, §16.2). FileProvider is for dev;
// production providers must maintain key state outside database restore scope.
package signing

import (
	"context"
	"crypto"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type Purpose string

const (
	DesiredState    Purpose = "desired-state"
	Lease           Purpose = "lease"
	Recovery        Purpose = "recovery"
	LedgerEpoch     Purpose = "ledger-epoch"
	NodeCA          Purpose = "node-ca"
	BootstrapSecret Purpose = "bootstrap-secret"
	Release         Purpose = "release"
)

var ErrUnavailable = errors.New("signing: key unavailable or unauthorized")

// Provider never exports private key bytes. SignEnvelope lets a remote KMS
// produce pkg/envelope's wire format without exposing an Ed25519 private key.
// CertificateSigner must recheck authorization on every Sign, even for handles
// acquired before revocation. Seal/Open bind ciphertext to caller-supplied AAD.
type Provider interface {
	SignEnvelope(context.Context, Purpose, any) (envelope.Envelope, error)
	CertificateSigner(context.Context, string) (crypto.Signer, error)
	Seal(context.Context, []byte, []byte) (Sealed, error)
	Open(context.Context, Sealed, []byte) ([]byte, error)
	TrustBundle(context.Context) (*envelope.TrustStore, error)
}

type Sealed struct {
	KeyID      string `json:"key_id"`
	Ciphertext []byte `json:"ciphertext"`
}

// Signer is bound to one purpose at construction. Consumers receive this
// interface, not a provider permitting arbitrary purpose selection.
type Signer interface {
	Sign(context.Context, any) (envelope.Envelope, error)
}
type purposeSigner struct {
	p       Provider
	purpose Purpose
}

func ForPurpose(p Provider, purpose Purpose) (Signer, error) {
	if p == nil || !central(purpose) {
		return nil, ErrUnavailable
	}
	return purposeSigner{p, purpose}, nil
}
func (s purposeSigner) Sign(ctx context.Context, v any) (envelope.Envelope, error) {
	return s.p.SignEnvelope(ctx, s.purpose, v)
}
func central(p Purpose) bool {
	return p == DesiredState || p == Lease || p == Recovery || p == LedgerEpoch
}

// Package auth implements the role and browser identity boundary (§9.1, §16.1).
package auth

import (
	"context"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	Admin  Role = "admin"
	Member Role = "member"
	Agent  Role = "agent"
)

type Principal struct {
	Role Role
	ID   uuid.UUID
	// SessionID is the authenticated database session row, never the cookie.
	// It is used only for browser rate keys; Scope keeps resource ownership.
	SessionID       uuid.UUID `json:"-"`
	NodeID          uuid.UUID
	IncarnationID   uuid.UUID
	DisplayLabel    string
	MFAVerified     bool
	AuthenticatedAt time.Time
	CSRFToken       string
}

func (p Principal) Scope() string {
	if p.Role == Agent {
		return "agent:" + p.NodeID.String() + ":" + p.IncarnationID.String()
	}
	return string(p.Role) + ":" + p.ID.String()
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// AgentAuthenticator must check the CURRENT database lifecycle, incarnation and
// revocation state on every call, including certificates within their TLS validity.
// Published/disabled forwarding intent must not revoke the control channel.
type AgentAuthenticator interface {
	AuthenticateAgent(context.Context, *x509.Certificate) (Principal, error)
}
type AgentAuthenticatorFunc func(context.Context, *x509.Certificate) (Principal, error)

func (f AgentAuthenticatorFunc) AuthenticateAgent(c context.Context, x *x509.Certificate) (Principal, error) {
	return f(c, x)
}

type SessionAuthenticator interface {
	AuthenticateSession(*http.Request) (Principal, error)
}
type SessionAuthenticatorFunc func(*http.Request) (Principal, error)

func (f SessionAuthenticatorFunc) AuthenticateSession(r *http.Request) (Principal, error) {
	return f(r)
}

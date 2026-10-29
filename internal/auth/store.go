package auth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Flow struct {
	State, Browser, Nonce, Verifier string
	Expires                         time.Time
}
type Identity struct {
	Issuer, ClientScope, Subject, Label string
	Role                                Role
	MFA                                 bool
	AuthTime, Expires                   time.Time
}
type Store interface {
	SaveFlow(context.Context, Flow) error
	ConsumeFlow(context.Context, string, string) (Flow, error)
	Login(context.Context, Identity, string, string, time.Duration, uuid.UUID) (Principal, error)
	Session(context.Context, string, time.Duration) (Principal, error)
	Logout(context.Context, string) error
}
type PostgresStore struct{ Pool *pgxpool.Pool }

func digest(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }
func (s PostgresStore) SaveFlow(ctx context.Context, f Flow) error {
	_, e := s.Pool.Exec(ctx, `INSERT INTO oidc_login_flows(state_hash,browser_hash,nonce,verifier,expires_at) VALUES($1,$2,$3,$4,$5)`, digest(f.State), digest(f.Browser), f.Nonce, f.Verifier, f.Expires)
	return e
}
func (s PostgresStore) ConsumeFlow(ctx context.Context, state, browser string) (Flow, error) {
	var f Flow
	e := s.Pool.QueryRow(ctx, `DELETE FROM oidc_login_flows WHERE state_hash=$1 AND browser_hash=$2 AND expires_at>clock_timestamp() RETURNING nonce,verifier,expires_at`, digest(state), digest(browser)).Scan(&f.Nonce, &f.Verifier, &f.Expires)
	return f, e
}
func (s PostgresStore) Login(ctx context.Context, i Identity, token, old string, absolute time.Duration, request uuid.UUID) (p Principal, err error) {
	if i.Subject == "" || i.Issuer == "" || i.ClientScope == "" || token == "" || (i.Role != Admin && i.Role != Member) {
		return p, errors.New("invalid login identity")
	}
	p = Principal{Role: i.Role, DisplayLabel: i.Label, MFAVerified: i.MFA, AuthenticatedAt: i.AuthTime}
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		// Serialize first login by the scoped identity, never by an email (§9.1).
		scoped, _ := json.Marshal([]string{i.Issuer, i.ClientScope, i.Subject})
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, string(scoped)); e != nil {
			return e
		}
		var closed bool
		e := tx.QueryRow(ctx, `SELECT b.member_id,m.status='closed' FROM identity_bindings b JOIN members m ON m.id=b.member_id WHERE issuer=$1 AND client_scope=$2 AND subject=$3`, i.Issuer, i.ClientScope, i.Subject).Scan(&p.ID, &closed)
		if errors.Is(e, pgx.ErrNoRows) {
			p.ID = uuid.Must(uuid.NewV7())
			binding := uuid.Must(uuid.NewV7())
			if _, e = tx.Exec(ctx, `INSERT INTO members(id,display_name,status) VALUES($1,$2,'pending_payment')`, p.ID, i.Label); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,$3,$4,$5,'sub','oidc',clock_timestamp())`, binding, p.ID, i.Issuer, i.ClientScope, i.Subject); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE members SET canonical_identity_ref=$2 WHERE id=$1`, p.ID, binding); e != nil {
				return e
			}
			e = db.AppendAudit(ctx, tx, db.AuditEvent{Actor: p.ID.String(), ActorType: string(i.Role), Action: "member.register", Resource: "member:" + p.ID.String(), Result: "succeeded", RequestID: &request, SchemaVersion: 1, Before: json.RawMessage(`{}`), After: json.RawMessage(`{"status":"pending_payment"}`)})
		}
		if e != nil {
			return e
		}
		if closed {
			return apierr.New(apierr.Forbidden, "Account is closed.")
		}
		if old != "" {
			if _, e = tx.Exec(ctx, `DELETE FROM browser_sessions WHERE token_hash=$1`, digest(old)); e != nil {
				return e
			}
		}
		expires := time.Now().UTC().Add(absolute)
		if i.Expires.Before(expires) {
			expires = i.Expires
		}
		_, e = tx.Exec(ctx, `INSERT INTO browser_sessions(token_hash,member_id,role,display_label,mfa_verified,authenticated_at,absolute_expires_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, digest(token), p.ID, p.Role, p.DisplayLabel, p.MFAVerified, p.AuthenticatedAt, expires)
		return e
	})
	return
}
func (s PostgresStore) Session(ctx context.Context, token string, idle time.Duration) (p Principal, err error) {
	err = s.Pool.QueryRow(ctx, `UPDATE browser_sessions s SET last_seen_at=clock_timestamp() FROM members m WHERE token_hash=$1 AND s.member_id=m.id AND m.status<>'closed' AND absolute_expires_at>clock_timestamp() AND last_seen_at>clock_timestamp()-$2::bigint*interval '1 millisecond' RETURNING s.member_id,s.role,s.display_label,s.mfa_verified,s.authenticated_at`, digest(token), idle.Milliseconds()).Scan(&p.ID, &p.Role, &p.DisplayLabel, &p.MFAVerified, &p.AuthenticatedAt)
	return
}
func (s PostgresStore) Logout(ctx context.Context, token string) error {
	_, e := s.Pool.Exec(ctx, `DELETE FROM browser_sessions WHERE token_hash=$1`, digest(token))
	return e
}

// Cleanup expires only authentication scratch, never member or audit history.
func (s PostgresStore) Cleanup(ctx context.Context, idle time.Duration) error {
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `DELETE FROM oidc_login_flows WHERE expires_at<=clock_timestamp()`); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `DELETE FROM browser_sessions WHERE absolute_expires_at<=clock_timestamp() OR last_seen_at<=clock_timestamp()-$1::bigint*interval '1 millisecond'`, idle.Milliseconds())
		return e
	})
}

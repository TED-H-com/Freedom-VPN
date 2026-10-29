package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// loginAdmin shares no membership identity or session rows (§16.1, D-032).
func (s PostgresStore) loginAdmin(ctx context.Context, i Identity, token, old string, absolute time.Duration, request uuid.UUID) (p Principal, err error) {
	if !i.MFA {
		return p, apierr.New(apierr.Forbidden, "Administrator MFA required.")
	}
	p = Principal{Role: Admin, DisplayLabel: i.Label, MFAVerified: true, AuthenticatedAt: i.AuthTime}
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		scoped, _ := json.Marshal([]string{"admin", i.Issuer, i.ClientScope, i.Subject})
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, string(scoped)); e != nil {
			return e
		}
		e := tx.QueryRow(ctx, `SELECT id FROM admin_identities WHERE issuer=$1 AND client_scope=$2 AND subject=$3`, i.Issuer, i.ClientScope, i.Subject).Scan(&p.ID)
		if errors.Is(e, pgx.ErrNoRows) {
			p.ID = uuid.Must(uuid.NewV7())
			_, e = tx.Exec(ctx, `INSERT INTO admin_identities(id,issuer,client_scope,subject) VALUES($1,$2,$3,$4)`, p.ID, i.Issuer, i.ClientScope, i.Subject)
		}
		if e != nil {
			return e
		}
		if old != "" {
			if _, e = tx.Exec(ctx, `DELETE FROM admin_sessions WHERE token_hash=$1`, digest(old)); e != nil {
				return e
			}
		}
		expires := time.Now().UTC().Add(absolute)
		if i.Expires.Before(expires) {
			expires = i.Expires
		}
		_, e = tx.Exec(ctx, `INSERT INTO admin_sessions(id,token_hash,admin_id,display_label,mfa_verified,authenticated_at,absolute_expires_at) VALUES($1,$2,$3,$4,true,$5,$6)`, uuid.Must(uuid.NewV7()), digest(token), p.ID, p.DisplayLabel, p.AuthenticatedAt, expires)
		if e != nil {
			return e
		}
		return appendSessionAudit(ctx, tx, p, "session.login", request)
	})
	return
}

// The auth package cannot import audit (its HTTP reader imports auth). These
// closed-set auth events use contract enums and never include credential data.
func appendSessionAudit(ctx context.Context, tx pgx.Tx, p Principal, action string, request uuid.UUID) error {
	if p.ID == uuid.Nil || (p.Role != Admin && p.Role != Member) || (action != "session.login" && action != "session.logout") {
		return errors.New("invalid session audit")
	}
	resource := api.ResourceTypeMember
	if p.Role == Admin {
		resource = api.ResourceTypeAdminIdentity
	}
	if request == uuid.Nil {
		request = uuid.Must(uuid.NewV7())
	}
	return db.AppendAudit(ctx, tx, db.AuditEvent{Actor: p.ID.String(), ActorType: string(p.Role), Action: action, Resource: string(resource) + ":" + p.ID.String(), Result: string(api.AuditEventResultSucceeded), RequestID: &request, SchemaVersion: 1, Before: []byte(`{}`), After: []byte(`{}`)})
}

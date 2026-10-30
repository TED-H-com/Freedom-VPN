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

type adminIdentityRow struct {
	id          uuid.UUID
	clientScope string
}

func lookupAdminIdentities(ctx context.Context, tx pgx.Tx, canonIssuer, rawIssuer, subject string) ([]adminIdentityRow, error) {
	rows, err := tx.Query(ctx, `SELECT id,client_scope FROM admin_identities WHERE (issuer=$1 OR issuer=$2) AND subject=$3 ORDER BY created_at ASC,id ASC LIMIT 2`, canonIssuer, rawIssuer, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []adminIdentityRow
	for rows.Next() {
		var r adminIdentityRow
		if err := rows.Scan(&r.id, &r.clientScope); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// loginAdmin shares no membership identity or session rows (§16.1, D-032).
func (s PostgresStore) loginAdmin(ctx context.Context, i Identity, rawIssuer, token, old string, absolute time.Duration, request uuid.UUID) (p Principal, err error) {
	if !i.MFA {
		return p, apierr.New(apierr.Forbidden, "Administrator MFA required.")
	}
	p = Principal{Role: Admin, DisplayLabel: i.Label, MFAVerified: true, AuthenticatedAt: i.AuthTime}
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		scoped, _ := json.Marshal([]string{"admin", i.Issuer, i.Subject})
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, string(scoped)); e != nil {
			return e
		}
		found, e := lookupAdminIdentities(ctx, tx, i.Issuer, rawIssuer, i.Subject)
		if e != nil {
			return e
		}
		if len(found) > 1 {
			return apierr.New(apierr.Forbidden, "Administrator identity is ambiguous.")
		}
		if len(found) == 1 {
			p.ID = found[0].id
			if found[0].clientScope != i.ClientScope {
				if _, e = tx.Exec(ctx, `UPDATE admin_identities SET client_scope=$2 WHERE id=$1`, p.ID, i.ClientScope); e != nil {
					return e
				}
			}
		} else {
			if _, e = tx.Exec(ctx, `SAVEPOINT auth_register_admin`); e != nil {
				return e
			}
			adminID := uuid.Must(uuid.NewV7())
			_, e = tx.Exec(ctx, `INSERT INTO admin_identities(id,issuer,client_scope,subject) VALUES($1,$2,$3,$4)`, adminID, i.Issuer, i.ClientScope, i.Subject)
			if isUniqueViolation(e) {
				if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT auth_register_admin`); rbErr != nil {
					return rbErr
				}
				if _, rlErr := tx.Exec(ctx, `RELEASE SAVEPOINT auth_register_admin`); rlErr != nil {
					return rlErr
				}
				rechecked, selErr := lookupAdminIdentities(ctx, tx, i.Issuer, rawIssuer, i.Subject)
				if selErr != nil {
					return selErr
				}
				if len(rechecked) != 1 {
					return apierr.New(apierr.Forbidden, "Administrator identity is ambiguous.")
				}
				p.ID = rechecked[0].id
				if rechecked[0].clientScope != i.ClientScope {
					if _, e = tx.Exec(ctx, `UPDATE admin_identities SET client_scope=$2 WHERE id=$1`, p.ID, i.ClientScope); e != nil {
						return e
					}
				}
			} else if e != nil {
				return e
			} else {
				p.ID = adminID
				if _, e = tx.Exec(ctx, `RELEASE SAVEPOINT auth_register_admin`); e != nil {
					return e
				}
			}
		}
		if old != "" {
			var oldAdminID uuid.UUID
			e = tx.QueryRow(ctx, `DELETE FROM admin_sessions WHERE token_hash=$1 RETURNING admin_id`, digest(old)).Scan(&oldAdminID)
			if e != nil && !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
			if e == nil && oldAdminID != uuid.Nil && oldAdminID != p.ID {
				if e = appendSessionAuditWithAfter(ctx, tx, Principal{ID: oldAdminID, Role: Admin}, "session.logout", request, []byte(`{"reason":"superseded"}`)); e != nil {
					return e
				}
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

const (
	systemAuditActor = "00000000-0000-0000-0000-000000000003"
	fleetConfigID    = "00000000-0000-0000-0000-000000000001"
)

// AuditLoginRejection records a rejected OIDC callback without storing any token.
func (s PostgresStore) AuditLoginRejection(ctx context.Context, reason string, request uuid.UUID) error {
	if request == uuid.Nil {
		request = uuid.Must(uuid.NewV7())
	}
	after, err := json.Marshal(map[string]string{"reason": reason})
	if err != nil {
		return err
	}
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         systemAuditActor,
			ActorType:     string(api.AuditEventActorTypeSystem),
			Action:        "session.login",
			Resource:      string(api.ResourceTypeFleetConfig) + ":" + fleetConfigID,
			Result:        string(api.AuditEventResultDenied),
			RequestID:     &request,
			SchemaVersion: 1,
			Before:        []byte(`{}`),
			After:         after,
		})
	})
}

// The auth package cannot import audit (its HTTP reader imports auth). These
// closed-set auth events use contract enums and never include credential data.
func appendSessionAudit(ctx context.Context, tx pgx.Tx, p Principal, action string, request uuid.UUID) error {
	return appendSessionAuditWithAfter(ctx, tx, p, action, request, []byte(`{}`))
}

func appendSessionAuditWithAfter(ctx context.Context, tx pgx.Tx, p Principal, action string, request uuid.UUID, after []byte) error {
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
	if len(after) == 0 {
		after = []byte(`{}`)
	}
	return db.AppendAudit(ctx, tx, db.AuditEvent{Actor: p.ID.String(), ActorType: string(p.Role), Action: action, Resource: string(resource) + ":" + p.ID.String(), Result: string(api.AuditEventResultSucceeded), RequestID: &request, SchemaVersion: 1, Before: []byte(`{}`), After: after})
}

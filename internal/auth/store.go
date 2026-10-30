package auth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Flow struct {
	State, Browser, Nonce, Verifier string
	Role                            Role
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
type loginRejectionAuditor interface {
	AuditLoginRejection(ctx context.Context, reason string, request uuid.UUID) error
}
type PostgresStore struct{ Pool *pgxpool.Pool }

func digest(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }
func (s PostgresStore) SaveFlow(ctx context.Context, f Flow) error {
	var role *string
	if f.Role != "" {
		r := string(f.Role)
		role = &r
	}
	_, e := s.Pool.Exec(ctx, `INSERT INTO oidc_login_flows(state_hash,browser_hash,nonce,verifier,role,expires_at,id) VALUES($1,$2,$3,$4,$5,$6,$7)`, digest(f.State), digest(f.Browser), f.Nonce, f.Verifier, role, f.Expires, uuid.Must(uuid.NewV7()))
	return e
}
func (s PostgresStore) ConsumeFlow(ctx context.Context, state, browser string) (Flow, error) {
	var f Flow
	var role *string
	e := s.Pool.QueryRow(ctx, `DELETE FROM oidc_login_flows WHERE state_hash=$1 AND browser_hash=$2 AND expires_at>clock_timestamp() RETURNING nonce,verifier,role,expires_at`, digest(state), digest(browser)).Scan(&f.Nonce, &f.Verifier, &role, &f.Expires)
	if e != nil {
		return f, e
	}
	if role != nil {
		f.Role = Role(*role)
	}
	if after, ok := strings.CutPrefix(f.Verifier, string(Admin)+":"); ok {
		if f.Role == "" {
			f.Role = Admin
		}
		f.Verifier = after
	} else if after, ok := strings.CutPrefix(f.Verifier, string(Member)+":"); ok {
		if f.Role == "" {
			f.Role = Member
		}
		f.Verifier = after
	}
	return f, nil
}
func canonicalIssuer(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || strings.TrimSpace(raw) != raw || strings.ContainsAny(raw, "?#") || u.User != nil || u.Opaque != "" || u.Hostname() == "" || strings.HasSuffix(u.Host, ":") {
		return raw
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "https" && u.Scheme != "http" {
		return raw
	}
	host, port := strings.ToLower(u.Hostname()), u.Port()
	if u.Scheme == "https" && port == "443" || u.Scheme == "http" && port == "80" {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	u.Host = host
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	return u.String()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type memberBindingRow struct {
	bindingID   uuid.UUID
	memberID    uuid.UUID
	clientScope string
	closed      bool
}

func lookupMemberBindings(ctx context.Context, tx pgx.Tx, canonIssuer, rawIssuer, subject string) ([]memberBindingRow, error) {
	rows, err := tx.Query(ctx, `SELECT b.id,b.member_id,b.client_scope,m.status='closed' FROM identity_bindings b JOIN members m ON m.id=b.member_id WHERE (b.issuer=$1 OR b.issuer=$2) AND b.subject=$3 ORDER BY b.id LIMIT 2`, canonIssuer, rawIssuer, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []memberBindingRow
	for rows.Next() {
		var r memberBindingRow
		if err := rows.Scan(&r.bindingID, &r.memberID, &r.clientScope, &r.closed); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func updateMemberBindingScope(ctx context.Context, tx pgx.Tx, memberID, bindingID uuid.UUID, role Role, clientScope string, request uuid.UUID) error {
	if _, err := tx.Exec(ctx, `SAVEPOINT auth_update_scope`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE identity_bindings SET client_scope=$2,verified_at=clock_timestamp() WHERE id=$1`, bindingID, clientScope); err != nil {
		if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT auth_update_scope`); rbErr != nil {
			return rbErr
		}
		if _, err := tx.Exec(ctx, `UPDATE identity_bindings SET verified_at=clock_timestamp() WHERE id=$1`, bindingID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT auth_update_scope`); err != nil {
		return err
	}
	after, err := json.Marshal(map[string]any{
		"identity_binding_id": bindingID,
		"reason":              "proof_updated",
	})
	if err != nil {
		return err
	}
	return db.AppendAudit(ctx, tx, db.AuditEvent{
		Actor:         memberID.String(),
		ActorType:     string(role),
		Action:        "identity_binding.proof_updated",
		Resource:      "member:" + memberID.String(),
		Result:        "succeeded",
		RequestID:     &request,
		SchemaVersion: 1,
		Before:        json.RawMessage(`{}`),
		After:         after,
	})
}

func insertOrSelectMemberBinding(ctx context.Context, tx pgx.Tx, i Identity, rawIssuer string, request uuid.UUID) (uuid.UUID, bool, error) {
	if _, err := tx.Exec(ctx, `SAVEPOINT auth_register_member`); err != nil {
		return uuid.Nil, false, err
	}
	memberID := uuid.Must(uuid.NewV7())
	binding := uuid.Must(uuid.NewV7())
	_, err := tx.Exec(ctx, `INSERT INTO members(id,display_name,status) VALUES($1,$2,'pending_payment')`, memberID, i.Label)
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,$3,$4,$5,'sub','oidc',clock_timestamp())`, binding, memberID, i.Issuer, i.ClientScope, i.Subject)
	}
	if isUniqueViolation(err) {
		if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT auth_register_member`); rbErr != nil {
			return uuid.Nil, false, rbErr
		}
		if _, rlErr := tx.Exec(ctx, `RELEASE SAVEPOINT auth_register_member`); rlErr != nil {
			return uuid.Nil, false, rlErr
		}
		rechecked, selErr := lookupMemberBindings(ctx, tx, i.Issuer, rawIssuer, i.Subject)
		if selErr != nil {
			return uuid.Nil, false, selErr
		}
		if len(rechecked) != 1 {
			return uuid.Nil, false, apierr.New(apierr.Forbidden, "Identity binding is ambiguous.")
		}
		if rechecked[0].closed {
			return uuid.Nil, false, apierr.New(apierr.Forbidden, "Account is closed.")
		}
		if rechecked[0].clientScope != i.ClientScope {
			if err = updateMemberBindingScope(ctx, tx, rechecked[0].memberID, rechecked[0].bindingID, i.Role, i.ClientScope, request); err != nil {
				return uuid.Nil, false, err
			}
		}
		return rechecked[0].memberID, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE members SET canonical_identity_ref=$2,resource_version=resource_version+1 WHERE id=$1`, memberID, binding); err != nil {
		return uuid.Nil, false, err
	}
	if err = db.AppendAudit(ctx, tx, db.AuditEvent{Actor: memberID.String(), ActorType: string(i.Role), Action: "member.register", Resource: "member:" + memberID.String(), Result: "succeeded", RequestID: &request, SchemaVersion: 1, Before: json.RawMessage(`{}`), After: json.RawMessage(`{"status":"pending_payment"}`)}); err != nil {
		return uuid.Nil, false, err
	}
	if _, err = tx.Exec(ctx, `RELEASE SAVEPOINT auth_register_member`); err != nil {
		return uuid.Nil, false, err
	}
	return memberID, true, nil
}

func (s PostgresStore) Login(ctx context.Context, i Identity, token, old string, absolute time.Duration, request uuid.UUID) (p Principal, err error) {
	if strings.TrimSpace(i.Subject) == "" || strings.TrimSpace(i.Issuer) == "" || strings.TrimSpace(i.ClientScope) == "" || token == "" || (i.Role != Admin && i.Role != Member) {
		return p, errors.New("invalid login identity")
	}
	rawIssuer := i.Issuer
	i.Issuer = canonicalIssuer(i.Issuer)
	if request == uuid.Nil {
		request = uuid.Must(uuid.NewV7())
	}
	if i.Role == Admin {
		return s.loginAdmin(ctx, i, rawIssuer, token, old, absolute, request)
	}
	p = Principal{Role: i.Role, DisplayLabel: i.Label, MFAVerified: i.MFA, AuthenticatedAt: i.AuthTime}
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		// Serialize first login by canonical (issuer, subject), never by client_scope or email (§9.1, D-052 F-04).
		scoped, _ := json.Marshal([]string{i.Issuer, i.Subject})
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, string(scoped)); e != nil {
			return e
		}
		found, e := lookupMemberBindings(ctx, tx, i.Issuer, rawIssuer, i.Subject)
		if e != nil {
			return e
		}
		if len(found) > 1 {
			return apierr.New(apierr.Forbidden, "Identity binding is ambiguous.")
		}
		if len(found) == 1 {
			if found[0].closed {
				return apierr.New(apierr.Forbidden, "Account is closed.")
			}
			p.ID = found[0].memberID
			if found[0].clientScope != i.ClientScope {
				if e = updateMemberBindingScope(ctx, tx, p.ID, found[0].bindingID, i.Role, i.ClientScope, request); e != nil {
					return e
				}
			}
		} else {
			p.ID, _, e = insertOrSelectMemberBinding(ctx, tx, i, rawIssuer, request)
			if e != nil {
				return e
			}
		}
		if old != "" {
			var oldMemberID uuid.UUID
			e = tx.QueryRow(ctx, `DELETE FROM browser_sessions WHERE token_hash=$1 AND role='member' RETURNING member_id`, digest(old)).Scan(&oldMemberID)
			if e != nil && !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
			if e == nil && oldMemberID != uuid.Nil && oldMemberID != p.ID {
				if e = appendSessionAuditWithAfter(ctx, tx, Principal{ID: oldMemberID, Role: Member}, "session.logout", request, []byte(`{"reason":"superseded"}`)); e != nil {
					return e
				}
			}
		}
		expires := time.Now().UTC().Add(absolute)
		if i.Expires.Before(expires) {
			expires = i.Expires
		}
		_, e = tx.Exec(ctx, `INSERT INTO browser_sessions(token_hash,member_id,role,display_label,mfa_verified,authenticated_at,absolute_expires_at,id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, digest(token), p.ID, p.Role, p.DisplayLabel, p.MFAVerified, p.AuthenticatedAt, expires, uuid.Must(uuid.NewV7()))
		if e != nil {
			return e
		}
		return appendSessionAudit(ctx, tx, p, "session.login", request)
	})
	return
}
func (s PostgresStore) Session(ctx context.Context, token string, idle time.Duration) (p Principal, err error) {
	err = s.Pool.QueryRow(ctx, `UPDATE browser_sessions s SET last_seen_at=clock_timestamp() FROM members m WHERE token_hash=$1 AND s.role='member' AND s.member_id=m.id AND m.status<>'closed' AND absolute_expires_at>clock_timestamp() AND last_seen_at>clock_timestamp()-$2::bigint*interval '1 millisecond' RETURNING s.member_id,s.role,s.display_label,s.mfa_verified,s.authenticated_at`, digest(token), idle.Milliseconds()).Scan(&p.ID, &p.Role, &p.DisplayLabel, &p.MFAVerified, &p.AuthenticatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		p.Role = Admin
		err = s.Pool.QueryRow(ctx, `UPDATE admin_sessions SET last_seen_at=clock_timestamp() WHERE token_hash=$1 AND absolute_expires_at>clock_timestamp() AND last_seen_at>clock_timestamp()-$2::bigint*interval '1 millisecond' RETURNING admin_id,display_label,mfa_verified,authenticated_at`, digest(token), idle.Milliseconds()).Scan(&p.ID, &p.DisplayLabel, &p.MFAVerified, &p.AuthenticatedAt)
	}
	return
}
func (s PostgresStore) Logout(ctx context.Context, token string) error {
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var p Principal
		e := tx.QueryRow(ctx, `DELETE FROM browser_sessions WHERE token_hash=$1 AND role='member' RETURNING member_id,role`, digest(token)).Scan(&p.ID, &p.Role)
		if errors.Is(e, pgx.ErrNoRows) {
			p.Role = Admin
			e = tx.QueryRow(ctx, `DELETE FROM admin_sessions WHERE token_hash=$1 RETURNING admin_id`, digest(token)).Scan(&p.ID)
		}
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		rid, _ := apierr.RequestIDFromContext(ctx)
		request, _ := uuid.Parse(rid)
		return appendSessionAudit(ctx, tx, p, "session.logout", request)
	})
}

// Cleanup expires only authentication scratch, never member or audit history.
func (s PostgresStore) Cleanup(ctx context.Context, idle time.Duration) error {
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `DELETE FROM oidc_login_flows WHERE expires_at<=clock_timestamp()`); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `DELETE FROM admin_sessions WHERE absolute_expires_at<=clock_timestamp() OR last_seen_at<=clock_timestamp()-$1::bigint*interval '1 millisecond'`, idle.Milliseconds()); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `DELETE FROM browser_sessions WHERE absolute_expires_at<=clock_timestamp() OR last_seen_at<=clock_timestamp()-$1::bigint*interval '1 millisecond'`, idle.Milliseconds())
		return e
	})
}

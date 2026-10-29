// Package enrollment implements §6 enrollment and authenticated bootstrap
// delivery. HTTP adapters enforce admin authorization, render apierr.Write and
// set Cache-Control: no-store on all secret-bearing responses.
package enrollment

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const TokenPrefix = "fvpn_enr_"
const DefaultInstallerValidity = 15 * time.Minute
const DefaultPreinstalledValidity = 7 * 24 * time.Hour

// TEST ONLY. Production bootstrap-delivery validity is a required setting.
const TestBootstrapValidity = 10 * time.Minute

type Config struct {
	InstallerValidity    time.Duration
	PreinstalledValidity time.Duration
	BootstrapValidity    time.Duration
	Logger               *slog.Logger
}
type Service struct {
	pool    *pgxpool.Pool
	ca      *pki.CA
	auth    *pki.Service
	secrets signing.Provider
	fence   pki.Revocations
	cfg     Config
}

func NewService(pool *pgxpool.Pool, ca *pki.CA, auth *pki.Service, secrets signing.Provider, fence pki.Revocations, cfg Config) (*Service, error) {
	if cfg.InstallerValidity == 0 {
		cfg.InstallerValidity = DefaultInstallerValidity
	}
	if cfg.PreinstalledValidity == 0 {
		cfg.PreinstalledValidity = DefaultPreinstalledValidity
	}
	if pool == nil || ca == nil || auth == nil || secrets == nil || fence == nil || cfg.InstallerValidity < time.Minute || cfg.PreinstalledValidity < time.Minute || cfg.InstallerValidity > 30*24*time.Hour || cfg.PreinstalledValidity > 30*24*time.Hour || cfg.BootstrapValidity <= 0 {
		return nil, errors.New("enrollment: dependencies and explicit bootstrap validity required")
	}
	return &Service{pool, ca, auth, secrets, fence, cfg}, nil
}

// Command identifies the already-authorized administrator and UUID idempotency
// key. The controller supplies Principal as admin:<uuid> from the session,
// never the body. Standalone callers may also supply the bare administrator UUID.
type Command struct {
	Principal string
	Key       uuid.UUID
}

func begin(ctx context.Context, tx pgx.Tx, cmd Command, route string, body any) (db.IdempotencyRecord, error) {
	// F-15: reject invalid audit actors before any irreversible journal write.
	principal, err := adminPrincipal(cmd.Principal)
	if err != nil || cmd.Key == uuid.Nil {
		return db.IdempotencyRecord{}, apierr.New(apierr.ValidationFailed, "Principal and Idempotency-Key required.")
	}
	raw, err := canonjson.Marshal(body)
	if err != nil {
		return db.IdempotencyRecord{}, apierr.New(apierr.ValidationFailed, "Invalid command.")
	}
	hash := sha256.Sum256(raw)
	key := sha256.Sum256([]byte(cmd.Key.String()))
	return commandtx.BeginIdempotent(ctx, tx, db.IdempotencyScope{Principal: principal.Scope(), Method: "POST", Route: route, KeyHash: key[:], RequestHash: hash[:]}, 0)
}

func adminPrincipal(scope string) (auth.Principal, error) {
	raw := strings.TrimPrefix(scope, "admin:")
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil || id.String() != raw {
		return auth.Principal{}, apierr.New(apierr.ValidationFailed, "Administrator UUID required.")
	}
	return auth.Principal{Role: auth.Admin, ID: id}, nil
}

func validIdentifier(value string) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) != "" && !strings.ContainsRune(value, 0)
}

// Standalone service calls need the same correlation ID for their operation and
// audit rows. HTTP callers retain the framework's trusted UUID (D-029).
func requestContext(ctx context.Context) context.Context {
	return apierr.ContextWithRequestID(ctx, commandtx.RequestID(ctx).String())
}

func complete(ctx context.Context, tx pgx.Tx, record db.IdempotencyRecord, kind api.ResourceType, id uuid.UUID, status int, operation *uuid.UUID) error {
	return commandtx.CompleteIdempotent(ctx, tx, record.ID, db.IdempotencyResult{ResourceType: string(kind), ResourceID: &id, ResponseStatus: status, OperationID: operation})
}
func safe(err error) error {
	if err == nil {
		return nil
	}
	var ae *apierr.Error
	if errors.As(err, &ae) {
		return ae
	}
	if errors.Is(err, db.ErrIdempotencyConflict) {
		return apierr.New(apierr.IdempotencyConflict, "Idempotency-Key payload changed.")
	}
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, db.ErrNotFound) {
		return apierr.New(apierr.ResourceNotFound, "Resource not found.")
	}
	if errors.Is(err, db.ErrUniqueViolation) {
		return apierr.New(apierr.StateConflict, "Identity or external binding already exists.")
	}
	return apierr.New(apierr.DependencyUnavailable, "Enrollment operation unavailable.")
}
func appendDomainAudit(ctx context.Context, tx pgx.Tx, actor auth.Principal, action string, resource api.ResourceRef, op *uuid.UUID, reason string) error {
	actorID, actorType := actor.ID, api.AuditEventActorTypeAdmin
	if actor.Role == auth.Agent {
		actorID, actorType = actor.NodeID, api.AuditEventActorTypeNode
	}
	request := commandtx.RequestID(ctx)
	event, err := audit.NewEvent(audit.Entry{
		Actor: audit.Actor{Type: actorType, ID: actorID}, Action: action,
		Resource: resource, Result: api.AuditEventResultSucceeded,
		OperationID: op, RequestID: &request,
		After: map[string]any{"reason": reason}, Fields: []string{"reason"},
	})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
}
func (s *Service) log(ctx context.Context, event string, id uuid.UUID) {
	if s.cfg.Logger != nil {
		s.cfg.Logger.InfoContext(ctx, event, "resource_id", id.String())
	}
}
func validateCreate(r api.EnrollmentCreateRequest) error {
	if !r.Delivery.Valid() || (r.ExpiresInSeconds != nil && (*r.ExpiresInSeconds < 60 || *r.ExpiresInSeconds > 2592000)) {
		return apierr.New(apierr.ValidationFailed, "Invalid enrollment settings.")
	}
	for _, field := range []struct {
		value    *string
		min, max int
	}{{r.Name, 1, 100}, {r.Region, 1, 64}, {r.Notes, 0, 2000}} {
		if field.value != nil && (!utf8.ValidString(*field.value) || strings.ContainsRune(*field.value, 0) || utf8.RuneCountInString(*field.value) < field.min || utf8.RuneCountInString(*field.value) > field.max) {
			return apierr.New(apierr.ValidationFailed, "Invalid enrollment metadata.")
		}
	}
	if r.MemberGroupIds != nil {
		if len(*r.MemberGroupIds) > 100 {
			return apierr.New(apierr.ValidationFailed, "Too many groups.")
		}
		seen := map[uuid.UUID]bool{}
		for _, id := range *r.MemberGroupIds {
			if id == uuid.Nil || seen[id] {
				return apierr.New(apierr.ValidationFailed, "Invalid or duplicate group.")
			}
			seen[id] = true
		}
	}
	return nil
}

func (s *Service) CreateEnrollment(ctx context.Context, cmd Command, r api.EnrollmentCreateRequest) (out api.EnrollmentCreated, err error) {
	ctx = requestContext(ctx)
	if err = validateCreate(r); err != nil {
		return out, err
	}
	ttl := s.cfg.InstallerValidity
	if r.Delivery == api.EnrollmentDeliveryPreinstalled {
		ttl = s.cfg.PreinstalledValidity
	}
	if r.ExpiresInSeconds != nil {
		ttl = time.Duration(*r.ExpiresInSeconds) * time.Second
	}
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		out = api.EnrollmentCreated{Token: presence.NewNullNullable[string]()}
		record, e := begin(ctx, tx, cmd, "/api/v1/admin/enrollments", r)
		if e != nil {
			return e
		}
		if !record.New {
			if record.ResourceID == nil {
				return errors.New("missing idempotent resource")
			}
			out.Enrollment, e = get(ctx, tx, *record.ResourceID)
			return e
		}
		id, e := uuid.NewV7()
		if e != nil {
			return e
		}
		node, e := uuid.NewV7()
		if e != nil {
			return e
		}
		exit, e := uuid.NewV7()
		if e != nil {
			return e
		}
		groups := []uuid.UUID{}
		if r.MemberGroupIds != nil {
			groups = append(groups, (*r.MemberGroupIds)...)
		}
		var count int
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM member_groups WHERE id=ANY($1::uuid[])`, groups).Scan(&count); e != nil {
			return e
		}
		if count != len(groups) {
			return apierr.New(apierr.ValidationFailed, "Unknown member group.")
		}
		secret := make([]byte, 32)
		if _, e = rand.Read(secret); e != nil {
			return e
		}
		token := TokenPrefix + base64.RawURLEncoding.EncodeToString(secret)
		hash := sha256.Sum256([]byte(token))
		if r.NodeId != nil {
			node = *r.NodeId
			var lifecycle string
			var active bool
			// §6.1–2 / AT07: reserve an authorized reinstall without resetting
			// lifecycle or clearing any old identity's revocation evidence.
			if e = tx.QueryRow(ctx, `SELECT lifecycle_state,EXISTS(SELECT 1 FROM node_incarnations i WHERE i.node_id=n.id AND i.revoked_at IS NULL) FROM nodes n WHERE n.id=$1 FOR UPDATE`, node).Scan(&lifecycle, &active); errors.Is(e, pgx.ErrNoRows) {
				return apierr.New(apierr.ResourceNotFound, "Node not found.")
			} else if e != nil {
				return e
			}
			if lifecycle == "retired" {
				return apierr.New(apierr.NodeRetired, "Node is retired.")
			}
			if active {
				return apierr.New(apierr.NodeIncarnationActive, "Revoke the current incarnation before reinstalling.")
			}
		} else {
			if _, e = tx.Exec(ctx, `INSERT INTO nodes(id,notes) VALUES($1,$2)`, node, value(r.Notes)); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO exits(id,node_id,display_name,region) VALUES($1,$2,$3,$4)`, exit, node, value(r.Name), value(r.Region)); e != nil {
				return e
			}
			for _, group := range groups {
				groupID, e := uuid.NewV7()
				if e != nil {
					return e
				}
				if _, e = tx.Exec(ctx, `INSERT INTO exit_groups(id,exit_id,group_id) VALUES($1,$2,$3)`, groupID, exit, group); e != nil {
					return e
				}
			}
		}
		if _, e = tx.Exec(ctx, `INSERT INTO node_enrollments(id,node_id,token_hash,expiry,delivery,name,region,notes,member_group_ids) VALUES($1,$2,$3,clock_timestamp()+$4::bigint*interval '1 millisecond',$5,$6,$7,$8,$9)`, id, node, hash[:], ttl.Milliseconds(), r.Delivery, r.Name, r.Region, r.Notes, groups); e != nil {
			return e
		}
		if e = complete(ctx, tx, record, api.ResourceTypeEnrollment, id, 201, nil); e != nil {
			return e
		}
		actor, e := adminPrincipal(cmd.Principal)
		if e != nil {
			return e
		}
		if e = appendDomainAudit(ctx, tx, actor, "enrollment.created", api.ResourceRef{Type: api.ResourceTypeEnrollment, Id: id}, nil, ""); e != nil {
			return e
		}
		out.Enrollment, e = get(ctx, tx, id)
		out.Token.Set(token)
		out.SecretAvailable = true
		return e
	})
	if err != nil {
		return api.EnrollmentCreated{}, safe(err)
	}
	s.log(ctx, "enrollment created or replayed", out.Enrollment.EnrollmentId)
	return out, nil
}
func value(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func get(ctx context.Context, q querier, id uuid.UUID) (out api.Enrollment, err error) {
	var fp []byte
	var name, region, notes *string
	var requestID, incarnationID *uuid.UUID
	var consumedAt, revokedAt *time.Time
	err = q.QueryRow(ctx, `SELECT id,node_id,delivery,CASE WHEN state='pending' AND expiry<=clock_timestamp() THEN 'expired' ELSE state END,expiry,name,region,notes,member_group_ids,request_id,incarnation_id,public_key_fingerprint,created_at,consumed_at,revoked_at FROM node_enrollments WHERE id=$1`, id).Scan(&out.EnrollmentId, &out.NodeId, &out.Delivery, &out.State, &out.ExpiresAt, &name, &region, &notes, &out.MemberGroupIds, &requestID, &incarnationID, &fp, &out.CreatedAt, &consumedAt, &revokedAt)
	out.Name = presence.FromPointer(name)
	out.Region = presence.FromPointer(region)
	out.Notes = presence.FromPointer(notes)
	out.EnrollmentRequestId = presence.FromPointer(requestID)
	out.NodeIncarnationId = presence.FromPointer(incarnationID)
	out.ConsumedAt = presence.FromPointer(consumedAt)
	out.RevokedAt = presence.FromPointer(revokedAt)
	out.BoundPublicKeySha256.SetNull()
	if len(fp) > 0 {
		v := hex.EncodeToString(fp)
		out.BoundPublicKeySha256.Set(v)
	}
	out.ExpiresAt = out.ExpiresAt.UTC()
	out.CreatedAt = out.CreatedAt.UTC()
	return
}
func (s *Service) GetEnrollment(ctx context.Context, id uuid.UUID) (api.Enrollment, error) {
	out, err := get(ctx, s.pool, id)
	return out, safe(err)
}
func (s *Service) RevokeEnrollment(ctx context.Context, cmd Command, id uuid.UUID, reason string) (out api.Enrollment, err error) {
	ctx = requestContext(ctx)
	if !validReason(reason) {
		return out, apierr.New(apierr.ValidationFailed, "Revocation reason required.")
	}
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		record, e := begin(ctx, tx, cmd, "/api/v1/admin/enrollments/"+id.String()+"/revoke", map[string]string{"reason": reason})
		if e != nil {
			return e
		}
		if !record.New {
			out, e = get(ctx, tx, id)
			return e
		}
		var state string
		if e = tx.QueryRow(ctx, `SELECT state FROM node_enrollments WHERE id=$1 FOR UPDATE`, id).Scan(&state); e != nil {
			return e
		}
		if state != "pending" {
			return apierr.New(apierr.StateConflict, "Only an unused enrollment can be revoked; revoke the node after consumption.")
		}
		if _, e = tx.Exec(ctx, `UPDATE node_enrollments SET state='revoked',revoked_at=clock_timestamp() WHERE id=$1`, id); e != nil {
			return e
		}
		if e = complete(ctx, tx, record, api.ResourceTypeEnrollment, id, 200, nil); e != nil {
			return e
		}
		actor, e := adminPrincipal(cmd.Principal)
		if e != nil {
			return e
		}
		if e = appendDomainAudit(ctx, tx, actor, "enrollment.revoked", api.ResourceRef{Type: api.ResourceTypeEnrollment, Id: id}, nil, reason); e != nil {
			return e
		}
		out, e = get(ctx, tx, id)
		return e
	})
	if err != nil {
		return api.Enrollment{}, safe(err)
	}
	return out, nil
}
func validReason(reason string) bool {
	return utf8.ValidString(reason) && !strings.ContainsRune(reason, 0) && strings.TrimSpace(reason) != "" && utf8.RuneCountInString(reason) <= 500
}

type EnrollResult struct {
	Response    api.EnrollResponse
	Replay      bool
	OperationID uuid.UUID
}

func tokenHash(token string) ([]byte, error) {
	if !strings.HasPrefix(token, TokenPrefix) {
		return nil, apierr.New(apierr.EnrollmentTokenInvalid, "Invalid enrollment token.")
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(strings.TrimPrefix(token, TokenPrefix))
	if err != nil || len(raw) != 32 || len(token) != len(TokenPrefix)+43 {
		return nil, apierr.New(apierr.EnrollmentTokenInvalid, "Invalid enrollment token.")
	}
	h := sha256.Sum256([]byte(token))
	return h[:], nil
}

func (s *Service) Enroll(ctx context.Context, token string, r api.EnrollRequest) (out EnrollResult, err error) {
	ctx = requestContext(ctx)
	hash, err := tokenHash(token)
	if err != nil {
		return out, err
	}
	csr, requestHash, err := VerifyProof(r)
	if err != nil {
		return out, err
	}
	fp := pki.Fingerprint(csr)
	// Read the reserved ID to prepare a certificate without external calls in a
	// transaction. All authorization and bindings are rechecked under row locks.
	var nodeID uuid.UUID
	var initialState string
	var initialExpired bool
	var initialFingerprint []byte
	var initialRequest, initialIncarnation *uuid.UUID
	if err = s.pool.QueryRow(ctx, `SELECT node_id,state,expiry<=clock_timestamp(),public_key_fingerprint,request_id,incarnation_id FROM node_enrollments WHERE token_hash=$1`, hash).Scan(&nodeID, &initialState, &initialExpired, &initialFingerprint, &initialRequest, &initialIncarnation); errors.Is(err, pgx.ErrNoRows) {
		return out, apierr.New(apierr.EnrollmentTokenInvalid, "Invalid enrollment token.")
	} else if err != nil {
		return out, safe(err)
	}
	if initialState == "revoked" {
		return out, apierr.New(apierr.EnrollmentTokenRevoked, "Enrollment revoked.")
	}
	if initialExpired || initialState == "expired" {
		return out, apierr.New(apierr.EnrollmentTokenExpired, "Enrollment expired.")
	}
	if initialState == "consumed" && (!bytes.Equal(initialFingerprint, fp) || initialRequest == nil || *initialRequest != r.EnrollmentRequestId || initialIncarnation == nil || *initialIncarnation != r.NodeIncarnationId) {
		return out, apierr.New(apierr.EnrollmentKeyMismatch, "Enrollment binding does not match.")
	}
	id := pki.NodeIdentity{NodeID: nodeID, IncarnationID: r.NodeIncarnationId}
	if err = s.auth.CheckFence(ctx, id); err != nil {
		return out, err
	}
	// Public recovery of a committed response does not require an available
	// signing key. A first-use race may prepare an unused certificate, but only
	// the row-lock winner persists and returns one.
	var issued pki.Issued
	if initialState == "pending" {
		issued, err = s.ca.Issue(ctx, id, r.CsrPem)
		if err != nil {
			return out, safe(err)
		}
	}
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		out = EnrollResult{}
		// All identity mutations lock node first, then enrollment/certificate.
		var lifecycle string
		if e := tx.QueryRow(ctx, `SELECT lifecycle_state FROM nodes WHERE id=$1 FOR UPDATE`, nodeID).Scan(&lifecycle); e != nil {
			return e
		}
		if e := s.auth.CheckFence(ctx, id); e != nil {
			return e
		}
		var enrollmentID uuid.UUID
		var state string
		var expired bool
		var boundFP []byte
		var requestID, incarnation, certID, opID *uuid.UUID
		e := tx.QueryRow(ctx, `SELECT id,state,expiry<=clock_timestamp(),public_key_fingerprint,request_id,incarnation_id,certificate_id,operation_id FROM node_enrollments WHERE token_hash=$1 FOR UPDATE`, hash).Scan(&enrollmentID, &state, &expired, &boundFP, &requestID, &incarnation, &certID, &opID)
		if e != nil {
			return e
		}
		if state == "revoked" {
			return apierr.New(apierr.EnrollmentTokenRevoked, "Enrollment revoked.")
		}
		if expired || state == "expired" {
			return apierr.New(apierr.EnrollmentTokenExpired, "Enrollment expired.")
		}
		if lifecycle == "retired" || lifecycle == "revoked" && state != "pending" {
			return apierr.New(apierr.IdentityRevoked, "Node identity revoked.")
		}
		var cert *x509.Certificate
		if state == "consumed" {
			if !bytes.Equal(boundFP, fp) || requestID == nil || *requestID != r.EnrollmentRequestId || incarnation == nil || *incarnation != r.NodeIncarnationId {
				return apierr.New(apierr.EnrollmentKeyMismatch, "Enrollment binding does not match.")
			}
			if certID == nil || opID == nil {
				return errors.New("missing enrollment result")
			}
			var der []byte
			e = tx.QueryRow(ctx, `SELECT c.public_certificate FROM node_certificates c JOIN node_incarnations i ON i.id=c.incarnation_id JOIN nodes n ON n.id=i.node_id WHERE c.id=$1 AND c.revoked_at IS NULL AND i.revoked_at IS NULL AND n.current_incarnation_id=i.id AND least(c.not_after,coalesce(c.usable_until,c.not_after))>clock_timestamp()`, *certID).Scan(&der)
			if errors.Is(e, pgx.ErrNoRows) {
				return apierr.New(apierr.IdentityRevoked, "Enrollment certificate no longer active.")
			}
			if e != nil {
				return e
			}
			cert, e = x509.ParseCertificate(der)
			if e != nil {
				return e
			}
			out.Replay = true
			out.OperationID = *opID
		} else if state == "pending" {
			// Recheck under the node lock: another authorized token may have
			// enrolled an incarnation since this token was issued (§6.2, AT07).
			var active bool
			if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM node_incarnations WHERE node_id=$1 AND revoked_at IS NULL)`, nodeID).Scan(&active); e != nil {
				return e
			}
			if active {
				return apierr.New(apierr.NodeIncarnationActive, "Revoke the current incarnation before reinstalling.")
			}
			if _, e = tx.Exec(ctx, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch) VALUES($1,$2,$3,$4,$5)`, r.NodeIncarnationId, nodeID, fp, r.Platform.Os, r.Platform.Arch); e != nil {
				return e
			}

			storedCert, e := pki.StoreCertificate(ctx, tx, r.NodeIncarnationId, issued)
			if e != nil {
				return e
			}
			cert = issued.Certificate
			actor := auth.Principal{Role: auth.Agent, NodeID: nodeID, IncarnationID: r.NodeIncarnationId}
			out.OperationID, e = operations.Create(ctx, tx, operations.CreateRequest{Kind: api.OperationKindNodeEnroll, Principal: actor, Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: nodeID}, RequestID: commandtx.RequestID(ctx), Steps: map[string]operations.Step{
				"netbird_setup": {Required: true}, "peer_readback": {Required: true}, "node_policy": {Required: true},
			}})
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE node_enrollments SET state='consumed',request_id=$2,public_key_fingerprint=$3,consumed_at=clock_timestamp(),incarnation_id=$4,request_hash=$5,certificate_id=$6,operation_id=$7 WHERE id=$1`, enrollmentID, r.EnrollmentRequestId, fp, r.NodeIncarnationId, requestHash, storedCert, out.OperationID); e != nil {
				return e
			}
			// 00111's pki_guard_terminal requires the consumed enrollment and
			// fresh certificate/incarnation above before allowing AT07 reentry.
			if _, e = tx.Exec(ctx, `UPDATE nodes SET current_incarnation_id=$2,lifecycle_state='registered',provisioning_state='applying',resource_version=resource_version+1 WHERE id=$1`, nodeID, r.NodeIncarnationId); e != nil {
				return e
			}
			payload, e := canonjson.Marshal(ProvisionEvent{1, nodeID, r.NodeIncarnationId, out.OperationID, true, 1, false, 86400, "staging"})
			if e != nil {
				return e
			}
			if e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: r.NodeIncarnationId, AggregateType: "node-incarnation", AggregateVersion: "1", EventType: "node.provision.requested", SchemaVersion: 1, Payload: payload}); e != nil {
				return e
			}
			if e = appendDomainAudit(ctx, tx, actor, "enrollment.consumed", api.ResourceRef{Type: api.ResourceTypeEnrollment, Id: enrollmentID}, &out.OperationID, ""); e != nil {
				return e
			}
		} else {
			return apierr.New(apierr.EnrollmentTokenInvalid, "Invalid enrollment state.")
		}
		var publication api.PublicationIntent
		if e = tx.QueryRow(ctx, `SELECT publication_intent FROM exits WHERE node_id=$1`, nodeID).Scan(&publication); e != nil {
			return e
		}
		out.Response = api.EnrollResponse{NodeId: nodeID, NodeIncarnationId: r.NodeIncarnationId, LifecycleState: api.LifecycleStateRegistered, PublicationIntent: publication, CertificatePem: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})), CaChainPem: s.ca.ChainPEM(), CertificateSerial: cert.SerialNumber.Text(16), CertificateNotAfter: cert.NotAfter.UTC(), Next: "/api/v1/agent/bootstrap"}
		return nil
	})
	if err != nil {
		return EnrollResult{}, safe(err)
	}
	s.log(ctx, "node enrollment accepted", nodeID)
	return out, nil
}

// ProvisionEvent is the versioned outbox contract. The worker must read back
// its deterministic OperationID before creating anything externally (§6.2).
// NetBird key expiry is separate from the shorter secret-delivery deadline.
type ProvisionEvent struct {
	SchemaVersion    int       `json:"schema_version"`
	NodeID           uuid.UUID `json:"node_id"`
	IncarnationID    uuid.UUID `json:"incarnation_id"`
	OperationID      uuid.UUID `json:"operation_id"`
	OneOff           bool      `json:"one_off"`
	UsageLimit       int       `json:"usage_limit"`
	Ephemeral        bool      `json:"ephemeral"`
	ExpiresInSeconds int       `json:"expires_in_seconds"`
	GroupRole        string    `json:"group_role"`
}

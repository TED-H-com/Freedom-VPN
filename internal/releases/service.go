package releases

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const FleetID = "00000000-0000-0000-0000-000000000001"

type Service struct {
	Pool            *pgxpool.Pool
	Validator       *Validator
	VerifyArtifacts bool
}
type Registration struct {
	ReleaseID   uuid.UUID
	OperationID *uuid.UUID
	Status      int
}

// Command is populated by the authenticated admin adapter, never by request JSON.
type Command struct{ ActorID, IdempotencyKey uuid.UUID }

func Begin(ctx context.Context, tx pgx.Tx, c Command, route string, body []byte) (db.IdempotencyRecord, error) {
	if c.ActorID == uuid.Nil || c.IdempotencyKey == uuid.Nil {
		return db.IdempotencyRecord{}, apierr.New(apierr.MalformedRequest, "Authenticated actor and Idempotency-Key are required")
	}
	h, e := canonjson.Hash(body)
	if e != nil {
		return db.IdempotencyRecord{}, e
	}
	key := sha256.Sum256([]byte(c.IdempotencyKey.String()))
	r, e := db.BeginIdempotent(ctx, tx, db.IdempotencyScope{Principal: c.ActorID.String(), Method: "POST", Route: route, KeyHash: key[:], RequestHash: h[:]}, 0)
	if errors.Is(e, db.ErrIdempotencyConflict) {
		e = apierr.New(apierr.IdempotencyConflict, "Idempotency-Key has different content")
	}
	return r, e
}

// Register takes raw JSON so signatures and request hashes are never reconstructed
// from a lossy decode. Rejected registrations retain only IDs/hashes/error codes.
func (s *Service) Register(ctx context.Context, c Command, raw []byte) (Registration, error) {
	var req api.ReleaseRegisterRequest
	if e := json.Unmarshal(raw, &req); e != nil {
		return Registration{}, apierr.New(apierr.ValidationFailed, "Invalid registration")
	}
	var body struct {
		Manifest json.RawMessage `json:"manifest"`
	}
	_ = json.Unmarshal(raw, &body)
	m, hash, e := s.Validator.ValidateManifest(body.Manifest, time.Now().UTC())
	if e == nil {
		e = s.Validator.ValidateBody("ReleaseRegisterRequest", raw)
	}
	if e == nil && !s.Validator.AllowURL(req.ManifestUrl) {
		e = apierr.New(apierr.ValidationFailed, "Manifest storage origin is not allowed")
	}
	if e != nil {
		if h, he := canonjson.Hash(body.Manifest); he == nil {
			var ae *apierr.Error
			code := apierr.ValidationFailed
			if errors.As(e, &ae) {
				code = ae.Code
			}
			_, save := s.Pool.Exec(ctx, `INSERT INTO release_registration_rejections(release_id,manifest_hash,error_code) VALUES($1,$2,$3)`, nullableID(req.Manifest.Signed.ReleaseId), h[:], string(code))
			if save != nil {
				return Registration{}, save
			}
		}
		return Registration{}, e
	}
	var out Registration
	e = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rec, e := Begin(ctx, tx, c, "/api/v1/admin/releases", raw)
		if e != nil {
			return e
		}
		if !rec.New {
			out = Registration{ReleaseID: *rec.ResourceID, OperationID: rec.OperationID, Status: *rec.ResponseStatus}
			return nil
		}
		// The fleet lock serializes policy changes and registration by content ID.
		if _, e = tx.Exec(ctx, `SELECT id FROM fleet_config WHERE id=$1 FOR UPDATE`, FleetID); e != nil {
			return e
		}
		id := m.Signed.ReleaseId
		var old []byte
		e = tx.QueryRow(ctx, `SELECT manifest_hash FROM releases WHERE id=$1`, id).Scan(&old)
		if e == nil {
			if hex.EncodeToString(old) != hash {
				return apierr.New(apierr.StateConflict, "Release ID already has different immutable content")
			}
			out = Registration{ReleaseID: id, Status: 201}
			var state string
			if e = tx.QueryRow(ctx, `SELECT state,operation_id FROM release_verifications WHERE release_id=$1`, id).Scan(&state, &out.OperationID); e != nil {
				return e
			}
			if state == "pending" {
				out.Status = 202
			}
		} else if errors.Is(e, pgx.ErrNoRows) {
			canonical, e := canonjson.Canonicalize(body.Manifest)
			if e != nil {
				return e
			}
			digest, _ := hex.DecodeString(hash)
			sig, _ := base64.StdEncoding.DecodeString(m.Signatures[0].SignatureB64)
			components, _ := json.Marshal(m.Signed.Components)
			_, e = tx.Exec(ctx, `INSERT INTO releases(id,version,release_sequence,manifest_schema_version,manifest,manifest_hash,signature,signing_key_ref,component_versions) VALUES($1,$2,$3::numeric,$4,$5,$6,$7,$8,$9)`, id, m.Signed.ProductVersion, m.Signed.ReleaseSequence.String(), m.Signed.ManifestSchemaVersion, canonical, digest, sig, m.Signatures[0].KeyId, components)
			if e != nil {
				return e
			}
			for _, a := range m.Signed.Artifacts {
				d, _ := hex.DecodeString(a.Sha256)
				for _, os := range a.OsVersions {
					_, e = tx.Exec(ctx, `INSERT INTO artifacts(release_id,platform,arch,digest,size_bytes,installed_size_bytes,storage_ref) VALUES($1,$2,$3,$4,$5::numeric,$6::numeric,$7)`, id, string(a.OsId)+"/"+os, a.Architecture, d, a.SizeBytes.String(), a.InstalledSizeBytes.String(), a.Url)
					if e != nil {
						return e
					}
				}
			}
			out = Registration{ReleaseID: id, Status: 201}
			state := "verified"
			if s.VerifyArtifacts {
				op := uuid.Must(uuid.NewV7())
				out.OperationID = &op
				out.Status = 202
				state = "pending"
				_, e = tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id) VALUES($1,'release.verify',$2,'release',$3)`, op, c.ActorID.String(), id)
				if e != nil {
					return e
				}
				if e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: id, AggregateType: "release", AggregateVersion: "1", EventType: "release.verify", SchemaVersion: 1, Payload: json.RawMessage(`{"schema_version":1}`)}); e != nil {
					return e
				}
			}
			_, e = tx.Exec(ctx, `INSERT INTO release_verifications(release_id,manifest_url,state,operation_id,verified_at) VALUES($1,$2,$3,$4,CASE WHEN $3='verified' THEN clock_timestamp() END)`, id, req.ManifestUrl, state, out.OperationID)
			if e != nil {
				return e
			}
			if e = Audit(ctx, tx, c.ActorID.String(), "admin", "release.register", id.String(), out.OperationID, map[string]any{"verification_state": state}); e != nil {
				return e
			}
		} else {
			return e
		}
		return db.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{ResourceType: "release", ResourceID: &out.ReleaseID, OperationID: out.OperationID, ResponseStatus: out.Status})
	})
	return out, e
}

func nullableID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func Audit(ctx context.Context, tx pgx.Tx, actor, actorType, action, resource string, op *uuid.UUID, after any) error {
	raw, e := json.Marshal(after)
	if e != nil {
		return e
	}
	return db.AppendAudit(ctx, tx, db.AuditEvent{Actor: actor, ActorType: actorType, Action: action, Resource: resource, OperationID: op, SchemaVersion: 1, Before: json.RawMessage(`{}`), After: raw, Result: "accepted"})
}

// Eligible is called inside the fleet transaction; signatures are rechecked so
// a trust-root revocation cannot be bypassed by old verification state (§13.4).
func (s *Service) Eligible(ctx context.Context, tx pgx.Tx, id uuid.UUID) (api.ReleaseManifestEnvelope, string, string, error) {
	var raw []byte
	var hash, url string
	var eligible bool
	e := tx.QueryRow(ctx, `SELECT r.manifest,encode(r.manifest_hash,'hex'),v.manifest_url,
 v.state='verified' AND f.supported_release_policy='allowlist' AND r.id=ANY(f.approved_release_ids) AND r.id=ANY(f.supported_release_ids) AND NOT r.id=ANY(f.revoked_release_ids)
 FROM releases r JOIN release_verifications v ON v.release_id=r.id CROSS JOIN fleet_config f WHERE r.id=$1`, id).Scan(&raw, &hash, &url, &eligible)
	if errors.Is(e, pgx.ErrNoRows) || e == nil && !eligible {
		return api.ReleaseManifestEnvelope{}, "", "", apierr.New(apierr.ReleaseNotApproved, "Release is not eligible for targeting")
	}
	if e != nil {
		return api.ReleaseManifestEnvelope{}, "", "", e
	}
	m, _, e := s.Validator.ValidateManifest(raw, time.Now().UTC())
	return m, hash, url, e
}

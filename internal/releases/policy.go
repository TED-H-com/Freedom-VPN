package releases

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"slices"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ApplySupportTx is the policy activation adapter. Call in the policy owner's
// transaction after locking fleet_config and validating If-Match/idempotency.
// Revocation is monotonic; a later policy cannot silently resurrect a release.
func (s *Service) ApplySupportTx(ctx context.Context, tx pgx.Tx, actor uuid.UUID, p api.ReleaseSupportPayload, reason string) error {
	raw, e := json.Marshal(p)
	if e != nil {
		return e
	}
	if e = s.Validator.ValidateBody("ReleaseSupportPayload", raw); e != nil {
		return e
	}
	if actor == uuid.Nil || reason == "" {
		return apierr.New(apierr.ValidationFailed, "Policy actor and reason are required")
	}
	serving := p.MinimumServingVersion.ServingReleaseIds
	for _, id := range p.ApprovedReleaseIds {
		if !slices.Contains(serving, id) {
			return apierr.New(apierr.ValidationFailed, "Approved releases must be serviceable")
		}
	}
	for _, id := range p.RevokedReleaseIds {
		if slices.Contains(serving, id) || slices.Contains(p.ApprovedReleaseIds, id) {
			return apierr.New(apierr.ValidationFailed, "Revoked releases cannot be approved or serve")
		}
	}
	var previous []uuid.UUID
	if e = tx.QueryRow(ctx, `SELECT revoked_release_ids FROM fleet_config WHERE id=$1 FOR UPDATE`, FleetID).Scan(&previous); e != nil {
		return e
	}
	for _, id := range previous {
		if !slices.Contains(p.RevokedReleaseIds, id) {
			return apierr.New(apierr.StateConflict, "Release revocation cannot be undone")
		}
	}
	for _, id := range append(append([]uuid.UUID{}, serving...), p.RevokedReleaseIds...) {
		var exists bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM releases WHERE id=$1)`, id).Scan(&exists); e != nil {
			return e
		}
		if !exists {
			return apierr.New(apierr.ReleaseNotApproved, "Policy references an unregistered release")
		}
	}
	_, e = tx.Exec(ctx, `UPDATE fleet_config SET supported_release_policy='allowlist',supported_release_ids=$1,approved_release_ids=$2,revoked_release_ids=$3,support_generation=support_generation+1,generation=generation+1,resource_version=resource_version+1`, serving, p.ApprovedReleaseIds, p.RevokedReleaseIds)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `UPDATE fleet_node_state s SET recovery_required=true FROM nodes n WHERE n.id=s.node_id AND n.last_good_release_id=ANY($1::uuid[])`, p.RevokedReleaseIds)
	if e != nil {
		return e
	}
	if e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: uuid.MustParse(FleetID), AggregateType: "fleet", AggregateVersion: "1", EventType: "fleet.release_support_changed", SchemaVersion: 1, Payload: json.RawMessage(`{"schema_version":1}`)}); e != nil {
		return e
	}
	return Audit(ctx, tx, actor.String(), "admin", "release_support.activate", FleetID, nil, map[string]any{"reason": reason, "policy": p})
}

// Cursor encodes a stable UUID keyset; callers must retain the same resource URL.
func Cursor(id uuid.UUID) string { return base64.RawURLEncoding.EncodeToString(id[:]) }
func ParseCursor(cursor string) (uuid.UUID, error) {
	if cursor == "" {
		return uuid.Nil, nil
	}
	b, e := base64.RawURLEncoding.DecodeString(cursor)
	if e != nil {
		return uuid.Nil, apierr.New(apierr.ValidationFailed, "Invalid cursor")
	}
	id, e := uuid.FromBytes(b)
	if e != nil {
		return uuid.Nil, apierr.New(apierr.ValidationFailed, "Invalid cursor")
	}
	return id, nil
}
func PageLimit(n int) (int, error) {
	if n == 0 {
		return 50, nil
	}
	if n < 1 || n > 100 {
		return 0, apierr.New(apierr.ValidationFailed, "Page limit must be between 1 and 100")
	}
	return n, nil
}

func (s *Service) List(ctx context.Context, cursor string, limit int) (api.ReleaseList, error) {
	return s.list(ctx, cursor, limit, nil)
}
func (s *Service) Get(ctx context.Context, id uuid.UUID) (api.Release, error) {
	out, e := s.list(ctx, "", 1, &id)
	if e != nil {
		return api.Release{}, e
	}
	if len(out.Items) != 1 {
		return api.Release{}, apierr.New(apierr.ResourceNotFound, "Release not found")
	}
	return out.Items[0], nil
}
func (s *Service) list(ctx context.Context, cursor string, limit int, only *uuid.UUID) (api.ReleaseList, error) {
	out := api.ReleaseList{Items: []api.Release{},
		NextCursor: presence.NewNullNullable[string]()}
	after, e := ParseCursor(cursor)
	if e != nil {
		return out, e
	}
	limit, e = PageLimit(limit)
	if e != nil {
		return out, e
	}
	e = commandtx.InTx(ctx, s.Pool, db.TxOptions{TxOptions: pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx) error {
		out.Items = []api.Release{}
		out.NextCursor.SetNull()
		var gen, rv string
		var defaultReleaseID *uuid.UUID
		var serving, approved, revoked []uuid.UUID
		e := tx.QueryRow(ctx, `SELECT desired_release_id,generation::text,resource_version::text,supported_release_ids,approved_release_ids,revoked_release_ids FROM fleet_config`).Scan(&defaultReleaseID, &gen, &rv, &serving, &approved, &revoked)
		if e != nil {
			return e
		}
		out.Fleet.Generation, e = decstr.Parse(gen)
		if e != nil {
			return e
		}
		out.Fleet.ResourceVersion, e = decstr.Parse(rv)
		if e != nil {
			return e
		}
		out.Fleet.ServingReleaseIds = serving
		out.Fleet.FleetDefaultReleaseId = presence.FromPointer(defaultReleaseID)
		out.Fleet.ReleaseSupportRevisionId.SetNull()
		rows, e := tx.Query(ctx, `SELECT r.id,r.manifest,encode(r.manifest_hash,'hex'),r.created_at,v.state,v.error_code FROM releases r JOIN release_verifications v ON v.release_id=r.id WHERE r.id>$1 AND ($3::uuid IS NULL OR r.id=$3) ORDER BY r.id LIMIT $2`, after, limit+1, only)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var r api.Release
			var verificationError *api.ErrorCode
			var raw []byte
			if e = rows.Scan(&r.ReleaseId, &raw, &r.ManifestSha256, &r.RegisteredAt, &r.VerificationState, &verificationError); e != nil {
				return e
			}
			r.VerificationError = presence.FromPointer(verificationError)
			if len(out.Items) == limit {
				c := Cursor(out.Items[len(out.Items)-1].ReleaseId)
				out.NextCursor.Set(c)
				break
			}
			var m api.ReleaseManifestEnvelope
			if e = json.Unmarshal(raw, &m); e != nil {
				return e
			}
			r.Artifacts = m.Signed.Artifacts
			r.Compatibility = m.Signed.Compatibility
			r.Components = m.Signed.Components
			r.ManifestCreatedAt = m.Signed.CreatedAt
			r.ProductVersion = m.Signed.ProductVersion
			r.ReleaseSequence = m.Signed.ReleaseSequence
			if r.VerificationState == "pending" {
				r.VerificationState = "verifying"
			}
			if r.VerificationState == "failed" {
				r.VerificationState = "rejected"
			}
			r.RegisteredAt = r.RegisteredAt.UTC()
			r.Architectures = []api.Architecture{}
			for _, a := range r.Artifacts {
				if !slices.Contains(r.Architectures, a.Architecture) {
					r.Architectures = append(r.Architectures, a.Architecture)
				}
			}
			r.SigningKeyIds = []string{}
			for _, sig := range m.Signatures {
				r.SigningKeyIds = append(r.SigningKeyIds, sig.KeyId)
			}
			r.IsFleetDefault = defaultReleaseID != nil && *defaultReleaseID == r.ReleaseId
			r.ServingAllowed = slices.Contains(serving, r.ReleaseId) && !slices.Contains(revoked, r.ReleaseId)
			r.SupportState = "not_approved"
			if slices.Contains(approved, r.ReleaseId) {
				r.SupportState = "approved"
			}
			if slices.Contains(revoked, r.ReleaseId) {
				r.SupportState = "revoked"
			}
			out.Items = append(out.Items, r)
		}
		return rows.Err()
	})
	return out, e
}

// Manifest reads immutable content; it does not grant installation authorization.
func (s *Service) Manifest(ctx context.Context, tx pgx.Tx, id uuid.UUID) (api.ReleaseManifestEnvelope, error) {
	var raw []byte
	var m api.ReleaseManifestEnvelope
	e := tx.QueryRow(ctx, `SELECT manifest FROM releases WHERE id=$1`, id).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return m, apierr.New(apierr.ReleaseNotApproved, "Release is not registered")
	}
	if e != nil {
		return m, e
	}
	e = json.Unmarshal(raw, &m)
	return m, e
}

package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ResolverDocument is immutable publisher output; Canonical must be served verbatim (D-056).
type ResolverDocument struct {
	Reference api.ResolverDocumentRef
	Canonical []byte
}

// ResolverDocumentSource reads publisher-owned documents. Each lookup returns
// one coherent reference and byte snapshot. Missing documents return pgx.ErrNoRows.
type ResolverDocumentSource interface {
	LatestPublished(context.Context, uuid.UUID) (ResolverDocument, error)
	DocumentByID(context.Context, uuid.UUID) (ResolverDocument, error)
}

// ValidateResolverDocument checks the signed metadata against the exact stored bytes.
func ValidateResolverDocument(d ResolverDocument) error {
	r := d.Reference
	if r.DocumentId == uuid.Nil || r.ProfileRevisionId == uuid.Nil || r.Generation == 0 || len(d.Canonical) == 0 || len(d.Canonical) > canonjson.MaxBytes || r.SizeBytes != len(d.Canonical) {
		return errors.New("invalid resolver document metadata")
	}
	canonical, err := canonjson.Canonicalize(d.Canonical)
	if err != nil || !bytes.Equal(canonical, d.Canonical) {
		return errors.New("invalid resolver document canonical bytes")
	}
	hash, err := canonjson.HashHex(d.Canonical)
	if err != nil || hash != r.ContentSha256 {
		return errors.New("invalid resolver document digest")
	}
	return nil
}

// RecordIssuedResolverDocumentTx retains the reference alongside the immutable
// desired-state history. It must run after compiler persistence in the same
// transaction; reference authorization never guesses from publication timestamps.
func RecordIssuedResolverDocumentTx(ctx context.Context, tx pgx.Tx, state api.DesiredState) error {
	ref, err := state.Dns.ResolverDocument.Get()
	if err != nil {
		return nil
	}
	raw, err := json.Marshal(ref)
	if err != nil {
		return err
	}
	var desiredID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT s.id FROM desired_states s JOIN desired_state_envelopes e ON e.desired_state_id=s.id
		WHERE s.node_id=$1 AND s.incarnation_id=$2 AND s.generation=$3::numeric
		AND e.envelope->'signed'->'dns'->'resolver_document'=$4::jsonb`, state.NodeId, state.NodeIncarnationId, state.ConfigGeneration.String(), raw).Scan(&desiredID)
	if err != nil {
		return err
	}
	event, err := audit.NewEvent(audit.Entry{
		Actor:  audit.Actor{Type: api.AuditEventActorTypeSystem, ID: SystemActorUUID},
		Action: "node.resolver_document_issued", Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: state.NodeId}, Result: api.AuditEventResultSucceeded,
		After:  map[string]any{"desired_state_id": desiredID.String(), "node_incarnation_id": state.NodeIncarnationId.String(), "config_generation": state.ConfigGeneration.String(), "resolver_document": ref},
		Fields: []string{"desired_state_id", "node_incarnation_id", "config_generation", "resolver_document"},
	})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
}

// ResolverDocumentForNode authorizes only references issued to this current
// incarnation at the node's current or immediately previous configuration generation.
func (s *Service) ResolverDocumentForNode(ctx context.Context, source ResolverDocumentSource, node, incarnation, document uuid.UUID) (ResolverDocument, error) {
	missing := apierr.New(apierr.NotFound, "Resolver document not found.")
	var raw []byte
	err := s.pool.QueryRow(ctx, `WITH issued AS (
		SELECT s.node_id,s.incarnation_id,s.generation,e.envelope->'signed'->'dns'->'resolver_document' AS reference
		FROM desired_state_envelopes e JOIN desired_states s ON s.id=e.desired_state_id WHERE s.node_id=$1 AND s.incarnation_id=$2
		UNION ALL
		SELECT s.node_id,s.incarnation_id,s.generation,a.after_diff->'resolver_document'
		FROM audit_events a JOIN desired_states s ON s.id::text=a.after_diff->>'desired_state_id'
		WHERE a.action='node.resolver_document_issued' AND a.resource='node:'||$1::text AND a.result='succeeded'
		AND s.node_id=$1 AND s.incarnation_id=$2 AND s.generation::text=a.after_diff->>'config_generation'
		AND s.incarnation_id::text=a.after_diff->>'node_incarnation_id'
	)
	SELECT i.reference FROM issued i JOIN nodes n ON n.id=i.node_id JOIN node_incarnations c ON c.id=i.incarnation_id
	WHERE n.current_incarnation_id=$2 AND n.lifecycle_state='registered' AND c.revoked_at IS NULL
	AND i.generation IN (n.desired_config_generation,n.desired_config_generation-1)
	AND i.reference->>'document_id'=$3::text ORDER BY i.generation DESC LIMIT 1`, node, incarnation, document).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResolverDocument{}, missing
	}
	if err != nil {
		return ResolverDocument{}, err
	}
	var ref api.ResolverDocumentRef
	if err = json.Unmarshal(raw, &ref); err != nil {
		return ResolverDocument{}, err
	}
	if source == nil {
		return ResolverDocument{}, errors.New("resolver document source required")
	}
	doc, err := source.DocumentByID(ctx, document)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResolverDocument{}, missing
	}
	if err != nil {
		return ResolverDocument{}, err
	}
	if doc.Reference != ref {
		return ResolverDocument{}, errors.New("resolver document differs from issued reference")
	}
	return doc, ValidateResolverDocument(doc)
}

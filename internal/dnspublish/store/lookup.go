package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DocumentReader provides immutable resolver documents to the nodes owner.
type DocumentReader struct{ pool *pgxpool.Pool }

var _ dnspublish.DocumentReader = (*DocumentReader)(nil)

func NewDocumentReader(pool *pgxpool.Pool) *DocumentReader {
	return &DocumentReader{pool: pool}
}

// LatestPublished resolves all reference fields and bytes in one database
// snapshot. A withdrawn head, or one whose revision is no longer active, is
// unavailable for new desired state even though its immutable bytes remain.
func (r *DocumentReader) LatestPublished(ctx context.Context, profileDefinitionID uuid.UUID) (dnspublish.PublishedDocument, error) {
	if r == nil || r.pool == nil {
		return dnspublish.PublishedDocument{}, errors.New("resolver document storage required")
	}
	return readDocument(r.pool.QueryRow(ctx, `SELECT d.id,d.profile_revision_id,d.policy_generation::text,d.document_sha256,d.canonical_bytes
 FROM dns_publication_heads h
 JOIN dns_resolver_documents d ON d.definition_id=h.definition_id AND d.profile_revision_id=h.profile_revision_id
  AND d.policy_generation=h.policy_generation AND d.document_sha256=h.document_sha256
 JOIN policy_definitions p ON p.id=h.definition_id AND p.active_revision_id=h.profile_revision_id
 WHERE h.definition_id=$1 AND h.status='active' AND p.type='dns_profile' AND dns_profile_feeds_eligible(d.profile_revision_id,clock_timestamp())`, profileDefinitionID))
}

// GetByID returns the version named by an already authorized issued reference,
// including after a newer publication or profile withdrawal, while every pinned
// feed approval is active and valid. It never follows the head.
func (r *DocumentReader) GetByID(ctx context.Context, documentID uuid.UUID) (dnspublish.PublishedDocument, error) {
	if r == nil || r.pool == nil {
		return dnspublish.PublishedDocument{}, errors.New("resolver document storage required")
	}
	return readDocument(r.pool.QueryRow(ctx, `SELECT id,profile_revision_id,policy_generation::text,document_sha256,canonical_bytes
 FROM dns_resolver_documents WHERE id=$1 AND dns_profile_feeds_eligible(profile_revision_id,clock_timestamp())`, documentID))
}

func readDocument(row pgx.Row) (dnspublish.PublishedDocument, error) {
	var out dnspublish.PublishedDocument
	var generation string
	err := row.Scan(&out.DocumentID, &out.ProfileRevisionID, &generation, &out.ContentSHA256, &out.Canonical)
	if errors.Is(err, pgx.ErrNoRows) {
		return dnspublish.PublishedDocument{}, dnspublish.ErrDocumentNotFound
	}
	if err != nil {
		return dnspublish.PublishedDocument{}, err
	}
	out.Generation, err = decstr.Parse(generation)
	if err != nil || out.Generation == 0 || out.DocumentID == uuid.Nil || out.ProfileRevisionID == uuid.Nil {
		return dnspublish.PublishedDocument{}, errors.New("invalid stored resolver reference")
	}
	out.SizeBytes = len(out.Canonical)
	if out.SizeBytes == 0 || out.SizeBytes > canonjson.MaxBytes {
		return dnspublish.PublishedDocument{}, errors.New("invalid stored resolver size")
	}
	var doc dnspublish.Document
	if err := json.Unmarshal(out.Canonical, &doc); err != nil {
		return dnspublish.PublishedDocument{}, errors.New("invalid stored resolver document")
	}
	if err := dnspublish.Verify(doc); err != nil {
		return dnspublish.PublishedDocument{}, errors.New("invalid stored resolver document")
	}
	hash, canonical, err := dnspublish.Hash(doc)
	if err != nil || hash != out.ContentSHA256 || !bytes.Equal(canonical, out.Canonical) {
		return dnspublish.PublishedDocument{}, errors.New("stored resolver document does not match its hash")
	}
	return out, nil
}

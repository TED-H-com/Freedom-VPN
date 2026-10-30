package httpwire

import (
	"context"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type resolverDocumentSource struct{ reader dnspublish.DocumentReader }

// NewResolverDocumentSource adapts the DNS owner's read-only API without copying its queries.
func NewResolverDocumentSource(pool *pgxpool.Pool) nodes.ResolverDocumentSource {
	return resolverDocumentSource{reader: dnsstore.NewDocumentReader(pool)}
}

func (s resolverDocumentSource) LatestPublished(ctx context.Context, definition uuid.UUID) (nodes.ResolverDocument, error) {
	doc, err := s.reader.LatestPublished(ctx, definition)
	return resolverDocument(doc, err)
}

func (s resolverDocumentSource) DocumentByID(ctx context.Context, id uuid.UUID) (nodes.ResolverDocument, error) {
	doc, err := s.reader.GetByID(ctx, id)
	return resolverDocument(doc, err)
}

func resolverDocument(doc dnspublish.PublishedDocument, err error) (nodes.ResolverDocument, error) {
	if errors.Is(err, dnspublish.ErrDocumentNotFound) {
		return nodes.ResolverDocument{}, pgx.ErrNoRows
	}
	if err != nil {
		return nodes.ResolverDocument{}, err
	}
	return nodes.ResolverDocument{Canonical: doc.Canonical, Reference: api.ResolverDocumentRef{
		DocumentId: doc.DocumentID, ProfileRevisionId: doc.ProfileRevisionID,
		Generation: doc.Generation, ContentSha256: doc.ContentSHA256, SizeBytes: doc.SizeBytes,
	}}, nil
}

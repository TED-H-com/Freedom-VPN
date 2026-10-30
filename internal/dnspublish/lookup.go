package dnspublish

import (
	"context"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// ErrDocumentNotFound covers an unknown document or a missing/withdrawn head.
// Delivery authorization is the caller's responsibility (D-056).
var ErrDocumentNotFound = errors.New("resolver document not found")

// PublishedDocument carries the immutable D-056 reference and the exact stored
// RFC 8785 bytes. Serve Canonical verbatim; encoding it again changes the digest.
// SizeBytes is the bounded byte length (1..1048576), as in ResolverDocumentRef.
type PublishedDocument struct {
	DocumentID        uuid.UUID
	ProfileRevisionID uuid.UUID
	Generation        decstr.U64
	ContentSHA256     string
	SizeBytes         int
	Canonical         []byte
}

// DocumentReader is the read-only delivery seam. store.NewDocumentReader
// implements it without requiring publisher configuration. LatestPublished
// selects the active publication head in one snapshot; GetByID retains access
// to immutable historical documents for callers authorizing a previously issued
// reference. Neither method grants a node permission to receive a document.
type DocumentReader interface {
	LatestPublished(ctx context.Context, profileDefinitionID uuid.UUID) (PublishedDocument, error)
	GetByID(ctx context.Context, documentID uuid.UUID) (PublishedDocument, error)
}

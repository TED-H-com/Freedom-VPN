//go:build integration

package store

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Called throughout the real activation/refresh/withdrawal integration flow.
// D-050: the publisher creates every document; tests only read these rows.
func assertDocumentLookup(t *testing.T, pool *pgxpool.Pool, reader dnspublish.DocumentReader, pub Publication) dnspublish.PublishedDocument {
	t.Helper()
	got, err := reader.LatestPublished(t.Context(), pub.DefinitionID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DocumentID == uuid.Nil || got.ProfileRevisionID != pub.ProfileRevisionID || got.Generation != pub.PolicyGeneration || got.ContentSHA256 != pub.DocumentHash || got.SizeBytes != len(pub.Canonical) || !bytes.Equal(got.Canonical, pub.Canonical) {
		t.Fatal("lookup does not match published metadata/bytes")
	}
	var storedID uuid.UUID
	var stored []byte
	if err := pool.QueryRow(t.Context(), `SELECT id,canonical_bytes FROM dns_resolver_documents WHERE profile_revision_id=$1 AND policy_generation=$2::numeric`, pub.ProfileRevisionID, pub.PolicyGeneration.String()).Scan(&storedID, &stored); err != nil {
		t.Fatal(err)
	}
	if storedID != got.DocumentID || !bytes.Equal(stored, got.Canonical) || got.ContentSHA256 != fmt.Sprintf("%x", sha256.Sum256(got.Canonical)) {
		t.Fatal("lookup re-encoded stored canonical bytes or returned wrong ID")
	}
	byID, err := reader.GetByID(t.Context(), got.DocumentID)
	if err != nil || byID.DocumentID != got.DocumentID || byID.Generation != got.Generation || byID.SizeBytes != got.SizeBytes || byID.ContentSHA256 != got.ContentSHA256 || byID.ProfileRevisionID != got.ProfileRevisionID || !bytes.Equal(byID.Canonical, got.Canonical) {
		t.Fatalf("ID lookup differs from head: %v", err)
	}
	return got
}

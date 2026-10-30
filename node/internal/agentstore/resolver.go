package agentstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// PutResolverCache stores verified resolver-document bytes under their
// document id. A later process with the same id can apply them without
// fetching again. The hash is the SHA-256 of body; a row that does not
// match is ignored on read.
func (s *Store) PutResolverCache(ctx context.Context, id, hash string, body []byte) error {
	if id == "" || hash == "" || len(body) == 0 || len(body) > canonjson.MaxBytes {
		return errors.New("agentstore: resolver cache rejected")
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != hash {
		return errors.New("agentstore: resolver cache hash mismatch")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO resolver_cache (document_id, content_sha256, body)
		VALUES (?, ?, ?)
		ON CONFLICT(document_id) DO UPDATE SET content_sha256 = excluded.content_sha256, body = excluded.body`,
		id, hash, body)
	return err
}

// ResolverCache returns the cached bytes for id when the stored hash matches
// those bytes. A missing or damaged row is ok false.
func (s *Store) ResolverCache(ctx context.Context, id string) ([]byte, bool, error) {
	var hash string
	var body []byte
	err := s.db.QueryRowContext(ctx, `SELECT content_sha256, body FROM resolver_cache WHERE document_id = ?`, id).Scan(&hash, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != hash {
		return nil, false, nil
	}
	return append([]byte(nil), body...), true, nil
}

package updatejournal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const seedPrefix = "seeded_baseline/"

// SeededBaseline is verified installation evidence, not a health check or a
// serving authorization. It is immutable per release, including on replay.
type SeededBaseline struct {
	ReleaseID      string    `json:"release_id"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	TreeHash       string    `json:"tree_hash"`
	BootID         string    `json:"boot_id"`
	SeededAt       time.Time `json:"seeded_at"`
}

// SeedBaseline atomically records the verified tree, current pointer, schema
// and seed evidence using the existing schema. It never rewrites established
// evidence, last_good, the forwarding gate, jobs or results. Callers hold the
// shared node lock and verify the live pointer and tree before calling.
func (j *Journal) SeedBaseline(ctx context.Context, r Release, bootID string) error {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var busy bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE phase != 'done') OR
		EXISTS(SELECT 1 FROM txns WHERE stage NOT IN ('locally_succeeded', 'rolled_back', 'recovery_required'))`).Scan(&busy)
	if err != nil {
		return err
	}
	if busy {
		return ErrActiveTransaction
	}
	current, _, err := getMeta(ctx, tx, metaCurrentRelease)
	if err != nil {
		return err
	}
	if current != "" && current != r.ReleaseID {
		return errors.New("updatejournal: seed refused: current release differs")
	}
	schema, hasSchema, err := getMeta(ctx, tx, metaStateSchema)
	if err != nil {
		return err
	}
	if hasSchema && schema != fmt.Sprint(r.StateSchemaTo) {
		return errors.New("updatejournal: seed refused: state schema differs")
	}
	var manifest, tree, artifact string
	var stateSchema int
	err = tx.QueryRowContext(ctx, `SELECT manifest_sha256, tree_hash, artifact_sha256, state_schema_to FROM releases WHERE release_id = ?`, r.ReleaseID).
		Scan(&manifest, &tree, &artifact, &stateSchema)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if exists && (manifest != r.ManifestSHA256 || tree != r.TreeHash || artifact != r.ArtifactSHA256 || stateSchema != r.StateSchemaTo) {
		return errors.New("updatejournal: seed refused: recorded release evidence differs")
	}
	raw, seeded, err := getMeta(ctx, tx, seedPrefix+r.ReleaseID)
	if err != nil {
		return err
	}
	if seeded {
		var old SeededBaseline
		if err := json.Unmarshal([]byte(raw), &old); err != nil {
			return err
		}
		if old.ReleaseID != r.ReleaseID || old.ManifestSHA256 != r.ManifestSHA256 || old.TreeHash != r.TreeHash || !exists || current != r.ReleaseID || !hasSchema {
			return errors.New("updatejournal: seed refused: baseline evidence differs")
		}
		return nil // Original boot, timestamp and release row survive replay.
	}
	if current != "" && !exists {
		return errors.New("updatejournal: seed refused: recorded current has no release evidence")
	}
	at := j.now().UTC()
	if !exists {
		_, err = tx.ExecContext(ctx, `INSERT INTO releases(release_id, manifest_sha256, manifest_envelope, artifact_sha256, tree_hash, state_schema_to, installed_at)
			VALUES(?, ?, ?, ?, ?, ?, ?)`, r.ReleaseID, r.ManifestSHA256, r.ManifestEnvelope, r.ArtifactSHA256, r.TreeHash, r.StateSchemaTo, at.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	baseline, err := json.Marshal(SeededBaseline{ReleaseID: r.ReleaseID, ManifestSHA256: r.ManifestSHA256, TreeHash: r.TreeHash, BootID: bootID, SeededAt: at})
	if err != nil {
		return err
	}
	for _, kv := range [][2]string{{metaCurrentRelease, r.ReleaseID}, {metaStateSchema, fmt.Sprint(r.StateSchemaTo)}, {seedPrefix + r.ReleaseID, string(baseline)}} {
		if err := setMeta(ctx, tx, kv[0], kv[1]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

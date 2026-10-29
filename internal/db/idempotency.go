package db

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// IdempotencyScope uses hashes, never the plaintext key or response (§14.5).
// The caller computes RequestHash from a validated, canonical request body.
type IdempotencyScope struct {
	Principal, Method, Route string
	KeyHash, RequestHash     []byte
}

// IdempotencyRecord returns the stable resource/operation even while in progress.
type IdempotencyRecord struct {
	ID                      uuid.UUID
	New                     bool
	State                   string
	ResourceType            *string
	ResourceID, OperationID *uuid.UUID
	ResponseStatus          *int
	CreatedAt, Expiry       time.Time
}

// BeginIdempotent shares the local state transaction. Only a New winner may
// execute the action; losers return the stored reference. Call CompleteIdempotent
// before committing the winner's transaction. An asynchronous operation is a
// completed HTTP command whose operation itself may still be pending.
// Zero ttl selects the §14.5 recommended minimum of seven days.
func BeginIdempotent(ctx context.Context, tx pgx.Tx, scope IdempotencyScope, ttl time.Duration) (IdempotencyRecord, error) {
	var record IdempotencyRecord
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour
	}
	if ttl < time.Millisecond || scope.Principal == "" || scope.Method == "" || scope.Route == "" || len(scope.KeyHash) != 32 || len(scope.RequestHash) != 32 {
		return record, errors.New("invalid idempotency request")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return record, err
	}
	var storedHash []byte
	err = tx.QueryRow(ctx, `INSERT INTO idempotency_requests(id,principal,method,route,key_hash,request_hash,expiry)
 VALUES($1,$2,$3,$4,$5,$6,clock_timestamp()+$7::bigint*interval '1 millisecond')
 ON CONFLICT(principal,method,route,key_hash) DO UPDATE SET
 id=excluded.id,request_hash=excluded.request_hash,state='in_progress',resource_type=NULL,resource_id=NULL,operation_id=NULL,response_status=NULL,created_at=clock_timestamp(),expiry=clock_timestamp()+$7::bigint*interval '1 millisecond'
 WHERE idempotency_requests.expiry<=clock_timestamp()
 RETURNING id,state,resource_type,resource_id,operation_id,response_status,created_at,expiry,request_hash`, id, scope.Principal, scope.Method, scope.Route, scope.KeyHash, scope.RequestHash, ttl.Milliseconds()).Scan(
		&record.ID, &record.State, &record.ResourceType, &record.ResourceID, &record.OperationID, &record.ResponseStatus, &record.CreatedAt, &record.Expiry, &storedHash)
	if err == nil {
		record.New = true
		return record, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return record, MapError(err)
	}
	err = tx.QueryRow(ctx, `SELECT id,state,resource_type,resource_id,operation_id,response_status,created_at,expiry,request_hash
 FROM idempotency_requests WHERE principal=$1 AND method=$2 AND route=$3 AND key_hash=$4 FOR UPDATE`, scope.Principal, scope.Method, scope.Route, scope.KeyHash).Scan(
		&record.ID, &record.State, &record.ResourceType, &record.ResourceID, &record.OperationID, &record.ResponseStatus, &record.CreatedAt, &record.Expiry, &storedHash)
	if err != nil {
		return record, MapError(err)
	}
	if !bytes.Equal(storedHash, scope.RequestHash) {
		return record, ErrIdempotencyConflict
	}
	return record, nil
}

// IdempotencyResult contains only references and an HTTP status. Secret-bearing
// responses must be reconstructed with secret_available=false on replay.
type IdempotencyResult struct {
	ResourceType            string
	ResourceID, OperationID *uuid.UUID
	ResponseStatus          int
}

// CompleteIdempotent is fenced by the record ID (renewed on key expiry).
func CompleteIdempotent(ctx context.Context, tx pgx.Tx, id uuid.UUID, result IdempotencyResult) error {
	if result.ResponseStatus < 100 || result.ResponseStatus > 599 || (result.ResourceID == nil && result.OperationID == nil) {
		return errors.New("invalid idempotency result")
	}
	tag, err := tx.Exec(ctx, `UPDATE idempotency_requests SET state='completed',resource_type=$2,resource_id=$3,operation_id=$4,response_status=$5
 WHERE id=$1 AND expiry>clock_timestamp() AND (state='in_progress' OR
 (resource_type IS NOT DISTINCT FROM $2 AND resource_id IS NOT DISTINCT FROM $3 AND operation_id IS NOT DISTINCT FROM $4 AND response_status=$5))`, id, result.ResourceType, result.ResourceID, result.OperationID, result.ResponseStatus)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrIdempotencyConflict
	}
	return nil
}

package db

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxEvent is a durable intent; its payload must already be schema-validated.
type OutboxEvent struct {
	ID, AggregateID                            uuid.UUID
	AggregateType, AggregateVersion, EventType string
	SchemaVersion                              int
	Payload                                    json.RawMessage
	AvailableAt                                time.Time
}

// EnqueueOutbox must share the domain transaction (§15.5).
func EnqueueOutbox(ctx context.Context, tx pgx.Tx, evt OutboxEvent) error {
	if evt.AggregateID == uuid.Nil || strings.TrimSpace(evt.AggregateType) == "" || strings.TrimSpace(evt.EventType) == "" || !decimalUint64(evt.AggregateVersion) || !jsonObject(evt.Payload) || evt.SchemaVersion < 1 {
		return errors.New("invalid outbox event")
	}
	if evt.ID == uuid.Nil {
		var err error
		evt.ID, err = uuid.NewV7()
		if err != nil {
			return err
		}
	}
	var available any
	if !evt.AvailableAt.IsZero() {
		available = evt.AvailableAt
	}
	_, err := tx.Exec(ctx, `INSERT INTO outbox_events(id,aggregate_id,aggregate_type,aggregate_version,event_type,schema_version,payload,available_at)
 VALUES($1,$2,$3,$4::numeric,$5,$6,$7,coalesce($8::timestamptz,clock_timestamp()))`, evt.ID, evt.AggregateID, evt.AggregateType, evt.AggregateVersion, evt.EventType, evt.SchemaVersion, evt.Payload, available)
	return MapError(err)
}

// ClaimedEvent includes a fencing token; a worker name alone is not ownership.
type ClaimedEvent struct {
	OutboxEvent
	Worker     string
	LeaseToken uuid.UUID
	LeaseUntil time.Time
	Attempt    int
}

// ClaimOutbox commits claims before returning. SKIP LOCKED gives disjoint claims
// to concurrent workers, and expired leases become eligible for redelivery.
func ClaimOutbox(ctx context.Context, pool *pgxpool.Pool, worker string, n int, leaseFor time.Duration) ([]ClaimedEvent, error) {
	if worker == "" || n < 1 || n > 1000 || leaseFor < time.Millisecond || leaseFor > 24*time.Hour {
		return nil, errors.New("invalid outbox claim")
	}
	token, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `WITH candidates AS (
 SELECT id FROM outbox_events WHERE delivered_at IS NULL AND available_at<=clock_timestamp()
 AND (lease_until IS NULL OR lease_until<=clock_timestamp()) ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT $1
 ), claimed AS (UPDATE outbox_events e SET lease_owner=$2,lease_token=$3,lease_until=clock_timestamp()+$4::bigint*interval '1 millisecond',attempt=attempt+1
 FROM candidates c WHERE e.id=c.id
 RETURNING e.id,e.aggregate_id,e.aggregate_type,e.aggregate_version::text,e.event_type,e.schema_version,e.payload,e.available_at,e.lease_until,e.attempt) SELECT * FROM claimed ORDER BY available_at,id`, n, worker, token, leaseFor.Milliseconds())
	if err != nil {
		return nil, MapError(err)
	}
	defer rows.Close()
	var events []ClaimedEvent
	for rows.Next() {
		evt := ClaimedEvent{Worker: worker, LeaseToken: token}
		if err := rows.Scan(&evt.ID, &evt.AggregateID, &evt.AggregateType, &evt.AggregateVersion, &evt.EventType, &evt.SchemaVersion, &evt.Payload, &evt.AvailableAt, &evt.LeaseUntil, &evt.Attempt); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(err)
	}
	return events, nil
}

// CompleteOutbox requires current, unexpired ownership and read-back evidence
// already obtained by the caller. A late worker cannot complete a newer claim.
func CompleteOutbox(ctx context.Context, pool *pgxpool.Pool, evt ClaimedEvent) error {
	tag, err := pool.Exec(ctx, `UPDATE outbox_events SET delivered_at=clock_timestamp(),lease_owner=NULL,lease_token=NULL,lease_until=NULL
 WHERE id=$1 AND lease_owner=$2 AND lease_token=$3 AND lease_until>clock_timestamp() AND delivered_at IS NULL`, evt.ID, evt.Worker, evt.LeaseToken)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

// RetryOutbox releases a current claim and sets a bounded caller-selected delay.
// Backoff may incorporate a provider Retry-After; no external error body is saved.
func RetryOutbox(ctx context.Context, pool *pgxpool.Pool, evt ClaimedEvent, backoff time.Duration) error {
	if backoff < 0 || backoff > 24*time.Hour {
		return errors.New("invalid outbox backoff")
	}
	tag, err := pool.Exec(ctx, `UPDATE outbox_events SET available_at=clock_timestamp()+$4::bigint*interval '1 millisecond',lease_owner=NULL,lease_token=NULL,lease_until=NULL
 WHERE id=$1 AND lease_owner=$2 AND lease_token=$3 AND lease_until>clock_timestamp() AND delivered_at IS NULL`, evt.ID, evt.Worker, evt.LeaseToken, backoff.Milliseconds())
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

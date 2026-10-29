package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// AdjustmentCreate describes an append-only usage adjustment (§15.4).
// Adjustments are operational byte corrections kept strictly separate from financial refunds.
type AdjustmentCreate struct {
	MemberID        uuid.UUID
	Period          string // YYYY-MM UTC month
	OriginalEventID *uuid.UUID
	IntervalStart   *time.Time
	IntervalEnd     *time.Time
	UploadBytes     int64
	DownloadBytes   int64
	Reason          string
	Actor           string
}

// Adjustment is a committed row in usage_adjustments (§15.4).
type Adjustment struct {
	ID              uuid.UUID  `json:"id"`
	MemberID        uuid.UUID  `json:"member_id"`
	Period          string     `json:"period"`
	OriginalEventID *uuid.UUID `json:"original_event_id,omitempty"`
	IntervalStart   *time.Time `json:"interval_start,omitempty"`
	IntervalEnd     *time.Time `json:"interval_end,omitempty"`
	UploadBytes     int64      `json:"upload_bytes"`
	DownloadBytes   int64      `json:"download_bytes"`
	Reason          string     `json:"reason"`
	Actor           string     `json:"actor"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CreateAdjustment records an append-only usage adjustment for a member and month period (§15.4).
func (s *Service) CreateAdjustment(ctx context.Context, req AdjustmentCreate) (Adjustment, error) {
	if req.MemberID == uuid.Nil {
		return Adjustment{}, apierr.Validation("member_id is required", apierr.FieldError{Field: "/member_id", Reason: "required"})
	}
	monthPeriod, err := ParseMonth(req.Period)
	if err != nil {
		return Adjustment{}, apierr.Validation(err.Error(), apierr.FieldError{Field: "/period", Reason: "format"})
	}
	if strings.TrimSpace(req.Reason) == "" {
		return Adjustment{}, apierr.Validation("reason is required", apierr.FieldError{Field: "/reason", Reason: "required"})
	}
	if strings.TrimSpace(req.Actor) == "" {
		return Adjustment{}, apierr.Validation("actor is required", apierr.FieldError{Field: "/actor", Reason: "required"})
	}

	var iStart, iEnd *time.Time
	if (req.IntervalStart != nil) != (req.IntervalEnd != nil) {
		return Adjustment{}, apierr.Validation("interval_start and interval_end must be provided together", apierr.FieldError{Field: "/interval_end", Reason: "required"})
	}
	if req.IntervalStart != nil && req.IntervalEnd != nil {
		sUTC := req.IntervalStart.UTC()
		eUTC := req.IntervalEnd.UTC()
		if !eUTC.After(sUTC) {
			return Adjustment{}, apierr.Validation("interval_end must be after interval_start", apierr.FieldError{Field: "/interval_end", Reason: "interval_order"})
		}
		iStart = &sUTC
		iEnd = &eUTC
	} else if req.OriginalEventID == nil {
		// Default interval to the accounting month window so the table constraint
		// CHECK(original_event_id IS NOT NULL OR interval_start IS NOT NULL) is satisfied.
		sUTC := monthPeriod.Start
		eUTC := monthPeriod.End
		iStart = &sUTC
		iEnd = &eUTC
	}

	now := s.now()
	adjID, err := uuid.NewV7()
	if err != nil {
		return Adjustment{}, fmt.Errorf("usage: generate adjustment id: %w", err)
	}

	var out Adjustment
	err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM members WHERE id = $1)`, req.MemberID).Scan(&exists); err != nil {
			return fmt.Errorf("usage: check member existence: %w", err)
		}
		if !exists {
			return apierr.New(apierr.ResourceNotFound, "member not found")
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO usage_adjustments (
				id, member_id, period, original_event_id,
				interval_start, interval_end,
				upload_bytes, download_bytes,
				reason, actor, created_at
			) VALUES (
				$1, $2, $3, $4,
				$5, $6,
				$7::numeric, $8::numeric,
				$9, $10, $11
			)
		`,
			adjID,
			req.MemberID,
			monthPeriod.ID,
			req.OriginalEventID,
			iStart,
			iEnd,
			strconv.FormatInt(req.UploadBytes, 10),
			strconv.FormatInt(req.DownloadBytes, 10),
			req.Reason,
			req.Actor,
			now,
		)
		if err != nil {
			return fmt.Errorf("usage: insert adjustment: %w", err)
		}

		actID, actType := auditActor(req.Actor)
		afterBytes, _ := json.Marshal(map[string]any{
			"adjustment_id":  adjID.String(),
			"member_id":      req.MemberID.String(),
			"period":         monthPeriod.ID,
			"upload_bytes":   req.UploadBytes,
			"download_bytes": req.DownloadBytes,
			"reason":         req.Reason,
		})
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         actID,
			ActorType:     actType,
			Action:        "usage.adjustment.create",
			Resource:      "member:" + req.MemberID.String(),
			Result:        "succeeded",
			SchemaVersion: 1,
			Before:        []byte(`{}`),
			After:         afterBytes,
		}); err != nil {
			return fmt.Errorf("usage: append adjustment audit: %w", err)
		}

		out = Adjustment{
			ID:              adjID,
			MemberID:        req.MemberID,
			Period:          monthPeriod.ID,
			OriginalEventID: req.OriginalEventID,
			IntervalStart:   iStart,
			IntervalEnd:     iEnd,
			UploadBytes:     req.UploadBytes,
			DownloadBytes:   req.DownloadBytes,
			Reason:          req.Reason,
			Actor:           req.Actor,
			CreatedAt:       now,
		}
		return nil
	})
	if err != nil {
		return Adjustment{}, err
	}
	return out, nil
}

// ListAdjustments returns all adjustments for a member in a UTC month period, ordered by created_at.
func (s *Service) ListAdjustments(ctx context.Context, memberID uuid.UUID, period string) ([]Adjustment, error) {
	if _, err := ParseMonth(period); err != nil {
		return nil, apierr.Validation(err.Error(), apierr.FieldError{Field: "/period", Reason: "format"})
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, member_id, period, original_event_id,
		       interval_start, interval_end,
		       upload_bytes::text, download_bytes::text,
		       reason, actor, created_at
		FROM usage_adjustments
		WHERE member_id = $1 AND period = $2
		ORDER BY created_at ASC, id ASC
	`, memberID, period)
	if err != nil {
		return nil, fmt.Errorf("usage: query adjustments: %w", err)
	}
	defer rows.Close()

	var result []Adjustment
	for rows.Next() {
		var (
			a              Adjustment
			upText, dnText string
		)
		if err := rows.Scan(
			&a.ID, &a.MemberID, &a.Period, &a.OriginalEventID,
			&a.IntervalStart, &a.IntervalEnd,
			&upText, &dnText,
			&a.Reason, &a.Actor, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("usage: scan adjustment: %w", err)
		}
		u, err := strconv.ParseInt(upText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("usage: parse adjustment upload_bytes: %w", err)
		}
		d, err := strconv.ParseInt(dnText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("usage: parse adjustment download_bytes: %w", err)
		}
		a.UploadBytes = u
		a.DownloadBytes = d
		a.CreatedAt = a.CreatedAt.UTC()
		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usage: iterate adjustments: %w", err)
	}
	return result, nil
}

// applyNumericDelta applies a signed big.Int delta to an unsigned uint64 base,
// clamping at [0, MaxUint64] so uint64 decimal string formatting never overflows.
func applyNumericDelta(base uint64, delta *big.Int) uint64 {
	if delta == nil || delta.Sign() == 0 {
		return base
	}
	b := new(big.Int).SetUint64(base)
	b.Add(b, delta)
	if b.Sign() <= 0 {
		return 0
	}
	maxU64 := new(big.Int).SetUint64(^uint64(0))
	if b.Cmp(maxU64) > 0 {
		return ^uint64(0)
	}
	return b.Uint64()
}

func clampBigUint64(v *big.Int) uint64 {
	if v == nil || v.Sign() <= 0 {
		return 0
	}
	maxU64 := new(big.Int).SetUint64(^uint64(0))
	if v.Cmp(maxU64) > 0 {
		return ^uint64(0)
	}
	return v.Uint64()
}
